package main

import (
	"image"
	"image/color"
	"math"
	"math/rand"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
)

// Tests of the events pe_tvcrew.go, pe_booth.go, pe_magic.go, pe_toiletq.go, pe_fireworks.go and of newspaper_events.go.
// They use the helpers of pe_test.go (peStart, peStep, peDraw...).

var peNewIDs = []string{"tvcrew", "booth", "magician", "toiletq"}

func TestPevNewRegisteredAndSometimes(t *testing.T) {
	g := partyGame(t)
	list := partyList()
	n := len(campaignLevels)
	for _, id := range append(append([]string{}, peNewIDs...), "fireworks") {
		e := partyByID(id)
		if e == nil || !strings.Contains(list, id) {
			t.Fatalf("%s is not registered or not listed", id)
		}
		if id == "fireworks" {
			if !e.Reactive || e.Prob != fwChance {
				t.Errorf("fireworks: reactive entry with %.2f", e.Prob)
			}
			continue
		}
		if e.Prob < 0.10 || e.Prob > 0.40 || e.AtMax <= e.AtMin || e.Start == nil || e.Update == nil || e.Items == nil {
			t.Errorf("%s: bad definition", id)
		}
		in := peEligible(g, id)
		t.Logf("%-9s fits %2d of %d campaign levels (%.0f %% each): %v", id, len(in), n, e.Prob*100, in)
		if len(in) == 0 || len(in) > n*2/3 {
			t.Errorf("%s should fit some levels, not all (%d of %d)", id, len(in), n)
		}
		if e.Eligible(g, partyTraits{}) {
			t.Errorf("%s fits an empty level", id)
		}
	}
	for _, id := range []string{"magician", "booth"} {
		if tr := partyTraitsOf(levelDefs[0], 0); tr.icon == "station" && partyByID(id).Eligible(g, tr) {
			t.Errorf("%s in a station", id)
		}
	}
	if m := partyByID("magician"); m.Group != "street" {
		t.Error("the magician shares the group of the street artists")
	}
}

// Every event starts where it fits, runs to its end and leaves nobody held; the frame ends it by itself.
func TestPevNewRunsAndReleasesTheGuests(t *testing.T) {
	for _, id := range peNewIDs {
		id := id
		t.Run(id, func(t *testing.T) {
			g := peStart(t, id, false)
			touched := map[*Guest]bool{}
			for s := 0; s < 30*200 && g.partyRunning(id) != nil; s++ {
				peStep(g, 1.0/30, true)
				if s%10 == 0 {
					peDraw(g)
				}
				peTrack(g, touched)
			}
			if g.partyRunning(id) != nil {
				t.Fatal("it did not finish in 200 s")
			}
			if gu := peStale(g, touched); gu != nil {
				t.Errorf("a guest is still held at the end (state %d timer %.0f dur %.0f fun %d)", gu.state, gu.timer, gu.dur, gu.fun)
			}
			if len(touched) > len(g.guests)/2 {
				t.Errorf("%d of %d guests were taken: too many", len(touched), len(g.guests))
			}
			if !g.pev.ran[id] {
				t.Errorf("%s did not leave its mark for the newspaper", id)
			}
			if g.progress.Stats["pev_"+id] < 1 {
				t.Errorf("no statistic pev_%s", id)
			}
			if len(g.partyItems(ebiten.NewImage(screenW, screenH), nil)) != 0 {
				t.Error("something is still drawn after the end")
			}
			if g.kills != 0 {
				t.Errorf("an ambience event must not count kills (%d)", g.kills)
			}
		})
	}
}

func TestPevNewDoesNotMoveGameRng(t *testing.T) {
	for _, id := range peNewIDs {
		g := peStart(t, id, false)
		g.rng = rand.New(rand.NewSource(99))
		for s := 0; s < 30*45; s++ {
			peStep(g, 1.0/30, false)
			if s%30 == 0 {
				peDraw(g)
			}
		}
		if got, want := g.rng.Int63(), rand.New(rand.NewSource(99)).Int63(); got != want {
			t.Errorf("%s moved g.rng", id)
		}
	}
}

