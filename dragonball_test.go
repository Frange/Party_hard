package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var dbSlugs = []string{"goku", "gohan", "vegeta", "krilin", "piccolo", "bulma", "maestro_roshi", "cell"}

// DB_SHEET=dir [DB_TAG=antes] go test -run TestDragonBallSheet writes an enlarged sheet
// (front, side, back standing; front and side walking; front dancing).
func TestDragonBallSheet(t *testing.T) {
	dir := os.Getenv("DB_SHEET")
	if dir == "" {
		t.Skip("set DB_SHEET to a directory to dump the sheet")
	}
	os.MkdirAll(dir, 0o755)
	type cell struct{ vw, anim, f int }
	cols := []cell{{vFront, AnimIdle, 0}, {vSide, AnimIdle, 0}, {vBack, AnimIdle, 0}, {vFront, AnimWalk, 1}, {vSide, AnimRun, 2}, {vBack, AnimWalk, 3}, {vFront, AnimDance3, 1}}
	z := 5
	slugs := dbSlugs
	if os.Getenv("DB_ONLY") != "" {
		slugs, z = strings.Split(os.Getenv("DB_ONLY"), ","), 10
	}
	cw, ch := charW*z+4, charH*z+4
	big := image.NewRGBA(image.Rect(0, 0, len(cols)*cw, len(slugs)*ch))
	for i := 0; i < len(big.Pix); i += 4 {
		big.Pix[i], big.Pix[i+1], big.Pix[i+2], big.Pix[i+3] = 0x6a, 0x6a, 0x74, 255
	}
	for r, slug := range slugs {
		sp := presetByName(slug)(newRand(5))
		for c, cc := range cols {
			pp := poseFor(cc.anim, cc.f)
			pp.Anim, pp.F = cc.anim, cc.f
			b := renderChar(&sp, cc.vw, pp)
			for y := 0; y < b.h; y++ {
				for x := 0; x < b.w; x++ {
					i := (y*b.w + x) * 4
					if b.p[i+3] == 0 {
						continue
					}
					col := color.RGBA{b.p[i], b.p[i+1], b.p[i+2], 255}
					for dy := 0; dy < z; dy++ {
						for dx := 0; dx < z; dx++ {
							big.SetRGBA(c*cw+x*z+dx, r*ch+y*z+dy, col)
						}
					}
				}
			}
		}
	}
	f, err := os.Create(filepath.Join(dir, fmt.Sprintf("%sdragonball.png", os.Getenv("DB_TAG"))))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	png.Encode(f, big)
}

// Every Dragon Ball guest is drawn by dragonball_art.go, inside its box in the three views (no more
// pixels on the outer ring than the plain body), its details really show, and walking changes the frames.
func TestDragonBallDrawsInsideAndWalks(t *testing.T) {
	anims := []int{AnimIdle, AnimWalk, AnimRun, AnimStab, AnimDance3, AnimPanic}
	for _, slug := range dbSlugs {
		f := presetByName(slug)
		if f == nil {
			t.Fatalf("%s: no preset", slug)
		}
		sp := f(newRand(5))
		if !dbOwn(&sp) {
			t.Fatalf("%s: not drawn by dragonball_art.go", slug)
		}
		twin := sp
		twin.Name = "twin"
		for vw := 0; vw < 3; vw++ {
			differs := false
			for _, a := range anims {
				for fr := 0; fr < animFrameCount(a); fr++ {
					pp := poseFor(a, fr)
					pp.Anim, pp.F = a, fr
					b := renderChar(&sp, vw, pp)
					tb := renderChar(&twin, vw, pp)
					n, edge := charimpAPix(b)
					_, tEdge := charimpAPix(tb)
					if edge > tEdge {
						t.Errorf("%s vw %d anim %d f %d: %d pixels on the edge (plain body %d)", slug, vw, a, fr, edge, tEdge)
					}
					if n < 80 {
						t.Errorf("%s vw %d anim %d f %d: only %d pixels", slug, vw, a, fr, n)
					}
					if string(b.p) != string(tb.p) {
						differs = true
					}
				}
			}
			if !differs {
				t.Errorf("%s vw %d: the details are not drawn", slug, vw)
			}
			w0 := renderChar(&sp, vw, poseFor(AnimWalk, 0))
			w1 := renderChar(&sp, vw, poseFor(AnimWalk, 1))
			if string(w0.p) == string(w1.p) {
				t.Errorf("%s vw %d: walking does not change the frames", slug, vw)
			}
		}
	}
}

// Piccolo and Cell are a bit taller than Goku; Krilin and Gohan shorter; the stature of the others is kept.
func TestDragonBallStature(t *testing.T) {
	st := map[string]int{}
	for _, slug := range dbSlugs {
		st[slug] = presetByName(slug)(newRand(1)).Stature
	}
	want := map[string]int{"goku": 0, "vegeta": 1, "krilin": 4, "gohan": 4, "piccolo": -2, "cell": -2, "maestro_roshi": 3, "bulma": 0}
	for k, v := range want {
		if st[k] != v {
			t.Errorf("%s stature %d, want %d", k, st[k], v)
		}
	}
}

// Sell (slug cell) is a new famous guest: preset, name, description in both languages, card in the album
// and in the Dragon Ball group; drawn the same with any random source (so its pictures can be exported).
func TestDragonBallCellGuest(t *testing.T) {
	if presetByName("cell") == nil {
		t.Fatal("no preset for cell")
	}
	// One changed letter (owner's naming policy, 2026-11-11); the slug stays "cell".
	if n := displayNameOverrides["cell"]; n != "Sell" {
		t.Errorf("name %q", n)
	}
	d, ok := charDescs["cell"]
	if !ok || d[0] == "" || d[1] == "" || langDataEN[d[0]] != d[1] {
		t.Errorf("description %v", d)
	}
	found := false
	for _, c := range charCards() {
		if c.slug == "cell" {
			found = true
		}
	}
	if !found {
		t.Error("cell has no card in the album")
	}
	inGroup := false
	for _, g := range starGroups {
		for _, s := range g {
			if s == "cell" && g[0] == "goku" {
				inGroup = true
			}
		}
	}
	if !inGroup {
		t.Error("cell is not in the Dragon Ball group")
	}
	for _, slug := range dbSlugs {
		a := presetByName(slug)(newRand(1))
		for _, seed := range []int64{7, 99, 12345} {
			b := presetByName(slug)(newRand(seed))
			for vw := 0; vw < 3; vw++ {
				if string(renderChar(&a, vw, poseFor(AnimIdle, 0)).p) != string(renderChar(&b, vw, poseFor(AnimIdle, 0)).p) {
					t.Errorf("%s vw %d: the drawing depends on chance", slug, vw)
				}
			}
		}
	}
}
