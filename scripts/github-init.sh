#!/usr/bin/env bash
# Создать репозиторий Ritm/3wayproxy на GitHub и отправить main.
# Требуется: gh auth login ИЛИ переменная GH_TOKEN (classic PAT: repo).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

export PATH="${HOME}/.local/bin:${PATH}"

if ! command -v gh >/dev/null 2>&1; then
  echo "Установите gh: sudo apt install gh  (или ~/.local/bin/gh)" >&2
  exit 1
fi

if [[ -n "${GH_TOKEN:-}" ]]; then
  echo "$GH_TOKEN" | gh auth login --with-token
elif ! gh auth status >/dev/null 2>&1; then
  echo "Нет авторизации GitHub. Выполните:" >&2
  echo "  gh auth login --hostname github.com --git-protocol ssh --web" >&2
  echo "или:" >&2
  echo "  export GH_TOKEN=ghp_..." >&2
  echo "  $0" >&2
  exit 1
fi

# Загрузить id_rsa на GitHub, если SSH ещё не работает
if ! ssh -o BatchMode=yes -T git@github.com 2>&1 | grep -qi 'successfully authenticated'; then
  if [[ -f "${HOME}/.ssh/id_rsa.pub" ]]; then
    echo "Добавляем ~/.ssh/id_rsa.pub в аккаунт GitHub..."
    gh ssh-key add "${HOME}/.ssh/id_rsa.pub" -t "user-pc-$(hostname)-3wayproxy" || true
  fi
fi

BRANCH="$(git branch --show-current)"
if [[ "$BRANCH" != "main" ]]; then
  git branch -M main
fi

if git remote get-url origin >/dev/null 2>&1; then
  git remote set-url origin git@github.com:Ritm/3wayproxy.git
else
  git remote add origin git@github.com:Ritm/3wayproxy.git
fi

if gh repo view Ritm/3wayproxy >/dev/null 2>&1; then
  echo "Репозиторий Ritm/3wayproxy уже существует — push."
  git push -u origin main
else
  gh repo create Ritm/3wayproxy \
    --public \
    --description "R&D multipath tunnel: Go client/aggregator, Python relay, binary WebSocket carrier" \
    --source=. \
    --remote=origin \
    --push
fi

echo "Готово: https://github.com/Ritm/3wayproxy"
