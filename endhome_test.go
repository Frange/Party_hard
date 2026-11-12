package main

import (
	"math"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// The party of the secret ending is a real crowd: many famous guests, all different, from the real list of
// famous characters (no killers, no cops, no kids), spread out without standing on top of each other.
func TestEndingPartyCrowdIsBigAndSpread(t *testing.T) {
	for _, two := range []bool{false, true} {
		g := layerGame(t, two)
		g.startEndingPre(true)
		g.partyBuild(g.end)
		p := &g.end.x.party
		if len(p.cast) < 50 {
			t.Fatalf("2P=%v: only %d guests at the party, want 50 or more", two, len(p.cast))
		}
		seen := map[string]bool{}
		for _, n := range p.names {
			if n == "ranger" {
				continue
			}
			if seen[n] {
				t.Errorf("%s comes twice", n)
			}
			seen[n] = true
			if !famousNames()[n] || isKidSlug(n) {
				t.Errorf("%s is not one of the famous guests", n)
			}
		}
		for _, n := range []string{"sonic", "shrek", "crash", "kratos", "neo", "joker", "gandalf", "chewbacca"} {
			if famousNames()[n] && !seen[n] {
				t.Errorf("%s is missing from the party", n)
			}
		}
		for _, tt := range []float64{2, partyActPhoto + 3, 24} {
			crowd := g.partyCrowd(p, tt, two)
			for i := range crowd {
				a := crowd[i]
				if a.x < 8 || a.x > endCW-8 {
					t.Errorf("t=%.0f: %s stands off the picture at x=%.0f", tt, a.name, a.x)
				}
				for j := i + 1; j < len(crowd); j++ {
					b := crowd[j]
					if math.Abs(a.y-b.y) < 8 && math.Abs(a.x-b.x) < 16 {
						t.Errorf("2P=%v t=%.0f: %s and %s stand on top of each other (%.0f,%.0f / %.0f,%.0f)", two, tt, a.name, b.name, a.x, a.y, b.x, b.y)
					}
				}
			}
		}
	}
}

// Most of the stars added today are in the fixed list of the party and are real famous characters.
func TestEndingPartyCastNamesAreReal(t *testing.T) {
	known := 0
	for _, n := range partyCastNames {
		if famousNames()[n] {
			known++
		}
	}
	if known < 35 {
		t.Errorf("only %d of the %d fixed guests exist as famous characters", known, len(partyCastNames))
	}
	if a, b := partyCastPick(1), partyCastPick(2); len(a) != partyGuests || strings.Join(a, ",") == strings.Join(b, ",") {
		t.Errorf("the crowd should have %d guests and change between viewings (%d)", partyGuests, len(a))
	}
}

// The credits: slower than before, with the testers, the thanks and the year.
func TestEndingCreditsTestersAndYear(t *testing.T) {
	if endCreditSpeed(true) > 40 {
		t.Errorf("the credits roll at %.0f px/s: too fast to read", endCreditSpeed(true))
	}
	all := ""
	for _, r := range endCreditRows(true) {
		all += r[0] + "|" + r[1] + "\n"
	}
	for _, w := range []string{"Probadores|Diego, César y Samuel", "Año|2026", "Personajes especiales|", "Diego|", "César|", "Samuel|"} {
		if !strings.Contains(all, w) {
			t.Errorf("the credits do not have %q", w)
		}
	}
	if strings.Contains(all, "alquilado") {
		t.Error("the rented helicopter joke is repeated in the credits")
	}
	for _, l := range preIslandLines {
		if strings.Contains(l.s, "alquilado") {
			t.Errorf("the island prelude repeats the joke of the arrival: %q", l.s)
		}
	}
}

// QA 2026-11-11: the testers line is English in English (no Spanish "y"); the robot and the funny hero keep one
// proper name in both languages (no film title, no original name).
func TestCreditsAndNamesEnglish(t *testing.T) {
	old := uiLang
	t.Cleanup(func() { uiLang = old })
	uiLang = "en"
	if got := tr("Diego, César y Samuel"); got != "Diego, César and Samuel" {
		t.Errorf("testers in English: %q", got)
	}
	for _, r := range endCreditRows(true) {
		if strings.Contains(tr(r[1]), " y ") {
			t.Errorf("credits row %q in English: %q", r[1], tr(r[1]))
		}
	}
	for _, c := range []struct{ es, en string }{{autoDisplayName("johnny5"), "Cortocirquito"}, {"Mega Manolo", "Mega Manolo"}} {
		if c.es != c.en || tr(c.es) != c.en {
			t.Errorf("%q -> %q, want %q in both languages", c.es, tr(c.es), c.en)
		}
	}
}

// Home at last: the scene comes after the stinger and before FIN, silent, and it draws in both modes.
func TestEndingHomeScene(t *testing.T) {
	if nextEndStage(endPaper, true) != endHome || nextEndStage(endHome, true) != endFinal {
		t.Fatal("the scene at home must go between the stinger and FIN")
	}
	if endMusicVol(&endState{stage: endHome, t: 5, trueEnd: true}) != 0 {
		t.Error("no music at home: only the cricket")
	}
	if endLenFor(endHome, true) < 12 || endLenFor(endHome, true) > 22 {
		t.Errorf("the scene at home lasts %.1fs", endLenFor(endHome, true))
	}
	dst := ebiten.NewImage(screenW, screenH)
	defer dst.Deallocate()
	for _, two := range []bool{false, true} {
		g := layerGame(t, two)
		g.startEndingPre(true)
		g.end.stage = endHome
		for _, tt := range []float64{0.5, 2, 4, 6, 9, 12, 14, 15.5, 17.5} {
			g.end.t, g.end.total = tt, tt
			g.drawEnding(dst)
		}
		if g.bedQuiet || g.bedEmpty || g.bedNoTalk {
			t.Error("the scene at home leaves the bedroom of the menu changed")
		}
	}
	seen := map[string]bool{}
	for tt := 0.0; tt < homeLen; tt += 0.5 {
		c := homeCaption(tt)
		if len([]rune(c)) > 52 {
			t.Errorf("caption too long: %q", c)
		}
		seen[c] = true
	}
	if len(seen) < 5 {
		t.Errorf("only %d captions at home", len(seen))
	}
}