func TestPevNewTwoPlayers(t *testing.T) {
	for _, id := range peNewIDs {
		g := peStart(t, id, true)
		if !g.is2P() || g.p2 == nil {
			t.Fatalf("%s: not a two player game", id)
		}
		g.p2.pos = Vec{g.p.pos.X + 20, g.p.pos.Y}
		touched := map[*Guest]bool{}
		for s := 0; s < 30*100; s++ {
			peStep(g, 1.0/30, true)
			if s%15 == 0 {
				peDraw(g)
			}
			peTrack(g, touched)
		}
		if gu := peStale(g, touched); gu != nil && g.partyRunning(id) == nil {
			t.Errorf("%s: a guest is held after the end", id)
		}
	}
}

func TestPevNewWithManyCorpses(t *testing.T) {
	for _, id := range peNewIDs {
		g := peStart(t, id, false)
		for i := 0; i < 24; i++ {
			g.killByPlayer = true
			g.spawnBody(Vec{g.p.pos.X + float64(i%6)*12 - 30, g.p.pos.Y + 20 + float64(i/6)*10}, g.guests[0].cs, DirDown)
			g.killByPlayer = false
		}
		mine := map[*Guest]bool{} // only the guests THIS event took (other events react to the corpses and keep their own)
		for s := 0; s < 30*120; s++ {
			peStep(g, 1.0/30, true)
			if s%20 == 0 {
				peDraw(g)
			}
			if a := g.partyRunning(id); a != nil {
				if h, ok := a.D.(interface{ pevHas(*Guest) bool }); ok {
					for _, gu := range g.guests {
						if h.pevHas(gu) {
							mine[gu] = true
						}
					}
				}
			}
		}
		if g.partyRunning(id) != nil {
			t.Errorf("%s: still running after 120 s", id)
		}
		for gu := range mine {
			if !gu.out() && peHeld(gu) && !g.pevTaken(gu) {
				t.Errorf("%s: a guest of the event is still held after 120 s with corpses (state %d timer %.0f fun %d)", id, gu.state, gu.timer, gu.fun)
			}
		}
	}
}

// ---- TV crew ---------------------------------------------------------------------------------------------------

func TestPevTVCrewWaitsForMessAndGuestsPose(t *testing.T) {
	// calm party before 150 s: it does not come (the frame retries and gives up); with a mess it comes
	g := peLoad(t, peEligible(partyGame(t), "tvcrew")[0], false)
	if tvMess(g) {
		t.Fatal("a fresh party is not a mess")
	}
	dbgPartyEvent = ""
	g.time = 20
	if tvStart(g, &partyAct{}) {
		t.Error("the crew came to a calm party at 20 s")
	}
	g.time = 200
	if !tvStart(g, &partyAct{}) {
		t.Error("the crew never comes by chance after 150 s")
	}
	peRetry(t, "tvcrew", false, func(g *Game) string {
		st := g.partyRunning("tvcrew").D.(*tvState)
		sawAct, posed := false, false
		for s := 0; s < 30*80 && g.partyRunning("tvcrew") != nil; s++ {
			peStep(g, 1.0/30, true)
			if st.phase == tvAct {
				sawAct = true
				for _, m := range st.crowd.mem {
					if m.gu.state == gFun && m.gu.fun == 18 {
						posed = true
					}
				}
				if st.say == "" && g.time-st.t0 > 5 {
					return "the reporter never spoke"
				}
			}
			peDraw(g)
		}
		if !sawAct || !posed {
			return "no act or nobody posed for the camera"
		}
		return ""
	})
}

