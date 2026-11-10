package main

import (
	"math"
	"math/rand"
)

// ---------------------------------------------------------------------------
// Doraemon, Nobita and the three wizard pupils (slugs "doraemon", "nobita", "harry_potter",
// "hermione" and the new "ron") have their own procedural drawing (genDnh), like Mortadelo or
// Crash: the generic body cannot give them their chibi builds. Our own small pixel-art parodies:
//   - Doraemon: blue robot cat, big round head with a dark blue rim and a big white face, red
//     nose, whiskers, a smile (wide open when laughing), red collar with a golden bell, white
//     belly with the pocket, round white hands and feet, a red tail with a ball.
//   - Nobita: black hair with a tuft in a peak, big round glasses, yellow shirt with a collar,
//     blue shorts, white socks.
//   - Harry: messy black hair, round glasses, the scar, an open black robe over a grey jumper,
//     white shirt and a red and gold striped tie, dark trousers.
//   - Hermione: long bushy brown hair with a fringe, the same robe and tie, a pleated grey
//     skirt and long white socks.
//   - Ron: messy orange hair, freckles, the robe with the red and gold tie.
// All are short (child height); only the drawing changes: hitbox, footprint, stats and
// collisions are the ones of every guest. The animation is the usual one, read from the pose.
// ---------------------------------------------------------------------------

const (
	dnhNone = iota
	dnhDoraemon
	dnhNobita
	dnhHarry
	dnhHermione
	dnhRon
)

func dnhKind(s *CharSpec) int {
	if s == nil {
		return dnhNone
	}
	switch s.Name {
	case "doraemon":
		return dnhDoraemon
	case "nobita":
		return dnhNobita
	case "harry_potter":
		return dnhHarry
	case "hermione":
		return dnhHermione
	case "ron":
		return dnhRon
	}
	return dnhNone
}

func isDnh(s *CharSpec) bool { return dnhKind(s) != dnhNone }

var (
	dnBlue    = hx(0x1e8ee0)
	dnBlueL   = hx(0x52b0f0)
	dnBlueD   = hx(0x0e4a8a)
	dnWhite   = hx(0xf8f8f4)
	dnWhiteD  = hx(0xc8d0d8)
	dnRed     = hx(0xe02828)
	dnRedD    = hx(0x961818)
	dnRedL    = hx(0xff7a6a)
	dnGold    = hx(0xf4c820)
	dnGoldD   = hx(0xa87810)
	dnInk     = hx(0x141418)
	dnTongue  = hx(0xf08070)
	dnSkin    = hx(0xf8d0a8)
	dnSkinD   = hx(0xd8a07a)
	dnSkinL   = hx(0xffe4c8)
	dnHairK   = hx(0x16141c)
	dnHairKL  = hx(0x34323e)
	dnYellow  = hx(0xf4d020)
	dnYellowL = hx(0xfff07a)
	dnYellowD = hx(0xc09a10)
	dnShorts  = hx(0x2e4ea8)
	dnShortsD = hx(0x1c3070)
	dnShoe    = hx(0x5a3a2a)
	dnShoeL   = hx(0x86604a)
	dnShoeK   = hx(0x18181e)
	dnShoeKL  = hx(0x40404c)
	dnFrame   = hx(0x1a1a20)
	dnLens    = hx(0xe8f2fa)
	dnRobe    = hx(0x1c1c24)
	dnRobeL   = hx(0x3a3a48)
	dnRobeD   = hx(0x0c0c10)
	dnGrey    = hx(0x8a8a92)
	dnGreyL   = hx(0xaeaeb6)
	dnGreyD   = hx(0x5e5e66)
	dnTieR    = hx(0xb81e22)
	dnTieG    = hx(0xe8b828)
	dnTrou    = hx(0x2a2a34)
	dnTrouL   = hx(0x3e3e4c)
	dnBrown   = hx(0x7a4a24)
	dnBrownL  = hx(0xa06a38)
	dnBrownD  = hx(0x50301a)
	dnOrange  = hx(0xe8661c)
	dnOrangeL = hx(0xff9a48)
	dnOrangeD = hx(0xa83e10)
	dnFreckle = hx(0xc87850)
	dnScar    = hx(0xc0303a)
	dnMouth   = hx(0x8a2a24)
	dnLip     = hx(0xc86858)
	dnEyeBl   = hx(0x2a5aa8)
	dnEyeBr   = hx(0x4a2a14)
)

func genDnh(s *CharSpec, vw int, p Pose) *pixBuf {
	if dnhKind(s) == dnhDoraemon {
		return genDoraemon(s, vw, p)
	}
	return genDnhKid(s, vw, p, dnhKind(s))
}

// ---- Doraemon -------------------------------------------------------------------------------

