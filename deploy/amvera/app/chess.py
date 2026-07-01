"""Anonymous chess lobby for active probing cover."""

from __future__ import annotations

import asyncio
import json
import logging
import uuid
from dataclasses import dataclass, field
from typing import Any

import chess
from fastapi import WebSocket, WebSocketDisconnect
from fastapi.responses import FileResponse

log = logging.getLogger("relay.chess")

GAME_ROOT = None  # set by register()


@dataclass
class PlayerSlot:
    ws: WebSocket
    player_id: str = field(default_factory=lambda: uuid.uuid4().hex[:8])


@dataclass
class ChessTable:
    """One anonymous table per relay shard."""

    lock: asyncio.Lock = field(default_factory=asyncio.Lock)
    waiting: PlayerSlot | None = None
    waiting_color: str | None = None  # color chosen by host
    white: PlayerSlot | None = None
    black: PlayerSlot | None = None
    board: chess.Board = field(default_factory=chess.Board)

    def status_public(self) -> dict[str, Any]:
        if self.white and self.black:
            turn = "white" if self.board.turn == chess.WHITE else "black"
            return {
                "phase": "playing",
                "fen": self.board.fen(),
                "turn": turn,
                "waiting_color": None,
                "is_check": self.board.is_check(),
                "is_game_over": self.board.is_game_over(),
                "result": self._result(),
            }
        if self.waiting:
            return {
                "phase": "waiting",
                "fen": chess.STARTING_FEN,
                "turn": None,
                "waiting_color": self.waiting_color,
                "is_check": False,
                "is_game_over": False,
                "result": None,
            }
        return {
            "phase": "empty",
            "fen": chess.STARTING_FEN,
            "turn": None,
            "waiting_color": None,
            "is_check": False,
            "is_game_over": False,
            "result": None,
        }

    def _result(self) -> str | None:
        if not self.board.is_game_over():
            return None
        return self.board.result(claim_draw=True)

    def player_color(self, ws: WebSocket) -> str | None:
        if self.white and self.white.ws is ws:
            return "white"
        if self.black and self.black.ws is ws:
            return "black"
        return None

    def clear_player(self, ws: WebSocket) -> None:
        if self.waiting and self.waiting.ws is ws:
            self.waiting = None
            self.waiting_color = None
        if self.white and self.white.ws is ws:
            self.white = None
        if self.black and self.black.ws is ws:
            self.black = None
        if not self.white or not self.black:
            self.board = chess.Board()


_table = ChessTable()
_sockets: set[WebSocket] = set()


def _msg(obj: dict[str, Any]) -> str:
    return json.dumps(obj, ensure_ascii=False)


async def _send(ws: WebSocket, obj: dict[str, Any]) -> None:
    await ws.send_text(_msg(obj))


async def _broadcast(obj: dict[str, Any]) -> None:
    dead: list[WebSocket] = []
    for ws in list(_sockets):
        try:
            await ws.send_text(_msg(obj))
        except Exception:
            dead.append(ws)
    for ws in dead:
        _sockets.discard(ws)


async def _push_state(extra: dict[str, Any] | None = None) -> None:
    base = _table.status_public()
    if extra:
        base.update(extra)
    payload = {"type": "state", **base}
    await _broadcast(payload)


async def _handle_host(ws: WebSocket, color: str) -> None:
    color = color.lower()
    if color not in ("white", "black"):
        await _send(ws, {"type": "error", "message": "Цвет: white или black"})
        return
    async with _table.lock:
        if _table.white and _table.black:
            await _send(ws, {"type": "error", "message": "Стол занят — идёт партия"})
            return
        if _table.waiting:
            await _send(ws, {"type": "error", "message": "Уже ждут соперника — нажмите «Присоединиться»"})
            return
        _table.waiting = PlayerSlot(ws=ws)
        _table.waiting_color = color
        await _send(ws, {"type": "hosted", "color": color, "player_id": _table.waiting.player_id})
        await _push_state()


