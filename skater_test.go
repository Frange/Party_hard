package main

import (
	"math"
	"math/rand"
	"testing"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
)

var skateParkIdx = campaignIdx("skate") // the SKATE PARK in the campaign (by icon: indices move when parties are inserted)

func skateKids(g *Game) map[string]*Guest {
	m := map[string]*Guest{}
	for _, gu := range g.guests {
		if !gu.out() && gu.cs != nil && gu.cs.Spec != nil && isKidSlug(gu.cs.Spec.Name) {
			m[gu.cs.Spec.Name] = gu
		}
	}
	return m
}

// In the skate park the three kids are always there from the start (several seeds, 1 and 2 players), once each,
// within the caps; every 8-20 s (plus the visit itself) one of them, not the same one twice in a row, rides into
// a bowl, does his tricks there and rolls out; nobody gets stuck on the way, nobody stands inside a wall. A kid who
// is not there at the end was killed or taken by an event (and never came back).
func TestSkateParkThreeKidsAndBowlTurns(t *testing.T) {
	for _, two := range []bool{false, true} {
		for _, seed := range []int64{1, 2, 3, 4} {
			g := playGame(t, skateParkIdx, seed)
			if two {
				twoPlayerMode, dbgTwoPlayers = true, true
				g.rng = rand.New(rand.NewSource(seed))
				g.loadLevel(skateParkIdx)
				g.mode, g.god, g.time = ModePlay, true, 30
			}
			if g.lvl.Def.Title != "SKATE PARK" || len(g.lvl.bowls) == 0 {
				t.Fatalf("level %d is %q", skateParkIdx, g.lvl.Def.Title)
			}
			maxTotal := g.goalMax()
			if k := skateKids(g); len(k) != 3 {
				t.Fatalf("[two=%v seed %d] the skate park starts with %d of the three kids", two, seed, len(k))
			}
			p0 := g.p.pos
			var walkT = map[*Guest]float64{}
			maxAlive, dupSlug, dupT := 0, false, 0.0
			lastVisit, visits, longest := g.time, 0, 0.0
			var lastTurnStart float64
			for i := 0; i < 60*150; i++ {
				g.p.pos = p0 // he stays at the start: the kids are not scared off all the time
				g.updatePlay(1.0 / 60)
				if skateTurn.visits != visits {
					visits = skateTurn.visits
					longest = math.Max(longest, g.time-lastVisit)
					lastVisit = g.time
					lastTurnStart = g.time
				}
				for gu, st := range kidSessions {
					if gu.out() || st.pr == nil {
						continue
					}
					if !containsGuest(g.guests, gu) {
						continue
					}
					if st.stage == 0 {
						walkT[gu] += 1.0 / 60
						if walkT[gu] > kidBowlWalkMax+1 {
							t.Fatalf("[two=%v seed %d] a kid has been on his way to a bowl for %.0f s", two, seed, walkT[gu])
						}
					} else {
						walkT[gu] = 0
					}
				}
				alive := 0
				bySlug := map[string]int{}
				for _, gu := range g.guests {
					if !gu.out() && gu.cs != nil && gu.cs.Spec != nil && isKidSlug(gu.cs.Spec.Name) {
						alive++
						bySlug[gu.cs.Spec.Name]++
						if bySlug[gu.cs.Spec.Name] > 1 {
							dupT += 1.0 / 60
						}
					}
				}
				maxAlive = max(maxAlive, alive)
				dupSlug = dupSlug || dupT > 0.5 // (an arrival dressed as one of them is made plain at once)
				for _, gu := range skateKids(g) {
					if inSolid(g.lvl, gu.pos) {
						t.Fatalf("[two=%v seed %d] %s inside a wall at (%.0f,%.0f)", two, seed, gu.cs.Spec.Name, gu.pos.X, gu.pos.Y)
					}
				}
				if g.total > maxTotal {
					t.Fatalf("[two=%v seed %d] %d guests in the party, the cap is %d", two, seed, g.total, maxTotal)
				}
			}
			_ = lastTurnStart
			kids := skateKids(g)
			for _, slug := range kidSlugs { // a kid who is not there any more was killed (by a killer, a trap...)
				if kids[slug] != nil {
					continue
				}
				out := false // killed, or taken by an event of the party (a UFO, the T-Rex...)
				for _, gu := range g.guests {
					out = out || (gu.out() && gu.cs != nil && gu.cs.Spec != nil && gu.cs.Spec.Name == slug)
				}
				if !out {
					t.Errorf("[two=%v seed %d] %s is not in the skate park (and he was not killed or taken)", two, seed, slug)
				}
			}
			if dupSlug {
				t.Errorf("[two=%v seed %d] two of the same kid at once (%.1f s), %d kids alive at most", two, seed, dupT, maxAlive)
			}
			if len(kids) == 3 && (skateTurn.visits < 6 || longest > 40) { // (with fewer left, fewer can go)
				t.Errorf("[two=%v seed %d] %d visits to the bowls in 150 s (longest wait %.0f s)", two, seed, skateTurn.visits, longest)
			}
			if skateTurn.repeatSlug > 0 {
				t.Errorf("[two=%v seed %d] the same kid went to the bowl twice in a row %d times", two, seed, skateTurn.repeatSlug)
			}
			t.Logf("[two=%v seed %d] visits %d, longest wait %.1f s, total %d/%d, kids alive %d, turns to the same kid after a long wait %d",
				two, seed, skateTurn.visits, longest, g.total, maxTotal, len(kids), skateTurn.graceUsed)
		}
	}
}

