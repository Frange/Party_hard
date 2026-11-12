package main

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// streakGame: a real party (level 1, after loadLevel) with no police after anybody and the clock far from any chain.
func streakGame(t *testing.T, two bool) *Game {
	t.Helper()
	g := hudPlay(t, two)
	snapshotGlobals(t)
	progressFrozen = true
	t.Cleanup(func() { g.run, g.wanted, g.challenge, g.stealth, g.toasts = runStats{}, 0, false, false, nil })
	g.run, g.wanted, g.points = runStats{}, 0, 0
	g.time = 100
	return g
}

// streakKillNow: one knife kill of player p scored like killGuard does (countKill, then killVal), 10 s after the
// previous one so that the chain (4 s) never mixes in. Returns the points it gave.
func streakKillNow(g *Game, p *Player, n int) int {
	g.time += 10
	v := 0
	g.withPlayer(p, func() {
		g.countKill(kcKnife, true)
		v = g.killVal(n)
		g.points += v
	})
	return v
}

func TestStreakMultTiers(t *testing.T) {
	want := map[int]float64{0: 1, 1: 1, 2: 1, 3: 1.1, 4: 1.1, 5: 1.2, 7: 1.2, 8: 1.3, 11: 1.3, 12: 1.4, 16: 1.5, 40: 1.5, 999: 1.5}
	for n, m := range want {
		if got := streakMult(n); got != m {
			t.Errorf("streakMult(%d) = %v, want %v", n, got, m)
		}
	}
	prev := 1.0
	for n := 0; n < 100; n++ {
		if m := streakMult(n); m < prev || m > 1.5 {
			t.Fatalf("streakMult(%d) = %v: not monotonic or over the x1.5 top", n, m)
		} else {
			prev = m
		}
	}
}

// Clean kills raise the streak; the multiplier goes on the kill value; the top is x1.5; the extra is kept apart.
func TestStreakGrowsAndMultiplies(t *testing.T) {
	g := streakGame(t, false)
	want := []int{30, 30, 33, 33, 36, 36, 36, 39, 39, 39, 39, 42, 42, 42, 42, 45, 45, 45, 45, 45}
	extra := 0
	for i, w := range want {
		if v := streakKillNow(g, g.p, 30); v != w {
			t.Fatalf("kill %d: %d points, want %d (streak %d)", i+1, v, w, g.run.streakN[0])
		}
		extra += w - 30
	}
	if g.run.streakN[0] != len(want) || g.run.streakBest[0] != len(want) {
		t.Fatalf("streak %d best %d, want %d", g.run.streakN[0], g.run.streakBest[0], len(want))
	}
	if g.run.streakExtra != extra || g.run.streakPts[0] != extra || g.run.chainExtra != 0 {
		t.Fatalf("extra %d (J1 %d, chain %d), want %d", g.run.streakExtra, g.run.streakPts[0], g.run.chainExtra, extra)
	}
	f := g.previewFinish()
	if f.streak != extra || f.streakBest != len(want) || f.base+f.streak+f.chain+f.clean != g.points {
		t.Fatalf("breakdown %+v with %d points", f, g.points)
	}
	// the newspaper shows the best streak and its points
	d := g.paperDataNow()
	found := false
	for _, row := range d.score {
		if strings.HasPrefix(row[0], "Racha") {
			found = row[0] == "Racha · mejor 20" && row[1] == "+"+itoa(extra)
		}
	}
	if !found {
		t.Fatalf("no streak row in the points table: %v", d.score)
	}
}

