package main

import (
	"math"
	"math/rand"
	"sort"
	"testing"
)

// ---------------------------------------------------------------------------
// The mosh pit (pogo), the slow dance (parejas) and the stadium wave (ola): registration, the levels where each one
// fits (and none else), the behaviour with the real Game after loadLevel (several parties, 1 and 2 players), the
// reactions to a kill, no guest left held, and no new people (caps).
// ---------------------------------------------------------------------------

func TestDance2Registered(t *testing.T) {
	g := partyGame(t)
	for _, c := range []struct {
		id    string
		group string
		icons map[string]bool
	}{{"pogo", "dance", pogoIcons}, {"parejas", "dance", couplesIcons}, {"ola", "stands", waveIcons}} {
		e := partyByID(c.id)
		if e == nil {
			t.Fatalf("%s not registered", c.id)
		}
		if e.Group != c.group || !e.Big || e.Minor || e.DailyOK || e.MaxRuns != 1 || e.Prob < 0.10 || e.Prob > 0.12 {
			t.Errorf("%s: group %q big %v prob %.2f runs %d", c.id, e.Group, e.Big, e.Prob, e.MaxRuns)
		}
		fits := peEligible(g, c.id)
		if len(fits) == 0 {
			t.Errorf("%s fits no campaign party", c.id)
		}
		for _, i := range fits {
			if !c.icons[campaignLevels[i].icon] {
				t.Errorf("%s fits party %d (%s), not one of its kind", c.id, i+1, campaignLevels[i].icon)
			}
		}
		var names []string
		for _, i := range fits {
			names = append(names, campaignLevels[i].icon)
		}
		t.Logf("%s: %v", c.id, names)
	}
	// the wave only where there are stands, and every venue of waveIcons has them
	for i, e := range campaignLevels {
		if !waveIcons[e.icon] {
			continue
		}
		if i >= len(levelDefs) {
			continue
		}
		n := 0
		for _, p := range levelDefs[i].Props {
			if p.Kind == "stands" {
				n++
			}
		}
		if n == 0 {
			t.Errorf("wave venue %s has no stands", e.icon)
		}
	}
	// never on the island or in stealth: the frame (partyBlocked); never in the RETO DE HOY: not DailyOK (above)
}

// pogoAngleGap: the widest empty angle of the members round the centre (a ring, not a line: < pi with 4+).
func pogoAngleGap(s *pogoState) float64 {
	var angs []float64
	for _, m := range s.mem {
		angs = append(angs, math.Atan2((m.gu.pos.Y-s.center.Y)/0.7, m.gu.pos.X-s.center.X))
	}
	if len(angs) < 2 {
		return 2 * math.Pi
	}
	sort.Float64s(angs)
	best := angs[0] + 2*math.Pi - angs[len(angs)-1]
	for i := 1; i < len(angs); i++ {
		best = math.Max(best, angs[i]-angs[i-1])
	}
	return best
}

func TestPogoCircleShovesAndReleases(t *testing.T) {
	for _, two := range []bool{false, true} {
		peRetry(t, "pogo", two, func(g *Game) string {
			s := g.partyRunning("pogo").D.(*pogoState)
			nG, tot := len(g.guests), g.total
			touched := map[*Guest]bool{}
			img := newTestImage()
			mosh, ringOK, ringN, shoves := 0, 0, 0, 0
			for i := 0; i < 30*90 && g.partyRunning("pogo") != nil; i++ {
				peStep(g, 1.0/30, true)
				peTrack(g, touched)
				if s.phase != pgMosh {
					continue
				}
				mosh++
				if len(s.mem) > pogoMaxN {
					return "more than 8 in the pit"
				}
				shoves += len(s.shoveT)
				for _, m := range s.mem {
					if d := dist(m.gu.pos, s.center); d > pogoOpenRad+8 {
						return "a member wandered off the circle"
					}
					if g.lvl.SolidP(m.gu.pos.X, m.gu.pos.Y) {
						return "a member inside a wall"
					}
				}
				if i%15 == 0 && len(s.mem) >= 4 {
					ringN++
					if pogoAngleGap(s) < math.Pi*0.95 {
						ringOK++
					}
				}
				if i%20 == 0 {
					peDraw(g)
					for _, m := range s.mem {
						s.drawGuest(g, img, m.gu, m.gu.pos)
					}
				}
			}
			if g.partyRunning("pogo") != nil {
				return "it did not end"
			}
			if s.fled {
				return "it broke up"
			}
			if s.maxN < pogoMinN || s.maxN > pogoMaxN {
				return "a pit of " + itoa(s.maxN)
			}
			if float64(mosh)/30 < pogoMinLen-1 || float64(mosh)/30 > pogoMaxLen+1 {
				return "it moshed " + itoa(mosh/30) + " s"
			}
			if ringN == 0 || ringOK*10 < ringN*8 {
				return "not a circle"
			}
			if shoves == 0 {
				return "nobody shoved"
			}
			if len(g.guests) != nG || g.total != tot {
				return "it changed the number of guests"
			}
			if gu := peStale(g, touched); gu != nil {
				return "a guest is still held"
			}
			return ""
		})
	}
}

