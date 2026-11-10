package main

import (
	"image"
	"image/color"
	"math"
)

// The wear of the invitation letter: it went through a space station party before anybody read it.
// The same recipe as the front page of the newspaper (newspaper_paper.go / newspaper_wear.go): grain, the creases of
// a letter folded in three and then in half (they cross), crumple lines, coffee rings, an ink blot, a couple of drops
// of blood in the margins, and big rips along the edges and a missing corner (the holes are transparent).
//
// The text is NOT part of this picture: it is drawn on top afterwards (paintPaper), so it stays crisp. Rips stop
// before any text box, and stains over a text box are only a faint tint.

// letterWear holds the canvas and the masks of the letter while it is worn.
type letterWear struct {
	c    *pcv
	w, h int
	s    uint64
	n    int
	text []bool // inside a text box (or the letterhead): stains are faint there
	near []bool // within a few pixels of a text box or the device: rips stop before it
	hole []bool // torn away
}

func (lw *letterWear) rnd(k int) int {
	lw.n++
	return int(paperHash(lw.n, 303, lw.s) % uint64(k))
}

// letterWearImage paints the worn, blank paper of the letter (invPW x invPH). texts are the boxes the text will take.
func letterWearImage(texts []image.Rectangle, seed uint64) *image.RGBA {
	W, H := int(invPW), int(invPH)
	lw := &letterWear{c: &pcv{im: image.NewRGBA(image.Rect(0, 0, W, H))}, w: W, h: H, s: seed*2654435761 + 4242,
		text: make([]bool, W*H), near: make([]bool, W*H), hole: make([]bool, W*H)}
	mark := func(r image.Rectangle, pad int, m []bool) {
		for y := r.Min.Y - pad; y < r.Max.Y+pad; y++ {
			for x := r.Min.X - pad; x < r.Max.X+pad; x++ {
				if x >= 0 && y >= 0 && x < W && y < H {
					m[y*W+x] = true
				}
			}
		}
	}
	head := image.Rect(22, 15, 58, 57) // the amber drop of the letterhead
	for _, r := range append([]image.Rectangle{head}, texts...) {
		mark(r, 1, lw.text)
		mark(r, 5, lw.near)
	}
	mark(image.Rect(W-150, H-100, W, H), 0, lw.near) // the device and its wires are stuck there
	lw.base()
	lw.folds()
	lw.wrinkles()
	lw.stains()
	lw.blood()
	lw.rips()
	return lw.c.im
}

// base: the paper with grain, a soft lump, a darker worn rim and the faint ruled lines of the stationery.
func (lw *letterWear) base() {
	W, H, s := lw.w, lw.h, lw.s
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			grain := int(paperHash(x, y, s)&7) - 4
			lump := math.Sin(float64(x)/71+1.3)*math.Sin(float64(y)/53+0.4)*5 + math.Sin(float64(x+y)/31)*2
			edge := math.Min(math.Min(float64(x), float64(W-1-x)), math.Min(float64(y), float64(H-1-y)))
			vig := 0.0
			if edge < 12 {
				vig = (12 - edge) * 3
			}
			d := grain + int(lump) - int(vig)
			lw.c.put(x, y, color.RGBA{pclamp(0xf2 + d), pclamp(0xe6 + d - int(lump)/2 - int(vig)/6), pclamp(0xc2 + d - int(lump) - int(vig)/3), 255})
		}
	}
	for y := 12; y < H-8; y += 18 {
		for x := 8; x < W-8; x++ {
			lw.c.tint(x, y, 0xd8c89c, 90)
		}
	}
	for i := 0; i < 700; i++ { // fibres
		h := paperHash(i, 7, s)
		x, y, n := int(h%uint64(W)), int((h>>12)%uint64(H)), 3+int((h>>24)%6)
		for k := 0; k < n; k++ {
			if (h>>30)&1 == 0 {
				lw.c.shade(x+k, y, 1, 1, -6)
			} else {
				lw.c.shade(x, y+k, 1, 1, -6)
			}
		}
	}
}

// folds: folded in three (two horizontal creases) and then in half (one vertical crease): shadow on one side, light
// on the other, worn white here and there along the crease, and the edges of the paper dented where a crease ends.
func (lw *letterWear) folds() {
	W, H, s := lw.w, lw.h, lw.s
	crease := func(k int) int {
		switch {
		case k < 0:
			return 8 * (7 + k) / 7
		case k == 0:
			return -38
		}
		return -24 * (12 - k) / 12
	}
	vx := W/2 + 6
	for y := 0; y < H; y++ {
		for k := -7; k <= 12; k++ {
			d := crease(k)
			if k == 0 && paperHash(y, 3, s)%5 == 0 {
				d = 14
			}
			lw.c.shade(vx+k, y, 1, 1, d)
		}
	}
	for _, hy := range []int{H / 3, 2 * H / 3} {
		for x := 0; x < W; x++ {
			for k := -7; k <= 12; k++ {
				d := crease(k) * 9 / 10
				if k == 0 && paperHash(x, hy, s)%5 == 0 {
					d = 14
				}
				lw.c.shade(x, hy+k, 1, 1, d)
			}
		}
	}
	// where the creases cross, the paper is worn thin: a lighter, rubbed patch
	for _, hy := range []int{H / 3, 2 * H / 3} {
		for y := -6; y <= 6; y++ {
			for x := -6; x <= 6; x++ {
				if x*x+y*y <= 36 {
					lw.c.shade(vx+x, hy+y, 1, 1, 6)
				}
			}
		}
	}
}

