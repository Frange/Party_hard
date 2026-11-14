package main

import (
	"image/color"
	"math"
)

// The second layer of wear, painted AFTER the page is laid out, so it knows where every text is: big rips along the four
// edges (they stop before any text), stains in the empty bottom of the page and a few faint ones in the middle, and drops
// of blood (more of them when the party had victims). Everything is a function of the seed.
//
// Text stays readable: the pixels of a letter are never touched, the background of a text box is only tinted while it
// keeps most of its contrast with the ink (paperWearKeep), and the banner and the paper's name are never touched.

const paperWearKeep = 70 // % of the ink contrast a text background keeps under a stain

type pwear struct {
	c    *pcv
	s    uint64
	n    int     // pseudo-random counter
	grid []int16 // per pixel: -1 free, -2 protected (banner, title), >= 0 the text box it belongs to
	near []bool  // per pixel: a text (or the banner) within 3 px: rips stop before it
	base []byte  // the page before the wear: contrast is always measured against it (stains may overlap)
}

func (w *pwear) rnd(k int) int {
	w.n++
	return int(paperHash(w.n, 101, w.s) % uint64(k))
}

// wear paints the second layer; kills > 0 brings blood.
func (c *pcv) wear(seed int64, kills int) {
	w := &pwear{c: c, s: uint64(seed)*0x9e3779b1 + 777}
	w.base = append([]byte(nil), c.im.Pix...)
	w.grid = make([]int16, screenW*screenH)
	w.near = make([]bool, screenW*screenH)
	for i := range w.grid {
		w.grid[i] = -1
	}
	mark := func(x0, y0, x1, y1 int, v int16, pad int) {
		for y := y0 - pad; y < y1+pad; y++ {
			for x := x0 - pad; x < x1+pad; x++ {
				if x < 0 || y < 0 || x >= screenW || y >= screenH {
					continue
				}
				if pad == 3 {
					w.near[y*screenW+x] = true
				} else if w.grid[y*screenW+x] != -2 {
					w.grid[y*screenW+x] = v
				}
			}
		}
	}
	for i, t := range c.lay.texts {
		v := int16(i)
		if t.title || t.dark {
			v = -2
		}
		mark(t.x, t.y, t.x+t.w, t.y+t.h, v, 1)
		mark(t.x, t.y, t.x+t.w, t.y+t.h, 0, 3)
	}
	b := c.lay.banner
	mark(b.x, b.y, b.x+b.w, b.y+b.h, -2, 1)
	mark(b.x, b.y, b.x+b.w, b.y+b.h, 0, 3)

	w.rips()
	w.bottomStains()
	w.middleStains()
	w.blood(kills)
}

// paint tints one pixel, keeping every text readable (see paperWearKeep).
func (w *pwear) paint(x, y int, hex uint32, a int) {
	if x < 0 || y < 0 || x >= screenW || y >= screenH || a <= 0 {
		return
	}
	g := w.grid[y*screenW+x]
	if g == -2 {
		return
	}
	if g < 0 {
		if pinkDist(w.c.im.RGBAAt(x, y), paperDesk) >= 36 { // the desk seen through a tear stays clean
			w.c.tint(x, y, hex, a)
		}
		return
	}
	inks := [3]color.RGBA{w.c.lay.texts[g].col, paperInk, paperRed}
	i := w.c.im.PixOffset(x, y)
	b := color.RGBA{w.base[i], w.base[i+1], w.base[i+2], 255}
	var floor [3]int
	for k, ink := range inks {
		if b == ink {
			return // a letter: never touched
		}
		floor[k] = maxI(140, pinkDist(b, ink)*paperWearKeep/100)
	}
	p := w.c.im.RGBAAt(x, y)
	t := color.RGBA{uint8(hex >> 16), uint8(hex >> 8), uint8(hex), 255}
	for ; a > 6; a = a * 2 / 3 {
		n := color.RGBA{
			pclamp((int(p.R)*(255-a) + int(t.R)*a) / 255),
			pclamp((int(p.G)*(255-a) + int(t.G)*a) / 255),
			pclamp((int(p.B)*(255-a) + int(t.B)*a) / 255), 255}
		ok := true
		for k, ink := range inks {
			ok = ok && pinkDist(n, ink) >= floor[k]
		}
		if ok {
			w.c.put(x, y, n)
			return
		}
	}
}

