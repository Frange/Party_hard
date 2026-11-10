package main

import (
	"math"
	"math/rand"
)

// ---------------------------------------------------------------------------
// TABLES WITH A CLOTH AND CHAIRS: decorative tables (the chairs are the game's own "chair" prop, items.go) for a few indoor parties (home, casino, birthday, 70s disco). The
// guests can hide under the tables when there is danger (pe_tablehide.go).
//
// They are not in levels.go or in the generators: buildLevel (vip.go) calls Level.addTables() once the level is built,
// like the other ensure* pieces, with its OWN dice seeded from the level (never g.rng, and the same party always gets the
// same sets). Every set is checked against what is already there (propClashes: props, walls, ground), against the dance
// floor, the VIP carpet and the start, and against the walking map: after adding it every cell that could be reached
// before and is not covered by the set must still be reachable (a set that would close a corridor is undone).
// The sets are drawn 1-3 per party: a table, two or three chairs, and two free places in front of it for the guests.
// ---------------------------------------------------------------------------

// tablesIcons: how many sets each venue gets (the venue icon of the campaign).
var tablesIcons = map[string]int{"house": 1, "casino": 2, "cake": 2, "disco70": 1}

// tablesOff: tests switch the sets off to look at a bare level.
var tablesOff bool

const tableHideDX = 7.0 // the two places in front of the table

func init() {
	pk := propKinds
	pk["clothtable"] = &PropKind{W: 40, H: 30, FW: 34, FH: 10, Vars: 3, Category: "furniture", Draw: func(b *pixBuf, v int) {
		cols := []uint32{0xe4e6f0, 0xc0394a, 0x3a78c0}
		base := c(cols[v%3])
		b.ellipse(20, 9, 19, 8, shade(base, 0.7))
		b.ellipse(20, 8, 18, 7, shade(base, 1.08))
		b.ellipse(20, 8, 12, 4.4, base)
		// the cloth falls straight down in front and at the sides, with a scalloped hem
		b.rect(2, 9, 36, 17, shade(base, 0.86))
		b.rect(2, 9, 3, 17, shade(base, 0.7))
		b.rect(35, 9, 3, 17, shade(base, 0.66))
		for x := 6; x < 34; x += 7 {
			b.rect(x, 10, 1, 15, shade(base, 0.74)) // folds
		}
		for x := 2; x < 38; x += 6 {
			b.rect(x, 26, 5, 2, shade(base, 0.8))
			b.rect(x+1, 28, 3, 1, shade(base, 0.62))
		}
		b.rect(2, 24, 36, 1, shade(base, 0.7))
		// on top: a candle, a glass and a little vase
		b.rect(10, 3, 2, 4, c(0xf4f0d8))
		b.rect(10, 1, 2, 2, c(0xffc840))
		b.rect(18, 4, 3, 4, c(0xd8e8f4))
		b.rect(27, 3, 3, 5, c(0x58a858))
		b.rect(28, 1, 1, 2, c(0xe85078))
	}}
}

// tablesSeed: the same level always gets the same sets.
func tablesSeed(d *LevelDef) int64 {
	h := int64(7919)
	for _, ch := range d.Title {
		h = h*31 + int64(ch)
	}
	return h*17 + int64(d.W)*13 + int64(d.H)*7 + int64(len(d.Props))
}

type tableSet struct {
	props []*Prop
}

// addTables puts the sets in a built level (called by buildLevel). It returns how many it placed.
func (l *Level) addTables() int {
	d := l.Def
	if d == nil || d.Title == "" || tablesOff || vipBuilding {
		return 0
	}
	n := tablesIcons[d.Icon]
	if n == 0 {
		return 0
	}
	tk, ck := propKinds["clothtable"], propKinds["chair"]
	fitAllFootprints()
	r := rand.New(rand.NewSource(tablesSeed(d)))
	// candidates: a grid over the roaming area (or the whole floor), in a fixed shuffled order
	var cand []Vec
	rects := d.Roam
	if len(rects) == 0 {
		rects = []Rect{{50, 80, l.W - 100, l.H - 120}}
	}
	for _, rc := range rects {
		for y := rc.Y + 14; y < rc.Y+rc.H; y += 20 {
			for x := rc.X + 20; x < rc.X+rc.W; x += 24 {
				cand = append(cand, Vec{x, y})
			}
		}
	}
	r.Shuffle(len(cand), func(i, j int) { cand[i], cand[j] = cand[j], cand[i] })
	reach0 := l.reachFrom(d.Start)
	var placed []Vec
	tries := 0
	for pass := 0; pass < 2; pass++ { // first the covered floors (a lounge), then anywhere
		for _, p := range cand {
			if len(placed) >= n || tries > 90 {
				break
			}
			if pass == 0 && l.Outdoor(p) {
				continue
			}
			if !l.tableSpotOK(p, tk, placed) {
				continue
			}
			tries++
			if l.tryTableSet(p, r, tk, ck, reach0) {
				placed = append(placed, p)
			}
		}
	}
	return len(placed)
}