// inSolid: the point is well inside something solid (2 px in: brushing an edge is not being stuck in it).
func inSolid(l *Level, p Vec) bool {
	for _, r := range l.solidsP {
		if p.X > r.X+2 && p.X < r.X+r.W-2 && p.Y > r.Y+2 && p.Y < r.Y+r.H-2 {
			return true
		}
	}
	return false
}

func containsGuest(l []*Guest, gu *Guest) bool {
	for _, o := range l {
		if o == gu {
			return true
		}
	}
	return false
}

// A kid killed in the skate park stays dead: he does not come back, the others go on with their turns, and
// the party's count is not changed by it.
func TestSkateParkKidDiesNoRespawn(t *testing.T) {
	g := playGame(t, skateParkIdx, 9)
	runPlay(g, 20, func() bool { return len(skateKids(g)) == 3 })
	kids := skateKids(g)
	if len(kids) != 3 {
		t.Fatalf("only %d kids", len(kids))
	}
	victim := kids["cesar"]
	g.killGuest(victim, true)
	total := g.total
	v0 := skateTurn.visits
	p0 := g.p.pos
	for i := 0; i < 60*90; i++ {
		g.p.pos = p0
		g.updatePlay(1.0 / 60)
		if g.kidPresent("cesar") {
			t.Fatal("César came back after dying")
		}
	}
	if skateTurn.visits < v0+2 {
		t.Errorf("the turns stopped after César died (%d -> %d visits)", v0, skateTurn.visits)
	}
	if g.total < total {
		t.Errorf("the party's total went down: %d -> %d", total, g.total)
	}
}

// Every kid of the three does every bowl trick of his board in a bowl, and it draws without trouble on every
// frame; the board stays under his feet (it turns with him); scooters for César and Sam-sam.
func TestBowlTricksDraw(t *testing.T) {
	g := layerGame(t, false)
	lvl, rng, guests := g.lvl, g.rng, g.guests
	t.Cleanup(func() { g.lvl, g.rng, g.mode, g.guests = lvl, rng, ModeMenu, guests })
	sg := skateGame(11)
	g.lvl, g.rng, g.mode = sg.lvl, sg.rng, ModePlay
	img := ebiten.NewImage(screenW, screenH)
	defer img.Deallocate()
	var bowl *Prop
	for _, p := range g.lvl.Props {
		if p.Name == "skatebowl" {
			bowl = p
			break
		}
	}
	for _, slug := range kidSlugs {
		gu := kidGuest(g, slug, Vec{bowl.X, bowl.Y - 30})
		st := newSkaterSt(gu, bowl, rand.New(rand.NewSource(3)))
		if st.scooter != (slug != "diego") {
			t.Errorf("%s: scooter %v", slug, st.scooter)
		}
		st.stage = 2
		kidSessions[gu] = st
		tricks := []int{trOllie, trSpin180, trSpin360, trGrab, trKickflip, trFlip, trHandplant, trSpin540, trHeelflip}
		if st.scooter {
			tricks = []int{trOllie, trSpin360, trTailwhip, trBarspin, trNoHander, trHandplant, trSpin540, trFlip}
		}
		for _, k := range tricks {
			sawUp := false
			for ph := math.Pi/2 - 0.3; ph < math.Pi/2+0.3; ph += 0.01 {
				st.ph, st.curN, st.trick, st.fallT = ph, 0, k, 0
				st.z, st.ang, st.bang, st.pose, st.view = 0, 0, 0, 0, finScSide
				st.ride(g)
				gu.pos = Vec{st.cx + st.x, st.cy + st.y}
				g.drawSkater(img, gu)
				if k == trHandplant && math.Abs(math.Mod(math.Abs(st.ang*st.face), 2*math.Pi)-math.Pi) < 0.3 {
					sawUp = true
				}
			}
			if k == trHandplant && !sawUp {
				t.Errorf("%s: no upside-down frame in a handplant", slug)
			}
		}
		delete(kidSessions, gu)
	}
}