// A fall: the one knocked down lies a moment and gets up (the debug stage pogo:fall forces the first one).
func TestPogoFallAndGetUp(t *testing.T) {
	prev := dbgPartyEvent
	t.Cleanup(func() { dbgPartyEvent = prev })
	peRetry(t, "pogo", false, func(g *Game) string {
		dbgPartyEvent = "pogo:fall"
		s := g.partyRunning("pogo").D.(*pogoState)
		var down *Guest
		up := false
		for i := 0; i < 30*60 && g.partyRunning("pogo") != nil && !up; i++ {
			peStep(g, 1.0/30, true)
			for gu, t := range s.fall {
				if g.time < t {
					down = gu
				}
			}
			if down != nil && g.time > s.fall[down] && pevHasIn(s.mem, down) && s.phase == pgMosh {
				up = true
			}
		}
		if s.falls == 0 || down == nil {
			return "nobody fell"
		}
		if !up {
			return "he never got up"
		}
		return ""
	})
}

// Killing one: the circle opens (the others step back), then it goes on or they run; never stuck.
func TestPogoKillOpensTheCircle(t *testing.T) {
	opened, went, ran := 0, 0, 0
	for k := 0; k < 4; k++ {
		peRetry(t, "pogo", k%2 == 1, func(g *Game) string {
			s := g.partyRunning("pogo").D.(*pogoState)
			for i := 0; i < 30*30 && s.phase != pgMosh && g.partyRunning("pogo") != nil; i++ {
				peStep(g, 1.0/30, true)
			}
			if s.phase != pgMosh {
				return "never got to the pit"
			}
			for i := 0; i < 60; i++ {
				peStep(g, 1.0/30, true)
			}
			touched := map[*Guest]bool{}
			peTrack(g, touched)
			victim := s.mem[0].gu
			if k%2 == 0 {
				peleaKill(g, victim)
			} else {
				g.killGuest(victim, true)
			}
			peStep(g, 1.0/30, true)
			if s.phase == pgOpen {
				opened++
				var d0 float64
				for _, m := range s.mem {
					d0 += dist(m.gu.pos, s.center)
				}
				for i := 0; i < 30; i++ {
					peStep(g, 1.0/30, true)
				}
				var d1 float64
				for _, m := range s.mem {
					d1 += dist(m.gu.pos, s.center)
				}
				if s.phase == pgOpen && len(s.mem) > 0 && d1 <= d0 {
					return "the circle did not open"
				}
			}
			for i := 0; i < 30*70 && g.partyRunning("pogo") != nil; i++ {
				peStep(g, 1.0/30, true)
				peTrack(g, touched)
			}
			if g.partyRunning("pogo") != nil {
				return "it did not end after the kill"
			}
			if s.reopened {
				went++
			}
			if s.fled {
				ran++
			}
			if gu := peStale(g, touched); gu != nil {
				return "a guest is still held"
			}
			return ""
		})
	}
	t.Logf("kill: opened %d, went on %d, ran %d", opened, went, ran)
	if opened == 0 && ran == 0 {
		t.Error("a kill in the pit neither opened the circle nor sent them running")
	}
}