// The streak breaks when a kill is seen (and THAT kill loses its extra, x3 included with the RETO), when filmed, and
// while the police want you; a wanted player does not grow one.
func TestStreakBreaks(t *testing.T) {
	for _, reto := range []bool{false, true} {
		g := streakGame(t, false)
		g.challenge = reto
		for i := 0; i < 5; i++ {
			streakKillNow(g, g.p, 30)
		}
		pts, ex := g.points, g.run.streakExtra
		v := streakKillNow(g, g.p, 30) // the 6th: x1.2 ... but somebody saw it (witnessAct, same frame)
		g.streakSeen(0)
		base := 30
		if reto {
			base *= challengeKillMult
		}
		if g.points != pts+base || g.run.streakExtra != ex || g.run.streakN[0] != 0 {
			t.Fatalf("reto=%v: seen kill gave %d (now %d, extra %d, streak %d), want plain %d", reto, v, g.points-pts, g.run.streakExtra-ex, g.run.streakN[0], base)
		}
		if reto && g.run.killExtra != g.points/3*2 {
			t.Fatalf("RETO extra %d out of step with %d points", g.run.killExtra, g.points)
		}
		if g.run.streakBest[0] != 6 {
			t.Fatalf("best %d, want 6", g.run.streakBest[0])
		}
		// a kill seen LATER (another frame) does not take back the extra of an older kill
		for i := 0; i < 4; i++ {
			streakKillNow(g, g.p, 30)
		}
		pts = g.points
		g.time += 1
		g.streakSeen(0)
		if g.points != pts || g.run.streakN[0] != 0 {
			t.Fatalf("an old kill lost points or the streak survived")
		}
	}
	// the camera
	g := streakGame(t, false)
	for i := 0; i < 4; i++ {
		streakKillNow(g, g.p, 30)
	}
	g.streakBreak(g.playerIndex(g.p))
	if g.run.streakN[0] != 0 {
		t.Fatal("filmed: the streak survived")
	}
	// wanted: the streak breaks and does not grow while the police want him
	for i := 0; i < 4; i++ {
		streakKillNow(g, g.p, 30)
	}
	g.wanted = 70
	g.streakTick()
	if g.run.streakN[0] != 0 {
		t.Fatal("wanted: the streak survived")
	}
	for i := 0; i < 4; i++ {
		if v := streakKillNow(g, g.p, 30); v != 30 {
			t.Fatalf("wanted kill gave %d", v)
		}
	}
	if g.run.streakN[0] != 0 {
		t.Fatal("a wanted player grew a streak")
	}
	g.wanted = 0
	// stealth: nothing
	g.stealth = true
	for i := 0; i < 5; i++ {
		if v := streakKillNow(g, g.p, 30); v != 30 {
			t.Fatalf("stealth kill gave %d", v)
		}
	}
}

// A seen kill through the real path (witnessAct with a guest looking at the player) breaks the streak.
func TestStreakBrokenByRealWitness(t *testing.T) {
	g := streakGame(t, false)
	for i := 0; i < 4; i++ {
		streakKillNow(g, g.p, 30)
	}
	if g.run.streakN[0] != 4 {
		t.Fatalf("streak %d", g.run.streakN[0])
	}
	var w *Guest
	for _, gu := range g.guests {
		if g.canWatch(gu) {
			w = gu
			break
		}
	}
	if w == nil {
		t.Skip("no guest able to watch in this party")
	}
	w.pos = Vec{g.p.pos.X - 30, g.p.pos.Y}
	w.dir = DirRight
	g.lastKiller = lastKill{}
	g.witness()
	if g.run.seen[0] == 0 {
		t.Skip("the guest did not see the player (walls); covered by TestStreakBreaks")
	}
	if g.run.streakN[0] != 0 {
		t.Fatalf("seen through witnessAct: streak %d", g.run.streakN[0])
	}
}

// Two players: each one has his own streak, its own extra and its own break.
func TestStreakTwoPlayerIndependent(t *testing.T) {
	g := streakGame(t, true)
	if !g.is2P() {
		t.Fatal("no J2")
	}
	for i := 0; i < 5; i++ {
		streakKillNow(g, g.p, 30)
	}
	for i := 0; i < 3; i++ {
		streakKillNow(g, g.p2, 30)
	}
	if g.run.streakN != [2]int{5, 3} {
		t.Fatalf("streaks %v, want [5 3]", g.run.streakN)
	}
	if v := streakKillNow(g, g.p2, 30); v != 33 { // J2's 4th: x1.1, not J1's x1.2
		t.Fatalf("J2 kill %d, want 33", v)
	}
	g.streakSeen(1)
	if g.run.streakN != [2]int{5, 0} {
		t.Fatalf("J2 seen: %v, want [5 0]", g.run.streakN)
	}
	if v := streakKillNow(g, g.p, 30); v != 36 {
		t.Fatalf("J1 kept x1.2: %d", v)
	}
	if g.run.streakPts[0] != 3+3+6+6 || g.run.streakPts[1] != 3 {
		t.Fatalf("per player extra %v", g.run.streakPts)
	}
	if g.run.streakPts[0]+g.run.streakPts[1] != g.run.streakExtra {
		t.Fatalf("extra %d != %v", g.run.streakExtra, g.run.streakPts)
	}
}

