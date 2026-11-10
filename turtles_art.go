package main

import (
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// The four Ninja Turtles (slugs tortuga_leonardo, tortuga_donatello, tortuga_michelangelo,
// tortuga_raphael) have their own procedural drawing (genTurtle), like Crash or SpongeBob: the
// generic body cannot show a chibi turtle. Our own pixel-art parody: short and big-headed, green
// skin (Donatello olive, Leonardo mid green, Michelangelo teal, Raphael dark green), brown shell
// on the back with a light rim, cream-yellow plastron plates on the chest, a brown belt, a mask
// whose long tails flutter to one side, wrist and knee pads in the mask colour, three-fingered
// hands and big green feet.
//   Leonardo (blue): two katanas crossed on the back.
//   Donatello (purple): a long bo staff in the hand.
//   Michelangelo (orange): nunchakus in the hand and a grin with the tongue out.
//   Raphael (red): two sais tucked in the belt and a grumpy face.
// Animation (visual only, hitbox and collisions do not change): arms and legs follow the pose,
// the body bobs and sways and the mask tails flap while moving.
// Standing rows: head 13..24, torso 24..32 (belt 31), legs 33..39, feet 40..42 (a guest's head
// starts around row 10, so the turtles are a little shorter with a much bigger head).
// noCharImpQ (the stock drawing used by the tests) turns this drawing off.
// ---------------------------------------------------------------------------

const (
	turtleLeo = iota
	turtleDon
	turtleMike
	turtleRaph
)

type turtleLook struct {
	skin, mask color.RGBA
	kind       int
}

var turtleLooks = map[string]turtleLook{
	"tortuga_leonardo":     {hx(0x4c9c3a), hx(0x2c5ee0), turtleLeo},
	"tortuga_donatello":    {hx(0x8aa23a), hx(0x8a3cc8), turtleDon},
	"tortuga_michelangelo": {hx(0x38a080), hx(0xf28a1a), turtleMike},
	"tortuga_raphael":      {hx(0x2e7230), hx(0xd82626), turtleRaph},
}

func turtleOf(s *CharSpec) (turtleLook, bool) {
	if s == nil || noCharImpQ {
		return turtleLook{}, false
	}
	l, ok := turtleLooks[s.Name]
	return l, ok
}

func isTurtle(s *CharSpec) bool {
	_, ok := turtleOf(s)
	return ok
}

var (
	ttShell   = hx(0x8a5626)
	ttShellL  = hx(0xc08a44)
	ttShellD  = hx(0x5a3412)
	ttPlast   = hx(0xf2d878)
	ttPlastL  = hx(0xfcecaa)
	ttPlastD  = hx(0xc8a446)
	ttBelt    = hx(0x6a4220)
	ttBuckle  = hx(0xd8b040)
	ttSteel   = hx(0xd4dae4)
	ttSteelD  = hx(0x7a8292)
	ttWood    = hx(0xa06a34)
	ttWoodD   = hx(0x6a4420)
	ttHilt    = hx(0x2a2028)
	ttEye     = hx(0xffffff)
	ttPupil   = hx(0x101014)
	ttMouth   = hx(0x3c1410)
	ttTongue  = hx(0xe85a6a)
	ttSaiWrap = hx(0x7a1a1a)
)

func genTurtle(s *CharSpec, vw int, p Pose) *pixBuf {
	lk, _ := turtleOf(s)
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
	skin, skinL, skinD := lk.skin, shade(lk.skin, 1.2), shade(lk.skin, 0.72)
	mask, maskL, maskD := lk.mask, shade(lk.mask, 1.25), shade(lk.mask, 0.68)
	flap := 0
	if p.Anim != AnimIdle {
		flap = p.F % 2
	}

	bob := clampI(rnd(p.Bob), -2, 3)
	if p.Sit {
		bob = maxI(bob, 3)
	}
	ox := clampI(rnd(p.Sway*0.5), -2, 2)
	hox := ox
	if vw == vSide {
		ox = clampI(rnd(p.Lean*0.5), -1, 2)
		hox = ox + clampI(rnd(p.Lean*0.3+p.HeadZ*0.5), -1, 1)
	} else if p.HeadZ != 0 {
		hox += clampI(rnd(p.HeadZ*0.5), -1, 1)
	}
	hipY := 32 + bob
	ty0 := hipY - 8 // torso top
	hy := ty0 - 11  // skull top
	lift := func(l Limb) int {
		if p.Sit {
			return 0
		}
		return clampI(rnd(10-l.Y), 0, 6)
	}

	// ---- legs with knee pads and big feet ----
	leg := func(hx0 float64, l Limb, sign float64, id uint8, dark bool) {
		b.cur = id
		top := 40 - lift(l)
		var fx float64
		if vw == vSide {
			fx = hx0 + l.Z*0.6
			if p.Sit {
				fx = hx0 + 4
				top = hipY + 3
			}
		} else {
			fx = hx0 + sign*clampF((l.X-1.5)*0.8, -2, 2)
		}
		bx, by := sign, 0.0
		if vw == vSide {
			bx, by = 1, 0
		}
		mx, my, ex, ey := ik2(hx0, float64(hipY+1), fx, float64(top)-0.5, 3.8, bx, by)
		thickLine(b, hx0, float64(hipY+1), mx, my, 3, dk(skin, dark))
		thickLine(b, mx, my, ex, ey, 3, dk(skin, dark))
		edge := 0.0
		if vw != vSide && sign > 0 {
			edge = 2
		}
		thickLine(b, mx+edge, my, ex+edge, ey, 1, dk(skinD, dark))
		kx, ky := int(math.Floor(mx)), int(math.Floor(my))
		b.rect(kx, ky, 3, 2, dk(mask, dark)) // knee pad
		b.set(kx+1, ky, dk(maskL, dark))
		b.set(kx+2, ky+1, dk(maskD, dark))
		ix := int(math.Floor(fx))
		if vw == vSide {
			b.rect(ix, top, 3, 1, dk(skin, dark))
			b.rect(ix-1, top+1, 6, 1, dk(skin, dark))
			b.rect(ix-1, top+2, 6, 1, dk(skinD, dark))
			b.set(ix+4, top+1, dk(skinL, dark)) // toes
			b.set(ix+3, top+2, dk(shade(skin, 0.55), dark))
		} else {
			x0 := ix - 1
			b.rect(x0+1, top, 3, 1, dk(skin, dark))
			b.rect(x0, top+1, 5, 1, dk(skin, dark))
			b.rect(x0, top+2, 5, 1, dk(skinD, dark))
			if vw == vFront { // three toes
				b.set(x0+1, top+2, dk(shade(skin, 0.5), dark))
				b.set(x0+3, top+2, dk(shade(skin, 0.5), dark))
				b.set(x0+2, top+1, dk(skinL, dark))
			}
		}
	}

	// ---- Leonardo's katanas (crossed on the back) ----
	katana := func(x0, y0, x1, y1 int) {
		b.cur = 2
		b.line(x0, y0, x1, y1, ttSteel)
		b.line(x0, y0+1, x1, y1+1, ttSteelD)
		sx := 1
		if x1 < x0 {
			sx = -1
		}
		b.set(x0, y0, ttHilt) // hilt and guard
		b.set(x0+sx, y0+1, ttHilt)
		b.set(x0+sx, y0, mask)
		b.set(x0+2*sx, y0+2, ttBuckle)
		b.set(x0+2*sx+sx, y0+2, ttBuckle)
	}
	katanas := func() {
		if lk.kind != turtleLeo {
			return
		}
		switch vw {
		case vSide:
			c := 11 + ox
			katana(c-6, ty0-4, c+1, ty0+8)
		default:
			katana(5+ox, ty0-4, 16+ox, ty0+8)
			katana(18+ox, ty0-4, 7+ox, ty0+8)
		}
	}

	// ---- torso: plastron in front, shell behind ----
	shellBack := func(cx, cy, rx, ry float64) {
		b.ellipse(cx, cy, rx+0.6, ry+0.6, ttShellL) // light rim
		b.ellipse(cx, cy, rx-0.3, ry-0.3, ttShell)
	}
	torso := func() {
		b.cur = 2
		if vw == vSide {
			c := 11 + ox
			shellBack(float64(c)-2.5, float64(ty0)+4, 3.0, 5.0)
			b.line(c-4, ty0+2, c-3, ty0+6, ttShellD)
			b.line(c-5, ty0+4, c-2, ty0+4, ttShellD)
			for r := 0; r <= 8; r++ {
				x0, x1 := c-2, c+2
				if r == 0 || r == 8 {
					x1 = c + 1
				}
				for x := x0; x <= x1; x++ {
					col := skin
					if x >= c+1 && r >= 1 && r <= 6 {
						col = ttPlast
						if r%2 == 0 {
							col = ttPlastD
						}
					}
					if x == x0 {
						col = skinD
					}
					b.set(x, ty0+r, col)
				}
			}
			b.rect(c-3, hipY-1, 6, 1, ttBelt)
			b.set(c+1, hipY-1, ttBuckle)
			return
		}
		if vw == vFront { // the shell peeks out at both sides
			shellBack(11.5+float64(ox), float64(ty0)+4, 5.6, 4.6)
		}
		widths := []int{3, 4, 5, 5, 5, 5, 5, 4, 4} // half widths around 11.5
		for r, hw := range widths {
			for x := 12 - hw; x <= 11+hw; x++ {
				col := skin
				if r == 0 {
					col = skinL
				}
				if x == 12-hw || x == 11+hw {
					col = skinD
				}
				b.set(x+ox, ty0+r, col)
			}
		}
		if vw == vBack {
			cx := 11.5 + float64(ox)
			shellBack(cx, float64(ty0)+3.6, 5.3, 4.8)
			o := ox
			// hexagonal plates
			for _, sg := range [][4]int{
				{10, 1, 13, 1}, {13, 1, 15, 4}, {15, 4, 13, 7}, {13, 7, 10, 7}, {10, 7, 8, 4}, {8, 4, 10, 1},
				{10, 1, 9, -1}, {13, 1, 14, -1}, {15, 4, 17, 4}, {8, 4, 6, 4}, {13, 7, 14, 8}, {10, 7, 9, 8},
			} {
				b.line(sg[0]+o, ty0+sg[1], sg[2]+o, ty0+sg[3], ttShellD)
			}
			b.set(11+o, ty0+3, ttShellL) // shine on the middle plate
			b.set(12+o, ty0+3, ttShellL)
			b.rect(7+o, hipY-1, 2, 1, ttBelt)
			b.rect(15+o, hipY-1, 2, 1, ttBelt)
			return
		}
		// plastron: cream-yellow plates with darker seams and a rim
		for r := 0; r <= 6; r++ {
			hw := 3
			if r == 0 || r == 6 {
				hw = 2
			}
			for x := 12 - hw; x <= 11+hw; x++ {
				col := ttPlast
				if x == 12-hw || x == 11+hw || r == 0 {
					col = ttPlastL // the rim
				}
				if (r == 2 || r == 4) && x != 12-hw && x != 11+hw {
					col = ttPlastD
				}
				b.set(x+ox, ty0+r, col)
			}
		}
		b.set(11+ox, ty0+1, ttPlastD) // centre seam
		b.set(12+ox, ty0+3, ttPlastD)
		b.set(11+ox, ty0+5, ttPlastD)
		b.rect(8+ox, hipY-1, 8, 1, ttBelt)
		b.rect(11+ox, hipY-1, 2, 1, ttBuckle)
	}

	// ---- Raphael's sais tucked in the belt ----
	sai := func(x, y, dir int) {
		b.cur = 2
		b.rect(x, y-3, 1, 3, ttSaiWrap) // handle up
		b.set(x, y-4, ttSteelD)         // pommel
		b.set(x-1, y, ttSteel)          // guard prongs
		b.set(x+1, y, ttSteel)
		b.set(x-1, y-1, ttSteelD)
		b.set(x+1, y-1, ttSteelD)
		b.line(x, y, x+dir, y+5, ttSteel) // blade down
	}
	sais := func() {
		if lk.kind != turtleRaph {
			return
		}
		switch vw {
		case vSide:
			sai(12+ox, hipY, -1)
		default:
			sai(8+ox, hipY, -1)
			sai(15+ox, hipY, 1)
		}
	}

	// ---- head: big round skull, mask with eye holes, tails fluttering ----
	tail := func(x0, y0, x1, y1 int) {
		b.line(x0, y0, clampI(x1, 2, 21), y1, mask)
		b.line(x0, y0+1, clampI(x1, 2, 21), y1+1, maskD)
	}
	head := func() {
		b.cur = 1
		hd := hox
		switch vw {
		case vSide:
			cx := 11.5 + float64(hd)
			b.ellipse(cx, float64(hy)+5.6, 5.0, 5.6, skin)
			b.ellipse(cx+2.6, float64(hy)+7.6, 3.2, 3.0, skin) // snout
			b.ellipse(cx-1.5, float64(hy)+2.5, 2.0, 1.4, skinL)
			b.rect(int(cx)-4, hy+9, 2, 2, skinD) // nape
			for y := hy + 2; y <= hy+5; y++ {
				for x := int(cx) - 6; x <= int(cx)+6; x++ {
					i := (y*b.w + x) * 4
					if x >= 0 && x < b.w && b.p[i+3] > 0 {
						col := mask
						if y == hy+5 {
							col = maskD
						} else if y == hy+2 {
							col = maskL
						}
						b.set(x, y, col)
					}
				}
			}
			ex := int(cx) + 2
			b.rect(ex, hy+3, 2, 2, ttEye)
			b.set(ex+1, hy+4, ttPupil)
			if lk.kind == turtleRaph {
				b.set(ex-1, hy+3, maskD) // frown
			}
			b.rect(int(cx)-6, hy+3, 2, 2, maskD) // the knot at the back
			b.line(int(cx)+2, hy+8, int(cx)+5, hy+8, ttMouth)
			if lk.kind == turtleMike {
				b.set(int(cx)+5, hy+7, ttMouth)
				b.set(int(cx)+4, hy+9, ttTongue)
				b.set(int(cx)+5, hy+9, ttTongue)
			} else if lk.kind != turtleRaph {
				b.set(int(cx)+5, hy+7, ttMouth)
			}
			kx := int(cx) - 6
			tail(kx, hy+3, kx-4, hy+4+flap)
			tail(kx, hy+4, kx-3, hy+7-flap)
		case vBack:
			cx := 11.5 + float64(hd)
			b.ellipse(cx, float64(hy)+5.6, 5.8, 5.6, skin)
			b.ellipse(cx-2, float64(hy)+2.4, 2.2, 1.4, skinL)
			b.rect(int(cx)-2, hy+10, 4, 2, skinD)
			for y := hy + 2; y <= hy+5; y++ {
				for x := int(cx) - 7; x <= int(cx)+7; x++ {
					i := (y*b.w + x) * 4
					if x >= 0 && x < b.w && b.p[i+3] > 0 {
						col := mask
						if y == hy+5 {
							col = maskD
						} else if y == hy+2 {
							col = maskL
						}
						b.set(x, y, col)
					}
				}
			}
			kx := int(cx)
			b.rect(kx-1, hy+3, 2, 2, maskD) // knot
			tail(kx, hy+5, kx+3+flap, hy+11)
			tail(kx-1, hy+5, kx+5, hy+9-flap)
		default:
			cx := 11.5 + float64(hd)
			b.ellipse(cx, float64(hy)+5.6, 5.8, 5.6, skin)
			b.ellipse(cx-2.2, float64(hy)+2.2, 2.2, 1.3, skinL)
			b.ellipse(cx, float64(hy)+9.0, 3.6, 2.2, shade(skin, 1.08)) // lighter muzzle
			ihx := int(math.Floor(cx))
			for y := hy + 2; y <= hy+5; y++ {
				for x := ihx - 7; x <= ihx+7; x++ {
					i := (y*b.w + x) * 4
					if x >= 0 && x < b.w && b.p[i+3] > 0 {
						col := mask
						if y == hy+5 {
							col = maskD
						} else if y == hy+2 {
							col = maskL
						}
						b.set(x, y, col)
					}
				}
			}
			// eye holes
			b.rect(ihx-3, hy+3, 2, 2, ttEye)
			b.rect(ihx+2, hy+3, 2, 2, ttEye)
			b.set(ihx-2, hy+4, ttPupil)
			b.set(ihx+2, hy+4, ttPupil)
			switch lk.kind {
			case turtleRaph: // angry brows and gritted teeth
				b.set(ihx-2, hy+3, maskD)
				b.set(ihx+2, hy+3, maskD)
				b.rect(ihx-2, hy+8, 4, 1, ttEye)
				b.rect(ihx-2, hy+9, 4, 1, ttMouth)
				b.set(ihx-3, hy+8, ttMouth)
				b.set(ihx+2, hy+8, ttMouth)
			case turtleMike: // big grin with the tongue out
				b.rect(ihx-3, hy+8, 6, 1, ttMouth)
				b.set(ihx-4, hy+7, ttMouth)
				b.set(ihx+3, hy+7, ttMouth)
				b.rect(ihx-2, hy+9, 4, 1, ttMouth)
				b.rect(ihx, hy+9, 2, 1, ttTongue)
				b.rect(ihx, hy+10, 2, 1, ttTongue)
				b.set(ihx+1, hy+10, shade(ttTongue, 0.8))
			case turtleDon: // smile with a gap tooth
				b.rect(ihx-2, hy+8, 4, 1, ttMouth)
				b.set(ihx-3, hy+7, ttMouth)
				b.set(ihx+2, hy+7, ttMouth)
				b.set(ihx-1, hy+9, ttEye)
			default: // Leonardo: calm smile
				b.rect(ihx-2, hy+8, 4, 1, ttMouth)
				b.set(ihx-3, hy+7, ttMouth)
				b.set(ihx+2, hy+7, ttMouth)
			}
			// tails fluttering out of the left side
			kx := ihx - 6
			tail(kx, hy+3, kx-3, hy+5+flap)
			tail(kx, hy+4, kx-2, hy+8-flap)
		}
	}

	// ---- arms: wrist pads and three-fingered hands ----
	type handAt struct{ x, y int }
	hand := func(gx, iy int, dark bool) {
		b.rect(gx, iy, 3, 2, dk(skin, dark))
		b.set(gx, iy+2, dk(skin, dark)) // three fingers
		b.set(gx+1, iy+2, dk(skinD, dark))
		b.set(gx+2, iy+2, dk(skin, dark))
		b.set(gx+1, iy, dk(skinL, dark))
	}
	arm := func(sx float64, l Limb, sign float64, id uint8, dark bool) handAt {
		b.cur = id
		sy := float64(ty0 + 1)
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
		mx, my, ex, ey := ik2(sx, sy, clampF(sx+tx, 3, 19), sy+ty, 3.4, bx, by) // the hand stays inside the box
		thickLine(b, sx, sy, mx, my, 2, dk(skin, dark))
		thickLine(b, mx, my, ex, ey, 2, dk(skin, dark))
		b.set(int(math.Floor(sx)), int(sy), dk(skinL, dark))
		ix, iy := int(math.Floor(ex)), int(math.Floor(ey))
		gx := ix
		if vw != vSide && sign < 0 {
			gx = ix - 1
		}
		b.rect(gx, iy-1, 3, 1, dk(mask, dark)) // wrist pad
		hand(gx, iy, dark)
		return handAt{gx, iy}
	}
	signature := func(h handAt, id uint8) {
		switch lk.kind {
		case turtleDon: // bo staff through the hand (on the torso layer: it does not swing like an arm)
			b.cur = 2
			x := h.x + 1
			y0, y1 := maxI(2, h.y-13), minI(42, h.y+8)
			b.rect(x, y0, 1, y1-y0+1, ttWood)
			b.set(x, y0, ttWoodD)
			b.set(x, y1, ttWoodD)
			b.set(x, h.y-4, ttWoodD) // grip wraps
			b.set(x, h.y+4, ttWoodD)
			b.cur = id
			hand(h.x, h.y, false)
		case turtleMike: // a nunchaku hanging from the fist
			b.cur = id
			x := h.x + 1
			nl, nd := hx(0xe8a850), hx(0x9a5a22)
			b.rect(x, h.y+3, 2, 3, nl)
			b.set(x+1, h.y+5, nd)
			b.set(x+1, h.y+6, ttSteel) // chain
			sx := x + 1 - 2*flap
			b.rect(sx, h.y+7, 2, 3, nl)
			b.set(sx+1, h.y+9, nd)
		}
	}
	held := func(h handAt, id uint8) {
		b.cur = id
		kind := p.Hold
		if kind == HoldNone {
			kind = s.Carry
		}
		if s.Weapon != 0 && (kind == HoldKnife || kind == HoldNone) {
			kind = s.Weapon
		}
		if lk.kind == turtleDon && kind == HoldBroom {
			kind = HoldNone // the bo is his staff
		}
		if kind != HoldNone {
			drawHeld(b, kind, s.CarryCol, vw, float64(h.x), float64(h.y+1))
			return
		}
		signature(h, id)
	}

	switch vw {
	case vSide:
		cx := float64(11 + ox)
		katanas()
		arm(cx-0.5, p.LA, 1, 3, true)
		leg(9.5, p.LF, 1, 5, true)
		leg(10.5, p.RF, 1, 6, false)
		torso()
		sais()
		head()
		held(arm(cx, p.RA, 1, 4, false), 4)
	case vBack:
		leg(13, p.LF, 1, 5, false)
		leg(8, p.RF, -1, 6, false)
		torso()
		sais()
		katanas()
		head()
		arm(float64(16+ox)+0.5, p.LA, 1, 3, false)
		held(arm(float64(6+ox)-0.5, p.RA, -1, 4, false), 4)
	default:
		katanas()
		leg(8, p.LF, -1, 5, false)
		leg(13, p.RF, 1, 6, false)
		torso()
		sais()
		head()
		arm(float64(6+ox)-0.5, p.LA, -1, 3, false)
		held(arm(float64(16+ox)+0.5, p.RA, 1, 4, false), 4)
	}
	return b
}
