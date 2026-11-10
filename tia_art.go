package main

import (
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// The two secret agents of the Spanish comic (slugs "mortadelo" and "filemon") have their own
// procedural drawing (genTia), like Crash or Johnny 5: the generic body cannot give them the
// very different builds of the pair. Our own pixel-art parody:
//   - Mortadelo: very tall and thin, a long bald egg of a head with a dark tuft over each ear
//     and two hairs on top, round glasses, a big drooping nose, a long neck, a black tailcoat
//     with long tails, white shirt and a red bow tie, black trousers and big black shoes.
//   - Filemon: short and stout, a big round head, bald with two curly hairs on top and ginger
//     tufts round the sides, a round nose, white shirt with a dark waistcoat over the belly,
//     maroon trousers with white stripes and brown shoes.
// Mortadelo's head sticks out well above an ordinary guest (like the other tall ones) and
// Filemon is shorter and wider; only the drawing changes: hitbox, footprint and collisions are
// the ones of every guest. Animation is the usual one (walk, run, dances...) read from the pose.
// Standing rows: Mortadelo hairs 1, skull 2..11, neck 12..13, coat 14..29, legs 27..39, shoes
// 40..42; Filemon hairs 14..15, skull 16..26, torso 25..33, legs 35..39, shoes 40..42.
// ---------------------------------------------------------------------------

const (
	tiaNone = iota
	tiaMortadelo
	tiaFilemon
)

func tiaKind(s *CharSpec) int {
	switch s.Name {
	case "mortadelo":
		return tiaMortadelo
	case "filemon":
		return tiaFilemon
	}
	return tiaNone
}

func isTia(s *CharSpec) bool { return tiaKind(s) != tiaNone }

var (
	tiSkin    = hx(0xf6c9a2)
	tiSkinL   = hx(0xffe2c8)
	tiSkinD   = hx(0xd69a72)
	tiNoseD   = hx(0xb8704e)
	tiCoat    = hx(0x24242c)
	tiCoatL   = hx(0x40404c)
	tiCoatD   = hx(0x121216)
	tiWhite   = hx(0xf6f6f2)
	tiWhiteD  = hx(0xc8ccd6)
	tiRed     = hx(0xd42a2a)
	tiRedD    = hx(0x8e1a1c)
	tiHair    = hx(0x16141a)
	tiGinger  = hx(0xa8441e)
	tiGingerD = hx(0x6e2a12)
	tiFrame   = hx(0x101014)
	tiLens    = hx(0xeef4f8)
	tiMouth   = hx(0x6a1a14)
	tiLip     = hx(0x4a2418)
	tiButton  = hx(0x6a6a78)
	tiVest    = hx(0x3e3648)
	tiVestL   = hx(0x5a506a)
	tiVestD   = hx(0x28222e)
	tiTrou    = hx(0x9c2030)
	tiTrouD   = hx(0x6a141e)
	tiStripe  = hx(0xf0dede)
	tiShoeB   = hx(0x6a3a18)
	tiShoeBL  = hx(0x94582a)
	tiShoeBD  = hx(0x3e200a)
	tiShoeK   = hx(0x141418)
	tiShoeKL  = hx(0x3c3c48)
)

// tiaArmIK: elbow and hand of an arm hanging from (sx, sy). k scales the pose targets (1 = an
// ordinary guest), seg is the length of each bone.
func tiaArmIK(vw int, sx, sy float64, l Limb, sign, k, seg float64) (mx, my, ex, ey float64) {
	var tx, ty float64
	if vw == vSide {
		tx, ty = l.Z*k, l.Y*k
	} else {
		tx, ty = sign*(l.X-0.7)*0.8*k, (l.Y+l.Z*0.1)*k
	}
	bx, by := sign*0.25, 0.9
	if vw == vSide {
		bx, by = -1, 0
	}
	mx, my, ex, ey = ik2(sx, sy, sx+tx, sy+ty, seg, bx, by)
	mx, my = clampF(mx, 2, 21), clampF(my, 2, 40)
	hi := 19.0
	if vw == vSide {
		hi = 16 // a knife held forward (6 px) must stay inside the box
	}
	ex, ey = clampF(ex, 3, hi), clampF(ey, 2, 40)
	return
}

// tiaLegIK: knee and ankle of a leg from the hip (hx0, hy0) to a foot whose top row is footY
// when it rests on the floor; top is the row where the shoe starts.
func tiaLegIK(vw int, p Pose, hx0, hy0 float64, l Limb, sign, k, seg float64, footY int) (mx, my, ex, ey float64, top int) {
	lift := 0
	if !p.Sit {
		lift = clampI(int(math.Floor((10-l.Y)*k+0.5)), 0, 5)
	}
	top = footY - lift
	fx := hx0 + sign*clampF((l.X-1.5)*0.8, -2, 2)
	if vw == vSide {
		fx = hx0 + l.Z*0.7*k
		if p.Sit {
			fx, top = hx0+5, int(hy0)+4
		}
	}
	bx := sign
	if vw == vSide {
		bx = 1
	}
	mx, my, ex, ey = ik2(hx0, hy0, clampF(fx, 2, 20), float64(top)-0.5, seg, bx, 0)
	return
}

// tiaHeld draws the held object (or the weapon) in the hand at (x, y).
func tiaHeld(b *pixBuf, s *CharSpec, p Pose, vw, x, y int, id uint8) {
	b.cur = id
	kind := p.Hold
	if kind == HoldNone {
		kind = s.Carry
	}
	if s.Weapon != 0 && (kind == HoldKnife || kind == HoldNone) {
		kind = s.Weapon
	}
	if kind != HoldNone {
		drawHeld(b, kind, s.CarryCol, vw, float64(x), float64(y))
	}
}

func genTia(s *CharSpec, vw int, p Pose) *pixBuf {
	if tiaKind(s) == tiaFilemon {
		return genFilemon(s, vw, p)
	}
	return genMortadelo(s, vw, p)
}

func tiaDk(c color.RGBA, dark bool) color.RGBA {
	if dark {
		return shade(c, 0.76)
	}
	return c
}

func tiaRnd(f float64) int { return int(math.Floor(f + 0.5)) }

// ---- Mortadelo ------------------------------------------------------------------------------

func genMortadelo(s *CharSpec, vw int, p Pose) *pixBuf {
	b := newPix(charW, charH)
	if trackParts {
		b.pid = make([]uint8, charW*charH)
	}
	bob := clampI(tiaRnd(p.Bob), 0, 3) // the hairs stand on row 1: he never rises higher
	if p.Sit {
		bob = maxI(bob, 3)
	}
	ox := clampI(tiaRnd(p.Sway*0.5), -2, 2)
	hd := ox
	if vw == vSide {
		ox = clampI(tiaRnd(p.Lean*0.5), -1, 2)
		hd = ox + clampI(tiaRnd(p.Lean*0.3+p.HeadZ*0.5), -1, 1)
	} else if p.HeadZ != 0 {
		hd += clampI(tiaRnd(p.HeadZ*0.5), -1, 1)
	}
	hipY := 26 + bob
	ty0 := hipY - 12 // coat top (shoulders)
	hy := ty0 - 12   // skull top
	open := p.Open || p.Anim == AnimLaugh || p.Anim == AnimCheer
	L, R := 11+hd, 12+hd // the two middle columns of the head

	leg := func(hx0 float64, l Limb, sign float64, id uint8, dark bool) {
		b.cur = id
		mx, my, ex, ey, top := tiaLegIK(vw, p, hx0, float64(hipY+1), l, sign, 0.9, 6.4, 40)
		thickLine(b, hx0, float64(hipY+1), mx, my, 2, tiaDk(tiCoat, dark))
		thickLine(b, mx, my, ex, ey, 2, tiaDk(tiCoat, dark))
		thickLine(b, hx0, float64(hipY+2), mx, my, 1, tiaDk(tiCoatL, dark)) // crease
		ix := int(math.Floor(ex))
		if vw == vSide { // a long shoe pointing forward
			b.rect(ix, top, 3, 1, tiaDk(tiShoeK, dark))
			b.rect(ix-1, top+1, 7, 2, tiaDk(tiShoeK, dark))
			b.rect(ix+1, top+1, 4, 1, tiaDk(tiShoeKL, dark))
			b.set(ix+5, top+2, tiaDk(tiShoeKL, dark))
			return
		}
		x0 := ix - 1
		if sign > 0 {
			x0 = ix
		}
		b.rect(x0, top, 3, 1, tiaDk(tiShoeK, dark))
		b.rect(x0-1+boolI(sign > 0), top+1, 4, 2, tiaDk(tiShoeK, dark))
		if vw == vFront {
			b.rect(x0, top+1, 2, 1, tiaDk(tiShoeKL, dark)) // toe shine
		}
	}

	coat := func() {
		b.cur = 2
		if vw == vSide {
			c := 11 + ox
			for r := 0; r <= hipY-ty0; r++ {
				x0, x1 := c-2, c+3
				if r == 0 {
					x0, x1 = c-1, c+2
				}
				for x := x0; x <= x1; x++ {
					col := tiCoat
					if x == x0 {
						col = tiCoatD
					}
					if r == 0 || x == x1-1 {
						col = tiCoatL
					}
					b.set(x, ty0+r, col)
				}
				if r >= 1 && r <= 5 { // shirt front
					b.set(x1, ty0+r, tiWhite)
				}
			}
			b.set(c+4, ty0+1, tiRed) // bow tie sticking out
			b.set(c+4, ty0+2, tiRedD)
			b.set(c+3, ty0+1, tiRedD)
			b.set(c+2, ty0, tiWhite) // collar
			// the long tails hang behind
			for r := 1; r <= 4; r++ {
				b.rect(c-2-r/3, hipY+r, 3, 1, tiCoat)
				b.set(c-2-r/3, hipY+r, tiCoatD)
			}
			return
		}
		for r := 0; r <= hipY-ty0; r++ {
			x0, x1 := 8, 15
			if r >= 9 {
				x0, x1 = 9, 14
			}
			for x := x0; x <= x1; x++ {
				col := tiCoat
				switch {
				case r == 0:
					col = tiCoatL
				case x == x0:
					col = tiCoatL
				case x == x1:
					col = tiCoatD
				}
				b.set(x+ox, ty0+r, col)
			}
		}
		if vw == vFront {
			b.rect(10+ox, ty0, 4, 1, tiWhite) // collar
			b.set(10+ox, ty0+1, tiRed)        // bow tie
			b.set(11+ox, ty0+1, tiRedD)
			b.set(12+ox, ty0+1, tiRedD)
			b.set(13+ox, ty0+1, tiRed)
			b.rect(11+ox, ty0+2, 2, 4, tiWhite) // shirt down the V
			b.set(12+ox, ty0+5, tiWhiteD)
			b.set(10+ox, ty0+2, tiCoatL) // lapels
			b.set(13+ox, ty0+2, tiCoatL)
			b.set(10+ox, ty0+3, tiCoatL)
			b.set(13+ox, ty0+3, tiCoatL)
			b.set(12+ox, ty0+7, tiButton)
			b.set(12+ox, ty0+9, tiButton)
			// the coat opens over the legs: two front flaps
			for r := 1; r <= 3; r++ {
				b.rect(8+ox, hipY+r, 2, 1, tiCoat)
				b.rect(14+ox, hipY+r, 2, 1, tiCoatD)
			}
		} else {
			b.rect(11+ox, ty0+3, 1, hipY-ty0-2, tiCoatD) // back seam
			b.set(9+ox, ty0+9, tiButton)
			b.set(14+ox, ty0+9, tiButton)
			for r := 1; r <= 4; r++ { // two long tails
				b.rect(9+ox, hipY+r, 2, 1, tiCoat)
				b.rect(13+ox, hipY+r, 2, 1, tiCoatD)
			}
		}
		b.cur = 1 // the long neck belongs to the head
		b.rect(L, hy+10, 2, 2, tiSkin)
		b.set(R, hy+11, tiSkinD)
	}

	head := func() {
		b.cur = 1
		switch vw {
		case vSide:
			b.rect(L, hy+10, 2, 2, tiSkin) // neck
			b.set(L, hy+11, tiSkinD)
			b.ellipse(11.5+float64(hd), float64(hy)+4.8, 3.6, 5.0, tiSkin)
			b.set(L-2, hy+1, tiSkinL)
			b.set(L-1, hy+1, tiSkinL)
			b.rect(L-2, hy+4, 1, 3, tiSkinD) // ear
			b.set(L-1, hy+5, tiSkinD)
			b.rect(L-3, hy+2, 2, 2, tiHair) // tuft behind the ear
			b.set(L-4, hy+3, tiHair)
			b.set(L, hy-1, tiHair) // one of the two hairs
			b.set(L+1, hy-1, tiHair)
			b.rect(L-1, hy+3, 3, 1, tiFrame) // glasses arm
			b.rect(L+2, hy+2, 3, 4, tiFrame) // round lens
			b.rect(L+3, hy+3, 2, 2, tiLens)
			b.set(L+4, hy+4, tiFrame) // pupil
			// the big drooping nose
			b.rect(R+4, hy+4, 1, 1, tiSkin)
			b.rect(R+4, hy+5, 2, 1, tiSkin)
			b.rect(R+4, hy+6, 3, 1, tiSkin)
			b.rect(R+4, hy+7, 2, 1, tiNoseD)
			b.set(R+6, hy+6, tiSkinD)
			b.set(R+3, hy+7, tiNoseD)
			if open {
				b.rect(R+1, hy+8, 3, 2, tiMouth)
			} else {
				b.rect(R+1, hy+8, 2, 1, tiLip)
			}
		case vBack:
			b.ellipse(11.5+float64(hd), float64(hy)+4.8, 3.7, 5.0, tiSkin)
			b.rect(L-4, hy+4, 1, 2, tiSkin) // ears
			b.rect(R+4, hy+4, 1, 2, tiSkin)
			b.rect(L-3, hy+2, 1, 2, tiHair)
			b.rect(R+3, hy+2, 1, 2, tiHair)
			b.set(L-1, hy-1, tiHair)
			b.set(R+1, hy-1, tiHair)
			b.set(L-3, hy+4, tiFrame) // glasses arms over the ears
			b.set(R+3, hy+4, tiFrame)
			b.rect(L-1, hy+7, 4, 1, tiSkinD)
		default:
			b.ellipse(11.5+float64(hd), float64(hy)+4.8, 3.7, 5.0, tiSkin)
			b.rect(R+3, hy+3, 1, 5, tiSkinD) // shaded cheek
			b.set(L-2, hy+1, tiSkinL)
			b.set(L-1, hy+1, tiSkinL)
			b.rect(L-4, hy+4, 1, 2, tiSkin) // ears
			b.rect(R+4, hy+4, 1, 2, tiSkinD)
			b.rect(L-4, hy+2, 1, 2, tiHair) // a tuft over each ear
			b.rect(R+4, hy+2, 1, 2, tiHair)
			b.set(L-3, hy+2, tiHair)
			b.set(R+3, hy+2, tiHair)
			b.set(L-1, hy-1, tiHair) // two hairs on top
			b.set(R+1, hy-1, tiHair)
			// round glasses: two lenses on a bridge
			b.rect(L-3, hy+3, 3, 3, tiFrame)
			b.rect(R+1, hy+3, 3, 3, tiFrame)
			b.rect(L-2, hy+3, 2, 2, tiLens)
			b.rect(R+1, hy+3, 2, 2, tiLens)
			b.set(L-1, hy+4, tiFrame)
			b.set(R+1, hy+4, tiFrame)
			// big nose hanging over the mouth
			b.rect(L, hy+4, 2, 2, tiSkin)
			b.rect(L-1, hy+6, 4, 1, tiSkin)
			b.rect(L, hy+7, 2, 1, tiNoseD)
			b.set(R+1, hy+6, tiSkinD)
			if open {
				b.rect(L, hy+8, 2, 2, tiMouth)
			} else {
				b.rect(L-1, hy+8, 3, 1, tiLip)
			}
		}
	}

	arm := func(sx float64, l Limb, sign float64, id uint8, dark bool) (int, int) {
		b.cur = id
		sy := float64(ty0 + 1)
		mx, my, ex, ey := tiaArmIK(vw, sx, sy, l, sign, 1.2, 5.4)
		thickLine(b, sx, sy, mx, my, 2, tiaDk(tiCoat, dark))
		thickLine(b, mx, my, ex, ey, 2, tiaDk(tiCoat, dark))
		thickLine(b, sx, sy, mx, my, 1, tiaDk(tiCoatL, dark))
		ix, iy := int(math.Floor(ex)), int(math.Floor(ey))
		b.rect(ix, iy, 2, 1, tiaDk(tiWhite, dark)) // cuff
		b.rect(ix, iy+1, 2, 2, tiaDk(tiSkin, dark))
		b.set(ix+1, iy+2, tiaDk(tiSkinD, dark))
		return ix, iy + 1
	}

	switch vw {
	case vSide:
		cx := float64(11 + ox)
		arm(cx-0.5, p.LA, 1, 3, true)
		leg(10, p.LF, 1, 5, true)
		leg(11, p.RF, 1, 6, false)
		coat()
		head()
		x, y := arm(cx, p.RA, 1, 4, false)
		tiaHeld(b, s, p, vw, x, y, 4)
	case vBack:
		leg(13, p.LF, 1, 5, false)
		leg(9, p.RF, -1, 6, false)
		coat()
		head()
		arm(float64(16+ox), p.LA, 1, 3, false)
		x, y := arm(float64(6+ox), p.RA, -1, 4, false)
		tiaHeld(b, s, p, vw, x, y, 4)
	default:
		leg(9, p.LF, -1, 5, false)
		leg(13, p.RF, 1, 6, false)
		coat()
		head()
		arm(float64(6+ox), p.LA, -1, 3, false)
		x, y := arm(float64(16+ox), p.RA, 1, 4, false)
		tiaHeld(b, s, p, vw, x, y, 4)
	}
	return b
}

// ---- Filemon --------------------------------------------------------------------------------

func genFilemon(s *CharSpec, vw int, p Pose) *pixBuf {
	b := newPix(charW, charH)
	if trackParts {
		b.pid = make([]uint8, charW*charH)
	}
	bob := clampI(tiaRnd(p.Bob), -2, 3)
	if p.Sit {
		bob = maxI(bob, 2)
	}
	ox := clampI(tiaRnd(p.Sway*0.5), -2, 2)
	hd := ox
	if vw == vSide {
		ox = clampI(tiaRnd(p.Lean*0.5), -1, 2)
		hd = ox + clampI(tiaRnd(p.Lean*0.3+p.HeadZ*0.5), -1, 1)
	} else if p.HeadZ != 0 {
		hd += clampI(tiaRnd(p.HeadZ*0.5), -1, 1)
	}
	hipY := 34 + bob
	ty0 := hipY - 9 // shoulders
	hy := ty0 - 9   // skull top (the double chin rests on the collar)
	open := p.Open || p.Anim == AnimLaugh || p.Anim == AnimCheer || p.Anim == AnimPanic
	L, R := 11+hd, 12+hd

	leg := func(hx0 float64, l Limb, sign float64, id uint8, dark bool) {
		b.cur = id
		mx, my, ex, ey, top := tiaLegIK(vw, p, hx0, float64(hipY+1), l, sign, 0.5, 2.8, 40)
		thickLine(b, hx0, float64(hipY+1), mx, my, 3, tiaDk(tiTrou, dark))
		thickLine(b, mx, my, ex, ey, 3, tiaDk(tiTrou, dark))
		thickLine(b, hx0+1, float64(hipY+1), mx+1, my, 1, tiaDk(tiStripe, dark)) // the stripes
		thickLine(b, mx+1, my, ex+1, ey, 1, tiaDk(tiStripe, dark))
		ix := int(math.Floor(ex))
		if vw == vSide {
			b.rect(ix, top, 4, 1, tiaDk(tiShoeBL, dark))
			b.rect(ix-1, top+1, 6, 1, tiaDk(tiShoeB, dark))
			b.rect(ix-1, top+2, 6, 1, tiaDk(tiShoeBD, dark))
			return
		}
		b.rect(ix, top, 3, 1, tiaDk(tiShoeBL, dark))
		b.rect(ix-1+boolI(sign > 0), top+1, 4, 1, tiaDk(tiShoeB, dark))
		b.rect(ix-1+boolI(sign > 0), top+2, 4, 1, tiaDk(tiShoeBD, dark))
	}

	torso := func() {
		b.cur = 2
		if vw == vSide {
			c := 11 + ox
			for r := 0; r <= hipY-ty0-1; r++ {
				x0, x1 := c-3, c+3
				if r == 0 {
					x0, x1 = c-2, c+2
				}
				if r >= 3 && r <= 8 { // the belly
					x1 = c + 4
				}
				for x := x0; x <= x1; x++ {
					col := tiVest
					if x == x0 {
						col = tiVestD
					}
					if r == 0 {
						col = tiWhite
					}
					if x >= x1-1 && r <= 3 {
						col = tiWhite // shirt front over the waistcoat
					}
					if x == x1 && r >= 4 {
						col = tiVestL
					}
					b.set(x, ty0+r, col)
				}
			}
			b.set(c+4, ty0+6, tiButton)
			b.rect(c-3, hipY, 7, 2, tiTrou) // seat of the trousers
			b.rect(c-3, hipY+1, 7, 1, tiTrouD)
			return
		}
		widths := [][2]int{{8, 15}, {7, 16}, {6, 17}, {6, 17}, {6, 17}, {6, 17}, {6, 17}, {7, 16}, {7, 16}}
		for r, w := range widths {
			for x := w[0]; x <= w[1]; x++ {
				col := tiVest
				switch {
				case x <= 7 || x >= 16 || r == 0:
					col = tiWhite // shoulders and sleeves tops of the shirt
					if x == w[1] {
						col = tiWhiteD
					}
				case x == 8:
					col = tiVestL
				case x == 15:
					col = tiVestD
				}
				b.set(x+ox, ty0+r, col)
			}
		}
		if vw == vFront {
			for r := 0; r <= 3; r++ { // the shirt shows in the V of the waistcoat
				hw := 2 - r/2
				b.rect(12-hw+ox, ty0+r, hw*2, 1, tiWhite)
			}
			b.set(10+ox, ty0, tiWhiteD) // collar points
			b.set(13+ox, ty0, tiWhiteD)
			for _, r := range []int{4, 6} {
				b.set(11+ox, ty0+r, tiButton)
			}
			b.rect(9+ox, ty0+6, 2, 1, tiVestD) // pocket
		} else {
			b.rect(9+ox, ty0+5, 6, 1, tiVestD) // back strap and buckle
			b.set(12+ox, ty0+5, tiButton)
		}
		b.rect(7, hipY, 10, 2, tiTrou) // waist of the trousers
		b.rect(7, hipY+1, 10, 1, tiTrouD)
		for _, x := range []int{8, 11, 14} {
			b.set(x, hipY, tiStripe)
		}
	}

	head := func() {
		b.cur = 1
		switch vw {
		case vSide:
			b.ellipse(11.5+float64(hd), float64(hy)+5.2, 4.3, 5.0, tiSkin)
			b.set(L-2, hy+1, tiSkinL)
			b.set(L-1, hy+1, tiSkinL)
			b.rect(L-1, hy+4, 2, 3, tiSkinD) // ear
			b.rect(L-4, hy+3, 2, 4, tiGinger)
			b.set(L-4, hy+6, tiGingerD)
			b.set(L-3, hy+2, tiGinger)
			b.set(L, hy-1, tiHair) // two curly hairs
			b.set(L+1, hy-2, tiHair)
			b.set(R, hy-1, tiHair)
			b.rect(R+1, hy+3, 2, 1, tiHair) // brow
			b.set(R+2, hy+4, tiWhite)
			b.set(R+2, hy+5, tiFrame)
			b.rect(R+4, hy+5, 2, 2, tiSkin) // round nose
			b.set(R+5, hy+6, tiSkinD)
			b.rect(R+4, hy+7, 1, 1, tiNoseD)
			if open {
				b.rect(R+1, hy+8, 3, 2, tiMouth)
			} else {
				b.rect(R+1, hy+8, 3, 1, tiMouth)
			}
			b.rect(R-1, hy+10, 4, 1, tiSkinD) // double chin
		case vBack:
			b.ellipse(11.5+float64(hd), float64(hy)+5.2, 4.6, 5.0, tiSkin)
			b.rect(L-5, hy+4, 1, 3, tiSkinD)
			b.rect(R+5, hy+4, 1, 3, tiSkinD)
			// the horseshoe of ginger hair round the back
			b.rect(L-4, hy+4, 10, 3, tiGinger)
			b.rect(L-3, hy+7, 8, 1, tiGingerD)
			b.set(L, hy-1, tiHair)
			b.set(L-1, hy-2, tiHair)
			b.set(R, hy-1, tiHair)
			b.set(R+1, hy-2, tiHair)
		default:
			b.ellipse(11.5+float64(hd), float64(hy)+5.2, 4.6, 5.0, tiSkin)
			b.rect(R+4, hy+3, 1, 5, tiSkinD)
			b.set(L-2, hy+1, tiSkinL)
			b.set(L-1, hy+1, tiSkinL)
			b.rect(L-5, hy+4, 1, 3, tiSkin) // ears
			b.rect(R+5, hy+4, 1, 3, tiSkinD)
			b.rect(L-4, hy+2, 2, 3, tiGinger) // ginger tufts
			b.rect(R+3, hy+2, 2, 3, tiGinger)
			b.set(L-4, hy+4, tiGingerD)
			b.set(R+4, hy+4, tiGingerD)
			b.set(L, hy-1, tiHair) // two curly hairs
			b.set(L-1, hy-2, tiHair)
			b.set(R, hy-1, tiHair)
			b.set(R+1, hy-2, tiHair)
			b.rect(L-2, hy+3, 2, 1, tiHair) // cross brows
			b.rect(R+1, hy+3, 2, 1, tiHair)
			b.set(L-1, hy+4, tiWhite)
			b.set(R+1, hy+4, tiWhite)
			b.set(L-1, hy+5, tiFrame)
			b.set(R+1, hy+5, tiFrame)
			b.rect(L, hy+5, 2, 1, tiSkin) // round nose
			b.rect(L-1, hy+6, 4, 1, tiSkin)
			b.rect(L, hy+7, 2, 1, tiNoseD)
			b.set(R+1, hy+6, tiSkinD)
			if open {
				b.rect(L-1, hy+8, 4, 2, tiMouth)
				b.rect(L, hy+9, 2, 1, tiRed)
			} else {
				b.rect(L-1, hy+8, 4, 1, tiMouth)
				b.set(L-2, hy+9, tiMouth) // grumpy corners
				b.set(R+2, hy+9, tiMouth)
			}
			b.rect(L-1, hy+10, 4, 1, tiSkinD) // double chin
		}
	}

	arm := func(sx float64, l Limb, sign float64, id uint8, dark bool) (int, int) {
		b.cur = id
		sy := float64(ty0 + 2)
		mx, my, ex, ey := tiaArmIK(vw, sx, sy, l, sign, 0.85, 3.9)
		thickLine(b, sx, sy, mx, my, 2, tiaDk(tiWhite, dark))
		thickLine(b, mx, my, ex, ey, 2, tiaDk(tiWhite, dark))
		thickLine(b, mx+1, my, ex+1, ey, 1, tiaDk(tiWhiteD, dark))
		ix, iy := int(math.Floor(ex)), int(math.Floor(ey))
		b.rect(ix, iy+1, 2, 2, tiaDk(tiSkin, dark))
		b.set(ix+1, iy+2, tiaDk(tiSkinD, dark))
		return ix, iy + 1
	}

	switch vw {
	case vSide:
		cx := float64(11 + ox)
		arm(cx-0.5, p.LA, 1, 3, true)
		leg(9.5, p.LF, 1, 5, true)
		leg(10.5, p.RF, 1, 6, false)
		torso()
		head()
		x, y := arm(cx, p.RA, 1, 4, false)
		tiaHeld(b, s, p, vw, x, y, 4)
	case vBack:
		leg(13, p.LF, 1, 5, false)
		leg(8, p.RF, -1, 6, false)
		torso()
		head()
		arm(float64(17+ox), p.LA, 1, 3, false)
		x, y := arm(float64(5+ox), p.RA, -1, 4, false)
		tiaHeld(b, s, p, vw, x, y, 4)
	default:
		leg(8, p.LF, -1, 5, false)
		leg(13, p.RF, 1, 6, false)
		torso()
		head()
		arm(float64(5+ox), p.LA, -1, 3, false)
		x, y := arm(float64(17+ox), p.RA, 1, 4, false)
		tiaHeld(b, s, p, vw, x, y, 4)
	}
	return b
}
