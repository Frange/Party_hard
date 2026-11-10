package main

import (
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// Lara Croft, Marty McFly, Doc Brown and Yoda (slugs "lara_croft", "marty_mcfly", "doc_brown"
// and "yoda") have their own procedural drawing (genLmdy), like Mortadelo or Harry: the generic
// body could not give them their silhouettes. Small pixel-art parodies with our own pixels:
//   - Lara: long brown braid over the shoulder, tight turquoise top, brown shorts with a belt
//     and holsters on the thighs, tall laced brown boots, dark gloves and a pistol in each hand,
//     a small backpack.
//   - Marty: short brown hair with volume, red puffy vest over a white and black checked shirt,
//     blue jeans and white trainers.
//   - Doc: wild white hair, crazy wide eyes, long open yellow coat over a patterned red shirt,
//     yellow trousers, dark red boots and a little remote with an aerial.
//   - Yoda: very short, big green head with long pointy ears, big dark eyes, wrinkles, white
//     wisps, cream robe with a brown collar and a crooked walking stick.
// Only the drawing changes: hitbox, footprint, stats, names and meta.json are the usual ones.
// The animation is the usual one, read from the pose.
// ---------------------------------------------------------------------------

const (
	lmNone = iota
	lmLara
	lmMarty
	lmDoc
	lmYoda
)

func lmdyKind(s *CharSpec) int {
	if s == nil {
		return lmNone
	}
	switch s.Name {
	case "lara_croft":
		return lmLara
	case "marty_mcfly":
		return lmMarty
	case "doc_brown":
		return lmDoc
	case "yoda":
		return lmYoda
	}
	return lmNone
}

func isLmdy(s *CharSpec) bool { return lmdyKind(s) != lmNone }

var (
	lmInk   = hx(0x18161c)
	lmWhite = hx(0xf6f6f2)
	lmEyeW  = hx(0xf4f2ea)
	lmMouth = hx(0x7a2018)

	// Lara
	laSkin   = hx(0xf0c49c)
	laSkinD  = hx(0xc89470)
	laSkinL  = hx(0xffdcbc)
	laHair   = hx(0x5c3418)
	laHairL  = hx(0x86522a)
	laHairD  = hx(0x38200c)
	laTop    = hx(0x24a6b4)
	laTopL   = hx(0x5cd2d8)
	laTopD   = hx(0x147080)
	laShort  = hx(0x8c6a3c)
	laShortL = hx(0xae8c56)
	laShortD = hx(0x5e4420)
	laBelt   = hx(0x3a2412)
	laBuckle = hx(0xd8b048)
	laHolst  = hx(0x2a1a0e)
	laBoot   = hx(0x6c4220)
	laBootL  = hx(0x946034)
	laBootD  = hx(0x40240c)
	laLace   = hx(0xd8c098)
	laGlove  = hx(0x2c2622)
	laGun    = hx(0x2a2a32)
	laGunL   = hx(0x70707c)
	laLip    = hx(0xc05a50)
	laPack   = hx(0x6c4a28)
	laPackD  = hx(0x46301a)

	// Marty
	maSkin  = hx(0xf0c8a0)
	maSkinD = hx(0xcc9a74)
	maHair  = hx(0x5c3a20)
	maHairL = hx(0x84582e)
	maHairD = hx(0x3a2410)
	maVest  = hx(0xd42a20)
	maVestL = hx(0xf45a44)
	maVestD = hx(0x8c1810)
	maChkW  = hx(0xf2f2ee)
	maChkK  = hx(0x34343c)
	maJean  = hx(0x3c5c9c)
	maJeanL = hx(0x5c80bc)
	maJeanD = hx(0x26406e)
	maBelt  = hx(0x4a3020)
	maShoe  = hx(0xf6f6f6)
	maSole  = hx(0xa8a8b2)
	maSwsh  = hx(0xd03028)

	// Doc
	dcSkin  = hx(0xf0c8a8)
	dcSkinD = hx(0xc8987a)
	dcHair  = hx(0xf2f2f0)
	dcHairL = hx(0xffffff)
	dcHairD = hx(0xb4b6c0)
	dcCoat  = hx(0xe8c22a)
	dcCoatL = hx(0xfae070)
	dcCoatD = hx(0xa88412)
	dcShirt = hx(0xc42828)
	dcShirD = hx(0x7a1210)
	dcDot   = hx(0xf0d060)
	dcPant  = hx(0xd0aa2c)
	dcPantD = hx(0x94741a)
	dcBoot  = hx(0x6a1c16)
	dcBootL = hx(0x983028)
	dcBox   = hx(0x5c5c66)
	dcBoxL  = hx(0x8c8c98)
	dcLed   = hx(0xff3a2a)

	// Yoda
	yoSkin   = hx(0x8cb460)
	yoSkinL  = hx(0xb0d482)
	yoSkinD  = hx(0x5e8440)
	yoEar    = hx(0xc8a088)
	yoEye    = hx(0x1e1a14)
	yoWisp   = hx(0xe2e2da)
	yoRobe   = hx(0xdac8a0)
	yoRobeL  = hx(0xeee2c4)
	yoRobeD  = hx(0xa89870)
	yoColl   = hx(0x6c4a2a)
	yoCollL  = hx(0x8e6640)
	yoStick  = hx(0x7a5634)
	yoStickL = hx(0xa07a4c)
)

func lmDk(c color.RGBA, dark bool) color.RGBA { return tiaDk(c, dark) }

func genLmdy(s *CharSpec, vw int, p Pose) *pixBuf {
	if lmdyKind(s) == lmYoda {
		return genYodaArt(s, vw, p)
	}
	return genLmdyAdult(s, vw, p, lmdyKind(s))
}

// ---- the three humans -------------------------------------------------------------------------

func genLmdyAdult(s *CharSpec, vw int, p Pose, k int) *pixBuf {
	b := newPix(charW, charH)
	if trackParts {
		b.pid = make([]uint8, charW*charH)
	}
	bob := clampI(tiaRnd(p.Bob), -1, 2)
	if p.Sit {
		bob = maxI(bob, 2)
	}
	ox := clampI(tiaRnd(p.Sway*0.5), -1, 1)
	hd := ox
	if vw == vSide {
		ox = clampI(tiaRnd(p.Lean*0.5), -1, 1)
		hd = ox + clampI(tiaRnd(p.Lean*0.3+p.HeadZ*0.5), -1, 1)
	} else if p.HeadZ != 0 {
		hd += clampI(tiaRnd(p.HeadZ*0.5), -1, 1)
	}
	open := p.Open || p.Anim == AnimLaugh || p.Anim == AnimCheer || p.Anim == AnimPanic
	sw := ciSwing(p)
	hipY, tH := 29+bob, 9
	legK, legSeg := 0.8, 5.0
	switch k {
	case lmMarty: // a bit shorter
		hipY, tH = 30+bob, 9
		legK, legSeg = 0.75, 4.5
	case lmDoc: // tall and lanky
		hipY, tH = 28+bob, 10
		legK, legSeg = 0.85, 5.5
	}
	ty0 := hipY - tH // shoulders
	hy := ty0 - 11   // skull top
	L, R := 11+hd, 12+hd
	hcx := 11.5 + float64(hd)

	leg := func(hxl float64, l Limb, sign float64, id uint8, dark bool) {
		b.cur = id
		hy0 := float64(hipY + 1)
		mx, my, ex, ey, top := tiaLegIK(vw, p, hxl, hy0, l, sign, legK, legSeg, 40)
		ix := int(math.Floor(ex))
		knee := func(f float64) (float64, float64) { return hxl + (mx-hxl)*f, hy0 + (my-hy0)*f }
		switch k {
		case lmLara:
			thickLine(b, hxl, hy0, mx, my, 2, lmDk(laSkin, dark))
			sx, sy := knee(0.35)
			thickLine(b, hxl, hy0, sx, sy, 2, lmDk(laShort, dark)) // shorts leg
			hx1, hy1 := knee(0.62)
			thickLine(b, hx1, hy1, hx1, hy1, 2, lmDk(laShortD, dark)) // thigh strap and the holster outside
			b.rect(int(math.Floor(hx1))+boolI(sign > 0 || vw == vSide), int(math.Floor(hy1)), 1, 2, lmDk(laHolst, dark))
			// tall laced boot from just under the knee
			kx, ky := mx+(ex-mx)*0.15, my+(ey-my)*0.15
			thickLine(b, mx, my, kx, ky, 2, lmDk(laSkin, dark))
			thickLine(b, kx, ky, ex, ey, 2, lmDk(laBoot, dark))
			thickLine(b, kx, ky, kx, ky, 2, lmDk(laBootD, dark)) // the cuff
			for f := 0.45; f < 0.95; f += 0.3 {
				lx, ly := kx+(ex-kx)*f, ky+(ey-ky)*f
				b.set(int(math.Floor(lx))+boolI(vw != vSide && sign > 0), int(math.Floor(ly)), lmDk(laLace, dark))
			}
			lmShoe(b, vw, ix, top, sign, laBoot, laBootL, laBootD, dark)
		case lmMarty:
			thickLine(b, hxl, hy0, mx, my, 2, lmDk(maJean, dark))
			thickLine(b, mx, my, ex, ey, 2, lmDk(maJean, dark))
			thickLine(b, hxl+boolF(sign > 0 && vw != vSide), hy0+1, mx+boolF(sign > 0 && vw != vSide), my, 1, lmDk(maJeanL, dark))
			b.rect(ix, top-1, 2, 1, lmDk(maJeanD, dark)) // turned-up hem
			lmShoe(b, vw, ix, top, sign, maShoe, maShoe, maSole, dark)
			if vw == vSide {
				b.set(ix+1, top+1, lmDk(maSwsh, dark))
				b.set(ix+2, top+1, lmDk(maSwsh, dark))
			} else {
				b.set(ix+boolI(sign > 0), top+1, lmDk(maSwsh, dark))
			}
		case lmDoc:
			thickLine(b, hxl, hy0, mx, my, 2, lmDk(dcPant, dark))
			thickLine(b, mx, my, ex, ey, 2, lmDk(dcPant, dark))
			thickLine(b, hxl+boolF(sign < 0 || vw == vSide), hy0+1, mx+boolF(sign < 0 || vw == vSide), my, 1, lmDk(dcPantD, dark))
			b.rect(ix, top-2, 2, 2, lmDk(dcBoot, dark)) // boot shaft
			lmShoe(b, vw, ix, top, sign, dcBoot, dcBootL, shade(dcBoot, 0.6), dark)
		}
	}

	// Lara's braid: behind in profile and from the back, over the shoulder from the front.
	braid := func(front bool) {
		if k != lmLara {
			return
		}
		b.cur = 1
		switch vw {
		case vSide:
			x, y := L-3, hy+5
			for i := 0; i < 11; i++ {
				c := laHair
				if i%2 == 1 {
					c = laHairL
				}
				dx := 0
				if i > 3 {
					dx = -(i-3)/3 - sw*boolI(i > 6)
				}
				b.rect(x+dx, y+i, 2, 1, c)
			}
			b.set(L-4-2-sw, hy+16, laHairD)
		case vBack:
			if front {
				return
			}
			for i := 0; i < 14; i++ {
				c := laHair
				if i%2 == 1 {
					c = laHairL
				}
				b.rect(L+sw*boolI(i > 8), hy+8+i, 2, 1, c)
			}
			b.set(L+sw, hy+22, laHairD)
			b.set(R+sw, hy+22, laHairD)
		default:
			if !front {
				return
			}
			// down over the right shoulder (screen left) onto the chest
			for i := 0; i < 10; i++ {
				c := laHair
				if i%2 == 1 {
					c = laHairL
				}
				x := L - 4
				if i > 4 {
					x = L - 3
				}
				b.rect(x, hy+8+i, 2, 1, c)
			}
			b.set(L-3, hy+18, laHairD)
			b.set(L-2, hy+18, laHairD)
		}
	}

	torso := func() {
		b.cur = 2
		switch k {
		case lmLara:
			laraTorso(b, vw, ox, ty0, hipY, L, R)
		case lmMarty:
			martyTorso(b, vw, ox, ty0, hipY, L, R)
		case lmDoc:
			docTorso(b, vw, ox, ty0, hipY, L, R)
		}
	}

	head := func() {
		b.cur = 1
		switch k {
		case lmLara:
			laraHead(b, vw, L, R, hy, hcx, open)
		case lmMarty:
			martyHead(b, vw, L, R, hy, hcx, open)
		case lmDoc:
			docHead(b, vw, L, R, hy, hcx, open, sw)
		}
	}

	arm := func(sx float64, l Limb, sign float64, id uint8, dark bool, near bool) (int, int) {
		b.cur = id
		sy := float64(ty0 + 1)
		mx, my, ex, ey := tiaArmIK(vw, sx, sy, l, sign, 1.0, 4.4)
		ix, iy := int(math.Floor(ex)), int(math.Floor(ey))
		switch k {
		case lmLara:
			thickLine(b, sx, sy, mx, my, 2, lmDk(laSkin, dark))
			thickLine(b, mx, my, ex, ey, 2, lmDk(laSkin, dark))
			thickLine(b, sx, sy, sx, sy, 2, lmDk(laTop, dark)) // the strap on the shoulder
			b.rect(ix, iy, 2, 1, lmDk(laGlove, dark))          // fingerless gloves
			b.rect(ix, iy+1, 2, 2, lmDk(laGlove, dark))
			b.set(ix+1, iy+2, lmDk(laSkinD, dark))
			// a pistol in each hand
			if vw == vSide {
				b.rect(ix+2, iy+1, 3, 1, lmDk(laGun, dark))
				b.set(ix+2, iy+2, lmDk(laGun, dark))
				b.set(ix+3, iy+1, lmDk(laGunL, dark))
			} else {
				gx := ix + boolI(sign > 0)
				b.rect(gx, iy+3, 1, 2, lmDk(laGun, dark))
				b.set(gx-int(sign), iy+3, lmDk(laGun, dark))
				b.set(gx, iy+3, lmDk(laGunL, dark))
			}
		case lmMarty:
			// checked shirt sleeves
			lmCheckLine(b, sx, sy, mx, my, dark)
			lmCheckLine(b, mx, my, ex, ey, dark)
			b.rect(ix, iy, 2, 1, lmDk(maChkW, dark))
			b.rect(ix, iy+1, 2, 2, lmDk(maSkin, dark))
			b.set(ix+1, iy+2, lmDk(maSkinD, dark))
		case lmDoc:
			thickLine(b, sx, sy, mx, my, 2, lmDk(dcCoat, dark))
			thickLine(b, mx, my, ex, ey, 2, lmDk(dcCoat, dark))
			thickLine(b, sx, sy, mx, my, 1, lmDk(dcCoatL, dark))
			b.rect(ix-boolI(sign < 0), iy, 3, 1, lmDk(dcCoatD, dark)) // wide cuff
			b.rect(ix, iy+1, 2, 2, lmDk(dcSkin, dark))
			b.set(ix+1, iy+2, lmDk(dcSkinD, dark))
			if near { // the little remote with its aerial
				gx := ix + 1
				if vw != vSide && sign < 0 {
					gx = ix - 1
				}
				if vw == vSide {
					gx = ix + 2
				}
				b.rect(gx, iy+1, 2, 3, lmDk(dcBox, dark))
				b.set(gx, iy+1, lmDk(dcBoxL, dark))
				b.set(gx+1, iy+2, dcLed)
				b.rect(gx+1, iy-1, 1, 2, lmDk(dcBoxL, dark))
			}
		}
		return ix, iy + 1
	}

	lx, rx := float64(6+ox), float64(16+ox)
	if k == lmMarty || k == lmDoc {
		lx, rx = float64(5+ox), float64(17+ox)
	}
	switch vw {
	case vSide:
		cx := float64(11 + ox)
		arm(cx-0.5, p.LA, 1, 3, true, false)
		leg(10, p.LF, 1, 5, true)
		leg(11, p.RF, 1, 6, false)
		braid(false)
		torso()
		head()
		x, y := arm(cx, p.RA, 1, 4, false, true)
		tiaHeld(b, s, p, vw, x, y, 4)
	case vBack:
		leg(13, p.LF, 1, 5, false)
		leg(9, p.RF, -1, 6, false)
		torso()
		arm(rx, p.LA, 1, 3, false, false)
		x, y := arm(lx, p.RA, -1, 4, false, true)
		head()
		braid(false)
		tiaHeld(b, s, p, vw, x, y, 4)
	default:
		leg(9, p.LF, -1, 5, false)
		leg(13, p.RF, 1, 6, false)
		torso()
		head()
		braid(true)
		arm(lx, p.LA, -1, 3, false, false)
		x, y := arm(rx, p.RA, 1, 4, false, true)
		tiaHeld(b, s, p, vw, x, y, 4)
	}
	return b
}

func boolF(v bool) float64 { return float64(boolI(v)) }

// lmShoe: a shoe whose top row is top under the ankle column ix.
func lmShoe(b *pixBuf, vw, ix, top int, sign float64, c, cl, cd color.RGBA, dark bool) {
	if vw == vSide {
		b.rect(ix, top, 3, 1, lmDk(c, dark))
		b.rect(ix-1, top+1, 5, 1, lmDk(c, dark))
		b.rect(ix+1, top, 2, 1, lmDk(cl, dark))
		b.rect(ix-1, top+2, 5, 1, lmDk(cd, dark))
		return
	}
	x0 := ix - 1
	if sign > 0 {
		x0 = ix
	}
	b.rect(x0, top, 3, 1, lmDk(c, dark))
	b.rect(x0-1+boolI(sign > 0), top+1, 3, 1, lmDk(c, dark))
	b.set(x0, top, lmDk(cl, dark))
	b.rect(x0-1+boolI(sign > 0), top+2, 3, 1, lmDk(cd, dark))
}

// lmCheckLine: a 2 px limb in Marty's white and black checks.
func lmCheckLine(b *pixBuf, x0, y0, x1, y1 float64, dark bool) {
	steps := int(math.Max(math.Abs(x1-x0), math.Abs(y1-y0))*2) + 1
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := int(math.Floor(x0 + (x1-x0)*t))
		y := int(math.Floor(y0 + (y1-y0)*t))
		for dx := 0; dx < 2; dx++ {
			c := maChkW
			if (x+dx+y)%2 == 0 {
				c = maChkK
			}
			b.set(x+dx, y, lmDk(c, dark))
		}
	}
}

