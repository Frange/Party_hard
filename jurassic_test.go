package main

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
)

// the island must be fully walkable: every spot a guest, a guard, a cop or the player has to reach is reachable
func TestJurassicReachable(t *testing.T) {
	fitAllFootprints() // as the game does when it starts
	def := jurassicLevel()
	if bad := genBad(def); len(bad) > 0 {
		t.Fatalf("unreachable spots on the island: %v", bad)
	}
	l := buildLevel(def)
	// the start and the key places can be walked to from each other
	for _, p := range []Vec{{478, 440}, {640, 150}, {250, 150}, {440, 320}, {200, 330}, {660, 320}, {830, 290}, {300, 420}, {640, 420}, {230, 130}} {
		if !l.Free(p) {
			p = l.NearestFree(p)
		}
		if l.Path(def.Start, p) == nil {
			t.Errorf("no path from the start to %v", p)
		}
	}
}

// the island is a stand-alone level: not in the campaign list, not in the random pool
func TestJurassicIsSpecial(t *testing.T) {
	for _, d := range builtinLevels() {
		if d.Title == "ISLA NUBLAR" {
			t.Fatal("the island must not be one of the normal levels")
		}
	}
}

// props, buildings, trees and fences must never stand on the roads (or in the way of the jeeps)
func TestJurassicRoadsClear(t *testing.T) {
	fitAllFootprints()
	def := jurassicLevel()
	l := buildLevel(def)                                               // the engine adds phones, items... too: check them as well
	roads := []Rect{jpRoadV.Grow(2), jpRoadH.Grow(2), jpRoadE.Grow(2)} // painted roads: nothing stands on them
	if len(def.Roads) != 0 || def.RoadCar != "" {
		t.Error("the island has no traffic: the jeeps are parked scenery")
	}
	for _, p := range l.Props {
		k := p.K
		foot := p.Foot
		if len(k.Solids) > 0 { // a gate: only its pillars count
			for _, s := range k.Solids {
				for _, r := range roads {
					if rectsOverlap(Rect{p.X + s.X, p.Y + s.Y, s.W, s.H}, r) {
						t.Errorf("%s at %.0f,%.0f stands on a road", p.Name, p.X, p.Y)
					}
				}
			}
			continue
		}
		// the foot, and the lower part of the picture (what you can see standing there)
		low := Rect{p.X - float64(k.W)/2, p.Y - math.Min(float64(k.H), 12), float64(k.W), math.Min(float64(k.H), 12)}
		for _, r := range roads {
			if rectsOverlap(foot, r) || rectsOverlap(low, r) {
				t.Errorf("%s at %.0f,%.0f stands on a road", p.Name, p.X, p.Y)
			}
		}
	}
	for _, w := range def.Walls {
		if w.Style == WallVoid {
			continue
		}
		for _, r := range roads {
			if rectsOverlap(w.R, r) {
				t.Errorf("a wall %v crosses a road", w.R)
			}
		}
	}
}

// the phone booths must not be hidden behind anything that is drawn after them
func TestJurassicPhonesVisible(t *testing.T) {
	fitAllFootprints()
	def := jurassicLevel()
	l := buildLevel(def)
	have := map[string]bool{}
	for _, p := range def.Props {
		have[fmt.Sprint(p.Kind, p.X, p.Y)] = true
	}
	for _, p := range jurassicProps() {
		if !have[fmt.Sprint(p.Kind, p.X, p.Y)] {
			t.Logf("pruned: %s %.0f,%.0f", p.Kind, p.X, p.Y)
		}
	}
	n := 0
	for _, p := range l.Props {
		if !p.K.Phone && !p.K.TechPhone {
			continue
		}
		n++
		box := Rect{p.X - float64(p.K.W)/2, p.Y - float64(p.K.H), float64(p.K.W), float64(p.K.H)}
		for _, q := range l.Props {
			if q == p || q.Sort <= p.Sort {
				continue
			}
			qb := Rect{q.X - float64(q.K.W)/2, q.Y - float64(q.K.H), float64(q.K.W), float64(q.K.H)}
			if rectsOverlap(box, qb) {
				t.Errorf("the phone at %.0f,%.0f is covered by a %s at %.0f,%.0f", p.X, p.Y, q.Name, q.X, q.Y)
			}
		}
	}
	if n < 3 {
		t.Errorf("only %d phones", n)
	}
}