func TestPevTVCrewCanBeStabbedAndLeaves(t *testing.T) {
	peRetry(t, "tvcrew", false, func(g *Game) string {
		st := g.partyRunning("tvcrew").D.(*tvState)
		for s := 0; s < 30*60 && st.phase != tvAct; s++ {
			peStep(g, 1.0/30, true)
		}
		if st.phase != tvAct {
			return "never reached the spot"
		}
		before, bodies := g.kills, len(g.bodies)
		g.p.pos = Vec{st.cam.pos.X + 6, st.cam.pos.Y}
		if !g.pevStab(g.p, 24) || !st.camDead {
			return "the stab did not reach the cameraman"
		}
		if g.kills != before+1 || len(g.bodies) != bodies+1 {
			return "no kill or no body"
		}
		peDraw(g)
		for s := 0; s < 30*60 && g.partyRunning("tvcrew") != nil; s++ {
			peStep(g, 1.0/30, true)
		}
		if g.partyRunning("tvcrew") != nil {
			return "the reporter never left"
		}
		return ""
	})
}

// ---- booth -----------------------------------------------------------------------------------------------------

func TestPevBoothFourFlashesAndAStrip(t *testing.T) {
	peRetry(t, "booth", false, func(g *Game) string {
		st := g.partyRunning("booth").D.(*boothState)
		maxFlash, sawStrip, sawHidden := 0, false, false
		for s := 0; s < 30*100 && g.partyRunning("booth") != nil; s++ {
			peStep(g, 1.0/30, true)
			if st.flashes > maxFlash {
				maxFlash = st.flashes
			}
			if st.phase == bhStrip && st.stripT > 0 {
				sawStrip = true
			}
			if st.phase == bhFlash && st.mem != nil && dist(st.mem.gu.pos, st.spot) < 1 {
				sawHidden = true
			}
			peDraw(g)
		}
		if maxFlash < boothFlashes {
			return "fewer than four flashes"
		}
		if !sawStrip || !sawHidden || st.stripCount < 1 {
			return "no strip or the guest never stood in the booth"
		}
		return ""
	})
}

// ---- magician --------------------------------------------------------------------------------------------------

func TestPevMagicianTipOncePerParty(t *testing.T) {
	prev := magicGiven
	t.Cleanup(func() { magicGiven = prev })
	peRetry(t, "magician", false, func(g *Game) string {
		magicGiven = nil
		st := g.partyRunning("magician").D.(*magicState)
		for s := 0; s < 30*40 && st.show.phase != showAct; s++ {
			peStep(g, 1.0/30, true)
		}
		if st.show.phase != showAct {
			return "never reached the act"
		}
		g.p.pos = Vec{st.show.pf.pos.X + 20, st.show.pf.pos.Y + 10}
		if g.pevInteract(g.p) {
			return "E answers before he picked anybody"
		}
		st.pickEnd, st.pickWho = g.time+5, g.p
		before := g.points
		if !g.pevInteract(g.p) || g.points != before+magicTip {
			return "following the trick gives no tip"
		}
		st.pickDone, st.pickEnd = false, g.time+5
		before = g.points
		if !g.pevInteract(g.p) || g.points != before {
			return "a second tip in the same party"
		}
		// not followed in time: "qué mal público" and no tip
		st2 := &magicState{}
		*st2 = *st
		st2.pickDone, st2.pickEnd = false, g.time-1
		if st2.pevUse(g, g.p) {
			return "E works after the window"
		}
		for s := 0; s < 30*60 && g.partyRunning("magician") != nil; s++ {
			peStep(g, 1.0/30, true)
			peDraw(g)
		}
		return ""
	})
}

