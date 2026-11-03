"""Draws the world into a premultiplied BGRA pixel buffer (one uint32 0xAARRGGBB per pixel).

The canvas remembers what it drew, so each frame clears only those pixels instead of the whole screen.
"""

from __future__ import annotations

import math

import numpy as np

from .sim import SETTLED, SPRAY, World

FLAKE_RGB = np.array([0.97, 0.985, 1.0], np.float32)


class Canvas:
    def __init__(self, pixels: np.ndarray):
        """pixels is an (h, w) uint32 array, usually a view on a Windows bitmap's memory."""
        self.buf = pixels
        self.h, self.w = pixels.shape
        self.flat = pixels.reshape(-1)
        self.bytes = pixels.view(np.uint8).reshape(self.h, self.w, 4)  # B, G, R, A
        self._rects: list[tuple[int, int, int, int]] = []
        self._dots: np.ndarray | None = None
        self._full = True

    def invalidate(self) -> None:
        """Clear the whole buffer on the next frame."""
        self._full = True

    def _begin(self) -> None:
        if self._full:
            self.buf[...] = 0
            self._full = False
        else:
            for t, b, l, r in self._rects:
                self.buf[t:b, l:r] = 0
            if self._dots is not None:
                self.flat[self._dots] = 0
        self._rects = []
        self._dots = None


def draw(world: World, canvas: Canvas) -> None:
    """Clear what the last frame drew and draw the world."""
    canvas._begin()
    for s in world.surfs:
        _draw_pile(canvas, s)
    _draw_particles(canvas, world)


def _hash(x: np.ndarray, y: np.ndarray) -> np.ndarray:
    h = x.astype(np.uint32) * np.uint32(374761393) + y.astype(np.uint32) * np.uint32(668265263)
    h = (h ^ (h >> np.uint32(13))) * np.uint32(1274126177)
    return h ^ (h >> np.uint32(16))


def _draw_pile(c: Canvas, s) -> None:
    if s.peak < 0.25:
        return
    bottom = min(s.top, c.h)
    y0 = max(0, s.top - int(math.ceil(s.peak)) - 1)
    l, r = max(s.l, 0), min(s.r, c.w)
    if y0 >= bottom or l >= r:
        return
    h = s.h[l - s.l : r - s.l]
    vis = s.vis[l - s.l : r - s.l]
    hv = np.where(vis & (h >= 0.25), h, 0).astype(np.float32)

    ys = np.arange(y0, bottom, dtype=np.float32)[:, None]
    ftop = s.top - hv[None, :]
    cover = np.clip(ys + 1 - ftop, 0, 1)  # how much of each pixel row lies under the drift's surface
    depth = ys - ftop
    # Light from the upper left: a slope facing left is brighter, one facing right a little shaded.
    hl = np.concatenate([h[:1], h[:-1]])
    hr = np.concatenate([h[1:], h[-1:]])
    slope_light = np.clip((hr - hl) * 0.12, -0.1, 0.1)[None, :]
    # A bright crust over a cool, slightly shadowed body.
    shade = 1 - np.clip(depth / (hv[None, :] + 5), 0, 1) * 0.13
    shade = shade + np.where(depth < 3, 0.03 - slope_light, 0)
    rgb = shade[..., None] * np.array([0.95, 0.97, 1.0], np.float32)
    # A few glints near the surface.
    xs = np.arange(l, r)[None, :]
    glint = (depth < 2.5) & ((_hash(np.broadcast_to(xs, depth.shape), ys.astype(np.int64) + 0 * xs) & 511) == 0)
    rgb[glint] = 1.0
    # A faint darker line where the snow meets the ledge.
    if bottom == s.top:
        rgb[-1] *= np.where(hv > 2, 1, 0)[:, None] * np.array([-0.1, -0.08, -0.04], np.float32) + 1
    np.clip(rgb, 0, 1, out=rgb)
    alpha = (0.97 * cover * (hv[None, :] > 0)).astype(np.float32)

    region = c.bytes[y0:bottom, l:r]
    dst = region.astype(np.float32)
    inv = 1 - alpha
    out = np.empty_like(dst)
    out[..., 0] = rgb[..., 2] * alpha * 255 + dst[..., 0] * inv
    out[..., 1] = rgb[..., 1] * alpha * 255 + dst[..., 1] * inv
    out[..., 2] = rgb[..., 0] * alpha * 255 + dst[..., 2] * inv
    out[..., 3] = alpha * 255 + dst[..., 3] * inv
    region[...] = np.clip(out + 0.5, 0, 255).astype(np.uint8)
    c._rects.append((y0, bottom, l, r))


