package main

import (
	"image/color"
	"math"
	"math/rand"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// weatherGame: the shared headless game on a campaign party (by icon), in play, with the weather globals restored.
func weatherGame(t *testing.T, two bool, icon string) *Game {
	g := layerGame(t, two)
	ds, dd, wr, ws := dbgStorm, dbgDawn, weatherRng, weatherSeed
	mode, snd := g.mode, g.snd
	t.Cleanup(func() {
		dbgStorm, dbgDawn, weatherRng, weatherSeed = ds, dd, wr, ws
		g.mode, g.snd, g.wth = mode, snd, weatherState{}
	})
	g.snd = &Sounds{muted: true, sfx: map[string][]byte{}}
	i := campaignIdx(icon)
	if i < 0 {
		t.Fatalf("no campaign party %q", icon)
	}
	g.loadLevel(i)
	g.mode = ModePlay
	return g
}

// indoorShare: the share of the floor area of a level that is indoor floor.
func indoorShare(d *LevelDef) float64 {
	var in, all float64
	for _, f := range d.Floors {
		a := f.R.W * f.R.H
		all += a
		if floorIndoor(f.Kind) {
			in += a
		}
	}
	if all == 0 {
		return 1
	}
	return in / all
}

// The lists are chosen by the data: storm parties are open-air (almost no indoor floor), at night, in the campaign;
// dawn parties are night parties in the open. ITALIA (day), the space station and the island are never in them.
func TestWeatherLevelsChosenByData(t *testing.T) {
	layerGame(t, false)
	for icon := range stormLevels {
		i := campaignIdx(icon)
		if i < 0 {
			t.Errorf("storm level %q is not a campaign party", icon)
			continue
		}
		d := levelDefs[i]
		if d.Day || indoorShare(d) > 0.12 { // the ski resort has its small chalet
			t.Errorf("storm level %q is not a night open-air party (day %v, indoor %.2f)", icon, d.Day, indoorShare(d))
		}
	}
	for icon := range dawnLevels {
		i := campaignIdx(icon)
		if i < 0 {
			t.Errorf("dawn level %q is not a campaign party", icon)
			continue
		}
		if d := levelDefs[i]; d.Day || indoorShare(d) > 0.2 {
			t.Errorf("dawn level %q is not a night open-air party (day %v, indoor %.2f)", icon, d.Day, indoorShare(d))
		}
	}
	if len(dawnLevels) < 3 || len(dawnLevels) > 5 {
		t.Errorf("%d dawn parties, want 3-5", len(dawnLevels))
	}
	for _, no := range []string{"station", "italy", "jurassic", "house", "disco", "casino", "cinema", "boat", "train", "hockey"} {
		if stormLevels[no] || dawnLevels[no] {
			t.Errorf("%q must not have weather", no)
		}
	}
}

// Storm: about one party in 5-6 of the list, at most one per party, never at the start, 40-80 s; the bolt on a guest
// is very rare and never in the snow. Dawn: about one in three of its parties. Same seed, same plan.
func TestWeatherStormAndDawnFrequency(t *testing.T) {
	const n = 30000
	for icon := range stormLevels {
		r := rand.New(rand.NewSource(int64(len(icon)) * 977))
		storms, strikes := 0, 0
		for i := 0; i < n; i++ {
			p := weatherPlanFor(r, icon, true, false, false, false)
			if !p.storm {
				continue
			}
			storms++
			if p.at < stormAtMin || p.at > stormAtMax || p.dur < stormDurMin || p.dur > stormDurMax {
				t.Fatalf("%s: plan out of range %+v", icon, p)
			}
			if p.dawn {
				t.Fatalf("%s: storm and dawn in the same party", icon)
			}
			if p.strike {
				strikes++
				if p.snow {
					t.Fatalf("%s: a bolt in the snow", icon)
				}
				if p.strikeAt < stormFadeIn || p.strikeAt > p.dur-stormFadeOut {
					t.Fatalf("%s: bolt outside the full storm %+v", icon, p)
				}
			}
		}
		f := float64(storms) / n
		if f < 1.0/6.5 || f > 1.0/4.5 {
			t.Errorf("%s: storm in %.3f of the parties, want about 1/5-1/6", icon, f)
		}
		if icon != "snow" {
			if s := float64(strikes) / n; s > 0.04 || strikes == 0 {
				t.Errorf("%s: bolt on a guest in %.4f of the parties, want very rare (>0, <0.04)", icon, s)
			}
		}
	}
	for icon := range dawnLevels {
		r := rand.New(rand.NewSource(int64(len(icon)) * 31))
		dawns := 0
		for i := 0; i < n; i++ {
			if weatherPlanFor(r, icon, true, false, false, false).dawn {
				dawns++
			}
		}
		f := float64(dawns) / n
		lo := 0.26
		if stormLevels[icon] {
			lo = 0.24 // a party with storm has no dawn
		}
		if f < lo || f > 0.42 {
			t.Errorf("%s: dawn in %.3f of the parties, want about 1/3", icon, f)
		}
	}
	a := weatherPlanFor(rand.New(rand.NewSource(5)), "festival", true, false, false, false)
	b := weatherPlanFor(rand.New(rand.NewSource(5)), "festival", true, false, false, false)
	if a != b {
		t.Error("same seed, different weather")
	}
}

// No weather in any other party, in stealth, in RETO DE HOY, on the island or in the random-mode extras.
func TestWeatherExcludedLevels(t *testing.T) {
	g := layerGame(t, false)
	r := rand.New(rand.NewSource(99))
	for i := 0; i < len(levelDefs); i++ {
		d := levelDefs[i]
		campaign := i < campaignN
		for k := 0; k < 1500; k++ {
			p := weatherPlanFor(r, d.Icon, campaign, d.Day, false, false)
			if (p.storm && (!campaign || !stormLevels[d.Icon])) || (p.dawn && (!campaign || !dawnLevels[d.Icon])) {
				t.Fatalf("party %d %q (%s) got weather %+v", i+1, d.Title, d.Icon, p)
			}
		}
	}
	for k := 0; k < 3000; k++ {
		for icon := range stormLevels {
			if p := weatherPlanFor(r, icon, true, false, true, false); p.storm || p.dawn {
				t.Fatal("weather in stealth")
			}
			if p := weatherPlanFor(r, icon, true, false, false, true); p.storm || p.dawn {
				t.Fatal("weather in RETO DE HOY")
			}
			if p := weatherPlanFor(r, icon, false, false, false, false); p.storm || p.dawn {
				t.Fatal("weather outside the campaign (island, random extras)")
			}
		}
	}
	// at run time: the island never gets it, whatever the dice
	ds, dd, li, lv, wr := dbgStorm, dbgDawn, g.levelIdx, g.lvl, weatherRng
	t.Cleanup(func() { dbgStorm, dbgDawn, g.levelIdx, g.lvl, weatherRng, g.wth = ds, dd, li, lv, wr, weatherState{} })
	dbgStorm, dbgDawn = 0, -1
	weatherRng = rand.New(rand.NewSource(3))
	g.levelIdx = specialJurassic
	g.lvl = buildLevel(jurassicLevel())
	for k := 0; k < 2000; k++ {
		g.weatherLevelReset()
		if g.wth.plan.storm || g.wth.plan.dawn {
			t.Fatal("weather on the island")
		}
	}
	if e := g.dawnE(); e != 0 {
		t.Fatalf("dawn %.2f without a dawn", e)
	}
}

// A storm runs once: it starts at its second, grows softly, has flashes, ends softly and never comes back; the drops
// never cover a player (1 and 2 players); the step and the batch of drops allocate nothing.
func TestStormRunsOnceSoftlyAndSparesPlayers(t *testing.T) {
	for _, two := range []bool{false, true} {
		g := weatherGame(t, two, "festival")
		g.wth = weatherState{thunder: -1, plan: weatherPlan{storm: true, at: 20, dur: 50, seed: 11}}
		g.wth.r = rand.New(rand.NewSource(11))
		dt := 1.0 / 60
		g.time = 0
		var prevK float64
		flashes, started := 0, false
		for step := 0; step < 60*90; step++ {
			g.time += dt
			was := g.wth.flashT
			g.updateWeather(dt)
			w := &g.wth
			if g.time < 20 && w.k > 0 {
				t.Fatal("rain before its second")
			}
			if w.started {
				started = true
			}
			if math.Abs(w.k-prevK) > 0.02 {
				t.Fatalf("intensity jumps %.3f -> %.3f at %.1f s", prevK, w.k, g.time)
			}
			prevK = w.k
			if w.flashT < was {
				flashes++
			}
			if w.done && w.k != 0 {
				t.Fatal("rain after the end")
			}
			if w.k > 0.5 {
				var keep [2]weatherRect
				nk := 0
				for _, p := range g.players() {
					x, y := g.wx(p.pos.X), g.wy(p.pos.Y)
					keep[nk] = weatherRect{x - 12, y - 40, x + 12, y + 4}
					nk++
				}
				n := w.rainBatch(w.k, keep, nk)
				if n == 0 {
					t.Fatal("full storm with no drops")
				}
				area := 0.0
				for q := 0; q < n; q++ {
					v := rainVs[q*4 : q*4+4]
					x0, y0 := float64(v[0].DstX), float64(v[0].DstY)
					x1, y1 := float64(v[3].DstX), float64(v[3].DstY)
					area += math.Abs(float64(v[1].DstX-v[0].DstX)) * math.Abs(y1-y0+0)
					if v[0].ColorA > 0.65 {
						t.Fatalf("a drop too opaque: %.2f", v[0].ColorA)
					}
					for j := 0; j < nk; j++ {
						r := keep[j]
						if math.Max(x0, x1) >= r.x0 && math.Min(x0, x1) <= r.x1 && math.Max(y0, y1) >= r.y0 && math.Min(y0, y1) <= r.y1 {
							t.Fatalf("a drop over player %d", j+1)
						}
					}
				}
				if f := area / (screenW * screenH); f > 0.015 {
					t.Fatalf("the rain covers %.3f of the screen", f)
				}
			}
		}
		if !started || !g.wth.done {
			t.Fatalf("2P=%v: the storm did not run to its end", two)
		}
		if flashes < 2 {
			t.Errorf("2P=%v: %d lightning flashes in 50 s, want some", two, flashes)
		}
		// never twice: more time, nothing
		for step := 0; step < 60*60; step++ {
			g.time += dt
			g.updateWeather(dt)
			if g.wth.k > 0 || !g.wth.done {
				t.Fatal("the storm came back")
			}
		}
		// no allocation per frame
		w := &g.wth
		w.done, w.k = false, 1
		keep := [2]weatherRect{{100, 100, 130, 140}}
		if a := testing.AllocsPerRun(200, func() { w.stepDrops(dt); w.rainBatch(1, keep, 1) }); a != 0 {
			t.Errorf("rain step allocates %.1f per frame", a)
		}
		// it draws in both modes without trouble
		g.wth.k = 1
		g.drawWeather(ebiten.NewImage(screenW, screenH))
	}
}

// The bolt on a guest: one guest dies (a burnt skeleton), nobody's kill: no stars, no points, no police.
func TestStormStrikeIsAnActOfNature(t *testing.T) {
	g := weatherGame(t, false, "beach")
	g.wth = weatherState{thunder: -1, plan: weatherPlan{storm: true, at: 1, dur: 40, strike: true, strikeAt: 12, seed: 4}}
	g.wth.r = rand.New(rand.NewSource(4))
	alive, stars, kills, pts, wanted := g.aliveCount(), g.stars, g.kills, g.points, g.wanted
	nb := len(g.bodies)
	dt := 1.0 / 60
	g.time = 0
	for step := 0; step < 60*20; step++ {
		g.time += dt
		g.updateWeather(dt)
	}
	if !g.wth.struck {
		t.Fatal("no bolt")
	}
	if g.aliveCount() != alive-1 || len(g.bodies) != nb+1 {
		t.Fatalf("the bolt killed %d (bodies +%d), want 1", alive-g.aliveCount(), len(g.bodies)-nb)
	}
	b := g.bodies[len(g.bodies)-1]
	if b.byPlayer || !b.skeleton {
		t.Errorf("body byPlayer=%v skeleton=%v, want an act of nature (skeleton, not the player's)", b.byPlayer, b.skeleton)
	}
	if !outdoorAt(g.lvl.Def, b.pos) {
		t.Error("the bolt fell under a roof")
	}
	if g.stars != stars || g.kills != kills || g.points != pts || g.wanted != wanted {
		t.Errorf("the bolt counted for the player: stars %d->%d kills %d->%d points %d->%d wanted %v->%v", stars, g.stars, kills, g.kills, pts, g.points, wanted, g.wanted)
	}
}

// ---- the look: "que se vea bien el nivel" --------------------------------------------------------------------------

// washLum: mean luminance / R-B of a frame after a premultiplied wash of colour c with alpha a (both are linear).
func washLum(lum, warm float64, c color.RGBA, a float64) (float64, float64) {
	cl := 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)
	return lum*(1-a) + cl*a, warm*(1-a) + (float64(c.R)-float64(c.B))*a
}