func genDoraemon(s *CharSpec, vw int, p Pose) *pixBuf {
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
	hipY := 35 + bob
	by := 31 + bob // body centre row
	hcy := 18 + bob
	cx := 11.5 + float64(ox)
	hx0 := 11.5 + float64(hd)
	L, R := 11+hd, 12+hd

	leg := func(hxl float64, l Limb, sign float64, id uint8, dark bool) {
		b.cur = id
		mx, my, ex, ey, top := tiaLegIK(vw, p, hxl, float64(hipY), l, sign, 0.45, 2.3, 40)
		thickLine(b, hxl, float64(hipY), mx, my, 3, tiaDk(dnBlue, dark))
		thickLine(b, mx, my, ex, ey, 3, tiaDk(dnBlue, dark))
		ix := int(math.Floor(ex))
		if vw == vSide { // a round white foot
			b.rect(ix, top, 4, 1, tiaDk(dnWhite, dark))
			b.rect(ix-1, top+1, 6, 1, tiaDk(dnWhite, dark))
			b.rect(ix-1, top+2, 6, 1, tiaDk(dnWhiteD, dark))
			return
		}
		x0 := ix - 1
		if sign > 0 {
			x0 = ix
		}
		b.rect(x0, top, 4, 1, tiaDk(dnWhite, dark))
		b.rect(x0-1, top+1, 5, 1, tiaDk(dnWhite, dark))
		b.rect(x0-1, top+2, 5, 1, tiaDk(dnWhiteD, dark))
	}

	body := func() {
		b.cur = 2
		if vw == vSide {
			b.ellipse(cx+0.5, float64(by)+0.5, 5.2, 5.2, dnBlueD)
			b.ellipse(cx+0.5, float64(by)+0.5, 4.5, 4.6, dnBlue)
			b.ellipse(cx+2.5, float64(by)+1, 2.6, 3.8, dnWhite) // belly
			b.rect(int(cx)+2, by+1, 3, 1, dnWhiteD)             // pocket
			b.set(int(cx)+2, by+2, dnWhiteD)
			b.set(int(cx)+4, by+2, dnWhiteD)
			b.rect(int(cx)+2, by+3, 3, 1, dnWhiteD)
			// the tail sticking out behind, red ball at its tip
			b.set(int(cx)-5, by+2, dnRedD)
			b.rect(int(cx)-7, by+1, 2, 2, dnRed)
			b.set(int(cx)-7, by+1, dnRedL)
			// collar and bell
			b.rect(int(cx)-3, by-5, 8, 2, dnRed)
			b.rect(int(cx)-3, by-4, 8, 1, dnRedD)
			b.rect(int(cx)+3, by-4, 3, 3, dnGold)
			b.set(int(cx)+3, by-4, hx(0xfff0a0))
			b.set(int(cx)+4, by-2, dnGoldD)
			return
		}
		b.ellipse(cx, float64(by)+0.5, 5.9, 5.2, dnBlueD)
		b.ellipse(cx, float64(by)+0.5, 5.2, 4.6, dnBlue)
		b.set(int(cx)-4, by-1, dnBlueL)
		b.set(int(cx)-4, by, dnBlueL)
		if vw == vFront {
			b.ellipse(cx, float64(by)+1, 3.9, 3.8, dnWhite) // belly
			// the half-moon pocket
			b.rect(L-2, by+1, 6, 1, dnWhiteD)
			b.set(L-2, by+2, dnWhiteD)
			b.set(R+2, by+2, dnWhiteD)
			b.rect(L-1, by+3, 4, 1, dnWhiteD)
		} else { // the tail and its red ball on his back
			b.rect(L, hipY-2, 2, 1, dnRedD)
			b.rect(L, hipY-1, 2, 2, dnRed)
			b.set(L, hipY-1, dnRedL)
		}
		// red collar
		b.rect(int(cx)-5, by-5, 11, 2, dnRed)
		b.rect(int(cx)-5, by-4, 11, 1, dnRedD)
		if vw == vFront { // golden bell
			b.rect(L-1, by-4, 4, 3, dnGold)
			b.set(L-1, by-4, hx(0xfff0a0))
			b.rect(L, by-2, 2, 1, dnGoldD)
			b.rect(L-1, by-3, 4, 1, dnGoldD)
			b.set(L, by-1, dnInk)
			b.set(R, by-1, dnInk)
		}
	}

	head := func() {
		b.cur = 1
		hy := float64(hcy)
		switch vw {
		case vSide:
			b.ellipse(hx0, hy+0.5, 7.6, 7.3, dnBlueD)
			b.ellipse(hx0, hy+0.5, 6.9, 6.6, dnBlue)
			b.ellipse(hx0+2.2, hy+2.0, 5.0, 5.2, dnWhite) // the white face looks right
			b.set(L-4, hcy-4, dnBlueL)
			b.set(L-3, hcy-5, dnBlueL)
			// eye
			b.rect(R+2, hcy-5, 3, 4, dnWhite)
			b.set(R+2, hcy-5, dnInk)
			b.rect(R+2, hcy-2, 1, 1, dnInk)
			b.rect(R+4, hcy-4, 1, 2, dnInk)
			// red nose sticking out
			b.rect(R+6, hcy-1, 2, 2, dnRed)
			b.set(R+6, hcy-1, dnRedL)
			// mouth line and smile
			b.rect(R+5, hcy+1, 1, 2, dnInk)
			if open {
				b.rect(R+3, hcy+3, 4, 3, dnRedD)
				b.rect(R+3, hcy+5, 3, 1, dnTongue)
				b.rect(R+3, hcy+3, 4, 1, dnInk)
			} else {
				b.rect(R+3, hcy+4, 4, 1, dnInk)
				b.set(R+7, hcy+3, dnInk)
			}
			// whiskers
			b.line(R+5, hcy+1, R+7, hcy, dnInk)
			b.line(R+5, hcy+2, R+7, hcy+2, dnInk)
		case vBack:
			b.ellipse(hx0, hy+0.5, 7.9, 7.4, dnBlueD)
			b.ellipse(hx0, hy+0.5, 7.2, 6.7, dnBlue)
			b.set(L-4, hcy-4, dnBlueL)
			b.set(L-3, hcy-5, dnBlueL)
			b.set(L-2, hcy-5, dnBlueL)
		default:
			b.ellipse(hx0, hy+0.5, 7.9, 7.4, dnBlueD)
			b.ellipse(hx0, hy+0.5, 7.2, 6.7, dnBlue)
			b.ellipse(hx0, hy+2.2, 6.0, 5.4, dnWhite)
			b.set(L-5, hcy-4, dnBlueL)
			b.set(L-4, hcy-5, dnBlueL)
			// two tall white eyes touching in the middle, pupils looking in
			b.rect(L-2, hcy-6, 3, 5, dnWhite)
			b.rect(R, hcy-6, 3, 5, dnWhite)
			b.set(L-2, hcy-6, dnInk)
			b.set(R+2, hcy-6, dnInk)
			b.rect(L, hcy-4, 1, 2, dnInk)
			b.rect(R, hcy-4, 1, 2, dnInk)
			// red nose
			b.rect(L, hcy-1, 2, 2, dnRed)
			b.set(L, hcy-1, dnRedL)
			b.set(R, hcy, dnRedD)
			b.rect(L, hcy+1, 1, 2, dnInk) // the line down to the mouth
			if open {                     // wide open smile
				b.rect(L-3, hcy+3, 8, 1, dnInk)
				b.rect(L-3, hcy+4, 8, 2, dnRedD)
				b.rect(L-2, hcy+6, 6, 1, dnRedD)
				b.rect(L-1, hcy+5, 4, 2, dnTongue)
				b.set(L-4, hcy+3, dnInk)
				b.set(R+4, hcy+3, dnInk)
			} else {
				b.rect(L-2, hcy+4, 6, 1, dnInk)
				b.set(L-3, hcy+3, dnInk)
				b.set(R+3, hcy+3, dnInk)
			}
			// whiskers, three on each side
			b.line(L-3, hcy, L-6, hcy-1, dnInk)
			b.line(L-3, hcy+1, L-6, hcy+1, dnInk)
			b.line(L-4, hcy+2, L-6, hcy+3, dnInk)
			b.line(R+3, hcy, R+6, hcy-1, dnInk)
			b.line(R+3, hcy+1, R+6, hcy+1, dnInk)
			b.line(R+4, hcy+2, R+6, hcy+3, dnInk)
		}
	}

	arm := func(sx float64, l Limb, sign float64, id uint8, dark bool) (int, int) {
		b.cur = id
		sy := float64(by - 2)
		mx, my, ex, ey := tiaArmIK(vw, sx, sy, l, sign, 0.7, 2.8)
		thickLine(b, sx, sy, mx, my, 2, tiaDk(dnBlue, dark))
		thickLine(b, mx, my, ex, ey, 2, tiaDk(dnBlue, dark))
		ix, iy := int(math.Floor(ex)), int(math.Floor(ey))
		b.ellipse(float64(ix)+1, float64(iy)+1.5, 1.6, 1.6, tiaDk(dnWhite, dark)) // round hand
		b.set(ix+1, iy+2, tiaDk(dnWhiteD, dark))
		return ix, iy + 1
	}

	switch vw {
	case vSide:
		arm(cx-1, p.LA, 1, 3, true)
		leg(10, p.LF, 1, 5, true)
		leg(11, p.RF, 1, 6, false)
		body()
		head()
		x, y := arm(cx, p.RA, 1, 4, false)
		tiaHeld(b, s, p, vw, x, y, 4)
	case vBack:
		leg(13, p.LF, 1, 5, false)
		leg(9, p.RF, -1, 6, false)
		body()
		arm(cx+5, p.LA, 1, 3, false)
		x, y := arm(cx-6, p.RA, -1, 4, false)
		head()
		tiaHeld(b, s, p, vw, x, y, 4)
	default:
		leg(9, p.LF, -1, 5, false)
		leg(13, p.RF, 1, 6, false)
		body()
		head()
		arm(cx-6, p.LA, -1, 3, false)
		x, y := arm(cx+5, p.RA, 1, 4, false)
		tiaHeld(b, s, p, vw, x, y, 4)
	}
	return b
}