// ---- Lara -------------------------------------------------------------------------------------

func laraTorso(b *pixBuf, vw, ox, ty0, hipY, L, R int) {
	if vw == vSide {
		c := 11 + ox
		for y := ty0; y <= hipY+1; y++ {
			r := y - ty0
			x0, x1 := c-2, c+2
			if r >= 6 && r <= 7 {
				x0, x1 = c-1, c+2 // waist
			}
			for x := x0; x <= x1; x++ {
				col := laTop
				if x == x0 {
					col = laTopD
				}
				if x == x1 {
					col = laTopL
				}
				b.set(x, y, col)
			}
		}
		b.set(c+3, ty0+3, laTop) // bust
		b.set(c+3, ty0+4, laTopD)
		b.rect(c-2, hipY-2, 5, 1, laBelt)
		b.set(c+1, hipY-2, laBuckle)
		b.rect(c-2, hipY-1, 5, 3, laShort)
		b.rect(c-2, hipY-1, 1, 3, laShortD)
		b.set(c+2, hipY, laShortL)
		// the small backpack
		b.rect(c-4, ty0+2, 2, 5, laPack)
		b.rect(c-4, ty0+6, 2, 1, laPackD)
		b.set(c-3, ty0+1, laPackD) // strap
		// the neck
		b.cur = 1
		b.rect(c, ty0-1, 2, 1, laSkin)
		return
	}
	for y := ty0; y <= hipY+1; y++ {
		r := y - ty0
		x0, x1 := 8, 15
		switch {
		case r >= 5 && r <= 7:
			x0, x1 = 9, 14 // narrow waist
		case r == 0:
			x0, x1 = 9, 14
		}
		for x := x0; x <= x1; x++ {
			col := laTop
			if x == x0 {
				col = laTopL
			}
			if x == x1 {
				col = laTopD
			}
			b.set(x+ox, y, col)
		}
	}
	if vw == vFront {
		b.rect(10+ox, ty0, 4, 2, laSkin) // scoop neck
		b.rect(11+ox, ty0+2, 2, 1, laSkinD)
		b.set(10+ox, ty0+3, laTopL) // bust highlights
		b.set(13+ox, ty0+3, laTopL)
		b.rect(10+ox, ty0+4, 4, 1, laTopD)
	} else {
		b.rect(10+ox, ty0, 4, 1, laSkin)
		// the backpack on her back
		b.rect(9+ox, ty0+2, 6, 5, laPack)
		b.rect(9+ox, ty0+6, 6, 1, laPackD)
		b.rect(10+ox, ty0+3, 4, 1, laPackD) // flap
		b.set(9+ox, ty0+1, laPackD)
		b.set(14+ox, ty0+1, laPackD)
	}
	b.rect(8+ox, hipY-2, 8, 1, laBelt)
	if vw == vFront {
		b.rect(11+ox, hipY-2, 2, 1, laBuckle)
	}
	b.rect(8+ox, hipY-1, 8, 3, laShort)
	b.rect(8+ox, hipY-1, 1, 3, laShortL)
	b.rect(15+ox, hipY-1, 1, 3, laShortD)
	b.rect(11+ox, hipY, 2, 2, laShortD) // crotch seam
	b.cur = 1
	b.rect(L, ty0-1, 2, 1, laSkin) // neck
	b.set(R, ty0-1, laSkinD)
}

