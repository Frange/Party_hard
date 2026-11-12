package main

import "testing"

// QA 2026-11-11: on the BARCO PIRATA a snake bit the player standing still at the start after 4 s. Nothing that
// kills (the snake, the animals that charge) may reach the player during his first ~10 s standing on Start, on any
// seed; and standing still on Start for 10 s he is alive.
func TestPirateStartSafeFromHazards(t *testing.T) {
	if len(levelDefs) < len(campaignLevels) {
		levelDefs = builtinLevels() // what newGame builds too (TestPirateKillTargetCapped does the same)
	}
	ps, pm, pr := snakeSeed, snakeMisses, snakeRng
	t.Cleanup(func() { snakeSeed, snakeMisses, snakeRng = ps, pm, pr })
	snakes := 0
	for seed := int64(1); seed <= 16; seed++ {
		snakeReseed(seed)
		snakeMisses = snakePityAfter // a snake in every one of these parties
		g := playGame(t, pirateIdx, seed)
		g.god, g.time = false, 0
		start := g.lvl.Def.Start
		g.p.pos = start
		for _, a := range g.animals {
			if a.kind != animSnake && !aggressiveAnimal(a.kind) {
				continue
			}
			if a.kind == animSnake {
				snakes++
			}
			if d := dist(a.pos, start); d < startSafeDist {
				t.Errorf("seed %d: %s born %.0f px from Start (min %.0f)", seed, animalKinds[a.kind], d, float64(startSafeDist))
			}
		}
		for f := 0; f < 600 && !g.p.dead; f++ {
			g.p.pos = start // standing still on Start
			g.updatePlay(1.0 / 60)
		}
		if g.p.dead {
			t.Errorf("seed %d: the player standing on Start died within 10 s: %s", seed, g.p.deathWhy)
		}
	}
	if snakes < 12 {
		t.Errorf("only %d snakes in 16 parties: the test did not meet the snake", snakes)
	}
}
