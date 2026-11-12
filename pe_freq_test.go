package main

import (
	"math/rand"
	"testing"
)

// ---------------------------------------------------------------------------
// HOW OFTEN does something happen in a party? (feedback: "no pasa nada de nada y los eventos raros casi ni los veo").
// Two measures over the real registry and the real campaign levels, for parties of 5-8 minutes:
//   - the party events of partyevents.go (planned by partyPlan, with the memory of the last parties and the reruns of
//     the ones with Cooldown);
//   - the rare set pieces of events.go (UFO, Superman, Satan, God, zombies, meteor, ghost, T-Rex...): which kinds can
//     really start in each level, and how many "big" ones a party sees.
// ---------------------------------------------------------------------------

// peFreqSim: n simulated parties on random campaign levels; returns the mean number of party events seen per party,
// the fraction of parties with at least one and the mean number of fights (id "pelea") per party where it fits.
func peFreqSim(g *Game, r *rand.Rand, n int) (mean, atLeast1, fights float64) {
	var hist [][]string
	tot, any, nf, fitN := 0, 0, 0, 0
	for k := 0; k < n; k++ {
		idx := r.Intn(len(campaignLevels))
		tr := partyTraitsOf(levelDefs[idx], idx)
		L := 300 + r.Float64()*180 // a party of 5 to 8 minutes
		plan := partyPlan(r, g, tr, hist, false)
		var ids []string
		seen := 0
		if e := partyByID("pelea"); e != nil && e.Eligible(g, tr) {
			fitN++
		}
		for _, p := range plan {
			ids = append(ids, p.ev.ID)
			at, runs := p.at, 0
			mr := p.ev.MaxRuns
			if mr < 1 {
				mr = 1
			}
			for at < L && runs < mr {
				runs++
				seen++
				if p.ev.ID == "pelea" {
					nf++
				}
				if p.ev.Cooldown <= 0 {
					break
				}
				at += p.ev.Dur + p.ev.Cooldown*(0.8+0.4*r.Float64())
			}
		}
		hist = append(hist, ids)
		if len(hist) > partyMemory {
			hist = hist[1:]
		}
		tot += seen
		if seen > 0 {
			any++
		}
	}
	if fitN > 0 {
		fights = float64(nf) / float64(fitN)
	}
	return float64(tot) / float64(n), float64(any) / float64(n), fights
}

func TestPartyEventsFrequencyPerParty(t *testing.T) {
	g := partyGame(t)
	mean, any, fights := peFreqSim(g, rand.New(rand.NewSource(4242)), 4000)
	t.Logf("party events per 5-8 min party: mean %.2f, parties with >=1: %.0f %%, fights per party where they fit: %.2f", mean, any*100, fights)
	if mean < 2 || mean > 4.5 {
		t.Errorf("mean party events per party %.2f, want about 2-3", mean)
	}
	if any < 0.85 {
		t.Errorf("only %.0f %% of the parties see a party event", any*100)
	}
	if fights < 1.2 {
		t.Errorf("fights per party where they fit %.2f: they should come every 1.5-2.5 min", fights)
	}
	// the shooting: rare (about one party in three or four where it fits), never in two parties in a row
	sh := partyByID("tiroteo")
	if sh == nil {
		t.Fatal("tiroteo not registered")
	}
	if sh.Prob > 0.40 || sh.Prob < 0.20 || sh.DailyOK || !sh.Big {
		t.Errorf("tiroteo: chance %.2f daily %v big %v", sh.Prob, sh.DailyOK, sh.Big)
	}
}

// rareBig: the rare set pieces the players remember (not the things that only fly across the sky).
var rareBig = map[int]bool{evUFO: true, evSuper: true, evSatan: true, evGod: true, evZombie: true, evMeteor: true, evGhost: true, evRex: true}