func laraHead(b *pixBuf, vw, L, R, hy int, hcx float64, open bool) {
	switch vw {
	case vSide:
		b.ellipse(hcx+0.5, float64(hy)+5, 3.6, 5, laSkin)
		b.ellipse(hcx-0.7, float64(hy)+3.6, 3.4, 3.8, laHair) // hair pulled back
		b.rect(R+1, hy+1, 2, 2, laHair)                       // hairline at the front
		b.set(L-2, hy+1, laHairL)
		b.set(L-1, hy+1, laHairL)
		b.rect(L-3, hy+6, 2, 2, laHairD) // the tie of the braid
		b.rect(L, hy+5, 1, 2, laSkinD)   // ear
		b.set(R+2, hy+4, laHairD)        // brow
		b.set(R+2, hy+5, lmInk)          // eye
		b.set(R+4, hy+6, laSkin)         // nose
		b.set(R+3, hy+6, laSkinD)
		if open {
			b.rect(R+2, hy+8, 2, 1, lmMouth)
		} else {
			b.rect(R+2, hy+8, 2, 1, laLip)
		}
		b.rect(R+1, hy+9, 2, 1, laSkinD) // jaw
	case vBack:
		b.ellipse(hcx, float64(hy)+5, 3.8, 5, laSkin)
		b.ellipse(hcx, float64(hy)+4.4, 4, 4.6, laHair)
		b.set(L-2, hy+2, laHairL)
		b.set(R+1, hy+1, laHairL)
		b.rect(L-1, hy+3, 1, 4, laHairD) // combed into the braid
		b.rect(R+1, hy+3, 1, 4, laHairD)
		b.rect(L, hy+7, 2, 1, laHairD) // the tie
		b.rect(L-4, hy+5, 1, 2, laSkin)
		b.rect(R+4, hy+5, 1, 2, laSkinD)
	default:
		b.ellipse(hcx, float64(hy)+5, 3.8, 5, laSkin)
		b.rect(R+3, hy+4, 1, 4, laSkinD) // shaded cheek
		b.set(L-1, hy+3, laSkinL)
		// hair: a middle parting, pulled back with two locks framing the face
		b.rect(L-3, hy, 8, 3, laHair)
		b.rect(L-2, hy-1, 6, 1, laHair)
		b.set(L, hy, laHairD) // parting
		b.set(R, hy-1, laHairD)
		b.set(L-2, hy, laHairL)
		b.set(R+2, hy, laHairL)
		b.rect(L-4, hy+2, 1, 6, laHair)
		b.rect(R+4, hy+2, 1, 6, laHairD)
		b.set(L-3, hy+3, laHair)
		b.set(R+3, hy+3, laHairD)
		// eyes with brows, small nose, full lips
		b.set(L-2, hy+4, laHairD)
		b.set(L-1, hy+4, laHairD)
		b.set(R+1, hy+4, laHairD)
		b.set(R+2, hy+4, laHairD)
		b.set(L-2, hy+5, lmEyeW)
		b.set(L-1, hy+5, hx(0x5a3a1a))
		b.set(R+1, hy+5, hx(0x5a3a1a))
		b.set(R+2, hy+5, lmEyeW)
		b.set(R, hy+7, laSkinD)
		if open {
			b.rect(L, hy+8, 2, 2, lmMouth)
		} else {
			b.rect(L, hy+8, 2, 1, laLip)
		}
	}
}

