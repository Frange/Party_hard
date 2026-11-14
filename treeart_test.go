package main

import (
	"image"
	"os"
	"path/filepath"
	"testing"
)

// Every tree and bush variant is drawn, in the same box as variant 0, different from the others, and stands on the
// same foot (something drawn right over the anchor at the bottom: the trunk or the base of the bush).
func TestTreeArtVariantsDrawn(t *testing.T) {
	for _, tc := range []struct {
		name string
		n    int
	}{{"tree", treeVars}, {"bush", bushVars}} {
		k := treeArtBuiltin[tc.name]
		if k == nil || k.Vars != tc.n {
			t.Fatalf("%s: built-in kind %+v", tc.name, k)
		}
		var prev []*pixBuf
		for v := 0; v < tc.n; v++ {
			b := renderPropSprite(tc.name, k, v)
			if b.w != k.W+2 || b.h != k.H+2 {
				t.Errorf("%s %d: size %dx%d", tc.name, v, b.w, b.h)
			}
			if o := opaque(b); o < b.w*b.h/12 { // the dead fir is a bare trunk
				t.Errorf("%s %d: only %d pixels drawn", tc.name, v, o)
			}
			foot := 0
			ax := k.W/2 + 1
			for y := b.h - 5; y < b.h; y++ {
				for x := ax - 4; x <= ax+4; x++ {
					if b.get(x, y).A > 0 {
						foot++
					}
				}
			}
			if foot < 6 {
				t.Errorf("%s %d: nothing stands on the anchor (%d px)", tc.name, v, foot)
			}
			for j, p := range prev {
				if string(p.p) == string(b.p) {
					t.Errorf("%s %d is the same drawing as %d", tc.name, v, j)
				}
			}
			prev = append(prev, b)
		}
	}
}

// The theme of the level picks the kinds: dead and dark firs in the graveyard, cherries and twisted pines in a
// Japanese place, firs in the snow; the pick is stable, follows the place, and an explicit variant is kept.
func TestTreeArtThemePicks(t *testing.T) {
	t.Cleanup(func() { treeThemeDef = nil })
	pick := func(def *LevelDef, name string) map[int]int {
		got := map[int]int{}
		for i := 0; i < 200; i++ {
			pr := &Prop{Name: name, X: float64(30 + i*37%900), Y: float64(60 + i*53%460)}
			got[treeVarFor(def, pr)]++
		}
		return got
	}
	only := func(what string, got map[int]int, allowed ...int) {
		ok := map[int]bool{}
		for _, a := range allowed {
			ok[a] = true
		}
		for v := range got {
			if !ok[v] {
				t.Errorf("%s: variant %d not expected (%v)", what, v, got)
			}
		}
	}
	grave := &LevelDef{Icon: "grave", Title: "CEMENTERIO"}
	only("graveyard trees", pick(grave, "tree"), 3, 4, 5)
	only("graveyard bushes", pick(grave, "bush"), 3, 4)
	if pick(grave, "tree")[5] == 0 {
		t.Errorf("no dead fir in the graveyard")
	}
	jp := &LevelDef{Title: "JARDÍN JAPONÉS"}
	g := pick(jp, "tree")
	only("japanese trees", g, 1, 6, 7)
	if g[6] == 0 || g[7] == 0 {
		t.Errorf("japanese place without cherries or twisted pines: %v", g)
	}
	only("snow trees", pick(&LevelDef{Icon: "snow"}, "tree"), 3, 4, 5)
	park := pick(&LevelDef{Icon: "house"}, "tree")
	if len(park) < 4 {
		t.Errorf("a park has little variety: %v", park)
	}
	pr := &Prop{Name: "tree", X: 100, Y: 200}
	a := treeVarFor(grave, pr)
	treeVarFor(jp, pr)
	if treeVarFor(grave, pr) != a {
		t.Errorf("the pick is not stable")
	}
	pr.Var = 6
	if treeVarFor(grave, pr) != 6 {
		t.Errorf("an explicit variant is not kept")
	}
}

