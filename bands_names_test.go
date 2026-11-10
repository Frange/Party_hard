package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Owner's policy (2026-11-10): the musicians whose name and role are known for sure show their real name (with the
// band on the card and the band and role in the OBJETIVOS window); the rest keep "<band> · <role>"; never the name
// of a member with a notorious serious criminal conviction (the singer of Lost Prophets).
func TestBandMusiciansRealNames(t *testing.T) {
	g := goalDetailGame(t)
	prev := uiLang
	t.Cleanup(func() { uiLang = prev })
	if _, named := bandRealNames["lp_cantante"]; named {
		t.Error("lp_cantante must not be named")
	}
	if n := displayNameOverrides["lp_cantante"]; n != "Lost Prophets · cantante" {
		t.Errorf("lp_cantante: display name %q", n)
	}
	// no fallback "<band> · <role>" shows a Spanish role in English
	for slug, n := range displayNameOverrides {
		i := strings.Index(n, " · ")
		if i < 0 || bandOfMember(slug) != "" {
			continue
		}
		role := n[i+len(" · "):]
		if _, ok := bandRoleEN[role]; !ok {
			continue
		}
		uiLang = "en"
		got := tr(n)
		uiLang = "es"
		if strings.HasSuffix(got, " · "+role) || got == n {
			t.Errorf("%s: English fallback %q still has the Spanish role", slug, got)
		}
		if !strings.HasPrefix(got, n[:i]+" · ") {
			t.Errorf("%s: English fallback %q lost the band", slug, got)
		}
	}
	members := map[string]bool{}
	for _, b := range bandList {
		for _, m := range b.members {
			members[m.slug] = true
		}
	}
	cards := map[string]albumCard{}
	for _, c := range charCards() {
		cards[c.slug] = c
	}
	for slug, m := range bandRealNames {
		if !members[slug] || bandOfMember(slug) == "" {
			t.Errorf("%s: not a member of a band", slug)
		}
		if n := displayNameOverrides[slug]; n != m.name || strings.Contains(n, " · ") {
			t.Errorf("%s: display name %q, want %q", slug, n, m.name)
		}
		if f := presetByName(slug); f == nil {
			t.Errorf("%s: no preset", slug)
		}
		if langDataEN[m.part] == "" {
			t.Errorf("%s: part %q has no English", slug, m.part)
		}
		c, ok := cards[slug]
		if !ok || c.cardName() != m.name || c.roleCard != bandOfMember(slug) {
			t.Errorf("%s: card %q / %q", slug, c.cardName(), c.roleCard)
		}
		// the exported folder says the same (the game reads the name from it when the assets folder is used)
		raw, err := os.ReadFile(filepath.Join("assets", "sprites", "special_people", slug, "meta.json"))
		if err != nil {
			t.Errorf("%s: %v", slug, err)
			continue
		}
		var meta struct {
			DisplayName string `json:"display_name"`
		}
		if json.Unmarshal(raw, &meta) != nil || meta.DisplayName != m.name {
			t.Errorf("%s: meta.json display_name %q, want %q", slug, meta.DisplayName, m.name)
		}
		for _, lg := range []string{"es", "en"} { // a proper noun: no translation touches it
			uiLang = lg
			if got := tr(m.name); got != m.name {
				t.Errorf("%s %s: tr(%q) = %q", lg, slug, m.name, got)
			}
			// on the card, name and band keep >= 6 px of padding on each side (margins rule)
			for _, s := range []string{m.name, tr(bandOfMember(slug))} {
				if w := g.textW(s, 1); w > albumCardW-12 {
					t.Errorf("%s %s: %q is %.0f px wide on a %.0f px card", lg, slug, s, w, albumCardW)
				}
			}
		}
	}
	// the OBJETIVOS window of a won musician: name, then "<band> · <part>" in the language in force
	g.progress.Album, g.progress.CharKills = map[string]int{}, map[string]int{}
	for _, c := range albumCards() {
		g.progress.Album[c.key] = albumKilled
	}
	g.goTab(tabChars)
	want := map[string][2]string{"met_cantante": {"Metallica · Cantante", "Metallica · Singer"}, "lp_ritmo": {"Lost Prophets · Guitarra rítmica", "Lost Prophets · Rhythm guitar"},
		"lp_corista": {"Lost Prophets · Coros", "Lost Prophets · Backing vocals"}, "deryck": {"Sum 41 · Cantante", "Sum 41 · Singer"}}
	for li, lg := range []string{"es", "en"} {
		uiLang = lg
		for i, c := range g.albumView() {
			w, ok := want[c.slug]
			if !ok {
				continue
			}
			g.albumSel = i
			ct := g.goalDetailContent()
			if ct.name != bandRealNames[c.slug].name || ct.sub != w[li] {
				t.Errorf("%s %s: window %q / %q, want %q", lg, c.slug, ct.name, ct.sub, w[li])
			}
		}
	}
}
