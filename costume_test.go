package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"math/rand"
	"os"
	"testing"
)

// costumeTestGame: a party of n ordinary guests on a level with the given costume theme.
func costumeTestGame(n int, seed int64) *Game {
	g := testGame()
	g.rng = rand.New(rand.NewSource(seed))
	r := rand.New(rand.NewSource(seed + 99))
	for i := 0; i < n; i++ {
		sp := genericSpec(r)
		sp.Name, sp.DisplayName = "", ""
		g.guests = append(g.guests, &Guest{cs: newCharSet(&sp)})
	}
	return g
}

func dressedSig(g *Game) []dressKey {
	var out []dressKey
	for _, gu := range g.guests {
		out = append(out, gu.cs.dress)
	}
	return out
}

// the same party seed dresses the same guests the same way; another seed changes it
func TestCostumeDeterministicBySeed(t *testing.T) {
	for _, kind := range []string{"beach", "grave", "casino"} {
		a, b := costumeTestGame(60, 4), costumeTestGame(60, 4)
		a.dressCostumesSeed(kind, 1234)
		b.dressCostumesSeed(kind, 1234)
		sa, sb := dressedSig(a), dressedSig(b)
		for i := range sa {
			if sa[i] != sb[i] {
				t.Fatalf("%s: guest %d dressed %v vs %v with the same seed", kind, i, sa[i], sb[i])
			}
		}
		c := costumeTestGame(60, 4)
		c.dressCostumesSeed(kind, 98765)
		same := true
		for i, d := range dressedSig(c) {
			if d != sa[i] {
				same = false
			}
		}
		if same {
			t.Errorf("%s: two seeds dressed the party exactly alike", kind)
		}
	}
}

// some guests, never all: 20-30 % per party, and not the same amount in every party
func TestCostumeFractionInRange(t *testing.T) {
	counts := map[int]bool{}
	for sd := int64(1); sd <= 200; sd++ {
		g := costumeTestGame(60, sd)
		g.dressCostumesSeed("funfair", sd*7919)
		n := 0
		looks := map[uint8]bool{}
		for _, gu := range g.guests {
			if gu.cs.dress.look != 0 {
				n++
				looks[gu.cs.dress.look] = true
			}
		}
		if f := float64(n) / 60; f < costumeMin-0.01 || f > costumeMax+0.01 {
			t.Fatalf("seed %d: %.0f %% dressed, want %.0f-%.0f %%", sd, f*100, costumeMin*100, costumeMax*100)
		}
		if n >= 10 && len(looks) < 2 {
			t.Errorf("seed %d: %d costumes, all the same look", sd, n)
		}
		counts[n] = true
	}
	if len(counts) < 4 {
		t.Errorf("every party dresses about the same number: %v", counts)
	}
	// late arrivals: the same share on average
	g := costumeTestGame(0, 3)
	g.dressCostumesSeed("snow", 55)
	n := 0
	for i := 0; i < 2000; i++ {
		sp := genericSpec(rand.New(rand.NewSource(int64(i))))
		sp.Name = ""
		gu := &Guest{cs: newCharSet(&sp)}
		g.costumeArrival(gu)
		if gu.cs.dress.look != 0 {
			n++
		}
	}
	if f := float64(n) / 2000; f < costumeMin-0.04 || f > costumeMax+0.04 {
		t.Errorf("arrivals: %.1f %% dressed", f*100)
	}
}

