package main

import (
	"math"
	"math/rand"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

type finBox struct{ x0, y0, x1, y1 float64 }

func finGap(a, b finBox) float64 {
	return math.Max(math.Max(a.x0-b.x1, b.x0-a.x1), math.Max(a.y0-b.y1, b.y0-a.y1))
}

// finPixBounds: the opaque pixels of a buffer (first and last column and row).
func finPixBounds(b *pixBuf) (x0, y0, x1, y1 int, ok bool) {
	x0, y0, x1, y1 = b.w, b.h, -1, -1
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			if b.p[(y*b.w+x)*4+3] != 0 {
				x0, y0 = minI(x0, x), minI(y0, y)
				x1, y1 = max(x1, x), max(y1, y)
			}
		}
	}
	return x0, y0, x1, y1, x1 >= 0
}

// finKidPix: the frame of the kid exactly as his CharSet builds it (frameImg), as pixels.
func finKidPix(t *testing.T, cs *CharSet, anim int, d Dir, fr float64) *pixBuf {
	t.Helper()
	vw := vFront
	switch d {
	case DirUp:
		vw = vBack
	case DirLeft, DirRight:
		vw = vSide
	}
	f := int(math.Floor(fr)) % animFrameCount(anim)
	if cs.rig != nil {
		return cs.rigFrame(anim, vw, f)
	}
	pp := poseFor(anim, f)
	pp.Anim, pp.F = anim, f
	return genChar(cs.Spec, vw, pp)
}

// finPartPix: the pixels of one part of a rider's plan.
func finPartPix(t *testing.T, a *finRideArt, i int, p finPart) *pixBuf {
	switch p.kind {
	case finPartShadow:
		return finDrawShadow(p.w)
	case finPartSkate:
		return finDrawSkate(p.view, p.w, hx(finRiders[i].deck))
	case finPartScooter:
		return finDrawScooter(p.view, p.w, hx(finRiders[i].deck))
	}
	return finKidPix(t, a.kids[i], p.anim, p.dir, p.fr)
}

