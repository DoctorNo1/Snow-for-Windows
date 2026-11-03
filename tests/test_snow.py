import os
import time
import unittest

import numpy as np

from snow_for_windows.render import Canvas, draw
from snow_for_windows.sim import LUMP, Rect, Settings, SurfaceSpec, World

GROUND = 1 << 63


def layout(w: World, top: int, left: int) -> None:
    occluders = [Rect(left, top, left + 600, top + 800)]
    w.set_layout(
        [
            SurfaceSpec(1, left, left + 600, top, z=0),
            SurfaceSpec(GROUND, 0, w.w, w.h - 48, ground=True, z=1),
        ],
        occluders,
    )


def run(w: World, seconds: float) -> None:
    for _ in range(int(seconds * 30)):
        w.update(1 / 30)


class SnowTest(unittest.TestCase):
    def test_accumulates_drops_and_clears(self):
        w = World(1920, 1080, 1.0, seed=42)
        w.settings = Settings(intensity=1, accumulate=True, melt=True)
        layout(w, 300, 400)
        run(w, 600)

        win, ground = w.by_id[1], w.by_id[GROUND]
        self.assertGreater(win.peak, 2, "no drift on the window after ten minutes")
        self.assertGreater(ground.peak, 2, "no drift on the ground after ten minutes")
        self.assertLessEqual(win.peak, win.max_h + 1e-3)
        self.assertLessEqual(ground.peak, ground.max_h + 1e-3)
        # Nothing settles on the ground hidden behind the window.
        self.assertLess(ground.h[450:950].max(), 0.5)

        # Moving the window drops its snow as lumps.
        layout(w, 310, 420)
        w._flush()
        self.assertGreater(int((w.p["kind"] == LUMP).sum()), 0)
        self.assertEqual(w.by_id[1].peak, 0)
        run(w, 3)

        # The cursor sweeps a drift aside.
        g = w.by_id[GROUND]
        col = 200
        before = float(g.h[col])
        w.set_cursor(col, g.top - 2, True, 1 / 30)
        w.update(1 / 30)
        if before > 1:
            self.assertLess(float(g.h[col]), before - 0.5)

        canvas = Canvas(np.zeros((w.h, w.w), np.uint32))
        w.set_cursor(0, 0, False, 0)
        run(w, 20)
        draw(w, canvas)
        draw(w, canvas)  # a second frame goes through the partial clearing
        self.assertGreater(int((canvas.buf >> 24).max()), 200)
        if os.environ.get("SNOW_PREVIEW"):
            write_preview(canvas, os.environ["SNOW_PREVIEW"])

        w.settings.intensity = 0
        run(w, 40)  # let the last flakes land
        w.clear()
        run(w, 5)
        self.assertTrue(w.idle(), "not idle after clearing with no snowfall")

    def test_partial_clear_matches_full_redraw(self):
        w = World(800, 600, 1.0, seed=3)
        w.settings = Settings(intensity=3, accumulate=True, melt=False)
        w.set_layout([SurfaceSpec(GROUND, 0, 800, 560, ground=True, z=0)], [])
        a = Canvas(np.zeros((600, 800), np.uint32))
        for _ in range(300):
            w.update(1 / 30)
            draw(w, a)
        b = Canvas(np.zeros((600, 800), np.uint32))
        draw(w, b)
        np.testing.assert_array_equal(a.buf, b.buf)

    def test_frame_is_fast(self):
        w = World(2560, 1440, 1.25, seed=7)
        w.settings = Settings(intensity=1.9, accumulate=True, melt=True)
        layout(w, 300, 400)
        run(w, 120)
        canvas = Canvas(np.zeros((w.h, w.w), np.uint32))
        start = time.perf_counter()
        frames = 60
        for _ in range(frames):
            w.update(1 / 30)
            draw(w, canvas)
        ms = (time.perf_counter() - start) / frames * 1000
        print(f"\n{ms:.1f} ms per frame at 2560x1440, heavy snow, {w.count} particles")
        self.assertLess(ms, 33)


def write_preview(c: Canvas, path: str) -> None:
    from PIL import Image

    bg = np.zeros((c.h, c.w, 3), np.float32)
    bg[:] = (30, 40, 60)
    bg[300:, 400:1000] = (90, 95, 105)
    px = c.bytes.astype(np.float32)
    a = px[..., 3:4] / 255
    rgb = px[..., [2, 1, 0]] + bg * (1 - a)
    Image.fromarray(np.clip(rgb, 0, 255).astype(np.uint8)).save(path)


if __name__ == "__main__":
    unittest.main()
