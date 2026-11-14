package main

import "math"

// ---------------------------------------------------------------------------
// The painted backdrop of PARÍS (paris.go): 960 x parBackH, made once, at night. The deep blue sky with stars and the
// moon; Montmartre on the left of the middle with the white domes of the Sacré-Cœur on its top; the Arc de Triomphe
// at the end of an avenue lined with trees (right); and in front the row of Haussmann buildings: cream stone, the
// zinc mansard roofs (blue grey) with their dormers and chimney pots, the long iron balconies of the second and the
// top floor, warm lit windows here and there. The ground floor (the shopfronts) is the real wall under the backdrop.
// The top left is under the HUD. Drawn by code in the palette of the game; no picture is copied.
// ---------------------------------------------------------------------------

var (
	parSkyTop = rgb(0x0e1430)
	parSkyLow = rgb(0x3a3260)
	parStone  = rgb(0xb8aa8a)
	parStoneD = rgb(0x8a7e66)
	parStoneL = rgb(0xd0c4a2)
	parZinc   = rgb(0x5a6478)
	parZincD  = rgb(0x3e4656)
	parZincL  = rgb(0x8a94a8)
	parIron   = rgb(0x16161c)
	parWinOn  = rgb(0xf0c870)
	parWinOff = rgb(0x26283a)
)

// parAvenue: the gap of the avenue between the buildings, where the arch shows (x0, x1).
var parAvenue = [2]int{604, 704}

func parisBackdrop() *pixBuf {
	if parBackBuf != nil {
		return parBackBuf
	}
	W, H := 960, int(parBackH)
	b := newPix(W, H)
	for y := 0; y < H; y++ {
		b.rect(0, y, W, 1, mixc(parSkyTop, parSkyLow, float64(y)/float64(H)))
	}
	for i := 0; i < 60; i++ { // the stars
		x, y := phHash(i, 61)%W, phHash(i, 67)%30
		c := rgb(0xc8d0f0)
		if i%4 == 0 {
			c = rgb(0xffffff)
		}
		b.set(x, y, c)
	}
	b.ellipse(560, 10, 6, 6, rgb(0xf4f0d8)) // the moon
	b.ellipse(563, 9, 5, 5, mixc(parSkyTop, parSkyLow, 0.12))
	// Montmartre and the Sacré-Cœur (white domes)
	for x := 270; x < 470; x++ {
		top := 22 + 14*math.Pow((float64(x)-370)/100, 2)
		b.rect(x, int(top), 1, H, rgb(0x2a3040))
	}
	sc := 370
	white, whiteD := rgb(0xf0ece2), rgb(0xb8b4c0)
	b.rect(sc-22, 14, 44, 10, white)
	b.rect(sc+12, 14, 10, 10, whiteD)
	dome := func(cx, top, r int) {
		for y := top; y < top+r; y++ {
			hw := int(float64(r) * math.Sqrt(math.Max(0, 1-math.Pow(float64(top+r-y)/float64(r), 2))))
			b.rect(cx-hw, y, 2*hw, 1, white)
			if hw > 1 {
				b.rect(cx+hw/2, y, hw-hw/2, 1, whiteD)
			}
		}
		b.rect(cx, top-3, 1, 3, white)
	}
	dome(sc, 2, 9)
	dome(sc-16, 8, 5)
	dome(sc+16, 8, 5)
	b.rect(sc+24, 0, 4, 16, white) // the bell tower
	b.ellipse(float64(sc+26), 0, 2, 2, white)
	for x := sc - 18; x < sc+18; x += 5 {
		b.rect(x, 18, 2, 4, parWinOff)
	}
	// the avenue with its trees and the Arc de Triomphe at its end
	ax := (parAvenue[0] + parAvenue[1]) / 2
	b.rect(ax-24, 10, 48, 34, parStoneL)
	b.rect(ax-24, 10, 48, 4, parStone)
	b.rect(ax-24, 16, 48, 1, parStoneD)
	b.rect(ax+14, 10, 10, 34, parStone)
	for y := 22; y < 44; y++ { // the opening
		for x := ax - 9; x <= ax+9; x++ {
			dx, dy := float64(x-ax)/9, float64(y-30)/8
			if y < 30 && dx*dx+dy*dy > 1 {
				continue
			}
			b.set(x, y, mixc(parSkyLow, rgb(0x6a5a7a), 0.4))
		}
	}
	for _, s := range []int{-1, 1} { // the trees of the avenue
		for i := 0; i < 4; i++ {
			x := ax + s*(30+i*5)
			b.ellipse(float64(x), float64(44+i*6), 5+float64(i), 4+float64(i), rgb(0x24402e))
		}
	}
	b.rect(parAvenue[0], 66, parAvenue[1]-parAvenue[0], H-66, rgb(0x4a4652)) // the avenue itself
	// the row of Haussmann buildings, interrupted by the avenue
	x := 0
	for i := 0; x < W; i++ {
		w := 70 + phHash(i, 71)%40
		if x < parAvenue[1] && x+w > parAvenue[0] {
			if x < parAvenue[0] {
				w = parAvenue[0] - x
			} else {
				x = parAvenue[1]
				continue
			}
		}
		parHaussmann(b, x, w, i)
		x += w
	}
	parBackBuf = b
	return b
}