// Everything a rider draws (shadow, board, kid, every frame of every trick) falls inside finRideBox, the rectangle
// the layout test uses, and on whole pixels at a whole scale (crisp). The kids are their own characters, without
// the little board the game draws under them, and 24 px wide frames (finRideKidCX).
func TestFinRidersInsideTheirBox(t *testing.T) {
	a := finRideArtGet()
	if pxScale(finRideSc) != finRideSc || finRideSc < 2 {
		t.Fatalf("scale %v: want a whole one", finRideSc)
	}
	for i, r := range finRiders {
		cs := a.kids[i]
		if cs == nil || cs.Spec == nil {
			t.Fatalf("%s: no character", r.slug)
		}
		if cs.Spec.Ride != 0 {
			t.Errorf("%s keeps the game's tiny board under his feet", r.slug)
		}
		if f := presetByName(r.slug); f == nil {
			t.Fatalf("%s: no preset", r.slug)
		} else if sp := f(rand.New(rand.NewSource(7))); hx(r.deck) != sp.RideCol || (sp.Ride == 2) != (r.ride == 2) {
			t.Errorf("%s: board %v colour %v, the game has %v %v", r.slug, r.ride, hx(r.deck), sp.Ride, sp.RideCol)
		}
		if im, _ := cs.img(AnimIdle, DirRight, 0); im.Bounds().Dx() != 2*finRideKidCX {
			t.Errorf("%s: frame %d px wide, want %d", r.slug, im.Bounds().Dx(), 2*finRideKidCX)
		}
		cache := map[[6]int]*pixBuf{}
		worst := 0.0
		for tt := finRidersFrom; tt < finRidersFrom+90; tt += 1.0 / 60 {
			s := finRideAt(i, tt)
			x0, y0, x1, y1, on := finRideBox(i, tt)
			if !on {
				t.Fatalf("%s off at %.2f", r.slug, tt)
			}
			parts, n := finRidePlan(i, &s)
			for _, p := range parts[:n] {
				if p.x != math.Round(p.x) || p.y != math.Round(p.y) || p.cx != math.Round(p.cx) || p.ay != math.Round(p.ay) || math.Abs(p.sx) != 1 {
					t.Fatalf("%s at %.2f: part %+v not on whole pixels", r.slug, tt, p)
				}
				key := [6]int{p.kind, p.view, p.w, p.anim, int(p.fr), int(p.dir)}
				b := cache[key]
				if b == nil {
					b = finPartPix(t, a, i, p)
					cache[key] = b
				}
				bx0, by0, bx1, by1, ok := finPixBounds(b)
				if !ok {
					t.Fatalf("%s: empty part %+v", r.slug, p)
				}
				sx0, sx1 := p.x+(float64(bx0)-p.cx)*finRideSc, p.x+(float64(bx1+1)-p.cx)*finRideSc
				if p.sx < 0 {
					sx0, sx1 = p.x-(float64(bx1+1)-p.cx)*finRideSc, p.x-(float64(bx0)-p.cx)*finRideSc
				}
				sy0, sy1 := p.y+(float64(by0)-p.ay)*finRideSc, p.y+(float64(by1+1)-p.ay)*finRideSc
				if sx0 < x0 || sx1 > x1 || sy0 < y0 || sy1 > y1 {
					t.Fatalf("%s at %.2f (trick %d p %.2f dir %v): part %d drawn at %v..%v x %v..%v, outside its box %v..%v x %v..%v",
						r.slug, tt, s.trick, s.p, s.dir, p.kind, sx0, sx1, sy0, sy1, x0, x1, y0, y1)
				}
				if p.kind == finPartKid {
					worst = math.Max(worst, s.g-s.h-sy0)
				}
			}
		}
		if worst < finRideH-6 {
			t.Errorf("%s: finRideH %v is far too loose (the head reaches %v px over the ground)", r.slug, finRideH, worst)
		}
	}
}

// The riders never come near the card: replaying 3 minutes of both last cards (secret and normal), Spanish and
// English, 1 and 2 players, every rider stays >= 16 px from the screen edges and >= 12 px from every text of the
// card (the title with its wobble, the lines, "Gracias por jugar", the hint to go back to the menu), from Murray
// and from his bubble in its biggest form.
func TestFinRidersClearOfCard(t *testing.T) {
	prev := uiLang
	t.Cleanup(func() { uiLang = prev; textRec = nil })
	for _, two := range []bool{false, true} {
		for _, trueEnd := range []bool{true, false} {
			for _, lang := range []string{"es", "en"} {
				g := layerGame(t, two)
				uiLang = lang
				g.startEndingPre(trueEnd)
				var obst []finBox
				for _, tt := range []float64{10, 30} { // Murray's bubble, and later the hint
					g.end.stage, g.end.t, g.end.total = endFinal, tt, tt
					textRec = &textRecorder{}
					img := ebiten.NewImage(screenW, screenH)
					before := finRidersDrawn
					g.drawEnding(img)
					img.Deallocate()
					if finRidersDrawn != before+1 {
						t.Errorf("two=%v trueEnd=%v: the riders are not drawn on the card", two, trueEnd)
					}
					for _, r := range textRec.texts {
						b := finBox{r.x, r.y, r.x + r.w, r.y + r.h}
						if r.scale >= 7 {
							b.y0, b.y1 = b.y0-2, b.y1+2
						}
						obst = append(obst, b)
					}
					textRec = nil
				}
				if len(obst) < 10 {
					t.Fatalf("only %d texts", len(obst))
				}
				im := g.as.props["murray"][0]
				footX, footY, _, _, _ := finMurrayGeom(trueEnd)
				sw, sh := float64(im.img.Bounds().Dx())*4, float64(im.img.Bounds().Dy())*4
				obst = append(obst, finBox{footX - im.ox*4, footY - im.oy*4, footX - im.ox*4 + sw, footY - im.oy*4 + sh})
				x0, y0, x1, y1 := finBubbleBox(trueEnd, finBubWrap, 3)
				obst = append(obst, finBox{x0, y0, x1, y1})
				minGap := 1e9
				for i := range finRiders {
					for tt := 0.0; tt < 183; tt += 1.0 / 30 {
						bx0, by0, bx1, by1, on := finRideBox(i, tt)
						if !on {
							if tt >= finRidersFrom {
								t.Fatalf("rider %d off at %.2f", i, tt)
							}
							continue
						}
						b := finBox{bx0, by0, bx1, by1}
						if b.x0 < 16 || b.y0 < 16 || b.x1 > screenW-16 || b.y1 > screenH-16 {
							t.Fatalf("[%s two=%v end=%v] %s at %.2f: %v closer than 16 px to the edge", lang, two, trueEnd, finRiders[i].slug, tt, b)
						}
						for _, o := range obst {
							d := finGap(b, o)
							minGap = math.Min(minGap, d)
							if d < 12 {
								s := finRideAt(i, tt)
								t.Fatalf("[%s two=%v end=%v] %s at %.2f (trick %d, h %.0f): %v only %.0f px from %v",
									lang, two, trueEnd, finRiders[i].slug, tt, s.trick, s.h, b, d, o)
							}
						}
					}
				}
				if minGap > 40 {
					t.Logf("[%s two=%v end=%v] closest approach %.0f px", lang, two, trueEnd, minGap)
				}
			}
		}
	}
}

