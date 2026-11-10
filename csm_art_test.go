package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// Chewbacca, Steve, the four plumbers, Gandalf, Frodo and Knuckles (csm_art.go).

var csmSlugs = []string{"chewbacca", "steve_cubos", "mario", "luigi", "wario", "waluigi", "gandalf", "frodo", "knuckles"}

func csmSpec(t *testing.T, slug string) CharSpec {
	f := presetByName(slug)
	if f == nil {
		t.Fatalf("%s: no preset", slug)
	}
	return f(newRand(5))
}

// csmBox: opaque pixels, pixels on the outer ring, top row and width of a frame.
func csmBox(b *pixBuf) (n, edge, top, x0, x1 int) {
	top, x0, x1 = b.h, b.w, -1
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			if b.p[(y*b.w+x)*4+3] == 0 {
				continue
			}
			n++
			if x == 0 || y == 0 || x == b.w-1 || y == b.h-1 {
				edge++
			}
			top = minI(top, y)
			x0 = minI(x0, x)
			x1 = maxI(x1, x)
		}
	}
	return
}

// Every one has its own drawing and draws in the three views, in every usual animation, inside
// the sprite box (nothing on the outer ring) with a full body.
func TestCsmDrawsInsideTheSprite(t *testing.T) {
	anims := []int{AnimIdle, AnimWalk, AnimRun, AnimStab, AnimDance0, AnimDance3, AnimDance7, AnimPanic, AnimHop, AnimWave, AnimCheer, AnimLaugh}
	for _, slug := range csmSlugs {
		sp := csmSpec(t, slug)
		if !isCsm(&sp) {
			t.Fatalf("%s: no own drawing", slug)
		}
		for vw := 0; vw < 3; vw++ {
			for _, a := range anims {
				for f := 0; f < animFrameCount(a); f++ {
					b := renderChar(&sp, vw, ownPose(a, f))
					if b.w != charW || b.h != charH {
						t.Fatalf("%s box %dx%d", slug, b.w, b.h)
					}
					n, edge, _, _, _ := csmBox(b)
					noCsmArt = true // the same spec on the stock body (held objects may reach the border there too)
					_, tEdge, _, _, _ := csmBox(renderChar(&sp, vw, ownPose(a, f)))
					noCsmArt = false
					if n < 80 {
						t.Errorf("%s view %d anim %d f %d: only %d pixels", slug, vw, a, f, n)
					}
					if edge > tEdge || ((a == AnimIdle || a == AnimWalk) && edge > 0) {
						t.Errorf("%s view %d anim %d f %d: %d pixels on the sprite border (stock body %d)", slug, vw, a, f, edge, tEdge)
					}
				}
			}
		}
	}
}

// Walking changes the picture (at least 3 different frames) in every view, and the own drawing is
// not the stock body.
func TestCsmWalksAndDiffersFromStock(t *testing.T) {
	for _, slug := range csmSlugs {
		sp := csmSpec(t, slug)
		for vw := 0; vw < 3; vw++ {
			seen := map[string]bool{}
			for f := 0; f < animFrameCount(AnimWalk); f++ {
				seen[string(renderChar(&sp, vw, ownPose(AnimWalk, f)).p)] = true
			}
			if len(seen) < 3 {
				t.Errorf("%s view %d: only %d different walking frames", slug, vw, len(seen))
			}
			own := genChar(&sp, vw, ownPose(AnimIdle, 0))
			noCsmArt = true
			plain := genChar(&sp, vw, ownPose(AnimIdle, 0))
			noCsmArt = false
			if string(own.p) == string(plain.p) {
				t.Errorf("%s view %d: same as the stock body", slug, vw)
			}
		}
	}
}

// csmDist: share of pixels whose opacity or colour differ clearly between two frames.
func csmDist(a, b *pixBuf) float64 {
	d := 0
	for i := 0; i < len(a.p); i += 4 {
		if (a.p[i+3] == 0) != (b.p[i+3] == 0) {
			d++
			continue
		}
		if a.p[i+3] == 0 {
			continue
		}
		dc := abs(int(a.p[i])-int(b.p[i])) + abs(int(a.p[i+1])-int(b.p[i+1])) + abs(int(a.p[i+2])-int(b.p[i+2]))
		if dc > 60 {
			d++
		}
	}
	return float64(d) / float64(len(a.p)/4)
}

// The nine are clearly different from each other in every view (Mario and Luigi too).
func TestCsmDistinguishable(t *testing.T) {
	for vw := 0; vw < 3; vw++ {
		frames := map[string]*pixBuf{}
		for _, slug := range csmSlugs {
			sp := csmSpec(t, slug)
			frames[slug] = renderChar(&sp, vw, ownPose(AnimIdle, 0))
		}
		for i, a := range csmSlugs {
			for _, b := range csmSlugs[i+1:] {
				if d := csmDist(frames[a], frames[b]); d < 0.08 {
					t.Errorf("view %d: %s and %s differ in only %.1f%% of the pixels", vw, a, b, d*100)
				}
			}
		}
	}
}