// only the themed parties (4-6 of them, never the pirate ship, the station or the island), and only ordinary guests
func TestCostumeExclusions(t *testing.T) {
	if n := len(costumeLevels); n < 4 || n > 6 {
		t.Errorf("%d themed parties, want 4-6", n)
	}
	for _, k := range []string{"station", "pirate", "cake", "hockey", "disco70", "italy", "house"} {
		if _, ok := costumeLevels[k]; ok {
			t.Errorf("%s must not have costumes", k)
		}
	}
	for kind, looks := range costumeLevels {
		for _, id := range looks {
			if costumeLookIdx(id) == 0 {
				t.Errorf("%s: unknown look %q", kind, id)
			}
		}
	}
	if costumeKindOf(&LevelDef{Icon: "station", Title: "CASINO"}) != "" || costumeKindOf(&LevelDef{Title: "CASINO"}) != "casino" {
		t.Error("theme lookup by icon / title is wrong")
	}
	if costumeKindOf(&LevelDef{Title: "ISLA NUBLAR"}) != "" {
		t.Error("the island got costumes")
	}
	g := costumeTestGame(0, 1)
	plain := func() *CharSet { sp := genericSpec(rand.New(rand.NewSource(3))); sp.Name = ""; return newCharSet(&sp) }
	kid := kidSlugs[0]
	kidSpec := genericSpec(rand.New(rand.NewSource(4)))
	kidSpec.Name = kid
	var famousName string
	for n := range famousNames() {
		famousName = n
		break
	}
	famSpec := genericSpec(rand.New(rand.NewSource(5)))
	famSpec.Name = famousName
	special := []*Guest{
		{cs: plain(), name: "Pepe"},
		{cs: plain(), perf: "dj"},
		{cs: plain(), keep: true},
		{cs: plain(), quirk: "mj"},
		{cs: plain(), py: &pyroSt{}},
		{cs: plain(), famousOK: true},
		{cs: plain(), dead: true},
		{cs: newCharSet(&kidSpec)},
		{cs: newCharSet(&famSpec)},
	}
	for i := 0; i < 400; i++ {
		g.guests = append(g.guests, special...)
	}
	for _, kind := range []string{"beach", "funfair", "grave", "snow", "casino", "eiffel", ""} {
		g.dressCostumesSeed(kind, 77)
		for i, gu := range special {
			if gu.cs.dress.look != 0 {
				t.Fatalf("%s: special guest %d dressed", kind, i)
			}
		}
	}
	// guards, police and the player never go through here; a level without a theme dresses nobody
	pg := costumeTestGame(50, 2)
	pg.p = &Player{cs: plain()}
	pg.guards = []*Guard{{cs: plain()}}
	pg.dressCostumesSeed("", 5)
	for _, gu := range pg.guests {
		if gu.cs.dress.look != 0 {
			t.Fatal("a party without a theme dressed someone")
		}
	}
	pg.dressCostumesSeed("grave", 5)
	if pg.p.cs.dress.look != 0 || pg.guards[0].cs.dress.look != 0 {
		t.Error("the player or a guard got a costume")
	}
	// two players: the same plan
	old := twoPlayerMode
	twoPlayerMode = true
	t.Cleanup(func() { twoPlayerMode = old })
	a, b := costumeTestGame(50, 6), costumeTestGame(50, 6)
	a.dressCostumesSeed("eiffel", 31)
	b.p2 = &Player{idx: 1, cs: plain()}
	b.dressCostumesSeed("eiffel", 31)
	for i := range a.guests {
		if a.guests[i].cs.dress != b.guests[i].cs.dress {
			t.Fatal("two players changes who wears a costume")
		}
	}
}

// the dice are our own: the game's rand and the shared rand of the built-in characters do not move
func TestCostumeKeepsSharedRand(t *testing.T) {
	g := costumeTestGame(60, 8)
	g.rng = rand.New(rand.NewSource(42))
	g.dressCostumesSeed("casino", 9)
	sp := genericSpec(rand.New(rand.NewSource(3)))
	sp.Name = ""
	for i := 0; i < 50; i++ {
		g.costumeArrival(&Guest{cs: newCharSet(&sp)})
	}
	if g.rng.Int63() != rand.New(rand.NewSource(42)).Int63() {
		t.Error("dressing the guests used g.rng")
	}
	before := builtinChars()
	g.dressCostumesSeed("grave", 10)
	after := builtinChars()
	if len(before) != len(after) {
		t.Fatal("builtin list changed")
	}
	for i := range before {
		if before[i].spec != after[i].spec {
			t.Fatalf("builtin %d changed after dressing", i)
		}
	}
}

