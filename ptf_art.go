package main

import "math"

// ---------------------------------------------------------------------------
// Our own pixel-art takes (not a copy of any artwork) for four looks, as overlay hooks that
// genChar calls (head, torso, legs, hands, held object). Visual only: hitbox, speed and stats
// do not change.
//   - the Pope VIP (vip.go, id "pope"): a white skullcap-mitre with a golden band and a golden
//     cross, grey hair at the sides, a long white cassock to the floor with a golden stole across
//     the chest and a golden stripe down the middle, black shoes and a golden cross staff.
//     VIP specs have no Name, so the cassock is marked with Pattern = ptfPopeMark.
//   - "michael_jackson": a stage outfit (black fedora, long black hair, black suit over a white
//     shirt, black trousers short enough to show bright white socks, black loafers and one
//     shining white glove).
//   - "don_quijote": a lean knight with a brass basin hat with a red feather, long white
//     moustache and pointed goatee, a long nose, a grey-blue breastplate over a light blue tunic,
//     tall boots, a round leather shield and a long lance slung on his back.
//   - "sancho_panza": short and round, a wide straw hat, brown beard, red neckerchief, white
//     shirt under a brown waistcoat, brown trousers and rope-soled sandals.
// The lance and the shield belong to the torso (part 2), so the exported pictures of
// special_people keep them when the game animates the rig.
// ---------------------------------------------------------------------------

const ptfPopeMark = 41 // Pattern of the Pope's cassock (an unused pattern number)

const (
	ptfNone = iota
	ptfPope
	ptfMJ
	ptfQuijote
	ptfSancho
)

func ptfKind(s *CharSpec) int {
	switch s.Name {
	case "michael_jackson":
		return ptfMJ
	case "don_quijote":
		return ptfQuijote
	case "sancho_panza":
		return ptfSancho
	case "":
		if s.Top == TopRobe && s.Pattern == ptfPopeMark {
			return ptfPope
		}
	}
	return ptfNone
}

var (
	ptfGold   = hx(0xe8c040)
	ptfGoldD  = hx(0xb08a20)
	ptfGoldL  = hx(0xfff0a0)
	ptfWhite  = hx(0xfafaf6)
	ptfWhiteD = hx(0xd8d6cc)
	ptfGrey   = hx(0xc4c4c8)
	ptfBlack  = hx(0x141418)
	ptfSock   = hx(0xffffff)
	ptfShine  = hx(0xd8f0ff)
	ptfBrass  = hx(0xd4aa48)
	ptfBrassD = hx(0x9a7a30)
	ptfBrassL = hx(0xf4dc8a)
	ptfRed    = hx(0xd0302c)
	ptfWood   = hx(0x8a6236)
	ptfWoodD  = hx(0x5a3e22)
	ptfSteel  = hx(0xdce2ea)
	ptfLeath  = hx(0xb08850)
	ptfLeathD = hx(0x6a4a28)
	ptfTunic  = hx(0x9cc4e4)
	ptfBoot   = hx(0x5a3a20)
	ptfStraw  = hx(0xe2c46a)
	ptfStrawD = hx(0xb0923e)
	ptfVest   = hx(0x7a5230)
	ptfVestD  = hx(0x5e3e22)
	ptfBeardS = hx(0x6a4426)
	ptfRope   = hx(0xe8dcb4)
)

