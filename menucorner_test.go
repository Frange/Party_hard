package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// menuTexts draws the main menu in a language and returns every text it drew.
func menuTexts(t *testing.T, lang string, two bool) []textRect {
	t.Helper()
	g := auditGame(t)
	pl, ps, pl2, pu := twoPlayerMode, gameLang, editorLang, uiLang
	t.Cleanup(func() { twoPlayerMode, gameLang, editorLang, uiLang = pl, ps, pl2, pu; auditReset(g) })
	gameLang, editorLang = lang, lang
	auditReset(g)
	twoPlayerMode = two
	g.mode = ModeMenu
	textRec = &textRecorder{}
	t.Cleanup(func() { textRec = nil })
	img := ebiten.NewImage(screenW, screenH)
	g.Draw(img)
	img.Deallocate()
	return append([]textRect(nil), textRec.texts...)
}

// the "?" is centred in its box: the painted pixels of the glyph, measured with the real font, within 1 px
func TestHelpGlyphCenteredInBox(t *testing.T) {
	for _, lang := range []string{"es", "en"} {
		found := false
		for _, r := range menuTexts(t, lang, false) {
			if r.s != "?" {
				continue
			}
			found = true
			x0, y0, x1, y1 := glyphInk('?')
			if x1 <= x0 || y1 <= y0 {
				t.Fatal("the glyph has no ink")
			}
			cx := r.x + (x0+x1)/2*r.scale
			cy := r.y + (y0+y1)/2*r.scale
			if r.scale != float64(int(r.scale)) {
				t.Fatalf("scale %v is not a whole number", r.scale)
			}
			if dx := cx - (helpBtnX + helpBtnW/2); dx < -1 || dx > 1 {
				t.Errorf("[%s] glyph off centre horizontally by %.1f px", lang, dx)
			}
			if dy := cy - (helpBtnY + helpBtnH/2); dy < -1 || dy > 1 {
				t.Errorf("[%s] glyph off centre vertically by %.1f px", lang, dy)
			}
		}
		if !found {
			t.Errorf("[%s] no '?' drawn in the main menu", lang)
		}
	}
}

// the foot lines of the main menu are gone (the keys still work and live in the help window)
func TestMenuHasNoRemovedFootLines(t *testing.T) {
	for _, lang := range []string{"es", "en"} {
		for _, r := range menuTexts(t, lang, false) {
			l := strings.ToLower(r.s)
			for _, bad := range []string{"f11", "pantalla completa", "full screen", "fullscreen", "seleccionar", "volver", "m: son", "hecho por", "versión", "version "} {
				if strings.Contains(l, bad) {
					t.Errorf("[%s] the main menu still draws %q", lang, r.s)
				}
			}
		}
	}
	if !strings.Contains(helpGeneral, "F11") || !strings.Contains(helpGeneral, "M ") {
		t.Error("the help window lost the fixed keys")
	}
}

// version and homage: two lines, both right aligned to the same margin, nothing over them
func TestMenuVersionRightAligned(t *testing.T) {
	for _, lang := range []string{"es", "en"} {
		texts := menuTexts(t, lang, false)
		var ver, hom *textRect
		for i := range texts {
			switch {
			case texts[i].s == "v."+gameVersion+" - Frange":
				ver = &texts[i]
			case strings.HasPrefix(texts[i].s, "Homenaje a Party Hard"), strings.HasPrefix(texts[i].s, "Homage to Party Hard"):
				hom = &texts[i]
			}
		}
		if ver == nil || hom == nil {
			t.Fatalf("[%s] version or homage line missing", lang)
		}
		if !strings.Contains(hom.s, "Pinokl Games · tinyBuild)") {
			t.Errorf("[%s] homage text cut: %q", lang, hom.s)
		}
		for _, r := range []*textRect{ver, hom} {
			if edge := r.x + r.w; edge < float64(screenW)-menuCornerMargin-0.5 || edge > float64(screenW)-menuCornerMargin+0.5 {
				t.Errorf("[%s] %q right edge %.1f, want %.1f", lang, r.s, edge, float64(screenW)-menuCornerMargin)
			}
		}
		if ver.y >= hom.y {
			t.Errorf("[%s] the version must sit above the homage", lang)
		}
		for i := range texts {
			o := &texts[i]
			if o == ver || o == hom {
				continue
			}
			for _, r := range []*textRect{ver, hom} {
				if o.x < r.x+r.w && r.x < o.x+o.w && o.y < r.y+r.h && r.y < o.y+o.h {
					t.Errorf("[%s] %q overlaps %q", lang, o.s, r.s)
				}
			}
		}
	}
}

// J1 and J2 never share a key in the defaults, nor the label shown for it
func TestDefaultKeysNotSharedBetweenPlayers(t *testing.T) {
	b1, b2 := defaultBindings(), defaultBindings2()
	for a := 0; a < actPlayerCount; a++ {
		for s := 0; s < 2; s++ {
			k := b1[a].Keys[s]
			if k == noKey || (s == 1 && isArrowKey(k)) { // the one-player arrow alias: player 1 ignores it in two players
				continue
			}
			for c := 0; c < actPlayerCount; c++ {
				for _, k2 := range b2[c].Keys {
					if k2 == k {
						t.Errorf("both players have %s (J1 %s, J2 %s)", keyLabel(k), actionNames[a], actionNames[c])
					}
				}
			}
		}
	}
	if b1[actRun].Keys[0] == b2[actRun].Keys[0] {
		t.Error("same run key")
	}
	// what the help window shows
	r1, r2 := helpKeyboardRows(0), helpKeyboardRows(1)
	for i := range r1 {
		if r1[i].label == "Ayuda" {
			continue
		}
		for _, c1 := range r1[i].caps {
			for _, c2 := range r2[i].caps {
				if c1 == c2 && r1[i].label != "Menú / salir" {
					t.Errorf("help row %q shows %q for both players", r1[i].label, c1)
				}
			}
		}
	}
	if capName(ebiten.KeyShiftLeft) == capName(ebiten.KeyShiftRight) {
		t.Error("both Shift keys have the same label")
	}
}