// costumeBases: a few ordinary people of the asset folders (rigs) and a few built-in guests.
func costumeBases(t *testing.T) []*CharSet {
	t.Helper()
	var out []*CharSet
	if st := loadAssetStore(); st != nil {
		people := st.Chars["people"]
		for i := 0; i < len(people); i += maxI(1, len(people)/14) {
			out = append(out, newRigCharSet(people[i]))
		}
	}
	for sd := int64(1); sd <= 4; sd++ {
		sp := genericSpec(rand.New(rand.NewSource(sd)))
		sp.Name = ""
		out = append(out, newCharSet(&sp))
	}
	return out
}

// baseFrame: the plain frame frameImg would build, as a pixBuf.
func baseFrame(cs *CharSet, anim, vw, f int) *pixBuf {
	if cs.rig != nil {
		return cs.rigFrame(anim, vw, f)
	}
	switch anim {
	case AnimLying:
		sp := *cs.Spec
		return genChar(&sp, 0, poseFor(AnimIdle, 0)).rotCW()
	case AnimSleep:
		sp := *cs.Spec
		return genChar(&sp, 0, poseFor(AnimIdle, 1)).rotCCW()
	case AnimPee:
		return genChar(cs.Spec, vw, peePose(f, cs.Spec.Female))
	}
	pp := poseFor(anim, f)
	pp.Anim, pp.F = anim, f
	return genChar(cs.Spec, vw, pp)
}

// no costume pixel ever falls outside the character's drawing box, in any look, animation, frame or view;
// and the costume shows on the standing guest
func TestCostumeStaysInFrame(t *testing.T) {
	bases := costumeBases(t)
	if len(bases) < 8 {
		t.Fatalf("only %d bases", len(bases))
	}
	for bi, base := range bases {
		for anim := 0; anim < AnimCount; anim++ {
			for vw := 0; vw < 3; vw++ {
				if (anim == AnimLying || anim == AnimSleep) && vw != 0 {
					continue
				}
				for f := 0; f < animFrameCount(anim); f++ {
					raw := baseFrame(base, anim, vw, f)
					for li := 1; li < len(costumeLooks); li++ {
						for _, v := range []uint8{0, 3} {
							cs := dressedCharSet(base, dressKey{uint8(li), v})
							up := raw
							switch anim {
							case AnimLying:
								up = raw.rotCCW()
							case AnimSleep:
								up = raw.rotCW()
							}
							o := &pixBuf{w: up.w, h: up.h, p: append([]byte(nil), up.p...)}
							if oob := cs.paintDress(o, anim, vw, f); oob != 0 {
								t.Fatalf("base %d %s: look %s anim %d view %d frame %d: %d pixels outside the %dx%d box",
									bi, base.Spec.Name, costumeLooks[li].id, anim, vw, f, oob, up.w, up.h)
							}
							if anim == AnimIdle && vw == vFront && f == 0 {
								diff := 0
								for i := 3; i < len(o.p); i += 4 {
									if o.p[i-3] != up.p[i-3] || o.p[i-2] != up.p[i-2] || o.p[i-1] != up.p[i-1] || o.p[i] != up.p[i] {
										diff++
									}
								}
								if diff < 3 {
									t.Errorf("base %d: look %s barely shows (%d pixels)", bi, costumeLooks[li].id, diff)
								}
							}
						}
					}
				}
			}
		}
	}
}

