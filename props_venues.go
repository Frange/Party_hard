package main

import (
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// Props of the three parties added before the space station: the BIRTHDAY party, the HOCKEY rink and the 70s DISCO.
// Everything is drawn by code (no letters: the signs are patterns), generic parody, no brands or real people.
// ---------------------------------------------------------------------------

func init() {
	pk := propKinds

	// ---- BIRTHDAY ----------------------------------------------------------------------------------------------

	// bdaycake: the giant cake on its round table (3 flavours)
	pk["bdaycake"] = &PropKind{W: 64, H: 64, FW: 50, FH: 16, Vars: 3, Draw: func(b *pixBuf, v int) {
		sponge := []uint32{0xf2d6a0, 0x6a3c22, 0xf4c8d8}[v%3]
		icing := []uint32{0xf4a0c8, 0xf4ecd8, 0x8ed0f0}[v%3]
		berry := []uint32{0xd02840, 0xd02840, 0xe04a5a}[v%3]
		b.ellipse(32, 60, 28, 4, c(0x1c1620))              // shadow
		b.rect(10, 50, 44, 9, c(0xf4f0f8))                 // table cloth
		b.rect(10, 55, 44, 2, c(0xe04a7a))                 // cloth trim
		b.ellipse(32, 50, 24, 5, c(0xfaf6fc))              // table top
		b.rect(12, 34, 40, 15, c(sponge))                  // lower tier
		b.ellipse(32, 49, 20, 4.5, shade(c(sponge), 0.82)) // its round bottom
		b.ellipse(32, 34, 20, 5, c(icing))                 // icing on top
		for x := 12; x < 52; x += 4 {                      // drips
			b.rect(x, 34, 3, 3+(x*7)%5, c(icing))
		}
		b.rect(12, 41, 40, 1, shade(c(icing), 0.85))
		b.rect(21, 22, 22, 12, shade(c(sponge), 1.08)) // upper tier
		b.ellipse(32, 34, 11, 3.2, shade(c(sponge), 0.8))
		b.ellipse(32, 22, 11, 3.6, c(icing))
		for x := 21; x < 43; x += 4 {
			b.rect(x, 22, 3, 2+(x*5)%4, c(icing))
		}
		for i, cx := range []int{16, 24, 38, 46, 30} { // sprinkles
			b.set(cx, 42+i%3*2, c([]uint32{0xf0d030, 0x40a0e0, 0xe04a5a, 0x60c070, 0xf4a0d0}[i]))
		}
		for i, cx := range []int{24, 28, 32, 36, 40} { // candles with flames
			col := []uint32{0xe04a5a, 0x40a0e0, 0xf0d030, 0x60c070, 0xe880c0}[i]
			b.rect(cx, 13, 2, 8, c(col))
			b.rect(cx, 10, 2, 3, c(0xffd850))
			b.set(cx, 9, c(0xfff4b0))
		}
		b.ellipse(32, 20, 2.2, 1.6, c(berry)) // cherry
	}}

	// giftable: a table full of wrapped presents
	pk["giftable"] = &PropKind{W: 46, H: 36, FW: 42, FH: 12, Vars: 3, Draw: func(b *pixBuf, v int) {
		b.ellipse(23, 34, 21, 3, c(0x1c1620))
		b.cube(2, 18, 42, 16, 4, c(0xf4f0f8), c([]uint32{0x3a6ac8, 0xc03840, 0x2a8a5a}[v%3]))
		b.rect(6, 31, 2, 4, c(0x4a3a30))
		b.rect(38, 31, 2, 4, c(0x4a3a30))
		boxes := [][4]int{{5, 9, 11, 10}, {19, 5, 10, 14}, {31, 10, 11, 9}}
		cols := []uint32{0xe04a5a, 0xf0d030, 0x40a0e0, 0x60c070, 0xe880c0, 0xf0a030}
		for i, bx := range boxes {
			col := c(cols[(i*2+v)%6])
			b.cube(bx[0], bx[1], bx[2], bx[3], 3, shade(col, 1.2), col)
			rb := c(cols[(i*2+v+3)%6])
			b.rect(bx[0]+bx[2]/2, bx[1], 2, bx[3], rb)
			b.rect(bx[0], bx[1]+bx[3]/2, bx[2], 2, rb)
			b.rect(bx[0]+bx[2]/2-2, bx[1]-2, 2, 2, rb)
			b.rect(bx[0]+bx[2]/2+1, bx[1]-2, 2, 2, rb)
		}
	}}

	// entertainer: the animator (v0, funny cone hat) or the magician (v1, tall hat and cape): decoration only
	pk["entertainer"] = &PropKind{W: 24, H: 42, FW: 14, FH: 6, Vars: 2, Draw: func(b *pixBuf, v int) {
		b.ellipse(12, 40, 9, 2.5, c(0x1c1620))
		skin := c(0xf0c8a0)
		if v == 1 { // magician
			b.rect(5, 18, 14, 18, c(0x2a1a4a)) // cape
			b.rect(5, 18, 14, 2, c(0xc83a50))
			b.rect(7, 20, 10, 15, c(0x14101c))
			b.rect(10, 20, 4, 6, c(0xf4f4f8)) // shirt front
			b.rect(7, 35, 4, 4, c(0x14101c))
			b.rect(13, 35, 4, 4, c(0x14101c))
			b.ellipse(12, 14, 4.2, 4.5, skin)
			b.rect(8, 7, 8, 5, c(0x14101c)) // tall hat
			b.rect(5, 12, 14, 2, c(0x14101c))
			b.rect(8, 10, 8, 1, c(0xc83a50))
			b.set(10, 14, c(0x201810))
			b.set(14, 14, c(0x201810))
			b.rect(10, 16, 4, 1, c(0xa0502a))   // moustache
			b.line(19, 24, 22, 17, c(0xe8e8f0)) // wand
			b.set(22, 16, c(0xf0d030))
			b.set(21, 15, c(0xf0d030))
			b.set(23, 15, c(0xf0d030))
			return
		}
		// animator
		b.rect(6, 19, 12, 12, c(0xe04a5a))
		b.rect(6, 23, 12, 2, c(0xf0d030))
		b.rect(6, 27, 12, 2, c(0x40a0e0))
		b.rect(7, 31, 4, 8, c(0x40a0e0))
		b.rect(13, 31, 4, 8, c(0xf0d030))
		b.rect(6, 38, 5, 2, c(0xe04a5a))
		b.rect(13, 38, 5, 2, c(0xe04a5a))
		b.ellipse(12, 14, 4.4, 4.6, skin)
		b.rect(8, 6, 2, 3, c(0xe04a5a)) // cone hat
		b.rect(9, 3, 6, 4, c(0x40a0e0))
		b.rect(10, 1, 4, 3, c(0xf0d030))
		b.rect(7, 9, 10, 2, c(0xe04a5a))
		b.set(11, 0, c(0xffffff))
		b.set(10, 14, c(0x201810))
		b.set(14, 14, c(0x201810))
		b.rect(11, 16, 3, 1, c(0xc03838)) // smile
		b.ellipse(12, 15, 1.2, 1.2, c(0xe04a5a))
		b.rect(3, 19, 3, 6, c(0xe04a5a)) // arms up with a balloon
		b.rect(18, 17, 3, 8, c(0xe04a5a))
		b.ellipse(21, 11, 3, 3.6, c(0xe880c0))
		b.line(21, 14, 20, 18, c(0xd8d8e0))
	}}

	// bunting: a garland of flags hanging across the party (decoration over everything, like the ropes)
	pk["bunting"] = &PropKind{W: 128, H: 18, FW: 4, FH: 4, NoSolid: true, Vars: 2, SortOff: -4, Draw: func(b *pixBuf, v int) {
		cols := []uint32{0xe04a5a, 0xf0d030, 0x40a0e0, 0x60c070, 0xe880c0, 0xf0a030}
		prev := -1
		for x := 0; x < 128; x++ {
			y := int(3 + 5*math.Sin(float64(x)/127*math.Pi))
			b.set(x, y, c(0x7a6a5a))
			if prev >= 0 && x%11 == 5 {
				i := x / 11
				col := c(cols[(i+v*3)%6])
				for k := 0; k < 6; k++ {
					b.rect(x-3+k/2, y+1+k, 7-k, 1, col)
				}
				b.set(x, y+7, shade(col, 0.8))
			}
			prev = y
		}
	}}

	// ---- HOCKEY ------------------------------------------------------------------------------------------------

	// hgoal: the net, facing the ice (v0: the net opens to the right, v1: to the left)
	pk["hgoal"] = &PropKind{W: 34, H: 34, FW: 22, FH: 26, Vars: 2, Draw: func(b *pixBuf, v int) {
		red, mesh := c(0xd03040), c(0xdce6f0)
		x0, x1 := 6, 28 // back / front of the frame
		if v == 1 {
			x0, x1 = 28, 6
		}
		b.ellipse(17, 32, 14, 2.5, c(0x7a98b4))
		for y := 6; y < 31; y++ { // mesh, a back panel
			for x := minI(x0, x1); x <= maxI(x0, x1); x++ {
				if (x+y)%3 == 0 {
					b.set(x, y, mesh)
				}
			}
		}
		b.rect(x0-1, 4, 3, 28, red) // back post
		b.rect(x1-1, 4, 3, 28, red) // front post
		b.rect(minI(x0, x1), 4, absI(x1-x0), 2, red)
		b.rect(minI(x0, x1), 30, absI(x1-x0), 2, red)
	}}

	// scoreboard: the hanging box with four amber digits
	pk["scoreboard"] = &PropKind{W: 62, H: 48, FW: 52, FH: 8, Vars: 3, Draw: func(b *pixBuf, v int) {
		b.ellipse(31, 46, 24, 2.5, c(0x1c1620))
		b.rect(10, 30, 4, 16, c(0x5a5e6a))
		b.rect(48, 30, 4, 16, c(0x5a5e6a))
		b.cube(2, 4, 58, 28, 3, c(0x44485a), c(0x1a1c26))
		b.rect(5, 9, 52, 20, c(0x07080c))
		digits := [][4]int{{2, 1, 3, 2}, {4, 0, 1, 5}, {1, 2, 2, 0}}[v%3]
		x := 8
		for i, d := range digits {
			if i == 2 {
				x += 5
			}
			if i == 1 { // the colon
				b.rect(x+13, 15, 2, 2, c(0xffb030))
				b.rect(x+13, 21, 2, 2, c(0xffb030))
				x += 8
			}
			rinkDigit(b, x+i*0, 13, d, c(0xffb030))
			x += 11
		}
		b.rect(4, 6, 54, 1, c(0xe04a5a))
		b.set(8, 6, c(0xffffff))
		b.set(53, 6, c(0xffffff))
	}}

	// rinkbench: the players' bench behind a glass shield (guests sit on it)
	pk["rinkbench"] = &PropKind{W: 58, H: 28, FW: 54, FH: 10, UseKind: "lounge", Use: []Vec{{-14, 10}, {14, 10}}, Draw: func(b *pixBuf, v int) {
		b.ellipse(29, 26, 27, 2.5, c(0x1c1620))
		b.cube(2, 12, 54, 12, 3, c(0x7a5a3a), c(0x4a3624)) // wooden bench
		b.rect(4, 22, 3, 4, c(0x3a2a1e))
		b.rect(51, 22, 3, 4, c(0x3a2a1e))
		b.rect(2, 2, 54, 10, c(0x9ac4de)) // glass shield
		b.rect(2, 2, 54, 1, c(0xf4fbff))
		b.rect(2, 11, 54, 1, c(0x5a7a94))
		for x := 8; x < 54; x += 14 {
			b.rect(x, 3, 2, 8, c(0xd6ecf8))
		}
		b.rect(25, 14, 8, 6, c(0x2a8ad0)) // water cooler
		b.rect(25, 14, 8, 1, c(0xe8f4ff))
	}}

	// icemachine: the ice resurfacer (a generic big machine with a blue cab)
	pk["icemachine"] = &PropKind{W: 66, H: 40, FW: 58, FH: 18, Vars: 2, Draw: func(b *pixBuf, v int) {
		body := c([]uint32{0x2e70c8, 0xd05a30}[v%2])
		b.ellipse(33, 38, 31, 3, c(0x1c1620))
		b.rect(4, 26, 8, 12, c(0x14141a)) // wheels
		b.rect(54, 26, 8, 12, c(0x14141a))
		b.cube(6, 12, 54, 24, 5, shade(body, 1.2), body) // tank
		b.rect(6, 28, 54, 2, shade(body, 0.7))
		b.rect(10, 4, 18, 14, c(0x3a3e4c)) // cab
		b.rect(11, 5, 16, 8, c(0xa8d8f0))
		b.rect(11, 5, 16, 1, c(0xf4fbff))
		b.rect(34, 2, 3, 12, c(0x5a5e6a)) // exhaust
		b.rect(33, 1, 5, 2, c(0x8a8e9a))
		b.rect(42, 18, 14, 4, c(0xe8e8f0)) // water tank band
		b.rect(36, 30, 20, 3, c(0xf0c030))
		b.set(9, 30, c(0xffe070))
		b.set(57, 30, c(0xff5050))
	}}

	// ---- 70s DISCO ---------------------------------------------------------------------------------------------

	// mirrorball: hangs over the floor (overlay: it does not block the tiles); a different glint per variant
	pk["mirrorball"] = &PropKind{W: 26, H: 46, FW: 4, FH: 4, NoSolid: true, Vars: 3, Light: true, LightR: 70, SortOff: 40, Draw: func(b *pixBuf, v int) {
		b.rect(13, 0, 1, 14, c(0xb8bcc8)) // chain
		cx, cy, r := 13.0, 29.0, 12.0
		for y := -12; y <= 12; y++ {
			for x := -12; x <= 12; x++ {
				if float64(x*x+y*y) > r*r {
					continue
				}
				gx, gy := (x+16)/3, (y+16)/3
				k := 0.6 + 0.4*float64((gx*7+gy*13+v*5)%5)/4 // each mirror has its own brightness
				if x < 0 && y < 0 {
					k += 0.18
				}
				if x > 4 || y > 6 {
					k -= 0.22
				}
				base := c(0xc8d0e0)
				if (x+16)%3 == 0 || (y+16)%3 == 0 {
					base = c(0x6a7284) // the grout of the mirror tiles
				}
				b.set(int(cx)+x, int(cy)+y, shade(base, math.Max(0.4, math.Min(1.3, k))))
			}
		}
		for i, g := range [][2]int{{4, 20}, {20, 24}, {8, 38}, {18, 14}} {
			b.set(g[0], g[1], c([]uint32{0xfff4b0, 0xffffff, 0xb0e8ff, 0xffd0f0}[(i+v)%4]))
		}
	}}

	// lavalamp: warm light and slow blobs (3 colours)
	pk["lavalamp"] = &PropKind{W: 14, H: 36, FW: 10, FH: 6, Vars: 3, Light: true, LightR: 52, LightC: color.RGBA{255, 128, 64, 150}, Draw: func(b *pixBuf, v int) {
		wax := []uint32{0xe8402a, 0x40d070, 0xb040e0}[v%3]
		liquid := []uint32{0x6a1a24, 0x14502a, 0x3a1a60}[v%3]
		b.ellipse(7, 34, 6, 2, c(0x1c1620))
		b.rect(3, 29, 8, 5, c(0xc89a30))
		b.rect(4, 30, 6, 1, c(0xf0d070))
		b.rect(5, 6, 4, 23, c(liquid)) // glass body
		b.rect(4, 12, 6, 14, c(liquid))
		b.rect(5, 6, 1, 22, shade(c(liquid), 1.6))
		b.ellipse(7, 24, 2.6, 3, c(wax))
		b.ellipse(7, 15, 2, 2.4, c(wax))
		b.ellipse(7, 9, 1.4, 1.6, c(wax))
		b.rect(5, 2, 4, 4, c(0xc89a30)) // cap
		b.rect(6, 0, 2, 3, c(0xc89a30))
	}}

	// retropanel: wall panels in the 70s browns, oranges and purples (3 patterns); hung on the wall like a poster
	pk["retropanel"] = &PropKind{W: 56, H: 44, FW: 52, FH: 4, NoSolid: true, Vars: 3, Draw: func(b *pixBuf, v int) {
		pal := [][]uint32{{0xd8761e, 0x7a3e1a, 0xf0b040, 0x5a2a6a}, {0x7a3e1a, 0xd8761e, 0xa04a8a, 0xf0b040}, {0x5a2a6a, 0xd8761e, 0xf0b040, 0x7a3e1a}}[v%3]
		b.rect(0, 0, 56, 44, c(0x2a1a14))
		b.rect(2, 2, 52, 40, c(pal[0]))
		for i := 0; i < 5; i++ { // concentric arcs, a classic of the time
			rad := 20 - float64(i)*3.6
			b.ellipse(28, 40, rad*1.35, rad, c(pal[i%4]))
		}
		for x := 2; x < 54; x += 8 {
			b.rect(x, 2, 1, 8, c(pal[2]))
		}
		b.rect(2, 41, 52, 1, c(0x14100c))
	}}

	// beadcurtain: a doorway of hanging bead strands
	pk["beadcurtain"] = &PropKind{W: 40, H: 58, FW: 38, FH: 4, NoSolid: true, Vars: 3, Draw: func(b *pixBuf, v int) {
		cols := [][]uint32{{0xe8402a, 0xf0b040, 0xd8761e}, {0x40b0d0, 0xe880c0, 0xf0d030}, {0xb040e0, 0xf0b040, 0x60c070}}[v%3]
		b.rect(0, 0, 40, 3, c(0x5a3a1e))
		b.rect(0, 0, 40, 1, c(0x8a5e30))
		for x := 2; x < 39; x += 3 {
			ln := 44 + (x*5)%12
			for y := 3; y < ln; y += 2 {
				b.set(x, y, c(cols[(y/2+x)%3]))
				b.set(x, y+1, c(0x2a1a14))
			}
		}
	}}

	// eggchair: the hanging pod chair of the time (guests sit in it)
	pk["eggchair"] = &PropKind{W: 30, H: 40, FW: 24, FH: 10, UseKind: "lounge", Use: []Vec{{0, 10}}, Vars: 3, Draw: func(b *pixBuf, v int) {
		col := c([]uint32{0xd8761e, 0xb040a0, 0x40a0b0}[v%3])
		b.ellipse(15, 38, 12, 2.5, c(0x1c1620))
		b.ellipse(15, 22, 13, 15, shade(col, 0.7))
		b.ellipse(15, 24, 10.5, 12, shade(col, 1.15))
		b.rect(6, 29, 18, 8, shade(col, 0.55)) // mouth of the pod, darker
		b.ellipse(15, 29, 8, 4, c(0xf4e8d0))   // cushion
		b.ellipse(15, 12, 3, 3, c(0x8a8e9a))
		b.rect(14, 4, 2, 6, c(0x8a8e9a))
		b.rect(11, 37, 8, 2, c(0xc8c8d0))
	}}
}

// rinkDigit paints a 7-segment-like 5x9 digit.
func rinkDigit(b *pixBuf, x, y, d int, cc color.RGBA) {
	seg := [10][7]bool{
		{true, true, true, false, true, true, true}, {false, false, true, false, false, true, false},
		{true, false, true, true, true, false, true}, {true, false, true, true, false, true, true},
		{false, true, true, true, false, true, false}, {true, true, false, true, false, true, true},
		{true, true, false, true, true, true, true}, {true, false, true, false, false, true, false},
		{true, true, true, true, true, true, true}, {true, true, true, true, false, true, true},
	}[((d%10)+10)%10]
	if seg[0] {
		b.rect(x+1, y, 4, 1, cc)
	}
	if seg[1] {
		b.rect(x, y+1, 1, 4, cc)
	}
	if seg[2] {
		b.rect(x+5, y+1, 1, 4, cc)
	}
	if seg[3] {
		b.rect(x+1, y+5, 4, 1, cc)
	}
	if seg[4] {
		b.rect(x, y+6, 1, 4, cc)
	}
	if seg[5] {
		b.rect(x+5, y+6, 1, 4, cc)
	}
	if seg[6] {
		b.rect(x+1, y+10, 4, 1, cc)
	}
}
