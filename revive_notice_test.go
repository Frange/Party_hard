package main

import (
	"math"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// setLang switches the language of the texts for one test.
func setLang(t *testing.T, l string) {
	t.Helper()
	a, b, c := gameLang, editorLang, uiLang
	t.Cleanup(func() { gameLang, editorLang, uiLang = a, b, c })
	gameLang, editorLang, uiLang = l, l, l
	revivePx(t)
}

// revivePx: the white pixel every rectangle is drawn with (newGame makes it; under -shuffle it may be missing).
func revivePx(t *testing.T) {
	if px == nil {
		px = ebiten.NewImage(1, 1)
		t.Cleanup(func() { px = nil })
	}
}

// The notice is there only while the dead partner can still be saved, in Spanish and in English.
func TestReviveNoticeShownOnlyWhenItHelps(t *testing.T) {
	g := reviveGame()
	if _, ok := g.reviveNoticeBox(0); ok {
		t.Fatal("nobody is dead: no notice")
	}
	kill(g, g.p2, "Un asesino te ha dado caza")
	b, ok := g.reviveNoticeBox(0)
	if !ok || b.dead != g.p2 || b.caller != g.p {
		t.Fatal("J2 dead and revivable: the notice must be on and J1 is the one who calls")
	}
	if b.l1 != "Tu compañero J2 ha muerto" || !strings.HasPrefix(b.l2, "Llama a la ambulancia en un teléfono [") {
		t.Errorf("Spanish notice: %q / %q", b.l1, b.l2)
	}
	setLang(t, "en")
	if got := tr(b.l1); got != "Your partner P2 is dead" && got != "Your partner J2 is dead" {
		t.Errorf("English line 1: %q", got)
	}
	if got := tr(b.l2); !strings.HasPrefix(got, "Call the ambulance from a phone [") {
		t.Errorf("English line 2: %q", got)
	}
	if got := tr("Llamar a la ambulancia"); got != "Call the ambulance" {
		t.Errorf("English phone prompt: %q", got)
	}
	if got := tr(g.reviveHudLine()); !strings.HasPrefix(got, "Ambulance: ") || strings.Count(got, " 1") != 2 {
		t.Errorf("English HUD line: %q", got)
	}
	// once called, the notice gives way to the countdown
	g.callAmbulance(g.p, nil)
	if _, ok := g.reviveNoticeBox(0); ok {
		t.Error("the ambulance is coming: no notice")
	}
	if got := tr(g.reviveHudLine()); !strings.HasPrefix(got, "Ambulance in ") {
		t.Errorf("countdown in English: %q", got)
	}
}

// Deaths without remedy: no notice, a short grey line instead; the phone keeps cutting the line.
func TestReviveNoticeBlockedDeaths(t *testing.T) {
	for _, why := range []string{"Te ha comido el T-Rex", "Te has achicharrado", "Has saltado por los aires con tu propia trampa", "Un zombi te ha mordido"} {
		g := reviveGame()
		kill(g, g.p2, why)
		if _, ok := g.reviveNoticeBox(0); ok {
			t.Errorf("%q: no ambulance notice", why)
		}
		if got := g.reviveHudLine(); got != "J2 no se puede reanimar" {
			t.Errorf("%q: HUD %q", why, got)
		}
		if g.reviveHudH() != 14 {
			t.Errorf("%q: the HUD row stays one line", why)
		}
	}
}

// One player: no ambulance row, no notice, the column under the HUD as before.
func TestReviveNoticeOnePlayerUnchanged(t *testing.T) {
	g := twoGame(false)
	g.mode = ModePlay
	_, _, rev, jail := g.hudSide()
	if rev != jail {
		t.Error("one player: the ambulance row takes no room")
	}
	if d, c := g.reviveNoticeWho(); d != nil || c != nil {
		t.Error("one player: no notice")
	}
	revivePx(t)
	textRec = &textRecorder{}
	defer func() { textRec = nil }()
	img := ebiten.NewImage(screenW, screenH)
	defer img.Deallocate()
	g.drawReviveHud(img)
	if len(textRec.texts) != 0 || len(textRec.panels) != 0 {
		t.Error("one player: the ambulance HUD draws nothing")
	}
}

// The margins (user rule): >= 6 px of padding, >= 12 px from the other blocks, >= 16 px from the screen edge,
// whole pixels. Checked with what is really drawn, in Spanish and in English.
func TestReviveNoticeMargins(t *testing.T) {
	for _, l := range []string{"es", "en"} {
		t.Run(l, func(t *testing.T) {
			setLang(t, l)
			g := reviveGame()
			g.hud.bottom = 80
			g.p.item = "poison" // the OBJETO row above it
			g.p.stamina = 3     // and the stamina bars
			kill(g, g.p2, "Un asesino te ha dado caza")
			_, stam, rev, jail := g.hudSide()
			b, ok := g.reviveNoticeBox(rev)
			if !ok {
				t.Fatal("the notice must be on")
			}
			if rev-stam < 12 {
				t.Errorf("the box starts %.0f px under the stamina row", rev-stam)
			}
			if jail-(b.y+b.h) < 12 {
				t.Errorf("the next row starts %.0f px under the box", jail-(b.y+b.h))
			}
			if b.x < 16 || b.y < 16 || b.x+b.w > screenW-16 || b.y+b.h > screenH-16 {
				t.Errorf("the box %v leaves the 16 px margin", b)
			}
			textRec = &textRecorder{}
			defer func() { textRec = nil }()
			img := ebiten.NewImage(screenW, screenH)
			defer img.Deallocate()
			g.drawReviveHud(img)
			if len(textRec.panels) == 0 {
				t.Fatal("no box was drawn")
			}
			box := textRec.panels[0]
			if box.x != b.x || box.y != b.y || box.w != b.w || box.h != b.h {
				t.Fatalf("drawn box %+v differs from the layout %+v", box, b)
			}
			var lines []textRect
			for _, tx := range textRec.texts {
				if tx.y >= box.y && tx.y < box.y+box.h {
					lines = append(lines, tx)
				}
			}
			if len(lines) != 2 {
				t.Fatalf("two lines in the box, got %d", len(lines))
			}
			for _, tx := range lines {
				if tx.x != float64(int(tx.x)) || tx.y != float64(int(tx.y)) || tx.scale != 1 {
					t.Errorf("%q not on whole pixels", tx.s)
				}
				if tx.x-box.x < 6 || box.x+box.w-(tx.x+tx.w) < 6 || tx.y-box.y < 6 || box.y+box.h-(tx.y+tx.h) < 6 {
					t.Errorf("%q has less than 6 px of padding in the box %+v (text at %.0f,%.0f %.0fx%.0f)", tx.s, box, tx.x, tx.y, tx.w, tx.h)
				}
				if l == "en" && strings.ContainsAny(tx.s, "áéíóúñ") {
					t.Errorf("Spanish in English: %q", tx.s)
				}
			}
			if lines[1].y-(lines[0].y+lines[0].h) < 2 {
				t.Error("the two lines touch")
			}
		})
	}
}

// The phone: the prompt "[E] Llamar a la ambulancia" and the label "AMBULANCIA" sit in a box 12 px over the
// green badge of the same red booth (the "phone" prop), inside the screen margins.
func TestRevivePhonePromptAndLabel(t *testing.T) {
	for _, l := range []string{"es", "en"} {
		t.Run(l, func(t *testing.T) {
			setLang(t, l)
			g := reviveGame()
			kill(g, g.p2, "Un asesino te ha dado caza")
			ph := g.nearestAmbulancePhone(g.p)
			if ph == nil {
				t.Skip("the level has no phone booth")
			}
			if ph.K != propKinds["phone"] || !ph.K.Phone {
				t.Fatal("the ambulance phone must be the red booth that cuts the police line")
			}
			// far from it: the label over the nearest phone
			g.p.pos = Vec{ph.X + 200, ph.Y + 120}
			g.p.pos = g.lvl.NearestFree(g.p.pos)
			g.camI = Vec{math.Round(screenW/2 - ph.X*wk), math.Round(screenH/2 - ph.Y*wk)}
			textRec = &textRecorder{}
			defer func() { textRec = nil }()
			img := ebiten.NewImage(screenW, screenH)
			defer img.Deallocate()
			g.drawReviveHud(img)
			found := false
			for _, tx := range textRec.texts {
				if tx.s == tr("AMBULANCIA") {
					found = true
				}
			}
			if !found && distRect(g.p.pos, ph.Foot) >= revPromptNear {
				t.Error("the label AMBULANCIA must be over the nearest phone")
			}
			// next to it: the use prompt in its box, and no label
			g.p.pos = g.lvl.NearestFree(Vec{ph.Foot.X + ph.Foot.W/2, ph.Foot.Y + ph.Foot.H + 4})
			if lab, _ := g.useLabel(ph); lab != "Llamar a la ambulancia" {
				t.Fatalf("the phone must offer the ambulance: %q", lab)
			}
			textRec = &textRecorder{}
			g.drawUseMarks(img)
			g.drawReviveHud(img)
			var prompt *textRect
			for i, tx := range textRec.texts {
				if strings.HasPrefix(tx.s, "[") && strings.HasSuffix(tx.s, tr("Llamar a la ambulancia")) {
					prompt = &textRec.texts[i]
				}
				if tx.s == tr("AMBULANCIA") {
					t.Error("next to the phone the label gives way to the prompt")
				}
			}
			if prompt == nil {
				t.Fatal("the prompt [E] Llamar a la ambulancia was not drawn")
			}
			x, y, w, h, _, ok := g.revPhoneBox(ph, prompt.s, 8, g.p)
			badgeTop := math.Round(g.wy(ph.Y)-60) - 2
			if !ok || (badgeTop-(y+h) < 12 && y < g.wy(g.p.pos.Y)+12) {
				t.Errorf("the box must sit 12 px over the badge or 12 px under the player: box y %.0f-%.0f, badge %.0f, feet %.0f", y, y+h, badgeTop, g.wy(g.p.pos.Y))
			}
			if rw, rh := g.revHudReserved(); x < rw && y < rh {
				t.Errorf("the prompt box %.0f,%.0f is on the HUD (reserved %.0fx%.0f)", x, y, rw, rh)
			}
			if x < 16 || y < 16 || x+w > screenW-16 {
				t.Errorf("the prompt box %.0f,%.0f %.0fx%.0f leaves the margins", x, y, w, h)
			}
			if prompt.x-x < 6 || x+w-(prompt.x+prompt.w) < 6 || prompt.y-y < 6 || y+h-(prompt.y+prompt.h) < 6 {
				t.Errorf("the prompt has less than 6 px of padding: box %.0f,%.0f %.0fx%.0f text %.0f,%.0f %.0fx%.0f", x, y, w, h, prompt.x, prompt.y, prompt.w, prompt.h)
			}
		})
	}
}

// A phone under the HUD panel (the top-left corner of the map): the prompt moves under the player, clear of the HUD.
func TestRevivePhonePromptClearOfTheHud(t *testing.T) {
	g := reviveGame()
	g.hud.bottom = hudY + hudBaseH + 4
	kill(g, g.p2, "Un asesino te ha dado caza")
	ph := g.nearestAmbulancePhone(g.p)
	if ph == nil {
		t.Skip("the level has no phone booth")
	}
	g.p.pos = g.lvl.NearestFree(Vec{ph.Foot.X + ph.Foot.W/2, ph.Foot.Y + ph.Foot.H + 4})
	for _, at := range []Vec{{60, 70}, {200, 40}, {480, 40}, {900, 70}, {480, 500}} {
		g.camI = Vec{math.Round(at.X - ph.X*wk), math.Round(at.Y - ph.Y*wk)}
		txt := "[E] Llamar a la ambulancia"
		x, y, w, h, _, ok := g.revPhoneBox(ph, txt, 8, g.p)
		rw, rh := g.revHudReserved()
		if !ok || (x < rw && y < rh) || x < 16 || y < 16 || x+w > screenW-16 || y+h > screenH-16 {
			t.Errorf("phone at %v on screen: prompt box %.0f,%.0f %.0fx%.0f (HUD reserved %.0fx%.0f)", at, x, y, w, h, rw, rh)
		}
	}
}