func withSettingsFile(t *testing.T, content string) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "settings.json")
	if content != "" {
		os.WriteFile(p, []byte(content), 0o644)
	}
	ov, ob, ob2, ot, od := settingsFileOverride, binds, binds2, twoPlayerMode, devPref
	settingsFileOverride = p
	t.Cleanup(func() { settingsFileOverride, binds, binds2, twoPlayerMode, devPref = ov, ob, ob2, ot, od })
}

// an old settings.json with the same Shift for both players is fixed in memory; the file is not written
func TestOldSettingsKeyClashResolvedInMemory(t *testing.T) {
	old := map[string]savedBinding{
		"run":    {Keys: []string{"ShiftLeft", "ShiftRight"}, Pad: int(ebiten.StandardGamepadButtonFrontBottomRight)},
		"p2_run": {Keys: []string{"ShiftRight", ""}, Pad: int(ebiten.StandardGamepadButtonFrontBottomRight)},
		"attack": {Keys: []string{"Space", "J"}, Pad: 0},
		"up":     {Keys: []string{"W", "ArrowUp"}, Pad: int(ebiten.StandardGamepadButtonLeftTop)},
	}
	data, _ := json.Marshal(old)
	withSettingsFile(t, string(data))
	before, _ := os.ReadFile(settingsFileOverride)
	binds = loadBindings()
	loadPlayerSettings()
	if binds[actRun].Keys[0] != ebiten.KeyShiftLeft || binds[actRun].Keys[1] != noKey {
		t.Errorf("J1 run keys %v", binds[actRun].Keys)
	}
	if binds2[actRun].Keys[0] != ebiten.KeyShiftRight {
		t.Errorf("J2 run keys %v", binds2[actRun].Keys)
	}
	if binds[actAttack].Keys[0] != ebiten.KeySpace || binds[actAttack].Keys[1] != ebiten.KeyJ || binds[actUp].Keys[0] != ebiten.KeyW {
		t.Error("saved keys that did not clash were changed")
	}
	after, _ := os.ReadFile(settingsFileOverride)
	if string(before) != string(after) {
		t.Error("loading rewrote settings.json")
	}
	// a file without the entries just gives the defaults
	withSettingsFile(t, "")
	binds = loadBindings()
	loadPlayerSettings()
	if binds[actRun] != defaultBindings()[actRun] || binds2[actRun] != defaultBindings2()[actRun] {
		t.Error("defaults changed when there is no file")
	}
}

// assigning a key the other player has swaps it; the same player may repeat his own keys
func TestSettingsAssignKeySwapsBetweenPlayers(t *testing.T) {
	withSettingsFile(t, "")
	binds, binds2 = defaultBindings(), defaultBindings2()
	defer func(n string, c int) { settingsNote, settingsNoteT = n, c }(settingsNote, settingsNoteT)
	// J2 takes J1's left Shift for running
	assignKey(setRowP2+int(actRun), 0, ebiten.KeyShiftLeft)
	if binds2[actRun].Keys[0] != ebiten.KeyShiftLeft {
		t.Fatal("J2 did not get the key")
	}
	if binds[actRun].Keys[0] != ebiten.KeyShiftRight {
		t.Errorf("J1 should receive J2's old key, has %v", binds[actRun].Keys)
	}
	if settingsNoteT <= 0 || settingsNote == "" {
		t.Error("no notice shown")
	}
	for a := 0; a < actPlayerCount; a++ {
		for s := 0; s < 2; s++ {
			k := binds[a].Keys[s]
			if k == noKey || (s == 1 && isArrowKey(k)) {
				continue
			}
			for c := 0; c < actPlayerCount; c++ {
				if binds2[c].Keys[0] == k || binds2[c].Keys[1] == k {
					t.Errorf("key %s still shared", keyLabel(k))
				}
			}
		}
	}
	// within one player nothing changes
	assignKey(int(actInteract), 1, ebiten.KeyE)
	if binds[actInteract].Keys[0] != ebiten.KeyE || binds[actInteract].Keys[1] != ebiten.KeyE {
		t.Error("a player may repeat his own key")
	}
}

// the signs of the stealth mode read the real bindings
func TestStealthSignsReadBindings(t *testing.T) {
	ob, ob2 := binds, binds2
	t.Cleanup(func() { binds, binds2 = ob, ob2 })
	binds, binds2 = defaultBindings(), defaultBindings2()
	binds[actInteract].Keys[0] = ebiten.KeyQ
	binds2[actDance].Keys[0] = ebiten.KeyM
	if actKeyName(0, actInteract) != "Q" || actKeyName(1, actDance) != "M" || actKeyName(0, actDance) != "F" {
		t.Errorf("%s %s %s", actKeyName(0, actInteract), actKeyName(1, actDance), actKeyName(0, actDance))
	}
}