func pinkDist(a, b color.RGBA) int {
	return absI(int(a.R)-int(b.R)) + absI(int(a.G)-int(b.G)) + absI(int(a.B)-int(b.B))
}

// ---- rips -----------------------------------------------------------------------------------------------------------

// rips: big lopsided bites out of the four edges, as deep as the free space allows (never into a text), with light torn
// fibres along the cut and the page's shadow on the desk.
func (w *pwear) rips() {
	// e: 0 top, 1 bottom, 2 left, 3 right
	plan := []struct {
		e, count, min, max int
	}{{0, 3 + w.rnd(3), 12, 26}, {1, 3 + w.rnd(2), 12, 30}, {2, 2 + w.rnd(2), 9, 20}, {3, 2 + w.rnd(2), 9, 20}}
	for _, p := range plan {
		long := screenW
		if p.e >= 2 {
			long = screenH
		}
		for i := 0; i < p.count; i++ {
			wd := 22 + w.rnd(40)
			pos := 30 + w.rnd(long-60-wd)
			if p.e == 0 && pos+wd > screenW-120 { // the burn and the clipped corner live up there
				pos = w.rnd(screenW - 140 - wd)
			}
			if p.e == 2 && pos+wd > screenH-24 { // the dog-ear
				pos = screenH - 24 - wd - w.rnd(40)
			}
			w.rip(p.e, pos, wd, p.min+w.rnd(p.max-p.min+1))
		}
	}
}

// xy maps (along, depth) on edge e to a pixel.
func ripXY(e, a, d int) (int, int) {
	switch e {
	case 0:
		return a, d
	case 1:
		return a, screenH - 1 - d
	case 2:
		return d, a
	}
	return screenW - 1 - d, a
}

func (w *pwear) rip(e, pos, wd, want int) {
	// how deep it may go: it stops 1 px before anything near a text
	room := want
	for a := pos - 2; a < pos+wd+2; a++ {
		for d := 0; d < room; d++ {
			x, y := ripXY(e, a, d)
			if x < 0 || y < 0 || x >= screenW || y >= screenH {
				break
			}
			if w.near[y*screenW+x] {
				room = d - 1
				break
			}
		}
	}
	if room < 6 {
		return
	}
	// profile: a lopsided wedge with a jagged, fibrous edge
	peak := 0.3 + float64(w.rnd(40))/100
	jag := 0.0
	for k := 0; k < wd; k++ {
		f := float64(k) / float64(wd)
		prof := f / peak
		if f > peak {
			prof = (1 - f) / (1 - peak)
		}
		prof = math.Pow(prof, 0.8)
		jag += (float64(paperHash(pos+k, 41+e, w.s)&7) - 3.5) * 0.45
		jag = math.Max(-2.5, math.Min(2.5, jag))
		d := int(float64(room)*prof + jag)
		if d < 0 {
			d = 0
		}
		if d > room {
			d = room
		}
		for dd := 0; dd < d; dd++ {
			x, y := ripXY(e, pos+k, dd)
			w.c.deskPx(x, y, w.s)
			if dd >= d-3 { // the page throws a little shadow on the desk
				w.c.shade(x, y, 1, 1, -8)
			}
		}
		// torn fibres: a white, uneven edge, then a slightly darker crumple
		x, y := ripXY(e, pos+k, d)
		w.c.shade(x, y, 1, 1, 26)
		if paperHash(k, 43, w.s)%3 != 0 {
			x, y = ripXY(e, pos+k, d+1)
			w.c.shade(x, y, 1, 1, 14)
		}
		x, y = ripXY(e, pos+k, d+2)
		w.c.shade(x, y, 1, 1, -10)
	}
}

// ---- stains ---------------------------------------------------------------------------------------------------------

