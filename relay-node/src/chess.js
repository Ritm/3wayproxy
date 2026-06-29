import { randomBytes } from 'crypto';
import { Chess } from 'chess.js';

const table = {
  waiting: null,
  waitingColor: null,
  white: null,
  black: null,
  board: new Chess(),
};

const sockets = new Set();

function playerId() {
  return randomBytes(4).toString('hex');
}

function statusPublic() {
  if (table.white && table.black) {
    const turn = table.board.turn() === 'w' ? 'white' : 'black';
    return {
      phase: 'playing',
      fen: table.board.fen(),
      turn,
      waiting_color: null,
      is_check: table.board.inCheck(),
      is_game_over: table.board.isGameOver(),
      result: table.board.isGameOver() ? table.board.isCheckmate() ? (table.board.turn() === 'w' ? '0-1' : '1-0') : '1/2-1/2' : null,
    };
  }
  if (table.waiting) {
    return {
      phase: 'waiting',
      fen: new Chess().fen(),
      turn: null,
      waiting_color: table.waitingColor,
      is_check: false,
      is_game_over: false,
      result: null,
    };
  }
  return {
    phase: 'empty',
    fen: new Chess().fen(),
    turn: null,
    waiting_color: null,
    is_check: false,
    is_game_over: false,
    result: null,
  };
}

function playerColor(ws) {
  if (table.white?.ws === ws) return 'white';
  if (table.black?.ws === ws) return 'black';
  return null;
}

function clearPlayer(ws) {
  if (table.waiting?.ws === ws) {
    table.waiting = null;
    table.waitingColor = null;
  }
  if (table.white?.ws === ws) table.white = null;
  if (table.black?.ws === ws) table.black = null;
  if (!table.white || !table.black) {
    table.board = new Chess();
  }
}

function send(ws, obj) {
  if (ws.readyState === ws.OPEN) {
    ws.send(JSON.stringify(obj));
  }
}

function broadcast(obj) {
  const payload = JSON.stringify(obj);
  for (const ws of sockets) {
    if (ws.readyState === ws.OPEN) {
      ws.send(payload);
    }
  }
}

function pushState(extra = {}) {
  broadcast({ type: 'state', ...statusPublic(), ...extra });
}

function handleHost(ws, color) {
  color = String(color).toLowerCase();
  if (color !== 'white' && color !== 'black') {
    send(ws, { type: 'error', message: 'Цвет: white или black' });
    return;
  }
  if (table.white && table.black) {
    send(ws, { type: 'error', message: 'Стол занят — идёт партия' });
    return;
  }
  if (table.waiting) {
    send(ws, { type: 'error', message: 'Уже ждут соперника — нажмите «Присоединиться»' });
    return;
  }
  table.waiting = { ws, playerId: playerId() };
  table.waitingColor = color;
  send(ws, { type: 'hosted', color, player_id: table.waiting.playerId });
  pushState();
}

function handleJoin(ws) {
  if (table.white && table.black) {
    send(ws, { type: 'error', message: 'Партия уже идёт' });
    return;
  }
  if (!table.waiting || !table.waitingColor) {
    send(ws, { type: 'error', message: 'Никто не ждёт — создайте игру и выберите цвет' });
    return;
  }
  const host = table.waiting;
  const hostColor = table.waitingColor;
  const guestColor = hostColor === 'white' ? 'black' : 'white';
  const guest = { ws, playerId: playerId() };
  table.waiting = null;
  table.waitingColor = null;
  table.board = new Chess();
  if (hostColor === 'white') {
    table.white = host;
    table.black = guest;
  } else {
    table.white = guest;
    table.black = host;
  }
  send(host.ws, { type: 'matched', color: hostColor, opponent: 'guest' });
  send(guest.ws, { type: 'matched', color: guestColor, opponent: 'host' });
  pushState();
}

function handleMove(ws, from, to, promotion) {
  const color = playerColor(ws);
  if (!color || !table.white || !table.black) {
    send(ws, { type: 'error', message: 'Вы не в партии' });
    return;
  }
  const turn = table.board.turn() === 'w' ? 'white' : 'black';
  if (turn !== color) {
    send(ws, { type: 'error', message: 'Не ваш ход' });
    return;
  }
  try {
    const move = table.board.move({
      from: from.toLowerCase(),
      to: to.toLowerCase(),
      promotion: promotion || undefined,
    });
    if (!move) {
      send(ws, { type: 'error', message: 'Недопустимый ход' });
      return;
    }
    pushState({ last_move: from + to });
  } catch {
    send(ws, { type: 'error', message: 'Некорректный ход' });
  }
}

function handleResign(ws) {
  const color = playerColor(ws);
  if (!color || !table.white || !table.black) return;
  const result = color === 'white' ? '0-1' : '1-0';
  broadcast({
    type: 'state',
    phase: 'empty',
    fen: new Chess().fen(),
    turn: null,
    waiting_color: null,
    is_check: false,
    is_game_over: true,
    result,
    resigned: color,
  });
  table.white = null;
  table.black = null;
  table.board = new Chess();
}

function handleLeave(ws) {
  clearPlayer(ws);
  pushState({ left: true });
}

export function attachChess(wss) {
  wss.on('connection', (ws) => {
    sockets.add(ws);
    send(ws, { type: 'hello', player_id: playerId() });
    send(ws, { type: 'state', ...statusPublic() });

    ws.on('message', (raw) => {
      let data;
      try {
        data = JSON.parse(raw.toString());
      } catch {
        send(ws, { type: 'error', message: 'Ожидается JSON' });
        return;
      }
      switch (data.type) {
        case 'status':
          send(ws, { type: 'state', ...statusPublic() });
          break;
        case 'host':
          handleHost(ws, data.color);
          break;
        case 'join':
          handleJoin(ws);
          break;
        case 'move':
          handleMove(ws, data.from, data.to, data.promotion);
          break;
        case 'resign':
          handleResign(ws);
          break;
        case 'leave':
          handleLeave(ws);
          break;
        default:
          send(ws, { type: 'error', message: `Неизвестная команда: ${data.type}` });
      }
    });

    ws.on('close', () => {
      sockets.delete(ws);
      clearPlayer(ws);
      pushState();
    });
  });
}
