package main

import (
	"image/color"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

// Full round trip of the data the editor produces: paint, save, load, animate, build a level.
func TestAssetRoundTrip(t *testing.T) {
	root := t.TempDir()

	// ---- character painted pixel by pixel
	ca := newCharAsset("pruebas", "people", 24, 42)
	pt := newPainter()
	pt.usePart = true
	pt.col = color.RGBA{200, 50, 50, 255}
	for v := 0; v < 3; v++ {
		b := ca.Views[v]
		pt.part = PartHead
		pt.rectFill(b, 10, 10, 14, 17)
		pt.part = PartTorso
		pt.rectFill(b, 9, 18, 15, 27)
		pt.part = PartArmL
		pt.rectFill(b, 7, 18, 8, 28)
		pt.part = PartArmR
		pt.rectFill(b, 16, 18, 17, 28)
		pt.part = PartLegL
		pt.rectFill(b, 10, 28, 11, 38)
		pt.part = PartLegR
		pt.rectFill(b, 13, 28, 14, 38)
	}
	if err := saveCharAsset(root, ca); err != nil {
		t.Fatal(err)
	}
	ca2, err := loadCharAsset(filepath.Join(spritesDir(root, "people"), "pruebas"))
	if err != nil {
		t.Fatal(err)
	}
	cs := newRigCharSet(ca2)
	for a := 0; a < AnimCount; a++ {
		for v := 0; v < 3; v++ {
			for f := 0; f < animFrames[a]; f++ {
				fb := cs.rigFrame(a, []int{vFront, vSide, vBack}[v], f)
				if fb == nil || (a != AnimLying && a != AnimSleep && fb.w != 24) {
					t.Fatalf("bad frame anim %d view %d frame %d", a, v, f)
				}
			}
		}
	}

	// ---- object
	oa := &ObjAsset{Meta: ObjMeta{Name: "mesa_test", Category: "furniture", AnchorX: 8, AnchorY: 15, FootX: 0, FootY: 10, FootW: 16, FootH: 5, Solid: true, Hide: true},
		Frames: []*pixBuf{newPix(16, 16)}}
	oa.Frames[0].rect(0, 0, 16, 16, rgb(0x884422))
	if err := saveObjAsset(root, oa); err != nil {
		t.Fatal(err)
	}
	ob2, err := loadObjAsset(filepath.Join(spritesDir(root, "objetos"), "mesa_test"))
	if err != nil || ob2.Meta.Hide != true || len(ob2.Frames) != 1 {
		t.Fatal("object round trip failed", err)
	}

	// ---- level
	la := newLevelAsset("nivel_test")
	for y := 40; y < 50; y++ {
		row := []byte(la.Meta.Solid[y])
		for x := 40; x < 100; x++ {
			row[x] = '#'
		}
		la.Meta.Solid[y] = string(row)
	}
	for y := 10; y < 30; y++ {
		row := []byte(la.Meta.Zone[y])
		for x := 20; x < 120; x++ {
			row[x] = 'r'
		}
		la.Meta.Zone[y] = string(row)
	}
	la.Meta.Guards = [][]LevelPoint{{{100, 100}, {200, 100}}}
	if err := saveLevelAsset(root, la); err != nil {
		t.Fatal(err)
	}
	la2, err := loadLevelAsset(filepath.Join(backgroundsDir(root), "nivel_test"))
	if err != nil {
		t.Fatal(err)
	}
	if la2.Meta.Solid[45][50] != '#' || la2.Meta.Zone[20][30] != 'r' {
		t.Fatal("level logic lost")
	}
	rects := mergeCells(la2.Meta.Solid, func(c byte) bool { return c == '#' })
	if len(rects) != 1 || rects[0].W != 60 || rects[0].H != 10 {
		t.Fatalf("merge gave %v", rects)
	}
	_ = os.Stdout
}

func TestPainterTools(t *testing.T) {
	b := newPix(20, 20)
	b.pid = make([]uint8, 400)
	pt := newPainter()
	pt.usePart = true
	pt.part = PartArmL
	pt.mirror = true
	pt.col = color.RGBA{1, 2, 3, 255}
	pt.setPx(b, 2, 3, false, true)
	if b.pid[3*20+2] != PartArmL || b.pid[3*20+17] != PartArmR {
		t.Fatal("mirror did not swap parts")
	}
	pt.mirror = false
	pt.rectFill(b, 5, 5, 8, 8)
	pt.col = color.RGBA{9, 9, 9, 255}
	pt.flood(b, 6, 6)
	if b.p[(6*20+6)*4] != 9 {
		t.Fatal("flood failed")
	}
}

func TestAutoPartsRig(t *testing.T) {
	spec := genericSpec(rand.New(rand.NewSource(4)))
	ca := charAssetFromSpec(spec, "auto", "people", "")
	for i, v := range ca.Views {
		for k := range v.pid {
			v.pid[k] = 0
		}
		autoParts(v, i == 1)
	}
	rig := buildRig(ca)
	for a := 0; a < AnimCount; a++ {
		for f := 0; f < animFrames[a]; f++ {
			for _, vw := range []int{vFront, vSide, vBack} {
				if rig.render(vw, poseFor(a, f)) == nil {
					t.Fatal("nil frame")
				}
			}
		}
	}
	for id := 1; id <= 4; id++ {
		if !rig.V[0].L[id].ok {
			t.Fatalf("front view lacks part %d after auto tagging", id)
		}
	}
}
