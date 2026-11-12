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
// of a member with a notorious serious criminal conviction (the singer of Lost Profets).
func TestBandMusiciansRealNames(t *testing.T) {
	g := goalDetailGame(t)
	prev := uiLang
	t.Cleanup(func() { uiLang = prev })
	if _, named := bandRealNames["lp_cantante"]; named {
		t.Error("lp_cantante must not be named")
	}
	// names with one letter changed (owner, 2026-11-11), the same in English
	for slug, want := range map[string]string{"met_cantante": "James Hetfiel", "deryck": "Deryck Whibbley", "travis_barker": "Trabis Barker",
		"sp_cantante": "Pierre Buvier", "bt_guitarra": "Ian D'Za", "lp_corista": "Jamie Olivier", "des_cantante": "Dani Markos", "lennon": "John Lenon"} {
		uiLang = "en"
		if n := autoDisplayName(slug); n != want || tr(n) != want {
			t.Errorf("%s: %q / EN %q, want %q", slug, n, tr(n), want)
		}
		uiLang = "es"
	}
	if n := displayNameOverrides["lp_cantante"]; n != "Lost Profets · cantante" {
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
	want := map[string][2]string{"met_cantante": {"Metalica · Cantante", "Metalica · Singer"}, "lp_ritmo": {"Lost Profets · Guitarra rítmica", "Lost Profets · Rhythm guitar"},
		"lp_corista": {"Lost Profets · Coros", "Lost Profets · Backing vocals"}, "deryck": {"Sum 42 · Cantante", "Sum 42 · Singer"}}
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

// Owner's policy (2026-11-11, "Simple Pan"): the band names carry one letter changed too, the same in Spanish and
// English (the originals are checked as hashes by TestNoOriginalIPNamesVisible); the slugs do not change, and the
// players' mp3 files in bands/ are still found by the slug (the real name) or by the new visible name.
func TestBandNamesOneLetterChanged(t *testing.T) {
	prev := uiLang
	t.Cleanup(func() { uiLang = prev })
	want := map[string]string{"beatles": "The Beetles", "sum41": "Sum 42", "blink182": "Blink-183", "simpleplan": "Simple Pan",
		"offspring": "The Offsprung", "metallica": "Metalica", "punksailor": "Punk Sailor", "billytalent": "Billy Tallent",
		"lostprophets": "Lost Profets", "despistaos": "Despistaus", "elvis": "Elviz"}
	if len(bandList) != len(want) {
		t.Fatalf("%d bands, the test knows %d", len(bandList), len(want))
	}
	for _, b := range bandList {
		if b.name != want[b.slug] {
			t.Errorf("%s: %q, want %q", b.slug, b.name, want[b.slug])
		}
		for _, l := range []string{"es", "en"} {
			uiLang = l
			if got := tr(b.name); got != b.name {
				t.Errorf("[%s] %q translated to %q", l, b.name, got)
			}
		}
		if ipOriginalNames[ipHash(ipWords(b.name))] || ipFind(b.name, ipOriginalInText) != "" {
			t.Errorf("%s: %q is an original name", b.slug, b.name)
		}
		// the key of the band is its slug; a file named after the slug or after the visible name finds it
		if k := bandFileKey(b.name); k != b.slug {
			t.Errorf("%s: key %q", b.slug, k)
		}
		for _, f := range []string{b.slug + "_song", "The " + b.slug + " - Song", b.name + " - Song", strings.ToUpper(b.name) + "_x"} {
			if !strings.HasPrefix(bandFileKey(f), b.slug) {
				t.Errorf("%s: file %q -> key %q", b.slug, f, bandFileKey(f))
			}
		}
	}
	// the members' fallback names carry the new band name
	uiLang = "en"
	for slug, n := range map[string]string{"sp_guitarra": "Simple Pan · guitarra", "lp_cantante": "Lost Profets · cantante", "off_bajo": "The Offsprung · bajo"} {
		if got := displayNameOverrides[slug]; got != n {
			t.Errorf("%s: %q, want %q", slug, got, n)
		}
		if got := tr(n); !strings.HasPrefix(got, n[:strings.Index(n, " · ")]+" · ") {
			t.Errorf("%s: EN %q", slug, got)
		}
	}
}
