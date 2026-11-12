package main

import (
	"image/color"
	"math"
	"testing"
)

// Tests of the street band (pe_murga.go) with the real Game after loadLevel.

// Only in the parties with a street, sometimes (a low chance), never in an empty level.
func TestMurgaRegisteredOnlyWithAStreet(t *testing.T) {
	g := partyGame(t)
	e := partyByID("murga")
	if e == nil {
		t.Fatal("murga is not registered")
	}
	if e.Prob < 0.10 || e.Prob > 0.13 || e.Reactive || e.MaxRuns > 1 || e.Group != "street" || !e.Big {
		t.Errorf("murga: chance %.2f reactive %v runs %d group %q big %v", e.Prob, e.Reactive, e.MaxRuns, e.Group, e.Big)
	}
	in := peEligible(g, "murga")
	if len(in) == 0 || len(in) == len(campaignLevels) {
		t.Errorf("murga fits %d campaign levels", len(in))
	}
	for _, i := range in {
		if !streetOK(levelDefs[i]) {
			t.Errorf("%s has no street", levelDefs[i].Title)
		}
	}
	if e.Eligible(g, partyTraits{}) {
		t.Error("murga fits an empty level")
	}
	t.Logf("murga fits %d of %d campaign levels", len(in), len(campaignLevels))
}

// The band gathers, marches by the street with its instruments, others follow for a while, and it breaks up: nobody is
// held afterwards, nobody is added (caps), with one and two players.
func TestMurgaMarchesAndBreaksUp(t *testing.T) {
	for _, two := range []bool{false, true} {
		peRetry(t, "murga", two, func(g *Game) string {
			a := g.partyRunning("murga")
			s := a.D.(*murgaState)
			nGuests, total := len(g.guests), g.total
			touched := map[*Guest]bool{}
			marched, maxBand, far := false, 0, 0.0
			street := streetOK(g.lvl.Def)
			for f := 0; f < 30*130 && g.partyRunning("murga") == a; f++ {
				peStep(g, 1.0/30, true)
				peTrack(g, touched)
				if s.march {
					marched = true
					maxBand = max(maxBand, len(s.band))
					if street && len(s.band) > 0 {
						far = math.Max(far, g.murgaStreetDist(s.band[0].gu.pos))
					}
					if f%20 == 0 {
						peDraw(g)
					}
				}
				if len(s.fol) > murgaFolMax {
					return "too many followers"
				}
			}
			if g.partyRunning("murga") == a {
				return "it never ended"
			}
			if gu := peStale(g, touched); gu != nil {
				return "a guest still held after the band"
			}
			if len(g.guests) != nGuests || g.total != total {
				return "the band changed the party"
			}
			if !marched {
				return "it never marched"
			}
			if maxBand < 3 || maxBand > 4 {
				t.Fatalf("a band of %d", maxBand)
			}
			if far > murgaStreetD+30 {
				t.Fatalf("the band went %.0f px away from the street", far)
			}
			t.Logf("%s two=%v: band %d, followers %d, %.0f s, max %.0f px from the street", g.lvl.Def.Title, two, maxBand, s.joined, g.time-a.t0, far)
			return ""
		})
	}
}

// Killing one of the band never jams it: it goes on with the others or ends, and lets everybody go.
func TestMurgaKillDoesNotJam(t *testing.T) {
	peRetry(t, "murga", false, func(g *Game) string {
		a := g.partyRunning("murga")
		s := a.D.(*murgaState)
		for f := 0; f < 30*30 && !s.march && g.partyRunning("murga") == a; f++ {
			peStep(g, 1.0/30, true)
		}
		if !s.march {
			return "it never marched"
		}
		touched := map[*Guest]bool{}
		peTrack(g, touched)
		g.killGuest(s.band[0].gu, true)
		for f := 0; f < 30*130 && g.partyRunning("murga") == a; f++ {
			peStep(g, 1.0/30, true)
			peTrack(g, touched)
		}
		if g.partyRunning("murga") == a {
			return "it never ended after the kill"
		}
		if gu := peStale(g, touched); gu != nil {
			return "a guest still held after the kill"
		}
		return ""
	})
}

// Every instrument draws inside a small box around the player (no stray pixels far away), facing both ways.
func TestMurgaInstrumentsDraw(t *testing.T) {
	k := func(v float64) int { return int(math.Round(v)) }
	for inst := 0; inst < instN; inst++ {
		for _, side := range []int{1, -1} {
			n := 0
			plot := func(x, y, w, h int, _ color.RGBA) {
				n++
				if x < 100-16 || x+w > 100+16 || y < 200-40 || y+h > 200 {
					t.Errorf("instrument %d side %d draws out of its box: %d,%d %dx%d", inst, side, x, y, w, h)
				}
			}
			drawInstrumentPix(plot, inst, 100, 200, 1, side, k, 0.3)
			if n == 0 {
				t.Errorf("instrument %d draws nothing", inst)
			}
		}
	}
}
