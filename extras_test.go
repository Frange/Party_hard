package main

import (
	"encoding/json"
	"math"
	"math/rand"
	"strings"
	"testing"
)

// Tests of the second batch of "sometimes" things: stray dog, skaters, stashes, memory of the guests, CARRERA page,
// prizes of the 2P newspaper and the help line. Names: TestDog*, TestSkater*, TestStash*, TestMemory*, TestCareer*,
// TestAward*, TestHelpUseNear*.

const rngProbeSeed = 424242

// rngIntact: g.rng is given a known seed; after fn it must still be at its first value.
func rngIntact(t *testing.T, g *Game, what string, fn func()) {
	t.Helper()
	g.rng = rand.New(rand.NewSource(rngProbeSeed))
	fn()
	if got, want := g.rng.Int63(), rand.New(rand.NewSource(rngProbeSeed)).Int63(); got != want {
		t.Errorf("%s moved g.rng", what)
	}
}

// ---- stray dog -------------------------------------------------------------------------------------------

// dogSetup: the first eligible level where a corpse is lying and the dog really starts. The body is a normal one.
func dogSetup(t *testing.T, two bool) (*Game, *Body) {
	t.Helper()
	probe := partyGame(t)
	for _, idx := range peEligible(probe, "dog") {
		g := peLoad(t, idx, two)
		sp, ok := g.pevSpot(rand.New(rand.NewSource(3)), 14, true)
		if !ok {
			continue
		}
		g.spawnBody(sp, g.p.cs, DirDown)
		b := g.bodies[len(g.bodies)-1]
		b.fallT = 0
		peForce(g, "dog")
		g.time = 3
		for s := 0; s < 20 && g.partyRunning("dog") == nil; s++ {
			peStep(g, 1.0/30, false)
		}
		if g.partyRunning("dog") != nil {
			return g, b
		}
	}
	t.Fatal("dog: it never started in the levels that fit it")
	return nil, nil
}

func TestDogEligibility(t *testing.T) {
	g := partyGame(t)
	idxs := peEligible(g, "dog")
	if len(idxs) < 5 || len(idxs) > 16 {
		t.Fatalf("the dog fits %d parties (%v): outdoors only, not all of them", len(idxs), idxs)
	}
	for _, i := range idxs {
		ic := campaignLevels[i].icon
		if partyIndoorIcons[ic] || ic == "boat" || ic == "hockey" {
			t.Errorf("a stray dog in %q", ic)
		}
	}
	e := partyByID("dog")
	if e.Prob < 0.10 || e.Prob > 0.40 || e.AtMax <= e.AtMin || e.Cooldown <= 0 || !strings.Contains(partyList(), "dog") {
		t.Error("bad chance or window")
	}
}

func TestDogNeedsACorpse(t *testing.T) {
	probe := partyGame(t)
	idxs := peEligible(probe, "dog")
	g := peLoad(t, idxs[0], false)
	g.bodies = nil
	dbgPartyEvent = ""
	g.pev = pevState{ready: true, traits: partyTraitsOf(g.lvl.Def, g.levelIdx), plan: []*partyPlanned{{ev: partyByID("dog"), at: 0}}}
	for s := 0; s < 90; s++ {
		peStep(g, 1.0/30, false)
	}
	if g.partyRunning("dog") != nil {
		t.Fatal("the dog came with no corpse")
	}
}

