package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

// the capybara is registered right after the guinea pig, is free, can be stabbed and has every name in ES and EN
func TestCapibaraRegistered(t *testing.T) {
	if animCapibara != animCobaya+1 || animalKinds[animCapibara] != "capibara" || animalKindIndex("capibara") != animCapibara {
		t.Fatalf("capibara not at %d: %v", animCapibara, animalKinds)
	}
	if !animalAllowed(animCapibara) || !killableAnimal(animCapibara) {
		t.Error("the capybara must be free and killable like the guinea pig")
	}
	for es, en := range map[string]string{animalLabel(animCapibara): "Capybara", animalNameEs(animCapibara): "a capybara", "Cría de capibara": "Baby capybara", "¡FIU, FIU!": "WHEET, WHEET!"} {
		if got := wildEN(es); got != en {
			t.Errorf("%q -> %q, want %q", es, got, en)
		}
	}
	if animalLabel(animCapibara) != "Capibara" || animalNameEs(animCapibara) != "una capibara" {
		t.Error("Spanish names")
	}
	found := false
	for _, n := range ptAnimals {
		found = found || n.key == "capibara" && n.en == "capybara" && n.es == "capibara"
	}
	if !found {
		t.Error("no newspaper noun for the capybara")
	}
}

// opaque bounding box of a frame
func capiBox(b *pixBuf) (x0, y0, x1, y1 int) {
	x0, y0, x1, y1 = b.w, b.h, -1, -1
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			if b.p[(y*b.w+x)*4+3] > 0 {
				x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
			}
		}
	}
	return
}

// the art: every frame and view inside its canvas (a free pixel at each edge for the outline), crisp (alpha 0/255),
// the walk moves, eating and resting are different poses, the coats differ, bigger than a guinea pig, the baby smaller
func TestCapibaraArt(t *testing.T) {
	check := func(name string, b *pixBuf) {
		x0, y0, x1, y1 := capiBox(b)
		if x1 < 0 || wildOpaque(b) < 60 {
			t.Errorf("%s: almost empty", name)
			return
		}
		if x0 < 1 || y0 < 1 || x1 > b.w-2 || y1 > b.h-2 {
			t.Errorf("%s: touches the canvas edge (%d,%d)-(%d,%d) in %dx%d", name, x0, y0, x1, y1, b.w, b.h)
		}
		for i := 3; i < len(b.p); i += 4 {
			if b.p[i] != 0 && b.p[i] != 255 {
				t.Errorf("%s: blurred pixel (alpha %d)", name, b.p[i])
				break
			}
		}
	}
	for coat := 0; coat < 3; coat++ {
		stand := genCapibaraFrame(fStand, coat)
		for _, f := range []int{fStand, fWalk0, fWalk1, fWalk2, fWalk3, fPeck0, fPeck1, fSit} {
			check(fmt.Sprintf("coat %d frame %d", coat, f), genCapibaraFrame(f, coat))
		}
		moved := 0
		for f := fWalk0; f <= fWalk3; f++ {
			if string(genCapibaraFrame(f, coat).p) != string(stand.p) {
				moved++
			}
		}
		if moved < 2 {
			t.Errorf("coat %d: the walk does not move", coat)
		}
		if p, s := string(genCapibaraFrame(fPeck0, coat).p), string(genCapibaraFrame(fSit, coat).p); p == string(stand.p) || s == string(stand.p) || p == s {
			t.Errorf("coat %d: eating / resting are not their own poses", coat)
		}
		for _, front := range []bool{true, false} {
			check(fmt.Sprintf("coat %d view front=%v", coat, front), genCapibaraView(coat, front, 0))
		}
		if string(genCapibaraView(coat, true, 0).p) == string(genCapibaraView(coat, false, 0).p) {
			t.Errorf("coat %d: front and back look the same", coat)
		}
	}
	if string(genCapibaraFrame(fStand, 0).p) == string(genCapibaraFrame(fStand, 1).p) || string(genCapibaraFrame(fStand, 1).p) == string(genCapibaraFrame(fStand, 2).p) {
		t.Error("capybara coats look the same")
	}
	adult := capibaraSprite(genCapibaraFrame(fStand, 0), 0)
	baby := capibaraSprite(genCapibaraFrame(fStand, 0), capiBaby)
	cobaya := genCobayaFrame(fStand, 0).outlinedSel(0.4)
	dog := genDogFrame(fStand, 0).scaled(0.5).outlinedSel(0.4)
	if adult.w <= cobaya.w || adult.h <= cobaya.h || adult.w < dog.w || adult.w > 2*dog.w {
		t.Errorf("size: capybara %dx%d, guinea pig %dx%d, dog %dx%d", adult.w, adult.h, cobaya.w, cobaya.h, dog.w, dog.h)
	}
	if baby.w >= adult.w || baby.h >= adult.h || wildOpaque(baby) < 40 {
		t.Errorf("baby %dx%d vs adult %dx%d", baby.w, baby.h, adult.w, adult.h)
	}
	// every sprite the game asks for exists (4 coats x 8 frames)
	out := AnimalSprites{}
	buildCapibaraSprites(out)
	for coat := 0; coat <= capiBaby; coat++ {
		for _, f := range []int{fStand, fWalk0, fWalk1, fWalk2, fWalk3, fPeck0, fPeck1, fSit} {
			if out[animalKey(animCapibara, coat, f)] == nil {
				t.Errorf("missing sprite coat %d frame %d", coat, f)
			}
		}
	}
	if !animalHasFreeView(animCapibara) {
		t.Error("no front/back view")
	}
	if len(capiWhistlePCM()) < 1000 {
		t.Error("no whistle")
	}
}

