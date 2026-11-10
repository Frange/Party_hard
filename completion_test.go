package main

import "testing"

func TestComplPercentPure(t *testing.T) {
	p := func(got, all int) complPart { return complPart{"1 x", "%d x", got, all} }
	cases := []struct {
		parts []complPart
		want  int
	}{
		{nil, 0},
		{[]complPart{p(0, 10), p(0, 10)}, 0},
		{[]complPart{p(5, 10), p(10, 10)}, 75},
		{[]complPart{p(303, 304), p(50, 50)}, 99}, // almost: never rounds up to 100
		{[]complPart{p(304, 304), p(50, 50), p(0, 0)}, 100},
		{[]complPart{p(999, 304)}, 100}, // over the total is clamped
	}
	for i, c := range cases {
		if got := complPercent(c.parts); got != c.want {
			t.Errorf("case %d: %d%%, want %d%%", i, got, c.want)
		}
	}
	m := complMissing([]complPart{p(0, 5), {"1 y", "%d y", 0, 20}, p(3, 3), {"1 z", "%d z", 19, 20}}, 2)
	if len(m) != 2 || m[0] != "20 y" || m[1] != "5 x" {
		t.Errorf("complMissing = %v", m)
	}
	if m := complMissing([]complPart{{"1 y", "%d y", 19, 20}}, 4); len(m) != 1 || m[0] != "1 y" {
		t.Errorf("singular: %v", m)
	}
}

func TestComplRealPartsSane(t *testing.T) {
	g := layerGame(t, false)
	parts := g.complParts()
	if len(parts) != 7 {
		t.Fatalf("%d parts", len(parts))
	}
	for _, p := range parts {
		if p.all <= 0 || p.got < 0 || p.got > p.all {
			t.Errorf("part %q: %d / %d", p.many, p.got, p.all)
		}
	}
	if pc := complPercent(parts); pc < 0 || pc > 100 {
		t.Errorf("percent %d", pc)
	}
}

// the block fits: top right, under the trophy counter (ends y=30), left of the right edge, away from the logo and rows.
func TestComplBlockFits(t *testing.T) {
	g := layerGame(t, false)
	old := uiLang
	t.Cleanup(func() { uiLang = old })
	big := []complPart{{"1 fiesta", "%d fiestas", 0, 999}, {"1 personaje", "%d personajes", 0, 999}, {"1 VIP", "%d VIPs", 0, 999},
		{"1 asesino", "%d asesinos", 0, 999}, {"1 policía", "%d policías", 0, 999}, {"1 trofeo", "%d trofeos", 0, 999}, {"1 de tienda", "%d de tienda", 0, 999}}
	for _, lang := range []string{"es", "en"} {
		uiLang = lang
		for _, parts := range [][]complPart{big, g.complParts(), {{"1 trofeo", "%d trofeos", 998, 999}}} {
			lines := g.complLines(parts)
			if len(lines) < 1 || len(lines) > 4 {
				t.Fatalf("[%s] %d lines", lang, len(lines))
			}
			for _, l := range lines {
				if w := g.textW(l, 1); w > complMaxW {
					t.Errorf("[%s] %q is %.0f px wide", lang, l, w)
				}
			}
			bottom := complY + float64(len(lines))*complPitch
			if complY < trophyHudY+trophyHudH || bottom > 130 {
				t.Errorf("[%s] block y %.0f..%.0f touches the trophy counter or the menu", lang, complY, bottom)
			}
		}
		if l := g.complLines([]complPart{{"1 x", "%d x", 5, 5}}); len(l) != 1 || l[0] != "100 %" {
			t.Errorf("100%%: %v", l)
		}
	}
	if screenW-complRight < 16 || complY-(trophyHudY+trophyHudH) < 12 {
		t.Errorf("margins: right %v, under the counter %v", screenW-complRight, complY-(trophyHudY+trophyHudH))
	}
}
