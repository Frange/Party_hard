package main

import (
	"strings"
	"testing"
)

// PARÍS (it was TORRE EIFFEL): same index, icon, editor key and medal; the tower stays; the city came around it
func TestParisRenamedInPlace(t *testing.T) {
	_ = layerGame(t, false)
	pi := campaignIdx("eiffel")
	if pi != 5 || campaignNumber(pi) != 6 || campaignKey(pi) != "c06_eiffel" {
		t.Fatalf("PARÍS at index %d (row %d, key %q): want index 5, row 6, c06_eiffel", pi, campaignNumber(pi), campaignKey(pi))
	}
	d := levelDefs[pi]
	if d.Title != parisTitle || campaignLevels[pi].title != parisTitle || d.Icon != "eiffel" {
		t.Fatalf("party 6 is %q (%s)", d.Title, d.Icon)
	}
	for _, x := range levelDefs {
		if x.Title == "TORRE EIFFEL" {
			t.Error("a party is still called TORRE EIFFEL")
		}
	}
	// the tables keyed by the title followed it: music, kind of party, costumes, VIP, background effect
	if genreForTitle(parisTitle).id != "clasica" || partyTypeOfTitle[parisTitle] == "" || vipPlan[parisTitle] != "pope" {
		t.Errorf("PARÍS lost its music %q / kind %q / VIP %q", genreForTitle(parisTitle).id, partyTypeOfTitle[parisTitle], vipPlan[parisTitle])
	}
	if fx, _ := ambientFor(parisTitle); fx == 0 {
		t.Error("PARÍS lost its leaves in the wind")
	}
	if langDataEN[parisTitle] != "PARIS" || langDataEN["TORRE EIFFEL"] != "" {
		t.Errorf("English title %q (old key %q)", langDataEN[parisTitle], langDataEN["TORRE EIFFEL"])
	}
	if levelIconArt["eiffel"] == nil {
		t.Error("PARÍS needs its list icon")
	}
	// the VIP zone still fits
	if z, _ := vipPlace(d, ""); z == nil {
		t.Error("the VIP zone does not fit in PARÍS")
	}
}

// the layout: the tower and everything of the city is there, the French flag waves, nothing stands on anything, every
// free spot is reachable, the backdrop is scenery and the shopfronts are a real wall (the graffiti painter's)
func TestParisLandmarksAndWalkable(t *testing.T) {
	e := campaignLevels[campaignIdx("eiffel")]
	for _, seed := range []int64{e.seed, e.seed + 1013, 1, 2, 3} {
		d := genNamed("paris", seed, "")
		if d == nil {
			t.Fatalf("seed %d: no valid layout", seed)
		}
		if bad := genBad(d); len(bad) > 0 {
			t.Errorf("seed %d: unreachable %v", seed, bad)
		}
		l := buildLevel(d)
		n := map[string]int{}
		france := 0
		for _, p := range d.Props {
			n[p.Kind]++
			if (p.Kind == "flagpole" || p.Kind == "flagmini") && p.Var == flagFrance {
				france++
			}
		}
		for k, min := range map[string]int{"eiffel": 1, "carousel": 1, "par_metro": 1, "par_kiosk": 1, "par_bouquiniste": 2,
			"par_bakery": 1, "par_bistro": 2, "par_lamp": 4, "par_busker": 2, "cafetable": 6, "dj": 1} {
			if n[k] < min {
				t.Errorf("seed %d: %s x%d, want at least %d", seed, k, n[k], min)
			}
		}
		if france < 2 {
			t.Errorf("seed %d: %d French flags, want one on a balcony and one on a pole", seed, france)
		}
		buskers := map[int]bool{}
		for _, p := range d.Props {
			if p.Kind == "par_busker" {
				buskers[p.Var] = true
			}
		}
		if !buskers[0] || !buskers[1] {
			t.Errorf("seed %d: the accordion player and the mime must both be there", seed)
		}
		// nothing new stands on anything (the old dance floor and performer spots of the gardens are not ours)
		for _, is := range levelOverlaps(l) {
			for _, k := range []string{"par_", "carousel", "flag"} {
				if strings.Contains(is.String(), k) {
					t.Errorf("seed %d: %v", seed, is)
				}
			}
		}
		cs := d.CellSize
		for y := 98 + cs/2; y < 440; y += cs {
			for x := cs / 2; x < 960; x += cs {
				p := Vec{x, y}
				if l.Free(p) && l.Path(d.Start, p) == nil {
					t.Errorf("seed %d: free spot %v cut off from the start", seed, p)
				}
			}
		}
		for x := 8.0; x < 960; x += 24 {
			for y := 8.0; y < parBackH-4; y += 16 {
				if l.Free(Vec{x, y}) {
					t.Fatalf("seed %d: the painted city is walkable at (%.0f,%.0f)", seed, x, y)
				}
			}
		}
		// the tower's arch can still be walked under
		if !l.Free(Vec{480, eiffelY - 6}) {
			t.Errorf("seed %d: the arch of the tower is blocked", seed)
		}
	}
	// the shopfronts are a real wall with a free strip under it: the graffiti painter still finds his wall
	g := partyGame(t)
	g.loadLevel(campaignIdx("eiffel"))
	if len(gfSegments(g.lvl)) == 0 {
		t.Error("PARÍS has no wall for the graffiti painter any more")
	}
}

// the street artists move (the bellows, the mime's hands) and the backdrop has its landmarks
func TestParisArtistsMoveAndBackdrop(t *testing.T) {
	a, k := propAnims["par_busker"], propKinds["par_busker"]
	if a == nil || k == nil || k.Vars != 2 {
		t.Fatal("the street artists are not animated")
	}
	for v := 0; v < 2; v++ {
		pics := map[string]bool{}
		for f := 0; f < a.n; f++ {
			b := newPix(k.W+2, k.H+2)
			b.ox, b.oy = 1, 1
			a.draw(b, k, v, f)
			pics[string(b.p)] = true
		}
		if len(pics) != a.n {
			t.Errorf("artist %d: %d pictures for %d frames", v, len(pics), a.n)
		}
	}
	b := parisBackdrop()
	if b.w != 960 || b.h != int(parBackH) {
		t.Fatalf("backdrop %dx%d", b.w, b.h)
	}
	// the arch shows in the gap of the avenue and the white domes on the hill
	ax := (parAvenue[0] + parAvenue[1]) / 2
	if c := b.get(ax-20, 20); c != parStoneL {
		t.Errorf("the Arc de Triomphe is not at the end of the avenue (%v)", c)
	}
	if c := b.get(370, 8); c.R < 200 || c.G < 200 {
		t.Errorf("the Sacré-Cœur is not white on its hill (%v)", c)
	}
}

// texts of the new lines: ES and EN, short bubbles
func TestParisTexts(t *testing.T) {
	for _, l := range venueLines["eiffel"] {
		if len([]rune(l.es)) > 48 || len([]rune(l.en)) > 48 {
			t.Errorf("bubble too long: %q / %q", l.es, l.en)
		}
		if langDataEN[l.es] == "" {
			t.Errorf("no English for %q", l.es)
		}
	}
	for _, p := range pVenue["eiffel"] {
		if langDataEN[p.es] == "" {
			t.Errorf("no English for %q", p.es)
		}
	}
	if len(venueLines["eiffel"]) < 10 || len(pVenue["eiffel"]) < 3 {
		t.Error("PARÍS needs its new guest lines and briefs")
	}
}
