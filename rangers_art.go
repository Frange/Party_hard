package main

import (
	"image/color"
	"math"
	"math/rand"
)

// ---------------------------------------------------------------------------
// The Power Rangers of the 5-star squad (police/ranger_*), with their own procedural drawing like
// the superheroes (hero_art.go): the generic body showed a policeman with a coloured helmet.
// Chibi style after the classic team: a big smooth helmet with a dark outline, a large black
// visor (a goggle that narrows into a trapezoid) with a light rim and a shine, a silver mouth
// plate, a full suit with thin tights, two white lines crossed on the chest, a white belt with a
// golden buckle, white gloves and boots. Each colour has its touch:
//   green: golden shield over the shoulders and a dagger at the belt; black: golden trim;
//   pink and yellow: a short skirt; pink a winged visor, yellow two fangs, black a wide visor.
// The suit takes the spec's shirt colour (so the shop uniforms still recolour it), the name
// chooses the rest. Visual only: footprint and stats are those of the unit spec. Rig parts:
// 1 helmet, 2 torso (skirt, shield and dagger too), 3/4 arms, 5/6 legs.
// ---------------------------------------------------------------------------

const (
	rgNone = iota
	rgRed
	rgBlue
	rgYellow
	rgGreen
	rgPink
	rgBlack
)

var rangerKinds = map[string]int{
	"ranger_red": rgRed, "ranger_blue": rgBlue, "ranger_yellow": rgYellow,
	"ranger_green": rgGreen, "ranger_pink": rgPink, "ranger_black": rgBlack,
}

// rangerSuitCols: the default suit colour of each ranger (the unit specs in units.go use them).
var rangerSuitCols = map[int]uint32{
	rgRed: 0xd01818, rgBlue: 0x1858d0, rgYellow: 0xe0c010, rgGreen: 0x18a030, rgPink: 0xe050a8, rgBlack: 0x2a2a34,
}

var (
	rgWhite  = hx(0xf6f6f4)
	rgWhiteD = hx(0xc4c6d0)
	rgVisor  = hx(0x0c0c14)
	rgShine  = hx(0x9ad0ff)
	rgRim    = hx(0xc8ccd6)
	rgSilver = hx(0xb4b8c4)
	rgGold   = hx(0xf2c230)
	rgGoldD  = hx(0xb08414)
)

func rangerKind(s *CharSpec) int {
	if s == nil {
		return rgNone
	}
	return rangerKinds[s.Name]
}

func isRangerArt(s *CharSpec) bool { return rangerKind(s) != rgNone }

// rangerHalo: the glow under a ranger of the squad, in the colour of the suit it wears (the
// pictures come from the folder in alphabetical order, so the squad index alone is not enough).
func rangerHalo(cs *CharSet, alt int) color.RGBA {
	if cs != nil && cs.Spec != nil {
		switch rangerKind(cs.Spec) {
		case rgRed:
			return color.RGBA{220, 30, 30, 46}
		case rgBlue:
			return color.RGBA{30, 90, 220, 46}
		case rgYellow:
			return color.RGBA{235, 210, 30, 46}
		case rgGreen:
			return color.RGBA{40, 180, 60, 46}
		case rgPink:
			return color.RGBA{235, 90, 190, 46}
		case rgBlack:
			return color.RGBA{150, 150, 170, 46}
		}
	}
	n := len(rangerHaloCols)
	return rangerHaloCols[((alt%n)+n)%n]
}

// rangerSquadOff: the squad has five rangers and the team six colours, so each squad starts at a
// different colour (taken from the clock, so the random source is not disturbed) and the five
// arrive with five different suits.
func rangerSquadOff(unit int, t float64) int {
	if unit != unitRangers {
		return 0
	}
	return int(t*10) % 6
}

// withPresetByName: the -with debug flag also takes the squad's rangers (ranger_red...), to see
// them at a party without five stars.
func withPresetByName(slug string) func(*rand.Rand) CharSpec {
	if f := presetByName(slug); f != nil {
		return f
	}
	for _, s := range unitSpecs().rangers {
		if s.Name == slug {
			sp := s
			return func(*rand.Rand) CharSpec { return sp }
		}
	}
	return nil
}