// The variants change nothing of the collisions: same footprint and solids whatever the variant.
func TestTreeArtCollisionsUnchanged(t *testing.T) {
	for _, n := range []string{"tree", "bush"} {
		k := treeArtBuiltin[n]
		def := &LevelDef{W: 960, H: 540, Props: []PropInst{PV(n, 200, 200, 0), PV(n, 400, 200, 3), PV(n, 600, 200, 5)}}
		l := buildLevelRaw(def)
		var mine []*Prop
		for _, pr := range l.Props {
			if pr.Name == n && pr.Y == 200 && (pr.X == 200 || pr.X == 400 || pr.X == 600) {
				mine = append(mine, pr)
			}
		}
		if len(mine) != 3 {
			t.Fatalf("%s: %d of the 3 props", n, len(mine))
		}
		f0 := mine[0].Foot
		for i, pr := range mine {
			f := pr.Foot
			if f.W != f0.W || f.H != f0.H || f.X-pr.X != f0.X-200 || f.Y-pr.Y != f0.Y-200 {
				t.Errorf("%s %d: foot %v vs %v", n, i, f, f0)
			}
		}
		if k.FW != propKinds[n].FW && propKinds[n].Native == nil {
			t.Errorf("%s: footprint changed", n)
		}
	}
	if k := treeArtBuiltin["tree"]; k.FW != 10 || k.FH != 6 {
		t.Errorf("tree footprint %vx%v", k.FW, k.FH)
	}
	if k := treeArtBuiltin["bush"]; k.FW != 26 || k.FH != 9 || !k.Hide {
		t.Errorf("bush footprint %vx%v hide %v", k.FW, k.FH, k.Hide)
	}
}

// The files of the asset folder are the stock export of every variant (so the game shows them) and the fireplace
// still sprite is the new hearth (so its moving frames match it).
func TestTreeArtAssetsAreStock(t *testing.T) {
	for _, n := range []string{"tree", "bush", "fireplace"} {
		oa, err := loadObjAsset(filepath.Join("assets", "sprites", "objetos", n))
		if err != nil {
			t.Fatalf("%s: %v", n, err)
		}
		k := treeArtBuiltin[n]
		cnt := k.Vars
		if cnt == 0 {
			cnt = 1
		}
		if len(oa.Frames) != cnt || (cnt > 1) != oa.Meta.Variants {
			t.Fatalf("%s: %d frames (variants %v), want %d", n, len(oa.Frames), oa.Meta.Variants, cnt)
		}
		for v, f := range oa.Frames {
			r := renderPropSprite(n, k, v)
			if r.w != f.w || r.h != f.h || string(r.p) != string(f.p) {
				t.Errorf("%s frame %d is not the built-in drawing", n, v)
			}
		}
	}
}

// TREEART_EXPORT=assets go test -run TestTreeArtExport rewrites only the tree, bush and fireplace folders (meta
// kept; frames = every built-in variant). It refuses a folder whose first frame is not the stock drawing of the
// classic variant (somebody painted it in the editor).
func TestTreeArtExport(t *testing.T) {
	root := os.Getenv("TREEART_EXPORT")
	if root == "" {
		t.Skip("set TREEART_EXPORT to an assets folder")
	}
	for _, n := range []string{"tree", "bush", "fireplace"} {
		oa, err := loadObjAsset(filepath.Join(spritesDir(root, "objetos"), n))
		if err != nil {
			t.Fatalf("%s: %v", n, err)
		}
		k := treeArtBuiltin[n]
		if n != "fireplace" {
			r := renderPropSprite(n, k, 0)
			if f := oa.Frames[0]; r.w != f.w || r.h != f.h || string(r.p) != string(f.p) {
				t.Fatalf("%s: the file is not the stock drawing: not overwritten", n)
			}
		}
		cnt := k.Vars
		if cnt == 0 {
			cnt = 1
		}
		oa.Frames = nil
		for v := 0; v < cnt; v++ {
			oa.Frames = append(oa.Frames, renderPropSprite(n, k, v))
		}
		oa.Meta.Variants = cnt > 1
		if err := saveObjAsset(root, oa); err != nil {
			t.Fatal(err)
		}
	}
}

// The flame painter: the frames move (they differ from one to the next), use the whole ramp (pale core .. dark red
// rim), throw sparks above the body, and the loop closes (frame n == frame 0).
func TestFireArtLayersAndMoves(t *testing.T) {
	paint := func(f int) *pixBuf {
		b := newPix(30, 50)
		paintFire(b, 15, 46, 16, 25, f, fireFrameN, 4, image.Rectangle{}, true)
		return b
	}
	var prev *pixBuf
	for f := 0; f < fireFrameN; f++ {
		b := paint(f)
		seen := map[[3]byte]bool{}
		for i := 0; i < len(b.p); i += 4 {
			if b.p[i+3] > 0 {
				seen[[3]byte{b.p[i], b.p[i+1], b.p[i+2]}] = true
			}
		}
		for j, col := range fireRamp {
			if !seen[[3]byte{col.R, col.G, col.B}] {
				t.Errorf("frame %d: tone %d of the ramp is missing", f, j)
			}
		}
		high := 0 // something above the body (tongues breaking off, sparks)
		for y := 0; y < 46-25; y++ {
			for x := 0; x < b.w; x++ {
				if b.get(x, y).A > 0 {
					high++
				}
			}
		}
		if high == 0 {
			t.Errorf("frame %d: no tongue or spark rises above the flame", f)
		}
		if prev != nil && string(prev.p) == string(b.p) {
			t.Errorf("frame %d is the same as the one before: no animation", f)
		}
		prev = b
	}
	if string(paint(fireFrameN).p) != string(paint(0).p) {
		t.Errorf("the loop does not close")
	}
}

