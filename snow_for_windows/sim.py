"""The snowfall simulation.

It knows nothing about Windows: the app tells it where the ledges are (the ground along each monitor and the tops
of the windows), where the cursor is and how much time passed, and it moves the snow. All the particles live in
numpy arrays so a frame costs a handful of vectorised operations rather than a Python loop per flake.
"""

from __future__ import annotations

import math
from dataclasses import dataclass, field
from typing import NamedTuple

import numpy as np

FLAKE, LUMP, SPRAY, SETTLED = 0, 1, 2, 3
MAX_PARTICLES = 9000
_FIELDS = ("x", "y", "vx", "vy", "r", "a", "phase", "freq", "amp", "fall", "mass", "life")


class Rect(NamedTuple):
    """An axis-aligned rectangle in buffer pixels; r and b are exclusive."""

    l: int
    t: int
    r: int
    b: int

    def contains(self, x: int, y: int) -> bool:
        return self.l <= x < self.r and self.t <= y < self.b


@dataclass
class SurfaceSpec:
    """A ledge snow can rest on: the ground of a monitor or the top edge of a window."""

    id: int
    l: int
    r: int
    top: int
    ground: bool = False
    # Only occluders[:z] can hide this ledge. For the ground it is len(occluders): every window is in front of it.
    z: int = 0


@dataclass
class Settings:
    intensity: float = 1.0  # 0 is no new snow, 1 a steady fall
    variable: bool = False  # let the snowfall rise and ease on its own
    accumulate: bool = True  # flakes build drifts instead of vanishing where they land
    melt: bool = True  # drifts slowly thaw


class Surface:
    def __init__(self, spec: SurfaceSpec, max_h: float):
        self.spec = spec
        n = spec.r - spec.l
        self.h = np.zeros(n, np.float32)  # snow depth per column
        self.vis = np.ones(n, bool)  # whether a window in front hides the column
        self.max_h = max_h
        self.spill_l = 0.0
        self.spill_r = 0.0
        self.peak = 0.0  # deepest column after the last update

    @property
    def l(self) -> int:
        return self.spec.l

    @property
    def r(self) -> int:
        return self.spec.r

    @property
    def top(self) -> int:
        return self.spec.top

    @property
    def ground(self) -> bool:
        return self.spec.ground


@dataclass
class _Pending:
    rows: dict = field(default_factory=lambda: {k: [] for k in _FIELDS + ("kind",)})
    count: int = 0


