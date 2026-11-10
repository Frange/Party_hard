package main

import (
	"math/rand"
	"testing"
)

// The fist fight and the shooting between guests (pe_pelea.go, pe_tiroteo.go).

func TestPevFightAndShootingRegistered(t *testing.T) {
	g := partyGame(t)
	pe, sh := partyByID("pelea"), partyByID("tiroteo")
	if pe == nil || sh == nil {
		t.Fatal("pelea / tiroteo not registered")
	}
	if !pe.Minor || pe.Big || pe.DailyOK || pe.Cooldown <= 0 || pe.MaxRuns < 3 {
		t.Errorf("pelea: minor %v big %v daily %v cooldown %.0f runs %d", pe.Minor, pe.Big, pe.DailyOK, pe.Cooldown, pe.MaxRuns)
	}
	if sh.Minor || !sh.Big || sh.DailyOK || sh.MaxRuns != 1 {
		t.Errorf("tiroteo: minor %v big %v daily %v runs %d", sh.Minor, sh.Big, sh.DailyOK, sh.MaxRuns)
	}
	inP, inS := peEligible(g, "pelea"), peEligible(g, "tiroteo")
	t.Logf("pelea fits %d campaign levels, tiroteo %d: %v", len(inP), len(inS), inS)
	if len(inP) < len(campaignLevels)*2/3 {
		t.Errorf("the fight should fit most levels (%d)", len(inP))
	}
	if len(inS) == 0 || len(inS) > len(campaignLevels)/2 {
		t.Errorf("the shooting should fit only some levels (%d)", len(inS))
	}
	for _, i := range inS {
		if ic := levelDefs[i].Icon; ic == "cake" || ic == "station" || ic == "house" {
			t.Errorf("tiroteo in level %d (%s)", i, ic)
		}
	}
	for _, i := range inP {
		if levelDefs[i].Icon == "cake" {
			t.Error("no fights at the kids' birthday")
		}
	}
	// the fights are never kept out by the three planned events nor by the memory: a fitting party almost always has one
	idx := inS[0]
	tr := partyTraitsOf(levelDefs[idx], idx)
	r := rand.New(rand.NewSource(3))
	hist := [][]string{{"tiroteo", "pelea"}, {"pelea"}}
	withFight, withShoot := 0, 0
	for k := 0; k < 1000; k++ {
		for _, p := range partyPlan(r, g, tr, hist, false) {
			switch p.ev.ID {
			case "pelea":
				withFight++
			case "tiroteo":
				withShoot++
			}
		}
	}
	if withFight < 850 {
		t.Errorf("fight planned in %d of 1000 parties", withFight)
	}
	if withShoot != 0 {
		t.Error("the shooting is remembered: never right after a party that had it")
	}
	// the frame: never in STEALTH nor on the island, and not in the challenge
	g.loadLevel(idx)
	g.stealth = true
	g.partyReset()
	if len(g.pev.plan) != 0 {
		t.Error("events in STEALTH")
	}
	g.stealth = false
	g.dc = &dailyChallenge{}
	for k := 0; k < 30; k++ {
		partyHist = nil
		g.partyReset()
		for _, p := range g.pev.plan {
			if p.ev.ID == "pelea" || p.ev.ID == "tiroteo" {
				t.Fatal("fight or shooting in the RETO DE HOY")
			}
		}
	}
}

// peNoBlame: nothing of the event fell on the player.
func peNoBlame(t *testing.T, g *Game, kills int, what string) {
	t.Helper()
	if g.wanted > 0 || g.kills != kills || g.playerIsSuspect() {
		t.Errorf("%s: the player was blamed (wanted %.0f kills %d->%d suspect %v)", what, g.wanted, kills, g.kills, g.playerIsSuspect())
	}
	for _, b := range g.bodies {
		if b.byPlayer {
			t.Errorf("%s: a body counts as the player's", what)
		}
	}
}

