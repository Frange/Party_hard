package main

import (
	"math/rand"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// Tests of the owner's feedback of v1.4.0 (2026-11-12): police crews, stars, the POLICÍAS page, machine guns, the
// streak, trophies and medal toasts, the gold suit while cuffed and traps against fast guests.

// crewGame: a party of the first car level with an open call and nobody on scene; the player out of sight.
func crewGame(t *testing.T) *Game {
	t.Helper()
	idx := 0
	for i := 0; i < len(campaignLevels); i++ {
		if levelDefs[i] != nil && levelDefs[i].Police == "car" {
			idx = i
			break
		}
	}
	g := peLoad(t, idx, false)
	snapshotGlobals(t)
	prevJail := jailHold
	t.Cleanup(func() {
		jailHold = prevJail
		g.cops, g.veh, g.bodies = nil, nil, nil
		g.policeCalled, g.swatKills, g.stars, g.starFloor, g.wanted = false, 0, 0, 0, 0
		g.p.hidden = false
	})
	jailHold = false
	g.cops, g.veh, g.bodies = nil, nil, nil
	g.policeCalled, g.wanted, g.suspect = true, 0, nil
	g.p.hidden = true
	return g
}

// addCrew: a parked vehicle of unit u with n officers sweeping around the level.
func addCrew(g *Game, u, n int) (*Vehicle, []*Cop) {
	def := g.lvl.Def
	v := &Vehicle{kind: 0, x: def.CarTo.X, y: def.CarFrom.Y, tx: def.CarTo.X, ty: def.CarFrom.Y, state: 1, spawned: true, unit: u}
	g.veh = append(g.veh, v)
	var out []*Cop
	r := rand.New(rand.NewSource(int64(5 + u)))
	for k := 0; k < n; k++ {
		c := &Cop{pos: g.lvl.RandomRoam(r), veh: v, kind: u, alt: k, state: cSweep, maxSearch: 12, maxChase: 5, onSceneT: 25}
		g.cops = append(g.cops, c)
		out = append(out, c)
	}
	g.respKind = u
	return v, out
}

func runCops(g *Game, secs float64, each func()) {
	dt := 1.0 / 60
	for f := 0; f < int(secs*60); f++ {
		g.time += dt
		for _, c := range append([]*Cop(nil), g.cops...) {
			g.updateCop(c, dt)
		}
		if each != nil {
			each()
		}
	}
}

// 1. How many come: police 1 or 2 (bodies found), FBI 2, army 3, SWAT 4, rangers 5; nobody carries a body.
func TestCopCrewSizesAndNobodyCarries(t *testing.T) {
	g := crewGame(t)
	def := g.lvl.Def
	r := rand.New(rand.NewSource(9))
	spawn := func(u int) int {
		g.cops = nil
		v := &Vehicle{kind: 0, x: def.CarTo.X, y: def.CarFrom.Y, tx: def.CarTo.X, ty: def.CarFrom.Y, state: 1, unit: u}
		g.veh = []*Vehicle{v}
		g.respKind = u
		g.updateVehicles(1.0 / 60)
		n := 0
		for _, c := range g.cops {
			if c.veh == v && !c.medic {
				n++
			}
		}
		return n
	}
	g.bodies = []*Body{{pos: g.lvl.RandomRoam(r), reported: true, cs: g.as.copCS}}
	if n := spawn(unitPolice); n != 1 {
		t.Errorf("police with 1 body found: %d officers, want 1", n)
	}
	g.bodies = append(g.bodies, &Body{pos: g.lvl.RandomRoam(r), reported: true, cs: g.as.copCS}, &Body{pos: g.lvl.RandomRoam(r), reported: true, cs: g.as.copCS})
	if n := spawn(unitPolice); n != 2 {
		t.Errorf("police with 3 bodies found: %d officers, want 2", n)
	}
	for u, want := range map[int]int{unitFBI: 2, unitMilitary: 3, unitSWAT: 4, unitRangers: 5} {
		if n := spawn(u); n != want {
			t.Errorf("unit %d: %d officers, want %d", u, n, want)
		}
	}
	// every type only covers the bodies and searches: nobody picks one up
	for _, u := range []int{unitPolice, unitFBI, unitMilitary, unitSWAT, unitRangers} {
		g.cops, g.veh = nil, nil
		for _, b := range g.bodies {
			b.bagged, b.carried, b.claimed = false, false, nil
		}
		_, crew := addCrew(g, u, groupStat[u].crew)
		for _, c := range crew {
			c.onSceneT = 0
			g.copNext(c)
		}
		runCops(g, 25, func() {
			for _, c := range g.cops {
				if c.carry != nil || c.state == cCollect || c.state == cBag || c.state == cReturn {
					t.Fatalf("unit %d: an officer handles a body (state %d)", u, c.state)
				}
			}
		})
		covered := 0
		for _, b := range g.bodies {
			if b.bagged && !b.carried {
				covered++
			}
		}
		if covered == 0 {
			t.Errorf("unit %d covered no body", u)
		}
	}
}

// 4a. A crew goes back complete: the one who is done waits for his mates, then they all leave together.
func TestCopCrewLeavesTogether(t *testing.T) {
	g := crewGame(t)
	_, crew := addCrew(g, unitFBI, 2)
	a, b := crew[0], crew[1]
	a.searchT, b.searchT = a.maxSearch, 0
	g.copNext(a)
	if a.state == cLeave || b.state == cLeave {
		t.Fatalf("an FBI agent left alone (states %d %d)", a.state, b.state)
	}
	b.searchT = b.maxSearch
	g.copNext(b)
	if a.state != cLeave || b.state != cLeave {
		t.Fatalf("the crew did not leave together (states %d %d)", a.state, b.state)
	}
	// with a dead mate (his body on the floor) the rest still go back together
	g.cops, g.veh = nil, nil
	_, crew = addCrew(g, unitMilitary, 3)
	g.removeCop(crew[2])
	crew[0].searchT = crew[0].maxSearch
	g.copNext(crew[0])
	if crew[0].state == cLeave {
		t.Fatal("a soldier left without his living mate")
	}
	crew[1].searchT = crew[1].maxSearch
	g.copNext(crew[1])
	if crew[0].state != cLeave || crew[1].state != cLeave {
		t.Fatal("the two living soldiers did not leave together")
	}
}

// 4b. Any type that sees nothing for copIdleLeave seconds goes back, all together; even the FBI, which otherwise
// never stops searching while there are 2 stars.
func TestCopCrewLeavesWhenIdle(t *testing.T) {
	for _, u := range []int{unitPolice, unitFBI, unitMilitary, unitSWAT, unitRangers} {
		g := crewGame(t)
		g.stars = 2
		_, crew := addCrew(g, u, groupStat[u].crew)
		for _, c := range crew {
			c.maxSearch = 1e9 // never done by searching: only the idle clock can send them back
		}
		runCops(g, copIdleLeave-2, nil)
		for _, c := range crew {
			if c.state == cLeave {
				t.Fatalf("unit %d: an officer left before %v s idle", u, copIdleLeave)
			}
		}
		runCops(g, 4, nil)
		for _, c := range crew {
			if c.state != cLeave {
				t.Errorf("unit %d: after %v s seeing nothing an officer stays (state %d)", u, copIdleLeave+2, c.state)
			}
		}
	}
}

// 5. Army and SWAT hold machine guns; at most one burst among all of them per call, then everybody goes back.
// The local police, the FBI and the rangers never shoot.
func TestCopMachineGunOncePerCall(t *testing.T) {
	g := crewGame(t)
	for _, u := range []int{unitMilitary, unitSWAT} {
		cs := copGunCS(g.as.militaryCS[0], u)
		if cs.Spec == nil || cs.Spec.Carry != HoldRifle {
			t.Errorf("unit %d holds no machine gun", u)
		}
	}
	if cs := copGunCS(g.as.copCS, unitPolice); cs != g.as.copCS {
		t.Error("the local police got a machine gun")
	}
	if c := (&Cop{kind: unitSWAT}); g.copCSFor(c).Spec.Carry != HoldRifle {
		t.Error("the SWAT officer is drawn without his machine gun")
	}
	near := func(crew []*Cop) {
		live := g.liveGuests()
		for i, c := range crew {
			gu := live[i%len(live)]
			c.pos = g.lvl.NearestFree(Vec{gu.pos.X + 20, gu.pos.Y})
			c.searchT, c.shootT = c.maxSearch*0.5, 0
		}
	}
	for _, u := range []int{unitPolice, unitFBI, unitRangers} {
		g.cops, g.veh, g.swatKills = nil, nil, 0
		_, crew := addCrew(g, u, groupStat[u].crew)
		near(crew)
		for f := 0; f < 60*60; f++ {
			for _, c := range crew {
				g.copBrutal(c, 1.0/60)
			}
		}
		if g.swatKills != 0 {
			t.Errorf("unit %d fired", u)
		}
	}
	g.cops, g.veh, g.swatKills = nil, nil, 0
	_, crew := addCrew(g, unitMilitary, 3)
	near(crew)
	shots := 0
	for f := 0; f < 60*240 && shots == 0; f++ {
		for _, c := range crew {
			g.copBrutal(c, 1.0/60)
		}
		shots = g.swatKills
	}
	if shots != 1 {
		t.Fatalf("the army never fired in 4 minutes (%d)", shots)
	}
	for _, c := range crew {
		if c.maxSearch > c.searchT+copGunReview+0.01 {
			t.Errorf("an officer keeps searching %.1f s after the burst", c.maxSearch-c.searchT)
		}
	}
	for f := 0; f < 60*120; f++ {
		for _, c := range crew {
			g.copBrutal(c, 1.0/60)
		}
	}
	if g.swatKills != 1 {
		t.Fatalf("%d bursts in one call, want 1", g.swatKills)
	}
	runCops(g, 30, nil)
	for _, c := range crew {
		if c.state != cLeave {
			t.Errorf("after the burst a soldier stays (state %d)", c.state)
		}
	}
	// a new call: they may fire again
	g.raisePolice(g.p.pos, false)
	if g.swatKills != 0 {
		t.Error("the burst of the last call still counts")
	}
}

// 2. The stars: as many as police types have come, the rest are not drawn at all.
func TestStarsCappedByPoliceTypes(t *testing.T) {
	g := crewGame(t)
	img := ebiten.NewImage(screenW, screenH)
	defer img.Deallocate()
	for _, n := range []int{1, 3, 5} {
		dbgPoliceGroups = n
		g.stars, g.starFloor = 0, 5
		g.updateStars(1.0 / 60)
		if g.stars != n || g.starFloor > n {
			t.Errorf("%d types: %d stars (floor %d), want %d", n, g.stars, g.starFloor, n)
		}
		starsDrawn = 0
		g.mode = ModePlay
		g.drawStars(img)
		if starsDrawn != n {
			t.Errorf("%d types: %d stars drawn", n, starsDrawn)
		}
	}
	dbgPoliceGroups = 3
	g.stars = 2
	starsDrawn = 0
	g.drawStars(img)
	if starsDrawn != 3 || g.starCap() != 3 {
		t.Errorf("2 of 3 stars: %d drawn", starsDrawn)
	}
}

// 3. POLICÍAS: always local -> FBI -> army -> SWAT -> rangers, whatever has been won.
func TestCopCardsFixedOrder(t *testing.T) {
	g := goalsGame(t)
	for i, tb := range goalTabs {
		if tb.kind == cardCop {
			g.trophyTab = i
		}
	}
	t.Cleanup(func() { g.trophyTab = 0 })
	g.progress.Album = map[string]int{copKey(unitPolice): albumKilled} // the local police won: it used to go last
	v := g.albumView()
	if len(v) != 5 {
		t.Fatalf("%d cards", len(v))
	}
	for i, c := range v {
		if c.key != copKey(copForces[i].unit) || copForces[i].unit != i {
			t.Errorf("card %d is %s", i, c.key)
		}
	}
}

// 7. Animals and traps feed the streak.
func TestAnimalAndTrapKillsCountForStreak(t *testing.T) {
	g, tr := mineGame(t, false)
	g.run = runStats{}
	a := &Animal{kind: 0, pos: Vec{300, 300}}
	g.animals = append(g.animals, a)
	g.killAnimal(a)
	if g.run.streakN[0] != 1 {
		t.Fatalf("an animal: streak %d, want 1", g.run.streakN[0])
	}
	g.armTrap(tr, g.p)
	tickProps(g, trapArmTime+0.2)
	gu := addGuestAt(g, Vec{400 + trapDetectR - 6, 304})
	tickProps(g, 0.1)
	if !gu.dead || g.run.streakN[0] != 2 {
		t.Fatalf("a trap kill: dead %v, streak %d, want 2", gu.dead, g.run.streakN[0])
	}
}

// 8. Trophies: the catch-up of the boot is quiet; a trophy is announced once per session, never again; nothing pops up
// in the menu or when a party starts with nothing done; a toast already seen does not come back after a pause; a
// medal waiting from another party is dropped; the medal has its own sound.
func TestTrophyToastsNoSpuriousOrRepeated(t *testing.T) {
	g := goalsGame(t)
	g.toasts, g.trophyToasted = nil, nil
	t.Cleanup(func() { g.toasts, g.trophyToasted, g.trophyQuiet = nil, nil, false })
	g.progress.Trophies = nil
	g.progress.Stats = map[string]int{"kills": 150}
	g.trophyQuiet, g.trophyDirty = true, true
	g.mode = ModeMenu
	g.checkTrophies()
	if !g.progress.Trophies["t_k100"] || len(g.toasts) != 0 {
		t.Fatalf("boot catch-up: granted %v, %d toasts (want granted, quiet)", g.progress.Trophies["t_k100"], len(g.toasts))
	}
	// the menu goes on: nothing else pops up
	for f := 0; f < 600; f++ {
		g.bankPoints()
		g.checkTrophies()
	}
	if len(g.toasts) != 0 {
		t.Fatalf("%d toasts in the menu", len(g.toasts))
	}
	// a stale copy of the save comes back: granted again, not announced again
	g.progress.Trophies = map[string]bool{}
	g.trophyDirty = true
	g.checkTrophies()
	if len(g.toasts) != 0 {
		t.Fatalf("a trophy announced twice: %+v", g.toasts)
	}
	// a new one in play: one toast, once
	g.stat("traps", 5)
	g.checkTrophies()
	g.trophyDirty = true
	g.checkTrophies()
	g.progress.Trophies["t_tr5"] = false
	g.trophyDirty = true
	g.checkTrophies()
	n := 0
	for _, tt := range g.toasts {
		if tt.name == "Artificiero" {
			n++
		}
	}
	if n != 1 || len(g.toasts) != 1 {
		t.Fatalf("Artificiero toasted %d times (%d toasts)", n, len(g.toasts))
	}
	// already read and then paused (newspaper, hint): it does not come back
	g.toasts[0].t = toastSeenT + 0.5
	g.mode = ModeLevelDone
	g.updateToasts(1.0 / 60)
	if len(g.toasts) != 0 {
		t.Error("a toast already read came back after the pause")
	}
	g.toasts = []trophyToast{{name: "Nuevo", reward: "x"}}
	g.updateToasts(1.0 / 60)
	if len(g.toasts) != 1 {
		t.Error("a toast not shown yet was lost in the pause")
	}
	// a medal of another party never shows
	g.mode = ModePlay
	g.toasts = []trophyToast{{medal: medalBronze, lvl: nil}}
	g.updateToasts(1.0 / 60)
	if len(g.toasts) != 0 && g.lvl != nil {
		t.Error("the medal of another party is still waiting")
	}
}

// 8b. A party that starts with nothing done shows no trophy, and the medal toast plays its own sound.
func TestNoTrophyAtPartyStartAndMedalSound(t *testing.T) {
	g := peLoad(t, 0, false)
	snapshotGlobals(t)
	progressFrozen = true
	prevP := cloneProgress(g.progress)
	t.Cleanup(func() { g.progress, g.toasts, g.trophyToasted = prevP, nil, nil })
	g.progress = Progress{Unlocked: 1}
	g.toasts, g.trophyToasted = nil, nil
	g.trophyDirty = true
	for f := 0; f < 60*8; f++ {
		g.updatePlay(1.0 / 60)
	}
	for _, tt := range g.toasts {
		if tt.medal == 0 {
			t.Errorf("a trophy at the start of a party with nothing done: %q", tt.name)
		}
	}
	prevSnd := g.snd
	t.Cleanup(func() { g.snd = prevSnd })
	g.snd = &Sounds{muted: true, sfx: map[string][]byte{"win": {1, 2, 3}}}
	g.run, g.toasts = runStats{}, nil
	if !g.partyMedalOn() {
		t.Fatal("level 0 gives no medal")
	}
	g.points = medalThresholds(g.partyMedalGoal())[0]
	g.medalLive()
	m := g.snd.sfx["medal"]
	if len(m) == 0 || string(m) == string(g.snd.sfx["win"]) {
		t.Error("the medal toast has no sound of its own")
	}
	if len(g.toasts) != 1 || g.toasts[0].lvl != g.lvl {
		t.Errorf("medal toast %+v", g.toasts)
	}
}

// 10. Picking up the gold suit while cuffed: new clothes and no more cuffs (like a disguise).
func TestGoldSuitWhileCuffedChangesClothesAndCuffs(t *testing.T) {
	g, pr := pickupSetup(t, false, "vipsuit")
	g.wanted = 70
	if !g.isSuspectP(g.p) {
		t.Fatal("setup: the player is not cuffed")
	}
	before := g.p.cs
	g.interact()
	if !pr.gone || g.p.suitT != suitDuration {
		t.Fatal("the suit was not picked up")
	}
	if g.p.cs == before || g.p.cs.Spec == nil || g.p.cs.Spec.Top != TopSuit || g.p.cs.Spec.Shirt != hx(0xd8b040) {
		t.Error("the clothes did not change to the gold suit")
	}
	if g.isSuspectP(g.p) || g.wanted != 0 {
		t.Errorf("still cuffed after the suit (wanted %.0f)", g.wanted)
	}
	// two players: only the one who wears it loses his cuffs
	g2, _ := pickupSetup(t, true, "vipsuit")
	g2.wanted = 70
	g2.markSuspect(g2.p)
	g2.markSuspect(g2.p2)
	g2.interact()
	if g2.isSuspectP(g2.p) || !g2.isSuspectP(g2.p2) {
		t.Errorf("2P: J1 cuffed %v, J2 cuffed %v (want false, true)", g2.isSuspectP(g2.p), g2.isSuspectP(g2.p2))
	}
}

// 11. A fast guest (a kid on wheels) who sets a trap off is always caught, even by the traps that act a moment later
// (gas, fireworks, the piano): before, he rolled through untouched.
func TestTrapCatchesFastGuestWhoSetsItOff(t *testing.T) {
	for _, kind := range []string{"explode", "gas", "fireworks", "piano"} {
		for _, slug := range []string{"sam_sam", "cesar", "diego"} {
			g, tr := mineGame(t, false)
			tr.K.Trap = kind
			g.p.pos = Vec{100, 100}
			g.armTrap(tr, g.p)
			tickProps(g, trapArmTime+0.2)
			sp := presetByName(slug)(rand.New(rand.NewSource(1)))
			c := trapCentre(tr)
			gu := &Guest{cs: newCharSet(&sp), pos: Vec{c.X + trapDetectR - 4, c.Y}}
			g.guests = append(g.guests, gu)
			dt := 1.0 / 60
			for f := 0; f < 60*4 && !gu.dead; f++ {
				g.time += dt
				gu.pos.X += guestSpeed * kidSpeedK * dt // rolling away on his wheels
				g.updateProps(dt)
				g.updateTraps2(dt)
			}
			if !tr.used {
				t.Fatalf("%s/%s: the trap never went off", kind, slug)
			}
			if !gu.dead {
				t.Errorf("%s: %s rolled through it untouched", kind, slug)
			}
		}
	}
}
