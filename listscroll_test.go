package main

import "testing"

// The hover scroll: only with the mouse in use, only on the strips, only when there is more, slowly.
func TestEdgeScroll(t *testing.T) {
	r := [4]float64{100, 100, 300, 300} // list 100..400
	run := func(e *edgeScroll, x, y int, keys, up, down bool, frames int) int {
		n := 0
		for i := 0; i < frames; i++ {
			n += e.tick(x, y, keys, r, up, down, 1.0/60, 4)
		}
		return n
	}
	e := &edgeScroll{}
	if run(e, 200, 395, false, true, true, 120) != 0 {
		t.Fatal("the mouse never moved: no scroll")
	}
	e.tick(201, 395, false, r, true, true, 1.0/60, 4) // it moves
	n := run(e, 201, 395, false, true, true, 60)
	if n < 2 || n > 4 {
		t.Fatalf("1 s at the bottom edge: %d rows, want 2..4 (slow)", n)
	}
	if run(e, 201, 395, false, true, false, 60) != 0 {
		t.Fatal("nothing more below: no scroll")
	}
	if run(e, 201, 250, false, true, true, 120) != 0 {
		t.Fatal("middle of the list: no scroll")
	}
	e.tick(202, 105, false, r, true, true, 1.0/60, 4)
	if n := run(e, 202, 105, false, true, true, 60); n > -2 {
		t.Fatalf("top strip must scroll up, got %d", n)
	}
	if run(e, 202, 50, false, true, true, 60) != 0 {
		t.Fatal("outside the list: no scroll")
	}
	// a key (or pad button) hands control back to the keyboard until the mouse moves
	e.tick(203, 395, true, r, true, true, 1.0/60, 4)
	if run(e, 203, 395, false, true, true, 120) != 0 {
		t.Fatal("after a key the resting mouse must not scroll")
	}
	e.tick(204, 395, false, r, true, true, 1.0/60, 4)
	if run(e, 204, 395, false, true, true, 60) <= 0 {
		t.Fatal("moving the mouse again re-enables it")
	}
	// near the inner border of the strip it is gentler than at the very edge
	a, b := &edgeScroll{seen: true, mouse: true, lastX: 200, lastY: 341}, &edgeScroll{seen: true, mouse: true, lastX: 200, lastY: 399}
	if run(a, 200, 341, false, true, true, 300) >= run(b, 200, 399, false, true, true, 300) {
		t.Error("the speed must grow towards the edge")
	}
}

// edgeApply moves the window and keeps the selection inside it (same column), within the list.
func TestEdgeApply(t *testing.T) {
	// grid of 5 columns, 3 rows visible, 23 items (5 rows)
	sel, sc := edgeApply(1, 2, 0, 3, 5, 23)
	if sc != 1 || sel/5 < 1 || sel%5 != 2 {
		t.Fatalf("down: sel %d scroll %d", sel, sc)
	}
	sel, sc = edgeApply(5, 2, 1, 3, 5, 23)
	if sc != 2 || sel != 12 {
		t.Fatalf("clamped at the end: sel %d scroll %d", sel, sc)
	}
	sel, sc = edgeApply(-1, 22, 2, 3, 5, 23)
	if sc != 1 || sel != 17 {
		t.Fatalf("up: sel %d scroll %d", sel, sc)
	}
	if s, c := edgeApply(0, 7, 1, 3, 5, 23); s != 7 || c != 1 {
		t.Fatal("no move")
	}
	// a plain list keeps the selection on the edge row so the usual clamp leaves the scroll alone
	sel, sc = edgeApply(1, 8, 0, 9, 1, 40)
	if sc != 1 || sel != 8 || sel < sc || sel >= sc+9 {
		t.Fatalf("list: sel %d scroll %d", sel, sc)
	}
}
