"""Binary frame helpers matching shared/proto (Go)."""

from __future__ import annotations

import struct

TYPE_HANDSHAKE = 0x01
TYPE_HANDSHAKE_ACK = 0x02
TYPE_RESUME = 0x03
TYPE_FRAGMENT = 0x04
TYPE_HANDSHAKE_AGG = 0xA1

PROTO_VER = 1
MTU_HINT = 1200


def parse_resume(data: bytes) -> int | None:
    if len(data) < 9 or data[0] != TYPE_RESUME:
        return None
    return struct.unpack_from(">Q", data, 1)[0]


def parse_handshake(data: bytes) -> tuple[int, int, bytes] | None:
    if len(data) < 28 or data[0] != TYPE_HANDSHAKE:
        return None
    shard_id = struct.unpack_from(">H", data, 2)[0]
    session_id = struct.unpack_from(">Q", data, 4)[0]
    nonce = data[12:28]
    return shard_id, session_id, nonce


def build_handshake_ack(session_id: int, relay_shard_id: int) -> bytes:
    return (
        bytes([TYPE_HANDSHAKE_ACK, PROTO_VER])
        + struct.pack(">QH", session_id, MTU_HINT)
        + bytes([relay_shard_id & 0xFF])
    )


def parse_handshake_agg(data: bytes) -> tuple[int, int] | None:
    if len(data) < 10 or data[0] != TYPE_HANDSHAKE_AGG:
        return None
    relay_id = data[1]
    session_id = struct.unpack_from(">Q", data, 2)[0]
    return relay_id, session_id


def is_fragment(data: bytes) -> bool:
    return len(data) >= 19 and data[0] == TYPE_FRAGMENT
