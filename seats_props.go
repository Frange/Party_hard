package main

import "image/color"

// ---------------------------------------------------------------------------
// SEATS: the furniture the guests sit on to rest (seats.go places it and runs the sitting). Every kind comes in the
// materials of seatMats (one per party theme: home wood, café iron, beach plastic, spaceship, pirate, velvet, snow), so
// the variant is mat*stride (+1 for the chair that looks left). The solid footprint is only the real foot of the piece
// (legs, pedestal, cloth down to the floor): a chair does not close a passage wider than itself.
// ---------------------------------------------------------------------------

type seatMat struct {
	frame, seat, cloth, hi uint32
}

// seatMats: 0 wood, 1 café (black iron), 2 white plastic, 3 spaceship, 4 pirate, 5 velvet and gold, 6 snow lodge.
var seatMats = []seatMat{
	{0x8a5a30, 0xb07a40, 0xe4e6f0, 0xd09a5a},
	{0x2a2a34, 0x9a6a3a, 0xc0394a, 0x5a5a6a},
	{0xd8dee4, 0xf4f6f8, 0x3a9ad0, 0xffffff},
	{0x8a94a8, 0x3a7ad0, 0xd8e0ec, 0xc8d4e8},
	{0x5a3a20, 0x7a5230, 0x8a2a2a, 0x9a6a3a},
	{0xc8a040, 0xb02a4a, 0x2a2a3a, 0xf0d070},
	{0x7a4e28, 0xc84030, 0xe8f0f8, 0xf4f8ff},
}

const seatMatN = 7

