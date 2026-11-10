package main

import (
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// Six superheroes with their own procedural drawing (genHero), like Crash or the Simpsons: the
// generic thin body could not show a cowl, a flowing cape or Hulk's shoulders. Our own pixel-art
// parodies in a clean "standing hero" style: a chunky head, broad chest with its emblem, belt,
// gloves and boots in their colours.
//   superman: black hair with a curl, blue suit, red-and-yellow S, red cape, briefs and boots.
//   batman: black cowl with pointed ears, white eyes, bare chin, grey suit with a black bat,
//           yellow belt with pouches, black gloves (yellow fin), boots and a long dark cape.
//   hulk: a green giant, huge shoulders and fists, messy black hair, black brows in a frown,
//         darker green muscles, torn short purple trousers, big bare green feet.
//   capitan_america: blue hood with white wings and an "A", pink ears, white star, red and white
//         stripes on the belly, yellow buckle, white gloves, red boots and a round shield.
//   lobezno: blue mask with tall pointed yellow-and-blue fins, yellow suit with blue shoulders
//         and striped sides, gold belt, yellow gloves and three silver claws.
//   deadpool: red hood with black eye patches (white eyes), red and black suit, belts and a
//         chest strap, two katanas crossed on the back, black gloves and boots.
// Animation (visual only: hitbox, footprint and stats stay those of the preset): arms and legs
// follow the pose, the body bobs and sways, capes flutter while walking. Parts for the rig:
// 1 head, 2 torso (cape and back katanas too), 3/4 arms (shield and left claws on 3), 5/6 legs.
// ---------------------------------------------------------------------------

const (
	heroNone = iota
	heroSuperman
	heroBatman
	heroHulk
	heroCap
	heroLogan
	heroDeadpool
)

var heroKinds = map[string]int{
	"superman": heroSuperman, "batman": heroBatman, "hulk": heroHulk,
	"capitan_america": heroCap, "lobezno": heroLogan, "deadpool": heroDeadpool,
}

func heroKind(s *CharSpec) int {
	if s == nil {
		return heroNone
	}
	return heroKinds[s.Name]
}

func isHeroArt(s *CharSpec) bool { return heroKind(s) != heroNone }

// heroPal: the colours of one hero (A==0 means "none").
type heroPal struct {
	skin, suit, arm, glove, hand, leg, boot, briefs, belt, buckle color.RGBA
	cape, capeIn                                                  color.RGBA
}

var (
	hrSkin    = hx(0xf0c8a0)
	hrSkinD   = hx(0xc89878)
	hrBlack   = hx(0x15151c)
	hrWhite   = hx(0xf6f6f2)
	hrYellow  = hx(0xf4cc30)
	hrYellowD = hx(0xc89818)
	hrRed     = hx(0xd42a2a)
	hrRedD    = hx(0x8e1418)
	hrEye     = hx(0x1a1a24)
	hrMouth   = hx(0x7a2a24)

	hrSupBlue = hx(0x2c5ad2)
	hrBatGrey = hx(0x7e808e)
	hrBatCape = hx(0x24242e)
	hrHulk    = hx(0x5cb43a)
	hrHulkD   = hx(0x3a8226)
	hrHulkL   = hx(0x86d45e)
	hrHulkHr  = hx(0x141018)
	hrPurple  = hx(0x7a3aa8)
	hrPurpleD = hx(0x54247a)
	hrCapBlue = hx(0x2a4cb8)
	hrPink    = hx(0xf0a0a0)
	hrLogY    = hx(0xf2ca22)
	hrLogB    = hx(0x2848b0)
	hrLogBD   = hx(0x182c74)
	hrGold    = hx(0xd89418)
	hrDPRed   = hx(0xc8202a)
	hrDPRedD  = hx(0x861018)
	hrSilver  = hx(0xe4eaf2)
	hrSilverD = hx(0x9aa4b4)
	hrHilt    = hx(0x3a3a46)
)

func heroPalette(k int) heroPal {
	switch k {
	case heroSuperman:
		return heroPal{skin: hrSkin, suit: hrSupBlue, arm: hrSupBlue, glove: hrSupBlue, hand: hrSkin, leg: hrSupBlue,
			boot: hrRed, briefs: hrRed, belt: hrYellow, buckle: hrYellowD, cape: hrRed, capeIn: hrRedD}
	case heroBatman:
		return heroPal{skin: hrSkin, suit: hrBatGrey, arm: hrBatGrey, glove: hrBlack, hand: hrBlack, leg: hrBatGrey,
			boot: hrBlack, briefs: hx(0x2c2c36), belt: hrYellow, buckle: hrYellowD, cape: hrBatCape, capeIn: hx(0x111116)}
	case heroHulk:
		return heroPal{skin: hrHulk, suit: hrHulk, arm: hrHulk, glove: hrHulk, hand: hrHulk, leg: hrHulk,
			boot: hrHulk, briefs: hrPurple, belt: hrPurpleD, buckle: hrPurpleD}
	case heroCap:
		return heroPal{skin: hrSkin, suit: hrCapBlue, arm: hrCapBlue, glove: hrWhite, hand: hrWhite, leg: hrCapBlue,
			boot: hrRed, briefs: hrCapBlue, belt: hrRed, buckle: hrYellow}
	case heroLogan:
		return heroPal{skin: hrSkin, suit: hrLogY, arm: hrLogB, glove: hrLogY, hand: hrLogY, leg: hrLogY,
			boot: hrLogB, briefs: hrLogB, belt: hrGold, buckle: hrRed}
	case heroDeadpool:
		return heroPal{skin: hrSkin, suit: hrDPRed, arm: hrDPRed, glove: hrBlack, hand: hrBlack, leg: hrDPRed,
			boot: hrBlack, briefs: hrDPRed, belt: hx(0x34343e), buckle: hrRed}
	}
	return heroPal{}
}

func genHero(s *CharSpec, vw int, p Pose) *pixBuf {
	k := heroKind(s)
	pal := heroPalette(k)
	b := newPix(charW, charH)
	if trackParts {
		b.pid = make([]uint8, charW*charH)
	}
	big := k == heroHulk
	rnd := func(f float64) int { return int(math.Floor(f + 0.5)) }
	dk := func(c color.RGBA, dark bool) color.RGBA {
		if dark {
			return shade(c, 0.78)
		}
		return c
	}
	hl := func(c color.RGBA) color.RGBA { return mixc(c, hrWhite, 0.28) }
	sd := func(c color.RGBA) color.RGBA { return shade(c, 0.72) }
	row := func(y, x0, x1 int, c color.RGBA) {
		for x := x0; x <= x1; x++ {
			b.set(x, y, c)
		}
	}
	mir := func(x int) int { return 23 - x }

	bob := clampI(rnd(p.Bob), -2, 3)
	if p.Sit {
		bob = maxI(bob, 3)
	}
	ox := clampI(rnd(p.Sway*0.5), -2, 2)
	if big {
		ox = clampI(ox, -1, 1)
	}
	hox := ox
	if vw == vSide {
		ox = clampI(rnd(p.Lean*0.5), -1, 2)
		hox = clampI(ox+rnd(p.Lean*0.3+p.HeadZ*0.5), -1, 2)
		if big {
			hox = clampI(hox+1, 0, 2) // Hulk's head juts forward
		}
	} else if p.HeadZ != 0 {
		hox += clampI(rnd(p.HeadZ*0.5), -1, 1)
	}
	moving := math.Abs(p.LF.Z)+math.Abs(p.RF.Z) > 1.5 || math.Abs(p.LA.Z)+math.Abs(p.RA.Z) > 3

	hipY := 29 + bob
	ty0 := hipY - 11 // torso top
	hy := ty0 - 11   // skull top
	footTop := 39
	legW, armW, fist := 3, 2, 3
	seg, armSeg := 4.6, 4.6
	if big {
		hipY = 30 + bob
		ty0 = hipY - 13
		hy = ty0 - 9
		legW, armW, fist = 4, 4, 4
		seg, armSeg = 4.0, 5.2
	}
	capeBot := hipY + 10

	// half widths of the torso rows (front and back), centre between columns 11 and 12
	tor := []int{5, 6, 6, 6, 6, 5, 5, 4, 4, 4}
	if big {
		tor = []int{7, 9, 9, 9, 9, 9, 8, 8, 7, 6, 5, 5, 5}
	}
	lift := func(l Limb) int {
		if p.Sit {
			return 0
		}
		return clampI(rnd(10-l.Y), 0, 6)
	}

	// ---- cape ----
	cape := func() {
		if pal.cape.A == 0 {
			return
		}
		b.cur = 2
		switch vw {
		case vSide:
			c := 11 + ox
			kk := 0.16
			if k == heroBatman {
				kk = 0.24 // the long bat cape flares more
			}
			if moving {
				kk = 0.36
			}
			for y := ty0; y <= capeBot; y++ {
				xb := c - 3 - int(float64(y-ty0)*kk)
				if moving && ((y/3)+p.F)%2 == 0 {
					xb--
				}
				xb = maxI(xb, 2)
				row(y, xb, c-1, pal.cape)
				b.set(xb, y, pal.capeIn)
				if y == capeBot {
					row(y, xb, c-1, pal.capeIn)
				}
			}
		case vBack:
			for y := ty0; y <= capeBot; y++ {
				hw := 6 + (y-ty0)*7/(2*(capeBot-ty0))
				x0, x1 := 12-hw+ox, 11+hw+ox
				if y == ty0 {
					x0, x1 = x0+1, x1-1
				}
				row(y, x0, x1, pal.cape)
				b.set(x0, y, pal.capeIn)
				b.set(x1, y, pal.capeIn)
				if y > ty0+3 { // folds
					f := (y - ty0) / 5
					b.set(9-f/2+ox, y, pal.capeIn)
					b.set(14+f/2+ox, y, pal.capeIn)
					b.set(11+ox+(y%2), y, shade(pal.cape, 0.88))
				}
				if y == capeBot {
					for x := x0; x <= x1; x++ {
						if moving && (x+p.F)%3 == 0 {
							b.set(x, y, color.RGBA{})
						} else {
							b.set(x, y, pal.capeIn)
						}
					}
				}
			}
			row(ty0, 9+ox, 14+ox, hl(pal.cape)) // collar line on the shoulders
		default: // the inside of the cape shows round the body
			for y := ty0 + 1; y <= capeBot; y++ {
				hw := 6 + (y-ty0)*7/(2*(capeBot-ty0))
				x0, x1 := 12-hw+ox, 11+hw+ox
				// only the edges show beside the body: the legs stay apart
				w := 2 + (y-ty0)/8
				row(y, x0, x0+w-1, pal.capeIn)
				row(y, x1-w+1, x1, pal.capeIn)
				b.set(x0, y, pal.cape)
				b.set(x1, y, pal.cape)
				if y == capeBot && moving {
					b.set(x0+(p.F%2), y, color.RGBA{})
				}
			}
		}
	}

	// ---- legs and boots ----
	leg := func(hx0 float64, l Limb, sign float64, id uint8, dark bool) {
		b.cur = id
		top := footTop - lift(l)
		var fx float64
		if vw == vSide {
			fx = hx0 + l.Z*0.7
			if p.Sit {
				fx = hx0 + 5
				top = hipY + 4
			}
			fx = clampF(fx, 2, 16)
		} else {
			fx = hx0 + sign*clampF((l.X-1.5)*0.8, -2, 2)
		}
		bx, by := sign, 0.0
		if vw == vSide {
			bx, by = 1, 0
		}
		hipTop := float64(hipY + 1)
		mx, my, ex, ey := ik2(hx0, hipTop, fx, float64(top)-0.5, seg, bx, by)
		thickLine(b, hx0, hipTop, mx, my, legW, dk(pal.leg, dark))
		shin := pal.boot
		if big {
			shin = pal.leg
		}
		thickLine(b, mx, my, ex, ey, legW, dk(shin, dark))
		ix, iy := int(math.Floor(mx)), int(math.Floor(my))
		if big { // torn trousers down to the knee, ragged hem
			thickLine(b, hx0, hipTop, mx, my, legW, dk(pal.briefs, dark))
			for i := 0; i < legW; i++ {
				b.set(ix+i, iy, dk(pal.briefs, dark))
				if (i+int(id))%2 == 0 {
					b.set(ix+i, iy+1, dk(hrPurpleD, dark))
				}
			}
			thickLine(b, hx0+float64(legW)-1, hipTop+1, mx+float64(legW)-1, my, 1, dk(hrPurpleD, dark))
		} else {
			if vw != vSide { // boot cuff
				b.rect(ix, iy+1, legW, 1, dk(hl(pal.boot), dark))
			}
			edge := 0.0
			if vw != vSide && sign > 0 {
				edge = float64(legW - 1)
			}
			thickLine(b, hx0+edge, hipTop+1, mx+edge, my, 1, dk(sd(pal.leg), dark))
		}
		fxI := int(math.Floor(ex))
		fy := int(math.Floor(ey)) + 1
		fc := dk(pal.boot, dark)
		if vw == vSide {
			w := 5
			if big {
				w = 6
			}
			b.rect(fxI-1, fy, w, 2, fc)
			b.rect(fxI-1, fy+1, w, 1, dk(sd(pal.boot), dark))
			if big { // toes
				b.set(fxI+w-2, fy+1, dk(hrHulkD, dark))
			}
		} else {
			w := 4
			x0 := fxI
			if sign < 0 {
				x0 = fxI - 1
			}
			if big {
				w = 6
				x0 = fxI - 1
			}
			b.rect(x0, fy, w, 2, fc)
			b.rect(x0, fy+1, w, 1, dk(sd(pal.boot), dark))
			if big && vw == vFront {
				for t := 0; t < 3; t++ {
					b.set(x0+1+t*2, fy+1, dk(hrHulkD, dark))
				}
			}
		}
	}

	// ---- torso ----
	torsoFront := func() {
		b.cur = 2
		for r, hw := range tor {
			y := ty0 + r
			x0, x1 := 12-hw+ox, 11+hw+ox
			row(y, x0, x1, pal.suit)
			b.set(x0, y, sd(pal.suit))
			b.set(x1, y, sd(pal.suit))
			if r == 0 {
				row(y, x0+1, x1-1, hl(pal.suit))
			}
		}
		wl, wr := 12-tor[len(tor)-1]+ox, 11+tor[len(tor)-1]+ox
		// belt and briefs / trousers
		row(hipY-1, wl, wr, pal.belt)
		row(hipY, wl, wr, pal.briefs)
		row(hipY+1, wl, wr, sd(pal.briefs))
		if big {
			row(hipY-1, wl, wr, pal.briefs)
			row(hipY-1, wl+1, wr-1, hrPurpleD)
		}
	}
	torsoSide := func() {
		b.cur = 2
		c := 11 + ox
		type rr struct{ a, z int }
		rows := []rr{{-2, 2}, {-3, 3}, {-3, 3}, {-3, 3}, {-3, 2}, {-2, 2}, {-2, 2}, {-2, 2}, {-2, 2}, {-2, 2}}
		if big {
			rows = []rr{{-4, 3}, {-5, 4}, {-5, 5}, {-5, 5}, {-5, 5}, {-4, 5}, {-4, 4}, {-4, 4}, {-3, 4}, {-3, 3}, {-3, 3}, {-3, 3}, {-3, 3}}
		}
		for r, q := range rows {
			y := ty0 + r
			row(y, c+q.a, c+q.z, pal.suit)
			b.set(c+q.a, y, sd(pal.suit))
			if r == 0 {
				row(y, c+q.a+1, c+q.z, hl(pal.suit))
			}
		}
		row(hipY-1, c-2, c+2, pal.belt)
		row(hipY, c-2, c+2, pal.briefs)
		row(hipY+1, c-2, c+2, sd(pal.briefs))
		if big {
			row(hipY-1, c-3, c+3, hrPurpleD)
			row(hipY, c-3, c+3, pal.briefs)
			row(hipY+1, c-3, c+3, sd(pal.briefs))
		}
	}

	// ---- per-hero details on the torso ----
	details := func() {
		b.cur = 2
		m := 12 + ox // right column of the centre pair is m, left is m-1
		front, side, back := vw == vFront, vw == vSide, vw == vBack
		c := 11 + ox
		switch k {
		case heroSuperman:
			if front { // the S shield
				pat := func(y, x0 int, str string) {
					for i, ch := range str {
						switch ch {
						case 'R':
							b.set(x0+i, y, hrRed)
						case 'Y':
							b.set(x0+i, y, hrYellow)
						}
					}
				}
				pat(ty0+1, m-3, "RRRRRR")
				pat(ty0+2, m-3, "RRYYYR")
				pat(ty0+3, m-3, ".YRRY.")
				pat(ty0+4, m-3, "..YR..")
				row(hipY-1, m-1, m, hrYellowD) // buckle
			}
			if side {
				row(ty0+1, c+1, c+3, hrRed)
				row(ty0+2, c+1, c+3, hrYellow)
				b.set(c+2, ty0+2, hrRed)
				row(ty0+3, c+1, c+2, hrRed)
			}
		case heroBatman:
			if front { // a black bat across the chest
				b.set(m-4, ty0+2, hrBlack)
				b.set(m+3, ty0+2, hrBlack)
				b.set(m-1, ty0+2, hrBlack)
				b.set(m, ty0+2, hrBlack)
				row(ty0+3, m-4, m+3, hrBlack)
				b.set(m-3, ty0+4, hrBlack)
				b.set(m-1, ty0+4, hrBlack)
				b.set(m, ty0+4, hrBlack)
				b.set(m+2, ty0+4, hrBlack)
			}
			// pouches on the yellow belt
			if !side {
				for x := 12 - tor[len(tor)-1] + ox; x <= 11+tor[len(tor)-1]+ox; x += 2 {
					b.set(x, hipY-1, hrYellowD)
				}
				if front {
					row(hipY-1, m-1, m, hx(0xfff0a0)) // buckle
				}
			} else {
				b.set(c, hipY-1, hrYellowD)
				b.set(c+2, hipY-1, hrYellowD)
			}
		case heroHulk:
			mD, mL := hrHulkD, hrHulkL
			if front {
				// pecs and the line between them
				row(ty0+5, m-7, m-2, mD)
				row(ty0+5, m+1, m+6, mD)
				for y := ty0 + 1; y <= ty0+5; y++ {
					b.set(m-1, y, mD)
				}
				row(ty0+1, m-6, m-3, mL)
				row(ty0+1, m+2, m+5, mL)
				// abs
				for y := ty0 + 6; y <= ty0+11; y++ {
					b.set(m-2, y, mD)
					b.set(m+1, y, mD)
				}
				row(ty0+8, m-2, m+1, mD)
				row(ty0+10, m-2, m+1, mD)
				// traps up to the neck
				row(ty0-1, m-5, m+4, pal.suit)
				row(ty0-2, m-3, m+2, pal.suit)
				b.set(m-8, ty0+2, mL)
				b.set(m+7, ty0+2, mL)
			} else if back {
				row(ty0-1, m-5, m+4, pal.suit)
				row(ty0-2, m-3, m+2, pal.suit)
				for y := ty0 + 2; y <= ty0+10; y++ {
					b.set(m-1, y, mD) // spine
				}
				row(ty0+4, m-6, m-3, mD) // shoulder blades
				row(ty0+4, m+2, m+5, mD)
				row(ty0+1, m-7, m+6, mL)
			} else {
				row(ty0-1, c-3, c+2, pal.suit)
				row(ty0+5, c+1, c+5, mD)
				row(ty0+1, c-3, c+2, mL)
				b.set(c+3, ty0+8, mD)
				b.set(c+3, ty0+10, mD)
			}
		case heroCap:
			if front { // white star and red/white stripes
				row(ty0+2, m-1, m, hrWhite)
				row(ty0+3, m-3, m+2, hrWhite)
				row(ty0+4, m-2, m+1, hrWhite)
				b.set(m-2, ty0+5, hrWhite)
				b.set(m+1, ty0+5, hrWhite)
			}
			if !back {
				x0, x1 := 12-tor[7]+ox, 11+tor[7]+ox
				if side {
					x0, x1 = c-2, c+2
				}
				row(ty0+7, x0, x1, hrRed)
				row(ty0+8, x0, x1, hrWhite)
				row(ty0+9, x0, x1, hrRed)
				if front {
					row(hipY-1, m-1, m, hrYellow)
				} else {
					b.set(c+2, hipY-1, hrYellow)
				}
			}
			if side { // star tip on the chest
				b.set(c+3, ty0+3, hrWhite)
				b.set(c+2, ty0+3, hrWhite)
			}
		case heroLogan:
			B, BD := hrLogB, hrLogBD
			if side {
				row(ty0, c-2, c+2, B)
				row(ty0+1, c-3, c+1, B)
				for y := ty0 + 2; y <= ty0+9; y++ {
					row(y, c-2, c-1, B)
				}
				b.set(c-2, ty0+4, hrBlack)
				b.set(c-1, ty0+5, hrBlack)
				b.set(c-2, ty0+7, hrBlack)
			} else {
				// blue shoulders that dip into a V on the chest (back: plain blue yoke)
				for r := 0; r < 3; r++ {
					hw := tor[r]
					row(ty0+r, 12-hw+ox, 11+hw+ox, B)
				}
				if front {
					row(ty0, m-2, m+1, hrLogY)
					row(ty0+1, m-1, m, hrLogY)
				}
				// blue sides with black tiger stripes
				for r := 3; r < len(tor); r++ {
					hw := tor[r]
					b.set(12-hw+ox, ty0+r, B)
					b.set(13-hw+ox, ty0+r, B)
					b.set(11+hw+ox, ty0+r, B)
					b.set(10+hw+ox, ty0+r, B)
				}
				for _, r := range []int{3, 5, 7} {
					hw := tor[r]
					b.set(13-hw+ox, ty0+r, hrBlack)
					b.set(10+hw+ox, ty0+r, hrBlack)
				}
				b.set(12-tor[0]+ox, ty0, BD)
				b.set(11+tor[0]+ox, ty0, BD)
			}
			if front {
				row(hipY-1, m-1, m, hrRed) // round buckle
			}
		case heroDeadpool:
			K := hrBlack
			if side {
				row(ty0, c-2, c+1, K) // shoulder
				for y := ty0 + 1; y <= ty0+9; y++ {
					row(y, c-1, c, K) // black side band
				}
				// sheaths behind the back, hilts over the shoulder
				for i := 0; i < 9; i++ {
					b.set(c-4+i/4, ty0-2+i, hrBlack)
				}
				b.set(c-4, ty0-3, hrHilt)
				b.set(c-5, ty0-4, hrHilt)
				b.set(c-3, ty0-2, hrRed)
				b.set(c+1, hipY-1, hrRed)
				break
			}
			// black shoulders and sides
			row(ty0, 12-tor[0]+ox, 13-tor[0]+ox, K)
			row(ty0, 10+tor[0]+ox, 11+tor[0]+ox, K)
			for r := 1; r < len(tor); r++ {
				hw := tor[r]
				row(ty0+r, 12-hw+ox, 13-hw+ox, K)
				row(ty0+r, 10+hw+ox, 11+hw+ox, K)
			}
			if front {
				// a strap across the chest
				for i := 0; i < 8; i++ {
					b.set(m-4+i, ty0+1+i, hrHilt)
				}
				// katana hilts over the shoulders
				b.set(m-6, ty0-1, hrHilt)
				b.set(m-7, ty0-2, hrHilt)
				b.set(m+5, ty0-1, hrHilt)
				b.set(m+6, ty0-2, hrHilt)
				row(hipY-1, m-1, m, hrRed)
				b.set(m-1, hipY-1, hrBlack)
				b.set(m-3, hipY, hrHilt) // second belt / pouch
				b.set(m+2, hipY, hrHilt)
			} else { // two katanas crossed on the back
				for i := 0; i <= 11; i++ {
					b.set(m-6+i, ty0-2+i, hrBlack)
					b.set(m+5-i, ty0-2+i, hrBlack)
				}
				for _, q := range [][2]int{{m - 7, ty0 - 3}, {m - 6, ty0 - 2}, {m + 6, ty0 - 3}, {m + 5, ty0 - 2}} {
					b.set(q[0], q[1], hrHilt)
				}
				b.set(m-6, ty0-1, hrSilverD) // guards
				b.set(m+5, ty0-1, hrSilverD)
			}
		}
	}

	// ---- heads ----
	head := func() {
		b.cur = 1
		hd := hox
		open := p.Open || p.Anim == AnimLaugh || p.Anim == AnimCheer
		// face shape (rows from hy): front and back
		if big {
			hulkHead(b, vw, hy, hd, open)
			return
		}
		shape := func(c color.RGBA) {
			if vw == vSide {
				row(hy, 10+hd, 14+hd, c)
				for r := 1; r <= 8; r++ {
					row(hy+r, 9+hd, 15+hd, c)
				}
				row(hy+9, 10+hd, 14+hd, c)
				return
			}
			row(hy, 9+hd, 14+hd, c)
			for r := 1; r <= 8; r++ {
				row(hy+r, 8+hd, 15+hd, c)
			}
			row(hy+9, 9+hd, 14+hd, c)
		}
		neck := func(c color.RGBA) {
			if vw == vSide {
				row(hy+10, 11+hd, 13+hd, c)
			} else {
				row(hy+10, 10+hd, 13+hd, c)
			}
		}
		eyesFront := func(white, pupil color.RGBA, y int) {
			row(y, 9+hd, 10+hd, white)
			row(y, 13+hd, 14+hd, white)
			b.set(10+hd, y, pupil)
			b.set(13+hd, y, pupil)
		}
		mouth := func(y, x0, x1 int) {
			if open {
				row(y, x0, x1, hrMouth)
				row(y+1, x0, x1, shade(hrMouth, 0.7))
			} else {
				row(y, x0, x1, shade(pal.skin, 0.7))
			}
		}
		switch k {
		case heroSuperman:
			neck(pal.skin)
			shape(pal.skin)
			H := hrBlack
			switch vw {
			case vSide:
				row(hy-1, 10+hd, 13+hd, H)
				row(hy, 9+hd, 15+hd, H)
				row(hy+1, 9+hd, 15+hd, H)
				for r := 2; r <= 7; r++ {
					row(hy+r, 9+hd, 11+hd, H)
				}
				b.set(16+hd, hy+2, H) // the curl
				b.set(15+hd, hy+3, H)
				b.set(12+hd, hy+4, hrSkinD) // ear
				b.set(12+hd, hy+5, hrSkinD)
				b.set(14+hd, hy+4, hrEye)
				b.set(16+hd, hy+5, pal.skin) // nose
				mouth(hy+8, 14+hd, 15+hd)
			case vBack:
				row(hy-1, 10+hd, 13+hd, H)
				shape(H)
				row(hy+9, 10+hd, 13+hd, pal.skin)
				b.set(8+hd, hy+5, hrSkinD)
				b.set(15+hd, hy+5, hrSkinD)
			default:
				row(hy-1, 10+hd, 13+hd, H)
				row(hy, 9+hd, 14+hd, H)
				row(hy+1, 8+hd, 15+hd, H)
				row(hy+2, 8+hd, 15+hd, H)
				b.set(8+hd, hy+3, H)
				b.set(15+hd, hy+3, H)
				b.set(8+hd, hy+4, H)
				b.set(15+hd, hy+4, H)
				b.set(12+hd, hy+3, H) // the curl on the forehead
				b.set(11+hd, hy+4, H)
				row(hy+4, 9+hd, 10+hd, shade(H, 1.4)) // brows
				row(hy+4, 13+hd, 14+hd, shade(H, 1.4))
				eyesFront(hrWhite, hx(0x2850b8), hy+5)
				b.set(11+hd, hy+7, hrSkinD) // nose
				mouth(hy+8, 11+hd, 12+hd)
			}
		case heroBatman:
			C := hrBlack
			neck(C)
			shape(C)
			switch vw {
			case vSide:
				for r := 1; r <= 3; r++ { // ear
					row(hy-r, 11+hd, 11+hd+(3-r)/2, C)
				}
				row(hy+6, 13+hd, 15+hd, pal.skin)
				row(hy+7, 13+hd, 15+hd, pal.skin)
				row(hy+8, 13+hd, 15+hd, pal.skin)
				row(hy+9, 13+hd, 14+hd, pal.skin)
				b.set(16+hd, hy+5, C) // cowl nose
				row(hy+4, 14+hd, 15+hd, hrWhite)
				mouth(hy+8, 14+hd, 15+hd)
			case vBack:
				for _, x := range []int{9, 14} {
					for r := 1; r <= 3; r++ {
						b.set(x+hd, hy-r, C)
					}
					b.set(x+hd+map[int]int{9: 1, 14: -1}[x], hy-1, C)
				}
				row(hy+3, 9+hd, 14+hd, shade(C, 1.6)) // a dull shine
			default:
				for _, x := range []int{9, 14} {
					for r := 1; r <= 3; r++ {
						b.set(x+hd, hy-r, C)
					}
					b.set(x+hd+map[int]int{9: 1, 14: -1}[x], hy-1, C)
				}
				row(hy+1, 10+hd, 13+hd, shade(C, 1.7))
				// white eye slits under a frowning brow
				row(hy+4, 9+hd, 10+hd, hrWhite)
				row(hy+4, 13+hd, 14+hd, hrWhite)
				b.set(11+hd, hy+4, C)
				b.set(12+hd, hy+4, C)
				// lower face: mouth and chin
				row(hy+6, 10+hd, 13+hd, pal.skin)
				row(hy+7, 9+hd, 14+hd, pal.skin)
				row(hy+8, 9+hd, 14+hd, pal.skin)
				row(hy+9, 10+hd, 13+hd, pal.skin)
				mouth(hy+8, 11+hd, 12+hd)
				b.set(10+hd, hy+9, hrSkinD)
				b.set(13+hd, hy+9, hrSkinD)
			}
		case heroCap:
			B := hrCapBlue
			neck(B)
			shape(B)
			switch vw {
			case vSide:
				row(hy+5, 13+hd, 15+hd, pal.skin)
				row(hy+6, 13+hd, 15+hd, pal.skin)
				row(hy+7, 13+hd, 15+hd, pal.skin)
				row(hy+8, 13+hd, 15+hd, pal.skin)
				row(hy+9, 13+hd, 14+hd, pal.skin)
				b.set(14+hd, hy+5, hrEye)
				b.set(16+hd, hy+6, pal.skin)
				mouth(hy+8, 14+hd, 15+hd)
				b.set(11+hd, hy+5, hrPink) // ear
				b.set(11+hd, hy+6, hrPink)
				// the white wing on the side
				row(hy+2, 9+hd, 11+hd, hrWhite)
				row(hy+3, 10+hd, 12+hd, hrWhite)
				b.set(8+hd, hy+1, hrWhite)
				b.set(14+hd, hy+1, hrWhite) // the A seen from the side
				b.set(14+hd, hy+2, hrWhite)
			case vBack:
				for _, x := range []int{8, 15} {
					b.set(x+hd, hy+2, hrWhite)
					b.set(x+hd, hy+3, hrWhite)
				}
				b.set(7+hd, hy+1, hrWhite)
				b.set(16+hd, hy+1, hrWhite)
				b.set(7+hd, hy+5, hrPink)
				b.set(16+hd, hy+5, hrPink)
			default:
				// the white A
				row(hy, 11+hd, 12+hd, hrWhite)
				b.set(10+hd, hy+1, hrWhite)
				b.set(13+hd, hy+1, hrWhite)
				row(hy+2, 10+hd, 13+hd, hrWhite)
				b.set(10+hd, hy+3, hrWhite)
				b.set(13+hd, hy+3, hrWhite)
				// wings
				b.set(8+hd, hy+2, hrWhite)
				b.set(15+hd, hy+2, hrWhite)
				b.set(7+hd, hy+1, hrWhite)
				b.set(16+hd, hy+1, hrWhite)
				b.set(8+hd, hy+3, hrWhite)
				b.set(15+hd, hy+3, hrWhite)
				// face opening and pink ears
				row(hy+5, 9+hd, 14+hd, pal.skin)
				row(hy+6, 10+hd, 13+hd, pal.skin)
				row(hy+7, 10+hd, 13+hd, pal.skin)
				row(hy+8, 10+hd, 13+hd, pal.skin)
				row(hy+9, 10+hd, 13+hd, pal.skin)
				b.set(10+hd, hy+5, hrEye)
				b.set(13+hd, hy+5, hrEye)
				b.set(7+hd, hy+5, hrPink)
				b.set(16+hd, hy+5, hrPink)
				b.set(7+hd, hy+6, hrPink)
				b.set(16+hd, hy+6, hrPink)
				mouth(hy+8, 11+hd, 12+hd)
			}
		case heroLogan:
			B, BD, Y := hrLogB, hrLogBD, hrLogY
			if vw == vBack {
				neck(B)
			} else {
				neck(pal.skin)
			}
			shape(B)
			fin := func(x, dir int) { // tall pointed fin, yellow inside a blue edge
				for r := 0; r <= 6; r++ {
					w := (r + 1) / 2
					for i := 0; i <= w; i++ {
						xx := x - dir*(3-r/2) + dir*i
						col := Y
						if i == 0 {
							col = BD
						}
						b.set(xx, hy-3+r, col)
					}
				}
			}
			switch vw {
			case vSide:
				fin(13+hd, -1)
				row(hy+6, 12+hd, 15+hd, pal.skin)
				row(hy+7, 12+hd, 15+hd, pal.skin)
				row(hy+8, 12+hd, 15+hd, pal.skin)
				row(hy+9, 12+hd, 14+hd, pal.skin)
				for r := 5; r <= 8; r++ {
					b.set(11+hd, hy+r, hrBlack) // sideburn
				}
				row(hy+4, 14+hd, 15+hd, hrWhite)
				b.set(14+hd, hy+3, hrBlack)
				b.set(16+hd, hy+6, pal.skin)
				mouth(hy+8, 14+hd, 15+hd)
			case vBack:
				fin(8+hd, 1)
				fin(15+hd, -1)
				row(hy+8, 9+hd, 14+hd, hrBlack) // hair under the mask
				row(hy+9, 10+hd, 13+hd, hrBlack)
			default:
				fin(8+hd, 1)
				fin(15+hd, -1)
				row(hy+3, 9+hd, 10+hd, hrBlack) // frowning brows
				row(hy+3, 13+hd, 14+hd, hrBlack)
				b.set(11+hd, hy+4, hrBlack)
				b.set(12+hd, hy+4, hrBlack)
				row(hy+4, 9+hd, 10+hd, hrWhite)
				row(hy+4, 13+hd, 14+hd, hrWhite)
				row(hy+6, 9+hd, 14+hd, pal.skin)
				row(hy+7, 8+hd, 15+hd, pal.skin)
				row(hy+8, 8+hd, 15+hd, pal.skin)
				row(hy+9, 9+hd, 14+hd, pal.skin)
				for r := 6; r <= 8; r++ {
					b.set(8+hd, hy+r, hrBlack) // sideburns
					b.set(15+hd, hy+r, hrBlack)
				}
				b.set(9+hd, hy+6, hrBlack)
				b.set(14+hd, hy+6, hrBlack)
				if open {
					mouth(hy+8, 10+hd, 13+hd)
				} else {
					row(hy+8, 10+hd, 13+hd, hrWhite) // a growl
					b.set(10+hd, hy+8, hrMouth)
					b.set(13+hd, hy+8, hrMouth)
				}
			}
		case heroDeadpool:
			R, K := hrDPRed, hrBlack
			neck(R)
			shape(R)
			switch vw {
			case vSide:
				row(hy+3, 13+hd, 15+hd, K)
				row(hy+4, 12+hd, 15+hd, K)
				row(hy+5, 13+hd, 15+hd, K)
				b.set(14+hd, hy+4, hrWhite)
				b.set(15+hd, hy+4, hrWhite)
				b.set(16+hd, hy+6, R)
				row(hy+8, 14+hd, 15+hd, hrDPRedD)
				b.set(10+hd, hy+3, hl(R))
			case vBack:
				for y := hy + 1; y <= hy+9; y++ {
					b.set(11+hd+(y%2), y, hrDPRedD) // seam
				}
			default:
				for _, fl := range []bool{false, true} {
					put := func(x, y int, c color.RGBA) {
						if fl {
							x = mir(x)
						}
						b.set(x+hd, y, c)
					}
					put(9, hy+3, K)
					put(10, hy+3, K)
					for x := 8; x <= 10; x++ {
						put(x, hy+4, K)
						put(x, hy+5, K)
					}
					put(9, hy+6, K)
					put(9, hy+4, hrWhite)
					put(10, hy+5, hrWhite)
					put(9, hy+5, hrWhite)
				}
				b.set(11+hd, hy+1, hl(R))
				b.set(12+hd, hy+2, hl(R))
				if open {
					row(hy+8, 11+hd, 12+hd, hrDPRedD)
				}
			}
		}
	}

	// ---- arms, gloves and fists ----
	arm := func(sx float64, l Limb, sign float64, id uint8, dark bool) (int, int) {
		b.cur = id
		sy := float64(ty0 + 1)
		if big {
			sy = float64(ty0 + 2)
		}
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
		// a 2px margin (the outline stays inside the box); Hulk's shoulders need the whole width
		lo, hi := 2.0, float64(22-armW)
		if big {
			lo, hi = 1, float64(23-armW)
		}
		sx = clampF(sx, lo, hi)
		hxT := clampF(sx+tx, lo, hi)
		mx, my, ex, ey := ik2(sx, sy, hxT, math.Max(sy+ty, 2), armSeg, bx, by)
		ex, ey = clampF(ex, lo, hi), math.Max(ey, 2)
		mx, my = clampF(mx, lo, hi), math.Max(my, 2)
		thickLine(b, sx, sy, mx, my, armW, dk(pal.arm, dark))
		fore := pal.glove
		if k == heroSuperman || k == heroHulk {
			fore = pal.arm
		}
		thickLine(b, mx, my, ex, ey, armW, dk(fore, dark))
		if pal.glove != pal.arm && k != heroSuperman { // glove cuff
			b.rect(int(math.Floor(mx)), int(math.Floor(my)), armW, 1, dk(hl(pal.glove), dark))
			if k == heroBatman { // the fin and its yellow trim
				fx := int(math.Floor(mx)) + armW
				if sign < 0 || vw == vSide {
					fx = int(math.Floor(mx)) - 1
				}
				if vw != vSide {
					b.set(clampI(fx, 1, 22), int(math.Floor(my))+1, dk(hrBlack, dark))
				}
				b.set(int(math.Floor(mx)), int(math.Floor(my)), dk(hrYellow, dark))
			}
		}
		thickLine(b, sx, sy, sx, sy+1, armW, dk(hl(pal.arm), dark)) // shoulder
		if big {
			thickLine(b, math.Min(sx+1, 22), sy+2, math.Min(mx+1, 22), my, 1, dk(hrHulkD, dark)) // bicep line
		}
		ix, iy := int(math.Floor(ex)), int(math.Floor(ey))
		gx := ix
		if vw != vSide && sign < 0 {
			gx = ix - 1
		}
		gx = clampI(gx, int(lo), int(hi)+armW-fist)
		b.rect(gx, iy, fist, fist, dk(pal.hand, dark))
		b.rect(gx, iy+fist-1, fist, 1, dk(sd(pal.hand), dark))
		b.set(gx+1, iy+1, dk(hl(pal.hand), dark))
		if id == 3 && k == heroLogan { // three claws on the other fist (the held claw is the weapon)
			for i, col := range []color.RGBA{hrSilver, hrSilverD, hrSilver} {
				if vw == vSide {
					b.line(gx+fist-1, iy+i, minI(gx+fist+3, 21), iy+i+1, dk(col, dark))
				} else {
					b.line(gx+i, iy+fist, clampI(gx+i+(i-1), 2, 21), iy+fist+4-abs(i-1), dk(col, dark))
				}
			}
		}
		if id == 3 && k == heroCap && vw == vFront { // the round shield on the forearm
			cx := clampF(float64(gx)+1.5+sign*1, 4.5, 19.5)
			cy := math.Max(float64(iy)-1, 4.5)
			b.ellipse(cx, cy, 3.6, 3.6, dk(hrRed, dark))
			b.ellipse(cx, cy, 2.6, 2.6, dk(hrWhite, dark))
			b.ellipse(cx, cy, 1.7, 1.7, dk(hrCapBlue, dark))
			b.set(int(math.Floor(cx)), int(math.Floor(cy)), dk(hrWhite, dark))
		}
		if vw == vSide { // the held object (a knife while stabbing) stays inside the box like everybody's
			return clampI(ix, 3, 12), iy + 1
		}
		return clampI(ix, 3, 18), iy + 1
	}
	shieldBack := func() { // Cap carries the shield on his back when seen from behind or the side
		if k != heroCap {
			return
		}
		b.cur = 2
		if vw == vBack {
			cx, cy := 12+float64(ox), float64(ty0)+4.5
			b.ellipse(cx, cy, 4.6, 4.6, hrRed)
			b.ellipse(cx, cy, 3.5, 3.5, hrWhite)
			b.ellipse(cx, cy, 2.4, 2.4, hrRed)
			b.ellipse(cx, cy, 1.5, 1.5, hrCapBlue)
			b.set(11+ox, ty0+4, hrWhite)
			b.set(12+ox, ty0+4, hrWhite)
		} else if vw == vSide {
			c := 11 + ox
			for y := ty0; y <= ty0+8; y++ {
				b.set(c-4, y, hrRed)
				b.set(c-5, y, hrWhite)
			}
			b.set(c-5, ty0+4, hrCapBlue)
			b.set(c-6, ty0+4, hrCapBlue)
			b.set(c-6, ty0+3, hrWhite)
			b.set(c-6, ty0+5, hrWhite)
		}
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

	// shoulders (front/back) and hips
	lsx, rsx := 4.0, 18.0
	lhx, rhx := 8.0, 13.0
	if big {
		lsx, rsx = 1.0, 19.0
		lhx, rhx = 7.0, 13.0
	}
	switch vw {
	case vSide:
		cx := float64(11 + ox)
		cape()
		arm(cx-0.5, p.LA, 1, 3, true)
		leg(9.5, p.LF, 1, 5, true)
		leg(10.5, p.RF, 1, 6, false)
		torsoSide()
		details()
		shieldBack()
		head()
		hx0, hy0 := arm(cx, p.RA, 1, 4, false)
		held(hx0, hy0, 4)
	case vBack:
		leg(rhx, p.LF, 1, 5, false)
		leg(lhx, p.RF, -1, 6, false)
		torsoFront()
		details()
		cape()
		shieldBack()
		head()
		arm(rsx+float64(ox), p.LA, 1, 3, false)
		hx0, hy0 := arm(lsx+float64(ox), p.RA, -1, 4, false)
		held(hx0, hy0, 4)
	default:
		cape()
		leg(lhx, p.LF, -1, 5, false)
		leg(rhx, p.RF, 1, 6, false)
		torsoFront()
		details()
		head()
		arm(lsx+float64(ox), p.LA, -1, 3, false)
		hx0, hy0 := arm(rsx+float64(ox), p.RA, 1, 4, false)
		held(hx0, hy0, 4)
	}
	return b
}

// hulkHead: a smallish square green head sunk between the shoulders, messy black hair, heavy
// black brows in a frown and a clenched grimace.
func hulkHead(b *pixBuf, vw, hy, hd int, open bool) {
	G, GD, H := hrHulk, hrHulkD, hrHulkHr
	row := func(y, x0, x1 int, c color.RGBA) {
		for x := x0; x <= x1; x++ {
			b.set(x+hd, y, c)
		}
	}
	if vw == vSide {
		for r := 0; r <= 7; r++ {
			row(hy+r, 9, 15, G)
		}
		row(hy+7, 10, 15, GD) // jaw
		row(hy-1, 9, 14, H)
		row(hy, 8, 15, H)
		row(hy+1, 8, 12, H)
		row(hy+2, 8, 10, H)
		b.set(7, hy+1, H)
		b.set(11, hy-2, H)
		b.set(13, hy-2, H)
		row(hy+2, 13, 15, H) // brow
		b.set(15, hy+3, hrEye)
		b.set(16, hy+4, G) // nose
		row(hy+6, 14, 15, hrWhite)
		b.set(13, hy+6, hrMouth)
		b.set(11, hy+4, GD) // ear
		return
	}
	for r := 0; r <= 7; r++ {
		row(hy+r, 8, 15, G)
	}
	row(hy+7, 9, 14, G)
	if vw == vBack {
		row(hy-1, 9, 14, H)
		for r := 0; r <= 4; r++ {
			row(hy+r, 8, 15, H)
		}
		for _, x := range []int{8, 10, 13, 15} {
			b.set(x+hd, hy+5, H)
		}
		b.set(10+hd, hy-2, H)
		b.set(13+hd, hy-2, H)
		return
	}
	// messy hair: a ragged fringe and spikes
	row(hy-1, 9, 14, H)
	row(hy, 8, 15, H)
	row(hy+1, 8, 15, H)
	for _, x := range []int{8, 9, 11, 14, 15} {
		b.set(x+hd, hy+2, H)
	}
	b.set(10+hd, hy-2, H)
	b.set(12+hd, hy-2, H)
	b.set(14+hd, hy-2, H)
	b.set(7+hd, hy+1, H)
	b.set(16+hd, hy+1, H)
	// brows in a V, small fierce eyes
	b.set(9+hd, hy+3, H)
	b.set(10+hd, hy+3, H)
	b.set(11+hd, hy+4, H)
	b.set(14+hd, hy+3, H)
	b.set(13+hd, hy+3, H)
	b.set(12+hd, hy+4, H)
	b.set(9+hd, hy+4, hrWhite)
	b.set(10+hd, hy+4, hrEye)
	b.set(14+hd, hy+4, hrWhite)
	b.set(13+hd, hy+4, hrEye)
	b.set(11+hd, hy+5, GD) // nose
	b.set(12+hd, hy+5, GD)
	if open {
		row(hy+6, 10, 13, hrMouth)
		row(hy+7, 10, 13, shade(hrMouth, 0.7))
		b.set(10+hd, hy+6, hrWhite)
		b.set(13+hd, hy+6, hrWhite)
	} else {
		row(hy+6, 10, 13, hrWhite) // clenched teeth
		b.set(9+hd, hy+6, GD)
		b.set(14+hd, hy+6, GD)
		row(hy+7, 10, 13, GD)
	}
}
