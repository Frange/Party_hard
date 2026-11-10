package main

import (
	"math"
	"os"
	"regexp"
	"strings"
	"testing"
)

// solidBox: the bounding box of the fully opaque pixels (shadows and foam are translucent).
func solidBox(b *pixBuf) (x0, y0, x1, y1 int) {
	x0, y0, x1, y1 = b.w, b.h, -1, -1
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			if b.p[(y*b.w+x)*4+3] == 255 {
				x0, y0 = min(x0, x), min(y0, y)
				x1, y1 = max(x1, x), max(y1, y)
			}
		}
	}
	return
}

// countPx: how many pixels of b satisfy f.
func countPx(b *pixBuf, f func(r, g, bl, a uint8) bool) int {
	n := 0
	for i := 0; i < len(b.p); i += 4 {
		if f(b.p[i], b.p[i+1], b.p[i+2], b.p[i+3]) {
			n++
		}
	}
	return n
}

func isInk(r, g, b, a uint8) bool { return a == 255 && r == vInk.R && g == vInk.G && b == vInk.B }

// inkEdge: the share of the opaque pixels on the silhouette's edge that are the dark outline.
func inkEdge(b *pixBuf) float64 {
	edge, ink := 0, 0
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			i := (y*b.w + x) * 4
			if b.p[i+3] != 255 {
				continue
			}
			out := false
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := x+d[0], y+d[1]
				if nx < 0 || ny < 0 || nx >= b.w || ny >= b.h || b.p[(ny*b.w+nx)*4+3] == 0 {
					out = true
				}
			}
			if out {
				edge++
				if isInk(b.p[i], b.p[i+1], b.p[i+2], b.p[i+3]) {
					ink++
				}
			}
		}
	}
	if edge == 0 {
		return 0
	}
	return float64(ink) / float64(edge)
}

// every livery the boat and space levels can send (police units, ambulance, hearse, limousine, fire,
// technicians and the kids' colours)
func transitLiveries() []*Vehicle {
	var vs []*Vehicle
	for u := unitPolice; u <= unitRangers; u++ {
		vs = append(vs, &Vehicle{kind: 0, unit: u})
	}
	for _, k := range []int{1, 4, 6, 7, 8} {
		vs = append(vs, &Vehicle{kind: k})
	}
	for a := 0; a < 3; a++ {
		vs = append(vs, &Vehicle{kind: 9, alt: a})
	}
	return vs
}

// the new boats and ships keep the sizes of the old sprites: the waterline, the lanes and the
// parking slots do not move
func TestVesselSizesKeepAnchors(t *testing.T) {
	body, trim := hx(0x2a50c0), hx(0xf4f4f8)
	for style := 0; style < crCount; style++ {
		for _, long := range []bool{false, true} {
			bw, sw, vw, svw := 90, 96, 48, 70
			if long {
				bw, sw, vw, svw = 112, 116, 58, 86
			}
			check := func(name string, b *pixBuf, w, h int) {
				t.Helper()
				if b.w != w || b.h != h {
					t.Errorf("style %d long=%v %s: %dx%d, want %dx%d", style, long, name, b.w, b.h, w, h)
				}
			}
			bt := genBoat(body, trim, long, style, 0)
			check("boat", bt, bw, 42)
			check("ship", genShip(body, trim, long, style), sw, 42)
			check("boat view", genBoatView(body, trim, long, true, style), vw, 44)
			check("ship view", genShipView(body, trim, long, false, style), svw, 46)
			// the keel sits on the waterline (row 34-36), as the old launch did
			if _, _, _, y1 := solidBox(bt); y1 < 34 || y1 > 36 {
				t.Errorf("style %d long=%v: the keel ends on row %d, want 34-36", style, long, y1)
			}
		}
	}
}

// proportions: long and low boats and ships, a tall rocket, a ship with sails taller than its hull
func TestVesselProportions(t *testing.T) {
	body, trim := hx(0xf4f4f8), hx(0xd02a2a)
	ratio := func(b *pixBuf) float64 {
		x0, y0, x1, y1 := solidBox(b)
		return float64(x1-x0+1) / float64(y1-y0+1)
	}
	for style := 0; style < crCount; style++ {
		if r := ratio(genBoat(body, trim, false, style, 0)); r < 2.2 {
			t.Errorf("boat style %d: too stubby (w/h %.2f)", style, r)
		}
		if r := ratio(genShip(body, trim, false, style)); r < 2.2 {
			t.Errorf("ship style %d: too stubby (w/h %.2f)", style, r)
		}
	}
	if r := ratio(genRocket()); r > 0.85 {
		t.Errorf("the rocket is not upright (w/h %.2f)", r)
	}
	if r := ratio(genPirateShip(0)); r < 0.9 || r > 1.6 {
		t.Errorf("the pirate ship has odd proportions (w/h %.2f)", r)
	}
	fb := newPix(124, 66)
	genFerry(fb)
	if r := ratio(fb); r < 1.5 {
		t.Errorf("the ferry is too tall (w/h %.2f)", r)
	}
}

