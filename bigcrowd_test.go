package main

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// Big-crowd parties (cap.go bigCrowd, user 2026-11-10): 2 or 3 campaign parties with a lot of free ground take
// 30 more people (kill target 95, live cap 90); every other party keeps the 65 of the kill target cap.

// freeRoamCells: navigation cells inside the Roam zones, free and reachable from the start (where the guests can
// really stand and walk).
func freeRoamCells(l *Level) int {
	reach := l.reachFrom(l.NearestFree(l.Def.Start))
	n := 0
	for c, ok := range reach {
		if !ok {
			continue
		}
		p := cellCenter(c%l.cw, c/l.cw)
		for _, r := range l.Def.Roam {
			if r.Has(p.X, p.Y) {
				n++
				break
			}
		}
	}
	return n
}

// bigPlay: like playGame, with one or two players.
func bigPlay(t *testing.T, level int, seed int64, two bool) *Game {
	t.Helper()
	g := layerGame(t, two)
	god, rng := g.god, g.rng
	t.Cleanup(func() {
		g.god, g.rng = god, rng
		g.cops, g.veh, g.bodies = nil, nil, nil
		g.policeCalled, g.pendingCar, g.pendingUnit, g.pendingAmb = false, -1, -1, -1
		g.ambCalled, g.callerGuest, g.callerGuard = false, nil, nil
		g.dc, g.dcArm, g.challenge, g.stealth = nil, nil, false, false
		g.mode = ModeMenu
	})
	g.rng = rand.New(rand.NewSource(seed))
	g.loadLevel(level)
	g.rng = rand.New(rand.NewSource(seed))
	g.mode = ModePlay
	g.titleT = 0
	g.god = true
	g.time = 30
	return g
}

// The choice is made with data: the big-crowd parties are 2 or 3, each among the 5 campaign parties with the
// most free walkable ground, and even full (95) they keep >= 45 free cells per person (the fullest normal parties
// sit at ~50-56 with 60). Every other party stays at 65.
func TestBigCrowdLevelsChosenByRoom(t *testing.T) {
	n := killTargetLevels(t)
	if len(bigCrowd) < 2 || len(bigCrowd) > 3 {
		t.Fatalf("%d big-crowd parties, want 2 or 3", len(bigCrowd))
	}
	type row struct {
		icon       string
		free, tot  int
		cap, bonus int
	}
	var rows []row
	found := map[string]bool{}
	for i := 0; i < n; i++ {
		g := playGame(t, i, int64(7+i))
		ic := g.lvl.Def.Icon
		b := bigCrowd[ic]
		found[ic] = found[ic] || b > 0
		if b < 0 || b > crowdMax {
			t.Errorf("%s: bonus %d outside 0..%d", ic, b, crowdMax)
		}
		if g.crowdExtra != b || g.goalMax() != goalCap+b || g.guestCap() != g.guestMax()+b {
			t.Errorf("%s: extra %d, target cap %d, live cap %d; want %d, %d, %d", ic, g.crowdExtra, g.goalMax(), g.guestCap(), b, goalCap+b, g.guestMax()+b)
		}
		rows = append(rows, row{ic, freeRoamCells(g.lvl), g.total, g.goalMax(), b})
	}
	for ic := range bigCrowd {
		if !found[ic] {
			t.Errorf("big-crowd icon %q is not a campaign party", ic)
		}
	}
	sort.Slice(rows, func(a, b int) bool { return rows[a].free > rows[b].free })
	for k, r := range rows {
		t.Logf("%2d %-9s free cells %5d  start %3d  cap %3d  cells/person at cap %3d", k+1, r.icon, r.free, r.tot, r.cap, r.free/r.cap)
		if r.bonus > 0 {
			if k >= 5 {
				t.Errorf("%s is big-crowd but only the %dth in free ground", r.icon, k+1)
			}
			if r.free/r.cap < 45 {
				t.Errorf("%s: only %d free cells per person at %d", r.icon, r.free/r.cap, r.cap)
			}
		}
	}
}

