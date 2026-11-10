package main

import (
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// Famous guests Q-Z drawn closer to their originals (our own parody pixel art, not a copy of any
// sprite). genChar (chars.go) calls four small layers on top of the usual CharSpec drawing:
//   qzHead  (masks, noses, moustaches, hats, ears, special hair),
//   qzTorso (emblems, vests, belts, bibs, shells, collars),
//   qzHand  (gloves and gauntlets on both hands),
//   qzHeld  (Thor's hammer and Zorro's rapier instead of a plain knife).
// Everything is baked into the cached frames like the rest of the character, so the cost is nil
// during play. noCharImpQ turns the layers off (tests compare with the stock drawing).
// ---------------------------------------------------------------------------

const (
	qzNone = iota
	qzSpider
	qzSuper
	qzWario
	qzWaluigi
	qzWoody
	qzThor
	qzThanos
	qzRambo
	qzRocky
	qzYoda
	qzStewie
	qzZoidberg
	qzSanta
	qzTrooper
	qzSpock
	qzRedshirt
	qzSteve
	qzSherlock
	qzZorro
	qzSailor
	qzVegeta
	qzRick
	qzTintin
	qzRaphael
)

var noCharImpQ bool

var qzKinds = map[string]int{
	"spiderman": qzSpider, "superman": qzSuper, "wario": qzWario, "waluigi": qzWaluigi, "woody": qzWoody,
	"thor": qzThor, "thanos": qzThanos, "rambo": qzRambo, "rocky": qzRocky, "yoda": qzYoda, "stewie": qzStewie,
	"zoidberg": qzZoidberg, "santa": qzSanta, "trooper": qzTrooper, "spock": qzSpock, "redshirt": qzRedshirt,
	"steve_cubos": qzSteve, "sherlock": qzSherlock, "zorro": qzZorro, "sailor_moon": qzSailor, "vegeta": qzVegeta,
	"rick": qzRick, "tintin": qzTintin, "tortuga_raphael": qzRaphael,
}

func qzKind(s *CharSpec) int {
	if noCharImpQ || s == nil {
		return qzNone
	}
	return qzKinds[s.Name]
}

var (
	qzBlack = hx(0x121216)
	qzWhite = hx(0xf4f4f6)
	qzGold  = hx(0xf0c838)
	qzRed   = hx(0xd02828)
	qzPink  = hx(0xf09a90)
)

// qzSwing: 0/1 bounce for tails, ponytails and tentacles while moving.
func qzSwing(p Pose) int {
	switch p.Anim {
	case AnimWalk, AnimRun, AnimHop, AnimPanic:
		return p.F % 2
	}
	return 0
}

// qzHead paints the head details. Same arguments as the other head layers (miHead, opHead).
func qzHead(b *pixBuf, s *CharSpec, vw int, p Pose, hxp, hyp, rx, ry float64, ey int) {
	k := qzKind(s)
	if k == qzNone {
		return
	}
	b.cur = 1
	ihx := int(math.Floor(hxp))
	top := int(math.Floor(hyp - ry))
	chin := int(math.Floor(hyp + ry))
	rxi := int(math.Floor(rx))
	sw := qzSwing(p)
	front, side, back := vw == vFront, vw == vSide, vw == vBack
	switch k {
	case qzSpider:
		web := shade(s.Mask, 0.55)
		switch {
		case front:
			b.set(ihx, top, web) // web seams
			b.set(ihx, top+1, web)
			b.set(ihx, ey+2, web)
			b.set(ihx-rxi, top+2, web)
			b.set(ihx+rxi, top+2, web)
			// big white lenses with a black rim
			b.rect(ihx-2, ey-1, 2, 2, qzWhite)
			b.rect(ihx+1, ey-1, 2, 2, qzWhite)
			b.set(ihx-2, ey-2, qzBlack)
			b.set(ihx+2, ey-2, qzBlack)
			b.set(ihx-1, ey+1, qzBlack)
			b.set(ihx+1, ey+1, qzBlack)
			b.set(ihx, ey, s.Mask)
		case side:
			b.rect(ihx+1, ey-1, 2, 2, qzWhite)
			b.set(ihx+1, ey-2, qzBlack)
			b.set(ihx+1, ey+1, qzBlack)
			b.set(ihx-1, top, web)
			b.set(ihx-1, top+1, web)
			b.set(ihx-2, ey, web)
		default:
			b.rect(ihx, top, 1, chin-top, web)
			b.rect(ihx-rxi+1, ey, 2*rxi-1, 1, web)
		}
	case qzSuper:
		if !back {
			b.set(ihx+1, top+1, s.Hair) // the famous curl on the forehead
			b.set(ihx, top+2, s.Hair)
		}
	case qzWario, qzWaluigi:
		wa := k == qzWario
		must := hx(0x2a1a10)
		nose := qzPink
		emb, letter := qzWhite, hx(0x3a3ab8)
		if !wa {
			letter = hx(0x6a2a8a)
		}
		switch {
		case front:
			b.set(ihx-1, top+1, emb) // cap badge with a letter
			b.set(ihx+1, top+1, emb)
			b.set(ihx, top+1, letter)
			b.set(ihx, top, emb)
			if wa {
				b.rect(ihx-1, ey, 3, 2, nose) // big round nose
				b.set(ihx-1, ey, s.Skin)
				b.set(ihx+1, ey, s.Skin)
				b.set(ihx-1, ey-1, qzBlack)
				b.set(ihx+1, ey-1, qzBlack)
				// zigzag moustache: tips up, middle down
				b.set(ihx-2, ey+1, must)
				b.set(ihx-1, ey+2, must)
				b.set(ihx, ey+2, must)
				b.set(ihx+1, ey+2, must)
				b.set(ihx+2, ey+1, must)
				b.set(ihx-2, ey+2, s.Skin)
				b.set(ihx+2, ey+2, s.Skin)
			} else {
				b.rect(ihx, ey, 1, 3, nose) // long thin nose
				b.set(ihx-1, ey+2, must)    // thin pointed moustache
				b.set(ihx+1, ey+2, must)
				b.set(ihx-2, ey+1, must)
				b.set(ihx+2, ey+1, must)
			}
		case side:
			b.set(ihx+rxi, top+1, emb)
			if wa {
				b.rect(ihx+rxi, ey, 2, 2, nose)
				b.set(ihx+rxi-1, ey+2, must)
				b.set(ihx+rxi, ey+2, must)
				b.set(ihx+rxi+1, ey+1, must)
			} else {
				b.rect(ihx+rxi, ey+1, 2, 1, nose)
				b.set(ihx+rxi+2, ey+1, nose)
				b.set(ihx+rxi, ey+2, must)
				b.set(ihx+rxi+1, ey+2, must)
			}
		}
	case qzRambo:
		band := hx(0xc82a2a)
		bd := shade(band, 0.72)
		switch {
		case front:
			b.rect(ihx-rxi, top+1, 2*rxi+1, 1, band) // headband
		case side:
			b.rect(ihx-rxi, top+1, 2*rxi+1, 1, band)
			b.set(ihx-rxi-1, top+1, bd) // the knot ends fly back
			b.set(ihx-rxi-2, top+2+sw, bd)
			b.set(ihx-rxi-3, top+2, bd)
		default:
			b.rect(ihx-rxi, top+1, 2*rxi+1, 1, band)
			b.rect(ihx-1+sw, top+2, 1, 4, bd)
			b.rect(ihx+1, top+2, 1, 3, bd)
		}
	case qzThanos:
		if front {
			ridge := shade(s.Skin, 0.72)
			b.rect(ihx-1, ey+2, 1, 2, ridge) // the grooved chin
			b.rect(ihx+1, ey+2, 1, 2, ridge)
			b.set(ihx, ey+3, ridge)
		} else if side {
			b.rect(ihx+1, ey+2, 1, 2, shade(s.Skin, 0.72))
		}
	case qzYoda:
		inner := hx(0xd89a8a)
		wisp := hx(0xe0e0d8)
		switch {
		case front, back:
			// long ears pointing out and a little down
			b.rect(ihx-rxi-3, ey-1, 3, 2, s.Skin)
			b.rect(ihx+rxi+1, ey-1, 3, 2, s.Skin)
			b.set(ihx-rxi-4, ey-2, s.Skin)
			b.set(ihx+rxi+4, ey-2, s.Skin)
			if front {
				b.rect(ihx-rxi-2, ey, 2, 1, inner)
				b.rect(ihx+rxi+1, ey, 2, 1, inner)
			}
			b.set(ihx-rxi+1, top, wisp)
			b.set(ihx+rxi-1, top, wisp)
			b.set(ihx-rxi, top+1, wisp)
		case side:
			b.rect(ihx-rxi-3, ey-1, 4, 2, s.Skin)
			b.set(ihx-rxi-4, ey-2, s.Skin)
			b.rect(ihx-rxi-2, ey, 2, 1, inner)
			b.set(ihx-1, top, wisp)
		}
		if front {
			b.set(ihx-1, ey, qzBlack)
			b.set(ihx+1, ey, qzBlack)
			b.set(ihx, ey+2, shade(s.Skin, 0.7))
		}
	case qzStewie:
		// the football head: pointed at both ends
		sk, sd := s.Skin, shade(s.Skin, 0.85)
		hair := hx(0x7a5a3a)
		if front || back {
			b.rect(ihx-rxi-1, top+3, 1, 3, sk)
			b.rect(ihx+rxi+1, top+3, 1, 3, sk)
			b.set(ihx-rxi-2, top+4, sd)
			b.set(ihx+rxi+2, top+4, sd)
			b.set(ihx-rxi-1, top+5, sd)
			b.set(ihx+rxi+1, top+5, sd)
			b.rect(ihx-2, top, 5, 1, hair)
			b.set(ihx-3, top+1, hair)
			b.set(ihx+3, top+1, hair)
		} else {
			b.rect(ihx-2, top, 4, 1, hair)
		}
	case qzZoidberg:
		t := shade(s.Skin, 0.8)
		if front || side {
			x0 := ihx - 2
			n := 4
			if side {
				x0, n = ihx, 3
			}
			// face tentacles that sway while he walks
			for i := 0; i < n; i++ {
				x := x0 + i + (i % 2)
				if side {
					x = x0 + i
				}
				l := 3 + (i+sw)%2
				for j := 0; j < l; j++ {
					off := 0
					if j == l-1 && (i+sw)%2 == 1 {
						off = 1
					}
					b.set(x+off, ey+2+j, t)
				}
			}
			b.rect(ihx-1, ey+1, 3, 1, shade(s.Skin, 0.65))
		}
		// head fins
		if !side {
			b.set(ihx-rxi-1, top+1, s.Skin)
			b.set(ihx-rxi-1, top+2, s.Skin)
			b.set(ihx+rxi+1, top+1, s.Skin)
			b.set(ihx+rxi+1, top+2, s.Skin)
		} else {
			b.rect(ihx-rxi-1, top+1, 1, 3, s.Skin)
		}
	case qzSanta:
		red := s.HatCol
		// the beanie of the base drawing hides the eyes: open the face again under a fur brim
		switch {
		case front:
			b.rect(ihx-rxi, ey, 2*rxi+1, 1, s.Skin)
			b.set(ihx-1, ey, qzBlack)
			b.set(ihx+1, ey, qzBlack)
			b.rect(ihx-rxi-1, ey-1, 2*rxi+3, 1, qzWhite) // fur brim
			b.rect(ihx+1, top-1, rxi, 1, red)            // floppy tip falls to one side
			b.set(ihx+rxi+1, top, red)
			b.rect(ihx+rxi+1, top+1, 2, 2, qzWhite) // pompom
			b.set(ihx, ey+1, hx(0xe89a90))          // red nose in the beard
		case back:
			b.rect(ihx-rxi, ey, 2*rxi+1, 2, s.Hair)
			b.rect(ihx-rxi-1, ey-1, 2*rxi+3, 1, qzWhite)
			b.rect(ihx-rxi, top-1, rxi, 1, red)
			b.set(ihx-rxi-1, top, red)
			b.rect(ihx-rxi-2, top+1, 2, 2, qzWhite)
		default:
			b.rect(ihx-1, ey, rxi+2, 1, s.Skin)
			b.rect(ihx-rxi, ey, 2, 2, s.Hair)
			b.set(ihx+1, ey, qzBlack)
			b.rect(ihx-rxi-1, ey-1, 2*rxi+2, 1, qzWhite)
			b.rect(ihx-rxi, top-1, rxi, 1, red)
			b.set(ihx-rxi-1, top, red)
			b.rect(ihx-rxi-2, top+1, 2, 2, qzWhite)
		}
	case qzTrooper:
		if front {
			b.rect(ihx-2, ey, 2, 1, qzBlack) // eye lenses
			b.rect(ihx+1, ey, 2, 1, qzBlack)
			b.set(ihx, ey, hx(0xd8d8e0))
			b.set(ihx-1, ey+2, hx(0x40404a)) // frown of the grille
			b.set(ihx+1, ey+2, hx(0x40404a))
			b.set(ihx-3, ey+1, hx(0x9aa0b0))
			b.set(ihx+3, ey+1, hx(0x9aa0b0))
		} else if side {
			b.rect(ihx+1, ey, 2, 1, qzBlack)
			b.set(ihx+2, ey+2, hx(0x40404a))
			b.set(ihx-1, ey+1, hx(0x9aa0b0))
		}
	case qzSpock:
		if !side {
			l, r := int(math.Floor(hxp-rx))-1, int(math.Floor(hxp+rx))
			b.rect(l, ey, 2, 2, s.Skin) // pointed ears
			b.rect(r-1, ey, 2, 2, s.Skin)
			b.set(l, ey-1, s.Skin)
			b.set(r, ey-1, s.Skin)
			b.set(l, ey+1, shade(s.Skin, 0.85))
			b.set(r, ey+1, shade(s.Skin, 0.85))
		} else {
			b.set(ihx-1, ey-2, s.Skin)
			b.set(ihx-1, ey-1, s.Skin)
		}
		if front {
			b.set(ihx-2, ey-1, s.Hair) // eyebrows swept up
			b.set(ihx-1, ey-2, s.Hair)
			b.set(ihx+2, ey-1, s.Hair)
			b.set(ihx+1, ey-2, s.Hair)
		} else if side {
			b.set(ihx+1, ey-1, s.Hair)
			b.set(ihx+2, ey-2, s.Hair)
		}
	case qzSteve:
		// the cube head
		hair := s.Hair
		sk := s.Skin
		switch {
		case front:
			b.rect(ihx-3, top-1, 7, 7, sk)
			b.rect(ihx-3, top-1, 7, 2, hair)
			b.set(ihx-3, top+1, hair)
			b.set(ihx+3, top+1, hair)
			b.set(ihx-2, ey, qzWhite)
			b.set(ihx-1, ey, hx(0x5a4ab0))
			b.set(ihx+1, ey, hx(0x5a4ab0))
			b.set(ihx+2, ey, qzWhite)
			b.set(ihx, ey+1, shade(sk, 0.8))
			b.rect(ihx-1, ey+2, 3, 1, shade(hair, 0.85))
			b.rect(ihx-3, top+5, 7, 1, shade(sk, 0.88))
		case back:
			b.rect(ihx-3, top-1, 7, 7, hair)
			b.rect(ihx-3, top+5, 7, 1, shade(hair, 0.8))
		default:
			b.rect(ihx-3, top-1, 7, 7, sk)
			b.rect(ihx-3, top-1, 7, 2, hair)
			b.rect(ihx-3, top+1, 3, 3, hair)
			b.set(ihx+2, ey, qzWhite)
			b.set(ihx+3, ey, hx(0x5a4ab0))
			b.rect(ihx+1, ey+2, 2, 1, shade(hair, 0.85))
			b.rect(ihx-3, top+5, 7, 1, shade(sk, 0.88))
		}
	case qzSherlock:
		tw := hx(0x9a8a62)
		ck := hx(0x6a5a3a)
		check := func(x, y, w, h int) {
			for yy := y; yy < y+h; yy++ {
				for xx := x; xx < x+w; xx++ {
					c := tw
					if (xx+yy)%3 == 0 {
						c = ck
					}
					b.set(xx, yy, c)
				}
			}
		}
		switch {
		case front, back:
			check(ihx-rxi-1, top-1, 2*rxi+3, 3) // deerstalker crown
			b.rect(ihx-rxi-1, top+2, 1, 3, tw)  // ear flaps
			b.rect(ihx+rxi+1, top+2, 1, 3, tw)
			b.set(ihx, top-2, ck) // the flaps tied on top
			if front {
				b.rect(ihx-1, top+2, 3, 1, ck) // front peak
				b.set(ihx+1, ey+2, hx(0x5a3a1a))
				b.set(ihx+2, ey+3, hx(0x5a3a1a)) // pipe
				b.rect(ihx+2, ey+2, 2, 1, hx(0x3a2414))
			}
		default:
			check(ihx-rxi, top-1, 2*rxi+1, 3)
			b.rect(ihx-rxi-2, top+1, 2*rxi+5, 1, ck) // a peak at the front and one at the back
			b.rect(ihx-rxi, top+2, 1, 3, tw)
			b.set(ihx+rxi+1, ey+2, hx(0x5a3a1a)) // pipe
			b.rect(ihx+rxi+2, ey+1, 1, 2, hx(0x3a2414))
		}
	case qzZorro:
		mask := hx(0x0e0e12)
		switch {
		case front:
			b.rect(ihx-rxi, ey-1, 2*rxi+1, 2, mask)
			b.set(ihx-1, ey, qzWhite)
			b.set(ihx+1, ey, qzWhite)
		case side:
			b.rect(ihx-rxi, ey-1, 2*rxi+1, 2, mask)
			b.set(ihx+1, ey, qzWhite)
			b.set(ihx-rxi-1, ey+sw, mask) // knot tails
			b.set(ihx-rxi-2, ey+1, mask)
		default:
			b.rect(ihx-rxi, ey-1, 2*rxi+1, 2, mask)
			b.set(ihx-1+sw, ey+1, mask)
			b.set(ihx+1, ey+2, mask)
		}
	case qzSailor:
		h := s.Hair
		hd := shade(h, 0.82)
		// two long pigtails falling from the buns, swinging when she walks
		tail := func(x int, dir int) {
			b.rect(x, int(math.Floor(hyp)), 2, 10, h)
			b.rect(x+dir*sw, int(math.Floor(hyp))+10, 2, 3, hd)
			b.set(x, int(math.Floor(hyp))+4, hd)
		}
		switch {
		case front, back:
			tail(ihx-rxi-3, -1)
			tail(ihx+rxi+2, 1)
		default:
			tail(ihx-rxi-2, -1)
		}
		if front {
			b.rect(ihx-1, top+1, 3, 1, qzGold) // tiara with a red gem
			b.set(ihx, top+1, qzRed)
		}
	case qzVegeta:
		h := s.Hair
		hl := shade(h, 1.8)
		switch {
		case front, back:
			// tall flame-shaped hair with a widow's peak
			b.rect(ihx-2, top-2, 5, 3, h)
			b.rect(ihx-2, top-4, 5, 2, h)
			b.rect(ihx-1, top-6, 3, 2, h)
			b.set(ihx, top-7, h)
			b.set(ihx-3, top-1, h)
			b.set(ihx+3, top-1, h)
			b.set(ihx-1, top-4, hl)
			b.set(ihx, top-2, hl)
			if front {
				b.set(ihx-2, top+1, s.Skin) // receding temples and a sharp peak
				b.set(ihx+2, top+1, s.Skin)
				b.set(ihx, top+1, h)
			}
		default:
			b.rect(ihx-2, top-3, 5, 4, h)
			b.rect(ihx-2, top-5, 4, 2, h)
			b.rect(ihx-1, top-6, 2, 1, h)
			b.set(ihx-3, top-2, h)
			b.set(ihx, top-3, hl)
		}
	case qzRick:
		if front {
			b.rect(ihx-2, ey-1, 5, 1, s.Hair) // unibrow
			b.set(ihx+1, ey+2, hx(0x8ad070))  // a drop of drool
		} else if side {
			b.rect(ihx, ey-1, 3, 1, s.Hair)
		}
	case qzRaphael:
		band := hx(0xd02828)
		bd := shade(band, 0.7)
		switch {
		case front:
			b.rect(ihx-rxi, ey-1, 2*rxi+1, 2, band)
			b.set(ihx-1, ey, qzWhite)
			b.set(ihx+1, ey, qzWhite)
			b.set(ihx, ey+2, shade(s.Skin, 0.7)) // a grumpy mouth
		case side:
			b.rect(ihx-rxi, ey-1, 2*rxi+1, 2, band)
			b.set(ihx+1, ey, qzWhite)
			b.set(ihx-rxi-1, ey-1+sw, bd) // the knot tails fly back
			b.set(ihx-rxi-2, ey+sw, bd)
			b.set(ihx-rxi-3, ey, bd)
		default:
			b.rect(ihx-rxi, ey-1, 2*rxi+1, 2, band)
			b.rect(ihx-1+sw, ey+1, 1, 3, bd)
			b.rect(ihx+1, ey+1, 1, 3, bd)
		}
	}
}

// qzTorso paints the chest, belt and back details over the torso rectangle [x0,x1) x [y0,y1].
func qzTorso(b *pixBuf, s *CharSpec, vw, x0, x1, y0, y1, mid int, p Pose) {
	k := qzKind(s)
	if k == qzNone {
		return
	}
	b.cur = 2
	tw := x1 - x0
	front, side, back := vw == vFront, vw == vSide, vw == vBack
	switch k {
	case qzSpider:
		blue := s.Shirt2
		web := shade(s.Shirt, 0.55)
		if !side && tw >= 5 {
			b.rect(x0, y0+3, 1, y1-y0-3, blue) // blue sides
			b.rect(x1-1, y0+3, 1, y1-y0-3, blue)
		}
		if front {
			b.rect(mid, y0+2, 1, 3, qzBlack) // spider emblem
			b.set(mid-1, y0+2, qzBlack)
			b.set(mid+1, y0+2, qzBlack)
			b.set(mid-1, y0+4, qzBlack)
			b.set(mid+1, y0+4, qzBlack)
			b.set(mid-1, y0+1, web)
			b.set(mid+1, y0+1, web)
		} else if back {
			b.rect(mid, y0+1, 1, 5, web)
			b.rect(x0+1, y0+3, tw-2, 1, web)
			b.set(mid-1, y0+2, shade(s.Shirt, 0.4))
			b.set(mid+1, y0+2, shade(s.Shirt, 0.4))
		} else {
			b.rect(x0, y0+3, tw, 1, web)
		}
	case qzSuper:
		red := hx(0xc82a2a)
		if front {
			// the diamond shield: red rim, yellow inside, red stroke
			b.rect(mid-2, y0+1, 5, 1, red)
			b.rect(mid-1, y0+1, 3, 1, qzGold)
			b.rect(mid-1, y0+2, 3, 1, red)
			b.set(mid, y0+2, qzGold)
			b.set(mid, y0+3, red)
		}
		b.rect(x0, y1-2, tw, 1, qzGold) // belt
		b.rect(x0, y1-1, tw, 2, red)    // trunks
		if front {
			b.set(mid, y1-2, hx(0xc89a20))
		}
	case qzWario, qzWaluigi:
		bib := s.Pants
		btn := qzGold
		if k == qzWaluigi {
			btn = hx(0xf0d040)
		}
		if front {
			b.rect(x0+1, y0+4, tw-2, y1-y0-3, bib)
			b.set(x0+1, y0+4, btn)
			b.set(x1-2, y0+4, btn)
			b.rect(x0+1, y0, 1, 4, bib) // straps
			b.rect(x1-2, y0, 1, 4, bib)
		} else if back {
			b.rect(x0+1, y0, 1, y1-y0, bib)
			b.rect(x1-2, y0, 1, y1-y0, bib)
			b.rect(x0, y1-3, tw, 4, bib)
		} else {
			b.rect(x0, y0+4, tw, y1-y0-3, bib)
			b.rect(x0+tw/2, y0, 1, 4, bib)
		}
	case qzWoody:
		spot := func(x, y, w, h int) {
			for yy := y; yy < y+h; yy++ {
				for xx := x; xx < x+w; xx++ {
					c := qzWhite
					if (xx*2+yy)%5 == 0 || (xx+yy*3)%7 == 0 {
						c = qzBlack
					}
					b.set(xx, yy, c)
				}
			}
		}
		switch {
		case front:
			vwd := maxI(1, tw/2-1)
			spot(x0, y0+1, vwd, y1-y0-2) // cow-print vest
			spot(x1-vwd, y0+1, vwd, y1-y0-2)
			b.set(x0, y0+3, qzGold) // sheriff star
			b.rect(mid-1, y0, 3, 1, qzRed)
			b.set(mid, y0+1, qzRed) // bandana knot
		case back:
			spot(x0, y0+1, tw, y1-y0-2)
			b.rect(x0+1, y0, tw-2, 1, qzRed)
		default:
			spot(x0, y0+1, tw/2+1, y1-y0-2)
			b.set(x1-1, y0, qzRed)
		}
		b.rect(x0, y1-1, tw, 1, hx(0x5a3418)) // belt
		if front {
			b.set(mid, y1-1, qzGold)
		}
	case qzThor:
		disc := hx(0xc8ccd8)
		if front {
			for _, d := range [][2]int{{-1, 2}, {1, 2}, {-1, 4}, {1, 4}} {
				b.set(mid+d[0], y0+d[1], disc)
			}
			b.set(mid-1, y0+2, qzWhite)
		} else if side {
			b.set(x1-2, y0+2, disc)
			b.set(x1-2, y0+4, disc)
		}
		b.rect(x0, y1-2, tw, 1, hx(0x5a4a2a))
	case qzThanos:
		g := hx(0xe0b028)
		gd := shade(g, 0.75)
		b.rect(x0, y0, tw, 2, g) // golden yoke and belt
		if front {
			b.rect(mid-1, y0+2, 3, 2, g)
			b.set(mid, y0+3, gd)
		}
		b.rect(x0, y1-2, tw, 1, g)
		b.set(x0, y0+1, gd)
		b.set(x1-1, y0+1, gd)
	case qzRambo:
		belt := hx(0x4a3a22)
		// a cartridge belt across the chest
		for i := 0; i <= y1-y0-2; i++ {
			x := x0 + i*tw/maxI(1, y1-y0-1)
			if back {
				x = x1 - 1 - i*tw/maxI(1, y1-y0-1)
			}
			if side {
				x = x0 + (i*tw)/maxI(1, y1-y0-1)
			}
			c := belt
			if i%2 == 0 {
				c = hx(0xd8b040)
			}
			b.set(x, y0+i, c)
		}
	case qzRocky:
		b.rect(x0, y1-1, tw, 1, qzWhite) // white waistband of the boxing trunks
	case qzStewie:
		red := s.Pants
		if front {
			b.rect(x0, y0+3, tw, y1-y0-2, red)
			b.rect(x0, y0, 1, 3, red)
			b.rect(x1-1, y0, 1, 3, red)
		} else {
			b.rect(x0, y0+3, tw, y1-y0-2, red)
		}
	case qzZoidberg:
		if front {
			b.rect(mid, y0+1, 1, y1-y0-2, hx(0x3a5aa0)) // the open coat over a shirt
			b.set(mid-1, y0, s.Skin)
			b.set(mid+1, y0, s.Skin)
		}
	case qzSanta:
		if front {
			b.rect(mid, y0, 1, y1-y0-1, qzWhite)
		}
		b.rect(x0, y1-3, tw, 2, qzBlack) // wide black belt
		if front {
			b.rect(mid-1, y1-3, 3, 2, qzGold)
			b.set(mid, y1-3, qzBlack)
		}
		b.rect(x0, y1-1, tw, 1, qzWhite) // fur hem
	case qzTrooper:
		b.rect(x0, y1-3, tw, 1, qzBlack)
		b.rect(x0+1, y1-2, tw-2, 1, hx(0xe8e8ee))
		if front {
			b.rect(x0, y0+3, tw, 1, hx(0x30303a))
			b.set(mid-1, y0+1, hx(0x30303a))
			b.set(mid+1, y0+1, hx(0x30303a))
			b.set(mid-1, y1-2, hx(0x5a5a64))
		} else if back {
			b.rect(x0+1, y0+1, tw-2, 3, hx(0x8a8a96))
		}
	case qzSpock, qzRedshirt:
		b.rect(x0+1, y0, tw-2, 1, qzBlack) // black collar
		if front {
			b.set(x1-2, y0+2, qzGold) // gold insignia
			b.set(x1-2, y0+3, shade(qzGold, 0.75))
		}
	case qzSherlock:
		tw2 := hx(0x8a7a52)
		if !side {
			b.rect(x0-1, y0, tw+2, 4, s.Shirt) // a short Inverness cape over the coat
			for x := x0 - 1; x < x1+1; x++ {
				if x%2 == 0 {
					b.set(x, y0+3, tw2)
				}
			}
		} else {
			b.rect(x0-1, y0, tw+1, 4, s.Shirt)
		}
		b.rect(x0-1, y0+4, tw+2, 1, shade(s.Shirt, 0.7))
	case qzZorro:
		if front {
			b.set(mid-1, y0, qzWhite) // shirt collar
			b.set(mid+1, y0, qzWhite)
		}
		b.rect(x0, y1-2, tw, 1, hx(0x9a1a1a)) // red sash
		if side || back {
			b.set(x0, y1-1, hx(0x9a1a1a))
		}
	case qzSailor:
		blue := hx(0x2a50c0)
		if front {
			b.rect(x0, y0, tw, 1, blue) // sailor collar
			b.set(x0, y0+1, blue)
			b.set(x1-1, y0+1, blue)
			b.rect(mid-1, y0+1, 3, 1, qzRed) // big red bow
			b.rect(mid, y0+2, 1, 2, qzRed)
			b.set(mid, y0+1, shade(qzRed, 0.7))
		} else if back {
			b.rect(x0, y0, tw, 3, blue)
			b.rect(x0+1, y0+2, tw-2, 1, qzWhite)
		} else {
			b.rect(x0, y0, 2, 3, blue)
		}
	case qzVegeta:
		yel := hx(0xe8c870)
		if front {
			b.rect(x0+1, y0+1, tw-2, y1-y0-3, qzWhite) // the white chest plate
			b.set(x0, y0, yel)                         // shoulder pads
			b.set(x1-1, y0, yel)
			b.rect(x0+1, y1-2, tw-2, 1, yel)
		} else if back {
			b.rect(x0+1, y0+1, tw-2, y1-y0-3, shade(qzWhite, 0.9))
			b.set(x0, y0, yel)
			b.set(x1-1, y0, yel)
		} else {
			b.rect(x0+1, y0+1, tw-1, y1-y0-3, qzWhite)
			b.set(x1-1, y0, yel)
		}
	case qzRick:
		shirt := hx(0x8ac8d8)
		if front {
			b.rect(mid-1, y0, 3, y1-y0-1, shirt) // the open lab coat over his blue shirt
		}
		// coat tails
		switch vw {
		case vFront:
			b.rect(x0, y1, 1, 4, s.Shirt)
			b.rect(x1-1, y1, 1, 4, s.Shirt)
		case vBack:
			b.rect(x0, y1-1, tw, 5, s.Shirt)
			b.rect(mid, y1+1, 1, 3, shade(s.Shirt, 0.82))
		default:
			b.rect(x0, y1, 1, 4, s.Shirt)
		}
	case qzTintin:
		if front {
			b.set(mid-1, y0+1, qzWhite) // white shirt collar under the sweater
			b.set(mid+1, y0+1, qzWhite)
			b.set(mid, y0+2, shade(qzWhite, 0.85))
		}
	case qzRaphael:
		plas := hx(0xd8c070)
		shell := hx(0x3a6a2a)
		belt := hx(0x7a5a2a)
		switch {
		case front:
			b.rect(x0+1, y0+1, tw-2, y1-y0-2, plas) // plastron
			for y := y0 + 2; y < y1-1; y += 2 {
				b.rect(x0+1, y, tw-2, 1, shade(plas, 0.8))
			}
		case back:
			b.rect(x0-1, y0, tw+2, y1-y0, shell) // shell
			for y := y0 + 1; y < y1; y++ {
				for x := x0 - 1; x < x1+1; x++ {
					if (x+y*2)%4 == 0 {
						b.set(x, y, shade(shell, 0.7))
					}
				}
			}
			b.rect(x0, y0, tw, 1, shade(shell, 1.3))
		default:
			b.rect(x0-1, y0+1, 2, y1-y0-1, shell)
			b.rect(x1-1, y0+1, 1, y1-y0-2, plas)
		}
		b.rect(x0, y1-1, tw, 1, belt)
	}
}

// qzHand paints gloves over the hands drawn by drawArm. id is 3 or 4 (the two arms).
func qzHand(b *pixBuf, s *CharSpec, vw int, ex, ey float64, armW int, dark bool, id uint8) {
	k := qzKind(s)
	if k == qzNone {
		return
	}
	x, y := int(math.Floor(ex)), int(math.Floor(ey))
	dk := func(c color.RGBA) color.RGBA {
		if dark {
			return shade(c, 0.8)
		}
		return c
	}
	switch k {
	case qzWario, qzWaluigi, qzVegeta:
		b.rect(x, y, armW, 2, dk(qzWhite))
	case qzSpider:
		b.rect(x, y, armW, 2, dk(s.Mask))
	case qzRocky:
		b.rect(x-1, y-1, armW+2, 3, dk(hx(0xc82020))) // boxing gloves
		b.set(x-1, y+1, dk(hx(0x8a1010)))
		b.rect(x, y-2, armW, 1, dk(qzWhite))
	case qzThanos:
		if id == 4 || (vw == vSide) {
			b.rect(x-1, y-1, armW+2, 3, dk(hx(0xe0b028))) // the gauntlet with its stones
			b.set(x-1, y, hx(0x40c0f0))
			b.set(x, y-1, hx(0xe03030))
			b.set(x+armW, y, hx(0x40d060))
		}
	case qzSanta:
		b.rect(x, y-1, armW, 1, dk(qzWhite))
	}
}

// qzHeld draws Thor's hammer and Zorro's rapier in place of a plain (or no) object. Returns true when
// it drew the object.
func qzHeld(b *pixBuf, s *CharSpec, kind int, vw int, hx0, hy0 float64, p Pose) bool {
	k := qzKind(s)
	if k != qzThor && k != qzZorro {
		return false
	}
	if kind != HoldNone && kind != HoldKnife {
		return false
	}
	x, y := int(math.Floor(hx0)), int(math.Floor(hy0))
	switch k {
	case qzThor:
		wood := hx(0x6a4426)
		head := hx(0x9aa0aa)
		b.rect(x, y-1, 1, 5, wood)
		b.set(x, y-2, hx(0x3a2a1a))
		hy := y + 3
		b.rect(x-2, hy, 5, 3, head)
		b.rect(x-2, hy, 5, 1, hx(0xc8ccd8))
		b.set(x+2, hy+2, shade(head, 0.7))
		if kind == HoldKnife {
			b.set(x+3, hy-1, hx(0x9ad8ff)) // a little spark when he swings
		}
	case qzZorro:
		steel := hx(0xd8dce4)
		b.rect(x-1, y, 3, 1, qzGold) // guard
		if vw == vSide {
			b.line(x+1, y+1, x+4, y+8, steel)
		} else {
			b.line(x, y+1, x+1, y+9, steel)
		}
	}
	return true
}