func washCol(c, w color.RGBA, a float64) color.RGBA {
	m := func(x, y uint8) uint8 { return uint8(math.Round(float64(x)*(1-a) + float64(y)*a)) }
	return color.RGBA{m(c.R, w.R), m(c.G, w.G), m(c.B, w.B), 255}
}

// rawLook: the frame of levelLook (italy_test.go) without any lighting wash.
func rawLook(d *LevelDef) (float64, float64) {
	raw := *d
	raw.Day, raw.Tint = true, color.RGBA{}
	return levelLook(&raw)
}

// The storm keeps the level readable: the frame loses at most 12 % of its mean luminance against the same party
// without storm, and the contrast of the people against every floor keeps at least 85 %.
func TestStormKeepsTheLevelVisible(t *testing.T) {
	layerGame(t, false)
	var people []color.RGBA
	for i, f := range famousPresets {
		if i >= 24 {
			break
		}
		s := f(rand.New(rand.NewSource(int64(i))))
		people = append(people, s.Skin, s.Shirt, s.Pants, s.Hair)
	}
	for icon := range stormLevels {
		d := levelDefs[campaignIdx(icon)]
		rl, rw := rawLook(d)
		base, aOut := nightWash(d, 0)
		nl, nw := washLum(rl, rw, base, aOut/255)
		if ol, ow := levelLook(d); math.Abs(ol-nl) > 0.6 || math.Abs(ow-nw) > 0.6 {
			t.Fatalf("%s: the wash model does not match levelLook (%.1f vs %.1f)", icon, nl, ol)
		}
		sl, _ := washLum(nl, nw, stormShadeCol, stormShadeA)
		// the brightest flash moment also stays a flash, not a white-out
		fl, _ := washLum(sl, 0, stormFlashCol, stormFlashA)
		t.Logf("%-8s luminance night %.1f  storm %.1f (%.0f %%)  flash %.1f", icon, nl, sl, 100*sl/nl, fl)
		if sl < nl*0.88 {
			t.Errorf("%s: the storm darkens the level to %.0f %% of its luminance (min 88 %%)", icon, 100*sl/nl)
		}
		if fl > 150 {
			t.Errorf("%s: the lightning flash blows the frame out (%.1f)", icon, fl)
		}
		worst := 10.0
		for _, f := range d.Floors {
			fc, ok := floorAvg(f.Kind)
			if !ok {
				continue
			}
			a := aOut
			if floorIndoor(f.Kind) {
				a = aOut * 0.3
			}
			fn := overNight(fc, d, a)
			fs := washCol(fn, stormShadeCol, stormShadeA)
			for _, pc := range people {
				pn := overNight(pc, d, a)
				ps := washCol(pn, stormShadeCol, stormShadeA)
				c0, c1 := lumContrast(pn, fn), lumContrast(ps, fs)
				if c0 < 1.5 {
					continue // already the same tone at night: the outline carries it
				}
				worst = math.Min(worst, c1/c0)
			}
		}
		t.Logf("%-8s people contrast kept: %.0f %% (worst pair)", icon, 100*worst)
		if worst < 0.85 {
			t.Errorf("%s: the storm takes the people's contrast down to %.0f %% (min 85 %%)", icon, 100*worst)
		}
	}
	// the measure notices a storm that would cover the level (a 45 % shade)
	d := levelDefs[campaignIdx("festival")]
	rl, rw := rawLook(d)
	base, aOut := nightWash(d, 0)
	nl, nw := washLum(rl, rw, base, aOut/255)
	if bad, _ := washLum(nl, nw, stormShadeCol, 0.45); bad >= nl*0.88 {
		t.Error("the check would not catch a dark storm")
	}
}

