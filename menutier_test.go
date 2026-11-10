package main

import "testing"

func TestMenuTierOf(t *testing.T) {
	cases := []struct {
		fin, tro, leg bool
		want          int
	}{
		{false, false, false, tierNormal},
		{false, true, true, tierNormal}, // trophies without finishing the story: still normal
		{true, false, true, tierBronze}, // everything but a trophy missing: not silver or gold
		{true, false, false, tierBronze},
		{true, false, true, tierBronze},
		{true, true, false, tierSilver},
		{true, true, true, tierGold},
	}
	for _, c := range cases {
		if got := menuTierOf(c.fin, c.tro, c.leg); got != c.want {
			t.Errorf("%+v: tier %d, want %d", c, got, c.want)
		}
	}
}

func TestMenuTierTitles(t *testing.T) {
	g := layerGame(t, false)
	old := uiLang
	t.Cleanup(func() { uiLang = old })
	if s, _ := menuTierTitle(tierNormal); s != "" {
		t.Errorf("normal tier has a title %q", s)
	}
	seen := map[string]bool{}
	for tier := tierBronze; tier <= tierGold; tier++ {
		s, _ := menuTierTitle(tier)
		if s == "" || seen[s] {
			t.Fatalf("tier %d: title %q missing or repeated", tier, s)
		}
		seen[s] = true
		if (s == "★ LEYENDA DE LA FIESTA ★") != (tier == tierGold) {
			t.Errorf("only gold is the legend (tier %d: %q)", tier, s)
		}
		for _, lang := range []string{"es", "en"} {
			uiLang = lang
			if lang == "en" && tr(s) == s {
				t.Errorf("%q has no English text", s)
			}
			// the title starts at x=110 and must stay left of the completion block (x >= 652)
			if w := g.textW(s, 1.3); 110+w > 600 {
				t.Errorf("[%s] %q is %.0f px wide", lang, s, w)
			}
		}
	}
	a, b := menuTierColors(tierNormal)
	for tier := tierBronze; tier <= tierGold; tier++ {
		c, h := menuTierColors(tier)
		if c == a || h == b {
			t.Errorf("tier %d keeps the normal colours", tier)
		}
	}
}

func TestMenuTierDebugOverride(t *testing.T) {
	g := layerGame(t, false)
	for i := 1; i <= 4; i++ {
		g.dbg.tier = i
		if g.menuTier() != i-1 {
			t.Errorf("override %d gives %d", i, g.menuTier())
		}
	}
	g.dbg.tier = 0
}
