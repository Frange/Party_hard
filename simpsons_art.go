package main

import (
	"image/color"
	"math"
	"math/rand"
)

// ---------------------------------------------------------------------------
// The yellow family (slugs "homer", "marge", "bart", "lisa" and "maggie") has its own procedural
// drawing (genSimpson), like Johnny 5: the generic body cannot give them the long block bodies,
// the huge white eyes and the hair of the cartoon. Our own pixel-art parody, flat colours and
// clean blocks: bright yellow skin everywhere, big round white eyes with a dot for the pupil.
//   - Homer: bald with two hairs on top and a zigzag over the ears, a brown stubble muzzle,
//     white short-sleeved shirt over a round belly, light blue trousers, black shoes.
//   - Marge: a very tall blue tower of hair (the tallest of the family), strapless long green
//     dress, red bead necklace and red shoes.
//   - Bart: crown of yellow spikes, red-orange T-shirt, blue shorts.
//   - Lisa: star of yellow spikes, red dress with a zigzag hem, white pearl necklace.
//   - Maggie: a baby, blue onesie, red dummy and one hair sticking up.
// The kids keep their kid size. Animation is the usual one (walk, run, dances and the gait hooks
// of gait.go, Homer staggers) read from the pose limbs: only the drawing changes, the hitbox,
// footprint and collisions are the ones of every guest (Marge's hair sticks out above the box
// like the other tall ones).
// ---------------------------------------------------------------------------

const (
	simpNone = iota
	simpHomer
	simpMarge
	simpBart
	simpLisa
	simpMaggie
)

func simpsonKind(s *CharSpec) int {
	switch s.Name {
	case "homer":
		return simpHomer
	case "marge":
		return simpMarge
	case "bart":
		return simpBart
	case "lisa":
		return simpLisa
	case "maggie":
		return simpMaggie
	}
	return simpNone
}

func isSimpson(s *CharSpec) bool { return simpsonKind(s) != simpNone }

// presetMaggie: the baby of the family (new famous guest, she comes with her family group).
func presetMaggie(r *rand.Rand) CharSpec {
	s := mk(r, "maggie")
	s.Skin, s.Hair = hx(0xffd521), hx(0xffd521)
	s.Top, s.Shirt, s.Pants, s.Shoes, s.Female = TopLong, hx(0x60a8f0), hx(0x60a8f0), hx(0x60a8f0), true
	s.Hair2, s.Body, s.Stature, s.Head = HairSpiky, BodySlim, 7, HeadBig
	return s
}

func init() {
	famousPresets = append(famousPresets, presetMaggie)
	displayNameOverrides["maggie"] = "Maggy"
	for i, g := range starGroups {
		if len(g) > 0 && g[0] == "homer" {
			starGroups[i] = append(append([]string(nil), g...), "maggie")
		}
	}
}

var (
	spSkin   = hx(0xffd521)
	spSkinD  = hx(0xe0a800)
	spEye    = hx(0xffffff)
	spPupil  = hx(0x141414)
	spMouth  = hx(0x6a3a1a)
	spHomerH = hx(0x1c1c22) // Homer's hairs and shoes
	spBeard  = hx(0xbc9460)
	spBeardD = hx(0x9a7444)
	spShirtW = hx(0xf6f6f6)
	spShirtG = hx(0xcfd2dc)
	spPantsH = hx(0x6aa0e0)
	spPantsD = hx(0x4a7cc0)
	spHairM  = hx(0x3a6ae0)
	spHairML = hx(0x6a9af0)
	spHairMD = hx(0x2448a8)
	spDressG = hx(0x5cc048)
	spDressD = hx(0x3c9a34)
	spRed    = hx(0xe02828)
	spRedD   = hx(0xa81818)
	spShirtB = hx(0xf05020)
	spShirtD = hx(0xc83a14)
	spShorts = hx(0x2a5ad0)
	spShortD = hx(0x1c3e9a)
	spShoeB  = hx(0x2a2a36)
	spDressL = hx(0xe8402a)
	spDressK = hx(0xb82a1c)
	spBaby   = hx(0x60a8f0)
	spBabyD  = hx(0x3a80c8)
)

