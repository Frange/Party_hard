package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// WallBoards: the boards of the hockey rink (white, kick plate, ad panels without letters). You cannot walk through
// them but you can see over the glass: they do not block sight (level.go) like the fence.
const WallBoards = 9

// rinkGoalLineInset: how far the goal lines are from the short boards.
const rinkGoalLineInset = 22.0

// drawRink paints the ice of a rink (the "rink" floor): the tiled ice plus the red centre line, the two blue lines,
// goal lines, creases and face-off circles, in design pixels into the baked background.
func (g *Game) drawRink(bg *ebiten.Image, r Rect) {
	drawTiled(bg, g.as.floors["ice"], r)
	red, blue, white := rgb(0xc43c46), rgb(0x3c6cc8), rgb(0xf0f4fa)
	cx, cy := r.X+r.W/2, r.Y+r.H/2
	// creases (light blue half discs) in front of the goal lines
	for _, left := range []bool{true, false} {
		gx := r.X + rinkGoalLineInset
		dir := 1.0
		if !left {
			gx, dir = r.X+r.W-rinkGoalLineInset, -1
		}
		for yy := -24.0; yy <= 24; yy++ {
			w := math.Sqrt(math.Max(0, 24*24-yy*yy))
			x := gx + 1
			if dir < 0 {
				x = gx - w
			}
			fillRect(bg, math.Round(x), math.Round(cy+yy), math.Round(w), 1, rgba(0x6aaee8, 120))
		}
		fillRect(bg, math.Round(gx-1), r.Y+4, 2, r.H-8, red) // goal line
	}
	// blue lines and the red centre line with its dashes
	for _, bx := range []float64{r.X + r.W*0.32, r.X + r.W*0.68} {
		fillRect(bg, math.Round(bx-3), r.Y, 6, r.H, blue)
		fillRect(bg, math.Round(bx-3), r.Y, 1, r.H, shade(blue, 1.3))
	}
	fillRect(bg, math.Round(cx-2), r.Y, 4, r.H, red)
	for y := r.Y + 2; y < r.Y+r.H; y += 8 {
		fillRect(bg, math.Round(cx-1), y, 2, 3, white)
	}
	// centre circle and face-off circles
	ringAt(bg, math.Round(cx), math.Round(cy), 38, 2, blue)
	fillRect(bg, math.Round(cx-3), math.Round(cy-3), 6, 6, blue)
	for _, fx := range []float64{r.X + r.W*0.16, r.X + r.W*0.84} {
		for _, fy := range []float64{r.Y + r.H*0.27, r.Y + r.H*0.73} {
			ringAt(bg, math.Round(fx), math.Round(fy), 24, 1, red)
			fillRect(bg, math.Round(fx-2), math.Round(fy-2), 4, 4, red)
			for _, s := range []float64{-1, 1} {
				fillRect(bg, math.Round(fx+s*26), math.Round(fy-3), 1, 6, red)
			}
		}
	}
	// a thin frosty edge along the boards
	fillRect(bg, r.X, r.Y, r.W, 2, rgba(0x9ec0dc, 140))
	fillRect(bg, r.X, r.Y+r.H-2, r.W, 2, rgba(0x9ec0dc, 140))
	fillRect(bg, r.X, r.Y, 2, r.H, rgba(0x9ec0dc, 140))
	fillRect(bg, r.X+r.W-2, r.Y, 2, r.H, rgba(0x9ec0dc, 140))
}

// drawBoards: the look of WallBoards, called from drawWall.
func drawBoards(dst *ebiten.Image, r Rect) {
	horiz := r.W >= r.H
	ads := []uint32{0x2c5aa8, 0xc03840, 0x2a8a5a, 0xd8a020}
	fillRect(dst, r.X, r.Y, r.W, r.H, rgb(0xe6ecf2))
	if horiz {
		fillRect(dst, r.X, r.Y, r.W, 2, rgb(0xffffff))
		fillRect(dst, r.X, r.Y+r.H-2, r.W, 2, rgb(0xe0b830)) // kick plate
		for x, i := 3.0, 0; x+20 < r.W; x, i = x+26, i+1 {
			fillRect(dst, r.X+x, r.Y+2, 20, math.Max(1, r.H-4), rgb(ads[i%len(ads)]))
		}
		return
	}
	fillRect(dst, r.X, r.Y, 1, r.H, rgb(0xffffff))
	fillRect(dst, r.X+r.W-2, r.Y, 2, r.H, rgb(0xe0b830))
	for y, i := 3.0, 0; y+16 < r.H; y, i = y+22, i+1 {
		fillRect(dst, r.X+1, r.Y+y, math.Max(1, r.W-4), 16, rgb(ads[(i+1)%len(ads)]))
	}
}
