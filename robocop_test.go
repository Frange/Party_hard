package main

import (
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"os"
	"testing"
)

func roboSpec() CharSpec { return presetByName("robocop")(rand.New(rand.NewSource(1))) }

// RoboCop is drawn in every view and pose inside its own box (one empty pixel around it for the outline).
func TestRobocopDrawnInBox(t *testing.T) {
	for _, nm := range []string{"robocop", "asterix", "obelix", "wally"} {
		testDrawnInBox(t, nm)
	}
}

func testDrawnInBox(t *testing.T, nm string) {
	s := presetByName(nm)(rand.New(rand.NewSource(1)))
	if nm == "robocop" && !isRobocop(&s) {
		t.Fatal("preset robocop is not recognised")
	}
	poses := []Pose{poseFor(AnimIdle, 0), poseFor(AnimIdle, 1), poseFor(AnimStab, 0), poseFor(AnimStab, 1), poseFor(AnimStab, 2), poseFor(AnimWalk, 2), poseFor(AnimRun, 4), poseFor(AnimPanic, 3)}
	for f := 0; f < animFrameCount(AnimHeavy); f++ {
		poses = append(poses, poseFor(AnimHeavy, f))
	}
	for vw := 0; vw < 3; vw++ {
		for i, p := range poses {
			b := genChar(&s, vw, p)
			o := s
			o.Name = "x_off"
			base := genChar(&o, vw, p)
			n := 0
			for y := 0; y < b.h; y++ {
				for x := 0; x < b.w; x++ {
					if b.p[(y*b.w+x)*4+3] == 0 {
						continue
					}
					n++
					if (x < 1 || y < 1 || x >= b.w-1 || y >= b.h-1) && base.p[(y*b.w+x)*4+3] == 0 { // only what the overlay adds
						t.Errorf("%s view %d pose %d: pixel on the border at %d,%d", nm, vw, i, x, y)
					}
				}
			}
			if n < 90 {
				t.Errorf("%s view %d pose %d: only %d pixels", nm, vw, i, n)
			}
		}
	}
}

// the overlay adds no allocations to drawing a frame (compared with the same armor without it)
func TestRobocopNoExtraAlloc(t *testing.T) {
	for _, nm := range []string{"robocop", "asterix", "obelix", "wally"} {
		testNoExtraAlloc(t, nm)
	}
}

func testNoExtraAlloc(t *testing.T, nm string) {
	s := presetByName(nm)(rand.New(rand.NewSource(1)))
	o := s
	o.Name = "x_off"
	p := poseFor(AnimStab, 1)
	for vw := 0; vw < 3; vw++ {
		a := testing.AllocsPerRun(30, func() { genChar(&s, vw, p) })
		c := testing.AllocsPerRun(30, func() { genChar(&o, vw, p) })
		if a > c {
			t.Errorf("%s view %d: %v allocs with the overlay, %v without", nm, vw, a, c)
		}
	}
}

// optional: ROBO_SHEET=/path.png writes a x6 sheet (front, back, side, stab) for a visual check
func TestRobocopSheet(t *testing.T) {
	path := os.Getenv("ROBO_SHEET")
	if path == "" {
		t.Skip("no ROBO_SHEET")
	}
	type cell struct {
		vw int
		p  Pose
		s  CharSpec
	}
	var cells []cell
	for _, nm := range []string{"robocop", "asterix", "obelix", "wally"} {
		sp := presetByName(nm)(rand.New(rand.NewSource(1)))
		for _, c := range []struct {
			vw int
			p  Pose
		}{{vFront, poseFor(AnimIdle, 0)}, {vBack, poseFor(AnimIdle, 1)}, {vSide, poseFor(AnimWalk, 1)}, {vFront, poseFor(AnimStab, 1)}, {vSide, poseFor(AnimStab, 1)}} {
			cells = append(cells, cell{c.vw, c.p, sp})
		}
	}
	const k = 6
	img := image.NewRGBA(image.Rect(0, 0, len(cells)*charW*k, charH*k))
	for x := 0; x < img.Rect.Dx(); x++ {
		for y := 0; y < img.Rect.Dy(); y++ {
			img.Set(x, y, color.RGBA{0x6a, 0x6e, 0x76, 255})
		}
	}
	for i, c := range cells {
		b := renderChar(&c.s, c.vw, c.p)
		for y := 0; y < b.h; y++ {
			for x := 0; x < b.w; x++ {
				o := (y*b.w + x) * 4
				if b.p[o+3] == 0 {
					continue
				}
				col := color.RGBA{b.p[o], b.p[o+1], b.p[o+2], 255}
				for dy := 0; dy < k; dy++ {
					for dx := 0; dx < k; dx++ {
						img.Set(i*charW*k+x*k+dx, y*k+dy, col)
					}
				}
			}
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	png.Encode(f, img)
}
