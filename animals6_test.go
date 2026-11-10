package main

import (
	"math/rand"
	"testing"
)

// ownWild: the four wild animals of the shop are owned during the test (restored afterwards).
func ownWild(t *testing.T) {
	t.Helper()
	prev := shopOwned
	t.Cleanup(func() { shopOwned = prev })
	shopOwned = map[string]bool{}
	for k, v := range prev {
		shopOwned[k] = v
	}
	for _, k := range []string{"boar", "croc", "cheetah", "lion"} {
		shopOwned["animal:"+k] = true
	}
}

// the six new kinds are registered in order and each one has its art: stand + 4 walk frames that really move
func TestWildKindsAndArt(t *testing.T) {
	for k, name := range map[int]string{animFrog: "frog", animSquirrel: "squirrel", animBoar: "boar", animCroc: "croc", animCheetah: "cheetah", animLion: "lion", animCobaya: "cobaya"} {
		if animalKinds[k] != name || animalKindIndex(name) != k {
			t.Fatalf("%s is not at %d", name, k)
		}
		if animalLabel(k) == "Animal" || animalNameEs(k) == "un animal" {
			t.Errorf("%s has no name", name)
		}
		if tr := wildEN(animalLabel(k)); tr == animalLabel(k) {
			t.Errorf("%s: no English name", name)
		}
	}
	gens := map[string]func(f, c int) *pixBuf{
		"frog": genFrogFrame, "squirrel": genSquirrelFrame, "boar": genBoarFrame, "croc": genCrocFrame,
		"cheetah": func(f, c int) *pixBuf { return genBigCatFrame(f, c, false) },
		"lion":    func(f, c int) *pixBuf { return genBigCatFrame(f, c, true) },
		"cobaya":  genCobayaFrame,
	}
	for name, gen := range gens {
		for coat := 0; coat < 3; coat++ {
			stand := gen(fStand, coat)
			if wildOpaque(stand) < 20 {
				t.Errorf("%s coat %d: almost empty", name, coat)
			}
			moved := 0
			for f := fWalk0; f <= fWalk3; f++ {
				if string(gen(f, coat).p) != string(stand.p) {
					moved++
				}
			}
			if moved < 2 {
				t.Errorf("%s coat %d: the walk does not move (%d frames differ)", name, coat, moved)
			}
		}
	}
	if string(genCrocFrame(fWarn, 0).p) == string(genCrocFrame(fStand, 0).p) {
		t.Error("the crocodile does not open its jaws")
	}
	if string(genBigCatFrame(fStand, 0, true).p) == string(genBigCatFrame(fStand, 0, false).p) {
		t.Error("lion and cheetah look the same")
	}
}

func wildOpaque(b *pixBuf) int {
	n := 0
	for i := 3; i < len(b.p); i += 4 {
		if b.p[i] > 0 {
			n++
		}
	}
	return n
}

func wildEN(s string) string {
	prev := uiLang
	uiLang = "en"
	defer func() { uiLang = prev }()
	return tr(s)
}

// where they come: crocodiles only by the water, the big cats very rarely and only on green ground, never on
// levels without outdoor roam (station); frogs and squirrels on some levels only; the cap keeps holding.
func TestWildSpawnRules(t *testing.T) {
	ownWild(t)
	prevDbg := dbgWild
	t.Cleanup(func() { dbgWild = prevDbg; wildReseed(0) })
	dbgWild = ""
	wildReseed(4242)
	defs := testLevels(t)
	parties, cats, levelsWithSmall, levelsWithCroc := 0, 0, 0, 0
	for i, def := range defs {
		small, croc := false, false
		for rep := int64(0); rep < 8; rep++ {
			g := capGame(def, int64(i)*31+rep)
			g.spawnAnimals()
			parties++
			n, b := animalCounts(g)
			if n > animalMax || b > bullMax {
				t.Errorf("%s: %d animals, %d bulls", def.Title, n, b)
			}
			for _, a := range g.animals {
				switch a.kind {
				case animCroc:
					croc = true
					if !g.nearWater(a.area.Center()) {
						t.Errorf("%s: a crocodile far from the water", def.Title)
					}
				case animCheetah, animLion:
					cats++
					if !g.greenSpot(a.area.Center()) {
						t.Errorf("%s: a big cat off the grass", def.Title)
					}
				case animFrog, animSquirrel, animBoar, animCobaya:
					small = true
					if !g.lvl.Outdoor(a.area.Center()) {
						t.Errorf("%s: %s indoors", def.Title, animalKinds[a.kind])
					}
				}
				if a.kind > animSnake && (def.Sky == "space" || len(def.Roam) == 0) {
					t.Errorf("%s: wild animal on a level with no outdoors", def.Title)
				}
			}
		}
		if small {
			levelsWithSmall++
		}
		if croc {
			levelsWithCroc++
		}
	}
	t.Logf("%d levels, %d parties: small wild animals on %d levels, crocodiles on %d, big cats %d times", len(defs), parties, levelsWithSmall, levelsWithCroc, cats)
	if levelsWithSmall == 0 || levelsWithSmall == len(defs) {
		t.Errorf("frogs/squirrels/boars should come on some levels, not none or all (%d of %d)", levelsWithSmall, len(defs))
	}
	if levelsWithCroc == 0 {
		t.Error("no crocodile anywhere, not even by the pools")
	}
	if float64(cats) > float64(parties)*0.08 {
		t.Errorf("big cats too often: %d in %d parties", cats, parties)
	}
	// not bought: no boar, crocodile or big cat
	shopOwned = map[string]bool{}
	wildReseed(7)
	for i, def := range defs {
		g := capGame(def, int64(i))
		g.spawnAnimals()
		for _, a := range g.animals {
			if a.kind == animBoar || a.kind == animCroc || a.kind == animCheetah || a.kind == animLion {
				t.Fatalf("%s: %s without buying it", def.Title, animalKinds[a.kind])
			}
		}
	}
}

