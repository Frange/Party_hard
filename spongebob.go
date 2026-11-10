package main

import "image/color"

// SpongeBob has his own drawing (genSponge) because the generic body cannot show
// his pores, giant eyes and toothy smile. Everything is procedural pixel art that
// copies the user's reference drawing: black outline, yellow square body with dark
// green pores, huge eyes with a cyan glint, red cheeks, big teeth, red-pink shorts,
// thin white legs, reddish shoes and little light-blue arms.
// Cheap animation: squash and stretch from Pose.Bob, blink while landing, a wider
// grin in the air or when Pose.Open, and arms that follow the pose limbs.

func isSponge(s *CharSpec) bool { return s.Name == "bob_esponja" }

var (
	spBlack = hx(0x14100c)
	spYel   = hx(0xf8d824)
	spYelD  = hx(0xd8b418)
	spYelL  = hx(0xfff25a)
	spPore  = hx(0x5a8a1c)
	spPants = hx(0xd2504a)
	spPantD = hx(0xa03a3c)
	spLeg   = hx(0xf4f4f4)
	spLegD  = hx(0xc4c8d0)
	spShoe  = hx(0xd04c48)
	spShoeD = hx(0x9c3030)
	spArm   = hx(0x6ad6e6)
	spCyan  = hx(0x3cc8d8)
	spCheek = hx(0xe02828)
	spTongu = hx(0xc83850)
)

const spBaseH = 18 // yellow rows of the body at rest (outline included)

// spongeBlink: the eyes squeeze shut on the landing frame of the hop and on the second idle frame.
func spongeBlink(p Pose) bool {
	return (p.Anim == AnimHop && p.F == 3) || (p.Anim == AnimIdle && p.F == 1)
}

