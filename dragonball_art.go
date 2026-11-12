package main

import (
	"image/color"
	"math"
	"math/rand"
)

// ---------------------------------------------------------------------------
// Dragon Ball: our own pixel-art parodies of the fighters (goku, gohan, vegeta, krilin, piccolo,
// bulma, maestro_roshi and the new "cell"). They keep the generic body (same size, stature and
// hit-box as before; Piccolo and Cell a little taller) and genChar calls these hooks on top of it:
//   dbTorso (gi, sash, emblem, armour, wings), dbArm (wristbands, gloves, spots), dbLeg (boots),
//   dbHead (hair, faces, antennae, crest) and dbOver (shoulder pads and capes over the arms).
// When a slug is drawn here the older A-H / I-P / Q-Z touch-ups (charimp_*.go) are skipped, so the
// two passes never paint over each other. Everything stays inside the 24x44 box and follows the
// pose, so walking, running and dancing move the details with the body.
// ---------------------------------------------------------------------------

const (
	dbNone = iota
	dbGoku
	dbGohan
	dbVegeta
	dbKrilin
	dbPiccolo
	dbBulma
	dbRoshi
	dbCell
)

var dbKinds = map[string]int{
	"goku": dbGoku, "gohan": dbGohan, "vegeta": dbVegeta, "krilin": dbKrilin, "piccolo": dbPiccolo,
	"bulma": dbBulma, "maestro_roshi": dbRoshi, "cell": dbCell,
}

func dbKind(s *CharSpec) int {
	if s == nil || s.Name == "" {
		return dbNone
	}
	return dbKinds[s.Name]
}

// dbOwn: is this character drawn by this file (the charimp touch-ups are skipped)?
func dbOwn(s *CharSpec) bool { return dbKind(s) != dbNone }

var (
	dbHair    = hx(0x14101a)
	dbHairHi  = hx(0x3c3a5c)
	dbNavy    = hx(0x1e3488)
	dbNavyL   = hx(0x3456b8)
	dbGold    = hx(0xf0c030)
	dbWhite   = hx(0xf4f4f0)
	dbWhiteD  = hx(0xc8ccd8)
	dbInk     = hx(0x101014)
	dbRed     = hx(0xd02828)
	dbBrown   = hx(0x6a3a1a)
	dbBrownL  = hx(0x8a5228)
	dbGreen   = hx(0x5cb848)
	dbGreenD  = hx(0x3a8a30)
	dbPink    = hx(0xe87890)
	dbYellow  = hx(0xe8c040)
	dbYellowD = hx(0xb8902a)
	dbPurple  = hx(0x8a2ab0)
	dbPurpleL = hx(0xb060d8)
	dbCellSk  = hx(0x8ccc58) // Cell's light green
	dbCellSp  = hx(0x1a1a1c) // his black spots
	dbCellFc  = hx(0xece0dc) // pale face
	dbCellMg  = hx(0xb04aa8) // purple marks
)

// dbStamp paints a small map: each byte of rows is a colour of pal ('.' or a missing key is skipped).
// (x0, y0) is the top left corner of the map.
func dbStamp(b *pixBuf, x0, y0 int, rows []string, pal map[byte]color.RGBA) {
	for r, row := range rows {
		for c := 0; c < len(row); c++ {
			if col, ok := pal[row[c]]; ok {
				b.set(x0+c, y0+r, col)
			}
		}
	}
}

// dbMirror flips the rows of a map (side view facing -x is never used: the game flips the picture).
func dbMirror(rows []string) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		bs := []byte(r)
		for a, z := 0, len(bs)-1; a < z; a, z = a+1, z-1 {
			bs[a], bs[z] = bs[z], bs[a]
		}
		out[i] = string(bs)
	}
	return out
}

func dbDk(c color.RGBA, dark bool) color.RGBA {
	if dark {
		return shade(c, 0.78)
	}
	return c
}

// ---- hair maps (centre column of every front/back map is index 6; side maps index 7) ----