// the style: a dark outline all round, the hull clearly lighter than the outline, windows / canopy
func TestVesselOutlineAndContrast(t *testing.T) {
	for _, v := range transitLiveries() {
		body, trim, long, _, _ := transitStyle(v)
		style := transitCraft(v)
		for _, sea := range []bool{true, false} {
			var b *pixBuf
			if sea {
				b = genBoat(body, trim, long, style, 1)
			} else {
				b = genShip(body, trim, long, style)
			}
			if e := inkEdge(b); e < 0.97 {
				t.Errorf("kind %d unit %d sea=%v: only %.0f%% of the edge is outlined", v.kind, v.unit, sea, e*100)
			}
			// the bulk of the vessel (its non-ink pixels) stands out from the outline
			sum, n := 0.0, 0
			for i := 0; i < len(b.p); i += 4 {
				if b.p[i+3] == 255 && !isInk(b.p[i], b.p[i+1], b.p[i+2], 255) {
					sum += 0.3*float64(b.p[i]) + 0.59*float64(b.p[i+1]) + 0.11*float64(b.p[i+2])
					n++
				}
			}
			if n == 0 || sum/float64(n)-vLum(vInk) < 30 {
				t.Errorf("kind %d unit %d sea=%v: the vessel blends into its outline", v.kind, v.unit, sea)
			}
			glass := countPx(b, func(r, g, bl, a uint8) bool { return a == 255 && bl > 150 && bl > r+40 })
			if glass < 10 {
				t.Errorf("kind %d unit %d sea=%v: no windows / canopy (%d px)", v.kind, v.unit, sea, glass)
			}
		}
	}
	for name, b := range map[string]*pixBuf{"rocket": genRocket(), "pirate": genPirateShip(1)} {
		if e := inkEdge(b); e < 0.97 {
			t.Errorf("%s: only %.0f%% of the edge is outlined", name, e*100)
		}
	}
}

// every style looks different, and the military launch carries its gunner and machine gun
func TestVesselStylesDistinct(t *testing.T) {
	body, trim := hx(0x2a50c0), hx(0xf4f4f8)
	for a := 0; a < crCount; a++ {
		for b := a + 1; b < crCount; b++ {
			if a == crPlain && b == crLimo || a == crPlain && b == crRangers {
				continue // the plain one has no extras to tell it apart in the short ship
			}
			if same(genBoat(body, trim, false, a, 0), genBoat(body, trim, false, b, 0)) && !(openCraft(a) && openCraft(b)) {
				t.Errorf("boat styles %d and %d are the same picture", a, b)
			}
		}
	}
	mil := genBoat(hx(0x4a5a30), hx(0x3cc846), false, crMilitary, 0)
	if mil.get(50, 10) != c(0x5a6448) {
		t.Error("the military launch has no gunner in his helmet")
	}
	barrel := 0
	for x := 66; x < 78; x++ {
		if mil.get(x, 15) == c(0x2a2a30) {
			barrel++
		}
	}
	if barrel < 10 {
		t.Errorf("the military launch has no machine-gun barrel (%d px)", barrel)
	}
	if same(genBoat(body, trim, false, crPolice, 0), genBoat(body, trim, false, crPolice, 1)) {
		t.Error("the pennant does not wave")
	}
	if same(genPirateShip(0), genPirateShip(1)) {
		t.Error("the pirate pennants do not wave")
	}
}

// the ferry of Isla Nublar: white superstructure, red and orange bands, blue windows
func TestFerrySprite(t *testing.T) {
	b := newPix(124, 66)
	genFerry(b)
	white := countPx(b, func(r, g, bl, a uint8) bool { return a == 255 && r > 220 && g > 220 && bl > 220 })
	if white < 600 {
		t.Errorf("the ferry is not white enough (%d px)", white)
	}
	red := countPx(b, func(r, g, bl, a uint8) bool { return r == 0xd8 && g == 0x3a && bl == 0x2a })
	orange := countPx(b, func(r, g, bl, a uint8) bool { return r == 0xf0 && g == 0x8a && bl == 0x20 })
	blue := countPx(b, func(r, g, bl, a uint8) bool { return r == 0x2c && g == 0x64 && bl == 0xb0 })
	if red < 100 || orange < 100 || blue < 60 {
		t.Errorf("ferry colours missing: red %d orange %d windows %d", red, orange, blue)
	}
	if k := propKinds["jp_boat"]; k == nil || k.Draw == nil {
		t.Fatal("jp_boat is not the built-in ferry")
	} else {
		pb := newPix(k.W, k.H)
		k.Draw(pb, 0)
		if same(pb, newPix(k.W, k.H)) {
			t.Error("jp_boat draws nothing")
		}
	}
}

