package main

import (
	"fmt"
	"testing"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
)

func hintRestore(t *testing.T) {
	t.Helper()
	on, fn := levelHintOn, lhActionFn
	t.Cleanup(func() { levelHintOn, lhActionFn = on, fn })
	levelHintOn = true
	lhActionFn = func() bool { return false }
}

// Every text of the table fits (48 characters), every theme of the campaign has its entry, and EN exists for all.
func TestHintTableComplete(t *testing.T) {
	hintRestore(t)
	loadLanguages()
	check := func(s string) {
		if n := utf8.RuneCountInString(s); n == 0 || n > 48 {
			t.Errorf("hint %q has %d characters", s, n)
		}
		if _, ok := langDataEN[s]; !ok {
			t.Errorf("no English text for %q", s)
		}
	}
	for k, th := range hintThemes {
		check(th.goal)
		check(th.tip)
		_ = k
	}
	for _, s := range []string{hintGeneric.goal, hintGeneric.tip, hintVIP, hintCams, hintTraps, hintPoison, hintDisguise} {
		check(s)
	}
	for _, e := range campaignLevels {
		if _, ok := hintThemes[e.icon]; !ok {
			t.Errorf("campaign theme %q has no hint", e.icon)
		}
	}
	for _, k := range []string{"pub", "plaza", "mall"} {
		if _, ok := hintThemes[k]; !ok {
			t.Errorf("theme %q has no hint", k)
		}
	}
}

// The hint is chosen by level and by the REAL data of the level (VIP, cameras, traps...).
func TestHintChosenPerLevel(t *testing.T) {
	hintRestore(t)
	g := hudPlay(t, false)
	goals := map[string]bool{}
	sawVIP, sawCam := false, false
	for i := 0; i < campaignN; i++ {
		g.loadLevel(i)
		lines := g.levelHintLines()
		if len(lines) != 2 || lines[0] == "" || lines[1] == "" {
			t.Fatalf("level %d: lines %v", i, lines)
		}
		goals[lines[0]] = true
		if g.lvl.vip != nil {
			sawVIP = true
			if lines[1] != hintVIP {
				t.Errorf("level %d has a VIP zone but the tip is %q", i, lines[1])
			}
		} else if len(g.cams) > 0 {
			sawCam = true
			if lines[1] != hintCams {
				t.Errorf("level %d has cameras but the tip is %q", i, lines[1])
			}
		}
		if g.lh.lines == nil || g.lh.lines[0] != lines[0] {
			t.Errorf("level %d: the hint was not prepared by loadLevel", i)
		}
	}
	if len(goals) < campaignN-3 {
		t.Errorf("only %d different goals for %d levels", len(goals), campaignN)
	}
	if !sawVIP && !sawCam {
		t.Error("no level with a VIP zone nor cameras: the data tips were never exercised")
	}
	// a level without an entry takes the generic text
	g.lvl.Def.Icon, g.lvl.Def.Title = "", "NIVEL RARO"
	if l := g.levelHintLines(); l[0] != hintGeneric.goal {
		t.Errorf("unknown level: %v", l)
	}
}

// Shown after the start marker, closes by action and by time, never in stealth, off with the switch, 1P and 2P.
func TestHintLifecycle(t *testing.T) {
	for _, two := range []bool{false, true} {
		t.Run(fmt.Sprint("two=", two), func(t *testing.T) {
			hintRestore(t)
			g := hudPlay(t, two)
			g.stealth = false
			g.loadLevel(0)
			g.mode = ModePlay
			if g.levelHintActive() {
				t.Fatal("shown while the start marker still blinks")
			}
			g.sb.t = sbDur + 1
			g.updateLevelHint(0.1)
			if !g.levelHintActive() {
				t.Fatal("not shown after the start marker")
			}
			img := ebiten.NewImage(screenW, screenH)
			defer img.Deallocate()
			g.drawLevelHint(img)
			if !g.lh.drawn {
				t.Fatal("not drawn")
			}
			r := g.lh.rect
			if r[1] < 400 || r[0] < 200 || r[0]+r[2] > 760 {
				t.Errorf("the plate is out of its place: %v", r)
			}
			// closes with an action key
			lhActionFn = func() bool { return true }
			g.updateLevelHint(0.1)
			lhActionFn = func() bool { return false }
			if g.levelHintActive() {
				t.Fatal("did not close with a key")
			}
			// closes by time
			g.lh.reset(g)
			for i := 0; i < 400 && g.levelHintActive(); i++ {
				g.updateLevelHint(0.05)
			}
			if g.levelHintActive() || g.lh.t < lhDur {
				t.Fatalf("did not end by time: t=%v", g.lh.t)
			}
			// stealth: never
			g.lh.reset(g)
			g.stealth = true
			g.updateLevelHint(0.1)
			if g.levelHintActive() {
				t.Fatal("shown in stealth")
			}
			g.stealth = false
			// not outside the party
			g.mode = ModeLevelDone
			if g.levelHintActive() {
				t.Fatal("shown outside the party")
			}
			g.mode = ModePlay
			// the switch
			levelHintOn = false
			if g.levelHintActive() {
				t.Fatal("shown with the switch off")
			}
			g.drawLevelHint(img)
			if g.lh.drawn {
				t.Fatal("drawn with the switch off")
			}
		})
	}
}

// The plate does not touch the HUD panel, the "?" button, the trophy counter, the minimap, the controls strips or the
// level title, and its texts do not overlap (the audit's rules), in Spanish and English.
func TestHintLayoutNoOverlap(t *testing.T) {
	for _, two := range []bool{false, true} {
		for _, lang := range []string{"es", "en"} {
			t.Run(fmt.Sprintf("two=%v/%s", two, lang), func(t *testing.T) {
				hintRestore(t)
				g := hudPlay(t, two)
				gameLang, editorLang = lang, lang
				g.loadLevel(0)
				g.mode = ModePlay
				g.sb.t = sbDur + 1
				g.updateLevelHint(0.5)
				textRec = &textRecorder{}
				img := ebiten.NewImage(screenW, screenH)
				defer img.Deallocate()
				g.drawLevelHint(img)
				rec := textRec
				textRec = nil
				if len(rec.texts) != 2 {
					t.Fatalf("wrote %d texts", len(rec.texts))
				}
				for _, f := range auditCheck("hint", rec, true, nil, nil) {
					t.Errorf("%s", f.msg)
				}
				r := g.lh.rect
				others := map[string][4]float64{
					"hud":    {hudX, hudY, hudW, 90},
					"help":   {helpBtnX, helpBtnY, helpBtnW, helpBtnH},
					"stars":  {300, 0, 360, 80},
					"strip1": {200, screenH - 28, 560, 20},
					"strip2": {200, screenH - 66, 560, 44},
				}
				mx, my, mw, mh := mmRect()
				others["minimap"] = [4]float64{mx, my, mw, mh}
				for n, o := range others {
					if n == "strip2" && !two {
						continue
					}
					if n == "strip1" && two {
						continue
					}
					if rectsHit(r[0], r[1], r[2], r[3], o[0], o[1], o[2], o[3]) {
						t.Errorf("the plate %v touches %s %v", r, n, o)
					}
				}
			})
		}
	}
}
