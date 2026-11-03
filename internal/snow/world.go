// Package snow holds the snowfall simulation and its software renderer. It knows nothing about Windows: the app
// feeds it the screen layout (the ground along each monitor and the tops of the windows), the cursor and the time,
// and it draws into a premultiplied BGRA pixel buffer.
package snow

import "math"

// Rect is an axis-aligned rectangle in buffer pixels, with R and B exclusive.
type Rect struct{ L, T, R, B int }

// Contains reports whether the point lies inside the rectangle.
func (r Rect) Contains(x, y int) bool { return x >= r.L && x < r.R && y >= r.T && y < r.B }

// SurfaceSpec describes a ledge snow can settle on: the ground of a monitor or the top edge of a window.
type SurfaceSpec struct {
	ID     uint64 // stable identity: a window handle, or a ground id
	L, R   int    // columns covered, R exclusive
	Top    int    // the row the snow rests on
	Ground bool
	// Z is the surface's place among the occluders: only occluders[0:Z] can hide it. For the ground it is
	// len(occluders), as every window is in front of the desktop.
	Z int
}

// Settings are the user's choices that shape the snowfall.
type Settings struct {
	Intensity  float32 // 0 is no new snow, 1 a steady fall
	Variable   bool    // let the snowfall rise and ease on its own
	Accumulate bool    // flakes build drifts rather than vanishing where they land
	Melt       bool    // drifts slowly thaw
}

type kind uint8

const (
	kindFlake   kind = iota // falling snowflake
	kindLump                // a clump of snow dropped from a ledge
	kindSpray               // powder thrown up by the cursor or a landing lump
	kindSettled             // a flake that just landed, fading into the drift
)

type particle struct {
	x, y, vx, vy float32
	r, a         float32
	phase, freq  float32
	amp, fall    float32
	mass, life   float32
	kind         kind
}

type surface struct {
	SurfaceSpec
	h      []float32
	vis    []bool
	maxH   float32
	spillL float32
	spillR float32
	peak   float32 // tallest column after the last update, for the dirty rectangle
}

// World is the whole simulation.
type World struct {
	W, H  int
	Scale float32 // physical pixels per 96-dpi pixel
	S     Settings

	parts   []particle
	surfs   []*surface
	byID    map[uint64]*surface
	rng     uint64
	t       float64
	spawn   float32
	clearT  float32
	cx, cy  float32
	cvx     float32
	cvy     float32
	cursor  bool
	scratch []particle
}

const maxParticles = 9000

// NewWorld makes an empty world of the given buffer size.
func NewWorld(w, h int, scale float32, seed uint64) *World {
	if scale <= 0 {
		scale = 1
	}
	return &World{W: w, H: h, Scale: scale, byID: map[uint64]*surface{}, rng: seed | 1}
}

// Resize changes the buffer size. The drifts are dropped since the layout changes with it.
func (w *World) Resize(width, height int, scale float32) {
	w.W, w.H, w.Scale = width, height, scale
	w.surfs = nil
	w.byID = map[uint64]*surface{}
	w.parts = w.parts[:0]
}

func (w *World) rand() float32 {
	// xorshift64*
	w.rng ^= w.rng >> 12
	w.rng ^= w.rng << 25
	w.rng ^= w.rng >> 27
	return float32((w.rng*2685821657736338717)>>40) / float32(1<<24)
}

func (w *World) between(a, b float32) float32 { return a + (b-a)*w.rand() }

// Idle reports whether nothing is falling and nothing is left on screen, so the app can stop drawing.
func (w *World) Idle() bool {
	if w.effectiveIntensity() > 0 || len(w.parts) > 0 {
		return false
	}
	for _, s := range w.surfs {
		if s.peak > 0.2 {
			return false
		}
	}
	return true
}

// Clear makes all the snow on the ledges vanish over a moment.
func (w *World) Clear() {
	w.clearT = 2.5
	for _, s := range w.surfs {
		if !s.Ground {
			w.dropPile(s)
		}
	}
}