// The kids' lines: short (the card's little bubble), in both languages, not repeated, and they fit the bubble.
func TestSkaterLinesFit(t *testing.T) {
	g := layerGame(t, false)
	prev := uiLang
	t.Cleanup(func() { uiLang = prev })
	for _, slug := range kidSlugs {
		set := skaterLines(slug)
		if len(set) < 15 {
			t.Errorf("%s: only %d lines", slug, len(set))
		}
		seen := map[string]bool{}
		for _, p := range set {
			if p[0] == "" || p[1] == "" || p[0] == p[1] || seen[p[0]] || seen[p[1]] {
				t.Errorf("%s: bad or repeated line %q / %q", slug, p[0], p[1])
			}
			seen[p[0]], seen[p[1]] = true, true
			for _, l := range []string{"es", "en"} {
				uiLang = l
				s := tr(p[0])
				if (l == "en") != (s == p[1]) || utf8.RuneCountInString(s) > 32 {
					t.Errorf("%s [%s]: %q -> %q", slug, l, p[0], s)
				}
				if w := g.textW(s, 1); w+2*finSayPad > finSayMaxW {
					t.Errorf("%s [%s]: %q is %.0f px, the bubble holds %.0f", slug, l, s, w, finSayMaxW-2*finSayPad)
				}
			}
		}
	}
	for k, p := range skaterTrickShout {
		if p[0] == "" || p[1] == "" || p[0] == p[1] || utf8.RuneCountInString(p[0]) > 24 {
			t.Errorf("trick %d: bad shout %q / %q", k, p[0], p[1])
		}
	}
}