// ptfHead: hats, hair, faces.
func ptfHead(b *pixBuf, s *CharSpec, vw int, p Pose, hxp, hyp, rx, ry float64, ey int) {
	k := ptfKind(s)
	if k == ptfNone {
		return
	}
	b.cur = 1
	ihx := int(math.Floor(hxp))
	top := int(math.Floor(hyp - ry))
	hw := int(math.Floor(rx + 0.5))
	nose := int(math.Floor(hxp + rx)) // front edge in profile
	front, side, back := vw == vFront, vw == vSide, vw == vBack
	switch k {
	case ptfPope:
		// grey hair at the sides (all round at the back)
		switch {
		case front:
			b.rect(ihx-hw, ey-1, 1, 3, ptfGrey)
			b.rect(ihx+hw, ey-1, 1, 3, ptfGrey)
		case side:
			b.rect(ihx-hw, ey-1, 2, 3, ptfGrey)
		case back:
			b.rect(ihx-hw, top+2, 2*hw+1, ey+1-(top+2)+1, ptfGrey)
		}
		// the mitre: white, a golden band at the base and a golden cross
		b.rect(ihx-hw, top+1, 2*hw+1, 1, ptfGold)
		for y := top - 3; y <= top; y++ {
			b.rect(ihx-hw+1, y, 2*hw-1, 1, ptfWhite)
		}
		b.rect(ihx-1, top-4, 3, 1, ptfWhite)
		b.set(ihx, top-5, ptfWhite)
		b.rect(ihx+hw-1, top-3, 1, 4, ptfWhiteD) // shade on one side
		if !side {
			b.rect(ihx, top-4, 1, 5, ptfGold)
			b.rect(ihx-1, top-2, 3, 1, ptfGold)
			b.set(ihx, top-4, ptfGoldL)
		} else {
			b.rect(ihx-hw+1, top-3, 1, 4, ptfGold) // the edge of the front plate
		}
	case ptfMJ:
		// shoulder-length black hair under the fedora, a lock over the forehead
		switch {
		case front:
			b.rect(ihx-hw-1, ey-1, 1, 5, s.Hair)
			b.rect(ihx+hw+1, ey-1, 1, 5, s.Hair)
			b.set(ihx-hw+1, ey-1, s.Hair)
		case side:
			b.rect(ihx-hw-1, ey-1, 2, 5, s.Hair)
		case back:
			b.rect(ihx-hw-1, ey-1, 2*hw+3, 5, s.Hair)
		}
	case ptfQuijote:
		white := hx(0xf2f2f0)
		skin := s.Skin
		switch {
		case front:
			b.set(ihx, ey+1, shade(skin, 0.8)) // the long nose
			b.set(ihx, ey, shade(skin, 0.9))
			b.set(ihx-1, ey+2, white) // drooping moustache
			b.set(ihx+1, ey+2, white)
			b.set(ihx-2, ey+3, white)
			b.set(ihx+2, ey+3, white)
			b.rect(ihx, ey+3, 1, 3, white) // pointed goatee
		case side:
			b.set(nose+1, ey+1, skin)
			b.set(nose+1, ey, shade(skin, 0.85))
			b.set(nose, ey+2, white)
			b.set(nose-1, ey+3, white)
			b.rect(nose-1, ey+3, 1, 3, white)
			b.set(nose-2, ey+5, white)
		}
		// the brass basin: a wide brim tilted to one side and a round crown
		bx0, bx1 := ihx-hw-2, ihx+hw+2
		for x := bx0; x <= bx1; x++ {
			y := top
			if x < ihx-hw {
				y = top + 1
			}
			b.set(x, y, ptfBrass)
		}
		b.set(bx1, top, ptfBrassD)
		b.rect(ihx-hw+1, top-1, 2*hw-1, 1, ptfBrass)
		b.rect(ihx-hw+2, top-2, 2*hw-3, 1, ptfBrass)
		b.set(ihx-1, top-2, ptfBrassL)
		b.set(ihx+hw-1, top-1, ptfBrassD)
		// the feather
		fx, dir := ihx+hw, 1
		if side {
			fx, dir = ihx-hw, -1
		}
		b.set(fx, top-2, ptfRed)
		b.set(fx+dir, top-3, ptfRed)
		b.set(fx+2*dir, top-4, white)
	case ptfSancho:
		// a wide straw hat
		b.rect(ihx-hw-2, top, 2*hw+5, 1, ptfStraw)
		b.set(ihx-hw-2, top, ptfStrawD)
		b.set(ihx+hw+2, top, ptfStrawD)
		b.rect(ihx-hw+1, top-1, 2*hw-1, 1, ptfStrawD) // the band
		b.rect(ihx-hw+1, top-3, 2*hw-1, 2, ptfStraw)
		b.set(ihx-hw+1, top-3, ptfStrawD)
		// a round brown beard
		switch {
		case front:
			b.rect(ihx-hw+1, ey+1, 2*hw-1, 2, ptfBeardS)
			b.rect(ihx-1, ey+3, 3, 1, ptfBeardS)
			b.set(ihx, ey+1, shade(s.Skin, 0.7)) // mouth
		case side:
			b.rect(nose-2, ey+1, 3, 2, ptfBeardS)
			b.rect(nose-3, ey+3, 3, 1, ptfBeardS)
		}
	}
}