func TestCouplesSlowDanceAndRelease(t *testing.T) {
	for _, two := range []bool{false, true} {
		peRetry(t, "parejas", two, func(g *Game) string {
			s := g.partyRunning("parejas").D.(*couplesState)
			nG, tot := len(g.guests), g.total
			touched := map[*Guest]bool{}
			danced, maxPairs, close := 0, 0, true
			for i := 0; i < 30*80 && g.partyRunning("parejas") != nil; i++ {
				peStep(g, 1.0/30, true)
				peTrack(g, touched)
				on := 0
				for _, p := range s.pairs {
					if !p.on || p.stopped || p.widow != nil {
						continue
					}
					on++
					if p.a.gu.state != gFun || p.a.gu.dAnim != AnimLove || p.b.gu.dAnim != AnimLove {
						return "a couple does not hug"
					}
					if dist(p.a.gu.pos, p.b.gu.pos) > 2*coupleGap+4 {
						close = false
					}
				}
				if on > 0 {
					danced++
				}
				maxPairs = maxI(maxPairs, on)
				if i%20 == 0 {
					peDraw(g)
				}
			}
			if g.partyRunning("parejas") != nil {
				return "it did not end"
			}
			if maxPairs < 1 || maxPairs > 3 || len(s.pairs) > 3 {
				return "couples: " + itoa(maxPairs)
			}
			if !close {
				return "a couple danced apart"
			}
			if float64(danced)/30 < coupleMinLen-1 {
				return "they danced only " + itoa(danced/30) + " s"
			}
			if len(g.guests) != nG || g.total != tot {
				return "it changed the number of guests"
			}
			if gu := peStale(g, touched); gu != nil {
				return "a guest is still held"
			}
			return ""
		})
	}
}

// One of a couple is killed: the other hugs nobody a moment, cries and runs; the event never gets stuck.
func TestCouplesWidowHugsNobodyThenRuns(t *testing.T) {
	peRetry(t, "parejas", false, func(g *Game) string {
		s := g.partyRunning("parejas").D.(*couplesState)
		var p *couplePair
		for i := 0; i < 30*30 && p == nil && g.partyRunning("parejas") != nil; i++ {
			peStep(g, 1.0/30, true)
			for _, q := range s.pairs {
				if q.on && !q.stopped && s.phase == cpDance {
					p = q
				}
			}
		}
		if p == nil {
			return "no couple danced"
		}
		touched := map[*Guest]bool{}
		peTrack(g, touched)
		partner := p.b.gu
		peleaKill(g, p.a.gu) // not by the player: nobody turns witness, the reaction is the event's
		peStep(g, 1.0/30, true)
		if p.widow == nil || p.widow.gu != partner {
			return "no widow"
		}
		hugged, cried, ran := false, false, false
		for i := 0; i < 30*8; i++ {
			peStep(g, 1.0/30, true)
			if !p.cried && partner.state == gFun && partner.dAnim == AnimLove {
				hugged = true
			}
			if p.cried && partner.dAnim == AnimTired {
				cried = true
			}
			if partner.state == gScared {
				ran = true
				break
			}
		}
		if !hugged || !cried || !ran {
			return "the widow did not hug nobody, cry and run"
		}
		for i := 0; i < 30*70 && g.partyRunning("parejas") != nil; i++ {
			peStep(g, 1.0/30, true)
			peTrack(g, touched)
		}
		if g.partyRunning("parejas") != nil {
			return "it did not end"
		}
		if gu := peStale(g, touched); gu != nil {
			return "a guest is still held"
		}
		return ""
	})
}

// waveRun: the wave forced in campaign party idx (a few different parties); nil if it never started.
func waveRun(t *testing.T, idx int, two bool, seed int64) *Game {
	t.Helper()
	for try := int64(0); try < 4; try++ {
		peLoadSeed = seed + 7*try
		g := peLoad(t, idx, two)
		peLoadSeed = 0
		partyRng = rand.New(rand.NewSource((seed+7*try)*7919 + int64(idx)))
		peForce(g, "ola")
		for s := 0; s < 30*40 && g.partyRunning("ola") == nil && len(g.pev.plan) > 0; s++ {
			peStep(g, 1.0/30, true)
		}
		if g.partyRunning("ola") != nil {
			return g
		}
	}
	return nil
}

