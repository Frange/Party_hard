package main

import "testing"

// the letter is worn like the newspaper: crossing creases, stains, blood, rips and a missing corner, but never over
// its text (the rips stop before every line, and the text is drawn on top afterwards)
func TestInviteLetterWear(t *testing.T) {
	g := textGame()
	lines, _ := g.inviteLayout(g.inviteScale())
	boxes := g.inviteTextBoxes(lines)
	if len(boxes) != len(lines) || len(boxes) < 8 {
		t.Fatalf("text boxes: %d for %d lines", len(boxes), len(lines))
	}
	im := letterWearImage(boxes, 7)
	W, H := int(invPW), int(invPH)
	if im.Rect.Dx() != W || im.Rect.Dy() != H {
		t.Fatalf("size %v", im.Rect)
	}
	holes := 0
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			if im.RGBAAt(x, y).A == 0 {
				holes++
			}
		}
	}
	if holes < 1500 {
		t.Fatalf("too few torn pixels: %d (big rips wanted)", holes)
	}
	if im.RGBAAt(W-2, 1).A != 0 {
		t.Error("the top right corner should be torn off")
	}
	for _, b := range boxes { // no hole in or right round a line of text
		for y := b.Min.Y - 2; y < b.Max.Y+2; y++ {
			for x := b.Min.X - 2; x < b.Max.X+2; x++ {
				if x >= 0 && y >= 0 && x < W && y < H && im.RGBAAt(x, y).A == 0 {
					t.Fatalf("a rip at %d,%d cuts into the text box %v", x, y, b)
				}
			}
		}
	}
	// the creases cross the paper: the crease column / rows are darker than the paper beside them
	sum := func(x0, y0, dx, dy, n int) int {
		s := 0
		for k := 0; k < n; k++ {
			c := im.RGBAAt(x0+dx*k, y0+dy*k)
			s += int(c.R) + int(c.G) + int(c.B)
		}
		return s
	}
	vx := W/2 + 6
	if sum(vx, 40, 0, 1, 300) >= sum(vx-30, 40, 0, 1, 300) {
		t.Error("no vertical crease")
	}
	for _, hy := range []int{H / 3, 2 * H / 3} {
		if sum(60, hy, 1, 0, 500) >= sum(60, hy-20, 1, 0, 500) {
			t.Errorf("no horizontal crease at %d", hy)
		}
	}
	// some red: the drops of blood
	red := 0
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			c := im.RGBAAt(x, y)
			if c.A == 255 && c.R > 90 && c.G < 50 && c.B < 50 {
				red++
			}
		}
	}
	if red < 20 {
		t.Errorf("no blood on the letter (%d red pixels)", red)
	}
}