// ptfTorso: cassock, stole, breastplate, shield, lance, waistcoat.
func ptfTorso(b *pixBuf, s *CharSpec, vw, x0, x1, y0, y1, mid int, p Pose) {
	k := ptfKind(s)
	if k == ptfNone {
		return
	}
	b.cur = 2
	tw := x1 - x0
	front, side := vw == vFront, vw == vSide
	switch k {
	case ptfPope:
		// the golden stole across the chest and the stripe down the cassock
		b.rect(x0, y0+2, tw, 1, ptfGold)
		b.rect(x0, y0+3, tw, 1, ptfGoldD)
		if !side {
			b.rect(mid, y0+2, 1, y1+13-(y0+2), ptfGold)
			b.set(mid, y0+2, ptfGoldL)
		} else {
			b.rect(x1, y0+2, 1, y1+13-(y0+2), ptfGold) // the stripe at the front of the profile
		}
		b.rect(x0-1, y1+12, tw+2, 1, ptfWhiteD) // the hem
	case ptfMJ:
		// sleeves cut short enough to show the white cuffs (at the hands), a black belt line
		if front {
			b.rect(mid-1, y0, 3, 1, ptfSock)
		}
	case ptfQuijote:
		steelL := shade(s.Shirt, 1.25)
		// a light blue tunic under the breastplate
		b.rect(x0-1, y1-1, tw+2, 3, ptfTunic)
		b.rect(x0-1, y1+1, tw+2, 1, shade(ptfTunic, 0.85))
		b.rect(x0, y0, tw, 1, ptfTunic) // collar
		if front {
			b.set(x0, y0+2, steelL)
			b.set(x1-1, y0+2, steelL)
		}
		// the long lance slung on his back, and the round shield
		switch vw {
		case vFront:
			b.line(x1+2, y1+6, x1+5, 2, ptfWood)
			b.line(x1+3, y1+6, x1+6, 2, ptfWoodD)
			b.rect(x1+5, 0, 2, 2, ptfSteel)
			b.set(x1+6, 0, shade(ptfSteel, 0.8))
		case vBack:
			b.line(x0-3, y1+6, x0-6, 2, ptfWood)
			b.line(x0-2, y1+6, x0-5, 2, ptfWoodD)
			b.rect(x0-7, 0, 2, 2, ptfSteel)
			b.ellipse(float64(mid)+0.5, float64(y0)+3.5, 3.6, 3.6, ptfLeathD)
			b.ellipse(float64(mid)+0.5, float64(y0)+3.5, 2.6, 2.6, ptfLeath)
			b.set(mid, y0+3, ptfBrass)
			b.set(mid, y0+4, ptfBrassD)
		case vSide:
			b.line(x0-5, y1+6, x0+3, 1, ptfWood)
			b.line(x0-4, y1+6, x0+4, 1, ptfWoodD)
			b.rect(x0+4, 0, 2, 2, ptfSteel)
			b.rect(x0-2, y0+1, 2, 7, ptfLeath)
			b.rect(x0-2, y0+1, 1, 7, ptfLeathD)
			b.set(x0-1, y0+4, ptfBrass)
		}
	case ptfSancho:
		// the brown waistcoat open over the white shirt, and the red neckerchief
		switch vw {
		case vFront:
			b.rect(x0, y0, 2, y1-y0, ptfVest)
			b.rect(x1-2, y0, 2, y1-y0, ptfVest)
			b.set(x0+1, y0+3, ptfVestD)
			b.set(x1-2, y0+3, ptfVestD)
			b.rect(x0+1, y0, tw-2, 1, ptfRed)
			b.rect(mid-1, y0+1, 3, 1, ptfRed)
			b.set(mid, y0+2, shade(ptfRed, 0.8))
		case vBack:
			b.rect(x0, y0+1, tw, y1-y0-1, ptfVest)
			b.rect(x0, y0, tw, 1, ptfRed)
			b.set(mid, y0+1, ptfRed)
		case vSide:
			b.rect(x0, y0, tw-2, y1-y0, ptfVest)
			b.rect(x0, y0, tw, 1, ptfRed)
			b.set(x1-1, y0+1, ptfRed)
		}
		b.rect(x0, y1-1, tw, 1, ptfVestD) // the belt
	}
}

// ptfLeg: socks and boots over the shoe (called from drawLeg).
func ptfLeg(b *pixBuf, s *CharSpec, vw int, sxx, syy, legW int, dark bool) {
	k := ptfKind(s)
	if k != ptfMJ && k != ptfQuijote && k != ptfSancho {
		return
	}
	x, w := sxx, legW
	if vw == vSide {
		x, w = sxx-1, legW+2
	}
	switch k {
	case ptfMJ: // bright white socks between the trousers and the loafers: the shins turn white
		for y := syy - 2; y < syy; y++ {
			for xx := sxx - 1; xx <= sxx+legW; xx++ {
				if xx >= 0 && y >= 0 && xx < b.w && y < b.h && b.p[(y*b.w+xx)*4+3] != 0 {
					b.set(xx, y, ptfSock)
				}
			}
		}
	case ptfQuijote: // tall riding boots
		boot := ptfBoot
		if dark {
			boot = shade(boot, 0.8)
		}
		b.rect(sxx, syy-4, legW, 4, boot)
		b.rect(sxx, syy-4, legW, 1, shade(boot, 1.3))
	case ptfSancho: // rope-soled sandals
		b.rect(x, syy+1, w, 1, ptfRope)
	}
}

// ptfHand: Michael's single shining glove (right hand).
func ptfHand(b *pixBuf, s *CharSpec, vw int, ex, ey float64, armW int, dark bool, id uint8) {
	if ptfKind(s) != ptfMJ || id != 4 {
		return
	}
	x, y := int(math.Floor(ex+0.5)), int(math.Floor(ey+0.5))
	b.rect(x, y, armW, 2, ptfSock)
	b.set(x, y, ptfShine)
	b.set(x+armW-1, y+1, hx(0xc8d8f0))
}

// ptfHeld: the Pope's golden cross staff instead of the plain staff. True when it painted.
func ptfHeld(b *pixBuf, s *CharSpec, kind, vw int, hx0, hy0 float64) bool {
	if ptfKind(s) != ptfPope || kind != HoldSaber {
		return false
	}
	x, y := int(math.Floor(hx0)), int(math.Floor(hy0))
	top := maxI(1, y-13)
	b.rect(x, top, 1, y+7-top, ptfGold)
	b.rect(x+1, top+2, 1, y+5-top, ptfGoldD)
	b.rect(x-1, top+2, 3, 1, ptfGold) // the cross bar
	b.set(x, top-1, ptfGoldL)
	return true
}
