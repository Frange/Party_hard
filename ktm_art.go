package main

import (
	"image/color"
	"math"
	"math/rand"
)

// ---------------------------------------------------------------------------
// Own pixel-art layers for four characters, drawn very close to the user's pixel-art references
// (palette, proportions, features), adapted to our 24x44 sprite and its three views:
//   - kratos: pale ash skin, red stripe over the bald head, short thick black beard, red tattoo
//     from the shoulder down the chest and arm, a dark leather strap across the chest with a gold
//     disc, striped leather bracers, a belt with a gold buckle, a dark red leather skirt, dark
//     trousers, tall black boots and a big axe with a red handle and a blue-grey blade;
//   - solid_snake: the classic dark blue sneaking suit with grey patches, harness and pouches,
//     a blue bandana whose tails flutter, brown fringe, black gloves and boots;
//   - big_boss: olive green fatigues, black bandana, an eye patch and a full beard;
//   - el_exterminador (a party killer, terminator_killer.go): a chrome endoskeleton, a metal
//     skull with glowing red eyes, ribs and spine, piston arms and legs, a big rifle (HoldRifle).
// genChar calls ktmHead / ktmTorso / ktmArm / ktmLeg / ktmHeld next to the other own layers.
// ---------------------------------------------------------------------------

const (
	ktmNone = iota
	ktmKratos
	ktmSnake
	ktmBoss
	ktmEndo
)

var ktmKinds = map[string]int{"kratos": ktmKratos, "solid_snake": ktmSnake, "big_boss": ktmBoss, "el_exterminador": ktmEndo}

func ktmKind(s *CharSpec) int { return ktmKinds[s.Name] }

// presetBigBoss: the veteran of the spy saga, a new famous guest. A fixed random source of its own, so the
// shared random stream of builtinChars (export.go) is not moved for the characters after it.
func presetBigBoss(*rand.Rand) CharSpec {
	return def("big_boss", skinc(0xd8b090), hairc(0x54463a, HairShort), beardc(BeardGoatee), tops(TopLong, 0x5a6438, 0x46502a),
		pantsc(0x4e5832), shoesc(0x16161c), bodyc(BodyBig), stat(-1)).F(rand.New(rand.NewSource(1964)))
}