// SetLayout updates the ledges. A window that moved, resized sideways or vanished drops its snow in lumps.
func (w *World) SetLayout(specs []SurfaceSpec, occluders []Rect) {
	next := make([]*surface, 0, len(specs))
	nextByID := make(map[uint64]*surface, len(specs))
	for _, sp := range specs {
		if sp.R > w.W {
			sp.R = w.W
		}
		if sp.L < 0 {
			sp.L = 0
		}
		if sp.R-sp.L < 2 || sp.Top <= 0 || sp.Top > w.H {
			continue
		}
		old := w.byID[sp.ID]
		var s *surface
		if old != nil && old.L == sp.L && old.R == sp.R && old.Top == sp.Top {
			s = old
			s.Z = sp.Z
			delete(w.byID, sp.ID)
		} else {
			if old != nil {
				if !old.Ground {
					w.dropPile(old)
				}
				delete(w.byID, sp.ID)
			}
			n := sp.R - sp.L
			s = &surface{SurfaceSpec: sp, h: make([]float32, n), vis: make([]bool, n)}
			if sp.Ground {
				s.maxH = 70 * w.Scale
			} else {
				s.maxH = 26 * w.Scale
			}
		}
		// A column is visible when no window in front of the surface covers the row just above it.
		y := s.Top - 1
		z := s.Z
		if z > len(occluders) {
			z = len(occluders)
		}
		for i := range s.vis {
			x := s.L + i
			v := true
			for _, o := range occluders[:z] {
				if o.Contains(x, y) {
					v = false
					break
				}
			}
			s.vis[i] = v
		}
		next = append(next, s)
		nextByID[sp.ID] = s
	}
	for _, gone := range w.byID {
		if !gone.Ground {
			w.dropPile(gone)
		}
	}
	w.surfs = next
	w.byID = nextByID
}

// dropPile turns the snow on a ledge into falling lumps.
func (w *World) dropPile(s *surface) {
	chunk := int(5*w.Scale + 0.5)
	if chunk < 3 {
		chunk = 3
	}
	for i := 0; i < len(s.h); i += chunk {
		end := i + chunk
		if end > len(s.h) {
			end = len(s.h)
		}
		var m, top float32
		for j := i; j < end; j++ {
			if !s.vis[j] {
				continue
			}
			m += s.h[j]
			if s.h[j] > top {
				top = s.h[j]
			}
		}
		if m < 0.6 {
			continue
		}
		// One lump per chunk, split vertically when the drift is deep.
		pieces := int(top/(6*w.Scale)) + 1
		for p := 0; p < pieces; p++ {
			pm := m / float32(pieces)
			r := float32(math.Sqrt(float64(pm) / math.Pi))
			r = clamp(r*1.15, 1.2*w.Scale, 5.5*w.Scale)
			w.add(particle{
				kind: kindLump,
				x:    float32(s.L+i) + float32(end-i)*0.5 + w.between(-1, 1),
				y:    float32(s.Top) - top*(float32(p)+0.5)/float32(pieces),
				vx:   w.between(-25, 25) * w.Scale,
				vy:   w.between(-10, 30) * w.Scale,
				r:    r,
				a:    0.95,
				mass: pm,
			})
		}
	}
	for j := range s.h {
		s.h[j] = 0
	}
	s.peak = 0
}

func (w *World) add(p particle) {
	if len(w.parts) < maxParticles {
		w.parts = append(w.parts, p)
	}
}

func (w *World) effectiveIntensity() float32 {
	i := w.S.Intensity
	if w.S.Variable {
		// Slow, irregular swells over tens of minutes.
		t := w.t
		v := 0.5 + 0.32*math.Sin(t/700*2*math.Pi) + 0.18*math.Sin(t/263*2*math.Pi+1.3)
		i *= float32(0.08 + 0.92*v*v)
	}
	return i
}

// SetCursor feeds the cursor position in buffer pixels; ok is false when it is not over the buffer.
func (w *World) SetCursor(x, y float32, ok bool, dt float32) {
	if ok && w.cursor && dt > 0 {
		vx, vy := (x-w.cx)/dt, (y-w.cy)/dt
		lim := 4000 * w.Scale
		vx, vy = clamp(vx, -lim, lim), clamp(vy, -lim, lim)
		w.cvx += (vx - w.cvx) * 0.6
		w.cvy += (vy - w.cvy) * 0.6
	} else {
		w.cvx, w.cvy = 0, 0
	}
	w.cx, w.cy, w.cursor = x, y, ok
}

