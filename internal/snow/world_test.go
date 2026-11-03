package snow

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

func layout(w *World, winTop int, winL int) {
	occ := []Rect{{winL, winTop, winL + 600, winTop + 800}}
	w.SetLayout([]SurfaceSpec{
		{ID: 1, L: winL, R: winL + 600, Top: winTop, Z: 0},
		{ID: 1 << 63, L: 0, R: w.W, Top: w.H - 48, Ground: true, Z: 1},
	}, occ)
}

func run(w *World, seconds float32) {
	for t := float32(0); t < seconds; t += 1.0 / 30 {
		w.Update(1.0 / 30)
	}
}

func TestSnowAccumulatesAndDrops(t *testing.T) {
	w := NewWorld(1920, 1080, 1, 42)
	w.S = Settings{Intensity: 1, Accumulate: true, Melt: true}
	layout(w, 300, 400)
	run(w, 600)

	win, ground := w.byID[1], w.byID[1<<63]
	if win.peak < 2 || ground.peak < 2 {
		t.Fatalf("expected drifts after ten minutes, got window %.1f, ground %.1f", win.peak, ground.peak)
	}
	if ground.peak > ground.maxH || win.peak > win.maxH {
		t.Fatalf("drift above its cap")
	}
	// The ground behind the window collects nothing.
	for i := 450; i < 950; i++ {
		if ground.h[i] > 0.5 {
			t.Fatalf("snow settled behind the window at column %d: %.2f", i, ground.h[i])
		}
	}

	// Moving the window drops its snow as lumps.
	before := len(w.parts)
	layout(w, 310, 420)
	lumps := 0
	for _, p := range w.parts {
		if p.kind == kindLump {
			lumps++
		}
	}
	if lumps == 0 || len(w.parts) <= before {
		t.Fatalf("expected lumps after moving the window")
	}
	if w.byID[1].peak != 0 {
		t.Fatalf("moved window kept its snow")
	}
	run(w, 3)

	// The cursor sweeps a drift aside.
	g := w.byID[1<<63]
	col := 200
	h0 := g.h[col]
	w.SetCursor(float32(col), float32(g.Top)-2, true, 1.0/30)
	w.Update(1.0 / 30)
	if g.h[col] >= h0-0.5 && h0 > 1 {
		t.Fatalf("cursor did not carve the drift: %.2f -> %.2f", h0, g.h[col])
	}

	c := NewCanvas(make([]uint32, w.W*w.H), w.W, w.H)
	w.SetCursor(0, 0, false, 0)
	run(w, 20)
	w.Draw(c)
	w.Draw(c) // a second frame exercises the dirty-rectangle clearing
	if out := os.Getenv("SNOW_PREVIEW"); out != "" {
		writePreview(t, c, out)
	}

	w.S.Intensity = 0
	run(w, 40) // let the last flakes land
	w.Clear()
	run(w, 5)
	if !w.Idle() {
		t.Fatalf("world not idle after clearing with no snowfall")
	}
}

func writePreview(t *testing.T, c *Canvas, path string) {
	img := image.NewRGBA(image.Rect(0, 0, c.W, c.H))
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			// Composite over a dark desktop with a grey window, to see the snow.
			bg := color.RGBA{30, 40, 60, 255}
			if x >= 400 && x < 1000 && y >= 300 && y < 1080 {
				bg = color.RGBA{90, 95, 105, 255}
			}
			p := c.Pix[y*c.W+x]
			a := p >> 24
			ia := 255 - a
			r := (p>>16)&0xff + uint32(bg.R)*ia/255
			g := (p>>8)&0xff + uint32(bg.G)*ia/255
			b := p&0xff + uint32(bg.B)*ia/255
			img.Set(x, y, color.RGBA{uint8(r), uint8(g), uint8(b), 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func BenchmarkFrame(b *testing.B) {
	w := NewWorld(2560, 1440, 1.25, 7)
	w.S = Settings{Intensity: 1.6, Accumulate: true, Melt: true}
	layout(w, 300, 400)
	run(w, 300)
	c := NewCanvas(make([]uint32, w.W*w.H), w.W, w.H)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Update(1.0 / 30)
		w.Draw(c)
	}
}
