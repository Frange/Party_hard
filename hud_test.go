package main

import (
	"flag"
	"fmt"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// hudPlay: the shared headless game in a party (1 or 2 players), with every global put back afterwards.
func hudPlay(t *testing.T, two bool) *Game {
	t.Helper()
	snapshotLang(t)
	g := auditGame(t)
	prevTwo, prevDbg, prevLang, prevEd := twoPlayerMode, dbgTwoPlayers, gameLang, editorLang
	t.Cleanup(func() {
		twoPlayerMode, dbgTwoPlayers, gameLang, editorLang = prevTwo, prevDbg, prevLang, prevEd
		textRec = nil
		g.challenge = false
		g.cops = nil
	})
	auditPlay(g, two)
	return g
}

// hudRecord draws only the HUD and returns what it wrote.
func hudRecord(g *Game) *textRecorder {
	textRec = &textRecorder{}
	defer func() { textRec = nil }()
	img := ebiten.NewImage(screenW, screenH)
	defer img.Deallocate()
	g.drawHUDPanel(img)
	return textRec
}

func rectsHit(ax, ay, aw, ah, bx, by, bw, bh float64) bool {
	return ax < bx+bw && bx < ax+aw && ay < by+bh && by < ay+ah
}

// The worst HUD there is (every force, RETO, a long chain, a huge score) in 1 and 2 players, Spanish and English:
// no text over another, every text inside the panel, and the panel clear of the "?" button, the trophy counter,
// the stealth panel, the police stars and the ringing phone.
func TestHUDLayoutNoOverlap(t *testing.T) {
	snapshotLang(t)
	for _, two := range []bool{false, true} {
		for _, lang := range []string{"es", "en"} {
			t.Run(fmt.Sprintf("players=%v/%s", two, lang), func(t *testing.T) {
				g := hudPlay(t, two)
				gameLang, editorLang = lang, lang
				g.challenge = true
				g.time = 6249 // the widest clock: 104:09
				g.points, g.hud.pts = 9999999, 9999999
				g.run.chainN[0], g.run.chainLast[0] = 12, g.time
				g.run.kills[0][kcKnife], g.run.kills[1][kcKnife] = 7, 5
				for _, k := range []int{unitPolice, unitFBI, unitMilitary, unitSWAT, unitRangers} {
					g.cops = append(g.cops, &Cop{kind: k, state: cChase})
				}
				rec := hudRecord(g)
				if len(rec.texts) < 6 {
					t.Fatalf("the HUD wrote only %d texts", len(rec.texts))
				}
				for _, f := range auditCheck("hud", rec, false, nil, nil) {
					t.Errorf("%s", f.msg)
				}
				ph := g.hud.bottom - 3 - hudY
				for _, tx := range rec.texts {
					if tx.x < hudX || tx.x+tx.w > hudX+hudW || tx.y+3*tx.scale < hudY || tx.y+tx.h-3*tx.scale > hudY+ph {
						t.Errorf("%q [%.0f,%.0f %.0fx%.0f] leaves the panel %.0f,%.0f %.0fx%.0f", tx.s, tx.x, tx.y, tx.w, tx.h, hudX, hudY, hudW, ph)
					}
				}
				others := map[string][4]float64{
					"help button":   {helpBtnX, helpBtnY, helpBtnW, helpBtnH},
					"trophy":        {trophyHudX, trophyHudY, trophyHudW, trophyHudH},
					"stealth panel": {650, 34, 290, 54},
					"stars":         {screenW/2 - 40, 16, 80, 10},
					"phone":         {screenW - 90, 40, 80, 80},
				}
				for name, r := range others {
					if rectsHit(hudX, hudY, hudW, ph, r[0], r[1], r[2], r[3]) {
						t.Errorf("the panel overlaps the %s", name)
					}
				}
			})
		}
	}
}

// Only the normal HUD: the level start must still look right with the plain numbers (no chips, no forces).
func TestHUDPlainHeight(t *testing.T) {
	g := hudPlay(t, false)
	hudRecord(g)
	if want := hudY + hudBaseH + 3; g.hud.bottom != want {
		t.Errorf("bottom %.0f, want %.0f", g.hud.bottom, want)
	}
	if g.hud.bottom > 88+12+12 { // 12 more: the panel stands 16 px from the edges
		t.Errorf("the panel reaches %.0f: it grew more than 12 px over the old 88", g.hud.bottom)
	}
}

func TestHUDPointsCountUpAndSnap(t *testing.T) {
	g := hudPlay(t, false)
	g.points, g.hud.pts = 500, 0
	prev := -1.0
	frames := 0
	for g.hud.pts < 500 && frames < 600 {
		g.updateHUD(1.0 / 60)
		if g.hud.pts < prev {
			t.Fatalf("the counter went back: %v -> %v", prev, g.hud.pts)
		}
		prev = g.hud.pts
		frames++
	}
	if g.hud.pts != 500 || frames < 5 || frames > 120 {
		t.Errorf("counted up to %v in %d frames (want 500 within 5..120)", g.hud.pts, frames)
	}
	g.points = 40 // lost points (a fine, a new party): no counting down, it snaps
	g.updateHUD(1.0 / 60)
	if g.hud.pts != 40 {
		t.Errorf("the counter shows %v after dropping to 40", g.hud.pts)
	}
}

func TestHUDProgressBarFollowsKills(t *testing.T) {
	g := hudPlay(t, false)
	if p := g.hudProgress(); p != 0 {
		t.Fatalf("progress %.2f at the start", p)
	}
	n := 0
	for _, gu := range g.guests {
		if !gu.out() && gu.py == nil && n < g.total/2 {
			gu.dead = true
			n++
		}
	}
	p := g.hudProgress()
	if p <= 0.3 || p >= 0.7 {
		t.Errorf("progress %.2f after killing about half (%d of %d)", p, n, g.total)
	}
	for i := 0; i < 200; i++ {
		g.updateHUD(1.0 / 60)
	}
	if d := g.hud.bar - p; d > 0.01 || d < -0.01 {
		t.Errorf("the bar shows %.2f, progress is %.2f", g.hud.bar, p)
	}
}

func TestHUDPopWhenPeopleDrop(t *testing.T) {
	g := hudPlay(t, false)
	for _, gu := range g.guests {
		if !gu.out() {
			gu.dead = true
			break
		}
	}
	g.updateHUD(1.0 / 60)
	if g.hud.pop <= 0 {
		t.Error("the number does not pop when somebody falls")
	}
}

// Drawing the HUD in every mode that shows it, in 1 and 2 players, must not panic.
func TestHUDDrawNoPanic(t *testing.T) {
	for _, two := range []bool{false, true} {
		g := hudPlay(t, two)
		for _, m := range []Mode{ModePlay, ModeLose, ModeLevelDone} {
			g.mode = m
			g.hud.people = -5
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("players=%v mode=%v: %v", two, m, r)
					}
				}()
				img := ebiten.NewImage(screenW, screenH)
				defer img.Deallocate()
				g.drawHUD(img)
			}()
		}
		g.mode = ModePlay
	}
}

