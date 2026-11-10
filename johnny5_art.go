package main

import "math"

// ---------------------------------------------------------------------------
// Cortocircuito (slug "johnny5") has his own procedural drawing (genJohnny5), like SpongeBob:
// the generic body cannot show a robot on caterpillar tracks. Our own pixel-art take on the
// film robot: blue-grey body, a head of two big golden lenses with black pupils on a bridge,
// two metal horns sticking up over the eyes, a thin neck with cables, a modular blue torso with
// dark grey panels, segmented grey arms with three-finger claws and, instead of legs, two black
// tracks with wheels under a grey axle housing.
// Animation (visual only, hitbox and collisions do not change): the tracks roll (moving treads)
// while he walks or runs, the head sways a little, the body rocks on its suspension, and when he
// stands still he waves with the right claw raised.
// ---------------------------------------------------------------------------

func isJohnny5(s *CharSpec) bool { return s.Name == "johnny5" }

var (
	j5Blue   = hx(0x5a7aa6)
	j5BlueL  = hx(0x88a8cc)
	j5BlueD  = hx(0x3e5a84)
	j5BlueDD = hx(0x2a3c5a)
	j5Met    = hx(0x9ca4b0)
	j5MetL   = hx(0xccd2da)
	j5MetD   = hx(0x626a76)
	j5MetDD  = hx(0x3a3f48)
	j5Track  = hx(0x18191d)
	j5Tread  = hx(0x464a54)
	j5Wheel  = hx(0x7a828e)
	j5Rim    = hx(0xa87818)
	j5Lens   = hx(0xf2c832)
	j5LensL  = hx(0xfff2a0)
	j5Pupil  = hx(0x0e0e10)
	j5Cable  = hx(0xc04030)
)

// j5Moving: the animations where he travels (the tracks roll).
func j5Moving(a int) bool {
	switch a {
	case AnimWalk, AnimRun, AnimCarry, AnimPanic, AnimZombie, AnimRobot, AnimHeavy, AnimMarch,
		AnimLumber, AnimSneak, AnimGlide, AnimMoon, AnimSwagger, AnimStagger, AnimHop:
		return true
	}
	return false
}

func j5Dancing(a int) bool {
	return (a >= AnimDance0 && a <= AnimDance11) || a == AnimRobotDance || a == AnimHips || a == AnimPirouette || a == AnimCheer
}

// j5Salute: standing still with empty hands he raises the right claw.
func j5Salute(s *CharSpec, p Pose) bool {
	if p.Hold != HoldNone || s.Carry != HoldNone || p.Sit {
		return false
	}
	switch p.Anim {
	case AnimIdle, AnimWave, AnimTalk, AnimLaugh:
		return true
	}
	return false
}