// The hearth stays inside the opening of the fireplace, moves between frames, and the moving frames of the fireplace
// (propanim.go) are the still sprite at frame 0 (they match it) and change between instants of the game clock.
func TestFireArtHearthMatchesFireplace(t *testing.T) {
	for f := 0; f < hearthFrames; f++ {
		b := newPix(60, 54)
		drawHearth(b, f)
		for y := 0; y < b.h; y++ {
			for x := 0; x < b.w; x++ {
				if b.get(x, y).A > 0 && !(image.Point{x, y}).In(image.Rect(12, 20, 48, 48)) {
					t.Fatalf("frame %d: the hearth paints outside the opening at %d,%d", f, x, y)
				}
			}
		}
	}
	k := treeArtBuiltin["fireplace"]
	a := propAnims["fireplace"]
	if a == nil || a.n != hearthFrames {
		t.Fatalf("fireplace animation %+v", a)
	}
	still := renderPropSprite("fireplace", k, 0)
	f0 := renderPropFrame("fireplace", k, a, 0, 0)
	if string(still.p) != string(f0.p) {
		t.Errorf("frame 0 of the moving fireplace is not the still sprite")
	}
	frames := map[int]bool{}
	for i := 0; i < 20; i++ {
		frames[propAnimFrameAt(a, float64(i)*0.07, 0, false)] = true
	}
	if len(frames) < 4 {
		t.Errorf("the fireplace barely moves: %v", frames)
	}
	for f := 1; f < a.n; f++ {
		if string(renderPropFrame("fireplace", k, a, 0, f).p) == string(renderPropFrame("fireplace", k, a, 0, f-1).p) {
			t.Errorf("frames %d and %d of the fireplace are the same", f-1, f)
		}
	}
}

// The phone icons: rectangular pixel maps with known colours; the marker handset fits its 16x12 box with a margin;
// the HUD handset has a body, a dark outline and the coiled cord in every tint.
func TestPhoneArtMaps(t *testing.T) {
	pal := phonePal(phoneRed)
	for _, m := range [][]string{phoneHandsetMap, phoneMarkMap} {
		for i, r := range m {
			if len(r) != len(m[0]) {
				t.Fatalf("row %d is %d wide, want %d", i, len(r), len(m[0]))
			}
			for _, ch := range []byte(r) {
				if _, ok := pal[ch]; !ok && ch != '.' {
					t.Fatalf("row %d: unknown colour %q", i, ch)
				}
			}
		}
	}
	if w, h := len(phoneMarkMap[0]), len(phoneMarkMap); w > 16-4 || h > 12-4 {
		t.Errorf("marker handset %dx%d does not fit the 16x12 box with a margin", w, h)
	}
	for tint := 0; tint < phoneTints; tint++ {
		p := phonePal(tint)
		b := pixFromMap(phoneHandsetMap, tint)
		body, outline, cord := 0, 0, 0
		for y := 0; y < b.h; y++ {
			for x := 0; x < b.w; x++ {
				switch b.get(x, y) {
				case p['B'], p['L'], p['D'], p['H']:
					body++
				case p['K']:
					outline++
				case p['G'], p['g']:
					cord++
				}
			}
		}
		if body < 80 || outline < 50 || cord < 5 {
			t.Errorf("tint %d: %d body, %d outline, %d cord pixels", tint, body, outline, cord)
		}
	}
}

