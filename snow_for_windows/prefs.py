"""The user's settings, kept in %APPDATA%\\SnowForWindows\\settings.json."""

from __future__ import annotations

import json
import os
from dataclasses import asdict, dataclass, fields

OFF, LIGHT, STEADY, HEAVY, BLIZZARD, VARIABLE = range(6)


@dataclass
class Prefs:
    intensity: int = STEADY
    accumulate: bool = True
    melt: bool = True
    snow_on_windows: bool = True
    show_in_screenshots: bool = False
    smooth_60fps: bool = False
    over_windows: bool = False  # draw the snow in front of every window instead of behind them
    welcomed: bool = False


def level(intensity: int) -> tuple[float, bool]:
    """How much snow a setting makes, relative to a steady fall, and whether it varies on its own."""
    return {
        LIGHT: (0.45, False),
        STEADY: (1.0, False),
        HEAVY: (1.9, False),
        BLIZZARD: (3.2, False),
        VARIABLE: (2.2, True),
    }.get(intensity, (0.0, False))


def folder() -> str:
    base = os.environ.get("APPDATA") or os.path.expanduser("~/.config")
    return os.path.join(base, "SnowForWindows")


def _path() -> str:
    return os.path.join(folder(), "settings.json")


def load() -> Prefs:
    prefs = Prefs()
    try:
        with open(_path(), encoding="utf-8") as f:
            data = json.load(f)
    except (OSError, ValueError):
        return prefs
    for fd in fields(Prefs):
        value = data.get(fd.name)
        if isinstance(value, type(getattr(prefs, fd.name))):
            setattr(prefs, fd.name, value)
    if not OFF <= prefs.intensity <= VARIABLE:
        prefs.intensity = STEADY
    return prefs


def save(prefs: Prefs) -> None:
    """Write the settings, quietly giving up if the folder can't be written."""
    try:
        os.makedirs(folder(), exist_ok=True)
        tmp = _path() + ".tmp"
        with open(tmp, "w", encoding="utf-8") as f:
            json.dump(asdict(prefs), f, indent=2)
        os.replace(tmp, _path())
    except OSError:
        pass
