package main

import "testing"

// the selected row is always inside the scrolling window, from the first row to the last
func TestSettingsSelectedRowVisible(t *testing.T) {
	layout := settingsLayout()
	for sel := 0; sel < setRows; sel++ {
		settingsEnsureVisible(sel)
		found := false
		for _, it := range layout {
			if it.row != sel {
				continue
			}
			found = true
			if it.y < settingsScroll-0.5 || it.y+it.h > settingsScroll+setViewH+0.5 {
				t.Fatalf("row %d not visible: y=%.0f h=%.0f scroll=%.0f", sel, it.y, it.h, settingsScroll)
			}
		}
		if !found {
			t.Fatalf("row %d is missing from the layout", sel)
		}
	}
	// going back to the first row shows the top of the list (with its title)
	settingsEnsureVisible(0)
	if settingsScroll != 0 {
		t.Fatalf("scroll at the first row: %.0f", settingsScroll)
	}
}

// the number of guests is no longer in the settings but it is in DEBUG
func TestGuestRowsOnlyInDebug(t *testing.T) {
	for _, it := range settingsLayout() {
		if it.title == "Invitados mínimo" || it.title == "Invitados máximo" {
			t.Fatal("guest limits in the settings")
		}
	}
	if setRows != int(actCount)+6+actPlayerCount {
		t.Fatalf("settings rows: %d", setRows)
	}
	g := &Game{}
	n := 0
	for _, r := range g.debugRows() {
		if r.label == "Invitados mínimo" || r.label == "Invitados máximo" {
			n++
		}
	}
	if n != 2 {
		t.Fatalf("debug guest rows: %d", n)
	}
}

// sections have a title each and every settings row sits under exactly one
func TestSettingsSections(t *testing.T) {
	seen := map[int]int{}
	titles := 0
	for _, it := range settingsLayout() {
		if it.row < 0 {
			titles++
		} else {
			seen[it.row]++
		}
	}
	if titles != 6 {
		t.Fatalf("titles: %d", titles)
	}
	for r := 0; r < setRows; r++ {
		if seen[r] != 1 {
			t.Fatalf("row %d appears %d times", r, seen[r])
		}
	}
}
