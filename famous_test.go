package main

import (
	"encoding/json"
	"math/rand"
	"testing"
)

func famousGuest(slug string, perf string) *Guest {
	spec := presetByName(slug)(rand.New(rand.NewSource(1)))
	return &Guest{cs: newCharSet(&spec), name: nameFromSpec(&spec), perf: perf, keep: true}
}

func countName(g *Game, name string) int {
	n := 0
	for _, gu := range g.guests {
		if gu.cs != nil && gu.cs.Spec != nil && gu.cs.Spec.Name == name {
			n++
		}
	}
	return n
}

// whoever adds them, a famous character is never twice in the same party
func TestFamousOncePerParty(t *testing.T) {
	g := testGame()
	g.rng = rand.New(rand.NewSource(3))
	g.famousOwned = map[string]*Guest{}
	for _, s := range []string{"sonic", "sonic", "sonic", "shrek", "espinete", "espinete", "don_pimpon", "homer", "homer"} {
		if presetByName(s) == nil {
			t.Fatalf("no preset %s", s)
		}
		g.guests = append(g.guests, famousGuest(s, ""))
	}
	g.uniqueFamous()
	for _, s := range []string{"sonic", "shrek", "espinete", "don_pimpon", "homer"} {
		if n := countName(g, s); n > 1 {
			t.Errorf("%s appears %d times", s, n)
		}
	}
	if len(g.guests) != 9 {
		t.Errorf("swapping must keep the head count: %d", len(g.guests))
	}
	// later arrivals (events, powers) are swapped too, and the original stays where it was
	first := g.guests[0]
	g.guests = append(g.guests, famousGuest("sonic", ""), famousGuest("shrek", ""))
	g.uniqueFamous()
	if countName(g, "sonic") != 1 || countName(g, "shrek") != 1 || first.cs.Spec.Name != "sonic" {
		t.Error("a late duplicate was not swapped, or the first copy was")
	}
}

func TestFamousExceptions(t *testing.T) {
	g := testGame()
	g.rng = rand.New(rand.NewSource(4))
	g.famousOwned = map[string]*Guest{}
	// the band member on stage owns the name; a wandering copy goes
	g.guests = []*Guest{famousGuest("sonic", ""), famousGuest("sonic", "guitar")}
	g.uniqueFamous()
	if g.guests[1].cs.Spec.Name != "sonic" || g.guests[0].cs.Spec.Name == "sonic" {
		t.Error("the performer must keep his place")
	}
	// the kids have their own rules, and a debug -with is on purpose
	g.guests = []*Guest{famousGuest("sam_sam", ""), famousGuest("sam_sam", "")}
	g.uniqueFamous()
	if countName(g, "sam_sam") != 2 {
		t.Error("the kids are not part of this rule")
	}
	g.famousOwned = map[string]*Guest{}
	a, b := famousGuest("homer", ""), famousGuest("homer", "")
	b.famousOK = true
	g.guests = []*Guest{a, b}
	g.uniqueFamous()
	if b.cs.Spec.Name != "homer" {
		t.Error("a guest forced by -with must stay")
	}
	// the generic archetypes and plain people are not "famous"
	if g.isFamous("") || g.isFamous(archetypePresets[0].Name) {
		t.Error("generic characters are not one-per-party")
	}
}

// the guests taken from the asset folders (and so the editor's own famous characters) do not repeat either
func TestFamousFromAssetsOnce(t *testing.T) {
	g := testGame()
	g.rng = rand.New(rand.NewSource(9))
	g.store = loadAssetStore()
	if g.store == nil || len(g.store.Chars["special_people"]) < 30 {
		t.Skip("no assets")
	}
	g.rigCache = map[*CharAsset]*CharSet{}
	g.famousOwned, g.famousPick = map[string]*Guest{}, map[string]bool{}
	for i := 0; i < 60; i++ {
		cs, _ := g.newGuestCS()
		g.guests = append(g.guests, &Guest{cs: cs})
	}
	g.uniqueFamous()
	seen := map[string]bool{}
	for _, gu := range g.guests {
		n := gu.cs.Spec.Name
		if g.isFamous(n) {
			if seen[n] {
				t.Fatalf("%s twice", n)
			}
			seen[n] = true
		}
	}
	if len(seen) < 5 {
		t.Errorf("too few famous faces among 60 guests: %d", len(seen))
	}
	// an edited character of the editor is respected: its own name counts once
	var edited *CharAsset
	for _, ca := range g.store.Chars["special_people"] {
		if !ca.Meta.ShopOnly {
			edited = ca
			break
		}
	}
	g.guests = []*Guest{{cs: g.rigFor(edited)}, {cs: g.rigFor(edited)}}
	g.famousOwned = map[string]*Guest{}
	g.uniqueFamous()
	if g.guests[0].cs.Spec.Name != edited.Meta.Name {
		t.Error("the first copy of an editor character must stay")
	}
}

// every party starts with 45 to 60 guests by default, in any level
func TestStartGuestsRange(t *testing.T) {
	g := testGame()
	g.rng = rand.New(rand.NewSource(11))
	lo, hi := 1000, 0
	for i := 0; i < 400; i++ {
		g.pickStartTarget()
		lo, hi = minI(lo, g.startTarget), maxI(hi, g.startTarget)
	}
	// 400 draws over the 16 values 45..60 almost surely cover the whole span (spread 15); 10 is a safe floor
	if lo < 45 || hi > 60 || hi-lo < 10 {
		t.Fatalf("start target range %d..%d, want inside 45..60 and varied", lo, hi)
	}
	for _, def := range testLevels(t) {
		g.lvl = buildLevel(def)
		g.guests = nil
		g.pickStartTarget()
		g.fillGuests(def.Start)
		if len(g.guests) != g.startTarget {
			t.Errorf("%s: %d guests, target %d", def.Title, len(g.guests), g.startTarget)
		}
		g.trimGuests()
		if len(g.guests) > g.startTarget {
			t.Errorf("%s: trim left %d", def.Title, len(g.guests))
		}
	}
}

// the settings keep minimum <= maximum, inside 10..100, and old settings files still work
func TestGuestSettings(t *testing.T) {
	g := testGame()
	if g.maxStart() != 45 || g.guestMax() != 60 {
		t.Fatalf("defaults %d/%d", g.maxStart(), g.guestMax())
	}
	g.progress.GuestMin, g.progress.GuestMax = 80, 50 // inconsistent file: the maximum follows the minimum
	if g.guestMax() < g.maxStart() {
		t.Error("maximum below minimum")
	}
	g.progress.GuestMin, g.progress.GuestMax = 500, 900
	if g.maxStart() > 100 || g.guestMax() > 100 {
		t.Error("values must stay inside 10..100")
	}
	// the cap is strict (cap.go): there is no separate "total" above the maximum any more;
	// the people who arrive later must fit inside guestCap (stress: cap_test.go)
	g.progress.GuestMin, g.progress.GuestMax = 0, 0
	g.twoExtra = 0
	if g.guestCap() != g.guestMax() {
		t.Error("with one player the cap is the maximum itself")
	}
}

// the old settings files stored 10 and 10 under other names: they are ignored, not obeyed
func TestOldGuestSettingsIgnored(t *testing.T) {
	var p Progress
	if err := json.Unmarshal([]byte(`{"unlocked":3,"maxStart":10,"maxTotal":10}`), &p); err != nil {
		t.Fatal(err)
	}
	g := testGame()
	g.progress = p
	if g.maxStart() != 45 || g.guestMax() != 60 {
		t.Errorf("an old file must give the defaults, got %d/%d", g.maxStart(), g.guestMax())
	}
}
