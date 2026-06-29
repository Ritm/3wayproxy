/** Binary frame helpers — must match relay/app/protocol.py and shared/proto (Go). */

export const TYPE_HANDSHAKE = 0x01;
export const TYPE_HANDSHAKE_ACK = 0x02;
export const TYPE_RESUME = 0x03;
export const TYPE_FRAGMENT = 0x04;
export const TYPE_HANDSHAKE_AGG = 0xa1;

export const PROTO_VER = 1;
export const MTU_HINT = 1200;

export function parseResume(data) {
  if (data.length < 9 || data[0] !== TYPE_RESUME) return null;
  return Number(data.readBigUInt64BE(1));
}

export function parseHandshake(data) {
  if (data.length < 28 || data[0] !== TYPE_HANDSHAKE) return null;
  const shardId = data.readUInt16BE(2);
  const sessionId = Number(data.readBigUInt64BE(4));
  const nonce = data.subarray(12, 28);
  return { shardId, sessionId, nonce };
}

export function buildHandshakeAck(sessionId, relayShardId) {
  const out = Buffer.alloc(13);
  out[0] = TYPE_HANDSHAKE_ACK;
  out[1] = PROTO_VER;
  out.writeBigUInt64BE(BigInt(sessionId), 2);
  out.writeUInt16BE(MTU_HINT, 10);
  out[12] = relayShardId & 0xff;
  return out;
}

export function parseHandshakeAgg(data) {
  if (data.length < 10 || data[0] !== TYPE_HANDSHAKE_AGG) return null;
  const relayId = data[1];
  const sessionId = Number(data.readBigUInt64BE(2));
  return { relayId, sessionId };
}

export function isFragment(data) {
  return data.length >= 19 && data[0] === TYPE_FRAGMENT;
}
