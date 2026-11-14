package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// medalToasts: the medals of the toast queue, in order (0 = a trophy toast).
func medalToasts(g *Game) []int {
	var out []int
	for _, t := range g.toasts {
		out = append(out, t.medal)
	}
	return out
}

func eqInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// One toast per medal and party, in order, never twice; nothing written into the save.
func TestMedalToastOncePerTier(t *testing.T) {
	g := medalGame(t)
	g.run, g.toasts, g.points = runStats{}, nil, 0
	before := len(g.progress.Medals)
	th := medalThresholds(g.partyMedalGoal())
	g.points = th[0] - 1
	g.medalLive()
	if len(g.toasts) != 0 {
		t.Fatalf("toast below bronze: %v", medalToasts(g))
	}
	g.points = th[0]
	g.medalLive()
	g.medalLive()
	if got := medalToasts(g); !eqInts(got, []int{medalBronze}) {
		t.Fatalf("after bronze (twice): %v", got)
	}
	g.points = th[1] + 10
	g.medalLive()
	g.points = th[0] // points going down (a seen streak kill taken back) never repeat a toast
	g.medalLive()
	g.points = th[1]
	g.medalLive()
	if got := medalToasts(g); !eqInts(got, []int{medalBronze, medalSilver}) {
		t.Fatalf("after silver: %v", got)
	}
	g.points = th[2]
	g.medalLive()
	g.points = th[2] * 3
	g.medalLive()
	if got := medalToasts(g); !eqInts(got, []int{medalBronze, medalSilver, medalGold}) {
		t.Fatalf("after gold: %v", got)
	}
	if len(g.progress.Medals) != before || g.run.medal != medalNone || g.run.medalNew {
		t.Fatalf("the toast touched the save / the medal of the party: %v %+v", g.progress.Medals, g.run)
	}
	// a new party (the same one retried) announces again
	g.loadLevel(0)
	g.mode = ModePlay
	g.toasts = nil
	g.points = th[0]
	g.medalLive()
	if got := medalToasts(g); !eqInts(got, []int{medalBronze}) {
		t.Fatalf("new party: %v", got)
	}
}

// Several medals at once: one toast each, queued after a trophy toast already waiting (never on top of it).
func TestMedalToastQueueAfterTrophy(t *testing.T) {
	g := medalGame(t)
	g.run, g.points = runStats{}, 0
	g.toasts = []trophyToast{{name: "Primera sangre", reward: "Gorro de novato"}}
	g.points = medalThresholds(g.partyMedalGoal())[2] + 500
	g.medalLive()
	if got := medalToasts(g); !eqInts(got, []int{0, medalBronze, medalSilver, medalGold}) {
		t.Fatalf("queue %v, want trophy then bronze, silver, gold", got)
	}
	// the queue shows one at a time: the first leaves after toastLife, then the bronze one
	g.updateToasts(toastLife + 0.1)
	if got := medalToasts(g); !eqInts(got, []int{medalBronze, medalSilver, medalGold}) {
		t.Fatalf("after the trophy toast: %v", got)
	}
}

// Only the parties that give a medal, only while playing; with the debug flags (frozen progress) it shows like the
// trophy toast does (checkTrophies also toasts with the progress frozen).
func TestMedalToastOnlyWhereMedals(t *testing.T) {
	g := medalGame(t)
	gold := medalThresholds(g.partyMedalGoal())[2]
	try := func(name string, set func(), want int) {
		t.Helper()
		g.run, g.toasts, g.points = runStats{}, nil, 0
		g.challenge, g.stealth, g.mode = false, false, ModePlay
		set()
		g.points = gold
		g.medalLive()
		if len(g.toasts) != want {
			t.Errorf("%s: %d toasts, want %d", name, len(g.toasts), want)
		}
	}
	try("normal party, frozen progress", func() { progressFrozen = true }, 3)
	try("RETO", func() { g.challenge = true }, 0)
	try("stealth", func() { g.stealth = true }, 0)
	try("end of the party (bonuses)", func() { g.run.done = true }, 0)
	try("paused / newspaper", func() { g.mode = ModeLevelDone }, 0)
	g.challenge, g.stealth, g.mode = false, false, ModePlay
}

// Two players: the points of the party (both together) against the thresholds of the real 2P target.
func TestMedalToastTwoPlayer(t *testing.T) {
	g := hudPlay(t, true)
	snapshotGlobals(t)
	progressFrozen = true
	t.Cleanup(func() { g.toasts, g.run = nil, runStats{} })
	if !g.is2P() || !g.partyMedalOn() {
		t.Fatal("no 2P campaign party")
	}
	two := medalThresholds(g.partyMedalGoal())
	one := medalThresholds(goalCap)
	if two[0] <= one[0] {
		t.Fatalf("2P bronze %d should be above the 1P one %d", two[0], one[0])
	}
	g.run, g.toasts = runStats{}, nil
	g.points = two[0] - 1
	g.medalLive()
	if len(g.toasts) != 0 {
		t.Fatalf("2P below its bronze: %v", medalToasts(g))
	}
	// J1 and J2 score: the combined g.points is what counts
	g.withPlayer(g.p, func() { g.points += 1 })
	g.medalLive()
	if got := medalToasts(g); !eqInts(got, []int{medalBronze}) {
		t.Fatalf("2P bronze: %v", got)
	}
}

// The medal toast is a smaller pill than the trophy one (two lines): inside the screen (16 px), clear of the HUD
// panel (12 px), every line inside the pill with >= 6 px, in Spanish and English, for every medal.
func TestMedalToastLayout(t *testing.T) {
	snapshotLang(t)
	for _, two := range []bool{false, true} {
		g := hudPlay(t, two)
		img := ebiten.NewImage(screenW, screenH)
		for _, lang := range []string{"es", "en"} {
			uiLang = lang
			for m := medalBronze; m <= medalGold; m++ {
				g.toasts = []trophyToast{{medal: m, medalGoal: g.partyMedalGoal(), t: 1}}
				g.drawHUDPanel(img)
				textRec = &textRecorder{}
				g.drawToast(img)
				rec := textRec
				textRec = nil
				const w, h = medalToastW, medalToastH // smaller than the trophy pill (2026-11-12)
				px, py := float64(screenW/2)-w/2, 16.0
				if py < 16 || px < 16 || px+w > screenW-16 {
					t.Fatalf("pill outside the margins")
				}
				if hudX+hudW+12 > px || g.hud.bottom < 0 {
					t.Fatalf("pill %v too close to the HUD panel (right edge %v)", px, hudX+hudW)
				}
				if len(rec.texts) != 2 {
					t.Fatalf("%s medal %d: %d lines", lang, m, len(rec.texts))
				}
				for _, tx := range rec.texts {
					if tx.x < px+6 || tx.x+tx.w > px+w-6 || tx.y < py+6 || tx.y+tx.h > py+h-6 {
						t.Errorf("%s medal %d: %q (%v,%v %vx%v) not 6 px inside the pill", lang, m, tx.s, tx.x, tx.y, tx.w, tx.h)
					}
					if tx.x != float64(int(tx.x)) || tx.y != float64(int(tx.y)) {
						t.Errorf("%q at a fractional position", tx.s)
					}
					if lang == "en" && (tx.s == "¡Medalla conseguida!" || tx.s == "Gana la fiesta para quedártela") {
						t.Errorf("Spanish in English: %q", tx.s)
					}
				}
			}
		}
		img.Deallocate()
		g.toasts = nil
	}
}
