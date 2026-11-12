package main

import (
	"image"
	"image/color"
	"math"
	"math/rand"
	"testing"
	"time"
)

// Tests of the amateur superhero (pe_hero.go), the umbrellas in the rain (umbrellas.go) and the lightning that never
// falls on a player (weather.go), on the real game after loadLevel, in several levels, with dice that change from run
// to run (no fixed seed decides the result).

// heroGlobals: every global of the hero, put back after the test.
func heroGlobals(t *testing.T) {
	t.Helper()
	r, s, in, lv := heroRng, heroSeed, heroInTests, heroLv
	t.Cleanup(func() { heroRng, heroSeed, heroInTests, heroLv = r, s, in, lv })
	heroRng = rand.New(rand.NewSource(time.Now().UnixNano()))
	heroLv = heroLevel{}
}

// heroLevelsWith: campaign levels (indexes) that pass the filter.
func heroLevelsWith(f func(d *LevelDef) bool) []int {
	var out []int
	for i := range campaignLevels {
		if i < len(levelDefs) && f(levelDefs[i]) {
			out = append(out, i)
		}
	}
	return out
}

// heroWatch runs the party until the hero event (forced or by its dice) starts and ends; it checks the phases on the
// way. It returns the state of the feat ("" fail = ok) or a reason.
func heroWatch(t *testing.T, g *Game, secs float64) (*heroState, int, string) {
	t.Helper()
	var cur *heroState
	starts := 0
	var seen *heroState
	for s := 0; s < int(secs*30); s++ {
		peStep(g, 1.0/30, true)
		a := g.partyRunning("heroe")
		if a == nil {
			if cur != nil {
				gu := cur.m.gu
				if !gu.out() && peHeld(gu) {
					return cur, starts, "the hero is still held after the feat"
				}
				if !gu.out() && (!g.lvl.Free(gu.pos) || g.lvl.SolidP(gu.pos.X, gu.pos.Y)) {
					return cur, starts, "the hero ended inside something solid"
				}
				if seen == nil || !seen.flown || cur.flown {
					seen = cur
				}
				cur = nil
			}
			continue
		}
		hs := a.D.(*heroState)
		if hs != cur {
			cur = hs
			starts++
		}
		if g.time-a.t0 > 31 {
			return cur, starts, "a feat lasted more than 31 s"
		}
		if hs.phase >= hpFly && !hs.m.gu.out() && hs.m.mine() {
			if p := hs.m.gu.pos; !g.lvl.Free(p) || g.lvl.SolidP(p.X, p.Y) {
				return cur, starts, "the hero landed on something solid"
			}
		}
	}
	if seen == nil {
		seen = cur
	}
	return seen, starts, ""
}