var (
	gokuHairFront = []string{
		"....#...#....",
		"...##..##....",
		"#..###.###..#",
		"##.#######.##",
		".##########+.",
		"..##+######..",
		".####...####.",
		"###.......###",
		"..#.#...#.#..",
		"..#.......#..",
	}
	gokuHairBack = []string{
		"....#...#....",
		"...##..##....",
		"#..###.###..#",
		"##.#######.##",
		".###########.",
		".#####+#####.",
		"#############",
		".###########.",
		"#.#########.#",
		"..#.#.#.#.#..",
	}
	gokuHairSide = []string{
		"..#...#......",
		"..##..##.....",
		"#.###.###....",
		"##########...",
		".#########+#.",
		"..##########.",
		"######....##.",
		".####.....#..",
		"#.##.........",
		"..#..........",
	}
	gohanHairFront = []string{
		".............",
		"....#..#.....",
		"..#.##.##.#..",
		"...######+#..",
		"..#########..",
		".###########.",
		"##.........##",
		".#..#..#....#",
		"....#........",
	}
	gohanHairBack = []string{
		".............",
		".....#..#....",
		"..#.##.##.#..",
		"..#########..",
		".####+######.",
		".###########.",
		"#.#########.#",
		"..#.#.#.#.#..",
	}
	gohanHairSide = []string{
		".............",
		"...#..#......",
		"..##.##.#....",
		".#######+....",
		"..#########..",
		"#.#########..",
		".##.......#..",
		"#.#.......#..",
	}
	vegetaHairFront = []string{
		"......#......",
		".....###.....",
		".....###.....",
		"....##+##....",
		"...###+###...",
		"...##+####...",
		"..##+######..",
		"..#########..",
		"..###...###..",
		"..##.....##..",
		"..#.......#..",
	}
	vegetaHairBack = []string{
		"......#......",
		".....###.....",
		".....###.....",
		"....##+##....",
		"...#######...",
		"...###+###...",
		"..#########..",
		"..###+#####..",
		"..#########..",
		"..#########..",
		"..#.#####.#..",
	}
	vegetaHairSide = []string{
		"....#........",
		"....##.......",
		"....###......",
		"...##+#......",
		"...##+##.....",
		"..###+###....",
		"..#######....",
		".#########...",
		".####...##...",
		".###.........",
		".##..........",
	}
)

// ---- torso ----