// ---- Nobita and the wizard pupils ----------------------------------------------------------

func genDnhKid(s *CharSpec, vw int, p Pose, k int) *pixBuf {
	b := newPix(charW, charH)
	if trackParts {
		b.pid = make([]uint8, charW*charH)
	}
	nob := k == dnhNobita
	herm := k == dnhHermione
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
	hipY, tH := 32+bob, 8 // wizard pupils
	legK, legSeg := 0.65, 4.2
	if nob {
		hipY, tH = 34+bob, 7
		legK, legSeg = 0.55, 3.2
	}
	ty0 := hipY - tH // shoulders
	hy := ty0 - 12   // skull top
	L, R := 11+hd, 12+hd
	hcx := 11.5 + float64(hd)

	hairC, hairL, hairD := dnHairK, dnHairKL, dnRobeD
	switch k {
	case dnhHermione:
		hairC, hairL, hairD = dnBrown, dnBrownL, dnBrownD
	case dnhRon:
		hairC, hairL, hairD = dnOrange, dnOrangeL, dnOrangeD
	}

	leg := func(hxl float64, l Limb, sign float64, id uint8, dark bool) {
		b.cur = id
		mx, my, ex, ey, top := tiaLegIK(vw, p, hxl, float64(hipY+1), l, sign, legK, legSeg, 40)
		thigh, shin := dnTrou, dnTrou
		switch {
		case nob:
			thigh, shin = dnSkin, dnSkin
		case herm:
			thigh, shin = dnSkin, dnWhite // long white socks
		}
		thickLine(b, hxl, float64(hipY+1), mx, my, 2, tiaDk(thigh, dark))
		thickLine(b, mx, my, ex, ey, 2, tiaDk(shin, dark))
		if !nob && !herm {
			thickLine(b, hxl, float64(hipY+2), mx, my, 1, tiaDk(dnTrouL, dark))
		}
		ix := int(math.Floor(ex))
		if nob { // white socks above the shoes
			b.rect(ix, top-1, 2, 1, tiaDk(dnWhite, dark))
		}
		shoe, shoeL := dnShoeK, dnShoeKL
		if nob {
			shoe, shoeL = dnShoe, dnShoeL
		}
		if vw == vSide {
			b.rect(ix, top, 3, 1, tiaDk(shoe, dark))
			b.rect(ix-1, top+1, 5, 2, tiaDk(shoe, dark))
			b.rect(ix+1, top+1, 2, 1, tiaDk(shoeL, dark))
			return
		}
		x0 := ix - 1
		if sign > 0 {
			x0 = ix
		}
		b.rect(x0, top, 3, 1, tiaDk(shoe, dark))
		b.rect(x0-1+boolI(sign > 0), top+1, 3, 2, tiaDk(shoe, dark))
		b.set(x0, top+1, tiaDk(shoeL, dark))
	}

	// Hermione's bushy hair falls behind the shoulders: drawn before the torso.
	backHair := func() {
		if !herm {
			return
		}
		b.cur = 1
		switch vw {
		case vSide:
			b.ellipse(hcx-1.5, float64(hy)+7.5, 4.6, 7.2, hairC)
			b.set(L-4, hy+11, hairD)
			b.set(L-2, hy+13, hairD)
		default:
			b.ellipse(hcx, float64(hy)+7.5, 6.4, 7.6, hairC)
			b.set(L-5, hy+12, hairD)
			b.set(R+5, hy+12, hairD)
			b.set(L-4, hy+14, hairD)
			b.set(R+4, hy+14, hairD)
		}
	}

	torso := func() {
		b.cur = 2
		if nob {
			nobTorso(b, vw, ox, ty0, hipY, L, R)
			return
		}
		wizTorso(b, vw, ox, ty0, hipY, L, R, herm)
	}

	head := func() {
		b.cur = 1
		hc := float64(hy) + 6
		rx, ry := 4.9, 5.6
		if nob {
			rx = 5.2
		}
		b.rect(L, hy+11, 2, 2, dnSkinD) // neck
		switch vw {
		case vSide:
			b.ellipse(hcx+0.5, hc, rx-0.3, ry, dnSkin)
			// hair cap over the back and the top of the head
			b.ellipse(hcx-0.6, hc-1.4, rx-0.4, ry-1.4, hairC)
			b.rect(R+1, hy+6, 4, 6, dnSkin) // the face in front
			b.rect(R, hy+7, 1, 4, dnSkin)
			b.rect(R+4, hy+5, 1, 6, dnSkinD)
			b.set(R+5, hy+8, dnSkin) // nose
			b.set(L-2, hy+2, hairL)
			b.rect(L-1, hy+7, 2, 2, dnSkinD) // ear
			switch k {
			case dnhNobita:
				b.set(R+1, hy, hairC) // the peak at the front
				b.set(R+2, hy-1, hairC)
				b.set(R+3, hy+5, hairC)
			case dnhHarry, dnhRon:
				b.set(L-4, hy+1, hairC) // messy tufts
				b.set(L-1, hy-1, hairC)
				b.set(R+1, hy, hairC)
				b.set(R+3, hy+5, hairC)
				b.set(L-5, hy+5, hairC)
			case dnhHermione:
				b.rect(R+2, hy+4, 3, 1, hairC) // fringe
				b.set(R+3, hy+5, hairL)
			}
			if k == dnhNobita || k == dnhHarry { // round glasses
				b.rect(R+2, hy+6, 3, 3, dnFrame)
				b.set(R+3, hy+7, dnLens)
				b.rect(L+1, hy+7, 2, 1, dnFrame)
			} else {
				eye := dnEyeBr
				if k == dnhRon {
					eye = dnEyeBl
				}
				b.set(R+3, hy+7, eye)
				b.set(R+3, hy+6, hairD)
			}
			if k == dnhHarry {
				b.set(R+4, hy+4, dnScar)
			}
			if k == dnhRon {
				b.set(R+2, hy+9, dnFreckle)
				b.set(R+4, hy+9, dnFreckle)
			}
			if open {
				b.rect(R+3, hy+10, 2, 1, dnMouth)
			} else {
				b.set(R+3, hy+10, dnMouth)
			}
		case vBack:
			b.ellipse(hcx, hc, rx, ry, dnSkin)
			b.rect(L-5, hy+6, 1, 2, dnSkinD) // ears
			b.rect(R+5, hy+6, 1, 2, dnSkinD)
			if herm {
				b.ellipse(hcx, float64(hy)+6.5, 6.2, 7.2, hairC)
				for _, q := range [][2]int{{-4, 3}, {2, 2}, {-1, 7}, {3, 9}, {-3, 11}, {1, 12}} {
					b.set(L+q[0], hy+q[1], hairL)
				}
			} else {
				b.ellipse(hcx, hc-0.6, rx, ry-0.3, hairC)
				b.rect(L-2, hy+10, 6, 1, dnSkinD) // nape
				b.set(L-2, hy+3, hairL)
				b.set(R+1, hy+2, hairL)
			}
			switch k {
			case dnhNobita:
				b.set(L+2, hy-1, hairC)
				b.set(L+3, hy-2, hairC)
			case dnhHarry, dnhRon:
				b.set(L-3, hy, hairC)
				b.set(L, hy-1, hairC)
				b.set(R+2, hy-1, hairC)
				b.set(R+4, hy+1, hairC)
				b.set(L-5, hy+4, hairC)
				b.set(R+5, hy+4, hairC)
			}
			if k == dnhNobita || k == dnhHarry {
				b.set(L-5, hy+7, dnFrame) // glasses arms over the ears
				b.set(R+5, hy+7, dnFrame)
			}
		default:
			b.ellipse(hcx, hc, rx, ry, dnSkin)
			b.rect(R+4, hy+6, 1, 4, dnSkinD) // shaded cheek
			b.rect(L-5, hy+6, 1, 2, dnSkin)  // ears
			b.rect(R+5, hy+6, 1, 2, dnSkinD)
			// hair cap and fringe
			b.ellipse(hcx, hc-2.6, rx+0.2, ry-2.4, hairC)
			b.set(L-2, hy+1, hairL)
			b.set(L-1, hy+1, hairL)
			switch k {
			case dnhNobita: // a neat fringe and the tuft in a peak
				b.rect(L-4, hy+4, 2, 1, hairC)
				b.rect(R+3, hy+4, 2, 1, hairC)
				b.set(L, hy-1, hairC)
				b.set(L+1, hy-2, hairC)
				b.set(L+1, hy-1, hairL)
			case dnhHarry, dnhRon: // messy spikes all round
				b.set(L-3, hy, hairC)
				b.set(L, hy-1, hairC)
				b.set(R+2, hy-1, hairC)
				b.set(R+4, hy+1, hairC)
				b.set(L-5, hy+3, hairC)
				b.set(R+5, hy+3, hairC)
				b.set(L-5, hy+4, hairC)
				b.set(R+5, hy+4, hairC)
				for _, x := range []int{L - 3, L - 1, R + 1, R + 3} { // a jagged fringe
					b.set(x, hy+5, hairC)
				}
				b.set(R, hy+5, hairD)
			case dnhHermione: // a bushy fringe and curls
				b.rect(L-4, hy+4, 10, 1, hairC)
				for _, x := range []int{L - 4, L - 1, R + 2} {
					b.set(x, hy+5, hairC)
				}
				for _, q := range [][2]int{{-4, 2}, {2, 1}, {-6, 8}, {7, 9}, {-6, 12}, {6, 13}, {-1, 3}} {
					b.set(L+q[0], hy+q[1], hairL)
				}
				b.set(L-5, hy+10, hairD)
				b.set(R+5, hy+10, hairD)
			}
			// eyes
			if k == dnhNobita || k == dnhHarry {
				b.rect(L-4, hy+6, 4, 4, dnFrame) // round glasses
				b.rect(R, hy+6, 4, 4, dnFrame)
				b.rect(L-3, hy+7, 2, 2, dnLens)
				b.rect(R+1, hy+7, 2, 2, dnLens)
				b.set(L-3, hy+6, dnSkin) // rounded corners
				b.set(L-4, hy+6, dnSkin)
				b.set(R+3, hy+6, dnSkin)
				b.set(R+3, hy+9, dnSkin)
				b.set(L-4, hy+9, dnSkin)
				pc := dnInk
				if k == dnhHarry {
					pc = hx(0x2a7a3a) // green eyes
				}
				b.set(L-2, hy+8, pc)
				b.set(R+1, hy+8, pc)
			} else {
				eye := dnEyeBr
				if k == dnhRon {
					eye = dnEyeBl
				}
				b.rect(L-3, hy+7, 2, 2, dnWhite)
				b.rect(R+1, hy+7, 2, 2, dnWhite)
				b.rect(L-2, hy+7, 1, 2, eye)
				b.rect(R+1, hy+7, 1, 2, eye)
				b.rect(L-3, hy+6, 2, 1, hairD) // brows
				b.rect(R+1, hy+6, 2, 1, hairD)
			}
			if k == dnhHarry { // the lightning scar
				b.set(R+1, hy+4, dnScar)
				b.set(R+2, hy+5, dnScar)
			}
			if k == dnhRon {
				for _, q := range [][2]int{{-4, 9}, {-3, 10}, {-2, 9}, {3, 9}, {4, 10}, {5, 9}} {
					b.set(L+q[0], hy+q[1], dnFreckle)
				}
			}
			b.set(L, hy+9, dnSkinD) // nose
			if open {
				b.rect(L-1, hy+10, 3, 1, dnMouth)
				b.rect(L, hy+10, 2, 1, dnLip)
				b.rect(L-1, hy+10, 4, 1, dnMouth)
			} else {
				b.rect(L, hy+10, 2, 1, dnMouth)
				b.set(L-1, hy+10, dnLip)
			}
		}
	}

	arm := func(sx float64, l Limb, sign float64, id uint8, dark bool) (int, int) {
		b.cur = id
		sy := float64(ty0 + 1)
		mx, my, ex, ey := tiaArmIK(vw, sx, sy, l, sign, 0.85, 3.6)
		up, low := dnRobe, dnRobe
		if nob {
			up, low = dnYellow, dnSkin
		}
		thickLine(b, sx, sy, mx, my, 2, tiaDk(up, dark))
		thickLine(b, mx, my, ex, ey, 2, tiaDk(low, dark))
		if !nob {
			thickLine(b, sx, sy, mx, my, 1, tiaDk(dnRobeL, dark))
		}
		ix, iy := int(math.Floor(ex)), int(math.Floor(ey))
		if !nob { // the wide sleeve end
			b.rect(ix-boolI(sign < 0), iy, 3, 1, tiaDk(dnRobe, dark))
		}
		b.rect(ix, iy+1, 2, 2, tiaDk(dnSkin, dark))
		b.set(ix+1, iy+2, tiaDk(dnSkinD, dark))
		return ix, iy + 1
	}

	lx, rx := float64(6+ox), float64(16+ox)
	if !nob {
		lx, rx = float64(5+ox), float64(17+ox)
	}
	switch vw {
	case vSide:
		cx := float64(11 + ox)
		arm(cx-0.5, p.LA, 1, 3, true)
		leg(10, p.LF, 1, 5, true)
		leg(11, p.RF, 1, 6, false)
		backHair()
		torso()
		head()
		x, y := arm(cx, p.RA, 1, 4, false)
		tiaHeld(b, s, p, vw, x, y, 4)
	case vBack:
		leg(13, p.LF, 1, 5, false)
		leg(9, p.RF, -1, 6, false)
		torso()
		arm(rx, p.LA, 1, 3, false)
		x, y := arm(lx, p.RA, -1, 4, false)
		head()
		tiaHeld(b, s, p, vw, x, y, 4)
	default:
		leg(9, p.LF, -1, 5, false)
		leg(13, p.RF, 1, 6, false)
		backHair()
		torso()
		head()
		arm(lx, p.LA, -1, 3, false)
		x, y := arm(rx, p.RA, 1, 4, false)
		tiaHeld(b, s, p, vw, x, y, 4)
	}
	return b
}

