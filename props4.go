package main

import "math"

// New set-dressing for the skate park, car circuit and space station
// campaign scenarios.

func init() {
	pk := propKinds

	pk["extinguisher"] = &PropKind{W: 10, H: 20, FW: 8, FH: 6, Trap: "explode", Draw: func(b *pixBuf, v int) {
		b.rect(2, 4, 6, 14, c(0xc02020))
		b.rect(2, 4, 6, 1, c(0xe84040))
		b.rect(3, 1, 4, 4, c(0x2a2a30))
		b.rect(1, 0, 2, 2, c(0x8a8a94))
		b.rect(3, 16, 4, 2, c(0xe8e8ec))
	}}

	pk["ramp"] = &PropKind{W: 54, H: 30, FW: 48, FH: 18, Draw: func(b *pixBuf, v int) {
		for x := 0; x < 48; x++ {
			h := x * 22 / 48
			b.rect(x+3, 26-h, 1, h+2, c(0x9aa0a8))
		}
		b.rect(3, 26, 48, 2, c(0x6a7078))
		b.rect(3, 2, 2, 24, c(0xc8ccd0))
	}}

	pk["rail"] = &PropKind{W: 70, H: 14, FW: 64, FH: 4, Draw: func(b *pixBuf, v int) {
		b.rect(2, 6, 66, 2, c(0xb8bcc2))
		b.rect(2, 7, 66, 1, c(0x6a6e74))
		b.rect(4, 8, 2, 6, c(0x5a5e64))
		b.rect(64, 8, 2, 6, c(0x5a5e64))
	}}

	pk["halfpipe"] = &PropKind{W: 96, H: 42, FW: 86, FH: 22, Draw: func(b *pixBuf, v int) {
		b.rect(2, 20, 92, 4, c(0x7a8088))
		for x := 0; x < 46; x++ {
			h := (46 - x) * 18 / 46
			b.rect(x+2, 20-h, 1, h, c(0x9aa0a8))
		}
		for x := 0; x < 46; x++ {
			h := x * 18 / 46
			b.rect(x+48, 20-h, 1, h, c(0x9aa0a8))
		}
	}}

	pk["lifebuoy"] = &PropKind{W: 12, H: 14, FW: 4, FH: 4, NoSolid: true, Draw: func(b *pixBuf, v int) {
		b.ellipse(6, 7, 5, 5, c(0xd8303a))
		b.ellipse(6, 7, 3.4, 3.4, c(0xf4f0e8))
		b.ellipse(6, 7, 1.6, 1.6, c(0x0a0a0c))
		for _, a := range []float64{0.8, 2.4, 3.9, 5.5} {
			x := 6 + math.Cos(a)*4.2
			y := 7 + math.Sin(a)*4.2
			b.set(int(x), int(y), c(0xf4f0e8))
		}
	}}

	// bridge: a purely decorative overpass for the car-circuit level. It is
	// not solid (pedestrians "walk over" it) and draws with a positive sort
	// offset so it appears in front of the road and the cars crossing under it.
	pk["bridge"] = &PropKind{W: 150, H: 56, FW: 4, FH: 4, NoSolid: true, SortOff: 70, Draw: func(b *pixBuf, v int) {
		b.rect(0, 20, 150, 16, c(0x6a6a72))
		b.rect(0, 20, 150, 2, c(0x8a8a94))
		b.rect(0, 34, 150, 2, c(0x3a3a40))
		for x := 4; x < 150; x += 14 {
			b.rect(x, 10, 3, 12, c(0x50505a))
		}
		b.rect(0, 0, 150, 4, c(0xd0d4da))
		b.rect(0, 48, 150, 4, c(0xd0d4da))
	}}
}