// wrinkles: thin crumple lines (a dark line with a light one beside it), very subtle.
func (lw *letterWear) wrinkles() {
	for i := 0; i < 8; i++ {
		h := paperHash(i, 21, lw.s)
		x, y := float64(h%uint64(lw.w)), float64((h>>11)%uint64(lw.h))
		ang := float64((h>>22)%628) / 100
		for k, n := 0, 60+int((h>>32)%140); k < n; k++ {
			ang += (float64(paperHash(i, k, lw.s+5)&15) - 7.5) * 0.012
			x += math.Cos(ang)
			y += math.Sin(ang)
			lw.c.shade(int(x), int(y), 1, 1, -10)
			lw.c.shade(int(x)+1, int(y), 1, 1, 7)
		}
	}
}

// tint a pixel with a stain: strong on the free paper, a faint shadow under the text (it keeps its contrast).
func (lw *letterWear) paint(x, y int, hex uint32, a int) {
	if x < 0 || y < 0 || x >= lw.w || y >= lw.h || a <= 0 {
		return
	}
	if lw.text[y*lw.w+x] {
		a /= 5
	}
	lw.c.tint(x, y, hex, a)
}

// spot: among a few candidates of the box, the centre whose disc covers the fewest text pixels.
func (lw *letterWear) spot(x0, y0, x1, y1, r int) (int, int) {
	bx, by, best := x0, y0, 1<<30
	for i := 0; i < 10; i++ {
		x, y := x0+lw.rnd(x1-x0+1), y0+lw.rnd(y1-y0+1)
		hit := 0
		for yy := y - r; yy <= y+r; yy += 2 {
			for xx := x - r; xx <= x+r; xx += 2 {
				if xx >= 0 && yy >= 0 && xx < lw.w && yy < lw.h && lw.text[yy*lw.w+xx] {
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

func (lw *letterWear) ring(cx, cy, r, a int) {
	for y := -r - 4; y <= r+4; y++ {
		for x := -r - 4; x <= r+4; x++ {
			dd := math.Sqrt(float64(x*x + y*y))
			wob := float64(paperHash(x*5, y*5, lw.s+51)&3) * 0.35
			switch {
			case math.Abs(dd-float64(r)) < 1.6+wob:
				lw.paint(cx+x, cy+y, 0x6a4420, a)
			case dd < float64(r):
				lw.paint(cx+x, cy+y, 0x8a6a40, a/5)
			}
		}
	}
}

func (lw *letterWear) blot(cx, cy, r int, hex uint32, a int) {
	k := lw.rnd(1000)
	for y := -r - 3; y <= r+3; y++ {
		for x := -r - 3; x <= r+3; x++ {
			ang := math.Atan2(float64(y), float64(x))
			edge := float64(r) * (1 + 0.22*math.Sin(3*ang+float64(k)) + 0.12*math.Sin(5*ang+float64(k/3)))
			dd := math.Sqrt(float64(x*x + y*y))
			switch {
			case dd < edge-1.5:
				lw.paint(cx+x, cy+y, hex, a*3/4)
			case dd < edge:
				lw.paint(cx+x, cy+y, hex, a)
			}
		}
	}
}

// stains: a coffee ring half off the left edge, another on the paper, a water mark and an ink blot, and a greasy thumb.
func (lw *letterWear) stains() {
	W, H := lw.w, lw.h
	lw.ring(-4, 120+lw.rnd(80), 26, 95)
	x, y := lw.spot(60, H/2, W/2, H-30, 24)
	lw.ring(x, y, 18+lw.rnd(8), 80)
	x, y = lw.spot(W/2, 60, W-40, H/2, 22)
	lw.blot(x, y, 16+lw.rnd(8), 0x9a7a3c, 46) // a water mark
	x, y = lw.spot(30, H-34, W-170, H-12, 8)
	lw.blot(x, y, 5+lw.rnd(4), 0x1e1a24, 160) // an ink blot
	x, y = lw.spot(W-60, 20, W-14, H/2, 10)
	for k := -16; k < 16; k++ { // a greasy thumb on the right margin
		f := 1 - math.Abs(float64(k))/16
		for j := -6; j <= 6; j++ {
			lw.paint(x+j, y+k, 0x3a3226, int(56*f*(1-math.Abs(float64(j))/7)))
		}
	}
}

// blood: two splashes in the margins (it was a party on a space station, after all).
func (lw *letterWear) blood() {
	W, H := lw.w, lw.h
	x, y := lw.spot(10, 60, 36, H-60, 6)
	lw.splash(x, y)
	x, y = lw.spot(W/3, H-26, W-180, H-10, 6)
	lw.splash(x, y)
}

func (lw *letterWear) splash(cx, cy int) {
	core, rim := uint32(0x7a0e0a), uint32(0x4e0606)
	drop := func(x0, y0, rr, a int) {
		for y := -rr - 1; y <= rr+1; y++ {
			for x := -rr - 1; x <= rr+1; x++ {
				d := math.Sqrt(float64(x*x+y*y)) + float64(paperHash(x0+x, y0+y, lw.s+71)&1)*0.6
				switch {
				case d < float64(rr)-0.8:
					lw.paint(x0+x, y0+y, core, a)
				case d < float64(rr)+0.4:
					lw.paint(x0+x, y0+y, rim, a)
				}
			}
		}
	}
	r := 3 + lw.rnd(3)
	drop(cx, cy, r, 225)
	ang := float64(lw.rnd(628)) / 100
	for k, m := 0, 6+lw.rnd(6); k < m; k++ {
		a := ang + float64(lw.rnd(110)-55)/100
		dist := float64(r + 3 + lw.rnd(16))
		drop(cx+int(math.Cos(a)*dist), cy+int(math.Sin(a)*dist), lw.rnd(3), 210)
	}
}

// ---- rips ---------------------------------------------------------------------------------------------------------

// edge e (0 top, 1 bottom, 2 left, 3 right): (along, depth) -> pixel
func (lw *letterWear) xy(e, a, d int) (int, int) {
	switch e {
	case 0:
		return a, d
	case 1:
		return a, lw.h - 1 - d
	case 2:
		return d, a
	}
	return lw.w - 1 - d, a
}

// tear removes a pixel (transparent: the night shows through) and lights the torn fibres round it.
func (lw *letterWear) tear(x, y int) {
	if x < 0 || y < 0 || x >= lw.w || y >= lw.h {
		return
	}
	i := lw.c.im.PixOffset(x, y)
	lw.c.im.Pix[i], lw.c.im.Pix[i+1], lw.c.im.Pix[i+2], lw.c.im.Pix[i+3] = 0, 0, 0, 0
	lw.hole[y*lw.w+x] = true
}

// rips: big lopsided bites out of the four edges and a corner torn off; never into a text.
func (lw *letterWear) rips() {
	plan := []struct{ e, count, min, max int }{{0, 3, 10, 24}, {1, 3, 12, 26}, {2, 2, 10, 22}, {3, 2, 10, 22}}
	for _, p := range plan {
		long := lw.w
		if p.e >= 2 {
			long = lw.h
		}
		for i := 0; i < p.count; i++ {
			wd := 26 + lw.rnd(44)
			pos := 20 + lw.rnd(long-40-wd)
			lw.rip(p.e, pos, wd, p.min+lw.rnd(p.max-p.min+1))
		}
	}
	// a corner torn away: the top right one (a jagged diagonal)
	size := 46 + lw.rnd(16)
	for y := 0; y < size; y++ {
		for x := lw.w - size; x < lw.w; x++ {
			jag := int(paperHash(x+y, 9, lw.s)%5) - 2
			if (x-(lw.w-size))+(size-1-y) >= size+jag && !lw.near[y*lw.w+x] {
				lw.tear(x, y)
			}
		}
	}
	lw.fibres()
}

func (lw *letterWear) rip(e, pos, wd, want int) {
	room := want
	for a := pos - 2; a < pos+wd+2; a++ {
		for d := 0; d < room; d++ {
			x, y := lw.xy(e, a, d)
			if x < 0 || y < 0 || x >= lw.w || y >= lw.h {
				break
			}
			if lw.near[y*lw.w+x] {
				room = d - 1
				break
			}
		}
	}
	if room < 5 {
		return
	}
	peak := 0.3 + float64(lw.rnd(40))/100
	jag := 0.0
	for k := 0; k < wd; k++ {
		f := float64(k) / float64(wd)
		prof := f / peak
		if f > peak {
			prof = (1 - f) / (1 - peak)
		}
		prof = math.Pow(prof, 0.8)
		jag += (float64(paperHash(pos+k, 41+e, lw.s)&7) - 3.5) * 0.45
		jag = math.Max(-2.5, math.Min(2.5, jag))
		d := int(float64(room)*prof + jag)
		if d > room {
			d = room
		}
		for dd := 0; dd < d; dd++ {
			x, y := lw.xy(e, pos+k, dd)
			lw.tear(x, y)
		}
	}
}

// fibres: the paper next to a hole is a whiter, uneven torn edge with a darker crumple behind it.
func (lw *letterWear) fibres() {
	W, H := lw.w, lw.h
	isHole := func(x, y int) bool { return x >= 0 && y >= 0 && x < W && y < H && lw.hole[y*W+x] }
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			if lw.hole[y*W+x] {
				continue
			}
			switch {
			case isHole(x-1, y) || isHole(x+1, y) || isHole(x, y-1) || isHole(x, y+1):
				lw.c.shade(x, y, 1, 1, 28)
			case isHole(x-2, y) || isHole(x+2, y) || isHole(x, y-2) || isHole(x, y+2):
				if paperHash(x, y, lw.s+13)%3 != 0 {
					lw.c.shade(x, y, 1, 1, -14)
				}
			}
		}
	}
}