// The big-crowd parties: a real big start (more than any normal party can have, room left for the arrivals), every
// arrival route forced again and again never takes the target over 95 nor the live crowd over 90, and killing
// everybody of the target ends the party at N/N.
func TestBigCrowdReachesItsTarget(t *testing.T) {
	killTargetLevels(t)
	for _, two := range []bool{false, true} {
		for i := 0; i < campaignN && i < len(levelDefs); i++ {
			if bigCrowd[levelDefs[i].Icon] == 0 {
				continue
			}
			capBreaches = 0
			g := bigPlay(t, i, int64(31+i), two)
			name := fmt.Sprintf("L%02d %s two=%v", i+1, g.lvl.Def.Title, two)
			want := goalCap + crowdMax
			if two {
				want = minI(goalCap+g.twoExtra+crowdMax, crowdTwoTotal)
				if g.crowdExtra <= 0 || g.goalMax() != want || g.twoExtra < extraGuest {
					t.Fatalf("%s: extra %d+%d, cap %d, want %d", name, g.twoExtra, g.crowdExtra, g.goalMax(), want)
				}
			}
			if g.goalMax() != want {
				t.Fatalf("%s: cap %d, want %d", name, g.goalMax(), want)
			}
			if g.total <= goalCap-goalStartRoom || g.total > want-g.startRoom() {
				t.Errorf("%s: start %d, want %d..%d", name, g.total, goalCap-goalStartRoom+1, want-g.startRoom())
			}
			if g.guestLive() > g.guestCap() {
				t.Errorf("%s: %d guests at the start, live cap %d", name, g.guestLive(), g.guestCap())
			}
			mx := killTargetStress(t, g, 100)
			if mx > want || g.guestLive() > g.guestCap() || capBreaches != 0 {
				t.Errorf("%s: target %d (cap %d), live %d/%d, breaches %d", name, mx, want, g.guestLive(), g.guestCap(), capBreaches)
			}
			total := g.total
			killAll(g)
			if g.aliveCount() != 0 || g.total != total {
				t.Errorf("%s: %d alive, total %d -> %d", name, g.aliveCount(), total, g.total)
			}
			ks := fmt.Sprintf("%d/%d", g.total-g.aliveCount(), g.total)
			if ks != fmt.Sprintf("%d/%d", total, total) {
				t.Errorf("%s: HUD %s, want N/N", name, ks)
			}
			if g.mode == ModePlay && !g.slow.active {
				t.Errorf("%s: everybody is out but the party did not end", name)
			}
			t.Logf("%s: start %d, total %d (max %d), cap %d", name, total-0, g.total, mx, want)
		}
	}
}

// RETO DE HOY and the stealth mode play the big-crowd parties with the normal 65.
func TestBigCrowdNotInDailyNorStealth(t *testing.T) {
	killTargetLevels(t)
	lv := -1
	for i := 0; i < campaignN; i++ {
		if bigCrowd[levelDefs[i].Icon] > 0 {
			lv = i
			break
		}
	}
	if lv < 0 {
		t.Fatal("no big-crowd party")
	}
	g := bigPlay(t, 0, 1, false)
	g.dcArm = &dailyChallenge{date: "2026-11-10", level: lv, mods: []dailyMod{dmNoMap, dmSight}}
	g.loadLevel(lv)
	if g.dc == nil || g.crowdExtra != 0 || g.goalMax() != goalCap || g.total > goalCap-goalStartRoom {
		t.Errorf("daily: extra %d, cap %d, start %d", g.crowdExtra, g.goalMax(), g.total)
	}
	g.loadLevel(lv) // a normal load leaves the challenge: big again
	if g.dc != nil || g.crowdExtra != crowdMax {
		t.Errorf("after the daily: extra %d", g.crowdExtra)
	}
	g.stealthNext = true // the real entry (startStealth): stealthBegin turns it on at the start of loadLevel
	g.loadLevel(lv)
	if !g.stealth {
		t.Fatal("stealth did not start")
	}
	if g.crowdExtra != 0 || g.goalMax() != goalCap {
		t.Errorf("stealth: extra %d, cap %d", g.crowdExtra, g.goalMax())
	}
}

// The police stretches with T = 95 (and 110 with two players): every group gets its share, in order, none empty.
func TestBigCrowdPoliceStretches(t *testing.T) {
	for _, total := range []int{goalCap + crowdMax, crowdTwoTotal} {
		for n := 1; n <= len(groupStat); n++ {
			cnt := make([]int, n)
			prev := 0
			for k := 1; k <= total; k++ {
				i := policeGroupFor(k, total, n)
				if i < prev || i >= n {
					t.Fatalf("T=%d N=%d: kill %d -> group %d after %d", total, n, k, i, prev)
				}
				prev = i
				cnt[i]++
			}
			for i, c := range cnt {
				if lo := total/n - 1; c < lo || c > total/n+2 {
					t.Errorf("T=%d N=%d: group %d handles %d kills, want ~%d", total, n, i, c, total/n)
				}
			}
		}
	}
}

// The BAJAS row with three figures (110/110 with two players, 95/95 alone), Spanish and English: inside the panel
// with >= 6 px to its right edge, and the figures and the players' chips never touch.
func TestBigCrowdHUDThreeFigures(t *testing.T) {
	for _, two := range []bool{false, true} {
		for _, lang := range []string{"es", "en"} {
			g := hudPlay(t, two)
			gameLang, editorLang = lang, lang
			g.total = goalCap + crowdMax
			if two {
				g.total = crowdTwoTotal
				g.run.kills[0][kcKnife], g.run.kills[1][kcKnife] = 100, 108
			}
			for _, gu := range g.guests {
				gu.dead = true
			}
			rec := hudRecord(g)
			ks := fmt.Sprintf("%d/%d", g.total-g.aliveCount(), g.total)
			var row []textRec1
			var figY float64 = -1
			for _, tx := range rec.texts {
				if tx.s == ks {
					figY = tx.y
				}
			}
			if figY < 0 {
				t.Fatalf("two=%v %s: no %q in the HUD", two, lang, ks)
			}
			for _, tx := range rec.texts {
				if tx.y == figY || (tx.y > figY && tx.y < figY+10 && tx.scale == 1) {
					row = append(row, textRec1{tx.s, tx.x, tx.w})
				}
			}
			sort.Slice(row, func(a, b int) bool { return row[a].x < row[b].x })
			for k, r := range row {
				if r.x+r.w > hudX+hudW-6 {
					t.Errorf("two=%v %s: %q ends at %.0f, panel edge %.0f (want >= 6 px)", two, lang, r.s, r.x+r.w, hudX+hudW)
				}
				if k > 0 && r.x < row[k-1].x+row[k-1].w+4 {
					t.Errorf("two=%v %s: %q touches %q", two, lang, r.s, row[k-1].s)
				}
			}
			t.Logf("two=%v %s: %v", two, lang, row)
		}
	}
}