// On the last card the kids talk in little bubbles: only in the empty sides, >= 16 px from the edges, >= 12 px
// from every text of the card, from Murray and his biggest bubble and from each other's bubbles; each of them
// talks several times in three minutes and says all his lines before repeating one.
func TestFinRiderSaysLayout(t *testing.T) {
	prev := uiLang
	t.Cleanup(func() { uiLang = prev; textRec = nil })
	for _, trueEnd := range []bool{true, false} {
		for _, lang := range []string{"es", "en"} {
			g := layerGame(t, false)
			uiLang = lang
			g.startEndingPre(trueEnd)
			var obst []finRect
			for _, tt := range []float64{10, 30} {
				g.end.stage, g.end.t, g.end.total = endFinal, tt, tt
				textRec = &textRecorder{}
				img := ebiten.NewImage(screenW, screenH)
				g.drawEnding(img)
				img.Deallocate()
				for _, r := range textRec.texts {
					if r.kind != "riderSay" {
						obst = append(obst, finRect{r.x, r.y, r.x + r.w, r.y + r.h})
					}
				}
				textRec = nil
			}
			im := g.as.props["murray"][0]
			footX, footY, _, _, _ := finMurrayGeom(trueEnd)
			sw, sh := float64(im.img.Bounds().Dx())*4, float64(im.img.Bounds().Dy())*4
			obst = append(obst, finRect{footX - im.ox*4, footY - im.oy*4, footX - im.ox*4 + sw, footY - im.oy*4 + sh})
			x0, y0, x1, y1 := finBubbleBox(trueEnd, finBubWrap, 3)
			obst = append(obst, finRect{x0, y0, x1, y1})
			said := [3][]int{}
			for tt := 0.0; tt < 183; tt += 1.0 / 30 {
				var on []finRect
				for i := range finRiders {
					ev, ok := finSayAt(trueEnd, i, tt)
					if !ok {
						continue
					}
					if n := len(said[i]); n == 0 || said[i][n-1] != ev.n {
						said[i] = append(said[i], ev.n)
					}
					line := tr(finSayLine(i, ev.n))
					bx, by, bw, bh, tip, ok := finSayRect(i, tt, g.textW(line, 1))
					if !ok {
						t.Fatalf("[%s end=%v] rider %d talks at %.2f with no room", lang, trueEnd, i, tt)
					}
					b := finRect{bx - 2, by - 2, bx + bw + 2, by + bh + finSayTail}
					if b.x0 < 16 || b.y0 < 16 || b.x1 > screenW-16 || b.y1 > screenH-16 || bx != math.Round(bx) || by != math.Round(by) {
						t.Fatalf("[%s end=%v] rider %d's bubble %v at %.2f: off the margins or not on whole pixels", lang, trueEnd, i, b, tt)
					}
					if tip < bx || tip > bx+bw {
						t.Fatalf("[%s end=%v] rider %d's bubble does not point at him", lang, trueEnd, i)
					}
					for _, o := range obst {
						if d := b.gap(o); d < 12 {
							t.Fatalf("[%s end=%v] rider %d's bubble %v at %.2f is %.0f px from %v", lang, trueEnd, i, b, tt, d, o)
						}
					}
					for _, o := range on {
						if d := b.gap(o); d < 12 {
							t.Fatalf("[%s end=%v] two kids' bubbles %.0f px apart at %.2f", lang, trueEnd, d, tt)
						}
					}
					on = append(on, b)
				}
			}
			for i, s := range said {
				if len(s) < 4 {
					t.Errorf("[%s end=%v] %s talks only %d times in three minutes", lang, trueEnd, finRiders[i].slug, len(s))
				}
				n := len(skaterLines(finRiders[i].slug))
				seen := map[string]bool{}
				for k, idx := range s {
					if k >= n {
						break
					}
					l := finSayLine(i, idx)
					if seen[l] {
						t.Errorf("%s repeats %q before saying all his lines", finRiders[i].slug, l)
					}
					seen[l] = true
				}
			}
			t.Logf("[%s end=%v] lines said in 3 min: %d %d %d", lang, trueEnd, len(said[0]), len(said[1]), len(said[2]))
		}
	}
}

// Every trick of the card's repertoire draws on every frame (both cards), with the kids' bubbles.
func TestFinRidersAllTricksDraw(t *testing.T) {
	g := layerGame(t, false)
	img := ebiten.NewImage(screenW, screenH)
	defer img.Deallocate()
	for _, trueEnd := range []bool{true, false} {
		g.startEndingPre(trueEnd)
		g.end.stage = endFinal
		seen := map[int]bool{}
		for tt := finRidersFrom; tt < finRidersFrom+240; tt += 1.0 / 20 {
			g.end.t, g.end.total = tt, tt
			g.drawFinRiders(img, g.end)
			g.drawFinRiderSays(img, g.end)
			for i := range finRiders {
				seen[finRideAt(i, tt).trick] = true
			}
		}
		for k := finTrickKickflip; k < finTrickCount; k++ {
			if !seen[k] {
				t.Errorf("trick %d never shows on the card", k)
			}
		}
	}
}

// The new Murray lines of the game: short for the bubble, translated, no duplicates.
func TestMurrayMoreLines(t *testing.T) {
	prev := uiLang
	t.Cleanup(func() { uiLang = prev })
	seen := map[string]bool{}
	for _, l := range murrayLines {
		if seen[l] {
			t.Errorf("Murray's line repeated: %q", l)
		}
		seen[l] = true
	}
	for _, p := range murrayLinesMore {
		uiLang = "en"
		if tr(p[0]) != p[1] || utf8.RuneCountInString(p[0]) > 48 || utf8.RuneCountInString(p[1]) > 48 {
			t.Errorf("bad line %q / %q", p[0], p[1])
		}
	}
	if len(finMurrayLines) < 60 || len(finMurrayLinesNormal) < 40 {
		t.Errorf("Murray's end-card lines: %d and %d", len(finMurrayLines), len(finMurrayLinesNormal))
	}
}
