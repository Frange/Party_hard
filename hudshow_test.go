package main

import (
	"fmt"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// v1.1.0 bug: the HUD panel was never seen, because every guest, cop or animal drawn in the top left corner (and
// every party has people there) hid it. These tests run the REAL world draw (g.Draw) after loadLevel.

// hudFrames runs n frames of the real draw + the HUD step (what Update/Draw do in a party).
func hudFrames(g *Game, img *ebiten.Image, n int) {
	for i := 0; i < n; i++ {
		g.Draw(img)
		g.updateHUD(1.0 / 60)
	}
}

// hudLoad loads a party for the HUD tests, the players away from the panel.
func hudLoad(g *Game, idx int) {
	g.loadLevel(idx)
	g.lvl.graded = true
	g.titleT = 0
	g.mode = ModePlay
	for i, pl := range g.players() {
		pl.pos = Vec{600 / wk, (360 + 40*float64(i)) / wk}
	}
}

func TestHUDVisibleAtLevelStart(t *testing.T) {
	for _, two := range []bool{false, true} {
		g := hudPlay(t, two)
		img := ebiten.NewImage(screenW, screenH)
		defer img.Deallocate()
		for _, idx := range []int{0, 3, 7, 10, 12, 18, 24, 25} {
			t.Run(fmt.Sprintf("two=%v/level=%d", two, idx+1), func(t *testing.T) {
				hudLoad(g, idx)
				if a := g.hudAlpha(); a != 1 {
					t.Fatalf("alpha %.2f right after loadLevel", a)
				}
				hudFrames(g, img, 90)
				if a := g.hudAlpha(); a != 1 || !hudShown(g) {
					t.Errorf("alpha %.2f after 90 frames with no player under the panel (guests there: %d)", a, hudGuestsUnder(g))
				}
			})
		}
	}
}

// hudGuestsUnder counts the guests whose feet are inside the hide box (to show the test means something).
func hudGuestsUnder(g *Game) int {
	x0, y0, x1, y1 := g.hudHideRect()
	n := 0
	for _, gu := range g.guests {
		if sx, sy := g.wx(gu.pos.X), g.wy(gu.pos.Y); sx > x0 && sx < x1 && sy > y0 && sy < y1+40 {
			n++
		}
	}
	return n
}

// A guest standing right under the panel does not hide it; a player does, and it comes back when he leaves.
func TestHUDOnlyPlayersHideIt(t *testing.T) {
	for _, two := range []bool{false, true} {
		t.Run(fmt.Sprintf("two=%v", two), func(t *testing.T) {
			g := hudPlay(t, two)
			img := ebiten.NewImage(screenW, screenH)
			defer img.Deallocate()
			hudLoad(g, 0)
			if len(g.guests) == 0 {
				t.Fatal("no guests in the party")
			}
			g.guests[0].pos = Vec{100 / wk, 60 / wk} // a guest under the panel
			hudFrames(g, img, 60)
			if a := g.hudAlpha(); a != 1 {
				t.Fatalf("a guest under the panel hid it (alpha %.2f)", a)
			}
			for i, pl := range g.players() {
				pl.pos = Vec{(100 + 40*float64(i)) / wk, 60 / wk}
				hudFrames(g, img, 30)
				if a := g.hudAlpha(); a != 0 {
					t.Errorf("player %d under the panel: alpha %.2f, want 0", i+1, a)
				}
				pl.pos = Vec{(600 + 40*float64(i)) / wk, 400 / wk}
				hudFrames(g, img, 60)
				if a := g.hudAlpha(); a != 1 {
					t.Errorf("player %d left the panel: alpha %.2f, want 1", i+1, a)
				}
			}
		})
	}
}

// Hidden by a player, then a new party, the pause window or the results: the panel is shown.
func TestHUDShownAfterLoadPauseAndModes(t *testing.T) {
	g := hudPlay(t, false)
	img := ebiten.NewImage(screenW, screenH)
	defer img.Deallocate()
	hide := func() {
		hudLoad(g, 0)
		g.p.pos = Vec{100 / wk, 60 / wk}
		hudFrames(g, img, 30)
		if g.hudAlpha() != 0 {
			t.Fatalf("setup: the player did not hide the panel (alpha %.2f)", g.hudAlpha())
		}
	}
	hide()
	hudLoad(g, 1)
	if a := g.hudAlpha(); a != 1 {
		t.Errorf("after loadLevel: alpha %.2f", a)
	}
	hide()
	g.pause.open = true
	if a := g.hudAlpha(); a != 1 {
		t.Errorf("pause window open: alpha %.2f", a)
	}
	g.pause.open = false
	for _, m := range []Mode{ModeLevelDone, ModeLose} {
		hide()
		g.mode = m
		if a := g.hudAlpha(); a != 1 {
			t.Errorf("mode %d: alpha %.2f", m, a)
		}
	}
	g.mode = ModePlay
}