func TestDogDragsTheBodyAway(t *testing.T) {
	g, b := dogSetup(t, false)
	kills, alive, animals, stars, pts := g.kills, g.aliveCount(), len(g.animals), g.stars, g.points
	var took bool
	var last Vec
	edge := func(p Vec) float64 { return math.Min(math.Min(p.X, g.lvl.Def.W-p.X), math.Min(p.Y, g.lvl.Def.H-p.Y)) }
	start := edge(b.pos)
	rngIntact(t, g, "the dog", func() {
		for s := 0; s < 30*150 && g.partyRunning("dog") != nil; s++ {
			peStep(g, 1.0/30, false)
			if a := g.partyRunning("dog"); a != nil {
				st := a.D.(*dogState)
				if st.took {
					took = true
					if !b.carried {
						t.Fatal("while dragged the body must be 'carried' (invisible to witnesses and police)")
					}
					if dist(st.dog.pos, b.pos) > 14 {
						t.Fatalf("the body is %.0f px from the dog", dist(st.dog.pos, b.pos))
					}
					if len(g.animals) != animals {
						t.Fatal("the dog is in g.animals (animalMax)")
					}
				}
				last = st.dog.pos
			}
		}
	})
	if !took {
		t.Fatal("the dog did not take the body away")
	}
	if e := edge(last); e > math.Max(60, start) {
		t.Errorf("the dog left %.0f px from the edge (the body was %.0f)", e, start)
	}
	for _, x := range g.bodies {
		if x == b {
			t.Fatal("the body is still there")
		}
	}
	if g.kills != kills || g.aliveCount() != alive || len(g.animals) != animals || g.stars != stars || g.points != pts {
		t.Errorf("counters moved: kills %d/%d alive %d/%d animals %d/%d points %d/%d", g.kills, kills, g.aliveCount(), alive, len(g.animals), animals, g.points, pts)
	}
}

func TestDogKilledGivesSmallPointsAndNoBlame(t *testing.T) {
	g, b := dogSetup(t, false)
	for s := 0; s < 30*120 && !g.partyRunning("dog").D.(*dogState).took; s++ {
		peStep(g, 1.0/30, false)
	}
	st := g.partyRunning("dog").D.(*dogState)
	if !st.took {
		t.Skip("(the dog never reached the body in this level)")
	}
	pts, wanted, an := g.points, g.wanted, g.progress.Stats["animals"]
	g.p.pos = st.dog.pos
	if !g.dogStab(g.p, 30) {
		t.Fatal("the blow did not reach the dog")
	}
	if d := g.points - pts; d != dogKillPts || d <= 0 || d > 10 {
		t.Errorf("points %d, want %d", d, dogKillPts)
	}
	if g.wanted != wanted || g.progress.Stats["animals"] != an || g.suspect != nil {
		t.Error("killing the dog must not blame the player or count as an animal kill")
	}
	if b.carried {
		t.Error("the body must be dropped when the dog dies")
	}
	for s := 0; s < 90 && g.partyRunning("dog") != nil; s++ {
		peStep(g, 1.0/30, false)
	}
	if g.partyRunning("dog") != nil {
		t.Error("the event did not end")
	}
	if g.dogStab(g.p, 30) {
		t.Error("a dead dog cannot be hit twice")
	}
}

func TestDogTwoPlayers(t *testing.T) {
	g, _ := dogSetup(t, true)
	if len(g.players()) != 2 {
		t.Fatal("not 2P")
	}
	for s := 0; s < 30*150 && g.partyRunning("dog") != nil; s++ {
		peStep(g, 1.0/30, false)
	}
	if g.partyRunning("dog") != nil {
		t.Error("the dog never left")
	}
}

// ---- skaters ---------------------------------------------------------------------------------------------

func TestSkaterEligibility(t *testing.T) {
	g := partyGame(t)
	idxs := peEligible(g, "skaters")
	if len(idxs) != 1 || campaignLevels[idxs[0]].icon != "hockey" {
		t.Fatalf("the skaters fit %v, want only the hockey rink", idxs)
	}
	e := partyByID("skaters")
	if e.Prob < 0.30 || e.Prob > 0.40 || e.Cooldown <= 0 || e.MaxRuns < 2 {
		t.Error("the highest chance the frame allows (10-40 %), with a window and a pause")
	}
}

