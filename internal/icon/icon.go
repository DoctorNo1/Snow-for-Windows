// Package icon draws the app's snowflake icon at any size.
package icon

import (
	"image"
	"image/color"
	"math"
)

type seg struct{ ax, ay, bx, by float64 }

func segments() []seg {
	var s []seg
	for k := 0; k < 6; k++ {
		a := float64(k)*math.Pi/3 - math.Pi/2
		ca, sa := math.Cos(a), math.Sin(a)
		at := func(r, side float64) (float64, float64) {
			// A point r along the arm, offset sideways.
			return ca*r - sa*side, sa*r + ca*side
		}
		x1, y1 := at(0.42, 0)
		s = append(s, seg{0, 0, x1, y1})
		for _, b := range []struct{ at, len float64 }{{0.24, 0.13}, {0.34, 0.08}} {
			bx, by := at(b.at, 0)
			for _, side := range []float64{-1, 1} {
				ex, ey := at(b.at+b.len*0.75, side*b.len)
				s = append(s, seg{bx, by, ex, ey})
			}
		}
	}
	return s
}

func distSeg(px, py float64, s seg) float64 {
	dx, dy := s.bx-s.ax, s.by-s.ay
	t := ((px-s.ax)*dx + (py-s.ay)*dy) / (dx*dx + dy*dy)
	t = math.Max(0, math.Min(1, t))
	x, y := s.ax+dx*t-px, s.ay+dy*t-py
	return math.Hypot(x, y)
}

// Draw renders the snowflake: an icy white flake with a blue rim, so it reads on light and dark taskbars.
func Draw(size int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	segs := segments()
	n := float64(size)
	inner := math.Max(0.026, 0.8/n) // stroke half-widths in icon units
	outer := inner + math.Max(0.016, 0.9/n)
	const ss = 4 // supersampling
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			var aIn, aOut float64
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					px := (float64(x)+(float64(sx)+0.5)/ss)/n - 0.5
					py := (float64(y)+(float64(sy)+0.5)/ss)/n - 0.5
					d := math.Inf(1)
					for _, s := range segs {
						d = math.Min(d, distSeg(px, py, s))
					}
					d = math.Min(d, math.Hypot(px, py)-0.035) // a small hub
					if d < outer {
						aOut++
					}
					if d < inner {
						aIn++
					}
				}
			}
			aIn /= ss * ss
			aOut /= ss * ss
			if aOut == 0 {
				continue
			}
			// Blend white over the blue rim by the inner coverage.
			t := aIn / aOut
			r := 30 + (240-30)*t
			g := 110 + (248-110)*t
			b := 210 + (255-210)*t
			img.SetNRGBA(x, y, color.NRGBA{uint8(r), uint8(g), uint8(b), uint8(aOut*255 + 0.5)})
		}
	}
	return img
}

// BGRA returns the icon as premultiplied 0xAARRGGBB pixels, as Windows bitmaps want them.
func BGRA(size int) []uint32 {
	img := Draw(size)
	out := make([]uint32, size*size)
	for i := range out {
		p := img.Pix[i*4 : i*4+4]
		a := uint32(p[3])
		out[i] = a<<24 | (uint32(p[0])*a/255)<<16 | (uint32(p[1])*a/255)<<8 | uint32(p[2])*a/255
	}
	return out
}
