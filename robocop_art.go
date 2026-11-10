package main

import (
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// RoboCop: procedural pixel art of the steel-blue cyborg cop (front and back views, side view
// simplified). It is only an overlay on the usual CharSpec drawing (genChar calls these hooks
// when the character is "robocop"): helmet with black visor and human chin, wide pauldrons,
// chest plate with a badge, segmented black abdomen, black belt, forearm gauntlets with
// black gloves, black knee pads, shin plates and armored boots.
// Visual only: the visor flickers red and a flash lights the hip when he fires (the stab
// pose, drawn with his pistol), and the chest light pulses with the idle bounce.
// ---------------------------------------------------------------------------

func isRobocop(s *CharSpec) bool { return s.Name == "robocop" }

var (
	rcHi    = hx(0xa4c6f4)
	rcLight = hx(0x7aa0e0)
	rcMid   = hx(0x4e74b8)
	rcBase  = hx(0x3c5c9c)
	rcDark  = hx(0x2a4278)
	rcDeep  = hx(0x1c2c54)
	rcBlack = hx(0x12141c)
	rcSeg   = hx(0x262b3a)
	rcSteel = hx(0xc4cedc)
	rcSleev = hx(0x9a7a74)
)

func rcShade(c color.RGBA, dark bool) color.RGBA {
	if dark {
		return shade(c, 0.78)
	}
	return c
}

// roboFiring: the stab pose is his shot (he holds the pistol while the arm points forward).
func roboFiring(p Pose) bool { return p.Hold == HoldKnife && p.RA.Z >= 6 }

// roboAlert: the arm raised in an attack-like pose makes the visor flicker.
func roboAlert(p Pose) bool { return p.Hold == HoldKnife }

// roboHead paints helmet, visor and the human lower face over the usual head and hat.
// ihx: center column, top: first head row, ey: eye row.
func roboHead(b *pixBuf, s *CharSpec, vw int, ihx, top int, p Pose) {
	b.cur = 1
	skin := s.Skin
	skinD := shade(skin, 0.8)
	slit, vis := hx(0xa8bcd8), rcBlack
	if roboAlert(p) {
		slit, vis = hx(0xff6450), hx(0x2c1216)
	}
	row := func(y, x0, x1 int, c color.RGBA) { // x0..x1 inclusive, relative to ihx
		b.rect(ihx+x0, y, x1-x0+1, 1, c)
	}
	siren := hx(0xe03030) // a tiny party siren on the helmet
	if p.Bob > 0.5 {
		siren = hx(0x3a70f0)
	}
	b.set(ihx, top-1, siren)
	switch vw {
	case vFront:
		row(top, -1, 1, rcLight)
		row(top+1, -2, 2, rcMid)
		b.set(ihx-1, top+1, rcHi)
		b.set(ihx-2, top+1, rcLight)
		row(top+2, -3, 3, rcBase)
		row(top+3, -3, 3, rcBase)
		b.set(ihx-3, top+2, rcLight)
		b.set(ihx+3, top+3, rcDark)
		row(top+2, -2, 2, vis)
		row(top+3, -2, 2, vis)
		row(top+2, -1, 1, slit) // the light slit of the visor
		b.set(ihx-2, top+3, rcDeep)
		row(top+4, -2, 2, rcDark) // cheek plates
		row(top+4, -1, 1, skin)
		b.set(ihx, top+4, skinD) // mouth line
		row(top+5, -1, 1, skinD) // chin
		b.set(ihx, top+5, skin)
		b.set(ihx-2, top+5, rcBlack)
		b.set(ihx+2, top+5, rcBlack)
	case vBack:
		row(top, -1, 1, rcLight)
		row(top+1, -2, 2, rcMid)
		b.set(ihx-1, top+1, rcHi)
		row(top+2, -3, 3, rcBase)
		row(top+3, -3, 3, rcMid)
		b.set(ihx, top+2, rcDark) // seam of the helmet
		b.set(ihx, top+3, rcDark)
		b.set(ihx-3, top+2, rcLight)
		b.set(ihx+3, top+3, rcDark)
		row(top+4, -2, 2, rcDark)
		row(top+4, -1, 1, hx(0x2a2220)) // hair at the nape
		row(top+5, -1, 1, rcBlack)
	default: // side view, faces +x
		row(top, -2, 1, rcLight)
		row(top+1, -3, 2, rcMid)
		b.set(ihx-1, top+1, rcHi)
		row(top+2, -3, 3, rcBase)
		row(top+3, -3, 3, rcBase)
		row(top+2, 0, 3, vis)
		row(top+3, 0, 3, vis)
		row(top+2, 1, 3, slit)
		b.set(ihx-3, top+2, rcLight)
		row(top+4, -3, -1, rcDark)
		row(top+4, 0, 2, skin)
		row(top+5, 0, 2, skinD)
		b.set(ihx-1, top+5, rcBlack)
	}
}

// roboTorso: chest, abdomen and belt over the plain torso rectangle [x0,x1) x [y0,y1].
func roboTorso(b *pixBuf, s *CharSpec, vw, x0, x1, y0, y1 int, p Pose) {
	b.cur = 2
	w := x1 - x0
	if w < 4 {
		return
	}
	mid := x0 + w/2
	bright := p.Bob > 0.5 // the idle bounce is the pulse of the light
	for y := y0; y <= y1; y++ {
		r := y - y0
		for x := x0; x < x1; x++ {
			var c color.RGBA
			edgeL, edgeR := x == x0, x == x1-1
			switch {
			case r == 0:
				c = rcHi
				if edgeL || edgeR {
					c = rcMid
				}
			case r <= 3: // chest plate
				c = rcMid
				if edgeL {
					c = rcLight
				} else if edgeR {
					c = rcDark
				} else if x == x1-2 {
					c = rcBase
				}
				if vw != vSide && x == mid {
					c = rcDark // seam between the pectorals
				}
				if vw == vBack && (x+r)%4 == 0 {
					c = rcDark // cracks of the back plate
				}
			case r == 4: // lower edge of the plate
				c = rcDark
				if edgeL {
					c = rcBase
				}
			case r <= 8: // segmented abdomen
				c = rcSeg
				if r%2 == 0 {
					c = rcBlack
				}
				if r == 5 {
					c = rcSeg
				}
				if vw != vSide && (edgeL || edgeR) {
					c = rcDark
				}
			case r == 9: // belt
				c = rcBlack
				if x == mid && vw != vBack {
					c = hx(0x8a96aa)
				}
			default: // hip plates
				c = rcDark
				if edgeL {
					c = rcBase
				}
				if x == mid && vw != vSide {
					c = rcBlack
				}
			}
			b.set(x, y, c)
		}
	}
	if vw == vFront {
		b.set(x1-3, y0+1, rcSteel) // badge
		b.set(x1-2, y0+1, hx(0x8a96aa))
		// light of the chest: soft pulse
		lc, glow := hx(0x5a9cff), rcLight
		if bright {
			lc, glow = hx(0xd8f4ff), hx(0x8ac4ff)
		}
		b.set(x0+2, y0+2, lc)
		// our own touch: a flower lei over the plates
		for i, x := 0, x0+1; x < x1-1; x, i = x+1, i+1 {
			b.set(x, y0+1, hx([]uint32{0xf060a0, 0xf8e050, 0xf08a30}[i%3]))
		}
		b.set(x0+2, y0+3, glow)
	}
}

// roboPauldron + forearm: called after the plain arm, from shoulder (sx,sy) through elbow (mx,my)
// to the hand (ex,ey).
func roboArm(b *pixBuf, vw int, sx, sy, mx, my, ex, ey float64, armW int, dark bool, id uint8) {
	b.cur = id
	fl := func(f float64) int { return int(math.Floor(f)) }
	// under-suit sleeve (a bit of skin shows between shoulder and gauntlet)
	thickLine(b, sx, sy+1, mx, my, armW, rcShade(rcSleev, dark))
	// gauntlet from the elbow
	thickLine(b, mx, my+1, ex, ey, armW, rcShade(rcBase, dark))
	thickLine(b, mx, my+1, ex, ey-1, 1, rcShade(rcLight, dark))
	b.rect(fl(mx), fl(my), armW, 1, rcShade(rcBlack, dark)) // elbow guard
	// pauldron
	px, py := fl(sx)-1, fl(sy)-1
	pw := armW + 2
	b.rect(px, py, pw, 1, rcShade(rcHi, dark))
	b.rect(px, py+1, pw, 1, rcShade(rcMid, dark))
	b.rect(px, py+2, pw, 1, rcShade(rcDark, dark))
	b.set(px, py, color.RGBA{})
	b.set(px+pw-1, py, color.RGBA{})
	b.set(px+1, py+1, rcShade(rcLight, dark))
	// glove
	b.rect(fl(ex), fl(ey), armW, 2, rcShade(rcBlack, dark))
	b.set(fl(ex), fl(ey), rcShade(hx(0x3a4258), dark))
}

// roboLeg: thigh plate light, black knee pad, shin plate with a highlight, armored boot.
// sign: -1 left leg, +1 right leg.
func roboLeg(b *pixBuf, vw int, hxp, hipY, mx, my, ex, ey float64, legW int, dark bool, sign float64) {
	fl := func(f float64) int { return int(math.Floor(f)) }
	thickLine(b, hxp, hipY+1, mx, my, legW, rcShade(rcBase, dark))
	b.line(fl(hxp), fl(hipY)+1, fl(mx), fl(my)-1, rcShade(rcLight, dark))
	kx := fl(mx)
	if vw == vSide {
		kx--
	}
	b.rect(kx, fl(my), legW, 2, rcShade(rcBlack, dark))
	b.set(kx, fl(my), rcShade(hx(0x3a4258), dark))
	thickLine(b, mx, my+2, ex, ey, legW, rcShade(rcMid, dark))
	b.line(fl(mx), fl(my)+2, fl(ex), fl(ey), rcShade(rcHi, dark))
	// boot
	sx, sy := fl(ex), fl(ey)
	w, x := legW, sx
	if vw == vSide {
		w, x = legW+2, sx-1
		b.rect(x, sy, w, 1, rcShade(rcLight, dark))
		b.rect(x, sy+1, w, 1, rcShade(rcDeep, dark))
		b.set(x+w-1, sy, rcShade(rcHi, dark))
		return
	}
	w++
	if sign < 0 {
		x--
	}
	b.rect(x, sy, w, 1, rcShade(rcLight, dark))
	b.rect(x, sy+1, w, 1, rcShade(rcDeep, dark))
	b.set(x+w/2, sy, rcShade(rcHi, dark))
}

// roboFlash: the shot, a small star of light at the hip while he fires.
func roboFlash(b *pixBuf, vw, x1, y1 int) {
	if vw == vBack {
		return
	}
	b.cur = 4
	cx, cy := x1+1, y1-1
	b.set(cx, cy, hx(0xfffbe0))
	b.set(cx+1, cy, hx(0xffd860))
	b.set(cx-1, cy, hx(0xffb030))
	b.set(cx, cy-1, hx(0xffd860))
	b.set(cx, cy+1, hx(0xff9020))
	if vw == vFront {
		b.set(cx+2, cy-1, hx(0xffb030))
	} else {
		b.set(cx+2, cy, hx(0xffb030))
	}
}