// Both run to their end and let every guest go; nothing falls on the player.
func TestPevFightAndShootingRunAndRelease(t *testing.T) {
	for _, id := range []string{"pelea", "tiroteo"} {
		id := id
		t.Run(id, func(t *testing.T) {
			peRetry(t, id, false, func(g *Game) string {
				kills := g.kills
				touched := map[*Guest]bool{}
				sawHeld, sawShot := false, false
				for s := 0; s < 30*120 && g.partyRunning(id) != nil; s++ {
					peStep(g, 1.0/30, true)
					if s%10 == 0 {
						peDraw(g)
					}
					peTrack(g, touched)
					for gu := range touched {
						if peHeld(gu) {
							sawHeld = true
						}
					}
					if a := g.partyRunning(id); a != nil {
						if st, ok := a.D.(*tiroState); ok && len(st.shots) > 0 {
							sawShot = true
						}
					}
				}
				if g.partyRunning(id) != nil {
					return "it did not finish in 120 s"
				}
				if !sawHeld {
					return "it never held a guest"
				}
				if id == "tiroteo" && !sawShot {
					return "nobody shot"
				}
				if gu := peStale(g, touched); gu != nil {
					return "a guest is still held at the end"
				}
				if len(touched) > 10 {
					t.Errorf("%d guests taken", len(touched))
				}
				peNoBlame(t, g, kills, id)
				return ""
			})
		})
	}
}

// peleaToBrawl: a fight that has reached the blows.
func peleaToBrawl(t *testing.T, g *Game) *peleaState {
	t.Helper()
	for s := 0; s < 30*30; s++ {
		a := g.partyRunning("pelea")
		if a == nil {
			return nil
		}
		if st := a.D.(*peleaState); st.phase == peleaBrawl {
			return st
		}
		peStep(g, 1.0/30, true)
	}
	return nil
}

// A death in the fight: a body that is not the player's, no stars, no suspicion, no points; the watchers run.
func TestPevFightDeathIsNotThePlayers(t *testing.T) {
	var seed int64
	for seed = 1; rand.New(rand.NewSource(seed)).Float64() >= peleaDeathP; seed++ {
	}
	peRetry(t, "pelea", false, func(g *Game) string {
		st := peleaToBrawl(t, g)
		if st == nil {
			return "never came to blows"
		}
		kills, pts, bodies := g.kills, g.points, len(g.bodies)
		loser := st.b.gu
		peleaOutcome(g, st, rand.New(rand.NewSource(seed)), st.b)
		if !loser.dead || len(g.bodies) != bodies+1 {
			return "the loser did not die"
		}
		if g.points != pts {
			t.Errorf("points %d -> %d for a death that is not his", pts, g.points)
		}
		for _, m := range st.crowd.mem {
			if peHeld(m.gu) {
				t.Error("a watcher is still held")
			}
		}
		for s := 0; s < 30*10 && g.partyRunning("pelea") != nil; s++ {
			peStep(g, 1.0/30, true)
		}
		if g.partyRunning("pelea") != nil {
			return "it did not end after the death"
		}
		if !g.pev.ran["pelea_muerto"] {
			t.Error("the death is not noted for the party")
		}
		peNoBlame(t, g, kills, "fight death")
		return ""
	})
}

// The player breaks the fight up with E: points once, both let go.
func TestPevFightBreakUpGivesPoints(t *testing.T) {
	peRetry(t, "pelea", false, func(g *Game) string {
		st := peleaToBrawl(t, g)
		if st == nil {
			return "never came to blows"
		}
		g.p.pos = Vec{st.center.X, st.center.Y + 12}
		pts := g.points
		if !g.pevInteract(g.p) {
			return "E did nothing"
		}
		if g.points != pts+peleaBonus {
			t.Errorf("points %d -> %d", pts, g.points)
		}
		if g.pevInteract(g.p) {
			t.Error("the bonus is given only once")
		}
		peStep(g, 1.0/30, true)
		if g.partyRunning("pelea") != nil {
			return "still running after breaking it up"
		}
		if peHeld(st.a.gu) || peHeld(st.b.gu) {
			t.Error("a fighter is still held")
		}
		return ""
	})
}