// nobTorso: Nobita's yellow shirt with its collar and the blue shorts.
func nobTorso(b *pixBuf, vw, ox, ty0, hipY, L, R int) {
	if vw == vSide {
		c := 11 + ox
		for r := 0; r < hipY-ty0; r++ {
			x0, x1 := c-2, c+2
			if r == 0 {
				x0, x1 = c-1, c+1
			}
			for x := x0; x <= x1; x++ {
				col := dnYellow
				if x == x0 {
					col = dnYellowD
				} else if x == x1 {
					col = dnYellowL
				}
				b.set(x, ty0+r, col)
			}
		}
		b.set(c+2, ty0, dnYellowL) // collar point
		b.rect(c-2, hipY, 5, 3, dnShorts)
		b.rect(c-2, hipY+2, 5, 1, dnShortsD)
		return
	}
	for r := 0; r < hipY-ty0; r++ {
		x0, x1 := 8, 15
		if r == 0 {
			x0, x1 = 9, 14
		}
		for x := x0; x <= x1; x++ {
			col := dnYellow
			switch {
			case x == x0:
				col = dnYellowL
			case x == x1:
				col = dnYellowD
			}
			b.set(x+ox, ty0+r, col)
		}
	}
	b.rect(8+ox, hipY-1, 8, 1, dnYellowD) // the hem
	if vw == vFront {
		b.rect(L, ty0, 2, 1, dnSkinD) // neck in the V
		b.set(L-1, ty0, dnYellowL)    // the collar
		b.set(R+1, ty0, dnYellowL)
		b.set(L-2, ty0+1, dnYellowL)
		b.set(L-1, ty0+1, dnYellowD)
		b.set(R+1, ty0+1, dnYellowD)
		b.set(R+2, ty0+1, dnYellowL)
		b.set(L, ty0+1, dnYellowD)
		b.set(R, ty0+1, dnYellowD)
	} else {
		b.rect(9+ox, ty0, 6, 1, dnYellowL) // collar on the back
		b.rect(9+ox, ty0+1, 6, 1, dnYellowD)
	}
	// the shorts, with a gap between the legs
	b.rect(8, hipY, 8, 2, dnShorts)
	b.rect(8, hipY+2, 3, 1, dnShorts)
	b.rect(13, hipY+2, 3, 1, dnShorts)
	b.rect(8, hipY, 8, 1, dnShortsD) // the waistband
}

