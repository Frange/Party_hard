package main

import (
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// Chewbacca, Steve (the block man), the four plumbers (Mario, Luigi, Wario, Waluigi), Gandalf,
// Frodo and Knuckles have their own procedural drawing (genCsm), like Mortadelo or Neo: the generic
// body cannot give them their builds (Steve's blocks, Wario's belly, Waluigi's stilts, the hobbit's
// big bare feet, the wizard's robe). Our own small pixel-art parodies, drawn close to the user's
// references:
//   - Chewbacca: taller than a guest, brown fur with a dark brown crown, dark eyes, a pink mouth,
//     broad shoulders, dark greenish brown body, lighter orange-brown arms, a grey bandolier with
//     cartridges across the chest, brown legs and broad brown feet.
//   - Steve: square head, dark brown hair, white and violet eyes, brown nose and beard, a short
//     sleeved turquoise tee, pinkish brown arms, indigo trousers, grey shoes; a stone sword.
//   - Mario / Luigi / Wario / Waluigi: cap with the letter (M, L, W, Γ), moustache, nose, shirt,
//     overalls with yellow buttons, white gloves and their shoes; Luigi taller and thinner than
//     Mario, Wario short and wide, Waluigi very tall and thin with pointy ears.
//   - Gandalf: tall pointed grey hat with a wide brim, long light grey beard down to the waist, a
//     long dark grey robe with light trims, brown belt with a golden buckle, a long brown staff
//     with a white knob.
//   - Frodo: short, wavy reddish brown hair, pale skin, a grey-green cloak with a golden brooch,
//     burgundy waistcoat on a white shirt, dark purple trousers and big bare feet.
//   - Knuckles: red echidna with dreadlocks swept back, white eyes, cream muzzle, the white
//     crescent on the chest, big white spiked fists and red shoes with a white strap.
// Only the drawing changes: hitbox, footprint, stats and collisions are the ones of every guest.
// The animation is the usual one, read from the pose. Parts: 1 head, 2 body, 3/4 arms, 5/6 legs.
// ---------------------------------------------------------------------------

const (
	csmNone = iota
	csmChewie
	csmSteve
	csmMario
	csmLuigi
	csmWario
	csmWaluigi
	csmGandalf
	csmFrodo
	csmKnuckles
)

// noCsmArt turns the own drawing off (tests compare with the stock body).
var noCsmArt bool

var csmKinds = map[string]int{
	"chewbacca": csmChewie, "steve_cubos": csmSteve, "mario": csmMario, "luigi": csmLuigi,
	"wario": csmWario, "waluigi": csmWaluigi, "gandalf": csmGandalf, "frodo": csmFrodo,
	"knuckles": csmKnuckles,
}

func csmKind(s *CharSpec) int {
	if noCsmArt || s == nil {
		return csmNone
	}
	return csmKinds[s.Name]
}

func isCsm(s *CharSpec) bool { return csmKind(s) != csmNone }

// csmLook: build and colours of one of them (rows are for a standing pose, bob 0).
type csmLook struct {
	top, sh, hip, foot int // skull (or hat) top, shoulder row, hip row, top row of the shoes
	half, waist, depth int // torso half width at shoulders / waist (front), half depth (side)
	legL, legR, legW   int // left pixel of each leg (front view) and leg width
	armL, armR, armW   int // left pixel of each arm at the shoulder (front view) and arm width
	armSeg, legSeg     float64
	armK, legK         float64
	sleeveFrac         float64 // share of the upper arm covered by the sleeve
	body, bodyL, bodyD color.RGBA
	sleeve, fore       color.RGBA
	hand, handD        color.RGBA
	handW, handH       int
	pants, pantsL      color.RGBA
	shoe, shoeL, shoeD color.RGBA
	shoeW, shoeLen     int
	shoeH              int
	belly              bool
}

var (
	// Chewbacca
	cwFur   = hx(0x7a4a26)
	cwFurL  = hx(0x9a6436)
	cwFurD  = hx(0x55321a)
	cwCrown = hx(0x40240f)
	cwFace  = hx(0xa06a3a)
	cwMuz   = hx(0xb8844e)
	cwEye   = hx(0x120806)
	cwNose  = hx(0x2a1610)
	cwPink  = hx(0xd47a7a)
	cwBody  = hx(0x4e4426)
	cwBodyL = hx(0x6a5c34)
	cwBodyD = hx(0x342c18)
	cwArm   = hx(0xb07038)
	cwArmD  = hx(0x80501f)
	cwHand  = hx(0x5e3818)
	cwStrap = hx(0x6e6e72)
	cwShell = hx(0xc4c4c8)
	cwShelD = hx(0x242426)
	// Steve
	svHair  = hx(0x4a2f1c)
	svHairL = hx(0x5e3c24)
	svSkin  = hx(0xbc8a6e)
	svSkinD = hx(0x9c6c52)
	svEyeW  = hx(0xf6f6f6)
	svEyeV  = hx(0x6a48b0)
	svNose  = hx(0x8a5236)
	svBeard = hx(0x5a3626)
	svShirt = hx(0x22a6aa)
	svShirL = hx(0x46c2c2)
	svShirD = hx(0x167a80)
	svPants = hx(0x403498)
	svPantL = hx(0x5446b4)
	svShoe  = hx(0x6c6c70)
	svShoeL = hx(0x8e8e94)
	svShoeD = hx(0x4a4a4e)
	svBlade = hx(0xa8a8b0)
	svBladL = hx(0xd8d8de)
	svBladD = hx(0x1c1c20)
	svGrip  = hx(0x6a4422)
	// plumbers
	plSkin   = hx(0xf8c498)
	plSkinD  = hx(0xd09a70)
	plNose   = hx(0xf8b088)
	plNoseD  = hx(0xd8865e)
	plHair   = hx(0x4a2810)
	plMous   = hx(0x1a1010)
	plWhite  = hx(0xf8f8f8)
	plWhiteD = hx(0xc8c8d4)
	plPupil  = hx(0x1a2a8a)
	plButton = hx(0xf8d020)
	plMouth  = hx(0x8a1a1a)
	// Gandalf
	gdHat   = hx(0x8a8c94)
	gdHatL  = hx(0xb0b2ba)
	gdHatD  = hx(0x5c5e66)
	gdRobe  = hx(0x55575f)
	gdRobeL = hx(0x72747c)
	gdRobeD = hx(0x3a3c42)
	gdTrim  = hx(0xb0b2b8)
	gdBeard = hx(0xe2e2e6)
	gdBearD = hx(0xb4b4bc)
	gdHair  = hx(0xbcbcc4)
	gdSkin  = hx(0xeec4a2)
	gdSkinD = hx(0xc89a7c)
	gdBelt  = hx(0x6a4422)
	gdGold  = hx(0xe8b830)
	gdStaff = hx(0x7a5232)
	gdStafD = hx(0x523418)
	gdBoot  = hx(0x3e2a1c)
	// Frodo
	frHair  = hx(0x6e3a3c)
	frHairL = hx(0x8e5050)
	frHairD = hx(0x48242a)
	frSkin  = hx(0xf8dcc8)
	frSkinD = hx(0xd8b4a0)
	frEye   = hx(0x2a4a8a)
	frVest  = hx(0x7a2840)
	frVestL = hx(0x9a3a58)
	frVestD = hx(0x561a2c)
	frShirt = hx(0xf0ece0)
	frShirD = hx(0xc8c4b8)
	frPants = hx(0x4a2850)
	frPantL = hx(0x643a6a)
	frCape  = hx(0x5e6e58)
	frCapeL = hx(0x7a8a72)
	frCapeD = hx(0x44503e)
	frGold  = hx(0xf0c840)
	frFeet  = hx(0xf2d0b4)
	frFeetD = hx(0xc8a080)
	frFeetH = hx(0x7a4430)
	// Knuckles
	knRed   = hx(0xd82a2a)
	knRedL  = hx(0xf05448)
	knRedD  = hx(0x961616)
	knCream = hx(0xf2d2a2)
	knCreaD = hx(0xd0aa7a)
	knPupil = hx(0x4a1a5a)
	knGreen = hx(0x2aa040)
	knYell  = hx(0xf0d020)
)

func csmLookOf(k int) csmLook {
	switch k {
	case csmChewie:
		return csmLook{top: 4, sh: 14, hip: 26, foot: 40, half: 6, waist: 5, depth: 3,
			legL: 8, legR: 13, legW: 3, armL: 4, armR: 17, armW: 3, armSeg: 5.6, legSeg: 6.4, armK: 1.15, legK: 0.85,
			sleeveFrac: 1, body: cwBody, bodyL: cwBodyL, bodyD: cwBodyD, sleeve: cwArm, fore: cwArm,
			hand: cwHand, handD: cwFurD, handW: 3, handH: 2, pants: cwFur, pantsL: cwFurL,
			shoe: cwFurD, shoeL: cwFur, shoeD: cwCrown, shoeW: 4, shoeLen: 6, shoeH: 3}
	case csmSteve:
		return csmLook{top: 7, sh: 15, hip: 26, foot: 40, half: 4, waist: 4, depth: 2,
			legL: 8, legR: 12, legW: 4, armL: 5, armR: 16, armW: 3, armSeg: 5.0, legSeg: 6.4, armK: 1.1, legK: 0.75,
			sleeveFrac: 0.5, body: svShirt, bodyL: svShirL, bodyD: svShirD, sleeve: svShirt, fore: svSkin,
			hand: svSkin, handD: svSkinD, handW: 3, handH: 2, pants: svPants, pantsL: svPantL,
			shoe: svShoe, shoeL: svShoeL, shoeD: svShoeD, shoeW: 4, shoeLen: 5, shoeH: 2}
	case csmMario:
		return csmLook{top: 11, sh: 23, hip: 32, foot: 40, half: 5, waist: 5, depth: 3,
			legL: 8, legR: 13, legW: 3, armL: 5, armR: 16, armW: 2, armSeg: 4.0, legSeg: 3.4, armK: 1, legK: 0.5,
			sleeveFrac: 1, body: hx(0xe02828), bodyL: hx(0xf45454), bodyD: hx(0xa01818), sleeve: hx(0xe02828), fore: hx(0xe02828),
			hand: plWhite, handD: plWhiteD, handW: 3, handH: 3, pants: hx(0x2a4ad0), pantsL: hx(0x4a6ae8),
			shoe: hx(0x7a4018), shoeL: hx(0x9a5828), shoeD: hx(0x4a2408), shoeW: 4, shoeLen: 6, shoeH: 3, belly: true}
	case csmLuigi:
		return csmLook{top: 8, sh: 20, hip: 30, foot: 40, half: 4, waist: 4, depth: 2,
			legL: 9, legR: 13, legW: 2, armL: 6, armR: 15, armW: 2, armSeg: 4.6, legSeg: 4.4, armK: 1, legK: 0.65,
			sleeveFrac: 1, body: hx(0x30a838), bodyL: hx(0x58cc5e), bodyD: hx(0x1e7a24), sleeve: hx(0x30a838), fore: hx(0x30a838),
			hand: plWhite, handD: plWhiteD, handW: 3, handH: 3, pants: hx(0x2a3ca8), pantsL: hx(0x4458c4),
			shoe: hx(0x7a4018), shoeL: hx(0x9a5828), shoeD: hx(0x4a2408), shoeW: 4, shoeLen: 6, shoeH: 3}
	case csmWario:
		return csmLook{top: 13, sh: 25, hip: 33, foot: 40, half: 6, waist: 7, depth: 4,
			legL: 7, legR: 14, legW: 3, armL: 3, armR: 18, armW: 3, armSeg: 3.6, legSeg: 3.0, armK: 0.95, legK: 0.45,
			sleeveFrac: 1, body: hx(0xf8d020), bodyL: hx(0xfff07a), bodyD: hx(0xc09a10), sleeve: hx(0xf8d020), fore: hx(0xf8d020),
			hand: plWhite, handD: plWhiteD, handW: 3, handH: 3, pants: hx(0x7a38a8), pantsL: hx(0x9a58c8),
			shoe: hx(0x2a9a3a), shoeL: hx(0x4cc05a), shoeD: hx(0x1a6a24), shoeW: 5, shoeLen: 7, shoeH: 3, belly: true}
	case csmWaluigi:
		return csmLook{top: 4, sh: 16, hip: 25, foot: 40, half: 3, waist: 3, depth: 2,
			legL: 9, legR: 13, legW: 2, armL: 7, armR: 15, armW: 2, armSeg: 5.4, legSeg: 6.9, armK: 1.1, legK: 0.9,
			sleeveFrac: 1, body: hx(0x6a2c9a), bodyL: hx(0x8a4cba), bodyD: hx(0x4a1a70), sleeve: hx(0x6a2c9a), fore: hx(0x6a2c9a),
			hand: plWhite, handD: plWhiteD, handW: 3, handH: 3, pants: hx(0x262650), pantsL: hx(0x3c3c70),
			shoe: hx(0xe88a20), shoeL: hx(0xf8aa48), shoeD: hx(0xa85a10), shoeW: 4, shoeLen: 7, shoeH: 2}
	case csmGandalf:
		return csmLook{top: 3, sh: 17, hip: 27, foot: 40, half: 5, waist: 4, depth: 3,
			legL: 9, legR: 13, legW: 2, armL: 5, armR: 16, armW: 3, armSeg: 4.8, legSeg: 6.2, armK: 1.1, legK: 0.8,
			sleeveFrac: 1, body: gdRobe, bodyL: gdRobeL, bodyD: gdRobeD, sleeve: gdRobe, fore: gdRobe,
			hand: gdSkin, handD: gdSkinD, handW: 2, handH: 2, pants: gdRobeD, pantsL: gdRobe,
			shoe: gdBoot, shoeL: hx(0x5a4030), shoeD: hx(0x261810), shoeW: 3, shoeLen: 5, shoeH: 2}
	case csmFrodo:
		return csmLook{top: 17, sh: 25, hip: 32, foot: 39, half: 4, waist: 4, depth: 2,
			legL: 9, legR: 13, legW: 2, armL: 6, armR: 15, armW: 2, armSeg: 3.4, legSeg: 2.9, armK: 0.85, legK: 0.45,
			sleeveFrac: 1, body: frVest, bodyL: frVestL, bodyD: frVestD, sleeve: frShirt, fore: frShirt,
			hand: frSkin, handD: frSkinD, handW: 2, handH: 2, pants: frPants, pantsL: frPantL,
			shoe: frFeet, shoeL: frFeet, shoeD: frFeetD, shoeW: 4, shoeLen: 6, shoeH: 3}
	case csmKnuckles:
		return csmLook{top: 9, sh: 21, hip: 30, foot: 39, half: 4, waist: 3, depth: 2,
			legL: 9, legR: 13, legW: 2, armL: 6, armR: 15, armW: 2, armSeg: 3.8, legSeg: 3.9, armK: 1, legK: 0.55,
			sleeveFrac: 1, body: knRed, bodyL: knRedL, bodyD: knRedD, sleeve: knRed, fore: knRed,
			hand: plWhite, handD: plWhiteD, handW: 3, handH: 3, pants: knRed, pantsL: knRedL,
			shoe: knRed, shoeL: knRedL, shoeD: hx(0x881010), shoeW: 4, shoeLen: 6, shoeH: 4}
	}
	return csmLook{}
}

func genCsm(s *CharSpec, vw int, p Pose) *pixBuf {
	k := csmKind(s)
	lk := csmLookOf(k)
	b := newPix(charW, charH)
	if trackParts {
		b.pid = make([]uint8, charW*charH)
	}
	bob := clampI(tiaRnd(p.Bob), -1, 3)
	if lk.top+bob < 1 {
		bob = 1 - lk.top // the hat never touches the top row
	}
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
	hipY := lk.hip + bob
	ty0 := lk.sh + bob
	hy := lk.top + bob
	open := p.Open || p.Anim == AnimLaugh || p.Anim == AnimCheer || p.Anim == AnimPanic
	L, R := 11+hd, 12+hd
	spread := math.Abs(p.LF.Z-p.RF.Z) + math.Abs(p.LF.X-p.RF.X)*0.5
	flap := clampI(tiaRnd(spread*0.3), 0, 3)
	if p.Anim == AnimRun {
		flap = clampI(flap+1, 0, 4)
	}
	wave := 0
	if flap > 0 {
		wave = p.F % 2
	}

	leg := func(hx0 float64, l Limb, sign float64, id uint8, dark bool) {
		b.cur = id
		mx, my, ex, ey, top := tiaLegIK(vw, p, hx0, float64(hipY+1), l, sign, lk.legK, lk.legSeg, lk.foot)
		w := lk.legW
		if vw == vSide && w > 2 {
			w--
		}
		thickLine(b, hx0, float64(hipY+1), mx, my, w, tiaDk(lk.pants, dark))
		thickLine(b, mx, my, ex, ey, w, tiaDk(lk.pants, dark))
		thickLine(b, hx0, float64(hipY+2), mx, my, 1, tiaDk(lk.pantsL, dark))
		if k == csmChewie { // shaggy fur on the shins
			b.set(int(math.Floor(mx))-1, int(math.Floor(my))+1, tiaDk(cwFurD, dark))
		}
		ix := int(math.Floor(ex))
		csmShoe(b, k, lk, vw, ix, top, sign, w, dark)
	}

	arm := func(sx float64, l Limb, sign float64, id uint8, dark, held bool) {
		b.cur = id
		sy := float64(ty0 + 1)
		mx, my, ex, ey := tiaArmIK(vw, sx, sy, l, sign, lk.armK, lk.armSeg)
		w := lk.armW
		if vw == vSide && w > 2 {
			w--
		}
		// thick arms and big fists stay inside the box
		mx, ex = clampF(mx, 1, float64(23-w)), clampF(ex, 1, float64(23-maxI(w, lk.handW)))
		my, ey = clampF(my, 1, 40), clampF(ey, 4, float64(41-lk.handH)) // room for a pompom above
		if vw == vSide {
			ex = math.Min(ex, 16) // a knife held forward stays inside the box
		}
		f := lk.sleeveFrac
		cx, cy := sx+(mx-sx)*f, sy+(my-sy)*f
		if f < 1 {
			thickLine(b, sx, sy, mx, my, w, tiaDk(lk.fore, dark))
		}
		thickLine(b, sx, sy, cx, cy, w, tiaDk(lk.sleeve, dark))
		thickLine(b, mx, my, ex, ey, w, tiaDk(lk.fore, dark))
		switch k {
		case csmChewie: // a lighter fringe of fur on the arm
			thickLine(b, sx, sy+1, mx, my, 1, tiaDk(hx(0xc8884a), dark))
		case csmGandalf: // the wide sleeve opens at the wrist with a light trim
			b.set(int(math.Floor(ex))+boolI(sign > 0)-1, int(math.Floor(ey)), tiaDk(gdTrim, dark))
			b.set(int(math.Floor(ex))+boolI(sign > 0)+1, int(math.Floor(ey)), tiaDk(gdTrim, dark))
		default:
			thickLine(b, sx, sy, cx, cy, 1, tiaDk(shade(lk.sleeve, 1.15), dark))
		}
		ix, iy := int(math.Floor(ex)), int(math.Floor(ey))
		hx0 := ix
		if lk.handW > w && sign < 0 {
			hx0 = maxI(ix-(lk.handW-w), 1)
		}
		if vw == vSide {
			hx0 = ix
		}
		gx, gy := hx0, iy+1
		if k == csmGandalf && !held { // the staff in the other hand, under the fingers
			csmStaff(b, vw, gx+lk.handW/2, gy)
		}
		b.rect(gx, gy, lk.handW, lk.handH, tiaDk(lk.hand, dark))
		b.rect(gx, gy+lk.handH-1, lk.handW, 1, tiaDk(lk.handD, dark))
		if lk.hand == plWhite { // glove cuff
			b.rect(gx, gy, lk.handW, 1, tiaDk(plWhiteD, dark))
			if k == csmKnuckles { // the two spikes on the knuckles
				if vw == vSide {
					b.set(gx+lk.handW, gy+1, tiaDk(plWhite, dark))
					b.set(gx+lk.handW, gy+2, tiaDk(plWhiteD, dark))
				} else {
					b.set(gx, gy+lk.handH, tiaDk(plWhite, dark))
					b.set(gx+lk.handW-1, gy+lk.handH, tiaDk(plWhite, dark))
				}
			}
		}
		if !held {
			return
		}
		x, y := gx+lk.handW/2, gy+lk.handH/2
		if vw == vSide {
			x = minI(x, 16) // the stock rule for a knife held forward
		}
		kind := p.Hold
		if kind == HoldNone {
			kind = s.Carry
		}
		if s.Weapon != 0 && (kind == HoldKnife || kind == HoldNone) {
			kind = s.Weapon
		}
		if k == csmSteve && (kind == HoldNone || kind == HoldKnife) {
			b.cur = id
			csmSword(b, vw, x, y)
			return
		}
		tiaHeld(b, s, p, vw, x, y, id)
	}

	cape := func(front bool) {
		if k != csmFrodo {
			return
		}
		b.cur = 2
		hem := hipY + 6
		if p.Sit {
			hem = hipY + 2
		}
		switch vw {
		case vSide:
			if front {
				return
			}
			c := 11 + ox
			for y := ty0; y <= hem; y++ {
				t := float64(y-ty0) / float64(hem-ty0)
				back := c - lk.depth - 1 - int(float64(flap)*t*1.4+0.5)
				if flap > 0 && y > ty0+4 && (y+wave)%3 == 0 {
					back--
				}
				back = maxI(back, 1)
				for x := back; x <= c; x++ {
					col := frCape
					if x == back {
						col = frCapeD
					}
					b.set(x, y, col)
				}
			}
			b.rect(c-1, ty0-1, 3, 1, frCapeD) // the hood bunched at the neck
		case vBack:
			if front {
				return
			}
			for y := ty0 - 1; y <= hem; y++ {
				t := float64(y-ty0) / float64(hem-ty0)
				hw := lk.half + int(t*1.5+0.5)
				x0, x1 := 12-hw+ox, 11+hw+ox
				if flap > 0 && y > ty0+5 && (y+wave)%2 == 0 {
					x0--
				}
				for x := x0; x <= x1; x++ {
					col := frCape
					switch {
					case x == x0:
						col = frCapeL
					case x == x1:
						col = frCapeD
					case y == hem:
						col = frCapeD
					}
					b.set(x, y, col)
				}
			}
			b.rect(10+ox, ty0-1, 4, 2, frCapeD) // the hood
			b.rect(10+ox, ty0+1, 4, 1, frCapeL)
			b.rect(11+ox, ty0+5, 2, 4, frCapeD) // a fold
		default:
			if !front {
				// the cloak behind: two strips beside the body and its hem between the legs
				for y := ty0 + 1; y <= hem; y++ {
					x0 := 12 - lk.half - 1 + ox
					x1 := 11 + lk.half + 1 + ox
					if y > hipY {
						x0--
						x1++
					}
					b.set(x0, y, frCapeD)
					b.set(x1, y, frCapeD)
					if y > hipY+1 {
						b.set(x0+1, y, frCape)
						b.set(x1-1, y, frCape)
					}
				}
				return
			}
			// the collar and its golden brooch
			b.rect(L-3, ty0-1, 8, 1, frCape)
			b.set(L-3, ty0, frCapeD)
			b.set(R+3, ty0, frCapeD)
			b.rect(L, ty0-1, 2, 2, frGold)
			b.set(R, ty0, hx(0xb08a20))
		}
	}

	skirt := func() { // Gandalf's robe below the belt, over the legs
		if k != csmGandalf {
			return
		}
		b.cur = 2
		hem := lk.foot - 1 + minI(bob, 0)
		if p.Sit {
			hem = hipY + 6
		}
		rows := hem - hipY
		if rows <= 0 {
			return
		}
		footFx := func(l Limb, sign float64) float64 { return sign * clampF((l.X-1.5)*0.8, -2, 2) }
		switch vw {
		case vSide:
			c := 11 + ox
			for r := 1; r <= rows; r++ {
				t := float64(r) / float64(rows)
				back := c - 3 - int(float64(flap+1)*t*1.2+0.5)
				front := c + 2 + int(t*1.5+0.5)
				if flap > 0 && r > rows/2 && (r+wave)%3 == 0 {
					back--
				}
				back = maxI(back, 1)
				front = minI(front, 21)
				for x := back; x <= front; x++ {
					col := gdRobe
					switch {
					case r == rows:
						col = gdTrim
					case x == back:
						col = gdRobeD
					case x == front:
						col = gdTrim
					}
					b.set(x, hipY+r, col)
				}
			}
		default:
			ls, rs := footFx(p.LF, -1), footFx(p.RF, 1)
			if vw == vBack {
				ls, rs = footFx(p.RF, -1), footFx(p.LF, 1)
			}
			for r := 1; r <= rows; r++ {
				t := float64(r) / float64(rows)
				hw := lk.waist + int(t*3+0.5)
				dl := int(math.Floor(ls*t + 0.5))
				dr := int(math.Floor(rs*t + 0.5))
				if flap > 0 && r > rows/2 && (r+wave)%3 == 0 {
					dl--
					dr++
				}
				x0, x1 := maxI(12-hw+dl+ox, 1), minI(11+hw+dr+ox, 22)
				for x := x0; x <= x1; x++ {
					col := gdRobe
					switch {
					case r == rows:
						col = gdTrim
					case x == x0:
						col = gdRobeL
					case x == x1:
						col = gdRobeD
					}
					b.set(x, hipY+r, col)
				}
				if vw == vFront { // the light trim down the opening
					b.set(L+(dl+dr)/2, hipY+r, gdTrim)
				}
			}
		}
	}

	torso := func() {
		b.cur = 2
		rows := hipY - ty0
		if vw == vSide {
			c := 11 + ox
			for r := 0; r <= rows; r++ {
				x0, x1 := c-lk.depth, c+lk.depth
				if r == 0 {
					x0++
					x1--
				}
				if lk.belly && r > rows/3 && r < rows-1 {
					x1++ // the belly
				}
				for x := x0; x <= x1; x++ {
					col := lk.body
					switch {
					case x == x0:
						col = lk.bodyD
					case x == x1 || r == 0:
						col = lk.bodyL
					}
					b.set(x, ty0+r, col)
				}
			}
		} else {
			for r := 0; r <= rows; r++ {
				hw := lk.half + int(math.Floor(float64((lk.waist-lk.half)*r)/float64(maxI(rows, 1))+0.5))
				if lk.belly && r > rows/3 && r < rows-1 {
					hw++
				}
				x0, x1 := 12-hw+ox, 11+hw+ox
				if r == 0 && k != csmSteve {
					x0++
					x1--
				}
				for x := x0; x <= x1; x++ {
					col := lk.body
					switch {
					case r == 0 || x == x0:
						col = lk.bodyL
					case x == x1:
						col = lk.bodyD
					}
					b.set(x, ty0+r, col)
				}
			}
		}
		csmTorso(b, k, lk, vw, ox, ty0, hipY, L, R)
	}

	head := func() {
		b.cur = 1
		switch k {
		case csmChewie:
			csmChewHead(b, vw, hy, L, R, hd, ty0, open)
		case csmSteve:
			csmSteveHead(b, vw, hy, L, R, ty0, open)
		case csmMario, csmLuigi, csmWario, csmWaluigi:
			csmPlumberHead(b, k, vw, hy, L, R, ty0, open)
		case csmGandalf:
			csmGandalfHead(b, vw, hy, L, R, hipY, open)
		case csmFrodo:
			csmFrodoHead(b, vw, hy, L, R, hd, ty0, open)
		case csmKnuckles:
			csmKnuxHead(b, vw, hy, L, R, hd, ty0, open)
		}
	}

	switch vw {
	case vSide:
		cx := float64(11 + ox)
		backL, frontL := p.LF, p.RF
		if p.LF.Z > p.RF.Z {
			backL, frontL = p.RF, p.LF
		}
		cape(false)
		arm(cx-0.5, p.LA, 1, 3, true, false)
		leg(10, backL, 1, 5, true)
		leg(11, frontL, 1, 6, false)
		skirt()
		torso()
		head()
		arm(cx, p.RA, 1, 4, false, true)
	case vBack:
		leg(float64(lk.legR), p.LF, 1, 5, false)
		leg(float64(lk.legL), p.RF, -1, 6, false)
		skirt()
		torso()
		head()
		arm(float64(lk.armR+ox), p.LA, 1, 3, false, false)
		arm(float64(lk.armL+ox), p.RA, -1, 4, false, true)
		cape(false)
	default:
		cape(false)
		leg(float64(lk.legL), p.LF, -1, 5, false)
		leg(float64(lk.legR), p.RF, 1, 6, false)
		skirt()
		torso()
		cape(true)
		head()
		arm(float64(lk.armL+ox), p.LA, -1, 3, false, false)
		arm(float64(lk.armR+ox), p.RA, 1, 4, false, true)
	}
	return b
}

// csmShoe: the shoe (or bare foot) whose top row is top under a leg whose left pixel is ix.
func csmShoe(b *pixBuf, k int, lk csmLook, vw, ix, top int, sign float64, legW int, dark bool) {
	d := func(c color.RGBA) color.RGBA { return tiaDk(c, dark) }
	if vw == vSide {
		b.rect(ix, top, legW+1, 1, d(lk.shoeL))
		for r := 1; r < lk.shoeH; r++ {
			col := lk.shoe
			if r == lk.shoeH-1 {
				col = lk.shoeD
			}
			b.rect(ix-1, top+r, lk.shoeLen, 1, d(col))
		}
		switch k {
		case csmKnuckles:
			b.rect(ix, top, legW+1, 1, d(knGreen)) // green cuff, white strap, yellow buckle
			b.rect(ix-1, top+2, lk.shoeLen-1, 1, d(plWhite))
			b.set(ix+1, top+2, d(knYell))
		case csmWaluigi:
			b.set(ix+lk.shoeLen-1, top, d(lk.shoeL)) // the pointed curled toe
		case csmFrodo:
			b.set(ix+1, top, d(frFeetH)) // hairy instep
			b.set(ix+2, top+1, d(frFeetH))
			b.set(ix+lk.shoeLen-2, top+2, d(frFeetD)) // toes
		case csmSteve:
			b.rect(ix-1, top, lk.shoeLen, lk.shoeH, d(lk.shoe))
			b.rect(ix-1, top+lk.shoeH-1, lk.shoeLen, 1, d(lk.shoeD))
		}
		return
	}
	x0 := ix
	if sign < 0 {
		x0 = ix + legW - lk.shoeW
	}
	for r := 0; r < lk.shoeH; r++ {
		col := lk.shoe
		switch {
		case r == 0:
			col = lk.shoeL
		case r == lk.shoeH-1:
			col = lk.shoeD
		}
		b.rect(x0, top+r, lk.shoeW, 1, d(col))
	}
	switch k {
	case csmKnuckles:
		b.rect(x0, top, lk.shoeW, 1, d(knGreen))
		b.rect(x0, top+2, lk.shoeW, 1, d(plWhite))
		b.set(x0+lk.shoeW/2, top+2, d(knYell))
	case csmFrodo:
		b.rect(x0+1, top, 2, 1, d(frFeetH)) // the hairy tops of a hobbit's feet
		if vw == vFront {
			for x := x0; x < x0+lk.shoeW; x += 2 {
				b.set(x, top+lk.shoeH-1, d(frFeetD)) // toes
			}
		}
	case csmChewie:
		b.set(x0, top, d(cwFur)) // fur over the feet
		b.set(x0+lk.shoeW-1, top, d(cwFur))
	}
}

// csmTorso: shirts, overalls, the bandolier, the belt, the chest crescent.
func csmTorso(b *pixBuf, k int, lk csmLook, vw, ox, ty0, hipY, L, R int) {
	rows := hipY - ty0
	front, side := vw == vFront, vw == vSide
	c := 11 + ox
	switch k {
	case csmChewie:
		// fur texture
		for r := 1; r < rows; r += 3 {
			for x := 12 - lk.half + 1 + ox + (r/3)%2; x <= 11+lk.half-1+ox; x += 3 {
				b.set(x, ty0+r, cwBodyL)
			}
		}
		// the bandolier: grey strap with cartridges, shoulder to the opposite hip
		var x0, y0, x1, y1 float64
		switch vw {
		case vFront:
			x0, y0, x1, y1 = float64(11+lk.half-1+ox), float64(ty0), float64(12-lk.half+1+ox), float64(hipY-1)
		case vBack:
			x0, y0, x1, y1 = float64(12-lk.half+1+ox), float64(ty0), float64(11+lk.half-1+ox), float64(hipY-1)
		default:
			x0, y0, x1, y1 = float64(c+lk.depth), float64(ty0+1), float64(c-lk.depth+1), float64(hipY-2)
		}
		n := int(math.Max(math.Abs(x1-x0), math.Abs(y1-y0)))
		for i := 0; i <= n; i++ {
			t := float64(i) / float64(maxI(n, 1))
			x := int(math.Floor(x0 + (x1-x0)*t + 0.5))
			y := int(math.Floor(y0 + (y1-y0)*t + 0.5))
			b.set(x, y, cwStrap)
			b.set(x, y+1, cwStrap)
			if vw == vFront && i%2 == 0 {
				b.set(x, y, cwShell) // the cartridges
				b.set(x, y+1, cwShelD)
			} else if vw == vSide && i%2 == 0 {
				b.set(x, y, cwShell)
			}
		}
		if !side { // fur round the neck
			b.rect(L-2, ty0-1, 6, 1, cwFur)
		}
	case csmSteve:
		if front { // the skin of the neck in the collar
			b.rect(L, ty0, 2, 1, svSkin)
			b.set(L, ty0+1, svSkinD)
			b.set(R, ty0+1, svSkinD)
		}
		if !side {
			b.rect(12-lk.half+ox, hipY, lk.half*2, 1, svShirD) // hem
		}
	case csmMario, csmLuigi, csmWario, csmWaluigi:
		ov := csmLookOf(k).pants
		ovL := shade(ov, 1.25)
		ovD := shade(ov, 0.75)
		belt := 2
		switch vw {
		case vFront:
			bw := 2
			if k == csmWario {
				bw = 3
			}
			if k == csmWaluigi {
				bw = 1
			}
			top := ty0 + 3
			for y := top; y <= hipY; y++ {
				b.rect(L-bw, y, 2*bw+2, 1, ov)
			}
			b.rect(L-bw, top, 2*bw+2, 1, ovL)
			for y := ty0; y < top; y++ { // straps
				b.set(L-bw, y, ov)
				b.set(R+bw, y, ov)
			}
			b.set(L-bw, top, plButton)
			b.set(R+bw, top, plButton)
			for y := hipY - belt; y <= hipY; y++ { // the waist of the overalls
				hw := lk.waist
				if lk.belly && y < hipY-1 {
					hw++
				}
				b.rect(12-hw+ox, y, 2*hw, 1, ov)
				b.set(12-hw+ox, y, ovL)
				b.set(11+hw+ox, y, ovD)
			}
			if k != csmWaluigi {
				b.set(L, ty0+5, ovD) // the bib pocket
				b.set(R, ty0+5, ovD)
			}
		case vBack:
			for i := 0; i <= rows-3; i++ { // the straps cross at the back
				t := float64(i) / float64(maxI(rows-3, 1))
				xa := int(math.Floor(float64(L-2) + 4*t + 0.5))
				xb := int(math.Floor(float64(R+2) - 4*t + 0.5))
				b.set(xa, ty0+i, ov)
				b.set(xb, ty0+i, ov)
			}
			for y := hipY - belt; y <= hipY; y++ {
				hw := lk.waist
				if lk.belly && y < hipY-1 {
					hw++
				}
				b.rect(12-hw+ox, y, 2*hw, 1, ov)
				b.set(11+hw+ox, y, ovD)
			}
		default:
			b.rect(c, ty0, 1, 3, ov) // strap
			bx1 := c + lk.depth
			if lk.belly {
				bx1++
			}
			for y := ty0 + 3; y <= hipY; y++ {
				x1 := c + lk.depth
				if lk.belly && y-ty0 > rows/3 && y-ty0 < rows-1 {
					x1 = bx1
				}
				b.rect(c, y, x1-c+1, 1, ov)
			}
			for y := hipY - belt; y <= hipY; y++ {
				b.rect(c-lk.depth, y, 2*lk.depth+1, 1, ov)
				b.set(c-lk.depth, y, ovD)
			}
			b.set(c, ty0+3, plButton)
		}
	case csmGandalf:
		switch vw {
		case vFront:
			b.rect(L, ty0+1, 1, rows, gdTrim) // the trim down the front (under the beard)
			b.rect(12-lk.half+1+ox, ty0, lk.half*2-2, 1, gdTrim)
		case vBack:
			b.rect(L, ty0+2, 1, rows-3, gdRobeD) // back seam
			b.rect(L-2, ty0-1, 6, 2, gdRobeL)    // hood folds
			b.rect(L-1, ty0, 4, 1, gdRobeD)
		default:
			b.rect(c+lk.depth, ty0+1, 1, rows, gdTrim)
		}
		// the brown belt and its golden buckle
		if side {
			b.rect(c-lk.depth, hipY-1, 2*lk.depth+1, 1, gdBelt)
			b.set(c+lk.depth, hipY-1, gdGold)
		} else {
			b.rect(12-lk.waist+ox, hipY-1, lk.waist*2, 1, gdBelt)
			if front {
				b.rect(L, hipY-1, 2, 1, gdGold)
				b.set(L, hipY-2, hx(0xa07a18))
			}
		}
	case csmFrodo:
		switch vw {
		case vFront:
			b.rect(L-1, ty0, 4, rows+1, frShirt) // white shirt in the open waistcoat
			b.rect(L, ty0, 2, rows+1, frShirt)
			b.set(L-1, ty0+rows, frShirD)
			b.set(R+1, ty0+rows, frShirD)
			b.set(L-2, ty0+3, frGold) // waistcoat buttons
			b.set(L-2, ty0+5, frGold)
			b.rect(12-lk.half+ox, hipY, lk.half*2, 1, frPants)
		case vBack:
			b.rect(12-lk.half+ox, hipY, lk.half*2, 1, frPants)
		default:
			b.rect(c+lk.depth-1, ty0, 2, rows, frShirt)
			b.set(c+lk.depth, ty0+3, frGold)
			b.rect(c-lk.depth, hipY, 2*lk.depth+1, 1, frPants)
			// the cloak over the shoulder, its brooch at the throat
			b.rect(c-lk.depth, ty0-1, 2*lk.depth, 2, frCape)
			b.set(c+lk.depth, ty0-1, frGold)
		}
	case csmKnuckles:
		switch vw {
		case vFront: // the white crescent on the chest
			b.set(L-3, ty0, plWhite)
			b.set(R+3, ty0, plWhite)
			b.rect(L-3, ty0+1, 2, 1, plWhite)
			b.rect(R+2, ty0+1, 2, 1, plWhite)
			b.rect(L-2, ty0+2, 6, 1, plWhite)
			b.rect(L-1, ty0+3, 4, 1, plWhiteD)
		case vSide:
			b.rect(c+lk.depth, ty0, 1, 3, plWhite)
			b.set(c+lk.depth-1, ty0+2, plWhite)
			b.set(c+lk.depth-1, ty0+3, plWhiteD)
		}
	}
}

// csmSword: Steve's stone sword held up in the hand at (x, y).
func csmSword(b *pixBuf, vw, x, y int) {
	if vw == vSide {
		n := minI(5, y-3)
		b.set(x+1, y, svGrip)
		b.rect(x, y-1, 3, 1, svBladD) // the guard
		for i := 0; i < n; i++ {
			bx, by := x+2+i, y-2-i
			if bx > 21 {
				break
			}
			b.set(bx, by, svBlade)
			b.set(bx-1, by, svBladL)
			b.set(bx, by+1, svBladD)
		}
		return
	}
	n := minI(7, y-4)
	b.set(x, y+1, svGrip)
	b.rect(x-1, y-1, 3, 1, svBladD)
	for i := 0; i < n; i++ {
		b.set(x, y-2-i, svBlade)
		b.set(x+1, y-2-i, svBladD)
		if i%2 == 0 {
			b.set(x, y-2-i, svBladL)
		}
	}
	if n > 0 {
		b.set(x, y-2-n, svBladD) // the tip
	}
}

// csmStaff: Gandalf's long staff gripped at (x, y), with a white knob on top.
func csmStaff(b *pixBuf, vw, x, y int) {
	top := maxI(y-13, 3)
	bot := minI(y+15, 41)
	for yy := top; yy <= bot; yy++ {
		col := gdStaff
		if (yy+x)%4 == 0 {
			col = gdStafD // knots in the wood
		}
		b.set(x, yy, col)
	}
	b.set(x-1, top, gdStaff) // the twisted top holding the white stone
	b.set(x+1, top, gdStaff)
	b.set(x, top-1, hx(0xf8f8f8))
	b.set(x-1, top-1, hx(0xd8dce4))
	b.set(x+1, top-1, hx(0xd8dce4))
	b.set(x, top-2, hx(0xffffff))
	_ = vw
}

// csmChewHead: shaggy brown head with the dark crown, dark eyes and the pink mouth.
func csmChewHead(b *pixBuf, vw, hy, L, R, hd, ty0 int, open bool) {
	cx := 11.5 + float64(hd)
	fy := float64(hy)
	switch vw {
	case vSide:
		b.ellipse(cx-0.5, fy+5.5, 4.4, 5.6, cwFur)
		b.rect(L-5, hy+3, 2, ty0-hy-2, cwFur) // long fur down the back of the head
		b.set(L-6, hy+6, cwFurD)
		b.set(L-6, hy+9, cwFurD)
		b.ellipse(cx-1.2, fy+2.6, 4.0, 2.8, cwCrown)
		b.rect(R, hy+5, 4, 4, cwFace) // the muzzle
		b.rect(R+1, hy+6, 3, 2, cwMuz)
		b.set(R+4, hy+6, cwNose)
		b.set(R+4, hy+7, cwNose)
		b.set(R, hy+4, cwFurD) // brow
		b.set(R+1, hy+4, cwFurD)
		b.set(R+1, hy+5, cwEye)
		b.rect(R+2, hy+8, 2, 1, cwPink)
		if open {
			b.rect(R+1, hy+8, 3, 2, hx(0x6a1a1a))
			b.set(R+3, hy+8, plWhite)
			b.rect(R+1, hy+9, 3, 1, cwPink)
		}
		b.set(L-2, hy+7, cwFurD)
		b.set(L-1, hy+9, cwFurL)
		b.rect(L-1, hy+10, 4, ty0-hy-9, cwFur) // neck fur
	case vBack:
		b.ellipse(cx, fy+5.5, 4.7, 5.6, cwFur)
		b.ellipse(cx, fy+2.6, 4.4, 2.8, cwCrown)
		b.rect(L-4, hy+5, 10, ty0-hy-4, cwFur) // long fur over the shoulders
		for _, x := range []int{L - 3, L, R + 2} {
			b.rect(x, hy+6, 1, 4, cwFurD)
		}
		b.rect(L-2, hy+4, 1, 3, cwFurL)
		b.rect(R+1, hy+4, 1, 3, cwFurL)
	default:
		b.ellipse(cx, fy+5.5, 4.6, 5.6, cwFur)
		b.rect(L-5, hy+4, 1, 6, cwFurD) // shaggy sides
		b.rect(R+5, hy+4, 1, 6, cwFurD)
		b.rect(L-4, hy+8, 1, ty0-hy-7, cwFur)
		b.rect(R+4, hy+8, 1, ty0-hy-7, cwFur)
		b.ellipse(cx, fy+2.6, 4.4, 2.8, cwCrown)
		b.ellipse(cx, fy+7.0, 3.0, 3.6, cwFace)
		b.rect(L-3, hy+4, 3, 1, cwFurD) // heavy brows
		b.rect(R+1, hy+4, 3, 1, cwFurD)
		b.set(L-2, hy+5, cwEye)
		b.set(R+2, hy+5, cwEye)
		b.set(L-1, hy+5, cwFurD)
		b.set(R+1, hy+5, cwFurD)
		b.rect(L-1, hy+6, 4, 2, cwMuz)
		b.rect(L, hy+6, 2, 1, cwNose)
		b.rect(L, hy+8, 2, 1, cwPink)
		if open {
			b.rect(L-1, hy+8, 4, 2, hx(0x6a1a1a))
			b.set(L-1, hy+8, plWhite)
			b.set(R+1, hy+8, plWhite)
			b.rect(L, hy+9, 2, 1, cwPink)
		}
		b.rect(L-2, hy+10, 6, 1, cwFur) // chin fur
		b.set(L-3, hy+6, cwFurL)
		b.set(R+3, hy+6, cwFurL)
		b.rect(L-3, hy+11, 8, maxI(ty0-hy-11, 1), cwFur)
	}
}

// csmSteveHead: the square head of the block man.
func csmSteveHead(b *pixBuf, vw, hy, L, R, ty0 int, open bool) {
	x0 := L - 3
	b.rect(x0, hy, 8, 8, svSkin)
	b.rect(L, hy+8, 2, maxI(ty0-hy-8, 0), svSkinD) // neck
	switch vw {
	case vSide:
		b.rect(x0, hy, 8, 2, svHair)
		b.rect(x0, hy, 4, 7, svHair) // hair at the back
		b.rect(x0+4, hy+2, 1, 1, svHair)
		b.set(x0+1, hy, svHairL)
		b.set(x0+3, hy+4, svSkinD) // ear
		b.set(x0+3, hy+5, svSkinD)
		b.set(x0+5, hy+4, svEyeW)
		b.set(x0+6, hy+4, svEyeV)
		b.set(x0+7, hy+5, svNose)
		b.rect(x0+5, hy+6, 3, 1, svBeard)
		b.set(x0+6, hy+7, svBeard)
		if open {
			b.set(x0+7, hy+6, hx(0x3a1a12))
		}
		b.rect(x0, hy+7, 8, 1, svSkinD)
		b.rect(x0, hy+6, 3, 2, svHair)
	case vBack:
		b.rect(x0, hy, 8, 8, svHair)
		b.rect(x0+1, hy+1, 2, 1, svHairL)
		b.set(x0+5, hy+3, svHairL)
		b.rect(x0, hy+7, 8, 1, shade(svHair, 0.85))
	default:
		b.rect(x0, hy, 8, 2, svHair) // hair: top rows and a fringe at the sides
		b.set(x0, hy+2, svHair)
		b.set(x0+7, hy+2, svHair)
		b.set(x0+1, hy+2, svHairL)
		b.set(x0+6, hy+2, svHairL)
		b.set(x0+3, hy, svHairL)
		b.rect(x0+1, hy+4, 1, 1, svEyeW) // white and violet eyes
		b.rect(x0+2, hy+4, 1, 1, svEyeV)
		b.rect(x0+5, hy+4, 1, 1, svEyeV)
		b.rect(x0+6, hy+4, 1, 1, svEyeW)
		b.rect(x0+3, hy+5, 2, 1, svNose) // brown nose
		b.rect(x0+2, hy+6, 4, 1, svBeard)
		b.rect(x0+2, hy+7, 4, 1, svBeard)
		b.set(x0+2, hy+7, svSkinD)
		b.set(x0+5, hy+7, svSkinD)
		if open {
			b.rect(x0+3, hy+6, 2, 1, hx(0x3a1a12))
		}
		b.rect(x0+7, hy+3, 1, 5, svSkinD) // shaded side
	}
}

// csmLetters: the cap emblem letters, 4x3, '#' = letter pixel.
var csmLetters = map[int][3]string{
	csmMario:   {".##.", "####", "#..#"},
	csmLuigi:   {".#..", ".#..", ".###"},
	csmWario:   {"#..#", "####", ".##."},
	csmWaluigi: {".###", ".#..", ".#.."},
}

// csmPlumberHead: cap with the letter, moustache and nose of the four plumbers.
func csmPlumberHead(b *pixBuf, k, vw, hy, L, R, ty0 int, open bool) {
	lk := csmLookOf(k)
	cap, capL, capD := lk.body, lk.bodyL, lk.bodyD
	emb, letter := plWhite, cap
	switch k {
	case csmWario:
		letter = hx(0x2a48c8)
	case csmWaluigi:
		emb = hx(0xf8d838)
	}
	fw := 4 // face half width
	switch k {
	case csmLuigi, csmWaluigi:
		fw = 3
	case csmWario:
		fw = 5
	}
	chinRows := 7
	if k == csmWario {
		chinRows = 8
	}
	if k == csmWaluigi {
		chinRows = 8
	}
	skin, skinD := plSkin, plSkinD
	nose, noseD := plNose, plNoseD
	if k == csmWario || k == csmWaluigi {
		nose, noseD = hx(0xf09a9a), hx(0xc86a6a)
	}
	// the neck down to the collar
	b.rect(L-1, hy+5+chinRows, 4, maxI(ty0-hy-5-chinRows+1, 0), skinD)
	switch vw {
	case vSide:
		// face
		b.rect(L-fw+2, hy+5, fw*2-1, chinRows-1, skin)
		b.rect(L-fw+3, hy+5+chinRows-1, fw*2-3, 1, skin)
		if k == csmWario {
			b.rect(L, hy+5+chinRows-1, fw+2, 2, skin) // the big chin
		}
		b.rect(L-fw, hy+4, 3, 5, plHair) // hair at the back
		b.set(L-fw+1, hy+9, plHair)
		b.rect(L-1, hy+6, 2, 2, skinD) // ear
		if k == csmWaluigi {
			b.set(L, hy+5, skinD) // pointy
			b.set(L+1, hy+4, skinD)
		}
		// cap: dome and the bill sticking forward
		b.rect(L-fw+1, hy, fw*2, 1, cap)
		b.rect(L-fw, hy+1, fw*2+2, 3, cap)
		b.rect(L-fw+2, hy, 3, 1, capL)
		b.rect(R+fw-2, hy+1, 2, 2, emb)
		b.set(R+fw-2, hy+2, letter)
		b.rect(L-fw, hy+4, fw*2+5, 1, capD)
		// eye
		b.set(R+fw-2, hy+5, plWhite)
		b.set(R+fw-2, hy+6, plWhite)
		b.set(R+fw-1, hy+5, plPupil)
		b.set(R+fw-1, hy+6, plPupil)
		// nose
		switch k {
		case csmWaluigi:
			b.rect(R+fw, hy+6, 2, 2, nose)
			b.rect(R+fw+2, hy+7, 1, 2, noseD) // long and pointed
		case csmWario:
			b.rect(R+fw-1, hy+6, 3, 3, nose)
			b.set(R+fw+1, hy+8, noseD)
		default:
			b.rect(R+fw-1, hy+6, 3, 2, nose)
			b.set(R+fw+1, hy+7, noseD)
		}
		// moustache and mouth
		my := hy + 8
		if k == csmWario {
			my = hy + 9
		}
		switch k {
		case csmWaluigi:
			b.rect(R+fw-3, my+1, 3, 1, plMous)
			b.set(R+fw-4, my, plMous)
		case csmWario:
			b.rect(R+fw-3, my, 4, 1, plMous)
			b.set(R+fw-2, my+1, plMous)
			b.set(R+fw, my-1, plMous)
			b.rect(R+fw-2, my+2, 3, 1, plWhite) // the grin
		default:
			b.rect(R+fw-3, my, 4, 1, plMous)
			b.rect(R+fw-2, my+1, 2, 1, plMous)
		}
		if open {
			b.set(R+fw-1, my+2, plMouth)
		}
	case vBack:
		b.ellipse(11.5+float64(L-11), float64(hy)+8.0, float64(fw)+0.6, 3.8, plHair)
		b.rect(L-fw-1, hy+6, 1, 2, skin) // ears
		b.rect(R+fw+1, hy+6, 1, 2, skin)
		if k == csmWaluigi {
			b.set(L-fw-2, hy+5, skin)
			b.set(R+fw+2, hy+5, skin)
		}
		b.rect(L-fw+1, hy, fw*2, 1, cap)
		b.rect(L-fw, hy+1, fw*2+2, 4, cap)
		b.rect(L-fw+1, hy, fw, 1, capL)
		b.rect(L-1, hy+4, 4, 1, capD) // the adjuster at the back
		b.set(L, hy+4, skin)
		b.rect(L-1, hy+5+chinRows-1, 4, 1, skinD)
	default:
		// face
		b.rect(L-fw+1, hy+5, fw*2, chinRows-1, skin)
		b.rect(L-fw+2, hy+5+chinRows-1, fw*2-2, 1, skin)
		if k == csmWario { // the big square chin
			b.rect(L-fw+1, hy+5+chinRows-1, fw*2, 2, skin)
			b.rect(L-fw+2, hy+5+chinRows+1, fw*2-2, 1, skinD)
		}
		if k == csmWaluigi { // the long narrow chin
			b.rect(L-1, hy+5+chinRows, 4, 1, skin)
		}
		b.rect(R+fw-1, hy+5, 1, chinRows-1, skinD)
		b.rect(L-fw, hy+5, 1, 3, plHair) // sideburns
		b.rect(R+fw+1, hy+5, 1, 3, plHair)
		b.rect(L-fw, hy+6, 1, 2, skin) // ears
		b.rect(R+fw+1, hy+6, 1, 2, skinD)
		if k == csmWaluigi {
			b.set(L-fw-1, hy+5, skin)
			b.set(R+fw+2, hy+5, skinD)
		}
		// cap: dome, the emblem with the letter, a wide brim
		b.rect(L-fw+1, hy, fw*2, 1, cap)
		b.rect(L-fw, hy+1, fw*2+2, 3, cap)
		b.rect(L-fw+1, hy, 2, 1, capL)
		b.rect(L-1, hy, 4, 4, emb)
		b.set(L-1, hy, cap)
		b.set(R+1, hy, cap)
		lt := csmLetters[k]
		for r := 0; r < 3; r++ {
			for cc := 0; cc < 4; cc++ {
				if lt[r][cc] == '#' {
					b.set(L-1+cc, hy+1+r, letter)
				}
			}
		}
		b.rect(L-fw-1, hy+4, fw*2+4, 1, capD)
		// eyes
		ey := hy + 5
		b.set(L-2, ey, plWhite)
		b.set(L-2, ey+1, plWhite)
		b.set(L-1, ey, plPupil)
		b.set(L-1, ey+1, plPupil)
		b.set(R+1, ey, plPupil)
		b.set(R+1, ey+1, plPupil)
		b.set(R+2, ey, plWhite)
		b.set(R+2, ey+1, plWhite)
		if k == csmWario || k == csmWaluigi { // mean brows
			b.set(L-2, ey, plMous)
			b.set(R+2, ey, plMous)
		}
		// nose
		switch k {
		case csmWaluigi:
			b.rect(L, ey+1, 2, 3, nose)
			b.rect(L, ey+4, 2, 1, noseD) // long and pointed
		case csmWario:
			b.rect(L-1, ey+1, 4, 3, nose)
			b.rect(L-1, ey+3, 4, 1, noseD)
		case csmLuigi:
			b.rect(L, ey+2, 2, 2, nose)
			b.set(L-1, ey+3, nose)
			b.set(R+1, ey+3, noseD)
		default:
			b.rect(L-1, ey+2, 4, 2, nose)
			b.rect(L-1, ey+3, 4, 1, noseD)
			b.rect(L, ey+1, 2, 1, nose)
		}
		// moustache and mouth
		my := ey + 4
		switch k {
		case csmWaluigi: // thin with curled ends
			b.rect(L-2, my+1, 2, 1, plMous)
			b.rect(R+1, my+1, 2, 1, plMous)
			b.set(L-3, my, plMous)
			b.set(R+3, my, plMous)
			if open {
				b.rect(L, my+2, 2, 1, plMouth)
			} else {
				b.rect(L, my+2, 2, 1, skinD)
			}
		case csmWario: // zigzag, and the toothy grin
			for x := L - 4; x <= R+4; x++ {
				b.set(x, my+((x+12)%2), plMous)
			}
			b.rect(L-2, my+2, 6, 1, plWhite)
			b.set(L-3, my+2, plMouth)
			b.set(R+3, my+2, plMouth)
			b.rect(L-2, my+3, 6, 1, plMouth)
			if open {
				b.rect(L-1, my+3, 4, 1, hx(0xd84a4a))
			}
		case csmLuigi:
			b.rect(L-3, my, 8, 1, plMous)
			b.rect(L-2, my+1, 2, 1, plMous)
			b.rect(R+1, my+1, 2, 1, plMous)
			if open {
				b.rect(L, my+1, 2, 1, plMouth)
			}
		default:
			b.rect(L-3, my, 8, 1, plMous)
			b.rect(L-2, my+1, 6, 1, plMous)
			if open {
				b.rect(L, my+2, 2, 1, plMouth)
			}
		}
	}
}

// csmGandalfHead: tall pointed hat with a wide brim, bushy brows, the long beard to the waist.
func csmGandalfHead(b *pixBuf, vw, hy, L, R, hipY int, open bool) {
	brim := hy + 7
	beardEnd := hipY - 2
	switch vw {
	case vSide:
		b.rect(L-1, brim+1, 4, 5, gdSkin) // face
		b.rect(L-3, brim, 3, 8, gdHair)   // hair at the back
		b.set(L-3, brim+8, gdHair)
		b.set(R+1, brim+2, hx(0x2a2a30)) // eye under the brow
		b.rect(R, brim+1, 3, 1, gdBeard)
		b.rect(R+2, brim+2, 1, 2, gdSkin) // nose
		b.set(R+3, brim+3, gdSkinD)
		// the beard falls from the chin down the chest
		for y := brim + 4; y <= beardEnd; y++ {
			t := float64(y-brim-4) / float64(maxI(beardEnd-brim-4, 1))
			x0 := L
			x1 := R + 2 - int(t*2+0.5)
			for x := x0; x <= x1; x++ {
				col := gdBeard
				if x == x0 || (y+x)%3 == 0 {
					col = gdBearD
				}
				b.set(x, y, col)
			}
		}
		if open {
			b.set(R+1, brim+5, hx(0x4a2020))
		}
		// hat: cone leaning back, wide brim
		for i := 0; i < 7; i++ {
			hw := float64(i)*0.55 + 0.5
			cx := 11.5 + float64(L-11) - float64(6-i)*0.45
			x0 := int(math.Floor(cx - hw))
			x1 := int(math.Floor(cx + hw - 0.01))
			for x := x0; x <= x1; x++ {
				col := gdHat
				switch {
				case x == x1:
					col = gdHatL
				case x == x0:
					col = gdHatD
				}
				b.set(x, hy+i, col)
			}
		}
		b.rect(L-3, brim-1, 7, 1, gdHatD) // band
		b.rect(L-5, brim, 11, 1, gdHat)
		b.set(L-5, brim, gdHatD)
		b.set(R+5, brim, gdHatL)
	case vBack:
		b.rect(L-3, brim, 8, 9, gdHair) // long hair down the back
		b.rect(L-2, brim+9, 6, 2, gdHair)
		for _, x := range []int{L - 2, L + 1, R + 2} {
			b.rect(x, brim+2, 1, 7, gdBearD)
		}
		csmGandalfHat(b, hy, L, R)
	default:
		b.rect(L-2, brim+1, 6, 5, gdSkin) // face
		b.rect(R+2, brim+1, 1, 4, gdSkinD)
		b.rect(L-4, brim+1, 2, 8, gdHair) // long hair at the sides
		b.rect(R+3, brim+1, 2, 8, gdHair)
		b.set(L-4, brim+9, gdBearD)
		b.set(R+4, brim+9, gdBearD)
		b.rect(L-2, brim+1, 2, 1, gdBeard) // bushy brows
		b.rect(R, brim+1, 2, 1, gdBeard)
		b.set(L-1, brim+2, hx(0x2a2a30)) // eyes
		b.set(R+1, brim+2, hx(0x2a2a30))
		b.rect(L, brim+3, 2, 1, gdSkinD)   // nose
		b.rect(L-2, brim+4, 6, 1, gdBeard) // moustache
		// the long beard down to the belt, narrower at the tip
		for y := brim + 5; y <= beardEnd; y++ {
			t := float64(y-brim-5) / float64(maxI(beardEnd-brim-5, 1))
			hw := 3 - int(t*2.2+0.3)
			if hw < 1 {
				hw = 1
			}
			for x := L - hw + 1; x <= R+hw-1; x++ {
				col := gdBeard
				if (x+y)%4 == 0 || x == L-hw+1 {
					col = gdBearD
				}
				b.set(x, y, col)
			}
		}
		b.set(L, beardEnd+1, gdBearD) // the tip
		if open {
			b.rect(L, brim+5, 2, 1, hx(0x4a2020))
		}
		csmGandalfHat(b, hy, L, R)
	}
}

func csmGandalfHat(b *pixBuf, hy, L, R int) {
	brim := hy + 7
	for i := 0; i < 7; i++ {
		hw := float64(i)*0.55 + 0.5
		cx := 11.5 + float64(L-11) + float64(6-i)*0.3 // the tip bends a little
		x0 := int(math.Floor(cx - hw))
		x1 := int(math.Floor(cx + hw - 0.01))
		for x := x0; x <= x1; x++ {
			col := gdHat
			switch {
			case x == x0:
				col = gdHatL
			case x == x1 && x1 > x0:
				col = gdHatD
			}
			b.set(x, hy+i, col)
		}
	}
	b.rect(L-3, brim-1, 8, 1, gdHatD) // band
	b.rect(L-6, brim, 14, 1, gdHat)   // the wide brim
	b.set(L-6, brim, gdHatL)
	b.set(R+6, brim, gdHatD)
	b.rect(L-5, brim+1, 12, 1, shade(gdHatD, 0.9))
}

// csmFrodoHead: wavy reddish brown hair, pale face, blue eyes.
func csmFrodoHead(b *pixBuf, vw, hy, L, R, hd, ty0 int, open bool) {
	cx := 11.5 + float64(hd)
	fy := float64(hy)
	b.rect(L, hy+8, 2, maxI(ty0-hy-8, 0), frSkinD) // neck
	switch vw {
	case vSide:
		b.ellipse(cx+0.3, fy+5.0, 3.4, 3.6, frSkin)
		b.ellipse(cx-1.0, fy+3.4, 3.6, 3.4, frHair)
		b.rect(L-4, hy+3, 2, 4, frHair)
		b.set(L-5, hy+4, frHairD)
		b.set(L-4, hy+7, frHairD)
		b.set(R+2, hy+2, frHair) // fringe
		b.set(R+2, hy+3, frHairD)
		b.set(L-2, hy+1, frHairL)
		b.set(L, hy+2, frHairL)
		b.set(L-3, hy+5, frHairL)
		b.set(L, hy+5, frSkinD) // ear, a little pointed
		b.set(L, hy+4, frSkinD)
		b.set(R+2, hy+4, frEye)
		b.set(R+4, hy+5, frSkinD) // nose
		if open {
			b.set(R+3, hy+7, hx(0x7a2a2a))
		} else {
			b.set(R+3, hy+7, hx(0xc87868))
		}
	case vBack:
		b.ellipse(cx, fy+4.4, 4.3, 4.2, frHair)
		for _, xy := range [][2]int{{L - 2, hy + 2}, {R + 1, hy + 3}, {L, hy + 5}, {L - 3, hy + 6}, {R + 2, hy + 6}} {
			b.set(xy[0], xy[1], frHairL) // curls
		}
		for _, xy := range [][2]int{{L - 1, hy + 3}, {R + 2, hy + 4}, {L + 1, hy + 7}, {L - 3, hy + 4}} {
			b.set(xy[0], xy[1], frHairD)
		}
	default:
		b.ellipse(cx, fy+5.2, 3.3, 3.4, frSkin)
		b.rect(R+2, hy+4, 1, 3, frSkinD)
		// wavy hair: a mop of curls over the brow and down to the ears
		b.ellipse(cx, fy+2.6, 4.4, 2.8, frHair)
		b.rect(L-4, hy+2, 1, 5, frHair)
		b.rect(R+4, hy+2, 1, 5, frHair)
		b.set(L-3, hy+4, frHair) // fringe locks
		b.set(L-1, hy+4, frHair)
		b.set(R+2, hy+4, frHair)
		b.set(L-4, hy+7, frHairD)
		b.set(R+4, hy+7, frHairD)
		for _, xy := range [][2]int{{L - 2, hy + 1}, {R + 1, hy + 1}, {L, hy + 2}, {L - 3, hy + 3}, {R + 3, hy + 3}} {
			b.set(xy[0], xy[1], frHairL)
		}
		b.set(L+1, hy+3, frHairD)
		b.set(L-2, hy+3, frHairD)
		b.set(L-1, hy+5, frEye) // big blue eyes
		b.set(R+1, hy+5, frEye)
		b.set(L-2, hy+5, plWhite)
		b.set(R+2, hy+5, plWhite)
		if open {
			b.rect(L, hy+7, 2, 1, hx(0x7a2a2a))
		} else {
			b.rect(L, hy+7, 2, 1, hx(0xc87868))
		}
	}
}

// csmKnuxHead: big red head, dreadlocks swept back, white eyes, cream muzzle.
func csmKnuxHead(b *pixBuf, vw, hy, L, R, hd, ty0 int, open bool) {
	cx := 11.5 + float64(hd)
	fy := float64(hy)
	b.rect(L-1, hy+10, 4, maxI(ty0-hy-9, 0), knRedD) // neck
	switch vw {
	case vSide:
		// the dreadlocks point back and down
		for i, d := range [][3]int{{hy + 2, 4, 3}, {hy + 5, 6, 3}, {hy + 8, 5, 3}, {hy + 11, 3, 2}} {
			y0, ln, th := d[0], d[1], d[2]
			for j := 0; j < ln; j++ {
				x := L - 3 - j
				t := th - (j*th)/ln
				if t < 1 {
					t = 1
				}
				for r := 0; r < t; r++ {
					col := knRed
					if r == t-1 {
						col = knRedD
					}
					b.set(x, y0+j/2+r, col)
				}
			}
			_ = i
		}
		b.ellipse(cx+0.2, fy+5.6, 4.4, 5.4, knRed)
		b.rect(L-1, hy+1, 4, 1, knRedL)
		b.rect(R, hy+3, 3, 4, plWhite) // eye
		b.set(R, hy+3, knRed)
		b.rect(R+2, hy+4, 1, 2, knPupil)
		b.rect(R, hy+7, 5, 3, knCream) // muzzle
		b.rect(R+1, hy+10, 3, 1, knCreaD)
		b.set(R+4, hy+7, plMous) // nose
		if open {
			b.rect(R+2, hy+9, 2, 1, plMouth)
		} else {
			b.rect(R+2, hy+9, 2, 1, knCreaD)
		}
	case vBack:
		b.ellipse(cx, fy+5.6, 4.6, 5.4, knRed)
		for _, x := range []int{L - 4, L - 1, R + 1, R + 3} { // dreadlocks down the back
			b.rect(x, hy+7, 2, 6, knRed)
			b.set(x, hy+13, knRedD)
			b.set(x+1, hy+12, knRedD)
		}
		b.rect(L-1, hy+2, 2, 1, knRedL)
		b.set(L-3, hy+4, knRedL)
	default:
		b.rect(L-5, hy+5, 2, 7, knRed) // dreadlocks hanging at the sides
		b.rect(R+4, hy+5, 2, 7, knRed)
		b.set(L-5, hy+12, knRedD)
		b.set(R+5, hy+12, knRedD)
		b.rect(L-3, hy+9, 1, 4, knRedD)
		b.rect(R+3, hy+9, 1, 4, knRedD)
		b.ellipse(cx, fy+5.6, 4.6, 5.4, knRed)
		b.rect(L-2, hy+1, 3, 1, knRedL)
		b.set(R+3, hy+3, knRedD)
		// the eyes, white with a dark pupil, set a little angry
		b.rect(L-3, hy+3, 3, 4, plWhite)
		b.rect(R+1, hy+3, 3, 4, plWhite)
		b.set(L-1, hy+3, knRed)
		b.set(R+1, hy+3, knRed)
		b.rect(L-1, hy+4, 1, 2, knPupil)
		b.rect(R+1, hy+4, 1, 2, knPupil)
		// cream muzzle and black nose
		b.rect(L-2, hy+7, 6, 3, knCream)
		b.rect(L-1, hy+10, 4, 1, knCreaD)
		b.rect(L, hy+7, 2, 1, plMous)
		if open {
			b.rect(L-1, hy+9, 4, 1, plMouth)
		} else {
			b.rect(L, hy+9, 2, 1, knCreaD)
		}
	}
}