async def _handle_join(ws: WebSocket) -> None:
    async with _table.lock:
        if _table.white and _table.black:
            await _send(ws, {"type": "error", "message": "Партия уже идёт"})
            return
        if not _table.waiting or not _table.waiting_color:
            await _send(ws, {"type": "error", "message": "Никто не ждёт — создайте игру и выберите цвет"})
            return
        host = _table.waiting
        host_color = _table.waiting_color
        guest_color = "black" if host_color == "white" else "white"
        guest = PlayerSlot(ws=ws)
        _table.waiting = None
        _table.waiting_color = None
        _table.board = chess.Board()
        if host_color == "white":
            _table.white = host
            _table.black = guest
        else:
            _table.white = guest
            _table.black = host
        await _send(host.ws, {"type": "matched", "color": host_color, "opponent": "guest"})
        await _send(guest.ws, {"type": "matched", "color": guest_color, "opponent": "host"})
        await _push_state()


async def _handle_move(ws: WebSocket, from_sq: str, to_sq: str, promotion: str | None) -> None:
    async with _table.lock:
        color = _table.player_color(ws)
        if not color or not _table.white or not _table.black:
            await _send(ws, {"type": "error", "message": "Вы не в партии"})
            return
        if (color == "white") != _table.board.turn:
            await _send(ws, {"type": "error", "message": "Не ваш ход"})
            return
        try:
            move = chess.Move.from_uci(from_sq + to_sq + (promotion or ""))
        except ValueError:
            await _send(ws, {"type": "error", "message": "Некорректный ход"})
            return
        if move not in _table.board.legal_moves:
            await _send(ws, {"type": "error", "message": "Недопустимый ход"})
            return
        _table.board.push(move)
        await _push_state({"last_move": from_sq + to_sq})


async def _handle_resign(ws: WebSocket) -> None:
    async with _table.lock:
        color = _table.player_color(ws)
        if not color or not _table.white or not _table.black:
            return
        winner = "black" if color == "white" else "white"
        result = "0-1" if winner == "black" else "1-0"
        await _broadcast({
            "type": "state",
            "phase": "empty",
            "fen": chess.STARTING_FEN,
            "turn": None,
            "waiting_color": None,
            "is_check": False,
            "is_game_over": True,
            "result": result,
            "resigned": color,
        })
        _table.white = None
        _table.black = None
        _table.board = chess.Board()


async def _handle_leave(ws: WebSocket) -> None:
    async with _table.lock:
        _table.clear_player(ws)
        await _push_state({"left": True})


async def chess_websocket(ws: WebSocket) -> None:
    await ws.accept()
    _sockets.add(ws)
    try:
        await _send(ws, {"type": "hello", "player_id": uuid.uuid4().hex[:8]})
        await _send(ws, {"type": "state", **_table.status_public()})
        while True:
            raw = await ws.receive_text()
            try:
                data = json.loads(raw)
            except json.JSONDecodeError:
                await _send(ws, {"type": "error", "message": "Ожидается JSON"})
                continue
            kind = data.get("type")
            if kind == "status":
                await _send(ws, {"type": "state", **_table.status_public()})
            elif kind == "host":
                await _handle_host(ws, str(data.get("color", "")))
            elif kind == "join":
                await _handle_join(ws)
            elif kind == "move":
                await _handle_move(
                    ws,
                    str(data.get("from", "")).lower(),
                    str(data.get("to", "")).lower(),
                    data.get("promotion"),
                )
            elif kind == "resign":
                await _handle_resign(ws)
            elif kind == "leave":
                await _handle_leave(ws)
            else:
                await _send(ws, {"type": "error", "message": f"Неизвестная команда: {kind}"})
    except WebSocketDisconnect:
        pass
    finally:
        _sockets.discard(ws)
        async with _table.lock:
            _table.clear_player(ws)
            await _push_state()


def register(app, game_root) -> None:
    global GAME_ROOT
    GAME_ROOT = game_root

    @app.get("/")
    async def chess_index() -> FileResponse:
        return FileResponse(game_root / "chess" / "index.html", media_type="text/html")

    app.add_api_websocket_route("/ws/chess", chess_websocket)
