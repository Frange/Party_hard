package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"image/color"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

// Full round trip of the data the editor produces: paint, save, load, animate, build a level.
func TestAssetRoundTrip(t *testing.T) {
	root := t.TempDir()

	// ---- character painted pixel by pixel
	ca := newCharAsset("pruebas", "people", 24, 42)
	pt := newPainter()
	pt.usePart = true
	pt.col = color.RGBA{200, 50, 50, 255}
	for v := 0; v < 3; v++ {
		b := ca.Views[v]
		pt.part = PartHead
		pt.rectFill(b, 10, 10, 14, 17)
		pt.part = PartTorso
		pt.rectFill(b, 9, 18, 15, 27)
		pt.part = PartArmL
		pt.rectFill(b, 7, 18, 8, 28)
		pt.part = PartArmR
		pt.rectFill(b, 16, 18, 17, 28)
		pt.part = PartLegL
		pt.rectFill(b, 10, 28, 11, 38)
		pt.part = PartLegR
		pt.rectFill(b, 13, 28, 14, 38)
	}
	if err := saveCharAsset(root, ca); err != nil {
		t.Fatal(err)
	}
	ca2, err := loadCharAsset(filepath.Join(spritesDir(root, "people"), "pruebas"))
	if err != nil {
		t.Fatal(err)
	}
	cs := newRigCharSet(ca2)
	for a := 0; a < AnimCount; a++ {
		for v := 0; v < 3; v++ {
			for f := 0; f < animFrames[a]; f++ {
				fb := cs.rigFrame(a, []int{vFront, vSide, vBack}[v], f)
				if fb == nil || (a != AnimLying && a != AnimSleep && fb.w != 24) {
					t.Fatalf("bad frame anim %d view %d frame %d", a, v, f)
				}
			}
		}
	}

	// ---- object
	oa := &ObjAsset{Meta: ObjMeta{Name: "mesa_test", Category: "furniture", AnchorX: 8, AnchorY: 15, FootX: 0, FootY: 10, FootW: 16, FootH: 5, Solid: true, Hide: true},
		Frames: []*pixBuf{newPix(16, 16)}}
	oa.Frames[0].rect(0, 0, 16, 16, rgb(0x884422))
	if err := saveObjAsset(root, oa); err != nil {
		t.Fatal(err)
	}
	ob2, err := loadObjAsset(filepath.Join(spritesDir(root, "objetos"), "mesa_test"))
	if err != nil || ob2.Meta.Hide != true || len(ob2.Frames) != 1 {
		t.Fatal("object round trip failed", err)
	}

	// ---- level
	la := newLevelAsset("nivel_test")
	for y := 40; y < 50; y++ {
		row := []byte(la.Meta.Solid[y])
		for x := 40; x < 100; x++ {
			row[x] = '#'
		}
		la.Meta.Solid[y] = string(row)
	}
	for y := 10; y < 30; y++ {
		row := []byte(la.Meta.Zone[y])
		for x := 20; x < 120; x++ {
			row[x] = 'r'
		}
		la.Meta.Zone[y] = string(row)
	}
	la.Meta.Guards = [][]LevelPoint{{{100, 100}, {200, 100}}}
	if err := saveLevelAsset(root, la); err != nil {
		t.Fatal(err)
	}
	la2, err := loadLevelAsset(filepath.Join(backgroundsDir(root), "nivel_test"))
	if err != nil {
		t.Fatal(err)
	}
	if la2.Meta.Solid[45][50] != '#' || la2.Meta.Zone[20][30] != 'r' {
		t.Fatal("level logic lost")
	}
	rects := mergeCells(la2.Meta.Solid, func(c byte) bool { return c == '#' })
	if len(rects) != 1 || rects[0].W != 60 || rects[0].H != 10 {
		t.Fatalf("merge gave %v", rects)
	}
	_ = os.Stdout
}

func TestPainterTools(t *testing.T) {
	b := newPix(20, 20)
	b.pid = make([]uint8, 400)
	pt := newPainter()
	pt.usePart = true
	pt.part = PartArmL
	pt.mirror = true
	pt.col = color.RGBA{1, 2, 3, 255}
	pt.setPx(b, 2, 3, false, true)
	if b.pid[3*20+2] != PartArmL || b.pid[3*20+17] != PartArmR {
		t.Fatal("mirror did not swap parts")
	}
	pt.mirror = false
	pt.rectFill(b, 5, 5, 8, 8)
	pt.col = color.RGBA{9, 9, 9, 255}
	pt.flood(b, 6, 6)
	if b.p[(6*20+6)*4] != 9 {
		t.Fatal("flood failed")
	}
}

func TestAutoPartsRig(t *testing.T) {
	spec := genericSpec(rand.New(rand.NewSource(4)))
	ca := charAssetFromSpec(spec, "auto", "people", "")
	for i, v := range ca.Views {
		for k := range v.pid {
			v.pid[k] = 0
		}
		autoParts(v, i == 1)
	}
	rig := buildRig(ca)
	for a := 0; a < AnimCount; a++ {
		for f := 0; f < animFrames[a]; f++ {
			for _, vw := range []int{vFront, vSide, vBack} {
				if rig.render(vw, poseFor(a, f)) == nil {
					t.Fatal("nil frame")
				}
			}
		}
	}
	for id := 1; id <= 4; id++ {
		if !rig.V[0].L[id].ok {
			t.Fatalf("front view lacks part %d after auto tagging", id)
		}
	}
}

