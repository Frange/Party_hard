package main

import (
	"math/rand"
	"sort"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// sackGame: campaign level idx loaded (1 or 2 players) with the sack race switched on for the test and its dice pinned.
func sackGame(t *testing.T, idx int, two bool, seed int64) *Game {
	t.Helper()
	prev := [...]any{sackInTests, sackRng, sackSeed}
	t.Cleanup(func() {
		sackInTests, sackRng, sackSeed = prev[0].(bool), prev[1].(*rand.Rand), prev[2].(int64)
		sackLv = sackLevel{}
	})
	g := peLoad(t, idx, two)
	sackInTests, sackRng, sackSeed = true, nil, seed*3301+int64(idx)
	sackLv = sackLevel{}
	sackDue(g, partyTraits{})
	sackLv.want, sackLv.next, sackLv.max = true, 0, 3
	return g
}

// sackExpected: the campaign parties that always have room for a race (an open venue and a clear course of sackCourseW x
// sackCourseH with no prop and no dance floor on it). Other open venues (sackIcons: the bullring, the beach, the piazza)
// may have one or not depending on the small things the level scatters at random; any other party never has one.
var sackExpected = []string{"eiffel", "funfair", "festival"}

func TestSackCoursesOnlyInSpaciousLevels(t *testing.T) {
	g := partyGame(t)
	got := map[string]bool{}
	for i := range campaignLevels {
		for try := 0; try < 4; try++ {
			g.loadLevel(i)
			cs := sackCourses(g.lvl)
			if len(cs) > 0 && !sackIcons[g.lvl.Def.Icon] {
				continue // spacious, but not a venue for it (cemetery, ski slope): sackReset never looks
			}
			for _, c := range cs {
				if c.W < sackCourseW || c.H < sackCourseH {
					t.Fatalf("%s: a course of %.0fx%.0f", g.lvl.Def.Title, c.W, c.H)
				}
				for _, pr := range g.lvl.Props {
					if b := sackPropBase(pr); b.W > 0 && b.Hits(c.Grow(-1)) {
						t.Fatalf("%s: the course %v has the prop %s on it", g.lvl.Def.Title, c, pr.Name)
					}
				}
				for _, f := range g.lvl.Def.Dance {
					if f.Hits(c) {
						t.Fatalf("%s: the course %v is on the dance floor", g.lvl.Def.Title, c)
					}
				}
				for y := c.Y + 4; y < c.Y+c.H; y += 8 {
					for x := c.X + 4; x < c.X+c.W; x += 8 {
						if !g.lvl.Free(Vec{x, y}) || !g.lvl.Outdoor(Vec{x, y}) || g.lvl.InPool(Vec{x, y}) {
							t.Fatalf("%s: the course %v is not clear at %.0f,%.0f", g.lvl.Def.Title, c, x, y)
						}
					}
				}
			}
			if len(cs) > 0 {
				got[g.lvl.Def.Icon] = true
				break
			}
			if !sackIcons[g.lvl.Def.Icon] {
				break
			}
		}
	}
	var list []string
	for ic := range got {
		list = append(list, ic)
		if !sackIcons[ic] {
			t.Errorf("%s: a sack race outside the open venues", ic)
		}
	}
	sort.Strings(list)
	t.Logf("levels with a course: %v", list)
	for _, ic := range sackExpected {
		if !got[ic] {
			t.Errorf("%s has no course for the sack race (levels with one: %v)", ic, list)
		}
	}
	// a cramped party never has one, even when it is wanted
	g2 := sackGame(t, campaignIdx("house"), false, 1)
	for k := 0; k < 30*5; k++ {
		peStep(g2, 1.0/30, true)
		if g2.partyRunning("sacos") != nil {
			t.Fatal("a sack race in the house party")
		}
	}
}

type sackSeen struct {
	runners0, fell, maxRunners int
	winner                     bool
	raced                      bool
	dur                        float64
	last                       *sackState
}

func sackRun(g *Game, max float64, each func(s *sackState)) sackSeen {
	var sn sackSeen
	t0 := -1.0
	for k := 0; k < int(max*30); k++ {
		peStep(g, 1.0/30, true)
		a := g.partyRunning("sacos")
		if a == nil {
			if sn.last != nil {
				break
			}
			continue
		}
		s := a.D.(*sackState)
		if t0 < 0 {
			t0 = g.time
			sn.runners0 = len(s.run)
		}
		sn.last = s
		if len(s.run) > sn.maxRunners {
			sn.maxRunners = len(s.run)
		}
		if s.phase == sakRace {
			sn.raced = true
		}
		sn.fell = s.fellN
		sn.winner = sn.winner || s.winner != nil
		sn.dur = g.time - t0
		if each != nil {
			each(s)
		}
	}
	return sn
}

// 3-5 runners line up, race in hops, somebody trips and gets up, there is a winner; it ends by itself, nobody is held
// afterwards and nobody is added.
func TestSackRaceRunsWithAWinner(t *testing.T) {
	for _, icon := range sackExpected {
		ok := ""
		for seed := int64(1); seed <= 5; seed++ {
			g := sackGame(t, campaignIdx(icon), false, seed)
			n0 := len(g.guests)
			touched := map[*Guest]bool{}
			lanes := true
			sn := sackRun(g, 70, func(s *sackState) {
				for _, ru := range s.run {
					touched[ru.m.gu] = true
					if s.phase == sakRace && !ru.done && (ru.m.gu.pos.Y != ru.y || !s.course.Grow(4).Has(ru.m.gu.pos.X, ru.m.gu.pos.Y)) {
						lanes = false
					}
				}
				for _, m := range s.fans {
					touched[m.gu] = true
				}
			})
			for k := 0; k < 30*3; k++ {
				peStep(g, 1.0/30, true)
			}
			switch {
			case sn.last == nil:
				ok = "it never started"
			case sn.runners0 < 3 || sn.maxRunners > 5:
				t.Fatalf("%s: %d runners", icon, sn.runners0)
			case g.partyRunning("sacos") != nil:
				ok = "it never ended"
			case !sn.raced:
				ok = "they never raced (gather/ready only)"
			case !sn.winner:
				ok = "no winner"
			case sn.fell < 1:
				ok = "nobody tripped"
			case sn.dur > 50:
				ok = "it lasted too long"
			case !lanes:
				t.Fatalf("%s: a runner left his lane or the course", icon)
			default:
				ok = ""
			}
			if peStale(g, touched) != nil {
				t.Errorf("%s: a runner or a fan is still held", icon)
			}
			if len(g.guests) != n0 {
				t.Errorf("%s: the race added guests", icon)
			}
			if ok == "" {
				t.Logf("%s, party %d: %d runners, %d trips, %.0f s", icon, seed, sn.runners0, sn.fell, sn.dur)
				break
			}
			t.Logf("%s, party %d: %s", icon, seed, ok)
		}
		if ok != "" {
			t.Errorf("%s: %s", icon, ok)
		}
	}
}

// A dead runner stops the race: the others freeze and look at the body, then they are let go.
func TestSackRaceStopsWhenARunnerDies(t *testing.T) {
	idx := campaignIdx("festival")
	for seed := int64(1); seed <= 5; seed++ {
		g := sackGame(t, idx, false, seed)
		var victim *Guest
		stopped := false
		looked := true
		sackRun(g, 70, func(s *sackState) {
			if victim == nil && s.phase == sakRace && s.t > 2 && len(s.run) >= 3 {
				victim = s.run[1].m.gu
				g.killGuest(victim, false)
			}
			if s.phase == sakStop {
				stopped = true
				for _, ru := range s.run {
					if ru.m.mine() && ru.m.gu != victim && ru.m.gu.dir != faceTo(ru.m.gu.pos, s.body) {
						looked = false
					}
				}
			}
		})
		if victim == nil {
			t.Logf("party %d: the race never got going", seed)
			continue
		}
		if !stopped || !looked {
			t.Fatalf("stopped %v, looked at the body %v", stopped, looked)
		}
		if g.partyRunning("sacos") != nil {
			t.Fatal("the race goes on")
		}
		return
	}
	t.Error("no race got going in five parties")
}

// An alarm near the course dissolves the race at once.
func TestSackRaceDissolvesOnAlarm(t *testing.T) {
	idx := campaignIdx("eiffel")
	for seed := int64(1); seed <= 5; seed++ {
		g := sackGame(t, idx, false, seed)
		alarmed := false
		var at float64
		sn := sackRun(g, 70, func(s *sackState) {
			if !alarmed && s.phase == sakRace {
				alarmed, at = true, g.time
				mid := Vec{s.course.X + s.course.W/2, s.course.Y + s.course.H/2}
				for _, gu := range g.guests {
					if !gu.out() && !s.pevHas(gu) {
						gu.pos = Vec{mid.X, s.course.Y + 4} // a frightened guest runs by the course
						gu.state, gu.timer, gu.path = gAlarm, 5, nil
						break
					}
				}
			}
		})
		if !alarmed {
			continue
		}
		if g.partyRunning("sacos") != nil || sn.last == nil || g.time-at > 3 {
			t.Fatalf("the race went on %.1f s after the alarm", g.time-at)
		}
		return
	}
	t.Error("no race got going in five parties")
}

func TestSackRaceTwoPlayers(t *testing.T) {
	idx := campaignIdx("funfair")
	for seed := int64(1); seed <= 5; seed++ {
		g := sackGame(t, idx, true, seed)
		if g.p2 == nil {
			t.Fatal("no second player")
		}
		n0 := len(g.guests)
		sn := sackRun(g, 70, nil)
		if len(g.guests) != n0 {
			t.Fatalf("guests %d -> %d", n0, len(g.guests))
		}
		if sn.winner {
			return
		}
	}
	t.Error("with two players no race finished in five parties")
}

// The drawing runs in every phase (lines, poles, tape, sacks, hops, trips, claps).
func TestSackRaceDraws(t *testing.T) {
	g := sackGame(t, campaignIdx("festival"), false, 2)
	phases := map[int]bool{}
	scr := ebiten.NewImage(screenW, screenH)
	sackRun(g, 70, func(s *sackState) {
		if !phases[s.phase] {
			phases[s.phase] = true
			peDraw(g)
			for _, ru := range s.run {
				s.drawGuest(g, scr, ru.m.gu, ru.m.gu.pos)
			}
		}
	})
	if !phases[sakRace] {
		t.Error("never drawn while racing")
	}
}
