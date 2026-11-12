package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math/rand"
	"os"
	"testing"
)

// ---- helpers -----------------------------------------------------------------------------------------------

// seatPaint pastes a pixBuf onto an image with its anchor (ax, ay) at (x, y) (alpha over).
func seatPaint(dst *image.RGBA, b *pixBuf, ax, ay, x, y int, flip bool) {
	src := b.toRGBA()
	if flip {
		f := image.NewRGBA(src.Bounds())
		w := src.Bounds().Dx()
		for yy := 0; yy < src.Bounds().Dy(); yy++ {
			for xx := 0; xx < w; xx++ {
				f.Set(w-1-xx, yy, src.At(xx, yy))
			}
		}
		src = f
		ax = w - ax
	}
	r := image.Rect(x-ax, y-ay, x-ax+src.Bounds().Dx(), y-ay+src.Bounds().Dy())
	draw.Draw(dst, r, src, image.Point{}, draw.Over)
}

func seatPropBuf(name string, v int) (*pixBuf, int, int) {
	k := propKinds[name]
	b := stylizeSprite(renderPropSprite(name, k, v))
	return b, k.W/2 + 1, k.H + 1
}

func seatSitterBuf(sp *CharSpec, face Dir, anim, f int) (*pixBuf, bool) {
	vw, flip := int(vFront), false
	switch face {
	case DirUp:
		vw = int(vBack)
	case DirLeft:
		vw, flip = int(vSide), true
	case DirRight:
		vw = int(vSide)
	}
	p := poseFor(anim, f)
	p.Anim, p.F = anim, f
	return genChar(sp, vw, p).outlinedSel(0.42), flip
}

// TestSeatSpritesPreview paints every kit with people sitting on it, x4, into a PNG (SEATS_PREVIEW=file.png).
func TestSeatSpritesPreview(t *testing.T) {
	out := os.Getenv("SEATS_PREVIEW")
	if out == "" {
		t.Skip("SEATS_PREVIEW not set")
	}
	const W, H = 520, 300
	im := image.NewRGBA(image.Rect(0, 0, W, H))
	draw.Draw(im, im.Bounds(), &image.Uniform{color.RGBA{0x5a, 0x4a, 0x3c, 255}}, image.Point{}, draw.Src)
	r := rand.New(rand.NewSource(5))
	type item struct {
		y    float64
		draw func()
	}
	row := func(mat int, y float64) []item {
		var its []item
		x := 50.0
		for kit := 0; kit < kitN; kit++ {
			p := Vec{x, y}
			for _, pc := range seatKitPieces(kit, mat, p, r) {
				pc := pc
				b, ax, ay := seatPropBuf(pc.name, pc.v)
				its = append(its, item{pc.at.Y, func() { seatPaint(im, b, ax, ay, int(pc.at.X), int(pc.at.Y), false) }})
				for i, sr := range seatRelsOf(pc.name, pc.v) {
					sp := genericSpec(rand.New(rand.NewSource(int64(len(its)*7 + i))))
					sit := Vec{pc.at.X + sr.dx, pc.at.Y + sr.dy}
					cb, flip := seatSitterBuf(&sp, sr.face, AnimSitRest+(len(its)+i)%4, 0)
					its = append(its, item{sit.Y + 0.5, func() { seatPaint(im, cb, charCX, charBase, int(sit.X), int(sit.Y), flip) }})
					if row, ok := seatFrontRow[pc.name]; ok && sr.face == DirDown {
						crop := newPix(b.w, b.h)
						copy(crop.p[(row+1)*b.w*4:], b.p[(row+1)*b.w*4:])
						its = append(its, item{sit.Y + 0.6, func() { seatPaint(im, crop, ax, ay, int(pc.at.X), int(pc.at.Y), false) }})
					}
				}
			}
			x += 78
		}
		return its
	}
	var all []item
	all = append(all, row(0, 70)...)
	all = append(all, row(3, 150)...)
	all = append(all, row(2, 230)...)
	for i := range all { // painter's order
		for j := i + 1; j < len(all); j++ {
			if all[j].y < all[i].y {
				all[i], all[j] = all[j], all[i]
			}
		}
	}
	for _, it := range all {
		it.draw()
	}
	big := image.NewRGBA(image.Rect(0, 0, W*3, H*3))
	for y := 0; y < H*3; y++ {
		for x := 0; x < W*3; x++ {
			big.Set(x, y, im.At(x/3, y/3))
		}
	}
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, big); err != nil {
		t.Fatal(err)
	}
}

