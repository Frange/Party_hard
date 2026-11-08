package main

import "image/color"

// The photo of the front page: a small pixel-art picture in four sepia tones, ordered dither, no anti-aliasing.
// Three scenes, chosen by the seed of the party.

var photoTone = [4]color.RGBA{{0x1c, 0x16, 0x10, 255}, {0x5a, 0x4a, 0x38, 255}, {0x9a, 0x88, 0x68, 255}, {0xd8, 0xc9, 0xa4, 255}}

const paperPhotoScenes = 3

func (c *pcv) figure(cx, by int, hat, arms bool) {
	t0 := photoTone[0]
	c.ellipse(cx, by-24, 4, 4, t0)
	c.fill(cx-5, by-19, 10, 13, t0)
	c.fill(cx-4, by-6, 3, 6, t0)
	c.fill(cx+1, by-6, 3, 6, t0)
	if hat {
		c.fill(cx-6, by-28, 12, 2, t0)
		c.fill(cx-3, by-32, 6, 4, t0)
	}
	if arms {
		c.fill(cx-8, by-26, 2, 9, t0)
		c.fill(cx+6, by-26, 2, 9, t0)
	}
}

func (c *pcv) photo(x, y, w, h, scene int, two bool, seed int64) {
	t := photoTone
	switch ((scene % paperPhotoScenes) + paperPhotoScenes) % paperPhotoScenes {
	case 0: // the house at night, police tape, the killers in front
		gy := y + h - 30
		for yy := 0; yy < gy-y; yy++ {
			f := float64(yy) / float64(gy-y)
			for xx := 0; xx < w; xx++ {
				col := t[1]
				if dither(f*1.1, xx, yy) {
					col = t[2]
				}
				c.put(x+xx, y+yy, col)
			}
		}
		c.ellipse(x+w-34, y+20, 9, 9, t[3])
		c.ellipse(x+w-31, y+18, 2, 2, t[2])
		bx := 0
		for i := 0; bx < w; i++ {
			hh := paperHash(i, 5, uint64(seed))
			bw := 12 + int(hh%12)
			bh := 16 + int((hh>>8)%26)
			c.fill(x+bx, gy-bh, bw, bh, t[0])
			for wy := gy - bh + 4; wy < gy-3; wy += 6 {
				for wx := bx + 3; wx < bx+bw-3; wx += 5 {
					if paperHash(wx, wy, uint64(seed))%3 == 0 {
						c.fill(x+wx, wy, 2, 3, t[3])
					}
				}
			}
			bx += bw + 1
		}
		hx := x + w/2 - 30
		c.fill(hx, gy-38, 60, 38, t[0])
		c.tri(hx-6, gy-38, hx+30, gy-62, hx+66, gy-38, t[0])
		for _, wx := range []int{8, 22, 38} {
			c.fill(hx+wx, gy-30, 8, 9, t[3])
			c.fill(hx+wx+3, gy-30, 1, 9, t[0])
		}
		c.fill(hx+26, gy-14, 9, 14, t[2])
		for yy := gy; yy < y+h; yy++ {
			for xx := 0; xx < w; xx++ {
				col := t[2]
				if dither(0.4, xx, yy) {
					col = t[1]
				}
				c.put(x+xx, yy, col)
			}
		}
		c.figure(x+w/2-24, y+h-5, true, false)
		if two {
			c.figure(x+w/2+26, y+h-5, false, false)
		}
		c.thickLine(x, y+h-34, x+w-1, y+h-50, 4, func(i int) color.RGBA {
			if (i/6)%2 == 0 {
				return t[3]
			}
			return t[0]
		})
		c.fill(x+6, y+h-40, 2, 36, t[0])
		c.fill(x+w-8, y+h-56, 2, 52, t[0])
	case 1: // the dance floor under a mirror ball
		c.fill(x, y, w, h, t[0])
		cx := x + w/2
		for k := -3; k <= 3; k++ {
			if k == 0 {
				continue
			}
			c.thickLine(cx, y+16, cx+k*30, y+h-24, 1, func(i int) color.RGBA {
				if i%2 == 0 {
					return t[1]
				}
				return t[0]
			})
		}
		for r := 0; r < 5; r++ {
			ry := y + h - 26 + r*6
			if ry >= y+h {
				break
			}
			tile := 10 + r*3
			for xx := 0; xx < w; xx++ {
				col := t[1]
				if ((xx/tile)+r)%2 == 0 {
					col = t[2]
				}
				c.fill(x+xx, ry, 1, minI(6, y+h-ry), col)
			}
		}
		c.fill(cx, y, 1, 10, t[2])
		c.ellipse(cx, y+17, 8, 8, t[3])
		for k := -6; k <= 6; k += 4 {
			c.fill(cx+k, y+10, 1, 15, t[1])
			c.fill(cx-8, y+17+k/2, 17, 1, t[1])
		}
		xs := []int{x + 40, x + 76, x + 118, x + 154}
		if two {
			xs = []int{x + 52, x + 98, x + 146}
		}
		for i, fx := range xs {
			c.figure(fx, y+h-6, i == 1, true)
		}
	default: // the pool and a suspicious duck
		for yy := 0; yy < h; yy++ {
			for xx := 0; xx < w; xx++ {
				col := t[3]
				if yy < 34 {
					col = t[2]
					if dither(0.5, xx, yy) {
						col = t[3]
					}
				} else if yy < 40 {
					col = t[0]
				} else {
					col = t[1]
					if (yy+xx/7)%5 == 0 {
						col = t[2]
					} else if dither(0.3, xx, yy) {
						col = t[2]
					}
				}
				c.put(x+xx, y+yy, col)
			}
		}
		for i := 0; i < 5; i++ {
			c.fill(x+10+i*40, y+8, 2, 26, t[0])
			c.fill(x+6+i*40, y+8, 10, 2, t[0])
		}
		dx, dy := x+w/2, y+h-34
		c.ellipse(dx, dy+14, 40, 5, t[1])
		c.ellipse(dx, dy+14, 30, 3, t[2])
		c.ellipse(dx, dy, 13, 8, t[0])
		c.ellipse(dx, dy, 11, 6, t[3])
		c.ellipse(dx+11, dy-9, 6, 6, t[0])
		c.ellipse(dx+11, dy-9, 4, 4, t[3])
		c.fill(dx+15, dy-8, 6, 3, t[1])
		c.fill(dx+12, dy-11, 2, 2, t[0])
		c.fill(dx-10, dy-3, 6, 2, t[2])
	}
	c.frame(x, y, w, h, paperInk)
	c.frame(x-1, y-1, w+2, h+2, paperShade)
}
