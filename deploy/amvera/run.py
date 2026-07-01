"""Точка входа для Amvera: scriptName: run.py"""
from __future__ import annotations

import os
import sys
from pathlib import Path

import uvicorn

ROOT = Path(__file__).resolve().parent


def _locate_app_root() -> Path:
    for base in (ROOT, ROOT / "relay"):
        if (base / "app" / "main.py").is_file():
            return base
    raise SystemExit(
        f"Не найден app/main.py (корень запуска: {ROOT}).\n"
        "Загрузите на Amvera папку deploy/amvera целиком или такую структуру:\n"
        "  run.py\n"
        "  requirements.txt\n"
        "  amvera.yml\n"
        "  app/__init__.py\n"
        "  app/main.py\n"
        "  app/protocol.py\n"
        "  app/chess.py\n"
        "  game/ ..."
    )


if __name__ == "__main__":
    app_root = _locate_app_root()
    root_s = str(app_root)
    if root_s not in sys.path:
        sys.path.insert(0, root_s)

    from app.main import app

    port = int(os.environ.get("PORT", "80"))
    uvicorn.run(app, host="0.0.0.0", port=port)