// dbTorso: gi, sash, emblem, armour... over the torso rows (x1 exclusive, y1 is the hip row).
func dbTorso(b *pixBuf, s *CharSpec, vw, x0, x1, y0, y1, mid int, p Pose) {
	k := dbKind(s)
	if k == dbNone {
		return
	}
	b.cur = 2
	tw := x1 - x0
	front, side, back := vw == vFront, vw == vSide, vw == vBack
	sash := func(c color.RGBA, tails bool) {
		b.rect(x0, y1-3, tw, 2, c)
		b.rect(x0, y1-3, tw, 1, shade(c, 1.2))
		if !tails {
			return
		}
		switch {
		case front:
			b.set(x0+1, y1-1, c) // the knot and its two tails
			b.set(x0+1, y1, shade(c, 0.85))
			b.set(x0, y1+1, shade(c, 0.85))
			b.set(x0+2, y1, c)
		case side:
			b.set(x0, y1-1, c)
			b.set(x0-1, y1, shade(c, 0.85))
		}
	}
	gi := func(main, under, belt color.RGBA, emblem bool) {
		dark := shade(main, 0.72)
		if front {
			// the crossed lapels of the gi and the undershirt in the V
			b.rect(mid-1, y0, 3, 1, under)
			b.set(mid, y0+1, under)
			b.set(mid-2, y0, shade(main, 1.15))
			b.set(mid+2, y0, shade(main, 1.15))
			b.set(mid-1, y0+1, dark)
			b.set(mid+1, y0+1, dark)
			b.set(mid+1, y0+2, dark)
			b.set(mid+1, y0+3, dark)
			b.set(mid, y0+4, dark)
			b.set(mid-1, y0+6, dark) // a fold under the chest
			if emblem {
				b.rect(mid+2, y0+2, 2, 2, dbWhite)
				b.set(mid+3, y0+3, dbInk)
				b.set(mid+2, y0+2, shade(dbWhite, 0.9))
			}
		}
		if side {
			b.set(x1-1, y0, under)
			b.set(x1-2, y0+1, dark)
			b.set(x1-1, y0+1, under)
			b.set(x0+1, y0+6, dark)
		}
		if back && emblem {
			// the round emblem on the back with a generic dark mark
			b.rect(mid-1, y0+1, 3, 4, dbWhite)
			b.rect(mid-2, y0+2, 5, 2, dbWhite)
			b.set(mid, y0+2, dbInk)
			b.rect(mid-1, y0+3, 3, 1, dbInk)
			b.set(mid, y0+4, shade(dbWhite, 0.85))
		}
		sash(belt, true)
	}
	switch k {
	case dbGoku:
		gi(s.Shirt, dbNavy, dbNavy, true)
	case dbKrilin:
		gi(s.Shirt, dbNavy, dbNavy, true)
	case dbGohan:
		gi(s.Shirt, s.Shirt, dbRed, false)
		if back {
			b.set(mid, y0+3, shade(s.Shirt, 0.8))
		}
	case dbVegeta:
		// white combat armour over the dark blue suit, gold flaps over the belly
		plate := func(xa, xb int) {
			for y := y0 + 1; y <= y0+6; y++ {
				for x := xa; x < xb; x++ {
					c := dbWhite
					if x == xb-1 || y == y0+6 {
						c = dbWhiteD
					}
					b.set(x, y, c)
				}
			}
		}
		switch {
		case front:
			plate(x0, x1)
			b.rect(x0+1, y0, tw-2, 1, s.Shirt) // the suit collar
			b.set(mid, y0+1, s.Shirt)
			b.rect(mid-2, y0+3, 2, 1, dbWhiteD) // pectoral lines
			b.rect(mid+1, y0+3, 2, 1, dbWhiteD)
			b.rect(x0, y1-3, 2, 3, dbGold) // belly flaps
			b.rect(x1-2, y1-3, 2, 3, dbGold)
			b.rect(x0+2, y1-3, tw-4, 1, dbWhiteD)
		case back:
			plate(x0, x1)
			b.set(mid, y0+2, dbWhiteD)
			b.set(mid, y0+4, dbWhiteD)
			b.rect(x0, y1-3, 2, 3, dbGold)
			b.rect(x1-2, y1-3, 2, 3, dbGold)
		default:
			plate(x0+1, x1)
			b.rect(x1-2, y1-3, 2, 3, dbGold)
		}
	case dbPiccolo:
		gi(s.Shirt, s.Shirt, dbNavyL, false)
		if front {
			b.set(mid, y0, s.Skin) // green chest in the V
			b.set(mid-1, y0, s.Skin)
			b.set(mid+1, y0, s.Skin)
		}
	case dbBulma:
		if front || side {
			b.rect(x0+1, y0, tw-2, 1, dbWhite) // collar
			if front {
				b.set(mid, y0+1, dbWhite)
				b.set(mid-1, y0+3, dbBulmaRad) // the radar in the pocket
			}
		}
		b.rect(x0, y1-3, tw, 1, dbWhite) // white belt
	case dbRoshi:
		shell, rim := hx(0x3a8a3a), hx(0xe8c870)
		strap := hx(0xb08a50)
		switch {
		case back:
			b.rect(x0-2, y0, tw+4, y1-y0+1, rim)
			b.rect(x0-1, y0+1, tw+2, y1-y0-1, shell)
			for _, d := range [][2]int{{0, 2}, {-2, 4}, {2, 4}, {0, 6}, {-2, 8}, {2, 8}} {
				b.rect(mid+d[0], y0+d[1], 1, 1, shade(shell, 0.62))
				b.set(mid+d[0]+1, y0+d[1], shade(shell, 1.25))
			}
		case side:
			b.rect(x0-3, y0, 3, y1-y0, rim)
			b.rect(x0-3, y0+1, 2, y1-y0-2, shell)
			b.set(x0-3, y0+4, shade(shell, 0.62))
			b.rect(x0, y0, 1, 2, strap)
		default:
			b.line(x0, y0, x0+1, y0+4, strap) // the straps of the shell
			b.line(x1-1, y0, x1-2, y0+4, strap)
			b.set(x0-1, y0+2, rim)
			b.set(x1, y0+2, rim)
		}
	case dbCell:
		// spotted green body, black wing plates wrapping the sides, pale segmented belly
		for y := y0; y <= y1; y++ {
			for x := x0; x < x1; x++ {
				if (x*5+y*3)%7 == 0 {
					b.set(x, y, dbCellSp)
				}
			}
		}
		switch {
		case front:
			// black chest plates (the wings wrap round the shoulders) over the spotted belly
			b.rect(x0, y0, tw, 3, dbCellSp)
			b.rect(x0, y0+3, 1, 3, dbCellSp)
			b.rect(x1-1, y0+3, 1, 3, dbCellSp)
			b.set(mid-2, y0+1, dbCellSk)
			b.set(mid+2, y0+1, dbCellSk)
			b.set(mid, y0, dbCellSk)
			b.set(mid, y0+1, shade(dbCellSp, 1.6))
			for y := y0 + 4; y < y1-1; y += 2 { // the segmented belly
				b.rect(mid-1, y, 3, 1, shade(dbCellSk, 0.78))
			}
		case back:
			// the folded black wings with green spots, wider than the body
			for y := y0 - 1; y <= y1-1; y++ {
				w := 1
				if y > y0+1 && y < y1-3 {
					w = 2
				}
				b.rect(x0-w, y, tw+2*w, 1, dbCellSp)
			}
			b.rect(mid, y0, 1, y1-y0-1, shade(dbCellSp, 1.8)) // the gap between the wings
			for _, d := range [][2]int{{-3, 2}, {2, 3}, {-2, 6}, {3, 7}, {-3, 9}, {1, 1}} {
				b.set(mid+d[0], y0+d[1], dbCellSk)
			}
			// the wide tail hangs down the back
			for i := 0; i < 8; i++ {
				w := 3
				if i > 5 {
					w = 1
				}
				b.rect(mid-w/2, y1-1+i, w, 1, shade(dbCellSp, 1+0.08*float64(i%2)))
			}
			b.set(mid, y1+2, dbCellSk)
		default:
			b.rect(x0, y0-1, 2, y1-y0, dbCellSp)
			b.rect(x0-1, y0+1, 1, y1-y0-3, dbCellSp)
			b.set(x0, y0+3, dbCellSk)
			b.set(x0-1, y0+6, dbCellSk)
			b.rect(x1-1, y0+4, 1, y1-y0-5, dbCellFc)
			// the tail behind the legs
			b.line(x0, y1-1, x0-4, y1+6, dbCellSp)
			b.line(x0+1, y1-1, x0-3, y1+6, dbCellSp)
			b.set(x0-4, y1+7, dbCellSp)
		}
	}
}