// The rare set pieces: in a 5-8 minute party the big ones (UFO, Superman, zombies...) come in most parties (they already
// did before: 69-81 %), more often the longer the party lasts, never more than rareBigMax, never in the first minute.
func TestRareEventsFrequencyPerParty(t *testing.T) {
	r := rand.New(rand.NewSource(77))
	const n = 6000
	withBig, bigTot, tot := 0, 0, 0
	byLen := [3][2]int{} // parties of 5, 6.5 and 8 minutes: [with a big one, total]
	for k := 0; k < n; k++ {
		L := 300 + r.Float64()*180
		s := rareSchedule(r, L)
		tot += len(s)
		nb := 0
		for _, big := range s {
			if big {
				nb++
			}
		}
		if nb > rareBigMax {
			t.Fatalf("%d big rare events in one party", nb)
		}
		bigTot += nb
		b := int((L - 300) / 60.01)
		byLen[b][1]++
		if nb > 0 {
			withBig++
			byLen[b][0]++
		}
	}
	f := float64(withBig) / n
	t.Logf("rare events per party %.2f; parties with a big one %.0f %% (5-6 min %.0f %%, 6-7 %.0f %%, 7-8 %.0f %%), big per party %.2f",
		float64(tot)/n, f*100, 100*float64(byLen[0][0])/float64(byLen[0][1]), 100*float64(byLen[1][0])/float64(byLen[1][1]),
		100*float64(byLen[2][0])/float64(byLen[2][1]), float64(bigTot)/n)
	if f < 0.55 || f > 0.92 {
		t.Errorf("parties with a big rare event %.0f %%, want most of them but not all", f*100)
	}
	for k := 0; k < 3000; k++ {
		if rareBigKinds[rareKind(r, 30, 0, false, false, func(int) bool { return true })] {
			t.Fatal("a big rare event in the first minute")
		}
		if rareBigKinds[rareKind(r, 300, rareBigMax, false, false, func(int) bool { return true })] {
			t.Fatal("more big rare events than rareBigMax")
		}
	}
	if byLen[2][0]*byLen[0][1] <= byLen[0][0]*byLen[2][1] {
		t.Error("the big rare events should be likelier the longer the party lasts")
	}
}

// Which rare kinds can really start in the campaign levels (forced, like the debug flag -event N).
func TestRareEventsStartInTheCampaign(t *testing.T) {
	g := partyGame(t)
	ok, tried := make([]int, evKinds), 0
	for idx := 0; idx < len(campaignLevels); idx += 3 {
		peLoad(t, idx, false)
		tried++
		for k := 0; k < evKinds; k++ {
			before, zb := len(g.events), 0
			for _, gu := range g.guests {
				if gu.state == gZombie {
					zb++
				}
			}
			g.forceEv = k + 1
			g.startEvent()
			za := 0
			for _, gu := range g.guests {
				if gu.state == gZombie {
					za++
				}
			}
			if len(g.events) > before || za > zb {
				ok[k]++
			}
			g.events, g.handSat, g.handGod = nil, false, false
		}
	}
	t.Logf("rare kinds that start (of %d levels): %v", tried, ok)
	for k := range rareBig {
		if ok[k] == 0 {
			t.Errorf("big rare kind %d never starts in the campaign", k)
		}
	}
}

// v1.2.0 rebalance (the new events made a party see 4.8 events): the chances went down, but every event still comes in
// some party of the kind of level it fits ("que ocurran a veces"): over 4000 parties each planned event fits, it is
// planned at least a few times.
func TestPartyEventsEveryOneComesSometimes(t *testing.T) {
	g := partyGame(t)
	r := rand.New(rand.NewSource(4242))
	var hist [][]string
	fits, planned := map[string]int{}, map[string]int{}
	for k := 0; k < 4000; k++ {
		idx := r.Intn(len(campaignLevels))
		tr := partyTraitsOf(levelDefs[idx], idx)
		for _, e := range partyEvents() {
			if !e.Reactive && e.External == nil && e.Eligible != nil && e.Eligible(g, tr) {
				fits[e.ID]++
			}
		}
		var ids []string
		for _, p := range partyPlan(r, g, tr, hist, false) {
			planned[p.ev.ID]++
			ids = append(ids, p.ev.ID)
		}
		hist = append(hist, ids)
		if len(hist) > partyMemory {
			hist = hist[1:]
		}
	}
	for id, n := range fits {
		if planned[id] < 5 {
			t.Errorf("%s fits %d of 4000 parties but was planned only %d times", id, n, planned[id])
		}
	}
}