func TestPevMagicianFailsSometimesAndBirthdayHasNoSecondOne(t *testing.T) {
	prevG := magicGiven
	t.Cleanup(func() { magicGiven = prevG })
	g := partyGame(t)
	fails, runs := 0, 0
	partySeed = 5
	for i := 0; i < 200; i++ {
		partyRng = nil
		partySeed = int64(i + 1)
		idxs := peEligible(g, "magician")
		g2 := peLoad(t, idxs[i%len(idxs)], false)
		peForce(g2, "magician")
		for s := 0; s < 30*30 && g2.partyRunning("magician") == nil && len(g2.pev.plan) > 0; s++ {
			peStep(g2, 1.0/30, false)
		}
		a := g2.partyRunning("magician")
		if a == nil {
			continue
		}
		runs++
		for _, f := range a.D.(*magicState).fail {
			if f {
				fails++
				break
			}
		}
		if i > 40 {
			break
		}
	}
	t.Logf("magician: %d of %d acts have a trick that fails", fails, runs)
	if runs < 8 || fails == 0 || fails == runs {
		t.Errorf("a trick should fail sometimes, not never or always (%d of %d)", fails, runs)
	}
	// the birthday party: the decorative magician does the show, nobody is added. Also with a VIP zone of every shape
	// (vipForce + several dice): the carpet once swept the decorative magician away and a second one came.
	prevForce, prevLS := vipForce, peLoadSeed
	t.Cleanup(func() { vipForce, peLoadSeed = prevForce, prevLS })
	for run := 0; run < 7; run++ {
		vipForce, peLoadSeed = run > 0, int64(run)
		for _, idx := range peEligible(g, "magician") {
			if levelDefs[idx].Icon != "cake" {
				continue
			}
			g3 := peLoad(t, idx, false)
			deco := false
			for _, p := range g3.lvl.Props {
				deco = deco || (p.Name == "entertainer" && p.Var == 1 && !p.gone)
			}
			if !deco {
				t.Errorf("run %d (vip zone %v): the birthday party lost its decorative magician", run, g3.lvl.vip != nil)
			}
			peBirthdayShow(t, g3)
		}
	}
}

// peBirthdayShow: the magician event of the birthday party uses the decorative magician and adds nobody.
func peBirthdayShow(t *testing.T, g3 *Game) {
	t.Helper()
	before := len(g3.lvl.Props)
	peForce(g3, "magician")
	for s := 0; s < 30*40 && g3.partyRunning("magician") == nil && len(g3.pev.plan) > 0; s++ {
		peStep(g3, 1.0/30, true)
	}
	if a := g3.partyRunning("magician"); a != nil {
		st := a.D.(*magicState)
		if st.prop == nil || st.prop.Name != "entertainer" || len(g3.lvl.Props) != before {
			t.Error("the birthday show should use the decorative magician")
		}
	}
}

// ---- toilet queue ----------------------------------------------------------------------------------------------

func TestPevToiletQueueGoesOneAtATimeAndSomeoneJumps(t *testing.T) {
	peRetry(t, "toiletq", false, func(g *Game) string {
		st := g.partyRunning("toiletq").D.(*tqState)
		n0 := len(st.line)
		if n0 < tqMin {
			return "the line has fewer than 3"
		}
		sawLine, maxIn, oye, pee := false, 0, false, false
		for s := 0; s < 30*120 && g.partyRunning("toiletq") != nil; s++ {
			peStep(g, 1.0/30, true)
			in := 0
			for _, gu := range g.guests {
				if gu.state == gToilet && gu.inside == st.prop && !gu.out() {
					in++
				}
			}
			if in > maxIn {
				maxIn = in
			}
			if len(st.line) >= 2 {
				a, b := st.line[0].gu, st.line[1].gu
				if dist(a.pos, st.door) < dist(b.pos, st.door) && dist(a.pos, b.pos) > 2 && dist(a.pos, st.door) < 70 {
					sawLine = true
				}
			}
			for _, m := range st.line {
				if m.gu.state == gFun && m.gu.fun == 18 && m.gu.dAnim == AnimPee {
					pee = true
				}
				for _, c := range tqCut {
					if m.gu.icon == c {
						oye = true
					}
				}
			}
			peDraw(g)
		}
		t.Logf("toiletq: line of %d, %d went in, max %d inside, jumped=%v", n0, st.served, maxIn, st.cutDone)
		switch {
		case maxIn > 1:
			return "two people inside the same toilet"
		case st.served < 2:
			return "fewer than two went in"
		case !sawLine || !pee:
			return "no line or nobody danced"
		case !st.cutDone || !oye:
			return "nobody jumped the queue or nobody complained"
		}
		for _, gu := range g.guests {
			if gu.state == gToilet && gu.inside == st.prop && gu.timer > 30 {
				return "somebody is stuck inside"
			}
		}
		return ""
	})
}

