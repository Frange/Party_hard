package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// withAssetProps swaps in the prop kinds a player gets when the assets folder holds an export of
// EVERY built-in object (what the first run writes when there is no assets folder, and what the
// editor can save) plus the real project assets on top: those kinds come from PNG frames and have
// no Draw func. Everything is put back when the test ends (and nothing is written outside TempDir).
func withAssetProps(t *testing.T) {
	t.Helper()
	snapshotPropKinds(t)
	prevBG := jpBGCache
	t.Cleanup(func() { jpBGCache = prevBG })
	tmp := t.TempDir()
	if exportObjects(tmp) == 0 {
		t.Fatal("exportObjects wrote nothing")
	}
	var objs []*ObjAsset
	for _, root := range []string{tmp, "assets"} { // the real assets are only read
		for _, name := range spriteNames(root, "objetos") {
			oa, err := loadObjAsset(spriteDirOf(root, "objetos", name))
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			if oa.Meta.Name == "" {
				oa.Meta.Name = name
			}
			objs = append(objs, oa)
		}
	}
	applyObjects(objs)
	if k := propKinds["jp_boat"]; k == nil || k.Draw != nil || len(k.Native) == 0 {
		t.Fatal("setup: jp_boat should now come from its exported PNG (Native frames, no Draw)")
	}
	jpBGCache = nil
}

// Regression: an exported jp_boat (assets/sprites/objetos/jp_boat) made paintJurassic call a nil
// Draw and the game crashed on reaching Isla Nublar. Every registered prop must also render.
func TestJurassicPaintsWithExportedProps(t *testing.T) {
	withAssetProps(t)
	b := paintJurassic()
	// the ferry is painted at the pier, from its exported frame
	ink := 0
	for y := 448; y < 448+propKinds["jp_boat"].H-2; y++ {
		for x := 26; x < 26+propKinds["jp_boat"].W-2; x++ {
			if c := b.get(x, y); c.R > 220 && c.G > 220 && c.B > 220 {
				ink++
			}
		}
	}
	if ink < 300 {
		t.Errorf("the exported ferry is not painted at the pier (%d white hull pixels)", ink)
	}
	imgs := buildPropImages()
	for n, k := range propKinds {
		if len(imgs[n]) == 0 {
			t.Errorf("prop %s has no image", n)
		}
		v := k.Vars
		if v == 0 {
			v = 1
		}
		for i := 0; i < v; i++ {
			propBaseRel(n, k, i)
		}
	}
}

// The island started the normal way (startLevel, as the arrival, the list of parties and DEBUG do)
// with every object coming from asset files: it must load, run and draw without a panic.
func TestJurassicStartsWithExportedProps(t *testing.T) {
	prev := noAssets
	noAssets = true // newGame must not read or write the real assets folder
	t.Cleanup(func() { noAssets = prev })
	progressFrozen = true
	g := newGame(true)
	withAssetProps(t)
	g.as.props = buildPropImages()
	g.startLevel(specialJurassic)
	if g.levelIdx != specialJurassic || g.mode != ModePlay {
		t.Fatalf("the island did not start (idx %d mode %v)", g.levelIdx, g.mode)
	}
	screen := ebiten.NewImage(screenW, screenH)
	for i := 0; i < 240; i++ {
		if err := g.Update(); err != nil {
			t.Fatalf("tick %d: %v", i, err)
		}
		if i%60 == 0 {
			g.Draw(screen)
		}
	}
}