// a big cat makes ONE short hunt (at most one victim, never the player) and then leaves the party
func TestBigCatHuntsOnceAndLeaves(t *testing.T) {
	for _, kind := range []int{animCheetah, animLion} {
		g := killerGame(t, false)
		prevAnimals := g.animals
		t.Cleanup(func() { g.animals = prevAnimals })
		g.rng = rand.New(rand.NewSource(3))
		var near *Guest
		for _, gu := range g.guests {
			if !gu.out() && gu.perf == "" {
				near = gu
				break
			}
		}
		if near == nil {
			t.Fatal("no guest")
		}
		alive := g.aliveCount()
		cat := &Animal{kind: kind, pos: g.lvl.NearestFree(Vec{near.pos.X - 40, near.pos.Y}), face: 1, area: Rect{near.pos.X - 120, near.pos.Y - 80, 240, 160}}
		g.animals = []*Animal{cat}
		pdead := g.p.dead
		charged := false
		for i := 0; i < 60*40 && len(g.animals) > 0; i++ {
			g.updateAnimals(1.0 / 60)
			charged = charged || cat.state == aCharge
		}
		if !charged {
			t.Errorf("%s never hunted", animalKinds[kind])
		}
		if len(g.animals) != 0 {
			t.Errorf("%s: still at the party (state %d, bites %d)", animalKinds[kind], cat.state, cat.bites)
		}
		if cat.bites > 1 || alive-g.aliveCount() > 1 {
			t.Errorf("%s: %d bites, %d guests down", animalKinds[kind], cat.bites, alive-g.aliveCount())
		}
		if g.p.dead != pdead {
			t.Errorf("%s killed the player", animalKinds[kind])
		}
	}
}

// the crocodile bites one guest (and only one), however many stand in front of it
func TestCrocBitesAtMostOnce(t *testing.T) {
	g := killerGame(t, false)
	prevAnimals := g.animals
	t.Cleanup(func() { g.animals = prevAnimals })
	croc := &Animal{kind: animCroc, face: 1}
	g.animals = []*Animal{croc}
	alive := g.aliveCount()
	for round := 0; round < 4; round++ {
		var v *Guest
		for _, gu := range g.guests {
			if !gu.out() && gu.perf == "" && gu.state != gSleep {
				v = gu
				break
			}
		}
		if v == nil {
			break
		}
		croc.pos, croc.area, croc.huntT, croc.state = Vec{v.pos.X - 18, v.pos.Y}, Rect{v.pos.X - 40, v.pos.Y - 20, 80, 40}, 0, aIdle
		v.path, v.state, v.timer = nil, gIdle, 99
		for i := 0; i < 90; i++ {
			v.pos = Vec{croc.pos.X + 18*croc.face, croc.pos.Y}
			g.updateCroc(croc, 1.0/60)
		}
	}
	if croc.bites != 1 || alive-g.aliveCount() != 1 {
		t.Fatalf("crocodile: %d bites, %d down", croc.bites, alive-g.aliveCount())
	}
}

// guinea pigs: free, they come in groups of 2-4, three coats, and the group helper places exactly n
func TestCobayaGroupsAndCoats(t *testing.T) {
	if string(genCobayaFrame(fStand, 0).p) == string(genCobayaFrame(fStand, 1).p) || string(genCobayaFrame(fStand, 1).p) == string(genCobayaFrame(fStand, 2).p) {
		t.Error("guinea pig coats look the same")
	}
	if !animalAllowed(animCobaya) {
		t.Error("the guinea pig should be free")
	}
	g := killerGame(t, false)
	prev := g.animals
	t.Cleanup(func() { g.animals = prev })
	g.animals = nil
	g.addCobayaGroup(rand.New(rand.NewSource(1)), g.lvl.Def.Start, 3)
	if len(g.animals) != 3 || g.animals[0].kind != animCobaya {
		t.Fatalf("group: %d animals", len(g.animals))
	}
}