func hasPart(b *pixBuf, id uint8) bool {
	_, _, _, _, ok := partBBox(b, id)
	return ok
}

// TestSynthesizeSideBack checks the wizard's front->side and front->back
// templates: the side view should paint only the near arm/leg (PartArmR/
// PartLegR), relying on buildRig's existing far-limb borrowing (rig.go) for
// the rest, and the back view should at least carry over a torso.
func TestSynthesizeSideBack(t *testing.T) {
	front := newPix(24, 42)
	front.pid = make([]uint8, 24*42)
	pt := newPainter()
	pt.usePart = true
	pt.col = color.RGBA{200, 50, 50, 255}
	pt.part = PartHead
	pt.rectFill(front, 10, 10, 14, 17)
	pt.part = PartTorso
	pt.rectFill(front, 9, 18, 15, 27)
	pt.part = PartArmL
	pt.rectFill(front, 7, 18, 8, 28)
	pt.part = PartArmR
	pt.rectFill(front, 16, 18, 17, 28)
	pt.part = PartLegL
	pt.rectFill(front, 10, 28, 11, 38)
	pt.part = PartLegR
	pt.rectFill(front, 13, 28, 14, 38)

	side := synthesizeSide(front)
	if !hasPart(side, PartHead) || !hasPart(side, PartTorso) {
		t.Fatal("side view missing head or torso")
	}
	if !hasPart(side, PartArmR) || !hasPart(side, PartLegR) {
		t.Fatal("side view missing the near arm/leg")
	}
	if hasPart(side, PartArmL) || hasPart(side, PartLegL) {
		t.Fatal("side view should leave the far arm/leg for buildRig to borrow")
	}

	ca := newCharAsset("wiztest", "people", 24, 42)
	ca.Views[0] = front
	ca.Views[1] = side
	ca.Views[2] = synthesizeBack(front)
	rig := buildRig(ca)
	if !rig.V[1].L[PartArmL].ok || !rig.V[1].L[PartLegL].ok {
		t.Fatal("buildRig did not borrow the missing far arm/leg from the near one")
	}
	if !hasPart(ca.Views[2], PartTorso) {
		t.Fatal("back view missing torso")
	}
}

// TestPeeGenderPose checks the one gender-aware pose: standing for male,
// sitting for female, at every frame of the animation.
func TestPeeGenderPose(t *testing.T) {
	for f := 0; f < animFrames[AnimPee]; f++ {
		if peePose(f, false).Sit {
			t.Fatalf("male pee pose should stand, frame %d", f)
		}
		if !peePose(f, true).Sit {
			t.Fatalf("female pee pose should sit, frame %d", f)
		}
	}
}

// reachable flood-fills the level the way the player walks it (SolidP, 4px steps).
func reachable(l *Level, from Vec) map[[2]int]bool {
	seen := map[[2]int]bool{}
	cell := func(v Vec) [2]int { return [2]int{int(v.X) / 4, int(v.Y) / 4} }
	start := l.NearestFree(from)
	q := []Vec{start}
	seen[cell(start)] = true
	for len(q) > 0 {
		c := q[0]
		q = q[1:]
		for _, d := range [4]Vec{{4, 0}, {-4, 0}, {0, 4}, {0, -4}} {
			n := Vec{c.X + d.X, c.Y + d.Y}
			if n.X < 0 || n.Y < 0 || n.X >= l.W || n.Y >= l.H || seen[cell(n)] || l.SolidP(n.X, n.Y) {
				continue
			}
			seen[cell(n)] = true
			q = append(q, n)
		}
	}
	return seen
}

// TestPlayerReachesPeople: on the hand-made levels the player must be able to
// walk to every performer, dance floor and roam rectangle centre (Miami's stage
// used to be walled off by its footprint, so the level could not be finished).
func TestPlayerReachesPeople(t *testing.T) {
	for i, def := range builtinLevels()[:5] {
		l := buildLevel(def)
		seen := reachable(l, def.Start)
		check := func(what string, p Vec) {
			near := false
			for dy := -8.0; dy <= 8 && !near; dy += 4 {
				for dx := -8.0; dx <= 8; dx += 4 {
					if seen[[2]int{int(p.X+dx) / 4, int(p.Y+dy) / 4}] {
						near = true
						break
					}
				}
			}
			if !near {
				t.Errorf("level %d (%s): player cannot reach %s at %v", i, def.Title, what, p)
			}
		}
		for _, pf := range def.Performers {
			check("performer "+pf.Role, pf.Pos)
		}
		for _, d := range def.Dance {
			check("dance floor", d.Center())
		}
	}
}

