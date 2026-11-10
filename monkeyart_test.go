package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

var miNames = []string{"guybrush", "lechuck"}

// the reference looks: Guybrush in white shirt, brown vest and trousers, black boots and no weapon;
// LeChuck green, black beard, red coat, dark maroon trousers, big and tall, no cape or stock hat.
func TestMonkeyLeadsLook(t *testing.T) {
	g := ownSpec("guybrush")
	if g.Hair2 != HairPony || g.Carry != HoldNone || g.Top != TopLong || g.Shirt != hx(0xf4f2ea) || g.Hat != HatNone {
		t.Errorf("guybrush spec: %+v", g)
	}
	l := ownSpec("lechuck")
	if l.Skin.G <= l.Skin.R || l.Beard != BeardFull || l.Hair != hx(0x141418) || l.Shirt.R < 0x90 || l.Cape.A != 0 || l.Hat != HatNone {
		t.Errorf("lechuck spec: %+v", l)
	}
	if l.Body != BodyBig || l.Stature >= 0 {
		t.Errorf("lechuck should be bigger and taller: body %d stature %d", l.Body, l.Stature)
	}
	if miKind(&g) != miGuybrush || miKind(&l) != miLeChuck {
		t.Error("own layers not picked")
	}
}

// the own layers really paint (differ from the stock body) in every view, stay inside the box,
// and animate: walking frames differ (legs, ponytail, plumes, coat tails).
func TestMonkeyLeadsLayers(t *testing.T) {
	count := func(b *pixBuf) (n, edge int) {
		for y := 0; y < b.h; y++ {
			for x := 0; x < b.w; x++ {
				if b.p[(y*b.w+x)*4+3] == 0 {
					continue
				}
				n++
				if x == 0 || y == 0 || x == b.w-1 || y == b.h-1 {
					edge++
				}
			}
		}
		return
	}
	anims := []int{AnimIdle, AnimWalk, AnimRun, AnimStab, AnimPunch, AnimDance3, AnimPanic, AnimHop}
	for _, name := range miNames {
		sp := ownSpec(name)
		twin := sp
		twin.Name = "twin"
		for vw := 0; vw < 3; vw++ {
			if string(renderChar(&sp, vw, ownPose(AnimIdle, 0)).p) == string(renderChar(&twin, vw, ownPose(AnimIdle, 0)).p) {
				t.Errorf("%s view %d: own layers paint nothing", name, vw)
			}
			if string(renderChar(&sp, vw, ownPose(AnimWalk, 0)).p) == string(renderChar(&sp, vw, ownPose(AnimWalk, 1)).p) {
				t.Errorf("%s view %d: walk frames 0 and 1 are identical", name, vw)
			}
			for _, a := range anims {
				for f := 0; f < animFrameCount(a); f++ {
					n, edge := count(renderChar(&sp, vw, ownPose(a, f)))
					_, tEdge := count(renderChar(&twin, vw, ownPose(a, f)))
					if edge > tEdge+2 {
						t.Errorf("%s vw %d anim %d frame %d: %d pixels on the edge (stock %d)", name, vw, a, f, edge, tEdge)
					}
					if n < 120 {
						t.Errorf("%s vw %d anim %d frame %d: only %d pixels", name, vw, a, f, n)
					}
				}
			}
		}
	}
	// LeChuck is visibly bigger than Guybrush (taller hat, wider coat)
	g, l := ownSpec("guybrush"), ownSpec("lechuck")
	ng, _ := count(renderChar(&g, vFront, ownPose(AnimIdle, 0)))
	nl, _ := count(renderChar(&l, vFront, ownPose(AnimIdle, 0)))
	if nl <= ng*5/4 {
		t.Errorf("lechuck %d px vs guybrush %d px: not big enough", nl, ng)
	}
}

// MONKEY_SHEET=dir go test -run TestMonkeyLeadsSheet writes a contact sheet to review the poses.
func TestMonkeyLeadsSheet(t *testing.T) {
	dir := os.Getenv("MONKEY_SHEET")
	if dir == "" {
		t.Skip("set MONKEY_SHEET to a directory to dump the sheet")
	}
	os.MkdirAll(dir, 0o755)
	type pick struct{ a, f, vw int }
	picks := []pick{{AnimIdle, 0, vFront}, {AnimWalk, 1, vFront}, {AnimWalk, 4, vFront}, {AnimIdle, 0, vBack}, {AnimWalk, 2, vBack}, {AnimIdle, 0, vSide}, {AnimWalk, 1, vSide}, {AnimWalk, 4, vSide}, {AnimRun, 2, vSide}, {AnimStab, 1, vFront}}
	const z = 8
	big := image.NewRGBA(image.Rect(0, 0, len(picks)*charW*z, len(miNames)*charH*z))
	for i := 0; i < len(big.Pix); i += 4 {
		big.Pix[i], big.Pix[i+1], big.Pix[i+2], big.Pix[i+3] = 0x50, 0x58, 0x70, 255
	}
	for ni, name := range miNames {
		sp := ownSpec(name)
		for pi, pk := range picks {
			b := renderChar(&sp, pk.vw, ownPose(pk.a, pk.f))
			for y := 0; y < b.h; y++ {
				for x := 0; x < b.w; x++ {
					i := (y*b.w + x) * 4
					if b.p[i+3] == 0 {
						continue
					}
					c := color.RGBA{b.p[i], b.p[i+1], b.p[i+2], 255}
					for dy := 0; dy < z; dy++ {
						for dx := 0; dx < z; dx++ {
							big.SetRGBA((pi*charW+x)*z+dx, (ni*charH+y)*z+dy, c)
						}
					}
				}
			}
		}
	}
	f, err := os.Create(filepath.Join(dir, "monkey_leads.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	png.Encode(f, big)
}