// -lastguests freezes the progress like every other debug flag.
func TestLastGuestsFlagFreezesProgress(t *testing.T) {
	found := false
	for _, n := range mainFlagNames(t) {
		found = found || n == "lastguests"
	}
	if !found {
		t.Fatal("main.go has no -lastguests flag")
	}
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	fs.Int("lastguests", -1, "")
	if err := fs.Parse([]string{"-lastguests", "2"}); err != nil {
		t.Fatal(err)
	}
	if !debugFlagsActive(fs) {
		t.Error("-lastguests does not freeze the progress")
	}
}

// The BAJAS figures are bigger than before (scale 2, they were 1) and keep air from the bar above and the TIME row below.
func TestHUDBajasRowBiggerAndSpaced(t *testing.T) {
	for _, two := range []bool{false, true} {
		g := hudPlay(t, two)
		rec := hudRecord(g)
		var fig, clock *textRect
		for i := range rec.texts {
			tx := &rec.texts[i]
			if tx.s == fmt.Sprintf("%d/%d", g.total-g.aliveCount(), g.total) {
				fig = tx
			}
			if tx.s == hudClockText(g.time) {
				clock = tx
			}
		}
		if fig == nil || clock == nil {
			t.Fatalf("players=%v: figures or clock not written", two)
		}
		if fig.scale < 2 {
			t.Errorf("players=%v: BAJAS figures scale %.0f, want >= 2 (was 1)", two, fig.scale)
		}
		// glyph body = box minus the 3*scale margins used by the layout test
		figTop, figBot := fig.y+3*fig.scale, fig.y+fig.h-3*fig.scale
		clkTop := clock.y + 3*clock.scale
		if gap := clkTop - figBot; gap < 3 {
			t.Errorf("players=%v: only %.0f px between BAJAS and TIME", two, gap)
		}
		if gap := figTop - (hudY + hudBarY + 4); gap < 3 {
			t.Errorf("players=%v: only %.0f px between the bar and BAJAS", two, gap)
		}
	}
}
