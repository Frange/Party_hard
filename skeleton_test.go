package main

import (
	"math/rand"
	"testing"
	"unicode/utf8"
)

// electrocuted bodies are skeletons, any other death is the normal body
func TestSkeletonBodyOnlyWhenElectrocuted(t *testing.T) {
	g := testGame()
	g.rng = rand.New(rand.NewSource(1))
	sp := playerSpec()
	cs := newCharSet(&sp)
	g.spawnBody(Vec{100, 100}, cs, DirDown)
	if g.bodies[0].skeleton {
		t.Fatal("a normal death must not leave a skeleton")
	}
	elecDeathNext = true
	g.spawnBody(Vec{140, 100}, cs, DirDown)
	if !g.bodies[1].skeleton {
		t.Fatal("an electrocuted body must be a skeleton")
	}
	if elecDeathNext {
		t.Fatal("the electrocution flag must be consumed by the body")
	}
	g.spawnBody(Vec{180, 100}, cs, DirDown)
	if g.bodies[2].skeleton {
		t.Fatal("the flag leaked into the next death")
	}
	if len(g.bodies) != 3 {
		t.Fatal("every death keeps its body")
	}
}

// the skeleton sprites exist, stand where a character stands and differ from each other
func TestSkeletonSprites(t *testing.T) {
	sp := playerSpec()
	cs := newCharSet(&sp)
	s := skeletonSet(cs)
	if s.up == nil || s.ly == nil {
		t.Fatal("no skeleton sprites")
	}
	if s.up.Bounds().Dy() < charH-2 || s.ly.Bounds().Dx() < charH-2 {
		t.Errorf("skeleton too small: %v %v", s.up.Bounds(), s.ly.Bounds())
	}
	b := genSkeleton(rgb(0x808080), rgb(0x404040))
	opaque := 0
	for i := 3; i < len(b.p); i += 4 {
		if b.p[i] > 0 {
			opaque++
		}
	}
	if opaque < 120 {
		t.Errorf("an almost empty skeleton: %d px", opaque)
	}
	if skeletonSet(cs) != s {
		t.Error("the skeleton must be cached")
	}
}

// the skeleton crowd: presets, lines, names, shop and English
func TestSkeletonCast(t *testing.T) {
	buildShopCatalog()
	for _, s := range skelSlugs {
		f := presetByName(s)
		if f == nil {
			t.Errorf("no preset %s", s)
			continue
		}
		if sp := f(newRand(1)); sp.Name != s || sp.MaskKind != MaskSkull {
			t.Errorf("preset %s: name %q mask %d", s, sp.Name, sp.MaskKind)
		}
		if len(skelCatch[s]) == 0 || quirkOf(s) == "" || len(quirkLines[quirkOf(s)]) == 0 || len(quirkAnims[quirkOf(s)]) == 0 {
			t.Errorf("%s has no lines", s)
		}
		if funnyDesc["skin:"+s] == "" || displayNameOverrides[s] == "" {
			t.Errorf("%s has no description or name", s)
		}
		found := false
		for _, it := range shopCatalog[shopChars] {
			if it.id == "skin:"+s {
				found = true
			}
		}
		if !found {
			t.Errorf("%s is not in the shop", s)
		}
		inPool := false
		for _, p := range specialPool {
			if p == s {
				inPool = true
			}
		}
		if !inPool {
			t.Errorf("%s never turns up", s)
		}
	}
	chk := func(where, s string) {
		if n := utf8.RuneCountInString(s); n > maxPhrase {
			t.Errorf("%s: %d chars: %q", where, n, s)
		}
		if _, ok := langDataEN[s]; !ok {
			t.Errorf("%s: no English for %q", where, s)
		}
	}
	for k, ls := range skelCatch {
		for _, l := range ls {
			chk("catch "+k, l)
		}
	}
	for _, d := range skelDuets {
		if presetByName(d.a) == nil || presetByName(d.b) == nil {
			t.Errorf("duet with unknown slug %s/%s", d.a, d.b)
		}
		for _, st := range d.steps {
			chk("duet", st.line)
		}
	}
	for k, v := range skelDesc {
		if _, ok := langDataEN[v]; !ok {
			t.Errorf("no English for the description of %s", k)
		}
	}
	for _, v := range skelNames {
		if _, ok := langDataEN[v]; !ok {
			t.Errorf("no English for the name %q", v)
		}
	}
	for _, l := range defaultNewsBank()["electrocuted"] {
		if _, ok := langDataEN[l]; !ok {
			t.Errorf("no English for the news %q", l)
		}
	}
}
