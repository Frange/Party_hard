package main

import (
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// Own drawing layers for the two Monkey Island leads (our own parody, not a copy of any sprite):
//   - Guybrush: big yellow mane swept back into a ponytail that bounces when he walks, dot eyes,
//     gold earring, open white shirt under a brown vest, red sash with a loose end hanging, brown
//     trousers and big black boots.
//   - LeChuck: green ghost pirate, brown tricorne with a skull and black plumes that wave, a huge
//     black beard, long red coat with gold buttons, belt and swinging tails, dark maroon boots.
// genChar calls miHead / miTorso / miLeg (chars.go); everything is baked into the cached frames.
// ---------------------------------------------------------------------------

const (
	miNone = iota
	miGuybrush
	miLeChuck
)

func miKind(s *CharSpec) int {
	switch s.Name {
	case "guybrush":
		return miGuybrush
	case "lechuck":
		return miLeChuck
	}
	return miNone
}

var (
	miGold    = hx(0xf0c838)
	miSash    = hx(0xc8282a)
	miVest    = hx(0x7a4a22)
	miCoatDk  = hx(0x7a1414)
	miHatBrn  = hx(0x6a3e1c)
	miBone    = hx(0xf0ecd8)
	miBlack   = hx(0x121216)
	miFeather = hx(0x3a3a44)
)

// miSwing: a small sideways bounce for hair, plumes and coat tails while moving.
func miSwing(p Pose) int {
	switch p.Anim {
	case AnimWalk, AnimRun, AnimHop:
		if p.F%2 == 1 {
			return 1
		}
	}
	return 0
}

// miHead paints the mane, ponytail and earring (Guybrush) or the tricorne, plumes and beard (LeChuck).
func miHead(b *pixBuf, s *CharSpec, vw int, p Pose, hxp, hyp, rx, ry float64, ey int) {
	k := miKind(s)
	if k == miNone {
		return
	}
	b.cur = 1
	ihx := int(math.Floor(hxp))
	top := int(math.Floor(hyp - ry))
	chin := int(math.Floor(hyp + ry))
	rxi := int(math.Floor(rx))
	sw := miSwing(p)
	if k == miGuybrush {
		h := s.Hair
		hl, hd := shade(h, 1.25), shade(h, 0.72)
		switch vw {
		case vFront:
			// a tall, wide mane: a crest over the head and locks down both sides
			b.rect(ihx-rxi, top-2, 2*rxi+1, 4, h)
			b.rect(ihx-rxi+1, top-3, 2*rxi-1, 1, h)
			b.rect(ihx-rxi-1, top-1, 1, ey-top+2, h)
			b.rect(ihx+rxi+1, top-1, 1, ey-top+2, h)
			b.set(ihx-1, top-2, hl) // shine of the combed-back mane
			b.set(ihx-rxi-1, ey+1, hd)
			b.set(ihx+rxi+1, ey+1, hd)
			b.set(ihx+rxi+2, ey-1+sw, hd)  // the ponytail sways behind his head
			b.set(ihx-rxi-1, ey+2, miGold) // earring
		case vBack:
			b.rect(ihx-rxi, top-2, 2*rxi+1, 2, h)
			b.rect(ihx-rxi-1, top-1, 2*rxi+3, ey-top+2, h)
			b.rect(ihx-rxi+1, top-3, 2*rxi-1, 1, h)
			b.rect(ihx, top, 1, 3, hd)
			// the ponytail hangs down the back with a dark tie
			b.rect(ihx-1, chin-1, 3, 1, hx(0x3a2414))
			b.rect(ihx-1+sw, chin, 3, 4, h)
			b.rect(ihx+sw, chin+4, 1, 2, hd)
			b.set(ihx-1+sw, chin+1, hl)
		case vSide:
			// swept back: the mass sits behind the head and ends in a ponytail
			b.rect(ihx-rxi-1, top-2, 2*rxi+1, 3, h)
			b.rect(ihx-rxi-1, top+1, rxi+1, 1, h)
			b.rect(ihx-rxi-2, top-1, 3, ey-top+2, h)
			b.set(ihx+rxi-1, top-1, h)
			b.set(ihx, top-2, hl)
			b.set(ihx-1, top-1, hd)
			b.rect(ihx-rxi-3, top, 1, 3, h)
			b.set(ihx-rxi-2, ey+1, hx(0x3a2414)) // tie
			b.rect(ihx-rxi-3-sw, ey+2, 2, 3, h)
			b.set(ihx-rxi-4-sw, ey+5, hd)
			b.set(ihx-1, ey+2, miGold) // earring
		}
		return
	}
	// LeChuck: the huge beard first, then the tricorne over everything
	beard := s.Hair
	streak := shade(beard, 2.2)
	switch vw {
	case vFront:
		b.rect(ihx-rxi-1, ey+1, 2*rxi+3, chin-ey+2, beard)
		b.rect(ihx-rxi, chin+2, 2*rxi+1, 2, beard)
		b.rect(ihx-1, chin+4, 3, 1, beard)
		b.set(ihx-rxi, chin+4, beard)
		b.set(ihx+rxi, chin+4, beard)
		b.set(ihx, chin+5, beard)
		b.set(ihx-2, ey+3, streak)
		b.set(ihx+1, chin+1, streak)
		b.set(ihx-1, chin+3, streak)
		b.rect(ihx-1, ey+2, 3, 1, hx(0x5a1010)) // a scowling mouth in the beard
		if p.Open {
			b.rect(ihx-1, ey+2, 3, 2, hx(0x8a1a1a))
		}
		b.set(ihx-2, ey-1, miBlack) // angry brows
		b.set(ihx+2, ey-1, miBlack)
		b.set(ihx-1, ey, hx(0xf8f0c0)) // glowing ghost eyes
		b.set(ihx+1, ey, hx(0xf8f0c0))
	case vSide:
		b.rect(ihx-1, ey+1, rxi+3, chin-ey+2, beard)
		b.rect(ihx, chin+2, rxi+2, 2, beard)
		b.set(ihx+rxi+2, ey+2, beard)
		b.set(ihx+1, chin+4, beard)
		b.set(ihx+1, ey+3, streak)
		b.set(ihx+rxi, ey, hx(0xf8f0c0))
		b.set(ihx+rxi, ey-1, miBlack)
	case vBack:
		b.rect(ihx-rxi-1, ey+1, 1, 4, beard)
		b.rect(ihx+rxi+1, ey+1, 1, 4, beard)
	}
	// tricorne: brim turned up at both points and dipping at the front, crown, a skull on it
	hc := miHatBrn
	hd := shade(hc, 0.7)
	hl := shade(hc, 1.25)
	clip := func(x, y, w, h int, c color.RGBA) { // never paints the top row of the box
		if y < 1 {
			h -= 1 - y
			y = 1
		}
		if h > 0 {
			b.rect(x, y, w, h, c)
		}
	}
	// black plumes behind the crown, a fluffy fan that waves while he walks
	fx, dir := ihx+1, 1
	if vw == vSide {
		fx, dir = ihx-1, -1
	}
	for i := 0; i < 5; i++ {
		y := top - 3 - i
		x := fx + dir*(i/2) + dir*sw*(i/3)
		clip(x-1, y, 3, 1, miBlack)
		if i%2 == 0 {
			clip(x+dir*2, y, 1, 1, miFeather)
		}
	}
	clip(fx+dir*3+dir*sw, top-7, 1, 2, miBlack)
	clip(ihx-rxi, top-3, 2*rxi+1, 3, hc)
	clip(ihx-rxi+1, top-4, 2*rxi-1, 1, hl)
	clip(ihx-rxi-3, top, 2*rxi+7, 1, hc)
	clip(ihx-rxi-2, top+1, 2*rxi+5, 1, hd)
	clip(ihx-rxi-4, top-2, 2, 2, hc) // the turned-up points
	clip(ihx+rxi+3, top-2, 2, 2, hc)
	clip(ihx-rxi-4, top-2, 1, 1, hl)
	if vw == vFront {
		clip(ihx-1, top+1, 3, 1, hc) // the front point dips over the brow
		clip(ihx-1, top-3, 3, 2, miBone)
		clip(ihx-1, top-2, 1, 1, miBlack)
		clip(ihx+1, top-2, 1, 1, miBlack)
		clip(ihx, top-1, 1, 1, miBone)
	} else if vw == vSide {
		clip(ihx+rxi+2, top+1, 2, 1, hc)
		clip(ihx+1, top-3, 2, 2, miBone)
		clip(ihx+2, top-2, 1, 1, miBlack)
	}
}

// miTorso dresses the torso: vest, open shirt and sash (Guybrush) or the long red coat (LeChuck).
func miTorso(b *pixBuf, s *CharSpec, vw, x0, x1, y0, y1, mid int, p Pose) {
	k := miKind(s)
	if k == miNone {
		return
	}
	b.cur = 2
	sw := miSwing(p)
	if k == miGuybrush {
		vest, vd := miVest, shade(miVest, 0.75)
		sash := miSash
		switch vw {
		case vFront:
			b.rect(x0, y0+1, 1, y1-y0-3, vest)
			b.rect(x1-1, y0+1, 1, y1-y0-3, vd)
			b.rect(x0+1, y0+4, 1, y1-y0-6, vest)
			b.rect(x1-2, y0+4, 1, y1-y0-6, vd)
			b.set(mid, y0, s.Skin) // the shirt is open at the neck
			b.set(mid, y0+1, s.Skin)
			b.set(mid, y0+3, hx(0xd8d4c8)) // a fold of the shirt
			b.rect(x0, y1-2, x1-x0, 2, sash)
			b.rect(x0, y1-2, x1-x0, 1, shade(sash, 1.2))
			b.rect(x1-2+sw, y1, 1, 3, sash) // the loose end of the sash
			b.set(x1-1+sw, y1+1, shade(sash, 0.8))
		case vBack:
			b.rect(x0, y0+1, x1-x0, y1-y0-3, vest)
			b.rect(mid, y0+2, 1, y1-y0-4, vd)
			b.rect(x0, y1-2, x1-x0, 2, sash)
			b.rect(mid-1+sw, y1, 1, 3, sash)
			b.rect(mid+sw, y1, 1, 2, shade(sash, 0.8))
		case vSide:
			b.rect(x0, y0+1, 2, y1-y0-3, vest)
			b.rect(x1-1, y0, 1, 2, s.Skin)
			b.rect(x0, y1-2, x1-x0, 2, sash)
			b.rect(x0-1-sw, y1-1, 1, 3, sash) // the loose end flies behind
			b.set(x0-2-sw, y1+1, shade(sash, 0.8))
		}
		return
	}
	// LeChuck: long red coat, gold buttons, belt and tails below the hips
	coat := s.Shirt
	coatD := miCoatDk
	belt := y1 - 3
	tail := 7
	switch vw {
	case vFront:
		b.rect(mid, y0+1, 1, y1-y0, coatD)
		for _, yy := range []int{y0 + 2, y0 + 4, y0 + 6} {
			b.set(mid-2, yy, miGold)
			b.set(mid+2, yy, miGold)
		}
		b.rect(x0, belt, x1-x0, 1, miBlack)
		b.rect(mid-1, belt, 3, 1, miGold)
		b.set(mid, belt, miBlack)
		for y := belt + 1; y <= y1+tail; y++ {
			t := float64(y-belt) / float64(tail+3)
			c := shade(coat, 1-0.25*t)
			off := 0
			if y > y1+2 {
				off = sw
			}
			b.rect(x0-1+off-(y-belt)/5, y, 3, 1, c)
			b.rect(x1-2+off, y, 3+(y-belt)/5, 1, shade(c, 0.85))
		}
		b.rect(x0-2+sw, y1+tail, 4, 1, miGold) // gold trim on the hem
		b.rect(x1-2+sw, y1+tail, 4, 1, miGold)
	case vBack:
		b.rect(x0, belt, x1-x0, 1, miBlack)
		b.set(mid-2, belt+1, miGold)
		b.set(mid+2, belt+1, miGold)
		for y := belt + 1; y <= y1+tail; y++ {
			t := float64(y-belt) / float64(tail+3)
			off := 0
			if y > y1+2 {
				off = sw
			}
			grow := (y - belt) / 5
			b.rect(x0-1-grow+off, y, x1-x0+2+2*grow, 1, shade(coat, 1-0.25*t))
			b.set(mid+off, y, coatD) // the slit at the back
		}
		b.rect(x0-2+sw, y1+tail, x1-x0+4, 1, miGold)
	case vSide:
		fx := x1 - 1
		for _, yy := range []int{y0 + 2, y0 + 4, y0 + 6} {
			b.set(fx-1, yy, miGold)
		}
		b.rect(x0, belt, x1-x0, 1, miBlack)
		b.set(fx-1, belt, miGold)
		for y := belt + 1; y <= y1+tail; y++ {
			t := float64(y-belt) / float64(tail+3)
			grow := (y - belt) / 3
			off := 0
			if y > y1+2 {
				off = sw
			}
			b.rect(x0-1-grow-off, y, 3+grow, 1, shade(coat, 1-0.25*t)) // the tails fly behind
			if y <= y1+4 {
				b.rect(fx-1, y, 2, 1, shade(coat, 0.9-0.2*t))
			}
		}
		b.rect(x0-1-(tail+3)/3-sw, y1+tail, 3+(tail+3)/3, 1, miGold)
	}
}

// miLeg turns the shoe into a big boot with a turned-down cuff.
func miLeg(b *pixBuf, s *CharSpec, vw int, sign float64, sxx, syy, legW int, dark bool) {
	k := miKind(s)
	if k == miNone {
		return
	}
	boot := s.Shoes
	cuff := hx(0x3a3438)
	if k == miLeChuck {
		cuff = hx(0x6a2234)
	}
	if dark {
		boot, cuff = shade(boot, 0.8), shade(cuff, 0.8)
	}
	var out int
	if sign < 0 {
		out = -1
	}
	switch vw {
	case vSide:
		b.rect(sxx, syy-3, legW, 3, boot)
		b.rect(sxx-1, syy-4, legW+1, 1, cuff)
		b.rect(sxx-1, syy, legW+3, 2, boot)
		b.set(sxx+legW+1, syy, shade(boot, 1.6)) // shine on the toe
	default:
		b.rect(sxx, syy-3, legW, 3, boot)
		b.rect(sxx+out, syy-4, legW+1, 1, cuff)
		b.rect(sxx+out, syy, legW+1, 2, boot)
		if vw == vFront {
			b.set(sxx+out+legW/2, syy, shade(boot, 1.6))
		}
	}
}
