package main

import (
	"math"
	"testing"
)

func wardKindGame(t *testing.T) *Game {
	g := goalsGame(t)
	pk, pkl, pr := wardKind, wardKiller, wardRow
	t.Cleanup(func() { wardKind, wardKiller, wardRow = pk, pkl, pr })
	wardKind, wardKiller, wardRow = 0, "", 0
	g.progress.Equip, g.progress.EquipP2 = map[string]string{}, nil
	return g
}

func optIDs(l []wardOption) map[string]bool {
	m := map[string]bool{}
	for _, o := range l {
		m[o.id] = true
	}
	return m
}

// the three pages: PERSONAJE | ASESINOS | POLICÍAS list only what is unlocked or owned
func TestWardrobePagesListOnlyWhatIsUnlocked(t *testing.T) {
	g := wardKindGame(t)
	if len(wardKindNames) != 3 || wardKindNames[0] != "PERSONAJE" || wardKindNames[1] != "ASESINOS" || wardKindNames[2] != "POLICÍAS" {
		t.Fatal(wardKindNames)
	}
	if got := optIDs(g.wardOptions("kwho")); len(got) != 1 || !got[""] {
		t.Fatalf("no killer unlocked yet: %v", got)
	}
	g.progress.Album = map[string]int{"killer:michael": albumKilled, "killer:jason": albumKilled}
	g.progress.Owned = []string{"kw:knife", "kw:saber"}
	g.syncShop()
	k := optIDs(g.wardOptions("kwho"))
	if len(k) != 2 || !k["michael"] || !k["jason"] || k["freddy"] {
		t.Fatalf("killers: %v", k)
	}
	w := optIDs(g.wardOptions("kweapon"))
	if !w[""] || !w["kw:knife"] || !w["kw:saber"] || w["kw:machete"] {
		t.Fatalf("weapons: %v", w)
	}
	if len(g.wardOptions("kdress")) != 1+len(tints) {
		t.Fatal("killer colours: original + every tint")
	}
	c := optIDs(g.wardOptions("rangers"))
	if len(c) != 1 || !c[""] {
		t.Fatalf("no rangers uniform yet: %v", c)
	}
	g.progress.Escapes = 120
	g.syncShop()
	if n := len(g.wardOptions("rangers")); n != 1+6 {
		t.Fatalf("rangers options %d, want original + 6", n)
	}
	if n := len(g.wardOptions("cop")); n != 1+12 {
		t.Fatalf("cop options %d, want original + 12", n)
	}
	// each page has its own rows
	wardKind = wardKindKiller
	if r := g.wardCurRows(); len(r) != 3 || r[0].slot != "kwho" {
		t.Fatal("killer rows")
	}
	wardKind = wardKindCop
	if r := g.wardCurRows(); len(r) != 5 || r[1].slot != "fbi" {
		t.Fatal("police rows")
	}
	wardKind = wardKindPlayer
	if len(g.wardCurRows()) != len(wardRows) {
		t.Fatal("player rows")
	}
	// the page tabs wrap and reset the row
	g.wardGoKind(-1)
	if wardKind != wardKindCop || wardRow != 0 {
		t.Fatal("the pages wrap")
	}
	g.wardGoKind(wardKind + 1)
	if wardKind != wardKindPlayer {
		t.Fatal("the pages wrap forwards")
	}
}

// the killer that comes to the party wears the colour chosen for his type; others are untouched
func TestKillerWardrobeAppliesInGame(t *testing.T) {
	g := wardKindGame(t)
	g.progress.Album = map[string]int{}
	for _, k := range killerCatalog {
		g.progress.Album["killer:"+k.slug] = albumKilled
	}
	g.syncShop()
	wardKind = wardKindKiller
	wardKiller = "michael"
	g.wardCycle(0, "kdress", 1) // first tint
	tint := g.shopEquip("kdress:michael")
	col, ok := tintColor(tint)
	if !ok {
		t.Fatalf("tint %q not saved", tint)
	}
	if g.shopEquip("kdress:jason") != "" {
		t.Fatal("only his type changes")
	}
	base := killerSpecOf("michael")
	dressed := killerSpecOf("michael")
	g.killerWear(&dressed, "michael")
	if dressed.Shirt != hx(col) || dressed.Shirt == base.Shirt {
		t.Fatal("the shirt must take the chosen colour")
	}
	other := killerSpecOf("jason")
	o2 := killerSpecOf("jason")
	g.killerWear(&o2, "jason")
	if other.Shirt != o2.Shirt {
		t.Fatal("jason keeps his clothes")
	}
	// through the same door the party uses
	seenMichael := false
	for i := 0; i < 400 && !seenMichael; i++ {
		cs, _, name := g.pickKiller()
		if name == "michael" {
			seenMichael = true
			if cs.Spec.Shirt != hx(col) {
				t.Fatal("pickKiller must dress michael with his wardrobe colour")
			}
		}
	}
	if !seenMichael {
		t.Skip("michael did not come in 400 draws")
	}
	// back to the original
	g.wardSet(0, "kdress", "")
	if g.shopEquip("kdress:michael") != "" {
		t.Fatal("original removes the tint")
	}
	if g.wardName(0, "kdress") != "Original" {
		t.Fatal(g.wardName(0, "kdress"))
	}
}

// the police page: the uniform chosen for a force is the one its officers wear; the others keep theirs
func TestPoliceWardrobeAppliesInGame(t *testing.T) {
	g := wardKindGame(t)
	g.progress.Escapes = 120
	g.syncShop()
	wardKind = wardKindCop
	g.wardCycle(0, "swat", 1)
	id := g.shopEquip("swat")
	if id == "" || copSetRank(id) == 0 {
		t.Fatalf("swat equip %q", id)
	}
	sp := copSpecOf(unitSWAT)
	wearFx[id](&sp)
	if g.shopEquip("fbi") != "" || g.shopEquip("cop") != "" {
		t.Fatal("only the SWAT changed")
	}
	wardRow = 3
	key, mk := g.wardPreviewSpec()
	pv := mk()
	if key == "" || pv.Shirt != sp.Shirt {
		t.Fatal("the mannequin shows the chosen SWAT uniform")
	}
	// the party: dressTiers builds the SWAT officer from the equipped uniform
	if g.as != nil && g.as.swatCS != nil {
		g.tierBase = nil
		g.dressTiers()
		if g.as.swatCS.Spec.Shirt != sp.Shirt {
			t.Fatal("dressTiers must dress the SWAT with the chosen uniform")
		}
	}
}

// the labels of the pages fit and the escapes line does not touch the tabs
func TestGoalTabsLeaveRoomForEscapes(t *testing.T) {
	g := goalsGame(t)
	rects := g.goalTabRects()
	last := rects[tabCount-1]
	right := last[0] + last[2]
	for _, lang := range []string{"es", "en"} {
		prev := uiLang
		uiLang = lang
		t.Cleanup(func() { uiLang = prev })
		rects = g.goalTabRects()
		last = rects[tabCount-1]
		right = last[0] + last[2]
		g.progress.Escapes = 7
		line := tr(g.copEscapeLine())
		x := 924 - math.Ceil(g.textW(line, 1))
		if x < right+8 {
			t.Fatalf("[%s] the escapes line (x=%.0f) touches the tabs (end %.0f)", lang, x, right)
		}
		for i := 1; i < tabCount; i++ {
			if rects[i][0] < rects[i-1][0]+rects[i-1][2] {
				t.Fatalf("[%s] tabs overlap", lang)
			}
		}
	}
}
