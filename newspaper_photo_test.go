package main

import (
	"fmt"
	"image"
	"strings"
	"testing"
)

// Every scene of the photo (and the booth strip), with many seeds, 1 and 2 players: nothing is painted outside the frame,
// and no row or column of the picture is a lone line (one colour over most of the width while the rows around it are
// something else): that is what the old wood grain and the tape across the whole photo looked like.
func TestPaperPhotoStaysInItsFrameWithoutStrayLines(t *testing.T) {
	const px, py, pw, ph = 40, 30, pbPhotoW, pbPhotoH
	scenes := []int{paperSceneBooth}
	for s := 0; s < paperPhotoScenes; s++ {
		scenes = append(scenes, s)
	}
	for _, sc := range scenes {
		for seed := int64(0); seed < 12; seed++ {
			for _, two := range []bool{false, true} {
				c := &pcv{im: image.NewRGBA(image.Rect(0, 0, pw+80, ph+60)), lay: &paperLayout{}}
				c.fill(0, 0, pw+80, ph+60, paperCream)
				c.photo(px, py, pw, ph, sc, two, seed*7717+3)
				for y := 0; y < ph+60; y++ {
					for x := 0; x < pw+80; x++ {
						in := x >= px-1 && x < px+pw+1 && y >= py-1 && y < py+ph+1
						if !in && c.im.RGBAAt(x, y) != paperCream {
							t.Fatalf("scene %d seed %d: pixel outside the frame at %d,%d", sc, seed, x-px, y-py)
						}
					}
				}
				// rows and columns inside the frame
				col := func(x, y int) uint32 {
					p := c.im.RGBAAt(x, y)
					return uint32(p.R)<<16 | uint32(p.G)<<8 | uint32(p.B)
				}
				line := func(n int, at func(i, k int) uint32, length int, what string) {
					for i := 2; i < n-2; i++ {
						cnt := map[uint32]int{}
						for k := 0; k < length; k++ {
							cnt[at(i, k)]++
						}
						for v, m := range cnt {
							if m*10 < length*8 {
								continue
							}
							a, b := 0, 0
							for k := 0; k < length; k++ {
								if at(i-1, k) == v {
									a++
								}
								if at(i+1, k) == v {
									b++
								}
							}
							if a*10 < length*4 && b*10 < length*4 {
								t.Fatalf("scene %d seed %d: a stray %s line at %d (%d of %d pixels one colour, %d / %d around)", sc, seed, what, i, m, length, a, b)
							}
						}
					}
				}
				line(ph, func(i, k int) uint32 { return col(px+1+k, py+i) }, pw-2, "row")
				line(pw, func(i, k int) uint32 { return col(px+i, py+1+k) }, ph-2, "column")
			}
		}
	}
}

// The photo suits the party and changes from one party to the next: eight scenes, the crime scene only after a crime,
// the sleeping DJ more likely with a DJ, captions in both languages, and in a session the photo of the party before is
// not repeated.
func TestPaperPhotoScenesVaryAndFitTheParty(t *testing.T) {
	seen := map[int]bool{}
	for seed := int64(0); seed < 300; seed++ {
		sc := paperPhotoScene(paperFacts{seed: seed * 911, kills: 3})
		seen[sc] = true
		if paperPhotoScene(paperFacts{seed: seed * 911}) == sceneHouse {
			t.Fatalf("seed %d: the crime-scene photo for a party without a crime", seed)
		}
	}
	if len(seen) != paperPhotoScenes {
		t.Errorf("only %d of %d scenes appear", len(seen), paperPhotoScenes)
	}
	for _, sc := range paperPhotoCaptions {
		if len(sc) < 5 {
			t.Errorf("a scene has only %d captions", len(sc))
		}
		for _, cp := range sc {
			if cp[1] == "" || cp[1] == cp[0] {
				t.Errorf("no English for caption %q", cp[0])
			}
			if strings.ContainsAny(cp[0], "%") {
				t.Errorf("a caption with a format verb: %q", cp[0])
			}
		}
	}
	old := paperSession
	t.Cleanup(func() { paperSession = old })
	paperSession = &paperSessionT{rng: newTestRand(9)}
	f := paperFacts{level: "4", levelN: 4, kills: 6, knife: 6, dj: true, secs: 150}
	prev, prevCap := -1, ""
	scenes := map[int]bool{}
	for i := 0; i < 30; i++ {
		g := paperVary(f, fmt.Sprint("photo-party-", i))
		sc := paperPhotoScene(g)
		cp := paperPhotoCaption(g, sc)
		if sc == prev || cp == prevCap {
			t.Fatalf("party %d: the same photo as the party before (scene %d, %q)", i, sc, cp)
		}
		prev, prevCap = sc, cp
		scenes[sc] = true
	}
	if len(scenes) < 6 {
		t.Errorf("30 parties showed only %d scenes", len(scenes))
	}
}