// Heights and builds: Chewbacca and Waluigi taller than a guest, Waluigi taller than Luigi, Luigi
// taller than Mario; Wario and Frodo short, Wario the widest, Waluigi thin.
func TestCsmStatures(t *testing.T) {
	top := map[string]int{}
	wid := map[string]int{}
	for _, slug := range append(csmSlugs, "guest") {
		if slug == "guest" { // an ordinary adult guest: the most common top row of many random ones
			cnt := map[int]int{}
			for seed := int64(1); seed <= 40; seed++ {
				sp := randomGuestSpec(newRand(seed))
				_, _, tp, _, _ := csmBox(renderChar(&sp, vFront, ownPose(AnimIdle, 0)))
				cnt[tp]++
			}
			best := 0
			for tp, c := range cnt {
				if c > best {
					best, top[slug] = c, tp
				}
			}
			continue
		}
		sp := csmSpec(t, slug)
		_, _, tp, _, _ := csmBox(renderChar(&sp, vFront, ownPose(AnimIdle, 0)))
		top[slug] = tp
		// torso width at the waist row, arms apart: count the row 2 above the hips (front view)
		if isCsm(&sp) {
			lk := csmLookOf(csmKind(&sp))
			wid[slug] = 2 * lk.waist
			if lk.belly {
				wid[slug] += 2
			}
		}
	}
	t.Logf("tops: %v widths: %v", top, wid)
	less := func(a, b string) {
		if top[a] >= top[b] {
			t.Errorf("%s (top %d) should be taller than %s (top %d)", a, top[a], b, top[b])
		}
	}
	less("chewbacca", "guest")
	less("waluigi", "guest")
	less("waluigi", "luigi")
	less("luigi", "mario")
	less("mario", "wario")
	less("guest", "wario")
	less("guest", "frodo")
	less("wario", "frodo")
	for _, s := range csmSlugs {
		if s != "wario" && wid[s] >= wid["wario"] {
			t.Errorf("%s (%d) as wide as Wario (%d)", s, wid[s], wid["wario"])
		}
	}
	if wid["waluigi"] > wid["luigi"] {
		t.Errorf("Waluigi (%d) wider than Luigi (%d)", wid["waluigi"], wid["luigi"])
	}
}

// CSM_SHEET=dir go test -run TestCsmSheet writes an enlarged sheet: front, side, back (idle) and two
// walking frames of each one.
func TestCsmSheet(t *testing.T) {
	dir := os.Getenv("CSM_SHEET")
	if dir == "" {
		t.Skip("set CSM_SHEET to a directory")
	}
	os.MkdirAll(dir, 0o755)
	const z, cols = 5, 6
	cellW, cellH := cols*charW*z+12, charH*z+6
	big := image.NewRGBA(image.Rect(0, 0, cellW, len(csmSlugs)*cellH))
	for i := 0; i < len(big.Pix); i += 4 {
		big.Pix[i], big.Pix[i+1], big.Pix[i+2], big.Pix[i+3] = 0x6a, 0x6a, 0x74, 255
	}
	for gi, slug := range csmSlugs {
		sp := csmSpec(t, slug)
		poses := []struct {
			vw int
			p  Pose
		}{
			{vFront, ownPose(AnimIdle, 0)}, {vSide, ownPose(AnimIdle, 0)}, {vBack, ownPose(AnimIdle, 0)},
			{vFront, ownPose(AnimWalk, 1)}, {vSide, ownPose(AnimWalk, 1)}, {vFront, ownPose(AnimStab, 1)},
		}
		for ci, ps := range poses {
			b := renderChar(&sp, ps.vw, ps.p)
			ox, oy := ci*charW*z, gi*cellH
			for y := 0; y < b.h; y++ {
				for x := 0; x < b.w; x++ {
					i := (y*b.w + x) * 4
					if b.p[i+3] == 0 {
						if (x+y)%2 == 0 {
							continue
						}
						for dy := 0; dy < z; dy++ {
							for dx := 0; dx < z; dx++ {
								big.SetRGBA(ox+x*z+dx, oy+y*z+dy, color.RGBA{0x62, 0x62, 0x6c, 255})
							}
						}
						continue
					}
					c := color.RGBA{b.p[i], b.p[i+1], b.p[i+2], 255}
					for dy := 0; dy < z; dy++ {
						for dx := 0; dx < z; dx++ {
							big.SetRGBA(ox+x*z+dx, oy+y*z+dy, c)
						}
					}
				}
			}
		}
	}
	f, err := os.Create(filepath.Join(dir, "csm_sheet.png"))
	if err != nil {
		t.Fatal(err)
	}
	png.Encode(f, big)
	f.Close()
}
