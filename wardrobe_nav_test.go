package main

import "testing"

// navFrame runs one frame of the shop screen with the given actions pressed (and the ATRÁS button when back is true).
func navFrame(g *Game, back bool, acts ...Action) {
	navFake = map[Action]bool{}
	for _, a := range acts {
		navFake[a] = true
	}
	backInject = back
	g.updateShop()
	navFake, backInject = nil, false
}

func navGame(t *testing.T) *Game {
	g := wardCharGame(t)
	pf, pi, pn, pw := backFocus, backInject, navFake, wardrobeNumPlayers
	t.Cleanup(func() { backFocus, backInject, navFake, wardrobeNumPlayers = pf, pi, pn, pw })
	backFocus = false
	g.mode, g.shopTab, wardKind, wardRow, wardP = ModeShop, shopWardrobe, wardKindPlayer, 0, 0
	g.progress.Album = map[string]int{"homer": albumSeen | albumElim | albumKilled}
	g.progress.CharKills = map[string]int{"homer": charKillsNeed}
	g.progress.Owned = []string{"skin:lisa"}
	g.syncShop()
	return g
}

// backNav: leaves with Esc / B (actBack) or with the on-screen ATRÁS button.
func backNav(g *Game, button bool) {
	if button {
		navFrame(g, true)
	} else {
		navFrame(g, false, actBack)
	}
}

// Tienda (VESTUARIO) > PERSONAJE > Personaje: Esc / B / the ATRÁS button leave the grid for the wardrobe, every time,
// and Atrás from the wardrobe goes out (never stuck in the grid).
func TestWardrobeNavBackFromCharPicker(t *testing.T) {
	for _, button := range []bool{false, true} {
		g := navGame(t)
		for round := 0; round < 3; round++ { // again and again, without re-entering the shop
			navFrame(g, false, actConfirm)
			if !wardPick {
				t.Fatalf("button=%v round %d: the row Personaje did not open the grid", button, round)
			}
			backNav(g, button)
			if wardPick || g.mode != ModeShop || g.shopTab != shopWardrobe || wardKind != wardKindPlayer || wardRow != 0 {
				t.Fatalf("button=%v round %d: back must land on the wardrobe, row Personaje (pick=%v mode=%v tab=%v row=%d)", button, round, wardPick, g.mode, g.shopTab, wardRow)
			}
		}
		backNav(g, button)
		if g.mode != ModeMenu {
			t.Fatalf("button=%v: Atrás from the wardrobe must leave the shop screen as always, mode %v", button, g.mode)
		}
	}
}

// confirming a character (key / pad A, or a click) wears it and goes back to the wardrobe, row Personaje.
func TestWardrobeNavConfirmReturnsToWardrobe(t *testing.T) {
	g := navGame(t)
	wardrobeNumPlayers = func(*Game) int { return 2 }
	for p := 0; p < 2; p++ {
		wardP = p
		navFrame(g, false, actConfirm)
		if !wardPick {
			t.Fatal("grid not open")
		}
		navFrame(g, false, actRight) // the cell after "Por defecto"
		navFrame(g, false, actConfirm)
		if wardPick || g.mode != ModeShop || g.shopTab != shopWardrobe || wardRow != 0 || wardP != p {
			t.Fatalf("player %d: after confirming the wardrobe must show (pick=%v row=%d wardP=%d)", p+1, wardPick, wardRow, wardP)
		}
		if g.equipOf(p, "base") == "" {
			t.Fatalf("player %d: confirmed character not worn", p+1)
		}
	}
	// a click on a cell does the same
	wardP = 0
	navFrame(g, false, actConfirm)
	x, y := wardPickCell(0)
	if !g.wardPickClickAt(int(x)+5, int(y)+5, g.wardPickList()) || wardPick || g.mode != ModeShop {
		t.Fatal("a click on a cell must wear it and return to the wardrobe")
	}
	// a locked cell stays in the grid
	navFrame(g, false, actConfirm)
	ll := g.wardPickList()
	for v := 0; v < wpkCols*wpkRows && v < len(ll); v++ {
		if ll[v].locked {
			cx, cy := wardPickCell(v)
			if g.wardPickClickAt(int(cx)+5, int(cy)+5, ll) || !wardPick {
				t.Fatal("a locked cell must not close the grid")
			}
			break
		}
	}
}

// the focused ATRÁS button (Backspace / LB, then A / Enter) does not open the grid: it goes back as usual
func TestWardrobeNavFocusedBackStillGoesBack(t *testing.T) {
	g := navGame(t)
	backFocus = true
	navFrame(g, true, actConfirm)
	if wardPick || g.mode != ModeMenu {
		t.Fatalf("focused ATRÁS + A: pick=%v mode=%v", wardPick, g.mode)
	}
}