// The usual night is untouched (dawn 0 = the old formula); the dawn only goes forward, ends lighter but below a soft
// day, and neutral (not yellow, the user rejected a golden wash in ITALIA).
func TestDawnMonotonicAndInRange(t *testing.T) {
	layerGame(t, false)
	for i := 0; i < campaignN; i++ {
		d := levelDefs[i]
		if d.Day {
			continue
		}
		base, a := nightWash(d, 0)
		oa, _ := nightAlphas(d)
		ob := mixc(rgb(0x14142a), color.RGBA{d.Tint.R, d.Tint.G, d.Tint.B, 255}, 0.5)
		if base != ob || a != oa {
			t.Fatalf("%s: the night without dawn changed", d.Title)
		}
	}
	for icon := range dawnLevels {
		d := levelDefs[campaignIdx(icon)]
		rl, rw := rawLook(d)
		prev := -1.0
		var l0, l1, w1 float64
		for e := 0.0; e <= 1.0001; e += 0.05 {
			base, a := nightWash(d, e)
			l, w := washLum(rl, rw, base, a/255)
			if l < prev-0.01 {
				t.Fatalf("%s: the dawn gets darker at %.2f (%.1f < %.1f)", icon, e, l, prev)
			}
			if e == 0 {
				l0 = l
			}
			prev, l1, w1 = l, l, w
		}
		t.Logf("%-8s dawn luminance %.1f -> %.1f  R-B %.1f", icon, l0, l1, w1)
		if l1 < l0+6 {
			t.Errorf("%s: the dawn hardly shows (%.1f -> %.1f)", icon, l0, l1)
		}
		if l1 > 110 {
			t.Errorf("%s: the dawn ends too bright (%.1f; a soft day is <= 118)", icon, l1)
		}
		if w1 > 12 {
			t.Errorf("%s: the dawn is yellowish (R-B %.1f, max 12)", icon, w1)
		}
	}
	// the party clock: never backwards, full by the expected time or by the guests out, no jump
	g := weatherGame(t, false, "festival")
	g.wth = weatherState{thunder: -1, plan: weatherPlan{dawn: true}}
	dt := 1.0 / 60
	g.time = 0
	prev := 0.0
	for step := 0; step < int(dawnFullS*60)+600; step++ {
		g.time += dt
		g.updateWeather(dt)
		k := g.wth.dawnK
		if k < prev || k-prev > dawnRate*dt+1e-9 {
			t.Fatalf("dawn %.4f -> %.4f at %.1f s", prev, k, g.time)
		}
		prev = k
	}
	if g.dawnE() < 0.99 {
		t.Errorf("no full dawn after the expected time (%.2f)", g.dawnE())
	}
	if lk := g.dawnLampK(); lk > 0.5 {
		t.Errorf("the street lamps do not dim at full dawn (%.2f)", lk)
	}
	if dawnTarget(10, 0.9) < 0.9 {
		t.Error("the dawn does not follow the guests already out")
	}
}

