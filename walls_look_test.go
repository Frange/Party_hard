package main

import (
	"image/color"
	"math"
	"testing"
)

// srgbLum is the WCAG relative luminance of a colour.
func srgbLum(c color.RGBA) float64 {
	ch := func(v uint8) float64 {
		f := float64(v) / 255
		if f <= 0.03928 {
			return f / 12.92
		}
		return math.Pow((f+0.055)/1.055, 2.4)
	}
	return 0.2126*ch(c.R) + 0.7152*ch(c.G) + 0.0722*ch(c.B)
}

// lumContrast is the WCAG contrast ratio between two colours (1..21).
func lumContrast(a, b color.RGBA) float64 {
	la, lb := srgbLum(a), srgbLum(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// overNight lays the night's darkness layer (drawLighting) over a colour with alpha a (0..255).
func overNight(c color.RGBA, def *LevelDef, a float64) color.RGBA {
	if def.Day || a <= 0 {
		return c
	}
	t := def.Tint
	base := mixc(rgb(0x14142a), color.RGBA{t.R, t.G, t.B, 255}, 0.5)
	k := a / 255
	m := func(x, y uint8) uint8 { return uint8(float64(x)*(1-k) + float64(y)*k) }
	return color.RGBA{m(c.R, base.R), m(c.G, base.G), m(c.B, base.B), 255}
}

// floorAvg is the average colour of a floor kind as the background paints it.
func floorAvg(kind string) (color.RGBA, bool) {
	switch kind {
	case "dance":
		return rgb(0x15121f), true
	case "disco70":
		return rgb(0x0e0a14), true
	case "dash", "rug", "rink", "water", "helipad":
		return color.RGBA{}, false
	}
	b := stylizeFloor(kind, genFloor(kind, 0))
	var r, g, bl, n float64
	for i := 0; i < len(b.p); i += 4 {
		if b.p[i+3] == 0 {
			continue
		}
		r, g, bl, n = r+float64(b.p[i]), g+float64(b.p[i+1]), bl+float64(b.p[i+2]), n+1
	}
	if n == 0 {
		return color.RGBA{}, false
	}
	return color.RGBA{uint8(r / n), uint8(g / n), uint8(bl / n), 255}, true
}

func nightAlphas(def *LevelDef) (aOut, aIn float64) {
	if def.Day {
		return 0, 0
	}
	aOut = clampF(float64(def.Tint.A)*1.2, 70, 112)
	return aOut, aOut * 0.3
}

// TestWallLumContrast: in every level with house walls, the wall (its light
// cornice or its dark outline) stands out from every floor beside it and from
// the dark outside, by day and under the night layer, and a doorway's sill
// reads as open floor, not as wall.
func TestWallLumContrast(t *testing.T) {
	checked := 0
	for i, def := range builtinLevels() {
		aOut, aIn := nightAlphas(def)
		dark := overNight(rgb(0x0a0912), def, aOut)
		styles := map[int]bool{}
		for _, w := range def.Walls {
			if wallThick(w.Style) {
				styles[w.Style] = true
			}
		}
		if len(styles) == 0 {
			continue
		}
		checked++
		for st := range styles {
			p := wallPal(st)
			top, edge := overNight(p.top, def, aIn), overNight(p.edge, def, aIn)
			if c := lumContrast(top, edge); c < 4 {
				t.Errorf("level %d %s style %d: cornice vs outline contrast %.2f < 4", i+1, def.Title, st, c)
			}
			if c := lumContrast(top, dark); c < 4 {
				t.Errorf("level %d %s style %d: cornice vs dark outside %.2f < 4", i+1, def.Title, st, c)
			}
			for _, f := range def.Floors {
				fc, ok := floorAvg(f.Kind)
				if !ok {
					continue
				}
				a := aIn
				if !floorIndoor(f.Kind) {
					a = aOut
				}
				fc = overNight(fc, def, a)
				best := math.Max(lumContrast(top, fc), lumContrast(edge, fc))
				if best < 3 {
					t.Errorf("level %d %s style %d: wall vs floor %q contrast %.2f < 3", i+1, def.Title, st, f.Kind, best)
				}
			}
		}
		sill := overNight(doorSill, def, aIn)
		for _, d := range def.Doors {
			if !wallThick(d.Style) {
				continue
			}
			p := wallPal(d.Style)
			if c := lumContrast(sill, overNight(p.edge, def, aIn)); c < 4 {
				t.Errorf("level %d %s: doorway sill vs wall outline %.2f < 4", i+1, def.Title, c)
			}
			if c := lumContrast(sill, overNight(p.top, def, aIn)); c < 1.5 {
				t.Errorf("level %d %s: doorway sill too close to the wall cornice (%.2f < 1.5)", i+1, def.Title, c)
			}
			if c := lumContrast(sill, dark); c < 4 {
				t.Errorf("level %d %s: doorway sill vs dark outside %.2f < 4", i+1, def.Title, c)
			}
		}
	}
	if checked < 5 {
		t.Fatalf("only %d levels with house walls checked", checked)
	}
}

// TestWallDrawCoversFootprint: what is drawn for a wall is exactly what
// blocks people (the thicker house wall footprint is in the solids), house
// walls are wallGrow thicker on each side across their run only, and no wall
// footprint eats into a doorway.
func TestWallDrawCoversFootprint(t *testing.T) {
	for i, def := range builtinLevels() {
		l := buildLevel(def)
		solid := map[Rect]bool{}
		for _, r := range l.solidsP {
			solid[r] = true
		}
		for _, w := range l.Walls {
			f := wallFoot(w)
			if !solid[f] {
				t.Errorf("level %d %s: wall %v footprint %v not in the solids", i+1, def.Title, w.R, f)
			}
			d := sr(f)
			const px = 0.5 // sr rounds each edge to the nearest pixel
			if d.X > f.X+px || d.Y > f.Y+px || d.X+d.W < f.X+f.W-px || d.Y+d.H < f.Y+f.H-px {
				t.Errorf("level %d %s: drawn wall %v does not cover footprint %v", i+1, def.Title, d, f)
			}
			if wallThick(w.Style) {
				if w.R.W >= w.R.H {
					if f.W != w.R.W || f.H != w.R.H+2*wallGrow {
						t.Errorf("level %d %s: horizontal wall %v grew wrong: %v", i+1, def.Title, w.R, f)
					}
				} else if f.H != w.R.H || f.W != w.R.W+2*wallGrow {
					t.Errorf("level %d %s: vertical wall %v grew wrong: %v", i+1, def.Title, w.R, f)
				}
			}
		}
		for _, dr := range def.Doors {
			df := doorFoot(dr)
			for _, w := range l.Walls {
				if w.Style == WallVoid {
					continue
				}
				f := wallFoot(w)
				x0, y0 := math.Max(df.X, f.X), math.Max(df.Y, f.Y)
				x1, y1 := math.Min(df.X+df.W, f.X+f.W), math.Min(df.Y+df.H, f.Y+f.H)
				if x1-x0 > 0.5 && y1-y0 > 0.5 {
					// only a wall lying along the doorway's own line would close it
					along := (dr.Horiz && f.W >= f.H) || (!dr.Horiz && f.H > f.W)
					if along {
						t.Errorf("level %d %s: wall %v closes doorway %v", i+1, def.Title, f, dr.R)
					}
				}
			}
		}
	}
}
