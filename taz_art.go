package main

import (
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// The Tasmanian Devil (slug "taz") has his own procedural drawing (genTaz), like Crash or
// Johnny 5: the generic body cannot show the cartoon beast. Our own pixel-art take on the
// classic cartoon look: chocolate-brown fur, a crest of spiky hair on top, pointed ears light pink
// inside, small white eyes under angry brows, a big round dark-brown nose on top of a HUGE open
// mouth full of white teeth and fangs with a red tongue hanging out, cream face, muzzle and pot
// belly, long arms
// with three-clawed hands held open and short bowed legs with clawed feet.
// The pirouette is a brown whirlwind (the game shows it from the exported anim/pirouette_*
// frames, see taz_art_test.go). Animation is visual only: hitbox and collisions do not change.
// Standing rows: crest 3..5, skull 6..18, mouth 14..21, tongue down to 25, torso 22..32,
// legs 32..38, feet 39..42.
// ---------------------------------------------------------------------------

func isTaz(s *CharSpec) bool { return s.Name == "taz" }

var (
	// cartoon palette: chocolate-brown fur, cream muzzle, face and belly, a big round
	// dark-brown nose, light pink inside the ears, a red-pink tongue
	tzFur    = hx(0x6e4428)
	tzFurL   = hx(0x8e5c38)
	tzFurD   = hx(0x4a2a16)
	tzBeige  = hx(0xf0d8a8)
	tzBeigeL = hx(0xfcecc8)
	tzBeigeD = hx(0xd6b682)
	tzPink   = hx(0xf4c0b4)
	tzEyeW   = hx(0xf8f8f4)
	tzPupil  = hx(0x101010)
	tzNose   = hx(0x3a2012)
	tzNoseL  = hx(0x86583a)
	tzMouth  = hx(0x3c0c0e)
	tzMouthL = hx(0x7a1c1c)
	tzTooth  = hx(0xfcfcf6)
	tzTongue = hx(0xd8505a)
	tzTongL  = hx(0xf07c80)
	tzTongD  = hx(0xa83440)
	tzClaw   = hx(0x2a160c)
	tzDust   = hx(0xb8a888)
)

// taRun: a row of one colour from x0 to x1, r relative to an anchor row.
type taRun struct{ r, x0, x1 int }

func genTaz(s *CharSpec, vw int, p Pose) *pixBuf {
	b := newPix(charW, charH)
	if trackParts {
		b.pid = make([]uint8, charW*charH)
	}
	if p.Anim == AnimPirouette {
		tazWhirl(b, vw, p.F)
		return b
	}
	rnd := func(f float64) int { return int(math.Floor(f + 0.5)) }
	dk := func(c color.RGBA, dark bool) color.RGBA {
		if dark {
			return shade(c, 0.76)
		}
		return c
	}
	// the crest tips stand at row 3: never more than 2 rows higher
	bob := clampI(rnd(p.Bob), -2, 3)
	if p.Sit {
		bob = maxI(bob, 3)
	}
	ox := clampI(rnd(p.Sway*0.5), -1, 1)
	hox := ox
	if vw == vSide {
		ox = clampI(rnd(p.Lean*0.5), -1, 1)
		hox = clampI(ox+rnd(p.Lean*0.3+p.HeadZ*0.5), -1, 0) // jaws and crest stay inside
	} else if p.HeadZ != 0 {
		hox += clampI(rnd(p.HeadZ*0.5), -1, 1)
	}
	if vw != vSide {
		hox = clampI(hox, -1, 1) // ears and cheek tufts stay inside
	}
	hipY := 31 + bob
	ty0 := hipY - 9 // torso top
	hy := ty0 - 16  // skull top (crest above)
	lift := func(l Limb) int {
		if p.Sit {
			return 0
		}
		return clampI(rnd(10-l.Y), 0, 6)
	}
	mir := func(x int) int { return 23 - x }
	put := func(runs []taRun, dx int, col color.RGBA, flip bool) {
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

	// ---- short bowed legs with clawed feet ----
	leg := func(hx0 float64, l Limb, sign float64, id uint8, dark bool) {
		b.cur = id
		top := 39 - lift(l)
		var fx float64
		if vw == vSide {
			fx = hx0 + l.Z*0.8
			if p.Sit {
				fx = hx0 + 5
				top = hipY + 4
			}
			fx = clampF(fx, 2, 16)
		} else {
			fx = hx0 + sign*clampF((l.X-1.5)*0.8+0.5, -1.5, 2.5)
		}
		bx, by := sign, 0.0
		if vw == vSide {
			bx, by = 1, 0
		}
		mx, my, ex, ey := ik2(hx0, float64(hipY+1), fx+1, float64(top)-0.5, 3.6, bx, by)
		thickLine(b, hx0, float64(hipY+1), mx, my, 3, dk(tzFur, dark))
		thickLine(b, mx, my, ex, ey, 3, dk(tzFur, dark))
		thickLine(b, hx0+1, float64(hipY+2), mx+1, my, 1, dk(tzFurL, dark))
		ix := int(math.Floor(fx))
		if vw == vSide {
			// a long flat foot pointing forward, three claws at the tip
			b.rect(ix-1, top, 5, 1, dk(tzFur, dark))
			b.rect(ix-1, top+1, 7, 2, dk(tzFur, dark))
			b.rect(ix-1, top+2, 7, 1, dk(tzFurD, dark))
			b.set(ix+6, top+1, dk(tzClaw, dark))
			b.set(ix+6, top+3, dk(tzClaw, dark))
			b.set(ix+5, top+3, dk(tzFurD, dark))
			b.set(ix+4, top+2, dk(tzClaw, dark))
			return
		}
		// front/back: a wide paw, toes spread
		fx0 := ix - 1
		if sign < 0 {
			fx0 = ix - 2
		}
		b.rect(fx0+1, top, 4, 1, dk(tzFur, dark))
		b.rect(fx0, top+1, 6, 2, dk(tzFur, dark))
		b.rect(fx0, top+2, 6, 1, dk(tzFurD, dark))
		if vw == vFront {
			for _, cx := range []int{fx0, fx0 + 2, fx0 + 5} {
				b.set(cx, top+3, dk(tzClaw, dark))
			}
			b.set(fx0+1, top+1, dk(tzFurL, dark))
			b.set(fx0+3, top+1, dk(tzFurL, dark))
		} else {
			b.rect(fx0+1, top+3, 4, 1, dk(tzFurD, dark)) // heel pad
		}
	}

	// ---- pot-bellied torso ----
	torso := func() {
		b.cur = 2
		if vw == vSide {
			c := 10 + ox
			rows := []taRun{{0, -3, 3}, {1, -4, 4}, {2, -4, 5}, {3, -4, 5}, {4, -4, 6}, {5, -4, 6}, {6, -4, 6}, {7, -4, 5}, {8, -3, 5}, {9, -3, 4}, {10, -2, 3}}
			for _, r := range rows {
				for x := r.x0; x <= r.x1; x++ {
					col := tzFur
					if x >= r.x1-2 && r.r >= 1 {
						col = tzBeige // belly at the front
						if x == r.x1-2 {
							col = tzBeigeD
						}
					}
					if x == r.x0 {
						col = tzFurD
					}
					b.set(c+x, ty0+r.r, col)
				}
			}
			b.set(c-5, ty0+8, tzFurD) // tail tuft
			b.set(c-5, ty0+9, tzFur)
			b.set(c-6, ty0+9, tzFurD)
			return
		}
		rows := []taRun{{0, 6, 17}, {1, 5, 18}, {2, 5, 18}, {3, 4, 19}, {4, 4, 19}, {5, 4, 19}, {6, 4, 19}, {7, 5, 18}, {8, 5, 18}, {9, 6, 17}, {10, 7, 16}}
		for _, r := range rows {
			for x := r.x0; x <= r.x1; x++ {
				col := tzFur
				if x == r.x0 || x == r.x1 {
					col = tzFurD
				}
				if vw == vBack && (x == 11 || x == 12) && r.r >= 2 && r.r <= 8 {
					col = tzFurL // fur parting on the back
				}
				b.set(x+ox, ty0+r.r, col)
			}
		}
		if vw == vFront {
			b.ellipse(11.5+float64(ox), float64(ty0)+5.5, 5.2, 5.0, tzBeige)
			b.ellipse(11.0+float64(ox), float64(ty0)+4.5, 2.6, 2.4, tzBeigeL)
			for _, x := range []int{8, 15} { // ragged fur edge of the belly
				b.set(x+ox, ty0+9, tzFur)
			}
		} else {
			b.set(11+ox, hipY+1, tzFurD) // little tail
			b.set(12+ox, hipY+1, tzFurD)
			b.set(11+ox, hipY+2, tzFur)
		}
	}

	// ---- the head: crest, ears, angry eyes, nose, huge open mouth and tongue ----
	wag := 0
	if p.F%2 == 1 && p.Anim != AnimIdle {
		wag = 1
	}
	head := func() {
		b.cur = 1
		hd := hox
		switch vw {
		case vSide:
			// ear at the back, skull, crest swept back, long open jaws forward
			put([]taRun{{0, 5, 6}, {1, 4, 7}, {2, 4, 7}, {3, 5, 7}}, hd, tzFur, false)
			put([]taRun{{1, 5, 6}, {2, 5, 6}}, hd, tzPink, false)
			b.ellipse(10+float64(hd), float64(hy)+7.5, 6.2, 6.8, tzFur)
			put([]taRun{{-3, 9, 9}, {-3, 13, 13}, {-2, 8, 10}, {-2, 12, 13}, {-1, 6, 14}, {0, 4, 13}, {1, 3, 6}, {2, 2, 4}, {3, 2, 3}, {5, 3, 4}, {6, 2, 4}}, hd, tzFur, false)
			put([]taRun{{-1, 9, 11}, {0, 7, 9}}, hd, tzFurL, false)
			put([]taRun{{1, 3, 4}, {6, 2, 2}}, hd, tzFurD, false)
			// beige cheek and lower face
			b.ellipse(12+float64(hd), float64(hy)+11.5, 4.2, 4.6, tzBeige)
			put([]taRun{{9, 6, 7}, {10, 6, 6}}, hd, tzFur, false) // cheek tuft
			b.set(5+hd, hy+10, tzFurD)
			// upper jaw and nose at the tip
			put([]taRun{{6, 12, 19}, {7, 12, 20}, {8, 13, 21}}, hd, tzBeige, false) // cream muzzle
			put([]taRun{{6, 12, 13}}, hd, tzFur, false)
			put([]taRun{{7, 14, 18}}, hd, tzBeigeL, false)
			put([]taRun{{5, 18, 20}, {6, 19, 21}, {7, 21, 22}}, hd, tzNose, false)
			b.set(19+hd, hy+5, tzNoseL)
			// the mouth gapes: teeth top and bottom, dark inside
			put([]taRun{{9, 13, 21}, {10, 12, 21}, {11, 12, 20}, {12, 12, 20}, {13, 13, 19}}, hd, tzMouth, false)
			put([]taRun{{10, 13, 16}, {11, 13, 15}}, hd, tzMouthL, false)
			for _, x := range []int{14, 16, 18, 20} {
				b.set(x+hd, hy+9, tzTooth)
			}
			b.set(20+hd, hy+10, tzTooth) // fang
			for _, x := range []int{13, 15, 17, 19} {
				b.set(x+hd, hy+13, tzTooth)
			}
			put([]taRun{{14, 11, 19}, {15, 10, 17}, {16, 10, 15}}, hd, tzBeige, false) // lower jaw
			put([]taRun{{16, 13, 15}}, hd, tzBeigeD, false)
			// the tongue flops out of the front of the mouth and down the chest
			tx := 17 + wag
			put([]taRun{{11, tx - 2, tx}, {12, tx - 2, tx + 1}, {13, tx - 1, tx + 2}, {14, tx, tx + 2}, {15, tx, tx + 3}, {16, tx, tx + 3}, {17, tx, tx + 3}, {18, tx + 1, tx + 2}}, hd, tzTongue, false)
			put([]taRun{{13, tx, tx}, {14, tx + 1, tx + 1}, {15, tx + 1, tx + 1}, {16, tx + 1, tx + 1}}, hd, tzTongD, false)
			put([]taRun{{15, tx + 3, tx + 3}, {16, tx + 3, tx + 3}}, hd, tzTongL, false)
			// small eye and an angry brow
			put([]taRun{{3, 14, 16}, {4, 14, 16}}, hd, tzEyeW, false)
			b.set(15+hd, hy+4, tzPupil)
			b.set(16+hd, hy+4, tzPupil)
			put([]taRun{{1, 13, 14}, {2, 15, 17}}, hd, tzFurD, false)
			b.set(17+hd, hy+3, tzFurD)
		case vBack:
			for _, fl := range []bool{false, true} {
				put([]taRun{{-1, 2, 2}, {0, 2, 4}, {1, 2, 5}, {2, 2, 5}, {3, 2, 5}, {4, 3, 5}}, hd, tzFur, fl)
				put([]taRun{{1, 3, 3}, {2, 3, 4}}, hd, tzFurD, fl)
				put([]taRun{{9, 2, 4}, {10, 2, 3}, {11, 3, 3}}, hd, tzFur, fl) // cheek tufts
			}
			b.ellipse(11.5+float64(hd), float64(hy)+8, 8.0, 8.4, tzFur)
			put([]taRun{{-3, 8, 8}, {-3, 11, 11}, {-3, 15, 15}, {-2, 7, 9}, {-2, 11, 12}, {-2, 14, 16}, {-1, 7, 16}, {0, 6, 17}}, hd, tzFur, false)
			put([]taRun{{-1, 8, 9}, {-1, 12, 13}, {0, 8, 15}, {1, 9, 14}, {2, 10, 13}, {3, 11, 12}}, hd, tzFurL, false)
			put([]taRun{{12, 6, 7}, {13, 7, 8}, {12, 16, 17}, {13, 15, 16}, {15, 9, 14}}, hd, tzFurD, false)
		default:
			for _, fl := range []bool{false, true} {
				put([]taRun{{-1, 2, 2}, {0, 2, 4}, {1, 2, 5}, {2, 2, 5}, {3, 2, 5}, {4, 3, 5}}, hd, tzFur, fl)
				put([]taRun{{1, 3, 3}, {2, 3, 4}, {3, 3, 4}}, hd, tzPink, fl)
				put([]taRun{{9, 2, 4}, {10, 2, 3}, {11, 3, 3}}, hd, tzFur, fl) // cheek tufts
			}
			b.ellipse(11.5+float64(hd), float64(hy)+8, 8.0, 8.4, tzFur)
			// crest of spiky hair, a bit lopsided
			put([]taRun{{-3, 8, 8}, {-3, 11, 11}, {-3, 15, 15}, {-2, 7, 9}, {-2, 11, 12}, {-2, 14, 16}, {-1, 7, 16}, {0, 6, 17}}, hd, tzFur, false)
			put([]taRun{{-1, 8, 9}, {-1, 12, 13}, {0, 9, 10}}, hd, tzFurL, false)
			// beige face: around the eyes and the whole lower half
			b.ellipse(11.5+float64(hd), float64(hy)+11.5, 7.6, 6.2, tzBeige)
			put([]taRun{{3, 7, 10}, {3, 13, 16}, {4, 7, 10}, {4, 13, 16}, {5, 7, 9}, {5, 14, 16}}, hd, tzBeige, false)
			put([]taRun{{3, 11, 12}, {4, 11, 12}}, hd, tzFur, false) // brown peak between the eyes, the nose sits on cream
			// small eyes under angry brows (inner ends low)
			put([]taRun{{4, 8, 9}, {5, 8, 9}}, hd, tzEyeW, false)
			put([]taRun{{4, 14, 15}, {5, 14, 15}}, hd, tzEyeW, false)
			b.set(9+hd, hy+5, tzPupil)
			b.set(14+hd, hy+5, tzPupil)
			put([]taRun{{2, 6, 7}, {3, 8, 10}, {2, 16, 17}, {3, 13, 15}}, hd, tzFurD, false)
			// big round dark-brown nose on top, with a shine
			put([]taRun{{6, 10, 13}, {7, 9, 14}, {8, 10, 13}}, hd, tzNose, false)
			b.set(10+hd, hy+7, tzNoseL)
			b.set(11+hd, hy+6, tzNoseL)
			// the huge mouth, wider than the head is tall
			put([]taRun{{9, 5, 18}, {10, 4, 19}, {11, 4, 19}, {12, 5, 18}, {13, 5, 18}, {14, 6, 17}, {15, 7, 16}}, hd, tzMouth, false)
			put([]taRun{{11, 6, 9}, {12, 6, 8}, {11, 14, 17}, {12, 15, 17}}, hd, tzMouthL, false)
			// upper teeth in a row, two long fangs; lower teeth
			for _, x := range []int{6, 8, 10, 13, 15, 17} {
				b.set(x+hd, hy+9, tzTooth)
			}
			put([]taRun{{9, 5, 5}, {10, 5, 5}, {11, 5, 5}, {9, 18, 18}, {10, 18, 18}, {11, 18, 18}}, hd, tzTooth, false)
			for _, x := range []int{8, 10, 13, 15} {
				b.set(x+hd, hy+15, tzTooth)
			}
			put([]taRun{{14, 7, 7}, {14, 16, 16}}, hd, tzTooth, false)
			put([]taRun{{16, 6, 17}, {17, 8, 15}}, hd, tzBeige, false) // chin
			put([]taRun{{17, 8, 8}, {17, 15, 15}}, hd, tzBeigeD, false)
			// the big red tongue hangs out, a bit to his right, over the chest
			tx := 9 + wag
			put([]taRun{{11, tx + 1, tx + 4}, {12, tx, tx + 5}, {13, tx, tx + 5}, {14, tx - 1, tx + 5}, {15, tx - 1, tx + 5}, {16, tx - 1, tx + 5}, {17, tx - 1, tx + 5}, {18, tx, tx + 4}, {19, tx + 1, tx + 3}}, hd, tzTongue, false)
			put([]taRun{{13, tx + 2, tx + 2}, {14, tx + 2, tx + 2}, {15, tx + 2, tx + 2}, {16, tx + 2, tx + 2}}, hd, tzTongD, false)
			put([]taRun{{13, tx + 4, tx + 4}, {14, tx + 4, tx + 4}, {15, tx + 4, tx + 4}, {16, tx, tx}}, hd, tzTongL, false)
		}
	}

	// ---- long arms with three-clawed hands, held open ----
	arm := func(sx float64, l Limb, sign float64, id uint8, dark bool) (int, int) {
		b.cur = id
		sy := float64(ty0 + 1)
		var tx, ty float64
		if vw == vSide {
			tx, ty = l.Z, l.Y
		} else {
			tx, ty = sign*(l.X+0.9)*0.9, l.Y+l.Z*0.1
		}
		bx, by := sign*0.6, 0.6
		if vw == vSide {
			bx, by = -1, 0
		}
		hxT := clampF(sx+tx, 2, 21)
		mx, my, ex, ey := ik2(sx, sy, hxT, math.Max(sy+ty, 1), 5.0, bx, by)
		ex, ey = clampF(ex, 3, 20), math.Max(ey, 2)
		mx, my = clampF(mx, 1, 21), math.Max(my, 1)
		thickLine(b, sx, sy, mx, my, 2, dk(tzFur, dark))
		thickLine(b, mx, my, ex, ey, 2, dk(tzFur, dark))
		thickLine(b, sx, sy, sx, sy+1, 2, dk(tzFurL, dark)) // shoulder
		ix, iy := int(math.Floor(ex)), int(math.Floor(ey))
		hx0 := ix
		if vw != vSide && sign < 0 {
			hx0 = ix - 1
		}
		hx0 = clampI(hx0, 2, 19) // the claws stay inside the box
		// palm and three spread claws
		b.rect(hx0, iy, 3, 2, dk(tzFur, dark))
		b.set(hx0+1, iy, dk(tzFurL, dark))
		b.set(hx0-1, iy+2, dk(tzClaw, dark))
		b.set(hx0+1, iy+2, dk(tzClaw, dark))
		b.set(hx0+1, iy+3, dk(tzClaw, dark))
		b.set(hx0+3, iy+2, dk(tzClaw, dark))
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
		cx := float64(10 + ox)
		arm(cx-0.5, p.LA, 1, 3, true)
		leg(8.5, p.LF, 1, 5, true)
		leg(10, p.RF, 1, 6, false)
		torso()
		head()
		hx0, hy0 := arm(cx+0.5, p.RA, 1, 4, false)
		held(hx0, hy0, 4)
	case vBack:
		leg(14, p.LF, 1, 5, false)
		leg(8, p.RF, -1, 6, false)
		torso()
		head()
		arm(float64(19+ox), p.LA, 1, 3, false)
		hx0, hy0 := arm(float64(4+ox), p.RA, -1, 4, false)
		held(hx0, hy0, 4)
	default:
		leg(8, p.LF, -1, 5, false)
		leg(14, p.RF, 1, 6, false)
		torso()
		head()
		arm(float64(4+ox), p.LA, -1, 3, false)
		hx0, hy0 := arm(float64(19+ox), p.RA, 1, 4, false)
		held(hx0, hy0, 4)
	}
	return b
}

// tazWhirl: the spin. A brown tornado, wide at the top and thin where it touches the floor,
// with the stripes running round (frame f), flashes of beige belly, teeth and the red
// tongue, and dust at the bottom. The same in every view.
func tazWhirl(b *pixBuf, vw, f int) {
	b.cur = 2
	const top, bot = 4, 40
	for y := top; y <= bot; y++ {
		t := float64(y-top) / float64(bot-top)
		hw := 9.5 - 7.5*t + 0.8*math.Sin(float64(y)*0.7+float64(f)*1.3) // half width, wobbling
		cx := 11.5 + 1.6*math.Sin(t*3.2+float64(f)*1.05)                // the column sways
		for x := int(math.Floor(cx - hw)); x <= int(math.Ceil(cx+hw))-1; x++ {
			if x < 1 || x > 22 {
				continue
			}
			// diagonal stripes moving with the frame
			k := ((x-y/2+f*2)%6 + 6) % 6
			col := tzFur
			switch k {
			case 0:
				col = tzFurL
			case 3, 4:
				col = tzFurD
			}
			u := (float64(x) + 0.5 - cx) / hw
			if math.Abs(u) > 0.8 {
				col = tzFurD
			}
			if y >= 12 && y <= 22 && k == 1 && math.Abs(u) < 0.6 {
				col = tzBeige // a flash of belly and face
			}
			b.set(x, y, col)
		}
	}
	// teeth and tongue whirling by: their place follows the frame
	ph := float64(f) * 1.05
	tx := int(math.Floor(11.5 + 5*math.Sin(ph)))
	if vw != vBack || f%2 == 0 {
		for i := 0; i < 3; i++ {
			b.set(tx+i, 10, tzTooth)
		}
		b.set(tx+1, 11, tzMouth)
		b.rect(tx, 12, 3, 2, tzTongue)
		b.set(tx+1, 14, tzTongL)
	}
	// spinning streaks at the sides
	for i, y := range []int{7, 16, 25} {
		sx := 2 + (f+i)%3
		b.set(sx, y, tzFurL)
		b.set(21-(f+i)%3, y+3, tzFurL)
	}
	// dust at the floor
	for i, dx := range []int{-5, -3, 3, 5} {
		yy := 41 + (i+f)%2
		b.set(clampI(11+dx+(f%2), 1, 22), yy, tzDust)
		b.set(clampI(12+dx-(f%2), 1, 22), yy-1, tzDust)
	}
}