// Fights and murders between other people never raise the criminality;
// only the player's own kills do.
func TestStarsOnlyFromPlayer(t *testing.T) {
	g := &Game{policeCalled: true, pendingCar: -1, pendingUnit: -1, total: 60} // 60 people: one star per 10 located kills
	for i := 0; i < 12; i++ {
		g.bodies = append(g.bodies, &Body{reported: true})
	}
	if g.playerHeat() || g.starsTarget() != 1 {
		t.Fatalf("NPC bodies must not count: heat=%v target=%d", g.playerHeat(), g.starsTarget())
	}
	for i := 0; i < 30; i++ {
		g.bodies = append(g.bodies, &Body{reported: true, byPlayer: true})
	}
	if !g.playerHeat() || g.starsTarget() != 3 {
		t.Fatalf("player bodies must count: target=%d", g.starsTarget())
	}
	if g.callerAvailable() {
		t.Fatal("nobody calls the police again once they are called")
	}
}

func TestCopMinimumStay(t *testing.T) {
	old := jailHold // global custody flag: nobody is held in this test
	jailHold = false
	t.Cleanup(func() { jailHold = old })
	c := &Cop{}
	if c.mayLeave() {
		t.Fatal("a fresh officer must stay")
	}
	c.onSceneT = 20
	if !c.mayLeave() {
		t.Fatal("after 20s the officer may leave")
	}
	// medics are not bound by the police minimum, but (2026-11-11) a paramedic never goes back
	// without having worked: a body taken, or the place examined for medicMinAct seconds
	if (&Cop{medic: true}).mayLeave() {
		t.Fatal("a paramedic who did nothing must not leave")
	}
	if !(&Cop{medic: true, carried: 1}).mayLeave() || !(&Cop{medic: true, actT: medicMinAct}).mayLeave() {
		t.Fatal("a paramedic who worked may leave")
	}
}

func TestBindingsDefaultsAndLabels(t *testing.T) {
	b := defaultBindings()
	if b[actAttack].Keys[0] != ebiten.KeySpace || b[actConfirm].Keys[0] != ebiten.KeyEnter {
		t.Fatal("defaults must match the old fixed controls")
	}
	if keyByName(ebiten.KeyF.String()) != ebiten.KeyF {
		t.Fatal("key names must round-trip")
	}
	if keyLabel(noKey) != "-" || padLabel(noPad) != "-" {
		t.Fatal("unset bindings show a dash")
	}
}

func TestShopGating(t *testing.T) {
	shopOwned = map[string]bool{"skin:": true, "police:0": true}
	if unitForStars(4) != unitPolice || unitForStars(2) != unitPolice {
		t.Fatal("locked police tiers must fall back to the local police")
	}
	shopOwned["police:1"] = true
	if unitForStars(2) != unitFBI || unitForStars(4) != unitFBI {
		t.Fatal("an unlocked tier answers for the stars above it too")
	}
	if animalAllowed(animalKindIndex("elephant")) || !animalAllowed(animalKindIndex("dog")) {
		t.Fatal("elephants are locked, dogs are free")
	}
	shopOwned["animal:elephant"] = true
	if !animalAllowed(animalKindIndex("elephant")) {
		t.Fatal("bought animals appear")
	}
	if !killerAllowed("jason") || !killerAllowed("freddy") || killerAllowed("pinhead") || killerAllowed("tr_k_sombra") {
		t.Fatal("killer gating: the 21 of the catalogue come, the retired ones and the unwon trophy ones do not")
	}
	shopOwned["killer:tr_k_sombra"] = true
	if !killerAllowed("tr_k_sombra") {
		t.Fatal("a trophy killer joins once the trophy is won")
	}
	shopOwned = map[string]bool{"skin:": true, "police:0": true}
}

func TestShopCatalog(t *testing.T) {
	buildShopCatalog()
	seen := map[string]bool{}
	total := 0
	for tab, items := range shopCatalog {
		if tab == shopWardrobe { // the wardrobe is a screen, not a list: nothing is bought there
			if len(items) != 0 {
				t.Fatalf("the wardrobe tab must have no catalog items, got %d", len(items))
			}
			continue
		}
		if len(items) == 0 {
			t.Fatalf("tab %d is empty", tab)
		}
		total += len(items)
		for i, it := range items {
			if seen[it.id] {
				t.Fatalf("duplicate id %s", it.id)
			}
			seen[it.id] = true
			if i > 0 && it.price < items[i-1].price {
				t.Fatalf("tab %d not sorted by price at %s", tab, it.name)
			}
			if it.slot != "" && it.slot != "skin" && it.slot != "kweapon" && wearFx[it.id] == nil {
				t.Fatalf("wearable %s has no effect", it.id)
			}
		}
	}
	for _, e := range []string{"animal:rabbit", "perk:speed1", "kw:knife"} {
		if !seen[e] {
			t.Fatalf("missing %s", e)
		}
	}
	// characters and killers are goals now: only the rewards of trophies (locked) are left in their lists
	for _, tab := range []int{shopChars, shopKillers} {
		for _, it := range shopCatalog[tab] {
			if it.trophy == "" {
				t.Fatalf("%s is for sale but characters and killers are no longer sold", it.id)
			}
		}
	}
	t.Log("shop items:", total)
}