// ---- fireworks -------------------------------------------------------------------------------------------------

func TestFireworksOnlyOutdoorAboutHalfAndNoRng(t *testing.T) {
	g := partyGame(t)
	var out, in int
	for i := range campaignLevels {
		if i >= len(levelDefs) {
			break
		}
		tr := partyTraitsOf(levelDefs[i], i)
		if tr.outdoor {
			out++
		} else {
			in++
		}
	}
	t.Logf("fireworks: %d outdoor and %d indoor campaign levels", out, in)
	if out == 0 || in == 0 {
		t.Fatal("the campaign needs both kinds of party")
	}
	for _, idx := range []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12} {
		g.loadLevel(idx)
		g.mode = ModePlay
		tr := partyTraitsOf(g.lvl.Def, idx)
		if got := g.fireworksOK(); got != tr.outdoor {
			t.Errorf("level %d (%s): fireworksOK=%v, outdoor=%v", idx, tr.icon, got, tr.outdoor)
		}
	}
	// the dice: ~55 % of the outdoor parties, g.rng untouched
	idx := 0
	for i := range campaignLevels {
		if tr := partyTraitsOf(levelDefs[i], i); tr.outdoor {
			idx = i
			break
		}
	}
	g.loadLevel(idx)
	g.mode = ModePlay
	g.rng = rand.New(rand.NewSource(7))
	armed := 0
	for i := 0; i < 2000; i++ {
		g.endTime = float64(i)
		g.fireworksArm(false)
		if g.fwx.armed {
			armed++
		}
	}
	if armed < 1000 || armed > 1200 {
		t.Errorf("armed %d of 2000 (want about 55 %%)", armed)
	}
	if got, want := g.rng.Int63(), rand.New(rand.NewSource(7)).Int63(); got != want {
		t.Error("the fireworks moved g.rng")
	}
	// never when a player is dead, in stealth or on the island
	g.p.dead = true
	if g.fireworksOK() {
		t.Error("fireworks with a dead player")
	}
	g.p.dead = false
	g.stealth = true
	if g.fireworksOK() {
		t.Error("fireworks in stealth")
	}
	g.stealth = false
	g.levelIdx = specialJurassic
	if g.fireworksOK() {
		t.Error("fireworks on the island")
	}
	g.levelIdx = idx
	// an indoor party: never armed
	for i := range campaignLevels {
		if tr := partyTraitsOf(levelDefs[i], i); tr.indoor {
			g.loadLevel(i)
			g.mode = ModePlay
			for k := 0; k < 100; k++ {
				g.fireworksArm(false)
				if g.fwx.armed {
					t.Fatalf("armed in the indoor level %d", i)
				}
			}
			break
		}
	}
}

func TestFireworksShowEndsIsCheapAndInsideTheScreen(t *testing.T) {
	var f fwState
	f.buf = make([]byte, fwW*fwH*4)
	f.rng = rand.New(rand.NewSource(3))
	steps, peak := 0, 0
	for ; steps < 60*20; steps++ {
		done := fwStep(&f, 1.0/60, nil)
		if steps%6 == 0 {
			fwRaster(&f, f.buf)
			for i := 0; i < len(f.buf); i += 4 {
				r, g, b, a := f.buf[i], f.buf[i+1], f.buf[i+2], f.buf[i+3]
				if r > a || g > a || b > a {
					t.Fatalf("pixel %d is not premultiplied: %d %d %d %d", i/4, r, g, b, a)
				}
			}
		}
		n := 0
		for i := range f.ps {
			if f.ps[i].on {
				n++
				p := f.ps[i]
				if p.x < -40 || p.x > fwW+40 || p.y < -80 || p.y > fwH+120 {
					t.Fatalf("a spark flew off to %.0f,%.0f", p.x, p.y)
				}
			}
		}
		if n > peak {
			peak = n
		}
		if done {
			break
		}
	}
	t.Logf("fireworks: %.1f s, %d launched, up to %d sparks", float64(steps)/60, f.launched, peak)
	if steps >= 60*20 || float64(steps)/60 > 9 {
		t.Fatalf("the show lasts too long (%d steps)", steps)
	}
	if f.launched < 5 || peak < 60 {
		t.Errorf("a poor show: %d rockets, %d sparks at most", f.launched, peak)
	}
	// no allocations while it plays
	f2 := fwState{rng: rand.New(rand.NewSource(4)), buf: make([]byte, fwW*fwH*4)}
	for i := 0; i < 90; i++ {
		fwStep(&f2, 1.0/60, nil)
	}
	if n := testing.AllocsPerRun(50, func() { fwStep(&f2, 1.0/60, nil); fwRaster(&f2, f2.buf) }); n != 0 {
		t.Errorf("%v allocations per frame", n)
	}
}

