package main

import (
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

var planetAll = append([]string{"capitan_planeta"}, planetSlugs...)

func planetSpecT(t *testing.T, slug string) CharSpec {
	t.Helper()
	f := presetByName(slug)
	if f == nil {
		t.Fatalf("%s: no preset", slug)
	}
	sp := f(rand.New(rand.NewSource(1)))
	if sp.Name != slug || planetKind(&sp) == plNone {
		t.Fatalf("%s: preset name %q", slug, sp.Name)
	}
	return sp
}

func planetCount(b *pixBuf, c color.RGBA) int {
	n := 0
	for i := 0; i < len(b.p); i += 4 {
		if b.p[i+3] > 0 && b.p[i] == c.R && b.p[i+1] == c.G && b.p[i+2] == c.B {
			n++
		}
	}
	return n
}

// The six draw in the three views and every animation inside the sprite box, with every body part,
// and their own details (emblem, belt, boots, vests, jacket, square, socks, rings) are painted.
func TestPlanetDrawInBox(t *testing.T) {
	for _, slug := range planetAll {
		sp := planetSpecT(t, slug)
		for vw := 0; vw < 3; vw++ {
			for a := 0; a < animGaitEnd; a++ {
				for f := 0; f < animFrameCount(a); f++ {
					pp := poseFor(a, f)
					pp.Anim, pp.F = a, f
					pp.Hold = HoldNone
					x0, y0, x1, y1 := bbox(genChar(&sp, vw, pp))
					if x1 < 0 {
						t.Fatalf("%s: empty frame vw=%d anim=%d f=%d", slug, vw, a, f)
					}
					// the generic body reaches the edge in some wide dances, like every guest:
					// standing, walking and running must stay well inside the box
					if a != AnimIdle && a != AnimWalk && a != AnimRun {
						continue
					}
					if x0 < 1 || y0 < 1 || x1 > charW-2 || y1 > charH-1 {
						t.Errorf("%s vw=%d anim=%d f=%d outside the box: %d,%d-%d,%d", slug, vw, a, f, x0, y0, x1, y1)
					}
				}
			}
			pp := poseFor(AnimIdle, 0)
			pp.Anim = AnimIdle
			trackParts = true
			b := genChar(&sp, vw, pp)
			trackParts = false
			for id := uint8(1); id <= 6; id++ {
				found := false
				for i, v := range b.pid {
					if v == id && b.p[i*4+3] > 0 {
						found = true
						break
					}
				}
				if !found && !(vw == vSide && (id == 3 || id == 5)) {
					t.Errorf("%s vw=%d: part %d missing", slug, vw, id)
				}
			}
			own := map[int]color.RGBA{plCaptain: plRedSuit, plEarth: plVestE, plWater: plVestW, plHeart: plVestH, plFire: hx(0xec7aac), plWind: plGreenSq}
			k := planetKind(&sp)
			if vw == vBack && (k == plFire || k == plWind) {
				continue // the T-shirt and the square are on the front
			}
			if n := planetCount(b, own[k]); n < 2 {
				t.Errorf("%s vw=%d: own colour only %d pixels", slug, vw, n)
			}
			if vw == vFront {
				if k == plCaptain && (planetCount(b, plEmblem) < 3 || planetCount(b, plBlack) < 3) {
					t.Errorf("capitan_planeta: no emblem or belt")
				}
				if k == plHeart && planetCount(b, plSock) < 2 {
					t.Errorf("planetario_corazon: no yellow socks")
				}
				if k != plCaptain && planetCount(b, hx(planetRing(k)[0])) < 1 {
					t.Errorf("%s: no ring on the hand", slug)
				}
			}
		}
	}
}

// Walking changes the frames in every view, and the six are distinguishable from each other.
func TestPlanetAnimateAndDistinct(t *testing.T) {
	seen := map[string]string{}
	for _, slug := range planetAll {
		sp := planetSpecT(t, slug)
		for vw := 0; vw < 3; vw++ {
			for _, a := range []int{AnimWalk, AnimRun} {
				p0, p1 := poseFor(a, 0), poseFor(a, 1)
				p0.Anim, p1.Anim, p1.F = a, a, 1
				if string(genChar(&sp, vw, p0).p) == string(genChar(&sp, vw, p1).p) {
					t.Errorf("%s vw=%d anim=%d: frames 0 and 1 are the same", slug, vw, a)
				}
			}
			p := poseFor(AnimIdle, 0)
			key := string(rune('0'+vw)) + string(genChar(&sp, vw, p).p)
			if o, ok := seen[key]; ok {
				t.Errorf("%s and %s look the same in view %d", slug, o, vw)
			}
			seen[key] = slug
		}
	}
	// different skins/shirts in front view: no two share both colours
	pairs := map[[2]color.RGBA]string{}
	for _, slug := range planetAll {
		sp := planetSpecT(t, slug)
		k := [2]color.RGBA{sp.Shirt, sp.Pants}
		if o, ok := pairs[k]; ok {
			t.Errorf("%s and %s wear the same clothes", slug, o)
		}
		pairs[k] = slug
	}
}

// The five friends are new famous guests with name, description ES/EN and the group; the Captain
// keeps his slug, name, body, stature and his shop flag.
func TestPlanetCast(t *testing.T) {
	fn := famousNames()
	for _, slug := range planetAll {
		if !fn[slug] {
			t.Errorf("%s is not a famous guest", slug)
		}
		sp := planetSpecT(t, slug)
		n := nameFromSpec(&sp)
		if n == "" || n == slug {
			t.Errorf("%s: no display name (%q)", slug, n)
		}
		if tr0 := langDataEN[n]; tr0 == "" {
			t.Errorf("%s: name %q has no English", slug, n)
		}
		if d, ok := charDescs[slug]; !ok || d[0] == "" || d[1] == "" || len([]rune(d[0])) > 36 {
			t.Errorf("%s: description %v", slug, d)
		}
	}
	cp := planetSpecT(t, "capitan_planeta")
	if nameFromSpec(&cp) != "Capitán Planeta" || cp.Body != BodyBig || cp.Stature != -1 {
		t.Errorf("Capitán Planeta changed: %q body %d stature %d", nameFromSpec(&cp), cp.Body, cp.Stature)
	}
	found := false
	for _, g := range starGroups {
		if len(g) == 6 && g[0] == "capitan_planeta" {
			found = true
			for i, s := range planetSlugs {
				if g[i+1] != s {
					found = false
				}
			}
		}
	}
	if !found {
		t.Errorf("no Planetarios group in starGroups")
	}
	cards := map[string]bool{}
	for _, c := range charCards() {
		cards[c.slug] = true
	}
	for _, slug := range planetAll {
		if !cards[slug] {
			t.Errorf("%s has no album card", slug)
		}
	}
}

// PH_PLANET_SHEET=/path.png dumps a x6 sheet: one block per character, rows = front/back/side.
func TestPlanetSheet(t *testing.T) {
	out := os.Getenv("PH_PLANET_SHEET")
	if out == "" {
		t.Skip("set PH_PLANET_SHEET to dump the sheet")
	}
	type pf struct{ a, f int }
	cols := []pf{{AnimIdle, 0}, {AnimWalk, 0}, {AnimWalk, 2}}
	k := 6
	bw := len(cols) * charW
	img := image.NewRGBA(image.Rect(0, 0, len(planetAll)*bw*k, 3*charH*k))
	for x := 0; x < img.Bounds().Dx(); x++ {
		for y := 0; y < img.Bounds().Dy(); y++ {
			c := color.RGBA{200, 200, 205, 255}
			if (x/(bw*k))%2 == 1 {
				c = color.RGBA{180, 186, 196, 255}
			}
			img.Set(x, y, c)
		}
	}
	for si, slug := range planetAll {
		sp := planetSpecT(t, slug)
		for vw := 0; vw < 3; vw++ {
			for ci, c := range cols {
				pp := poseFor(c.a, c.f)
				pp.Anim, pp.F = c.a, c.f
				b := renderChar(&sp, vw, pp)
				for y := 0; y < b.h; y++ {
					for x := 0; x < b.w; x++ {
						i := (y*b.w + x) * 4
						if b.p[i+3] == 0 {
							continue
						}
						col := color.RGBA{b.p[i], b.p[i+1], b.p[i+2], 255}
						for dy := 0; dy < k; dy++ {
							for dx := 0; dx < k; dx++ {
								img.Set((si*bw+ci*charW+x)*k+dx, (vw*charH+y)*k+dy, col)
							}
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
	png.Encode(f, img)
}

// PLANET_EXPORT=assets-root go test -run TestPlanetExport creates the folders of the five new
// friends (same meta as exportMissingCharacters: display name, gender, shop flag); the pictures of
// the Captain are written again by TestCharimpReexport (charimpRenewed).
func TestPlanetExport(t *testing.T) {
	root := os.Getenv("PLANET_EXPORT")
	if root == "" {
		t.Skip("set PLANET_EXPORT to an assets folder")
	}
	want := map[string]bool{}
	for _, s := range planetSlugs {
		want[s] = true
	}
	n := 0
	for _, b := range builtinChars() {
		if !want[b.name] || b.cat != "special_people" {
			continue
		}
		if _, err := os.Stat(filepath.Join(spritesDir(root, b.cat), b.name, "meta.json")); err == nil {
			continue
		}
		ca := charimpFresh(b)
		ca.Meta.ShopOnly = true
		ca.Meta.DisplayName = nameFromSpec(&b.spec)
		ca.Meta.Gender = "m"
		if b.spec.Female {
			ca.Meta.Gender = "f"
		}
		if err := saveCharAsset(root, ca); err != nil {
			t.Fatal(err)
		}
		n++
	}
	t.Logf("created %d", n)
}