// simpGeo: rows of the body (before the bob) and limb sizes of each member of the family.
type simpGeo struct {
	torsoTop, hipY   int     // first torso row, first leg row
	shL, shR, shY    int     // shoulders (front view)
	hipL, hipR       int     // hips (front view)
	legW, armW, hand int     // widths
	k                float64 // limb reach (adult 1, kids less)
	sleeve           float64 // part of the arm covered by the sleeve
	sleeveCol        color.RGBA
	legCol, shoe     color.RGBA
}

func simpGeoOf(kind int) simpGeo {
	switch kind {
	case simpHomer:
		return simpGeo{18, 30, 6, 16, 19, 9, 13, 3, 2, 2, 0.95, 0.4, spShirtW, spPantsH, spHomerH}
	case simpMarge:
		return simpGeo{21, 29, 8, 15, 22, 10, 13, 2, 1, 2, 0.9, 0, spSkin, spSkin, spRed}
	case simpBart:
		return simpGeo{24, 31, 7, 16, 25, 9, 13, 2, 1, 2, 0.7, 0.35, spShirtB, spSkin, spShoeB}
	case simpLisa:
		return simpGeo{24, 31, 8, 15, 25, 10, 13, 1, 1, 2, 0.68, 0, spSkin, spSkin, spRed}
	}
	return simpGeo{34, 38, 8, 15, 35, 10, 13, 2, 1, 1, 0.38, 0.6, spBaby, spBaby, spBabyD}
}

