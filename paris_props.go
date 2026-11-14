package main

import (
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// Props of PARÍS (paris.go). All drawn by code in the palette of the game:
//   par_metro        the art nouveau metro entrance: green cast-iron stems curling into two orange lamps, the sign
//                    and the stairs going down behind a green railing
//   par_kiosk        the green newspaper kiosk with its little dome
//   par_bouquiniste  the green boxes of the booksellers of the quays, open, full of books and prints (2 variants)
//   par_bakery       the bakery stall: blue and white awning, baguettes standing in baskets, croissants
//   par_bistro       a bistro table of marble with two wicker chairs (guests sit on them; 2 variants)
//   par_lamp         a period street lamp (dark green, a lantern of four panes)
//   par_busker       the street artists (animated): 0 the accordion player (the bellows open and close), 1 the mime
//                    (his hands feel the invisible wall)
// The carousel is the one of the funfair; the French flags are flags.go.
// ---------------------------------------------------------------------------

var (
	parGreen  = rgb(0x3a7058)
	parGreenD = rgb(0x24483a)
	parGreenL = rgb(0x62a07e)
	parCream  = rgb(0xf0e6c8)
)

const parBuskerFrames = 4

func init() {
	pk := propKinds
	pk["par_metro"] = &PropKind{W: 60, H: 62, FW: 46, FH: 14, Draw: drawParMetro}
	pk["par_kiosk"] = &PropKind{W: 44, H: 68, FW: 36, FH: 20, Hide: true, Draw: drawParKiosk}
	pk["par_bouquiniste"] = &PropKind{W: 48, H: 30, FW: 44, FH: 12, Vars: 2, Draw: drawParBouquiniste}
	pk["par_bakery"] = &PropKind{W: 70, H: 62, FW: 64, FH: 16, Vars: 2, Hide: true, UseKind: "drink", Use: []Vec{{0, 12}},
		Draw: drawParBakery}
	pk["par_bistro"] = &PropKind{W: 44, H: 30, FW: 38, FH: 10, Vars: 2, UseKind: "lounge", Use: []Vec{{-14, 10}, {14, 10}},
		Draw: drawParBistro}
	pk["par_lamp"] = &PropKind{W: 14, H: 56, FW: 6, FH: 4, Light: true, Draw: drawParLamp}
	pk["par_busker"] = &PropKind{W: 26, H: 40, FW: 12, FH: 6, Vars: 2, Draw: func(b *pixBuf, v int) { drawParBusker(b, v, 0) }}
	propAnims["par_busker"] = &propAnim{n: parBuskerFrames, fps: 4, draw: func(b *pixBuf, k *PropKind, v, f int) { drawParBusker(b, v, f) }}
	// the letters the bakery's sign needs that the sign font of GUADALAJARA does not have
	for r, g := range map[rune][]string{
		'B': {"##.", "#.#", "##.", "#.#", "##."},
		'O': {".#.", "#.#", "#.#", "#.#", ".#."},
		'U': {"#.#", "#.#", "#.#", "#.#", "###"},
		'G': {".##", "#..", "#.#", "#.#", ".##"},
	} {
		if _, ok := gdlGlyphs[r]; !ok {
			gdlGlyphs[r] = g
		}
	}
}

// ---- the metro entrance ----------------------------------------------------------------------------------------

func drawParMetro(b *pixBuf, v int) {
	W, H := 60, 62
	b.ellipse(float64(W)/2, float64(H)-2, 28, 4, color.RGBA{0, 0, 0, 60})
	// the stairs going down (seen from above: steps getting darker)
	for i := 0; i < 6; i++ {
		b.rect(10, H-16+i*2, W-20, 2, shade(rgb(0x8a8270), 1-float64(i)*0.12))
	}
	// the railing round the back and the sides (green iron with little curls)
	for x := 8; x < W-8; x++ {
		b.set(x, H-18, parGreenD)
		if x%4 == 0 {
			b.rect(x, H-22, 1, 4, parGreen)
			b.set(x+1, H-22, parGreenL)
		}
	}
	b.rect(8, H-18, 2, 16, parGreen)
	b.rect(W-10, H-18, 2, 16, parGreen)
	// the two stems curling up like plants, with their orange lamps
	for _, s := range []int{-1, 1} {
		x0 := W/2 + s*22
		for y := H - 4; y > 14; y-- {
			t := float64(H-4-y) / float64(H-18)
			x := x0 - s*int(math.Round(6*t*t))
			b.set(x, y, parGreen)
			b.set(x+s, y, parGreenD)
		}
		lx := x0 - s*6
		b.ellipse(float64(lx), 12, 3, 4, rgb(0xf08a2a))
		b.ellipse(float64(lx), 11, 2, 2.5, rgb(0xffc060))
		b.rect(lx-1, 16, 3, 2, parGreenD)
	}
	// the sign between them
	b.rect(W/2-16, 18, 32, 9, parCream)
	b.rect(W/2-16, 18, 32, 1, parGreenL)
	b.rect(W/2-16, 26, 32, 1, parGreenD)
	gdlTextC(b, "METRO", W/2, 20, parGreenD)
	b.rect(W/2-17, 27, 34, 2, parGreen) // the arch that holds it
}

// ---- the newspaper kiosk ---------------------------------------------------------------------------------------

func drawParKiosk(b *pixBuf, v int) {
	W, H := 44, 68
	b.ellipse(float64(W)/2, float64(H)-2, 20, 3, color.RGBA{0, 0, 0, 60})
	b.cube(5, 30, W-10, H-32, 8, parGreenD, parGreen)
	for r := 0; r < 2; r++ { // the magazines on the front
		for i := 0; i < 4; i++ {
			c := rgb([]uint32{0xf0e8d0, 0xd84a4a, 0x4a7ad8, 0xf0c040}[(i+r)%4])
			b.rect(8+i*7, 42+r*10, 6, 8, c)
			b.rect(9+i*7, 43+r*10, 4, 2, shade(c, 0.6))
		}
	}
	// the roof: a wide green brim, the little dome and the finial
	b.rect(1, 26, W-2, 4, parGreenD)
	b.rect(1, 26, W-2, 1, parGreenL)
	for y := 8; y < 26; y++ {
		hw := int(16 * math.Sqrt(math.Max(0, 1-math.Pow(float64(26-y)/18, 2))))
		b.rect(W/2-hw, y, 2*hw, 1, mixc(parGreenL, parGreen, float64(y-8)/18))
	}
	b.rect(W/2-1, 2, 2, 6, parGreenD)
	b.ellipse(float64(W)/2, 3, 2, 2, rgb(0xd8b84a))
	b.rect(8, 20, W-16, 5, parCream) // the band of the sign
	gdlTextC(b, "PRESSE", W/2, 20, parGreenD)
}

// ---- the bouquinistes ------------------------------------------------------------------------------------------

func drawParBouquiniste(b *pixBuf, v int) {
	W, H := 48, 30
	b.ellipse(float64(W)/2, float64(H)-2, 23, 3, color.RGBA{0, 0, 0, 60})
	// the open lid, leaning back
	b.rect(3, 2, W-6, 10, parGreen)
	b.rect(3, 2, W-6, 1, parGreenL)
	if v == 0 { // prints pinned inside the lid
		for i := 0; i < 3; i++ {
			b.rect(7+i*13, 4, 10, 7, parCream)
			b.rect(8+i*13, 5, 8, 4, rgb([]uint32{0x6a8ab8, 0xc87a5a, 0x7aa86a}[i]))
		}
	}
	// the box with the books standing in it
	b.rect(2, 12, W-4, H-14, parGreenD)
	b.rect(2, 12, W-4, 2, parGreen)
	for x := 4; x < W-4; x += 3 {
		c := rgb([]uint32{0xc84a3a, 0x3a5a9a, 0xe8d8b0, 0x7a5a3a, 0x4a8a5a, 0xd8a040}[(x/3+v)%6])
		h := 6 + phHash(x, v)%3
		b.rect(x, 18-h+4, 2, h, c)
	}
	b.rect(2, 20, W-4, 1, parGreenL)
}

// ---- the bakery ------------------------------------------------------------------------------------------------

func drawParBakery(b *pixBuf, v int) {
	W, H := 70, 62
	wood, woodD := rgb(0x9a6a40), rgb(0x6a4628)
	bread, breadD := rgb(0xe0a85a), rgb(0xa86a2a)
	b.ellipse(float64(W)/2, float64(H)-2, 33, 4, color.RGBA{0, 0, 0, 60})
	// the counter
	b.cube(4, 34, W-8, H-36, 8, woodD, wood)
	// baskets of baguettes standing up, and croissants on the counter
	for i := 0; i < 3; i++ {
		bx := 10 + i*18
		for k := 0; k < 4; k++ {
			x := bx + k*3
			b.rect(x, 22-k%2*2, 2, 16, bread)
			b.rect(x, 22-k%2*2, 2, 1, rgb(0xf0c880))
			for y := 24; y < 36; y += 3 {
				b.set(x, y-k%2*2, breadD) // the cuts of the crust
			}
		}
		b.rect(bx-2, 32, 14, 6, rgb(0xb88a50)) // the wicker basket
		for x := bx - 2; x < bx+12; x += 2 {
			b.set(x, 34, rgb(0x7a5428))
		}
	}
	for i := 0; i < 4; i++ {
		x := 8 + i*15 + v*4
		b.ellipse(float64(x), 44, 4, 2, bread)
		b.set(x, 44, breadD)
		b.set(x-2, 44, breadD)
	}
	// the awning (blue and white) on two poles, the sign above
	for y := 10; y < 18; y++ {
		for x := 0; x < W; x++ {
			c := rgb(0x2a4a9a)
			if (x/6)%2 == 0 {
				c = rgb(0xf2f0ea)
			}
			b.set(x, y, shade(c, 1-float64(y-10)*0.03))
		}
	}
	for x := 0; x < W; x += 6 {
		b.rect(x, 18, 6, 2, rgb(0x2a4a9a))
		b.set(x+3, 19, rgb(0xf2f0ea))
	}
	b.rect(2, 18, 2, 18, woodD)
	b.rect(W-4, 18, 2, 18, woodD)
	b.rect(6, 1, W-12, 8, parCream)
	b.rect(6, 1, W-12, 1, rgb(0x2a4a9a))
	gdlTextC(b, "BOULANGERIE", W/2, 3, rgb(0x2a4a9a))
}

// ---- the bistro table with its wicker chairs -------------------------------------------------------------------

func drawParBistro(b *pixBuf, v int) {
	W, H := 44, 30
	cane, caneD := rgb(0xc8a46a), rgb(0x8a6a3a)
	seat := []color.RGBA{rgb(0xb83a3a), rgb(0x2a6a5a)}[v%2]
	b.ellipse(float64(W)/2, float64(H)-3, 20, 3, color.RGBA{0, 0, 0, 55})
	// the two wicker chairs, facing the table
	for _, x0 := range []int{2, W - 14} {
		for y := 6; y < 22; y++ {
			for x := x0; x < x0+12; x++ {
				if y < 14 && (x == x0 || x == x0+11 || y == 6) || y >= 14 && y < 17 {
					c := cane
					if (x+y)%2 == 0 {
						c = caneD // the weave
					}
					b.set(x, y, c)
				} else if y < 14 && (x+y)%3 == 0 {
					b.set(x, y, caneD)
				}
			}
		}
		b.rect(x0+1, 14, 10, 2, seat) // the cushion
		b.rect(x0+1, 17, 1, 9, caneD)
		b.rect(x0+10, 17, 1, 9, caneD)
	}
	// the round marble table on its black iron foot
	b.rect(W/2-1, 16, 2, 10, rgb(0x22222a))
	b.rect(W/2-4, 25, 8, 1, rgb(0x22222a))
	b.ellipse(float64(W)/2, 14, 8, 3, rgb(0xd8d4cc))
	b.ellipse(float64(W)/2, 13, 7, 2, rgb(0xf4f2ec))
	b.rect(W/2-3, 11, 2, 2, rgb(0x6a3a2a)) // a little cup of coffee
	b.set(W/2+2, 12, rgb(0xd8b84a))
}

// ---- the period street lamp ------------------------------------------------------------------------------------

func drawParLamp(b *pixBuf, v int) {
	W, H := 14, 56
	post, postL := rgb(0x24382e), rgb(0x46604e)
	b.ellipse(float64(W)/2, float64(H)-2, 6, 2, color.RGBA{0, 0, 0, 60})
	b.rect(W/2-3, H-8, 6, 6, post) // the fluted base
	b.rect(W/2-3, H-8, 1, 6, postL)
	b.rect(W/2-1, 16, 2, H-24, post)
	b.rect(W/2-1, 16, 1, H-24, postL)
	for y := H - 30; y < H-10; y += 6 {
		b.rect(W/2-2, y, 4, 1, postL) // the rings
	}
	b.rect(W/2-4, 14, 8, 2, post) // the arms under the lantern
	// the lantern: four panes of warm light under a little roof
	b.rect(W/2-4, 5, 8, 9, rgb(0xffd890))
	b.rect(W/2-4, 5, 1, 9, post)
	b.rect(W/2+3, 5, 1, 9, post)
	b.rect(W/2, 5, 1, 9, rgb(0xe8b860))
	b.rect(W/2-5, 3, 10, 2, post)
	b.rect(W/2-1, 0, 2, 3, post)
}

// ---- the street artists ----------------------------------------------------------------------------------------

func drawParBusker(b *pixBuf, v, f int) {
	W, H := 26, 40
	skin, skinD := rgb(0xe8bc94), rgb(0xb88a64)
	cx := W / 2
	b.ellipse(float64(cx), float64(H)-2, 8, 2, color.RGBA{0, 0, 0, 60})
	// legs and shoes
	b.rect(cx-3, 28, 2, 10, rgb(0x22222a))
	b.rect(cx+1, 28, 2, 10, rgb(0x22222a))
	b.rect(cx-4, 37, 3, 2, rgb(0x101014))
	b.rect(cx+1, 37, 3, 2, rgb(0x101014))
	// the striped shirt (both wear the sailor stripes)
	for y := 15; y < 29; y++ {
		c := rgb(0xf2f0ea)
		if y%3 == 0 {
			c = rgb(0x1e2a5a)
		}
		b.rect(cx-4, y, 8, 1, c)
	}
	// the head and the black beret
	face := skin
	if v == 1 {
		face = rgb(0xf6f4f0) // the mime's white face
	}
	b.rect(cx-3, 6, 6, 8, face)
	b.rect(cx-3, 12, 6, 2, shade(face, 0.9))
	b.rect(cx-4, 4, 8, 3, rgb(0x16161c))
	b.set(cx+3, 3, rgb(0x16161c))
	b.set(cx-2, 9, rgb(0x16161c)) // the eyes
	b.set(cx+1, 9, rgb(0x16161c))
	if v == 1 {
		b.rect(cx-1, 12, 2, 1, rgb(0xc82a3a)) // red lips
		// the hands on the invisible wall, sliding up and down
		dy := []int{0, -2, -3, -1}[f%parBuskerFrames]
		for _, s := range []int{-1, 1} {
			x := cx + s*7
			b.rect(x-1+(1-s)/2, 16+dy, 2, 8, rgb(0xf2f0ea))
			b.rect(x-1, 12+dy, 3, 4, rgb(0xfafafa)) // white gloves
		}
		return
	}
	b.rect(cx-3, 12, 6, 1, skinD) // the moustache
	// the accordion: two red boxes, the bellows between them opening and closing
	open := []int{2, 5, 8, 6}[f%parBuskerFrames] // opening wide, closing a little slower
	left, right := cx-3-open/2, cx+3+(open+1)/2
	b.rect(left-4, 17, 4, 10, rgb(0xb8282a))
	b.rect(left-4, 17, 4, 1, rgb(0xe85a4a))
	for y := 19; y < 26; y += 2 {
		b.set(left-3, y, rgb(0xf0f0f0)) // the buttons
	}
	b.rect(right, 17, 4, 10, rgb(0xb8282a))
	b.rect(right, 17, 4, 1, rgb(0xe85a4a))
	for x := left; x < right; x++ {
		for y := 18; y < 26; y++ {
			c := rgb(0x2a2a30)
			if (x-left)%2 == 0 {
				c = rgb(0xe8e0d0)
			}
			b.set(x, y, c)
		}
	}
	b.rect(left-5, 20, 2, 3, skin) // the hands on the boxes
	b.rect(right+3, 20, 2, 3, skin)
}
