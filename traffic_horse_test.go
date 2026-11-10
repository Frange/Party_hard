package main

import (
	"math/rand"
	"testing"
)

// Road levels: a vehicle every ~11 s on average (was ~16 s): count cars over 600 simulated seconds.
func TestRoadTrafficFrequency(t *testing.T) {
	def := level1()
	if def.Police != "car" {
		t.Skip("level1 has no road")
	}
	g := hudPlay(t, false)
	prevDef, prevLvl := g.lvl.Def, g.lvl
	t.Cleanup(func() { g.lvl = prevLvl; g.lvl.Def = prevDef })
	g.lvl = buildLevel(def)
	g.rng = rand.New(rand.NewSource(11))
	g.guests, g.amb = nil, nil
	g.eachPlayer(func(p *Player) { p.hidden = true })
	g.ambT = 1
	seen := map[*Ambient]bool{}
	empty, maxEmpty, run := 0, 0, 0
	const secs = 600
	for i := 0; i < 30*secs; i++ {
		g.updateAmbient(1.0 / 30)
		n := 0
		for _, a := range g.amb {
			if roadKind(a.kind) {
				seen[a] = true
				n++
			}
		}
		if n == 0 {
			empty++
			run++
			if run > maxEmpty {
				maxEmpty = run
			}
		} else {
			run = 0
		}
	}
	rate := float64(len(seen)) / secs * 60 // per minute
	t.Logf("%d vehicles in %ds = %.1f/min, empty %.0f%% of time, longest gap %.1fs", len(seen), secs, rate, 100*float64(empty)/(30*secs), float64(maxEmpty)/30)
	if rate < 4.5 || rate > 9 {
		t.Errorf("%.1f vehicles/min, want 4.5-9", rate)
	}
	if float64(maxEmpty)/30 > 25 {
		t.Errorf("road empty for %.1fs in a row", float64(maxEmpty)/30)
	}
}

// A moving horse tramples a guest in its path; one outside its hitbox, or a halted horse, kills nobody.
func TestHorseTramplesGuest(t *testing.T) {
	g := hudPlay(t, false)
	var gu *Guest
	for _, c := range g.guests {
		if !c.out() {
			gu = c
			break
		}
	}
	if gu == nil {
		t.Fatal("no guest")
	}
	g.eachPlayer(func(p *Player) { p.hidden = true })
	place := func(dx, dy, vx float64) bool {
		gu.pos = Vec{300 + dx, 200 + dy}
		before := gu.out()
		g.horseTrample(&Ambient{kind: ambHorse, x: 300, y: 200, vx: vx})
		return !before && gu.out()
	}
	if place(60, 0, 85) || place(0, 30, 85) {
		t.Error("guest outside the hitbox was killed")
	}
	if place(0, 0, 5) {
		t.Error("a halted horse killed")
	}
	if !place(5, 3, 85) {
		t.Error("a moving horse did not kill the guest in its path")
	}
}
