"""3wayproxy relay — WebSocket carrier between player and aggregator."""

from __future__ import annotations

import asyncio
import contextlib
import logging
import os
from collections import defaultdict
from dataclasses import dataclass, field
from pathlib import Path

from fastapi import FastAPI, WebSocket, WebSocketDisconnect
from fastapi.responses import FileResponse, HTMLResponse, PlainTextResponse
from fastapi.staticfiles import StaticFiles

from app.protocol import (
    build_handshake_ack,
    is_fragment,
    parse_handshake,
    parse_handshake_agg,
    parse_resume,
)

logging.basicConfig(level=logging.INFO)
log = logging.getLogger("relay")

RELAY_SHARD_ID = int(os.environ.get("RELAY_SHARD_ID", "0"))
GAME_ROOT = Path(__file__).resolve().parents[2] / "game"

app = FastAPI(title="3wayproxy relay", docs_url=None, redoc_url=None)

if GAME_ROOT.is_dir():
    app.mount("/game", StaticFiles(directory=str(GAME_ROOT)), name="game")


@dataclass
class SessionHub:
    uplink: asyncio.Queue[bytes] = field(default_factory=asyncio.Queue)
    downlink: asyncio.Queue[bytes] = field(default_factory=asyncio.Queue)
    players: int = 0
    spectators: int = 0


_sessions: dict[int, SessionHub] = defaultdict(SessionHub)


def _hub(session_id: int) -> SessionHub:
    return _sessions[session_id]


@app.get("/")
async def index() -> HTMLResponse:
    return HTMLResponse(
        "<!doctype html><html><head><title>Game</title></head>"
        "<body><h1>3wayproxy relay</h1>"
        "<p>Phase 3: carrier page at <a href=\"/play.html\">/play.html</a> "
        "(snake game cover — later).</p>"
        "</body></html>"
    )


@app.get("/play.html")
async def play_page() -> FileResponse:
    path = GAME_ROOT / "stub" / "play.html"
    return FileResponse(path, media_type="text/html")


@app.get("/health")
async def health() -> PlainTextResponse:
    return PlainTextResponse("ok")


async def _pump_downlink(ws: WebSocket, hub: SessionHub) -> None:
    """Player: send aggregator → client frames without waiting on player input."""
    while True:
        frame = await hub.downlink.get()
        await ws.send_bytes(frame)


async def _pump_uplink(ws: WebSocket, hub: SessionHub) -> None:
    """Spectator: forward player → aggregator frames immediately."""
    while True:
        frame = await hub.uplink.get()
        await ws.send_bytes(frame)


@app.websocket("/ws/play")
async def ws_play(ws: WebSocket) -> None:
    await ws.accept(subprotocol="snake-game-v1")
    session_id: int | None = None
    hub: SessionHub | None = None
    pump: asyncio.Task[None] | None = None
    try:
        while True:
            data = await ws.receive_bytes()
            sid_resume = parse_resume(data)
            if sid_resume is not None:
                session_id = sid_resume
                hub = _hub(session_id)
                if pump is None:
                    pump = asyncio.create_task(_pump_downlink(ws, hub))
                log.info("player resume session=%d", session_id)
                await ws.send_bytes(build_handshake_ack(session_id, RELAY_SHARD_ID))
                continue
            hs = parse_handshake(data)
            if hs is not None:
                shard_id, sid, _ = hs
                session_id = sid
                hub = _hub(session_id)
                hub.players += 1
                if pump is None:
                    pump = asyncio.create_task(_pump_downlink(ws, hub))
                log.info("player handshake session=%d shard=%d relay=%d", session_id, shard_id, RELAY_SHARD_ID)
                await ws.send_bytes(build_handshake_ack(session_id, RELAY_SHARD_ID))
                continue
            if session_id is None or hub is None:
                continue
            if is_fragment(data):
                await hub.uplink.put(data)
                log.debug("uplink fragment session=%d len=%d", session_id, len(data))
    except WebSocketDisconnect:
        pass
    finally:
        if pump is not None:
            pump.cancel()
            with contextlib.suppress(asyncio.CancelledError):
                await pump
        if hub is not None:
            hub.players = max(0, hub.players - 1)


@app.websocket("/ws/spectator")
async def ws_spectator(ws: WebSocket) -> None:
    await ws.accept(subprotocol="snake-game-v1")
    session_id: int | None = None
    hub: SessionHub | None = None
    pump: asyncio.Task[None] | None = None
    try:
        while True:
            data = await ws.receive_bytes()
            agg = parse_handshake_agg(data)
            if agg is not None:
                relay_id, sid = agg
                session_id = sid
                hub = _hub(session_id)
                hub.spectators += 1
                if pump is None:
                    pump = asyncio.create_task(_pump_uplink(ws, hub))
                log.info("spectator handshake session=%d relay=%d", session_id, relay_id)
                continue
            if session_id is None or hub is None:
                continue
            if is_fragment(data):
                await hub.downlink.put(data)
                log.debug("downlink fragment session=%d len=%d", session_id, len(data))
    except WebSocketDisconnect:
        pass
    finally:
        if pump is not None:
            pump.cancel()
            with contextlib.suppress(asyncio.CancelledError):
                await pump
        if hub is not None:
            hub.spectators = max(0, hub.spectators - 1)
