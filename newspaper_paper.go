package main

import (
	"image"
	"image/color"
	"math"
)

// The aged paper under the front page: grain, fibres, yellowing, a vignette, two folds, small tears and stains.
// Everything is a function of the seed (no randomness at run time) and is painted BEFORE any text.

func paperHash(x, y int, s uint64) uint64 {
	h := uint64(x+1)*73856093 ^ uint64(y+1)*19349663 ^ (s+1)*83492791
	h ^= h >> 13
	h *= 0x9e3779b97f4a7c15
	h ^= h >> 29
	return h
}

var paperDesk = color.RGBA{0x2c, 0x25, 0x1e, 255}

func newPaperCanvas(seed int64, lay *paperLayout) *pcv {
	c := &pcv{im: image.NewRGBA(image.Rect(0, 0, screenW, screenH)), lay: lay}
	s := uint64(seed)*2654435761 + 12345
	ph1, ph2 := float64(s%628)/100, float64((s/7)%628)/100
	for y := 0; y < screenH; y++ {
		for x := 0; x < screenW; x++ {
			grain := int(paperHash(x, y, s)&7) - 4
			lump := math.Sin(float64(x)/83+ph1)*math.Sin(float64(y)/61+ph2)*5 + math.Sin(float64(x+y)/37)*2
			dx, dy := (float64(x)-screenW/2)/(screenW/2), (float64(y)-screenH/2)/(screenH/2)
			r := math.Sqrt(dx*dx+dy*dy) / 1.4142
			vig := 46 * r * r * r
			edge := math.Min(math.Min(float64(x), float64(screenW-1-x)), math.Min(float64(y), float64(screenH-1-y)))
			if edge < 14 {
				vig += (14 - edge) * 2.4
			}
			d := grain + int(lump) - int(vig)
			c.put(x, y, color.RGBA{pclamp(0xe8 + d), pclamp(0xdc + d - int(lump)/2 - int(vig)/8), pclamp(0xb6 + d - int(lump) - int(vig)/4), 255})
		}
	}
	// fibres: short darker streaks
	for i := 0; i < 1500; i++ {
		h := paperHash(i, 7, s)
		x, y, n := int(h%screenW), int((h>>12)%screenH), 3+int((h>>24)%7)
		horiz := (h>>30)&1 == 0
		for k := 0; k < n; k++ {
			if horiz {
				c.shade(x+k, y, 1, 1, -7)
			} else {
				c.shade(x, y+k, 1, 1, -7)
			}
		}
	}
	// folds: a vertical one and a horizontal one, shadow on one side and light on the other, worn along the crease
	foldV, foldH := screenW/2, screenH/2
	for y := 0; y < screenH; y++ {
		for k := -8; k <= 14; k++ {
			d := 0
			switch {
			case k < 0:
				d = 7 * (8 + k) / 8
			case k == 0:
				d = -34
			default:
				d = -22 * (14 - k) / 14
			}
			if k == 0 && paperHash(y, 3, s)%6 == 0 {
				d = 12 // worn crease
			}
			c.shade(foldV+k, y, 1, 1, d)
		}
	}
	for x := 0; x < screenW; x++ {
		for k := -8; k <= 12; k++ {
			d := 0
			switch {
			case k < 0:
				d = 6 * (8 + k) / 8
			case k == 0:
				d = -30
			default:
				d = -18 * (12 - k) / 12
			}
			if k == 0 && paperHash(x, 5, s)%6 == 0 {
				d = 12
			}
			c.shade(x, foldH+k, 1, 1, d)
		}
	}
	// coffee rings and ink spots
	ring := func(cx, cy, r int, a int) {
		for y := -r - 2; y <= r+2; y++ {
			for x := -r - 2; x <= r+2; x++ {
				dd := math.Sqrt(float64(x*x + y*y))
				wob := float64(paperHash(x*3, y*3, s)&3) * 0.3
				switch {
				case math.Abs(dd-float64(r)) < 1.6+wob:
					c.tint(cx+x, cy+y, 0x7a5430, a)
				case dd < float64(r):
					c.tint(cx+x, cy+y, 0x8a6a40, a/5)
				}
			}
		}
	}
	ring(int(s%90)+830, 478+int(s%30), 24, 62)
	ring(int(s%70)+60, 36+int(s%40), 13, 44)
	for i := 0; i < 9; i++ {
		h := paperHash(i, 11, s)
		cx, cy, r := int(h%(screenW-40))+20, int((h>>10)%(screenH-40))+20, 1+int((h>>20)%2)
		for y := -r; y <= r; y++ {
			for x := -r; x <= r; x++ {
				if x*x+y*y <= r*r {
					c.tint(cx+x, cy+y, 0x2a2014, 55)
				}
			}
		}
	}
	c.wrinkles(s)
	c.tears(s)
	c.dirt(s)
	c.ragged(s)
	return c
}

