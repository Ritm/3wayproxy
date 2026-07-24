#!/usr/bin/env bash
# Отправить на Amvera только relay (плоская структура app/, game/, run.py).
# Не пушьте main monorepo — на Amvera окажется client, aggregator и т.д.
#
# Использование:
#   bash deploy/amvera/push.sh https://git.amvera.ru/ritm/rel3
#   bash deploy/amvera/push.sh amvera-relay0   # если remote уже добавлен
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
TARGET="${1:?URL или имя git remote Amvera (например https://git.amvera.ru/ritm/rel3)}"

STAGING="$(mktemp -d)"
trap 'rm -rf "$STAGING"' EXIT

bash "$ROOT/deploy/amvera/pack.sh" "$STAGING"

cd "$STAGING"
git init -q
git add -A
git commit -q -m "relay deploy $(date -u +%Y-%m-%dT%H:%M:%SZ)"

if git -C "$ROOT" remote get-url "$TARGET" &>/dev/null; then
  git remote add amvera "$(git -C "$ROOT" remote get-url "$TARGET")"
else
  git remote add amvera "$TARGET"
fi

echo "Push relay-only → amvera master (force, перезаписывает monorepo на сервере)..."
git push amvera HEAD:master --force

echo "Готово. Проверьте сборку в панели Amvera и curl .../health"