// Wind is the breeze in pixels per second at the current time.
func (w *World) wind() float32 {
	t := w.t
	v := 0.55*math.Sin(t*0.043) + 0.3*math.Sin(t*0.117+1.1) + 0.15*math.Sin(t*0.29+2.3)
	return float32(v) * 16 * w.Scale
}

// Update advances the simulation by dt seconds.
func (w *World) Update(dt float32) {
	if dt <= 0 {
		return
	}
	if dt > 0.1 {
		dt = 0.1
	}
	w.t += float64(dt)
	sc := w.Scale
	wind := w.wind()

	// New flakes, spread along the top edge.
	if in := w.effectiveIntensity(); in > 0 && w.W > 0 && w.H > 0 {
		onScreen := in * float32(w.W) / 1920 * 240
		fallTime := float32(w.H) / (46 * sc)
		w.spawn += onScreen / fallTime * dt
		for w.spawn >= 1 {
			w.spawn--
			w.spawnFlake(wind)
		}
	}

	grav := 900 * sc
	cr := 70 * sc // cursor's reach on falling snow
	for i := range w.parts {
		p := &w.parts[i]
		switch p.kind {
		case kindFlake:
			p.phase += p.freq * dt * 2 * math.Pi
			tvx := wind*(0.4+p.r/(3*sc)) + p.amp*float32(math.Sin(float64(p.phase)))
			p.vx += (tvx - p.vx) * min32(1, dt*1.8)
			p.vy += (p.fall - p.vy) * min32(1, dt*1.5)
		case kindLump:
			p.vy += grav * dt
			p.vx *= 1 - min32(1, dt*0.6)
		case kindSpray:
			p.vy += grav * 0.55 * dt
			p.vx *= 1 - min32(1, dt*1.6)
			if p.vy > 260*sc {
				p.vy = 260 * sc
			}
			p.life -= dt
		case kindSettled:
			p.life -= dt
			continue
		}
		if w.cursor && p.kind != kindLump {
			dx, dy := p.x-w.cx, p.y-w.cy
			d2 := dx*dx + dy*dy
			if d2 < cr*cr && d2 > 0.01 {
				d := float32(math.Sqrt(float64(d2)))
				f := 1 - d/cr
				push := f * f * 1600 * sc * dt
				p.vx += dx/d*push + w.cvx*f*dt*3
				p.vy += dy/d*push + w.cvy*f*dt*3
			}
		}
		p.x += p.vx * dt
		p.y += p.vy * dt
	}

	w.collide(dt)
	w.carve()
	w.settle(dt)

	// Drop the particles that are gone.
	out := w.parts[:0]
	for _, p := range w.parts {
		if p.kind == kindSettled || p.kind == kindSpray {
			if p.life <= 0 {
				continue
			}
		}
		if p.y-p.r > float32(w.H)+4 || p.x < -60*sc || p.x > float32(w.W)+60*sc {
			continue
		}
		out = append(out, p)
	}
	w.parts = out
}

func (w *World) spawnFlake(wind float32) {
	sc := w.Scale
	u := w.rand()
	r := (0.7 + 2.5*u*u*u) * sc // most flakes small, a few large
	depth := (r/sc - 0.7) / 2.5 // 0 far, 1 near
	margin := absf(wind) * float32(w.H) / (40 * sc)
	x := w.between(-margin, float32(w.W)+margin)
	if wind > 0 {
		x -= margin * 0.5
	} else {
		x += margin * 0.5
	}
	fall := (30 + 38*depth + w.between(-4, 4)) * sc
	w.add(particle{
		kind:  kindFlake,
		x:     x,
		y:     -r - w.between(0, 30*sc),
		vy:    fall,
		r:     r,
		a:     0.5 + 0.45*depth + w.between(-0.05, 0.05),
		phase: w.between(0, 2*math.Pi),
		freq:  w.between(0.15, 0.55),
		amp:   w.between(4, 16) * sc,
		fall:  fall,
		mass:  math.Pi * r * r,
	})
}

