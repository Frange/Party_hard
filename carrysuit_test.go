package main

import "testing"

// With a body on the shoulder the player still picks up and puts on the gold suit, and a disguise
// bag is picked up and used as soon as he is wanted.
func TestCarryingBodyStillWearsSuit(t *testing.T) {
	g, pr := pickupSetup(t, false, "vipsuit")
	g.p.pos = Vec{200, 200}
	g.p.carrying = &Body{}
	g.interact()
	if g.p.suitT != suitDuration || !pr.gone {
		t.Fatalf("suit not worn with a body in the arms (suitT=%v gone=%v)", g.p.suitT, pr.gone)
	}
}

// The wardrobe ("Cambiarse de ropa") also works with a body in the arms: the label shows, so the
// press must do the change (it used to do nothing at all).
func TestCarryingBodyChangesClothesAtWardrobe(t *testing.T) {
	g, _ := mineGame(t, false)
	g.lvl.Props = []*Prop{{Name: "wardrobe", K: &PropKind{H: 20}, X: g.p.pos.X + 4, Y: g.p.pos.Y + 4, Foot: Rect{g.p.pos.X + 2, g.p.pos.Y + 2, 8, 4}}}
	g.p.carrying = &Body{}
	g.wanted = 60
	g.interact()
	if g.wanted != 0 {
		t.Fatalf("the change of clothes did nothing with a body in the arms (wanted=%v)", g.wanted)
	}
}