func capiNoSpawn(t *testing.T) {
	t.Helper()
	prevDbg := dbgWild
	t.Cleanup(func() { dbgWild = prevDbg; wildReseed(0); capiSpawn = true })
	dbgWild = ""
}

// where and how often: only on levels with water or grass, beside the water (never in it), outdoors, in groups of
// 1-3 (at most one baby), in some parties only; the animal cap keeps holding
func TestCapibaraSpawn(t *testing.T) {
	capiNoSpawn(t)
	wildReseed(9090)
	defs := testLevels(t)
	parties, withCapi, eligible := 0, 0, 0
	levels := map[string]bool{}
	for i, def := range defs {
		for rep := int64(0); rep < 8; rep++ {
			g := capGame(def, int64(i)*37+rep)
			g.spawnAnimals()
			parties++
			_, green := g.wildSpot(rand.New(rand.NewSource(1)), g.greenSpot)
			_, wet := g.wildSpot(rand.New(rand.NewSource(1)), g.nearWater)
			if green || wet {
				eligible++
			}
			if n, b := animalCounts(g); n > animalMax || b > bullMax {
				t.Errorf("%s: %d animals, %d bulls", def.Title, n, b)
			}
			n, babies := 0, 0
			for _, a := range g.animals {
				if a.kind != animCapibara {
					continue
				}
				n++
				if a.coat == capiBaby {
					babies++
				}
				c := a.area.Center()
				if !g.lvl.Outdoor(c) || g.inWater(c) || g.lvl.InPool(c) || g.inWater(a.pos) {
					t.Errorf("%s: a capybara indoors or in the water at %v", def.Title, a.pos)
				}
				if def.Sky == "space" || len(def.Roam) == 0 {
					t.Errorf("%s: capybara on a level without outdoors", def.Title)
				}
			}
			if n > 3 || babies > 1 || (babies == 1 && n < 2) {
				t.Errorf("%s: group of %d with %d babies", def.Title, n, babies)
			}
			if n > 0 {
				withCapi++
				levels[def.Title] = true
			}
		}
	}
	rate := float64(withCapi) / float64(max(eligible, 1))
	t.Logf("%d parties, %d eligible, capybaras in %d (%.0f%%) on %d of %d levels", parties, eligible, withCapi, rate*100, len(levels), len(defs))
	if withCapi == 0 || len(levels) == len(defs) {
		t.Errorf("capybaras should come on some levels, not none or all (%d of %d)", len(levels), len(defs))
	}
	if rate < 0.08 || rate > 0.45 {
		t.Errorf("capybara rate %.2f out of [0.08, 0.45] (chance %.2f)", rate, wildCapibaraChance)
	}
}

// the capybara comes last with the wild dice: every other wild animal of the party (kind, coat, place) is the same
// as without capybaras, and g.rng (the guests, the critters) is never touched by the wild animals
func TestCapibaraKeepsOtherDraws(t *testing.T) {
	capiNoSpawn(t)
	defs := testLevels(t)
	run := func(def *LevelDef, seed int64, on bool) (others []string, capis int, rngNext int64) {
		capiSpawn = on
		wildReseed(seed)
		g := capGame(def, seed)
		g.scatterWild()
		for _, a := range g.animals {
			if a.kind == animCapibara {
				capis++
				continue
			}
			others = append(others, fmt.Sprintf("%d/%d/%.2f,%.2f", a.kind, a.coat, a.pos.X, a.pos.Y))
		}
		return others, capis, g.rng.Int63()
	}
	checked := 0
	for i, def := range defs {
		for rep := int64(1); rep <= 4; rep++ {
			seed := int64(i)*101 + rep
			off, _, rOff := run(def, seed, false)
			on, capis, rOn := run(def, seed, true)
			if rOn != rOff {
				t.Errorf("%s: the capybara moved g.rng", def.Title)
			}
			if capis > 0 {
				checked++
			}
			if fmt.Sprint(on) != fmt.Sprint(off) {
				t.Errorf("%s seed %d: the other wild animals changed: %v vs %v", def.Title, seed, on, off)
			}
		}
	}
	if checked == 0 {
		t.Error("no party with capybaras to compare")
	}
	t.Logf("%d parties with capybaras compared", checked)
}