// wizTorso: the open black school robe, grey jumper, white collar and striped tie (with
// Hermione's pleated skirt).
func wizTorso(b *pixBuf, vw, ox, ty0, hipY, L, R int, herm bool) {
	if vw == vSide {
		c := 11 + ox
		for r := 0; r <= hipY-ty0+2; r++ {
			x0, x1 := c-2, c+2
			if r == 0 {
				x0, x1 = c-1, c+1
			}
			if r > hipY-ty0 { // the robe hangs below the hips, behind the legs
				x0, x1 = c-3, c-1
			}
			for x := x0; x <= x1; x++ {
				col := dnRobe
				if x == x0 {
					col = dnRobeD
				}
				b.set(x, ty0+r, col)
			}
		}
		b.rect(c+2, ty0+1, 1, hipY-ty0-1, dnGrey) // the jumper shows at the front
		b.set(c+2, ty0, dnWhite)
		b.set(c+3, ty0+1, dnTieR) // the tie
		b.set(c+3, ty0+2, dnTieG)
		b.set(c+3, ty0+3, dnTieR)
		if herm {
			b.rect(c-1, hipY, 4, 3, dnGrey)
			b.set(c+2, hipY+2, dnGreyD)
			b.set(c, hipY+2, dnGreyD)
		}
		return
	}
	h := hipY - ty0
	for r := 0; r <= h+2; r++ {
		x0, x1 := 7, 16
		if r == 0 {
			x0, x1 = 8, 15
		}
		if r > h { // two flaps below the hips
			for _, x := range []int{7, 8, 15, 16} {
				b.set(x+ox, ty0+r, dnRobe)
			}
			if vw == vBack {
				b.rect(9+ox, ty0+r, 6, 1, dnRobe)
			}
			continue
		}
		for x := x0; x <= x1; x++ {
			col := dnRobe
			switch {
			case x == x0:
				col = dnRobeL
			case x == x1:
				col = dnRobeD
			}
			b.set(x+ox, ty0+r, col)
		}
	}
	if vw == vFront {
		// the open front: grey jumper with a V of white shirt and the tie
		b.rect(10+ox, ty0+1, 4, h-1, dnGrey)
		b.rect(10+ox, ty0+1, 1, h-1, dnGreyL)
		b.rect(13+ox, ty0+1, 1, h-1, dnGreyD)
		b.rect(L-1, ty0, 4, 1, dnWhite) // collar
		b.rect(L, ty0+1, 2, 2, dnWhite)
		for r := 1; r <= 5; r++ { // red and gold stripes down the tie
			c := dnTieR
			if r%2 == 0 {
				c = dnTieG
			}
			b.rect(L, ty0+r, 2, 1, c)
		}
		b.set(L, ty0+6, dnTieR)
		b.rect(9+ox, ty0+1, 1, h, dnRobeL) // robe lapels
		b.rect(14+ox, ty0+1, 1, h, dnRobeL)
		// the house crest on the robe
		b.set(8+ox, ty0+3, dnTieR)
		b.set(8+ox, ty0+4, dnTieG)
		if herm { // the pleated grey skirt
			b.rect(9+ox, hipY, 6, 3, dnGrey)
			b.rect(8+ox, hipY+2, 8, 1, dnGrey)
			for x := 9; x <= 15; x += 2 {
				b.set(x+ox, hipY+1, dnGreyD)
				b.set(x+ox, hipY+2, dnGreyD)
			}
		} else {
			b.rect(9+ox, hipY, 6, 1, dnTrou)
		}
	} else {
		// the hood folded on the shoulders and the back seam
		b.rect(9+ox, ty0, 6, 2, dnRobeL)
		b.rect(10+ox, ty0+2, 4, 1, dnRobeL)
		b.rect(11+ox, ty0+3, 2, h, dnRobeD)
		if herm {
			b.rect(9+ox, hipY+3, 6, 1, dnGrey)
		}
	}
}

