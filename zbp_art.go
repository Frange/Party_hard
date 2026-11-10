package main

import (
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// Four famous guests with their own procedural drawing (genZbp), like Mortadelo or Crash: the
// generic body cannot give them their builds. Our own pixel-art parodies:
//   - Zipi and Zape (slugs "zipi", "zape"), the twins of the Spanish comic: children with a big
//     head, red V-neck jumper with yellow trims and a blue collar over a white shirt, black
//     shorts, blue socks fallen round the ankles and brown shoes. Zipi: blond helmet of hair, a
//     smile and his hands behind his back when he stands. Zape: straight black hair with locks
//     sticking out, a frown and clenched fists.
//   - The Pink Panther (slug "pantera_rosa"): very tall and thin, pink with a lighter belly, a
//     long snout with a red nose, yellow heavy-lidded eyes, pointed ears, thin arms, long thin
//     legs with big flat feet and a long thin tail hanging behind.
//   - Mr Bean (slug "mr_bean", as in the cartoon series): short black hair with a side parting,
//     big ears, thick brows, big eyes and a big round nose, a beige-brown tweed jacket with
//     brown elbow patches, white shirt, red tie with a pin, black trousers and black shoes; thin
//     with long legs.
// Only the drawing changes: hitbox, footprint and collisions are the ones of every guest. The
// animation is the usual one (walk, run, dances...) read from the pose.
// Standing rows: kids hair 13..14, head 15..26, jumper 26..33, shorts 34..36, shoes 40..42;
// panther ears 2, head 4..13, neck 14..15, body 16..25, legs 26..40, feet 41..42; Bean hair 3,
// head 4..14, jacket 15..28, legs 27..39, shoes 40..42.
// ---------------------------------------------------------------------------

const (
	zbpNone = iota
	zbpZipi
	zbpZape
	zbpPanther
	zbpBean
)

func zbpKind(s *CharSpec) int {
	switch s.Name {
	case "zipi":
		return zbpZipi
	case "zape":
		return zbpZape
	case "pantera_rosa":
		return zbpPanther
	case "mr_bean":
		return zbpBean
	}
	return zbpNone
}

func isZbp(s *CharSpec) bool { return zbpKind(s) != zbpNone }

func genZbp(s *CharSpec, vw int, p Pose) *pixBuf {
	switch zbpKind(s) {
	case zbpPanther:
		return genPanther(s, vw, p)
	case zbpBean:
		return genBean(s, vw, p)
	}
	return genZipiZape(s, vw, p)
}

var (
	zbSkin   = hx(0xf8cca4)
	zbSkinL  = hx(0xffe0c4)
	zbSkinD  = hx(0xd89c74)
	zbCheek  = hx(0xf0a08c)
	zbMouth  = hx(0x6a1a14)
	zbTongue = hx(0xd84848)
	zbEyeW   = hx(0xffffff)
	zbPupil  = hx(0x101014)
	zbWhite  = hx(0xf6f6f2)
	zbWhiteD = hx(0xc8ccd6)

	zbJumper  = hx(0xd42a24)
	zbJumperL = hx(0xee4a3a)
	zbJumperD = hx(0x9a1a18)
	zbTrim    = hx(0xf6d030)
	zbCollar  = hx(0x2a5ac8)
	zbShorts  = hx(0x1c1c22)
	zbShortsL = hx(0x3a3a46)
	zbSock    = hx(0x3a6ad4)
	zbSockL   = hx(0x6a96ee)
	zbShoe    = hx(0x8a5228)
	zbShoeL   = hx(0xb07040)
	zbShoeD   = hx(0x5a3014)
	zbBlond   = hx(0xf4cc38)
	zbBlondL  = hx(0xfff07a)
	zbBlondD  = hx(0xc4961c)
	zbBlack   = hx(0x16121a)
	zbBlackL  = hx(0x3c3648)

	zbPink   = hx(0xf080b0)
	zbPinkL  = hx(0xf8a6cc)
	zbPinkD  = hx(0xc8588c)
	zbPinkDD = hx(0x98386a)
	zbBelly  = hx(0xfcd2e4)
	zbNose   = hx(0xd8202a)
	zbYellow = hx(0xf8e040)

	zbTweed  = hx(0xa88a64)
	zbTweedL = hx(0xc6a882)
	zbTweedD = hx(0x7c6244)
	zbPatch  = hx(0x5a3c22)
	zbTie    = hx(0xcc2028)
	zbTieD   = hx(0x8a1418)
	zbPin    = hx(0xe8c840)
	zbTrou   = hx(0x1c1c24)
	zbTrouL  = hx(0x363642)
	zbBeanH  = hx(0x231a14)
	zbBeanHL = hx(0x4a3a2c)
	zbShoeK  = hx(0x121216)
	zbShoeKL = hx(0x3c3c48)
	zbBtn    = hx(0x4a3622)
)

// zbOval paints the pixels of an ellipse for which keep says yes (nil: all of them).
func zbOval(b *pixBuf, cx, cy, rx, ry float64, c color.RGBA, keep func(x, y int) bool) {
	for y := int(math.Floor(cy - ry)); y <= int(math.Ceil(cy+ry)); y++ {
		for x := int(math.Floor(cx - rx)); x <= int(math.Ceil(cx+rx)); x++ {
			dx := (float64(x) + 0.5 - cx) / rx
			dy := (float64(y) + 0.5 - cy) / ry
			if dx*dx+dy*dy <= 1 && (keep == nil || keep(x, y)) {
				b.set(x, y, c)
			}
		}
	}
}

// zbFrame: the shared start of the three drawings (buffer, bob, body and head shifts).
func zbFrame(vw int, p Pose, bobLo, bobHi int) (b *pixBuf, bob, ox, hd int) {
	b = newPix(charW, charH)
	if trackParts {
		b.pid = make([]uint8, charW*charH)
	}
	bob = clampI(tiaRnd(p.Bob), bobLo, bobHi)
	if p.Sit {
		bob = maxI(bob, 2)
	}
	ox = clampI(tiaRnd(p.Sway*0.5), -2, 2)
	hd = ox
	if vw == vSide {
		ox = clampI(tiaRnd(p.Lean*0.5), -1, 2)
		hd = ox + clampI(tiaRnd(p.Lean*0.3+p.HeadZ*0.5), -1, 1)
	} else if p.HeadZ != 0 {
		hd += clampI(tiaRnd(p.HeadZ*0.5), -1, 1)
	}
	return
}

func zbOpen(p Pose) bool {
	return p.Open || p.Anim == AnimLaugh || p.Anim == AnimCheer || p.Anim == AnimPanic
}

// ---- Zipi and Zape ------------------------------------------------------------------------------

func genZipiZape(s *CharSpec, vw int, p Pose) *pixBuf {
	zape := zbpKind(s) == zbpZape
	b, bob, ox, hd := zbFrame(vw, p, -2, 3)
	hipY := 34 + bob
	ty0 := hipY - 8 // shoulders
	hy := ty0 - 11  // skull top
	open := zbOpen(p)
	L, R := 11+hd, 12+hd
	// Zipi keeps his hands behind his back while he stands still (and holds nothing)
	behind := !zape && p.Anim == AnimIdle && p.Hold == HoldNone && s.Carry == HoldNone && !p.Sit

	leg := func(hx0 float64, l Limb, sign float64, id uint8, dark bool) {
		b.cur = id
		mx, my, ex, ey, top := tiaLegIK(vw, p, hx0, float64(hipY+1), l, sign, 0.5, 2.7, 40)
		thickLine(b, hx0, float64(hipY+1), mx, my, 2, tiaDk(zbSkin, dark))
		thickLine(b, mx, my, ex, ey, 2, tiaDk(zbSkin, dark))
		// the leg of the shorts covers the thigh
		thickLine(b, hx0, float64(hipY+1), hx0+(mx-hx0)*0.5, float64(hipY+1)+(my-float64(hipY+1))*0.5, 3, tiaDk(zbShorts, dark))
		ix := int(math.Floor(ex))
		if vw == vSide {
			b.rect(ix, top-2, 3, 2, tiaDk(zbSock, dark)) // the sock fallen round the ankle
			b.rect(ix, top-2, 3, 1, tiaDk(zbSockL, dark))
			b.rect(ix, top, 4, 1, tiaDk(zbShoeL, dark))
			b.rect(ix-1, top+1, 6, 1, tiaDk(zbShoe, dark))
			b.rect(ix-1, top+2, 6, 1, tiaDk(zbShoeD, dark))
			return
		}
		x0 := ix - 1 + boolI(sign > 0)
		b.rect(x0, top-2, 3, 2, tiaDk(zbSock, dark))
		b.rect(x0, top-2, 3, 1, tiaDk(zbSockL, dark))
		b.set(x0+1, top-1, tiaDk(zbSockL, dark))
		b.rect(x0, top, 3, 1, tiaDk(zbShoeL, dark))
		b.rect(x0-boolI(sign < 0), top+1, 4, 1, tiaDk(zbShoe, dark))
		b.rect(x0-boolI(sign < 0), top+2, 4, 1, tiaDk(zbShoeD, dark))
	}

	torso := func() {
		b.cur = 2
		if vw == vSide {
			c := 11 + ox
			for r := 0; r < 8; r++ {
				x0, x1 := c-2, c+2
				if r == 0 {
					x0, x1 = c-1, c+2
				}
				for x := x0; x <= x1; x++ {
					col := zbJumper
					switch {
					case r == 7:
						col = zbTrim // the yellow rib at the hem
					case x == x0:
						col = zbJumperD
					case x == x1-1 && r > 0:
						col = zbJumperL
					}
					b.set(x, ty0+r, col)
				}
			}
			b.set(c+2, ty0, zbCollar) // the collar over the white shirt, the V trimmed in yellow
			b.set(c+1, ty0, zbWhite)
			b.set(c+3, ty0, zbCollar)
			b.set(c+2, ty0+1, zbWhite)
			b.set(c+2, ty0+2, zbTrim)
			b.set(c+1, ty0+1, zbTrim)
			b.rect(c-2, hipY, 5, 2, zbShorts) // the shorts
			b.rect(c-2, hipY, 5, 1, zbShortsL)
			return
		}
		for r := 0; r < 8; r++ {
			x0, x1 := 8, 15
			if r == 0 {
				x0, x1 = 9, 14
			}
			for x := x0; x <= x1; x++ {
				col := zbJumper
				switch {
				case r == 7:
					col = zbTrim
				case x == x0:
					col = zbJumperL
				case x == x1:
					col = zbJumperD
				}
				b.set(x+ox, ty0+r, col)
			}
		}
		if vw == vFront {
			// the V of the jumper: yellow trim, white shirt inside, the blue collar on top
			b.set(9+ox, ty0, zbTrim)
			b.set(14+ox, ty0, zbTrim)
			b.set(10+ox, ty0, zbCollar)
			b.set(13+ox, ty0, zbCollar)
			b.rect(11+ox, ty0, 2, 2, zbWhite)
			b.set(10+ox, ty0+1, zbTrim)
			b.set(13+ox, ty0+1, zbTrim)
			b.set(11+ox, ty0+1, zbCollar)
			b.set(12+ox, ty0+1, zbWhiteD)
			b.rect(11+ox, ty0+2, 2, 1, zbTrim)
		} else {
			b.rect(10+ox, ty0, 4, 1, zbCollar)
			b.rect(11+ox, ty0+1, 2, 1, zbJumperD)
		}
		b.rect(8, hipY, 8, 2, zbShorts)
		b.rect(8, hipY, 8, 1, zbShortsL)
		b.set(11, hipY+1, zbShortsL)
		b.cur = 1 // the neck belongs to the head
		b.set(L, ty0-1, zbSkinD)
		b.set(R, ty0-1, zbSkinD)
	}

	hairC, hairL, hairD := zbBlond, zbBlondL, zbBlondD
	if zape {
		hairC, hairL, hairD = zbBlack, zbBlackL, zbBlack
	}
	cx := 11.5 + float64(hd)
	cy := float64(hy) + 5.5
	head := func() {
		b.cur = 1
		switch vw {
		case vSide:
			hcx := cx - 0.5
			zbOval(b, hcx, cy, 4.6, 5.4, zbSkin, nil)
			b.set(R+4, hy+7, zbSkin) // a small nose
			b.set(R+4, hy+8, zbSkinD)
			// hair: the top and the back of the head
			zbOval(b, hcx, cy-0.4, 4.9, 5.6, hairC, func(x, y int) bool {
				if zape {
					return y <= hy+3 || x <= L-1 && y <= hy+7 || x <= L-3 && y <= hy+8
				}
				return y <= hy+3 || x <= L-1 && y <= hy+8 || x <= L+2 && y == hy+4
			})
			b.rect(L-2, hy+6, 2, 2, zbSkinD) // ear
			b.set(L-2, hy+6, zbSkin)
			if zape { // locks sticking out at the back and on top
				b.set(L-5, hy+3, hairC)
				b.set(L-6, hy+4, hairC)
				b.set(L-5, hy+6, hairC)
				b.set(L-1, hy-1, hairC)
				b.set(L+1, hy-1, hairC)
				b.set(L+2, hy-2, hairC)
				b.set(R+3, hy+4, hairC) // a lock over the brow
				b.set(R+1, hy+4, hairC)
				b.set(R+2, hy+5, zbPupil) // the frown
				b.set(R+3, hy+5, zbPupil)
				b.set(R+3, hy+6, zbPupil)
				b.set(R+2, hy+6, zbEyeW)
				if open {
					b.rect(R+1, hy+9, 2, 2, zbMouth)
				} else {
					b.rect(R+1, hy+9, 2, 1, zbMouth)
					b.set(R+2, hy+10, zbMouth)
				}
			} else {
				b.set(L, hy-1, hairL)
				b.set(L+1, hy-1, hairC)
				b.rect(L-1, hy+1, 3, 1, hairL) // shine
				b.set(R+3, hy+6, zbPupil)
				b.set(R+3, hy+5, zbPupil)
				b.set(R+2, hy+8, zbCheek)
				if open {
					b.rect(R+1, hy+9, 2, 2, zbMouth)
					b.set(R+2, hy+10, zbTongue)
				} else {
					b.rect(R+1, hy+9, 2, 1, zbMouth)
					b.set(R+3, hy+8, zbMouth) // smile
				}
			}
			b.set(R+3, hy+11, zbSkinD)
		case vBack:
			zbOval(b, cx, cy, 4.7, 5.4, zbSkin, nil)
			b.rect(L-5, hy+6, 1, 2, zbSkinD)
			b.rect(R+5, hy+6, 1, 2, zbSkinD)
			zbOval(b, cx, cy-0.3, 5.0, 5.6, hairC, func(x, y int) bool { return y <= hy+8 })
			b.rect(L-1, hy+1, 4, 1, hairL)
			b.rect(L-2, hy+8, 6, 1, hairD)
			if zape {
				b.set(L-3, hy-1, hairC)
				b.set(L, hy-2, hairC)
				b.set(R+2, hy-1, hairC)
				b.set(L-5, hy+3, hairC)
				b.set(R+5, hy+3, hairC)
				b.set(L+1, hy+9, hairC)
			} else {
				b.set(L, hy-1, hairL)
			}
		default:
			zbOval(b, cx, cy, 4.7, 5.4, zbSkin, nil)
			b.rect(R+4, hy+5, 1, 5, zbSkinD) // shaded cheek
			b.rect(L-5, hy+6, 1, 2, zbSkin)  // ears
			b.rect(R+5, hy+6, 1, 2, zbSkinD)
			if zape {
				// straight black hair, a ragged fringe of locks and two sticking out on top
				zbOval(b, cx, cy-0.3, 5.0, 5.6, hairC, func(x, y int) bool {
					return y <= hy+3 || (x <= L-4 || x >= R+4) && y <= hy+5
				})
				for _, x := range []int{L - 2, L + 1, R + 2} {
					b.set(x, hy+4, hairC)
				}
				b.set(L-3, hy-1, hairC)
				b.set(L, hy-2, hairC)
				b.set(L+1, hy-1, hairC)
				b.set(R+3, hy-1, hairC)
				b.set(L-6, hy+4, hairC)
				b.set(R+6, hy+3, hairC)
				b.rect(L-1, hy+1, 3, 1, hairL)
				// the frown: brows slanting down to the nose
				b.set(L-3, hy+4, zbPupil)
				b.rect(L-2, hy+5, 2, 1, zbPupil)
				b.set(R+3, hy+4, zbPupil)
				b.rect(R+1, hy+5, 2, 1, zbPupil)
				b.set(L-2, hy+6, zbPupil)
				b.set(R+2, hy+6, zbPupil)
				b.set(L-3, hy+6, zbEyeW)
				b.set(R+3, hy+6, zbEyeW)
				b.rect(L, hy+7, 2, 1, zbSkinD)
				if open { // shouting
					b.rect(L-1, hy+9, 4, 2, zbMouth)
					b.rect(L, hy+10, 2, 1, zbTongue)
				} else {
					b.rect(L-1, hy+9, 4, 1, zbMouth)
					b.set(L-2, hy+10, zbMouth) // corners down
					b.set(R+2, hy+10, zbMouth)
				}
			} else {
				// the blond helmet: a straight fringe and the sides down over the ears
				zbOval(b, cx, cy-0.3, 5.0, 5.6, hairC, func(x, y int) bool {
					return y <= hy+3 || (x <= L-4 || x >= R+4) && y <= hy+6
				})
				b.rect(L-3, hy+4, 8, 1, hairC)
				b.set(L-2, hy+4, hairD)
				b.set(R+2, hy+4, hairD)
				b.set(L, hy+4, hairD)
				b.rect(L-2, hy+1, 4, 1, hairL) // shine
				b.set(L-3, hy+2, hairL)
				b.set(L, hy-1, hairC)
				b.set(R, hy-1, hairL)
				b.rect(L-2, hy+5, 1, 2, zbPupil) // eyes
				b.rect(R+2, hy+5, 1, 2, zbPupil)
				b.set(L-3, hy+8, zbCheek)
				b.set(R+3, hy+8, zbCheek)
				b.rect(L, hy+7, 2, 1, zbSkinD)
				if open {
					b.rect(L-1, hy+9, 4, 2, zbMouth)
					b.rect(L, hy+10, 2, 1, zbTongue)
					b.set(L-2, hy+8, zbMouth)
					b.set(R+2, hy+8, zbMouth)
				} else { // the smile
					b.rect(L-1, hy+9, 4, 1, zbMouth)
					b.set(L-2, hy+8, zbMouth)
					b.set(R+2, hy+8, zbMouth)
				}
			}
		}
	}

	hand := func(ix, iy int, dark bool) {
		if zape { // a clenched fist with the knuckles
			b.rect(ix, iy, 2, 2, tiaDk(zbSkin, dark))
			b.set(ix, iy+2, tiaDk(zbSkinD, dark))
			b.set(ix+1, iy+2, tiaDk(zbSkinD, dark))
			b.set(ix+1, iy, tiaDk(zbSkinL, dark))
			return
		}
		b.rect(ix, iy, 2, 2, tiaDk(zbSkin, dark))
		b.set(ix+1, iy+1, tiaDk(zbSkinD, dark))
	}
	arm := func(sx float64, l Limb, sign float64, id uint8, dark bool) (int, int) {
		b.cur = id
		sy := float64(ty0 + 1)
		if behind { // the hands go behind the back
			l.Y, l.X = 8.5, 0.3
			if vw == vSide {
				l.Z, l.Y = -2.5, 6
			}
		}
		if zape && p.Anim == AnimIdle { // fists clenched a little away from the body
			l.X += 0.5
		}
		mx, my, ex, ey := tiaArmIK(vw, sx, sy, l, sign, 0.75, 3.3)
		thickLine(b, sx, sy, mx, my, 2, tiaDk(zbJumper, dark))
		thickLine(b, mx, my, ex, ey, 2, tiaDk(zbJumper, dark))
		thickLine(b, mx, my, ex, ey, 1, tiaDk(zbJumperD, dark))
		ix, iy := int(math.Floor(ex)), int(math.Floor(ey))
		b.rect(ix, iy, 2, 1, tiaDk(zbTrim, dark)) // the yellow cuff
		if behind && vw != vBack {
			return ix, iy + 1 // the hand is hidden behind him
		}
		hand(ix, iy+1, dark)
		return ix, iy + 1
	}

	switch vw {
	case vSide:
		ax := float64(11 + ox)
		if behind {
			x, y := arm(ax, p.RA, 1, 4, false) // both hands behind the back, under the jumper
			_, _ = x, y
			leg(10, p.LF, 1, 5, true)
			leg(11, p.RF, 1, 6, false)
			torso()
			head()
			b.cur = 3
			b.rect(10+ox, ty0+1, 2, 4, zbJumperD) // the near arm folded back
			b.rect(9+ox, ty0+5, 2, 1, zbTrim)
			b.rect(8+ox, ty0+6, 2, 2, zbSkin)
			return b
		}
		arm(ax-0.5, p.LA, 1, 3, true)
		leg(10, p.LF, 1, 5, true)
		leg(11, p.RF, 1, 6, false)
		torso()
		head()
		x, y := arm(ax, p.RA, 1, 4, false)
		tiaHeld(b, s, p, vw, x, y, 4)
	case vBack:
		leg(13, p.LF, 1, 5, false)
		leg(9, p.RF, -1, 6, false)
		torso()
		head()
		if behind { // the hands clasped behind the back
			arm(float64(16+ox), p.LA, 1, 3, false)
			arm(float64(7+ox), p.RA, -1, 4, false)
			b.cur = 4
			b.rect(10+ox, hipY-2, 4, 2, zbSkin)
			b.rect(10+ox, hipY-1, 4, 1, zbSkinD)
			return b
		}
		arm(float64(16+ox), p.LA, 1, 3, false)
		x, y := arm(float64(7+ox), p.RA, -1, 4, false)
		tiaHeld(b, s, p, vw, x, y, 4)
	default:
		leg(9, p.LF, -1, 5, false)
		leg(13, p.RF, 1, 6, false)
		torso()
		head()
		arm(float64(7+ox), p.LA, -1, 3, false)
		x, y := arm(float64(16+ox), p.RA, 1, 4, false)
		tiaHeld(b, s, p, vw, x, y, 4)
	}
	return b
}

// ---- The Pink Panther ---------------------------------------------------------------------------

func genPanther(s *CharSpec, vw int, p Pose) *pixBuf {
	b, bob, ox, hd := zbFrame(vw, p, 0, 3) // the ears stand on row 2: never higher
	hipY := 25 + bob
	ty0 := hipY - 9 // shoulders
	hy := ty0 - 12  // skull top
	open := zbOpen(p)
	L, R := 11+hd, 12+hd
	swing := 0 // the tail sways as he walks
	if p.Anim != AnimIdle {
		swing = []int{0, 1, 0, -1}[p.F%4]
	}

	leg := func(hx0 float64, l Limb, sign float64, id uint8, dark bool) {
		b.cur = id
		mx, my, ex, ey, top := tiaLegIK(vw, p, hx0, float64(hipY+1), l, sign, 1.0, 7.6, 41)
		thickLine(b, hx0, float64(hipY+1), mx, my, 2, tiaDk(zbPink, dark))
		thickLine(b, mx, my, ex, ey, 2, tiaDk(zbPink, dark))
		thickLine(b, mx, my, ex, ey, 1, tiaDk(zbPinkD, dark))
		ix := int(math.Floor(ex))
		if vw == vSide { // the big flat foot
			b.rect(ix-1, top, 7, 1, tiaDk(zbPink, dark))
			b.rect(ix-1, top+1, 7, 1, tiaDk(zbPinkD, dark))
			b.set(ix+5, top, tiaDk(zbPinkL, dark))
			return
		}
		x0 := ix - 1 - boolI(sign < 0)
		b.rect(x0, top, 4, 1, tiaDk(zbPink, dark))
		b.rect(x0, top+1, 4, 1, tiaDk(zbPinkD, dark))
		if vw == vFront { // toes
			b.set(x0+1, top+1, tiaDk(zbPinkDD, dark))
			b.set(x0+2, top+1, tiaDk(zbPinkDD, dark))
		}
	}

	// tail: a long thin line hanging from the rump, a little curl at the tip
	tail := func(pts [][2]int) {
		b.cur = 2
		for i := 0; i+1 < len(pts); i++ {
			thickLine(b, float64(pts[i][0]), float64(pts[i][1]), float64(pts[i+1][0]), float64(pts[i+1][1]), 1, zbPinkD)
		}
	}

	body := func() {
		b.cur = 2
		if vw == vSide {
			c := 11 + ox
			for r := 0; r <= hipY-ty0; r++ {
				x0, x1 := c-1, c+1
				if r >= 6 {
					x0 = c - 2 // the rump
				}
				for x := x0; x <= x1; x++ {
					col := zbPink
					if x == x0 {
						col = zbPinkD
					}
					if x == x1 && r >= 2 && r <= 8 {
						col = zbBelly // the light belly in front
					}
					b.set(x, ty0+r, col)
				}
			}
			b.cur = 1
			b.rect(L, ty0-2, 2, 2, zbPink) // the thin neck
			b.set(L, ty0-1, zbPinkD)
			return
		}
		for r := 0; r <= hipY-ty0; r++ {
			x0, x1 := 10, 13
			if r == 0 || r >= 8 {
				x0, x1 = 9, 14
			}
			for x := x0; x <= x1; x++ {
				col := zbPink
				switch {
				case x == x0:
					col = zbPinkL
				case x == x1:
					col = zbPinkD
				}
				if vw == vFront && x >= 11 && x <= 12 && r >= 2 && r <= 8 {
					col = zbBelly
				}
				if vw == vBack && x == 11 && r >= 1 {
					col = zbPinkD // the spine
				}
				b.set(x+ox, ty0+r, col)
			}
		}
		b.cur = 1
		b.rect(L, ty0-2, 2, 2, zbPink)
		b.set(R, ty0-1, zbPinkD)
	}

	cx := 11.5 + float64(hd)
	head := func() {
		b.cur = 1
		switch vw {
		case vSide:
			// a smallish skull and the long snout forward, the red nose at its tip
			zbOval(b, cx-0.5, float64(hy)+4.5, 3.4, 4.3, zbPink, nil)
			b.rect(L+2, hy+5, 5, 3, zbPink)
			b.rect(L+2, hy+7, 4, 1, zbPinkL)
			b.set(R+5, hy+7, zbPinkD)
			b.rect(R+5, hy+4, 2, 2, zbNose)
			b.set(R+6, hy+5, zbTieD)
			b.rect(L-3, hy+3, 1, 4, zbPinkD) // back of the head
			// the ear, pointed, behind the top
			b.set(L-2, hy-1, zbPink)
			b.set(L-2, hy-2, zbPink)
			b.set(L-1, hy-1, zbPinkD)
			b.set(L-1, hy, zbPinkD)
			// the yellow eye under a heavy lid
			b.rect(R, hy+2, 3, 1, zbPinkD)
			b.rect(R, hy+3, 2, 1, zbYellow)
			b.set(R+1, hy+3, zbPupil)
			// whiskers and mouth
			b.set(R+3, hy+5, zbPinkDD)
			b.set(R+2, hy+6, zbPinkDD)
			if open {
				b.rect(R, hy+8, 4, 2, zbMouth)
			} else {
				b.rect(R, hy+8, 4, 1, zbPinkD)
			}
		case vBack:
			zbOval(b, cx, float64(hy)+4.5, 3.6, 4.5, zbPink, nil)
			zbOval(b, cx, float64(hy)+5.5, 3.0, 3.0, zbPinkD, func(x, y int) bool { return y >= hy+7 })
			for _, e := range [][2]int{{L - 3, -1}, {R + 3, 1}} { // the ears
				b.set(e[0], hy, zbPink)
				b.set(e[0], hy-1, zbPink)
				b.set(e[0]-e[1], hy-2, zbPink)
			}
		default:
			zbOval(b, cx, float64(hy)+4.5, 3.6, 4.5, zbPink, nil)
			b.rect(R+3, hy+3, 1, 4, zbPinkD)
			for _, e := range [][2]int{{L - 3, -1}, {R + 3, 1}} { // pointed ears, dark inside
				b.set(e[0], hy, zbPink)
				b.set(e[0], hy-1, zbPink)
				b.set(e[0]+e[1], hy-2, zbPink)
				b.set(e[0]-e[1], hy, zbPinkDD)
			}
			// the yellow eyes, close together, half closed by the lids
			b.rect(L-2, hy+2, 2, 1, zbPinkD)
			b.rect(R, hy+2, 2, 1, zbPinkD)
			b.set(L-2, hy+3, zbYellow)
			b.set(L-1, hy+3, zbPupil)
			b.set(R, hy+3, zbPupil)
			b.set(R+1, hy+3, zbYellow)
			// the snout coming forward, the red nose on it
			b.rect(L-1, hy+5, 4, 3, zbPinkL)
			b.rect(L, hy+4, 2, 2, zbNose)
			b.set(R, hy+5, zbTieD)
			b.set(L-2, hy+6, zbPinkDD) // whiskers
			b.set(R+2, hy+6, zbPinkDD)
			if open {
				b.rect(L-1, hy+8, 4, 1, zbMouth)
			} else {
				b.rect(L, hy+8, 2, 1, zbPinkD)
			}
		}
	}

	arm := func(sx float64, l Limb, sign float64, id uint8, dark bool) (int, int) {
		b.cur = id
		sy := float64(ty0 + 1)
		mx, my, ex, ey := tiaArmIK(vw, sx, sy, l, sign, 1.15, 5.2)
		thickLine(b, sx, sy, mx, my, 1, tiaDk(zbPink, dark))
		thickLine(b, mx, my, ex, ey, 1, tiaDk(zbPink, dark))
		ix, iy := int(math.Floor(ex)), int(math.Floor(ey))
		b.rect(ix, iy, 2, 2, tiaDk(zbPink, dark))
		b.set(ix+1, iy+1, tiaDk(zbPinkD, dark))
		return ix, iy + 1
	}

	switch vw {
	case vSide:
		c := 11 + ox
		tail([][2]int{{c - 2, hipY - 1}, {c - 4, hipY + 1}, {c - 6, hipY + 4}, {c - 6 + swing, hipY + 9}, {c - 5 + swing, hipY + 13}, {c - 3 + swing, hipY + 14}})
		ax := float64(11 + ox)
		arm(ax-0.5, p.LA, 1, 3, true)
		leg(10, p.LF, 1, 5, true)
		leg(11, p.RF, 1, 6, false)
		body()
		head()
		x, y := arm(ax, p.RA, 1, 4, false)
		tiaHeld(b, s, p, vw, x, y, 4)
	case vBack:
		leg(13, p.LF, 1, 5, false)
		leg(10, p.RF, -1, 6, false)
		body()
		head()
		c := 11 + ox
		tail([][2]int{{c, hipY - 1}, {c, hipY + 3}, {c + 1 + swing, hipY + 8}, {c + 2 + swing, hipY + 12}, {c + 4 + swing, hipY + 13}})
		arm(float64(15+ox), p.LA, 1, 3, false)
		x, y := arm(float64(8+ox), p.RA, -1, 4, false)
		tiaHeld(b, s, p, vw, x, y, 4)
	default:
		c := 13 + ox // the tail peeps out beside his leg
		tail([][2]int{{c, hipY}, {c + 3, hipY + 4}, {c + 4 + swing, hipY + 9}, {c + 4 + swing, hipY + 12}, {c + 6 + swing, hipY + 13}})
		leg(10, p.LF, -1, 5, false)
		leg(13, p.RF, 1, 6, false)
		body()
		head()
		arm(float64(8+ox), p.LA, -1, 3, false)
		x, y := arm(float64(15+ox), p.RA, 1, 4, false)
		tiaHeld(b, s, p, vw, x, y, 4)
	}
	return b
}

// ---- Mr Bean ------------------------------------------------------------------------------------

func genBean(s *CharSpec, vw int, p Pose) *pixBuf {
	b, bob, ox, hd := zbFrame(vw, p, -1, 3)
	hipY := 26 + bob
	ty0 := hipY - 11 // shoulders
	hy := ty0 - 11   // skull top
	open := zbOpen(p)
	L, R := 11+hd, 12+hd

	leg := func(hx0 float64, l Limb, sign float64, id uint8, dark bool) {
		b.cur = id
		mx, my, ex, ey, top := tiaLegIK(vw, p, hx0, float64(hipY+1), l, sign, 0.9, 6.6, 40)
		thickLine(b, hx0, float64(hipY+1), mx, my, 2, tiaDk(zbTrou, dark))
		thickLine(b, mx, my, ex, ey, 2, tiaDk(zbTrou, dark))
		thickLine(b, hx0, float64(hipY+2), mx, my, 1, tiaDk(zbTrouL, dark))
		ix := int(math.Floor(ex))
		if vw == vSide {
			b.rect(ix, top, 3, 1, tiaDk(zbShoeK, dark))
			b.rect(ix-1, top+1, 6, 2, tiaDk(zbShoeK, dark))
			b.rect(ix+1, top+1, 3, 1, tiaDk(zbShoeKL, dark))
			return
		}
		x0 := ix - 1 + boolI(sign > 0)
		b.rect(x0, top, 3, 1, tiaDk(zbShoeK, dark))
		b.rect(x0-boolI(sign < 0), top+1, 4, 2, tiaDk(zbShoeK, dark))
		if vw == vFront {
			b.rect(x0, top+1, 2, 1, tiaDk(zbShoeKL, dark))
		}
	}

	jacket := func() {
		b.cur = 2
		if vw == vSide {
			c := 11 + ox
			for r := 0; r <= hipY-ty0+1; r++ {
				x0, x1 := c-2, c+2
				if r == 0 {
					x0, x1 = c-1, c+2
				}
				for x := x0; x <= x1; x++ {
					col := zbTweed
					switch {
					case x == x0:
						col = zbTweedD
					case r == 0 || x == x1-1 && r < 5:
						col = zbTweedL
					}
					if (x+r)%3 == 0 && col == zbTweed {
						col = zbTweedL // the tweed weave
					}
					b.set(x, ty0+r, col)
				}
			}
			b.set(c+3, ty0, zbWhite) // collar and tie poking out in front
			b.set(c+2, ty0+1, zbWhite)
			b.set(c+3, ty0+1, zbTie)
			b.set(c+3, ty0+2, zbTieD)
			b.set(c+2, ty0+6, zbBtn)
			return
		}
		for r := 0; r <= hipY-ty0+1; r++ {
			x0, x1 := 8, 15
			if r >= 7 {
				x0, x1 = 9, 14
			}
			for x := x0; x <= x1; x++ {
				col := zbTweed
				switch {
				case r == 0 || x == x0:
					col = zbTweedL
				case x == x1:
					col = zbTweedD
				case (x+r)%3 == 0:
					col = zbTweedL
				}
				b.set(x+ox, ty0+r, col)
			}
		}
		if vw == vFront {
			// white shirt in the V, the red tie with its pin, the lapels
			b.rect(10+ox, ty0, 4, 1, zbWhite)
			for r := 1; r <= 4; r++ {
				hw := 2 - r/3
				b.rect(12-hw+ox, ty0+r, hw*2, 1, zbWhite)
			}
			b.rect(11+ox, ty0+1, 2, 1, zbTie) // the knot
			b.rect(11+ox, ty0+2, 2, 4, zbTie)
			b.set(12+ox, ty0+2, zbTieD)
			b.set(12+ox, ty0+5, zbTieD)
			b.set(11+ox, ty0+4, zbPin)
			b.set(12+ox, ty0+4, zbPin)
			b.set(9+ox, ty0+1, zbTweedD) // lapels
			b.set(14+ox, ty0+1, zbTweedD)
			b.set(10+ox, ty0+4, zbTweedD)
			b.set(13+ox, ty0+4, zbTweedD)
			b.set(12+ox, ty0+7, zbBtn)
			b.set(12+ox, ty0+9, zbBtn)
			b.rect(9+ox, ty0+8, 2, 1, zbTweedD) // pocket
		} else {
			b.rect(10+ox, ty0, 4, 1, zbWhite)
			b.rect(11+ox, ty0+5, 1, hipY-ty0-3, zbTweedD) // back seam
		}
		b.cur = 1
		b.rect(L, ty0-1, 2, 1, zbSkin)
		b.set(R, ty0-1, zbSkinD)
	}

	cx := 11.5 + float64(hd)
	cy := float64(hy) + 5.2
	head := func() {
		b.cur = 1
		switch vw {
		case vSide:
			zbOval(b, cx-0.5, cy, 4.0, 5.3, zbSkin, nil)
			// hair: short and flat on top, down to the ear at the back
			zbOval(b, cx-0.5, cy-0.3, 4.3, 5.6, zbBeanH, func(x, y int) bool {
				return y <= hy+2 || x <= L-1 && y <= hy+5
			})
			b.set(L+1, hy, zbBeanHL)
			b.set(L+2, hy, zbBeanHL)
			// the big ear
			b.rect(L-1, hy+4, 2, 4, zbSkin)
			b.set(L-1, hy+4, zbSkinD)
			b.rect(L, hy+5, 1, 2, zbSkinD)
			// thick brow, big eye
			b.rect(R+1, hy+3, 3, 1, zbBeanH)
			b.set(R+2, hy+4, zbEyeW)
			b.set(R+3, hy+4, zbPupil)
			b.set(R+2, hy+5, zbEyeW)
			b.set(R+3, hy+5, zbPupil)
			// the big round nose
			b.rect(R+4, hy+5, 2, 3, zbSkin)
			b.set(R+5, hy+7, zbSkinD)
			b.set(R+4, hy+7, zbSkinD)
			b.set(R+4, hy+5, zbSkinL)
			if open {
				b.rect(R+2, hy+8, 3, 2, zbMouth)
			} else {
				b.rect(R+2, hy+9, 3, 1, zbMouth)
			}
			b.rect(R+1, hy+10, 3, 1, zbSkinD) // the long chin
		case vBack:
			zbOval(b, cx, cy, 4.2, 5.3, zbSkin, nil)
			b.rect(L-5, hy+4, 1, 3, zbSkin) // big ears
			b.rect(R+5, hy+4, 1, 3, zbSkin)
			b.set(L-5, hy+5, zbSkinD)
			b.set(R+5, hy+5, zbSkinD)
			zbOval(b, cx, cy-0.3, 4.4, 5.6, zbBeanH, func(x, y int) bool { return y <= hy+7 })
			b.rect(L-1, hy, 3, 1, zbBeanHL)
		default:
			zbOval(b, cx, cy, 4.2, 5.3, zbSkin, nil)
			b.rect(R+4, hy+4, 1, 5, zbSkinD)
			// big ears standing out
			b.rect(L-5, hy+4, 1, 3, zbSkin)
			b.rect(R+5, hy+4, 1, 3, zbSkin)
			b.set(L-5, hy+5, zbSkinD)
			b.set(R+5, hy+5, zbSkinD)
			b.set(L-6, hy+5, zbSkin)
			b.set(R+6, hy+5, zbSkinD)
			// short black hair with the side parting
			zbOval(b, cx, cy-0.3, 4.4, 5.6, zbBeanH, func(x, y int) bool {
				return y <= hy+1 || (x <= L-3 || x >= R+3) && y <= hy+3
			})
			b.set(L-2, hy, zbSkinL) // the parting
			b.set(L-2, hy-1, zbBeanHL)
			b.rect(L, hy, 3, 1, zbBeanHL)
			b.rect(L-1, hy+2, 4, 1, zbBeanH) // a fringe swept to the side
			// thick brows, big eyes
			b.rect(L-3, hy+3, 3, 1, zbBeanH)
			b.rect(R+1, hy+3, 3, 1, zbBeanH)
			b.rect(L-3, hy+4, 2, 2, zbEyeW)
			b.rect(R+2, hy+4, 2, 2, zbEyeW)
			b.set(L-2, hy+5, zbPupil)
			b.set(R+2, hy+5, zbPupil)
			b.set(L-2, hy+4, zbPupil)
			b.set(R+2, hy+4, zbPupil)
			// the big round nose
			b.rect(L-1, hy+5, 4, 2, zbSkin)
			b.rect(L, hy+4, 2, 1, zbSkin)
			b.set(L-1, hy+5, zbSkinD)
			b.set(R+1, hy+6, zbSkinD)
			b.rect(L, hy+7, 2, 1, zbSkinD)
			b.set(L, hy+5, zbSkinL)
			if open {
				b.rect(L-1, hy+8, 4, 2, zbMouth)
				b.rect(L, hy+9, 2, 1, zbTongue)
			} else { // pressed lips
				b.rect(L-1, hy+9, 4, 1, zbMouth)
				b.set(R+2, hy+8, zbMouth)
			}
			b.rect(L-1, hy+10, 4, 1, zbSkinD) // the long chin
		}
	}

	arm := func(sx float64, l Limb, sign float64, id uint8, dark bool) (int, int) {
		b.cur = id
		sy := float64(ty0 + 1)
		mx, my, ex, ey := tiaArmIK(vw, sx, sy, l, sign, 1.1, 5.0)
		if vw != vSide {
			ex = math.Min(ex, 18) // the briefcase (7 px) stays inside the box with the arms up
		}
		thickLine(b, sx, sy, mx, my, 2, tiaDk(zbTweed, dark))
		thickLine(b, mx, my, ex, ey, 2, tiaDk(zbTweed, dark))
		thickLine(b, sx, sy, mx, my, 1, tiaDk(zbTweedL, dark))
		// the brown patch on the elbow
		b.rect(int(math.Floor(mx))-boolI(sign < 0 && vw != vSide), int(math.Floor(my))-1, 2, 2, tiaDk(zbPatch, dark))
		ix, iy := int(math.Floor(ex)), int(math.Floor(ey))
		b.rect(ix, iy, 2, 1, tiaDk(zbWhite, dark))
		b.rect(ix, iy+1, 2, 2, tiaDk(zbSkin, dark))
		b.set(ix+1, iy+2, tiaDk(zbSkinD, dark))
		return ix, iy + 1
	}

	switch vw {
	case vSide:
		ax := float64(11 + ox)
		arm(ax-0.5, p.LA, 1, 3, true)
		leg(10, p.LF, 1, 5, true)
		leg(11, p.RF, 1, 6, false)
		jacket()
		head()
		x, y := arm(ax, p.RA, 1, 4, false)
		tiaHeld(b, s, p, vw, x, y, 4)
	case vBack:
		leg(13, p.LF, 1, 5, false)
		leg(10, p.RF, -1, 6, false)
		jacket()
		head()
		arm(float64(16+ox), p.LA, 1, 3, false)
		x, y := arm(float64(7+ox), p.RA, -1, 4, false)
		tiaHeld(b, s, p, vw, x, y, 4)
	default:
		leg(10, p.LF, -1, 5, false)
		leg(13, p.RF, 1, 6, false)
		jacket()
		head()
		arm(float64(7+ox), p.LA, -1, 3, false)
		x, y := arm(float64(16+ox), p.RA, 1, 4, false)
		tiaHeld(b, s, p, vw, x, y, 4)
	}
	return b
}
