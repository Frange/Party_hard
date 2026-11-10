package main

import "testing"

// trophyFrozen: the shared headless party with the progress frozen and empty (never the real one).
func trophyFrozen(t *testing.T) *Game {
	g, _, _ := albumNearGame(t)
	g.progress.Owned = nil
	prevOwned := cloneProgress(g.progress).Owned
	t.Cleanup(func() { g.progress.Owned = prevOwned; g.syncShop() })
	return g
}

func TestTrophy3Registered(t *testing.T) {
	for _, id := range []string{"s_lastone", "s_noknife", "s_shopall"} {
		n := 0
		for _, tr := range trophyList {
			if tr.id == id {
				n++
				if !secretTrophy[id] {
					t.Errorf("%s must be secret (it does not gate the last trophy)", id)
				}
				if tr.name == "" || tr.how == "" || tr.reward.id == "" {
					t.Errorf("%s incomplete: %+v", id, tr)
				}
			}
		}
		if n != 1 {
			t.Errorf("%s registered %d times", id, n)
		}
	}
	if trophyList[len(trophyList)-1].id != legendTrophy {
		t.Error("the legend trophy must stay last")
	}
}

// Dying with exactly ONE left to finish the party gives the trophy; with two left, no.
func TestTrophyDieWithOneLeft(t *testing.T) {
	for _, left := range []int{2, 1} {
		g := trophyFrozen(t)
		// The trophy depends on die() -> aliveCount() == 1, which reads state the shared game inherits from the
		// previous test and from the random party: pin it all (each global restored afterwards) instead of hoping
		// that the first guests of a random party happen to be alive and that nothing else is.
		prevGod, prevImm, prevDbg, prevStealth, prevMode, prevTwo := g.god, g.powerImmune, g.dbg.immune, g.stealth, g.mode, twoPlayerMode
		t.Cleanup(func() {
			g.god, g.powerImmune, g.dbg.immune, g.stealth, g.mode, twoPlayerMode = prevGod, prevImm, prevDbg, prevStealth, prevMode, prevTwo
		})
		g.god, g.powerImmune, g.dbg.immune, g.stealth, g.mode, twoPlayerMode = false, false, false, false, ModePlay, false
		g.guards, g.cops, g.slasher, g.slot2 = nil, nil, nil, slasherSlot{}
		g.p.dead = false
		kept := 0
		for _, gu := range g.guests {
			if gu.py != nil || gu.out() {
				continue // the arsonist does not count, and a guest who is already out cannot be one of the survivors
			}
			if kept < left {
				kept++
				continue
			}
			gu.dead = true
		}
		if g.aliveCount() != left {
			t.Fatalf("setup: %d alive, want %d", g.aliveCount(), left)
		}
		g.die("test")
		g.checkTrophies()
		if got := g.progress.Trophies["s_lastone"]; got != (left == 1) {
			t.Errorf("left=%d: trophy=%v", left, got)
		}
	}
}

// A party without a knife kill (poison / traps only) gives s_noknife; one knife kill spoils it.
func TestTrophyNoKnife(t *testing.T) {
	for _, knife := range []bool{false, true} {
		g := trophyFrozen(t)
		g.run = runStats{}
		g.countKill(kcPoison, true)
		g.countKill(kcTrap, true)
		g.countKill(kcBlast, false)
		if knife {
			g.countKill(kcNone, true) // the knife / hands
		}
		for _, gu := range g.guests {
			gu.dead = true
		}
		g.guards, g.cops = nil, nil
		g.completeLevel()
		g.checkTrophies()
		if got := g.progress.Trophies["s_noknife"]; got == knife {
			t.Errorf("knife=%v: trophy=%v", knife, got)
		}
	}
}

// Owning everything the shop sells gives s_shopall; missing one item does not.
func TestTrophyShopAll(t *testing.T) {
	g := trophyFrozen(t)
	n, total := shopTotal()
	if n < 50 || total < 10000 {
		t.Fatalf("the shop looks empty: %d items, %d points", n, total)
	}
	t.Logf("shop: %d items, %d points", n, total)
	var ids []string
	for _, list := range shopCatalog {
		for _, it := range list {
			if shopBuyable(it) {
				ids = append(ids, it.id)
			}
		}
	}
	g.progress.Owned = append([]string(nil), ids[1:]...)
	g.syncShop()
	g.trophyDirty = true
	g.checkTrophies()
	if g.progress.Trophies["s_shopall"] {
		t.Fatal("won with one item missing")
	}
	g.progress.Owned = append(g.progress.Owned, ids[0])
	g.syncShop()
	g.trophyDirty = true
	g.checkTrophies()
	if !g.progress.Trophies["s_shopall"] {
		t.Fatal("not won owning everything")
	}
}