// ---- the new guest: Ron ----

// presetRon: the red-haired friend of the wizard pupils (new famous guest, comes with Harry and
// Hermione). It takes nothing from r, so adding him does not change the random looks of the
// guests listed after him.
func presetRon(r *rand.Rand) CharSpec {
	s := CharSpec{Name: "ron", Accent: dnTieR, HatCol: dnRobe}
	s.Skin, s.Hair = dnSkin, dnOrange
	s.Top, s.Shirt, s.Shirt2 = TopRobe, dnRobe, dnRobe
	s.Bottom, s.Pants, s.Shoes = BotPants, dnTrou, dnShoeK
	s.Hair2, s.Body, s.Stature = HairMessy, BodyNormal, 1
	s.Glasses, s.Beard, s.Hat, s.Carry, s.Weapon = GlassNone, BeardNone, HatNone, HoldNone, HoldNone
	s.Pattern, s.Face, s.Eyes, s.Ears, s.Head = 0, 0, 0, 0, HeadNormal
	return s
}

func init() {
	famousPresets = append(famousPresets, presetRon)
	displayNameOverrides["ron"] = "Ron"
	starGroups = append(starGroups, []string{"harry_potter", "hermione", "ron"})
	charDescs["ron"] = [2]string{"Pelirrojo con varita rota y miedo a las arañas", "Redhead with a broken wand, scared of spiders"}
	if _, ok := catchphrases["ron"]; !ok {
		catchphrases["ron"] = []string{"¿Por qué arañas? ¿Por qué no mariposas?", "¡Sangre de dragón!"}
	}
	for es, en := range map[string]string{
		"Pelirrojo con varita rota y miedo a las arañas": "Redhead with a broken wand, scared of spiders",
		"¿Por qué arañas? ¿Por qué no mariposas?":        "Why spiders? Why not butterflies?",
		"¡Sangre de dragón!":                             "Dragon's blood!",
	} {
		if _, ok := langDataEN[es]; !ok {
			langDataEN[es] = en
		}
	}
}