// Dawn and storm only change the drawing: the sight of guests, guards and cameras is the same.
func TestWeatherDoesNotChangeSight(t *testing.T) {
	g, c := camGame(t)
	g.snd = &Sounds{muted: true, sfx: map[string][]byte{}}
	t.Cleanup(func() { g.wth = weatherState{} })
	q := Vec{200, 220}
	type sight struct{ cam, guest bool }
	look := func() sight {
		return sight{c.sees(g, q), g.sees(Vec{200, 150}, DirDown, q, guestRange, guestDot)}
	}
	before := look()
	g.wth = weatherState{plan: weatherPlan{dawn: true, storm: true}, dawnK: 1, k: 1}
	if look() != before {
		t.Error("the weather changed what people and cameras see")
	}
}

// -storm and -dawn force the weather (and nothing else) for the captures.
func TestWeatherDebugFlags(t *testing.T) {
	g := weatherGame(t, false, "eiffel")
	dbgStorm = 2
	g.weatherLevelReset()
	if p := g.wth.plan; !p.storm || p.at > 5 || !p.strike {
		t.Errorf("-storm 2: %+v", p)
	}
	dbgStorm, dbgDawn = 0, 100
	g.weatherLevelReset()
	g.updateWeather(0.1)
	if !g.wth.plan.dawn || g.dawnE() < 0.99 {
		t.Errorf("-dawn 100: plan %+v dawn %.2f", g.wth.plan, g.dawnE())
	}
}