// spot picks, among a few candidates in the box, the centre whose disc covers the fewest text pixels.
func (w *pwear) spot(x0, y0, x1, y1, r int) (int, int) {
	bx, by, best := x0, y0, 1<<30
	for i := 0; i < 8; i++ {
		x, y := x0+w.rnd(x1-x0+1), y0+w.rnd(y1-y0+1)
		hit := 0
		for yy := y - r; yy <= y+r; yy += 2 {
			for xx := x - r; xx <= x+r; xx += 2 {
				if xx >= 0 && yy >= 0 && xx < screenW && yy < screenH && w.grid[yy*screenW+xx] != -1 {
					hit++
				}
			}
		}
		if hit < best {
			bx, by, best = x, y, hit
		}
	}
	return bx, by
}

// ring: a coffee cup ring with a soft inside and a little drip.
func (w *pwear) ring(cx, cy, r, a int) {
	for y := -r - 4; y <= r+4; y++ {
		for x := -r - 4; x <= r+4; x++ {
			dd := math.Sqrt(float64(x*x + y*y))
			wob := float64(paperHash(x*5, y*5, w.s+51)&3) * 0.35
			switch {
			case math.Abs(dd-float64(r)) < 1.5+wob:
				w.paint(cx+x, cy+y, 0x6a4420, a)
			case dd < float64(r):
				w.paint(cx+x, cy+y, 0x8a6a40, a/5)
			}
		}
	}
}

// blot: an irregular spot (ink, water, wine) with a darker rim.
func (w *pwear) blot(cx, cy, r int, hex uint32, a int) {
	k := w.rnd(1000)
	for y := -r - 3; y <= r+3; y++ {
		for x := -r - 3; x <= r+3; x++ {
			ang := math.Atan2(float64(y), float64(x))
			edge := float64(r) * (1 + 0.22*math.Sin(3*ang+float64(k)) + 0.12*math.Sin(5*ang+float64(k/3)))
			dd := math.Sqrt(float64(x*x + y*y))
			switch {
			case dd < edge-1.5:
				w.paint(cx+x, cy+y, hex, a*3/4)
			case dd < edge:
				w.paint(cx+x, cy+y, hex, a)
			}
		}
	}
}

// rub: a greyish smudge, like a thumb dragged across the print.
func (w *pwear) rub(cx, cy, l int) {
	ang := float64(w.rnd(60)-30) / 100
	for k := -l / 2; k < l/2; k++ {
		f := 1 - math.Abs(float64(k))/float64(l/2)
		h := 3 + int(5*f)
		for j := -h; j <= h; j++ {
			x := cx + int(float64(k)*math.Cos(ang)-float64(j)*math.Sin(ang))
			y := cy + int(float64(k)*math.Sin(ang)+float64(j)*math.Cos(ang))
			a := int(62 * f * (1 - math.Abs(float64(j))/float64(h+1)))
			if paperHash(x, y, w.s+61)&3 == 0 {
				a /= 2
			}
			w.paint(x, y, 0x3a3226, a)
		}
	}
}

// bottomStains: the strip at the bottom of the page, around the footer, is mostly empty: two or three stains there.
func (w *pwear) bottomStains() {
	n := 2 + w.rnd(2)
	for i := 0; i < n; i++ {
		switch (i + w.rnd(3)) % 3 {
		case 0:
			x, y := w.spot(30, 502, screenW-30, screenH-14, 16)
			w.ring(x, y, 12+w.rnd(8), 95)
		case 1:
			x, y := w.spot(30, 500, screenW-30, screenH-12, 8)
			w.blot(x, y, 4+w.rnd(5), 0x1e1a24, 150) // an ink blot
		default:
			x, y := w.spot(40, 502, screenW-40, screenH-12, 14)
			w.rub(x, y, 34+w.rnd(30))
		}
	}
}

// middleStains: one or two faint ones on the body; the text over them keeps its contrast.
func (w *pwear) middleStains() {
	n := 1 + w.rnd(2)
	for i := 0; i < n; i++ {
		x, y := w.spot(60, pbY0+20, screenW-60, pbY1-20, 20)
		if w.rnd(2) == 0 {
			w.ring(x, y, 16+w.rnd(10), 56)
		} else {
			w.blot(x, y, 10+w.rnd(8), 0x9a7a3c, 42) // a water mark
		}
	}
}