// ---- Marty ------------------------------------------------------------------------------------

func martyTorso(b *pixBuf, vw, ox, ty0, hipY, L, R int) {
	vest := func(x, y, x0, x1, r int) color.RGBA {
		col := maVest
		if r%3 == 2 { // the quilting
			col = maVestD
		}
		if x == x0 && r%3 != 2 {
			col = maVestL
		}
		if x == x1 {
			col = maVestD
		}
		return col
	}
	if vw == vSide {
		c := 11 + ox
		for y := ty0; y <= hipY-2; y++ {
			r := y - ty0
			for x := c - 2; x <= c+3; x++ {
				b.set(x, y, vest(x, y, c+3, c-2, r))
			}
		}
		b.rect(c-1, ty0-2, 3, 2, maVest) // the high collar
		b.set(c-1, ty0-2, maVestD)
		b.set(c+2, ty0-1, maChkW) // shirt collar
		b.rect(c+3, ty0+1, 1, 3, maChkK)
		b.rect(c-2, hipY-1, 6, 1, maBelt)
		b.rect(c-2, hipY, 6, 2, maJean)
		b.set(c+3, hipY, maJeanL)
		return
	}
	for y := ty0; y <= hipY-2; y++ {
		r := y - ty0
		x0, x1 := 7, 16
		if r == 0 {
			x0, x1 = 8, 15
		}
		for x := x0; x <= x1; x++ {
			b.set(x+ox, y, vest(x, y, x0, x1, r))
		}
	}
	if vw == vFront {
		// open front showing the checked shirt
		for y := ty0; y <= hipY-2; y++ {
			for x := 11; x <= 12; x++ {
				c := maChkW
				if (x+ox+y)%2 == 0 {
					c = maChkK
				}
				b.set(x+ox, y, c)
			}
		}
		b.set(10+ox, ty0, maChkW) // collar points
		b.set(13+ox, ty0, maChkW)
		b.rect(8+ox, ty0-2, 2, 2, maVest) // the turned-up vest collar
		b.rect(14+ox, ty0-2, 2, 2, maVestD)
		b.set(10+ox, ty0+4, maVestD) // pocket flaps
		b.set(13+ox, ty0+4, maVestD)
	} else {
		b.rect(8+ox, ty0-2, 8, 2, maVest)
		b.rect(8+ox, ty0-2, 8, 1, maVestL)
	}
	b.rect(8+ox, hipY-1, 8, 1, maBelt)
	if vw == vFront {
		b.set(11+ox, hipY-1, maSole)
		b.set(12+ox, hipY-1, maSole)
	}
	b.rect(8+ox, hipY, 8, 2, maJean)
	b.rect(8+ox, hipY, 1, 2, maJeanL)
	b.rect(15+ox, hipY, 1, 2, maJeanD)
	b.cur = 1
	b.rect(L, ty0-1, 2, 1, maSkin)
}