// tint blends a colour over a pixel (a = 0..255) for stains.
func (c *pcv) tint(x, y int, hex uint32, a int) {
	if x < 0 || y < 0 || x >= c.im.Rect.Dx() || y >= c.im.Rect.Dy() || a <= 0 {
		return
	}
	i := c.im.PixOffset(x, y)
	t := [3]int{int(hex >> 16 & 255), int(hex >> 8 & 255), int(hex & 255)}
	for k := 0; k < 3; k++ {
		c.im.Pix[i+k] = pclamp((int(c.im.Pix[i+k])*(255-a) + t[k]*a) / 255)
	}
}

// tears: a clipped corner, a bite out of the bottom edge and a short slit on the left edge. All of it stays in the
// outer 12 px, where there is no text.
func (c *pcv) tears(s uint64) {
	desk := func(x, y int) {
		if x < 0 || y < 0 || x >= screenW || y >= screenH {
			return
		}
		n := int(paperHash(x, y, s+3)&7) - 3
		c.put(x, y, color.RGBA{pclamp(0x2c + n), pclamp(0x25 + n), pclamp(0x1e + n), 255})
	}
	// top-right corner cut
	for y := 0; y < 11; y++ {
		for x := screenW - 11; x < screenW; x++ {
			if (x-(screenW-11))+(10-y) >= 9+int(paperHash(x, y, s)&1) {
				desk(x, y)
			}
		}
	}
	// the paper edge next to the cut is lighter (torn fibres)
	for k := 0; k < 11; k++ {
		c.shade(screenW-11+k-1, 10-k+int(paperHash(k, 2, s)&1)-0, 2, 1, 14)
	}
	// bite on the bottom edge, ragged
	bx := 560 + int(s%120)
	for x := bx; x < bx+26; x++ {
		depth := 5 - absI(x-bx-13)*5/13
		depth += int(paperHash(x, 9, s) & 1)
		for y := screenH - depth; y < screenH; y++ {
			desk(x, y)
		}
		if depth > 0 {
			c.shade(x, screenH-depth-1, 1, 1, 16)
		}
	}
	// slit on the left edge
	sy := 250 + int(s%90)
	for x := 0; x < 9; x++ {
		desk(x, sy)
		if x < 6 {
			desk(x, sy+1)
		}
		c.shade(x, sy-1, 1, 1, 12)
	}
	// dog-ear at the bottom-left corner: a lighter triangle with a shadow along the fold
	for k := 0; k < 14; k++ {
		for x := 0; x <= k; x++ {
			c.shade(x, screenH-14+k, 1, 1, 10)
		}
		c.shade(k, screenH-14+k, 2, 1, -36)
	}
}

// deskPx paints one pixel of the table showing through a torn edge.
func (c *pcv) deskPx(x, y int, s uint64) {
	if x < 0 || y < 0 || x >= screenW || y >= screenH {
		return
	}
	n := int(paperHash(x, y, s+3)&7) - 3
	c.put(x, y, color.RGBA{pclamp(0x2c + n), pclamp(0x25 + n), pclamp(0x1e + n), 255})
}

// inMargin: the strip around the page where there is no text (stains may be strong there).
func inMargin(x, y int) bool { return x < 13 || x >= screenW-13 || y < 13 || y >= screenH-11 }

// wrinkles: crumple lines across the whole page, a thin dark line with a light one beside it. Very subtle (a few grey
// levels), so the reading text over them stays as sharp as everywhere else.
func (c *pcv) wrinkles(s uint64) {
	for i := 0; i < 11; i++ {
		h := paperHash(i, 21, s)
		x, y := float64(h%screenW), float64((h>>11)%screenH)
		ang := float64((h>>22)%628) / 100
		n := 70 + int((h>>32)%170)
		for k := 0; k < n; k++ {
			ang += (float64(paperHash(i, k, s+5)&15) - 7.5) * 0.012
			x += math.Cos(ang)
			y += math.Sin(ang)
			xi, yi := int(x), int(y)
			c.shade(xi, yi, 1, 1, -11)
			c.shade(xi+1, yi, 1, 1, 7)
			if k%7 == 0 {
				c.shade(xi-1, yi+1, 1, 1, -4)
			}
		}
	}
}

