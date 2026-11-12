package main

import (
	"math"
	"testing"
)

// The dance battle: two guests meet, a ring of 5-7 gathers (open at the back: never a closed circle), they take turns
// with two different dances, one wins, and everybody is let go at the end. 1 and 2 players.
func TestDanceBattleRunsTakesTurnsAndReleases(t *testing.T) {
	e := partyByID("batalla")
	if e == nil || e.Group != "dance" || !e.Big || e.Prob < 0.10 || e.Prob > 0.40 || e.MaxRuns > 1 {
		t.Fatal("batalla: bad registration")
	}
	for _, two := range []bool{false, true} {
		peRetry(t, "batalla", two, func(g *Game) string {
			a := g.partyRunning("batalla")
			s := a.D.(*battleState)
			if s.stepA == s.stepB {
				return "the two dance the same"
			}
			touched := map[*Guest]bool{}
			turns, ring, last := 0, 0, -1
			var maxGap float64
			for i := 0; i < 30*80 && g.partyRunning("batalla") != nil; i++ {
				peStep(g, 1.0/30, true)
				if i%10 == 0 {
					peDraw(g)
				}
				peTrack(g, touched)
				if s.phase == btDance {
					if s.turn != last {
						turns++
						last = s.turn
					}
					on, off := s.a.gu, s.b.gu
					if s.turn == 1 {
						on, off = s.b.gu, s.a.gu
					}
					if on.state != gFun || on.dAnim < AnimDance0 || on.dAnim > AnimDance11 || off.dAnim == on.dAnim {
						return "the one whose turn it is does not dance"
					}
					if n := len(s.crowd.mem); n > ring {
						ring = n
						maxGap = battleRingGap(s)
					}
				}
			}
			if g.partyRunning("batalla") != nil {
				return "it did not end"
			}
			if s.fled {
				return "it broke up"
			}
			if turns < 4 {
				return "too few turns"
			}
			if ring < 4 || ring > 7 {
				return "a ring of " + itoa(ring)
			}
			if maxGap < math.Pi/2 {
				return "the ring is closed (no way out)"
			}
			if gu := peStale(g, touched); gu != nil {
				return "a guest is still held"
			}
			return ""
		})
	}
}

// battleRingGap: the widest empty angle around the centre (radians).
func battleRingGap(s *battleState) float64 {
	var angs []float64
	for _, m := range s.crowd.mem {
		angs = append(angs, math.Atan2(m.pt.Y-s.center.Y, m.pt.X-s.center.X))
	}
	if len(angs) < 2 {
		return 2 * math.Pi
	}
	for i := 1; i < len(angs); i++ {
		for j := i; j > 0 && angs[j] < angs[j-1]; j-- {
			angs[j], angs[j-1] = angs[j-1], angs[j]
		}
	}
	best := angs[0] + 2*math.Pi - angs[len(angs)-1]
	for i := 1; i < len(angs); i++ {
		best = math.Max(best, angs[i]-angs[i-1])
	}
	return best
}

// Interrupted: when one of the dancers is killed (or the police are called), the ring and the dancers run off and the
// event ends at once.
func TestDanceBattleFleesWhenInterrupted(t *testing.T) {
	for _, how := range []string{"kill", "police"} {
		how := how
		peRetry(t, "batalla", false, func(g *Game) string {
			s := g.partyRunning("batalla").D.(*battleState)
			for i := 0; i < 30*40 && s.phase != btDance && g.partyRunning("batalla") != nil; i++ {
				peStep(g, 1.0/30, true)
			}
			if s.phase != btDance {
				return "it never got to the dance"
			}
			ring := append([]*pevMem(nil), s.crowd.mem...)
			if how == "kill" {
				g.killGuest(s.a.gu, true)
			} else {
				g.policeCalled = true
			}
			peStep(g, 1.0/30, false)
			if g.partyRunning("batalla") != nil || !s.fled {
				return "it did not break up"
			}
			for _, m := range ring {
				if !m.gu.out() && m.gu.state != gScared {
					return "a watcher did not run"
				}
			}
			return ""
		})
	}
}
