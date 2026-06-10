"""3wayproxy relay — game facade + WebSocket carrier (phase 1 stub)."""

from __future__ import annotations

import asyncio
from collections import defaultdict, deque
from typing import Deque

from fastapi import FastAPI, WebSocket, WebSocketDisconnect
from fastapi.responses import HTMLResponse

app = FastAPI(title="Snake Game Relay", docs_url=None, redoc_url=None)

# session_id -> queue of raw binary frames (uplink from player, for spectator)
_uplink: dict[int, Deque[bytes]] = defaultdict(lambda: deque(maxlen=4096))
# session_id -> queue of downlink frames (from spectator to player)
_downlink: dict[int, Deque[bytes]] = defaultdict(lambda: deque(maxlen=4096))


@app.get("/")
async def index() -> HTMLResponse:
    return HTMLResponse(
        "<!doctype html><title>Snake</title>"
        "<canvas id=c width=400 height=400></canvas>"
        "<script>/* game + ws carrier — phase 3 */</script>"
    )


@app.websocket("/ws/play")
async def ws_play(ws: WebSocket) -> None:
    await ws.accept(subprotocol="snake-game-v1")
    session_id: int | None = None
    try:
        while True:
            data = await ws.receive_bytes()
            if len(data) >= 9 and data[0] == 0x01:  # HANDSHAKE
                session_id = int.from_bytes(data[2:10], "big")
                ack = bytes([0x02, 1]) + session_id.to_bytes(8, "big") + (1200).to_bytes(2, "big") + bytes([data[2]])
                await ws.send_bytes(ack)
                continue
            if session_id is not None:
                _uplink[session_id].append(data)
                # deliver downlink if any
                q = _downlink[session_id]
                while q:
                    await ws.send_bytes(q.popleft())
    except WebSocketDisconnect:
        pass


@app.websocket("/ws/spectator")
async def ws_spectator(ws: WebSocket) -> None:
    await ws.accept(subprotocol="snake-game-v1")
    session_id: int | None = None
    try:
        while True:
            if session_id is not None:
                q = _uplink[session_id]
                while q:
                    await ws.send_bytes(q.popleft())
            try:
                data = await asyncio.wait_for(ws.receive_bytes(), timeout=0.05)
            except asyncio.TimeoutError:
                continue
            if len(data) >= 9 and data[0] == 0xA1:
                session_id = int.from_bytes(data[2:10], "big")
                continue
            if session_id is not None:
                _downlink[session_id].append(data)
    except WebSocketDisconnect:
        pass
