package main

import "testing"

// the formatted hint line (with something added at the end) must still translate
func TestTrTemplate(t *testing.T) {
	snapshotLang(t)
	uiLang = "en"
	s := "W/A/S/D mover · Espacio apuñalar · E interactuar · F bailar · Shift izq. correr · V suceso"
	if got := tr(s); got == s {
		t.Fatal("not translated: ", got)
	}
	uiLang = "es"
	if got := tr("KILLS:05/10"); got != "BAJAS:05/10" {
		t.Fatal(got)
	}
	// words that are the same in both languages stay alone
	if got := tr("Hola"); got == "" {
		t.Fatal("empty")
	}
}
