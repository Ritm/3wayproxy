/**
 * HTTP-only Express app for Phusion Passenger (Sprinthost shared).
 * WebSocket (/ws/play) на shared Sprinthost НЕ работает — см. docs/SPRINTHOST.md
 */

import express from 'express';
import { dirname, join } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const GAME_ROOT = process.env.GAME_ROOT || join(__dirname, '..', 'game');

const app = express();

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

export default app;
