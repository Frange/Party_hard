package main

import "testing"

// 2 players: the dead of a trap are the bajas and points of the player who armed it, and of nobody else
// (run statistics for the newspaper / level score and the co-op summary), whoever's turn it is.
func TestTrapKillsBelongToItsOwner(t *testing.T) {
	for owner := 0; owner < 2; owner++ {
		for _, kind := range []string{"explode", "gas"} {
			g := coopGame(t)
			g.run = runStats{}
			g.guards, g.cops, g.animals = nil, nil, nil
			g.guests = nil
			for k := 0; k < 3; k++ {
				g.guests = append(g.guests, &Guest{cs: g.plainGuestCS(), pos: Vec{500 + float64(k)*3, 400}, state: gIdle})
			}
			g.guests = append(g.guests, &Guest{cs: g.plainGuestCS(), pos: Vec{900, 100}, state: gIdle}) // the party goes on
			g.total = len(g.guests)
			trap := &Prop{Name: "bomb", K: &PropKind{Trap: kind, H: 20}, X: 500, Y: 408, Foot: Rect{495, 404, 10, 8}}
			g.lvl.Props = append(g.lvl.Props, trap)
			g.coopOwn(trap, g.players()[owner])
			g.p.pos, g.p2.pos = Vec{50, 50}, Vec{80, 50}
			other := 1 - owner
			g.triggerTrap(trap)
			for i := 0; i < 600; i++ { // the gas needs a few seconds
				g.time += 1.0 / 60
				g.asPlayer(other, func() { g.coopCreditOwner(g.coopTrapOwnerNow(), func() { g.updateTraps2(1.0 / 60) }) })
			}
			if g.run.killsOf(owner) < 1 || g.run.killsOf(other) != 0 {
				t.Errorf("%s owner=%d: run kills owner %d other %d", kind, owner, g.run.killsOf(owner), g.run.killsOf(other))
			}
			if g.coop.score[owner].kills < 1 || g.coop.score[other].kills != 0 || g.coop.score[other].points != 0 {
				t.Errorf("%s owner=%d: coop score owner %+v other %+v", kind, owner, g.coop.score[owner], g.coop.score[other])
			}
		}
	}
}
