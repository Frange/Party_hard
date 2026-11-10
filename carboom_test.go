package main

import (
	"math"
	"testing"
)

func f1Games(t *testing.T) []*Game {
	var out []*Game
	for i, def := range testLevels(t) {
		if i > 10 {
			break
		}
		g := capGame(def, int64(i)+7)
		if _, ok := g.clearStrip(12); ok {
			out = append(out, g)
		}
	}
	if len(out) == 0 {
		t.Skip("no level has a clear strip")
	}
	return out
}

// the car moves exactly as it always did (same path and speed) until it blows up where it used to leave;
// the explosion happens once, hurts nobody and the event cleans up by itself
func TestF1KeepsItsRunAndBlowsUpAtTheEnd(t *testing.T) {
	const dt = 1.0 / 60
	for _, g := range f1Games(t) {
		e := &Event{kind: evF1}
		if !g.startF1(e) {
			continue
		}
		x0, y0, vx := e.x, e.tgt[0], e.vx
		amp, ph := e.sp, e.tgt[1]
		booms := 0
		deadAtBoom := -1
		steps := 0
		for ; steps < 60*30; steps++ {
			e.t += dt
			was := e.boom != nil
			alive := g.stepF1(e, dt)
			if e.boom != nil && !was {
				booms++
				deadAtBoom = 0
				for _, gu := range g.guests {
					if gu.dead {
						deadAtBoom++
					}
				}
			}
			if e.boom == nil {
				// the original run: constant speed in x, the sway in y
				wantX := x0 + vx*e.t
				wantY := y0 + amp*math.Sin(e.t*3.4+ph)
				if math.Abs(e.x-wantX) > 0.01 || math.Abs(e.y-wantY) > 0.01 {
					t.Fatalf("%s: the car's path changed at t=%.2f (x %.2f/%.2f y %.2f/%.2f)", g.lvl.Def.Title, e.t, e.x, wantX, e.y, wantY)
				}
			}
			if !alive {
				break
			}
		}
		if steps >= 60*30 {
			t.Fatalf("%s: the event never ends", g.lvl.Def.Title)
		}
		if booms != 1 {
			t.Errorf("%s: %d explosions (want exactly 1)", g.lvl.Def.Title, booms)
		}
		// the blast kills nobody and leaves nothing behind
		dead := 0
		for _, gu := range g.guests {
			if gu.dead {
				dead++
			}
		}
		if dead != deadAtBoom {
			t.Errorf("%s: somebody died after the explosion (%d -> %d)", g.lvl.Def.Title, deadAtBoom, dead)
		}
		for _, p := range g.lvl.Props {
			if p.Name == "f1wreck" {
				t.Errorf("%s: a wreck was left behind", g.lvl.Def.Title)
			}
		}
		if len(e.boom.parts) == 0 {
			t.Errorf("%s: no pieces flew off", g.lvl.Def.Title)
		}
		if e.vx > 0 && e.x < g.lvl.W*0.5 || e.vx < 0 && e.x > g.lvl.W*0.5 {
			t.Errorf("%s: exploded in the wrong half (x=%.0f)", g.lvl.Def.Title, e.x)
		}
	}
}
