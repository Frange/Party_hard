package main

import (
	"math"
	"testing"
)

func opaque(b *pixBuf) int {
	n := 0
	for i := 3; i < len(b.p); i += 4 {
		if b.p[i] > 0 {
			n++
		}
	}
	return n
}

func same(a, b *pixBuf) bool {
	if a.w != b.w || a.h != b.h {
		return false
	}
	for i := range a.p {
		if a.p[i] != b.p[i] {
			return false
		}
	}
	return true
}

// every livery has a front and a rear view, both painted and different from each other
func TestHeliViews(t *testing.T) {
	for i, l := range heliLivery {
		f := genHeliView(l[0], l[1], int(l[2]), true)
		r := genHeliView(l[0], l[1], int(l[2]), false)
		if opaque(f) < 400 || opaque(r) < 400 {
			t.Fatalf("livery %d: empty view (%d / %d px)", i, opaque(f), opaque(r))
		}
		if same(f, r) {
			t.Fatalf("livery %d: front and rear views are identical", i)
		}
		side := genHeliPal(l[0], l[1], int(l[2]))
		if side.w == f.w && same(side, f) {
			t.Fatalf("livery %d: the front view is the side sprite", i)
		}
	}
	// the liveries differ from each other
	a := genHeliView(heliLivery[0][0], heliLivery[0][1], 0, true)
	b := genHeliView(heliLivery[3][0], heliLivery[3][1], 3, true)
	if same(a, b) {
		t.Fatal("police and SWAT helicopters look the same")
	}
}

// boats and shuttles: front, rear and side are three different pictures, in both lengths and every style
func TestTransitViews(t *testing.T) {
	body, trim := hx(0x2a50c0), hx(0xf4f4f8)
	for style := 0; style < crCount; style++ {
		for _, long := range []bool{false, true} {
			for _, sea := range []bool{true, false} {
				f, r := genShipView(body, trim, long, true, style), genShipView(body, trim, long, false, style)
				side := genShip(body, trim, long, style)
				if sea {
					f, r, side = genBoatView(body, trim, long, true, style), genBoatView(body, trim, long, false, style), genBoat(body, trim, long, style, 0)
				}
				if opaque(f) < 300 || opaque(r) < 300 {
					t.Fatalf("style %d sea=%v long=%v: empty view", style, sea, long)
				}
				if same(f, r) || same(f, side) || same(r, side) {
					t.Fatalf("style %d sea=%v long=%v: two of the three views are identical", style, sea, long)
				}
				if f.w >= side.w { // seen from the bow or the stern it is narrower than long
					t.Fatalf("style %d sea=%v long=%v: front view (%d) not narrower than the side (%d)", style, sea, long, f.w, side.w)
				}
			}
		}
	}
}

// a vehicle prepared for a horizontal lane drives down or up a vertical lane, parks and leaves
func TestVerticalLane(t *testing.T) {
	for _, d := range []float64{1, -1} {
		def := &LevelDef{Transit: "sea", CarFrom: Vec{300, 40}, CarTo: Vec{300, 440}}
		if d < 0 {
			def.CarFrom, def.CarTo = def.CarTo, def.CarFrom
		}
		if !laneVertical(def) || laneDir(def) != d {
			t.Fatalf("dir %v: lane not recognised (vertical=%v dir=%v)", d, laneVertical(def), laneDir(def))
		}
		// as dispatch builds it: second vehicle, one slot back
		v := &Vehicle{kind: 0, x: def.CarFrom.X + 40, y: def.CarFrom.Y + 8, tx: def.CarTo.X - 100, ty: def.CarFrom.Y + 8}
		laneRemap(def, v)
		if !v.lane || v.vdir != d {
			t.Fatal("not remapped")
		}
		if math.Abs(v.ty-(def.CarTo.Y-d*100)) > 0.1 {
			t.Fatalf("parking spot %v, want one slot before the end of the lane %v", v.ty, def.CarTo.Y-d*100)
		}
		start := v.y
		if (v.ty-start)*d <= 0 {
			t.Fatal("it starts after its parking spot")
		}
		for i := 0; v.state == 0; i++ {
			laneStep(def, v, 1.0/60)
			if i > 1000 {
				t.Fatal("never arrives")
			}
			if v.y*d < start*d-0.01 {
				t.Fatal("moves the wrong way while arriving")
			}
		}
		if v.y != v.ty || v.state != 1 {
			t.Fatal("did not stop at its spot")
		}
		v.state = 2
		for i := 0; ; i++ {
			if laneStep(def, v, 1.0/60) {
				break
			}
			if i > 1000 {
				t.Fatal("never leaves")
			}
			if v.y*d > v.ty*d+0.01 {
				t.Fatal("moves the wrong way while leaving")
			}
		}
		// a horizontal lane is left alone
		h := &LevelDef{Transit: "sea", CarFrom: Vec{1000, 400}, CarTo: Vec{500, 400}}
		hv := &Vehicle{x: 1040, y: 400, tx: 500, ty: 400}
		laneRemap(h, hv)
		if hv.lane || hv.x != 1040 {
			t.Fatal("a horizontal lane was remapped")
		}
	}
}

// the stock helicopter is recognised so that the liveries keep working with the exported copy
func TestStockHeliSprite(t *testing.T) {
	a := genHeli().scaled(wk).outlinedSel(0.45)
	b := genHeli().scaled(wk).outlinedSel(0.45)
	if !sameSprite(a, b) {
		t.Fatal("the stock helicopter does not match itself")
	}
	for i := 0; i < len(b.p); i += 4 { // a redrawn helicopter: every colour changed
		b.p[i], b.p[i+1] = 255-b.p[i], 255-b.p[i+1]
	}
	if sameSprite(a, b) {
		t.Fatal("a redrawn helicopter is taken for the stock one")
	}
	if c := genHeli().scaled(wk).outlinedSel(0.45); !sameSprite(a, c) {
		t.Fatal("identical sprites differ")
	}
	// the copy exported into assets/ by an older build must still count as the stock one
	if oa, err := loadObjAsset("assets/sprites/objetos/heli"); err == nil && !sameSprite(oa.Frames[0], a) {
		t.Fatal("the exported helicopter in assets/ is not recognised as the stock one")
	}
}
