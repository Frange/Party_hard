package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// The window is shown with whole-number scaling only. With a fractional scale (1.38x, 2.76x on a Retina
// display...) every logical pixel covers 1 or 2 screen pixels at random, so the thin strokes of the
// pixel font come out with uneven widths and soft edges. DrawFinalScreen keeps the biggest whole
// scale that fits and centres the picture; the rest of the window stays black.

// DrawFinalScreen implements ebiten.FinalScreenDrawer.
func (g *Game) DrawFinalScreen(screen ebiten.FinalScreen, off *ebiten.Image, geoM ebiten.GeoM) {
	s := geoM.Element(0, 0)
	if s < 1 || s == math.Floor(s) {
		ebiten.DefaultDrawFinalScreen(screen, off, geoM)
		return
	}
	k := math.Floor(s)
	sb, ob := screen.Bounds(), off.Bounds()
	ox := math.Round((float64(sb.Dx()) - float64(ob.Dx())*k) / 2)
	oy := math.Round((float64(sb.Dy()) - float64(ob.Dy())*k) / 2)
	screen.Clear()
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	op.GeoM.Scale(k, k)
	op.GeoM.Translate(ox, oy)
	screen.DrawImage(off, op)
}

// wholeWindowScale returns the window scale (in logical pixels) closest to want that is a whole
// number of device pixels.
func wholeWindowScale(want float64) (sc float64) {
	dsf := 1.0
	func() {
		defer func() { recover() }()
		if m := ebiten.Monitor(); m != nil {
			if f := m.DeviceScaleFactor(); f > 0 {
				dsf = f
			}
		}
	}()
	return wholeScaleFor(want, dsf)
}

// wholeScaleFor: the logical window scale that is a whole number of device pixels (at least 1 device pixel).
func wholeScaleFor(want, dsf float64) float64 {
	k := math.Floor(want * dsf)
	if k < 1 {
		k = 1
	}
	return k / dsf
}

var _ ebiten.FinalScreenDrawer = (*Game)(nil)