// nothing (no prop, no vehicle) may float on the sea or in the water
func TestJurassicNothingOnWater(t *testing.T) {
	fitAllFootprints()
	def := jurassicLevel()
	l := buildLevel(def)
	for _, p := range l.Props {
		if p.Name == "jp_gate" { // it stands on the dock
			continue
		}
		if islandSD(p.X, p.Y-3) > -4 || l.InPool(Vec{p.X, p.Y - 3}) && p.K.Category != "" {
			t.Errorf("%s at %.0f,%.0f is on the sea or the water", p.Name, p.X, p.Y)
		}
		if p.Name == "jp_fin" {
			t.Errorf("a shark fin at %.0f,%.0f", p.X, p.Y)
		}
	}
}

// the gate has its own space: nothing else is placed on it or in front of it
func TestJurassicGateZoneClear(t *testing.T) {
	fitAllFootprints()
	def := jurassicLevel()
	l := buildLevel(def)
	for _, p := range l.Props {
		if p.Name == "jp_gate" {
			continue
		}
		low := Rect{p.X - float64(p.K.W)/2, p.Y - math.Min(float64(p.K.H), 14), float64(p.K.W), math.Min(float64(p.K.H), 14)}
		if rectsOverlap(p.Foot, jpGateZone) || rectsOverlap(low, jpGateZone) {
			t.Errorf("%s at %.0f,%.0f stands in the gate zone", p.Name, p.X, p.Y)
		}
	}
}

// whoever is stuck for more than 5 seconds is moved to a free spot nearby
func TestStuckEntitiesAreFreed(t *testing.T) {
	fitAllFootprints()
	def := jurassicLevel()
	g := &Game{lvl: buildLevel(def), rng: rand.New(rand.NewSource(3)), mode: ModePlay, walkBot: true}
	g.p = &Player{pos: Vec{500, 360}}
	pillar := Vec{408, 480} // inside the solid foot of a pillar of the gate
	if g.lvl.Free(pillar) && !g.lvl.SolidP(pillar.X, pillar.Y) {
		t.Fatal("the test spot is not blocked")
	}
	gu := &Guest{pos: pillar, state: gGoto, path: []Vec{{300, 300}}}
	g.guests = []*Guest{gu}
	g.rex = &JRex{pos: pillar, state: jrWander, scaredAt: map[*Guest]float64{}, skip: map[any]float64{}}
	for i := 0; i < 60*5+10; i++ { // 5 seconds and a bit
		g.time += 1.0 / 60
		g.watchAllStuck(1.0 / 60)
	}
	for name, p := range map[string]Vec{"guest": gu.pos, "rex": g.rex.pos} {
		if dist(p, pillar) < 6 || g.lvl.SolidP(p.X, p.Y) || !g.lvl.Free(p) {
			t.Errorf("the %s was not freed: %v", name, p)
		}
	}
	if g.rex.state != jrWander || len(gu.path) != 0 {
		t.Error("their old paths were not dropped")
	}
}

// the rex goes for the nearest person it sees in front of it: the player has no priority
func TestRexPicksNearestSeen(t *testing.T) {
	fitAllFootprints()
	def := jurassicLevel()
	g := &Game{lvl: buildLevel(def), rng: rand.New(rand.NewSource(5)), mode: ModePlay}
	g.rex = &JRex{pos: Vec{600, 420}, face: -1, state: jrWander, scaredAt: map[*Guest]float64{}, skip: map[any]float64{}}
	near := &Guest{pos: Vec{560, 420}}
	far := &Guest{pos: Vec{500, 420}}
	behind := &Guest{pos: Vec{660, 420}} // behind its back: not seen
	guard := &Guard{pos: Vec{530, 420}}
	g.guests = []*Guest{far, near, behind}
	g.guards = []*Guard{guard}
	g.p = &Player{pos: Vec{690, 420}} // the player is behind it too
	pick := func() any {
		pr, ok := g.rexPick()
		if !ok {
			return nil
		}
		return pr.id
	}
	if pick() != any(near) {
		t.Fatalf("it should go for the nearest person in front of it, got %v", pick())
	}
	g.p.pos = Vec{572, 420} // the player is now the nearest one in front of it
	if pr, _ := g.rexPick(); !pr.player {
		t.Fatal("the player is the nearest one: he is the target")
	}
	g.p.pos = Vec{520, 420} // farther than a guest: no priority
	if pick() != any(near) {
		t.Fatal("the player has no priority over somebody nearer")
	}
	near.dead = true // the guard is the next one in front of it
	if pick() != any(guard) {
		t.Fatalf("guards are prey too, got %v", pick())
	}
	g.guests, g.guards, g.p = nil, nil, &Player{pos: Vec{700, 420}}
	if _, ok := g.rexPick(); ok {
		t.Fatal("it sees nobody: it should wander")
	}
}