func init() {
	pk := propKinds
	// a chair seen from the side: even variants have the back on the left (the sitter looks right), odd ones on the right
	pk["seatchair"] = &PropKind{W: 12, H: 22, FW: 10, FH: 3, Vars: seatMatN * 2, Category: "furniture", Draw: func(b *pixBuf, v int) {
		m := seatMats[(v/2)%seatMatN]
		fr, st := c(m.frame), c(m.seat)
		bx := 1
		if v%2 == 1 {
			bx = 9
		}
		b.ellipse(6, 21, 5, 1.2, color.RGBA{0, 0, 0, 60})
		b.rect(bx, 1, 2, 19, shade(fr, 0.85)) // the back and the back leg
		b.rect(bx, 1, 2, 1, c(m.hi))
		b.rect(1, 10, 10, 3, st) // the seat
		b.rect(1, 10, 10, 1, shade(st, 1.25))
		b.rect(1, 12, 10, 1, shade(st, 0.75))
		lx := 9
		if v%2 == 1 {
			lx = 1
		}
		b.rect(lx, 13, 2, 7, shade(fr, 0.7)) // the front leg
		if (v/2)%seatMatN == 3 {             // spaceship: a pillar and a round foot
			b.rect(lx, 13, 2, 7, color.RGBA{})
			b.rect(5, 13, 2, 7, shade(fr, 0.8))
			b.rect(2, 19, 8, 1, shade(fr, 0.6))
		}
	}}
	// a chair seen from the front (the sitter looks at us)
	pk["seatfront"] = &PropKind{W: 14, H: 22, FW: 12, FH: 3, Vars: seatMatN, Category: "furniture", Draw: func(b *pixBuf, v int) {
		m := seatMats[v%seatMatN]
		fr, st := c(m.frame), c(m.seat)
		b.ellipse(7, 21, 6, 1.3, color.RGBA{0, 0, 0, 60})
		b.rect(2, 1, 10, 9, shade(fr, 0.9)) // back
		b.rect(2, 1, 10, 1, c(m.hi))
		b.rect(4, 3, 6, 5, shade(st, 0.85))
		b.rect(1, 10, 12, 3, st) // seat
		b.rect(1, 10, 12, 1, shade(st, 1.25))
		b.rect(1, 12, 12, 1, shade(st, 0.75))
		b.rect(2, 13, 2, 7, shade(fr, 0.7))
		b.rect(10, 13, 2, 7, shade(fr, 0.7))
	}}
	// a little round table: the solid part is only the pedestal and its foot
	pk["seattable"] = &PropKind{W: 26, H: 20, FW: 12, FH: 4, Vars: seatMatN, Category: "furniture", Draw: func(b *pixBuf, v int) {
		m := seatMats[v%seatMatN]
		fr, top := c(m.frame), c(m.seat)
		if v%seatMatN == 2 || v%seatMatN == 3 {
			top = c(m.cloth)
		}
		b.ellipse(13, 18, 6, 1.6, shade(fr, 0.6))
		b.rect(12, 8, 2, 10, shade(fr, 0.8))
		b.ellipse(13, 6, 12, 4.2, shade(top, 0.7))
		b.ellipse(13, 5, 12, 4, top)
		b.ellipse(13, 4, 9, 2.6, shade(top, 1.12))
		// a glass and a little plate
		b.rect(8, 2, 2, 3, c(0xd8e8f4))
		b.rect(8, 1, 2, 1, c(0xf4f8ff))
		b.ellipse(17, 4, 2.6, 1, c(0xf4f0e8))
	}}
	// a long table with a cloth down to the floor (the guests can hide under it, pe_tablehide.go)
	pk["seatlong"] = &PropKind{W: 60, H: 22, FW: 56, FH: 8, Vars: seatMatN, Category: "furniture", Draw: func(b *pixBuf, v int) {
		m := seatMats[v%seatMatN]
		cl := c(m.cloth)
		b.rect(2, 2, 56, 7, shade(cl, 1.08)) // the top
		b.rect(2, 2, 56, 1, shade(cl, 1.22))
		b.rect(2, 9, 56, 11, shade(cl, 0.86)) // the cloth falls in front
		b.rect(2, 9, 2, 11, shade(cl, 0.7))
		b.rect(56, 9, 2, 11, shade(cl, 0.66))
		for x := 8; x < 54; x += 8 {
			b.rect(x, 10, 1, 9, shade(cl, 0.74))
		}
		b.rect(2, 19, 56, 1, shade(cl, 0.62))
		b.rect(2, 9, 56, 1, shade(cl, 0.7))
		// on top: plates, a jug, glasses
		for _, x := range []int{9, 22, 36, 49} {
			b.ellipse(float64(x), 5, 3, 1.4, c(0xf4f0e8))
		}
		b.rect(29, 2, 3, 5, c(m.hi))
		b.rect(15, 3, 2, 3, c(0xd8e8f4))
		b.rect(43, 3, 2, 3, c(0xd8e8f4))
	}}
	// a bench with a back, seen from the front (park, stadium, station, ship...)
	pk["seatbench"] = &PropKind{W: 34, H: 20, FW: 30, FH: 3, Vars: seatMatN, Category: "furniture", Draw: func(b *pixBuf, v int) {
		m := seatMats[v%seatMatN]
		fr, st := c(m.frame), c(m.seat)
		if v%seatMatN == 2 { // stadium: plastic slats in the team colour
			st = c(m.cloth)
		}
		b.ellipse(17, 19, 15, 1.3, color.RGBA{0, 0, 0, 60})
		b.rect(2, 2, 30, 3, st) // back slats
		b.rect(2, 2, 30, 1, shade(st, 1.25))
		b.rect(2, 6, 30, 2, shade(st, 0.9))
		b.rect(1, 9, 32, 3, st) // seat
		b.rect(1, 9, 32, 1, shade(st, 1.25))
		b.rect(1, 11, 32, 1, shade(st, 0.72))
		for _, x := range []int{3, 29} {
			b.rect(x, 2, 2, 17, shade(fr, 0.75)) // the sides
		}
		if v%seatMatN == 6 { // snow on the back
			b.rect(3, 1, 28, 1, c(0xf8fbff))
			b.rect(6, 0, 9, 1, c(0xf8fbff))
		}
	}}
	// a folding deckchair with a striped canvas (beach, pool, ship)
	pk["seatdeck"] = &PropKind{W: 16, H: 20, FW: 12, FH: 3, Vars: 3, Category: "furniture", Draw: func(b *pixBuf, v int) {
		a := []uint32{0xe04a4a, 0x3a9ad0, 0xf0b030}[v%3]
		fr := c(0xb8864a)
		b.ellipse(8, 19, 7, 1.2, color.RGBA{0, 0, 0, 60})
		b.rect(1, 2, 2, 17, fr)
		b.rect(13, 2, 2, 17, fr)
		for y := 2; y < 14; y++ { // the canvas sags from the top bar to the seat
			col := c(0xf4f2ec)
			if (y/2)%2 == 0 {
				col = c(a)
			}
			b.rect(3, y, 10, 1, col)
		}
		b.rect(3, 13, 10, 1, shade(c(a), 0.7))
		b.rect(1, 2, 14, 1, shade(fr, 1.2))
	}}
	// a round pouf (club, home, casino)
	pk["seatpouf"] = &PropKind{W: 14, H: 12, FW: 12, FH: 4, Vars: 4, Category: "furniture", Draw: func(b *pixBuf, v int) {
		col := c([]uint32{0xd83a8a, 0x3ac8d8, 0xf0c040, 0x8a4ad8}[v%4])
		b.ellipse(7, 10, 6.5, 1.8, shade(col, 0.55))
		b.rect(1, 4, 12, 6, shade(col, 0.82))
		b.rect(1, 4, 1, 6, shade(col, 0.66))
		b.rect(12, 4, 1, 6, shade(col, 0.62))
		b.ellipse(7, 4, 6, 2.6, col)
		b.ellipse(7, 3.6, 3.4, 1.2, shade(col, 1.25))
		b.set(7, 4, shade(col, 0.6)) // the button
	}}
}
