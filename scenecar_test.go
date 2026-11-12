package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"image/color"
	"os"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// The car of the interlude is the game's car model: same silhouette (alpha mask) as the "car" prop
// sprites, pixel for pixel, and painted through the vehicle art kit.
func TestSceneCarIsGameCarModel(t *testing.T) {
	before := vaKitUse.build
	got := sceneCarPix(0, -1)
	if vaKitUse.build == before {
		t.Fatal("the scene car must be painted with the vehicle art kit (vaBuild)")
	}
	ref := newPix(sceneCarW, sceneCarH)
	ref.ox, ref.oy = 1, 1
	propKinds["car"].Draw(ref, 0) // the plain red car of the game
	ref = stylizeSprite(ref.scaled(wk).detail(1, 0.035).outlinedSel(0.45))
	if got.w != ref.w || got.h != ref.h {
		t.Fatalf("size %dx%d, the game car is %dx%d", got.w, got.h, ref.w, ref.h)
	}
	diff, solid := 0, 0
	for i := 3; i < len(got.p); i += 4 {
		if (got.p[i] > 0) != (ref.p[i] > 0) {
			diff++
		}
		if got.p[i] > 0 {
			solid++
		}
	}
	if diff != 0 || solid < 1200 {
		t.Fatalf("the scene car's silhouette differs from the game car in %d px (%d px solid)", diff, solid)
	}
	// and the body is the same paint layer: the turquoise glass of the kit is there
	glass := 0
	for i := 0; i < len(got.p); i += 4 {
		if got.p[i+3] == 255 && got.p[i+2] > got.p[i] && got.p[i+1] > got.p[i] {
			glass++
		}
	}
	if glass < 40 {
		t.Fatalf("only %d px of window glass", glass)
	}
}

// Every door position and wheel frame renders; the door changes only the door, the wheels turn.
func TestSceneCarFramesAndDoor(t *testing.T) {
	closed := sceneCarPix(0, -1)
	for d := 0; d <= sceneCarDoors; d++ {
		for s := -1; s < sceneCarSpinFr; s++ {
			if im := sceneCarImg(d, s); im == nil || im.Bounds().Dx() != sceneCarW || im.Bounds().Dy() != sceneCarH {
				t.Fatalf("door %d spin %d: bad image", d, s)
			}
		}
	}
	open := sceneCarPix(sceneCarDoors, -1)
	changed := 0
	for y := 0; y < closed.h; y++ {
		for x := 0; x < closed.w; x++ {
			i := (y*closed.w + x) * 4
			if closed.p[i] != open.p[i] || closed.p[i+1] != open.p[i+1] || closed.p[i+2] != open.p[i+2] {
				changed++
				if x < 18 || x > 36 || y < 8 || y > 28 { // the front door (design x 19..33, +1 margin)
					t.Fatalf("opening the door changed pixel (%d,%d), outside the door", x, y)
				}
			}
		}
	}
	if changed < 60 {
		t.Fatalf("the open door changed only %d px", changed)
	}
	s0, s2 := sceneCarPix(0, 0), sceneCarPix(0, 2)
	spinDiff := 0
	for i := range s0.p {
		if s0.p[i] != s2.p[i] {
			spinDiff++
		}
	}
	if spinDiff < 4 {
		t.Fatal("the wheel frames are all the same: the wheels do not turn")
	}
}

// The interlude draws that car (nose right, at 3x) and never paints its own any more; the police car of
// the memories is the animated police car of the game.
func TestInterludeUsesSceneCar(t *testing.T) {
	src, err := os.ReadFile("interlude.go")
	if err != nil {
		t.Fatal(err)
	}
	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, "interlude.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	var body string
	for _, d := range f.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Name.Name == "interCar" && fd.Recv != nil {
			body = string(src[fs.Position(fd.Body.Pos()).Offset:fs.Position(fd.Body.End()).Offset])
		}
	}
	if body == "" {
		t.Fatal("interCar not found")
	}
	if !strings.Contains(body, "drawSceneCar(") {
		t.Error("interCar must draw the car with drawSceneCar (scenecar.go)")
	}
	for _, old := range []string{"vaTyre", "vaRimC", "vaChromeC", "vaHeadC", "body2"} {
		if strings.Contains(body, old) {
			t.Errorf("interCar paints the car by hand again (%s): use drawSceneCar", old)
		}
	}
	if !strings.Contains(string(src), "emergencyImg(0, unitPolice") || strings.Contains(string(src), `props["car"]`) {
		t.Error("the police car of the memories must be emergencyImg, not a car prop with a painted light")
	}

	// drawn on screen: the right part of the car is exactly the scene car sprite, 1P and 2P, and
	// the whole timeline draws (closed, opening, open, closing, leaving)
	g := auditGame(t)
	oldPx := px
	if px == nil {
		px = ebiten.NewImage(1, 1)
		px.Fill(color.White)
	}
	t.Cleanup(func() { px = oldPx; g.inter = nil })
	for _, two := range []bool{false, true} {
		in := &interState{kind: interCar, two: two, did: map[string]bool{}}
		dst := ebiten.NewImage(screenW, screenH)
		for tt := 0.0; tt < interLen; tt += 0.2 {
			in.t = tt
			g.interVehicleAndHero(dst, in)
		}
		use := sceneCarUse
		in.t = itStop
		g.interVehicleAndHero(dst, in)
		if sceneCarUse != use+1 {
			t.Fatalf("two=%v: the interlude did not draw the scene car at the stop", two)
		}
		// at the stop the car stands where the old one stood: wheels on the road line, crisp 3x, inside the screen
		op := sceneCarOp(interVehicleX(itStop), interGround+46)
		x0, y0 := op.GeoM.Apply(sceneCarW, 0)
		x1, y1 := op.GeoM.Apply(0, sceneCarH)
		if x0 != float64(int(x0)) || y0 != float64(int(y0)) || x1-x0 != sceneCarW*sceneCarK || y1-y0 != sceneCarH*sceneCarK {
			t.Fatalf("the car is not at an integer 3x placement: (%v,%v)-(%v,%v)", x0, y0, x1, y1)
		}
		if x0 < 16 || x1 > screenW-16 || y1 < interGround+40 || y1 > interGround+60 {
			t.Fatalf("the parked car is at (%v,%v)-(%v,%v)", x0, y0, x1, y1)
		}
	}
}