// no object of the island is bigger than the rex (about 100 px tall on screen)
func TestJurassicObjectsSmallerThanRex(t *testing.T) {
	def := jurassicLevel()
	for _, in := range def.Props {
		k := propKinds[in.Kind]
		if k == nil {
			continue
		}
		h := float64(k.H)
		if in.Scale > 0 {
			h *= in.Scale
		}
		if h > 100 {
			t.Errorf("%s is %.0f px tall: nearly as big as the rex", in.Kind, h)
		}
	}
}

func propBox(p *Prop) Rect {
	sc := p.sc()
	w, h := float64(p.K.W)*sc, float64(p.K.H)*sc
	return Rect{p.X - w/2, p.Y - h, w, h}
}

// no booth may hide part of another one (the silhouettes, not the footprints, are compared), and no tall
// object may be mostly covered by another; this includes what the engine adds when the level loads
func TestJurassicNoVisualOverlap(t *testing.T) {
	fitAllFootprints()
	def := jurassicLevel()
	l := buildLevel(def)
	cabin := func(p *Prop) bool {
		return p.K.Phone || p.K.TechPhone || p.Name == "portapotty" || p.Name == "fusebox" || p.Name == "closet"
	}
	loose := func(p *Prop) bool { // scenery that may overlap freely
		switch p.Name {
		case "palm", "tree", "bush", "jp_hut", "jp_egg", "jp_amber", "jp_warn", "jp_goat", "fountain":
			return true
		}
		return false
	}
	for i, a := range l.Props {
		for _, b := range l.Props[i+1:] {
			ar, br := propBox(a), propBox(b)
			ov := overlapArea(ar, br)
			if ov <= 0 {
				continue
			}
			if cabin(a) && cabin(b) {
				t.Errorf("%s at %.0f,%.0f and %s at %.0f,%.0f overlap (booths)", a.Name, a.X, a.Y, b.Name, b.X, b.Y)
				continue
			}
			small := math.Min(ar.W*ar.H, br.W*br.H)
			if !loose(a) && !loose(b) && a.Name != "jp_gate" && b.Name != "jp_gate" && ov > 0.35*small {
				t.Errorf("%s at %.0f,%.0f is mostly covered by %s at %.0f,%.0f", a.Name, a.X, a.Y, b.Name, b.X, b.Y)
			}
		}
	}
}

// every prop of the level, the ones the engine adds when it loads included, is drawn at the level's size
func TestJurassicAllPropsScaled(t *testing.T) {
	fitAllFootprints()
	def := jurassicLevel()
	l := buildLevel(def)
	for _, p := range l.Props {
		if p.Name == "jp_gate" || p.Name[:min(3, len(p.Name))] == "jp_" {
			continue
		}
		if p.scale <= 0 || p.scale > 0.7 {
			t.Errorf("%s at %.0f,%.0f is not scaled (%.2f)", p.Name, p.X, p.Y, p.scale)
		}
		if h := propBox(p).H; h > 100 {
			t.Errorf("%s at %.0f,%.0f is %.0f px tall", p.Name, p.X, p.Y, h)
		}
		switch p.Name {
		case "sign", "posters", "screen":
			if propBox(p).H > 30 {
				t.Errorf("the %s is bigger than a person (%.0f px)", p.Name, propBox(p).H)
			}
		}
	}
}