// -wildnow capibara: a group of three (one of them a baby) next to the start; they live, rest, flee and can die
func TestCapibaraDebugGroupAndLife(t *testing.T) {
	capiNoSpawn(t)
	g := killerGame(t, false)
	prev := g.animals
	t.Cleanup(func() { g.animals = prev })
	dbgWild = "capibara"
	wildReseed(5)
	g.animals = nil
	g.scatterWild()
	if len(g.animals) != 3 || g.animals[2].coat != capiBaby || g.animals[2].label != "Cría de capibara" {
		t.Fatalf("debug group: %d animals", len(g.animals))
	}
	dbgWild = ""
	group := append([]*Animal(nil), g.animals...)
	a := group[0]
	rested, ate := false, false
	for i := 0; i < 60*120; i++ {
		g.updateAnimals(1.0 / 60)
		for _, c := range group {
			f := capibaraFrame(c)
			rested = rested || f == fSit
			ate = ate || f == fPeck0 || f == fPeck1
		}
	}
	if !rested || !ate {
		t.Errorf("in 2 min: rested %v, ate %v", rested, ate)
	}
	for _, c := range g.animals {
		if c.kind == animCapibara && g.inWater(c.pos) {
			t.Error("a capybara walked into the water")
		}
	}
	// startled by the player right next to it
	a.state, a.timer = aIdle, 5
	g.p.pos, g.p.hidden = Vec{a.pos.X + 6, a.pos.Y}, false
	g.updateCapibara(a, 1.0/60)
	if a.state != aFlee || a.icon != "¡FIU, FIU!" {
		t.Errorf("not startled: state %d icon %q", a.state, a.icon)
	}
	if g.progress.Stats == nil {
		g.progress.Stats = map[string]int{}
	}
	before := g.progress.Stats["animals"]
	t.Cleanup(func() { g.progress.Stats["animals"] = before })
	g.killAnimal(a)
	if !a.dead || g.progress.Stats["animals"] != before+1 {
		t.Error("killing a capybara does not count as a fallen animal")
	}
}

// PH_CAPI_SHEET=dir writes the sprite sheet (all coats and frames, x4) for a look
func TestCapibaraSheet(t *testing.T) {
	dir := os.Getenv("PH_CAPI_SHEET")
	if dir == "" {
		t.Skip("set PH_CAPI_SHEET to write the sheet")
	}
	frames := []int{fStand, fWalk0, fWalk1, fWalk2, fWalk3, fPeck0, fPeck1, fSit}
	const cw, ch = 36, 26
	sheet := newPix(cw*(len(frames)+2), ch*4)
	for i := range sheet.p {
		if i%4 == 3 {
			sheet.p[i] = 255
		} else {
			sheet.p[i] = []byte{0x7a, 0xa8, 0x50}[i%4]
		}
	}
	blit := func(b *pixBuf, cx, cy int) {
		for y := 0; y < b.h; y++ {
			for x := 0; x < b.w; x++ {
				i := (y*b.w + x) * 4
				if b.p[i+3] > 0 {
					sheet.set(cx+x, cy+y, rgb(uint32(b.p[i])<<16|uint32(b.p[i+1])<<8|uint32(b.p[i+2])))
				}
			}
		}
	}
	for coat := 0; coat <= capiBaby; coat++ {
		src := coat
		if coat == capiBaby {
			src = 0
		}
		for j, f := range frames {
			blit(capibaraSprite(genCapibaraFrame(f, src), coat), j*cw+2, coat*ch+2)
		}
		blit(capibaraSprite(genCapibaraView(src, true, 0), coat), len(frames)*cw+2, coat*ch+2)
		blit(capibaraSprite(genCapibaraView(src, false, 1), coat), (len(frames)+1)*cw+6, coat*ch+2)
	}
	if err := savePNG(filepath.Join(dir, "capibara_sheet.png"), sheet.scaled(4)); err != nil {
		t.Fatal(err)
	}
}