func martyHead(b *pixBuf, vw, L, R, hy int, hcx float64, open bool) {
	switch vw {
	case vSide:
		b.ellipse(hcx+0.5, float64(hy)+5, 3.6, 5, maSkin)
		b.ellipse(hcx-0.6, float64(hy)+2.8, 3.7, 3.2, maHair)
		b.rect(R-1, hy-1, 4, 2, maHair) // the quiff at the front
		b.set(R+3, hy, maHair)
		b.set(R, hy-1, maHairL)
		b.set(R+1, hy-1, maHairL)
		b.rect(L-3, hy+4, 2, 3, maHair) // short back
		b.rect(L, hy+5, 1, 2, maSkinD)  // ear
		b.set(R+2, hy+4, maHairD)
		b.set(R+2, hy+5, lmInk)
		b.set(R+4, hy+6, maSkin)
		b.set(R+3, hy+7, maSkinD)
		if open {
			b.rect(R+2, hy+8, 2, 1, lmMouth)
		} else {
			b.rect(R+2, hy+8, 2, 1, hx(0xa85a4a))
		}
	case vBack:
		b.ellipse(hcx, float64(hy)+5, 3.8, 5, maSkin)
		b.ellipse(hcx, float64(hy)+3.8, 4.1, 4.2, maHair)
		b.rect(L-3, hy+6, 8, 2, maHair)
		b.set(L-2, hy+1, maHairL)
		b.set(R+1, hy, maHairL)
		b.rect(L-1, hy+8, 4, 1, maSkinD) // nape
		b.rect(L-4, hy+5, 1, 2, maSkin)
		b.rect(R+4, hy+5, 1, 2, maSkinD)
	default:
		b.ellipse(hcx, float64(hy)+5, 3.8, 5, maSkin)
		b.rect(R+3, hy+4, 1, 4, maSkinD)
		b.rect(L-4, hy+5, 1, 2, maSkin) // ears
		b.rect(R+4, hy+5, 1, 2, maSkinD)
		// volume on top, side parting, short at the sides
		b.rect(L-3, hy-1, 8, 3, maHair)
		b.rect(L-2, hy-2, 6, 1, maHair)
		b.rect(L-4, hy+1, 1, 4, maHair)
		b.rect(R+4, hy+1, 1, 4, maHairD)
		b.rect(L-1, hy-2, 3, 1, maHairL) // shine of the quiff
		b.set(L-2, hy-1, maHairL)
		b.set(R+1, hy+2, maHair) // fringe swept to one side
		b.set(R+2, hy+2, maHair)
		b.set(L-3, hy+2, maHairD)
		b.set(R+3, hy+2, maHairD)
		b.set(L-2, hy+4, maHairD) // brows
		b.set(L-1, hy+4, maHairD)
		b.set(R+1, hy+4, maHairD)
		b.set(R+2, hy+4, maHairD)
		b.set(L-1, hy+5, lmInk)
		b.set(R+1, hy+5, lmInk)
		b.set(L-2, hy+5, lmEyeW)
		b.set(R+2, hy+5, lmEyeW)
		b.set(R, hy+7, maSkinD)
		if open {
			b.rect(L-1, hy+8, 4, 2, lmMouth)
			b.rect(L-1, hy+8, 4, 1, lmWhite)
		} else { // the cheeky lopsided smile
			b.rect(L-1, hy+8, 3, 1, hx(0xa85a4a))
			b.set(R+1, hy+7, hx(0xa85a4a))
		}
	}
}

