package main

import (
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// Crash (slug "crash") has his own procedural drawing (genCrash), like SpongeBob or Johnny 5:
// the generic body cannot show a cartoon marsupial. Our own pixel-art parody of the bandicoot
// of the games: orange fur, a big head with a long cream muzzle, dark nose and a toothy grin,
// big green eyes under thick black brows, pointed ears dark inside, a brown-orange crest that
// spikes backwards, bare cream chest, wide shoulders on a short body, big brown gloves, blue
// jeans with a white waistband and chunky lilac trainers with white soles.
// Animation (visual only, hitbox and collisions do not change): arms and legs follow the pose
// (walk, run, dance...), the body bobs and sways, the head leans forward in profile and the grin
// opens when the pose opens the mouth. The dance spin lives in gait.go ("spin").
// Standing rows: crest/ears 3, skull 6..16, muzzle down to 19, torso 19..28, waistband 29,
// pelvis 30..31, legs 31..38, trainers 39..42.
// ---------------------------------------------------------------------------

func isCrash(s *CharSpec) bool { return s.Name == "crash" }

var (
	crFur    = hx(0xe8781c)
	crFurL   = hx(0xf8a044)
	crFurD   = hx(0xb4520c)
	crCream  = hx(0xf8e4a8)
	crCreamD = hx(0xdcbc78)
	crCrest  = hx(0xa8401a)
	crCrestL = hx(0xd0642a)
	crCrestD = hx(0x7a2c12)
	crNose   = hx(0x24160e)
	crNoseL  = hx(0x6a5a50)
	crEyeW   = hx(0xffffff)
	crIris   = hx(0x30a83a)
	crPupil  = hx(0x0c0c0c)
	crBrow   = hx(0x161010)
	crEarIn  = hx(0x5a2810)
	crMouth  = hx(0x6a1a12)
	crTongue = hx(0xd84a4a)
	crJeans  = hx(0x3a66c0)
	crJeansL = hx(0x5a88d8)
	crJeansD = hx(0x284a94)
	crBand   = hx(0xf2f2f2)
	crShoe   = hx(0x9a66d4)
	crShoeL  = hx(0xbc90ee)
	crShoeD  = hx(0x6c42a6)
	crSole   = hx(0xf6f6f6)
	crGlove  = hx(0x5e3418)
	crGloveL = hx(0x84502a)
	crGloveD = hx(0x3a1e0c)
)

// crRun: rows of one colour, as pairs of columns (x0, x1) relative to an anchor.
type crRun struct{ r, x0, x1 int }

func genCrash(s *CharSpec, vw int, p Pose) *pixBuf {
	b := newPix(charW, charH)
	if trackParts {
		b.pid = make([]uint8, charW*charH)
	}
	rnd := func(f float64) int { return int(math.Floor(f + 0.5)) }
	dk := func(c color.RGBA, dark bool) color.RGBA {
		if dark {
			return shade(c, 0.76)
		}
		return c
	}
	// the crest tip stands at row 3: never more than 2 rows higher (stays inside the box)
	bob := clampI(rnd(p.Bob), -2, 3)
	if p.Sit {
		bob = maxI(bob, 3)
	}
	ox := clampI(rnd(p.Sway*0.5), -2, 2)
	hox := ox // head shift
	if vw == vSide {
		ox = clampI(rnd(p.Lean*0.5), -1, 2)
		hox = clampI(ox+rnd(p.Lean*0.3+p.HeadZ*0.5), -1, 2) // snout tip and crest stay inside
	} else if p.HeadZ != 0 {
		hox += clampI(rnd(p.HeadZ*0.5), -1, 1)
	}
	hipY := 30 + bob
	ty0 := hipY - 11 // torso top
	hy := ty0 - 13   // skull top
	lift := func(l Limb) int {
		if p.Sit {
			return 0
		}
		return clampI(rnd(10-l.Y), 0, 6)
	}
	mir := func(x int) int { return 23 - x }

	// ---- legs and trainers ----
	shoeFront := func(x0, top int, back, dark bool) {
		for i := 0; i < 5; i++ {
			b.set(x0+i, top+1, dk(crShoe, dark))
			b.set(x0+i, top+2, dk(crShoeD, dark))
			b.set(x0+i, top+3, dk(crSole, dark))
		}
		for i := 1; i < 4; i++ {
			b.set(x0+i, top, dk(crShoeL, dark))
		}
		if back { // the heel tab sticks up at the back
			b.set(x0+2, top-1, dk(crShoeL, dark))
			b.set(x0+2, top, dk(crSole, dark))
		} else { // laces
			b.set(x0+2, top, dk(crSole, dark))
			b.set(x0+2, top+1, dk(crSole, dark))
		}
	}
	shoeSide := func(fx, top int, dark bool) {
		for x := fx; x <= fx+2; x++ {
			b.set(x, top, dk(crShoeL, dark))
		}
		for x := fx - 1; x <= fx+4; x++ {
			b.set(x, top+1, dk(crShoe, dark))
			b.set(x, top+2, dk(crShoeD, dark))
			b.set(x, top+3, dk(crSole, dark))
		}
		b.set(fx-1, top, dk(crShoeL, dark)) // heel tab
		b.set(fx+2, top+1, dk(crSole, dark))
		b.set(fx+3, top+1, dk(crSole, dark))
		b.set(fx+4, top+1, dk(crShoeD, dark))
	}
	leg := func(hx0 float64, l Limb, sign float64, id uint8, dark bool) {
		b.cur = id
		top := 39 - lift(l)
		var fx float64
		if vw == vSide {
			fx = hx0 + l.Z*0.7
			if p.Sit {
				fx = hx0 + 5
				top = hipY + 4
			}
		} else {
			fx = hx0 + sign*clampF((l.X-1.5)*0.8, -2, 2)
		}
		if vw == vSide {
			fx = clampF(fx, 2, 17) // the trainer stays inside the box
		}
		bx, by := sign, 0.0
		if vw == vSide {
			bx, by = 1, 0
		}
		mx, my, ex, ey := ik2(hx0, float64(hipY+1), fx, float64(top)-0.5, 3.85, bx, by)
		thickLine(b, hx0, float64(hipY+1), mx, my, 3, dk(crJeans, dark))
		thickLine(b, mx, my, ex, ey, 3, dk(crJeans, dark))
		// seams: outer side darker, a lighter crease in the middle
		edge := 0.0
		if vw != vSide && sign > 0 {
			edge = 2
		}
		if vw == vSide {
			edge = 0
		}
		thickLine(b, hx0+edge, float64(hipY+2), mx+edge, my, 1, dk(crJeansD, dark))
		thickLine(b, mx+edge, my, ex+edge, ey, 1, dk(crJeansD, dark))
		thickLine(b, hx0+1, float64(hipY+3), mx+1, my, 1, dk(crJeansL, dark))
		ix := int(math.Floor(fx))
		b.rect(ix, top-1, 3, 1, dk(crJeansD, dark)) // turned-up hem
		if vw == vSide {
			shoeSide(ix, top, dark)
		} else {
			shoeFront(ix-1, top, vw == vBack, dark)
		}
	}

	// ---- torso, waistband and pelvis ----
	torso := func() {
		b.cur = 2
		if vw == vSide {
			c := 11 + ox
			rows := []crRun{{0, -2, 2}, {1, -3, 3}, {2, -3, 3}, {3, -3, 3}, {4, -3, 3}, {5, -3, 2}, {6, -2, 2}, {7, -2, 2}, {8, -2, 2}, {9, -2, 2}}
			for _, r := range rows {
				for x := r.x0; x <= r.x1; x++ {
					col := crFur
					if x >= r.x1-1 && r.r >= 1 && r.r <= 8 {
						col = crCream // bare chest at the front
						if r.r == 4 && x == r.x1 {
							col = crCreamD
						}
					}
					if x == r.x0 {
						col = crFurD
					}
					if r.r == 0 && x < r.x1-1 {
						col = crFurL
					}
					b.set(c+x, ty0+r.r, col)
				}
			}
			b.rect(c-2, hipY-1, 5, 1, crBand)
			b.rect(c-2, hipY, 5, 2, crJeans)
			b.set(c-2, hipY, crJeansD)
			b.set(c+2, hipY+1, crJeansD)
			return
		}
		rows := []crRun{{0, 7, 16}, {1, 6, 17}, {2, 6, 17}, {3, 7, 16}, {4, 7, 16}, {5, 7, 16}, {6, 8, 15}, {7, 8, 15}, {8, 8, 15}, {9, 8, 15}}
		cream := map[int]int{1: 2, 2: 3, 3: 3, 4: 3, 5: 3, 6: 2, 7: 2, 8: 1} // half widths around 11.5
		for _, r := range rows {
			for x := r.x0; x <= r.x1; x++ {
				col := crFur
				if r.r == 0 {
					col = crFurL
				}
				if x == r.x0 || x == r.x1 {
					col = crFurD
				}
				if vw == vFront {
					if hw, ok := cream[r.r]; ok && x >= 12-hw && x <= 11+hw {
						col = crCream
						if r.r == 4 && (x == 9 || x == 10 || x == 13 || x == 14) {
							col = crCreamD // under the pecs
						}
					}
				} else if (x == 11 || x == 12) && r.r >= 2 && r.r <= 8 {
					col = shade(crFur, 0.9) // spine
				}
				b.set(x+ox, ty0+r.r, col)
			}
		}
		b.rect(8+ox, hipY-1, 8, 1, crBand)
		b.rect(7, hipY, 10, 2, crJeans)
		b.rect(7, hipY+1, 10, 1, crJeansD)
		if vw == vBack { // back pockets
			b.rect(8, hipY, 2, 1, crJeansL)
			b.rect(14, hipY, 2, 1, crJeansL)
		} else {
			b.set(11, hipY, crJeansD) // fly
			b.set(12, hipY, crJeansD)
		}
	}

	// ---- head ----
	put := func(runs []crRun, dx int, col color.RGBA, flip bool) {
		for _, r := range runs {
			for x := r.x0; x <= r.x1; x++ {
				xx := x
				if flip {
					xx = mir(x)
				}
				b.set(xx+dx, hy+r.r, col)
			}
		}
	}
	head := func() {
		b.cur = 1
		hd := hox
		open := p.Open || p.Anim == AnimLaugh || p.Anim == AnimCheer
		switch vw {
		case vSide:
			// ear behind the crest, then skull, crest spiking backwards, muzzle forward
			put([]crRun{{-3, 8, 8}, {-2, 7, 8}, {-1, 7, 9}, {0, 7, 9}}, hd, crFur, false)
			put([]crRun{{-2, 8, 8}, {-1, 8, 8}}, hd, crEarIn, false)
			b.ellipse(10.5+float64(hd), float64(hy)+5.5, 5.0, 5.6, crFur)
			b.rect(10+hd, hy+11, 3, 3, crFur) // neck
			put([]crRun{{1, 7, 9}, {2, 6, 8}}, hd, crFurL, false)
			// two spikes up front, one long spike swept back
			put([]crRun{{-3, 12, 12}, {-2, 10, 12}, {-1, 8, 13}, {0, 5, 12}, {1, 3, 7}, {2, 2, 3}}, hd, crCrest, false)
			put([]crRun{{-2, 9, 9}, {-1, 9, 11}, {0, 6, 8}}, hd, crCrestL, false)
			put([]crRun{{1, 6, 7}, {2, 2, 2}}, hd, crCrestD, false)
			b.ellipse(13.5+float64(hd), float64(hy)+10.5, 2.8, 3.0, crCream) // cheek and jaw
			put([]crRun{{6, 15, 16}, {7, 15, 18}, {8, 15, 18}}, hd, crFur, false)
			put([]crRun{{9, 12, 19}, {10, 12, 19}, {11, 12, 18}, {12, 12, 17}, {13, 12, 15}}, hd, crCream, false)
			put([]crRun{{7, 19, 20}, {8, 19, 20}}, hd, crNose, false)
			b.set(19+hd, hy+7, crNoseL)
			if open {
				put([]crRun{{10, 14, 19}}, hd, crEyeW, false)
				put([]crRun{{11, 14, 18}}, hd, crMouth, false)
				put([]crRun{{12, 15, 16}}, hd, crTongue, false)
				b.set(13+hd, hy+10, crMouth)
			} else {
				put([]crRun{{10, 14, 19}}, hd, crMouth, false)
				b.set(13+hd, hy+9, crMouth) // the corner goes up: a grin
				put([]crRun{{9, 16, 18}}, hd, crEyeW, false)
			}
			put([]crRun{{4, 13, 14}}, hd, crEyeW, false)
			b.set(13+hd, hy+5, crIris)
			b.set(14+hd, hy+5, crPupil)
			b.set(13+hd, hy+6, crEyeW)
			b.set(14+hd, hy+6, crIris)
			put([]crRun{{3, 12, 15}, {2, 12, 13}}, hd, crBrow, false)
		case vBack:
			for _, fl := range []bool{false, true} {
				put([]crRun{{-3, 5, 5}, {-2, 5, 6}, {-1, 5, 7}, {0, 5, 7}, {1, 6, 7}}, hd, crFur, fl)
				put([]crRun{{-2, 6, 6}, {-1, 6, 6}}, hd, crFurD, fl)
			}
			b.ellipse(12+float64(hd), float64(hy)+5.5, 6.1, 5.6, crFur)
			b.rect(10+hd, hy+11, 4, 2, crFurD) // nape
			// the crest runs down the back of the head
			put([]crRun{{-3, 10, 10}, {-3, 13, 13}, {-2, 10, 11}, {-2, 13, 14}, {-1, 9, 14}, {0, 9, 14}, {1, 9, 14}, {2, 10, 13}, {3, 11, 12}, {4, 11, 12}}, hd, crCrest, false)
			put([]crRun{{-1, 9, 10}, {0, 9, 9}, {1, 9, 9}}, hd, crCrestL, false)
			put([]crRun{{2, 13, 13}, {3, 12, 12}, {4, 12, 12}}, hd, crCrestD, false)
			put([]crRun{{6, 6, 6}, {7, 6, 6}}, hd, crFurD, false)
			put([]crRun{{6, 17, 17}, {7, 17, 17}}, hd, crFurD, false)
		default:
			for _, fl := range []bool{false, true} {
				put([]crRun{{-3, 5, 5}, {-2, 5, 6}, {-1, 5, 7}, {0, 5, 7}, {1, 6, 7}}, hd, crFur, fl)
				put([]crRun{{-1, 6, 6}, {0, 6, 6}}, hd, crEarIn, fl)
			}
			b.ellipse(12+float64(hd), float64(hy)+5.5, 6.1, 5.6, crFur)
			put([]crRun{{1, 8, 9}, {2, 7, 8}}, hd, crFurL, false)
			b.ellipse(12+float64(hd), float64(hy)+10.5, 4.6, 3.2, crCream) // muzzle and chin
			put([]crRun{{-3, 10, 10}, {-3, 13, 13}, {-2, 10, 11}, {-2, 13, 14}, {-1, 9, 14}, {0, 9, 14}, {1, 10, 13}}, hd, crCrest, false)
			put([]crRun{{-1, 9, 10}, {0, 9, 9}}, hd, crCrestL, false)
			put([]crRun{{1, 13, 13}}, hd, crCrestD, false)
			for _, fl := range []bool{false, true} {
				put([]crRun{{3, 7, 10}, {2, 7, 8}}, hd, crBrow, fl)
				put([]crRun{{4, 8, 10}, {5, 8, 8}, {6, 8, 8}}, hd, crEyeW, fl)
				put([]crRun{{5, 9, 9}, {6, 9, 10}}, hd, crIris, fl)
				put([]crRun{{5, 10, 10}}, hd, crPupil, fl)
			}
			put([]crRun{{8, 10, 13}, {9, 11, 12}}, hd, crNose, false)
			b.set(11+hd, hy+8, crNoseL)
			if open {
				put([]crRun{{10, 8, 8}, {10, 15, 15}, {11, 9, 9}, {11, 14, 14}, {12, 10, 13}}, hd, crMouth, false)
				put([]crRun{{11, 10, 13}}, hd, crEyeW, false)
				put([]crRun{{12, 11, 12}}, hd, crTongue, false)
				put([]crRun{{13, 11, 12}}, hd, crMouth, false)
			} else {
				put([]crRun{{10, 8, 8}, {10, 15, 15}, {11, 9, 9}, {11, 14, 14}, {12, 10, 13}}, hd, crMouth, false)
				put([]crRun{{11, 10, 13}}, hd, crEyeW, false)
			}
		}
	}

	// ---- arms with big gloves ----
	arm := func(sx float64, l Limb, sign float64, id uint8, dark bool) (int, int) {
		b.cur = id
		sy := float64(ty0 + 2)
		var tx, ty float64
		if vw == vSide {
			tx, ty = l.Z, l.Y
		} else {
			tx, ty = sign*(l.X-0.7)*0.8, l.Y+l.Z*0.1
		}
		bx, by := sign*0.25, 0.9
		if vw == vSide {
			bx, by = -1, 0
		}
		// the glove (3x3) never leaves the sprite box
		hxT := clampF(sx+tx, 2, 19)
		if vw == vSide {
			hxT = clampF(sx+tx, 1, 19)
		}
		mx, my, ex, ey := ik2(sx, sy, hxT, math.Max(sy+ty, 1), 4.1, bx, by)
		ex, ey = clampF(ex, 2, 19.9), math.Max(ey, 1)
		mx, my = clampF(mx, 1, 21.9), math.Max(my, 1)
		if vw == vSide {
			ex = clampF(ex, 1, 19.9)
		}
		thickLine(b, sx, sy, mx, my, 2, dk(crFur, dark))
		thickLine(b, mx, my, ex, ey, 2, dk(crFur, dark))
		thickLine(b, sx, sy, sx, sy+1, 2, dk(crFurL, dark)) // shoulder
		ix, iy := int(math.Floor(ex)), int(math.Floor(ey))
		gx := ix
		if vw != vSide && sign < 0 {
			gx = ix - 1
		}
		b.rect(gx, iy, 3, 3, dk(crGlove, dark))
		b.rect(gx, iy, 3, 1, dk(crGloveL, dark)) // cuff
		b.rect(gx, iy+2, 3, 1, dk(crGloveD, dark))
		b.set(gx+1, iy+1, dk(crGloveL, dark)) // knuckles shine
		return ix, iy + 1
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

	switch vw {
	case vSide:
		cx := float64(11 + ox)
		arm(cx-0.5, p.LA, 1, 3, true)
		leg(9.5, p.LF, 1, 5, true)
		leg(10.5, p.RF, 1, 6, false)
		torso()
		head()
		hx0, hy0 := arm(cx, p.RA, 1, 4, false)
		held(hx0, hy0, 4)
	case vBack:
		leg(14, p.LF, 1, 5, false)
		leg(7, p.RF, -1, 6, false)
		torso()
		head()
		arm(float64(17+ox), p.LA, 1, 3, false)
		hx0, hy0 := arm(float64(5+ox), p.RA, -1, 4, false)
		held(hx0, hy0, 4)
	default:
		leg(7, p.LF, -1, 5, false)
		leg(14, p.RF, 1, 6, false)
		torso()
		head()
		arm(float64(5+ox), p.LA, -1, 3, false)
		hx0, hy0 := arm(float64(17+ox), p.RA, 1, 4, false)
		held(hx0, hy0, 4)
	}
	return b
}
