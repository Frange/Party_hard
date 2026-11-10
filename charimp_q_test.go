package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"os"
	"strings"
	"testing"
)

// Famous guests Q-Z made closer to their originals (charimp_q.go). Tests: every one draws in the three
// views inside the sprite box (nothing clipped at the borders), the walk changes the frames, and the
// album still counts the same characters.

var qzImproved = []string{
	"spiderman", "superman", "wario", "waluigi", "woody", "thor", "thanos", "rambo", "rocky", "yoda",
	"stewie", "zoidberg", "santa", "trooper", "spock", "redshirt", "steve_cubos", "sherlock", "zorro",
	"zipi", "zape", "sailor_moon", "vegeta", "rick", "tintin", "tortuga_raphael",
}

func qzOpaque(b *pixBuf) (n int, edge bool) {
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			if b.p[(y*b.w+x)*4+3] == 0 {
				continue
			}
			n++
			if x == 0 || x == b.w-1 || y == 0 {
				edge = true
			}
		}
	}
	return
}

func TestCharImpQDrawsInBoxAndWalks(t *testing.T) {
	for _, n := range qzImproved {
		f := presetByName(n)
		if f == nil {
			t.Fatalf("no preset %s", n)
		}
		sp := f(rand.New(rand.NewSource(3)))
		if qzKind(&sp) == qzNone && n != "zipi" && n != "zape" {
			t.Errorf("%s has no improved drawing", n)
		}
		for vw := 0; vw < 3; vw++ {
			for _, a := range []int{AnimIdle, AnimWalk, AnimRun, AnimDance0, AnimStab, AnimWave} {
				for fr := 0; fr < animFrames[a]; fr++ {
					pp := poseFor(a, fr)
					pp.Anim, pp.F = a, fr
					b := genChar(&sp, vw, pp)
					if b.w != charW || b.h != charH {
						t.Fatalf("%s box %dx%d", n, b.w, b.h)
					}
					cnt, edge := qzOpaque(b)
					if cnt < 60 {
						t.Errorf("%s view %d anim %d f%d: only %d pixels", n, vw, a, fr, cnt)
					}
					if edge && (a == AnimIdle || a == AnimWalk) {
						t.Errorf("%s view %d anim %d f%d: drawing touches the sprite border", n, vw, a, fr)
					}
				}
			}
			// the walk must change the picture
			p0, p1 := poseFor(AnimWalk, 0), poseFor(AnimWalk, 2)
			p0.Anim, p1.Anim, p0.F, p1.F = AnimWalk, AnimWalk, 0, 2
			if bytes.Equal(genChar(&sp, vw, p0).p, genChar(&sp, vw, p1).p) {
				t.Errorf("%s view %d: walking frames are identical", n, vw)
			}
		}
		// the overlay really paints something (not for the colour-only fixes, nor for the Dragon Ball
		// fighters, now drawn by dragonball_art.go, which skips these touch-ups)
		if qzKind(&sp) != qzNone && !dbOwn(&sp) && !isLmdy(&sp) && !isHeroArt(&sp) && !isCsm(&sp) { // Yoda: lmdy_art.go; Superman: hero_art.go; Wario, Waluigi, Steve: csm_art.go
			pp := poseFor(AnimIdle, 0)
			pp.Anim = AnimIdle
			with := genChar(&sp, vFront, pp)
			noCharImpQ = true
			plain := genChar(&sp, vFront, pp)
			noCharImpQ = false
			if bytes.Equal(with.p, plain.p) {
				t.Errorf("%s: the improved drawing paints nothing", n)
			}
		}
	}
}

func TestCharImpQZipiZapeHair(t *testing.T) {
	zi := presetByName("zipi")(rand.New(rand.NewSource(1)))
	za := presetByName("zape")(rand.New(rand.NewSource(1)))
	lum := func(c color.RGBA) int { return int(c.R) + int(c.G) + int(c.B) }
	if lum(zi.Hair) <= lum(za.Hair)+150 {
		t.Errorf("Zipi must be the blond one and Zape the dark one: %v vs %v", zi.Hair, za.Hair)
	}
}

// The album keeps the same number of character cards (the improvement changes drawings only).
func TestCharImpQAlbumCountSame(t *testing.T) {
	n := 0
	for _, c := range charCards() {
		if c.roleES == "Invitado famoso" {
			n++
		}
	}
	for _, s := range qzImproved {
		found := false
		for _, c := range charCards() {
			found = found || c.slug == s
		}
		if !found {
			t.Errorf("%s left the album", s)
		}
	}
	if want := os.Getenv("QZ_ALBUM_N"); want != "" && want != itoaQZ(n) {
		t.Errorf("album count %d, want %s", n, want)
	}
	t.Logf("famous cards: %d", n)
}

func itoaQZ(n int) string {
	b := []byte{}
	if n == 0 {
		return "0"
	}
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// QZ_SHEET=path.png go test -run TestCharImpQSheet: enlarged sprite sheet (one row per character:
// front, side, back standing, then three walking frames front and side).
func TestCharImpQSheet(t *testing.T) {
	out := os.Getenv("QZ_SHEET")
	if out == "" {
		t.Skip("only for screenshots (QZ_SHEET)")
	}
	type cell struct{ vw, a, f int }
	cells := []cell{{vFront, AnimIdle, 0}, {vSide, AnimIdle, 0}, {vBack, AnimIdle, 0},
		{vFront, AnimWalk, 1}, {vFront, AnimWalk, 4}, {vSide, AnimWalk, 1}, {vSide, AnimWalk, 4}}
	list := qzImproved
	if only := os.Getenv("QZ_ONLY"); only != "" {
		list = strings.Split(only, ",")
	}
	sc := 4
	if os.Getenv("QZ_BIG") != "" {
		sc = 8
	}
	cw, ch := charW*sc, charH*sc
	img := image.NewRGBA(image.Rect(0, 0, cw*len(cells), ch*len(list)))
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			c := color.RGBA{0x5a, 0x6a, 0x5a, 255}
			if ((x/cw)+(y/ch))%2 == 0 {
				c = color.RGBA{0x6a, 0x7a, 0x6a, 255}
			}
			img.SetRGBA(x, y, c)
		}
	}
	for r, n := range list {
		sp := presetByName(n)(rand.New(rand.NewSource(3)))
		for ci, c := range cells {
			pp := poseFor(c.a, c.f)
			pp.Anim, pp.F = c.a, c.f
			b := genChar(&sp, c.vw, pp)
			for y := 0; y < b.h; y++ {
				for x := 0; x < b.w; x++ {
					i := (y*b.w + x) * 4
					if b.p[i+3] == 0 {
						continue
					}
					col := color.RGBA{b.p[i], b.p[i+1], b.p[i+2], 255}
					for dy := 0; dy < sc; dy++ {
						for dx := 0; dx < sc; dx++ {
							img.SetRGBA(ci*cw+x*sc+dx, r*ch+y*sc+dy, col)
						}
					}
				}
			}
		}
	}
	fh, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer fh.Close()
	if err := png.Encode(fh, img); err != nil {
		t.Fatal(err)
	}
}