// collide lands falling snow on the first visible ledge it crosses.
func (w *World) collide(dt float32) {
	for i := range w.parts {
		p := &w.parts[i]
		if p.kind == kindSettled || p.vy <= 0 {
			continue
		}
		bottom := p.y + p.r*0.4
		prev := bottom - p.vy*dt
		ix := int(p.x)
		var hit *surface
		var hitY float32
		for _, s := range w.surfs {
			if ix < s.L || ix >= s.R {
				continue
			}
			c := ix - s.L
			if !s.vis[c] {
				continue
			}
			sy := float32(s.Top) - s.h[c]
			if prev <= sy+0.5 && bottom >= sy && (hit == nil || sy < hitY) {
				hit, hitY = s, sy
			}
		}
		if hit == nil {
			continue
		}
		switch p.kind {
		case kindFlake:
			life := float32(1.4)
			if w.S.Accumulate && w.clearT <= 0 {
				w.deposit(hit, p.x, p.mass*1.4)
			} else {
				life = 3.5
			}
			p.kind = kindSettled
			p.y = hitY - p.r*0.5
			p.vx, p.vy = 0, 0
			p.life = life + w.between(0, 1)
			p.mass = p.life // fade reference
		case kindLump:
			// Burst into powder: part of the lump stays, the rest sprays and lands nearby.
			keep := p.mass * 0.55
			if w.clearT <= 0 {
				w.deposit(hit, p.x, keep)
			}
			rest := p.mass - keep
			n := int(rest/2) + 2
			if n > 10 {
				n = 10
			}
			for k := 0; k < n; k++ {
				w.spray(p.x+w.between(-p.r, p.r), hitY-1, w.between(-90, 90)*w.Scale+p.vx*0.3,
					-w.between(40, 160)*w.Scale, rest/float32(n))
			}
			p.life, p.kind = 0, kindSpray // removed this frame
		case kindSpray:
			if w.S.Accumulate && w.clearT <= 0 {
				w.deposit(hit, p.x, p.mass)
			}
			p.life = 0
		}
	}
}

func (w *World) spray(x, y, vx, vy, mass float32) {
	r := clamp(float32(math.Sqrt(float64(mass)/math.Pi)), 0.6*w.Scale, 1.8*w.Scale)
	w.add(particle{kind: kindSpray, x: x, y: y, vx: vx, vy: vy, r: r, a: 0.9, mass: mass, life: 3})
}

// deposit adds snow around column x of a ledge, in a small mound.
func (w *World) deposit(s *surface, x, mass float32) {
	c := int(x) - s.L
	k := int(float32(math.Sqrt(float64(mass)))*0.6) + 1
	var total float32
	for d := -k; d <= k; d++ {
		total += float32(k + 1 - absi(d))
	}
	for d := -k; d <= k; d++ {
		j := c + d
		if j < 0 || j >= len(s.h) || !s.vis[j] {
			continue
		}
		s.h[j] += mass * float32(k+1-absi(d)) / total
		if s.h[j] > s.maxH {
			s.h[j] = s.maxH
		}
	}
}

// carve lets the cursor sweep the drifts aside.
func (w *World) carve() {
	if !w.cursor {
		return
	}
	sc := w.Scale
	rad := 16 * sc
	cx, cy := w.cx, w.cy
	for _, s := range w.surfs {
		if cy < float32(s.Top)-s.maxH-rad || cy > float32(s.Top)+rad {
			continue
		}
		lo := int(cx-rad) - s.L
		hi := int(cx+rad) - s.L
		if hi < 0 || lo >= len(s.h) {
			continue
		}
		lo, hi = maxi(lo, 0), mini(hi, len(s.h)-1)
		top := float32(s.Top)
		for c := lo; c <= hi; c++ {
			if !s.vis[c] || s.h[c] <= 0 {
				continue
			}
			dx := float32(s.L+c) + 0.5 - cx
			if dx*dx >= rad*rad {
				continue
			}
			d := float32(math.Sqrt(float64(rad*rad - dx*dx)))
			if cy-d > top || cy+d < top-s.h[c] {
				continue
			}
			nh := top - (cy + d)
			if nh < 0 {
				nh = 0
			}
			if nh >= s.h[c] {
				continue
			}
			removed := s.h[c] - nh
			s.h[c] = nh
			// Throw what was removed into the air, away from the cursor and along its motion.
			side := float32(1)
			if dx < 0 {
				side = -1
			}
			for removed > 0 {
				m := min32(removed, 2.5)
				removed -= m
				if w.rand() < 0.6 {
					w.spray(float32(s.L+c), top-nh-1,
						side*w.between(30, 140)*sc+w.cvx*0.5,
						-w.between(50, 200)*sc+min32(w.cvy, 0)*0.3, m)
				}
			}
		}
	}
}

