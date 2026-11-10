package main

import (
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// Neo and the Joker (slugs "neo" and "joker") have their own procedural drawing (genNjg), like
// Mortadelo or Crash: the generic body cannot give them their long coats. Our own pixel-art
// parodies, drawn close to the user's references:
//   - Neo: tall, all in black: short black hair combed back, small oval black sunglasses, pale
//     skin, a very long black coat with a high mandarin collar, closed down to the waist and
//     open below down to the ankles; its tails swing and trail behind while he walks. Black
//     trousers and shiny black shoes.
//   - Joker: green hair combed back with loose locks, a white face, a huge red grin with teeth,
//     yellow-green eyes in dark sockets, a purple suit (jacket with tails down to the thigh),
//     green shirt, yellow waistcoat, dark tie, white gloves with black stripes and black and
//     white shoes.
// Only the drawing changes: hitbox, footprint, stats and collisions are the ones of every guest.
// Animation is the usual one (walk, run, dances...) read from the pose.
// Standing rows: skull 3..12, collar 12..14, coat 14..25, tails down to 38 (Neo) or 31 (Joker),
// legs 26..39, shoes 40..42.
// ---------------------------------------------------------------------------

const (
	njgNone = iota
	njgNeo
	njgJoker
)

func njgKind(s *CharSpec) int {
	if s == nil {
		return njgNone
	}
	switch s.Name {
	case "neo":
		return njgNeo
	case "joker":
		return njgJoker
	}
	return njgNone
}

// isNjg: drawn here (the two, and the four Ghostbusters further down).
func isNjg(s *CharSpec) bool { return njgKind(s) != njgNone || isGbArt(s) }

var (
	// Neo
	ngCoat  = hx(0x16161c)
	ngCoatL = hx(0x30303c)
	ngCoatH = hx(0x4c4c5c)
	ngCoatD = hx(0x08080a)
	ngTrou  = hx(0x1e1e26)
	ngTrouL = hx(0x34343e)
	ngShoe  = hx(0x0a0a0e)
	ngShine = hx(0x8a8a9a)
	ngSkin  = hx(0xf2dccc)
	ngSkinD = hx(0xcaa898)
	ngSkinL = hx(0xfff0e6)
	ngHair  = hx(0x0e0c10)
	ngHairL = hx(0x3a3644)
	ngLens  = hx(0x050507)
	ngGlint = hx(0x7a8a9c)
	ngLip   = hx(0xa87a70)
	ngMouth = hx(0x5a2a26)
	// Joker
	jkCoat  = hx(0x6a2a8c)
	jkCoatL = hx(0x9048b0)
	jkCoatD = hx(0x441a5c)
	jkTrou  = hx(0x58227a)
	jkTrouL = hx(0x7a3a9a)
	jkShirt = hx(0x3aa040)
	jkShirD = hx(0x22702a)
	jkVest  = hx(0xf0c030)
	jkVestD = hx(0xb88a18)
	jkTie   = hx(0x1e4a3a)
	jkSkin  = hx(0xf6f6f2)
	jkSkinD = hx(0xc8ccd2)
	jkHair  = hx(0x3c9c38)
	jkHairL = hx(0x78d060)
	jkHairD = hx(0x1e5a22)
	jkSock  = hx(0x3a2440)
	jkIris  = hx(0xd0d838)
	jkRed   = hx(0xd8202c)
	jkRedD  = hx(0x8e1018)
	jkTeeth = hx(0xfcfcf0)
	jkGlove = hx(0xf8f8f4)
	jkGlovS = hx(0x202024)
	jkShoeW = hx(0xf4f4f0)
	jkShoeK = hx(0x121216)
)

// njgLook: the colours and lengths of one of the two.
type njgLook struct {
	coat, coatL, coatD, trou, trouL color.RGBA
	tail                            int // rows of coat below the hips
}

func njgLookOf(k int) njgLook {
	if k == njgJoker {
		return njgLook{jkCoat, jkCoatL, jkCoatD, jkTrou, jkTrouL, 6}
	}
	return njgLook{ngCoat, ngCoatL, ngCoatD, ngTrou, ngTrouL, 13}
}

func genNjg(s *CharSpec, vw int, p Pose) *pixBuf {
	if isGbArt(s) {
		return genGbArt(s, vw, p)
	}
	k := njgKind(s)
	lk := njgLookOf(k)
	b := newPix(charW, charH)
	if trackParts {
		b.pid = make([]uint8, charW*charH)
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
	hipY := 25 + bob
	ty0 := hipY - 11 // shoulders
	hy := ty0 - 11   // skull top
	open := p.Open || p.Anim == AnimLaugh || p.Anim == AnimCheer || p.Anim == AnimPanic
	L, R := 11+hd, 12+hd
	hem := minI(hipY+lk.tail, 38)
	if p.Sit {
		hem = minI(hem, hipY+5)
	}
	// how much the tails fly: the legs apart while walking or running
	spread := math.Abs(p.LF.Z-p.RF.Z) + math.Abs(p.LF.X-p.RF.X)*0.5
	flap := clampI(tiaRnd(spread*0.3), 0, 3)
	if p.Anim == AnimRun {
		flap = clampI(flap+1, 0, 4)
	}
	wave := 0
	if flap > 0 {
		wave = p.F % 2
	}
	footFx := func(l Limb, sign float64) float64 { // sideways foot offset in front/back views
		return sign * clampF((l.X-1.5)*0.8, -2, 2)
	}

	leg := func(hx0 float64, l Limb, sign float64, id uint8, dark bool) {
		b.cur = id
		mx, my, ex, ey, top := tiaLegIK(vw, p, hx0, float64(hipY+1), l, sign, 0.9, 6.9, 40)
		thickLine(b, hx0, float64(hipY+1), mx, my, 2, tiaDk(lk.trou, dark))
		thickLine(b, mx, my, ex, ey, 2, tiaDk(lk.trou, dark))
		thickLine(b, hx0, float64(hipY+2), mx, my, 1, tiaDk(lk.trouL, dark)) // crease
		ix := int(math.Floor(ex))
		up, toe, shine := ngShoe, ngShoe, ngShine
		if k == njgJoker {
			up, toe, shine = jkShoeW, jkShoeK, hx(0x50505a)
			b.set(ix, top-1, tiaDk(jkSock, dark))
		}
		if vw == vSide {
			b.rect(ix, top, 3, 1, tiaDk(up, dark))
			b.rect(ix-1, top+1, 6, 1, tiaDk(toe, dark))
			b.rect(ix-1, top+2, 6, 1, tiaDk(ngShoe, dark))
			b.rect(ix+2, top+1, 2, 1, tiaDk(shine, dark))
			if k == njgJoker {
				b.rect(ix-1, top+1, 3, 1, tiaDk(up, dark))
			}
			return
		}
		x0 := ix - 1
		if sign > 0 {
			x0 = ix
		}
		b.rect(x0, top, 3, 1, tiaDk(up, dark))
		b.rect(x0-1+boolI(sign > 0), top+1, 4, 2, tiaDk(toe, dark))
		if vw == vFront {
			b.set(x0+boolI(sign > 0), top+1, tiaDk(shine, dark)) // toe shine
		}
	}

	// the coat below the hips (drawn over the legs, open in front)
	tails := func() {
		b.cur = 2
		rows := hem - hipY
		if rows <= 0 {
			return
		}
		switch vw {
		case vSide:
			c := 11 + ox
			for r := 1; r <= rows; r++ {
				t := float64(r) / float64(rows)
				back := c - 2 - int(float64(flap+1)*t*1.6+0.5)
				if r > rows/2 && (r+wave)%3 == 0 {
					back-- // the hem ripples
				}
				front := c + 1 - int(t*1.5)
				if flap == 0 {
					front = c + 2
					back = c - 2 - r/6
				}
				back = maxI(back, 1)
				for x := back; x <= front; x++ {
					col := lk.coat
					switch {
					case x == back:
						col = lk.coatD
					case x == front:
						col = lk.coatL
					}
					b.set(x, hipY+r, col)
				}
				if k == njgNeo && r%4 == 2 && flap > 0 {
					b.set(back+1, hipY+r, ngCoatH) // the wind lights the folds
				}
			}
		default:
			ls := footFx(p.LF, -1)
			rs := footFx(p.RF, 1)
			if vw == vBack {
				ls, rs = footFx(p.RF, -1), footFx(p.LF, 1)
			}
			for r := 1; r <= rows; r++ {
				t := float64(r) / float64(rows)
				spreadOut := int(t*2.2 + 0.5)
				dl := int(math.Floor(ls*t + 0.5))
				dr := int(math.Floor(rs*t + 0.5))
				if r > rows/2 && flap > 0 && (r+wave)%3 == 0 {
					dl--
					dr++
				}
				lx0, lx1 := 8-spreadOut+dl+ox, 10+dl+ox
				rx0, rx1 := 13+dr+ox, 15+spreadOut+dr+ox
				if vw == vBack { // closed at the back, a vent opens below
					lx1, rx0 = 11+dl+ox, 12+dr+ox
					if r > rows/3 && flap > 0 {
						lx1--
					}
				} else if k == njgNeo {
					lx1 -= r / 7 // the front opens wider down the legs
					rx0 += r / 7
				}
				lx0 = maxI(lx0, 1)
				rx1 = minI(rx1, charW-2)
				for x := lx0; x <= lx1; x++ {
					col := lk.coat
					switch {
					case x == lx0:
						col = lk.coatL
					case x == lx1 && vw == vFront:
						col = lk.coatL // the lining edge
					}
					b.set(x, hipY+r, col)
				}
				for x := rx0; x <= rx1; x++ {
					col := lk.coat
					switch {
					case x == rx1:
						col = lk.coatD
					case x == rx0 && vw == vFront:
						col = lk.coatL
					}
					b.set(x, hipY+r, col)
				}
			}
		}
	}

	torso := func() {
		b.cur = 2
		if vw == vSide {
			c := 11 + ox
			for r := 0; r <= hipY-ty0; r++ {
				x0, x1 := c-2, c+2
				if r == 0 {
					x0, x1 = c-1, c+1
				}
				for x := x0; x <= x1; x++ {
					col := lk.coat
					switch {
					case x == x0:
						col = lk.coatD
					case x == x1 || r == 0:
						col = lk.coatL
					}
					b.set(x, ty0+r, col)
				}
			}
			if k == njgJoker {
				b.rect(c+2, ty0+1, 1, 6, jkVest) // waistcoat and shirt show at the front
				b.set(c+2, ty0, jkShirt)
				b.set(c+3, ty0+1, jkTie)
				b.set(c+3, ty0+2, jkTie)
			} else {
				b.set(c+1, ty0+3, ngCoatH)
				b.set(c+1, ty0+4, ngCoatH)
			}
			// the high collar round the neck
			b.rect(c-1, ty0-2, 3, 2, lk.coat)
			b.set(c-1, ty0-2, lk.coatL)
			if k == njgJoker {
				b.set(c+1, ty0-1, jkShirt)
			}
			return
		}
		for r := 0; r <= hipY-ty0; r++ {
			x0, x1 := 8, 15
			if r <= 1 {
				x0, x1 = 7, 16 // square shoulders
			}
			if r >= 8 {
				x0, x1 = 9, 14 // fitted waist
			}
			for x := x0; x <= x1; x++ {
				col := lk.coat
				switch {
				case r == 0 || x == x0:
					col = lk.coatL
				case x == x1:
					col = lk.coatD
				}
				b.set(x+ox, ty0+r, col)
			}
		}
		if vw == vFront {
			if k == njgNeo {
				b.rect(12+ox, ty0+1, 1, hipY-ty0, ngCoatD) // the closure
				for _, r := range []int{3, 6, 9} {
					b.set(12+ox, ty0+r, ngCoatL) // hidden buttons
				}
				b.rect(9+ox, ty0+2, 1, 4, ngCoatH) // the sheen of the leather
				b.set(15+ox, ty0+3, ngCoatL)
				// mandarin collar standing up to the jaw, a little gap in front
				b.rect(9+ox, ty0-2, 6, 2, ngCoat)
				b.rect(9+ox, ty0-2, 6, 1, ngCoatL)
				b.set(11+ox, ty0-2, ngSkinD)
				b.set(12+ox, ty0-2, ngSkinD)
			} else {
				// lapels, the yellow waistcoat, the green shirt and the tie in the V
				for r := 0; r <= 7; r++ {
					hw := 3 - r/3
					if hw < 1 {
						hw = 1
					}
					b.rect(12-hw+ox, ty0+r, hw*2, 1, jkVest)
					b.set(11-hw+ox, ty0+r, jkCoatL) // lapel edges
					b.set(12+hw+ox, ty0+r, jkCoatD)
				}
				b.rect(10+ox, ty0, 4, 1, jkShirt)
				b.rect(11+ox, ty0+1, 2, 1, jkShirt)
				b.set(10+ox, ty0, jkShirD)
				b.set(13+ox, ty0, jkShirD)
				b.rect(11+ox, ty0+1, 2, 1, jkTie) // the knot
				b.rect(11+ox, ty0+2, 2, 3, jkTie)
				b.set(12+ox, ty0+5, jkTie)
				b.set(11+ox, ty0+6, jkVestD) // waistcoat buttons
				b.set(11+ox, ty0+8, jkCoatD)
				b.set(12+ox, ty0+9, hx(0x2a2a30))   // jacket button
				b.rect(14+ox, ty0+3, 2, 1, jkShirt) // a flower in the pocket
				b.rect(9+ox, ty0-1, 6, 1, jkShirt)  // shirt collar
				b.set(9+ox, ty0-1, jkShirD)
			}
		} else {
			b.rect(11+ox, ty0+3, 1, hipY-ty0-2, lk.coatD) // back seam
			b.rect(9+ox, ty0-2, 6, 2, lk.coat)            // the collar from behind
			b.rect(9+ox, ty0-2, 6, 1, lk.coatL)
			if k == njgJoker {
				b.rect(10+ox, ty0-1, 4, 1, jkShirt)
			}
		}
	}

	head := func() {
		b.cur = 1
		if k == njgJoker {
			njgJokerHead(b, vw, hy, L, R, hd, open)
			return
		}
		skin := ngSkin
		switch vw {
		case vSide:
			b.ellipse(11.5+float64(hd), float64(hy)+5.0, 3.4, 4.7, skin)
			b.rect(L-1, hy+4, 2, 2, ngSkinD) // ear
			// short hair combed back, tight to the skull
			b.ellipse(10.6+float64(hd), float64(hy)+3.0, 3.4, 3.0, ngHair)
			b.rect(L-3, hy+4, 2, 3, ngHair)
			b.rect(L-1, hy, 3, 1, ngHairL) // the shine of the gel
			b.set(L-2, hy+1, ngHairL)
			b.rect(L, hy+2, 4, 1, ngHair)   // hairline
			b.rect(L+1, hy+4, 3, 1, ngLens) // the oval shades and the arm
			b.rect(R+1, hy+5, 2, 1, ngLens)
			b.set(R+2, hy+4, ngGlint)
			b.set(L, hy+4, ngLens)
			b.set(R+3, hy+6, ngSkinD) // nose
			if open {
				b.rect(R+1, hy+8, 2, 1, ngMouth)
			} else {
				b.set(R+2, hy+8, ngLip)
			}
			b.rect(L+1, hy+9, 3, 1, ngSkinD) // jaw
		case vBack:
			b.ellipse(11.5+float64(hd), float64(hy)+5.0, 3.5, 4.7, ngHair)
			b.rect(L-4, hy+4, 1, 2, ngSkinD) // ears
			b.rect(R+4, hy+4, 1, 2, ngSkinD)
			b.rect(L-1, hy+1, 3, 1, ngHairL)
			b.set(L-2, hy+2, ngHairL)
			b.rect(L-1, hy+9, 4, 1, skin) // the nape
		default:
			b.ellipse(11.5+float64(hd), float64(hy)+5.0, 3.4, 4.7, skin)
			b.rect(R+3, hy+3, 1, 5, ngSkinD) // shaded cheek
			b.rect(L-4, hy+4, 1, 2, ngSkin)  // ears
			b.rect(R+4, hy+4, 1, 2, ngSkinD)
			// hair: short, combed straight back, a clean hairline
			b.ellipse(11.5+float64(hd), float64(hy)+2.0, 3.6, 2.2, ngHair)
			b.rect(L-3, hy+2, 1, 2, ngHair) // temples
			b.rect(R+3, hy+2, 1, 2, ngHair)
			b.rect(L-1, hy, 3, 1, ngHairL)
			b.set(R+1, hy+1, ngHairL)
			// small oval black sunglasses on a thin bridge
			b.rect(L-3, hy+4, 3, 1, ngLens)
			b.rect(R+1, hy+4, 3, 1, ngLens)
			b.rect(L-2, hy+5, 2, 1, ngLens)
			b.rect(R+1, hy+5, 2, 1, ngLens)
			b.rect(L, hy+4, 2, 1, ngLens)
			b.set(L-3, hy+4, ngGlint)
			b.set(R+1, hy+4, ngGlint)
			b.set(L-1, hy+3, ngSkinL)
			b.set(R, hy+6, ngSkinD) // nose
			if open {
				b.rect(L, hy+8, 2, 1, ngMouth)
			} else {
				b.rect(L, hy+8, 2, 1, ngLip)
			}
			b.rect(L-1, hy+9, 4, 1, ngSkinD) // jaw
		}
	}

	arm := func(sx float64, l Limb, sign float64, id uint8, dark bool) (int, int) {
		b.cur = id
		sy := float64(ty0 + 1)
		mx, my, ex, ey := tiaArmIK(vw, sx, sy, l, sign, 1.15, 5.2)
		thickLine(b, sx, sy, mx, my, 2, tiaDk(lk.coat, dark))
		thickLine(b, mx, my, ex, ey, 2, tiaDk(lk.coat, dark))
		thickLine(b, sx, sy, mx, my, 1, tiaDk(lk.coatL, dark))
		ix, iy := int(math.Floor(ex)), int(math.Floor(ey))
		if k == njgJoker { // white glove with black stripes on the back
			b.rect(ix, iy, 2, 1, tiaDk(jkShirt, dark)) // shirt cuff
			b.rect(ix, iy+1, 2, 3, tiaDk(jkGlove, dark))
			b.set(ix+boolI(sign > 0), iy+2, tiaDk(jkGlovS, dark))
		} else {
			b.rect(ix, iy+1, 2, 2, tiaDk(ngSkin, dark))
			b.set(ix+1, iy+2, tiaDk(ngSkinD, dark))
		}
		return ix, iy + 1
	}

	switch vw {
	case vSide:
		cx := float64(11 + ox)
		backL, frontL := p.LF, p.RF
		if p.LF.Z > p.RF.Z {
			backL, frontL = p.RF, p.LF
		}
		arm(cx-0.5, p.LA, 1, 3, true)
		leg(10, backL, 1, 5, true)
		tails()
		leg(11, frontL, 1, 6, false)
		torso()
		head()
		x, y := arm(cx, p.RA, 1, 4, false)
		tiaHeld(b, s, p, vw, x, y, 4)
	case vBack:
		leg(13, p.LF, 1, 5, false)
		leg(9, p.RF, -1, 6, false)
		tails()
		torso()
		head()
		arm(float64(16+ox), p.LA, 1, 3, false)
		x, y := arm(float64(6+ox), p.RA, -1, 4, false)
		tiaHeld(b, s, p, vw, x, y, 4)
	default:
		leg(9, p.LF, -1, 5, false)
		leg(13, p.RF, 1, 6, false)
		tails()
		torso()
		head()
		arm(float64(6+ox), p.LA, -1, 3, false)
		x, y := arm(float64(16+ox), p.RA, 1, 4, false)
		tiaHeld(b, s, p, vw, x, y, 4)
	}
	return b
}

// njgJokerHead: green hair combed back with loose locks, a white face, dark sockets with
// yellow-green eyes and the huge red grin full of teeth.
func njgJokerHead(b *pixBuf, vw, hy, L, R, hd int, open bool) {
	switch vw {
	case vSide:
		b.ellipse(11.5+float64(hd), float64(hy)+5.0, 3.4, 4.7, jkSkin)
		b.rect(L-1, hy+4, 2, 2, jkSkinD) // ear
		b.ellipse(10.4+float64(hd), float64(hy)+3.0, 3.5, 3.0, jkHair)
		b.rect(L-3, hy+3, 2, 5, jkHair) // the long back hair
		b.set(L-4, hy+5, jkHair)
		b.set(L-4, hy+7, jkHairD) // loose locks behind
		b.set(L-3, hy+8, jkHairD)
		b.rect(L-1, hy, 4, 1, jkHairL)
		b.set(L-2, hy+1, jkHairL)
		b.set(R+2, hy+1, jkHair) // a lock falling on the brow
		b.set(R+2, hy+2, jkHairD)
		b.rect(R, hy+4, 3, 2, hx(0x3c2a44)) // dark socket
		b.set(R+1, hy+4, jkIris)
		b.set(R+3, hy+6, jkSkinD) // nose
		// the grin runs back along the cheek
		b.rect(R-1, hy+7, 4, 1, jkRed)
		b.set(R-2, hy+6, jkRed)
		b.rect(R, hy+8, 3, 1, jkRedD)
		b.rect(R, hy+7, 3, 1, jkTeeth)
		if open {
			b.rect(R, hy+8, 3, 1, hx(0x3a0a10))
			b.rect(R, hy+9, 3, 1, jkRed)
		}
	case vBack:
		b.ellipse(11.5+float64(hd), float64(hy)+5.0, 3.6, 4.8, jkHair)
		b.rect(L-4, hy+4, 1, 2, jkSkinD)
		b.rect(R+4, hy+4, 1, 2, jkSkinD)
		b.rect(L-1, hy+1, 3, 1, jkHairL)
		b.set(R+1, hy+3, jkHairL)
		b.set(L-2, hy+4, jkHairL)
		b.rect(L-2, hy+8, 1, 2, jkHairD) // locks over the collar
		b.rect(R+1, hy+8, 1, 2, jkHairD)
		b.set(L, hy+9, jkHair)
	default:
		b.ellipse(11.5+float64(hd), float64(hy)+5.0, 3.4, 4.7, jkSkin)
		b.rect(R+3, hy+3, 1, 4, jkSkinD)
		b.rect(L-4, hy+4, 1, 2, jkSkin) // ears
		b.rect(R+4, hy+4, 1, 2, jkSkinD)
		// green hair combed back, a high forehead and loose locks at the sides
		b.ellipse(11.5+float64(hd), float64(hy)+1.6, 3.8, 2.2, jkHair)
		b.rect(L-1, hy, 3, 1, jkHairL)
		b.set(R+1, hy+1, jkHairL)
		b.rect(L-4, hy+1, 1, 4, jkHair) // locks sticking out over the ears
		b.rect(R+4, hy+1, 1, 4, jkHair)
		b.set(L-5, hy+2, jkHairD)
		b.set(R+5, hy+2, jkHairD)
		b.set(L-4, hy+5, jkHairD)
		b.set(R+4, hy+5, jkHairD)
		b.set(L-1, hy+2, jkHair) // a lock on the brow
		b.set(L-2, hy+3, jkHairD)
		// dark sockets, yellow-green eyes
		b.rect(L-3, hy+4, 3, 2, hx(0x3c2a44))
		b.rect(R+1, hy+4, 3, 2, hx(0x3c2a44))
		b.set(L-2, hy+4, jkIris)
		b.set(R+2, hy+4, jkIris)
		b.set(R, hy+6, jkSkinD) // nose
		// the huge grin from cheek to cheek, teeth showing
		b.set(L-3, hy+6, jkRed)
		b.set(R+3, hy+6, jkRed)
		b.rect(L-2, hy+7, 6, 1, jkRed)
		b.rect(L-1, hy+7, 4, 1, jkTeeth)
		b.rect(L-1, hy+8, 4, 1, jkRed)
		b.set(L-2, hy+8, jkRedD)
		b.set(R+2, hy+8, jkRedD)
		if open {
			b.rect(L-1, hy+8, 4, 1, hx(0x3a0a10))
			b.rect(L-1, hy+9, 4, 1, jkRed)
		}
	}
}

// ---- the Ghostbusters ------------------------------------------------------------------------
// The four (gb_venkman, gb_stantz, gb_spengler, gb_winston) keep the generic body of their spec
// (beige overalls, each one's hair and build) with the details of the reference on top: no grey
// cape any more but a brown proton pack on the back with a black cable, a grey belt, black boots
// and coloured patches (the red and white logo on the arm, a name tag on the chest).

func isGbArt(s *CharSpec) bool {
	if s == nil {
		return false
	}
	switch s.Name {
	case "gb_venkman", "gb_stantz", "gb_spengler", "gb_winston":
		return true
	}
	return false
}

var (
	gbPack   = hx(0x6a4a2a)
	gbPackL  = hx(0x8e6a40)
	gbPackD  = hx(0x402c18)
	gbMetal  = hx(0x9a9ca4)
	gbMetalD = hx(0x5a5c64)
	gbCable  = hx(0x18181c)
	gbBelt   = hx(0x6e7078)
	gbBeltL  = hx(0xa8aab2)
	gbBoot   = hx(0x141418)
	gbLogoR  = hx(0xd8242a)
	gbLogoW  = hx(0xf6f6f2)
	gbTag    = hx(0x2a5ab0)
	gbLight  = hx(0xf0d040)
)

func genGbArt(s *CharSpec, vw int, p Pose) *pixBuf {
	plain := *s
	plain.Name = s.Name + "~" // the generic body (no famous hook answers this name)
	plain.Cape = color.RGBA{}
	plain.Shoes = gbBoot
	body := genChar(&plain, vw, p)

	bob := math.Round(p.Bob)
	if p.Sit {
		bob = math.Max(bob, 3)
	}
	hipY := int(26 + float64(s.Stature) + bob)
	top := hipY - 10
	tw := bodyWidth(s.Body)
	if vw == vSide {
		tw = maxI(3, tw-1)
	}
	c := charCX
	if vw == vSide {
		c += int(math.Round(p.Lean * 0.7))
	} else {
		c += int(math.Round(p.Sway * 0.7))
	}
	x0 := c - tw/2 // first column of the torso
	x1 := x0 + tw - 1
	pw := maxI(6, tw+1) // the pack is as wide as the back

	out := newPix(charW, charH)
	if trackParts {
		out.pid = make([]uint8, charW*charH)
	}
	pack := func(px0, py0, w int) {
		out.cur = 2
		out.rect(px0, py0, w, 10, gbPack)
		out.rect(px0, py0, w, 1, gbPackL)
		out.rect(px0, py0+9, w, 1, gbPackD)
		out.rect(px0+1, py0+1, w-2, 2, gbMetal) // the cyclotron on top
		out.rect(px0+1, py0+2, w-2, 1, gbMetalD)
		out.set(px0+1, py0+5, gbLight)
		out.set(px0+w-2, py0+5, gbLogoR)
		out.rect(px0+2, py0+6, w-4, 2, gbPackD) // the box below
	}
	under := func() { // copy the body over what is already drawn
		for i := 0; i < len(body.p); i += 4 {
			if body.p[i+3] == 0 {
				continue
			}
			copy(out.p[i:i+4], body.p[i:i+4])
			if out.pid != nil && body.pid != nil {
				out.pid[i/4] = body.pid[i/4]
			}
		}
	}
	switch vw {
	case vSide: // the pack sticks out behind (he faces right)
		pack(x0-4, top, 5)
		under()
		out.cur = 2
		for r := 0; r < 4; r++ { // the cable loops down from the pack to the hip
			out.set(x0-3+r/2, top+10+r, gbCable)
		}
		out.set(x0-1, top+1, gbCable) // shoulder strap
		out.set(x0, top+2, gbCable)
	case vBack: // the whole pack on the back
		under()
		pack(c-pw/2, top+1, pw)
		out.cur = 2
		for r := 0; r < 5; r++ {
			out.set(c+pw/2-r/2, top+11+r, gbCable)
		}
		out.rect(x0, hipY-1, tw, 1, gbBelt)
	default: // front: the pack shows over the shoulders, straps and patches on the overalls
		pack(c-pw/2, top-1, pw)
		under()
		out.cur = 2
		out.set(x0, top+1, gbCable) // the straps
		out.set(x0, top+2, gbCable)
		out.set(x1, top+1, gbCable)
		out.set(x1, top+2, gbCable)
		out.set(x0+1, top+3, gbLogoW) // the logo patch
		out.set(x0+1, top+4, gbLogoR)
		if tw >= 4 {
			out.set(x1-1, top+3, gbTag) // name tag
			out.set(x1-2, top+3, gbLogoW)
		}
	}
	if vw != vBack {
		out.cur = 2
		out.rect(x0, hipY-1, tw, 1, gbBelt) // grey belt
		out.set(c, hipY-1, gbBeltL)
	}
	return out
}