// seatKitProps: the furniture of the kits (seats_props.go) in a level.
func seatKitProps(l *Level) []*Prop {
	var out []*Prop
	for _, p := range l.Props {
		switch p.Name {
		case "seatchair", "seatfront", "seattable", "seatlong", "seatbench", "seatdeck", "seatpouf":
			out = append(out, p)
		}
	}
	return out
}

func seatWanted(d *LevelDef) int {
	th, ok := seatThemeOf(d)
	if !ok {
		return 0
	}
	n := 0
	for _, k := range th.kits {
		n += k
	}
	return n
}

// Every campaign party gets kits; none of their pieces stands on anything (props, walls, forbidden ground, the dance
// floor, the VIP carpet, the start) and nothing that could be walked before is cut off by them.
func TestSeatKitsEveryLevelSafeAndReachable(t *testing.T) {
	n := killTargetLevels(t)
	for i := 0; i < n; i++ {
		g := playGame(t, i, int64(11+i))
		l := g.lvl
		d := l.Def
		th, _ := seatThemeOf(d)
		kits := seatKitProps(l)
		if len(kits) == 0 {
			t.Errorf("level %d (%s): no seats placed", i+1, d.Icon)
		}
		for _, pa := range kits {
			for _, pb := range l.Props {
				if pa != pb && pairClash(pa.Name, pa.K, pa.Foot, pa.base(), pb.Name, pb.K, pb.Foot, pb.base()) {
					t.Errorf("level %d: %s at %.0f,%.0f clashes with %s at %.0f,%.0f", i+1, pa.Name, pa.X, pa.Y, pb.Name, pb.X, pb.Y)
				}
			}
			for _, w := range l.Walls {
				if w.Style != WallVoid && wallClash(pa.Name, pa.K, pa.Foot, pa.base(), w) {
					t.Errorf("level %d: %s on a wall", i+1, pa.Name)
				}
			}
			at := Vec{pa.X, pa.Y}
			if what := groundClash(d, pa.Name, pa.K, pa.Foot, at); what != "" {
				t.Errorf("level %d: %s at %.0f,%.0f on %s", i+1, pa.Name, pa.X, pa.Y, what)
			}
			if l.seatBadGround(at, th.edge) || l.InPool(at) {
				t.Errorf("level %d: %s on bad ground", i+1, pa.Name)
			}
			for _, dr := range d.Dance {
				if dr.Grow(18).Has(pa.X, pa.Y) {
					t.Errorf("level %d: %s on the dance floor", i+1, pa.Name)
				}
			}
			if dist(at, d.Start) < 60 {
				t.Errorf("level %d: %s next to the start", i+1, pa.Name)
			}
			if z := l.vip; z != nil {
				for _, rc := range []Rect{z.in, z.trig, z.gate} {
					if rc.Grow(20).Has(pa.X, pa.Y) {
						t.Errorf("level %d: %s in the VIP zone", i+1, pa.Name)
					}
				}
			}
		}
		// the same party without the kits: everything that was walkable is still reachable, every place of use too
		a := buildLevelRaw(d)
		a.addTables()
		r0 := a.reachFrom(a.NearestFree(d.Start))
		a.addSeats()
		r1 := a.reachFrom(a.NearestFree(d.Start))
		for c := range r0 {
			if r0[c] && !a.blocked[c] && !r1[c] {
				t.Errorf("level %d: the cell %d,%d was reachable and is cut off by the seats", i+1, c%a.cw, c/a.cw)
				break
			}
		}
		for kind, ps := range a.POIs {
			for _, p := range ps {
				q := a.NearestFree(p)
				cx, cy := a.cellOf(q)
				if cx < 0 || cy < 0 || cx >= a.cw || cy >= a.ch {
					continue
				}
				if r0[cy*a.cw+cx] && !r1[cy*a.cw+cx] {
					t.Errorf("level %d: the %s place at %.0f,%.0f is cut off", i+1, kind, p.X, p.Y)
				}
			}
		}
		// every seat of the party can be reached
		b := g.seatBookOf()
		reach := l.reachFrom(l.NearestFree(d.Start))
		for _, s := range b.spots {
			cx, cy := l.cellOf(s.app)
			if !l.Free(s.app) || !reach[cy*l.cw+cx] || dist(s.app, s.sit) > 18 {
				t.Errorf("level %d: the seat of %s at %.0f,%.0f is out of reach", i+1, s.prop.Name, s.sit.X, s.sit.Y)
			}
		}
		t.Logf("level %2d %-22s kits=%d pieces=%d seats=%d", i+1, d.Title, seatKitCount(l), len(kits), len(b.spots))
	}
}