var dbBulmaRad = hx(0x3ac04a)

// ---- arms ----

// dbArm: wristbands, gloves and spots along the arm (sx,sy shoulder, mx,my elbow, ex,ey hand).
func dbArm(b *pixBuf, s *CharSpec, vw int, sx, sy, mx, my, ex, ey float64, armW int, dark bool, id uint8) {
	k := dbKind(s)
	if k == dbNone {
		return
	}
	b.cur = id
	d := math.Hypot(mx-ex, my-ey)
	if d < 0.01 {
		d = 0.01
	}
	ux, uy := (mx-ex)/d, (my-ey)/d
	band := func(c color.RGBA, from, to float64) {
		thickLine(b, ex+ux*from, ey+uy*from, ex+ux*to, ey+uy*to, armW, dbDk(c, dark))
	}
	hand := func(c color.RGBA) {
		b.rect(int(math.Floor(ex)), int(math.Floor(ey)), armW, 2, dbDk(c, dark))
	}
	switch k {
	case dbGoku, dbKrilin:
		band(dbNavy, 0.9, 1.9)
	case dbGohan:
		band(dbRed, 0.9, 1.6)
	case dbVegeta:
		band(dbWhiteD, 0.9, 1.6) // glove cuffs and white gloves
		hand(dbWhite)
	case dbPiccolo:
		// pink patches on the green forearm and a red wristband
		b.set(int(math.Floor(mx-ux*1.2)), int(math.Floor(my-uy*1.2)), dbDk(dbPink, dark))
		band(dbRed, 0.9, 1.6)
	case dbCell:
		b.set(int(math.Floor(mx)), int(math.Floor(my)), dbDk(dbCellSp, dark))
		b.set(int(math.Floor((sx+mx)/2)), int(math.Floor((sy+my)/2)), dbDk(dbCellSp, dark))
		band(dbCellFc, 0.9, 1.9) // pale forearms
		hand(dbCellFc)
	case dbBulma:
		band(dbWhite, 0.9, 1.4)
	}
}

// ---- legs ----

