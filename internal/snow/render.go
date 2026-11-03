package snow

import "math"

// Canvas is a premultiplied BGRA buffer (0xAARRGGBB per pixel) that remembers what it drew, so the next frame
// only has to clear those areas instead of the whole screen.
type Canvas struct {
	Pix  []uint32
	W, H int

	dirty []Rect // what the last frame drew
	full  bool
}

// NewCanvas wraps a pixel slice of w*h pixels.
func NewCanvas(pix []uint32, w, h int) *Canvas {
	return &Canvas{Pix: pix, W: w, H: h, full: true}
}

// Invalidate makes the next frame clear the whole buffer.
func (c *Canvas) Invalidate() { c.full = true }

func (c *Canvas) begin() {
	if c.full {
		clear(c.Pix)
		c.full = false
	} else {
		for _, r := range c.dirty {
			for y := r.T; y < r.B; y++ {
				clear(c.Pix[y*c.W+r.L : y*c.W+r.R])
			}
		}
	}
	c.dirty = c.dirty[:0]
}

func (c *Canvas) mark(r Rect) {
	r.L, r.T = maxi(r.L, 0), maxi(r.T, 0)
	r.R, r.B = mini(r.R, c.W), mini(r.B, c.H)
	if r.L < r.R && r.T < r.B {
		c.dirty = append(c.dirty, r)
	}
}

// Draw clears what the last frame drew and draws the world.
func (w *World) Draw(c *Canvas) {
	c.begin()
	for _, s := range w.surfs {
		drawPile(c, s)
	}
	for i := range w.parts {
		p := &w.parts[i]
		a := p.a
		switch p.kind {
		case kindSettled:
			if p.mass > 0 {
				a *= clamp(p.life/minf(p.mass, 1.2), 0, 1)
			}
		case kindSpray:
			a *= clamp(p.life*2, 0, 1)
		}
		drawDisc(c, p.x, p.y, p.r, a)
	}
}

func minf(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func hash(x, y int) uint32 {
	h := uint32(x)*374761393 + uint32(y)*668265263
	h = (h ^ (h >> 13)) * 1274126177
	return h ^ (h >> 16)
}

func pack(a, r, g, b float32) uint32 {
	// Inputs are 0..1, colour not yet premultiplied.
	ai := uint32(a*255 + 0.5)
	return ai<<24 | uint32(r*a*255+0.5)<<16 | uint32(g*a*255+0.5)<<8 | uint32(b*a*255+0.5)
}

// over composites a premultiplied source pixel onto the destination.
func over(dst, src uint32) uint32 {
	sa := src >> 24
	if sa == 255 {
		return src
	}
	ia := 255 - sa
	db := dst & 0xff
	dg := (dst >> 8) & 0xff
	dr := (dst >> 16) & 0xff
	da := dst >> 24
	b := (src & 0xff) + (db*ia+127)/255
	g := ((src >> 8) & 0xff) + (dg*ia+127)/255
	r := ((src >> 16) & 0xff) + (dr*ia+127)/255
	a := sa + (da*ia+127)/255
	return a<<24 | r<<16 | g<<8 | b
}

func drawPile(c *Canvas, s *surface) {
	if s.peak < 0.25 {
		return
	}
	top := s.Top
	if top > c.H {
		top = c.H
	}
	c.mark(Rect{s.L, s.Top - int(s.peak) - 2, s.R, top})
	n := len(s.h)
	for i := 0; i < n; i++ {
		hv := s.h[i]
		if !s.vis[i] || hv < 0.25 {
			continue
		}
		x := s.L + i
		if x < 0 || x >= c.W {
			continue
		}
		// Light from the upper left: a column facing left is brighter, one facing right a little shaded.
		var hl, hr float32 = hv, hv
		if i > 0 {
			hl = s.h[i-1]
		}
		if i < n-1 {
			hr = s.h[i+1]
		}
		slopeLight := clamp((hr-hl)*0.12, -0.1, 0.1)
		ftop := float32(s.Top) - hv
		ytop := int(math.Floor(float64(ftop)))
		cover := 1 - (ftop - float32(ytop)) // partial coverage of the topmost pixel
		for y := ytop; y < top; y++ {
			if y < 0 {
				continue
			}
			depth := float32(y) - ftop
			// Bright crust, cool shadowed body, a faint darker line where it meets the ledge.
			shade := 1 - clamp(depth/(hv+5), 0, 1)*0.13
			if depth < 3 {
				shade += 0.03 - slopeLight
			}
			r, g, b := 0.95*shade, 0.97*shade, 1.0*shade
			if depth < 2.5 && hash(x, y)&511 == 0 {
				r, g, b = 1, 1, 1
			}
			a := float32(0.97)
			if y == ytop {
				a *= cover
			}
			if y == top-1 && hv > 2 {
				r, g, b = r*0.9, g*0.92, b*0.96
			}
			px := &c.Pix[y*c.W+x]
			*px = over(*px, pack(a, clamp(r, 0, 1), clamp(g, 0, 1), clamp(b, 0, 1)))
		}
	}
}

// drawDisc draws a soft round flake centred at (x, y).
func drawDisc(c *Canvas, x, y, r, a float32) {
	if a <= 0.01 {
		return
	}
	ext := r + 1
	x0, x1 := int(math.Floor(float64(x-ext))), int(math.Ceil(float64(x+ext)))
	y0, y1 := int(math.Floor(float64(y-ext))), int(math.Ceil(float64(y+ext)))
	if x1 <= 0 || y1 <= 0 || x0 >= c.W || y0 >= c.H {
		return
	}
	x0, y0 = maxi(x0, 0), maxi(y0, 0)
	x1, y1 = mini(x1, c.W), mini(y1, c.H)
	c.mark(Rect{x0, y0, x1, y1})
	rr := r
	if rr < 0.7 {
		// A tiny flake keeps a pixel's footprint but grows fainter.
		a *= rr / 0.7
		rr = 0.7
	}
	for py := y0; py < y1; py++ {
		dy := float32(py) + 0.5 - y
		row := c.Pix[py*c.W:]
		for px := x0; px < x1; px++ {
			dx := float32(px) + 0.5 - x
			d := float32(math.Sqrt(float64(dx*dx + dy*dy)))
			cov := rr + 0.5 - d
			if cov <= 0 {
				continue
			}
			if cov > 1 {
				cov = 1
			}
			t := d / (rr + 0.5)
			al := a * cov * (1 - 0.35*t*t)
			row[px] = over(row[px], pack(al, 0.97, 0.985, 1))
		}
	}
}
