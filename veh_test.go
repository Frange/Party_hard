package main

import "testing"

// every vehicle sprite with a visible headlight must be listed in the right nose side
func TestVehicleNoseSide(t *testing.T) {
	for n, k := range propKinds {
		if k.Draw == nil || !(k.Ambient > 0 || n == "car" || n == "motorbike") {
			continue
		}
		var sx, cnt float64
		b := newPix(k.W, k.H)
		k.Draw(b, 0)
		for y := 0; y < b.h; y++ {
			for x := 0; x < b.w; x++ {
				if c := b.get(x, y); c.A > 0 && c.R == 0xff && c.G == 0xf0 && c.B == 0xb0 {
					sx += float64(x)
					cnt++
				}
			}
		}
		if cnt == 0 {
			continue
		}
		left := sx/cnt < float64(k.W)/2
		if left != vehNoseLeft[n] {
			t.Errorf("%s: headlight on the %v side but vehNoseLeft=%v", n, map[bool]string{true: "left", false: "right"}[left], vehNoseLeft[n])
		}
	}
	// a truck (nose right) driving right is not mirrored, driving left it is; a car is the other way round
	if vehMirror("boxtruck", true) || !vehMirror("boxtruck", false) || !vehMirror("car", true) || vehMirror("car", false) {
		t.Fatal("vehMirror wrong")
	}
}

// every road vehicle has a profile, a front and a rear view, and the three differ
func TestVehicleViews(t *testing.T) {
	for n, k := range propKinds {
		if k.Draw == nil || !(k.Ambient > 0 || n == "car" || n == "motorbike") {
			continue
		}
		d := vehViewDefs[n]
		if d == nil {
			t.Errorf("%s: no front/rear views", n)
			continue
		}
		vars := k.Vars
		if vars == 0 {
			vars = 1
		}
		for v := 0; v < vars; v++ {
			side := newPix(k.W, k.H)
			k.Draw(side, v)
			f, b := newPix(d.w, d.h), newPix(d.w, d.h)
			d.draw(f, v, true)
			d.draw(b, v, false)
			same := func(x, y *pixBuf) bool {
				if x.w != y.w || x.h != y.h {
					return false
				}
				for i := range x.p {
					if x.p[i] != y.p[i] {
						return false
					}
				}
				return true
			}
			opaque := func(x *pixBuf) int {
				n := 0
				for i := 3; i < len(x.p); i += 4 {
					if x.p[i] > 0 {
						n++
					}
				}
				return n
			}
			if opaque(f) < 100 || opaque(b) < 100 {
				t.Errorf("%s v%d: empty view", n, v)
			}
			if same(f, b) || same(f, side) || same(b, side) {
				t.Errorf("%s v%d: views are not different", n, v)
			}
		}
	}
}