class World:
    def __init__(self, width: int, height: int, scale: float = 1.0, seed: int | None = None):
        self.w, self.h = width, height
        self.scale = scale if scale > 0 else 1.0
        self.settings = Settings()
        self.rng = np.random.default_rng(seed)
        self.t = 0.0
        self.surfs: list[Surface] = []
        self.by_id: dict[int, Surface] = {}
        self._spawn = 0.0
        self._clear_t = 0.0
        self.cx = self.cy = 0.0
        self.cvx = self.cvy = 0.0
        self.cursor = False
        self._reset_particles()

    # ---- particles -------------------------------------------------------------------------------------------

    def _reset_particles(self) -> None:
        self.p = {k: np.zeros(0, np.float32) for k in _FIELDS}
        self.p["kind"] = np.zeros(0, np.int8)
        self._pending = _Pending()

    @property
    def count(self) -> int:
        return len(self.p["x"])

    def _add(self, kind: int, **values: float) -> None:
        """Queue one particle; queued particles join the arrays at the next flush."""
        if self.count + self._pending.count >= MAX_PARTICLES:
            return
        rows = self._pending.rows
        for k in _FIELDS:
            rows[k].append(values.get(k, 0.0))
        rows["kind"].append(kind)
        self._pending.count += 1

    def _add_many(self, kind: int, n: int, **values) -> None:
        """Queue n particles at once; each value is a scalar or an array of length n."""
        n = min(n, MAX_PARTICLES - self.count - self._pending.count)
        if n <= 0:
            return
        new = {}
        for k in _FIELDS:
            v = np.asarray(values.get(k, 0.0), np.float32)
            new[k] = np.full(n, v, np.float32) if v.ndim == 0 else v[:n]
        new["kind"] = np.full(n, kind, np.int8)
        self._flush()
        for k in self.p:
            self.p[k] = np.concatenate([self.p[k], new[k].astype(self.p[k].dtype)])

    def _flush(self) -> None:
        if not self._pending.count:
            return
        rows = self._pending.rows
        for k in self.p:
            self.p[k] = np.concatenate([self.p[k], np.asarray(rows[k], self.p[k].dtype)])
        self._pending = _Pending()

    def _keep(self, mask: np.ndarray) -> None:
        for k in self.p:
            self.p[k] = self.p[k][mask]

    # ---- layout ----------------------------------------------------------------------------------------------

    def resize(self, width: int, height: int, scale: float) -> None:
        """A new buffer size drops everything, since the layout changes with it."""
        self.w, self.h, self.scale = width, height, scale if scale > 0 else 1.0
        self.surfs, self.by_id = [], {}
        self._reset_particles()

    def set_layout(self, specs: list[SurfaceSpec], occluders: list[Rect]) -> None:
        """Update the ledges. A window that moved, resized sideways or vanished drops its snow in lumps."""
        nxt: list[Surface] = []
        nxt_by_id: dict[int, Surface] = {}
        old_by_id = dict(self.by_id)
        occ = np.array([tuple(o) for o in occluders], np.int64).reshape(-1, 4)
        for sp in specs:
            sp = SurfaceSpec(sp.id, max(sp.l, 0), min(sp.r, self.w), sp.top, sp.ground, sp.z)
            if sp.r - sp.l < 2 or sp.top <= 0 or sp.top > self.h:
                continue
            old = old_by_id.pop(sp.id, None)
            if old is not None and (old.l, old.r, old.top) == (sp.l, sp.r, sp.top):
                s = old
                s.spec.z = sp.z
            else:
                if old is not None and not old.ground:
                    self._drop_pile(old)
                s = Surface(sp, (70 if sp.ground else 26) * self.scale)
            # A column shows when no window in front of the ledge covers the row just above it.
            z = min(sp.z, len(occ))
            cols = np.arange(sp.l, sp.r)
            vis = np.ones(len(cols), bool)
            y = sp.top - 1
            for l, t, r, b in occ[:z]:
                if t <= y < b:
                    vis &= ~((cols >= l) & (cols < r))
            s.vis = vis
            nxt.append(s)
            nxt_by_id[sp.id] = s
        for gone in old_by_id.values():
            if not gone.ground:
                self._drop_pile(gone)
        self.surfs, self.by_id = nxt, nxt_by_id

    def _drop_pile(self, s: Surface) -> None:
        """Turn the snow on a ledge into falling lumps."""
        if s.peak < 0.3 and not s.h.any():
            return
        sc = self.scale
        chunk = max(3, int(5 * sc + 0.5))
        h = np.where(s.vis, s.h, 0)
        for i in range(0, len(h), chunk):
            part = h[i : i + chunk]
            m = float(part.sum())
            if m < 0.6:
                continue
            top = float(part.max())
            pieces = int(top / (6 * sc)) + 1
            pm = m / pieces
            r = min(max(math.sqrt(pm / math.pi) * 1.15, 1.2 * sc), 5.5 * sc)
            for k in range(pieces):
                self._add(
                    LUMP,
                    x=s.l + i + len(part) * 0.5 + self._u(-1, 1),
                    y=s.top - top * (k + 0.5) / pieces,
                    vx=self._u(-25, 25) * sc,
                    vy=self._u(-10, 30) * sc,
                    r=r,
                    a=0.95,
                    mass=pm,
                )
        s.h[:] = 0
        s.peak = 0.0

    def clear(self) -> None:
        """Make the snow on every ledge go away over a moment."""
        self._clear_t = 2.5
        for s in self.surfs:
            if not s.ground:
                self._drop_pile(s)

    def idle(self) -> bool:
        """Nothing falling and nothing left on screen, so the app can stop drawing."""
        if self._intensity() > 0 or self.count or self._pending.count:
            return False
        return all(s.peak <= 0.2 for s in self.surfs)

    # ---- inputs ----------------------------------------------------------------------------------------------

    def set_cursor(self, x: float, y: float, ok: bool, dt: float) -> None:
        if ok and self.cursor and dt > 0:
            lim = 4000 * self.scale
            vx = min(max((x - self.cx) / dt, -lim), lim)
            vy = min(max((y - self.cy) / dt, -lim), lim)
            self.cvx += (vx - self.cvx) * 0.6
            self.cvy += (vy - self.cvy) * 0.6
        else:
            self.cvx = self.cvy = 0.0
        self.cx, self.cy, self.cursor = x, y, ok

    # ---- simulation ------------------------------------------------------------------------------------------

    def _u(self, a: float, b: float) -> float:
        return a + (b - a) * float(self.rng.random())

    def _intensity(self) -> float:
        i = self.settings.intensity
        if self.settings.variable:
            t = self.t
            v = 0.5 + 0.32 * math.sin(t / 700 * 2 * math.pi) + 0.18 * math.sin(t / 263 * 2 * math.pi + 1.3)
            i *= 0.08 + 0.92 * v * v
        return i

    def wind(self) -> float:
        t = self.t
        v = 0.55 * math.sin(t * 0.043) + 0.3 * math.sin(t * 0.117 + 1.1) + 0.15 * math.sin(t * 0.29 + 2.3)
        return v * 16 * self.scale

    def update(self, dt: float) -> None:
        if dt <= 0:
            return
        dt = min(dt, 0.1)
        self.t += dt
        sc = self.scale
        wind = self.wind()

        intensity = self._intensity()
        if intensity > 0 and self.w > 0 and self.h > 0:
            on_screen = intensity * self.w / 1920 * 240
            fall_time = self.h / (46 * sc)
            self._spawn += on_screen / fall_time * dt
            n = int(self._spawn)
            if n:
                self._spawn -= n
                self._spawn_flakes(n, wind)
        self._flush()

        p = self.p
        kind = p["kind"]
        flake, lump, spray = kind == FLAKE, kind == LUMP, kind == SPRAY
        moving = kind != SETTLED

        # Flakes sway around the breeze and ease toward their own falling speed.
        p["phase"][flake] += p["freq"][flake] * dt * 2 * math.pi
        tvx = wind * (0.4 + p["r"] / (3 * sc)) + p["amp"] * np.sin(p["phase"])
        p["vx"] = np.where(flake, p["vx"] + (tvx - p["vx"]) * min(1.0, dt * 1.8), p["vx"])
        p["vy"] = np.where(flake, p["vy"] + (p["fall"] - p["vy"]) * min(1.0, dt * 1.5), p["vy"])
        # Lumps drop, powder drifts down more gently and fades.
        grav = 900 * sc
        p["vy"] = np.where(lump, p["vy"] + grav * dt, p["vy"])
        p["vx"] = np.where(lump, p["vx"] * (1 - min(1.0, dt * 0.6)), p["vx"])
        p["vy"] = np.where(spray, np.minimum(p["vy"] + grav * 0.55 * dt, 260 * sc), p["vy"])
        p["vx"] = np.where(spray, p["vx"] * (1 - min(1.0, dt * 1.6)), p["vx"])
        p["life"] = np.where(spray | (kind == SETTLED), p["life"] - dt, p["life"])

        if self.cursor:
            # The cursor pushes falling snow aside and drags it along its motion.
            reach = 70 * sc
            dx, dy = p["x"] - self.cx, p["y"] - self.cy
            d = np.sqrt(dx * dx + dy * dy)
            near = (flake | spray) & (d < reach) & (d > 0.1)
            if near.any():
                f = np.where(near, 1 - d / reach, 0)
                push = f * f * 1600 * sc * dt
                inv = np.where(near, 1 / np.maximum(d, 0.1), 0)
                p["vx"] += dx * inv * push + self.cvx * f * dt * 3
                p["vy"] += dy * inv * push + self.cvy * f * dt * 3

        p["x"] = np.where(moving, p["x"] + p["vx"] * dt, p["x"])
        p["y"] = np.where(moving, p["y"] + p["vy"] * dt, p["y"])

        self._collide(dt)
        self._carve()
        self._settle(dt)
        self._flush()

        p = self.p
        kind = p["kind"]
        alive = ~(((kind == SPRAY) | (kind == SETTLED)) & (p["life"] <= 0))
        alive &= p["y"] - p["r"] <= self.h + 4
        alive &= (p["x"] > -60 * sc) & (p["x"] < self.w + 60 * sc)
        if not alive.all():
            self._keep(alive)

    def _spawn_flakes(self, n: int, wind: float) -> None:
        sc = self.scale
        rng = self.rng
        u = rng.random(n)
        r = (0.7 + 2.5 * u**3) * sc  # mostly small flakes, a few big ones
        depth = (r / sc - 0.7) / 2.5  # 0 far away, 1 close
        margin = abs(wind) * self.h / (40 * sc)
        x = rng.uniform(-margin, self.w + margin, n) - math.copysign(margin * 0.5, wind)
        fall = (30 + 38 * depth + rng.uniform(-4, 4, n)) * sc
        self._add_many(
            FLAKE,
            n,
            x=x,
            y=-r - rng.uniform(0, 30 * sc, n),
            vy=fall,
            r=r,
            a=0.5 + 0.45 * depth + rng.uniform(-0.05, 0.05, n),
            phase=rng.uniform(0, 2 * math.pi, n),
            freq=rng.uniform(0.15, 0.55, n),
            amp=rng.uniform(4, 16, n) * sc,
            fall=fall,
            mass=math.pi * r * r,
        )

    def _collide(self, dt: float) -> None:
        """Land falling snow on the first visible ledge it crosses."""
        p = self.p
        n = self.count
        if not n or not self.surfs:
            return
        kind = p["kind"]
        moving = (kind != SETTLED) & (p["vy"] > 0)
        bottom = p["y"] + p["r"] * 0.4
        prev = bottom - p["vy"] * dt
        ix = np.floor(p["x"]).astype(np.int64)
        hit_y = np.full(n, np.inf, np.float32)
        hit_s = np.full(n, -1, np.int64)
        for si, s in enumerate(self.surfs):
            idx = np.nonzero(moving & (ix >= s.l) & (ix < s.r))[0]
            if not len(idx):
                continue
            c = ix[idx] - s.l
            sy = s.top - s.h[c]
            ok = s.vis[c] & (prev[idx] <= sy + 0.5) & (bottom[idx] >= sy) & (sy < hit_y[idx])
            sel = idx[ok]
            hit_y[sel] = sy[ok]
            hit_s[sel] = si
        hits = np.nonzero(hit_s >= 0)[0]
        if not len(hits):
            return
        building = self.settings.accumulate and self._clear_t <= 0
        hk = kind[hits]

        # Flakes settle and fade into the drift.
        fl = hits[hk == FLAKE]
        if len(fl):
            if building:
                for si in np.unique(hit_s[fl]):
                    on = fl[hit_s[fl] == si]
                    self._deposit(self.surfs[si], p["x"][on], p["mass"][on] * 1.4)
            life = (1.4 if building else 3.5) + self.rng.random(len(fl)).astype(np.float32)
            kind[fl] = SETTLED
            p["y"][fl] = hit_y[fl] - p["r"][fl] * 0.5
            p["vx"][fl] = 0
            p["vy"][fl] = 0
            p["life"][fl] = life
            p["mass"][fl] = life  # remembered to fade against

        # Powder joins the drift where it lands.
        sp = hits[hk == SPRAY]
        if len(sp):
            if building:
                for si in np.unique(hit_s[sp]):
                    on = sp[hit_s[sp] == si]
                    self._deposit(self.surfs[si], p["x"][on], p["mass"][on])
            p["life"][sp] = 0

        # Lumps burst: part stays, the rest sprays up and lands nearby.
        sc = self.scale
        for i in hits[hk == LUMP]:
            s = self.surfs[hit_s[i]]
            x, mass, r, vx = float(p["x"][i]), float(p["mass"][i]), float(p["r"][i]), float(p["vx"][i])
            keep = mass * 0.55
            if self._clear_t <= 0:
                self._deposit(s, np.array([x], np.float32), np.array([keep], np.float32))
            rest = mass - keep
            k = min(int(rest / 2) + 2, 10)
            for _ in range(k):
                self._spray(x + self._u(-r, r), hit_y[i] - 1, self._u(-90, 90) * sc + vx * 0.3,
                            -self._u(40, 160) * sc, rest / k)
            kind[i] = SPRAY
            p["life"][i] = 0

    def _spray(self, x: float, y: float, vx: float, vy: float, mass: float) -> None:
        r = min(max(math.sqrt(mass / math.pi), 0.6 * self.scale), 1.8 * self.scale)
        self._add(SPRAY, x=x, y=y, vx=vx, vy=vy, r=r, a=0.9, mass=mass, life=3.0)

    def _deposit(self, s: Surface, x: np.ndarray, mass: np.ndarray) -> None:
        """Add snow in small mounds around the given columns of a ledge."""
        c = np.floor(x).astype(np.int64) - s.l
        k = (np.sqrt(mass) * 0.6).astype(np.int64) + 1
        big = int(k.max())
        d = np.arange(-big, big + 1)
        w = np.clip(k[:, None] + 1 - np.abs(d)[None, :], 0, None).astype(np.float32)
        w /= w.sum(axis=1, keepdims=True)
        cols = c[:, None] + d[None, :]
        amount = mass[:, None] * w
        ok = (cols >= 0) & (cols < len(s.h)) & (w > 0)
        cols, amount = cols[ok], amount[ok]
        ok = s.vis[cols]
        np.add.at(s.h, cols[ok], amount[ok])
        np.minimum(s.h, s.max_h, out=s.h)

    def _carve(self) -> None:
        """Let the cursor sweep the drifts aside, throwing the snow it moves into the air."""
        if not self.cursor:
            return
        sc = self.scale
        rad = 16 * sc
        cx, cy = self.cx, self.cy
        for s in self.surfs:
            if cy < s.top - s.max_h - rad or cy > s.top + rad:
                continue
            lo = max(int(cx - rad) - s.l, 0)
            hi = min(int(cx + rad) - s.l, len(s.h) - 1)
            if hi < lo:
                continue
            c = np.arange(lo, hi + 1)
            h = s.h[c]
            dx = s.l + c + 0.5 - cx
            inside = dx * dx < rad * rad
            d = np.sqrt(np.maximum(rad * rad - dx * dx, 0))
            nh = np.maximum(s.top - (cy + d), 0)
            cut = s.vis[c] & (h > 0) & inside & (cy - d <= s.top) & (cy + d >= s.top - h) & (nh < h)
            if not cut.any():
                continue
            cc, removed, left = c[cut], (h - nh)[cut], nh[cut]
            s.h[cc] = left
            # One puff of powder per 2.5 px of snow moved, most of them kept.
            puffs = np.ceil(removed / 2.5).astype(np.int64)
            col = np.repeat(cc, puffs)
            mass = np.repeat(removed / puffs, puffs)
            keep = self.rng.random(len(col)) < 0.6
            col, mass = col[keep], mass[keep]
            if not len(col):
                continue
            side = np.where(s.l + col + 0.5 - cx < 0, -1.0, 1.0)
            n = len(col)
            self._add_many(
                SPRAY,
                n,
                x=s.l + col.astype(np.float32),
                y=s.top - np.repeat(left, puffs)[keep] - 1,
                vx=side * self.rng.uniform(30, 140, n) * sc + self.cvx * 0.5,
                vy=-self.rng.uniform(50, 200, n) * sc + min(self.cvy, 0) * 0.3,
                r=np.clip(np.sqrt(mass / math.pi), 0.6 * sc, 1.8 * sc),
                a=0.9,
                mass=mass,
                life=3.0,
            )

    def _settle(self, dt: float) -> None:
        """Slump steep drifts, round them off, spill snow off window ends, and melt."""
        slope = 0.5
        clearing = self._clear_t > 0
        if clearing:
            self._clear_t -= dt
        melt = self.settings.melt
        sc = self.scale
        k = min(0.2, dt * 2.5)
        for s in self.surfs:
            h, vis = s.h, s.vis
            if len(h) < 2:
                continue
            both = vis[:-1] & vis[1:]
            for _ in range(2):
                d = h[:-1] - h[1:]
                t = np.where(d > slope, (d - slope) * 0.25, np.where(d < -slope, (d + slope) * 0.25, 0)) * both
                h[:-1] -= t
                h[1:] += t
            # A gentle diffusion turns the little peaks single flakes leave into soft drifts.
            three = vis[:-2] & vis[1:-1] & vis[2:]
            h[1:-1] += k * (h[:-2] + h[2:] - 2 * h[1:-1]) * three
            if not s.ground:
                lip = 2.5 * sc
                if vis[0] and h[0] > lip:
                    t = (h[0] - lip) * 0.3
                    h[0] -= t
                    s.spill_l += float(t)
                if vis[-1] and h[-1] > lip:
                    t = (h[-1] - lip) * 0.3
                    h[-1] -= t
                    s.spill_r += float(t)
                for side in ("l", "r"):
                    spill = getattr(s, "spill_" + side)
                    if spill > 5:
                        self._add(
                            LUMP,
                            x=s.l - 1 if side == "l" else s.r,
                            y=s.top - 2,
                            vx=(-15 if side == "l" else 15) * sc,
                            r=min(max(math.sqrt(spill / math.pi), 1.0), 3 * sc),
                            a=0.95,
                            mass=spill,
                        )
                        setattr(s, "spill_" + side, 0.0)
            if clearing:
                h -= h * dt * 4 + dt * 3 * sc
            elif melt:
                # A constant thaw plus a share of the depth, so drifts level off instead of growing forever.
                h -= (0.004 * sc + 0.0022 * h) * dt
            np.maximum(h, 0, out=h)
            s.peak = float(h.max())
