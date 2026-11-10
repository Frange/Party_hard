package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// House walls (plaster, brick, club, wood, roof parapet) used to be drawn as a
// near-black strip that read like the dark ground outside or like a doorway.
// Now they are a touch thicker (wallGrow per side, the same in the drawing and
// in the collision), have a dark outline and a light cornice on top, and the
// doorways get a stone threshold, so a wall and a way through never look alike.

// wallGrow is how much a house wall grows on each side across its thickness
// (never along its length, so doorways keep their width).
const wallGrow = 1.0

// wallThick tells which wall styles are house walls that get the thicker,
// outlined look. Fences, hedges, rails, rink boards and the void are left alone.
func wallThick(style int) bool {
	switch style {
	case WallPlaster, WallBrick, WallClub, WallWood, WallRoof:
		return true
	}
	return false
}

// wallFoot is the real footprint of a wall: what blocks people, sight and the
// navigation grid, and exactly what drawWall paints.
func wallFoot(w Wall) Rect {
	if !wallThick(w.Style) {
		return w.R
	}
	if w.R.W >= w.R.H {
		return Rect{w.R.X, w.R.Y - wallGrow, w.R.W, w.R.H + 2*wallGrow}
	}
	return Rect{w.R.X - wallGrow, w.R.Y, w.R.W + 2*wallGrow, w.R.H}
}

// doorFoot is a doorway's rectangle grown like its wall, so the threshold
// lines up with the wall ends on both sides.
func doorFoot(d Door) Rect {
	if !wallThick(d.Style) {
		return d.R
	}
	if d.Horiz {
		return Rect{d.R.X, d.R.Y - wallGrow, d.R.W, d.R.H + 2*wallGrow}
	}
	return Rect{d.R.X - wallGrow, d.R.Y, d.R.W + 2*wallGrow, d.R.H}
}

// wallPalette: edge is the dark outline, top the light cornice (the top of
// the wall seen from above), face the front of a horizontal wall.
type wallPalette struct{ edge, top, face color.RGBA }

func wallPal(style int) wallPalette {
	switch style {
	case WallBrick:
		return wallPalette{rgb(0x140a0a), rgb(0xe0b496), rgb(0x7a4436)}
	case WallClub:
		return wallPalette{rgb(0x07050c), rgb(0xbca0f0), rgb(0x3a2a5a)}
	case WallWood:
		return wallPalette{rgb(0x180c04), rgb(0xecbc7a), rgb(0x8a5e30)}
	case WallRoof:
		return wallPalette{rgb(0x0a0a10), rgb(0xc8ccdc), rgb(0x3a3c4c)}
	}
	return wallPalette{rgb(0x120e10), rgb(0xe4d2b4), rgb(0x6a5646)} // plaster
}

// doorSill is the colour of the threshold laid across a house doorway.
var doorSill = rgb(0x8c8272)

