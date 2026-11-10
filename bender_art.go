package main

import (
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// Bender (slug "bender") has his own procedural drawing (genBenderArt), like Johnny 5: the
// generic robot body cannot show the bending robot of the cartoon. Our own pixel-art parody:
// a blue-grey metal cylinder of a head with a dome on top and a single antenna (a stick with a
// ball), a black visor sticking out with small white eyes and square pupils, a white grille
// mouth with square teeth, a barrel of a torso with a hatch in the belly, thin flexible arms
// with round pincer hands (a cigar in the left one, smoking; the beer goes in the right one as
// the held object), thin legs and big round feet.
// Animation (visual only, hitbox and collisions do not change): arms and legs follow the pose,
// the body bobs, the cigar smoke drifts and the mouth opens when the pose opens it.
// Standing rows: antenna ball 2..3, head 7..18, torso 19..29, hips 30..31, legs 32..39,
// feet 40..42.
// ---------------------------------------------------------------------------

func isBenderArt(s *CharSpec) bool { return s.Name == "bender" }

var (
	bdMet    = hx(0x90a0b4)
	bdMetL   = hx(0xbccadb)
	bdMetLL  = hx(0xdce6f0)
	bdMetD   = hx(0x66758a)
	bdMetDD  = hx(0x445064)
	bdVisor  = hx(0x121418)
	bdEye    = hx(0xfaf8e8)
	bdPupil  = hx(0x0a0a0a)
	bdTeeth  = hx(0xeef2f0)
	bdGrille = hx(0x6a7280)
	bdCigar  = hx(0x7a4420)
	bdCigarD = hx(0x4e2a12)
	bdEmber  = hx(0xff6a20)
	bdSmoke  = hx(0xd6d8de)
	bdFoot   = hx(0x525e70)
	bdFootL  = hx(0x7a889c)
)

func genBenderArt(s *CharSpec, vw int, p Pose) *pixBuf {
	b := newPix(charW, charH)
	if trackParts {
		b.pid = make([]uint8, charW*charH)
	}
	dk := func(c color.RGBA, dark bool) color.RGBA {
		if dark {
			return shade(c, 0.76)
		}
		return c
	}
	bob := clampI(tiaRnd(p.Bob), -1, 3)
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
	hipY := 30 + bob
	ty0 := hipY - 11 // top of the barrel
	hy := ty0 - 12   // top of the dome
	open := p.Open || p.Anim == AnimLaugh || p.Anim == AnimCheer || p.Anim == AnimPanic
	L, R := 11+hd, 12+hd

	// ---- legs: thin rods and big round feet ----
	leg := func(hx0 float64, l Limb, sign float64, id uint8, dark bool) {
		b.cur = id
		mx, my, ex, ey, top := tiaLegIK(vw, p, hx0, float64(hipY+2), l, sign, 0.9, 4.2, 40)
		thickLine(b, hx0, float64(hipY+2), mx, my, 2, dk(bdMet, dark))
		thickLine(b, mx, my, ex, ey, 2, dk(bdMet, dark))
		thickLine(b, hx0, float64(hipY+2), mx, my, 1, dk(bdMetL, dark))
		thickLine(b, mx, my, ex, ey, 1, dk(bdMetL, dark))
		b.set(int(math.Floor(mx)), int(math.Floor(my)), dk(bdMetDD, dark)) // knee ring
		ix := int(math.Floor(ex))
		if vw == vSide {
			b.ellipse(float64(ix)+1.5, float64(top)+1.5, 3.0, 1.6, dk(bdFoot, dark))
			b.rect(ix, top, 3, 1, dk(bdFootL, dark))
			return
		}
		b.ellipse(float64(ix)+1, float64(top)+1.5, 2.4, 1.6, dk(bdFoot, dark))
		b.rect(ix, top, 2, 1, dk(bdFootL, dark))
	}

	// ---- the barrel ----
	torso := func() {
		b.cur = 2
		x0, x1 := 6+ox, 17+ox
		if vw == vSide {
			x0, x1 = 8+ox, 16+ox
		}
		for r := 0; r <= hipY-ty0-1; r++ {
			a, z := x0, x1
			if r == 0 || r == hipY-ty0-1 {
				a, z = x0+1, x1-1
			}
			for x := a; x <= z; x++ {
				col := bdMet
				switch {
				case x <= x0+1:
					col = bdMetL
				case x >= x1-1:
					col = bdMetD
				case x == x0+2:
					col = bdMetLL
				}
				if r == 0 {
					col = bdMetLL
				}
				if r == hipY-ty0-1 {
					col = bdMetD
				}
				b.set(x, ty0+r, col)
			}
		}
		switch vw {
		case vFront: // the belly hatch
			hx0, hy0 := 9+ox, ty0+3
			b.rect(hx0, hy0, 6, 6, bdMetDD)
			b.rect(hx0+1, hy0+1, 4, 4, bdMetL)
			b.rect(hx0+1, hy0+1, 4, 1, bdMetLL)
			b.set(hx0+4, hy0+3, bdMetDD) // handle
			b.set(hx0+1, hy0+4, bdMetD)
		case vSide:
			b.rect(x1-2, ty0+3, 2, 6, bdMetDD) // the hatch seen edge on
			b.rect(x1-1, ty0+4, 1, 4, bdMetL)
			b.set(x0+3, ty0+2, bdMetDD) // rivets
			b.set(x0+3, ty0+8, bdMetDD)
		default:
			b.rect(11+ox, ty0+1, 1, hipY-ty0-3, bdMetD) // back seam
			for _, y := range []int{ty0 + 2, ty0 + 5, ty0 + 8} {
				b.set(8+ox, y, bdMetDD)
				b.set(15+ox, y, bdMetDD)
			}
		}
		// hips: a narrower dark ring
		if vw == vSide {
			b.rect(9+ox, hipY, 6, 2, bdMetD)
			b.rect(9+ox, hipY, 6, 1, bdMet)
		} else {
			b.rect(8+ox, hipY, 8, 2, bdMetD)
			b.rect(8+ox, hipY, 8, 1, bdMet)
		}
	}

	// ---- the head ----
	head := func() {
		b.cur = 1
		// antenna: a stick with a ball
		ax := R
		if vw == vSide {
			ax = L
		}
		b.rect(ax-1, hy-1, 2, 1, bdMetD)
		b.rect(ax, hy-3, 1, 2, bdMetL)
		b.rect(ax-1, hy-5, 2, 2, bdMet)
		b.set(ax-1, hy-5, bdMetLL)
		var x0, x1 int
		if vw == vSide {
			x0, x1 = L-3, R+2
		} else {
			x0, x1 = L-3, R+3
		}
		for y := hy; y <= hy+11; y++ {
			a, z := x0, x1
			switch y - hy {
			case 0:
				a, z = x0+2, x1-2
			case 1:
				a, z = x0+1, x1-1
			}
			for x := a; x <= z; x++ {
				col := bdMet
				switch {
				case x == a:
					col = bdMetL
				case x == z:
					col = bdMetDD
				case x == z-1:
					col = bdMetD
				case x == a+1:
					col = bdMetLL
				}
				if y == hy {
					col = bdMetL
				}
				b.set(x, y, col)
			}
		}
		b.rect(x0+1, hy+11, x1-x0-1, 1, bdMetDD) // neck ring under the head
		switch vw {
		case vSide:
			// the visor sticks out forward with one eye showing
			b.rect(x1-1, hy+3, 4, 3, bdVisor)
			b.rect(x1, hy+3, 2, 2, bdEye)
			b.set(x1+1, hy+4, bdPupil)
			b.rect(x1-1, hy+7, 3, 3, bdGrille)
			b.rect(x1, hy+7, 2, 3, bdTeeth)
			b.set(x1, hy+8, bdGrille)
			b.set(x1+1, hy+8, bdGrille)
			if open {
				b.rect(x1, hy+8, 2, 1, bdVisor)
			}
		case vBack:
			b.rect(L, hy+3, 2, 6, bdMetD) // back seam
			b.set(x0+1, hy+6, bdMetDD)
			b.set(x1-1, hy+6, bdMetDD)
		default:
			// visor: a black band a pixel wider than the head
			b.rect(x0, hy+3, x1-x0+1, 3, bdVisor)
			b.set(x0-1, hy+4, bdVisor)
			b.set(x1+1, hy+4, bdVisor)
			b.rect(L-2, hy+3, 2, 2, bdEye)
			b.rect(R+1, hy+3, 2, 2, bdEye)
			pd := 0
			if p.Anim == AnimTalk || p.Anim == AnimLaugh {
				pd = p.F % 2
			}
			b.set(L-1+pd, hy+4, bdPupil)
			b.set(R+1+pd, hy+4, bdPupil)
			// grille mouth with square teeth
			b.rect(L-3, hy+7, 8, 4, bdGrille)
			b.rect(L-2, hy+7, 6, 3, bdTeeth)
			b.rect(L-2, hy+8, 6, 1, bdGrille)
			b.set(L-1+1, hy+7, bdGrille)
			b.set(L-1+1, hy+9, bdGrille)
			b.set(R+1, hy+7, bdGrille)
			b.set(R+1, hy+9, bdGrille)
			if open {
				b.rect(L-2, hy+8, 6, 1, bdVisor)
			}
		}
	}

	// ---- arms: thin flexible tubes and round pincer hands ----
	arm := func(sx float64, l Limb, sign float64, id uint8, dark, cigar bool) (int, int) {
		b.cur = id
		sy := float64(ty0 + 2)
		mx, my, ex, ey := tiaArmIK(vw, sx, sy, l, sign, 1.0, 4.6)
		thickLine(b, sx, sy, mx, my, 2, dk(bdMet, dark))
		thickLine(b, mx, my, ex, ey, 2, dk(bdMet, dark))
		thickLine(b, sx, sy, mx, my, 1, dk(bdMetL, dark))
		thickLine(b, mx, my, ex, ey, 1, dk(bdMetL, dark))
		b.set(int(math.Floor((sx+mx)/2)), int(math.Floor((sy+my)/2)), dk(bdMetD, dark)) // tube rings
		b.set(int(math.Floor(mx)), int(math.Floor(my)), dk(bdMetD, dark))
		b.set(int(math.Floor((mx+ex)/2)), int(math.Floor((my+ey)/2)), dk(bdMetD, dark))
		b.rect(int(sx)-1, int(sy)-1, 3, 2, dk(bdMetD, dark)) // shoulder socket
		ix, iy := int(math.Floor(ex)), int(math.Floor(ey))
		gx := ix
		if vw != vSide && sign < 0 {
			gx = ix - 1
		}
		b.ellipse(float64(gx)+1.5, float64(iy)+1.5, 1.6, 1.6, dk(bdMetL, dark))
		b.set(gx+1, iy+2, dk(bdMetDD, dark)) // the pincer split
		b.set(gx+1, iy+1, dk(bdMetLL, dark))
		if cigar {
			dir := int(sign)
			if vw == vSide {
				dir = 1
			}
			if e := gx + 1 + dir*4; e < 2 || e > 21 { // near the edge of the box: point it inwards
				dir = -dir
			}
			cx := gx + 1 + dir*2
			b.set(gx+1+dir, iy+1, bdCigar)
			b.set(cx, iy+1, bdCigar)
			b.set(cx, iy+2, bdCigarD)
			b.set(cx+dir, iy+1, bdEmber)
			// smoke drifting up
			sy := iy - 1 - p.F%2
			b.set(cx+dir, sy, bdSmoke)
			b.set(cx+dir*(1+p.F%2), sy-1, bdSmoke)
		}
		return ix, iy + 1
	}

	switch vw {
	case vSide:
		cx := float64(11 + ox)
		arm(cx-0.5, p.LA, 1, 3, true, true)
		leg(10, p.LF, 1, 5, true)
		leg(11, p.RF, 1, 6, false)
		torso()
		head()
		x, y := arm(cx+0.5, p.RA, 1, 4, false, false)
		tiaHeld(b, s, p, vw, x, y, 4)
	case vBack:
		leg(13, p.LF, 1, 5, false)
		leg(9, p.RF, -1, 6, false)
		torso()
		head()
		arm(float64(18+ox), p.LA, 1, 3, false, true)
		x, y := arm(float64(5+ox), p.RA, -1, 4, false, false)
		tiaHeld(b, s, p, vw, x, y, 4)
	default:
		leg(9, p.LF, -1, 5, false)
		leg(13, p.RF, 1, 6, false)
		torso()
		head()
		arm(float64(5+ox), p.LA, -1, 3, false, true)
		x, y := arm(float64(18+ox), p.RA, 1, 4, false, false)
		tiaHeld(b, s, p, vw, x, y, 4)
	}
	return b
}
