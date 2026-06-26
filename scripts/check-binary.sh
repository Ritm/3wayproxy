#!/usr/bin/env bash
# Проверка бинарника на целевой машине (запускать НА удалённом сервере).
set -euo pipefail
BIN="${1:-./bin/3wayproxy-agg}"

echo "=== uname ==="
uname -a
echo
echo "=== file ==="
file "$BIN" || { echo "Файл не найден: $BIN"; exit 1; }
echo
echo "=== first bytes (должно быть 7f454c46 = ELF) ==="
head -c 4 "$BIN" | xxd
echo
echo "=== execute bit ==="
ls -la "$BIN"
echo

if file "$BIN" | grep -q "ELF.*executable"; then
  echo "OK: это ELF-бинарник. Запуск:"
  echo "  chmod +x $BIN"
  echo "  sudo $BIN --config config/aggregator.dev.yaml"
  echo
  echo "НЕ запускайте: sh $BIN  или  bash $BIN"
else
  echo "ОШИБКА: это НЕ ELF. Скорее всего:"
  echo "  - скопирован не тот файл (скрипт, текст, обрывок)"
  echo "  - бинарник собран под другую ОС"
  echo "  - FTP/scp в текстовом режиме (редко)"
  echo
  echo "Пересоберите на сервере или используйте scripts/build-release.sh на dev-машине"
  exit 1
fi

# Пробная загрузка (без TUN)
if "$BIN" --help 2>/dev/null; then
  :
elif "$BIN" 2>&1 | head -1; then
  :
fi
