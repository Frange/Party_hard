package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"os"
	"testing"
)

// The Ninja Turtles drawn by genTurtle (turtles_art.go): every one draws inside the sprite box in
// the three views, the walk changes the frames, they are told apart by the mask colour, they are
// shorter than the stock guest drawing and the back view shows the shell.

var turtleSlugs = []string{"tortuga_leonardo", "tortuga_donatello", "tortuga_michelangelo", "tortuga_raphael"}

func turtleSpec(t *testing.T, n string) CharSpec {
	f := presetByName(n)
	if f == nil {
		t.Fatalf("no preset %s", n)
	}
	return f(rand.New(rand.NewSource(5)))
}

func turtlePose(a, f int) Pose {
	p := poseFor(a, f)
	p.Anim, p.F = a, f
	return p
}

func turtleCount(b *pixBuf, c color.RGBA) int {
	n := 0
	for i := 0; i+3 < len(b.p); i += 4 {
		if b.p[i+3] > 0 && b.p[i] == c.R && b.p[i+1] == c.G && b.p[i+2] == c.B {
			n++
		}
	}
	return n
}

func turtleTop(b *pixBuf) int {
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			if b.p[(y*b.w+x)*4+3] > 0 {
				return y
			}
		}
	}
	return b.h
}

func TestTurtlesDrawInBoxAndWalk(t *testing.T) {
	for _, n := range turtleSlugs {
		sp := turtleSpec(t, n)
		sp.Weapon, sp.Carry = 0, 0 // the held object is not part of the picture (export strips it too)
		if !isTurtle(&sp) {
			t.Fatalf("%s is not drawn by genTurtle", n)
		}
		for vw := 0; vw < 3; vw++ {
			for _, a := range []int{AnimIdle, AnimWalk, AnimRun, AnimDance0, AnimDance3, AnimStab, AnimWave} {
				for fr := 0; fr < animFrames[a]; fr++ {
					b := genChar(&sp, vw, turtlePose(a, fr))
					if b.w != charW || b.h != charH {
						t.Fatalf("%s box %dx%d", n, b.w, b.h)
					}
					cnt, edge := qzOpaque(b)
					if cnt < 150 {
						t.Errorf("%s view %d anim %d f%d: only %d pixels", n, vw, a, fr, cnt)
					}
					if edge && turtlePose(a, fr).Hold == HoldNone { // a held knife may stick out, like any guest's

						t.Errorf("%s view %d anim %d f%d: touches the sprite border", n, vw, a, fr)
					}
				}
			}
			if bytes.Equal(genChar(&sp, vw, turtlePose(AnimWalk, 0)).p, genChar(&sp, vw, turtlePose(AnimWalk, 2)).p) {
				t.Errorf("%s view %d: walking frames are identical", n, vw)
			}
			if bytes.Equal(genChar(&sp, vw, turtlePose(AnimDance0, 0)).p, genChar(&sp, vw, turtlePose(AnimDance0, 1)).p) {
				t.Errorf("%s view %d: dancing frames are identical", n, vw)
			}
		}
	}
}

func TestTurtlesToldApartByMask(t *testing.T) {
	for _, n := range turtleSlugs {
		sp := turtleSpec(t, n)
		own := turtleLooks[n]
		for vw := 0; vw < 3; vw++ {
			b := genChar(&sp, vw, turtlePose(AnimIdle, 0))
			if c := turtleCount(b, own.mask); c < 15 {
				t.Errorf("%s view %d: only %d pixels of its mask colour", n, vw, c)
			}
			for m, other := range turtleLooks {
				if m != n && turtleCount(b, other.mask) > 0 {
					t.Errorf("%s view %d shows %s's mask colour", n, vw, m)
				}
			}
			if c := turtleCount(b, own.skin); c < 40 {
				t.Errorf("%s view %d: only %d pixels of green skin", n, vw, c)
			}
		}
		back := genChar(&sp, vBack, turtlePose(AnimIdle, 0))
		if c := turtleCount(back, ttShell) + turtleCount(back, ttShellL) + turtleCount(back, ttShellD); c < 40 {
			t.Errorf("%s: the shell shows only %d pixels from the back", n, c)
		}
		front := genChar(&sp, vFront, turtlePose(AnimIdle, 0))
		if c := turtleCount(front, ttPlast) + turtleCount(front, ttPlastD) + turtleCount(front, ttPlastL); c < 25 {
			t.Errorf("%s: the plastron shows only %d pixels from the front", n, c)
		}
		// shorter than a guest: a bald guest of normal stature (stock drawing) starts 2+ rows higher
		guest := sp
		guest.Stature, guest.Hat, guest.Hair2 = 0, 0, HairBald
		noCharImpQ = true
		stock := genChar(&guest, vFront, turtlePose(AnimIdle, 0))
		noCharImpQ = false
		if turtleTop(front) < turtleTop(stock)+2 {
			t.Errorf("%s: top row %d, a guest starts at %d (should be shorter)", n, turtleTop(front), turtleTop(stock))
		}
	}
	// the four front pictures are all different
	seen := map[string]string{}
	for _, n := range turtleSlugs {
		sp := turtleSpec(t, n)
		k := string(genChar(&sp, vFront, turtlePose(AnimIdle, 0)).p)
		if o, ok := seen[k]; ok {
			t.Errorf("%s and %s draw the same", n, o)
		}
		seen[k] = n
	}
}

// TURTLE_SHEET=path.png go test -run TestTurtlesSheet: enlarged sheet, one row per turtle
// (front, side, back standing, then walk front/side/back, run side, dance front).
func TestTurtlesSheet(t *testing.T) {
	out := os.Getenv("TURTLE_SHEET")
	if out == "" {
		t.Skip("only for screenshots (TURTLE_SHEET)")
	}
	type cell struct{ vw, a, f int }
	cells := []cell{{vFront, AnimIdle, 0}, {vSide, AnimIdle, 0}, {vBack, AnimIdle, 0},
		{vFront, AnimWalk, 1}, {vSide, AnimWalk, 1}, {vBack, AnimWalk, 3}, {vSide, AnimRun, 1}, {vFront, AnimDance0, 1}}
	z := 6
	cw, ch := charW*z, charH*z
	img := image.NewRGBA(image.Rect(0, 0, cw*len(cells), ch*len(turtleSlugs)))
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			c := color.RGBA{70, 70, 84, 255}
			if (x/cw+y/ch)%2 == 0 {
				c = color.RGBA{86, 86, 100, 255}
			}
			img.SetRGBA(x, y, c)
		}
	}
	for r, n := range turtleSlugs {
		sp := turtleSpec(t, n)
		for i, c := range cells {
			b := renderChar(&sp, c.vw, turtlePose(c.a, c.f))
			for y := 0; y < b.h; y++ {
				for x := 0; x < b.w; x++ {
					k := (y*b.w + x) * 4
					if b.p[k+3] == 0 {
						continue
					}
					col := color.RGBA{b.p[k], b.p[k+1], b.p[k+2], 255}
					for dy := 0; dy < z; dy++ {
						for dx := 0; dx < z; dx++ {
							img.SetRGBA(i*cw+x*z+dx, r*ch+y*z+dy, col)
						}
					}
				}
			}
		}
	}
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}