func TestSkatersGlideOnTheIce(t *testing.T) {
	for _, two := range []bool{false, true} {
		peRetry(t, "skaters", two, func(g *Game) string {
			a := g.partyRunning("skaters")
			st := a.D.(*skateState)
			if len(st.sk) < skMin {
				return "too few skaters"
			}
			for _, k := range st.sk {
				if !skIsSkater(k.m.gu) {
					return "a guest without skates skates"
				}
			}
			start := map[*Guest]Vec{}
			for _, k := range st.sk {
				start[k.m.gu] = k.m.gu.pos
			}
			box := skBox(g)
			moved, spun, fell := 0.0, false, false
			rngIntact(t, g, "the skaters", func() {
				for s := 0; s < 30*60 && g.partyRunning("skaters") != nil; s++ {
					g.time += 1.0 / 30
					g.updateParty(1.0 / 30)
					if a2 := g.partyRunning("skaters"); a2 != nil {
						for _, k := range st.sk {
							gu := k.m.gu
							if !box.Has(gu.pos.X, gu.pos.Y) || !g.lvl.OnIce(gu.pos) {
								t.Fatalf("a skater left the ice at %v", gu.pos)
							}
							spun = spun || k.mode == skSpin
							fell = fell || k.mode == skDown
						}
					}
				}
			})
			for gu, p := range start {
				moved = math.Max(moved, dist(gu.pos, p))
			}
			if moved < 30 {
				return "they did not move"
			}
			_ = spun
			_ = fell
			if g.partyRunning("skaters") != nil {
				return "it did not end"
			}
			for _, k := range st.sk {
				if peHeld(k.m.gu) || k.m.gu.vel != (Vec{}) {
					return "a skater was not let go"
				}
				if p := g.lvl.PathG(k.m.gu.pos, g.lvl.Def.Start); p == nil {
					return "a released skater cannot reach the start"
				}
			}
			return ""
		})
	}
}

func TestSkaterFallsAndSpinsHappen(t *testing.T) {
	// the dice of the party decide, but over many runs both a pirouette and a fall must appear
	spun, fell := false, false
	for seed := int64(1); seed <= 6 && !(spun && fell); seed++ {
		g := peStartSeed(t, "skaters", false, seed)
		st := g.partyRunning("skaters").D.(*skateState)
		for s := 0; s < 30*80 && g.partyRunning("skaters") != nil; s++ {
			g.time += 1.0 / 30
			g.updateParty(1.0 / 30)
			for _, k := range st.sk {
				spun = spun || k.mode == skSpin
				fell = fell || k.mode == skDown
			}
		}
	}
	if !spun || !fell {
		t.Errorf("pirouette %v, fall %v", spun, fell)
	}
}

// ---- stashes ---------------------------------------------------------------------------------------------

func stashGame(t *testing.T, idx int) *Game {
	t.Helper()
	g := peLoad(t, idx, false)
	prevSeed := stashSeed
	stashSeed = 99
	stash = stashState{}
	t.Cleanup(func() { stashSeed, stash = prevSeed, stashState{} })
	return g
}

func TestStashRatioAndKinds(t *testing.T) {
	tot, with := 0, 0
	for i := range campaignLevels {
		g := stashGame(t, i)
		for seed := int64(1); seed <= 40; seed++ { // many dice over the same containers
			stash, stashSeed = stashState{}, seed
			stashRoll(g)
			for _, pr := range g.lvl.Props {
				if stashKinds[pr.Name] {
					tot++
				}
			}
			for pr := range stash.item {
				with++
				if !stashKinds[pr.Name] {
					t.Fatalf("a %q hides an object", pr.Name)
				}
			}
		}
	}
	if tot < 40*20 {
		t.Fatalf("only %d containers in the whole campaign", tot)
	}
	if r := float64(with) / float64(tot); r < 0.10 || r > 0.28 {
		t.Errorf("%d of %d containers hide something (%.0f %%), want 15-20 %%", with, tot, r*100)
	}
}

