/**
 * WebSocket carrier for 3wayproxy (phase 3).
 * Runs inside Chromium; native code bridges via window.tunDeliver binding.
 */
(function () {
  'use strict';

  const TYPE_HANDSHAKE = 0x01;
  const TYPE_HANDSHAKE_ACK = 0x02;
  const TYPE_RESUME = 0x03;
  const TYPE_FRAGMENT = 0x04;
  const PROTO_VERSION = 1;
  const SUBPROTO = 'snake-game-v1';

  let ws = null;
  let ready = false;

  function encodeHandshake(shardId, sessionId) {
    const buf = new ArrayBuffer(28);
    const v = new DataView(buf);
    v.setUint8(0, TYPE_HANDSHAKE);
    v.setUint8(1, PROTO_VERSION);
    v.setUint16(2, shardId);
    v.setBigUint64(4, BigInt(sessionId));
    return buf;
  }

  function encodeResume(sessionId, lastSeqSent, lastSeqRecv) {
    const buf = new ArrayBuffer(17);
    const v = new DataView(buf);
    v.setUint8(0, TYPE_RESUME);
    v.setBigUint64(1, BigInt(sessionId));
    v.setUint32(9, lastSeqSent);
    v.setUint32(13, lastSeqRecv);
    return buf;
  }

  function u8FromMessage(data) {
    if (data instanceof ArrayBuffer) return new Uint8Array(data);
    if (ArrayBuffer.isView(data)) return new Uint8Array(data.buffer, data.byteOffset, data.byteLength);
    return new Uint8Array(data);
  }

  function u8ToBase64(u8) {
    let bin = '';
    const chunk = 0x8000;
    for (let i = 0; i < u8.length; i += chunk) {
      bin += String.fromCharCode.apply(null, u8.subarray(i, i + chunk));
    }
    return btoa(bin);
  }

  function deliverToNative(u8) {
    if (typeof window.tunDeliver !== 'function') return;
    window.tunDeliver(u8ToBase64(u8));
  }

  async function connect(wsUrl, shardId, sessionId) {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.close(1000, 'reconnect');
    }
    ready = false;
    ws = new WebSocket(wsUrl, SUBPROTO);
    ws.binaryType = 'arraybuffer';

    await new Promise((resolve, reject) => {
      const t = setTimeout(() => reject(new Error('ws open timeout')), 15000);
      ws.onopen = () => {
        clearTimeout(t);
        resolve();
      };
      ws.onerror = () => {
        clearTimeout(t);
        reject(new Error('ws error'));
      };
    });

    ws.send(encodeHandshake(shardId, sessionId));

    const ack = await new Promise((resolve, reject) => {
      const t = setTimeout(() => reject(new Error('handshake ack timeout')), 10000);
      ws.onmessage = (ev) => {
        clearTimeout(t);
        resolve(u8FromMessage(ev.data));
      };
    });

    if (ack[0] !== TYPE_HANDSHAKE_ACK) {
      throw new Error('expected handshake ack, got 0x' + ack[0].toString(16));
    }

    ws.onmessage = (ev) => {
      const u8 = u8FromMessage(ev.data);
      if (u8[0] === TYPE_FRAGMENT) {
        deliverToNative(u8);
      }
    };

    ready = true;
    document.dispatchEvent(new CustomEvent('carrier-ready'));
  }

  function sendBytes(u8) {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      throw new Error('carrier: ws not open');
    }
    ws.send(u8);
  }

  function sendBase64(b64) {
    const bin = atob(b64);
    const u8 = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i++) u8[i] = bin.charCodeAt(i);
    sendBytes(u8);
  }

  function resume(sessionId, lastSeqSent, lastSeqRecv) {
    const sid = typeof sessionId === 'bigint' ? sessionId : BigInt(sessionId);
    sendBytes(encodeResume(sid, lastSeqSent >>> 0, lastSeqRecv >>> 0));
  }

  window.__carrier = {
    connect,
    sendBytes,
    sendBase64,
    resume,
    isReady: () => ready,
  };
})();