func seatKitCount(l *Level) int {
	n := 0
	for _, p := range seatKitProps(l) {
		switch p.Name {
		case "seattable", "seatlong", "seatbench":
			n++
		case "seatdeck", "seatpouf": // two per kit
			n++
		}
	}
	return n
}

// The same party always gets the same kits; tests can switch them off.
func TestSeatKitsDeterministicAndSwitch(t *testing.T) {
	killTargetLevels(t)
	for i := range campaignLevels {
		d := levelDefs[i]
		a, b := buildLevelRaw(d), buildLevelRaw(d)
		a.addTables()
		b.addTables()
		na, nb := a.addSeats(), b.addSeats()
		pa, pb := seatKitProps(a), seatKitProps(b)
		if na != nb || len(pa) != len(pb) {
			t.Errorf("level %d: %d kits then %d", i+1, na, nb)
			continue
		}
		for k := range pa {
			if pa[k].X != pb[k].X || pa[k].Y != pb[k].Y || pa[k].Name != pb[k].Name || pa[k].Var != pb[k].Var {
				t.Errorf("level %d: the kits are not deterministic", i+1)
				break
			}
		}
	}
	seatsOff = true
	t.Cleanup(func() { seatsOff = false })
	if n := buildLevelRaw(levelDefs[1]).addSeats(); n != 0 {
		t.Error("seatsOff is ignored")
	}
}

// The levels of the random mode get kits too, just as safely.
func TestSeatKitsRandomLevels(t *testing.T) {
	killTargetLevels(t)
	got, all := 0, 0
	for _, kind := range genKinds {
		d := genNamed(kind, 977, "")
		if d == nil {
			continue
		}
		all++
		l := buildLevel(d)
		kits := seatKitProps(l)
		if len(kits) > 0 {
			got++
		}
		for _, pa := range kits {
			for _, pb := range l.Props {
				if pa != pb && pairClash(pa.Name, pa.K, pa.Foot, pa.base(), pb.Name, pb.K, pb.Foot, pb.base()) {
					t.Errorf("%s: %s clashes with %s", kind, pa.Name, pb.Name)
				}
			}
		}
		r0 := buildLevelRaw(d)
		r0.addTables()
		c0 := r0.reachFrom(r0.NearestFree(d.Start))
		r0.addSeats()
		c1 := r0.reachFrom(r0.NearestFree(d.Start))
		for c := range c0 {
			if c0[c] && !r0.blocked[c] && !c1[c] {
				t.Errorf("%s: a walkable cell is cut off by the seats", kind)
				break
			}
		}
	}
	if all == 0 || got*10 < all*7 {
		t.Errorf("only %d of %d random levels got seats", got, all)
	}
}

// seatCheckBook: nobody shares a seat and every booking points both ways.
func seatCheckBook(t *testing.T, g *Game, where string) (sitting int) {
	t.Helper()
	b := &g.seats
	seen := map[*Guest]*seatSpot{}
	for _, s := range b.spots {
		if s.by == nil {
			continue
		}
		if o := seen[s.by]; o != nil {
			t.Fatalf("%s: a guest holds two seats", where)
		}
		seen[s.by] = s
		if s.by.seat == nil || s.by.seat.s != s {
			t.Fatalf("%s: a seat is booked by a guest who does not know it", where)
		}
	}
	for _, gu := range g.guests {
		if gu.seat == nil {
			continue
		}
		if gu.seat.s.by != gu {
			t.Fatalf("%s: a guest thinks he has a seat that is not his", where)
		}
		if gu.out() {
			t.Fatalf("%s: a dead guest keeps a seat", where)
		}
		if g.seatDrawn(gu) {
			sitting++
			if g.time-gu.seat.sitAt > seatHardMax+0.5 {
				t.Fatalf("%s: a guest sits for %.0f s", where, g.time-gu.seat.sitAt)
			}
		} else if gu.seat.sitAt == 0 && g.time-gu.seat.t0 > seatWalkMax+0.5 {
			t.Fatalf("%s: a guest walks to his seat for %.0f s", where, g.time-gu.seat.t0)
		}
	}
	return sitting
}

