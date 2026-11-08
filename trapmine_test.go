package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func mineGame(t *testing.T, two bool) (*Game, *Prop) {
	t.Helper()
	g := twoGame(two)
	g.time = 100
	g.p.pos = Vec{200, 200}
	if g.p2 != nil {
		g.p2.pos = Vec{210, 200}
	}
	g.lvl.Props = nil
	tr := &Prop{Name: "bomb", K: &PropKind{Trap: "explode", H: 20}, X: 400, Y: 304, Foot: Rect{395, 300, 10, 8}}
	g.lvl.Props = append(g.lvl.Props, tr)
	return g, tr
}

func tickProps(g *Game, secs float64) {
	for i := 0; i < int(secs*60); i++ {
		g.time += 1.0 / 60
		g.updateProps(1.0 / 60)
	}
}

func addGuestAt(g *Game, v Vec) *Guest {
	gu := &Guest{cs: g.plainGuestCS(), pos: v}
	g.guests = append(g.guests, gu)
	return gu
}

func TestTrapLoadsThenArmsAndDoesNotExplodeWhileLoading(t *testing.T) {
	g, tr := mineGame(t, false)
	gu := addGuestAt(g, Vec{402, 306}) // right on top of it
	g.armTrap(tr, g.p)
	tickProps(g, trapArmTime-0.3)
	if tr.used || tr.armed || tr.armT <= 0 {
		t.Fatalf("still loading: used=%v armed=%v armT=%.2f", tr.used, tr.armed, tr.armT)
	}
	if gu.dead {
		t.Fatal("a loading trap must not explode")
	}
	tickProps(g, 0.5)
	if !tr.armed && !tr.used {
		t.Fatal("after the load it is armed (or went off)")
	}
}

func TestTrapStaysArmedForEverWithNobodyNear(t *testing.T) {
	g, tr := mineGame(t, false)
	g.armTrap(tr, g.p)
	tickProps(g, trapArmTime+0.2)
	if !tr.armed {
		t.Fatal("must be armed")
	}
	tickProps(g, 600) // ten minutes
	if !tr.armed || tr.used {
		t.Fatal("an armed trap never expires")
	}
	far := addGuestAt(g, Vec{400 + trapDetectR + 20, 304})
	tickProps(g, 2)
	if tr.used || far.dead {
		t.Fatal("somebody far away must not set it off")
	}
}

func TestTrapExplodesWhenAGuestComesNear(t *testing.T) {
	g, tr := mineGame(t, false)
	g.armTrap(tr, g.p)
	tickProps(g, trapArmTime+0.2)
	gu := addGuestAt(g, Vec{400 + trapDetectR - 6, 304})
	tickProps(g, 0.1)
	if !tr.used || tr.armed {
		t.Fatal("must have gone off")
	}
	if !gu.dead {
		t.Fatal("the guest next to it dies")
	}
}

func TestTrapDetectsGuardCopAndAnimal(t *testing.T) {
	for _, who := range []string{"guard", "cop", "animal"} {
		g, tr := mineGame(t, false)
		g.armTrap(tr, g.p)
		tickProps(g, trapArmTime+0.2)
		near := Vec{405, 304}
		switch who {
		case "guard":
			g.guards = append(g.guards, &Guard{pos: near, cs: g.plainGuestCS()})
		case "cop":
			g.cops = append(g.cops, &Cop{pos: near})
		case "animal":
			g.animals = append(g.animals, &Animal{pos: near})
		}
		tickProps(g, 0.1)
		if !tr.used {
			t.Errorf("%s: an armed trap must detect it", who)
		}
	}
}

func TestTrapWaitsForTheTeamToLeaveThenAffectsEveryone(t *testing.T) {
	g, tr := mineGame(t, false)
	g.p.pos = Vec{402, 306}
	g.armTrap(tr, g.p)
	tickProps(g, trapArmTime+1)
	if tr.used || tr.armed || !tr.armWait {
		t.Fatal("loaded with the owner inside: it waits, not armed, not gone off")
	}
	g.p.pos = Vec{400 + trapDetectR + 30, 304}
	tickProps(g, 0.1)
	if !tr.armed || tr.used {
		t.Fatal("he left the radius: now it is armed")
	}
	g.p.pos = Vec{402, 306}
	tickProps(g, 0.1)
	if !tr.used {
		t.Fatal("coming back he sets it off like anybody")
	}
	if !g.p.dead {
		t.Fatal("and it really kills him (no immunity for the owner)")
	}
}