// parHaussmann: one building from x0, w wide, standing on the bottom of the backdrop.
func parHaussmann(b *pixBuf, x0, w, seed int) {
	H := int(parBackH)
	roof := 26 + phHash(seed, 73)%5 // the top of the mansard
	facade := roof + 12             // where the stone starts
	// the stone facade with its joints
	b.rect(x0, facade, w, H-facade, parStone)
	for y := facade + 3; y < H; y += 4 {
		b.rect(x0, y, w, 1, shade(parStone, 0.92))
	}
	b.rect(x0, facade, 1, H-facade, parStoneD)
	b.rect(x0+w-1, facade, 1, H-facade, parStoneL)
	b.rect(x0, facade, w, 2, parStoneL) // the cornice
	// the zinc mansard: a sloping roof, its dormers and the chimney pots
	for y := roof; y < facade; y++ {
		in := (facade - y) / 3
		b.rect(x0+in, y, w-2*in, 1, mixc(parZincL, parZinc, float64(y-roof)/12))
	}
	b.rect(x0+4, roof-1, w-8, 1, parZincD)
	for dx := 8; dx+8 < w; dx += 16 {
		b.rect(x0+dx, roof+3, 6, 7, parStoneL) // the dormer
		b.rect(x0+dx+1, roof+5, 4, 5, parWinOff)
		if phHash(seed*7+dx, 79)%3 == 0 {
			b.rect(x0+dx+1, roof+5, 4, 5, parWinOn)
		}
		b.rect(x0+dx-1, roof+2, 8, 1, parZincD)
	}
	for k := 0; k < 2; k++ {
		cxp := x0 + 6 + phHash(seed, 83+k)%maxI(1, w-14)
		b.rect(cxp, roof-6, 6, 6, rgb(0x9a6a52))
		b.rect(cxp, roof-6, 6, 1, rgb(0xb8846a))
		for p := 0; p < 3; p++ {
			b.rect(cxp+p*2, roof-8, 1, 2, rgb(0xc8784a)) // the pots
		}
	}
	// the windows, tall, in rows; the long iron balconies of the second and the top floor
	rows := []int{facade + 5, facade + 15, facade + 25}
	for r, wy := range rows {
		if wy+7 > H {
			break
		}
		for dx := 5; dx+5 < w; dx += 9 {
			c := parWinOff
			if phHash(seed*13+dx*3+r, 89)%4 == 0 {
				c = parWinOn
			}
			b.rect(x0+dx, wy, 4, 7, c)
			b.rect(x0+dx-1, wy-1, 6, 1, parStoneL)
		}
		if r == 0 || r == 2 {
			b.rect(x0+2, wy+7, w-4, 1, parIron)
			for bx := x0 + 2; bx < x0+w-2; bx += 2 {
				b.set(bx, wy+6, parIron)
			}
		}
	}
}