// dbLeg: boots over the shin and the shoe (mx,my knee, ex,ey ankle, sxx,syy top left of the shoe).
func dbLeg(b *pixBuf, s *CharSpec, vw int, mx, my, ex, ey float64, sxx, syy, legW int, dark bool) {
	k := dbKind(s)
	if k == dbNone {
		return
	}
	d := math.Hypot(mx-ex, my-ey)
	if d < 0.01 {
		d = 0.01
	}
	ux, uy := (mx-ex)/d, (my-ey)/d
	boot := func(c, trim, toe color.RGBA, h float64) {
		c, trim, toe = dbDk(c, dark), dbDk(trim, dark), dbDk(toe, dark)
		thickLine(b, ex, ey, ex+ux*h, ey+uy*h, legW, c)
		thickLine(b, ex+ux*h, ey+uy*h, ex+ux*(h+0.4), ey+uy*(h+0.4), legW, trim)
		if vw == vSide {
			b.rect(sxx-1, syy, legW+2, 2, c)
			b.set(sxx+legW, syy, toe)
			b.set(sxx+legW, syy+1, toe)
		} else {
			b.rect(sxx, syy, legW, 2, c)
			b.rect(sxx, syy+1, legW, 1, toe)
		}
	}
	switch k {
	case dbGoku:
		boot(dbNavy, dbGold, dbNavyL, 2.5)
	case dbKrilin:
		boot(dbNavy, dbNavyL, dbInk, 2)
	case dbGohan, dbPiccolo:
		boot(dbBrown, dbBrownL, shade(dbBrown, 0.7), 2)
	case dbVegeta:
		boot(dbWhite, dbWhiteD, dbYellow, 2.5)
	case dbCell:
		b.set(int(math.Floor(mx)), int(math.Floor(my)), dbDk(dbCellSp, dark))
		b.set(int(math.Floor(mx))+legW-1, int(math.Floor(my))-3, dbDk(dbCellSp, dark)) // spots on the thigh
		boot(dbCellFc, dbCellSp, dbCellSp, 2)
	}
}

// ---- head ----