func TestStashFixedPerLevelAndDiceUntouched(t *testing.T) {
	g := stashGame(t, 0)
	rngIntact(t, g, "the stashes", func() {
		stashRoll(g)
		first := len(stash.item)
		for i := 0; i < 5; i++ {
			g.stashItems(nil, nil)
		}
		stashRoll(g)
		if len(stash.item) != first {
			t.Error("the stashes changed within the level")
		}
	})
}

func TestStashTakeWithE(t *testing.T) {
	g := stashGame(t, 0)
	stashRoll(g)
	var cont *Prop
	for _, pr := range g.lvl.Props {
		if stashKinds[pr.Name] {
			cont = pr
			break
		}
	}
	if cont == nil {
		t.Fatal("no container in the first party")
	}
	near := func() { g.p.pos = Vec{cont.Foot.X + cont.Foot.W/2, cont.Foot.Y + cont.Foot.H + 3} }
	// money: points only
	stash.item[cont] = &stashTable[2]
	near()
	pts := g.points
	if !g.stashInteract(g.p) || g.points-pts != stashTable[2].pts || stash.item[cont] != nil {
		t.Fatalf("the banknote: points %d, still there %v", g.points-pts, stash.item[cont] != nil)
	}
	if g.stashInteract(g.p) {
		t.Error("the same container gave twice")
	}
	// poison: the normal pick-up
	stash.item[cont] = &stashTable[0]
	g.p.item, g.p.spare, g.p.doses = "", "", 0
	if !g.stashInteract(g.p) || g.p.item != "poison" || g.p.doses != poisonDoses {
		t.Errorf("poison: item %q doses %d", g.p.item, g.p.doses)
	}
	// hands full (a body): it is not taken
	stash.item[cont] = &stashTable[3]
	g.p.carrying = &Body{}
	if g.stashInteract(g.p) {
		t.Error("taken with a body in the hands")
	}
	g.p.carrying = nil
	// far away: nothing
	g.p.pos = Vec{cont.Foot.X + 300, cont.Foot.Y + 300}
	if g.stashInteract(g.p) {
		t.Error("taken from far away")
	}
}

func TestStashGlintIsRareAndTexts(t *testing.T) {
	n := 0
	for i := 0; i < 3000; i++ {
		if stashGlint(float64(i)*0.1, 3) {
			n++
		}
	}
	if f := float64(n) / 3000; f < 0.03 || f > 0.12 {
		t.Errorf("the glint is on %.0f %% of the time", f*100)
	}
	tot := 0
	for _, s := range stashTable {
		tot += s.weight
		if s.id != "poison" && s.id != "disguise" && (s.say == "" || s.pts <= 0 || s.pts > 40) {
			t.Errorf("stash %s: say %q pts %d", s.id, s.say, s.pts)
		}
	}
	if tot < 80 || len(stashTable) < 8 {
		t.Error("too few kinds")
	}
}

// ---- memory of the guests ------------------------------------------------------------------------------