// seatSend books seats for up to n idle guests (what guestPick does in its "rest" choices).
func seatSend(g *Game, n int) int {
	sent := 0
	for _, gu := range g.guests {
		if sent >= n {
			break
		}
		if gu.out() || gu.seat != nil || gu.state != gIdle && gu.state != gDance && gu.state != gGoto {
			continue
		}
		gu.state, gu.timer, gu.path = gIdle, 0, nil
		if g.seatPick(gu) {
			sent++
		}
	}
	return sent
}

// The guests go to free seats, sit down, rest and stand up; nobody shares a seat, nobody gets stuck. One and two
// players, in several kinds of party.
func TestSeatGuestsSitAndStandUp(t *testing.T) {
	killTargetLevels(t)
	for _, tc := range []struct {
		lvl int
		two bool
	}{{0, false}, {1, false}, {5, false}, {8, false}, {9, false}, {19, false}, {24, false}, {25, false}, {1, true}, {10, true}} {
		g := bigPlay(t, tc.lvl, int64(31+tc.lvl), tc.two)
		where := g.lvl.Def.Title
		if tc.two {
			where += " 2P"
		}
		capBefore := g.goalMax()
		sent := seatSend(g, 12)
		if sent < 2 {
			t.Errorf("%s: only %d guests went to sit", where, sent)
			continue
		}
		maxSit, stoodUp := 0, 0
		wasSitting := map[*Guest]bool{}
		for f := 0; f < 60*75; f++ {
			g.updatePlay(1.0 / 60)
			if g.mode != ModePlay {
				break
			}
			if f%30 == 0 {
				s := seatCheckBook(t, g, where)
				if s > maxSit {
					maxSit = s
				}
				for _, gu := range g.guests {
					sit := g.seatDrawn(gu)
					if wasSitting[gu] && !sit {
						stoodUp++
					}
					wasSitting[gu] = sit
				}
			}
			if f == 60*20 {
				seatSend(g, 6)
			}
		}
		if maxSit < 2 {
			t.Errorf("%s: at most %d guests sat at once", where, maxSit)
		}
		if stoodUp < 2 {
			t.Errorf("%s: only %d stood up again", where, stoodUp)
		}
		if g.goalMax() != capBefore || g.total > g.goalMax() {
			t.Errorf("%s: the caps moved (%d -> %d, total %d)", where, capBefore, g.goalMax(), g.total)
		}
		t.Logf("%s: sent %d, up to %d sitting, %d stood up", where, sent, maxSit, stoodUp)
	}
}

// seatOne sits a guest down on a free seat at once (as if he had just arrived).
func seatOne(t *testing.T, g *Game, skip *Guest) (*Guest, *seatSpot) {
	t.Helper()
	b := g.seatBookOf()
	for _, s := range b.spots {
		if s.by != nil {
			continue
		}
		for _, gu := range g.guests {
			if gu == skip || gu.out() || !seatCanSit(g, gu) {
				continue
			}
			gu.pos, gu.path = s.app, nil
			gu.state, gu.timer = gIdle, 0
			s.by, gu.seat = gu, &seatSit{s: s, t0: g.time}
			gu.state, gu.timer = gLounge, 20
			g.updateSeats(1.0 / 60)
			if !g.seatDrawn(gu) {
				t.Fatal("the guest did not sit down")
			}
			return gu, s
		}
	}
	t.Fatal("no free seat")
	return nil, nil
}