// dbHead: hair, faces, antennae and crests at the end of head().
func dbHead(b *pixBuf, s *CharSpec, vw int, p Pose, hxp, hyp, rx, ry float64, ey int) {
	k := dbKind(s)
	if k == dbNone {
		return
	}
	b.cur = 1
	ihx := int(math.Floor(hxp))
	top := int(math.Floor(hyp - ry))
	ex := int(math.Floor(hxp + rx*0.45))
	front, side, back := vw == vFront, vw == vSide, vw == vBack
	moving := p.Anim != AnimIdle && !p.Sit
	lift := 0
	if moving && p.F%2 == 1 {
		lift = -1 // the spikes bounce with the steps
	}
	hairPal := func(h color.RGBA) map[byte]color.RGBA {
		return map[byte]color.RGBA{'#': h, '+': shade(h, 2.4)}
	}
	stamp := func(fr, bk, sd []string, dy int, h color.RGBA) {
		pal := hairPal(h)
		if h == dbHair {
			pal['+'] = dbHairHi
		}
		switch vw {
		case vFront:
			dbStamp(b, ihx-6, top+dy+lift, fr, pal)
		case vBack:
			dbStamp(b, ihx-6, top+dy+lift, bk, pal)
		default:
			dbStamp(b, ihx-7, top+dy+lift, sd, pal)
		}
	}
	brows := func(c color.RGBA) { // determined brows over the eyes
		if front {
			b.set(ihx-1, ey-1, c)
			b.set(ihx+1, ey-1, c)
			b.set(ihx-1, ey, dbInk)
			b.set(ihx+1, ey, dbInk)
		}
		if side {
			b.set(ex, ey-1, c)
			b.set(ex, ey, dbInk)
		}
	}
	switch k {
	case dbGoku:
		stamp(gokuHairFront, gokuHairBack, gokuHairSide, -5, s.Hair)
		brows(dbHair)
		if front && !p.Open {
			b.set(ihx, ey+2, shade(s.Skin, 0.6))
		}
	case dbGohan:
		stamp(gohanHairFront, gohanHairBack, gohanHairSide, -4, s.Hair)
		if front {
			b.set(ihx-2, ey-1, dbHair)
			b.set(ihx+2, ey-1, dbHair)
		}
	case dbVegeta:
		stamp(vegetaHairFront, vegetaHairBack, vegetaHairSide, -8, s.Hair)
		if front {
			// widow's peak: the temples stay bare, a point comes down in the middle
			b.set(ihx-2, top+2, s.Skin)
			b.set(ihx+2, top+2, s.Skin)
			b.set(ihx-1, top+2, s.Skin)
			b.set(ihx+1, top+2, s.Skin)
			b.set(ihx, top+2, s.Hair)
			// a frown over the eyes and a scowl
			b.set(ihx-1, ey-1, shade(s.Skin, 0.6))
			b.set(ihx+1, ey-1, shade(s.Skin, 0.6))
			b.set(ihx-1, ey, dbInk)
			b.set(ihx+1, ey, dbInk)
			b.rect(ihx-1, ey+2, 3, 1, shade(s.Skin, 0.7))
		}
		if side {
			b.set(ex+1, top+2, s.Skin)
			b.set(ex, top+2, s.Skin)
			b.set(ex, ey-1, dbInk)
			b.set(ex+1, ey-1, dbInk)
		}
	case dbKrilin:
		dot := hx(0xa86a48) // six small marks on the forehead
		if front {
			for _, dx := range []int{-2, 0, 2} {
				b.set(ihx+dx, ey-4, dot)
				b.set(ihx+dx, ey-2, dot)
			}
		}
		if side {
			b.set(ex-1, ey-4, dot)
			b.set(ex+1, ey-4, dot)
			b.set(ex, ey-2, dot)
			b.set(ex+2, ey-2, dot)
		}
		if back {
			b.set(ihx, top+2, shade(s.Skin, 1.15))
		}
	case dbPiccolo:
		tb := dbWhite
		td := dbWhiteD
		ears := func() {
			if side {
				b.set(ihx-1, ey, s.Skin)
				b.set(ihx-2, ey-1, s.Skin)
				b.set(ihx-3, ey-2, shade(s.Skin, 0.8))
				return
			}
			b.set(ihx-3, ey, s.Skin)
			b.set(ihx-4, ey-1, s.Skin)
			b.set(ihx-5, ey-2, shade(s.Skin, 0.8))
			b.set(ihx+3, ey, s.Skin)
			b.set(ihx+4, ey-1, s.Skin)
			b.set(ihx+5, ey-2, shade(s.Skin, 0.8))
		}
		// the generic turban hangs too low: give the face back
		for y := top + 2; y <= ey+3; y++ {
			for x := ihx - 3; x <= ihx+3; x++ {
				if b.get(x, y) == s.HatCol {
					b.set(x, y, s.Skin)
				}
			}
		}
		ears()
		// the turban: a white dome with a fold, sitting high on the head
		switch vw {
		case vSide:
			b.rect(ihx-3, top-2, 5, 4, tb)
			b.rect(ihx-2, top-3, 4, 1, tb)
			b.rect(ihx-3, top+1, 6, 1, td)
			b.rect(ihx-1, top-4, 2, 1, tb)
			b.set(ihx+1, top+2, s.Skin)
			b.line(ihx+2, top+1, ihx+3, top-1, dbGreenD) // antenna
		default:
			b.rect(ihx-3, top-2, 7, 4, tb)
			b.rect(ihx-2, top-3, 5, 1, tb)
			b.rect(ihx-1, top-4, 3, 1, tb)
			b.rect(ihx-3, top+1, 7, 1, td)
			if front {
				b.rect(ihx-1, top-1, 3, 2, hx(0xe8e0f0))
				b.set(ihx, top-1, dbPurpleL) // a small purple badge
				// antennae from the brow up into the turban
				b.set(ihx-1, top+2, dbGreenD)
				b.set(ihx+1, top+2, dbGreenD)
				b.set(ihx-2, ey-1, dbInk)
				b.set(ihx+2, ey-1, dbInk)
				b.set(ihx-1, ey, dbInk)
				b.set(ihx+1, ey, dbInk)
			} else {
				b.set(ihx, top-1, td)
			}
		}
	case dbBulma:
		if !back {
			b.set(ihx+2, top, dbRed) // a red bow in the hair
			b.set(ihx+3, top-1, dbRed)
			b.set(ihx+3, top+1, dbRed)
		} else {
			b.set(ihx, top+1, dbRed)
			b.set(ihx-1, top, dbRed)
			b.set(ihx+1, top, dbRed)
		}
	case dbRoshi:
		beard, bd := dbWhite, dbWhiteD
		switch vw {
		case vFront:
			b.set(ihx-3, ey-1, beard) // white tufts over the ears
			b.set(ihx+3, ey-1, beard)
			b.rect(ihx-2, ey+1, 5, 2, beard)
			b.rect(ihx-1, ey+3, 3, 2, beard)
			b.set(ihx, ey+5, beard)
			b.set(ihx, ey+1, bd)
			b.rect(ihx-1, ey+1, 3, 1, bd) // moustache
			b.set(ihx-2, ey-1, beard)     // white eyebrows over the shades
			b.set(ihx+2, ey-1, beard)
			b.set(ihx, top+2, shade(s.Skin, 1.2)) // shiny head
		case vSide:
			b.set(ihx-2, ey-1, beard)
			b.rect(ihx, ey+1, 3, 2, beard)
			b.rect(ihx, ey+3, 2, 2, beard)
			b.set(ihx+1, ey+5, bd)
			b.set(ihx, top+1, shade(s.Skin, 1.2))
		default:
			b.set(ihx-3, ey-1, beard)
			b.set(ihx+3, ey-1, beard)
			b.rect(ihx-2, ey, 5, 1, beard)
		}
	case dbCell:
		// pale face with purple marks, a wide black mouth, and the pointed crest
		crest := map[byte]color.RGBA{'#': dbCellSk, 'x': dbCellSp, 'm': dbPurple, 'd': dbGreenD}
		switch vw {
		case vFront:
			b.rect(ihx-2, ey-1, 5, 4, dbCellFc)
			b.set(ihx-2, ey+2, shade(dbCellFc, 0.85))
			b.set(ihx+2, ey+2, shade(dbCellFc, 0.85))
			b.set(ihx-2, ey, dbCellMg)
			b.set(ihx+2, ey, dbCellMg)
			b.set(ihx-1, ey, dbCellMg) // eyes
			b.set(ihx+1, ey, dbCellMg)
			b.set(ihx-1, ey-1, dbInk)
			b.set(ihx+1, ey-1, dbInk)
			b.rect(ihx-1, ey+2, 3, 1, dbInk) // the wide mouth
			if p.Open {
				b.set(ihx, ey+3, hx(0x6a1a2a))
			}
			dbStamp(b, ihx-4, top-5, []string{
				"x.......x",
				"#x.....x#",
				"#mx...xm#",
				".#mmxmm#.",
				".##mmm##.",
				"..#d#d#..",
				"..x###x..",
			}, crest)
		case vBack:
			dbStamp(b, ihx-4, top-5, []string{
				"x.......x",
				"#x.....x#",
				"##x...x##",
				".##xxx##.",
				".##x#x##.",
				"..#####..",
				"..#x#x#..",
				"..#####..",
			}, crest)
		default:
			b.rect(ihx, ey-1, 3, 4, dbCellFc)
			b.set(ex, ey, dbCellMg)
			b.set(ex, ey-1, dbInk)
			b.rect(ihx+1, ey+2, 2, 1, dbInk)
			b.set(ihx, ey+1, dbCellMg)
			dbStamp(b, ihx-6, top-5, []string{
				"x........",
				"#x.......",
				".#mx.....",
				"..#mmx...",
				"...##m##.",
				"....#x##.",
				"....x##..",
			}, crest)
		}
	}
}

