package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// the characters with their own drawing layers (vader.go, foxes.go)
var ownLayerNames = []string{"vader", "sonic", "tails"}

func ownSpec(name string) CharSpec { return presetByName(name)(newRand(5)) }

func ownPose(a, f int) Pose {
	p := poseFor(a, f)
	p.Anim, p.F = a, f
	return p
}

// Each one draws in the four directions and in every animation inside its box:
// his own layers never add pixels on the outer ring.
func TestVaderDrawsInsideItsBox(t *testing.T) {
	anims := []int{AnimIdle, AnimWalk, AnimRun, AnimStab, AnimGlide, AnimPunch, AnimDance3, AnimDance7, AnimPanic}
	for _, name := range ownLayerNames {
		sp := ownSpec(name)
		twin := sp
		twin.Name = "twin"
		count := func(s *CharSpec, vw, a, f int) (n, edge int) {
			b := renderChar(s, vw, ownPose(a, f))
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
		for vw := 0; vw < 3; vw++ {
			for _, a := range anims {
				for f := 0; f < animFrameCount(a); f++ {
					n, edge := count(&sp, vw, a, f)
					_, tEdge := count(&twin, vw, a, f)
					if edge > tEdge {
						t.Errorf("%s vw %d anim %d frame %d: %d pixels on the edge (generic body: %d)", name, vw, a, f, edge, tEdge)
					}
					if n < 120 {
						t.Errorf("%s vw %d anim %d frame %d: only %d pixels", name, vw, a, f, n)
					}
				}
			}
		}
	}
}

// Frames are cached: drawing them again costs no allocation.
func TestVaderNoAllocPerFrame(t *testing.T) {
	for _, name := range ownLayerNames {
		sp := ownSpec(name)
		cs := newCharSet(&sp)
		pass := func() {
			for _, d := range []Dir{DirUp, DirDown, DirLeft, DirRight} {
				for f := 0; f < 6; f++ {
					cs.img(AnimWalk, d, float64(f))
					cs.img(AnimRun, d, float64(f))
					cs.img(AnimIdle, d, float64(f))
					cs.img(AnimStab, d, float64(f))
					cs.img(AnimDance2, d, float64(f))
				}
			}
		}
		pass()
		if n := testing.AllocsPerRun(30, pass); n != 0 {
			t.Errorf("%s: allocs per pass: %v", name, n)
		}
	}
}

// The details move: panel lights, saber cut, spinning tails, fluttering quills, running feet.
func TestVaderAnimatesItsDetails(t *testing.T) {
	render := func(name string, a, f, vw int) string {
		sp := ownSpec(name)
		return string(renderChar(&sp, vw, ownPose(a, f)).p)
	}
	for _, c := range []struct {
		name string
		a    int
		f, g int
	}{
		{"vader", AnimIdle, 0, 1}, {"vader", AnimRun, 0, 2}, {"vader", AnimStab, 0, 1},
		{"sonic", AnimRun, 0, 1}, {"sonic", AnimWalk, 0, 1},
		{"tails", AnimRun, 0, 1}, {"tails", AnimDance0, 0, 1}, {"tails", AnimWalk, 0, 2},
	} {
		for _, vw := range []int{vFront, vBack, vSide} {
			if render(c.name, c.a, c.f, vw) == render(c.name, c.a, c.g, vw) {
				t.Errorf("%s anim %d view %d: frames %d and %d are identical", c.name, c.a, vw, c.f, c.g)
			}
		}
	}
}

// VADER_SHEET=dir go test -run TestVaderSheet writes a contact sheet for reviewing the poses.
func TestVaderSheet(t *testing.T) {
	dir := os.Getenv("VADER_SHEET")
	if dir == "" {
		t.Skip("set VADER_SHEET to a directory to dump the sheet")
	}
	os.MkdirAll(dir, 0o755)
	type pick struct{ a, f, vw int }
	picks := []pick{{AnimIdle, 0, vFront}, {AnimIdle, 1, vFront}, {AnimIdle, 0, vBack}, {AnimIdle, 0, vSide}, {AnimRun, 1, vFront}, {AnimRun, 2, vSide}, {AnimStab, 1, vFront}, {AnimDance3, 1, vFront}}
	const z = 6
	big := image.NewRGBA(image.Rect(0, 0, len(picks)*charW*z, len(ownLayerNames)*charH*z))
	for i := 0; i < len(big.Pix); i += 4 {
		big.Pix[i], big.Pix[i+1], big.Pix[i+2], big.Pix[i+3] = 0x70, 0x70, 0x78, 255
	}
	for ni, name := range ownLayerNames {
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
	g, err := os.Create(filepath.Join(dir, "own_big.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	png.Encode(g, big)
}