// settle slumps steep drifts, spills snow off window edges, and melts.
func (w *World) settle(dt float32) {
	const slope = 0.5
	melt := w.S.Melt
	clearing := w.clearT > 0
	if clearing {
		w.clearT -= dt
	}
	sc := w.Scale
	odd := int(w.t*30)&1 == 1
	for _, s := range w.surfs {
		h := s.h
		n := len(h)
		for pass := 0; pass < 2; pass++ {
			if odd != (pass == 1) {
				for i := 0; i < n-1; i++ {
					slump(h, s.vis, i, i+1, slope)
				}
			} else {
				for i := n - 2; i >= 0; i-- {
					slump(h, s.vis, i, i+1, slope)
				}
			}
		}
		// A gentle diffusion rounds the little peaks single flakes leave into soft drifts.
		k := min32(0.2, dt*2.5)
		prevH := float32(0)
		for i := 0; i < n; i++ {
			cur := h[i]
			if i > 0 && i < n-1 && s.vis[i-1] && s.vis[i] && s.vis[i+1] {
				h[i] = cur + k*(prevH+h[i+1]-2*cur)
			}
			prevH = cur
		}
		if !s.Ground && n > 0 {
			// The ends of a window's top: snow above a small lip slides off.
			lip := 2.5 * sc
			if s.vis[0] && h[0] > lip {
				t := (h[0] - lip) * 0.3
				h[0] -= t
				s.spillL += t
			}
			if s.vis[n-1] && h[n-1] > lip {
				t := (h[n-1] - lip) * 0.3
				h[n-1] -= t
				s.spillR += t
			}
			if s.spillL > 5 {
				w.add(particle{kind: kindLump, x: float32(s.L) - 1, y: float32(s.Top) - 2, vx: -15 * sc,
					r: clamp(float32(math.Sqrt(float64(s.spillL)/math.Pi)), 1, 3*sc), a: 0.95, mass: s.spillL})
				s.spillL = 0
			}
			if s.spillR > 5 {
				w.add(particle{kind: kindLump, x: float32(s.R), y: float32(s.Top) - 2, vx: 15 * sc,
					r: clamp(float32(math.Sqrt(float64(s.spillR)/math.Pi)), 1, 3*sc), a: 0.95, mass: s.spillR})
				s.spillR = 0
			}
		}
		var peak float32
		for i := range h {
			v := h[i]
			if clearing {
				v -= v*dt*4 + dt*3*sc
			} else if melt {
				// A constant thaw plus a share of the depth, so drifts level off rather than grow forever.
				v -= (0.004*sc + 0.0022*v) * dt
			}
			if v < 0 {
				v = 0
			}
			h[i] = v
			if v > peak {
				peak = v
			}
		}
		s.peak = peak
	}
}

func slump(h []float32, vis []bool, a, b int, slope float32) {
	if !vis[a] || !vis[b] {
		return
	}
	d := h[a] - h[b]
	if d > slope {
		t := (d - slope) * 0.25
		h[a] -= t
		h[b] += t
	} else if d < -slope {
		t := (-d - slope) * 0.25
		h[b] -= t
		h[a] += t
	}
}

func clamp(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
func min32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
func absf(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
func absi(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
func mini(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func maxi(a, b int) int {
	if a > b {
		return a
	}
	return b
}