// the head box of a rig frame is exactly where render drew the head (front and back: the head is drawn last)
func TestCostumeRigHeadMatchesRender(t *testing.T) {
	n := 0
	for _, cs := range costumeBases(t) {
		if cs.rig == nil {
			continue
		}
		for _, anim := range []int{AnimIdle, AnimWalk, AnimRun, AnimDance0, AnimDance5, AnimFall, AnimPanic, AnimCheer, AnimCrouch} {
			for _, vw := range []int{vFront, vBack} {
				for f := 0; f < animFrameCount(anim); f++ {
					p := poseFor(anim, f)
					d, ok := cs.rig.dressGeom(vw, p)
					if !ok {
						continue
					}
					l := &cs.rig.V[assetView(vw)].L[PartHead]
					bare := *cs.rig
					bare.carry = HoldNone // nothing in the hands: the head is the last thing drawn
					p.Hold = HoldNone
					b := bare.render(vw, p)
					same, all := 0, 0
					for y := 0; y < l.h; y++ {
						for x := 0; x < l.w; x++ {
							c := l.pix[y*l.w+x]
							if c.A == 0 {
								continue
							}
							all++
							if b.get(d.hx+x, d.hy+y) == c {
								same++
							}
						}
					}
					if same != all {
						t.Fatalf("%s anim %d view %d frame %d: only %d of %d head pixels where the box says", cs.Spec.Name, anim, vw, f, same, all)
					}
					n += same
				}
			}
		}
	}
	if n == 0 {
		t.Skip("no rig characters in the asset folder")
	}
}

// the dead keep their costume: the lying and sleeping frames are dressed too; killing keeps the dressed set
func TestCostumeCorpseKeepsDress(t *testing.T) {
	for _, base := range costumeBases(t) {
		for li := 1; li < len(costumeLooks); li++ {
			cs := dressedCharSet(base, dressKey{uint8(li), 1})
			for _, anim := range []int{AnimLying, AnimSleep} {
				raw := baseFrame(base, anim, 0, 0)
				got := cs.dressFrame(raw, anim, 0, 0)
				if got.w != raw.w || got.h != raw.h {
					t.Fatalf("lying frame changed size %dx%d -> %dx%d", raw.w, raw.h, got.w, got.h)
				}
				diff := 0
				for i := range raw.p {
					if raw.p[i] != got.p[i] {
						diff++
					}
				}
				if diff == 0 {
					t.Errorf("%s: look %s vanished on the body (anim %d)", base.Spec.Name, costumeLooks[li].id, anim)
				}
			}
		}
	}
	g := scoreGame(t)
	g.dressCostumesSeed("grave", 3)
	var gu *Guest
	for _, x := range g.guests {
		if x.cs.dress.look != 0 && !x.out() {
			gu = x
			break
		}
	}
	if gu == nil {
		t.Fatal("nobody dressed in the test party")
	}
	cs := gu.cs
	g.killGuest(gu, true)
	if !gu.dead || gu.cs.dress != cs.dress {
		t.Errorf("the body lost its costume: dead=%v %v -> %v", gu.dead, cs.dress, gu.cs.dress)
	}
}

// sets are shared per (base, look, colour): no new set and no new frame per guest or per frame
func TestCostumeSetsCached(t *testing.T) {
	g := costumeTestGame(0, 1)
	g.dressCostumesSeed("beach", 1)
	sp := genericSpec(rand.New(rand.NewSource(1)))
	base := newCharSet(&sp)
	a := g.costume.dressedSet(base, dressKey{1, 2})
	b := g.costume.dressedSet(base, dressKey{1, 2})
	c := g.costume.dressedSet(base, dressKey{1, 3})
	if a != b || a == c || a == base {
		t.Error("dressed sets are not cached per combination")
	}
	if base.dress.look != 0 || len(base.cache) != 0 {
		t.Error("dressing touched the plain set")
	}
}