// The counter: hidden below 2, "RACHA N" from 2 and "RACHA N x1,1" from 3 (STREAK N x1.1 in English), whole pixels,
// 16 px inside the screen, under the player's feet, never over the shown HUD panel, in 1 and 2 players.
func TestStreakLabelLayout(t *testing.T) {
	snapshotLang(t)
	for _, two := range []bool{false, true} {
		g := streakGame(t, two)
		for i := range g.players() {
			g.run.streakN[i] = 1
		}
		for i, p := range g.players() {
			if _, _, _, _, _, ok := g.streakLabelRect(p, i); ok {
				t.Fatal("a streak of 1 shows")
			}
		}
		for _, n := range []int{2, 3, 16} {
			for _, lang := range []string{"es", "en"} {
				uiLang = lang
				for i, p := range g.players() {
					g.run.streakN[i] = n
					s, x, y, w, h, ok := g.streakLabelRect(p, i)
					if !ok {
						t.Fatalf("streak %d hidden", n)
					}
					want := map[int]string{2: "RACHA 2", 3: "RACHA 3 x1,1", 16: "RACHA 16 x1,5"}[n]
					if s != want {
						t.Fatalf("label %q, want %q", s, want)
					}
					if lang == "en" && n == 3 && tr(s) != "STREAK 3 x1.1" {
						t.Fatalf("English %q", tr(s))
					}
					if x != float64(int(x)) || y != float64(int(y)) || x < 16 || y < 16 || x+w > screenW-16 || y+h > screenH-16 {
						t.Fatalf("label at %v,%v %vx%v", x, y, w, h)
					}
					if y < g.wy(p.pos.Y) {
						t.Fatalf("label above the feet")
					}
				}
			}
		}
		// the keys help of the start sits at the bottom: a player standing over it gets the label over his head
		p := g.players()[0]
		g.time = 2
		p.pos = Vec{(480 - g.camI.X) / wk, (500 - g.camI.Y) / wk}
		if _, _, y, _, h, ok := g.streakLabelRect(p, 0); !ok || y+h+12 > g.streakBottomFree() || g.streakBottomFree() >= screenH {
			t.Fatalf("two=%v: label at y %v..%v over the keys help (from %v)", two, y, y+h, g.streakBottomFree())
		}
		g.time = 100
		// a player standing under the panel: the label is not drawn over the shown panel
		g.hud.hide = 0
		textRec = &textRecorder{}
		p.pos = Vec{(60 - g.camI.X) / wk, (40 - g.camI.Y) / wk}
		img := ebiten.NewImage(screenW, screenH)
		g.drawStreaks(img)
		img.Deallocate()
		x0, y0, x1, y1 := g.hudHideRect()
		for _, tx := range textRec.texts {
			if tx.x < x1 && tx.x+tx.w > x0 && tx.y < y1 && tx.y+tx.h > y0 {
				t.Fatalf("streak label %q over the HUD panel", tx.s)
			}
		}
		textRec = nil
	}
}

// Two players standing together: their counters never touch (4 px), with and without the keys help at the bottom.
func TestStreakTwoPlayerLabelsApart(t *testing.T) {
	snapshotLang(t)
	g := streakGame(t, true)
	g.run.streakN = [2]int{3, 16}
	for _, tm := range []float64{2, 100} {
		g.time = tm
		g.p.pos = Vec{(480 - g.camI.X) / wk, (300 - g.camI.Y) / wk}
		g.p2.pos = Vec{g.p.pos.X + 6, g.p.pos.Y}
		for _, lang := range []string{"es", "en"} {
			uiLang = lang
			ls := g.streakLabels()
			if len(ls) != 2 {
				t.Fatalf("time %v: %d labels", tm, len(ls))
			}
			a, b := ls[0], ls[1]
			if a.x < b.x+b.w+4 && a.x+a.w+4 > b.x && a.y < b.y+b.h+4 && a.y+a.h+4 > b.y {
				t.Fatalf("time %v %s: labels touch %+v %+v", tm, lang, a, b)
			}
			for _, l := range ls {
				if l.y < 16 || l.y+l.h > screenH-16 || l.y+l.h+12 > g.streakBottomFree() {
					t.Fatalf("label %+v out of its room", l)
				}
			}
		}
	}
}
