package main

import (
	"image"
	"image/color"
)

// The photo of the front page: a small pixel-art picture in four sepia tones, ordered dither, no anti-aliasing.
// Eight scenes (paperPhotoScene picks one by the facts of the party and the seed, and not the one of the last parties).
// Everything is drawn with a clip on the photo's rectangle: nothing can leave the frame.
// Textures are dithered areas, never 1 px stripes: a single line across the photo reads as a printing fault.

var photoTone = [4]color.RGBA{{0x1c, 0x16, 0x10, 255}, {0x5a, 0x4a, 0x38, 255}, {0x9a, 0x88, 0x68, 255}, {0xd8, 0xc9, 0xa4, 255}}

const (
	paperPhotoScenes = 8
	sceneHouse       = 0 // the house at night, police tape
	sceneDance       = 1 // the dance floor under a mirror ball
	scenePool        = 2 // the pool and a suspicious duck
	sceneCone        = 3 // a guest wearing a traffic cone
	sceneDog         = 4 // a dog in a party hat, eating the canapés
	sceneCake        = 5 // a cake with a footprint on it
	sceneDJ          = 6 // the DJ asleep on the decks
	sceneSofa        = 7 // a guest asleep upside down on the sofa
)

func (c *pcv) figure(cx, by int, hat, arms bool) { c.figureC(cx, by, hat, arms, photoTone[0]) }

func (c *pcv) figureC(cx, by int, hat, arms bool, t0 color.RGBA) {
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

// dfill: a dithered rectangle, frac of the pixels in col b, the rest in col a (the pattern is fixed to the page).
func (c *pcv) dfill(x, y, w, h int, frac float64, a, b color.RGBA) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			col := a
			if dither(frac, xx, yy) {
				col = b
			}
			c.put(xx, yy, col)
		}
	}
}

// room: a wall and a floor (the floor starts fy px from the top).
func (c *pcv) room(x, y, w, h, fy int) {
	t := photoTone
	c.dfill(x, y, w, fy, 0.25, t[2], t[3])
	c.dfill(x, y+fy, w, h-fy, 0.35, t[1], t[2])
	c.fill(x, y+fy, w, 2, t[0]) // skirting board
}

func (c *pcv) photo(x, y, w, h, scene int, two bool, seed int64) {
	c.clip = image.Rect(x, y, x+w, y+h)
	defer func() { c.clip = image.Rectangle{} }()
	if scene == paperSceneBooth {
		c.boothStrip(x, y, w, h, seed)
	} else {
		c.photoScene(x, y, w, h, ((scene%paperPhotoScenes)+paperPhotoScenes)%paperPhotoScenes, two, seed)
	}
	c.clip = image.Rectangle{}
	c.frame(x, y, w, h, paperInk)
	c.frame(x-1, y-1, w+2, h+2, paperShade)
}

