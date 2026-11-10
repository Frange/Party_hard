package main

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func hudIconsSetAll(g *Game) {
	for _, p := range g.players() {
		p.item, p.spare, p.suitT, p.disguised = "poison", "disguise", suitDuration, true
	}
	g.wanted = 60
	for _, p := range g.players() {
		p.jail.sus = true
	}
}

// Icons only exist while their state is active, in 1 and 2 players.
func TestHUDStatusIconsOnlyWhenActive(t *testing.T) {
	for _, two := range []bool{false, true} {
		g := hudPlay(t, two)
		g.wanted = 0
		if n := len(g.hudIcons()); n != 0 {
			t.Fatalf("two=%v: %d icons in a calm party", two, n)
		}
		g.p.item = "poison"
		if ic := g.hudIcons(); len(ic) != 1 || ic[0].kind != hudPoison {
			t.Fatalf("two=%v: poison icon missing: %v", two, ic)
		}
		g.p.item, g.p.spare = "", "disguise"
		if ic := g.hudIcons(); len(ic) != 1 || ic[0].kind != hudDisguise {
			t.Fatalf("two=%v: disguise icon missing: %v", two, ic)
		}
		g.p.spare, g.p.suitT = "", suitDuration
		if ic := g.hudIcons(); len(ic) != 1 || ic[0].kind != hudSuit {
			t.Fatalf("two=%v: suit icon missing: %v", two, ic)
		}
		g.p.suitT = 0
		g.wanted = 60
		g.p.jail.sus = true
		if ic := g.hudIcons(); len(ic) != 1 || ic[0].kind != hudCuffs {
			t.Fatalf("two=%v: cuffs icon missing: %v", two, ic)
		}
		g.wanted = 0
		g.p.jail.sus = false
		gu := &Guest{sawKiller: true, state: gToPhone, sawP: g.p}
		g.guests = append(g.guests, gu)
		if ic := g.hudIcons(); len(ic) != 1 || ic[0].kind != hudAlert {
			t.Fatalf("two=%v: alert icon missing: %v", two, ic)
		}
		if two { // the other player has his own
			g.p2.item = "poison"
			if n := len(g.hudIcons()); n != 2 {
				t.Fatalf("two players: %d icons, want 2", n)
			}
		}
	}
}

// The strip of icons never touches the BAJAS figures nor the clock row, and it really paints pixels.
func TestHUDStatusIconsDrawAndFit(t *testing.T) {
	for _, two := range []bool{false, true} {
		g := hudPlay(t, two)
		g.hud.pts = 0
		g.wanted = 0
		for _, p := range g.players() {
			p.jail.sus = false
		}
		hudIconsSetAll(g)
		g.time = 3 // the suit does not blink
		for _, p := range g.players() {
			p.suitT = suitDuration
		}
		rec := hudRecord(g)
		icons := g.hudIcons()
		x0 := g.hudIconsX()
		right := x0 + hudStripW(icons)
		y0 := hudY + hudRowTY + hudIconDY
		for _, tx := range rec.texts {
			if rectsHit(x0, y0, right-x0, hudIconW+2, tx.x, tx.y+3*tx.scale, tx.w, tx.h-6*tx.scale) {
				t.Errorf("two=%v: the icons touch the text %q", two, tx.s)
			}
		}
		// the strip sits on the LEFT, next to the clock: after the clock text, inside the panel
		if x0 < hudX+hudPad+11 || right > hudX+hudW-hudPad {
			t.Errorf("two=%v: the strip is not next to the clock or leaves the panel (%v..%v)", two, x0, right)
		}
		if x0 > hudX+hudW/3 {
			t.Errorf("two=%v: the strip starts too far right (%v)", two, x0)
		}
		hudIconDrawn = 0
		on := ebiten.NewImage(screenW, screenH)
		g.drawHUDPanel(on)
		if hudIconDrawn != len(icons) {
			t.Errorf("two=%v: painted %d icons, want %d", two, hudIconDrawn, len(icons))
		}
	}
}

// With every icon of one player AND the RETO / CADENA chips (and the time left of the daily challenge) on the same row,
// the strip on the left never touches the chips on the right.
func TestHUDStatusIconsClearOfChips(t *testing.T) {
	g := hudPlay(t, false)
	hudIconsSetAll(g)
	g.time = 3
	g.p.suitT = suitDuration
	g.challenge = true
	g.dc = &dailyChallenge{mods: []dailyMod{dmTime}}
	t.Cleanup(func() { g.dc = nil })
	g.run.chainN[0], g.run.chainLast[0] = 12, g.time
	rec := hudRecord(g)
	x0 := g.hudIconsX()
	w := hudStripW(g.hudIcons())
	y0 := hudY + hudRowTY + hudIconDY
	chips := 0
	for _, tx := range rec.texts {
		if tx.s == "CADENA x12" || strings.HasPrefix(tx.s, "RETO") {
			chips++
			if rectsHit(x0, y0, w, hudIconW, tx.x-10, tx.y, tx.w+10, tx.h) {
				t.Errorf("the icons touch the chip %q", tx.s)
			}
		}
	}
	if chips < 2 {
		t.Fatalf("found %d chips, want 2", chips)
	}
}
