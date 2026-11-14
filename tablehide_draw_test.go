package main

import (
	"math/rand"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// tableDesignBuf: the design picture of a table (no outline), to ask which pixels are cloth.
func tableDesignBuf(name string) *pixBuf {
	k := propKinds[name]
	b := newPix(k.W+2, k.H+2)
	b.ox, b.oy = 1, 1
	k.Draw(b, 0)
	return b
}

// painted: design pixel (x, y) of the table (anchor at bottom centre) holds colour.
func tablePainted(b *pixBuf, k *PropKind, tbl Vec, wx, wy float64) bool {
	x := int(wx-(tbl.X-float64(k.W)/2)) + 1
	y := int(wy-(tbl.Y-float64(k.H))) + 1
	if x < 0 || y < 0 || x >= b.w || y >= b.h {
		return false
	}
	return b.p[(y*b.w+x)*4+3] > 0
}

// A guest hiding under a table is drawn crouched INSIDE the lifted front of the cloth: the opening is all cloth (never
// over the top of the table, the hem or the floor), the picture fits in the opening and starts at the top of his head
// (before, a dark hole with two shoes in it looked like feet without a body).
func TestTableHideDrawsTheBodyInsideTheCloth(t *testing.T) {
	for name := range hideCloth {
		k := propKinds[name]
		if k == nil || !seatHideKinds[name] {
			t.Fatalf("%s is not a hiding table", name)
		}
		b := tableDesignBuf(name)
		tbl := &Prop{Name: name, K: k, X: 300, Y: 200}
		var slots []float64
		if name == "clothtable" {
			slots = []float64{tbl.X - tableHideDX, tbl.X + tableHideDX}
		} else {
			slots = []float64{tbl.X - 30, tbl.X - tableHideDX, tbl.X + tableHideDX, tbl.X + 30} // off the edge: clamped
		}
		for _, sx := range slots {
			op := hideOpening(tbl, sx)
			if op.H < 8 {
				t.Errorf("%s: the opening is only %.0f px tall (no room for a head)", name, op.H)
			}
			for y := op.Y; y < op.Y+op.H; y++ {
				for x := op.X; x < op.X+op.W; x++ {
					if !tablePainted(b, k, Vec{tbl.X, tbl.Y}, x, y) {
						t.Fatalf("%s slot %.0f: opening pixel %.0f,%.0f is not on the cloth", name, sx, x, y)
					}
				}
			}
			// the row over the opening is still the table (the top is not cut) and the opening stays above the floor
			if !tablePainted(b, k, Vec{tbl.X, tbl.Y}, op.X+op.W/2, op.Y-2) || op.Y+op.H > tbl.Y-1 {
				t.Errorf("%s: the opening %+v cuts the top of the table or reaches the floor", name, op)
			}
			for seed := int64(1); seed <= 12; seed++ {
				sp := genericSpec(rand.New(rand.NewSource(seed)))
				cs := &CharSet{Spec: &sp, cache: map[int]*ebiten.Image{}}
				ht := hideHeadTop(cs)
				pp := poseFor(AnimSitRest, 0)
				pp.Anim = AnimSitRest
				body := genChar(&sp, int(vFront), pp)
				if ht < 0 || ht >= charBase-10 || rowPainted(body, ht) || !rowPainted(body, ht+1) {
					t.Fatalf("seed %d: head top %d is not the top of the picture", seed, ht)
				}
				src, at := hideFigure(op, ht)
				if src.Dy() < 7 || src.Min.Y != ht {
					t.Errorf("%s seed %d: only %d rows of the guest shown (from %d)", name, seed, src.Dy(), src.Min.Y)
				}
				fig := Rect{at.X, at.Y, float64(src.Dx()), float64(src.Dy())}
				if fig.X < op.X || fig.Y < op.Y || fig.X+fig.W > op.X+op.W || fig.Y+fig.H > op.Y+op.H {
					t.Errorf("%s seed %d: the guest %+v spills out of the opening %+v", name, seed, fig, op)
				}
				// his body is really there: most rows of the shown part carry pixels of the picture
				n := 0
				for y := src.Min.Y; y < src.Max.Y; y++ {
					if rowPaintedIn(body, y, src.Min.X, src.Max.X) {
						n++
					}
				}
				if n < src.Dy()-1 {
					t.Errorf("%s seed %d: only %d of %d rows of the opening show the guest", name, seed, n, src.Dy())
				}
			}
		}
	}
}

func rowPainted(b *pixBuf, y int) bool { return rowPaintedIn(b, y, 0, b.w) }

func rowPaintedIn(b *pixBuf, y, x0, x1 int) bool {
	if y < 0 || y >= b.h {
		return false
	}
	for x := maxI(0, x0); x < minI(b.w, x1); x++ {
		if b.p[(y*b.w+x)*4+3] > 0 {
			return true
		}
	}
	return false
}

// The guests sitting BEHIND a table (the back chair of a café table, of a long table and of a cloth table) show their head and
// shoulders over the table top: the table never swallows the whole body.
func TestSeatBackSittersKeepHeadAndShoulders(t *testing.T) {
	type back struct {
		table string
		dy    float64 // the sitter's feet from the table's anchor
	}
	cases := []back{{"seattable", -8}, {"seatlong", -10}, {"clothtable", -17}}
	for _, c := range cases {
		k := propKinds[c.table]
		b := tableDesignBuf(c.table)
		top := -1
		for y := 0; y < b.h && top < 0; y++ {
			if rowPainted(b, y) {
				top = y - 1 // design row
			}
		}
		tableTop := -float64(k.H) + float64(top) // world, from the anchor
		for seed := int64(1); seed <= 12; seed++ {
			sp := genericSpec(rand.New(rand.NewSource(seed)))
			pp := poseFor(AnimSitRest, 0)
			pp.Anim = AnimSitRest
			body := genChar(&sp, int(vFront), pp)
			ht := 0
			for !rowPainted(body, ht) {
				ht++
			}
			head := c.dy - float64(charBase-ht) // world y of the top of his head
			if tableTop-head < 12 {
				t.Errorf("%s seed %d: only %.0f px of the sitter over the table top", c.table, seed, tableTop-head)
			}
		}
	}
}
