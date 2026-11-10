package main

import (
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// Famous guests A-H, touched up so they read closer to the characters they parody: a cowl with
// ears, a chest emblem, a shell, a tie, a visor... Our own small caricatures on top of the
// generic body (not copies of any reference). genChar calls caTorso at the end of torso() and
// caHead at the end of head(); both only draw inside the 24x44 box (pixBuf.set clips) and follow
// the body geometry, so the details move with every walk and dance frame.
// ---------------------------------------------------------------------------

const (
	caNone = iota
	caBatman
	caDeadpool
	caCapAmerica
	caFlash
	caCatwoman
	caAquaman
	caGoku
	caBowser
	caDonkeyKong
	caCreeper
	caDoraemon
	caBobaFett
	caHanSolo
	caBuzz
	caGru
	caHarry
	caSnowWhite
	caBeetlejuice
	caHeMan
	caBender
)

var caKinds = map[string]int{
	"batman": caBatman, "deadpool": caDeadpool, "capitan_america": caCapAmerica, "flash": caFlash,
	"catwoman": caCatwoman, "aquaman": caAquaman, "goku": caGoku, "bowser": caBowser,
	"donkey_kong": caDonkeyKong, "creeper": caCreeper, "doraemon": caDoraemon, "boba_fett": caBobaFett,
	"han_solo": caHanSolo, "buzz": caBuzz, "gru": caGru, "harry_potter": caHarry,
	"blancanieves": caSnowWhite, "beetlejuice": caBeetlejuice, "he_man": caHeMan, "bender": caBender,
}

func caKind(s *CharSpec) int {
	if s == nil || s.Name == "" {
		return caNone
	}
	return caKinds[s.Name]
}

var (
	caBlack  = hx(0x121218)
	caWhite  = hx(0xf4f4f0)
	caYellow = hx(0xf0c830)
	caRed    = hx(0xc82828)
)

// caTorso: emblems, belts, shells and backpacks over the torso rows (x1 exclusive, y1 is the hip row).
func caTorso(b *pixBuf, s *CharSpec, vw, x0, x1, y0, y1, mid int, p Pose) {
	k := caKind(s)
	if k == caNone {
		return
	}
	b.cur = 2
	tw := x1 - x0
	belt := func(c color.RGBA) { b.rect(x0, y1-2, tw, 1, c) }
	front, side, back := vw == vFront, vw == vSide, vw == vBack
	switch k {
	case caBatman:
		belt(caYellow)
		if front {
			b.set(mid, y1-2, shade(caYellow, 0.7))
			// a black bat across the chest
			b.set(mid-1, y0+1, caBlack)
			b.set(mid+1, y0+1, caBlack)
			b.rect(mid-3, y0+2, 7, 1, caBlack)
			b.rect(mid-1, y0+3, 3, 1, caBlack)
			b.set(mid-3, y0+3, caBlack)
			b.set(mid+3, y0+3, caBlack)
		}
	case caDeadpool:
		dk := hx(0x1a1a1e)
		if front || back {
			b.rect(x0, y0+1, 1, y1-y0-2, dk)
			b.rect(x1-1, y0+1, 1, y1-y0-2, dk)
		}
		belt(hx(0x3a3a40))
		if front {
			b.set(mid, y1-2, hx(0xe8c040))
		}
		if back { // two katanas crossed on the back, hilts over the shoulders
			b.line(x0+1, y0-3, x1-2, y1-2, hx(0x2a2a30))
			b.line(x1-2, y0-3, x0+1, y1-2, hx(0x2a2a30))
			b.rect(x0, y0-4, 2, 2, hx(0x5a3a2a))
			b.rect(x1-2, y0-4, 2, 2, hx(0x5a3a2a))
		}
		if side {
			b.line(x0-1, y0-3, x0+1, y1-2, hx(0x2a2a30))
			b.rect(x0-2, y0-4, 2, 2, hx(0x5a3a2a))
		}
	case caCapAmerica:
		if front {
			// white star, red and white stripes on the belly
			b.set(mid, y0+1, caWhite)
			b.rect(mid-1, y0+2, 3, 1, caWhite)
			b.set(mid-1, y0+3, caWhite)
			b.set(mid+1, y0+3, caWhite)
			for y := y0 + 5; y <= y1-2; y++ {
				c := caRed
				if (y-y0)%2 == 0 {
					c = caWhite
				}
				b.rect(x0, y, tw, 1, c)
			}
		}
		if back { // the round shield on his back
			cx, cy := float64(mid)+0.5, float64(y0)+4.5
			b.ellipse(cx, cy, 3.6, 3.6, caRed)
			b.ellipse(cx, cy, 2.6, 2.6, caWhite)
			b.ellipse(cx, cy, 1.7, 1.7, hx(0x2a4ab0))
			b.set(mid, y0+4, caWhite)
		}
		if side {
			b.rect(x0-2, y0+1, 2, 7, caRed)
			b.rect(x0-2, y0+3, 2, 3, caWhite)
			b.set(x0-2, y0+4, hx(0x2a4ab0))
			for y := y0 + 5; y <= y1-2; y++ {
				if (y-y0)%2 == 1 {
					b.rect(x0, y, tw, 1, caRed)
				} else {
					b.rect(x0, y, tw, 1, caWhite)
				}
			}
		}
	case caFlash:
		belt(caYellow)
		if front {
			// white disc with a yellow bolt
			b.rect(mid-1, y0+1, 3, 3, caWhite)
			b.set(mid+1, y0+1, caYellow)
			b.set(mid, y0+2, caYellow)
			b.set(mid-1, y0+3, caYellow)
			b.set(mid, y0+3, caYellow)
		}
	case caCatwoman:
		if front {
			b.rect(mid, y0, 1, y1-y0-1, hx(0x8a8a96)) // the zip
		}
		belt(hx(0x2a2a30))
	case caAquaman:
		belt(caYellow)
		if front {
			b.set(mid, y1-2, hx(0xa86a10))
		}
	case caGoku:
		blue := hx(0x2a50c0)
		belt(blue)
		if front {
			b.set(mid, y0, blue) // the blue undershirt in the V of the gi
			b.set(mid, y0+1, blue)
			b.set(mid-1, y0, blue)
			b.set(mid+1, y0, blue)
			b.set(x1-2, y0+3, caWhite) // the round badge
			b.set(mid+1, y1-1, blue)   // knot of the belt
		}
		if back {
			b.rect(mid-1, y0+2, 3, 3, caWhite)
			b.set(mid, y0+3, caBlack)
		}
	case caBowser:
		cream := hx(0xf0d890)
		if front {
			b.rect(x0+1, y0+1, tw-2, y1-y0-2, cream)
			for y := y0 + 3; y < y1-1; y += 2 {
				b.rect(x0+1, y, tw-2, 1, shade(cream, 0.8))
			}
		}
		// spiky black collar
		b.rect(x0, y0, tw, 1, caBlack)
		for x := x0; x < x1; x += 2 {
			b.set(x, y0-1, caWhite)
		}
		shell := hx(0x2a8a30)
		if back {
			cx, cy := float64(x0)+float64(tw)/2, float64(y0+y1)/2
			b.ellipse(cx, cy, float64(tw)/2+1.4, float64(y1-y0)/2+0.8, cream)
			b.ellipse(cx, cy, float64(tw)/2+0.6, float64(y1-y0)/2, shell)
			for _, o := range [][2]int{{-2, -2}, {2, -2}, {0, 1}, {-2, 3}, {2, 3}} {
				b.set(mid+o[0], (y0+y1)/2+o[1], caWhite)
				b.set(mid+o[0], (y0+y1)/2+o[1]+1, shade(caWhite, 0.7))
			}
		}
		if side {
			b.ellipse(float64(x0)-0.5, float64(y0+y1)/2, 2.8, float64(y1-y0)/2, shell)
			b.rect(x0+1, y0+1, 1, y1-y0-2, cream)
			b.set(x0-3, y0+3, caWhite)
			b.set(x0-3, y0+6, caWhite)
			b.set(x0-2, y0+1, caWhite)
		}
	case caDonkeyKong:
		if front || side {
			x := mid - 1
			if side {
				x = x1 - 2
			}
			tie := hx(0xd02020)
			b.rect(x, y0, 3, 1, tie)
			b.rect(x, y0+1, 3, 4, tie)
			b.rect(x+1, y0+5, 1, 1, tie)
			if front {
				b.set(x, y0+2, caYellow) // D K
				b.set(x+2, y0+2, caYellow)
				b.set(x+2, y0+3, caYellow)
			}
		}
	case caCreeper:
		// blocky body: darker pixels in a grid
		for y := y0; y < y1; y += 2 {
			for x := x0 + (y/2)%2; x < x1; x += 3 {
				b.set(x, y, hx(0x3a8a2a))
			}
		}
	case caDoraemon:
		red := hx(0xd82020)
		b.rect(x0, y0, tw, 1, red) // collar
		if front {
			b.ellipse(float64(mid)+0.5, float64(y0)+5.5, float64(tw)/2-0.6, 4, caWhite)
			b.rect(mid-2, y0+5, 5, 1, hx(0xb0b0b8)) // the pocket
			b.rect(mid-2, y0+5, 1, 2, hx(0xb0b0b8))
			b.rect(mid+2, y0+5, 1, 2, hx(0xb0b0b8))
			b.rect(mid-2, y0+7, 5, 1, hx(0xb0b0b8))
			b.rect(mid, y0+1, 1, 2, caYellow) // the bell
		}
		if side {
			b.set(x1-1, y0+1, caYellow)
			b.rect(x1-2, y0+3, 2, 5, caWhite)
		}
		if back {
			b.ellipse(float64(mid)+0.5, float64(y1)-1, 1.3, 1.3, red) // tail
		}
	case caBobaFett:
		grey := hx(0x9a9aa2)
		if back { // the jetpack
			b.rect(x0, y0, tw, y1-y0-1, grey)
			b.rect(x0+1, y0+1, 1, y1-y0-3, shade(grey, 0.75))
			b.rect(x1-2, y0+1, 1, y1-y0-3, shade(grey, 0.75))
			b.rect(mid-1, y0-2, 3, 3, hx(0x5a7a5a))
			b.set(mid, y0-3, hx(0xc8a040))
			b.rect(x0, y1-2, 2, 1, hx(0x3a3a40))
			b.rect(x1-2, y1-2, 2, 1, hx(0x3a3a40))
		}
		if side {
			b.rect(x0-3, y0, 3, y1-y0-1, grey)
			b.rect(x0-3, y0-2, 1, 2, hx(0x5a7a5a))
			b.set(x0-3, y1-2, hx(0x3a3a40))
		}
		if front {
			b.rect(x0, y0, tw, 1, hx(0xc8a040)) // the yellow collar of the armour
		}
		belt(hx(0x6a5a3a))
	case caHanSolo:
		vest := hx(0x16161c)
		switch vw {
		case vFront:
			b.rect(x0, y0, 2, y1-y0-1, vest)
			b.rect(x1-2, y0, 2, y1-y0-1, vest)
		case vBack:
			b.rect(x0, y0, tw, y1-y0-1, vest)
		default:
			b.rect(x0, y0, tw-1, y1-y0-1, vest)
		}
		belt(hx(0x6a4a2a))
		// the holster on his hip
		hxo := x0 - 1
		if side {
			hxo = x0 + 1
		}
		b.rect(hxo, y1-1, 1, 4, hx(0x4a3020))
		b.set(hxo, y1-2, hx(0x3a3a40))
	case caBuzz:
		purple, green := hx(0x7a4ab0), hx(0x40b040)
		b.rect(x0, y0, tw, 1, purple)
		b.rect(x0, y1-3, tw, 1, green)
		if front {
			b.rect(mid-1, y0+2, 3, 2, hx(0xd8dce8))
			b.set(mid-1, y0+3, caRed)
			b.set(mid, y0+3, green)
			b.set(mid+1, y0+3, hx(0x3a6ae0))
			b.rect(x0, y0+1, 1, 3, purple)
			b.rect(x1-1, y0+1, 1, 3, purple)
		}
		if back { // folded wings and backpack
			b.rect(x0-2, y0+2, tw+4, 2, caWhite)
			b.set(x0-2, y0+3, caRed)
			b.set(x1+1, y0+3, caRed)
			b.rect(x0+1, y0+1, tw-2, 5, hx(0xe8e8f0))
			b.rect(x0+1, y0+4, tw-2, 1, purple)
		}
		if side {
			b.rect(x0-2, y0+1, 2, 5, hx(0xe8e8f0))
			b.set(x0-2, y0+4, purple)
		}
	case caGru:
		// the striped scarf
		for x := x0; x < x1; x++ {
			c := hx(0x3a3a44)
			if (x-x0)%2 == 1 {
				c = hx(0x9a9aa6)
			}
			b.set(x, y0, c)
			b.set(x, y0+1, c)
		}
		if front {
			b.rect(x0+1, y0+2, 1, 3, hx(0x9a9aa6))
			b.set(x0+1, y0+3, hx(0x3a3a44))
		}
	case caHarry:
		red, gold := hx(0xa82020), hx(0xe8b830)
		for x := x0; x < x1; x++ {
			c := red
			if (x-x0)%2 == 1 {
				c = gold
			}
			b.set(x, y0, c)
		}
		if front {
			for i := 0; i < 4; i++ {
				c := red
				if i%2 == 1 {
					c = gold
				}
				b.set(x0+1, y0+1+i, c)
			}
			b.set(x1-2, y0+3, red) // the house crest
		}
	case caSnowWhite:
		blue, red := hx(0x2a3ea0), hx(0xc82828)
		b.rect(x0, y0, tw, y1-y0, blue) // blue bodice over the yellow skirt
		if front {
			b.set(mid, y0, hx(0xf0d0b0))
			b.set(mid, y0+2, red)
			b.set(mid, y0+4, red)
		}
		// puffed sleeves with red slashes
		b.rect(x0-1, y0, 1, 2, blue)
		b.rect(x1, y0, 1, 2, blue)
		b.set(x0-1, y0, red)
		b.set(x1, y0, red)
		if !side { // the stand-up white collar
			b.set(x0, y0-1, caWhite)
			b.set(x1-1, y0-1, caWhite)
		}
	case caBeetlejuice:
		for y := y0; y <= y1-2; y++ {
			for x := x0; x < x1; x++ {
				if (x-x0)%2 == 1 {
					b.set(x, y, hx(0x16161a))
				} else {
					b.set(x, y, hx(0xe8e8e0))
				}
			}
		}
		if front {
			b.rect(mid, y0+1, 1, 4, hx(0x2a8a2a)) // green tie
		}
	case caHeMan:
		strap := hx(0x5a5a64)
		if front || back {
			b.line(x0, y0, x1-1, y0+4, strap)
			b.line(x1-1, y0, x0, y0+4, strap)
		}
		if front {
			b.rect(mid-1, y0+1, 3, 3, hx(0xa8a8b4))
			b.rect(mid, y0+1, 1, 3, caRed)
			b.rect(mid-1, y0+2, 3, 1, caRed)
		}
		if side {
			b.line(x0, y0, x1-1, y0+4, strap)
		}
		belt(hx(0x6a4a2a))
		if front {
			b.set(mid, y1-2, caYellow)
		}
	case caBender:
		if front {
			b.rect(x0+1, y0+3, tw-2, 4, hx(0xb4bcc8)) // the chest door
			b.rect(x0+1, y0+3, tw-2, 1, hx(0x7a828e))
		}
	}
}

// caHead: cowls, masks, faces and hair details at the end of head().
func caHead(b *pixBuf, s *CharSpec, vw int, p Pose, hxp, hyp, rx, ry float64, ey int) {
	k := caKind(s)
	if k == caNone {
		return
	}
	b.cur = 1
	ihx := int(math.Floor(hxp))
	top := int(math.Floor(hyp - ry))
	ex := int(math.Floor(hxp + rx*0.45)) // eye column in profile
	nose := int(math.Floor(hxp + rx))    // front edge in profile
	front, side, back := vw == vFront, vw == vSide, vw == vBack
	skin := s.Skin
	mouth := shade(skin, 0.55)
	// lowerFace shows the jaw under a cowl
	lowerFace := func() {
		switch vw {
		case vFront:
			b.rect(ihx-1, ey+1, 3, 2, skin)
			b.set(ihx, ey+2, mouth)
		case vSide:
			b.rect(ihx+1, ey+1, nose-ihx, 2, skin)
			b.set(nose-1, ey+2, mouth)
		}
	}
	walking := p.Anim == AnimWalk || p.Anim == AnimRun
	switch k {
	case caBatman:
		m := s.Mask
		if side {
			b.rect(ihx-1, top-2, 1, 3, m)
		} else {
			b.rect(ihx-2, top-2, 1, 3, m)
			b.rect(ihx+2, top-2, 1, 3, m)
		}
		lowerFace()
	case caDeadpool:
		if front {
			b.rect(ihx-2, ey-1, 2, 2, caBlack)
			b.rect(ihx+1, ey-1, 2, 2, caBlack)
			b.set(ihx-1, ey, caWhite)
			b.set(ihx+1, ey, caWhite)
		}
		if side {
			b.rect(ex-1, ey-1, 2, 2, caBlack)
			b.set(ex, ey, caWhite)
		}
	case caCapAmerica:
		if front {
			b.set(ihx, top+1, caWhite) // the A on the forehead
			b.set(ihx-1, top+2, caWhite)
			b.set(ihx+1, top+2, caWhite)
			b.rect(ihx-3, ey-2, 1, 2, caWhite) // little wings
			b.rect(ihx+3, ey-2, 1, 2, caWhite)
		}
		if side {
			b.rect(ihx-1, ey-2, 2, 1, caWhite)
			b.set(ihx-2, ey-1, caWhite)
		}
		lowerFace()
	case caFlash:
		if front {
			b.rect(ihx-3, ey-2, 1, 2, caYellow)
			b.set(ihx-4, ey-3, caYellow)
			b.rect(ihx+3, ey-2, 1, 2, caYellow)
			b.set(ihx+4, ey-3, caYellow)
		}
		if side {
			b.rect(ihx-1, ey-1, 2, 1, caYellow)
			b.set(ihx-2, ey-2, caYellow)
		}
		if back {
			b.set(ihx-3, ey-1, caYellow)
			b.set(ihx+3, ey-1, caYellow)
		}
		lowerFace()
	case caCatwoman:
		m := s.Mask
		if side {
			b.set(ihx-1, top-1, m)
		} else {
			b.set(ihx-2, top-1, m)
			b.set(ihx+2, top-1, m)
			b.set(ihx-2, top-2, m)
			b.set(ihx+2, top-2, m)
		}
		lowerFace()
		if front {
			b.set(ihx, ey+2, hx(0xc8203a)) // red lips
			b.rect(ihx-2, ey-1, 2, 1, hx(0x6a8ab0))
			b.rect(ihx+1, ey-1, 2, 1, hx(0x6a8ab0))
		}
		if side {
			b.set(nose-1, ey+2, hx(0xc8203a))
		}
	case caGoku:
		h := s.Hair
		lift := 0
		if walking && p.F%2 == 1 {
			lift = 1
		}
		switch vw {
		case vFront, vBack:
			b.rect(ihx-4, top, 1, 2, h)
			b.set(ihx-5, top-1+lift, h)
			b.rect(ihx+4, top, 1, 2, h)
			b.set(ihx+5, top-1+lift, h)
			b.rect(ihx-2, top-4, 1, 2, h)
			b.rect(ihx+1, top-5+lift, 1, 2, h)
			if front {
				b.set(ihx-1, top+2, h) // fringe
				b.set(ihx+1, top+2, h)
			}
		default:
			b.rect(ihx-4, top, 2, 2, h)
			b.set(ihx-5, top-1+lift, h)
			b.rect(ihx-3, top-3, 1, 2, h)
			b.rect(ihx, top-4+lift, 1, 2, h)
			b.set(nose-1, top+2, h)
		}
	case caBowser:
		cream := hx(0xf0d890)
		if front {
			b.rect(ihx-1, ey+1, 3, 2, cream)
			b.set(ihx-1, ey+2, caWhite)
			b.set(ihx+1, ey+2, caWhite)
			b.set(ihx-2, ey-1, hx(0xd84a20)) // angry red brows
			b.set(ihx+2, ey-1, hx(0xd84a20))
		}
		if side {
			b.rect(nose-1, ey+1, 2, 2, cream)
			b.set(nose, ey+2, caWhite)
		}
	case caDonkeyKong:
		tan := hx(0xd8a878)
		if front {
			b.rect(ihx-2, ey+2, 5, 2, tan)
			b.set(ihx-1, ey+2, shade(tan, 0.55))
			b.set(ihx+1, ey+2, shade(tan, 0.55))
			b.rect(ihx-3, ey-1, 7, 1, shade(skin, 0.7)) // heavy brow
			b.set(ihx, top-1, skin)                     // the tuft
			b.set(ihx+1, top-2, skin)
		}
		if side {
			b.rect(nose-1, ey+1, 3, 3, tan)
			b.set(nose+1, ey+2, shade(tan, 0.55))
			b.set(ihx, top-1, skin)
		}
	case caCreeper:
		g1, g2 := hx(0x58b040), hx(0x7ac860)
		x0, w := ihx-3, 7
		if side {
			x0, w = ihx-3, 6
		}
		for y := top; y < top+8; y++ {
			for x := x0; x < x0+w; x++ {
				c := g1
				if (x*7+y*3)%5 == 0 {
					c = g2
				} else if (x+y*5)%7 == 0 {
					c = hx(0x3a8a2a)
				}
				b.set(x, y, c)
			}
		}
		face := hx(0x10140c)
		if front {
			b.rect(ihx-3, top+2, 2, 2, face)
			b.rect(ihx+2, top+2, 2, 2, face)
			b.rect(ihx-1, top+4, 3, 2, face)
			b.rect(ihx-2, top+5, 1, 2, face)
			b.rect(ihx+2, top+5, 1, 2, face)
		}
		if side {
			b.rect(x0+w-2, top+2, 2, 2, face)
			b.rect(x0+w-1, top+4, 1, 3, face)
		}
	case caDoraemon:
		red := hx(0xd82020)
		if front {
			b.ellipse(hxp, hyp+1.2, rx-1, ry-1.5, caWhite)
			b.rect(ihx-2, ey-3, 2, 3, caWhite)
			b.rect(ihx+1, ey-3, 2, 3, caWhite)
			b.set(ihx-1, ey-2, caBlack)
			b.set(ihx+1, ey-2, caBlack)
			b.rect(ihx, ey, 1, 1, red)
			b.set(ihx, ey+1, caBlack)
			b.set(ihx, ey+2, caBlack)
			b.rect(ihx-2, ey+3, 5, 1, caBlack)
			b.set(ihx-1, ey+4, red)
			b.set(ihx, ey+4, red)
			b.set(ihx+1, ey+4, red)
			for _, d := range []int{-4, -3, 3, 4} {
				b.set(ihx+d, ey+1, caBlack)
				b.set(ihx+d, ey+3-absI(d)/4, caBlack)
			}
		}
		if side {
			b.ellipse(hxp+1.4, hyp+1.2, rx-1.8, ry-1.5, caWhite)
			b.rect(ex, ey-3, 2, 3, caWhite)
			b.set(ex+1, ey-2, caBlack)
			b.rect(nose, ey, 2, 2, red)
			b.rect(ex, ey+3, 3, 1, caBlack)
			b.set(nose-1, ey+1, caBlack)
		}
	case caBobaFett:
		m := s.Mask
		visor := hx(0x0e0e12)
		if front {
			b.set(ihx-1, ey, m)
			b.set(ihx+1, ey, m)
			b.set(ihx, ey+2, m)
			b.rect(ihx-2, ey-1, 5, 1, visor)
			b.rect(ihx, ey, 1, 3, visor)
			b.rect(ihx-2, top, 5, 1, hx(0xa83a2a))   // the dented red stripe
			b.rect(ihx+3, top-2, 1, 4, hx(0x8a8a90)) // rangefinder
		}
		if side {
			b.set(ex, ey, m)
			b.rect(ex, ey-1, nose-ex+1, 1, visor)
			b.rect(nose-1, ey, 1, 3, visor)
			b.rect(ihx-1, top-2, 1, 3, hx(0x8a8a90))
		}
		if back {
			b.rect(ihx-2, top, 5, 1, hx(0xa83a2a))
			b.rect(ihx-3, top-2, 1, 4, hx(0x8a8a90))
		}
	case caBuzz:
		purple, dome := hx(0x7a4ab0), hx(0xc8e4f8)
		if front {
			b.rect(ihx-2, top+1, 1, ey-top+2, purple)
			b.rect(ihx+2, top+1, 1, ey-top+2, purple)
			b.set(ihx, ey+2, skin) // the big chin
			b.set(ihx, ey+3, shade(skin, 0.85))
			b.rect(ihx-3, top+1, 1, ey-top+1, dome)
			b.rect(ihx+3, top+1, 1, ey-top+1, dome)
			b.rect(ihx-2, top-1, 5, 1, dome)
		}
		if side {
			b.rect(ihx-2, top+1, 2, ey-top+2, purple)
			b.rect(nose+1, top+1, 1, ey-top+1, dome)
			b.rect(ihx-1, top-1, nose-ihx+2, 1, dome)
		}
		if back {
			b.rect(ihx-3, top+1, 1, ey-top+1, dome)
			b.rect(ihx+3, top+1, 1, ey-top+1, dome)
		}
	case caGru:
		sk := shade(skin, 0.85)
		if front {
			b.rect(ihx, ey+2, 1, 3, sk) // the long pointed nose
			b.set(ihx+1, ey+2, sk)
		}
		if side {
			b.rect(nose, ey+1, 3, 2, sk)
			b.set(nose+3, ey+2, sk)
		}
	case caHarry:
		if front {
			b.set(ihx, ey-1, hx(0xb03030)) // the lightning scar
			b.set(ihx-1, ey-2, hx(0xb03030))
			// round frames around the eyes
			fr := hx(0x2a2a30)
			b.set(ihx-2, ey, fr)
			b.set(ihx+2, ey, fr)
			b.set(ihx, ey, fr)
		}
		if side {
			b.rect(ex-1, ey, 1, 1, hx(0x2a2a30))
			b.set(ex+1, ey, hx(0x2a2a30))
		}
	case caSnowWhite:
		red := hx(0xd82828)
		if side {
			b.rect(ihx-1, top-1, 3, 1, red)
			b.set(ihx-1, top-2, red)
		} else {
			b.rect(ihx-2, top-1, 2, 2, red)
			b.rect(ihx+1, top-1, 2, 2, red)
			b.set(ihx, top, shade(red, 0.7))
		}
	case caBeetlejuice:
		bags := hx(0x5a3a5a)
		if front {
			b.set(ihx-1, ey+1, bags)
			b.set(ihx+1, ey+1, bags)
			b.set(ihx-2, ey, bags)
			b.set(ihx+2, ey, bags)
			b.set(ihx+1, ey+2, hx(0x7a9a5a)) // a bit of mould
		}
		if side {
			b.set(ex, ey+1, bags)
		}
	case caBender:
		visor, eye := hx(0x30343c), hx(0xf4f0c8)
		if front {
			b.rect(ihx-3, ey-1, 6, 3, visor)
			b.rect(ihx-2, ey, 2, 1, eye)
			b.rect(ihx+1, ey, 2, 1, eye)
			b.set(ihx-1, ey, caBlack)
			b.set(ihx+1, ey, caBlack)
			b.rect(ihx-2, ey+3, 5, 2, eye) // the grille mouth
			b.set(ihx-1, ey+3, hx(0x6a6a60))
			b.set(ihx+1, ey+3, hx(0x6a6a60))
			b.rect(ihx-2, ey+4, 5, 1, hx(0x6a6a60))
		}
		if side {
			b.rect(ex-1, ey-1, nose-ex+2, 3, visor)
			b.rect(ex, ey, 2, 1, eye)
			b.rect(ex-1, ey+3, nose-ex+1, 2, eye)
		}
	}
}