func TestMemoryRecognisesYouNextTime(t *testing.T) {
	idx := 0
	peKeepMemory = true
	t.Cleanup(func() { peKeepMemory = false })
	g := peLoad(t, idx, false)
	t.Cleanup(func() { memState.seen, memState.lvl, memState.pick, memState.spoke = nil, nil, nil, nil })
	memState.seen = nil
	// nobody saw anything: nothing is remembered
	g.time = 40
	g.memRemember()
	if memState.seen[idx] {
		t.Fatal("remembered with no witness")
	}
	// two witnesses survive; one witness is dead
	n := 0
	for _, gu := range g.guests {
		if n < 3 {
			gu.sawKiller = true
			if n == 2 {
				gu.dead = true
			}
			n++
		}
	}
	g.time = 5 // a level that barely ran does not count
	g.memRemember()
	if memState.seen[idx] {
		t.Fatal("remembered after 5 s")
	}
	g.time = 40
	g.memRemember()
	if !memState.seen[idx] {
		t.Fatal("a witness survived and nothing was remembered")
	}
	// the next party of the same level
	g = peLoad(t, idx, false)
	if !memState.seen[idx] {
		t.Fatal("loadLevel lost the memory")
	}
	wanted, stars, pts := g.wanted, g.stars, g.points
	g.time = 1
	rngIntact(t, g, "the memory", func() {
		peStep(g, 1.0/30, false)
		if len(memState.pick) < 2 || len(memState.pick) > 3 {
			t.Fatalf("%d guests picked", len(memState.pick))
		}
		for _, gu := range memState.pick {
			gu.state, gu.timer, gu.icon = gIdle, 0, ""
		}
		for _, gu := range memState.pick {
			g.p.pos = gu.pos
			g.time += memGap + 1
			peStep(g, 1.0/30, false)
		}
	})
	said := 0
	for _, gu := range memState.pick {
		if gu.icon != "" {
			said++
			ok := false
			for _, l := range memLines {
				ok = ok || l == gu.icon
			}
			if !ok {
				t.Errorf("odd line %q", gu.icon)
			}
		}
	}
	if said == 0 {
		t.Log("(nobody could speak: the guests were not in sight of the player in this party)")
	}
	if g.wanted != wanted || g.stars != stars || g.points != pts || g.suspect != nil {
		t.Error("the memory has consequences")
	}
	if len(memLines) < 20 {
		t.Errorf("%d lines, want 20 or more", len(memLines))
	}
	// the party after that one is not remembered again (the next loadLevel recomputes: nobody saw anything)
	g.time = 40
	for _, gu := range g.guests {
		gu.sawKiller = false
	}
	g.memRemember()
	if memState.seen[idx] {
		t.Error("the memory did not clear")
	}
}

func TestMemoryDoesNotTouchProgress(t *testing.T) {
	g := auditGame(t)
	prev := progressFrozen
	progressFrozen = true
	t.Cleanup(func() { progressFrozen = prev })
	before, _ := json.Marshal(g.progress)
	memSet(3, true)
	memSet(3, false)
	after, _ := json.Marshal(g.progress)
	if string(before) != string(after) {
		t.Error("the memory writes to Progress")
	}
}

// ---- career page -------------------------------------------------------------------------------------------

