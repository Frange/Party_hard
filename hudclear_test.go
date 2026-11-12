package main

import (
	"math"
	"math/rand"
	"strings"
	"testing"
)

// hudClearGame: the shared game in a party of campaign level idx with one or two players, playing.
func hudClearGame(t *testing.T, idx int, two bool) *Game {
	t.Helper()
	g := layerGame(t, two)
	snapshotGlobals(t)
	prevRng, prevBand, prevGod := g.rng, g.band, g.god
	t.Cleanup(func() {
		g.rng, g.band, g.god = prevRng, prevBand, prevGod
		g.cops, g.guards, g.bodies = nil, nil, nil
		g.mode = ModeMenu
	})
	g.rng = rand.New(rand.NewSource(11))
	g.loadLevel(idx)
	g.mode, g.titleT, g.time = ModePlay, 0, 30
	g.god = true
	if two && g.p2 == nil {
		t.Fatal("no second player")
	}
	return g
}

// rectGap: the free space between two rectangles (0 when they touch or overlap).
func hudRectGap(a, b Rect) float64 {
	dx := math.Max(0, math.Max(b.X-(a.X+a.W), a.X-(b.X+b.W)))
	dy := math.Max(0, math.Max(b.Y-(a.Y+a.H), a.Y-(b.Y+b.H)))
	return math.Max(dx, dy)
}

// screenToWorld: the world point drawn at screen (sx, sy).
func screenToWorld(g *Game, sx, sy float64) Vec {
	return Vec{(sx - g.camI.X) / wk, (sy - g.camI.Y) / wk}
}

// The BARCO PIRATA with two players: the band banner was drawn 3 px under "Ambulancia: J1 1 · J2 1". Wherever the band
// plays, the banner keeps hudClearGap (12 px) from the HUD block (panel + ambulance row), in Spanish and in English,
// hanging lower on shorter poles or beside the block; far from the HUD it does not move.
func TestBandSignClearOfHUDPirate2P(t *testing.T) {
	g := hudClearGame(t, pirateIdx, true)
	g.band = &bandList[3] // Simple Pan
	var band []*Guest
	for _, gu := range g.guests {
		if gu.perf != "" && gu.perf != "dj" {
			gu.perf = "" // only our four play
		}
		if len(band) < 4 && !gu.out() && gu.perf == "" && gu.quirk == "" {
			band = append(band, gu)
		}
	}
	for _, gu := range band {
		gu.perf = "guitar"
	}
	setup := func(g *Game) { g.lvl.graded = true }
	if _, err := auditRender(g, auditShot{name: "band", setup: setup}, "es"); err != "" { // the HUD sets its layout
		t.Fatal(err)
	}
	_, _, ry, _ := g.hudSide()
	// the musicians where the banner, without the rule, would hang 3 px under the ambulance row (x as on the ship)
	for _, sx := range []float64{60, 140, 230} {
		for _, gap := range []float64{3, -6, 8} {
			by := ry + 10 + gap + 24 // top of the banner = ambulance row bottom + gap
			for i, gu := range band {
				gu.pos = screenToWorld(g, sx+float64(i-1)*20, by+58)
			}
			for _, lang := range []string{"es", "en"} {
				rec, err := auditRender(g, auditShot{name: "band", setup: setup}, lang)
				if err != "" {
					t.Fatal(err)
				}
				hr, ok := g.hudBlockRect()
				if !ok {
					t.Fatal("the HUD is not on screen")
				}
				var amb *textRect
				for i, tx := range rec.texts {
					if strings.HasPrefix(tx.s, "Ambulancia") || strings.HasPrefix(tx.s, "Ambulance") {
						amb = &rec.texts[i]
					}
				}
				if amb == nil {
					t.Fatal("no ambulance row on screen")
				}
				if ar := (Rect{amb.x, amb.y, amb.w, amb.h}); ar.X < hr.X || ar.Y < hr.Y || ar.X+ar.W > hr.X+hr.W+1 || ar.Y+ar.H > hr.Y+hr.H+1 {
					t.Errorf("the HUD block %v does not hold the ambulance row %v", hr, ar)
				}
				bs := g.lastBandSign
				if bs.W == 0 {
					t.Fatalf("x %.0f gap %.0f %s: no banner drawn", sx, gap, lang)
				}
				if d := hudRectGap(bs, hr); d < hudClearGap {
					t.Errorf("x %.0f gap %.0f %s: the banner %v is %.0f px from the HUD %v", sx, gap, lang, bs, d, hr)
				}
				if bs.X < 0 || bs.X+bs.W > screenW || bs.Y < 0 || bs.Y+bs.H > screenH {
					t.Errorf("the banner %v leaves the screen", bs)
				}
			}
		}
	}
	// far from the HUD the banner hangs where it always did: 58 px over the highest musician, on 14 px poles
	for i, gu := range band {
		gu.pos = screenToWorld(g, 600+float64(i-1)*20, 400)
	}
	if _, err := auditRender(g, auditShot{name: "band", setup: setup}, "es"); err != "" {
		t.Fatal(err)
	}
	if bs := g.lastBandSign; bs.Y != g.wy(screenToWorld(g, 0, 400).Y)-58-24 || bs.H != 24+14 {
		t.Errorf("far from the HUD the banner moved: %v", bs)
	}
}