// The flow: armed at the end of an outdoor party, the show plays in ModeLevelDone before the paper and a press skips it.
func TestFireworksBeforeThePaperAndSkippable(t *testing.T) {
	g := partyGame(t)
	idx := 0
	for i := range campaignLevels {
		if partyTraitsOf(levelDefs[i], i).outdoor {
			idx = i
			break
		}
	}
	g.loadLevel(idx)
	g.mode = ModeLevelDone
	g.endTime = 123
	g.fwx = fwState{}
	g.fwx.armed, g.fwx.endT = true, 123
	if !g.fireworksUpdate(1.0/60) || !g.fwx.on || !g.fwx.shown {
		t.Fatal("the show did not start")
	}
	scr := ebiten.NewImage(screenW, screenH)
	if !g.fireworksDraw(scr) {
		t.Fatal("the show is on but it did not draw")
	}
	n := 0
	for g.fireworksUpdate(1.0/60) && n < 60*20 {
		n++
	}
	if g.fwx.on || n >= 60*20 {
		t.Fatalf("the show never ended (%d frames)", n)
	}
	if g.fireworksDraw(scr) || g.fireworksUpdate(1.0/60) {
		t.Error("after the show the paper must come back at once")
	}
	// another party's arm does not play here
	g.fwx = fwState{armed: true, endT: 5}
	if g.fireworksUpdate(1.0 / 60) {
		t.Error("a stale arm played in another party")
	}
}

// ---- pixels ----------------------------------------------------------------------------------------------------