// dirt: a cigarette burn at the top right, coffee rings, oil and greasy fingerprints. Strong only on the margins; over
// the printed area they are a faint tint (the text is painted afterwards and stays crisp).
func (c *pcv) dirt(s uint64) {
	stain := func(x, y int, hex uint32, a int) {
		if !inMargin(x, y) {
			a /= 5
		}
		c.tint(x, y, hex, a)
	}
	ring := func(cx, cy, r, a int) {
		for y := -r - 3; y <= r+3; y++ {
			for x := -r - 3; x <= r+3; x++ {
				dd := math.Sqrt(float64(x*x + y*y))
				wob := float64(paperHash(x*3, y*3, s+9)&3) * 0.3
				switch {
				case math.Abs(dd-float64(r)) < 1.8+wob:
					stain(cx+x, cy+y, 0x6a4420, a)
				case dd < float64(r):
					stain(cx+x, cy+y, 0x8a6a40, a/6)
				case dd < float64(r)+4: // the drip that ran outside
					stain(cx+x, cy+y, 0x8a6a40, a/9)
				}
			}
		}
	}
	ring(-6, 140+int(s%60), 24, 90)
	ring(screenW+7, 290+int(s%70), 20, 80)
	ring(int(s%200)+120, screenH+8, 18, 80)
	// oil: a soft yellowish blot that darkens the paper like grease
	oil := func(cx, cy, r int) {
		for y := -r; y <= r; y++ {
			for x := -r; x <= r; x++ {
				d2 := float64(x*x+y*y) / float64(r*r)
				if d2 <= 1 {
					stain(cx+x, cy+y, 0x9a7a1c, int(110*(1-d2)))
				}
			}
		}
	}
	oil(14+int(s%40), -4, 15)
	oil(screenW-8, int(s%90)+390, 17)
	oil(int(s%260)+420, screenH+5, 12)
	// greasy fingerprints: an oval with curved ridges
	finger := func(cx, cy int, tilt float64) {
		for y := -10; y <= 10; y++ {
			for x := -7; x <= 7; x++ {
				u := float64(x)*math.Cos(tilt) + float64(y)*math.Sin(tilt)
				v := -float64(x)*math.Sin(tilt) + float64(y)*math.Cos(tilt)
				e := u*u/49 + v*v/100
				if e > 1 {
					continue
				}
				if int(math.Sqrt(u*u*1.6+v*v*0.5)*1.7)%3 == 0 {
					stain(cx+x, cy+y, 0x4a3a22, 66)
				} else {
					stain(cx+x, cy+y, 0x4a3a22, 14)
				}
			}
		}
	}
	finger(5, 410+int(s%40), 0.3)
	finger(screenW-5, 60+int(s%50), -0.4)
	finger(int(s%300)+300, 4, 1.2)
	finger(int(s%200)+640, screenH-4, 0.8)
	// the cigarette burn: a hole with a charred ring and a brown halo; it stays in the top-right margin above the ear
	bx, by := screenW-52+int(s%6), 9
	for y := -17; y <= 17; y++ {
		for x := -17; x <= 17; x++ {
			dd := math.Sqrt(float64(x*x + y*y))
			dd += float64(paperHash(x+40, y+40, s+13)&3)*0.55 - 0.8 // ragged edge
			px, py := bx+x, by+y
			switch {
			case dd < 3.4:
				c.deskPx(px, py, s) // burnt through
			case dd < 5.2:
				c.tint(px, py, 0x120a06, 245)
			case dd < 7.4:
				c.tint(px, py, 0x4a2a12, 225)
			case dd < 9.6:
				c.tint(px, py, 0x8a5424, 175)
			case dd < 12.5:
				c.tint(px, py, 0xa8742e, 90)
			case dd < 15.5:
				c.tint(px, py, 0xa8742e, 34)
			}
		}
	}
}

// ragged: the top edge is irregular, with a few cuts and rips; none goes below y = 11, the page rules start at 14 and the
// first text at 20.
func (c *pcv) ragged(s uint64) {
	depth := 2.0
	for x := 0; x < screenW-12; x++ {
		depth += (float64(paperHash(x, 31, s)&7) - 3.5) * 0.28
		depth = math.Max(0, math.Min(3.4, depth))
		d := int(depth)
		for y := 0; y < d; y++ {
			c.deskPx(x, y, s)
		}
		if d > 0 {
			c.shade(x, d, 1, 1, 14) // torn fibre catching the light
		}
	}
	rips := []struct{ x, w, d int }{{90, 16, 9}, {330, 22, 7}, {560, 12, 10}, {770, 19, 8}}
	for i, r := range rips {
		x0 := r.x + int((s>>uint(3*i))%90)
		for k := 0; k < r.w; k++ {
			// a lopsided wedge: steep on one side, long slope on the other
			f := float64(k) / float64(r.w)
			prof := f * 2
			if f > 0.5 {
				prof = (1 - f) * 0.9
			}
			d := int(float64(r.d)*math.Min(1, prof)) + int(paperHash(x0+k, 33, s)&1)
			for y := 0; y < d; y++ {
				c.deskPx(x0+k, y, s)
			}
			c.shade(x0+k, d, 1, 1, 18)
			c.shade(x0+k, d+1, 1, 1, -12)
		}
	}
}
