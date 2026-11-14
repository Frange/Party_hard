package main

import (
	"math/rand"
	"path/filepath"
	"testing"
)

// levelByIcon: the index in levelDefs of the campaign party with that icon.
func levelByIcon(t *testing.T, icon string) int {
	t.Helper()
	killTargetLevels(t)
	for i, d := range levelDefs {
		if d.Icon == icon {
			return i
		}
	}
	t.Fatalf("no party with icon %q", icon)
	return -1
}

// Every moving prop: its frames have the size of the still sprite (same anchor, same footprint: collisions are
// untouched), and two different instants give two different pictures.
func TestPropAnimFramesMoveAndKeepTheSize(t *testing.T) {
	propAnimRemember()
	for name, a := range propAnims {
		k := propAnimBuiltin[name]
		if k == nil {
			k = propKinds[name]
		}
		if k == nil || k.Native != nil {
			t.Fatalf("%s: no built-in kind", name)
		}
		vars := k.Vars
		if vars == 0 {
			vars = 1
		}
		for v := 0; v < vars; v++ {
			still := renderPropSprite(name, k, v)
			frames := make([]string, a.n)
			for f := 0; f < a.n; f++ {
				fb := renderPropFrame(name, k, a, v, f)
				if fb.w != still.w || fb.h != still.h {
					t.Fatalf("%s v%d f%d: frame %dx%d, still sprite %dx%d", name, v, f, fb.w, fb.h, still.w, still.h)
				}
				frames[f] = string(fb.p)
			}
			if name == "gondola" && v == 0 {
				continue // the boat without gondolier does not move
			}
			if name == "alc_stork" {
				continue // the storks stand still most of the time and move in bouts (TestAlcalaStorksMove)
			}
			// two instants of the game clock a third of a second apart (the slot machine: idle and played)
			busy := []bool{false}
			if name == "slot" {
				busy = append(busy, true)
			}
			for _, bz := range busy {
				f0 := propAnimFrameAt(a, 10.0, 0, bz)
				f1 := propAnimFrameAt(a, 10.0+1/a.fps+0.01, 0, bz)
				if bz {
					f1 = propAnimFrameAt(a, 10.0+0.09, 0, bz)
				}
				if f0 == f1 || frames[f0] == frames[f1] {
					t.Errorf("%s v%d busy=%v: frames %d and %d are the same picture", name, v, bz, f0, f1)
				}
			}
		}
	}
}

// The frame comes from the game clock and the place only: the same instant gives the same picture (captures and
// pause are stable).
func TestPropAnimDeterministic(t *testing.T) {
	for _, a := range propAnims {
		for _, tm := range []float64{0, 3.3, 77.7, 401.25} {
			if propAnimFrameAt(a, tm, 13, false) != propAnimFrameAt(a, tm, 13, false) {
				t.Fatal("not deterministic")
			}
			if f := propAnimFrameAt(a, tm, 13, false); f < 0 || f >= a.n {
				t.Fatalf("frame %d out of [0,%d)", f, a.n)
			}
		}
	}
}

// The objects of the asset folder are the stock export of the built-in drawings: they animate. A sprite the user
// painted himself (one pixel changed) stays still.
func TestPropAnimStockAssetsAnimateEditedOnesDoNot(t *testing.T) {
	propAnimRemember()
	for _, name := range []string{"ferris", "carousel", "balloons", "stands", "fireplace", "fountain"} {
		oa, err := loadObjAsset(filepath.Join("assets", "sprites", "objetos", name))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		ak := &PropKind{Native: oa.Frames}
		if propAnimKind(name, ak) == nil {
			t.Errorf("%s: the stock asset is not recognised: it would not move in the game", name)
		}
		ed := *oa.Frames[0]
		ed.p = append([]byte(nil), ed.p...)
		ed.p[len(ed.p)/2] ^= 0x40
		fr := append([]*pixBuf{&ed}, oa.Frames[1:]...)
		if propAnimKind(name, &PropKind{Native: fr}) != nil {
			t.Errorf("%s: an edited sprite would be replaced by the animation", name)
		}
	}
}