// ---- Doc --------------------------------------------------------------------------------------

func docTorso(b *pixBuf, vw, ox, ty0, hipY, L, R int) {
	shirt := func(x, y int) color.RGBA { // red with a small pattern
		switch (x*3 + y*2) % 7 {
		case 0:
			return dcDot
		case 3:
			return dcShirD
		}
		return dcShirt
	}
	if vw == vSide {
		c := 11 + ox
		for y := ty0; y <= hipY+6; y++ {
			r := y - ty0
			x0, x1 := c-2, c+2
			if y > hipY {
				x0, x1 = c-3+(y-hipY)/4, c+1 // the long tail of the coat
			}
			for x := x0; x <= x1; x++ {
				col := dcCoat
				if x == x0 {
					col = dcCoatD
				}
				if x == x1-1 && r > 0 {
					col = dcCoatL
				}
				b.set(x, y, col)
			}
			if y <= hipY-1 && r >= 1 {
				b.set(c+3, y, shirt(c+3, y)) // the open coat shows the shirt
			}
		}
		b.rect(c-1, ty0-1, 3, 1, dcCoatD) // collar
		b.set(c+3, ty0, dcShirt)
		return
	}
	for y := ty0; y <= hipY+6; y++ {
		r := y - ty0
		x0, x1 := 7, 16
		if r == 0 {
			x0, x1 = 8, 15
		}
		if y > hipY+2 {
			x0, x1 = 7, 16
		}
		for x := x0; x <= x1; x++ {
			if vw == vFront && y > hipY && x >= 10 && x <= 13 {
				continue // the coat hangs open over the legs
			}
			col := dcCoat
			if x == x0 {
				col = dcCoatL
			}
			if x == x1 {
				col = dcCoatD
			}
			b.set(x+ox, y, col)
		}
	}
	if vw == vFront {
		// open coat: the shirt down the middle, the flaps over the legs
		for y := ty0; y <= hipY+6; y++ {
			for x := 10; x <= 13; x++ {
				if y <= hipY-1 {
					b.set(x+ox, y, shirt(x+ox, y))
				} else if y <= hipY {
					b.set(x+ox, y, dcPantD) // belt
				}
			}
		}
		b.rect(9+ox, ty0, 1, hipY+6-ty0, dcCoatD) // lapel edges
		b.rect(14+ox, ty0, 1, hipY+6-ty0, dcCoatL)
		b.set(10+ox, ty0, dcCoatL) // lapels
		b.set(13+ox, ty0, dcCoatD)
		b.set(8+ox, ty0+5, dcCoatD) // pockets
		b.set(15+ox, ty0+5, shade(dcCoatD, 0.85))
	} else {
		b.rect(8+ox, ty0-1, 8, 1, dcCoatD)   // the collar
		b.rect(11+ox, hipY+1, 2, 6, dcCoatD) // the back vent
		b.rect(8+ox, hipY-2, 8, 1, dcCoatD)  // the belt at the back
	}
	b.cur = 1
	b.rect(L, ty0-1, 2, 1, dcSkin)
}

func docHead(b *pixBuf, vw, L, R, hy int, hcx float64, open bool, sw int) {
	hairBlob := func(cx, cy, rx, ry float64) {
		b.ellipse(cx, cy, rx, ry, dcHair)
	}
	switch vw {
	case vSide:
		hairBlob(hcx-1.8, float64(hy)+3.8, 4.6, 4.8)
		b.ellipse(hcx+0.8, float64(hy)+5.2, 3.4, 4.8, dcSkin)
		b.rect(R, hy+1, 3, 2, dcSkin) // high bald forehead
		for _, q := range [][2]int{{-6, 1}, {-7, 4}, {-6, 7}, {-4, -1}, {-1, -2}, {2, -1}, {-7, 6}} {
			b.set(L+q[0]-sw*boolI(q[0] < -5), hy+q[1], dcHair)
		}
		for _, q := range [][2]int{{-5, 2}, {-3, 0}, {-5, 6}, {-2, 5}} {
			b.set(L+q[0], hy+q[1], dcHairD)
		}
		b.rect(L, hy+5, 1, 2, dcSkinD)   // ear
		b.rect(R+1, hy+3, 2, 1, dcHairD) // brow up high
		b.rect(R+2, hy+4, 2, 2, lmEyeW)  // wide eye
		b.set(R+3, hy+5, lmInk)
		b.set(R+5, hy+6, dcSkin) // big nose
		b.rect(R+4, hy+6, 1, 2, dcSkinD)
		if open {
			b.rect(R+2, hy+8, 3, 2, lmMouth)
			b.rect(R+2, hy+8, 3, 1, lmWhite)
		} else {
			b.rect(R+2, hy+8, 2, 1, lmMouth)
			b.set(R+4, hy+8, lmWhite)
		}
		b.rect(R+1, hy+7, 1, 2, dcSkinD) // wrinkles
	case vBack:
		hairBlob(hcx, float64(hy)+4.4, 5.6, 5.0)
		for _, q := range [][2]int{{-6, 0}, {-3, -2}, {1, -2}, {4, -1}, {6, 1}, {-7, 4}, {7, 4}, {-7, 7}, {7, 7}, {-5, 9}, {6, 9}} {
			b.set(L+q[0], hy+q[1], dcHair)
		}
		for _, q := range [][2]int{{-4, 2}, {-1, 1}, {2, 3}, {4, 6}, {-3, 6}, {0, 7}} {
			b.set(L+q[0], hy+q[1], dcHairD)
		}
		b.set(L-2, hy+1, dcHairL)
		b.rect(L-1, hy+10, 4, 1, dcSkinD) // nape
	default:
		hairBlob(hcx, float64(hy)+4.6, 5.8, 4.8) // the big white mane
		b.ellipse(hcx, float64(hy)+5.4, 3.7, 4.8, dcSkin)
		b.rect(L-2, hy+1, 6, 2, dcSkin) // receding: a high bald forehead
		b.set(L-3, hy+2, dcSkin)
		b.set(R+3, hy+2, dcSkin)
		// tufts shooting off everywhere
		for _, q := range [][2]int{{-6, 0}, {-4, -1}, {-1, -2}, {2, -2}, {5, -1}, {7, 0}, {-7, 3}, {8, 3}, {-7, 6}, {8, 6}, {-6, 9}, {7, 9}} {
			b.set(L+q[0]+sw*boolI(q[1] > 5)*boolI(q[0] > 0), hy+q[1], dcHair)
		}
		for _, q := range [][2]int{{-5, 2}, {6, 2}, {-5, 6}, {6, 6}, {-4, 8}, {5, 8}} {
			b.set(L+q[0], hy+q[1], dcHairD)
		}
		b.set(L-4, hy, dcHairL)
		b.set(R+3, hy, dcHairL)
		// raised brows over wide crazy eyes
		b.rect(L-3, hy+3, 2, 1, dcHairD)
		b.rect(R+1, hy+3, 2, 1, dcHairD)
		b.rect(L-3, hy+4, 2, 2, lmEyeW)
		b.rect(R+1, hy+4, 2, 2, lmEyeW)
		b.set(L-2, hy+5, lmInk)
		b.set(R+1, hy+4, lmInk)
		b.rect(L, hy+6, 2, 1, dcSkinD) // big nose
		b.set(R, hy+7, dcSkinD)
		if open {
			b.rect(L-1, hy+8, 4, 2, lmMouth)
			b.rect(L-1, hy+8, 4, 1, lmWhite)
		} else { // a toothy grimace
			b.rect(L-1, hy+8, 4, 1, lmWhite)
			b.set(L-2, hy+8, lmMouth)
			b.set(R+2, hy+8, lmMouth)
		}
		b.set(L-3, hy+7, dcSkinD) // wrinkles
		b.set(R+3, hy+7, dcSkinD)
	}
}