// The handset takes the colour of the booth it rings from: the yellow booth of the technicians -> yellow, the other
// booths -> red like them; blue only in a level without any booth. The marker over a booth is its colour too.
func TestPhoneArtColourOfTheBooth(t *testing.T) {
	red, yel := propKinds["phone"], propKinds["techphone"]
	if red == nil || yel == nil {
		t.Fatal("booth kinds missing")
	}
	withBooth := &Level{Props: []*Prop{{Name: "phone", K: red}, {Name: "techphone", K: yel}}}
	if got := phoneTintForCall(withBooth, 2); got != phoneYellow {
		t.Errorf("call from the yellow booth: tint %d, want yellow", got)
	}
	if got := phoneTintForCall(withBooth, 1); got != phoneRed {
		t.Errorf("call from a red booth: tint %d, want red", got)
	}
	if got := phoneTintForCall(&Level{}, 1); got != phoneBlue {
		t.Errorf("no booth: tint %d, want blue", got)
	}
	if phoneTintForProp(withBooth.Props[0]) != phoneRed || phoneTintForProp(withBooth.Props[1]) != phoneYellow {
		t.Errorf("markers do not take the colour of their booth")
	}
	// the tints really are the colours of the booths
	rb := pixFromMap(phoneHandsetMap, phoneRed)
	yb := pixFromMap(phoneHandsetMap, phoneYellow)
	r, y := rb.get(10, 6), yb.get(10, 6) // a pixel of the body ('B')
	if !(r.R > 150 && r.G < 80 && r.B < 80) || !(y.R > 200 && y.G > 160 && y.B < 80) {
		t.Errorf("body colours red %v yellow %v", r, y)
	}
}

// ART_SHEET=file.png writes every new drawing (trees, bushes, flames, hearth, phones) at 4x to look at it.
func TestArtSheet(t *testing.T) {
	out := os.Getenv("ART_SHEET")
	if out == "" {
		t.Skip("set ART_SHEET")
	}
	var items []*pixBuf
	for v := 0; v < treeVars; v++ {
		items = append(items, stylizeSprite(renderPropSprite("tree", treeArtBuiltin["tree"], v)))
	}
	for v := 0; v < bushVars; v++ {
		items = append(items, stylizeSprite(renderPropSprite("bush", treeArtBuiltin["bush"], v)))
	}
	for f := 0; f < 4; f++ {
		b := newPix(30, 54)
		paintFire(b, 15, 50, 16, 25, f*2, fireFrameN, 4, image.Rectangle{}, true)
		items = append(items, b)
	}
	for f := 0; f < 4; f++ {
		items = append(items, stylizeSprite(renderPropFrame("fireplace", treeArtBuiltin["fireplace"], propAnims["fireplace"], 0, f*2)))
	}
	for tint := 0; tint < phoneTints; tint++ {
		items = append(items, pixFromMap(phoneHandsetMap, tint), pixFromMap(phoneMarkMap, tint))
	}
	const z = 4
	W, H, x, y, rowH := 760, 0, 0, 0, 0
	type at struct{ x, y int }
	pos := []at{}
	for _, it := range items {
		if x+it.w+4 > W {
			x, y, rowH = 0, y+rowH+4, 0
		}
		pos = append(pos, at{x, y})
		x += it.w + 4
		if it.h > rowH {
			rowH = it.h
		}
	}
	H = y + rowH
	big := newPix(W*z, H*z)
	for i := 0; i < len(big.p); i += 4 {
		big.p[i], big.p[i+1], big.p[i+2], big.p[i+3] = 40, 70, 40, 255
	}
	for i, it := range items {
		for yy := 0; yy < it.h*z; yy++ {
			for xx := 0; xx < it.w*z; xx++ {
				if col := it.get(xx/z, yy/z); col.A > 0 {
					big.set(pos[i].x*z+xx, pos[i].y*z+yy, col)
				}
			}
		}
	}
	if err := savePNG(out, big); err != nil {
		t.Fatal(err)
	}
}

// The graveyard has living trees and bushes too (v1.4.0), drawn with its own variants, standing on nothing else.
func TestGraveyardHasTreesAndBushes(t *testing.T) {
	i := campaignIdx("grave")
	d := builtinLevels()[i]
	l := buildLevel(d)
	n := map[string]int{}
	for _, pr := range l.Props {
		if pr.Name == "tree" || pr.Name == "bush" {
			n[pr.Name]++
			if v := treeVarFor(d, pr); pr.Name == "tree" && v != 3 && v != 4 && v != 5 || pr.Name == "bush" && v != 3 && v != 4 {
				t.Errorf("%s at %.0f,%.0f drawn as variant %d, not a graveyard one", pr.Name, pr.X, pr.Y, v)
			}
		}
	}
	if n["tree"] < 2 || n["bush"] < 2 {
		t.Errorf("graveyard: %d trees and %d bushes", n["tree"], n["bush"])
	}
	if ov := levelOverlaps(l); len(ov) > 0 {
		t.Errorf("graveyard: things on top of each other: %v", ov)
	}
}
