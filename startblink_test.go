package main

import "testing"

func sbGame(t *testing.T, two bool, lvl int) *Game {
	g := layerGame(t, two)
	g.loadLevel(lvl)
	g.mode = ModePlay
	return g
}

func TestStartBlinkExistsAndColours(t *testing.T) {
	for _, two := range []bool{false, true} {
		g := sbGame(t, two, 0)
		if !g.sb.active() || g.sb.alpha() <= 0 {
			// the blink may be at a dark phase of its sine: step a few ticks
			g.sb.t = 0.02
		}
		if !g.sb.active() {
			t.Fatalf("two=%v: the marker must exist at the start", two)
		}
		n := len(g.players())
		if two && n != 2 || !two && n != 1 {
			t.Fatalf("players=%d for two=%v", n, two)
		}
		for i, pl := range g.players() {
			if len(g.lvl.sbShape(pl.pos, 1, sbRadius(g))) == 0 {
				t.Errorf("two=%v player %d: empty marker", two, i)
			}
		}
		if playerColor(0) != rgb(0x2f7bff) || playerColor(1) != rgb(0xff4a4a) {
			t.Fatal("player colours: J1 saturated blue 0x2f7bff, J2 red 0xff4a4a")
		}
		if playerColor(0) == playerColor(1) {
			t.Fatal("the players must have different colours")
		}
	}
}

func TestStartBlinkEnds(t *testing.T) {
	g := sbGame(t, false, 0)
	for i := 0; i < 60*3+30; i++ {
		g.sb.update(g, 1.0/60)
	}
	if g.sb.active() || g.sb.alpha() != 0 {
		t.Fatal("the marker must be gone after 3 s")
	}
	g = sbGame(t, false, 0)
	g.p.moving = true
	for i := 0; i < 30; i++ { // 0.5 s: moved, but never less than 1.5 s
		g.sb.update(g, 1.0/60)
	}
	if !g.sb.active() {
		t.Fatal("it must stay at least 1.5 s")
	}
	for i := 0; i < 120; i++ {
		g.sb.update(g, 1.0/60)
	}
	if g.sb.active() {
		t.Fatal("after moving (and 1.5 s) it must go")
	}
}

func TestStartBlinkOnFreeGround(t *testing.T) {
	for _, lvl := range []int{0, 3, 6, 10, 15} {
		for _, two := range []bool{false, true} {
			g := sbGame(t, two, lvl)
			for i, pl := range g.players() {
				for _, c := range g.lvl.sbShape(pl.pos, 1, sbRadius(g)) {
					for x := c.x; x < c.x+c.w; x++ {
						if !g.lvl.sbFree(x, c.y) {
							t.Fatalf("lvl %d p%d: marker cell (%v,%v) on a solid", lvl, i, x, c.y)
						}
						for _, r := range g.lvl.solidsP {
							if x >= r.X && x < r.X+r.W && c.y >= r.Y && c.y < r.Y+r.H {
								t.Fatalf("lvl %d: cell inside a solid", lvl)
							}
						}
					}
				}
			}
		}
	}
}

func TestStartBlinkFitsBetweenPlayers(t *testing.T) {
	g := sbGame(t, true, 0)
	if 2*sbRadius(g) >= playerGap {
		t.Fatalf("the two markers would overlap: r=%v gap=%v", sbRadius(g), float64(playerGap))
	}
}

func TestStartBlinkHidesWhenNotPlaying(t *testing.T) {
	g := sbGame(t, false, 0)
	g.mode = ModeMenu
	// drawStartBlink must be a no-op outside play (no panic, nothing to draw)
	g.drawStartBlink(nil)
}