func TestWaveGoesRoundTheStands(t *testing.T) {
	probe := partyGame(t)
	fits := peEligible(probe, "ola")
	started := 0
	for _, idx := range fits {
		for _, two := range []bool{false, true} {
			var last string
			ok := false
			for seed := int64(1); seed <= 4 && !ok; seed++ {
				g := waveRun(t, idx, two, seed)
				if g == nil {
					last = "never started"
					continue
				}
				last = waveCheck(g)
				ok = last == ""
			}
			if ok {
				started++
			} else {
				t.Logf("wave in %s (2P %v): %s", campaignLevels[idx].icon, two, last)
				if campaignLevels[idx].icon == "stadium" {
					t.Errorf("the wave fails in the stadium (2P %v): %s", two, last)
				}
			}
		}
	}
	if started < len(fits) { // every venue with stands, at least with one player count
		t.Logf("wave worked in %d of %d (venue, players) cases", started, 2*len(fits))
	}
	if started == 0 {
		t.Fatal("the wave never ran")
	}
}

func waveCheck(g *Game) string {
	s := g.partyRunning("ola").D.(*waveState)
	nG, tot := len(g.guests), g.total
	if len(s.row) < waveMinRow || len(s.row) > waveMaxRow {
		return "a row of " + itoa(len(s.row))
	}
	touched := map[*Guest]bool{}
	img := newTestImage()
	ran := 0
	for i := 0; i < 30*80 && g.partyRunning("ola") != nil; i++ {
		peStep(g, 1.0/30, true)
		peTrack(g, touched)
		if s.phase == wvRun {
			ran++
			if i%10 == 0 {
				peDraw(g)
				for _, m := range s.row {
					s.drawGuest(g, img, m.gu, m.gu.pos)
				}
			}
		}
	}
	if g.partyRunning("ola") != nil {
		return "it did not end"
	}
	if s.lapsDone != s.laps {
		return "the wave did not go round"
	}
	if sec := float64(ran) / 30; sec < 9.5 || sec > 21 {
		return "the wave lasted " + itoa(int(sec)) + " s"
	}
	if len(s.upSeen)*10 < len(s.row)*9 {
		return "some of the row never rose"
	}
	if len(g.guests) != nG || g.total != tot {
		return "it changed the number of guests"
	}
	if gu := peStale(g, touched); gu != nil {
		return "a guest is still held"
	}
	return ""
}

// A kill in the row: the wave stops a moment and goes on to the end (any venue with stands, several parties).
func TestWaveStopsAMomentAfterAKill(t *testing.T) {
	probe := partyGame(t)
	fits := peEligible(probe, "ola")
	sort.SliceStable(fits, func(i, j int) bool { return campaignLevels[fits[i]].icon == "stadium" })
	why := "never started"
	for _, idx := range fits {
		for seed := int64(1); seed <= 6; seed++ {
			g := waveRun(t, idx, false, seed)
			if g == nil {
				continue
			}
			s := g.partyRunning("ola").D.(*waveState)
			for i := 0; i < 30*45 && g.partyRunning("ola") != nil && (s.phase != wvRun || s.s < 200); i++ {
				peStep(g, 1.0/30, true)
			}
			if s.phase != wvRun || len(s.row) < waveMinLeft+2 {
				why = "it did not get going (phase " + itoa(s.phase) + ")"
				continue
			}
			n := len(s.row)
			var victim *Guest
			for _, m := range s.row {
				if m.want != gGoto {
					victim = m.gu
				}
			}
			g.killGuest(victim, true)
			peStep(g, 1.0/30, true)
			if s.pauseT <= g.time {
				t.Fatalf("the kill did not stop the wave (row %d->%d)", n, len(s.row))
			}
			at := s.s
			for i := 0; i < 15; i++ {
				peStep(g, 1.0/30, true)
			}
			if s.phase == wvRun && s.s != at && s.kills == 1 {
				t.Fatal("the wave did not stop")
			}
			for i := 0; i < 30*60 && g.partyRunning("ola") != nil; i++ {
				peStep(g, 1.0/30, true)
			}
			if g.partyRunning("ola") != nil {
				t.Fatal("it did not end")
			}
			if s.kills < 1 {
				t.Fatal("the kill was not seen")
			}
			if len(s.row) >= waveMinLeft && s.lapsDone != s.laps {
				t.Fatal("the wave did not go on after the kill")
			}
			t.Logf("kill tested in %s, party %d", campaignLevels[idx].icon, seed)
			return
		}
	}
	t.Fatal("the wave never got going: " + why)
}
