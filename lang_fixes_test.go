package main

import (
	"strings"
	"testing"
)

// TestPoliceUnitNamesTranslate guards the unit names shown in the wardrobe/shop: the
// substring translator once turned the English "Military A" into "Militaryy A".
func TestPoliceUnitNamesTranslate(t *testing.T) {
	old := uiLang
	t.Cleanup(func() { uiLang = old })
	u := unitSpecs()
	all := []CharSpec{copSpec(), u.swat}
	all = append(all, u.copAlt...)
	all = append(all, u.fbi...)
	all = append(all, u.military...)
	all = append(all, u.rangers...)
	for i := range all {
		n := nameFromSpec(&all[i])
		uiLang = "en"
		en := tr(n)
		for _, bad := range []string{"Militaryy", "Militar ", "Policía", "Ejército"} {
			if strings.Contains(en+" ", bad) {
				t.Errorf("%s: EN name %q contains %q", all[i].Name, en, bad)
			}
		}
	}
	uiLang = "en"
	for es, want := range map[string]string{"Militar A": "Military A", "Militar B": "Military B", "Militar": "Military"} {
		if got := tr(es); got != want {
			t.Errorf("tr(%q) = %q in EN, want %q", es, got, want)
		}
	}
}

// TestSkipTwiceHintEnglish: the "skip (press twice)" line of the intro, arrival and
// interludes came out as "skip (dos veces)" in English.
func TestSkipTwiceHintEnglish(t *testing.T) {
	old := uiLang
	t.Cleanup(func() { uiLang = old })
	uiLang = "en"
	if got := tr(skipTwiceHint()); !strings.HasSuffix(got, ": skip (press twice)") {
		t.Errorf("EN skip hint = %q", got)
	}
	uiLang = "es"
	if got := tr(skipTwiceHint()); !strings.HasSuffix(got, ": saltar (dos veces)") {
		t.Errorf("ES skip hint = %q", got)
	}
}