func TestPevNewSpritesAreSharpAndInsideTheirBox(t *testing.T) {
	const W, H = 320, 240
	ax, ay := 160, 170
	for _, px := range []int{1, 2} {
		k := func(v float64) int { return int(math.Round(v * float64(px))) }
		draws := map[string]func(plot plotFn){
			"camera":   func(p plotFn) { drawTVCameraPix(p, ax, ay, px, k, 0.1) },
			"camera2":  func(p plotFn) { drawTVCameraPix(p, ax, ay, px, k, 0.9) },
			"mic":      func(p plotFn) { drawTVMicPix(p, ax, ay, px, k) },
			"rabbit":   func(p plotFn) { drawRabbitPix(p, ax, ay, px, 0.3) },
			"card":     func(p plotFn) { drawCardPix(p, ax, ay-k(20), px, 1) },
			"scarves":  func(p plotFn) { drawScarvesPix(p, ax, ay-k(26), px, 14, 0.7) },
			"tophat":   func(p plotFn) { drawTopHatPix(p, ax, ay, px) },
			"boothbk":  func(p plotFn) { drawBoothBackPix(p, ax, ay, px, k, 0.4, false) },
			"boothfl":  func(p plotFn) { drawBoothBackPix(p, ax, ay, px, k, 0.4, true) },
			"boothcl":  func(p plotFn) { drawBoothFrontPix(p, ax, ay, px, k, true) },
			"boothop":  func(p plotFn) { drawBoothFrontPix(p, ax, ay, px, k, false) },
			"spill":    func(p plotFn) { drawFlashSpillPix(p, ax, ay, px, k, 0.1) },
			"strip":    func(p plotFn) { drawPhotoStripPix(p, ax+k(15), ay-k(14), px, [4]int{0, 1, 2, 3}, 1) },
			"stripdrp": func(p plotFn) { drawPhotoStripPix(p, ax+k(15), ay-k(14), px, [4]int{0, 1, 2, 3}, 0.4) },
			"shake":    func(p plotFn) { drawShakePix(p, ax, ay, px, k, 0.3) },
		}
		for kind := 0; kind < mgTricks; kind++ {
			for _, fail := range []bool{false, true} {
				for _, u := range []float64{0.5, 1.5, 2.2, 3.1, 4.0, 5.2, 6.4} {
					kind, fail, u := kind, fail, u
					draws["trick"+string(rune('0'+kind))+map[bool]string{false: "o", true: "x"}[fail]+string(rune('a'+int(u*2)))] = func(p plotFn) {
						magicTrick(p, ax, ay, px, k, kind, fail, u, u)
					}
				}
			}
		}
		for name, fn := range draws {
			painted := 0
			fn(func(x, y, w, h int, c color.RGBA) {
				if c.R > c.A || c.G > c.A || c.B > c.A {
					t.Fatalf("%s: colour %v is not premultiplied", name, c)
				}
				for yy := y; yy < y+h; yy++ {
					for xx := x; xx < x+w; xx++ {
						if !image.Pt(xx, yy).In(image.Rect(0, 0, W, H)) || absI(xx-ax) > 70*px || yy-ay < -80*px || yy-ay > 24*px {
							t.Fatalf("%s (px %d): pixel %d,%d outside the box", name, px, xx, yy)
						}
						if c.A != 0 {
							painted++
						}
					}
				}
			})
			if painted == 0 && !strings.HasPrefix(name, "trick") && name != "spill" {
				t.Errorf("%s paints nothing", name)
			}
		}
	}
}

// ---- newspaper -------------------------------------------------------------------------------------------------

func TestPaperEventBriefsForEveryEventBothLanguages(t *testing.T) {
	ids := []string{"conga", "chairs", "drinks", "bbq", "bachelor", "mime", "juggler", "busker", "toast", "selfie", "contest", "pizza",
		"pizzadead", "abuela_dead", "tvcrew", "booth", "magician", "toiletq", "fireworks"}
	loadLanguages()
	for _, id := range ids {
		p, ok := paperEventPools[id]
		if !ok || len(p.pool) < 2 {
			t.Fatalf("%s: needs at least two variants for the newspaper", id)
		}
		if _, ok := paperBriefHeads[p.head]; !ok {
			t.Errorf("%s: head %q has no English", id, p.head)
		}
		seen := map[string]bool{}
		for _, v := range p.pool {
			if seen[v.es] || !paperEventTextOK(v.es) || utf8.RuneCountInString(v.en) > 100 || langDataEN[v.es] != v.en {
				t.Errorf("%s: bad variant %q", id, v.es)
			}
			seen[v.es] = true
			checkWords(t, v.es, pcLW-10)
			for _, bad := range []string{"Trump", "Messi", "Obama", "Putin", "Biden", "Shakira", "Madonna"} {
				if strings.Contains(v.es+v.en, bad) {
					t.Errorf("real name %q", bad)
				}
			}
		}
		// each event gives a brief, and different seeds give more than one variant
		variants := map[string]bool{}
		for seed := int64(0); seed < 40; seed++ {
			bs := paperEventBriefs(paperFacts{seed: seed * 7919, events: map[string]bool{id: true}})
			if len(bs) != 1 || bs[0].es == "" {
				t.Fatalf("%s: no brief", id)
			}
			variants[bs[0].es] = true
		}
		if len(variants) < 2 {
			t.Errorf("%s: always the same brief", id)
		}
	}
	// at most three, none twice, and they go first
	all := map[string]bool{}
	for _, id := range ids {
		all[id] = true
	}
	f := paperFacts{seed: 12345, events: all, kills: 4}
	bs := paperBriefsFor(f, "3:00")
	if len(paperEventBriefs(f)) != 3 {
		t.Errorf("%d event briefs, want 3", len(paperEventBriefs(f)))
	}
	seen := map[string]bool{}
	for i, b := range bs {
		if seen[b.es] {
			t.Errorf("brief %q twice", b.es)
		}
		seen[b.es] = true
		if i < 3 && !paperEventTextOK(b.es) {
			t.Errorf("event brief %d too long", i)
		}
	}
	if none := paperBriefsFor(paperFacts{seed: 12345, kills: 4}, "3:00"); len(none) != len(bs)-3 && len(none) < len(bs)-3 {
		t.Errorf("without events: %d briefs, with: %d", len(none), len(bs))
	}
}

