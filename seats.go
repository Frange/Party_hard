package main

import (
	"flag"
	"image"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// SITTING DOWN: chairs, tables, benches, deckchairs and poufs where the guests rest for a while.
//
// PLACING (Level.addSeats, called by buildLevel after the cloth tables): every party gets a few "kits" in the style of
// its theme (seatThemes: a café table with three chairs, a long table with a cloth and four chairs, a park bench, two
// deckchairs, two poufs, a row of seats). Like the cloth tables they are not written in levels.go or the generators:
// they are placed with the level's OWN dice (the same party always gets the same kits), only on open floor (every
// navigation cell around the kit must be free: no corridors, no doors), away from the dance floor, the VIP carpet,
// the start, the performers and every place of use (bars, toilets, phones), and a kit that would cut a way or leave a
// seat out of reach is undone. Random-mode levels get them too (by title, or a plain default).
//
// SITTING (seatPick from guestPick, updateSeats every frame): now and then (12 % of the choices, with a cooldown of
// 40-90 s per guest) an idle guest books a free seat (nobody else may take it), walks there, sits down (a short
// glide onto the seat), looks into the table or the scene, and for 15-40 s rests, talks (bubbles when somebody else
// sits at the same table), drinks or looks at the phone; then he stands up. He stays an ordinary guest: gu.pos is the
// walkable spot next to the seat (so he can be killed, he witnesses, and any alarm makes him stand up and run: his
// state stops being gLounge and the seat is freed). If he dies there his body lies beside the seat. A guest who
// never arrives (30 s) or sits too long (45 s) is stood up.
// ---------------------------------------------------------------------------

const (
	kitCafe  = iota // round table, two side chairs and one behind
	kitLong         // long table with a cloth (hiding place), four chairs
	kitBench        // a bench for two
	kitDeck         // two deckchairs
	kitPouf         // two poufs
	kitRow          // a row of three seats
	kitN
)

type seatTheme struct {
	mat  int
	kits [kitN]int
	edge bool // benches may stand along the edge of the pitch / the sand of the ring (the dugouts)
}

// seatThemes: the kits of every campaign party (by its icon). Cafés, gardens, parks, stadium, beach and ships get the
// most; clubs get poufs; parties full of furniture already (home, casino, birthday) only a few.
var seatThemes = map[string]seatTheme{
	"house":    {0, [kitN]int{1, 0, 0, 0, 1, 0}, false},
	"beach":    {2, [kitN]int{2, 0, 1, 2, 0, 0}, false},
	"neon":     {5, [kitN]int{1, 0, 0, 0, 2, 0}, false},
	"bull":     {0, [kitN]int{1, 0, 2, 0, 0, 0}, true},
	"eiffel":   {1, [kitN]int{3, 0, 1, 0, 0, 0}, false},
	"boat":     {2, [kitN]int{1, 0, 0, 2, 0, 0}, false},
	"funfair":  {2, [kitN]int{2, 1, 2, 0, 0, 0}, false},
	"stadium":  {2, [kitN]int{1, 0, 3, 0, 0, 0}, true},
	"disco":    {5, [kitN]int{1, 0, 0, 0, 2, 0}, false},
	"festival": {2, [kitN]int{1, 2, 1, 0, 0, 0}, false},
	"skate":    {2, [kitN]int{0, 0, 2, 0, 0, 0}, false},
	"grave":    {0, [kitN]int{0, 0, 2, 0, 0, 0}, false},
	"cinema":   {5, [kitN]int{0, 0, 0, 0, 1, 1}, false},
	"circuit":  {2, [kitN]int{1, 0, 2, 0, 0, 0}, false},
	"snow":     {6, [kitN]int{2, 1, 2, 0, 0, 0}, false},
	"roof":     {1, [kitN]int{2, 0, 0, 1, 0, 0}, false},
	"casino":   {5, [kitN]int{1, 0, 0, 0, 1, 0}, false},
	"highway":  {2, [kitN]int{1, 0, 1, 0, 0, 0}, false},
	"italy":    {1, [kitN]int{3, 1, 1, 0, 0, 0}, false},
	"train":    {0, [kitN]int{1, 0, 2, 0, 0, 1}, false},
	"cake":     {0, [kitN]int{1, 1, 0, 0, 1, 0}, false},
	"hockey":   {6, [kitN]int{1, 0, 2, 0, 0, 0}, false},
	"disco70":  {5, [kitN]int{1, 0, 0, 0, 2, 0}, false},
	"pirate":   {4, [kitN]int{0, 1, 2, 1, 0, 0}, false},
	"station":  {3, [kitN]int{2, 0, 0, 0, 0, 2}, false},
}

// seatDefault: random-mode levels whose theme is not known.
var seatDefault = seatTheme{0, [kitN]int{1, 0, 1, 0, 0, 0}, false}

// seatsOff: tests switch the kits off to look at a bare level.
var seatsOff bool

var seatTitleIcon map[string]string

// seatThemeOf: the kits of a level (false = none: the island keeps its own scaled props).
func seatThemeOf(d *LevelDef) (seatTheme, bool) {
	if d == nil || d.Title == "" || (d.PropScale != 0 && d.PropScale != 1) {
		return seatTheme{}, false
	}
	icon := d.Icon
	if icon == "" {
		if seatTitleIcon == nil {
			seatTitleIcon = map[string]string{}
			for _, e := range campaignLevels {
				if e.title != "" {
					seatTitleIcon[e.title] = e.icon
				}
			}
		}
		icon = seatTitleIcon[d.Title]
	}
	if th, ok := seatThemes[icon]; ok {
		return th, true
	}
	return seatDefault, true
}

func seatsSeed(d *LevelDef) int64 {
	h := int64(104729)
	for _, ch := range d.Title {
		h = h*37 + int64(ch)
	}
	return h*11 + int64(d.W)*7 + int64(d.H)*5 + int64(len(d.Props))*3
}

type seatPiece struct {
	name string
	v    int
	at   Vec
}

// seatKitPieces: the furniture of a kit standing at p.
func seatKitPieces(kit, mat int, p Vec, r *rand.Rand) []seatPiece {
	switch kit {
	case kitCafe:
		return []seatPiece{{"seattable", mat, p}, {"seatchair", mat * 2, Vec{p.X - 17, p.Y + 1}},
			{"seatchair", mat*2 + 1, Vec{p.X + 17, p.Y + 1}}, {"seatfront", mat, Vec{p.X, p.Y - 8}}}
	case kitLong:
		return []seatPiece{{"seatlong", mat, p}, {"seatfront", mat, Vec{p.X - 14, p.Y - 10}}, {"seatfront", mat, Vec{p.X + 14, p.Y - 10}},
			{"seatchair", mat * 2, Vec{p.X - 37, p.Y + 1}}, {"seatchair", mat*2 + 1, Vec{p.X + 37, p.Y + 1}}}
	case kitBench:
		return []seatPiece{{"seatbench", mat, p}}
	case kitDeck:
		v := r.Intn(3)
		return []seatPiece{{"seatdeck", v, Vec{p.X - 11, p.Y}}, {"seatdeck", v, Vec{p.X + 11, p.Y}}}
	case kitPouf:
		return []seatPiece{{"seatpouf", r.Intn(4), Vec{p.X - 10, p.Y}}, {"seatpouf", r.Intn(4), Vec{p.X + 10, p.Y}}}
	case kitRow:
		return []seatPiece{{"seatfront", mat, Vec{p.X - 16, p.Y}}, {"seatfront", mat, p}, {"seatfront", mat, Vec{p.X + 16, p.Y}}}
	}
	return nil
}

// seatRel: one place to sit on a piece: where the sitter's feet are drawn (from the anchor) and where he looks.
type seatRel struct {
	dx, dy float64
	face   Dir
}

// seatRelsOf: the places of a kind of furniture (the old lounge furniture of the levels counts too).
func seatRelsOf(name string, v int) []seatRel {
	down := DirDown
	switch name {
	case "seatchair":
		if v%2 == 0 {
			return []seatRel{{-1, 0, DirRight}}
		}
		return []seatRel{{1, 0, DirLeft}}
	case "seatfront", "seatdeck", "chair", "armchair":
		return []seatRel{{0, 0, down}}
	case "seatpouf":
		return []seatRel{{0, 3, down}}
	case "stool":
		return []seatRel{{0, -4, down}}
	case "seatbench":
		return []seatRel{{-8, 0, down}, {8, 0, down}}
	case "bench":
		return []seatRel{{-9, 0, down}, {9, 0, down}}
	case "couch":
		return []seatRel{{-12, 0, down}, {12, 0, down}}
	}
	return nil
}

// seatTableKinds: the tables people gather round (their seats make one group).
var seatTableKinds = map[string]bool{"seattable": true, "seatlong": true, "clothtable": true}

// addSeats puts the kits in a built level (called by buildLevel). It returns how many it placed.
func (l *Level) addSeats() int {
	d := l.Def
	if seatsOff || vipBuilding {
		return 0
	}
	th, ok := seatThemeOf(d)
	if !ok {
		return 0
	}
	total := 0
	for _, n := range th.kits {
		total += n
	}
	if total == 0 || l.cw == 0 {
		return 0
	}
	fitAllFootprints()
	r := rand.New(rand.NewSource(seatsSeed(d)))
	var cand []Vec
	rects := d.Roam
	if len(rects) == 0 {
		rects = []Rect{{50, 80, l.W - 100, l.H - 120}}
	}
	for _, rc := range rects {
		for y := rc.Y + 12; y < rc.Y+rc.H; y += 16 {
			for x := rc.X + 16; x < rc.X+rc.W; x += 20 {
				cand = append(cand, Vec{x, y})
			}
		}
	}
	r.Shuffle(len(cand), func(i, j int) { cand[i], cand[j] = cand[j], cand[i] })
	// the kits in a fixed order: the big ones first (they need the most room)
	var order []int
	for _, k := range []int{kitLong, kitCafe, kitRow, kitBench, kitDeck, kitPouf} {
		for i := 0; i < th.kits[k]; i++ {
			order = append(order, k)
		}
	}
	reach0 := l.reachFrom(l.NearestFree(d.Start))
	var avoid []Vec // every place of use: bars, toilets, phones, lounges...
	for _, ps := range l.POIs {
		avoid = append(avoid, ps...)
	}
	for _, pf := range d.Performers {
		avoid = append(avoid, pf.Pos)
	}
	var placed []Vec
	n := 0
	for _, kit := range order {
		tries := 0
		for ci, p := range cand {
			if tries > 40 {
				break
			}
			if p.X < 0 { // used up
				continue
			}
			pieces := seatKitPieces(kit, th.mat, p, r)
			if !l.seatKitSpotOK(p, pieces, placed, avoid, th.edge && (kit == kitBench || kit == kitRow)) {
				continue
			}
			tries++
			if l.trySeatKit(pieces, reach0) {
				placed = append(placed, p)
				cand[ci].X = -1
				n++
				break
			}
		}
	}
	if n > 0 {
		l.initNav()
	}
	return n
}

// seatPieceRect: the drawn box of a piece (sprite rectangle) in the world.
func seatPieceRect(pc seatPiece) Rect {
	k := propKinds[pc.name]
	return Rect{pc.at.X - float64(k.W)/2, pc.at.Y - float64(k.H), float64(k.W), float64(k.H)}
}

// seatKitSpotOK: the quick checks of a spot, before building anything.
func (l *Level) seatKitSpotOK(p Vec, pieces []seatPiece, others, avoid []Vec, edge bool) bool {
	d := l.Def
	if dist(p, d.Start) < 80 {
		return false
	}
	for _, o := range others {
		if dist(o, p) < 90 {
			return false
		}
	}
	var box Rect
	for i, pc := range pieces {
		rc := seatPieceRect(pc)
		if i == 0 {
			box = rc
		} else {
			box = unionRect(box, rc)
		}
		if pc.at.X < 30 || pc.at.X > l.W-30 || pc.at.Y < 40 || pc.at.Y > l.H-20 {
			return false
		}
		if l.seatBadGround(pc.at, edge) || l.InPool(pc.at) || l.propClashes(pc.name, propKinds[pc.name], pc.at, 0) {
			return false
		}
		for _, dr := range d.Dance {
			if dr.Grow(20).Has(pc.at.X, pc.at.Y) {
				return false
			}
		}
		if l.vip != nil {
			for _, rc := range []Rect{l.vip.in, l.vip.trig, l.vip.gate} {
				if rc.Grow(26).Has(pc.at.X, pc.at.Y) {
					return false
				}
			}
		}
	}
	// open floor all round: no corridor, door or narrow pass (the bottom rows, where the feet are, with a margin)
	area := Rect{box.X - 14, box.Y + box.H - 22, box.W + 28, 22 + 18}
	for y := area.Y; y <= area.Y+area.H; y += navCell / 2 {
		for x := area.X; x <= area.X+area.W; x += navCell / 2 {
			q := Vec{x, y}
			if !l.Free(q) || !edge && l.badGround(q) || l.InPool(q) { // (the middle of a pitch is fine as open floor)
				return false
			}
		}
	}
	for _, a := range avoid {
		if box.Grow(26).Has(a.X, a.Y) {
			return false
		}
	}
	return true
}

// seatBadGround is badGround, except that with edge a pitch or a bullring sand is good ground within 40 px of its
// border (the benches of the substitutes, the seats round the ring); the middle stays clear.
func (l *Level) seatBadGround(p Vec, edge bool) bool {
	if !edge {
		return l.badGround(p)
	}
	d := l.Def
	if d.Highway && d.Road.Has(p.X, p.Y) {
		return true
	}
	for _, f := range d.Floors {
		if !spotBadFloors[f.Kind] || !f.R.Grow(6).Has(p.X, p.Y) {
			continue
		}
		if f.Kind != "pitch" && f.Kind != "arena" {
			return true
		}
		if f.R.Grow(-40).Has(p.X, p.Y) {
			return true
		}
	}
	for _, pd := range d.Pads {
		if math.Hypot(pd.X-p.X, pd.Y-p.Y) < 60 {
			return true
		}
	}
	for _, rs := range d.Roads {
		if rs.R.Grow(8).Has(p.X, p.Y) {
			return true
		}
	}
	return false
}

func unionRect(a, b Rect) Rect {
	x0, y0 := math.Min(a.X, b.X), math.Min(a.Y, b.Y)
	x1, y1 := math.Max(a.X+a.W, b.X+b.W), math.Max(a.Y+a.H, b.Y+b.H)
	return Rect{x0, y0, x1 - x0, y1 - y0}
}

// blockRect marks the navigation cells a new solid closes (what initNav would do for it).
func (l *Level) blockRect(r Rect) {
	g := r.Grow(4 + l.extraGrow)
	for cy := maxI(0, int(g.Y/navCell)-1); cy < l.ch && float64(cy)*navCell < g.Y+g.H+navCell; cy++ {
		for cx := maxI(0, int(g.X/navCell)-1); cx < l.cw && float64(cx)*navCell < g.X+g.W+navCell; cx++ {
			if g.Has((float64(cx)+0.5)*navCell, (float64(cy)+0.5)*navCell) {
				l.blocked[cy*l.cw+cx] = true
			}
		}
	}
}

// trySeatKit builds the kit; false (and nothing changed) if it cuts a way or leaves a seat out of reach.
func (l *Level) trySeatKit(pieces []seatPiece, reach0 []bool) bool {
	nProps, nP, nN := len(l.Props), len(l.solidsP), len(l.solidsN)
	saved := append([]bool(nil), l.blocked...)
	undo := func() {
		l.Props, l.solidsP, l.solidsN = l.Props[:nProps], l.solidsP[:nP], l.solidsN[:nN]
		l.blocked = saved
	}
	for _, pc := range pieces {
		pr := l.addTableProp(pc.name, propKinds[pc.name], pc.at, pc.v)
		if !pr.K.NoSolid {
			l.blockRect(pr.Foot)
		}
	}
	reach1 := l.reachFrom(l.NearestFree(l.Def.Start))
	for i := range reach0 {
		if reach0[i] && !reach1[i] && !l.blocked[i] { // a floor cell that was walkable and now is cut off
			undo()
			return false
		}
	}
	for _, pr := range l.Props[nProps:] {
		for _, sr := range seatRelsOf(pr.Name, pr.Var) {
			sit := Vec{pr.X + sr.dx, pr.Y + sr.dy}
			if _, ok := l.seatApproach(sit, reach1); !ok {
				undo()
				return false
			}
		}
	}
	return true
}

// seatApproach: the walkable spot next to a seat where the sitter stands (and stays, logically, while he sits).
func (l *Level) seatApproach(sit Vec, reach []bool) (Vec, bool) {
	a := l.NearestFree(sit)
	if dist(a, sit) > 18 || !l.Free(a) || l.SolidP(a.X, a.Y) {
		return Vec{}, false
	}
	cx, cy := l.cellOf(a)
	if reach != nil && (cx < 0 || cy < 0 || cx >= l.cw || cy >= l.ch || !reach[cy*l.cw+cx]) {
		return Vec{}, false
	}
	return a, true
}

// ---- the seats of the party and who sits on them ------------------------------------------------------------

type seatSpot struct {
	sit   Vec // the sitter's feet (drawn)
	app   Vec // where he stands before sitting (gu.pos)
	face  Dir
	prop  *Prop
	group int // seats round the same table share it
	sort  float64
	by    *Guest
}

type seatSit struct {
	s        *seatSpot
	t0       float64 // when he booked it
	sitAt    float64 // when he sat down (0 = still walking)
	act      int     // 0 resting, 1 talking, 2 drinking, 3 phone
	actT     float64 // next change of activity
	nextLine float64
}

type seatBook struct {
	lvl   *Level
	n     int // props when it was built (a fire or the VIP may change them)
	spots []*seatSpot
	next  map[*Guest]float64 // cooldown: no new seat before this game time
}

const (
	seatMinRest = 15.0
	seatMaxRest = 40.0
	seatWalkMax = 30.0 // never arrived: give up
	seatHardMax = 45.0 // safety: nobody sits longer
	seatGlide   = 0.35 // seconds to sit down / stand up
)

// seatBookOf: the seats of the current level (built when the level changes).
func (g *Game) seatBookOf() *seatBook {
	b := &g.seats
	l := g.lvl
	if l == nil {
		return b
	}
	if b.lvl == l && b.n == len(l.Props) {
		return b
	}
	oldSpots := b.spots
	if b.lvl != l { // a new party: nobody from the old one keeps a seat or a cooldown
		for _, s := range oldSpots {
			if s.by != nil && s.by.seat != nil && s.by.seat.s == s {
				s.by.seat = nil
			}
		}
		oldSpots, b.next = nil, map[*Guest]float64{}
	}
	if b.next == nil {
		b.next = map[*Guest]float64{}
	}
	b.lvl, b.n, b.spots = l, len(l.Props), nil
	// props were added or removed in the party (rare): the seats are rebuilt and the sitters keep theirs
	defer func() {
		for _, o := range oldSpots {
			gu := o.by
			if gu == nil || gu.seat == nil || gu.seat.s != o {
				continue
			}
			st := gu.seat
			gu.seat = nil
			for _, s := range b.spots {
				if s.prop == o.prop && s.sit == o.sit && s.by == nil {
					s.by, st.s, gu.seat = gu, s, st
					break
				}
			}
		}
	}()
	reach := l.reachFrom(l.NearestFree(l.Def.Start))
	var tables []*Prop
	for _, pr := range l.Props {
		if seatTableKinds[pr.Name] {
			tables = append(tables, pr)
		}
	}
	for i, pr := range l.Props {
		rels := seatRelsOf(pr.Name, pr.Var)
		if len(rels) == 0 || pr.gone {
			continue
		}
		sc := pr.sc()
		group := 1000 + i
		for ti, t := range tables {
			if dist(Vec{t.X, t.Y}, Vec{pr.X, pr.Y}) < 44 {
				group = ti
				break
			}
		}
		for _, sr := range rels {
			sit := Vec{math.Round(pr.X + sr.dx*sc), math.Round(pr.Y + sr.dy*sc)}
			app, ok := l.seatApproach(sit, reach)
			if !ok {
				continue
			}
			b.spots = append(b.spots, &seatSpot{sit: sit, app: app, face: sr.face, prop: pr, group: group,
				sort: math.Max(pr.Sort, sit.Y) + 0.5})
		}
	}
	return b
}

// seatCanSit: a guest free to go and sit.
func seatCanSit(g *Game, gu *Guest) bool {
	if gu.out() || gu.seat != nil || gu.perf != "" || gu.py != nil || gu.quirk != "" || gu.leader != nil || gu.armed ||
		gu.partner != nil || gu.poison > 0 || gu.bitten > 0 || gu.hurry || gu.tech || gu.state == gZombie {
		return false
	}
	if gu.cs != nil && gu.cs.Spec != nil && gu.cs.Spec.Ride != 0 { // kids on scooters
		return false
	}
	return !g.pevTaken(gu)
}

// seatPick: called by guestPick in its "rest" choices; true = the guest is on his way to a seat.
func (g *Game) seatPick(gu *Guest) bool {
	if g.lvl == nil {
		return false
	}
	b := g.seatBookOf()
	if len(b.spots) == 0 || b.next[gu] > g.time || !seatCanSit(g, gu) {
		return false
	}
	seated := map[int]int{}
	for _, s := range b.spots {
		if s.by != nil {
			seated[s.group]++
		}
	}
	var best *seatSpot
	bs := math.MaxFloat64
	for _, s := range b.spots {
		if s.by != nil {
			continue
		}
		d := dist(gu.pos, s.app)
		if d > 340 {
			continue
		}
		sc := d + g.rng.Float64()*90
		if n := seated[s.group]; n > 0 && n < 4 && s.group < 1000 {
			sc -= 120 // somebody is at that table: join them
		}
		if sc < bs {
			best, bs = s, sc
		}
	}
	if best == nil {
		return false
	}
	path := g.lvl.PathG(gu.pos, best.app)
	if path == nil {
		b.next[gu] = g.time + 10
		return false
	}
	best.by = gu
	gu.seat = &seatSit{s: best, t0: g.time}
	gu.path = path
	gu.state, gu.next, gu.dur = gGoto, gLounge, seatMinRest+g.rng.Float64()*(seatMaxRest-seatMinRest)
	return true
}

// seatFree gives the seat back (and starts the guest's cooldown).
func (g *Game) seatFree(gu *Guest) {
	st := gu.seat
	if st == nil {
		return
	}
	if st.s.by == gu {
		st.s.by = nil
	}
	gu.seat = nil
	if g.seats.next == nil {
		g.seats.next = map[*Guest]float64{}
	}
	g.seats.next[gu] = g.time + 40 + float64(int(gu.off*1000)%50)
}

// seatDropOut: the seats of the guests who are out of the party (dead, gone) are free again (end of every frame).
func (g *Game) seatDropOut() {
	for _, gu := range g.guests {
		if gu.seat != nil && gu.out() {
			g.seatFree(gu)
		}
	}
}

// seatsNowF: -seatsnow sends a dozen guests to sit down a second after the start (debug, for captures).
var seatsNowF = flag.Bool("seatsnow", false, "debug: a dozen guests go to sit down a second after the start")
var seatsNowLvl *Level

// updateSeats: called every frame after the guests moved.
func (g *Game) updateSeats(dt float64) {
	if g.lvl == nil {
		return
	}
	b := g.seatBookOf()
	if *seatsNowF && seatsNowLvl != g.lvl && g.time > 1 {
		seatsNowLvl = g.lvl
		sent := 0
		for _, gu := range g.guests {
			if sent < 14 && !gu.out() && gu.seat == nil && (gu.state == gIdle || gu.state == gDance || gu.state == gGoto) && gu.timer < 1000 {
				gu.state, gu.timer, gu.path = gIdle, 0, nil
				if g.seatPick(gu) {
					sent++
				}
			}
		}
	}
	for _, gu := range g.guests {
		st := gu.seat
		if st == nil {
			continue
		}
		if b.lvl != g.lvl || st.s.by != gu || gu.out() {
			g.seatFree(gu)
			continue
		}
		s := st.s
		if st.sitAt == 0 { // on the way
			switch {
			case gu.state == gLounge:
				if dist(gu.pos, s.app) > 10 {
					g.seatFree(gu)
					continue
				}
				st.sitAt = g.time
				gu.pos, gu.dir, gu.path = s.app, s.face, nil
				if gu.timer > seatMaxRest {
					gu.timer = seatMaxRest
				}
				st.actT = g.time
				st.nextLine = g.time + 2 + float64(int(gu.off*100)%5)
			case gu.state != gGoto || gu.next != gLounge:
				g.seatFree(gu) // something else took him (a fright, a party event...)
			case g.time-st.t0 > seatWalkMax:
				g.seatFree(gu)
				gu.state, gu.timer, gu.path = gIdle, 1, nil
			}
			continue
		}
		// sitting
		if gu.state != gLounge || dist(gu.pos, s.app) > 6 {
			g.seatFree(gu) // the rest is over, or an alarm made him jump up
			continue
		}
		if g.time-st.sitAt > seatHardMax {
			gu.state, gu.timer = gIdle, 1
			g.seatFree(gu)
			continue
		}
		gu.dir, gu.moving = s.face, false
		if g.time >= st.actT {
			st.actT = g.time + 5 + g.rng.Float64()*5
			mates := g.seatMates(b, gu)
			switch r := g.rng.Float64(); {
			case mates > 0 && r < 0.55:
				st.act = 1
			case r < 0.35:
				st.act = 2
			case r < 0.65:
				st.act = 3
			default:
				st.act = 0
			}
		}
		if st.act == 1 && g.time >= st.nextLine && gu.icon == "" && gu.iconT <= 0 {
			st.nextLine = g.time + 6 + g.rng.Float64()*6
			if g.seatMates(b, gu) > 0 && !g.seatGroupTalking(b, gu) {
				gu.icon, gu.iconT = seatLines[g.rng.Intn(len(seatLines))], 2.6
			}
		}
	}
}

// seatMates: how many others sit at the same table right now.
func (g *Game) seatMates(b *seatBook, gu *Guest) int {
	s := gu.seat.s
	if s.group >= 1000 {
		return 0
	}
	n := 0
	for _, o := range b.spots {
		if o != s && o.group == s.group && o.by != nil && o.by.seat != nil && o.by.seat.sitAt > 0 {
			n++
		}
	}
	return n
}

// seatGroupTalking: somebody at the table already has a bubble (one at a time per table).
func (g *Game) seatGroupTalking(b *seatBook, gu *Guest) bool {
	s := gu.seat.s
	for _, o := range b.spots {
		if o != s && o.group == s.group && o.by != nil && o.by.icon != "" {
			return true
		}
	}
	return false
}

// seatDrawn: this guest is drawn by seatItems (sitting), not by the plain guest loop.
func (g *Game) seatDrawn(gu *Guest) bool {
	st := gu.seat
	return st != nil && st.sitAt > 0 && gu.state == gLounge && !gu.out() && st.s.by == gu
}

// seatItems adds the sitters to the sorted draw list (they sort with their seat, not with gu.pos).
func (g *Game) seatItems(dst *ebiten.Image, items []drawItem) []drawItem {
	for _, gu := range g.guests {
		if !g.seatDrawn(gu) {
			continue
		}
		gu := gu
		items = append(items, drawItem{gu.seat.s.sort, func() { g.drawSitter(dst, gu) }})
	}
	return items
}

// seatGlideK: 0 standing at the approach spot .. 1 sitting on the seat.
func (g *Game) seatGlideK(gu *Guest) float64 {
	st := gu.seat
	k := math.Min(1, math.Max(0, (g.time-st.sitAt)/seatGlide))
	if gu.timer < seatGlide {
		k = math.Min(k, math.Max(0, gu.timer/seatGlide))
	}
	return k
}

func (g *Game) drawSitter(dst *ebiten.Image, gu *Guest) {
	st := gu.seat
	s := st.s
	k := g.seatGlideK(gu)
	pos := Vec{math.Round(s.app.X + (s.sit.X-s.app.X)*k), math.Round(s.app.Y + (s.sit.Y-s.app.Y)*k)}
	anim, ph := AnimIdle, g.time*1.3+gu.off
	if k >= 0.5 {
		anim, ph = seatAnim(st.act, g.time, gu.off)
	}
	g.drawActor(dst, gu.cs, anim, ph, pos, s.face, 0, false)
	if k >= 0.5 && s.face == DirDown {
		g.drawSeatFront(dst, s.prop)
	}
}

// seatFrontRow: the sprite row where the seat starts on furniture seen from the front. Its lower part is drawn
// again over the sitter, so the seat hides his lap and only the shins show between the legs.
var seatFrontRow = map[string]int{"seatfront": 10, "seatbench": 9, "seatdeck": 12, "seatpouf": 3, "chair": 13, "bench": 9,
	"couch": 12, "armchair": 12, "stool": 3}

func (g *Game) drawSeatFront(dst *ebiten.Image, pr *Prop) {
	row, ok := seatFrontRow[pr.Name]
	imgs := g.as.props[pr.Name]
	if !ok || len(imgs) == 0 || pr.gone {
		return
	}
	pi := imgs[pr.Var%len(imgs)]
	sc := pr.sc()
	b := pi.img.Bounds()
	top := int(math.Round(float64(row+1) * wk))
	if top >= b.Dy() {
		return
	}
	sub := pi.img.SubImage(image.Rect(b.Min.X, b.Min.Y+top, b.Max.X, b.Max.Y)).(*ebiten.Image)
	op := &ebiten.DrawImageOptions{}
	x, y := g.wx(pr.X), g.wy(pr.Y)
	if sc != 1 {
		op.Filter = ebiten.FilterLinear
		op.GeoM.Translate(-pi.ox, -pi.oy+float64(top))
		op.GeoM.Scale(sc, sc)
		op.GeoM.Translate(math.Round(x), math.Round(y))
	} else {
		op.GeoM.Translate(math.Round(x-pi.ox), math.Round(y-pi.oy)+float64(top))
	}
	if pr.burn > 0 {
		op.ColorScale.Scale(0.35, 0.3, 0.28, 1)
	}
	dst.DrawImage(sub, op)
}

// ---- the sitting poses (hidden animation kinds after the gaits, like gait.go) ---------------------------------

const (
	AnimSitRest = animGaitEnd + iota
	AnimSitTalk
	AnimSitDrink
	AnimSitPhone
	animSitEnd
)

func init() {
	gaitFrames[AnimSitRest], gaitFrames[AnimSitTalk], gaitFrames[AnimSitDrink], gaitFrames[AnimSitPhone] = 2, 4, 6, 2
}

func seatAnim(act int, t, off float64) (int, float64) {
	switch act {
	case 1:
		return AnimSitTalk, t*5 + off
	case 2:
		return AnimSitDrink, t*1.6 + off
	case 3:
		return AnimSitPhone, t*0.8 + off
	}
	return AnimSitRest, t*0.9 + off
}

// sitPoseFor: hips down on the seat, knees forward, feet on the floor.
func sitPoseFor(kind, f int) Pose {
	p := Pose{LA: restArm(), RA: restArm(), Sit: true, Bob: 4}
	p.LF, p.RF = Limb{1.5, 10, 5}, Limb{1.5, 10, 5}
	p.LA = Limb{2, 7, 3} // hands on the lap
	p.RA = Limb{2, 7, 3}
	switch kind {
	case AnimSitRest:
		p.Bob += float64(f) * 0.5
	case AnimSitTalk:
		s := math.Sin(float64(f) / 4 * 2 * math.Pi)
		p.RA = Limb{5 + 1.5*s, 2 - 2*math.Abs(s), 4}
		p.Open = f%2 == 0
	case AnimSitDrink:
		u := 1 - math.Abs(2*float64(f)/6-1)
		u = math.Min(1, u*1.8)
		p.RA = Limb{0.5, lerp(6, -4.5, u), lerp(3, 3.5, u)}
		p.Hold = HoldCup
	case AnimSitPhone:
		p.RA = Limb{1.5, 2, 5}
		p.LA = Limb{1.5, 3, 5}
		p.Hold = HoldPhone
		p.HeadZ = float64(f) * 0.3
	}
	return p
}

// seatLines: what they say at the table.
var seatLines = []string{
	"¡Qué bien se está sentado!", "Mis pies me lo agradecen", "¿Me guardas la silla?", "Esta fiesta es lo más",
	"¿Has visto quién ha venido?", "Uf, necesitaba esto", "Brindemos por el anfitrión", "Cinco minutos y vuelvo a bailar",
	"¿Y tú de qué conoces al anfitrión?", "¡Qué buena está la música!", "No me levanto ni a tiros", "Otra ronda y me voy",
}

// seatHideKinds: the tables with a cloth down to the floor that the guests hide under when there is danger
// (pe_tablehide.go): the cloth tables of pe_tables.go and the long tables of the kits.
var seatHideKinds = map[string]bool{"clothtable": true, "seatlong": true}

// seatHideIcon: the parties whose kits include a long table.
func seatHideIcon(icon string) bool { return seatThemes[icon].kits[kitLong] > 0 }