// Forced (-pevent heroe:...): the jump over a small thing, the plain flight and the pool, each with a crash or a pose;
// the hero walks to the run-up, flies, lands on free ground (or in the water) and is let go. 1 and 2 players.
func TestHeroForcedEachKind(t *testing.T) {
	heroGlobals(t)
	// the campaign has (almost) no swimming pool in Level.Pools: the water feat is checked with a pool laid on open
	// floor near the first player (InPool reads Level.Pools; the nav is the same, pools are walkable)
	all := heroLevelsWith(func(d *LevelDef) bool { return len(d.Roam) > 0 })
	pools := all
	cases := []struct {
		flag string
		kind int
		lv   []int
		two  bool
	}{
		{"heroe:salto-cae", hkJump, all, false},
		{"heroe:salto-posa", hkJump, all, true},
		{"heroe:vuelo-cae", hkFly, all, false},
		{"heroe:vuelo-posa", hkFly, all, true},
		{"heroe:agua-cae", hkWater, pools, false},
		{"heroe:agua-posa", hkWater, pools, false},
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for _, c := range cases {
		lv := append([]int(nil), c.lv...)
		r.Shuffle(len(lv), func(i, j int) { lv[i], lv[j] = lv[j], lv[i] })
		ok, last := false, ""
		for i, idx := range lv {
			if i >= 8 {
				break
			}
			g := peLoad(t, idx, c.two)
			heroLv = heroLevel{}
			if c.kind == hkWater && !heroAddPool(g) {
				last = "no room for a pool"
				continue
			}
			dbgPartyEvent = c.flag
			s, starts, why := heroWatch(t, g, 40)
			dbgPartyEvent = ""
			if why != "" {
				t.Fatalf("%s level %d: %s", c.flag, idx, why)
			}
			if s == nil || starts == 0 {
				last = "no start"
				continue
			}
			if s.plan.kind != c.kind {
				t.Fatalf("%s level %d: kind %d", c.flag, idx, s.plan.kind)
			}
			if !s.flown {
				last = "he never flew (frightened away?)"
				continue
			}
			wantCrash := c.flag[len(c.flag)-3:] == "cae"
			if s.plan.crash != wantCrash {
				t.Fatalf("%s: crash %v", c.flag, s.plan.crash)
			}
			if c.kind == hkWater && !g.lvl.InPool(s.plan.land) {
				t.Fatalf("%s level %d: the water jump does not land in the pool", c.flag, idx)
			}
			if c.kind == hkJump && (s.plan.prop == nil || !heroHop[s.plan.prop.Name]) {
				t.Fatalf("%s level %d: jump without a small thing", c.flag, idx)
			}
			if c.kind != hkWater && s.phase != hpUp && s.phase != hpPose && s.phase != hpLie {
				last = "it did not end with a pose or a crash"
				continue
			}
			t.Logf("%s: level %d (%s) ok, %d guests", c.flag, idx, levelDefs[idx].Title, len(g.guests))
			ok = true
			break
		}
		if !ok {
			t.Errorf("%s: no level worked (%s)", c.flag, last)
		}
	}
}

// The real dice: a hero comes only in some parties, one at a time, at most twice, he never starts near a player, and
// nobody is added or left held.
func TestHeroSometimesRealDiceNoStuck(t *testing.T) {
	heroGlobals(t)
	// the chance per party, measured on the dice themselves
	g0 := &Game{}
	want := 0
	for i := 0; i < 4000; i++ {
		heroLv.lvl = nil
		heroReset(g0)
		if heroLv.want {
			want++
			if heroLv.next < heroAtMin || heroLv.next > heroAtMin+heroAtVar {
				t.Fatalf("hero time %.0f outside the window", heroLv.next)
			}
		}
	}
	if f := float64(want) / 4000; f < 0.25 || f > 0.45 {
		t.Errorf("a hero in %.0f %% of the parties, want about a third", f*100)
	}
	heroInTests = true
	levels := heroLevelsWith(func(d *LevelDef) bool { return len(d.Roam) > 0 })
	rand.New(rand.NewSource(time.Now().UnixNano())).Shuffle(len(levels), func(i, j int) { levels[i], levels[j] = levels[j], levels[i] })
	runs := 0
	for i, idx := range levels[:4] {
		g := peLoad(t, idx, i == 3)
		n0 := len(g.guests)
		heroReset(g)
		heroLv.want, heroLv.next = true, 5 // this party has its hero early (the rest is the real dice)
		s, starts, why := heroWatch(t, g, 120)
		if why != "" {
			t.Fatalf("level %d: %s", idx, why)
		}
		if starts > 2 {
			t.Fatalf("level %d: %d heroes in two minutes", idx, starts)
		}
		if len(g.guests) != n0 {
			t.Fatalf("level %d: the hero changed the guests (%d -> %d)", idx, n0, len(g.guests))
		}
		if s != nil {
			runs++
			if d := heroMinPlayerDist(g, s.plan.a); d < 22 {
				t.Errorf("level %d: the run-up is %.0f px from a player", idx, d)
			}
		}
	}
	if runs == 0 {
		t.Error("no hero in four parties that had one planned")
	}
}

// heroAddPool lays a pool of 90x56 on open dry floor 60-200 px from the first player (false = no room).
func heroAddPool(g *Game) bool {
	l := g.lvl
	t0 := len(l.Pools)
	for dy := -160.0; dy <= 160; dy += 20 {
		for dx := -200.0; dx <= 200; dx += 20 {
			r := Rect{math.Round(g.p.pos.X + dx), math.Round(g.p.pos.Y + dy), 90, 56}
			if math.Hypot(dx, dy) < 60 {
				continue
			}
			ok := true
			for y := r.Y - 8; y <= r.Y+r.H+8 && ok; y += 4 {
				for x := r.X - 44; x <= r.X+r.W+44 && ok; x += 4 {
					p := Vec{x, y}
					ok = l.Free(p) && !l.SolidP(x, y) && !l.badGround(p) && !l.InPool(p)
				}
			}
			if ok {
				l.Pools = append(l.Pools, r)
				return len(l.Pools) > t0
			}
		}
	}
	return false
}

func heroMinPlayerDist(g *Game, p Vec) float64 {
	d := 1e9
	for _, pl := range g.players() {
		if pl != nil {
			d = math.Min(d, dist(pl.pos, p))
		}
	}
	return d
}

// Killed in the middle of his feat: the event ends, his cape stays on the floor and is drawn; he counts as a normal kill.
func TestHeroKilledLeavesCape(t *testing.T) {
	heroGlobals(t)
	levels := heroLevelsWith(func(d *LevelDef) bool { return len(d.Roam) > 0 })
	for _, idx := range levels[:6] {
		g := peLoad(t, idx, false)
		heroLv = heroLevel{}
		dbgPartyEvent = "heroe:vuelo"
		var hs *heroState
		for s := 0; s < 30*30 && hs == nil; s++ {
			peStep(g, 1.0/30, true)
			if a := g.partyRunning("heroe"); a != nil {
				hs = a.D.(*heroState)
			}
		}
		if hs == nil {
			dbgPartyEvent = ""
			continue
		}
		gu := hs.m.gu
		g.killGuest(gu, false)
		for s := 0; s < 30 && g.partyRunning("heroe") != nil; s++ {
			peStep(g, 1.0/30, true)
		}
		dbgPartyEvent = ""
		if g.partyRunning("heroe") != nil && g.partyRunning("heroe").D == hs {
			t.Fatal("the feat goes on with the hero dead")
		}
		if len(heroLv.capes) != 1 || dist(heroLv.capes[0].pos, gu.pos) > 2 {
			t.Fatalf("the cape is not where he fell: %v", heroLv.capes)
		}
		if len(g.heroItems(nil, nil)) != 1 {
			t.Fatal("the cape on the floor is not drawn")
		}
		return
	}
	t.Error("no level started a hero")
}

// The costume and the flight are drawn inside the hero's own box: a plot into an image never writes far from him.
func TestHeroCostumePixels(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for _, sc := range []float64{1, 0.7} {
		k := func(v float64) int { return int(math.Round(v * sc)) }
		cx, cy := 100, 100
		for n := 0; n < 30; n++ {
			l := heroLookFrom(r)
			if l.cape == l.briefs {
				t.Fatal("briefs the colour of the cape")
			}
			box := image.Rect(cx-14, cy-46, cx+15, cy+2)
			plot := func(x, y, w, h int, c color.RGBA) {
				if !image.Rect(x, y, x+w, y+h).In(box) {
					t.Fatalf("costume pixel (%d,%d) outside the hero's box at scale %.1f", x-cx, y-cy, sc)
				}
			}
			for _, d := range []Dir{DirDown, DirUp, DirLeft, DirRight} {
				heroCapeBehind(plot, cx, cy, 1, k, d, n%2 == 0, float64(n)*0.37, l, n%3 == 0)
				heroCostumeFront(plot, cx, cy, 1, k, d, n%2 == 0, float64(n)*0.37, l, n%3 == 0)
			}
		}
	}
	// names and look: a generic hero of nobody's comic (no initials, no real logo) and texts in both languages
	for _, s := range append(append(append([]string{}, heroGoL...), heroCrashL...), heroPoseL...) {
		if tr2EN(s) == s {
			t.Errorf("no English for %q", s)
		}
	}
	e := partyByID("heroe")
	if e == nil || !e.Reactive || !e.Minor || e.Big {
		t.Fatal("heroe must be a reactive Minor event (it never takes a place of the frame)")
	}
}

func tr2EN(s string) string {
	if en, ok := langDataEN[s]; ok {
		return en
	}
	return s
}

// ---- umbrellas -------------------------------------------------------------------------------------------

// stormLevel: a campaign level with rain (not the snow).
func umbStormLevels() []int {
	return heroLevelsWith(func(d *LevelDef) bool { return stormLevels[d.Icon] && d.Icon != "snow" && len(d.Roam) > 0 })
}

func umbRain(g *Game, k float64, snow bool) {
	g.wth = weatherState{thunder: -1, plan: weatherPlan{storm: true, snow: snow, at: 0, dur: 60, seed: 3}, started: true, k: k}
	g.wth.r = rand.New(rand.NewSource(3))
}

// Umbrellas only while it rains, only some of the guests in the open, never under a roof, many colours; they open as
// the rain grows and are all put away when it stops.
func TestUmbrellasOnlyInTheRainAndOnlySome(t *testing.T) {
	levels := umbStormLevels()
	if len(levels) == 0 {
		t.Fatal("no rainy level")
	}
	rand.New(rand.NewSource(time.Now().UnixNano())).Shuffle(len(levels), func(i, j int) { levels[i], levels[j] = levels[j], levels[i] })
	colours := map[int]bool{}
	for i, idx := range levels {
		if i >= 4 {
			break
		}
		g := peLoad(t, idx, i == 1)
		t.Cleanup(func() { g.wth = weatherState{thunder: -1} })
		count := func() (open, out int) {
			for _, gu := range g.guests {
				if gu.out() {
					continue
				}
				if outdoorAt(g.lvl.Def, gu.pos) {
					out++
				}
				if l, ok := g.umbrellaOf(gu); ok {
					open++
					colours[l.ci] = true
					if !outdoorAt(g.lvl.Def, gu.pos) {
						t.Fatalf("level %d: an umbrella under a roof", idx)
					}
				}
			}
			return
		}
		g.wth = weatherState{thunder: -1}
		if n, _ := count(); n != 0 {
			t.Fatalf("level %d: %d umbrellas without rain", idx, n)
		}
		umbRain(g, 0.3, false)
		few, _ := count()
		umbRain(g, 1, false)
		full, out := count()
		if out >= 10 && (full < out/5 || full > out*7/10) {
			t.Errorf("level %d: %d umbrellas of %d guests in the open, want some (about 45 %%)", idx, full, out)
		}
		if few > full {
			t.Errorf("level %d: more umbrellas at the start of the rain (%d) than at full rain (%d)", idx, few, full)
		}
		umbRain(g, 1, true)
		if n, _ := count(); n != 0 {
			t.Fatalf("level %d: umbrellas in the snow", idx)
		}
		umbRain(g, 1, false)
		g.wth.done, g.wth.k = true, 0
		if n, _ := count(); n != 0 {
			t.Fatalf("level %d: %d umbrellas after the storm", idx, n)
		}
	}
	// many colours over many guests (and the real ones above)
	share, cols, pats := 0, map[int]bool{}, map[int]bool{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 600; i++ {
		gu := &Guest{name: string(rune('a'+r.Intn(26))) + string(rune('a'+r.Intn(26))), off: r.Float64() * 6, style: r.Intn(4)}
		if umbCarrier(gu) {
			share++
			l := umbLookOf(gu)
			cols[l.ci], pats[l.pat] = true, true
		}
	}
	if f := float64(share) / 600; f < 0.33 || f > 0.57 {
		t.Errorf("%.0f %% of the guests carry an umbrella, want about 45 %%", f*100)
	}
	if len(cols) < 6 || len(pats) < 4 {
		t.Errorf("only %d colours and %d patterns", len(cols), len(pats))
	}
	t.Logf("colours seen in the real levels: %d", len(colours))
}

// The umbrella stays inside the guest's own sprite box (so it never reaches the HUD farther than he does) and is
// never drawn over a player.
func TestUmbrellaInsideTheGuestAndNeverOverThePlayer(t *testing.T) {
	for _, idx := range umbStormLevels() {
		if cs := levelDefs[idx].CharScale; cs != 0 && cs != 1 {
			t.Errorf("level %d has rain and small people (%.2f): the umbrella would be too big", idx, cs)
		}
	}
	for _, sc := range []float64{1} {
		k := func(v float64) int { return int(math.Round(v * sc)) }
		cx, cy := 200, 200
		box := image.Rect(cx-k(12), cy-k(43), cx+k(12)+1, cy+1)
		for n := 0; n < 40; n++ {
			gu := &Guest{name: "x", off: float64(n) * 0.31}
			l := umbLookOf(gu)
			x0, y0, x1, y1 := umbRect(cx, cy, 1, k, l.side)
			rr := image.Rect(x0, y0, x1, y1)
			if !rr.In(box) {
				t.Fatalf("scale %.1f: umbrella rect %v outside the guest's box %v", sc, rr, box)
			}
			for _, sway := range []int{-1, 0, 1} {
				for tt := 0.0; tt < 2; tt += 0.07 {
					drawUmbrellaPix(func(x, y, w, h int, c color.RGBA) {
						if !image.Rect(x, y, x+w, y+h).In(rr) {
							t.Fatalf("scale %.1f: umbrella pixel (%d,%d) outside its rect", sc, x-cx, y-cy)
						}
					}, cx, cy, 1, k, l, sway, tt, 1)
				}
			}
		}
	}
	// a guest right in front of the player: his umbrella would cover the player, so it is not drawn
	levels := umbStormLevels()
	g := peLoad(t, levels[0], true)
	t.Cleanup(func() { g.wth = weatherState{thunder: -1} })
	umbRain(g, 1, false)
	k := g.pevK()
	for _, p := range g.players() {
		cx, cy := g.pevXY(Vec{p.pos.X + 4, p.pos.Y + 10})
		x0, y0, x1, y1 := umbRect(cx, cy, g.pevPx(), k, 1)
		if !g.umbCoversPlayer(x0, y0, x1, y1) {
			t.Error("an umbrella just in front of a player is not seen as covering him")
		}
		cx, cy = g.pevXY(Vec{p.pos.X + 120, p.pos.Y})
		x0, y0, x1, y1 = umbRect(cx, cy, g.pevPx(), k, 1)
		if g.umbCoversPlayer(x0, y0, x1, y1) && heroMinPlayerDist(g, Vec{p.pos.X + 120, p.pos.Y}) > 60 {
			t.Error("an umbrella far from the players is hidden")
		}
	}
}

// ---- lightning -------------------------------------------------------------------------------------------

// The bolt never falls on a player: players standing in the middle of the guests in the open, many bolts, several
// levels and seeds, 1 and 2 players. Only guests are struck (some, one per bolt), never the player's kill.
func TestLightningNeverHitsAPlayer(t *testing.T) {
	levels := umbStormLevels()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(levels), func(i, j int) { levels[i], levels[j] = levels[j], levels[i] })
	struck := 0
	for i, idx := range levels {
		if i >= 4 {
			break
		}
		two := i%2 == 1
		g := peLoad(t, idx, two)
		t.Cleanup(func() { g.wth = weatherState{thunder: -1} })
		for b := 0; b < 12; b++ {
			umbRain(g, 1, false)
			g.wth.r = rand.New(rand.NewSource(r.Int63()))
			// the players stand on top of guests in the open
			live := g.liveGuests()
			for j, p := range g.players() {
				for _, gu := range live {
					if outdoorAt(g.lvl.Def, gu.pos) && r.Intn(3) == 0 {
						p.pos = gu.pos
						break
					}
				}
				_ = j
			}
			dead0 := map[*Guest]bool{}
			for _, gu := range g.guests {
				dead0[gu] = gu.dead
			}
			kills0 := g.kills
			g.stormStrike()
			for _, p := range g.players() {
				if p.dead {
					t.Fatalf("level %d: a bolt killed a player", idx)
				}
				px, py := g.pevXY(p.pos)
				if g.wth.flashT == 0 && math.Abs(g.wth.boltX2-float64(px)) < 14 && g.wth.boltY > float64(py)-46 && g.wth.boltY < float64(py)+4 {
					t.Fatalf("level %d: the bolt ends on a player", idx)
				}
			}
			n := 0
			for _, gu := range g.guests {
				if gu.dead && !dead0[gu] {
					n++
					if d := heroMinPlayerDist(g, gu.pos); d < stormSafeR {
						t.Fatalf("level %d: a guest %.0f px from a player was struck", idx, d)
					}
				}
			}
			if n > 1 {
				t.Fatalf("level %d: one bolt killed %d guests", idx, n)
			}
			if g.kills != kills0 {
				t.Fatalf("level %d: the bolt counted as the player's kill", idx)
			}
			struck += n
		}
	}
	if struck == 0 {
		t.Error("no guest was ever struck")
	}
}