// What they do: each one rolls and does his own tricks every few seconds (Diego kickflips and ollies, César 360
// spins, Sam-sam tailwhips), jumps in an arc and lands, faces the way he goes, and the three cross each other.
// All of it is a pure function of the card's time.
func TestFinRidersTricksAndCrossing(t *testing.T) {
	want := map[string][]int{"diego": {finTrickKickflip, finTrickOllie}, "cesar": {finTrickSpin}, "sam_sam": {finTrickWhip}}
	speeds := map[float64]bool{}
	for i, r := range finRiders {
		speeds[r.speed] = true
		if s := finRideAt(i, finRidersFrom-0.01); s.on {
			t.Errorf("%s rides before %v s", r.slug, finRidersFrom)
		}
		seen := map[int]int{}
		prevTrick, prevX, starts, lastStart := finTrickNone, -1.0, 0, finRidersFrom
		maxGapT := 0.0
		for tt := finRidersFrom; tt < finRidersFrom+60; tt += 1.0 / 60 {
			s := finRideAt(i, tt)
			if s != finRideAt(i, tt) {
				t.Fatalf("%s: not deterministic at %.2f", r.slug, tt)
			}
			if s.trick != finTrickNone && prevTrick == finTrickNone {
				seen[s.trick]++
				starts++
				maxGapT = math.Max(maxGapT, tt-lastStart)
				lastStart = tt
			}
			if s.trick == finTrickNone && s.h != 0 {
				t.Fatalf("%s off the ground with no trick at %.2f", r.slug, tt)
			}
			if s.trick != finTrickNone && math.Abs(s.p-0.5) < 0.01 && math.Abs(s.h-r.jumpH) > 1 {
				t.Errorf("%s: apex %v, want %v", r.slug, s.h, r.jumpH)
			}
			if (s.dir == DirLeft || s.dir == DirRight) && s.dir != s.dirOfMove {
				t.Fatalf("%s faces %v but goes %v at %.2f", r.slug, s.dir, s.dirOfMove, tt)
			}
			if prevX >= 0 && math.Abs(s.x-prevX) >= 2 && s.trick == finTrickNone {
				if (s.x > prevX) != (s.dirOfMove == DirRight) {
					// the tangent is taken over a few px: allow the turning points only
					if s.dir == DirLeft || s.dir == DirRight {
						x0, _ := finRidePos(i, s.d-30)
						x1, _ := finRidePos(i, s.d+30)
						if (x1 > x0) != (s.x > prevX) {
							continue
						}
						t.Fatalf("%s moves %v->%v but faces %v at %.2f", r.slug, prevX, s.x, s.dir, tt)
					}
				}
			}
			prevTrick, prevX = s.trick, s.x
		}
		for _, k := range want[r.slug] {
			if seen[k] == 0 {
				t.Errorf("%s never does trick %d in a minute (%v)", r.slug, k, seen)
			}
		}
		for k := range seen {
			found := false
			for _, w := range want[r.slug] {
				found = found || w == k
			}
			if !found {
				t.Errorf("%s does trick %d, not his", r.slug, k)
			}
		}
		if starts < 5 || maxGapT > 16 {
			t.Errorf("%s: %d tricks in a minute, longest wait %.1f s (want one every few seconds)", r.slug, starts, maxGapT)
		}
	}
	if len(speeds) != len(finRiders) {
		t.Errorf("the three ride at the same speed")
	}
	for a := 0; a < 3; a++ {
		for b := a + 1; b < 3; b++ {
			cross, prev := 0, 0.0
			for tt := finRidersFrom; tt < finRidersFrom+60; tt += 1.0 / 30 {
				d := finRideAt(a, tt).x - finRideAt(b, tt).x
				if prev != 0 && d != 0 && (d > 0) != (prev > 0) {
					cross++
				}
				if d != 0 {
					prev = d
				}
			}
			if cross < 4 {
				t.Errorf("%s and %s cross only %d times in a minute", finRiders[a].slug, finRiders[b].slug, cross)
			}
		}
	}
}

