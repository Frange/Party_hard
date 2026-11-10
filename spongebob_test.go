package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"os"
	"strings"
	"testing"
)

// All four directions, every animation: the drawing stays inside the sprite box and
// cached frames cost no allocation.
func TestSpongeBobFourDirectionsInBox(t *testing.T) {
	for _, name := range []string{"bob_esponja", "fry", "stan", "kyle", "cartman", "kenny"} {
		sp := presetByName(name)(rand.New(rand.NewSource(1)))
		if sp.Name != name {
			t.Fatalf("%s: name %q", name, sp.Name)
		}
		checkCharBox(t, &sp)
	}
}

func bbox(b *pixBuf) (minX, minY, maxX, maxY int) {
	minX, minY, maxX, maxY = b.w, b.h, -1, -1
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			if b.p[(y*b.w+x)*4+3] != 0 {
				minX, maxX = minI(minX, x), maxI(maxX, x)
				minY, maxY = minI(minY, y), maxI(maxY, y)
			}
		}
	}
	return
}

// The sponge must stay inside the box with a 1px margin. The other characters use the stock
// body, so their extras may not grow the stock silhouette by more than the margin.
func checkCharBox(t *testing.T, sp *CharSpec) {
	for vw := 0; vw < 3; vw++ {
		for a := 0; a < AnimCount+len(gaitKinds)+4; a++ {
			for f := 0; f < animFrameCount(a); f++ {
				pp := poseFor(a, f)
				pp.Anim, pp.F = a, f
				pp.Hold = HoldNone // held props are drawn by drawHeld and may stick out
				b := genChar(sp, vw, pp)
				minX, minY, maxX, maxY := bbox(b)
				if maxX < 0 {
					t.Fatalf("empty %s frame vw=%d anim=%d f=%d", sp.Name, vw, a, f)
				}
				lo, hi := 1, 3
				if !isSponge(sp) {
					noCharExtras = true
					sx0, sy0, sx1, sy1 := bbox(genChar(sp, vw, pp))
					noCharExtras = false
					if minX < sx0-lo || minY < sy0-4 || maxX > sx1+hi || maxY > sy1 {
						t.Errorf("%s vw=%d anim=%d f=%d extras grow the box: %d,%d-%d,%d vs stock %d,%d-%d,%d", sp.Name, vw, a, f, minX, minY, maxX, maxY, sx0, sy0, sx1, sy1)
					}
					continue
				}
				if minX < 1 || minY < 1 || maxX > b.w-2 || maxY > b.h-2 {
					t.Errorf("%s vw=%d anim=%d f=%d outside the box: %d,%d-%d,%d", sp.Name, vw, a, f, minX, minY, maxX, maxY)
				}
			}
		}
	}
}

func TestSpongeBobNoAllocOnCachedFrames(t *testing.T) {
	cs := &CharSet{Spec: func() *CharSpec { s := presetByName("bob_esponja")(rand.New(rand.NewSource(1))); return &s }(), cache: map[int]*ebiten.Image{}}
	for _, d := range []Dir{DirUp, DirDown, DirLeft, DirRight} {
		cs.img(AnimWalk, d, 0)
		cs.img(AnimHop, d, 1)
	}
	n := testing.AllocsPerRun(50, func() {
		for _, d := range []Dir{DirUp, DirDown, DirLeft, DirRight} {
			cs.img(AnimWalk, d, 0)
			cs.img(AnimHop, d, 1)
		}
	})
	if n != 0 {
		t.Fatalf("allocs per run = %v", n)
	}
}

// PH_SPONGE_SHEET=/path.png dumps a x4 sheet: rows = front/side/back, columns = poses.
func TestSpongeBobSheet(t *testing.T) {
	out := os.Getenv("PH_SPONGE_SHEET")
	if out == "" {
		t.Skip("set PH_SPONGE_SHEET to dump the sheet")
	}
	names := []string{"bob_esponja", "fry", "stan", "kyle", "cartman", "kenny"}
	if n := os.Getenv("PH_SHEET_NAMES"); n != "" {
		names = strings.Split(n, ",")
	}
	type pf struct{ a, f int }
	cols := []pf{{AnimIdle, 0}, {AnimHop, 0}, {AnimHop, 1}, {AnimHop, 2}, {AnimHop, 3}, {AnimRun, 1}, {AnimRun, 4}, {AnimStab, 1}, {AnimDrink, 2}}
	const k = 4
	img := image.NewRGBA(image.Rect(0, 0, len(cols)*charW*k, 3*len(names)*charH*k))
	for x := 0; x < img.Bounds().Dx(); x++ {
		for y := 0; y < img.Bounds().Dy(); y++ {
			img.Set(x, y, color.RGBA{200, 200, 205, 255})
		}
	}
	for ni, name := range names {
		sp := presetByName(name)(rand.New(rand.NewSource(1)))
		for vw := 0; vw < 3; vw++ {
			for ci, c := range cols {
				pp := poseFor(c.a, c.f)
				pp.Anim, pp.F = c.a, c.f
				b := genChar(&sp, vw, pp).outlinedSel(0.42)
				for y := 0; y < b.h; y++ {
					for x := 0; x < b.w; x++ {
						i := (y*b.w + x) * 4
						if b.p[i+3] == 0 {
							continue
						}
						col := color.RGBA{b.p[i], b.p[i+1], b.p[i+2], 255}
						for dy := 0; dy < k; dy++ {
							for dx := 0; dx < k; dx++ {
								img.Set((ci*charW+x)*k+dx, ((ni*3+vw)*charH+y)*k+dy, col)
							}
						}
					}
				}
			}
		}
	}
	f, _ := os.Create(out)
	defer f.Close()
	png.Encode(f, img)
}