// the costume colours stand out from the floors of their party (under the night layer) and do not look like its walls
func TestCostumeContrast(t *testing.T) {
	found := map[string]bool{}
	for _, def := range builtinLevels() {
		kind := costumeKindOf(def)
		if kind == "" || def.Icon == "" {
			continue
		}
		found[kind] = true
		aOut, aIn := nightAlphas(def)
		area := 0.0
		for _, f := range def.Floors {
			area += f.R.W * f.R.H
		}
		type floorAt struct {
			c color.RGBA
			a float64
		}
		var floors []floorAt
		for _, f := range def.Floors {
			if f.R.W*f.R.H < area*0.05 {
				continue
			}
			c, ok := floorAvg(f.Kind)
			if !ok {
				continue
			}
			a := aIn
			if !floorIndoor(f.Kind) {
				a = aOut
			}
			floors = append(floors, floorAt{overNight(c, def, a), a})
		}
		var walls []color.RGBA
		for _, w := range def.Walls {
			if wallThick(w.Style) {
				p := wallPal(w.Style)
				walls = append(walls, overNight(p.top, def, aIn), overNight(p.edge, def, aIn))
			}
		}
		for _, id := range costumeLevels[kind] {
			li := costumeLookIdx(id)
			for v := 0; v < 4; v++ { // every colour variant, with the colours it really paints
				cols := costumePainted(t, li, v)
				worst, wf := math.Inf(1), color.RGBA{}
				for _, fl := range floors { // on every floor of the party, some colour of the costume stands out
					best := 0.0
					for _, k := range cols {
						kc := overNight(k, def, fl.a)
						wallLike := false
						for _, w := range walls {
							if rgbDist(overNight(k, def, aIn), w) < 40 {
								wallLike = true // the same colour as a wall does not count
							}
						}
						if !wallLike {
							best = math.Max(best, lumContrast(kc, fl.c)+0.4*hueDist(kc, fl.c))
						}
					}
					if best < worst {
						worst, wf = best, fl.c
					}
				}
				if worst < 2.0 {
					t.Errorf("%s: look %s colour %d has nothing that stands out from the floor %v (%.2f)", def.Title, id, v, wf, worst)
				}
			}
		}
	}
	for kind := range costumeLevels {
		if !found[kind] {
			t.Errorf("no campaign party for the costume theme %s", kind)
		}
	}
}

// costumePainted: the colours a look paints (in a few pixels at least) on a standing built-in guest.
func costumePainted(t *testing.T, li, v int) []color.RGBA {
	t.Helper()
	sp := genericSpec(rand.New(rand.NewSource(11)))
	sp.Name = ""
	base := newCharSet(&sp)
	cs := dressedCharSet(base, dressKey{uint8(li), uint8(v)})
	n := map[color.RGBA]int{}
	for vw := 0; vw < 3; vw++ {
		raw := baseFrame(base, AnimIdle, vw, 0)
		o := cs.dressFrame(raw, AnimIdle, vw, 0)
		for i := 0; i < len(o.p); i += 4 {
			if o.p[i+3] != 0 && (o.p[i] != raw.p[i] || o.p[i+1] != raw.p[i+1] || o.p[i+2] != raw.p[i+2]) {
				n[color.RGBA{o.p[i], o.p[i+1], o.p[i+2], 255}]++
			}
		}
	}
	var out []color.RGBA
	for c, k := range n {
		if k >= 2 {
			out = append(out, c)
		}
	}
	return out
}

func rgbDist(a, b color.RGBA) float64 {
	dr, dg, db := float64(a.R)-float64(b.R), float64(a.G)-float64(b.G), float64(a.B)-float64(b.B)
	return math.Sqrt(dr*dr + dg*dg + db*db)
}

// hueDist: 0..~1.7, how different two colours are once their brightness is set aside.
func hueDist(a, b color.RGBA) float64 {
	n := func(c color.RGBA) (float64, float64, float64) {
		s := float64(c.R) + float64(c.G) + float64(c.B) + 1
		return float64(c.R) / s, float64(c.G) / s, float64(c.B) / s
	}
	ar, ag, ab := n(a)
	br, bg, bb := n(b)
	return math.Sqrt((ar-br)*(ar-br)+(ag-bg)*(ag-bg)+(ab-bb)*(ab-bb)) * 3
}