// the arrival shuttle is the same ship as the ones of the space station, crisp at every size
func TestArrivalShuttleIsTheShip(t *testing.T) {
	im, k := arrShipImg(1)
	if k != 1 || im.Bounds().Dx() != 96 || im.Bounds().Dy() != 42 {
		t.Fatalf("shuttle at k=1: %v at %v, want 96x42", im.Bounds(), k)
	}
	im2, k2 := arrShipImg(0.5)
	if k2 != 0.5 || im2.Bounds().Dx() != 48 {
		t.Errorf("shuttle at k=0.5: %v at %v", im2.Bounds(), k2)
	}
	if im3, _ := arrShipImg(2); im3.Bounds().Dx() != 192 || im3.Bounds().Dy() != 84 { // landed on the Earth
		t.Errorf("shuttle at k=2: %v, want 192x84", im3.Bounds())
	}
	if again, _ := arrShipImg(0.51); again != im2 {
		t.Error("the shuttle is not cached per size step")
	}
}

// the pirate ship sails by now and then: about a third of the time, entering and leaving off screen,
// at most a pixel per frame, below the horizon and above the sand
func TestPirateShipSometimes(t *testing.T) {
	seen, frames := 0, 0
	prevOK := false
	var px float64
	w := float64(genPirateShip(0).w)
	for k := 0; k <= 60*1800; k++ {
		tm := float64(k) / 60
		x, y, ok := seaPiratePos(tm)
		frames++
		if ok {
			seen++
			if x != math.Round(x) || y != math.Round(y) {
				t.Fatalf("t=%.2f: not on whole pixels (%v, %v)", tm, x, y)
			}
			if !prevOK && x < screenW {
				t.Fatalf("t=%.2f: appears on screen at x=%v", tm, x)
			}
			if prevOK && math.Abs(x-px) > 1 {
				t.Fatalf("t=%.2f: too fast (%v -> %v)", tm, px, x)
			}
			if y+seaPirateRow > 70 || y < 10 {
				t.Fatalf("t=%.2f: its waterline %v is off the sea band", tm, y+seaPirateRow)
			}
		} else if prevOK && px+w > 0 {
			t.Fatalf("t=%.2f: vanished on screen at x=%v", tm, px)
		}
		prevOK, px = ok, x
	}
	if f := float64(seen) / float64(frames); f < 0.2 || f > 0.6 {
		t.Errorf("the pirate ship is at sea %.0f%% of the time, want now and then (20-60%%)", f*100)
	}
	if _, _, ok := seaPiratePos(1); ok {
		t.Error("the pirate ship is there from the first second")
	}
}

// every use draws with the shared drawers of vessels.go, and the old drawing code is gone
func TestVesselDrawersShared(t *testing.T) {
	read := func(f string) string {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	uses := map[string][]string{
		"transit.go":        {"genBoat(", "genShip(", "transitCraft(v)"},
		"airseaviews.go":    {"genBoatView(", "genShipView(", "transitCraft(v)"},
		"arrival_travel.go": {"genShip("},
		"props8.go":         {"genFerry(b)"},
		"events.go":         {"genRocket()"},
		"seasky.go":         {"genHorizonShip(", "genPirateShip("},
	}
	for f, calls := range uses {
		src := read(f)
		for _, cl := range calls {
			if !strings.Contains(src, cl) {
				t.Errorf("%s does not draw with %s", f, cl)
			}
		}
	}
	gone := map[string][]string{
		"transit.go":        {"func genBoat(", "func genShip("},
		"airseaviews.go":    {"func genBoatView(", "func genShipView("},
		"seasky.go":         {"seaShipRects"},
		"events.go":         {"b(-5, -38"},                                                        // the old nose cone
		"arrival_travel.go": {"filledEllipse(cv, math.Round(x), math.Round(y), math.Round(48*k)"}, // the old hull
	}
	for f, olds := range gone {
		src := read(f)
		for _, o := range olds {
			if strings.Contains(src, o) {
				t.Errorf("%s still holds the old drawing %q", f, o)
			}
		}
	}
	// no orphans: every function of vessels.go is called somewhere
	var all strings.Builder
	ents, _ := os.ReadDir(".")
	for _, e := range ents {
		if n := e.Name(); strings.HasSuffix(n, ".go") && !strings.HasSuffix(n, "_test.go") {
			all.WriteString(read(n))
		}
	}
	src := all.String()
	for _, m := range regexp.MustCompile(`(?m)^func (\w+)\(`).FindAllStringSubmatch(read("vessels.go"), -1) {
		if strings.Count(src, m[1]+"(") < 2 {
			t.Errorf("vessels.go: %s is never used", m[1])
		}
	}
}