// Drawing them costs nothing new each frame: the art is built once, and the state and plan of a frame allocate nothing.
func TestFinRidersNoAlloc(t *testing.T) {
	a := finRideArtGet()
	if finRideArtGet() != a {
		t.Fatal("the art is rebuilt")
	}
	tt := 11.3
	if n := testing.AllocsPerRun(200, func() {
		tt += 0.37
		for i := range finRiders {
			s := finRideAt(i, tt)
			finRidePlan(i, &s)
			finRideBox(i, tt)
		}
	}); n != 0 {
		t.Errorf("%v allocations a frame in the riders' state", n)
	}
	g := layerGame(t, false)
	g.startEndingPre(true)
	g.end.stage, g.end.t = endFinal, 12
	img := ebiten.NewImage(screenW, screenH)
	defer img.Deallocate()
	g.drawFinRiders(img, g.end) // the kids' frames are cached on first use
	for k := 0; k < 400; k++ {
		g.end.t += 1.0 / 60
		g.drawFinRiders(img, g.end)
	}
	n := testing.AllocsPerRun(100, func() {
		g.end.t += 1.0 / 60
		g.drawFinRiders(img, g.end)
	})
	// what ebiten itself allocates to queue one DrawImage: the riders may cost 9 of those (3 images each), no more
	src := finRideArtGet().shadow[0]
	var op ebiten.DrawImageOptions
	base := testing.AllocsPerRun(100, func() { img.DrawImage(src, &op) })
	if n > 9*base {
		t.Errorf("%v allocations a frame drawing the riders, ebiten's 9 DrawImage alone cost %v", n, 9*base)
	}
	t.Logf("allocations a frame drawing the riders: %v (one DrawImage: %v)", n, base)
}

// The riders do not change when the way back to the menu opens (20 s secret, 8 s normal).
func TestFinRidersMenuLockUntouched(t *testing.T) {
	if finMenuOpen(true, finMenuAfter-0.01) || !finMenuOpen(true, finMenuAfter) || finMenuOpen(false, finMenuNorm-0.01) || !finMenuOpen(false, finMenuNorm) {
		t.Fatal("menu lock changed")
	}
}
