package main

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// ---- EXCUSAS --------------------------------------------------------------------------------------

func TestExcuseLinesAreManyShortAndTranslated(t *testing.T) {
	if len(excuseLines) < 30 {
		t.Fatalf("only %d excuses", len(excuseLines))
	}
	seen := map[string]bool{}
	for _, l := range excuseLines {
		if utf8.RuneCountInString(l.es) > 48 || utf8.RuneCountInString(l.en) > 48 {
			t.Errorf("too long: %q / %q", l.es, l.en)
		}
		if seen[l.es] {
			t.Errorf("repeated %q", l.es)
		}
		seen[l.es] = true
		if langDataEN[l.es] != l.en {
			t.Errorf("no English registered for %q", l.es)
		}
	}
}

func TestPendingWitnessBlurtsAnExcuseWithRests(t *testing.T) {
	t.Cleanup(excuseReset)
	excuseReset()
	g := alertSetup(t)
	g.mode = ModePlay
	a := suspectGuest(g, Vec{230, 300}, DirLeft)
	b := suspectGuest(g, Vec{235, 306}, DirLeft)
	suspectCop(g, Vec{900, 500}, DirLeft)
	g.witness()
	b.state = gToPhone // a second witness on his way to a phone
	b.sawKiller, b.sawP, b.warned = true, g.p, false
	if !g.witnessPending(a, g.p) || !g.witnessPending(b, g.p) {
		t.Fatalf("both guests must be pending witnesses (%v %v)", a.state, b.state)
	}
	known := map[string]bool{}
	for _, l := range excuseLines {
		known[l.es] = true
	}
	a.icon, b.icon = "", ""
	g.time = 100
	var first *Guest
	for i := 0; i < 600 && first == nil; i++ { // 10 s
		g.time += 1.0 / 60
		g.guestExcuse(a, 1.0/60)
		g.guestExcuse(b, 1.0/60)
		switch {
		case a.icon != "":
			first = a
		case b.icon != "":
			first = b
		}
	}
	if first == nil || !known[first.icon] {
		t.Fatalf("no excuse said (icon %q)", map[bool]string{true: "", false: "?"}[first == nil])
	}
	other := a
	if first == a {
		other = b
	}
	other.icon = ""
	for i := 0; i < 300; i++ { // 5 s < the global rest
		g.time += 1.0 / 60
		g.guestExcuse(other, 1.0/60)
	}
	if other.icon != "" {
		t.Fatalf("two excuses inside the global rest: %q", other.icon)
	}
	first.icon = ""
	for i := 0; i < 1200; i++ { // 20 s: the global rest is over but not the guest's own
		g.time += 1.0 / 60
		g.guestExcuse(first, 1.0/60)
	}
	if first.icon != "" {
		t.Fatalf("the same guest repeated before its own rest: %q", first.icon)
	}
}

func TestNoExcuseWithoutSuspicion(t *testing.T) {
	t.Cleanup(excuseReset)
	excuseReset()
	g := alertSetup(t)
	g.mode = ModePlay
	gu := suspectGuest(g, Vec{230, 300}, DirLeft)
	g.time = 50
	for i := 0; i < 1200; i++ {
		g.time += 1.0 / 60
		g.guestExcuse(gu, 1.0/60)
	}
	if gu.icon != "" {
		t.Fatalf("an unsuspicious guest said %q", gu.icon)
	}
}

// ---- FRASES POR NIVEL -------------------------------------------------------------------------------

func TestEveryCampaignLevelHasVenueLines(t *testing.T) {
	for _, e := range campaignLevels {
		ls := venueLines[e.icon]
		if len(ls) < 6 {
			t.Errorf("level %q has %d venue lines (need 6)", e.icon, len(ls))
		}
		for _, l := range ls {
			if utf8.RuneCountInString(l.es) > 48 || utf8.RuneCountInString(l.en) > 48 {
				t.Errorf("%s: too long %q / %q", e.icon, l.es, l.en)
			}
			if langDataEN[l.es] != l.en || l.en == "" {
				t.Errorf("%s: English missing for %q", e.icon, l.es)
			}
		}
	}
}

