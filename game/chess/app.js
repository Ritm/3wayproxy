(function () {
  'use strict';

  const PIECES = {
    p: '♟', r: '♜', n: '♞', b: '♝', q: '♛', k: '♚',
    P: '♙', R: '♖', N: '♘', B: '♗', Q: '♕', K: '♔',
  };

  let ws = null;
  let myColor = null;
  let chess = new Chess();
  let selected = null;
  let pendingPromotion = null;
  let lastState = { phase: 'empty', fen: 'start' };

  const $ = (id) => document.getElementById(id);

  function wsUrl() {
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    return proto + '//' + location.host + '/ws/chess';
  }

  function send(obj) {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(obj));
    }
  }

  function setStatus(text) {
    $('status-line').textContent = text;
  }

  function showLobby(state) {
    $('game').classList.add('hidden');
    $('lobby').classList.remove('hidden');
    $('lobby-empty').classList.add('hidden');
    $('lobby-waiting').classList.add('hidden');
    $('lobby-join').classList.add('hidden');

    if (state.phase === 'empty') {
      $('lobby-empty').classList.remove('hidden');
      setStatus('Стол свободен — создайте игру или подождите соперника.');
      if (!state.resigned) myColor = null;
    } else if (state.phase === 'waiting') {
      if (myColor) {
        $('lobby-waiting').classList.remove('hidden');
        $('my-color-label').textContent = myColor === 'white' ? 'белых' : 'чёрных';
        setStatus('Ждём второго игрока…');
      } else {
        $('lobby-join').classList.remove('hidden');
        const c = state.waiting_color === 'white' ? 'белые' : 'чёрные';
        $('waiting-color-label').textContent = c;
        setStatus('Можно присоединиться к игре.');
      }
    }
  }

  function showGame(state) {
    $('lobby').classList.add('hidden');
    $('game').classList.remove('hidden');
    chess.load(state.fen);
    renderBoard();
    const turn = state.turn === 'white' ? 'белых' : 'чёрных';
    $('game-role').textContent = myColor
      ? 'Вы: ' + (myColor === 'white' ? 'белые' : 'чёрные')
      : '';
    $('game-turn').textContent = state.is_game_over
      ? 'Партия окончена'
      : 'Ход: ' + turn;
    if (state.is_game_over && state.result) {
      $('game-message').textContent = 'Результат: ' + state.result;
    } else if (state.is_check) {
      $('game-message').textContent = 'Шах!';
    } else {
      $('game-message').textContent = '';
    }
  }

  function squareName(row, col) {
    return 'abcdefgh'[col] + (8 - row);
  }

  function renderBoard() {
    const board = $('board');
    board.innerHTML = '';
    const legalTargets = new Set();
    if (selected && myColor && lastState.turn === myColor && !lastState.is_game_over) {
      chess.moves({ square: selected, verbose: true }).forEach((m) => legalTargets.add(m.to));
    }
    for (let r = 0; r < 8; r++) {
      for (let c = 0; c < 8; c++) {
        const sq = squareName(r, c);
        const piece = chess.get(sq);
        const div = document.createElement('div');
        const light = (r + c) % 2 === 0;
        div.className = 'square ' + (light ? 'light' : 'dark');
        div.dataset.sq = sq;
        if (piece) {
          const ch = piece.type;
          div.textContent = PIECES[piece.color === 'w' ? ch.toUpperCase() : ch] || '';
        }
        if (selected === sq) div.classList.add('selected');
        if (legalTargets.has(sq)) div.classList.add('legal');
        div.addEventListener('click', () => onSquareClick(sq));
        board.appendChild(div);
      }
    }
  }

  function onSquareClick(sq) {
    const state = lastState;
    if (!myColor || state.phase !== 'playing' || state.turn !== myColor || state.is_game_over) {
      return;
    }

    const piece = chess.get(sq);
    if (!selected) {
      if (piece && piece.color === myColor[0]) {
        selected = sq;
        renderBoard();
      }
      return;
    }

    if (sq === selected) {
      selected = null;
      renderBoard();
      return;
    }

    const moves = chess.moves({ square: selected, verbose: true });
    const move = moves.find((m) => m.to === sq);
    if (!move) {
      if (piece && piece.color === myColor[0]) {
        selected = sq;
      } else {
        selected = null;
      }
      renderBoard();
      return;
    }

    if (move.flags.includes('p')) {
      pendingPromotion = { from: selected, to: sq };
      $('promo-bar').classList.remove('hidden');
      return;
    }

    send({ type: 'move', from: selected, to: sq });
    selected = null;
    renderBoard();
  }

  function completePromotion(p) {
    if (!pendingPromotion) return;
    send({
      type: 'move',
      from: pendingPromotion.from,
      to: pendingPromotion.to,
      promotion: p,
    });
    pendingPromotion = null;
    $('promo-bar').classList.add('hidden');
    selected = null;
  }

  function onState(state) {
    lastState = state;
    if (state.phase === 'playing') {
      showGame(state);
    } else {
      showLobby(state);
      if (state.is_game_over && state.result) {
        setStatus('Партия завершена: ' + state.result);
      }
    }
  }

  function connect() {
    ws = new WebSocket(wsUrl());
    ws.onopen = () => setStatus('Соединение установлено');
    ws.onclose = () => {
      setStatus('Соединение потеряно. Переподключение…');
      setTimeout(connect, 2000);
    };
    ws.onmessage = (ev) => {
      const msg = JSON.parse(ev.data);
      if (msg.type === 'hosted') {
        myColor = msg.color;
      }
      if (msg.type === 'matched') {
        myColor = msg.color;
        selected = null;
      }
      if (msg.type === 'state') {
        onState(msg);
      }
      if (msg.type === 'error') {
        setStatus(msg.message || 'Ошибка');
      }
    };
  }

  $('btn-host-white').addEventListener('click', () => send({ type: 'host', color: 'white' }));
  $('btn-host-black').addEventListener('click', () => send({ type: 'host', color: 'black' }));
  $('btn-join').addEventListener('click', () => send({ type: 'join' }));
  $('btn-cancel-wait').addEventListener('click', () => {
    send({ type: 'leave' });
    myColor = null;
  });
  $('btn-resign').addEventListener('click', () => send({ type: 'resign' }));
  $('btn-leave').addEventListener('click', () => {
    send({ type: 'leave' });
    myColor = null;
    send({ type: 'status' });
  });

  document.querySelectorAll('#promo-bar button').forEach((btn) => {
    btn.addEventListener('click', () => completePromotion(btn.dataset.p));
  });

  connect();
})();
