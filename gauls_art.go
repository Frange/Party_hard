package main

import "image/color"

// ---------------------------------------------------------------------------
// Astérix and Obélix: our own cartoon take (not a copy of any artwork). Overlay hooks called by
// genChar for the two characters: winged helmet (the wings flap while walking or running),
// big droopy moustache, party glasses for Astérix; striped top, belt with a flower lei,
// red braids and a fist with stars for Obélix. Visual only.
// ---------------------------------------------------------------------------

const (
	gaulNone = iota
	gaulAsterix
	gaulObelix
)

func gaulKind(s *CharSpec) int {
	switch s.Name {
	case "asterix":
		return gaulAsterix
	case "obelix":
		return gaulObelix
	}
	return gaulNone
}

var (
	gaulSteel  = hx(0xb4bac8)
	gaulSteelH = hx(0xe2e6ee)
	gaulSteelD = hx(0x7c8294)
	gaulWing   = hx(0xf8f8fc)
	gaulWingD  = hx(0xb8c0d0)
)

// gaulHead paints helmet with wings, moustache, glasses and braids. hw: half width of the head.
func gaulHead(b *pixBuf, s *CharSpec, kind, vw, ihx, top, ey, hw int, p Pose) {
	b.cur = 1
	up := p.LF.Z > 0.5 || p.RF.Z > 0.5 // the stride of the walk and the run flaps the wings
	// helmet cap
	b.rect(ihx-hw, top, 2*hw+1, 2, gaulSteel)
	b.rect(ihx-hw, top+1, 2*hw+1, 1, gaulSteelD)
	b.set(ihx-hw, top, color.RGBA{})
	b.set(ihx+hw, top, color.RGBA{})
	b.set(ihx-1, top, gaulSteelH)
	b.set(ihx, top, gaulSteelH)
	if kind == gaulObelix {
		b.rect(ihx-hw, top+1, 2*hw+1, 1, hx(0xc0402e)) // red band
	}
	// wings
	wing := func(dir int) {
		x := ihx + dir*(hw+1)
		if up {
			b.set(x, top, gaulWing)
			b.set(x+dir, top-1, gaulWing)
			b.set(x+dir*2, top-2, gaulWing)
			b.set(x+dir, top, gaulWingD)
		} else {
			b.set(x, top, gaulWing)
			b.set(x+dir, top, gaulWing)
			b.set(x+dir*2, top+1, gaulWing)
			b.set(x+dir, top+1, gaulWingD)
		}
	}
	hair := hx(0xe0a024)
	if kind == gaulObelix {
		hair = hx(0xd04a28)
	}
	switch vw {
	case vFront:
		wing(-1)
		wing(1)
		if kind == gaulAsterix {
			// party glasses: two bright round lenses
			b.rect(ihx-2, ey, 2, 2, hx(0xe840a8))
			b.rect(ihx+1, ey, 2, 2, hx(0x40d0e8))
			b.set(ihx-2, ey, hx(0xf8d8f0))
			b.set(ihx+1, ey, hx(0xd8f8ff))
			b.set(ihx, ey, hx(0x2a2a34))
		}
		// moustache: two thick tufts with a drooping tip
		b.rect(ihx-hw, ey+2, hw, 1, hair)
		b.rect(ihx+1, ey+2, hw, 1, hair)
		b.set(ihx-hw, ey+3, hair)
		b.set(ihx+hw, ey+3, hair)
		if kind == gaulObelix {
			for _, sd := range []int{-1, 1} { // braids
				x := ihx + sd*(hw+1)
				b.rect(x, top+2, 1, ey-top+4, hair)
				b.set(x, ey+4, hx(0x6ab4e8))
				b.set(x, ey+5, hair)
			}
		}
	case vBack:
		wing(-1)
		wing(1)
		if kind == gaulObelix {
			b.rect(ihx-hw-1, top+2, 1, ey-top+4, hair)
			b.rect(ihx+hw+1, top+2, 1, ey-top+4, hair)
		}
	default:
		wing(-1)
		b.rect(ihx+1, ey+2, hw, 1, hair)
		b.set(ihx+hw, ey+3, hair)
		if kind == gaulAsterix {
			b.rect(ihx+hw-2, ey, 2, 2, hx(0xe840a8))
			b.set(ihx+hw-2, ey, hx(0xf8d8f0))
		}
		if kind == gaulObelix {
			b.rect(ihx-hw, top+2, 1, ey-top+4, hair)
		}
	}
	// the hard punch of Obélix: little stars
	if kind == gaulObelix && p.Hold == HoldKnife && p.RA.Z >= 2 {
		star := func(x, y int) {
			c := hx(0xffe040)
			b.set(x, y, hx(0xfffbe0))
			b.set(x-1, y, c)
			b.set(x+1, y, c)
			b.set(x, y-1, c)
			b.set(x, y+1, c)
		}
		star(ihx+hw+4, ey+5)
		b.set(ihx+hw+4, ey+9, hx(0xffb030))
		b.set(ihx+hw+6, ey+7, hx(0xffb030))
	}
}