// ---- Yoda -------------------------------------------------------------------------------------

func genYodaArt(s *CharSpec, vw int, p Pose) *pixBuf {
	b := newPix(charW, charH)
	if trackParts {
		b.pid = make([]uint8, charW*charH)
	}
	bob := clampI(tiaRnd(p.Bob), -2, 2)
	if p.Sit {
		bob = maxI(bob, 2)
	}
	ox := clampI(tiaRnd(p.Sway*0.5), -1, 1)
	hd := ox
	if vw == vSide {
		ox = clampI(tiaRnd(p.Lean*0.5), -1, 1)
		hd = ox + clampI(tiaRnd(p.Lean*0.3+p.HeadZ*0.5), -1, 1)
	} else if p.HeadZ != 0 {
		hd += clampI(tiaRnd(p.HeadZ*0.5), -1, 1)
	}
	open := p.Open || p.Anim == AnimLaugh || p.Anim == AnimCheer || p.Anim == AnimPanic
	sw := ciSwing(p)
	hipY := 37 + bob
	ty0 := hipY - 8 // shoulders
	hy := ty0 - 10  // skull top
	L, R := 11+hd, 12+hd
	hcx := 11.5 + float64(hd)

	leg := func(hxl float64, l Limb, sign float64, id uint8, dark bool) {
		b.cur = id
		hy0 := float64(hipY)
		mx, my, ex, ey, top := tiaLegIK(vw, p, hxl, hy0, l, sign, 0.5, 2.3, 41)
		thickLine(b, hxl, hy0, mx, my, 2, lmDk(yoRobeD, dark))
		thickLine(b, mx, my, ex, ey, 2, lmDk(yoRobeD, dark))
		ix := int(math.Floor(ex))
		// a little green clawed foot
		if vw == vSide {
			b.rect(ix, top, 3, 1, lmDk(yoSkin, dark))
			b.rect(ix-1, top+1, 5, 1, lmDk(yoSkinD, dark))
			b.set(ix+3, top+1, lmDk(yoSkinL, dark))
			return
		}
		x0 := ix - 1 + boolI(sign > 0)
		b.rect(x0, top, 3, 1, lmDk(yoSkin, dark))
		b.rect(x0, top+1, 3, 1, lmDk(yoSkinD, dark))
		b.set(x0+boolI(sign > 0)*2, top+1, lmDk(yoSkinL, dark))
	}

	robe := func() {
		b.cur = 2
		if vw == vSide {
			c := 11 + ox
			for y := ty0; y <= hipY+2; y++ {
				r := y - ty0
				x0, x1 := c-2-r/4, c+2+r/5
				for x := x0; x <= x1; x++ {
					col := yoRobe
					if x == x0 {
						col = yoRobeD
					}
					if x == x1 {
						col = yoRobeL
					}
					b.set(x, y, col)
				}
			}
			b.rect(c-2, ty0-1, 5, 2, yoColl) // the brown collar
			b.set(c+2, ty0, yoCollL)
			b.rect(c-2-2, hipY+2, 6, 1, yoRobeD) // ragged hem
			return
		}
		for y := ty0; y <= hipY+2; y++ {
			r := y - ty0
			x0, x1 := 8-r/4, 15+r/4
			for x := x0; x <= x1; x++ {
				col := yoRobe
				if x == x0 {
					col = yoRobeL
				}
				if x == x1 {
					col = yoRobeD
				}
				b.set(x+ox, y, col)
			}
		}
		b.rect(6+ox, hipY+2, 12, 1, yoRobeD) // ragged hem
		b.set(8+ox, hipY+2, yoRobe)
		b.set(14+ox, hipY+2, yoRobe)
		if vw == vFront {
			// the brown under-robe and collar crossing in a V
			b.rect(9+ox, ty0-1, 6, 2, yoColl)
			b.set(11+ox, ty0+1, yoColl)
			b.set(12+ox, ty0+1, yoColl)
			b.set(11+ox, ty0+2, yoCollL)
			b.rect(11+ox, ty0+3, 1, hipY-ty0-1, yoRobeD) // the robe overlap
			b.rect(9+ox, hipY-3, 6, 1, yoColl)           // the sash
		} else {
			b.rect(9+ox, ty0-1, 6, 2, yoColl)
			b.rect(9+ox, ty0-1, 6, 1, yoCollL)
			b.rect(11+ox, ty0+3, 2, 4, yoRobeD) // folds
		}
	}

	head := func() {
		b.cur = 1
		hc := float64(hy) + 5
		switch vw {
		case vSide:
			// the long ear pointing back
			for i := 0; i < 6; i++ {
				b.rect(L-3-i, hy+4-i/3+sw*boolI(i > 3), 2, 2-boolI(i > 3), yoSkin)
			}
			b.rect(L-6, hy+4, 3, 1, yoEar)
			b.ellipse(hcx+0.5, hc, 4.4, 5, yoSkin)
			b.rect(R+2, hy+7, 3, 3, yoSkin) // snout
			b.set(R+5, hy+7, yoSkinD)
			b.set(L-1, hy+1, yoSkinL)
			b.rect(L+1, hy+3, 3, 1, yoSkinD) // wrinkles
			b.set(R+1, hy+2, yoSkinD)
			b.rect(R+2, hy+5, 2, 2, yoEye) // big eye
			b.set(R+2, hy+5, lmWhite)
			b.rect(R+1, hy+4, 3, 1, yoSkinD) // heavy lid
			if open {
				b.rect(R+3, hy+9, 2, 1, hx(0x3a2418))
			} else {
				b.rect(R+3, hy+9, 2, 1, yoSkinD)
			}
			b.set(L-2, hy, yoWisp)
			b.set(L-3, hy+1, yoWisp)
			b.set(L, hy-1, yoWisp)
		default:
			// long ears out to the sides, drooping a little
			for i := 0; i < 6; i++ {
				dy := i / 3
				if i > 3 {
					dy += sw
				}
				b.rect(L-4-i, hy+4+dy, 1, 3-boolI(i > 3)-boolI(i > 4), yoSkin)
				b.rect(R+4+i, hy+4+dy, 1, 3-boolI(i > 3)-boolI(i > 4), yoSkinD)
			}
			if vw == vFront { // pink inner ears
				b.rect(L-8, hy+5, 4, 1, yoEar)
				b.rect(R+5, hy+5, 4, 1, yoEar)
			}
			b.ellipse(hcx, hc, 4.6, 5, yoSkin)
			b.set(L-2, hy+1, yoSkinL)
			b.set(L-1, hy, yoSkinL)
			b.rect(R+4, hy+4, 1, 4, yoSkinD)
			if vw == vBack {
				b.rect(L-2, hy+3, 2, 1, yoSkinD)
				b.rect(R+1, hy+6, 2, 1, yoSkinD)
				b.set(L-3, hy, yoWisp)
				b.set(R+2, hy-1, yoWisp)
				b.set(R+4, hy+1, yoWisp)
				b.rect(L-1, hy+9, 4, 1, yoSkinD)
				break
			}
			b.rect(L-1, hy+2, 4, 1, yoSkinD) // forehead wrinkles
			b.set(L-2, hy+3, yoSkinD)
			b.set(R+1, hy+3, yoSkinD)
			b.rect(L-3, hy+4, 3, 1, yoSkinD) // heavy lids
			b.rect(R+1, hy+4, 3, 1, yoSkinD)
			b.rect(L-3, hy+5, 3, 2, yoEye) // big dark eyes
			b.rect(R+1, hy+5, 3, 2, yoEye)
			b.set(L-3, hy+5, lmWhite)
			b.set(R+1, hy+5, lmWhite)
			b.rect(L, hy+7, 2, 1, yoSkinD) // little nose
			b.set(L-4, hy+8, yoSkinD)      // cheek lines
			b.set(R+4, hy+8, yoSkinD)
			if open {
				b.rect(L-1, hy+9, 4, 1, hx(0x3a2418))
			} else {
				b.rect(L-1, hy+9, 4, 1, yoSkinD)
			}
			b.set(L-3, hy, yoWisp) // white wisps
			b.set(L-4, hy+1, yoWisp)
			b.set(R+3, hy, yoWisp)
			b.set(R+4, hy+1, yoWisp)
		}
	}

	arm := func(sx float64, l Limb, sign float64, id uint8, dark bool, stick bool) (int, int) {
		b.cur = id
		sy := float64(ty0 + 1)
		mx, my, ex, ey := tiaArmIK(vw, sx, sy, l, sign, 0.55, 2.8)
		thickLine(b, sx, sy, mx, my, 2, lmDk(yoRobe, dark))
		thickLine(b, mx, my, ex, ey, 2, lmDk(yoRobe, dark))
		ix, iy := int(math.Floor(ex)), int(math.Floor(ey))
		b.rect(ix-boolI(sign < 0), iy, 3, 1, lmDk(yoRobeD, dark)) // wide sleeve
		b.rect(ix, iy+1, 2, 2, lmDk(yoSkin, dark))
		b.set(ix+1, iy+2, lmDk(yoSkinD, dark))
		if stick { // the crooked walking stick down to the floor
			gx := ix
			if vw == vSide {
				gx = ix + 2
			} else if sign < 0 {
				gx = ix - 1
			}
			for y := iy - 2; y <= 42; y++ {
				dx := 0
				if y > iy+6 {
					dx = boolI(sign > 0 || vw == vSide)
				}
				c := yoStick
				if y%3 == 0 {
					c = yoStickL
				}
				b.set(gx+dx, y, lmDk(c, dark))
			}
			b.set(gx-1, iy-2, lmDk(yoStick, dark)) // the knob
		}
		return ix, iy + 1
	}

	lx, rx := float64(7+ox), float64(16+ox)
	switch vw {
	case vSide:
		cx := float64(11 + ox)
		arm(cx-0.5, p.LA, 1, 3, true, true)
		leg(10, p.LF, 1, 5, true)
		leg(12, p.RF, 1, 6, false)
		robe()
		head()
		x, y := arm(cx, p.RA, 1, 4, false, false)
		tiaHeld(b, s, p, vw, x, y, 4)
	case vBack:
		leg(13, p.LF, 1, 5, false)
		leg(9, p.RF, -1, 6, false)
		robe()
		arm(rx, p.LA, 1, 3, false, true)
		x, y := arm(lx, p.RA, -1, 4, false, false)
		head()
		tiaHeld(b, s, p, vw, x, y, 4)
	default:
		leg(9, p.LF, -1, 5, false)
		leg(13, p.RF, 1, 6, false)
		robe()
		head()
		arm(lx, p.LA, -1, 3, false, true)
		x, y := arm(rx, p.RA, 1, 4, false, false)
		tiaHeld(b, s, p, vw, x, y, 4)
	}
	return b
}