// A page that tells about events is as sharp and whole as any other (the same check as TestPaperTextIsSharpAndWhole).
func TestPaperWithEventsIsSharpWholeAndShowsTheBooth(t *testing.T) {
	all := map[string]bool{}
	for id := range paperEventPools {
		all[id] = true
	}
	for _, lang := range []string{"es", "en"} {
		inLang(t, lang, func() {
			sawBooth, sawEvent := false, false
			for seed := int64(0); seed < 24; seed++ {
				f := paperFacts{seed: seed*977 + 3, kills: int(seed % 9), seen: int(seed % 3), events: all}
				d := paperSample(f, seed%2 == 0, false, 5)
				if d.photoVar == paperSceneBooth {
					sawBooth = true
				}
				im, lay := renderPaper(d, int(seed%2))
				for i := 3; i < len(im.Pix); i += 4 {
					if im.Pix[i] != 255 {
						t.Fatalf("translucent pixel in the page (seed %d)", seed)
					}
				}
				for _, b := range lay.blocks {
					if b.kind == "brief" && (b.x < 0 || b.x+b.w > screenW || b.y+b.h > pbY1) {
						t.Fatalf("%s: a brief leaves the body: %+v", lang, b)
					}
				}
				for _, tx := range lay.texts {
					if tx.x < 0 || tx.y < 0 || tx.x+tx.w > screenW-12 || tx.y+tx.h > screenH || tx.h < 7*tx.scale {
						t.Fatalf("%s: %q leaves the page or is too small", lang, tx.s)
					}
					if tx.title {
						continue
					}
					for y := tx.y; y < tx.y+tx.h; y++ {
						for x := tx.x; x < tx.x+tx.w; x++ {
							p := im.RGBAAt(x, y)
							if dd := pDist(p, tx.col); dd != 0 && dd <= 90 {
								t.Fatalf("%s: %q has a blended pixel %v at %d,%d", lang, tx.s, p, x, y)
							}
						}
					}
				}
				for _, b := range lay.briefs {
					for _, p := range paperEventPools {
						for _, v := range p.pool {
							if tr(v.es) == b {
								sawEvent = true
							}
						}
					}
				}
			}
			if !sawBooth || !sawEvent {
				t.Errorf("%s: booth photo seen=%v, event brief seen=%v", lang, sawBooth, sawEvent)
			}
		})
	}
	// no events: the photo is one of the three scenes
	if d := paperSample(paperFacts{seed: 5}, false, false, 3); d.photoVar >= paperPhotoScenes {
		t.Error("a booth photo without the booth")
	}
}

// The facts of a real party carry the events that took place (ran) and the ones of partylife.go.
func TestPaperFactsCarryTheEventsOfTheParty(t *testing.T) {
	g := partyGame(t)
	g.loadLevel(0)
	g.mode = ModePlay
	g.pev = pevState{ready: true}
	e := partyByID("conga")
	g.partyNoteRan(e)
	g.life.ready, g.life.pizzaRan, g.life.pizzaDead, g.time = true, true, true, 100
	f := g.paperFactsNow()
	if !f.events["conga"] || !f.events["pizzadead"] || f.events["pizza"] || f.events["booth"] {
		t.Errorf("events of the facts: %v", f.events)
	}
	g.life.pizzaDead = false
	if f := g.paperFactsNow(); !f.events["pizza"] {
		t.Error("the pizza boy came and the paper does not know")
	}
}