func TestVenueTalkIsRareAndNeverRepeatsInARow(t *testing.T) {
	g := suspectGame()
	old := g.lvl.Def.Icon
	t.Cleanup(func() { g.lvl.Def.Icon = old; venueLast = "" })
	g.lvl.Def.Icon = "boat"
	hits, last := 0, ""
	for i := 0; i < 20000; i++ {
		if l := g.venueTalk(); l != "" {
			hits++
			if l == last {
				t.Fatalf("%q twice in a row", l)
			}
			last = l
		}
	}
	if r := float64(hits) / 20000; r < 0.08 || r > 0.2 {
		t.Fatalf("venue chance %.3f", r)
	}
	g.lvl.Def.Icon = "nowhere"
	for i := 0; i < 500; i++ {
		if g.venueTalk() != "" {
			t.Fatal("an unknown level must stay silent")
		}
	}
}

// ---- RESUMEN DE RACHA -------------------------------------------------------------------------------

func TestStreakRulesAndPriority(t *testing.T) {
	if len(streakRules) < 25 {
		t.Fatalf("only %d rules", len(streakRules))
	}
	f := paperFacts{seed: 5}
	cases := []struct {
		name string
		p    paperPlayer
		want string
	}{
		{"unseen", paperPlayer{kills: [kcN]int{kcKnife: 6}, clean: 2}, "6 bajas sin que nadie te mirara"},
		{"noknife", paperPlayer{kills: [kcN]int{kcTrap: 3}, seen: 1}, "Ni un solo cuchillo"},
		{"nothing", paperPlayer{}, "Cero bajas"},
		{"chain", paperPlayer{kills: [kcN]int{kcKnife: 9}, chainMax: 9, seen: 4}, "Cadena de 9"},
	}
	old := uiLang
	t.Cleanup(func() { uiLang = old })
	uiLang = "es"
	for _, c := range cases {
		if got := streakFor(f, c.p, 0, 200); !strings.Contains(got, c.want) {
			t.Errorf("%s: %q does not contain %q", c.name, got, c.want)
		}
	}
	a, b := streakFor(f, cases[3].p, 0, 200), streakFor(f, cases[3].p, 0, 200)
	if a != b {
		t.Fatal("the line must be the same on every frame")
	}
}

func TestStreakLinesFitTheStatBoxInBothLanguages(t *testing.T) {
	big := &streakIn{k: 12, knife: 12, poison: 5, traps: 5, blast: 5, chain: 12, clean: 12, seen: 12, cuff: 7, secs: 1000, hits: 12, floor: 60,
		f: paperFacts{animalsDead: 9}, two: true}
	for _, lang := range []string{"es", "en"} {
		inLang(t, lang, func() {
			for i, r := range streakRules {
				format := r.es
				if lang == "en" {
					format = r.en
				}
				s := format
				if r.args != nil {
					s = strings.Replace(strings.Replace(format, "%d", "99", 1), "%d", "99", 1)
				}
				_ = big
				if n := len(wrapPx(s, pcLW-14)); n > 2 {
					t.Errorf("%s rule %d takes %d lines: %q", lang, i, n, s)
				}
			}
		})
	}
}

func TestPaperShowsTheStreak(t *testing.T) {
	for _, two := range []bool{false, true} {
		d := paperSample(paperFacts{seed: 9}, two, false, 3)
		want := map[bool]int{false: 1, true: 2}[two]
		if len(d.stats) != map[bool]int{false: 1, true: 2}[two] {
			t.Fatalf("stats %d", len(d.stats))
		}
		for i, sb := range d.stats {
			if sb.streak == "" {
				t.Fatalf("two=%v: player %d has no streak line", two, i)
			}
		}
		_, lay := renderPaper(d, 0)
		found := 0
		for _, tx := range lay.texts {
			for _, sb := range d.stats {
				if len(sb.streak) > 0 && strings.HasPrefix(sb.streak, tx.s) {
					found++
				}
			}
		}
		if found < want {
			t.Fatalf("two=%v: streak lines were not drawn (%d)", two, found)
		}
	}
}