// ---- blood ----------------------------------------------------------------------------------------------------------

// blood: drops and splashes, dark red, dried at the rim; the more victims, the more and the bigger they are. None
// without victims except, sometimes, a red wine drop. After a massacre the hands that picked the paper up were full of
// blood too (hands).
func (w *pwear) blood(kills int) {
	n := 0
	switch {
	case kills <= 0:
		n = w.rnd(2)
	case kills < 6:
		n = 2 + w.rnd(2)
	case kills < 15:
		n = 4 + w.rnd(2)
	default:
		n = 5 + w.rnd(3) + minI(kills/10, 4)
	}
	for i := 0; i < n; i++ {
		var x, y int
		switch w.rnd(3) {
		case 0: // the bottom strip
			x, y = w.spot(30, 498, screenW-30, screenH-14, 12)
		case 1: // a side margin
			if w.rnd(2) == 0 {
				x, y = w.spot(4, 30, 10, screenH-40, 8)
			} else {
				x, y = w.spot(screenW-11, 40, screenW-5, screenH-40, 8)
			}
		default: // anywhere on the page (over text it stays a pale pink)
			x, y = w.spot(30, 90, screenW-30, screenH-30, 10)
		}
		w.splash(x, y, kills > 0, kills)
	}
	w.hands(kills)
}

// paperHandPrints: how many bloody prints the hands leave on a page with that many victims (0 = clean hands).
func paperHandPrints(kills int) int {
	if kills <= 0 {
		return 0
	}
	return minI(2+kills/3, 14)
}

// hands: where the paper is held (the left and right edges, half way down) the bloody thumbs leave their prints,
// smeared where the hand slid; from 10 victims on, a whole palm with its fingers grabs the edge (both edges from 20).
func (w *pwear) hands(kills int) {
	n := paperHandPrints(kills)
	if n == 0 {
		return
	}
	core := uint32(0x7a0e0a)
	a := minI(170+kills*3, 240) // fresher, darker blood the more victims
	for i := 0; i < n; i++ {
		// where the thumb holds the page, on the free paper between the texts if there is any
		x, y := w.spot(6, 160, 28, 440, 10)
		if i%2 == 1 {
			x = screenW - 1 - x
		}
		tilt := float64(w.rnd(80)-40) / 100
		w.thumb(x, y, tilt, core, a)
		if w.rnd(3) == 0 { // the thumb slid: a fading smear
			dx, dy := 2-w.rnd(5), 3+w.rnd(4)
			for k := 1; k <= 3; k++ {
				w.thumb(x+dx*k, y+dy*k, tilt, core, a*(4-k)/6)
			}
		}
	}
	if kills >= 10 {
		_, y := w.spot(18, 180, 34, 430, 26)
		w.palm(true, y, core, a)
	}
	if kills >= 20 {
		_, y := w.spot(18, 180, 34, 430, 26)
		w.palm(false, y, core, a)
	}
}

// thumb: an oval print with the curved ridges of a fingertip.
func (w *pwear) thumb(cx, cy int, tilt float64, hex uint32, a int) {
	for y := -11; y <= 11; y++ {
		for x := -11; x <= 11; x++ {
			u := float64(x)*math.Cos(tilt) + float64(y)*math.Sin(tilt)
			v := -float64(x)*math.Sin(tilt) + float64(y)*math.Cos(tilt)
			if u*u/56+v*v/110 > 1 {
				continue
			}
			if int(math.Sqrt(u*u*1.6+v*v*0.5)*1.6)%3 == 0 {
				w.paint(cx+x, cy+y, 0x4e0606, a/3)
			} else {
				w.paint(cx+x, cy+y, hex, a)
			}
		}
	}
}