// tableSpotOK: the quick checks of a spot, before building anything.
func (l *Level) tableSpotOK(p Vec, tk *PropKind, others []Vec) bool {
	d := l.Def
	if p.X < 70 || p.X > l.W-70 || p.Y < 100 || p.Y > l.H-50 || dist(p, d.Start) < 70 {
		return false
	}
	for _, o := range others {
		if dist(o, p) < 110 {
			return false
		}
	}
	// the whole footprint, the chairs around it and the two places in front must be open floor
	for _, o := range []Vec{{0, 0}, {-16, 0}, {16, 0}, {-16, -9}, {16, -9}, {0, -9}, {-28, 2}, {28, 2}, {-tableHideDX, 9}, {tableHideDX, 9}, {0, 14}, {0, -18}} {
		q := Vec{p.X + o.X, p.Y + o.Y}
		if !l.Free(q) || l.badGround(q) || l.InPool(q) {
			return false
		}
	}
	for _, dr := range d.Dance {
		if dr.Grow(18).Has(p.X, p.Y) {
			return false
		}
	}
	if l.vip != nil {
		z := l.vip
		for _, rc := range []Rect{z.in, z.trig, z.gate} {
			if rc.Grow(26).Has(p.X, p.Y) {
				return false
			}
		}
	}
	return !l.propClashes("clothtable", tk, p, 0)
}

func (l *Level) addTableProp(name string, k *PropKind, p Vec, v int) *Prop {
	pr := &Prop{Name: name, K: k, X: p.X, Y: p.Y, Var: v, Foot: Rect{p.X - k.FW/2, p.Y - k.FH, k.FW, k.FH}, Sort: p.Y + k.SortOff}
	if k.hasFoot {
		pr.Foot = Rect{p.X + k.FootDX, p.Y + k.FootDY, k.FW, k.FH}
	}
	l.Props = append(l.Props, pr)
	if !k.NoSolid {
		l.solidsP = append(l.solidsP, pr.Foot)
		l.solidsN = append(l.solidsN, pr.Foot)
	}
	return pr
}

// tryTableSet builds the table and its chairs at p; false (and nothing changed) if it closes a way or covers a place.
func (l *Level) tryTableSet(p Vec, r *rand.Rand, tk, ck *PropKind, reach0 []bool) bool {
	nProps, nP, nN := len(l.Props), len(l.solidsP), len(l.solidsN)
	undo := func() {
		l.Props, l.solidsP, l.solidsN = l.Props[:nProps], l.solidsP[:nP], l.solidsN[:nN]
		l.initNav()
	}
	l.addTableProp("clothtable", tk, p, r.Intn(3))
	chairs := 0
	for i, o := range []Vec{{-27, 3}, {27, 3}, {0, -17}} {
		q := Vec{p.X + o.X, p.Y + o.Y}
		if i == 2 && chairs == 2 && r.Float64() < 0.5 {
			continue
		}
		if !l.Free(q) || l.badGround(q) || l.InPool(q) || l.propClashes("chair", ck, q, 0) {
			continue
		}
		l.addTableProp("chair", ck, q, r.Intn(2))
		chairs++
	}
	if chairs < 2 {
		undo()
		return false
	}
	l.initNav()
	slots := tableSlots(l, p)
	reach1 := l.reachFrom(l.Def.Start)
	for _, s := range slots {
		if s == (Vec{}) || !l.Free(s) || !reach1[int(s.Y/navCell)*l.cw+int(s.X/navCell)] {
			undo()
			return false
		}
	}
	if len(reach0) == len(reach1) {
		for i := range reach0 {
			if reach0[i] && !reach1[i] && !l.blocked[i] { // a floor cell that was walkable and now is cut off
				undo()
				return false
			}
		}
	}
	return true
}

// tableSlots: the two places in front of the table where a guest can stand (Vec{} = none).
func tableSlots(l *Level, p Vec) [2]Vec {
	var out [2]Vec
	for i, dx := range []float64{-tableHideDX, tableHideDX} {
		want := Vec{p.X + dx, p.Y + 9}
		got := l.NearestFree(want)
		if math.Abs(got.X-want.X) <= 4 && math.Abs(got.Y-want.Y) <= 4 {
			out[i] = got
		}
	}
	return out
}