func genSponge(s *CharSpec, vw int, p Pose) *pixBuf {
	b := newPix(charW, charH)
	if trackParts {
		b.pid = make([]uint8, charW*charH)
	}
	round := func(f float64) int {
		if f < 0 {
			return -int(-f + 0.5)
		}
		return int(f + 0.5)
	}
	k := clampI(round(-p.Bob), -1, 2) // >0 stretch (up in the jump), <0 squash (landing)
	H := spBaseH + k
	W := 15
	if vw == vSide {
		W = 11
	}
	if k < 0 {
		W += 2
	}
	x0 := 12 - W/2 + clampI(round(p.Sway*0.5), -2, 1)
	// legs: foot lift from the pose
	lift := func(l Limb) int { return clampI(round(10-l.Y), 0, 6) }
	lL, lR := lift(p.LF), lift(p.RF)
	if p.Sit {
		lL, lR = 0, 0
	}
	footL, footR := 41-lL, 41-lR
	pbot := (footL+footR)/2 - 9
	if p.Sit {
		pbot += 3
	}
	yt := pbot - 4 - H
	if yt < 8 {
		pbot += 8 - yt
		yt = 8
	}
	bodyBottom := yt + H + 4
	_ = bodyBottom

	// legs and shoes first (behind the body)
	leg := func(hipX, footY, dx int, id uint8, dark bool) {
		b.cur = id
		footTop := footY - 4
		n := footTop - pbot
		w := spLeg
		if dark {
			w = spLegD
		}
		for i := 0; i <= n; i++ {
			x := hipX
			if n > 0 {
				x += dx * i / n
			}
			y := pbot + i
			b.set(x-1, y, spBlack)
			b.set(x, y, w)
			b.set(x+1, y, w)
			b.set(x+2, y, spBlack)
		}
	}
	shoe := func(sx, footY, w int, id uint8) {
		b.cur = id
		t := footY - 4
		for i := 1; i < w-1; i++ {
			b.set(sx+i, t, spBlack)
			b.set(sx+i, t+3, spBlack)
			b.set(sx+i, t+1, spShoe)
			b.set(sx+i, t+2, spShoeD)
		}
		for _, y := range []int{t + 1, t + 2} {
			b.set(sx, y, spBlack)
			b.set(sx+w-1, y, spBlack)
		}
		b.set(sx+1, t+1, hx(0xea6c66))
	}
	switch vw {
	case vSide:
		fxL, fxR := 10+round(p.LF.Z*0.6), 12+round(p.RF.Z*0.6)
		leg(10, footL, fxL-10, 5, true)
		shoe(fxL-2, footL, 7, 5)
		leg(12, footR, fxR-12, 6, false)
		shoe(fxR-2, footR, 7, 6)
	default:
		oL, oR := clampI(round((p.LF.X-1.2)*0.5), -2, 2), clampI(round((p.RF.X-1.2)*0.5), -2, 2)
		leg(8, footL, -oL, 5, false)
		shoe(5-oL, footL, 6, 5)
		leg(15, footR, oR, 6, false)
		shoe(14+oR, footR, 6, 6)
	}

	// body: black silhouette, yellow inside, shorts underneath
	b.cur = 2
	b.rect(x0, yt+1, W, H+3, spBlack)
	b.rect(x0+1, yt, W-2, 1, spBlack)
	b.rect(x0+1, yt+1, W-2, H-2, spYel)
	b.rect(x0+1, yt+1, W-2, 1, spYelL)
	b.rect(x0+W-2, yt+2, 1, H-3, spYelD)
	b.rect(x0+2, yt+H-2, W-3, 1, spYelD)
	b.rect(x0+1, yt+H, W-2, 3, spPants)
	b.rect(x0+1, yt+H, W-2, 1, spPantD)
	b.rect(x0+1, yt+H, 1, 1, hx(0xea7480))
	b.rect(x0, yt+H+3, W, 1, spBlack)
	b.rect(x0+1, yt+H+3, W-2, 1, spBlack)
	// the whole body outline needs its corners trimmed so it reads as a sponge, not a box
	for _, c := range [][2]int{{x0, yt + 1}, {x0 + W - 1, yt + 1}, {x0, yt + H + 3}, {x0 + W - 1, yt + H + 3}} {
		b.set(c[0], c[1], color.RGBA{})
	}
	// pores (relative to the top-left of the body, or to its bottom for the last ones)
	pore := func(u, v, w, h int, fromBottom bool) {
		if fromBottom {
			v = H - 2 - v
		}
		b.rect(x0+u, yt+v, w, h, spPore)
	}
	switch vw {
	case vBack:
		pore(2, 2, 2, 2, false)
		pore(8, 1, 1, 1, false)
		pore(9, 4, 2, 2, false)
		pore(1, 7, 1, 2, false)
		pore(5, 8, 2, 1, false)
		pore(W-3, 8, 1, 2, false)
		pore(2, 2, 2, 1, true)
		pore(W-4, 1, 1, 1, true)
		pore(7, 0, 2, 1, true)
	case vSide:
		pore(1, 1, 2, 1, false)
		pore(2, 4, 1, 2, false)
		pore(4, 8, 2, 1, false)
		pore(1, 1, 2, 1, true)
		pore(5, 0, 1, 1, true)
		pore(8, 2, 1, 1, false)
	default:
		pore(2, 1, 1, 1, false)
		pore(8, 1, 1, 1, false)
		pore(W-4, 1, 1, 1, false)
		pore(1, 3, 1, 2, false)
		pore(W-2, 5, 1, 2, false)
		pore(1, 2, 2, 1, true)
		pore(5, 0, 1, 1, true)
		pore(W-4, 1, 2, 1, true)
		pore(9, 0, 1, 1, true)
	}

	// face
	blink := spongeBlink(p)
	open := p.Open || (p.Anim == AnimHop && (p.F == 1 || p.F == 2))
	eyeY := float64(yt) + 5
	eye := func(cxe float64, pupX int, side bool) {
		b.cur = 1
		if blink {
			ex := int(cxe)
			b.rect(ex-2, int(eyeY)+1, 5, 1, spBlack)
			b.set(ex-2, int(eyeY), spBlack)
			b.set(ex+2, int(eyeY), spBlack)
			return
		}
		b.ellipse(cxe, eyeY, 2.8, 3.3, spBlack)
		b.ellipse(cxe, eyeY, 1.9, 2.5, hx(0xffffff))
		b.rect(pupX, int(eyeY)-1, 2, 3, spBlack)
		b.set(pupX-1+boolI(side), int(eyeY)+1, spCyan)
		b.set(pupX+boolI(side), int(eyeY)-1, hx(0xd8f4f8))
	}
	mouth := func(mx0, mx1, my int, teeth []int) {
		b.cur = 1
		rows := 2
		if open {
			rows = 4
		}
		b.rect(mx0, my, mx1-mx0+1, rows+2, spBlack)
		b.rect(mx0+1, my+1, mx1-mx0-1, 2, hx(0xf8f8f8))
		if open {
			b.rect(mx0+1, my+3, mx1-mx0-1, 2, spTongu)
			b.set(mx0+1, my+3, spBlack)
			b.set(mx1-1, my+3, spBlack)
		}
		for _, tx := range teeth {
			b.set(tx, my+1, spBlack)
			b.set(tx, my+2, spBlack)
		}
	}
	switch vw {
	case vFront:
		eye(9.2, 10, false)
		eye(15.8, 14, true)
		// nose, cheeks, smile
		b.cur = 1
		b.rect(12, yt+8, 1, 2, hx(0xd6a818))
		b.set(12, yt+9, hx(0xb88810))
		b.rect(x0+1, yt+9, 2, 2, hx(0xe86a30))
		b.rect(x0+W-3, yt+8, 2, 2, spCheek)
		mouth(7, 17, yt+10, []int{12})
		b.set(6, yt+10, spBlack)
		b.set(18, yt+10, spBlack)
	case vSide:
		xf := x0 + W - 1
		b.cur = 1
		b.ellipse(float64(xf)-3, eyeY, 2.8, 3.3, spBlack)
		if blink {
			b.rect(xf-5, int(eyeY)+1, 5, 1, spBlack)
		} else {
			b.ellipse(float64(xf)-3, eyeY, 1.9, 2.5, hx(0xffffff))
			b.rect(xf-3, int(eyeY)-1, 2, 3, spBlack)
			b.set(xf-3, int(eyeY)+1, spCyan)
		}
		b.rect(xf+1, yt+7, 2, 3, spBlack)
		b.rect(xf+1, yt+8, 1, 1, hx(0xd6a818))
		b.set(xf-4, yt+9, spCheek)
		mouth(xf-6, xf, yt+10, []int{xf - 3})
	}

	// own touches: a party cone hat tilted on the head and a flower lei over the shorts
	b.cur = 1
	if vw != vSide {
		hatBase := yt
		for i := 0; i < 6; i++ {
			c := hx(0xe04aa0)
			if i%2 == 1 {
				c = hx(0x40c0e8)
			}
			w := 7 - i
			b.rect(x0+4+(i+1)/2+boolI(vw == vBack)*0, hatBase-1-i, w, 1, c)
		}
		b.set(x0+7, hatBase-7, hx(0xf8f040))
		b.set(x0+6, hatBase-7, hx(0xf8f040))
	} else {
		for i := 0; i < 6; i++ {
			c := hx(0xe04aa0)
			if i%2 == 1 {
				c = hx(0x40c0e8)
			}
			b.rect(x0+2+(i+1)/2, yt-1-i, 6-i, 1, c)
		}
		b.set(x0+4, yt-7, hx(0xf8f040))
	}
	b.cur = 2
	for i := 0; i < W-2; i++ {
		c := []color.RGBA{hx(0xf86ab0), hx(0xf8f8f8), hx(0xff9a30)}[i%3]
		b.set(x0+1+i, yt+H-1, c)
	}
	// arms (light blue stubs that follow the pose limbs)
	arm := func(sx, sy int, l Limb, sign int, id uint8) (int, int) {
		b.cur = id
		var hxp, hyp int
		if vw == vSide {
			hxp = sx + round(l.Z*0.6)
		} else {
			hxp = sx + sign*(2+round(l.X*0.4))
		}
		hxp = clampI(hxp, 2, charW-5)
		hyp = sy + clampI(round((l.Y-5)*0.6), -5, 5)
		n := maxI(absI(hxp-sx), absI(hyp-sy)) + 1
		for pass := 0; pass < 2; pass++ {
			for i := 0; i <= n; i++ {
				x := sx + (hxp-sx)*i/n
				y := sy + (hyp-sy)*i/n
				x = clampI(x, 1, charW-5)
				if pass == 0 {
					b.rect(x-1, y-1, 4, 4, spBlack)
				} else {
					b.rect(x, y, 2, 2, spArm)
				}
			}
		}
		return hxp, hyp
	}
	held := func(hx0, hy0 int, id uint8) {
		b.cur = id
		kind := p.Hold
		if kind == HoldNone {
			kind = s.Carry
		}
		if s.Weapon != 0 && (kind == HoldKnife || kind == HoldNone) {
			kind = s.Weapon
		}
		drawHeld(b, kind, s.CarryCol, vw, float64(hx0), float64(hy0))
	}
	sy := yt + H - 6
	switch vw {
	case vSide:
		arm(10, sy+2, p.LA, 1, 3)
		hx0, hy0 := arm(9, sy+2, p.RA, 1, 4)
		held(hx0, hy0, 4)
	default:
		arm(x0-1, sy, p.LA, -1, 3)
		hx0, hy0 := arm(x0+W-1, sy, p.RA, 1, 4)
		held(hx0, hy0, 4)
	}
	return b
}

func boolI(v bool) int {
	if v {
		return 1
	}
	return 0
}
