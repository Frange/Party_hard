package main

import (
	"math"
	"testing"
)

// text scales and outline offsets must be whole numbers, or the bitmap font gets strokes of uneven width
func TestTextScalesAreWhole(t *testing.T) {
	for _, s := range []float64{0.5, 1, 1.1, 1.15, 1.3, 1.4, 1.5, 1.6, 2, 2.2, 2.5, 3, 4.4} {
		k := pxScale(s)
		if k != math.Floor(k) || k < 1 {
			t.Errorf("pxScale(%v) = %v, not a whole number >= 1", s, k)
		}
		if o := pxOff(k); o != math.Floor(o) || o < 1 {
			t.Errorf("pxOff(%v) = %v", k, o)
		}
	}
	if pxScale(1.3) != 1 || pxScale(1.6) != 1 || pxScale(2.2) != 2 || pxScale(3) != 3 {
		t.Error("unexpected snapping")
	}
}

// the window is always shown at a whole number of device pixels
func TestWholeWindowScale(t *testing.T) {
	for _, dsf := range []float64{1, 2} {
		for _, want := range []float64{0.8, 1, 1.38, 1.76, 2.4, 3.15} {
			sc := wholeScaleFor(want, dsf)
			if f := sc * dsf; f != math.Floor(f) || f < 1 {
				t.Errorf("dsf %v want %v: %v is not whole in device pixels", dsf, want, f)
			}
		}
	}
	if wholeScaleFor(1.38, 2) != 1 || wholeScaleFor(2.4, 1) != 2 {
		t.Error("unexpected scales")
	}
}
