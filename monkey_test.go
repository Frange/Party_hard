package main

import (
	"testing"
	"unicode/utf8"
)

// the Monkey Island cast: presets exist, lines are short and translated, Murray stands in his parties
func TestMonkeyCast(t *testing.T) {
	slugs := []string{"guybrush", "stan_barcos", "elaine_marley", "lechuck", "voodoo_lady", "meathook", "pirata_importante"}
	for _, s := range slugs {
		if presetByName(s) == nil {
			t.Errorf("no preset %s", s)
			continue
		}
		if sp := presetByName(s)(newRand(1)); sp.Name != s {
			t.Errorf("preset %s has name %q", s, sp.Name)
		}
		if len(monkeyCatch[s]) == 0 || quirkOf(s) == "" || len(quirkLines[quirkOf(s)]) == 0 {
			t.Errorf("%s has no lines", s)
		}
		if funnyDesc["skin:"+s] == "" || displayNameOverrides[s] == "" {
			t.Errorf("%s has no description or name", s)
		}
	}
	langDataEN := langDataEN
	chk := func(where, s string) {
		if n := utf8.RuneCountInString(s); n > maxPhrase {
			t.Errorf("%s: %d chars: %q", where, n, s)
		}
		if _, ok := langDataEN[s]; !ok {
			t.Errorf("%s: no English for %q", where, s)
		}
	}
	for k, ls := range monkeyCatch {
		for _, l := range ls {
			chk("catch "+k, l)
		}
	}
	for _, l := range murrayLines {
		chk("murray", l)
	}
	for _, d := range monkeyDuets {
		if presetByName(d.a) == nil || presetByName(d.b) == nil {
			t.Errorf("duet with unknown slug %s/%s", d.a, d.b)
		}
		for _, st := range d.steps {
			chk("duet", st.line)
		}
	}
	for k, v := range monkeyDesc {
		if _, ok := langDataEN[v]; !ok {
			t.Errorf("no English for the description of %s", k)
		}
	}
	for _, v := range monkeyNames {
		if _, ok := langDataEN[v]; !ok {
			t.Errorf("no English for the name %q", v)
		}
	}
	found := 0
	for _, d := range builtinLevels()[:campaignN] {
		for _, p := range d.Props {
			if p.Kind == "murray" {
				found++
			}
		}
	}
	if found != len(murrayLevels) {
		t.Errorf("Murray is in %d parties, want %d", found, len(murrayLevels))
	}
}