func TestCareerRowsAndOldSaves(t *testing.T) {
	g := auditGame(t)
	prev, prevF := g.progress, progressFrozen
	t.Cleanup(func() { g.progress, progressFrozen = prev, prevF })
	progressFrozen = true
	// a save written before this page existed: no "secs", nothing else changed
	old := `{"unlocked":5,"stats":{"played":12,"kills":40,"traps":3,"poison":2,"wins":7,"won_0":1,"won_1":1,"earned":9000,"best":1200,"chain":6,"deaths":4},"trophies":{},"wallet":30,"escapes":3}`
	var p Progress
	if err := json.Unmarshal([]byte(old), &p); err != nil {
		t.Fatal(err)
	}
	rt, _ := json.Marshal(p)
	g.progress = p
	left, right := g.careerRows()
	if len(left)+len(right) < 14 {
		t.Fatalf("%d rows", len(left)+len(right))
	}
	got := map[string]string{}
	for _, r := range append(left, right...) {
		got[r[0]] = r[1]
	}
	for k, v := range map[string]string{"Partidas jugadas": "12", "Bajas totales": "45", "Esposas evitadas": "3", "Fiestas ganadas": "7",
		"Tiempo jugado": "0 min 00 s", "Puntos ganados": "9000", "Mejor cadena": "6", "Veces que has caído": "4"} {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
	if !strings.HasPrefix(got["Niveles completados"], "2 / ") {
		t.Errorf("levels %q", got["Niveles completados"])
	}
	// every label has an English version
	loadLanguages()
	for _, r := range append(left, right...) {
		if _, ok := langDataEN[r[0]]; !ok {
			t.Errorf("no English for %q", r[0])
		}
	}
	if _, ok := langDataEN["CARRERA"]; !ok {
		t.Error("no English for the tab")
	}
	// the time counter
	g = peLoad(t, 0, false)
	careerFrac = 0
	g.progress.Stats = map[string]int{}
	for i := 0; i < 130; i++ {
		g.careerTick(1.0 / 60)
	}
	if g.progress.Stats["secs"] != 2 {
		t.Errorf("secs %d after 130/60 s", g.progress.Stats["secs"])
	}
	// nothing new in Progress's own fields: it round-trips like before (omitempty keeps old files identical)
	var q Progress
	if err := json.Unmarshal(rt, &q); err != nil || q.Escapes != 3 || q.Stats["played"] != 12 || q.Wallet != 30 {
		t.Errorf("the save does not round-trip: %+v", q)
	}
	if careerTime(3725) != "1 h 02 min" || careerTime(125) != "2 min 05 s" {
		t.Errorf("time formats %q %q", careerTime(3725), careerTime(125))
	}
}

func TestCareerTabFitsAndDraws(t *testing.T) {
	g := auditGame(t)
	old := uiLang
	t.Cleanup(func() { uiLang = old })
	loadLanguages()
	if goalTabs[tabCareer].name != "CARRERA" || tabCount != 6 {
		t.Fatal("the tab is not the sixth page")
	}
	for _, lang := range []string{"es", "en"} {
		uiLang = lang
		r := g.goalTabRects()
		last := r[tabCareer]
		if last[0]+last[2] > 760 {
			t.Errorf("%s: the tab row ends at %.0f (the counter starts at 770)", lang, last[0]+last[2])
		}
		line := tr(g.copEscapeLine())
		if end := last[0] + last[2]; end > 924-math.Ceil(g.textW(line, 1))-8 {
			t.Errorf("%s: the tabs (end %.0f) touch the line of the escapes (%q)", lang, end, line)
		}
	}
}

// ---- prizes of the 2P newspaper ----------------------------------------------------------------------------

func TestAwardsDistinctDeterministicAndEnough(t *testing.T) {
	if len(paperAwards) < 12 {
		t.Fatalf("%d prizes, want 12 or more", len(paperAwards))
	}
	seen := map[string]bool{}
	for _, a := range paperAwards {
		if a.es == "" || a.en == "" || seen[a.es] {
			t.Errorf("bad or repeated prize %q", a.es)
		}
		seen[a.es] = true
	}
	a := paperPlayer{kills: [kcN]int{kcKnife: 9}, chainMax: 6, clean: 4, seen: 1, coop: coopStats{points: 640, hits: 3, danceT: 30}}
	b := paperPlayer{kills: [kcN]int{kcBlast: 5}, chainMax: 3, clean: 2, seen: 9, coop: coopStats{points: 570, floorT: 20}}
	x, y := awardsFor(a, b), awardsFor(a, b)
	if x != y {
		t.Error("not deterministic")
	}
	if x[0] == x[1] || x[0][0] == "" || x[1][0] == "" {
		t.Errorf("same or empty prize: %v", x)
	}
	if x[0][0] != "El más sigiloso" && x[0][0] != "Mejor bailarín" && x[0][0] != "Mano dura" {
		t.Logf("player 1 got %q", x[0][0])
	}
	if x[1][0] != "El más caótico" && x[1][0] != "El más ruidoso" && x[1][0] != "Siesta oficial" {
		t.Logf("player 2 got %q", x[1][0])
	}
	// a perfectly even party still gives each of them a (different) prize
	e := awardsFor(paperPlayer{}, paperPlayer{})
	if e[0] == e[1] || e[0][0] == "" {
		t.Errorf("even party: %v", e)
	}
	// the same player never gets a prize he lost
	for i := 0; i < 200; i++ {
		r := rand.New(rand.NewSource(int64(i)))
		p := func() paperPlayer {
			return paperPlayer{kills: [kcN]int{kcKnife: r.Intn(9), kcPoison: r.Intn(5), kcTrap: r.Intn(5), kcBlast: r.Intn(5)}, chainMax: r.Intn(9), clean: r.Intn(6),
				seen: r.Intn(12), cuff: r.Intn(4), coop: coopStats{points: r.Intn(900), hits: r.Intn(6), floorT: float64(r.Intn(30)), danceT: float64(r.Intn(40)), helps: r.Intn(4), fives: r.Intn(3)}}
		}
		a, b := p(), p()
		w := awardsFor(a, b)
		if w[0] == w[1] {
			t.Fatalf("seed %d: both got %v", i, w[0])
		}
	}
}

func TestAwardsOnThePaper(t *testing.T) {
	for _, lang := range []string{"es", "en"} {
		inLang(t, lang, func() {
			one := paperSample(paperFacts{seed: 5}, false, false, 5)
			if one.stats[0].award != "" {
				t.Errorf("%s: 1 player has a prize ribbon", lang)
			}
			two := paperSample(paperFacts{seed: 5}, true, false, 5)
			if len(two.stats) != 2 {
				t.Fatal("2P without two boxes")
			}
			for i, sb := range two.stats {
				if !strings.HasPrefix(sb.award, map[string]string{"es": "PREMIO: ", "en": "PRIZE: "}[lang]) {
					t.Errorf("%s: box %d award %q", lang, i, sb.award)
				}
				if w := paperTextW(sb.award); w > pcLW-8-10 {
					t.Errorf("%s: %q is %d px, the ribbon holds %d", lang, sb.award, w, pcLW-18)
				}
				if h := statBoxH(sb); h > pbY1-pbY0 {
					t.Errorf("%s: the box is %d px tall, the column %d", lang, h, pbY1-pbY0)
				}
			}
			if two.stats[0].award == two.stats[1].award {
				t.Errorf("%s: same prize for both", lang)
			}
		})
	}
	// every prize fits the ribbon in both languages, with the longest prefix
	for _, a := range paperAwards {
		for _, s := range []string{"PREMIO: " + a.es, "PRIZE: " + a.en} {
			if w := paperTextW(s); w > pcLW-18 {
				t.Errorf("%q is %d px (max %d)", s, w, pcLW-18)
			}
		}
	}
	for _, f := range paperAwardFallback {
		for _, s := range []string{"PREMIO: " + f[0], "PRIZE: " + f[1]} {
			if w := paperTextW(s); w > pcLW-18 {
				t.Errorf("%q is %d px", s, w)
			}
		}
	}
}

// ---- help -----------------------------------------------------------------------------------------------

func TestHelpUseNearLine(t *testing.T) {
	found := false
	for _, m := range helpModes {
		if m.text == helpUseNear {
			found = true
			if m.act != actInteract || binds[m.act].Pad == noPad {
				t.Error("the uses of E must read the interact binding")
			}
		}
	}
	if !found {
		t.Fatal("not in the help")
	}
	g := auditGame(t)
	old := uiLang
	t.Cleanup(func() { uiLang = old })
	loadLanguages()
	// the row: up to three caps (two players + pad) and then the text, inside the panel
	for _, lang := range []string{"es", "en"} {
		uiLang = lang
		w := g.textW(helpUseNear, 1)
		if 20+3*40+10+w > helpPanW-40 {
			t.Errorf("%s: the line is %.0f px wide", lang, w)
		}
	}
	// the rows of MODOS Y OBJETOS end above the footer line
	my0 := helpModesY0
	last := my0 + 28 + float64(len(helpModes))*helpModeRowH + helpCapH // (the minimap row is one more)
	if last+12 > helpFooterY {
		t.Errorf("the last row ends at %.0f, the footer starts at %.0f (12 px apart needed)", last, helpFooterY)
	}
	if _, ok := langDataEN[helpUseNear]; !ok {
		t.Error("no English for the help line")
	}
}