def _draw_particles(c: Canvas, world: World) -> None:
    p = world.p
    n = world.count
    if not n:
        return
    a = p["a"].copy()
    kind = p["kind"]
    settled = kind == SETTLED
    a[settled] *= np.clip(p["life"][settled] / np.minimum(np.maximum(p["mass"][settled], 1e-3), 1.2), 0, 1)
    spray = kind == SPRAY
    a[spray] *= np.clip(p["life"][spray] * 2, 0, 1)
    r = p["r"]
    # A tiny flake keeps a pixel's footprint but grows fainter.
    a = np.where(r < 0.7, a * r / 0.7, a)
    r = np.maximum(r, 0.7)
    x, y = p["x"], p["y"]
    show = (a > 0.01) & (x > -10) & (x < c.w + 10) & (y > -10) & (y < c.h + 10)
    if not show.any():
        return
    a, r, x, y = a[show], r[show], x[show], y[show]
    ext = np.ceil(r + 1).astype(np.int64)

    idx_parts, val_parts = [], []
    for e in np.unique(ext):
        sel = ext == e
        xs, ys, rs, als = x[sel], y[sel], r[sel], a[sel]
        size = 2 * int(e) + 1
        o = np.arange(size)
        px = np.floor(xs - e).astype(np.int64)[:, None] + o  # (k, size)
        py = np.floor(ys - e).astype(np.int64)[:, None] + o
        dx = px + 0.5 - xs[:, None]
        dy = py + 0.5 - ys[:, None]
        d = np.sqrt(dy[:, :, None] ** 2 + dx[:, None, :] ** 2)  # (k, row, col)
        rr = rs[:, None, None]
        cover = np.clip(rr + 0.5 - d, 0, 1)
        t = d / (rr + 0.5)
        val = als[:, None, None] * cover * (1 - 0.35 * t * t)
        inside = (cover > 0) & ((px >= 0) & (px < c.w))[:, None, :] & ((py >= 0) & (py < c.h))[:, :, None]
        flat = py[:, :, None] * c.w + px[:, None, :]
        idx_parts.append(flat[inside])
        val_parts.append(val[inside])
    idx = np.concatenate(idx_parts)
    if not len(idx):
        return
    val = np.concatenate(val_parts).astype(np.float32)
    # Sum the flakes that share a pixel, then composite once.
    order = np.argsort(idx, kind="stable")
    idx, val = idx[order], val[order]
    uniq, start = np.unique(idx, return_index=True)
    alpha = np.minimum(np.add.reduceat(val, start), 1.0)

    dst = c.flat[uniq]
    inv = 1 - alpha
    b = (dst & 0xFF).astype(np.float32)
    g = ((dst >> 8) & 0xFF).astype(np.float32)
    rr = ((dst >> 16) & 0xFF).astype(np.float32)
    aa = (dst >> 24).astype(np.float32)
    ob = np.clip(FLAKE_RGB[2] * alpha * 255 + b * inv + 0.5, 0, 255).astype(np.uint32)
    og = np.clip(FLAKE_RGB[1] * alpha * 255 + g * inv + 0.5, 0, 255).astype(np.uint32)
    orr = np.clip(FLAKE_RGB[0] * alpha * 255 + rr * inv + 0.5, 0, 255).astype(np.uint32)
    oa = np.clip(alpha * 255 + aa * inv + 0.5, 0, 255).astype(np.uint32)
    c.flat[uniq] = (oa << 24) | (orr << 16) | (og << 8) | ob
    c._dots = uniq