// ---- over everything: shoulder pads and capes ----

// dbOver runs after the whole body is drawn (cx: centre column of the torso, ty0: torso top).
func dbOver(b *pixBuf, s *CharSpec, vw int, p Pose, cx, hipY, ty0 float64, tw int) {
	k := dbKind(s)
	if k != dbPiccolo && k != dbVegeta {
		return
	}
	b.cur = 2
	x0 := int(math.Floor(cx - float64(tw)/2))
	x1 := x0 + tw
	y0 := int(math.Floor(ty0))
	y1 := int(math.Floor(hipY))
	switch k {
	case dbVegeta:
		pad := func(x int) {
			b.rect(x, y0, 2, 2, dbGold)
			b.set(x, y0+1, dbYellowD)
			b.set(x+1, y0+1, dbYellowD)
		}
		if vw == vSide {
			b.rect(x0+tw/2-1, y0, 3, 2, dbGold)
			b.rect(x0+tw/2-1, y0+1, 3, 1, dbYellowD)
			return
		}
		pad(x0 - 2)
		pad(x1)
	case dbPiccolo:
		cape, capeD, capeL := dbWhite, dbWhiteD, hx(0xffffff)
		foot := 40
		switch vw {
		case vBack:
			// the cape covers the back and the arms, wide at the shoulders, down to the ankles
			for y := y0 - 1; y <= foot; y++ {
				grow := (y - y0) / 6
				xa, xb := x0-3-grow, x1+3+grow
				if y < y0+1 {
					xa, xb = x0-3, x1+3
				}
				for x := xa; x < xb; x++ {
					c := cape
					if (x-xa)%3 == 2 && y > y0+3 {
						c = capeD // folds
					}
					if x == xb-1 || y == foot {
						c = capeD
					}
					b.set(x, y, c)
				}
			}
			b.rect(x0-4, y0-1, tw+8, 2, capeL) // the shoulder pads
			b.rect(x0-4, y0+1, tw+8, 1, capeD)
		case vSide:
			for y := y0; y <= foot; y++ {
				grow := (y - y0) / 5
				b.rect(x0-2-grow, y, 2, 1, capeD)
				b.set(x0-grow, y, cape)
			}
			b.rect(x0-2, y0-1, tw+3, 3, cape)
			b.rect(x0-2, y0+1, tw+3, 1, capeD)
		default:
			// the cape falls behind the body, seen at both sides, flaring down to the ankles
			for y := y0 + 1; y <= foot; y++ {
				g := (y - y0) / 6
				for x := x0 - 3 - g; x <= x0-1; x++ {
					if c := b.get(x, y); c.A == 0 || c == s.Cape {
						col := capeD
						if x == x0-3-g {
							col = cape
						}
						b.set(x, y, col)
					}
				}
				for x := x1; x <= x1+2+g; x++ {
					if c := b.get(x, y); c.A == 0 || c == s.Cape {
						col := shade(capeD, 0.92)
						if x == x1+2+g {
							col = capeD
						}
						b.set(x, y, col)
					}
				}
			}
			// very wide rounded shoulder pads over the arms
			b.rect(x0-4, y0-1, 4, 3, cape)
			b.rect(x1, y0-1, 4, 3, cape)
			b.rect(x0-3, y0-2, 2, 1, cape)
			b.rect(x1+1, y0-2, 2, 1, cape)
			b.rect(x0-4, y0+1, 4, 1, capeD)
			b.rect(x1, y0+1, 4, 1, capeD)
			b.set(x0-3, y0-1, capeL)
			b.set(x1+1, y0-1, capeL)
			b.set(x0-1, y0-1, cape) // the collar around the neck
			b.set(x1, y0-1, cape)
		}
	}
	_ = y1
}