func TestTrapPerPlayerInTwoPlayers(t *testing.T) {
	g, tr := mineGame(t, true)
	g.p.pos, g.p2.pos = Vec{402, 306}, Vec{700, 700}
	g.armTrap(tr, g.p)
	if g.coop.trapOwner[tr] != 0 {
		t.Fatal("player 1 owns it")
	}
	tickProps(g, trapArmTime+0.5)
	if tr.used || tr.armed {
		t.Fatal("its owner is inside: it waits")
	}
	// the partner walks by while it waits: still nothing; the owner leaves, the partner stays inside
	g.p.pos, g.p2.pos = Vec{900, 900}, Vec{404, 306}
	tickProps(g, 0.1)
	if tr.armed || tr.used {
		t.Fatal("the partner is still inside: it waits for the whole team")
	}
	g.p2.pos = Vec{700, 700}
	tickProps(g, 0.1)
	if !tr.armed {
		t.Fatal("nobody of the team inside: armed")
	}
	// the partner walks by: it goes off and really kills him (no stun-only)
	g.p2.pos = Vec{404, 306}
	tickProps(g, 0.1)
	if !tr.used {
		t.Fatal("the partner is detected")
	}
	if !g.p2.dead {
		t.Fatal("the partner dies like anybody")
	}
	// player 2's own, independent trap
	g.p.pos = Vec{900, 900}
	t2 := &Prop{Name: "bomb2", K: &PropKind{Trap: "explode", H: 20}, X: 600, Y: 304, Foot: Rect{595, 300, 10, 8}}
	g.lvl.Props = append(g.lvl.Props, t2)
	g.armTrap(t2, g.p2)
	if g.coop.trapOwner[t2] != 1 {
		t.Fatal("player 2 owns the second")
	}
}

func TestTrapDrawingDoesNotPanic(t *testing.T) {
	layerGame(t, false) // creates the shared 1x1 pixel the drawing helpers use
	g, tr := mineGame(t, false)
	img := ebiten.NewImage(320, 240)
	g.armTrap(tr, g.p)
	for _, armT := range []float64{trapArmTime, trapArmTime / 2, 0.01} {
		tr.armT = armT
		g.drawTrapState(img, tr, 100, 100)
	}
	tr.armT, tr.armed = 0, true
	g.drawTrapState(img, tr, 100, 100)
	g.time += 0.3
	g.drawTrapState(img, tr, 100, 100)
	drawChargeArc(img, 20, 20, 7, 1.7, rgb(trapChargeCol))
	drawChargeArc(img, 20, 20, 7, -1, rgb(trapChargeCol))
}

// ---- poison and clothes can always be picked up -----------------------------------------------

func pickupSetup(t *testing.T, two bool, kind string) (*Game, *Prop) {
	g, _ := mineGame(t, two)
	k := propKinds[kind]
	pr := &Prop{Name: kind, K: k, X: g.p.pos.X + 6, Y: g.p.pos.Y + 4, Foot: Rect{g.p.pos.X + 4, g.p.pos.Y + 2, 8, 4}}
	g.lvl.Props = append(g.lvl.Props, pr)
	return g, pr
}

func TestPoisonAndClothesAlwaysPickable(t *testing.T) {
	for _, kind := range []string{"poisonbottle", "disguisebag"} {
		for _, state := range []string{"calm", "wanted", "alert", "cuffed", "carrying"} {
			g, pr := pickupSetup(t, false, kind)
			switch state {
			case "wanted":
				g.wanted = 60
			case "alert":
				g.wanted = 60
				g.dbgAlert = true
			case "cuffed":
				g.wanted = 60
				g.markSuspect(g.p)
			case "carrying":
				g.p.carrying = &Body{}
			}
			if lab, _ := g.useLabel(pr); lab == "" {
				t.Errorf("%s/%s: the label must show", kind, state)
			}
			g.interact()
			want := propKinds[kind].Item
			if g.p.item != want || !pr.gone {
				t.Errorf("%s/%s: not picked up (item=%q)", kind, state, g.p.item)
			}
		}
	}
}

func TestPoisonAndClothesPickableTwoPlayers(t *testing.T) {
	for idx := 0; idx < 2; idx++ {
		for _, kind := range []string{"poisonbottle", "disguisebag"} {
			g, _ := pickupSetup(t, true, kind)
			pl := g.players()[idx]
			g.lvl.Props = g.lvl.Props[:0]
			pr := &Prop{Name: kind, K: propKinds[kind], X: pl.pos.X + 6, Y: pl.pos.Y + 4, Foot: Rect{pl.pos.X + 4, pl.pos.Y + 2, 8, 4}}
			g.lvl.Props = append(g.lvl.Props, pr)
			g.wanted = 60
			g.markSuspect(pl)
			other := g.otherPlayer(pl)
			other.knockT = 5 // the partner is down next to him: the carry gesture used to take the key
			other.pos = Vec{pl.pos.X + 8, pl.pos.Y}
			g.asPlayer(idx, func() { g.interact() })
			if pl.item != propKinds[kind].Item || !pr.gone {
				t.Errorf("player %d %s: not picked up", idx+1, kind)
			}
			if other.item != "" {
				t.Errorf("player %d %s: the partner got it", idx+1, kind)
			}
		}
	}
}

func TestPickSecondKindKeepsBoth(t *testing.T) {
	g, pr := pickupSetup(t, false, "poisonbottle")
	g.p.item = "disguise"
	g.interact()
	if !pr.gone || g.p.item != "disguise" || g.p.spare != "poison" || g.p.spareDoses != poisonDoses {
		t.Fatalf("the poison goes to the second slot: item=%q spare=%q", g.p.item, g.p.spare)
	}
	g.useDisguise()
	if g.p.item != "poison" || g.p.doses != poisonDoses {
		t.Fatalf("after the disguise the poison comes out: %q %d", g.p.item, g.p.doses)
	}
}

func TestPickSameKindDoesNotWasteIt(t *testing.T) {
	g, pr := pickupSetup(t, false, "disguisebag")
	g.p.item = "disguise"
	g.interact()
	if pr.gone {
		t.Fatal("already wearing a disguise: the bag stays")
	}
}
