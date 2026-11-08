package main

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// Gabbo never leaves his place and every DJ of the pool takes the booth sooner or later.
func TestDJCameoRotation(t *testing.T) {
	seen := map[djKind]bool{}
	for tt := 0.0; tt < 40; tt += 0.25 {
		if djAt(djGabboSlot, tt) != djGabbo {
			t.Fatalf("Gabbo left the booth at %.2f", tt)
		}
		in := map[djKind]bool{}
		for s := 0; s < 4; s++ {
			k := djAt(s, tt)
			if in[k] {
				t.Fatalf("%s twice in the booth at %.2f", djNames[k], tt)
			}
			in[k] = true
			seen[k] = true
		}
	}
	if len(seen) != int(djCount) {
		t.Fatalf("only %d of %d DJs ever appear", len(seen), djCount)
	}
}

// balloons: <= 48 chars, translated, inside the party and not twice at once; the names are invented
func TestDJCameoTexts(t *testing.T) {
	old := uiLang
	t.Cleanup(func() { uiLang = old })
	for _, l := range djLines {
		if utf8.RuneCountInString(l.s) > 48 || l.b <= l.a {
			t.Errorf("bad line %q", l.s)
		}
		if langDataEN[l.s] == "" || utf8.RuneCountInString(langDataEN[l.s]) > 48 {
			t.Errorf("no (or long) English for %q", l.s)
		}
	}
	for _, n := range djNames {
		if utf8.RuneCountInString(djNameplate(1)+n) > 48 {
			t.Errorf("name too long %q", n)
		}
		if strings.TrimSpace(n) == "" {
			t.Error("empty name")
		}
	}
	// no text of the cameo (ES or EN) may contain the name of a real DJ
	forbidden := []string{"guetta", "ponte", "ponti", "garrix", "avicii", "harris", "daft punk", "deadmau5", "gabry"}
	all := append([]string{}, djNames[:]...)
	for _, n := range djNames {
		all = append(all, "DJ "+n, langDataEN["DJ "+n])
	}
	for _, l := range djLines {
		all = append(all, l.s, langDataEN[l.s])
	}
	for _, e := range djLinesEN {
		all = append(all, e)
	}
	for _, txt := range all {
		for _, real := range forbidden {
			if strings.Contains(strings.ToLower(txt), real) {
				t.Errorf("%q contains the real name %q", txt, real)
			}
		}
	}
	if djNameplate(15) != "" || djNameplate(2) == "" || djDropU(23.9) >= 0 || djDropU(24.5) <= 0 {
		t.Error("nameplate or drop timing")
	}
}