// rangerVisor: the visor seen from the front, rows from hy+2, columns 8..15. K black, w shine,
// r light rim, s silver mouth plate, '.' helmet.
func rangerVisor(k int) []string {
	v := []string{
		"........",
		".rrrrrr.",
		"rKwwKKKr",
		"rKKKKKKr",
		"rKKKKKKr",
		".rrKKrr.",
		"...ss...",
	}
	switch k {
	case rgPink: // pterodactyl: the visor opens like wings
		v[1] = "rr....rr"
		v[2] = "KKwwKKKK"
		v[3] = "KKKKKKKK"
	case rgYellow: // sabertooth: two fangs
		v[5] = ".rKrrKr."
		v[6] = "..K..K.."
	case rgBlack: // mastodon: a wide band
		v[1] = "rrrrrrrr"
		v[2] = "KwwKKKKK"
		v[3] = "KKKKKKKK"
	case rgBlue: // triceratops: a little horn on the forehead
		v[0] = "...rr..."
	case rgGreen: // dragon: a golden ridge over the visor
		v[0] = "..gggg.."
	}
	return v
}

func genRanger(s *CharSpec, vw int, p Pose) *pixBuf {
	k := rangerKind(s)
	suit := hx(rangerSuitCols[k])
	if s.Shirt.A != 0 {
		suit = s.Shirt
	}
	b := newPix(charW, charH)
	if trackParts {
		b.pid = make([]uint8, charW*charH)
	}
	rnd := func(f float64) int { return int(math.Floor(f + 0.5)) }
	dk := func(c color.RGBA, dark bool) color.RGBA {
		if dark {
			return shade(c, 0.78)
		}
		return c
	}
	hl := func(c color.RGBA) color.RGBA { return mixc(c, rgWhite, 0.3) }
	sd := func(c color.RGBA) color.RGBA { return shade(c, 0.72) }
	edge := shade(suit, 0.5)
	if k == rgBlack {
		edge = hx(0x0e0e14)
	}
	row := func(y, x0, x1 int, c color.RGBA) {
		for x := x0; x <= x1; x++ {
			b.set(x, y, c)
		}
	}
	skirt := k == rgPink || k == rgYellow

	bob := clampI(rnd(p.Bob), -2, 3)
	if p.Sit {
		bob = maxI(bob, 3)
	}
	ox := clampI(rnd(p.Sway*0.5), -2, 2)
	hox := ox
	if vw == vSide {
		ox = clampI(rnd(p.Lean*0.5), -1, 2)
		hox = clampI(ox+rnd(p.Lean*0.3+p.HeadZ*0.5), -1, 2)
	} else if p.HeadZ != 0 {
		hox += clampI(rnd(p.HeadZ*0.5), -1, 1)
	}

	hipY := 29 + bob
	ty0 := hipY - 10 // torso top
	hy := ty0 - 12   // helmet top
	footTop := 39
	const legW, armW, fist = 2, 2, 3
	seg, armSeg := 4.6, 4.4
	tor := []int{4, 5, 5, 5, 4, 4, 4, 4, 4, 4}
	lift := func(l Limb) int {
		if p.Sit {
			return 0
		}
		return clampI(rnd(10-l.Y), 0, 6)
	}

	// ---- legs: thin tights and white boots ----
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
		thickLine(b, hx0, hipTop, mx, my, legW, dk(suit, dark))
		// the boot starts a little under the knee
		bmx, bmy := mx+(ex-mx)*0.25, my+(ey-my)*0.25
		thickLine(b, mx, my, bmx, bmy, legW, dk(suit, dark))
		thickLine(b, bmx, bmy, ex, ey, legW, dk(rgWhite, dark))
		b.rect(int(math.Floor(bmx)), int(math.Floor(bmy)), legW, 1, dk(rgWhiteD, dark)) // boot cuff
		ed := 0.0
		if vw != vSide && sign > 0 {
			ed = float64(legW - 1)
		}
		thickLine(b, hx0+ed, hipTop+1, mx+ed, my, 1, dk(sd(suit), dark))
		fxI := int(math.Floor(ex))
		fy := int(math.Floor(ey)) + 1
		if vw == vSide {
			b.rect(fxI-1, fy, 4, 2, dk(rgWhite, dark))
			b.rect(fxI-1, fy+1, 4, 1, dk(rgWhiteD, dark))
		} else {
			x0 := fxI
			if sign < 0 {
				x0 = fxI - 1
			}
			b.rect(x0, fy, 3, 2, dk(rgWhite, dark))
			b.rect(x0, fy+1, 3, 1, dk(rgWhiteD, dark))
		}
	}

	// ---- torso ----
	skirtRows := func(c int, hws []int) {
		for r, hw := range hws {
			col := suit
			if r == len(hws)-1 {
				col = hl(suit)
			}
			row(hipY+r, c-hw+1, c+hw, col)
			b.set(c-hw+1, hipY+r, sd(suit))
			b.set(c+hw, hipY+r, sd(suit))
		}
	}
	torsoFront := func(front bool) {
		b.cur = 2
		m := 12 + ox
		for r, hw := range tor {
			y := ty0 + r
			x0, x1 := 12-hw+ox, 11+hw+ox
			row(y, x0, x1, suit)
			b.set(x0, y, sd(suit))
			b.set(x1, y, sd(suit))
			if r == 0 {
				row(y, x0+1, x1-1, hl(suit))
			}
		}
		wl, wr := 12-tor[len(tor)-1]+ox, 11+tor[len(tor)-1]+ox
		row(hipY, wl, wr, suit)
		row(hipY+1, wl, wr, sd(suit))
		switch k {
		case rgGreen: // the golden shield over the shoulders
			row(ty0, 12-tor[0]+ox, 11+tor[0]+ox, rgGold)
			row(ty0+1, 12-tor[1]+ox, 11+tor[1]+ox, rgGold)
			row(ty0+2, m-4, m+3, rgGold)
			row(ty0+3, m-2, m+1, rgGoldD)
			if front {
				for _, x := range []int{m - 3, m - 1, m, m + 2} {
					b.set(x, ty0+1, rgGoldD)
				}
				b.set(m-1, ty0+2, rgWhite)
				b.set(m, ty0+2, rgWhite)
			}
		case rgBlack: // golden trim at the collar and down the sides
			row(ty0, 12-tor[0]+ox, 11+tor[0]+ox, rgGold)
			for r := 1; r < len(tor); r++ {
				b.set(13-tor[r]+ox, ty0+r, rgGoldD)
				b.set(10+tor[r]+ox, ty0+r, rgGoldD)
			}
		}
		if front { // two white lines crossed on the chest
			top := ty0 + 1
			if k == rgGreen {
				top = ty0 + 4
			}
			for y := top; y <= hipY-3; y++ {
				d := y - top
				b.set(m-4+d, y, rgWhite)
				b.set(m+3-d, y, rgWhite)
			}
		}
		// white belt and golden buckle
		row(hipY-1, wl, wr, rgWhite)
		if front {
			row(hipY-2, m-1, m, rgGold)
			row(hipY-1, m-2, m+1, rgGold)
			b.set(m-1, hipY-1, rgGoldD)
			b.set(m, hipY-1, rgGoldD)
			if k == rgGreen { // the dagger at the belt
				b.set(m+3, hipY-2, rgGold)
				for y := hipY - 1; y <= hipY+2; y++ {
					b.set(m+3, y, rgWhite)
				}
				b.set(m+3, hipY+2, rgWhiteD)
			}
		} else {
			b.set(m-1, hipY-1, rgWhiteD)
		}
		if skirt {
			skirtRows(11+ox, []int{5, 6, 6})
		}
	}
	torsoSide := func() {
		b.cur = 2
		c := 11 + ox
		type rr struct{ a, z int }
		rows := []rr{{-2, 1}, {-2, 2}, {-3, 2}, {-3, 2}, {-2, 2}, {-2, 2}, {-2, 2}, {-2, 2}, {-2, 2}, {-2, 2}}
		for r, q := range rows {
			y := ty0 + r
			row(y, c+q.a, c+q.z, suit)
			b.set(c+q.a, y, sd(suit))
			if r == 0 {
				row(y, c+q.a+1, c+q.z, hl(suit))
			}
		}
		row(hipY, c-2, c+2, suit)
		row(hipY+1, c-2, c+2, sd(suit))
		switch k {
		case rgGreen:
			row(ty0, c-2, c+2, rgGold)
			row(ty0+1, c-1, c+2, rgGold)
			b.set(c+2, ty0+2, rgGold)
		case rgBlack:
			row(ty0, c-2, c+1, rgGold)
			for y := ty0 + 1; y < hipY-1; y++ {
				b.set(c-1, y, rgGoldD)
			}
		}
		top := ty0 + 1
		if k == rgGreen {
			top = ty0 + 3
		}
		for y := top; y <= hipY-3; y++ { // the crossed lines seen edge-on
			b.set(c+2-(y-top)%2, y, rgWhite)
		}
		row(hipY-1, c-2, c+2, rgWhite)
		b.set(c+2, hipY-1, rgGold)
		if k == rgGreen {
			for y := hipY - 1; y <= hipY+1; y++ {
				b.set(c, y, rgWhite)
			}
			b.set(c, hipY-2, rgGold)
		}
		if skirt {
			for r, w := range [][2]int{{-3, 3}, {-4, 3}, {-4, 4}} {
				col := suit
				if r == 2 {
					col = hl(suit)
				}
				row(hipY+r, c+w[0], c+w[1], col)
				b.set(c+w[0], hipY+r, sd(suit))
			}
		}
	}

	// ---- helmet ----
	head := func() {
		b.cur = 1
		hd := hox
		// neck
		if vw == vSide {
			row(hy+11, 11+hd, 12+hd, sd(suit))
		} else {
			row(hy+11, 10+hd, 13+hd, sd(suit))
		}
		var rows [][2]int // helmet rows from hy (x0, x1 before the head offset)
		if vw == vSide {
			rows = [][2]int{{10, 14}, {9, 15}, {8, 16}, {8, 16}, {8, 16}, {8, 16}, {8, 16}, {8, 16}, {8, 16}, {9, 15}, {10, 14}}
		} else {
			rows = [][2]int{{10, 13}, {9, 14}, {8, 15}, {7, 16}, {7, 16}, {7, 16}, {7, 16}, {7, 16}, {7, 16}, {8, 15}, {9, 14}}
		}
		for r, q := range rows { // dark outline
			row(hy+r, q[0]+hd, q[1]+hd, edge)
		}
		for r := 1; r < len(rows)-1; r++ { // the shell
			q := rows[r]
			x0, x1 := q[0]+1, q[1]-1
			if r == 1 {
				x0, x1 = rows[0][0], rows[0][1]
			}
			row(hy+r, x0+hd, x1+hd, suit)
		}
		// the shine on top
		switch vw {
		case vSide:
			row(hy+1, 11+hd, 12+hd, hl(suit))
			b.set(10+hd, hy+2, hl(suit))
		case vBack:
			row(hy+1, 10+hd, 12+hd, hl(suit))
			b.set(9+hd, hy+2, hl(suit))
			row(hy+9, 9+hd, 14+hd, sd(suit)) // the rim at the nape
		default:
			row(hy+1, 10+hd, 11+hd, hl(suit))
			b.set(9+hd, hy+2, hl(suit))
		}
		switch vw {
		case vSide:
			// visor at the front, a round ear disc behind it
			row(hy+3, 13+hd, 16+hd, rgRim)
			row(hy+4, 13+hd, 16+hd, rgVisor)
			b.set(15+hd, hy+4, rgShine)
			row(hy+5, 13+hd, 16+hd, rgVisor)
			row(hy+6, 14+hd, 16+hd, rgVisor)
			b.set(13+hd, hy+6, rgRim)
			row(hy+8, 14+hd, 16+hd, rgSilver)
			b.set(10+hd, hy+5, sd(suit))
			b.set(11+hd, hy+5, rgSilver)
			b.set(10+hd, hy+6, sd(suit))
			b.set(11+hd, hy+6, sd(suit))
			if k == rgGreen {
				row(hy+2, 12+hd, 15+hd, rgGold)
			}
		case vBack:
			b.set(7+hd, hy+5, sd(suit))
			b.set(16+hd, hy+5, sd(suit))
		default:
			for r, str := range rangerVisor(k) {
				for i, ch := range str {
					x, y := 8+i+hd, hy+2+r
					switch ch {
					case 'K':
						b.set(x, y, rgVisor)
					case 'w':
						b.set(x, y, rgShine)
					case 'r':
						b.set(x, y, rgRim)
					case 's':
						b.set(x, y, rgSilver)
					case 'g':
						b.set(x, y, rgGold)
					}
				}
			}
		}
	}

	// ---- arms: suit sleeves, white gloves ----
	arm := func(sx float64, l Limb, sign float64, id uint8, dark bool) (int, int) {
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
		lo, hi := 2.0, float64(22-armW)
		sx = clampF(sx, lo, hi)
		hxT := clampF(sx+tx, lo, hi)
		mx, my, ex, ey := ik2(sx, sy, hxT, math.Max(sy+ty, 2), armSeg, bx, by)
		ex, ey = clampF(ex, lo, hi), math.Max(ey, 2)
		mx, my = clampF(mx, lo, hi), math.Max(my, 2)
		thickLine(b, sx, sy, mx, my, armW, dk(suit, dark))
		gmx, gmy := mx+(ex-mx)*0.35, my+(ey-my)*0.35
		thickLine(b, mx, my, gmx, gmy, armW, dk(suit, dark))
		thickLine(b, gmx, gmy, ex, ey, armW, dk(rgWhite, dark))
		b.rect(int(math.Floor(gmx)), int(math.Floor(gmy)), armW, 1, dk(rgWhiteD, dark)) // glove cuff
		thickLine(b, sx, sy, sx, sy+1, armW, dk(hl(suit), dark))                        // shoulder
		ix, iy := int(math.Floor(ex)), int(math.Floor(ey))
		gx := ix
		if vw != vSide && sign < 0 {
			gx = ix - 1
		}
		gx = clampI(gx, int(lo), int(hi)+armW-fist)
		b.rect(gx, iy, fist, fist, dk(rgWhite, dark))
		b.rect(gx, iy+fist-1, fist, 1, dk(rgWhiteD, dark))
		if vw == vSide {
			return clampI(ix, 3, 12), iy + 1
		}
		return clampI(ix, 3, 18), iy + 1
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

	lsx, rsx := 5.0, 17.0
	lhx, rhx := 9.0, 13.0
	switch vw {
	case vSide:
		cx := float64(11 + ox)
		arm(cx-0.5, p.LA, 1, 3, true)
		leg(10, p.LF, 1, 5, true)
		leg(11, p.RF, 1, 6, false)
		torsoSide()
		head()
		hx0, hy0 := arm(cx, p.RA, 1, 4, false)
		held(hx0, hy0, 4)
	case vBack:
		leg(rhx, p.LF, 1, 5, false)
		leg(lhx, p.RF, -1, 6, false)
		torsoFront(false)
		head()
		arm(rsx+float64(ox), p.LA, 1, 3, false)
		hx0, hy0 := arm(lsx+float64(ox), p.RA, -1, 4, false)
		held(hx0, hy0, 4)
	default:
		leg(lhx, p.LF, -1, 5, false)
		leg(rhx, p.RF, 1, 6, false)
		torsoFront(true)
		head()
		arm(lsx+float64(ox), p.LA, -1, 3, false)
		hx0, hy0 := arm(rsx+float64(ox), p.RA, 1, 4, false)
		held(hx0, hy0, 4)
	}
	return b
}