// The use prompts ("[E] ...") and the HUD come before the speech bubbles: the sleeping guard's "¡Eh!" sat on
// "[E] Necesita veneno" and a policeman's "¡Policía! Hay un loco suelto" on the HUD panel. A bubble that would cover
// one moves up off it or waits; it never covers it. 1 and 2 players.
func TestBubblesKeepOffUsePromptsAndHUD(t *testing.T) {
	for _, two := range []bool{false, true} {
		g := hudClearGame(t, 0, two)
		prevMgr := bubbleMgr
		t.Cleanup(func() { bubbleMgr = prevMgr })
		bubbleMgr = bubMgr{}
		// a prop with a use prompt, the player at it
		var pr *Prop
		for _, p := range g.lvl.Props {
			if lab, _ := g.useLabel(p); lab != "" && g.wy(p.Y) > 200 && g.wx(p.X) > 150 && g.wx(p.X) < screenW-150 {
				pr = p
				break
			}
		}
		if pr == nil {
			t.Fatal("no usable prop in the middle of the screen")
		}
		g.p.pos = g.lvl.NearestFree(Vec{pr.X, pr.Y + 12})
		if two {
			g.p2.pos = g.lvl.NearestFree(Vec{pr.X + 60, pr.Y + 60})
		}
		if g.useTextProp() != pr {
			t.Fatalf("the player at %v does not get the prompt of %s", g.p.pos, pr.Name)
		}
		for _, gu := range g.guests {
			gu.icon = ""
		}
		// a guest right beside the prop shouting, and a policeman under the HUD
		talker := g.guests[0]
		mx, my := g.useMarkXY(pr)
		talker.pos = screenToWorld(g, mx, my+52-16) // his bubble would sit on the prompt line
		talker.icon, talker.iconT = "¡Eh!", 30
		hr0 := Rect{hudX, hudY, hudW, 100}
		cop := &Cop{pos: screenToWorld(g, hr0.X+60, hr0.Y+hr0.H+30), icon: "¡Policía! Hay un loco suelto"}
		g.cops = append(g.cops, cop)
		setup := func(g *Game) { g.lvl.graded = true }
		for frame := 0; frame < 4; frame++ {
			bubbleMgr.frame += 60 // a second between frames: one newcomer bubble is let in each time (bubGap)
			rec, err := auditRender(g, auditShot{name: "bubbles", setup: setup}, "es")
			if err != "" {
				t.Fatal(err)
			}
			hr, ok := g.hudBlockRect()
			if !ok {
				t.Fatal("the HUD is not on screen")
			}
			var prompt *textRect
			for i, tx := range rec.texts {
				if strings.HasPrefix(tx.s, "[") && strings.Contains(tx.s, "] ") {
					prompt = &rec.texts[i]
				}
			}
			if prompt == nil {
				t.Fatal("the use prompt is not drawn")
			}
			pRect := Rect{prompt.x, prompt.y, prompt.w, prompt.h}
			for _, tx := range rec.texts {
				if tx.s != "¡Eh!" && !strings.HasPrefix(tx.s, "¡Policía!") && !strings.HasPrefix(tx.s, "Hay un loco") {
					continue
				}
				r := Rect{tx.x, tx.y, tx.w, tx.h}
				if rectsOverlap(r, pRect) {
					t.Errorf("2P=%v frame %d: the bubble %q %v covers the prompt %q %v", two, frame, tx.s, r, prompt.s, pRect)
				}
				if rectsOverlap(r, hr) {
					t.Errorf("2P=%v frame %d: the bubble %q %v covers the HUD %v", two, frame, tx.s, r, hr)
				}
			}
		}
	}
}

// The fixed "(E)" of the VIP trophy and the "Enter:" of the clothes in the shop name the pad button on a pad (O to
// help, X to accept by default), in Spanish and in English; on the keyboard they stay as they were.
func TestHowAndShopHintsNameThePadButton(t *testing.T) {
	prevPad, prevUI := dbgPadHints, uiLang
	t.Cleanup(func() { dbgPadHints, uiLang = prevPad, prevUI })
	loadLanguages()
	how := "Ayuda a un VIP que da propina (E)."
	desc := "Se lleva puesto en la fiesta. Enter: ponértelo / quitártelo."
	dbgPadHints = false
	if !hintOnPad(0) && (howHint(how) != "Ayuda a un VIP que da propina ("+actionHint(actInteract)+")." || shopDescHint(desc) != desc) {
		t.Errorf("keyboard: %q / %q", howHint(how), shopDescHint(desc))
	}
	dbgPadHints = true
	o, x := padShort(binds[actInteract].Pad), padShort(binds[actConfirm].Pad)
	for _, lang := range []string{"es", "en"} {
		uiLang = lang
		h, d := tr(howHint(how)), tr(shopDescHint(desc))
		wantH, wantD := "Ayuda a un VIP que da propina ("+o+").", "Se lleva puesto en la fiesta. "+x+": ponértelo / quitártelo."
		if lang == "en" {
			wantH, wantD = "Help a VIP who tips ("+tr(o)+").", "Worn at the party. "+tr(x)+": put on / take off."
		}
		if h != wantH || d != wantD {
			t.Errorf("%s pad: %q (want %q) / %q (want %q)", lang, h, wantH, d, wantD)
		}
	}
}
