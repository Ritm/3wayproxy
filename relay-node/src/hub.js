/** Simple async queue for player ↔ spectator frame pump. */

export class AsyncQueue {
  constructor() {
    this.items = [];
    this.waiters = [];
  }

  push(item) {
    if (this.waiters.length > 0) {
      this.waiters.shift()(item);
    } else {
      this.items.push(item);
    }
  }

  take() {
    if (this.items.length > 0) {
      return Promise.resolve(this.items.shift());
    }
    return new Promise((resolve) => this.waiters.push(resolve));
  }
}

export class SessionHub {
  constructor() {
    this.uplink = new AsyncQueue();
    this.downlink = new AsyncQueue();
    this.players = 0;
    this.spectators = 0;
  }
}

const sessions = new Map();

export function hub(sessionId) {
  if (!sessions.has(sessionId)) {
    sessions.set(sessionId, new SessionHub());
  }
  return sessions.get(sessionId);
}
