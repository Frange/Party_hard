package main

import "image/color"

// ---------------------------------------------------------------------------
// Props for the graveyard, the casino and the ski resort scenarios.
// ---------------------------------------------------------------------------

func init() {
	pk := propKinds
	pk["tombstone"] = &PropKind{W: 20, H: 28, FW: 14, FH: 6, Vars: 3, Draw: func(b *pixBuf, v int) {
		st, lt, dk, moss := c(0x8a8e9a), c(0xb8bcc8), c(0x565a68), c(0x4a7a4a)
		b.ellipse(10, 25, 9, 2.6, color.RGBA{0, 0, 0, 90})
		switch v {
		case 0: // rounded slab
			b.rect(3, 8, 14, 17, st)
			b.ellipse(10, 9, 7, 6, st)
			b.rect(3, 8, 2, 17, lt)
			b.rect(15, 8, 2, 17, dk)
			b.rect(7, 12, 6, 1, dk) // RIP
			b.rect(9, 12, 1, 5, dk)
			b.rect(7, 16, 6, 1, dk)
			b.rect(3, 22, 14, 3, moss)
		case 1: // cross
			b.rect(8, 2, 4, 23, st)
			b.rect(2, 8, 16, 4, st)
			b.rect(8, 2, 1, 23, lt)
			b.rect(11, 2, 1, 23, dk)
			b.rect(2, 8, 16, 1, lt)
			b.rect(7, 22, 6, 3, moss)
		default: // pointed obelisk on a plinth
			b.rect(5, 20, 10, 5, dk)
			b.rect(7, 6, 6, 15, st)
			for i := 0; i < 5; i++ {
				b.rect(8+i/2, 1+i, 4-i/2*2+0, 1, st)
			}
			b.rect(7, 6, 1, 15, lt)
			b.rect(12, 6, 1, 15, dk)
		}
	}}
	pk["crypt"] = &PropKind{W: 124, H: 96, FW: 104, FH: 26, Draw: func(b *pixBuf, v int) {
		st, lt, dk, door := c(0x7a7e8c), c(0xa8acba), c(0x4a4e5c), c(0x0e0a14)
		b.ellipse(62, 90, 60, 6, color.RGBA{0, 0, 0, 90})
		b.rect(8, 40, 108, 50, st) // body with brick courses
		for y := 44; y < 90; y += 8 {
			b.rect(8, y, 108, 1, dk)
			for x := 8 + (y/8%2)*9; x < 116; x += 18 {
				b.rect(x, y, 1, 8, dk)
			}
		}
		b.rect(8, 40, 108, 2, lt)
		// pediment
		for i := 0; i < 22; i++ {
			b.rect(4+i*2, 40-i, 116-i*4, 1, st)
		}
		b.rect(20, 24, 84, 2, lt)
		b.ellipse(62, 34, 6, 5, c(0xd8d4c0)) // skull emblem
		b.rect(59, 32, 2, 2, door)
		b.rect(63, 32, 2, 2, door)
		// columns
		for _, x := range []int{14, 100} {
			b.rect(x, 42, 10, 46, lt)
			b.rect(x, 42, 2, 46, st)
			b.rect(x+8, 42, 2, 46, dk)
			b.rect(x-2, 40, 14, 3, lt)
			b.rect(x-2, 88, 14, 3, lt)
		}
		// iron door
		b.rect(46, 52, 32, 38, door)
		b.ellipse(62, 53, 16, 8, door)
		for x := 48; x < 76; x += 4 {
			b.rect(x, 50, 1, 38, c(0x3a3a46))
		}
		b.rect(46, 66, 32, 1, c(0x3a3a46))
		// steps
		b.rect(40, 88, 44, 3, lt)
		b.rect(36, 91, 52, 3, st)
		// green torches
		for _, x := range []int{36, 88} {
			b.rect(x, 56, 2, 8, c(0x4a3a2a))
			b.ellipse(float64(x)+1, 54, 3, 4, c(0x40e060))
			b.ellipse(float64(x)+1, 53, 1.5, 2.5, c(0xc8ffd0))
		}
	}}
	pk["deadtree"] = &PropKind{W: 60, H: 80, FW: 10, FH: 6, Draw: func(b *pixBuf, v int) {
		bark, dk := c(0x3a2c26), c(0x241a18)
		b.ellipse(30, 77, 14, 2.5, color.RGBA{0, 0, 0, 80})
		b.rect(26, 38, 8, 40, bark)
		b.rect(26, 38, 2, 40, c(0x54403a))
		b.rect(32, 38, 2, 40, dk)
		br := [][4]int{{28, 40, 8, 18}, {32, 42, 52, 20}, {29, 50, 14, 38}, {32, 54, 50, 44}, {30, 34, 22, 8}, {31, 32, 40, 6}, {30, 28, 30, 2}}
		for _, l := range br {
			b.line(l[0], l[1], l[2], l[3], bark)
			b.line(l[0]+1, l[1], l[2]+1, l[3], dk)
		}
		b.line(8, 18, 3, 12, bark)
		b.line(8, 18, 12, 10, bark)
		b.line(52, 20, 57, 12, bark)
		b.line(52, 20, 47, 14, bark)
		b.line(14, 38, 6, 36, bark)
		b.line(50, 44, 56, 40, bark)
	}}
	pk["pumpkin"] = &PropKind{W: 18, H: 16, FW: 12, FH: 5, Vars: 2, Draw: func(b *pixBuf, v int) {
		b.ellipse(9, 14, 8, 2, color.RGBA{0, 0, 0, 80})
		b.ellipse(9, 9, 8, 6, c(0xe8801c))
		b.ellipse(5, 9, 4, 6, c(0xd06a10))
		b.ellipse(13, 9, 4, 6, c(0xd06a10))
		b.ellipse(9, 9, 3, 6, c(0xf09a30))
		b.rect(8, 2, 2, 3, c(0x4a7a2a))
		if v == 1 {
			glow := c(0xfff070)
			b.rect(5, 7, 2, 2, glow)
			b.rect(11, 7, 2, 2, glow)
			b.rect(6, 11, 6, 1, glow)
			b.rect(7, 12, 1, 1, glow)
			b.rect(10, 12, 1, 1, glow)
		}
	}}
	pk["slot"] = &PropKind{W: 24, H: 42, FW: 18, FH: 8, Vars: 2, Light: false, Draw: func(b *pixBuf, v int) {
		body := []color.RGBA{c(0xb02030), c(0x2a50b8)}[v%2]
		b.ellipse(12, 39, 11, 2.4, color.RGBA{0, 0, 0, 80})
		b.rect(2, 4, 18, 36, body)
		b.rect(2, 4, 18, 2, shade(body, 1.35))
		b.rect(2, 6, 2, 34, shade(body, 1.2))
		b.rect(18, 6, 2, 34, shade(body, 0.7))
		b.rect(4, 8, 14, 5, c(0xf0d040)) // marquee
		for x := 5; x < 17; x += 3 {
			b.set(x, 10, c(0xe03030))
		}
		b.rect(4, 15, 14, 10, c(0x101018)) // reels
		for i, col := range []uint32{0xf04040, 0xf0e040, 0x40d060} {
			b.rect(5+i*4, 16, 3, 8, c(0xf4f4f0))
			b.rect(6+i*4, 18, 1, 3, c(col))
		}
		b.rect(4, 27, 14, 3, c(0x2a2a34)) // coin slot
		b.rect(8, 28, 6, 1, c(0xf0d040))
		b.rect(4, 34, 14, 4, c(0x14141c))  // payout tray
		b.rect(21, 12, 2, 12, c(0x9a9aa6)) // lever
		b.ellipse(22, 11, 2, 2, c(0xe03030))
	}}
	pk["roulette"] = &PropKind{W: 70, H: 44, FW: 62, FH: 18, Draw: func(b *pixBuf, v int) {
		b.ellipse(35, 40, 33, 4, color.RGBA{0, 0, 0, 80})
		b.rect(4, 30, 62, 10, c(0x5a3a22)) // wooden edge
		b.ellipse(35, 28, 33, 14, c(0x7a4a28))
		b.ellipse(35, 26, 31, 12, c(0x1e7a3e)) // green felt
		b.ellipse(22, 25, 10, 7, c(0x2a1a10))  // wheel
		b.ellipse(22, 25, 8, 5.4, c(0xd02828))
		b.ellipse(22, 25, 5.5, 3.6, c(0x14141a))
		b.ellipse(22, 25, 2.5, 1.6, c(0xe8c040))
		for i := 0; i < 8; i++ {
			b.set(14+i*2, 25+(i%3)-1, c(0xf4f4f0))
		}
		for x := 40; x < 62; x += 6 { // number grid
			b.rect(x, 21, 5, 3, c(0xf4f4f0))
			b.rect(x, 25, 5, 3, c(0xd02828))
		}
		for _, p := range [][3]int{{44, 31, 0xe83030}, {47, 30, 0x3050e0}, {52, 32, 0xf0f0f0}, {56, 30, 0x20b050}} {
			b.ellipse(float64(p[0]), float64(p[1]), 2.4, 1.4, c(uint32(p[2])))
		}
	}}
	pk["pine"] = &PropKind{W: 44, H: 76, FW: 10, FH: 6, Draw: func(b *pixBuf, v int) {
		b.ellipse(22, 73, 12, 2.5, color.RGBA{0, 0, 0, 70})
		b.rect(19, 56, 6, 18, c(0x5a3c28))
		g1, g2, snow := c(0x1e5a3a), c(0x2a7048), c(0xe8f0fa)
		for i, w := range []int{40, 32, 24, 16} {
			y := 50 - i*13
			for r := 0; r < 16; r++ {
				ww := w * (r + 2) / 18
				b.rect(22-ww/2, y+r-12, ww, 1, g1)
			}
			b.rect(22-w/2+2, y+2, w-4, 1, g2)
			// snow caps on each tier
			for r := 0; r < 5; r++ {
				ww := w*(r+2)/18 - 2
				if ww > 0 {
					b.rect(22-ww/2, y+r-12, ww, 1, snow)
				}
			}
			b.rect(22-w/2+3, y+3, 4, 2, snow)
			b.rect(22+w/2-8, y+5, 5, 2, snow)
		}
	}}
	pk["snowman"] = &PropKind{W: 26, H: 38, FW: 14, FH: 6, Draw: func(b *pixBuf, v int) {
		w, shd := c(0xf2f6fc), c(0xc4cede)
		b.ellipse(13, 35, 10, 2.5, color.RGBA{0, 0, 0, 60})
		b.ellipse(13, 28, 10, 9, shd)
		b.ellipse(13, 27, 9.5, 8.5, w)
		b.ellipse(13, 16, 7.5, 7, shd)
		b.ellipse(13, 15.5, 7, 6.5, w)
		b.ellipse(13, 7, 5.5, 5.5, w)
		b.rect(8, 3, 10, 2, c(0x14141a)) // top hat
		b.rect(10, -1, 6, 5, c(0x14141a))
		b.rect(10, 2, 6, 1, c(0xd02828))
		b.set(11, 6, c(0x14141a))
		b.set(15, 6, c(0x14141a))
		b.rect(13, 8, 5, 1, c(0xf08a20)) // carrot
		for _, y := range []int{18, 23, 28} {
			b.set(13, y, c(0x14141a))
		}
		b.rect(5, 16, 3, 1, c(0x5a3c28)) // twig arms
		b.rect(19, 16, 3, 1, c(0x5a3c28))
		b.rect(7, 12, 12, 2, c(0x2a6ac8)) // scarf
	}}
	pk["igloo"] = &PropKind{W: 84, H: 54, FW: 70, FH: 20, Draw: func(b *pixBuf, v int) {
		w, shd, ice := c(0xe8f0fa), c(0xb4c4da), c(0xc8dcf0)
		b.ellipse(42, 48, 40, 5, color.RGBA{0, 0, 0, 70})
		b.ellipse(42, 36, 38, 28, shd)
		b.ellipse(42, 34, 37, 27, w)
		for y := 12; y < 48; y += 7 {
			b.rect(10, y, 64, 1, ice)
			for x := 10 + (y/7%2)*8; x < 74; x += 16 {
				b.rect(x, y, 1, 7, ice)
			}
		}
		b.rect(34, 30, 16, 18, c(0x14202e))
		b.ellipse(42, 31, 8, 7, c(0x14202e))
		b.rect(33, 46, 18, 2, shd)
	}}
	pk["chalet"] = &PropKind{W: 150, H: 116, FW: 130, FH: 32, Draw: func(b *pixBuf, v int) {
		wood, wd, snow, snd := c(0x8a5a34), c(0x5a3a22), c(0xf0f4fa), c(0xc0cce0)
		b.ellipse(75, 108, 70, 6, color.RGBA{0, 0, 0, 80})
		b.rect(10, 52, 130, 56, wood) // log walls
		for y := 54; y < 108; y += 7 {
			b.rect(10, y, 130, 1, wd)
			b.ellipse(14, float64(y)+3, 2, 3, c(0xa87848))
			b.ellipse(136, float64(y)+3, 2, 3, c(0xa87848))
		}
		// roof with snow
		for i := 0; i < 36; i++ {
			b.rect(2+i, 52-i, 146-i*2, 1, c(0x6a2a24))
		}
		for i := 0; i < 36; i++ {
			b.rect(2+i, 52-i, 146-i*2, 1, wd)
			if i%6 == 5 {
				b.rect(2+i, 52-i, 146-i*2, 1, c(0x3a1a18))
			}
		}
		for i := 0; i < 34; i++ {
			b.rect(5+i, 51-i, 140-i*2, 3, snow)
		}
		b.rect(2, 52, 146, 3, snow)
		b.rect(2, 55, 146, 2, snd)
		for x := 4; x < 146; x += 9 { // icicles
			b.rect(x, 57, 2, 3+((x/9)%3), c(0xdcecfa))
		}
		// chimney
		b.rect(104, 6, 14, 24, c(0x7a5a4a))
		b.rect(102, 4, 18, 4, snow)
		// door and windows lit from inside
		b.rect(62, 70, 26, 38, c(0x4a2a18))
		b.rect(64, 72, 22, 36, c(0x6a3a22))
		b.rect(82, 90, 2, 2, c(0xe8c040))
		for _, x := range []int{22, 106} {
			b.rect(x, 70, 22, 18, c(0x3a2414))
			b.rect(x+2, 72, 18, 14, c(0xffd070))
			b.rect(x+10, 72, 2, 14, c(0x3a2414))
			b.rect(x+2, 78, 18, 2, c(0x3a2414))
		}
		b.rect(56, 106, 38, 4, snow) // snowy step
	}}
}