// COSTUME_SHEET=path.png go test -run TestCostumeSheet: every look on one ordinary guest (front, side, back, lying), for a look.
func TestCostumeSheet(t *testing.T) {
	out := os.Getenv("COSTUME_SHEET")
	if out == "" {
		t.Skip("set COSTUME_SHEET to write the sheet")
	}
	bases := costumeBases(t)
	base := bases[0]
	for _, b := range bases {
		if !tallHead(b) {
			base = b
			break
		}
	}
	const z = 4
	cellW, cellH := 24*4+44, 46
	img := image.NewRGBA(image.Rect(0, 0, cellW*z*2, (len(costumeLooks)+1)/2*cellH*z))
	for i := range img.Pix {
		img.Pix[i] = 0x30
		if i%4 == 3 {
			img.Pix[i] = 255
		}
	}
	blit := func(b *pixBuf, ox, oy int) {
		for y := 0; y < b.h; y++ {
			for x := 0; x < b.w; x++ {
				i := (y*b.w + x) * 4
				if b.p[i+3] == 0 {
					continue
				}
				for dy := 0; dy < z; dy++ {
					for dx := 0; dx < z; dx++ {
						img.Set((ox+x)*z+dx, (oy+y)*z+dy, color.RGBA{b.p[i], b.p[i+1], b.p[i+2], 255})
					}
				}
			}
		}
	}
	for li := 0; li < len(costumeLooks); li++ {
		cs := dressedCharSet(base, dressKey{uint8(li), uint8(li % 4)})
		ox, oy := (li%2)*cellW, (li/2)*cellH
		for k, vw := range []int{vFront, vSide, vBack} {
			b := baseFrame(base, AnimIdle, vw, 0)
			if li > 0 {
				b = cs.dressFrame(b, AnimIdle, vw, 0)
			}
			blit(b, ox+k*24, oy)
		}
		b := baseFrame(base, AnimLying, 0, 0)
		if li > 0 {
			b = cs.dressFrame(b, AnimLying, 0, 0)
		}
		blit(b, ox+72, oy+10)
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

// a guest who already wears a hat or a big hairdo never gets a hat (no hat on a hat) nor a face piece
func TestCostumeNoHatOnHat(t *testing.T) {
	tall, plain := 0, 0
	for _, cs := range costumeBases(t) {
		for kind := range costumeLevels {
			fits := costumeFits(kind, cs)
			for _, li := range fits {
				if tallHead(cs) && (costumeLooks[li].hat || costumeLooks[li].face) {
					t.Errorf("%s: %s offered to a tall head", cs.Spec.Name, costumeLooks[li].id)
				}
			}
			if !tallHead(cs) && len(fits) != len(costumeLevels[kind]) {
				t.Errorf("%s: a plain head should fit every look of %s", cs.Spec.Name, kind)
			}
		}
		if tallHead(cs) {
			tall++
		} else {
			plain++
		}
	}
	if tall == 0 || plain == 0 {
		t.Logf("bases: %d tall heads, %d plain", tall, plain)
	}
	hatted := genericSpec(rand.New(rand.NewSource(1)))
	hatted.Name, hatted.Hat = "", HatCap
	g := costumeTestGame(0, 1)
	for i := 0; i < 300; i++ {
		g.guests = append(g.guests, &Guest{cs: newCharSet(&hatted)})
	}
	g.dressCostumesSeed("eiffel", 4)
	n := 0
	for _, gu := range g.guests {
		if l := gu.cs.dress.look; l != 0 {
			n++
			if costumeLooks[l].hat {
				t.Fatalf("a guest with a cap got %s", costumeLooks[l].id)
			}
		}
	}
	if n == 0 {
		t.Error("eiffel has non-hat looks: some capped guests should still dress up")
	}
}