type textRec1 struct {
	s    string
	x, w float64
}

// Frame cost: the same big-crowd party played with 95 and with 65 (bigCrowd set to 0 for the comparison). Update
// (AI, paths, collisions, events) and Draw stay well inside the 16.7 ms of a frame and grow at most ~proportionally.
func TestBigCrowdFrameCost(t *testing.T) {
	if testing.Short() {
		t.Skip("timing")
	}
	killTargetLevels(t)
	lv := -1
	for i := 0; i < campaignN; i++ {
		if levelDefs[i].Icon == "festival" {
			lv = i
		}
	}
	if lv < 0 {
		t.Fatal("no festival")
	}
	var movedK [2]float64
	var walledN [2]int
	measure := func(bonus int) (live int, upd, drw float64) {
		old := bigCrowd["festival"]
		bigCrowd["festival"] = bonus
		defer func() { bigCrowd["festival"] = old }()
		g := bigPlay(t, lv, 5, false)
		g.lvl.graded = true // the colour grade of the background reads pixels: only inside the running game
		img := ebiten.NewImage(screenW, screenH)
		defer img.Deallocate()
		for i := 0; i < 120; i++ { // warm up (paths, caches)
			g.updatePlay(1.0 / 60)
		}
		live = g.guestLive()
		const frames = 900
		before := make(map[*Guest]Vec, len(g.guests))
		for _, gu := range g.guests {
			before[gu] = gu.pos
		}
		ran := 0
		t0 := time.Now()
		for ; ran < frames && g.mode == ModePlay; ran++ {
			g.updatePlay(1.0 / 60)
		}
		upd = float64(time.Since(t0).Microseconds()) / 1000 / float64(maxI(ran, 1))
		moved := 0
		for gu, p := range before {
			if !gu.out() && dist(gu.pos, p) > 8 {
				moved++
			}
		}
		if ran < frames || moved < live/3 {
			t.Fatalf("bonus %d: only %d frames played, %d of %d guests moved: not a real party", bonus, ran, moved, live)
		}
		walled := 0 // inside a wall or a prop (a few sit on props or use them in both runs)
		for _, gu := range g.guests {
			if !gu.out() && gu.perf == "" && (!g.lvl.Free(gu.pos) || g.lvl.SolidP(gu.pos.X, gu.pos.Y)) {
				walled++
			}
		}
		movedK[minI(bonus, 1)] = float64(moved) / float64(live)
		t.Logf("bonus %d: %d of %d guests walked > 8 px in 15 s, %d inside solids", bonus, moved, live, walled)
		walledN[minI(bonus, 1)] = walled
		const dframes = 60
		t0 = time.Now()
		for i := 0; i < dframes; i++ {
			g.Draw(img)
		}
		drw = float64(time.Since(t0).Microseconds()) / 1000 / dframes
		return
	}
	n65, u65, d65 := measure(0)
	n95, u95, d95 := measure(crowdMax)
	t.Logf("65: %d live, update %.2f ms, draw %.2f ms (CPU side)", n65, u65, d65)
	t.Logf("95: %d live, update %.2f ms, draw %.2f ms (CPU side)", n95, u95, d95)
	if walledN[1] > walledN[0]+2 {
		t.Errorf("more guests inside solids with the big crowd: %d vs %d", walledN[1], walledN[0])
	}
	if movedK[1] < movedK[0]*0.8 {
		t.Errorf("the big crowd moves less (jams?): %.2f vs %.2f of the guests walked", movedK[1], movedK[0])
	}
	if n95 <= n65+15 {
		t.Errorf("the big crowd has %d live vs %d", n95, n65)
	}
	if u95 > 6 {
		t.Errorf("update %.2f ms per frame with %d guests: over the 6 ms budget", u95, n95)
	}
	if u95 > u65*2.2+0.3 {
		t.Errorf("update grows too much: %.2f -> %.2f ms", u65, u95)
	}
	if d95 > 10 || d95 > d65*2.2+0.5 {
		t.Errorf("draw %.2f ms (was %.2f)", d95, d65)
	}
}