// gaulTorso: tunic of Astérix (night blue with a green and gold belt) and the striped top of Obélix.
func gaulTorso(b *pixBuf, s *CharSpec, kind, vw, x0, x1, y0, y1 int) {
	b.cur = 2
	mid := (x0 + x1) / 2
	if kind == gaulAsterix {
		for y := y0; y <= y1; y++ {
			for x := x0; x < x1; x++ {
				c := hx(0x2c2e60)
				if x == x0 {
					c = hx(0x3c4090)
				} else if x == x1-1 {
					c = hx(0x1c1c40)
				}
				if y == y0 {
					c = hx(0xf0e4b8) // pale collar
				}
				b.set(x, y, c)
			}
		}
		if vw != vBack {
			b.set(mid, y0+1, hx(0xf0e4b8))
			b.set(mid, y0+3, hx(0xf0c030)) // little gold star on the chest
		}
		b.rect(x0, y1-2, x1-x0, 1, hx(0x32a060)) // belt
		b.rect(x0, y1-1, x1-x0, 1, hx(0xf0c030))
		if vw != vSide {
			b.rect(mid, y1-2, 1, 2, hx(0xfff0a0))
		}
		return
	}
	for y := y0; y <= y1; y++ {
		for x := x0; x < x1; x++ {
			c := hx(0x72b8ea)
			if (y-y0)/2%2 == 1 {
				c = hx(0xf2eee2)
			}
			if x == x1-1 {
				c = shade(c, 0.82)
			}
			b.set(x, y, c)
		}
	}
	b.rect(x0, y1-3, x1-x0, 2, hx(0x3e8a46)) // belt
	if vw != vBack {
		b.rect(mid-1, y1-3, 3, 2, hx(0xf4c830))
		b.set(mid, y1-3, hx(0xfff0a0))
		// flower lei
		cols := []uint32{0xf060a0, 0xf8e050, 0xf08a30, 0xf4f4f4}
		for i, x := 0, x0+1; x < x1-1; x, i = x+1, i+1 {
			b.set(x, y0, hx(cols[i%4]))
			if x > x0+2 && x < x1-3 {
				b.set(x, y0+1, hx(cols[(i+2)%4]))
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Wally (our own take): bobble hat with a bouncing pompom, big square glasses, a smile, a red and
// cream jumper with a party scarf and a cane that taps the ground. Visual only.
// ---------------------------------------------------------------------------

func isWally(s *CharSpec) bool { return s.Name == "wally" }

func wallyHead(b *pixBuf, s *CharSpec, vw, ihx, top, ey, hw int, p Pose) {
	b.cur = 1
	hop := p.Bob > 0.5 || p.LF.Z > 0.5 || p.RF.Z > 0.5
	py := top - 3
	if hop {
		py--
	}
	b.rect(ihx, py, 2, 2, hx(0xfff8e8))
	b.set(ihx, py, hx(0xffffff))
	b.set(ihx+1, py+1, hx(0xd8c8b0))
	b.rect(ihx-hw, top+2, 2*hw+1, 1, hx(0xf6f0e0)) // folded brim of the hat
	frame, glass := hx(0x1a1a24), hx(0xdcecf8)
	switch vw {
	case vFront:
		b.rect(ihx-3, ey-1, 3, 3, frame)
		b.rect(ihx+1, ey-1, 3, 3, frame)
		b.set(ihx-2, ey, glass)
		b.set(ihx+2, ey, glass)
		b.set(ihx, ey-1, frame)
		b.set(ihx-1, ey+2, hx(0xb04030)) // smile
		b.set(ihx, ey+3, hx(0xb04030))
		b.set(ihx+1, ey+2, hx(0xb04030))
	case vSide:
		b.rect(ihx+hw-2, ey-1, 3, 3, frame)
		b.set(ihx+hw-1, ey, glass)
		b.set(ihx+hw-1, ey+2, hx(0xb04030))
	}
}

func wallyTorso(b *pixBuf, x0, x1, y0, y1, vw int) {
	b.cur = 2
	for y := y0; y <= y1; y++ {
		c := hx(0xd8342c)
		if (y-y0)%3 == 2 {
			c = hx(0xf6f0e0)
		}
		for x := x0; x < x1; x++ {
			cc := c
			if x == x1-1 {
				cc = shade(c, 0.82)
			}
			b.set(x, y, cc)
		}
	}
	// party scarf
	for x := x0; x < x1; x++ {
		c := hx(0x38a860)
		if (x-x0)%2 == 1 {
			c = hx(0xf0d040)
		}
		b.set(x, y0, c)
	}
	if vw != vBack {
		b.rect(x0+1, y0+1, 2, 3, hx(0x38a860))
		b.set(x0+1, y0+2, hx(0xf0d040))
		b.set(x0+2, y0+3, hx(0xf0d040))
	}
}

// wallyCane: a cane with a crook; the tip lifts a little while he steps.
func wallyCane(b *pixBuf, hx0, hy0 float64, p Pose) {
	x, y := int(hx0)+1, int(hy0)
	lift := 0
	if p.LF.Z > 0.5 {
		lift = 1
	}
	c := hx(0x8a5a2a)
	b.rect(x, y-1, 2, 1, c)
	b.set(x+2, y, c)
	for i := 1; i < 10-lift; i++ {
		b.set(x, y+i, c)
	}
	b.set(x, y+9-lift, hx(0x2a2a30))
}
