"""Draws the app's snowflake icon at any size, and writes it as an .ico file."""

from __future__ import annotations

import math
import struct
import zlib

import numpy as np


def _segments() -> np.ndarray:
    segs = []
    for k in range(6):
        ang = k * math.pi / 3 - math.pi / 2
        ca, sa = math.cos(ang), math.sin(ang)

        def at(r: float, side: float) -> tuple[float, float]:
            return ca * r - sa * side, sa * r + ca * side

        segs.append((0.0, 0.0, *at(0.42, 0)))
        for pos, length in ((0.24, 0.13), (0.34, 0.08)):
            bx, by = at(pos, 0)
            for side in (-1, 1):
                segs.append((bx, by, *at(pos + length * 0.75, side * length)))
    return np.array(segs, np.float64)


def draw(size: int) -> np.ndarray:
    """The flake as an (size, size, 4) RGBA uint8 image, not premultiplied: icy white with a blue rim."""
    ss = 4  # supersampling
    n = size * ss
    c = (np.arange(n) + 0.5) / n - 0.5
    px, py = np.meshgrid(c, c)
    d = np.hypot(px, py) - 0.035  # a small hub
    for ax, ay, bx, by in _segments():
        dx, dy = bx - ax, by - ay
        t = np.clip(((px - ax) * dx + (py - ay) * dy) / (dx * dx + dy * dy), 0, 1)
        d = np.minimum(d, np.hypot(ax + dx * t - px, ay + dy * t - py))
    inner = max(0.026, 0.8 / size)
    outer = inner + max(0.016, 0.9 / size)
    a_in = (d < inner).reshape(size, ss, size, ss).mean(axis=(1, 3))
    a_out = (d < outer).reshape(size, ss, size, ss).mean(axis=(1, 3))
    t = np.divide(a_in, a_out, out=np.zeros_like(a_in), where=a_out > 0)
    img = np.zeros((size, size, 4), np.uint8)
    img[..., 0] = 30 + (240 - 30) * t
    img[..., 1] = 110 + (248 - 110) * t
    img[..., 2] = 210 + (255 - 210) * t
    img[..., 3] = np.round(a_out * 255)
    return img


def bgra_premultiplied(size: int) -> np.ndarray:
    """The flake as (size, size) uint32 0xAARRGGBB pixels, premultiplied, as Windows bitmaps want them."""
    img = draw(size).astype(np.uint32)
    a = img[..., 3]
    r, g, b = (img[..., i] * a // 255 for i in range(3))
    return (a << 24) | (r << 16) | (g << 8) | b


def png_bytes(img: np.ndarray) -> bytes:
    """Encode an RGBA image as PNG with nothing but zlib."""
    h, w, _ = img.shape
    raw = b"".join(b"\x00" + img[y].tobytes() for y in range(h))

    def chunk(kind: bytes, data: bytes) -> bytes:
        return struct.pack(">I", len(data)) + kind + data + struct.pack(">I", zlib.crc32(kind + data) & 0xFFFFFFFF)

    return (b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", struct.pack(">IIBBBBB", w, h, 8, 6, 0, 0, 0))
            + chunk(b"IDAT", zlib.compress(raw, 9)) + chunk(b"IEND", b""))


def write_ico(path: str, sizes=(16, 20, 24, 32, 40, 48, 64, 128, 256)) -> None:
    images = [png_bytes(draw(s)) for s in sizes]
    out = struct.pack("<HHH", 0, 1, len(images))
    offset = 6 + 16 * len(images)
    for s, data in zip(sizes, images):
        out += struct.pack("<BBBBHHII", s % 256, s % 256, 0, 0, 1, 32, len(data), offset)
        offset += len(data)
    with open(path, "wb") as f:
        f.write(out + b"".join(images))


if __name__ == "__main__":
    # python -m snow_for_windows.icon  regenerates assets/snowflake.ico and assets/snowflake.png
    import os

    os.makedirs("assets", exist_ok=True)
    write_ico("assets/snowflake.ico")
    with open("assets/snowflake.png", "wb") as f:
        f.write(png_bytes(draw(256)))