func genSimpson(s *CharSpec, vw int, p Pose) *pixBuf {
	kind := simpsonKind(s)
	b := newPix(charW, charH)
	if trackParts {
		b.pid = make([]uint8, charW*charH)
	}
	g := simpGeoOf(kind)
	rnd := func(f float64) int { return int(math.Floor(f + 0.5)) }
	dy := clampI(rnd(p.Bob), -1, 2)
	if p.Sit {
		dy = 3
		if kind == simpMaggie {
			dy = 1
		}
	}
	ox := clampI(rnd(p.Sway*0.7), -2, 2)
	if vw == vSide {
		ox = clampI(rnd(p.Lean*0.5), -1, 3)
	}
	footY := 41 // top row of the shoes
	if s.Ride != 0 {
		footY = 39
	}

	// --- limbs ---
	seg := func(x0, y0, x1, y1 float64, w int, c0, c1 color.RGBA, frac float64) {
		n := int(math.Max(math.Abs(x1-x0), math.Abs(y1-y0))) + 1
		for i := 0; i <= n; i++ {
			t := float64(i) / float64(n)
			x := rnd(x0 + (x1-x0)*t - float64(w-1)/2)
			y := rnd(y0 + (y1-y0)*t)
			c := c1
			if t < frac {
				c = c0
			}
			b.rect(x, y, w, 1, c)
		}
	}
	dark := func(c color.RGBA, d bool) color.RGBA {
		if d {
			return shade(c, 0.78)
		}
		return c
	}
	// leg: hip at hx (screen), side = -1/+1 screen side (0 in profile)
	leg := func(hx int, l Limb, side int, d bool, id uint8) {
		b.cur = id
		hy := float64(g.hipY + dy)
		fx := float64(hx)
		fy := float64(footY) + (l.Y-10)*g.k
		if vw == vSide {
			fx += l.Z * g.k * 0.8
		} else {
			fx += float64(side) * (l.X - 1.5) * 0.6 * g.k
		}
		if p.Sit {
			fy = math.Min(fy, hy+4*g.k+1)
			if vw == vSide {
				fx = float64(hx) + 5*g.k
			}
		}
		fy = math.Max(fy, hy+1)
		lc := g.legCol
		if kind == simpMarge || kind == simpLisa || kind == simpBart {
			lc = spSkin
		}
		seg(float64(hx), hy, fx, fy-1, g.legW, dark(lc, d), dark(lc, d), 0)
		if kind == simpHomer { // trouser shade on the inner edge
			b.rect(rnd(fx-float64(g.legW-1)/2)+boolI(side < 0)*(g.legW-1), rnd(fy-2), 1, 2, dark(spPantsD, d))
		}
		sx, sy := rnd(fx-float64(g.legW-1)/2), rnd(fy)
		sh := dark(g.shoe, d)
		if vw == vSide {
			w := g.legW + 2
			if kind == simpMaggie {
				w = g.legW + 1
			}
			b.rect(sx, sy, w, 2, sh)
		} else {
			w := g.legW + 1
			x := sx
			if side < 0 {
				x = sx - 1
			}
			b.rect(x, sy, w, 2, sh)
		}
	}
	// arm from the shoulder; side = screen side (outward), 0 in profile. Returns the hand.
	arm := func(sx, sy int, l Limb, side int, d bool, id uint8) (int, int) {
		b.cur = id
		var tx, ty float64
		if vw == vSide {
			tx, ty = l.Z*g.k*0.85, l.Y*g.k
		} else {
			tx, ty = float64(side)*l.X*0.8*g.k, l.Y*g.k
		}
		hx := clampI(sx+rnd(tx), 2, charW-3)
		hy := clampI(sy+rnd(ty), 2, 40)
		fore := spSkin
		if kind == simpMaggie {
			fore = spBaby
		}
		seg(float64(sx), float64(sy), float64(hx), float64(hy), g.armW, dark(g.sleeveCol, d), dark(fore, d), g.sleeve)
		hw := g.hand
		x := hx
		if side < 0 && hw > 1 {
			x = hx - 1
		}
		b.rect(x, hy, hw, hw, dark(spSkin, d))
		return hx, hy
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
	torso := func() {
		b.cur = 2
		simpTorso(b, kind, vw, ox, dy)
	}
	head := func() {
		b.cur = 1
		simpHead(b, kind, vw, ox, dy, p)
	}
	shY := g.shY + dy
	cxs := charCX - 1 + ox // profile column of hips and shoulders
	switch vw {
	case vSide:
		farArm := func() {
			lx, ly := arm(cxs, shY, p.LA, 0, true, 3)
			if p.Hold == HoldPompom {
				held(lx, ly, 3)
			}
		}
		raised := p.LA.Y < -2 // a raised far arm would hide behind the head
		leg(cxs, p.LF, 0, true, 5)
		if !raised {
			farArm()
		}
		leg(cxs, p.RF, 0, false, 6)
		torso()
		head()
		if raised {
			farArm()
		}
		hx0, hy0 := arm(cxs+1, shY, p.RA, 0, false, 4)
		held(hx0, hy0, 4)
	case vBack:
		leg(g.hipL+ox, p.RF, -1, false, 5)
		leg(g.hipR+ox, p.LF, 1, false, 6)
		torso()
		hl, hly := arm(g.shL+ox, shY, p.RA, -1, false, 3)
		held(hl, hly, 3)
		rx, ry := arm(g.shR+ox, shY, p.LA, 1, false, 4)
		if p.Hold == HoldPompom {
			held(rx, ry, 4)
		}
		head()
	default:
		leg(g.hipL+ox, p.LF, -1, false, 5)
		leg(g.hipR+ox, p.RF, 1, false, 6)
		torso()
		lx, ly := arm(g.shL+ox, shY, p.LA, -1, false, 3)
		if p.Hold == HoldPompom {
			held(lx, ly, 3)
		}
		hx0, hy0 := arm(g.shR+ox, shY, p.RA, 1, false, 4)
		head()
		held(hx0, hy0, 4)
	}
	if p.Hold == HoldPee {
		drawStream(b, vw, float64(charCX+ox)+0.5, float64(g.hipY+dy+2), float64(footY), p.Sit)
	}
	if s.Ride != 0 {
		drawRide(b, s, vw, charCX, g.hipY+dy-2)
	}
	return b
}

// span paints row y from x0 to x1 (inclusive) with c, the first column lit and the last shaded.
func simpSpan(b *pixBuf, y, x0, x1 int, c, lit, dk color.RGBA) {
	if x1 < x0 {
		return
	}
	b.rect(x0, y, x1-x0+1, 1, c)
	if x1-x0 >= 3 {
		if lit.A != 0 {
			b.set(x0, y, lit)
		}
		b.set(x1, y, dk)
	}
}

// simpTorso: clothes of the body (the trousers top and dresses included), offset by (ox, dy).
func simpTorso(b *pixBuf, kind, vw, ox, dy int) {
	none := color.RGBA{}
	row := func(y, x0, x1 int, c, lit, dk color.RGBA) { simpSpan(b, y+dy, x0+ox, x1+ox, c, lit, dk) }
	side := vw == vSide
	switch kind {
	case simpHomer:
		// shirt: narrow shoulders, round belly, untucked over the belt
		type w struct{ y, a, z int }
		rows := []w{{18, 9, 14}, {19, 8, 15}, {20, 8, 15}, {21, 8, 15}, {22, 7, 16}, {23, 7, 16}, {24, 7, 16}, {25, 7, 16}, {26, 7, 16}, {27, 7, 16}, {28, 8, 15}}
		if side {
			rows = []w{{18, 9, 13}, {19, 9, 14}, {20, 9, 14}, {21, 9, 15}, {22, 9, 15}, {23, 9, 16}, {24, 9, 16}, {25, 9, 16}, {26, 9, 16}, {27, 9, 15}, {28, 9, 15}}
		}
		for _, r := range rows {
			row(r.y, r.a, r.z, spShirtW, none, spShirtG)
		}
		row(29, 8, 15, spPantsH, none, spPantsD) // trousers top
		if side {
			row(29, 9, 14, spPantsH, none, spPantsD)
			b.set(15+ox, 29+dy, color.RGBA{})
			b.set(14+ox, 18+dy, spShirtG) // collar
			b.set(16+ox, 26+dy, spShirtG) // belly curve
		} else if vw == vFront {
			b.set(10+ox, 18+dy, spShirtG) // open collar
			b.set(13+ox, 18+dy, spShirtG)
			b.rect(11+ox, 18+dy, 2, 1, spSkin)
			b.set(12+ox, 26+dy, spShirtG) // a shadow under the belly
			b.set(11+ox, 27+dy, spShirtG)
		} else {
			b.rect(10+ox, 18+dy, 4, 1, spShirtG) // collar seen from behind
		}
	case simpMarge:
		// bare shoulders, red beads, strapless long green dress widening down to the ankles
		if side {
			row(21, 11, 13, spSkin, none, spSkinD)
			for x := 10; x <= 14; x++ {
				c := spRed
				if x%2 == 1 {
					c = spRedD
				}
				b.set(x+ox, 22+dy, c)
			}
			for y := 23; y <= 39; y++ {
				a, z := 10, 14
				if y >= 30 {
					a, z = 9, 15
				}
				if y >= 35 {
					a, z = 8, 15
				}
				row(y, a, z, spDressG, none, spDressD)
			}
		} else {
			row(21, 9, 14, spSkin, none, spSkinD)
			if vw == vFront {
				for x := 9; x <= 14; x++ {
					c := spRed
					if x%2 == 1 {
						c = spRedD
					}
					b.set(x+ox, 22+dy, c)
				}
				b.rect(11+ox, 23+dy, 2, 1, spRed) // the necklace hangs a little in front
			} else {
				row(22, 9, 14, spSkin, none, spSkinD)
				b.rect(11+ox, 21+dy, 2, 1, spRedD) // clasp at the nape
			}
			for y := 23; y <= 39; y++ {
				a, z := 9, 14
				switch {
				case y >= 37:
					a, z = 6, 17
				case y >= 34:
					a, z = 7, 16
				case y >= 30:
					a, z = 8, 15
				}
				row(y, a, z, spDressG, spDressG, spDressD)
				if vw == vFront && y == 23 {
					b.rect(11+ox, y+dy, 2, 1, spRed)
				}
			}
			b.rect(6+ox, 39+dy, 12, 1, spDressD) // hem
		}
	case simpBart:
		a, z := 8, 15
		if side {
			a, z = 9, 14
		}
		for y := 24; y <= 30; y++ {
			row(y, a, z, spShirtB, none, spShirtD)
		}
		for y := 31; y <= 33; y++ {
			row(y, a, z, spShorts, none, spShortD)
		}
		if !side {
			b.rect(11+ox, 33+dy, 2, 1, spShortD) // the gap between the legs of the shorts
			if vw == vFront {
				b.rect(10+ox, 24+dy, 4, 1, spShirtD) // neckline
			}
		}
	case simpLisa:
		a0, z0 := 9, 14
		if side {
			a0, z0 = 10, 14
		}
		for y := 24; y <= 34; y++ {
			a, z := a0, z0
			if y >= 29 {
				a, z = a0-1, z0+1
			}
			if y >= 32 && !side {
				a, z = a0-2, z0+2
			}
			row(y, a, z, spDressL, none, spDressK)
		}
		// zigzag hem
		a, z := a0-2, z0+2
		if side {
			a, z = a0-1, z0+1
		}
		for x := a; x <= z; x++ {
			if (x-a)%2 == 0 {
				b.set(x+ox, 35+dy, spDressL)
			}
		}
		if vw == vFront {
			for x := 10; x <= 13; x++ { // pearl necklace
				b.set(x+ox, 24+dy, spEye)
			}
			b.set(10+ox, 25+dy, spEye)
			b.set(13+ox, 25+dy, spEye)
		} else if side {
			b.rect(12+ox, 24+dy, 3, 1, spEye)
		}
	default: // Maggie: blue onesie
		a, z := 9, 14
		if side {
			a, z = 10, 14
		}
		for y := 34; y <= 38; y++ {
			row(y, a, z, spBaby, none, spBabyD)
		}
		if vw == vFront {
			b.set(11+ox, 35+dy, spBabyD) // a little star on the chest
		}
	}
}

// simpHead: face, ears and hair, offset by (ox, dy).
func simpHead(b *pixBuf, kind, vw, ox, dy int, p Pose) {
	px := func(x, y int, c color.RGBA) { b.set(x+ox, y+dy, c) }
	rc := func(x, y, w, h int, c color.RGBA) { b.rect(x+ox, y+dy, w, h, c) }
	// block head with the two top corners cut, shaded on the far column
	block := func(x0, y0, x1, y1 int) {
		rc(x0, y0+1, x1-x0+1, y1-y0, spSkin)
		rc(x0+1, y0, x1-x0-1, 1, spSkin)
		rc(x1, y0+1, 1, y1-y0, spSkinD)
	}
	blink := p.Anim == AnimIdle && p.F == 1 && kind == simpHomer
	// big eye of w x h with the pupil looking at (px0, py0) inside
	eye := func(x, y, w, h, pdx, pdy int) {
		rc(x, y, w, h, spEye)
		if blink {
			rc(x, y, w, 1, spSkinD)
			rc(x, y+1, w, h-1, spSkinD)
			rc(x, y+h-1, w, 1, spPupil)
			return
		}
		px(x+pdx, y+pdy, spPupil)
	}
	side, back := vw == vSide, vw == vBack
	switch kind {
	case simpHomer:
		if side {
			block(8, 7, 15, 17)
			rc(15, 8, 1, 10, spSkin)
			eye(14, 9, 3, 3, 2, 1)
			rc(16, 12, 2, 1, spSkin) // nose
			px(17, 13, spSkinD)
			for y := 13; y <= 17; y++ { // stubble muzzle bulging forward
				x1 := 16
				if y == 17 {
					x1 = 15
				}
				rc(11, y, x1-10, 1, spBeard)
			}
			rc(13, 16, 4, 1, spMouth)
			px(11, 13, spBeardD)
			rc(9, 12, 2, 2, spSkinD) // ear
			px(10, 12, spSkin)
			px(9, 10, spHomerH) // zigzag over the ear
			px(10, 11, spHomerH)
			px(11, 10, spHomerH)
			px(11, 6, spHomerH) // the two hairs
			px(12, 5, spHomerH)
			px(13, 6, spHomerH)
			px(14, 5, spHomerH)
			rc(9, 18, 4, 1, spSkin) // neck behind the collar
			return
		}
		block(8, 7, 15, 17)
		px(11, 6, spHomerH)
		px(12, 5, spHomerH)
		px(13, 6, spHomerH)
		px(14, 5, spHomerH)
		rc(7, 12, 1, 2, spSkinD) // ears
		rc(16, 12, 1, 2, spSkinD)
		if back {
			for x := 9; x <= 14; x++ { // the M of hair at the back
				px(x, 13+(x%2), spHomerH)
			}
			px(8, 11, spHomerH)
			px(15, 11, spHomerH)
			return
		}
		px(8, 11, spHomerH) // zigzags over the ears
		px(8, 12, spSkin)
		px(15, 11, spHomerH)
		eye(9, 9, 3, 3, 1, 1)
		eye(12, 9, 3, 3, 1, 1)
		rc(11, 12, 2, 1, spSkin) // nose
		rc(11, 13, 2, 1, spSkinD)
		rc(9, 13, 2, 1, spBeard)
		rc(13, 13, 2, 1, spBeard)
		rc(8, 14, 8, 3, spBeard)
		rc(9, 17, 6, 1, spBeard)
		rc(15, 14, 1, 3, spBeardD)
		rc(10, 15, 4, 1, spMouth)
		if p.Open || p.Anim == AnimLaugh || p.Anim == AnimPanic {
			rc(10, 16, 4, 1, spMouth)
		}
	case simpMarge:
		hairTop := 2
		if dy < 0 {
			hairTop = 2 - dy // keeps the tower inside the sprite box when the body bobs up
		}
		tower := func(a, z int) {
			for y := hairTop; y <= 12; y++ {
				x0, x1 := a, z
				if y == hairTop {
					x0, x1 = a+2, z-2
				} else if y == hairTop+1 {
					x0, x1 = a+1, z-1
				}
				rc(x0, y, x1-x0+1, 1, spHairM)
				px(x0, y, spHairML)
				px(x1, y, spHairMD)
				if (y-hairTop)%3 == 2 { // curls
					px(x0+2, y, spHairMD)
					px(x0+3, y, spHairMD)
				}
			}
		}
		if side {
			tower(7, 14)
			block(10, 11, 15, 19)
			rc(8, 11, 3, 6, spHairM) // hair covering the back of the head
			rc(8, 11, 1, 6, spHairMD)
			px(10, 17, spHairMD)
			eye(14, 13, 3, 3, 2, 1)
			px(15, 12, spPupil) // lashes
			px(16, 12, spPupil)
			px(16, 16, spSkin) // nose
			rc(14, 18, 2, 1, spMouth)
			px(11, 15, spSkinD) // ear
			rc(11, 20, 2, 1, spSkin)
			return
		}
		tower(8, 15)
		if back {
			rc(8, 13, 8, 7, spHairM)
			rc(15, 13, 1, 7, spHairMD)
			rc(10, 17, 4, 2, spHairMD) // bun at the nape
			rc(11, 20, 2, 1, spSkin)
			return
		}
		block(9, 11, 14, 19)
		rc(8, 11, 1, 4, spHairM) // hair framing the face
		rc(15, 11, 1, 4, spHairMD)
		eye(9, 13, 3, 3, 1, 1)
		eye(12, 13, 3, 3, 1, 1)
		px(9, 12, spPupil) // lashes
		px(11, 12, spPupil)
		px(12, 12, spPupil)
		px(14, 12, spPupil)
		rc(11, 16, 2, 1, spSkinD) // nose
		rc(10, 18, 4, 1, spMouth)
		rc(11, 20, 2, 1, spSkin) // neck
	case simpBart:
		spikes := func(a, z int) {
			for x := a; x <= z; x++ {
				top := 11
				if (x-a)%2 == 0 {
					top = 10
				}
				rc(x, top, 1, 13-top, spSkin)
				if top == 10 {
					px(x, 10, spSkinD)
				}
			}
		}
		if side {
			spikes(8, 14)
			block(8, 13, 15, 23)
			eye(14, 16, 3, 3, 2, 1)
			rc(16, 19, 1, 1, spSkin) // nose
			rc(13, 21, 3, 1, spMouth)
			rc(9, 17, 2, 2, spSkinD) // ear
			px(10, 17, spSkin)
			return
		}
		spikes(8, 15)
		block(8, 13, 15, 23)
		rc(7, 17, 1, 2, spSkinD) // ears
		rc(16, 17, 1, 2, spSkinD)
		if back {
			return
		}
		eye(9, 16, 3, 3, 1, 1)
		eye(12, 16, 3, 3, 1, 1)
		rc(11, 19, 2, 1, spSkinD) // nose
		rc(10, 21, 4, 1, spMouth)
		if p.Anim == AnimLaugh || p.Open {
			rc(11, 22, 2, 1, spMouth)
		}
	case simpLisa:
		star := []([2]int){}
		if side {
			star = [][2]int{{10, 13}, {10, 12}, {11, 11}, {12, 13}, {12, 12}, {13, 11}, {14, 13}, {14, 12}, {9, 14}, {8, 14}, {7, 14}, {8, 15}, {9, 15}}
			for _, q := range star {
				px(q[0], q[1], spSkin)
			}
			block(9, 14, 15, 23)
			eye(14, 16, 3, 3, 2, 1)
			px(15, 15, spPupil) // lashes
			px(16, 19, spSkin)  // nose
			rc(14, 21, 2, 1, spMouth)
			rc(10, 17, 2, 2, spSkinD) // ear
			return
		}
		// the star: three spikes up, one at each corner and one out of each side
		star = [][2]int{{11, 11}, {11, 12}, {11, 13}, {12, 11}, {12, 12}, {12, 13}, {11, 10}, {12, 10},
			{9, 12}, {9, 13}, {8, 12}, {14, 12}, {14, 13}, {15, 12},
			{7, 15}, {6, 15}, {7, 16}, {16, 15}, {17, 15}, {16, 16}}
		for _, q := range star {
			px(q[0], q[1], spSkin)
		}
		px(11, 10, spSkinD)
		px(8, 12, spSkinD)
		px(15, 12, spSkinD)
		block(8, 14, 15, 23)
		if back {
			return
		}
		eye(9, 16, 3, 3, 1, 1)
		eye(12, 16, 3, 3, 1, 1)
		px(9, 15, spPupil) // lashes
		px(14, 15, spPupil)
		rc(11, 19, 2, 1, spSkinD) // nose
		rc(10, 21, 4, 1, spMouth)
	default: // Maggie
		if side {
			px(11, 26, spSkin)
			px(11, 25, spSkinD)
			px(12, 24, spSkinD)
			block(9, 27, 14, 33)
			eye(13, 29, 2, 2, 1, 1)
			rc(15, 31, 1, 2, spRed) // dummy
			px(16, 31, spSkin)
			rc(10, 30, 1, 2, spSkinD) // ear
			return
		}
		px(12, 26, spSkin) // the one hair sticking up
		px(12, 25, spSkinD)
		px(13, 24, spSkinD)
		block(9, 27, 14, 33)
		if back {
			return
		}
		eye(9, 29, 2, 2, 1, 1)
		eye(13, 29, 2, 2, 0, 1)
		rc(11, 31, 2, 2, spRed) // dummy with its ring
		px(11, 31, spRedD)
		if p.F%4 == 3 {
			rc(11, 32, 2, 1, spRedD) // sucking
		}
	}
}