// The lightning bolt never crosses the HUD band (panels, stars, the news line) and the far flashes stay in the middle.
func TestStormBoltClearOfHUD(t *testing.T) {
	g := weatherGame(t, false, "stadium")
	g.wth = weatherState{thunder: -1, plan: weatherPlan{storm: true, at: 0, dur: 60}, r: rand.New(rand.NewSource(8))}
	for i := 0; i < 500; i++ {
		g.stormFlash(-1, -1, 0.5)
		w := &g.wth
		if w.boltY <= stormBoltTop+16 || w.boltX < hudX+hudW+16 || w.boltX2 < 320 || w.boltX2 > screenW-320 {
			t.Fatalf("far bolt at top x %.0f, end %.0f,%.0f", w.boltX, w.boltX2, w.boltY)
		}
	}
	for _, x := range []float64{20, 120, 900} { // a bolt on a guest anywhere: its top never over the panel
		g.stormFlash(x, 300, 0.1)
		if g.wth.boltX < hudX+hudW+16 {
			t.Errorf("bolt on a guest at x %.0f starts over the HUD panel (x %.0f)", x, g.wth.boltX)
		}
	}
	if stormBoltTop < hudY+hudBaseH {
		t.Errorf("the bolt starts at y %.0f, over the HUD panel (to %.0f)", stormBoltTop, float64(hudY+hudBaseH))
	}
}
