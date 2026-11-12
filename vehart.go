package main

import (
	"flag"
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// Land vehicle art kit (2026-11-10 redesign). Every road vehicle of the game
// (traffic cars, taxi, vans, trucks, buses, the Rocket League cars, the
// motorbike, the fire engine, the technicians' van, the hearse, the police car
// of every unit and the ambulance) is painted with these helpers, in side,
// front and rear view: a dark 1 px contour, clean three-tone shading,
// turquoise glass with a diagonal shine and wheels with a light rim.
// Coordinates are design pixels (wk = 1): integer, crisp.
// ---------------------------------------------------------------------------

var (
	vaInk      = c(0x15151b) // the contour
	vaTyre     = c(0x1c1c22)
	vaTyreSide = c(0x34343e)
	vaRimC     = c(0xc8ccd6)
	vaHub      = c(0x3a3a44)
	vaGlassC   = c(0x6ec4d2)
	vaShineC   = c(0xd4f4f8)
	vaHeadC    = c(0xfff0b0) // TestVehicleNoseSide looks for this exact colour on the nose
	vaTailC    = c(0xe03030)
	vaAmberC   = c(0xf0a020)
	vaChromeC  = c(0x9aa0aa)
	vaSkirtC   = c(0x26262e)
)

// vaKitUse counts calls to the kit (tests check that every vehicle drawing goes through it).
var vaKitUse struct{ build, wheel int }

// vaBuild paints a vehicle with draw on a clean layer, gives that layer a dark contour and lays it
// over b (whose shadow is already there). post paints on top afterwards without contour (light rays).
func vaBuild(b *pixBuf, shadowW int, draw func(t *pixBuf), post func(t *pixBuf)) {
	vaKitUse.build++
	if shadowW > 0 {
		b.ellipse(float64(shadowW)/2, 32, float64(shadowW)/2-2, 3.5, color.RGBA{0, 0, 0, 90})
	}
	t := &pixBuf{w: b.w, h: b.h, ox: b.ox, oy: b.oy, p: make([]byte, len(b.p))}
	draw(t)
	vaContour(t)
	if post != nil {
		post(t)
	}
	for i := 0; i < len(t.p); i += 4 {
		if t.p[i+3] > 0 {
			copy(b.p[i:i+4], t.p[i:i+4])
		}
	}
}

// vaContour: every empty pixel that touches the painted vehicle becomes the dark contour.
func vaContour(t *pixBuf) {
	solid := func(x, y int) bool {
		return x >= 0 && y >= 0 && x < t.w && y < t.h && t.p[(y*t.w+x)*4+3] >= 200
	}
	var edge []int
	for y := 0; y < t.h; y++ {
		for x := 0; x < t.w; x++ {
			if t.p[(y*t.w+x)*4+3] != 0 {
				continue
			}
			if solid(x-1, y) || solid(x+1, y) || solid(x, y-1) || solid(x, y+1) {
				edge = append(edge, (y*t.w+x)*4)
			}
		}
	}
	for _, i := range edge {
		t.p[i], t.p[i+1], t.p[i+2], t.p[i+3] = vaInk.R, vaInk.G, vaInk.B, 255
	}
}

// vaMirror flips a layer left to right (a vehicle drawn nose right becomes nose left).
func vaMirror(t *pixBuf) {
	for y := 0; y < t.h; y++ {
		for x := 0; x < t.w/2; x++ {
			i, j := (y*t.w+x)*4, (y*t.w+t.w-1-x)*4
			for k := 0; k < 4; k++ {
				t.p[i+k], t.p[j+k] = t.p[j+k], t.p[i+k]
			}
		}
	}
}

// vaRow fills the pixels x0..x1 (both included) of row y.
func vaRow(t *pixBuf, y, x0, x1 int, col color.RGBA) {
	if x1 >= x0 {
		t.rect(x0, y, x1-x0+1, 1, col)
	}
}

// vaWheel: a tyre seen from the side with a light rim, a dark hub and a glint.
func vaWheel(t *pixBuf, cx, cy, r float64, rim color.RGBA) {
	vaKitUse.wheel++
	t.ellipse(cx, cy, r, r, vaTyre)
	t.ellipse(cx, cy, r*0.78, r*0.78, vaTyreSide)
	t.ellipse(cx, cy, r*0.6, r*0.6, rim)
	t.ellipse(cx, cy, r*0.6-1, r*0.6-1, shade(rim, 0.82))
	t.ellipse(cx, cy, math.Max(1, r*0.22), math.Max(1, r*0.22), vaHub)
	t.set(int(cx-r*0.35), int(cy-r*0.35), shade(rim, 1.25))
}

// vaArch cuts the dark wheel arch into the body before the wheel goes in.
func vaArch(t *pixBuf, cx, cy, r float64) {
	rx, ry := r+1.5, r+1
	for y := int(math.Floor(cy - ry)); y <= int(cy); y++ {
		for x := int(math.Floor(cx - rx)); x <= int(math.Ceil(cx+rx)); x++ {
			dx, dy := (float64(x)+0.5-cx)/rx, (float64(y)+0.5-cy)/ry
			if dx*dx+dy*dy <= 1 {
				t.set(x, y, c(0x101016))
			}
		}
	}
}

// vaTyreFront: the visible side of a tyre in a front or rear view (tread blocks).
func vaTyreFront(t *pixBuf, x, top, w, h int) {
	vaKitUse.wheel++
	t.rect(x, top, w, h, vaTyre)
	for y := top + 1; y < top+h; y += 2 {
		t.rect(x+1, y, w-2, 1, vaTyreSide)
	}
}

// vaGlassPane: a window with a light frame-free pane and a diagonal shine.
func vaGlassPane(t *pixBuf, x, y, w, h int, glass color.RGBA) {
	if w <= 0 || h <= 0 {
		return
	}
	t.rect(x, y, w, h, glass)
	t.rect(x, y+h-1, w, 1, shade(glass, 0.78))
	for k := 0; k < h-2 && k < 4; k++ { // one clean diagonal glint
		t.set(x+2+k, y+h-2-k, vaShineC)
		if x+3+k < x+w-1 {
			t.set(x+3+k, y+h-2-k, vaShineC)
		}
	}
}

// ---- light bars ------------------------------------------------------------------------------

// vaBarFrames is how many pictures the animated emergency lights have.
const vaBarFrames = 4

// vaBarLevels: how bright the left (red) and right (blue) halves are in frame f.
func vaBarLevels(f int) (l, r float64) {
	switch f % vaBarFrames {
	case 0:
		return 1, 0.25
	case 1:
		return 0.55, 0.55
	case 2:
		return 0.25, 1
	}
	return 1, 1
}

func vaLamp(base color.RGBA, k float64) color.RGBA {
	if k >= 0.99 {
		return mixc(base, c(0xffffff), 0.25)
	}
	return shade(base, 0.35+0.6*k)
}

// vaLightbar: a roof bar w wide (left red, right blue) at x, y (3 rows) in frame f.
func vaLightbar(t *pixBuf, x, y, w, f int, left, right color.RGBA) {
	l, r := vaBarLevels(f)
	t.rect(x, y+2, w, 1, c(0x30323a))
	t.rect(x+1, y, w/2-1, 2, vaLamp(left, l))
	t.rect(x+w/2, y, w-w/2-1, 2, vaLamp(right, r))
	t.rect(x+w/2-1, y, 1, 2, c(0xe8e8f0))
	t.set(x+2, y, shade(vaLamp(left, l), 1.3))
	t.set(x+w/2+1, y, shade(vaLamp(right, r), 1.3))
}

// vaRays: the short flashes over a bright light (drawn after the contour, no outline).
func vaRays(t *pixBuf, x, y, w, f int, left, right color.RGBA) {
	l, r := vaBarLevels(f)
	ray := func(cx int, col color.RGBA) {
		col = mixc(col, c(0xffffff), 0.35)
		t.set(cx, y-2, col)
		t.set(cx, y-3, col)
		t.set(cx-3, y-1, col)
		t.set(cx-4, y-2, col)
		t.set(cx+3, y-1, col)
		t.set(cx+4, y-2, col)
	}
	if l >= 0.99 {
		ray(x+w/4, left)
	}
	if r >= 0.99 {
		ray(x+w-w/4-1, right)
	}
}

var (
	vaRed  = c(0xe82a2a)
	vaBlue = c(0x2a5aff)
)

// ---- the sedan family (nose on the left) -------------------------------------------------------

type vaCar struct {
	body   color.RGBA
	roof   color.RGBA // 0 alpha: same as the body
	doors  color.RGBA // 0 alpha: same as the body (the police car has white doors)
	glass  color.RGBA
	rim    color.RGBA
	hatch  bool // a longer roof down to the tail (taxi, hearse)
	push   bool // push bar on the nose (police)
	wheelR float64
}

// vaSedanEdges: where the front (windscreen) and rear glass of the cabin are on row y (7..16).
func vaSedanEdges(y int, hatch bool) (fx, rx int) {
	fx = 17 + (16-y)*8/9
	rx = 52 - (16-y)*6/9
	if hatch {
		rx = 57 - (16-y)*2/9
	}
	return
}

// vaSedanSide paints a car 66 wide with the nose on the left (wheels at 14 and 51, on y 30).
func vaSedanSide(t *pixBuf, p vaCar) {
	body := p.body
	roof := p.roof
	if roof.A == 0 {
		roof = body
	}
	glass := p.glass
	if glass.A == 0 {
		glass = vaGlassC
	}
	rim := p.rim
	if rim.A == 0 {
		rim = vaRimC
	}
	wr := p.wheelR
	if wr == 0 {
		wr = 5.5
	}
	// the cabin
	for y := 7; y <= 16; y++ {
		fx, rx := vaSedanEdges(y, p.hatch)
		vaRow(t, y, fx, rx, roof)
	}
	{
		fx, rx := vaSedanEdges(7, p.hatch)
		vaRow(t, 7, fx+1, rx-1, shade(roof, 1.3))
	}
	for y := 9; y <= 15; y++ {
		fx, rx := vaSedanEdges(y, p.hatch)
		vaRow(t, y, fx+2, 31, glass)
		vaRow(t, y, 35, rx-2, shade(glass, 0.86))
	}
	// shine on both windows
	for k := 0; k < 3; k++ {
		t.set(24+k, 14-k*2, vaShineC)
		t.set(25+k, 14-k*2, vaShineC)
		t.set(39+k, 14-k*2, vaShineC)
	}
	vaRow(t, 16, 18, 50, shade(roof, 0.75)) // belt line
	// the body
	vaRow(t, 17, 7, 60, shade(body, 1.28))
	vaRow(t, 18, 3, 63, shade(body, 1.12))
	for y := 19; y <= 22; y++ {
		vaRow(t, y, 2, 63, body)
	}
	vaRow(t, 23, 2, 63, shade(body, 0.86))
	for y := 24; y <= 26; y++ {
		vaRow(t, y, 2, 63, shade(body, 0.92))
	}
	if p.doors.A > 0 {
		d := p.doors
		vaRow(t, 17, 18, 49, shade(d, 1.1))
		for y := 18; y <= 25; y++ {
			vaRow(t, y, 18, 49, d)
		}
		vaRow(t, 23, 18, 49, shade(d, 0.9))
	}
	vaRow(t, 27, 3, 62, vaSkirtC)
	vaRow(t, 28, 5, 60, vaSkirtC)
	// doors, handles, mirror
	t.rect(34, 17, 1, 9, shade(body, 0.55))
	t.rect(49, 17, 1, 8, shade(body, 0.55))
	t.rect(18, 17, 1, 9, shade(body, 0.6))
	t.rect(29, 19, 3, 1, c(0xe0e4ea))
	t.rect(44, 19, 3, 1, c(0xe0e4ea))
	t.rect(19, 14, 3, 2, shade(roof, 0.8))
	// lights and bumpers
	t.rect(2, 19, 3, 2, vaHeadC)
	t.rect(2, 21, 2, 1, vaAmberC)
	t.rect(62, 18, 2, 3, vaTailC)
	t.rect(1, 25, 4, 2, vaChromeC)
	t.rect(61, 25, 4, 2, vaChromeC)
	if p.push {
		t.rect(0, 19, 2, 8, c(0x2a2a30))
		t.rect(0, 20, 1, 6, c(0x4a4a54))
	}
	// wheels
	for _, x := range []float64{14, 51} {
		vaArch(t, x, 30, wr)
		vaWheel(t, x, 30, wr, rim)
	}
}

func vaCarCol(v int) color.RGBA {
	return c([]uint32{0xc8383a, 0x2f6fd0, 0x3aa65a, 0xe8c040}[v%4])
}

// genCar: the plain car (four colours), the police car (4) and the ambulance (5).
func genCar(b *pixBuf, v int) {
	switch v {
	case 4:
		vaPoliceSide(b, unitPolice, 0)
	case 5:
		vaAmbulanceSide(b, 0)
	default:
		vaBuild(b, 66, func(t *pixBuf) { vaSedanSide(t, vaCar{body: vaCarCol(v)}) }, nil)
	}
}

// vaTaxiSide: the yellow taxi with its roof sign and a chequered band.
func vaTaxiSide(b *pixBuf) {
	vaBuild(b, 66, func(t *pixBuf) {
		vaSedanSide(t, vaCar{body: c(0xf2c21a), hatch: true})
		for x := 4; x < 62; x += 2 {
			col := c(0x1c1c22)
			if (x/2)%2 == 0 {
				col = c(0xf4f4f6)
			}
			t.rect(x, 22, 2, 1, col)
		}
		t.rect(28, 3, 12, 4, c(0xf4f4f6)) // the sign
		t.rect(29, 4, 10, 1, c(0x1c1c22))
		t.rect(29, 5, 10, 1, c(0xf2c21a))
		t.rect(30, 2, 8, 1, vaAmberC)
	}, nil)
}

// vaHearseSide: long black wagon with curtains and a chrome landau bar.
func vaHearseSide(b *pixBuf) {
	vaBuild(b, 66, func(t *pixBuf) {
		vaSedanSide(t, vaCar{body: c(0x241e2c), glass: c(0x3a4a5a), rim: c(0xb8bcc8), hatch: true})
		for y := 9; y <= 15; y++ { // curtains in the long rear window
			_, rx := vaSedanEdges(y, true)
			vaRow(t, y, 36, rx-2, c(0x5a2a6a))
			for x := 37; x < rx-2; x += 3 {
				t.set(x, y, c(0x7a4a8a))
			}
		}
		t.rect(44, 11, 9, 1, vaChromeC) // landau bar
		t.rect(44, 12, 2, 2, vaChromeC)
		t.rect(51, 12, 2, 2, vaChromeC)
		t.rect(4, 20, 58, 1, c(0x8a6aa8)) // purple pinstripe
		t.rect(36, 4, 18, 3, c(0x5a3c28)) // the coffin on the roof rack
		t.rect(36, 4, 18, 1, c(0x8a6040))
		t.rect(42, 5, 6, 1, c(0xd0a840))
		t.rect(35, 7, 20, 1, vaChromeC)
	}, nil)
}

// vaPoliceSide: the patrol car of each unit, nose left, with its light bar in frame f.
func vaPoliceSide(b *pixBuf, unit, f int) {
	bar := true
	var post func(t *pixBuf)
	vaBuild(b, 66, func(t *pixBuf) {
		switch unit {
		case unitFBI: // unmarked black sedan; the lights hide in the grille and on the dash
			vaSedanSide(t, vaCar{body: c(0x1a1c22), glass: c(0x2c4250), rim: c(0x6a6e78)})
			l, r := vaBarLevels(f)
			t.rect(2, 22, 2, 1, vaLamp(vaRed, l))
			t.rect(2, 23, 2, 1, vaLamp(vaBlue, r))
			t.rect(22, 15, 2, 1, vaLamp(vaRed, l))
			t.rect(24, 15, 2, 1, vaLamp(vaBlue, r))
			t.rect(6, 20, 52, 1, c(0xb89a40)) // thin gold line
			bar = false
		case unitMilitary: // olive, camouflage blotches, a white star and a roof rack
			vaSedanSide(t, vaCar{body: c(0x56663a), rim: c(0x8a8456), push: true})
			for _, s := range [][4]int{{6, 19, 7, 3}, {37, 22, 9, 2}, {53, 19, 6, 3}, {24, 24, 6, 2}} {
				t.rect(s[0], s[1], s[2], s[3], c(0x3e4a28))
			}
			t.rect(14, 25, 5, 1, c(0x8a8456))
			t.rect(41, 18, 3, 1, c(0xf0f0f0)) // star
			t.rect(40, 19, 5, 1, c(0xf0f0f0))
			t.rect(41, 20, 3, 1, c(0xf0f0f0))
			t.rect(40, 21, 1, 1, c(0xf0f0f0))
			t.rect(44, 21, 1, 1, c(0xf0f0f0))
			t.rect(21, 6, 6, 1, c(0x3e4a28))
			t.rect(42, 6, 6, 1, c(0x3e4a28))
		case unitSWAT: // armoured van, dark with a yellow band
			vaVanShape(t, 66, c(0x30343e), c(0x2c3e4c), c(0x8a8e98))
			t.rect(4, 20, 58, 2, c(0xf0c020))
			vaMirror(t)
			bar = true
		case unitRangers: // white car with the five colours along its side
			vaSedanSide(t, vaCar{body: c(0xf0f0f4), push: true})
			for i, col := range []uint32{0xe03030, 0x2a5ae0, 0xf0c820, 0xf060a8, 0x30b050} {
				t.rect(4+i*11, 21, 10, 2, c(col))
			}
		default: // the police: black and white
			vaSedanSide(t, vaCar{body: c(0x22252c), roof: c(0xf0f2f6), doors: c(0xf0f2f6), push: true})
			t.rect(25, 19, 3, 3, c(0xd8b040)) // badge
			t.set(26, 18, c(0xd8b040))
			t.rect(36, 21, 10, 1, c(0x2a4ab8))
		}
		if bar {
			y := 4
			if unit == unitSWAT {
				y = 3
			}
			vaLightbar(t, 26, y, 15, f, vaRed, vaBlue)
			post = func(t *pixBuf) { vaRays(t, 26, y, 15, f, vaRed, vaBlue) }
		}
	}, func(t *pixBuf) {
		if post != nil {
			post(t)
		}
	})
}

// ---- vans (nose on the right) ------------------------------------------------------------------

// vaVanShape: a van w wide (w >= 66), nose right, wheels near both ends.
func vaVanShape(t *pixBuf, w int, body, glass, rim color.RGBA) {
	front := w - 3 // the bumper
	ws := w - 20   // where the windscreen starts at the roof
	top := 8
	if w <= 66 {
		top = 6
	}
	for y := top; y <= 28; y++ {
		x1 := front
		if y < 18 {
			x1 = ws + (y-top)*(front-2-ws)/(18-top)
		}
		col := body
		switch {
		case y == top:
			col = shade(body, 1.28)
		case y == top+1:
			col = shade(body, 1.12)
		case y >= 25:
			col = shade(body, 0.9)
		}
		vaRow(t, y, 2, x1, col)
	}
	vaRow(t, 27, 3, front-1, vaSkirtC)
	vaRow(t, 28, 4, front-3, vaSkirtC)
	// cab glass following the windscreen
	for y := top + 3; y <= 17; y++ {
		x1 := ws + (y-top)*(front-2-ws)/(18-top) - 2
		vaRow(t, y, ws-8, x1, glass)
	}
	for k := 0; k < 3; k++ {
		t.set(ws-5+k, 15-k*2, vaShineC)
		t.set(ws-4+k, 15-k*2, vaShineC)
	}
	t.rect(ws-10, top+2, 1, 24-top, shade(body, 0.55)) // cab door
	t.rect(w/2-6, top+2, 1, 24-top, shade(body, 0.55)) // sliding door
	t.rect(ws-6, 20, 3, 1, c(0xe0e4ea))
	t.rect(front-1, 20, 2, 2, vaHeadC)
	t.rect(front, 22, 1, 1, vaAmberC)
	t.rect(2, 19, 2, 3, vaTailC)
	t.rect(front-2, 25, 4, 2, vaChromeC)
	t.rect(1, 25, 3, 2, vaChromeC)
	for _, x := range []float64{14, float64(w) - 15} {
		vaArch(t, x, 30, 5.5)
		vaWheel(t, x, 30, 5.5, rim)
	}
}

func vaVanSide(b *pixBuf, v int) {
	cols := [][2]uint32{{0xf2f2f6, 0x2a5ac8}, {0x2f63c8, 0xf2f2f6}, {0xc83a3a, 0xf0c020}}[v%3]
	vaBuild(b, 78, func(t *pixBuf) {
		vaVanShape(t, 78, c(cols[0]), vaGlassC, vaRimC)
		t.rect(4, 21, 50, 2, c(cols[1]))
		vaGlassPane(t, 8, 11, 12, 6, shade(vaGlassC, 0.85))
	}, nil)
}

// vaTechVanSide: the technicians' yellow van with the phone logo, cable reel and beacon.
func vaTechVanSide(b *pixBuf) {
	vaBuild(b, 86, func(t *pixBuf) {
		vaVanShape(t, 86, c(0xf0c828), vaGlassC, vaRimC)
		t.rect(10, 12, 40, 8, c(0xfce070))
		t.rect(12, 14, 6, 4, c(0x2a4ab8))
		t.rect(19, 15, 26, 2, c(0x2a4ab8))
		t.rect(4, 22, 56, 1, c(0x2a4ab8))
		t.rect(8, 5, 52, 2, c(0xc8c8d0))
		t.rect(9, 7, 2, 1, c(0x8a8a94))
		t.rect(56, 7, 2, 1, c(0x8a8a94))
		t.ellipse(34, 4, 5, 2.4, c(0x4a4a54))
		t.rect(46, 3, 3, 2, c(0xff9a20))
	}, nil)
}

// vaIceTruckSide: cream with a pink stripe, a serving hatch with awning and the cone on the roof.
func vaIceTruckSide(b *pixBuf) {
	vaBuild(b, 84, func(t *pixBuf) {
		vaVanShape(t, 84, c(0xf6e8d0), vaGlassC, c(0xf0a0c0))
		t.rect(3, 22, 60, 2, c(0xf08aa8))
		t.rect(14, 13, 28, 8, c(0x2a2a34))
		for x := 14; x < 42; x += 4 {
			t.rect(x, 11, 2, 3, c(0xf0508a))
			t.rect(x+2, 11, 2, 3, c(0xfff4f8))
		}
		t.rect(18, 17, 4, 3, c(0xf8c8d8))
		t.rect(28, 16, 4, 4, c(0xa8e0c0))
		t.rect(33, 4, 3, 4, c(0xd8a860))
		t.ellipse(34.5, 3, 5, 3.5, c(0xf8c8d8))
		t.ellipse(34.5, 0.5, 3, 2.5, c(0xe85088))
	}, nil)
}

// ---- trucks and buses (nose on the right) --------------------------------------------------------

// vaCab: a truck cab from x0 to x0+27 (nose right), roof at y 12, wheels drawn by the caller.
func vaCab(t *pixBuf, x0 int, body color.RGBA) {
	front := x0 + 27
	for y := 12; y <= 28; y++ {
		x1 := front
		if y < 20 {
			x1 = x0 + 16 + (y-12)*(front-2-x0-16)/8
		}
		col := body
		switch {
		case y == 12:
			col = shade(body, 1.3)
		case y >= 25:
			col = shade(body, 0.88)
		}
		vaRow(t, y, x0, x1, col)
	}
	for y := 14; y <= 19; y++ {
		vaRow(t, y, x0+4, x0+14+(y-12)*(front-2-x0-16)/8, vaGlassC)
	}
	t.set(x0+8, 18, vaShineC)
	t.set(x0+9, 17, vaShineC)
	t.set(x0+10, 16, vaShineC)
	t.rect(x0+2, 13, 1, 13, shade(body, 0.55))
	t.rect(x0+10, 21, 3, 1, c(0xe0e4ea))
	t.rect(front-1, 22, 2, 2, vaHeadC)
	t.rect(front-2, 25, 4, 3, vaChromeC)
	t.rect(front-3, 21, 1, 4, c(0x2a2a34)) // grille
	vaRow(t, 27, x0, front-2, vaSkirtC)
}

// vaRibs: the vertical ribs of a box or container.
func vaRibs(t *pixBuf, x0, y0, x1, y1, step int, col color.RGBA) {
	for x := x0 + 2; x < x1; x += step {
		t.rect(x, y0+1, 1, y1-y0-1, col)
	}
}

// vaTruckSide: 0 delivery truck, 1 container truck, 2 bin lorry, 3 grey lorry.
func vaTruckSide(b *pixBuf, v int) {
	vaBuild(b, 100, func(t *pixBuf) {
		cab := c([]uint32{0x2f63c8, 0x3a9a48, 0xb8bcc4, 0xc8ccd4}[v%4])
		t.rect(4, 28, 92, 2, vaSkirtC) // chassis
		switch v % 4 {
		case 0, 3:
			box := c(0xe8ecf0)
			if v%4 == 3 {
				box = c(0x8a9098)
			}
			t.rect(2, 4, 66, 24, box)
			t.rect(2, 4, 66, 1, shade(box, 1.15))
			t.rect(2, 25, 66, 3, shade(box, 0.78))
			vaRibs(t, 2, 4, 67, 25, 4, shade(box, 0.86))
			t.rect(3, 20, 2, 3, vaTailC)
		case 1: // a red container on a trailer
			box := c(0xb8302a)
			t.rect(2, 5, 66, 21, box)
			t.rect(2, 5, 66, 1, shade(box, 1.25))
			vaRibs(t, 2, 5, 67, 25, 3, shade(box, 0.78))
			t.rect(2, 26, 66, 2, c(0x3a3a44))
			t.rect(3, 22, 2, 3, vaTailC)
		case 2: // the bin lorry: rounded blue body and the hopper at the back
			box := c(0x2f58b8)
			t.rect(8, 6, 60, 22, box)
			t.ellipse(10, 17, 8, 11, box)
			t.rect(8, 6, 60, 1, shade(box, 1.25))
			t.rect(14, 10, 48, 2, shade(box, 0.8))
			t.rect(14, 20, 48, 2, shade(box, 0.8))
			t.rect(1, 20, 7, 8, c(0x5a5e68)) // hopper
			t.rect(2, 21, 5, 2, c(0x2a2a30))
			t.rect(2, 18, 2, 2, vaAmberC)
		}
		vaCab(t, 70, cab)
		xs := []float64{14, 50, 82}
		if v%4 == 1 {
			xs = []float64{12, 24, 52, 82}
		}
		for _, x := range xs {
			vaArch(t, x, 30, 5.5)
			vaWheel(t, x, 30, 5.5, vaRimC)
		}
	}, nil)
}

// vaBusSide: 0 the yellow school bus, 1 a green city bus with a cream band.
func vaBusSide(b *pixBuf, v int) {
	vaBuild(b, 112, func(t *pixBuf) {
		body, band := c(0xf0b818), c(0x2a2a30)
		if v%2 == 1 {
			body, band = c(0x46a050), c(0xf0e8b0)
		}
		for y := 6; y <= 28; y++ {
			col := body
			switch {
			case y == 6:
				col = shade(body, 1.3)
			case y >= 25:
				col = shade(body, 0.88)
			}
			x0, x1 := 2, 109
			if y == 6 {
				x0, x1 = 4, 106
			}
			vaRow(t, y, x0, x1, col)
		}
		vaRow(t, 27, 3, 108, vaSkirtC)
		vaRow(t, 28, 4, 106, vaSkirtC)
		if v%2 == 1 {
			t.rect(2, 19, 108, 3, band)
			t.rect(30, 3, 26, 3, shade(band, 0.9)) // air conditioning on the roof
		} else {
			t.rect(2, 21, 108, 1, band)
			t.rect(2, 24, 108, 1, band)
		}
		for x := 8; x < 82; x += 11 {
			vaGlassPane(t, x, 9, 9, 8, vaGlassC)
		}
		t.rect(87, 8, 11, 19, c(0x2a2a34)) // the folding door
		t.rect(88, 9, 4, 17, vaGlassC)
		t.rect(93, 9, 4, 17, vaGlassC)
		vaGlassPane(t, 100, 8, 9, 11, vaGlassC) // windscreen
		t.rect(108, 21, 2, 2, vaHeadC)
		t.rect(108, 25, 3, 2, vaChromeC)
		t.rect(2, 20, 2, 3, vaTailC)
		t.rect(1, 25, 3, 2, vaChromeC)
		if v%2 == 0 {
			t.rect(104, 4, 3, 2, vaAmberC)
			t.rect(6, 4, 3, 2, vaTailC)
		}
		for _, x := range []float64{22, 90} {
			vaArch(t, x, 30, 6)
			vaWheel(t, x, 30, 6, vaRimC)
		}
	}, nil)
}

// vaFireTruckSide: the red fire engine with ladder, lockers and its bar on the cab.
func vaFireTruckSide(b *pixBuf) {
	vaBuild(b, 96, func(t *pixBuf) {
		red := c(0xd02a2a)
		t.rect(2, 12, 66, 16, red)
		t.rect(2, 12, 66, 1, shade(red, 1.3))
		t.rect(2, 25, 66, 3, shade(red, 0.8))
		for x := 8; x < 64; x += 11 {
			t.rect(x, 14, 9, 9, shade(red, 0.82))
			t.rect(x+1, 18, 7, 1, c(0xd8d8e0))
		}
		t.rect(2, 21, 92, 2, c(0xf4f4f4))
		t.rect(4, 7, 62, 2, c(0xc8c8d0)) // ladder
		t.rect(4, 10, 62, 1, c(0xc8c8d0))
		for x := 6; x < 64; x += 5 {
			t.rect(x, 7, 1, 4, c(0x9a9aa4))
		}
		t.rect(8, 11, 2, 1, c(0x6a6a74))
		t.rect(58, 11, 2, 1, c(0x6a6a74))
		t.ellipse(14, 19, 4, 3.5, c(0x8a8a94))
		t.ellipse(14, 19, 2, 1.6, c(0x4a4a54))
		vaCab(t, 68, red)
		t.rect(68, 21, 26, 2, c(0xf4f4f4))
		t.rect(74, 9, 11, 3, c(0x30323a))
		t.rect(75, 9, 4, 2, vaRed)
		t.rect(80, 9, 4, 2, vaBlue)
		t.rect(4, 28, 88, 1, vaSkirtC)
		for _, x := range []float64{16, 46, 80} {
			vaArch(t, x, 30, 5.5)
			vaWheel(t, x, 30, 5.5, vaRimC)
		}
	}, nil)
}

// ---- the ambulance (nose left) ---------------------------------------------------------------------

// vaAmbulanceSide: white box with a blue cross, a red band, turquoise windows and a row of coloured
// lights on top that take turns in frame f. Painted nose right and mirrored.
func vaAmbulanceSide(b *pixBuf, f int) {
	lamps := []color.RGBA{c(0xe82a2a), c(0x2a7aff), c(0xf08a20), c(0xf0d030)}
	vaBuild(b, 66, func(t *pixBuf) {
		white := c(0xf4f4f6)
		for y := 3; y <= 28; y++ { // the box
			col := white
			switch {
			case y == 3:
				col = c(0xffffff)
			case y >= 23:
				col = c(0xe4d8c8)
			}
			vaRow(t, y, 2, 44, col)
		}
		for y := 9; y <= 28; y++ { // the cab
			x1 := 63
			if y < 17 {
				x1 = 52 + (y-9)*9/8
			}
			col := white
			switch {
			case y == 9:
				col = c(0xffffff)
			case y >= 23:
				col = c(0xe4d8c8)
			}
			vaRow(t, y, 45, x1, col)
		}
		vaRow(t, 27, 3, 62, vaSkirtC)
		vaRow(t, 28, 4, 60, vaSkirtC)
		t.rect(2, 19, 62, 2, c(0xd82828)) // red band
		t.rect(2, 21, 62, 1, c(0xf08a20))
		// the cross
		t.rect(11, 9, 11, 5, c(0x1c3a78))
		t.rect(14, 6, 5, 11, c(0x1c3a78))
		t.rect(12, 10, 9, 3, c(0x3a8ae8))
		t.rect(15, 7, 3, 9, c(0x3a8ae8))
		t.set(15, 8, c(0x9ad0ff))
		// windows and doors
		vaGlassPane(t, 31, 8, 10, 6, vaGlassC)
		for y := 11; y <= 16; y++ {
			vaRow(t, y, 47, 50+(y-9)*9/8, vaGlassC)
		}
		t.set(49, 15, vaShineC)
		t.set(50, 14, vaShineC)
		t.rect(29, 6, 1, 20, shade(white, 0.6))
		t.rect(44, 6, 1, 20, shade(white, 0.6))
		t.rect(3, 5, 1, 21, shade(white, 0.7))
		t.rect(40, 16, 3, 1, c(0x8a8e98))
		t.rect(55, 18, 3, 1, c(0x8a8e98))
		t.rect(62, 19, 2, 2, vaHeadC)
		t.rect(62, 25, 3, 2, vaChromeC)
		t.rect(2, 16, 2, 2, vaTailC)
		t.rect(1, 25, 3, 2, vaChromeC)
		// the coloured lights along the top and the beacon over the cab
		for i, col := range lamps {
			k := 0.3
			if (i+f)%2 == 0 {
				k = 1
			}
			t.rect(5+i*6, 1, 4, 2, vaLamp(col, k))
		}
		l, r := vaBarLevels(f)
		t.rect(37, 0, 4, 3, vaLamp(vaRed, l))
		t.rect(41, 0, 3, 3, vaLamp(vaBlue, r))
		for _, x := range []float64{14, 54} {
			vaArch(t, x, 30, 5.5)
			vaWheel(t, x, 30, 5.5, c(0xd8dce4))
			t.rect(int(x)-1, 29, 2, 2, c(0xd82828))
		}
		vaMirror(t)
	}, nil)
}

// ---- Rocket League style cars (nose on the right) ---------------------------------------------------

type vaRLShape struct {
	w             int
	top           int // top of the body at the back
	noseY         int // top of the body at the nose
	cabX0, cabX1  int // cabin (canopy) from .. to at its base
	cabTop        int
	rearX, frontX float64 // wheel centres
	wheelY, r     float64
	wing          bool
}

func rlTeam(v int) (body, trim color.RGBA) {
	if v%2 == 0 {
		return c(0x2a72f0), c(0xe8f2ff)
	}
	return c(0xf07a18), c(0xffe2b8)
}

// vaRLSide: a low, fat-wheeled sports car with a dark canopy, white stripes, red details and a wing.
func vaRLSide(b *pixBuf, s vaRLShape, v int) {
	body, trim := rlTeam(v)
	vaBuild(b, s.w, func(t *pixBuf) {
		front := s.w - 3
		bot := 26
		for y := s.top; y <= bot; y++ {
			x1 := front
			if y < s.noseY { // the hood slopes down to the nose
				x1 = s.cabX1 + (y-s.top)*(front-s.cabX1)/maxI(1, s.noseY-s.top)
			}
			col := body
			switch {
			case y == s.top:
				col = shade(body, 1.3)
			case y >= bot-2:
				col = shade(body, 0.7)
			}
			vaRow(t, y, 4, x1, col)
		}
		vaRow(t, bot+1, 6, front-2, vaSkirtC)
		// the canopy
		for y := s.cabTop; y < s.top; y++ {
			k := s.top - y
			x0 := s.cabX0 + k*2/3
			x1 := s.cabX1 - k
			vaRow(t, y, x0, x1, c(0x1a2030))
		}
		vaRow(t, s.cabTop, s.cabX0+(s.top-s.cabTop)*2/3+1, s.cabX1-(s.top-s.cabTop)-1, c(0x3a4a68))
		t.set(s.cabX1-(s.top-s.cabTop)/2-2, s.cabTop+2, c(0x8ab0e0))
		t.set(s.cabX1-(s.top-s.cabTop)/2-3, s.cabTop+3, c(0x8ab0e0))
		// stripes and details
		vaRow(t, s.top+2, s.cabX1-2, front-4, trim)
		vaRow(t, s.top+4, 8, front-6, trim)
		t.rect(s.cabX0-4, s.top-1, 4, 2, c(0xd82828)) // intake behind the cabin
		t.rect(front/2, s.top+6, 5, 1, c(0xd82828))   // side vent
		t.rect(front-1, s.noseY+1, 2, 2, vaHeadC)
		t.rect(front-3, s.noseY+1, 2, 1, c(0x40e0f0))
		t.rect(4, s.top+2, 2, 2, vaTailC)
		t.rect(1, s.top+4, 3, 2, c(0x8a8e98)) // the boost exhaust
		if s.wing {
			t.rect(1, s.top-7, 12, 2, c(0x1c1c24))
			t.rect(1, s.top-7, 12, 1, trim)
			t.rect(6, s.top-5, 2, 5, c(0x2a2a34))
		}
		for _, x := range []float64{s.rearX, s.frontX} {
			vaArch(t, x, s.wheelY, s.r)
			vaWheel(t, x, s.wheelY, s.r, trim)
		}
	}, nil)
}

var (
	vaOctane  = vaRLShape{w: 64, top: 16, noseY: 20, cabX0: 20, cabX1: 42, cabTop: 9, rearX: 16, frontX: 48, wheelY: 28, r: 6, wing: true}
	vaFennec  = vaRLShape{w: 58, top: 13, noseY: 17, cabX0: 14, cabX1: 44, cabTop: 7, rearX: 15, frontX: 43, wheelY: 28, r: 6, wing: true}
	vaDominus = vaRLShape{w: 68, top: 18, noseY: 21, cabX0: 20, cabX1: 42, cabTop: 12, rearX: 16, frontX: 52, wheelY: 27, r: 5.5, wing: true}
)

// ---- the motorbike (nose on the right) -----------------------------------------------------------------

func vaMotorbikeSide(b *pixBuf, v int) {
	cc := c([]uint32{0x8a3ad0, 0xd02a2a}[v%2])
	vaBuild(b, 0, func(t *pixBuf) {
		t.line(10, 24, 22, 18, c(0x5a5e68)) // swing arm
		t.line(42, 24, 38, 10, c(0x9aa0aa)) // fork
		t.line(43, 24, 39, 10, c(0x6a6e78))
		t.rect(18, 15, 20, 6, cc) // tank and fairing
		t.rect(18, 15, 20, 1, shade(cc, 1.35))
		t.rect(36, 10, 8, 9, cc)
		t.rect(36, 10, 8, 1, shade(cc, 1.35))
		t.rect(38, 8, 5, 2, vaGlassC)     // windscreen
		t.rect(8, 13, 12, 3, c(0x2a2a34)) // seat
		t.rect(4, 13, 5, 3, cc)           // tail
		t.rect(3, 13, 2, 2, vaTailC)
		t.rect(43, 13, 2, 2, vaHeadC)
		t.rect(22, 21, 12, 3, c(0x4a4e58)) // engine
		t.rect(23, 22, 10, 1, c(0x6a6e78))
		t.rect(8, 19, 14, 2, vaChromeC)  // exhaust
		t.rect(36, 6, 6, 2, c(0x3a3a46)) // handlebar
		vaWheel(t, 10, 24, 5.5, vaRimC)
		vaWheel(t, 42, 24, 5.5, vaRimC)
	}, nil)
}

// ---- famous cars repainted with the kit --------------------------------------------------------------------

func vaBatmobileSide(b *pixBuf) {
	vaBuild(b, 84, func(t *pixBuf) {
		k := c(0x1c1c26)
		t.rect(2, 20, 80, 7, k)
		t.rect(2, 20, 80, 1, c(0x4a4a58))
		t.rect(4, 27, 76, 1, vaSkirtC)
		t.rect(8, 14, 30, 7, k)
		t.rect(28, 11, 20, 10, c(0x24242e))
		vaGlassPane(t, 30, 12, 14, 6, c(0x2a3a58))
		t.rect(54, 17, 26, 4, k)
		t.rect(0, 8, 8, 13, k) // fins
		t.rect(2, 4, 5, 6, k)
		t.rect(66, 22, 14, 2, c(0xe8c030))
		t.rect(80, 21, 2, 2, vaHeadC)
		t.rect(2, 23, 3, 3, c(0xff4a20))
		for _, x := range []float64{18, 64} {
			vaArch(t, x, 28, 5.5)
			vaWheel(t, x, 28, 5.5, c(0x5a5e68))
		}
	}, nil)
}

func vaDeloreanSide(b *pixBuf) {
	vaBuild(b, 72, func(t *pixBuf) {
		s := c(0xb8bcc6)
		t.rect(2, 18, 66, 9, s)
		t.rect(2, 18, 66, 1, c(0xe8ecf4))
		t.rect(2, 25, 66, 2, shade(s, 0.75))
		t.rect(3, 27, 64, 1, vaSkirtC)
		t.rect(12, 9, 34, 10, shade(s, 0.9))
		vaGlassPane(t, 14, 11, 14, 7, c(0x4a6080))
		vaGlassPane(t, 30, 11, 14, 7, c(0x4a6080))
		for y := 20; y < 26; y += 2 { // rear vents
			t.rect(2, y, 6, 1, c(0x3a3a44))
		}
		t.rect(66, 21, 2, 2, vaHeadC)
		t.rect(34, 6, 18, 3, c(0x6a6a74)) // gull-wing door, half open
		t.rect(2, 22, 66, 1, c(0x8a8e98))
		for _, x := range []float64{16, 54} {
			vaArch(t, x, 30, 5.5)
			vaWheel(t, x, 30, 5.5, vaRimC)
		}
	}, nil)
}

func vaWackyMeanSide(b *pixBuf) {
	vaBuild(b, 80, func(t *pixBuf) {
		p := c(0x7a30c0)
		t.rect(2, 20, 76, 7, p)
		t.rect(2, 20, 76, 1, c(0xb070f0))
		t.rect(3, 27, 74, 1, vaSkirtC)
		t.rect(34, 12, 18, 9, shade(p, 0.85))
		t.rect(38, 8, 8, 5, c(0xf0c8a0)) // driver
		t.rect(0, 10, 7, 12, p)
		t.rect(70, 24, 8, 2, c(0xe8c030))
		t.rect(2, 21, 2, 2, vaHeadC)
		t.rect(62, 14, 3, 7, c(0x2a2a34)) // exhausts
		t.rect(66, 14, 3, 7, c(0x2a2a34))
		for _, x := range []float64{14, 62} {
			vaArch(t, x, 28, 5.5)
			vaWheel(t, x, 28, 5.5, c(0xe8c030))
		}
	}, nil)
}

func vaWackyBoulderSide(b *pixBuf) {
	vaBuild(b, 70, func(t *pixBuf) {
		g := c(0x8a8a90)
		t.ellipse(34, 17, 30, 12, g)
		t.ellipse(30, 13, 18, 7, shade(g, 1.15))
		t.rect(18, 8, 4, 2, shade(g, 0.6))
		t.rect(42, 20, 8, 2, shade(g, 0.6))
		vaGlassPane(t, 26, 6, 14, 5, vaGlassC)
		for _, x := range []float64{14, 54} {
			vaWheel(t, x, 30, 5.5, c(0x8a6a40))
		}
	}, nil)
}

func vaMonsterTruckSide(b *pixBuf, v int) {
	body := c([]uint32{0x28a840, 0xd02828, 0x2858d0}[v%3])
	vaBuild(b, 0, func(t *pixBuf) {
		t.rect(10, 26, 80, 14, body)
		t.rect(10, 26, 80, 2, shade(body, 1.3))
		t.rect(10, 37, 80, 3, shade(body, 0.65))
		t.rect(58, 14, 26, 13, shade(body, 0.92)) // cab
		vaGlassPane(t, 60, 16, 18, 8, vaGlassC)
		t.rect(84, 28, 6, 6, vaHeadC)
		t.rect(10, 30, 5, 5, vaTailC)
		for i := 0; i < 5; i++ { // flame decals
			t.rect(16+i*7, 31, 5-i%2, 2, c(0xffb020))
			t.rect(18+i*7, 29, 2, 2, c(0xe85020))
		}
		t.rect(18, 40, 6, 12, c(0x5a5a64))
		t.rect(74, 40, 6, 12, c(0x5a5a64))
		vaWheel(t, 21, 49, 14, c(0xd8d8e0))
		vaWheel(t, 77, 49, 14, c(0xd8d8e0))
	}, nil)
}

// vaJeepSide: the park jeep (nose right) with its roll bar, spare wheel, driver and roundel.
func vaJeepSide(b *pixBuf, v int) {
	body := []color.RGBA{c(0xf2c418), c(0x4aa048), c(0xf2c418)}[v%3]
	trim := []color.RGBA{c(0x2c8a3c), c(0xf2c418), c(0xb83030)}[v%3]
	vaBuild(b, 0, func(t *pixBuf) {
		for _, px := range []int{14, 30} { // roll bar
			t.rect(px, 5, 2, 13, c(0x23232b))
		}
		t.rect(14, 4, 18, 2, c(0x23232b))
		t.ellipse(24, 14, 3, 3, c(0xd8a070)) // a driver
		t.rect(23, 12, 3, 1, c(0x6a4a2a))
		t.rect(38, 7, 2, 11, vaGlassC) // windscreen
		t.rect(38, 7, 1, 11, vaShineC)
		for y := 17; y <= 26; y++ {
			col := body
			switch {
			case y == 17:
				col = shade(body, 1.3)
			case y >= 24:
				col = shade(body, 0.75)
			}
			vaRow(t, y, 3, 55, col)
		}
		t.rect(40, 13, 15, 4, shade(body, 1.05)) // hood
		t.rect(40, 13, 15, 1, shade(body, 1.3))
		t.rect(11, 20, 28, 2, trim) // stripe
		t.rect(14, 18, 20, 2, c(0x2a2a32))
		t.rect(21, 17, 6, 2, c(0xe8e0d0))
		t.ellipse(26, 21, 2.6, 2.6, c(0xb82a2a)) // logo roundel
		t.ellipse(26, 21, 1.2, 1.2, c(0x14101a))
		t.rect(53, 18, 3, 2, vaHeadC)
		t.rect(55, 23, 2, 3, vaChromeC)
		t.rect(2, 20, 2, 3, vaTailC)
		vaWheel(t, 6, 15, 4, c(0x8a8a94)) // the spare wheel
		vaRow(t, 26, 4, 54, vaSkirtC)
		for _, wx := range []float64{14, 44} {
			vaArch(t, wx, 27, 5.6)
			vaWheel(t, wx, 27, 5.6, vaRimC)
		}
	}, nil)
}

// ---- the animated emergency vehicles in play ------------------------------------------------------------

// sirenFrame: which light picture the police and ambulance show at time t.
func sirenFrame(t float64) int {
	return int(t*sirenHz*vaBarFrames) % vaBarFrames
}

var emergencyCache = map[[3]int]PropImg{}

// emergencyPix renders a police car of a unit (kind 0) or the ambulance (kind 1) in light frame f,
// with the same finish as the "car" prop sprites (same size and anchor).
func emergencyPix(kind, unit, f int) *pixBuf {
	k := propKinds["car"]
	w, h := 66, 36
	if k != nil && k.Draw != nil {
		w, h = k.W, k.H
	}
	b := newPix(w+2, h+2)
	b.ox, b.oy = 1, 1
	v := 4
	if kind == 1 {
		v = 5
		vaAmbulanceSide(b, f)
	} else {
		vaPoliceSide(b, unit, f)
	}
	seed := int64(len("car")*31+v) + int64('c')*977
	return stylizeSprite(b.scaled(wk).detail(seed, 0.035).outlinedSel(0.45))
}

// emergencyImg is the cached picture of emergencyPix, anchored at the middle of its base.
func emergencyImg(kind, unit, f int) PropImg {
	if kind != 1 {
		kind = 0
	} else {
		unit = 0
	}
	key := [3]int{kind, unit, f % vaBarFrames}
	if pi, ok := emergencyCache[key]; ok {
		return pi
	}
	sb := emergencyPix(kind, unit, f%vaBarFrames)
	pi := PropImg{img: sb.img(), ox: math.Round(float64(sb.w) / 2), oy: float64(sb.h - 1)}
	emergencyCache[key] = pi
	return pi
}

var hearseCache *PropImg

// hearseImg: the killer's hearse, same size and anchor as a car.
func hearseImg() PropImg {
	if hearseCache != nil {
		return *hearseCache
	}
	b := newPix(68, 38)
	b.ox, b.oy = 1, 1
	vaHearseSide(b)
	sb := stylizeSprite(b.scaled(wk).detail(4242, 0.035).outlinedSel(0.45))
	pi := PropImg{img: sb.img(), ox: 34, oy: 37}
	hearseCache = &pi
	return pi
}

// ---- debug: -vehshow parks every emergency vehicle on the street, to look at them -----------------

var vehShowF = flag.Bool("vehshow", false, "park the police car of every unit, the ambulance and the hearse on the street (debug)")
var vehShowDone bool

func (g *Game) vehShowTick() {
	if !*vehShowF || vehShowDone || g.time < 0.3 || g.lvl == nil || g.lvl.Def.Transit != "" {
		return
	}
	vehShowDone = true
	d := g.lvl.Def
	add := func(i int, v *Vehicle) {
		v.x = d.CarTo.X - float64(i)*78 // the same way as parkSlot
		v.y, v.tx, v.ty, v.state, v.spawned, v.hold = d.CarFrom.Y, v.x, d.CarFrom.Y, 1, true, 999
		g.veh = append(g.veh, v)
	}
	for i, u := range []int{unitPolice, unitFBI, unitMilitary, unitSWAT, unitRangers} {
		add(i, &Vehicle{kind: 0, unit: u})
	}
	add(5, &Vehicle{kind: 1})
	add(6, &Vehicle{kind: 4})
}