// The casino has a roulette (with the wheel turning) and slot machines, and the guests go to play: some stand at a
// machine for a while (its reels spin), every place has one player at most, every place is on walkable ground
// reachable from the start, and they leave again.
func TestCasinoRouletteAndGuestsPlay(t *testing.T) {
	ci := levelByIcon(t, "casino")
	g := bigPlay(t, ci, 41, false)
	rou, slots := 0, 0
	for _, pr := range g.lvl.Props {
		switch pr.Name {
		case "roulette":
			rou++
		case "slot":
			slots++
		}
	}
	if rou < 1 || slots < 2 {
		t.Fatalf("casino: %d roulettes, %d slot machines", rou, slots)
	}
	b := g.machBookOf()
	if len(b.spots) < 4 {
		t.Fatalf("casino: only %d places to play", len(b.spots))
	}
	reach := g.lvl.reachFrom(g.lvl.NearestFree(g.lvl.Def.Start))
	hasRou := false
	for _, s := range b.spots {
		cx, cy := g.lvl.cellOf(s.at)
		if !g.lvl.Free(s.at) || g.lvl.SolidP(s.at.X, s.at.Y) || !reach[cy*g.lvl.cw+cx] {
			t.Errorf("place at %v of %s is not walkable / reachable", s.at, s.prop.Name)
		}
		hasRou = hasRou || s.prop.Name == "roulette"
	}
	if !hasRou {
		t.Errorf("nobody can play the roulette")
	}
	g.rng = rand.New(rand.NewSource(5))
	sent := 0
	for _, gu := range g.guests {
		if sent >= 10 {
			break
		}
		if gu.out() || gu.state != gIdle && gu.state != gDance && gu.state != gGoto {
			continue
		}
		gu.state, gu.timer, gu.path = gIdle, 0, nil
		for i := 0; i < 4 && !g.machPick(gu); i++ {
		}
		if b.who[gu] != nil {
			sent++
		}
	}
	if sent < 3 {
		t.Fatalf("only %d guests went to play", sent)
	}
	maxPlay, busySeen, left := 0, false, 0
	was := map[*Guest]bool{}
	for f := 0; f < 60*60 && g.mode == ModePlay; f++ {
		g.updatePlay(1.0 / 60)
		if f%30 != 0 {
			continue
		}
		n := 0
		owner := map[*machSpot]*Guest{}
		for _, gu := range g.guests {
			st := g.machPlaying(gu)
			if st != nil {
				n++
				if o := owner[st.s]; o != nil && o != gu {
					t.Fatalf("two players on one place")
				}
				owner[st.s] = gu
				if a, _, ok := g.machPose(gu); !ok || (st.s.prop.Name == "slot" && a != AnimDJ && a != AnimCheer) {
					t.Errorf("a slot player is not drawn playing")
				}
				if st.s.prop.Name == "slot" && g.machBusy(st.s.prop) {
					busySeen = true
				}
			}
			if was[gu] && st == nil {
				left++
			}
			was[gu] = st != nil
		}
		if n > maxPlay {
			maxPlay = n
		}
	}
	if maxPlay < 2 {
		t.Errorf("at most %d guests played at once", maxPlay)
	}
	if !busySeen {
		t.Errorf("no slot machine spun its reels while played")
	}
	if left < 1 {
		t.Errorf("nobody stopped playing in a minute")
	}
	t.Logf("casino: %d roulettes, %d slots, %d places, %d sent, up to %d playing, %d left", rou, slots, len(b.spots), sent, maxPlay, left)
}

// A party without machines: nobody is sent and the dice of the party are not touched.
func TestMachinesNoneElsewhere(t *testing.T) {
	fi := levelByIcon(t, "funfair")
	g := bigPlay(t, fi, 7, false)
	if n := len(g.machBookOf().spots); n != 0 {
		t.Fatalf("funfair: %d places to play", n)
	}
	g.rng = rand.New(rand.NewSource(7))
	for _, gu := range g.guests {
		if g.machPick(gu) {
			t.Fatal("a guest went to play in the funfair")
		}
	}
	if g.rng.Int63() != rand.New(rand.NewSource(7)).Int63() {
		t.Errorf("machPick moved the dice of a party without machines")
	}
}

// The moving props of the levels are drawn with their frames (the ferris wheel, the carousel, the balloons of the
// funfair; the stands; the gondola of Italy): the image changes between two instants and is the same at the same
// instant. Nothing of the level changes (the navigation grid is the same as with the animation off).
func TestPropAnimInLevels(t *testing.T) {
	for _, tc := range []struct {
		icon  string
		names []string
	}{{"funfair", []string{"ferris", "carousel"}}, {"stadium", []string{"stands"}}, {"italy", []string{"gondola"}}, {"casino", []string{"roulette", "slot"}}} {
		li := levelByIcon(t, tc.icon)
		g := bigPlay(t, li, 3, false)
		for _, name := range tc.names {
			var pr *Prop
			for _, p := range g.lvl.Props {
				if p.Name == name && !p.gone && (name != "gondola" || p.Var%2 == 1) {
					pr = p
					break
				}
			}
			if pr == nil {
				t.Errorf("%s: no %s", tc.icon, name)
				continue
			}
			a := propAnims[name]
			g.time = 50
			i0, ok0 := g.propAnimImg(pr, pr.Var)
			i0b, _ := g.propAnimImg(pr, pr.Var)
			g.time = 50 + 1/a.fps + 0.01
			i1, ok1 := g.propAnimImg(pr, pr.Var)
			if !ok0 || !ok1 {
				t.Errorf("%s: %s is not animated", tc.icon, name)
				continue
			}
			if i0.img == i1.img {
				t.Errorf("%s: %s shows the same frame at two instants", tc.icon, name)
			}
			if i0.img != i0b.img {
				t.Errorf("%s: %s is not deterministic", tc.icon, name)
			}
			still := g.as.props[name][pr.Var%len(g.as.props[name])]
			if i0.ox != still.ox || i0.oy != still.oy || i0.img.Bounds() != still.img.Bounds() {
				t.Errorf("%s: %s frame has another size/anchor than the still sprite", tc.icon, name)
			}
		}
		// the walkable grid does not depend on the animation
		nav := append([]bool(nil), g.lvl.reachFrom(g.lvl.NearestFree(g.lvl.Def.Start))...)
		propAnimOff = true
		nav2 := g.lvl.reachFrom(g.lvl.NearestFree(g.lvl.Def.Start))
		propAnimOff = false
		for i := range nav {
			if nav[i] != nav2[i] {
				t.Fatalf("%s: the flood fill changed", tc.icon)
			}
		}
	}
}
