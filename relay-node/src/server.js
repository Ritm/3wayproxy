#!/usr/bin/env node
/**
 * 3wayproxy relay — Node.js (phase 3+).
 * Endpoints: GET /, /play.html, /health, /game/*
 *            WS  /ws/play, /ws/spectator, /ws/chess
 */

import { createServer } from 'http';
import { dirname, join } from 'path';
import { fileURLToPath } from 'url';
import express from 'express';
import { WebSocketServer } from 'ws';

import { attachChess } from './chess.js';
import { hub } from './hub.js';
import {
  buildHandshakeAck,
  isFragment,
  parseHandshake,
  parseHandshakeAgg,
  parseResume,
} from './protocol.js';

const __dirname = dirname(fileURLToPath(import.meta.url));
const ROOT = join(__dirname, '..', '..');
const GAME_ROOT = join(ROOT, 'game');

const RELAY_SHARD_ID = parseInt(process.env.RELAY_SHARD_ID || '0', 10);
const PORT = parseInt(process.env.PORT || '8000', 10);

const app = express();
const server = createServer(app);

app.use('/game', express.static(GAME_ROOT));

app.get('/play.html', (_req, res) => {
  res.sendFile(join(GAME_ROOT, 'stub', 'play.html'));
});

app.get('/health', (_req, res) => {
  res.type('text/plain').send('ok');
});

app.get('/', (_req, res) => {
  res.sendFile(join(GAME_ROOT, 'chess', 'index.html'));
});

const wsOpts = {
  handleProtocols(protocols) {
    if (protocols.has('snake-game-v1')) return 'snake-game-v1';
    return false;
  },
};

const wssPlay = new WebSocketServer({ noServer: true, ...wsOpts });
const wssSpectator = new WebSocketServer({ noServer: true, ...wsOpts });
const wssChess = new WebSocketServer({ noServer: true });

attachChess(wssChess);

async function pumpDownlink(ws, sessionHub) {
  try {
    for (;;) {
      const frame = await sessionHub.downlink.take();
      if (ws.readyState === ws.OPEN) {
        ws.send(frame);
      }
    }
  } catch {
    /* connection closed */
  }
}

async function pumpUplink(ws, sessionHub) {
  try {
    for (;;) {
      const frame = await sessionHub.uplink.take();
      if (ws.readyState === ws.OPEN) {
        ws.send(frame);
      }
    }
  } catch {
    /* connection closed */
  }
}

wssPlay.on('connection', (ws) => {
  let sessionId = null;
  let sessionHub = null;
  let pumpStarted = false;

  ws.on('message', (raw) => {
    const data = Buffer.isBuffer(raw) ? raw : Buffer.from(raw);

    const sidResume = parseResume(data);
    if (sidResume !== null) {
      sessionId = sidResume;
      sessionHub = hub(sessionId);
      if (!pumpStarted) {
        pumpStarted = true;
        pumpDownlink(ws, sessionHub);
      }
      console.log(`player resume session=${sessionId}`);
      ws.send(buildHandshakeAck(sessionId, RELAY_SHARD_ID));
      return;
    }

    const hs = parseHandshake(data);
    if (hs) {
      sessionId = hs.sessionId;
      sessionHub = hub(sessionId);
      sessionHub.players += 1;
      if (!pumpStarted) {
        pumpStarted = true;
        pumpDownlink(ws, sessionHub);
      }
      console.log(`player handshake session=${sessionId} shard=${hs.shardId} relay=${RELAY_SHARD_ID}`);
      ws.send(buildHandshakeAck(sessionId, RELAY_SHARD_ID));
      return;
    }

    if (sessionId === null || !sessionHub) return;
    if (isFragment(data)) {
      sessionHub.uplink.push(data);
    }
  });

  ws.on('close', () => {
    if (sessionHub) {
      sessionHub.players = Math.max(0, sessionHub.players - 1);
    }
  });
});

wssSpectator.on('connection', (ws) => {
  let sessionId = null;
  let sessionHub = null;
  let pumpStarted = false;

  ws.on('message', (raw) => {
    const data = Buffer.isBuffer(raw) ? raw : Buffer.from(raw);

    const agg = parseHandshakeAgg(data);
    if (agg) {
      sessionId = agg.sessionId;
      sessionHub = hub(sessionId);
      sessionHub.spectators += 1;
      if (!pumpStarted) {
        pumpStarted = true;
        pumpUplink(ws, sessionHub);
      }
      console.log(`spectator handshake session=${sessionId} relay=${agg.relayId}`);
      return;
    }

    if (sessionId === null || !sessionHub) return;
    if (isFragment(data)) {
      sessionHub.downlink.push(data);
    }
  });

  ws.on('close', () => {
    if (sessionHub) {
      sessionHub.spectators = Math.max(0, sessionHub.spectators - 1);
    }
  });
});

server.on('upgrade', (req, socket, head) => {
  const { pathname } = new URL(req.url || '/', `http://${req.headers.host || 'localhost'}`);

  if (pathname === '/ws/play') {
    wssPlay.handleUpgrade(req, socket, head, (ws) => {
      wssPlay.emit('connection', ws, req);
    });
    return;
  }
  if (pathname === '/ws/spectator') {
    wssSpectator.handleUpgrade(req, socket, head, (ws) => {
      wssSpectator.emit('connection', ws, req);
    });
    return;
  }
  if (pathname === '/ws/chess') {
    wssChess.handleUpgrade(req, socket, head, (ws) => {
      wssChess.emit('connection', ws, req);
    });
    return;
  }
  socket.destroy();
});

server.listen(PORT, '0.0.0.0', () => {
  console.log(`3wayproxy relay-node shard=${RELAY_SHARD_ID} listening on 0.0.0.0:${PORT}`);
});