// A sitter killed on his seat leaves it free and his body lies beside it; a fright makes the others jump up and run.
func TestSeatKilledAndScaredSitters(t *testing.T) {
	killTargetLevels(t)
	g := bigPlay(t, 5, 5, false)
	gu, s := seatOne(t, g, nil)
	if gu.dir != s.face {
		t.Error("the sitter does not look into the table / the scene")
	}
	g.killGuest(gu, true)
	g.updateSeats(1.0 / 60)
	if s.by != nil || gu.seat != nil || g.seatDrawn(gu) {
		t.Error("a dead sitter keeps his seat")
	}
	found := false
	for _, bd := range g.bodies {
		if bd.cs == gu.cs && dist(bd.pos, s.sit) < 20 {
			found = true
		}
	}
	if !found {
		t.Error("no body beside the seat")
	}
	// the seat can be taken again
	g2, s2 := seatOne(t, g, gu)
	if s2 == nil || g2 == nil {
		t.Fatal("no seat")
	}
	g.scareAround(s2.app, 60)
	g.updateSeats(1.0 / 60)
	if g2.seat != nil || s2.by != nil || g.seatDrawn(g2) {
		t.Errorf("a scared sitter stays seated (state %d)", g2.state)
	}
	// and a witness: a seated guest still sees (his state and pos are those of a plain guest)
	g3, _ := seatOne(t, g, g2)
	if g3.out() || g3.state != gLounge {
		t.Error("a sitter is not a plain guest")
	}
}

// The pieces stay inside their box (no pixel on the 1 px border of the sprite), so nothing is cut when drawn.
func TestSeatSpritesInsideTheirBox(t *testing.T) {
	for _, name := range []string{"seatchair", "seatfront", "seattable", "seatlong", "seatbench", "seatdeck", "seatpouf"} {
		k := propKinds[name]
		if k == nil {
			t.Fatalf("%s missing", name)
		}
		if k.Category != "furniture" {
			t.Errorf("%s: not in the furniture of the editor", name)
		}
		n := maxI(1, k.Vars)
		for v := 0; v < n; v++ {
			b := newPix(k.W+2, k.H+2)
			b.ox, b.oy = 1, 1
			k.Draw(b, v)
			opaque := 0
			for y := 0; y < b.h; y++ {
				for x := 0; x < b.w; x++ {
					a := b.p[(y*b.w+x)*4+3]
					if a == 0 {
						continue
					}
					opaque++
					if x == 0 || y == 0 || x == b.w-1 || y == b.h-1 {
						t.Fatalf("%s v%d: a pixel outside the box at %d,%d", name, v, x, y)
					}
				}
			}
			if opaque < 30 {
				t.Errorf("%s v%d: almost empty", name, v)
			}
		}
		// the footprint is only the bottom of the piece (legs, pedestal, cloth), never the whole drawing
		if k.FH > 8 || k.FW > float64(k.W) {
			t.Errorf("%s: footprint %vx%v is too big", name, k.FW, k.FH)
		}
	}
}

// The sitting animations: hips down on the seat (Sit), and every frame can be drawn.
func TestSeatPoses(t *testing.T) {
	sp := genericSpec(rand.New(rand.NewSource(9)))
	for a := AnimSitRest; a < animSitEnd; a++ {
		n := animFrameCount(a)
		if n < 2 {
			t.Errorf("anim %d: %d frames", a, n)
		}
		for f := 0; f < n; f++ {
			p := poseFor(a, f)
			if !p.Sit {
				t.Errorf("anim %d frame %d is not a sitting pose", a, f)
			}
			for _, vw := range []int{int(vFront), int(vSide), int(vBack)} {
				if b := genChar(&sp, vw, p); b == nil || b.w == 0 {
					t.Errorf("anim %d: no picture", a)
				}
			}
		}
	}
}

// The long tables have a cloth: the guests can hide under them (pe_tablehide.go) like under the cloth tables.
func TestSeatLongTablesAreHidingPlaces(t *testing.T) {
	killTargetLevels(t)
	found := 0
	for i, e := range campaignLevels {
		if !seatHideIcon(e.icon) {
			continue
		}
		g := playGame(t, i, int64(3+i))
		reach := g.lvl.reachFrom(g.lvl.NearestFree(g.lvl.Def.Start))
		for _, p := range g.lvl.Props {
			if p.Name != "seatlong" {
				continue
			}
			found++
			for _, s := range tableSlots(g.lvl, Vec{p.X, p.Y}) {
				cx, cy := g.lvl.cellOf(s)
				if s == (Vec{}) || !g.lvl.Free(s) || !reach[cy*g.lvl.cw+cx] {
					t.Errorf("level %d: a place under the long table at %.0f,%.0f is not usable", i+1, p.X, p.Y)
				}
			}
		}
	}
	if found < 3 {
		t.Errorf("only %d long tables in the campaign", found)
	}
}
