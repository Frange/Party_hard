package main

import (
	"math/rand"
	"testing"
)

func capGame(def *LevelDef, seed int64) *Game {
	g := testGame()
	g.rng = rand.New(rand.NewSource(seed))
	g.snd = &Sounds{muted: true}
	g.lvl = buildLevel(def)
	g.p = &Player{pos: def.Start, cs: newCharSet(func() *CharSpec { s := playerSpec(); return &s }())}
	g.famousOwned, g.famousPick = map[string]*Guest{}, map[string]bool{}
	g.pwFamAt = -999
	g.mode = ModePlay
	return g
}

// the cap is strict: however many arrivals, events and powers are forced, never more than the
// maximum of guests are present at the same time
func TestGuestCapNeverExceeded(t *testing.T) { capStress(t, false) }

// same with two players: the party starts with base (45..60) + 20..30 extra, and the cap grows the same
func TestGuestCapNeverExceededTwoPlayers(t *testing.T) {
	old := twoPlayerMode
	twoPlayerMode = true
	defer func() { twoPlayerMode = old }()
	capStress(t, true)
}

func capStress(t *testing.T, two bool) {
	defs := testLevels(t)
	if len(defs) > 14 {
		defs = defs[:14]
	}
	capBreaches = 0
	for di, def := range defs {
		g := capGame(def, int64(di)+1)
		g.pickStartTarget()
		g.fillGuests(def.Start)
		g.total = len(g.guests)
		lo, hi := 45, 60
		if two {
			// exact count of the code: base 45..60 plus twoExtra 20..30 => 65..90
			lo, hi = 45+20, 60+30
			if g.twoExtra < 20 || g.twoExtra > 30 || g.guestCap() != g.guestMax()+g.twoExtra {
				t.Fatalf("%s: twoExtra %d cap %d", def.Title, g.twoExtra, g.guestCap())
			}
		} else if g.twoExtra != 0 || g.guestCap() != 60 {
			t.Fatalf("%s: one player must keep cap 60, got %d", def.Title, g.guestCap())
		}
		if n := g.guestLive(); n < lo || n > hi {
			t.Errorf("%s: starts with %d guests, want %d..%d", def.Title, n, lo, hi)
		}
		maxSeen := 0
		for it := 0; it < 3000; it++ {
			pos := g.lvl.RandomRoam(g.rng)
			switch it % 9 {
			case 0:
				g.addGuestAt(pos)
			case 1:
				g.powerFamous()
				g.pwFamAt = -999
			case 2:
				g.startTheme(&Event{})
			case 3:
				g.startGhost(&Event{})
			case 4:
				g.addKidAt("sam_sam", pos)
			case 5:
				g.limoArrives(&Vehicle{alt: 1 + g.rng.Intn(6)})
			case 6:
				g.exhibited = false
				g.exhibitT = 0
				g.exhibitPlan = true
				g.updateExhibitionists()
			case 7:
				g.streaked = false
				g.streakT = 0
				g.updateStreaker()
			case 8:
				g.cheerGroups, g.cheerT = 0, 0
				g.updateCheer()
			}
			if n := g.guestLive(); n > maxSeen {
				maxSeen = n
			}
			if g.guestLive() > g.guestCap() {
				t.Fatalf("%s: %d guests at once (cap %d) after step %d", def.Title, g.guestLive(), g.guestCap(), it%9)
			}
			// some of them die or leave so that new ones can come: many cycles
			if it%5 == 0 {
				for _, gu := range g.guests {
					if !gu.out() && gu.perf == "" && g.rng.Intn(6) == 0 {
						gu.dead = true
					}
				}
			}
		}
		if maxSeen > g.guestCap() {
			t.Errorf("%s: max simultaneous %d above the cap %d", def.Title, maxSeen, g.guestCap())
		}
		if maxSeen < g.guestCap()-10 {
			t.Logf("%s: the party only reached %d guests", def.Title, maxSeen)
		}
	}
	if capBreaches != 0 {
		t.Errorf("the safety net had to act %d times: some route skips the cap", capBreaches)
	}
}

func TestGuestCapSafetyNet(t *testing.T) {
	defs := testLevels(t)
	g := capGame(defs[0], 5)
	capBreaches = 0
	for i := 0; i < 70; i++ { // a route that forgets to ask
		g.guests = append(g.guests, &Guest{cs: g.plainGuestCS()})
	}
	g.enforceGuestCap()
	if g.guestLive() != g.guestCap() || capBreaches != 10 {
		t.Errorf("live %d, breaches %d", g.guestLive(), capBreaches)
	}
	capBreaches = 0
	// the dead, the gone, guards and performers do not count
	g.guests = nil
	for i := 0; i < 80; i++ {
		g.guests = append(g.guests, &Guest{cs: g.plainGuestCS(), dead: i%2 == 0})
	}
	if g.guestLive() != 40 || g.guestRoom() != 20 {
		t.Errorf("live %d room %d", g.guestLive(), g.guestRoom())
	}
}