func init() {
	famousPresets = append(famousPresets, presetBigBoss)
	specialPool = append(specialPool, "big_boss")
	charDescs["big_boss"] = [2]string{"Leyenda con parche y barba de veterano", "Eyepatch legend with a veteran beard"}
	funnyDesc["skin:big_boss"] = "Un veterano con parche que dice que lo de la caja de cartón fue idea suya."
	for k, v := range map[string]string{
		"Leyenda con parche y barba de veterano":                                     "Eyepatch legend with a veteran beard",
		"Un veterano con parche que dice que lo de la caja de cartón fue idea suya.": "A veteran with an eyepatch who says the cardboard box was his idea.",
	} {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
}

var (
	// Kratos
	ktRed     = hx(0xc81e1e)
	ktRedD    = hx(0x8a1414)
	ktLeather = hx(0x4a2c1a)
	ktLeathL  = hx(0x7a5232)
	ktGold    = hx(0xe0b030)
	ktGoldL   = hx(0xfff0a0)
	ktSkirt   = hx(0x6a1a18)
	ktSkirtL  = hx(0x8e2a22)
	ktFlap    = hx(0x5a5660)
	ktBoot    = hx(0x18161c)
	ktBootL   = hx(0x3a3842)
	ktAxeH    = hx(0x9a2a1a)
	ktAxeHL   = hx(0xd04a2a)
	ktBlade   = hx(0x7a92ae)
	ktBladeL  = hx(0xd0e0f0)
	ktBladeD  = hx(0x46566e)
	// Snake / Big Boss
	snBand   = hx(0x2a56c8)
	snBandL  = hx(0x5a8af0)
	snPatch  = hx(0x6a7688)
	snPatchL = hx(0x8e9aac)
	snStrap  = hx(0x161e2c)
	snGlove  = hx(0x16161c)
	snGloveL = hx(0x34343e)
	bbBand   = hx(0x1a1a1e)
	bbPatchE = hx(0x101014)
	bbStrap  = hx(0x5a4228)
	bbPouch  = hx(0x46502a)
	// endoskeleton
	enChrome = hx(0xc4c8d2)
	enHi     = hx(0xf0f2f6)
	enMid    = hx(0x8e929e)
	enDark   = hx(0x4a4c56)
	enGap    = hx(0x1e1e24)
	enEye    = hx(0xff2a1a)
	enEyeHot = hx(0xffd0a0)
	enTeeth  = hx(0xe8e8e0)
)

func ktmDark(c color.RGBA, dark bool) color.RGBA {
	if dark {
		return shade(c, 0.78)
	}
	return c
}

// ktmHead paints the head details (after the stock face, beard and hats).
func ktmHead(b *pixBuf, s *CharSpec, vw int, p Pose, hxp, hyp, rx, ry float64, ey int) {
	k := ktmKind(s)
	if k == ktmNone {
		return
	}
	b.cur = 1
	ihx := int(math.Floor(hxp))
	top := int(math.Floor(hyp - ry))
	chin := int(math.Floor(hyp + ry))
	rxi := int(math.Floor(rx))
	sw := ciSwing(p)
	switch k {
	case ktmKratos:
		brow := shade(s.Skin, 0.62)
		switch vw {
		case vFront: // the red stripe over the crown and down to the left eye, a heavy brow
			b.rect(ihx-1, top, 1, ey-top-1, ktRed)
			b.set(ihx-1, top, ktRedD)
			b.set(ihx-2, ey-1, brow)
			b.set(ihx+1, ey-1, brow)
			b.set(ihx+2, ey-1, brow)
			b.set(ihx-1, ey-1, ktRedD)
			b.set(ihx-1, ey+1, ktRed)         // the stripe goes on under the eye
			b.rect(ihx-2, ey+2, 5, 1, s.Hair) // the short thick beard round the mouth and the jaw
			b.rect(ihx-1, ey+3, 3, 1, s.Hair)
			b.set(ihx, ey+2, shade(s.Skin, 0.55))
		case vSide:
			b.rect(ihx-rxi+1, top, rxi+1, 1, ktRed)
			b.set(ihx+1, top+1, ktRed)
			b.set(ihx+1, ey-1, brow)
			b.set(ihx+2, ey-1, brow)
			b.rect(ihx, ey+2, rxi+1, 2, s.Hair) // the beard on the jaw
		case vBack:
			b.rect(ihx-1, top, 1, chin-top, ktRed)
			b.set(ihx-1, chin-1, ktRedD)
		}
	case ktmSnake, ktmBoss:
		band, bandL := snBand, snBandL
		if k == ktmBoss {
			band, bandL = bbBand, hx(0x3a3a42)
		}
		by := ey - 1
		switch vw {
		case vFront:
			b.rect(ihx-rxi, by, 2*rxi+1, 1, band)
			b.set(ihx-1, by, bandL)
			// the tails peep out behind the head and flutter
			b.set(ihx+rxi+1, by+sw, band)
			b.set(ihx+rxi+2, by+1, bandL)
		case vSide:
			b.rect(ihx-rxi, by, 2*rxi+1, 1, band)
			b.set(ihx+1, by, bandL)
			b.set(ihx-rxi-1, by, band) // knot
			b.set(ihx-rxi-2, by+1-sw, band)
			b.set(ihx-rxi-3, by+1, bandL)
			b.set(ihx-rxi-4, by+2-sw, band)
		case vBack:
			b.rect(ihx-rxi, by, 2*rxi+1, 1, band)
			b.set(ihx, by+1, band) // knot and two tails
			b.rect(ihx-1+sw, by+2, 1, 3, band)
			b.rect(ihx+1, by+2, 1, 2+sw, bandL)
		}
		if k == ktmBoss && vw != vBack { // the eye patch over his right eye, the strap over the band
			switch vw {
			case vFront:
				b.rect(ihx-2, ey, 2, 2, bbPatchE)
				b.set(ihx-2, ey-1, bbPatchE)
				b.set(ihx+1, by-1, bbPatchE)
			case vSide:
				b.set(ihx-1, ey-1, bbPatchE)
				b.set(ihx-2, ey-2, bbPatchE)
			}
		}
	case ktmEndo:
		ktmSkull(b, vw, p, ihx, top, ey, chin, rxi)
	}
}

// ktmSkull: the chrome skull of the endoskeleton, with the red eyes (hotter while he shoots).
func ktmSkull(b *pixBuf, vw int, p Pose, ihx, top, ey, chin, rxi int) {
	eye := enEye
	if p.Anim == AnimStab {
		eye = enEyeHot
	}
	row := func(y, x0, x1 int, c color.RGBA) { b.rect(ihx+x0, y, x1-x0+1, 1, c) }
	switch vw {
	case vFront:
		row(top, -1, 1, enHi)
		row(top+1, -2, 2, enChrome)
		b.set(ihx-1, top+1, enHi)
		for y := top + 2; y <= chin; y++ {
			row(y, -rxi, rxi, enChrome)
		}
		b.set(ihx+rxi, top+2, enMid)
		// brow ridge and the dark sockets with the red eyes
		row(ey-1, -2, 2, enMid)
		row(ey, -2, 2, enGap)
		b.set(ihx, ey, enMid)
		b.set(ihx-1, ey, eye)
		b.set(ihx+1, ey, eye)
		b.set(ihx, ey+1, enGap) // nose
		row(ey+1, -rxi, -rxi, enMid)
		row(ey+1, rxi, rxi, enMid)
		// teeth
		for x := -2; x <= 2; x++ {
			c := enTeeth
			if x%2 != 0 {
				c = enDark
			}
			b.set(ihx+x, ey+2, c)
		}
		row(chin, -1, 1, enMid) // jaw
	case vSide:
		row(top, -2, 1, enHi)
		row(top+1, -3, 2, enChrome)
		for y := top + 2; y <= chin; y++ {
			row(y, -rxi, rxi, enChrome)
		}
		row(top+2, -rxi, -rxi+1, enMid)
		b.set(ihx+1, ey-1, enMid)
		b.rect(ihx+1, ey, 2, 1, enGap)
		b.set(ihx+1, ey, eye)
		b.set(ihx+rxi, ey+1, enGap)
		b.set(ihx-1, ey+1, enDark) // jaw hinge
		b.set(ihx-1, ey+2, enGap)
		b.rect(ihx, ey+2, rxi+1, 1, enTeeth)
		b.set(ihx+1, ey+2, enDark)
		row(chin, -1, rxi-1, enMid)
	case vBack:
		row(top, -1, 1, enHi)
		row(top+1, -2, 2, enChrome)
		for y := top + 2; y <= chin; y++ {
			row(y, -rxi, rxi, enChrome)
		}
		row(top+3, -1, 1, enMid)
		b.set(ihx, top+2, enMid)
		b.rect(ihx, ey, 1, chin-ey+1, enDark) // the top of the spine
		row(chin, -1, 1, enMid)
	}
}

// ktmTorso paints the torso details over [x0,x1) x [y0,y1].
func ktmTorso(b *pixBuf, s *CharSpec, vw, x0, x1, y0, y1, mid int, p Pose) {
	k := ktmKind(s)
	if k == ktmNone {
		return
	}
	b.cur = 2
	tw := x1 - x0
	switch k {
	case ktmKratos:
		skD := shade(s.Skin, 0.82)
		if vw != vSide { // pectorals and abs
			b.rect(x0+1, y0+3, tw-2, 1, skD)
			b.set(mid, y0+1, skD)
			b.set(mid, y0+2, skD)
			b.set(mid, y0+5, skD)
			b.set(mid, y0+7, skD)
			b.set(mid-2, y0+5, skD)
			b.set(mid+2, y0+5, skD)
		}
		switch vw {
		case vFront:
			// the red tattoo on his left side (screen right), from the shoulder to the belly
			b.rect(x1-3, y0, 2, 1, ktRed)
			b.line(x1-2, y0+1, x1-2, y1-3, ktRed)
			b.line(x1-3, y0+2, mid+1, y0+4, ktRed)
			b.set(x1-3, y1-3, ktRedD)
			// the leather strap across the chest, from his right shoulder, with the gold disc
			b.line(x0, y0, x1-1, y1-2, ktLeather)
			b.line(x0+1, y0, x1-1, y1-3, ktLeather)
			b.line(x0, y0+1, x1-2, y1-2, ktLeathL)
			b.rect(x0+1, y0, 2, 2, ktGold)
			b.set(x0+1, y0, ktGoldL)
		case vBack:
			b.line(x0+1, y0, x0+1, y1-3, ktRed)
			b.line(x1-1, y0, x0, y1-2, ktLeather)
			b.line(x1-2, y0, x0, y1-3, ktLeather)
		case vSide:
			b.rect(mid, y0, 1, 5, ktRed)
			b.line(x0, y0, x1-1, y1-3, ktLeather)
			b.line(x0+1, y0, x1-1, y1-4, ktLeathL)
			b.set(x1-2, y0+1, ktGold)
		}
		// the belt with the gold buckle
		b.rect(x0, y1-1, tw, 2, ktLeather)
		b.rect(x0, y1-1, tw, 1, ktLeathL)
		if vw == vFront {
			b.rect(mid-1, y1-1, 3, 2, ktGold)
			b.set(mid-1, y1-1, ktGoldL)
		} else if vw == vSide {
			b.set(x1-1, y1-1, ktGold)
		}
		// the dark red leather skirt with grey flaps and gold studs
		for y := y1 + 1; y <= y1+5; y++ {
			ex := (y - y1) / 3
			for x := x0 - ex; x < x1+ex; x++ {
				c := ktSkirt
				if (x-x0)%3 == 1 {
					c = ktSkirtL
				}
				if vw != vSide && (x-x0)%6 == 4 {
					c = ktFlap
				}
				if y == y1+5 {
					c = shade(c, 0.75)
				}
				b.set(x, y, c)
			}
		}
		if vw == vFront {
			b.set(x0+1, y1+1, ktGold)
			b.set(x1-2, y1+1, ktGold)
		}
	case ktmSnake, ktmBoss:
		patch, patchL, strap := snPatch, snPatchL, snStrap
		if k == ktmBoss {
			patch, patchL, strap = shade(s.Shirt, 0.8), shade(s.Shirt, 1.2), bbStrap
		}
		// turtleneck and shoulder pads
		b.rect(x0+1, y0, tw-2, 1, shade(s.Shirt, 1.25))
		b.rect(x0, y0, 2, 2, patch)
		b.rect(x1-2, y0, 2, 2, patch)
		b.set(x0, y0, patchL)
		switch vw {
		case vFront:
			b.rect(x0+1, y0+2, 1, y1-y0-3, strap) // harness
			b.rect(x1-2, y0+2, 1, y1-y0-3, strap)
			b.rect(mid-1, y0+3, 3, 2, patch) // chest plate
			b.set(mid-1, y0+3, patchL)
			if k == ktmBoss {
				b.set(x0+2, y0+3, shade(s.Shirt, 0.7)) // pockets
				b.set(x1-3, y0+3, shade(s.Shirt, 0.7))
			}
		case vBack:
			b.line(x0+1, y0+1, x1-2, y1-3, strap)
			b.line(x1-2, y0+1, x0+1, y1-3, strap)
			b.rect(mid-1, y0+2, 3, 3, patch) // the pack
		case vSide:
			b.rect(x0, y0+2, 1, y1-y0-3, strap)
			b.rect(x0-1, y0+2, 1, 3, patch) // the pack on his back
		}
		// belt with pouches
		b.rect(x0, y1-2, tw, 1, strap)
		pouch := patch
		if k == ktmBoss {
			pouch = bbPouch
		}
		for x := x0; x < x1; x += 3 {
			b.rect(x, y1-1, 2, 1, pouch)
		}
	case ktmEndo:
		// dark body under the chrome: clavicles, ribs, sternum, spine and pelvis
		for y := y0; y <= y1; y++ {
			b.rect(x0, y, tw, 1, enGap)
		}
		b.rect(x0, y0, tw, 1, enChrome) // shoulders and clavicles
		b.set(x0, y0, enHi)
		b.set(mid, y0, enMid)
		switch vw {
		case vFront, vBack:
			for _, r := range []int{2, 4, 6} {
				if y0+r < y1-2 {
					b.rect(x0+1, y0+r, tw-2, 1, enChrome)
					b.set(x0+1, y0+r, enHi)
					b.set(x1-2, y0+r, enMid)
				}
			}
			for y := y0 + 1; y < y1-1; y++ { // sternum / spine
				c := enMid
				if (y-y0)%2 == 0 {
					c = enHi
				}
				if vw == vBack {
					c = enChrome
					if (y-y0)%2 == 1 {
						c = enDark
					}
				}
				b.set(mid, y, c)
			}
			if vw == vFront { // the power core glows red between the ribs
				b.set(mid-2, y0+5, enEye)
			}
		case vSide:
			for y := y0 + 1; y < y1-1; y++ { // spine at the back, ribs to the front
				c := enChrome
				if (y-y0)%2 == 1 {
					c = enDark
				}
				b.set(x0, y, c)
			}
			for _, r := range []int{2, 4, 6} {
				if y0+r < y1-2 {
					b.rect(x0+1, y0+r, tw-1, 1, enChrome)
					b.set(x1-1, y0+r, enMid)
				}
			}
		}
		b.rect(x0, y1-1, tw, 2, enMid) // pelvis
		b.rect(x0+1, y1-1, tw-2, 1, enChrome)
		if vw != vSide {
			b.set(mid, y1, enGap)
		}
	}
}

// ktmArm paints the arm details after the plain arm (shoulder sx,sy, elbow mx,my, hand ex,ey).
func ktmArm(b *pixBuf, s *CharSpec, vw int, sx, sy, mx, my, ex, ey float64, armW int, dark bool, id uint8) {
	k := ktmKind(s)
	if k == ktmNone {
		return
	}
	b.cur = id
	fl := func(f float64) int { return int(math.Floor(f)) }
	lerp := func(a, c, t float64) float64 { return a + (c-a)*t }
	switch k {
	case ktmKratos:
		// striped leather bracer on the forearm
		for i := 0; i <= 4; i++ {
			t := 0.3 + 0.7*float64(i)/4
			c := ktLeather
			if i%2 == 1 {
				c = ktLeathL
			}
			b.rect(fl(lerp(mx, ex, t)), fl(lerp(my, ey, t)), armW, 1, ktmDark(c, dark))
		}
		// the tattoo runs down his left arm (screen right from the front, the near arm from the side)
		if (vw == vFront && id == 4) || (vw == vBack && id == 3) || (vw == vSide && id == 4) {
			b.line(fl(sx), fl(sy), fl(mx), fl(my), ktmDark(ktRed, dark))
		}
	case ktmSnake, ktmBoss:
		b.rect(fl(sx), fl(sy), armW, 1, ktmDark(snPatch, dark)) // shoulder pad
		if k == ktmBoss {
			b.rect(fl(sx), fl(sy), armW, 1, ktmDark(shade(s.Shirt, 0.8), dark))
		}
		b.rect(fl(ex), fl(ey), armW, 2, ktmDark(snGlove, dark))
		b.set(fl(ex), fl(ey), ktmDark(snGloveL, dark))
		b.rect(fl(lerp(mx, ex, 0.75)), fl(lerp(my, ey, 0.75)), armW, 1, ktmDark(snStrap, dark)) // cuff
	case ktmEndo:
		bone, hi := ktmDark(enChrome, dark), ktmDark(enHi, dark)
		thickLine(b, sx, sy, mx, my, 1, bone)
		thickLine(b, mx, my, ex, ey, 1, bone)
		if armW > 1 { // the piston beside each bone
			thickLine(b, sx+1, sy+1, mx+1, my, 1, ktmDark(enMid, dark))
			thickLine(b, mx+1, my+1, ex+1, ey-1, 1, ktmDark(enMid, dark))
		}
		b.rect(fl(sx), fl(sy), armW, 1, hi) // shoulder ball
		b.set(fl(mx), fl(my), hi)           // elbow
		b.rect(fl(ex), fl(ey), armW, 2, ktmDark(enMid, dark))
		b.set(fl(ex), fl(ey)+1, ktmDark(enDark, dark)) // fingers
	}
}

// ktmLeg paints the leg details after the plain leg (hip hxp,hipY, knee mx,my, foot ex,ey; shoe at sxx,syy).
func ktmLeg(b *pixBuf, s *CharSpec, vw int, hxp, hipY, mx, my, ex, ey float64, sxx, syy, legW int, dark bool) {
	k := ktmKind(s)
	if k == ktmNone {
		return
	}
	fl := func(f float64) int { return int(math.Floor(f)) }
	switch k {
	case ktmKratos, ktmSnake, ktmBoss: // tall boots from the knee down
		boot, bootL := ktBoot, ktBootL
		if k != ktmKratos {
			boot, bootL = snGlove, snGloveL
		}
		thickLine(b, mx, my+2, ex, ey, legW, ktmDark(boot, dark))
		b.line(fl(mx), fl(my)+2, fl(ex), fl(ey), ktmDark(bootL, dark))
		if k != ktmKratos { // knee pads
			pad := snPatch
			if k == ktmBoss {
				pad = shade(s.Pants, 0.75)
			}
			b.rect(fl(mx), fl(my), legW, 1, ktmDark(pad, dark))
		} else {
			b.rect(fl(mx), fl(my)+2, legW, 1, ktmDark(bootL, dark)) // the cuff of the boot
		}
	case ktmEndo:
		bone := ktmDark(enChrome, dark)
		thickLine(b, hxp, hipY+1, mx, my, 1, bone)
		thickLine(b, mx, my, ex, ey, 1, bone)
		if legW > 1 {
			thickLine(b, hxp+1, hipY+1, mx+1, my, 1, ktmDark(enMid, dark))
			thickLine(b, mx+1, my+1, ex+1, ey, 1, ktmDark(enMid, dark))
		}
		b.rect(fl(mx), fl(my), legW, 1, ktmDark(enHi, dark)) // knee joint
		// the metal foot
		if vw == vSide {
			b.rect(sxx-1, syy, legW+2, 2, ktmDark(enMid, dark))
			b.rect(sxx-1, syy, legW+2, 1, ktmDark(enChrome, dark))
			b.set(sxx+legW, syy+1, ktmDark(enDark, dark))
		} else {
			b.rect(sxx, syy, legW, 2, ktmDark(enMid, dark))
			b.rect(sxx, syy, legW, 1, ktmDark(enChrome, dark))
		}
	}
}

// ktmHeld paints Kratos's axe (when the hand is empty). True when it painted.
func ktmHeld(b *pixBuf, s *CharSpec, kind, vw int, hx0, hy0 float64) bool {
	if ktmKind(s) != ktmKratos || kind != HoldNone {
		return false
	}
	x, y := int(math.Floor(hx0)), int(math.Floor(hy0))
	// keep the whole axe inside the sprite box (arms raised in a dance, the hand near a side)
	y = maxI(y, 11)
	x = minI(maxI(x, 2), charW-8)
	// the blade, drawn from a small pattern: D dark back, B blade, L the bright cutting edge (pointing away)
	blade := []string{
		".BL.",
		"BBBL",
		"DBBBL",
		"DBBBL",
		"BBBL",
		".BL.",
	}
	paint := func(bx, by, dir int) {
		for r, row := range blade {
			for c, ch := range row {
				col := ktBlade
				switch ch {
				case '.':
					continue
				case 'D':
					col = ktBladeD
				case 'L':
					col = ktBladeL
				}
				b.set(bx+dir*c, by+r, col)
			}
		}
	}
	switch vw {
	case vSide: // the handle slants forward, the blade up and forward
		b.line(x+1, y+2, x+2, y-6, ktAxeH)
		b.set(x+1, y-1, ktAxeHL)
		paint(x+2, y-10, 1)
	default:
		dir := 1
		if vw == vBack && x > charW/2 {
			dir = -1 // the blade towards the body, never out of the box
		}
		b.rect(x+dir, y-7, 1, 10, ktAxeH)
		b.set(x+dir, y-3, ktAxeHL)
		b.set(x+dir, y+2, ktLeather)
		paint(x+2*dir, y-10, dir)
	}
	return true
}

// drawRifle: the big rifle of the endoskeleton (HoldRifle, drawn by drawHeld for genChar and the rig).
func drawRifle(b *pixBuf, vw, x, y int) {
	body, dk, hi := hx(0x3a3c44), hx(0x1c1c22), hx(0x8e929e)
	switch vw {
	case vSide: // pointing forward
		b.rect(x-3, y-2, 9, 2, body)
		b.rect(x-3, y-2, 9, 1, hi)
		b.rect(x+6, y-2, 2, 1, dk) // barrel
		b.rect(x-4, y-1, 2, 2, dk) // stock
		b.rect(x+1, y, 2, 2, dk)   // magazine
		b.set(x+3, y-3, hi)        // sight
	case vBack:
		b.rect(x-1, y-7, 2, 10, body)
		b.rect(x-1, y-7, 1, 10, hi)
		b.rect(x-1, y+3, 2, 1, dk)
	default: // held upright beside the body, the muzzle up
		b.rect(x, y-8, 2, 11, body)
		b.rect(x, y-8, 1, 11, hi)
		b.rect(x, y-10, 1, 2, dk)  // barrel
		b.rect(x+2, y-2, 1, 2, dk) // magazine
		b.rect(x, y+3, 2, 1, dk)   // stock
	}
}