// ---- the new guest: Cell ----

// presetCell: the perfect bio-android (new famous guest, comes with the Dragon Ball group).
// It takes nothing from r, so adding him does not change the random looks of the guests listed after him.
func presetCell(r *rand.Rand) CharSpec {
	s := CharSpec{Name: "cell", Accent: dbCellSp, HatCol: dbCellSk}
	s.Skin, s.Hair = dbCellSk, dbCellSk
	s.Top, s.Shirt, s.Shirt2 = TopTank, dbCellSk, dbCellSp
	s.Bottom, s.Pants, s.Shoes = BotPants, dbCellSk, dbCellFc
	s.Hair2, s.Body, s.Stature = HairBald, BodyBig, -2
	s.Glasses, s.Beard, s.Hat, s.Carry, s.Weapon = GlassNone, BeardNone, HatNone, HoldNone, HoldNone
	s.Pattern, s.Face, s.Eyes, s.Ears, s.Head = 0, 0, 0, 0, HeadNormal
	return s
}

func init() {
	famousPresets = append(famousPresets, presetCell)
	displayNameOverrides["cell"] = "Sell"
	for i, g := range starGroups {
		if len(g) > 0 && g[0] == "goku" {
			starGroups[i] = append(append([]string(nil), g...), "cell")
		}
	}
	// walking styles (drawing only, gait.go): the proud prince marches, the Namekian and Cell glide
	for slug, st := range map[string]string{"vegeta": "march", "piccolo": "glide", "cell": "glide"} {
		if _, ok := gaitSlugs[slug]; !ok {
			gaitSlugs[slug] = st
		}
	}
	charDescs["cell"] = [2]string{"Androide perfecto que presume de cola", "Perfect android who shows off his tail"}
	if _, ok := langDataEN["Androide perfecto que presume de cola"]; !ok {
		langDataEN["Androide perfecto que presume de cola"] = "Perfect android who shows off his tail"
	}
}