func genJohnny5(s *CharSpec, vw int, p Pose) *pixBuf {
	b := newPix(charW, charH)
	if trackParts {
		b.pid = make([]uint8, charW*charH)
	}
	rnd := func(f float64) int { return int(math.Floor(f + 0.5)) }
	n := animFrameCount(p.Anim)
	if n < 1 {
		n = 1
	}
	ph := 2 * math.Pi * float64(p.F) / float64(n)
	moving, dancing := j5Moving(p.Anim), j5Dancing(p.Anim)
	roll := 0
	if moving {
		roll = p.F
		if p.Anim == AnimRun || p.Anim == AnimPanic {
			roll = p.F * 2
		}
	} else if dancing {
		roll = p.F % 2 // rocks back and forth on the spot
	}
	bodyDY := clampI(rnd(p.Bob*0.5), -2, 1)
	if p.Sit {
		bodyDY = 1
	}
	sway := clampI(rnd(p.Sway*0.5), -1, 1)
	headDX, headDY := 0, 0
	if moving {
		headDX = rnd(math.Sin(ph) * 0.9)
		if vw == vSide {
			headDX, headDY = 0, rnd((math.Sin(ph)-1)*0.5)
		}
	} else if dancing {
		headDX, headDY = rnd(math.Sin(ph)*1.4), -(p.F % 2)
	}
	if vw == vSide {
		headDX = 0
	}

	// --- tracks (under everything) ---
	b.cur = 5
	if vw == vSide {
		// far track: just its upper edge peeks out above the near one
		b.rect(6, 34, 12, 1, j5MetDD)
		// near track: a stadium with three wheels, treads running round it
		for y := 35; y <= 42; y++ {
			x0, x1 := 5, 18
			if y == 35 || y == 42 {
				x0, x1 = 6, 17
			}
			for x := x0; x <= x1; x++ {
				b.set(x, y, j5Track)
			}
		}
		for x := 6; x <= 17; x++ {
			if (x-roll)%3 == 0 { // top run moves forward (+x)
				b.set(x, 35, j5Tread)
			}
			if (x+roll)%3 == 0 { // bottom run moves backward
				b.set(x, 42, j5Tread)
			}
		}
		for _, wx := range []int{8, 12, 16} {
			b.ellipse(float64(wx)+0.5, 38.8, 1.7, 1.7, j5Wheel)
			b.set(wx, 38, j5MetL)
			hub := j5MetDD
			b.set(wx, 39, hub)
			// spoke turns with the roll
			sp := [][2]int{{-1, 0}, {0, -1}, {1, 0}, {0, 1}}[roll%4]
			b.set(wx+sp[0], 39+sp[1], j5MetD)
		}
	} else {
		track := func(x0 int, outer int) {
			for y := 35; y <= 42; y++ {
				for x := x0; x < x0+4; x++ {
					if y == 35 && (x == x0 || x == x0+3) {
						continue
					}
					b.set(x, y, j5Track)
				}
				if (y+roll)%3 == 0 {
					b.rect(x0+1, y, 2, 1, j5Tread)
				}
			}
			// wheel hubs showing on the outer flank
			b.set(outer, 37, j5Wheel)
			b.set(outer, 40, j5Wheel)
			b.set(outer, 38+(roll%2), j5MetD)
		}
		track(5, 5)
		track(15, 18)
	}

	// --- axle housing and the column up to the torso ---
	b.cur = 2
	cx := 12 + sway
	tTop := 24 + bodyDY
	tBot := 32 + bodyDY
	if vw == vSide {
		b.rect(8, 33, 8, 3, j5MetD)
		b.rect(8, 33, 8, 1, j5Met)
		b.rect(11, tBot, 3, 33-tBot, j5MetDD)
	} else {
		b.rect(9, 34, 6, 3, j5MetD)
		b.rect(9, 34, 6, 1, j5Met)
		b.set(9, 35, j5MetL)
		b.rect(cx-2, tBot, 4, 34-tBot+1, j5MetDD)
		b.set(cx-1, tBot+1, j5MetD)
	}

	// --- torso ---
	var x0, x1 int
	if vw == vSide {
		x0, x1 = 9+sway, 15+sway
	} else {
		x0, x1 = cx-5, cx+4
	}
	for y := tTop; y <= tBot; y++ {
		for x := x0; x <= x1; x++ {
			c := j5Blue
			switch {
			case y == tTop:
				c = j5BlueL
			case y == tBot:
				c = j5BlueDD
			case x == x0:
				c = j5BlueL
			case x == x1:
				c = j5BlueD
			}
			b.set(x, y, c)
		}
	}
	switch vw {
	case vFront:
		b.rect(x0+2, tTop+2, 6, 4, j5MetDD) // chest panel
		b.rect(x0+3, tTop+3, 4, 2, j5MetD)
		b.set(x0+3, tTop+3, hx(0x60e070)) // status lights
		b.set(x0+5, tTop+3, hx(0xf04a3a))
		if p.F%2 == 1 {
			b.set(x0+5, tTop+3, hx(0xffa040))
		}
		b.rect(x0+1, tTop+7, x1-x0-1, 1, j5MetD) // lower module seam
		b.set(x0+1, tTop+1, j5MetD)              // corner bolts
		b.set(x1-1, tTop+1, j5MetD)
	case vBack:
		for y := tTop + 2; y <= tBot-2; y += 2 { // vents
			b.rect(x0+2, y, x1-x0-3, 1, j5MetDD)
		}
		b.rect(x0+1, tTop+1, 2, 2, j5MetD)
	default:
		b.rect(x1-3, tTop+2, 3, 4, j5MetDD)
		b.set(x1-2, tTop+3, hx(0x60e070))
		b.rect(x0+1, tTop+7, x1-x0-1, 1, j5MetD)
	}

	// --- neck with cables: a short metal post with two black hoses ---
	b.cur = 1
	nTop := tTop - 4 + headDY
	nx := cx + headDX/2
	if vw == vSide {
		nx = 11 + sway
	}
	for y := nTop; y < tTop; y++ {
		b.set(nx-1, y, j5MetL)
		b.set(nx, y, j5Met)
		if vw != vSide {
			b.set(nx-2+((y+p.F)%2)*0, y, j5MetDD)
			b.set(nx+1, y, j5MetDD)
		} else {
			b.set(nx+1, y, j5MetDD)
		}
	}
	b.set(nx+1, nTop+1, j5Cable)       // a red wire peeking out
	b.rect(nx-2, tTop-1, 4, 1, j5MetD) // collar

	// --- head: two big lenses on a bridge, metal brows and horns ---
	hy := nTop - 4 // eye centre row
	hcx := cx + headDX
	horn := func(bx, by, tx, ty int) {
		b.line(bx, by, tx, ty, j5MetL)
		b.line(bx+1, by, tx+1, ty, j5MetD)
		b.set(tx, ty, j5MetL)
	}
	switch vw {
	case vSide:
		hb := 11 + sway
		b.rect(hb-3, hy-3, 7, 7, j5Met) // eye barrel
		b.rect(hb-3, hy-3, 7, 1, j5MetL)
		b.rect(hb-3, hy+3, 7, 1, j5MetD)
		b.rect(hb-4, hy-2, 1, 5, j5MetD)
		b.rect(hb-1, hy-1, 3, 1, j5MetD)  // barrel ring
		b.rect(hb-2, hy+4, 4, 1, j5MetDD) // jaw piece
		lx := float64(hb) + 5.2
		b.ellipse(lx, float64(hy)+0.5, 1.8, 3.4, j5Rim)
		b.ellipse(lx+0.2, float64(hy)+0.5, 1.1, 2.5, j5Lens)
		b.rect(hb+5, hy, 1, 2, j5Pupil)
		b.set(hb+5, hy-2, j5LensL)
		b.rect(hb-3, hy-4, 6, 1, j5MetD) // brow
		horn(hb-2, hy-4, hb-4, hy-9)
	case vBack:
		for _, ex := range []int{hcx - 3, hcx + 3} {
			b.ellipse(float64(ex)+0.5, float64(hy)+0.5, 3.1, 3.1, j5MetD)
			b.ellipse(float64(ex)+0.5, float64(hy)+0.5, 2.0, 2.0, j5Met)
			b.set(ex, hy, j5MetDD)
		}
		b.rect(hcx-1, hy-1, 3, 4, j5MetD)
		b.rect(hcx-1, hy+3, 3, 1, j5MetDD)
		b.rect(hcx-6, hy-4, 5, 1, j5MetD)
		b.rect(hcx+2, hy-4, 5, 1, j5MetD)
		horn(hcx-5, hy-4, hcx-7, hy-9)
		horn(hcx+4, hy-4, hcx+6, hy-9)
	default:
		b.rect(hcx-1, hy-2, 3, 5, j5Met)   // bridge between the eyes
		b.rect(hcx-1, hy+3, 3, 1, j5MetDD) // little jaw
		eye := func(ex int, pupDX int) {
			fx, fy := float64(ex)+0.5, float64(hy)+0.5
			b.ellipse(fx, fy, 3.1, 3.1, j5Rim)
			b.ellipse(fx, fy, 2.2, 2.2, j5Lens)
			b.rect(ex+pupDX, hy, 1, 2, j5Pupil)
			b.rect(ex+pupDX-1+boolI(pupDX < 0)*2, hy, 1, 2, j5Pupil)
			b.set(ex-2, hy-1, j5LensL)
			b.set(ex-1, hy-2, j5LensL)
		}
		pd := 0
		if dancing && p.F%3 == 1 {
			pd = 1
		}
		eye(hcx-3, pd)
		eye(hcx+3, pd)
		// metal brows over each lens and the horns rising from them
		b.rect(hcx-6, hy-4, 5, 1, j5MetD)
		b.rect(hcx+2, hy-4, 5, 1, j5MetD)
		b.rect(hcx-5, hy-4, 3, 1, j5Met)
		b.rect(hcx+3, hy-4, 3, 1, j5Met)
		horn(hcx-5, hy-4, hcx-7, hy-9)
		horn(hcx+4, hy-4, hcx+6, hy-9)
	}

	// --- arms with claws: thick upper arm, segmented forearm ---
	shY := tTop + 1
	arm := func(sx int, l Limb, sign int, dark, salute bool, id uint8) (int, int) {
		b.cur = id
		var hxp, hyp int
		switch {
		case salute:
			hxp, hyp = sx+sign*3+sign*(p.F%2), shY-8
		case vw == vSide:
			hxp = sx + clampI(rnd(l.Z*0.7), -5, 6)
			hyp = shY + clampI(rnd(l.Y*0.75), -9, 7)
		default:
			hxp = sx + sign*(1+clampI(rnd(l.X*0.5), -2, 3))
			hyp = shY + clampI(rnd(l.Y*0.75), -9, 7)
		}
		hxp = clampI(hxp, 2, charW-3)
		hyp = clampI(hyp, 3, 39)
		ex, ey := (sx+hxp)/2+sign, (shY+hyp)/2
		if vw == vSide {
			ex = (sx+hxp)/2 - 1
		}
		up, fo, edge := j5Met, j5MetL, j5MetD
		if dark {
			up, fo, edge = j5MetD, j5Met, j5MetDD
		}
		ox := 0
		if sign < 0 {
			ox = -1
		}
		thickLine(b, float64(sx+ox), float64(shY), float64(ex+ox), float64(ey), 2, up)
		b.line(sx+ox+1, shY, ex+ox+1, ey, edge)
		b.line(ex, ey, hxp, hyp, fo)
		b.line(ex+sign, ey, hxp+sign, hyp, edge)
		b.set(ex, ey, j5MetDD)                 // elbow joint
		b.set((ex+hxp)/2, (ey+hyp)/2, j5MetDD) // segment ring of the forearm
		b.rect(sx-1, shY-1, 3, 2, j5MetD)      // shoulder joint
		b.set(sx, shY-1, j5MetL)
		// three-finger claw: two fingers and a thumb, pointing away from the elbow
		dy := 1
		if hyp < ey {
			dy = -1
		}
		b.set(hxp, hyp, j5MetDD)
		b.set(hxp-1, hyp+dy, j5MetD)
		b.set(hxp+1, hyp+dy, j5MetD)
		b.set(hxp, hyp+2*dy, j5MetL)
		if salute {
			b.set(hxp-1, hyp+2*dy, j5MetD)
			b.set(hxp+1, hyp+2*dy, j5MetD)
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
		if kind != HoldNone {
			drawHeld(b, kind, s.CarryCol, vw, float64(hx0), float64(hy0))
		}
	}
	sal := j5Salute(s, p)
	switch vw {
	case vSide:
		arm(x0+2, p.LA, 1, true, false, 3)
		hx0, hy0 := arm(x0+3, p.RA, 1, false, false, 4)
		held(hx0, hy0, 4)
	case vBack:
		arm(x1+1, p.LA, 1, false, false, 3)
		hx0, hy0 := arm(x0-1, p.RA, -1, false, sal, 4)
		held(hx0, hy0, 4)
	default:
		arm(x0-1, p.LA, -1, false, false, 3)
		hx0, hy0 := arm(x1+1, p.RA, 1, false, sal, 4)
		held(hx0, hy0, 4)
	}
	return b
}