// drawThickWall paints a house wall over its whole footprint r (wallFoot,
// in pixels): a dark outline all round, a light cornice and, on horizontal
// runs, the wall's face with the style's texture below it.
func drawThickWall(dst *ebiten.Image, r Rect, style int) {
	p := wallPal(style)
	fillRect(dst, r.X, r.Y, r.W, r.H, p.edge)
	in := Rect{r.X + 1, r.Y + 1, r.W - 2, r.H - 2}
	if in.W <= 0 || in.H <= 0 {
		return
	}
	if r.W >= r.H {
		capH := math.Min(4, math.Max(2, math.Round(in.H*0.4)))
		fillRect(dst, in.X, in.Y, in.W, capH, p.top)
		fillRect(dst, in.X, in.Y, in.W, 1, shade(p.top, 1.18))
		fy, fh := in.Y+capH, in.H-capH
		if fh <= 0 {
			return
		}
		fillRect(dst, in.X, fy, in.W, 1, shade(p.top, 0.62)) // the cornice's lip
		fy, fh = fy+1, fh-1
		if fh <= 0 {
			return
		}
		fillRect(dst, in.X, fy, in.W, fh, p.face)
		switch style {
		case WallBrick:
			for y := 0.0; y < fh; y += 3 {
				fillRect(dst, in.X, fy+y, in.W, 1, shade(p.face, 0.78))
				off := 0.0
				if int(y/3)%2 == 1 {
					off = 3
				}
				for x := off; x < in.W; x += 6 {
					fillRect(dst, in.X+x, fy+y, 1, math.Min(3, fh-y), shade(p.face, 0.78))
				}
			}
			if r.H > 26 {
				for x := 7.0; x+9 < in.W; x += 26 {
					fillRect(dst, in.X+x, fy+6, 9, 10, rgb(0x2a3a58))
					fillRect(dst, in.X+x, fy+6, 9, 2, rgb(0x6a86b0))
				}
			}
		case WallClub:
			fillRect(dst, in.X, in.Y+in.H-2, in.W, 1, rgb(0xff2090))
			fillRect(dst, in.X, in.Y+in.H-1, in.W, 1, rgb(0x20c0ff))
		case WallRoof:
			fillRect(dst, in.X, in.Y+in.H-1, in.W, 1, shade(p.face, 0.7))
		case WallWood:
			for x := 0.0; x < in.W; x += 4 {
				fillRect(dst, in.X+x, fy, 1, fh, shade(p.face, 0.72))
			}
			fillRect(dst, in.X, in.Y+in.H-1, in.W, 1, shade(p.face, 0.6))
		default:
			for x := 0.0; x < in.W; x += 4 {
				fillRect(dst, in.X+x, fy, 1, fh, shade(p.face, 0.85))
			}
			fillRect(dst, in.X, in.Y+in.H-1, in.W, 1, shade(p.face, 0.7))
		}
		if (style == WallPlaster || style == WallWood) && fh >= 8 {
			drawPictures(dst, Rect{in.X, in.Y, in.W, in.H}, fy-in.Y, style)
		}
		return
	}
	// a vertical run is seen from above: all cornice, lit on its left edge
	fillRect(dst, in.X, in.Y, in.W, in.H, p.top)
	fillRect(dst, in.X, in.Y, 1, in.H, shade(p.top, 1.18))
	if in.W >= 3 {
		fillRect(dst, in.X+in.W-1, in.Y, 1, in.H, shade(p.top, 0.72))
	}
	switch style {
	case WallBrick:
		for y := 3.0; y < in.H; y += 6 {
			fillRect(dst, in.X+1, in.Y+y, in.W-2, 1, shade(p.top, 0.82))
		}
		if r.H > 60 && in.W >= 4 {
			for y := 10.0; y+12 < in.H; y += 34 {
				fillRect(dst, in.X+1, in.Y+y, in.W-2, 12, rgb(0x2a3a58))
				fillRect(dst, in.X+1, in.Y+y, in.W-2, 2, rgb(0x6a86b0))
			}
		}
	case WallWood:
		for y := 4.0; y < in.H; y += 5 {
			fillRect(dst, in.X+1, in.Y+y, in.W-2, 1, shade(p.top, 0.8))
		}
	case WallClub:
		fillRect(dst, in.X+math.Floor(in.W/2), in.Y, 1, in.H, rgb(0xff2090))
	}
}

// drawThreshold lays a stone sill (lighter than the dark outside, darker than
// the cornice) across a house doorway (its jambs are the
// wall ends on either side), so a way through always reads as open floor.
func drawThreshold(bg *ebiten.Image, d Door, ws []Wall) {
	r := sr(doorFoot(d))
	// a jamb is drawn only where a wall really ends next to the doorway
	wallAt := func(x, y float64) bool {
		for _, w := range ws {
			if w.Style != WallVoid && sr(wallFoot(w)).Has(x, y) {
				return true
			}
		}
		return false
	}
	fillRect(bg, r.X, r.Y, r.W, r.H, doorSill)
	if d.Horiz {
		// grooves along the doorway (never the wall face's vertical boards)
		for y := 3.0; y < r.H-2; y += 3 {
			fillRect(bg, r.X+1, r.Y+y, r.W-2, 1, shade(doorSill, 0.88))
		}
		fillRect(bg, r.X, r.Y, r.W, 1, shade(doorSill, 1.15))
		fillRect(bg, r.X, r.Y+r.H-1, r.W, 1, shade(doorSill, 0.8))
		// door jambs: the wall ends either side get a wooden post
		if wallAt(r.X-1, r.Y+r.H/2) {
			fillRect(bg, r.X-2, r.Y, 2, r.H, rgb(0x4a3018))
			fillRect(bg, r.X-2, r.Y, 1, r.H, rgb(0x8a6038))
		}
		if wallAt(r.X+r.W+1, r.Y+r.H/2) {
			fillRect(bg, r.X+r.W, r.Y, 2, r.H, rgb(0x4a3018))
			fillRect(bg, r.X+r.W, r.Y, 1, r.H, rgb(0x8a6038))
		}
		return
	}
	for y := 6.0; y < r.H-2; y += 7 {
		fillRect(bg, r.X+1, r.Y+y, r.W-2, 1, shade(doorSill, 0.85))
	}
	fillRect(bg, r.X, r.Y, 1, r.H, shade(doorSill, 1.15))
	fillRect(bg, r.X+r.W-1, r.Y, 1, r.H, shade(doorSill, 0.8))
	if wallAt(r.X+r.W/2, r.Y-1) {
		fillRect(bg, r.X, r.Y-2, r.W, 2, rgb(0x4a3018))
		fillRect(bg, r.X, r.Y-2, r.W, 1, rgb(0x8a6038))
	}
	if wallAt(r.X+r.W/2, r.Y+r.H+1) {
		fillRect(bg, r.X, r.Y+r.H, r.W, 2, rgb(0x4a3018))
		fillRect(bg, r.X, r.Y+r.H, r.W, 1, rgb(0x8a6038))
	}
}