func (c *pcv) photoScene(x, y, w, h, scene int, two bool, seed int64) {
	t := photoTone
	switch scene {
	case sceneHouse:
		gy := y + h - 30
		for yy := y; yy < gy; yy++ {
			f := float64(yy-y) / float64(gy-y)
			for xx := x; xx < x+w; xx++ {
				col := t[1]
				if dither(f*1.1, xx, yy) {
					col = t[2]
				}
				c.put(xx, yy, col)
			}
		}
		c.ellipse(x+w-34, y+20, 9, 9, t[3])
		c.ellipse(x+w-31, y+18, 2, 2, t[2])
		for bx, i := 0, 0; bx < w; i++ {
			hh := paperHash(i, 5, uint64(seed))
			bw := minI(12+int(hh%12), w-bx)
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
		c.dfill(x, gy, w, y+h-gy, 0.4, t[2], t[1])
		// the police tape: two posts in the front and a striped band between them (no line across the whole photo)
		px0, px1 := x+20, x+w-22
		for _, px := range []int{px0, px1} {
			c.fill(px-1, y+h-34, 3, 30, t[0])
			c.fill(px-3, y+h-5, 7, 2, t[0])
		}
		for xx := px0 + 2; xx < px1-1; xx++ {
			sag := (xx - px0) * (px1 - xx) * 6 / ((px1 - px0) * (px1 - px0) / 4)
			yy := y + h - 32 + sag
			for k := 0; k < 4; k++ {
				col := t[3]
				if ((xx+k)/5)%2 == 1 {
					col = t[0]
				}
				c.put(xx, yy+k, col)
			}
		}
		c.figure(x+w/2-24, y+h-5, true, false)
		if two {
			c.figure(x+w/2+26, y+h-5, false, false)
		}
	case sceneDance:
		c.fill(x, y, w, h, t[0])
		cx := x + w/2
		fy := y + h - 26
		for k := -2; k <= 2; k++ { // soft cones of light from the ball
			if k == 0 {
				continue
			}
			bx := cx + k*44
			for yy := y + 24; yy < fy; yy++ {
				f := float64(yy-y-24) / float64(fy-y-24)
				half := int(2 + f*12)
				mid := cx + int(float64(bx-cx)*f)
				for xx := mid - half; xx <= mid+half; xx++ {
					if dither(0.3, xx, yy) {
						c.put(xx, yy, t[1])
					}
				}
			}
		}
		for r := 0; r < 5; r++ {
			ry := fy + r*6
			tile := 10 + r*3
			for xx := 0; xx < w; xx += tile {
				col := t[1]
				if ((xx/tile)+r)%2 == 0 {
					col = t[2]
				}
				c.fill(x+xx, ry, tile, 6, col)
			}
		}
		c.fill(cx, y, 1, 10, t[2])
		c.ellipse(cx, y+17, 8, 8, t[3])
		for k := -4; k <= 4; k += 4 { // the mirror tiles, inside the ball only
			c.fill(cx+k, y+11, 1, 13, t[2])
			c.fill(cx-6, y+17+k, 13, 1, t[2])
		}
		xs := []int{x + 40, x + 76, x + 118, x + 154}
		if two {
			xs = []int{x + 52, x + 98, x + 146}
		}
		for i, fx := range xs {
			c.figureC(fx+1, y+h-5, i == 1, true, t[2]) // a lit rim so the dancers stand out from the dark
			c.figure(fx, y+h-6, i == 1, true)
		}
	case scenePool:
		c.dfill(x, y, w, 34, 0.5, t[2], t[3])
		c.fill(x, y+34, w, 6, t[0])
		c.dfill(x, y+40, w, h-40, 0.25, t[1], t[2])
		for i := 0; i < 14; i++ { // a few ripples
			hh := paperHash(i, 17, uint64(seed))
			rx, ry := x+int(hh%uint64(w-12)), y+46+int((hh>>9)%uint64(h-52))
			c.fill(rx, ry, 5+int((hh>>20)%6), 1, t[3])
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
		c.fill(dx+9, dy-17, 6, 2, t[0]) // a tiny hat: the duck came dressed for the party
		c.fill(dx+10, dy-21, 4, 4, t[0])
	case sceneCone:
		c.room(x, y, w, h, 82)
		c.fill(x+12, y+14, 34, 26, t[0]) // a picture on the wall
		c.dfill(x+14, y+16, 30, 22, 0.5, t[1], t[2])
		gx, by := x+w/2-10, y+h-8
		c.figure(gx, by, false, true)
		// the cone over the head: stripes and a wide base
		c.tri(gx-7, by-21, gx, by-44, gx+7, by-21, t[3])
		c.fill(gx-9, by-22, 19, 3, t[3])
		c.fill(gx-9, by-22, 19, 1, t[0])
		c.fill(gx-4, by-30, 9, 2, t[0])
		c.fill(gx-2, by-37, 5, 2, t[0])
		c.fill(gx+8, by-30, 5, 6, t[3]) // a cup in the hand
		// a friend pointing and laughing
		fx := x + w/2 + 44
		c.figure(fx, by, false, false)
		c.fill(fx-14, by-20, 9, 2, t[0])
		for i := 0; i < 3; i++ {
			c.fill(fx+6+i*3, by-36-i*3, 2, 2, t[0]) // "ha ha" puffs
		}
		if two {
			c.figure(x+30, by, true, false)
		}
	case sceneDog:
		c.room(x, y, w, h, 74)
		// a low table with canapés
		c.fill(x+20, y+70, 70, 4, t[0])
		c.fill(x+24, y+74, 3, 26, t[0])
		c.fill(x+83, y+74, 3, 26, t[0])
		for i := 0; i < 5; i++ {
			c.fill(x+26+i*12, y+66, 7, 4, t[3])
			c.fill(x+26+i*12, y+66, 7, 1, t[1])
		}
		// the dog: body, legs, tail, head with an ear, and a party hat
		dx, dy := x+w/2+40, y+h-30
		c.ellipse(dx, dy, 22, 10, t[0])
		for _, lx := range []int{-16, -8, 8, 16} {
			c.fill(dx+lx, dy+6, 4, 16, t[0])
		}
		c.fill(dx+20, dy-12, 3, 10, t[0])
		hx, hy := dx-22, dy-12
		c.ellipse(hx, hy, 10, 8, t[0])
		c.fill(hx-18, hy+1, 10, 6, t[0]) // snout
		c.fill(hx+2, hy-2, 5, 12, t[1])  // ear
		c.fill(hx-6, hy-3, 2, 2, t[3])   // eye
		c.tri(hx-7, hy-7, hx-1, hy-30, hx+5, hy-7, t[3])
		c.fill(hx-4, hy-16, 7, 2, t[1])
		c.ellipse(hx-1, hy-31, 2, 2, t[1])
		c.fill(hx-21, hy+2, 6, 3, t[3]) // a canapé in its mouth
		if two {
			c.figure(x+12, y+h-6, false, true)
		}
	case sceneCake:
		c.room(x, y, w, h, 60)
		// the table, with legs
		c.fill(x+8, y+72, w-16, 5, t[0])
		c.fill(x+14, y+77, 4, h-80, t[0])
		c.fill(x+w-18, y+77, 4, h-80, t[0])
		// the squashed cake: a flat slab with a layer of jam, cream spread around and a shoe print on top
		cx, cy := x+w/2, y+66
		c.fill(cx-32, cy-4, 64, 10, t[3])
		c.fill(cx-32, cy, 64, 2, t[1])
		c.ellipse(cx, cy-5, 40, 3, t[3])
		for i := 0; i < 6; i++ { // cream dripping over the edge
			hh := paperHash(i, 23, uint64(seed))
			c.fill(cx-30+i*12+int(hh%5), cy+6, 3, 2+int((hh>>4)%4), t[3])
		}
		c.ellipse(cx-8, cy-5, 7, 2, t[0]) // the sole
		c.ellipse(cx+8, cy-5, 4, 2, t[0])
		c.fill(cx+22, cy-16, 2, 11, t[2]) // a bent candle
		c.fill(cx+24, cy-18, 6, 2, t[2])
		c.fill(cx+29, cy-21, 2, 3, t[0])
		// the guilty party, walking off with one creamy shoe
		c.figure(x+w-26, y+h-4, false, false)
		c.fill(x+w-30, y+h-6, 4, 2, t[3])
		if two {
			c.figure(x+22, y+h-4, true, true)
		}
	case sceneDJ:
		c.fill(x, y, w, h, t[1])
		c.dfill(x, y, w, 60, 0.3, t[1], t[0])
		// speakers
		for _, sx := range []int{x + 8, x + w - 38} {
			c.fill(sx, y+40, 30, 70, t[0])
			c.ellipse(sx+15, y+60, 9, 9, t[2])
			c.ellipse(sx+15, y+60, 4, 4, t[0])
			c.ellipse(sx+15, y+90, 11, 11, t[2])
			c.ellipse(sx+15, y+90, 5, 5, t[0])
		}
		// the desk with two turntables
		dx := x + w/2
		c.fill(dx-54, y+74, 108, 40, t[0])
		c.fill(dx-54, y+74, 108, 3, t[2])
		for _, px := range []int{dx - 30, dx + 30} {
			c.ellipse(px, y+70, 16, 4, t[2])
			c.ellipse(px, y+70, 3, 1, t[0])
		}
		// the DJ, asleep with the head on the desk, headphones on
		c.fill(dx-16, y+58, 24, 16, t[0])  // the hunched back
		c.fill(dx+4, y+70, 24, 4, t[0])    // an arm as a pillow
		c.ellipse(dx+16, y+64, 7, 6, t[0]) // the head
		c.fill(dx+9, y+56, 14, 2, t[2])    // the headphones' band
		c.ellipse(dx+16, y+64, 3, 3, t[2]) // and one ear cup
		c.fill(dx+20, y+68, 4, 1, t[3])    // a little drool
		// "Z z z" rising
		for i, s := range []int{6, 5, 4} {
			zx, zy := dx+16+i*10, y+42-i*12
			c.fill(zx, zy, s, 1, t[3])
			for k := 0; k < s; k++ {
				c.put(zx+s-1-k, zy+k, t[3])
			}
			c.fill(zx, zy+s-1, s, 1, t[3])
		}
		if two {
			c.figure(x+56, y+h-2, true, true)
		}
	default: // sceneSofa
		c.room(x, y, w, h, 80)
		// a standing lamp
		c.fill(x+20, y+30, 2, 62, t[0])
		c.tri(x+10, y+30, x+21, y+14, x+32, y+30, t[3])
		c.fill(x+14, y+92, 16, 3, t[0])
		// the sofa
		sx := x + 48
		c.fill(sx, y+60, 120, 18, t[0])
		c.fill(sx-6, y+64, 10, 30, t[0])
		c.fill(sx+116, y+64, 10, 30, t[0])
		c.fill(sx, y+78, 120, 14, t[1])
		c.fill(sx+4, y+92, 4, 6, t[0])
		c.fill(sx+112, y+92, 4, 6, t[0])
		// the guest, upside down: the head on the seat, legs over the back
		gx := sx + 60
		c.ellipse(gx-30, y+84, 5, 5, t[3])
		c.ellipse(gx-30, y+84, 3, 3, t[2])
		c.fill(gx-26, y+80, 30, 8, t[2])
		c.fill(gx+2, y+48, 5, 34, t[2])
		c.fill(gx+9, y+44, 5, 38, t[2])
		c.fill(gx, y+44, 8, 4, t[0]) // shoes
		c.fill(gx+8, y+40, 8, 4, t[0])
		// a pizza box on the floor
		c.fill(x+w-44, y+h-10, 30, 4, t[3])
		c.fill(x+w-44, y+h-10, 30, 1, t[1])
		if two {
			c.figure(x+w-16, y+h-6, false, true)
		}
	}
}