// palm: a hand that grabbed the page by one edge: the heel of the palm cut by the edge and four fingers reaching in.
func (w *pwear) palm(left bool, cy int, hex uint32, a int) {
	sx := 1
	ex := 2
	if !left {
		sx, ex = -1, screenW-3
	}
	blob := func(cx, cy int, rx, ry float64, tilt float64) {
		for y := -int(ry) - 1; y <= int(ry)+1; y++ {
			for x := -int(rx) - 1; x <= int(rx)+1; x++ {
				u := float64(x)*math.Cos(tilt) + float64(y)*math.Sin(tilt)
				v := -float64(x)*math.Sin(tilt) + float64(y)*math.Cos(tilt)
				e := u*u/(rx*rx) + v*v/(ry*ry)
				if e > 1 {
					continue
				}
				aa := a
				if paperHash(cx+x, cy+y, w.s+83)%7 == 0 { // the skin leaves gaps
					aa = a / 3
				}
				w.paint(cx+x, cy+y, hex, aa)
			}
		}
	}
	blob(ex, cy, 16, 24, 0) // the heel of the palm
	for f := 0; f < 4; f++ {
		fy := cy - 21 + f*13 + w.rnd(3)
		l := 30 + w.rnd(10)
		if f == 0 || f == 3 {
			l -= 6
		}
		ang := float64(f-1)*0.12*float64(sx) - 0.06*float64(sx)
		cx := ex + sx*(10+l/2)
		blob(cx, fy, float64(l)/2, 4.5, ang)
		w.thumb(ex+sx*(10+l), fy, 0, 0x6a0a08, a) // the fingertip presses harder
	}
}

// splash: a round drop with satellites thrown in one direction and one or two elongated drops with a tail.
func (w *pwear) splash(cx, cy int, blood bool, kills int) {
	core, rim := uint32(0x7a0e0a), uint32(0x4e0606)
	if !blood { // red wine: more purple
		core, rim = 0x6a1430, 0x48102a
	}
	r := 4 + w.rnd(5) + minI(kills/12, 3) // bigger drops after a massacre
	drop := func(x0, y0, rr int, a int) {
		for y := -rr - 1; y <= rr+1; y++ {
			for x := -rr - 1; x <= rr+1; x++ {
				d := math.Sqrt(float64(x*x+y*y)) + float64(paperHash(x0+x, y0+y, w.s+71)&1)*0.6
				switch {
				case d < float64(rr)-0.8:
					w.paint(x0+x, y0+y, core, a)
				case d < float64(rr)+0.4:
					w.paint(x0+x, y0+y, rim, a)
				}
			}
		}
		if rr >= 2 {
			w.paint(x0-rr/2, y0-rr/2, 0xc06050, a/3) // a little shine
		}
	}
	drop(cx, cy, r, 225)
	ang := float64(w.rnd(628)) / 100
	for k, m := 0, 9+w.rnd(8); k < m; k++ {
		a := ang + float64(w.rnd(110)-55)/100
		dist := float64(r + 3 + w.rnd(24))
		drop(cx+int(math.Cos(a)*dist), cy+int(math.Sin(a)*dist), 1+w.rnd(3), 210)
	}
	for k, m := 0, 1+w.rnd(2); k < m; k++ {
		a := ang + float64(w.rnd(60)-30)/100
		dist := float64(r + 6 + w.rnd(10))
		x, y := cx+int(math.Cos(a)*dist), cy+int(math.Sin(a)*dist)
		drop(x, y, 2, 220)
		for t := 1; t < 5+w.rnd(5); t++ { // the tail points back to the main drop
			w.paint(x-int(math.Cos(a)*float64(t)), y-int(math.Sin(a)*float64(t)), core, 200-t*14)
		}
	}
	if w.rnd(3) == 0 { // a few drops in a row, as if someone walked past bleeding
		a := ang + math.Pi/2
		for k, m := 1, 3+w.rnd(3); k <= m; k++ {
			d := float64(k * (9 + w.rnd(6)))
			drop(cx+int(math.Cos(a)*d), cy+int(math.Sin(a)*d), 1+w.rnd(2), 215)
		}
	}
}

// paperNoWear: the tests paint the page without the second layer, to compare.
var paperNoWear bool
