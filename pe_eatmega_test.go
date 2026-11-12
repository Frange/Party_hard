package main

import (
	"image/color"
	"math/rand"
	"strings"
	"testing"
)

// ---- the eating contest (pe_eatcontest.go) and the fat look (pe_eatfat.go) ----------------------------------------

func TestEatContestRegisteredOnlyWhereItFits(t *testing.T) {
	e := partyByID("comer")
	if e == nil || e.Prob < 0.10 || e.Prob > 0.12 || e.MaxRuns != 1 || e.Cooldown != 0 || !e.Big || e.Dur < 25 {
		t.Fatalf("comer: bad registration %+v", e)
	}
	g := partyGame(t)
	idxs := peEligible(g, "comer")
	if len(idxs) < 4 || len(idxs) > len(campaignLevels)/2 {
		t.Fatalf("comer fits %d campaign parties %v: want some, not most", len(idxs), idxs)
	}
	for _, i := range idxs {
		tr := partyTraitsOf(levelDefs[i], i)
		if !(tr.bbq && tr.outdoor) && !eatIcons[tr.icon] {
			t.Errorf("comer fits party %d (%s) without a barbecue nor a fitting venue", i, tr.icon)
		}
	}
	for _, icon := range []string{"house", "funfair", "festival"} {
		if i := campaignIdx(icon); i >= 0 && !e.Eligible(g, partyTraitsOf(levelDefs[i], i)) {
			t.Errorf("comer does not fit %s", icon)
		}
	}
	for _, icon := range []string{"station", "disco", "casino", "neon"} {
		if i := campaignIdx(icon); i >= 0 && e.Eligible(g, partyTraitsOf(levelDefs[i], i)) {
			t.Errorf("comer fits %s", icon)
		}
	}
	t.Logf("comer fits the campaign parties %v", idxs)
}

// noFatGuests: nobody in the party keeps the fat look (dead or alive).
func noFatGuests(g *Game) *Guest {
	for _, gu := range g.guests {
		if gu.cs != nil && gu.cs.fat {
			return gu
		}
	}
	return nil
}

// The contest: 2-4 fat contestants (none of them a new guest), they eat in bites, one wins, one throws up, and everybody gets
// his own look back and is let go. 1 and 2 players.
func TestEatContestRunsFatWinnerAndCleansUp(t *testing.T) {
	for _, two := range []bool{false, true} {
		peRetry(t, "comer", two, func(g *Game) string {
			s := g.partyRunning("comer").D.(*eatState)
			nGuests, total, decals := len(g.guests), g.total, len(g.decals)
			if n := len(s.parts); n < 2 || n > 4 {
				return "contestants: " + itoa(n)
			}
			parts := append([]*eatPart(nil), s.parts...)
			for _, p := range parts {
				if !p.m.gu.cs.fat || p.orig == nil || p.orig.fat {
					return "a contestant is not fat"
				}
			}
			touched := map[*Guest]bool{}
			sawEat, maxCrowd := false, 0
			for i := 0; i < 30*90 && g.partyRunning("comer") != nil; i++ {
				peStep(g, 1.0/30, true)
				if i%10 == 0 {
					peDraw(g)
				}
				peTrack(g, touched)
				if s.phase == ecEat {
					sawEat = true
					maxCrowd = maxI(maxCrowd, len(s.crowd.mem))
					for _, p := range s.parts {
						if !p.m.gu.cs.fat {
							return "a contestant lost his fat look while eating"
						}
					}
				}
				if len(s.parts) > 4 {
					return "more than four at the table"
				}
			}
			if g.partyRunning("comer") != nil {
				return "it did not end"
			}
			if s.fled || !sawEat {
				return "it broke up"
			}
			if s.winner == nil || s.loser == nil || s.winner == s.loser || s.winner.eaten < s.loser.eaten {
				return "no proper winner and loser"
			}
			if s.winner.eaten < 5 {
				return "the winner ate only " + itoa(s.winner.eaten)
			}
			if maxCrowd < 2 {
				return "nobody watched"
			}
			if !s.loser.choked || len(g.decals) <= decals {
				return "the loser did not throw up"
			}
			if gu := noFatGuests(g); gu != nil {
				return "a guest is still fat"
			}
			for _, p := range parts {
				if p.m.gu.cs != p.orig {
					return "a contestant did not get his own look back"
				}
			}
			if gu := peStale(g, touched); gu != nil {
				return "a guest is still held"
			}
			if len(g.guests) != nGuests || g.total != total {
				return "the contest changed the number of guests"
			}
			return ""
		})
	}
}

// Killing a contestant in the middle: the others and the audience run, it is over at once, and nobody stays fat (not
// even the body).
func TestEatContestKillBreaksItUp(t *testing.T) {
	peRetry(t, "comer", false, func(g *Game) string {
		s := g.partyRunning("comer").D.(*eatState)
		for i := 0; i < 30*30 && s.phase != ecEat && g.partyRunning("comer") != nil; i++ {
			peStep(g, 1.0/30, true)
		}
		if s.phase != ecEat {
			return "it never got to the eating"
		}
		peStep(g, 1.0/30, true)
		parts := append([]*eatPart(nil), s.parts...)
		watchers := append([]*pevMem(nil), s.crowd.mem...)
		g.killGuest(parts[0].m.gu, true)
		peStep(g, 1.0/30, false)
		if g.partyRunning("comer") != nil || !s.fled {
			return "it did not break up"
		}
		for _, p := range parts[1:] {
			if !p.m.gu.out() && (peHeld(p.m.gu) || p.m.gu.state == gFun) {
				return "a contestant is still at the table"
			}
		}
		for _, m := range watchers {
			if !m.gu.out() && peHeld(m.gu) {
				return "a watcher is still held"
			}
		}
		if gu := noFatGuests(g); gu != nil {
			return "a guest is still fat"
		}
		return ""
	})
}

// The fat look: much wider at the belly, a little at the face, with no new colours (whole pixels: sharp), in every view and
// lying down; the normal CharSet is not touched.
func TestEatFatFramesAreWiderAndSharp(t *testing.T) {
	g := peLoad(t, 0, false)
	checked := 0
	for _, gu := range g.guests {
		if checked >= 9 || !fatRoom(gu.cs) { // the contest only takes the ones the fat look shows on (eatCanBe)
			continue
		}
		base := gu.cs
		fat := fatCharSet(base)
		if fat == base || base.fat || !fat.fat || fatCharSet(fat) != fat {
			t.Fatal("fatCharSet must copy once")
		}
		for _, vw := range []int{vFront, vBack, vSide} {
			var b *pixBuf
			if base.rig != nil {
				b = base.rigFrame(AnimIdle, vw, 0)
			} else {
				pp := poseFor(AnimIdle, 0)
				b = genChar(base.Spec, vw, pp)
			}
			f := fattenFrame(b, AnimIdle, vw)
			top, bot := opaqueRows(b)
			if top < 0 {
				continue
			}
			belly := top + (bot-top)*52/100
			wb, wf := rowWidth(b, belly), rowWidth(f, belly)
			if wf < wb+5 {
				t.Errorf("%s view %d: belly %d -> %d px, not clearly fat", guestSlug(gu), vw, wb, wf)
			}
			if rowWidth(f, top) != rowWidth(b, top) {
				t.Errorf("%s view %d: the top of the head changed", guestSlug(gu), vw)
			}
			pal := map[color.RGBA]bool{}
			for i := 0; i < len(b.p); i += 4 {
				pal[color.RGBA{b.p[i], b.p[i+1], b.p[i+2], b.p[i+3]}] = true
			}
			for i := 0; i < len(f.p); i += 4 {
				if c := (color.RGBA{f.p[i], f.p[i+1], f.p[i+2], f.p[i+3]}); !pal[c] {
					t.Fatalf("%s view %d: a new colour %v (blurred)", guestSlug(gu), vw, c)
				}
			}
			checked++
		}
		if fat.frameImg(AnimLying, 0, 0) == nil || fat.frameImg(AnimWalk, vSide, 1) == nil {
			t.Error("a fat frame is not drawn")
		}
	}
	if checked < 6 {
		t.Fatalf("only %d frames checked", checked)
	}
}

func opaqueRows(b *pixBuf) (top, bot int) {
	top, bot = -1, -1
	for y := 0; y < b.h; y++ {
		if rowWidth(b, y) > 0 {
			if top < 0 {
				top = y
			}
			bot = y
		}
	}
	return
}

func rowWidth(b *pixBuf, y int) int {
	x0, x1 := -1, -1
	for x := 0; x < b.w; x++ {
		if b.p[(y*b.w+x)*4+3] != 0 {
			if x0 < 0 {
				x0 = x
			}
			x1 = x
		}
	}
	if x0 < 0 {
		return 0
	}
	return x1 - x0 + 1
}

// ---- the megaphone (pe_megaphone.go) ----------------------------------------------------------------------------

func TestMegaphoneRegisteredOnlyWhereItFits(t *testing.T) {
	e := partyByID("megafono")
	if e == nil || e.Prob < 0.10 || e.Prob > 0.12 || e.Cooldown < 180 || e.MaxRuns > 2 || e.Group != "speech" {
		t.Fatalf("megafono: bad registration %+v", e)
	}
	if len(megaLines) < 15 || len(megaLines) > 20 {
		t.Errorf("%d announcements, want 15-20", len(megaLines))
	}
	g := partyGame(t)
	idxs := peEligible(g, "megafono")
	if len(idxs) < 5 || len(idxs) >= len(campaignLevels)*2/3 {
		t.Fatalf("megafono fits %d campaign parties %v: want some, not all", len(idxs), idxs)
	}
	for _, icon := range []string{"station", "cinema", "casino", "disco"} {
		if i := campaignIdx(icon); i >= 0 && e.Eligible(g, partyTraitsOf(levelDefs[i], i)) {
			t.Errorf("megafono fits %s", icon)
		}
	}
	t.Logf("megafono fits the campaign parties %v", idxs)
}

// Every announcement fits a bubble (two lines at most, nothing cut) in Spanish and in English, and its English is not the
// Spanish.
func TestMegaphoneLinesFitInBothLanguages(t *testing.T) {
	g := partyGame(t)
	prev := uiLang
	t.Cleanup(func() { uiLang = prev })
	for _, lang := range []string{"es", "en"} {
		uiLang = lang
		for _, s := range append(append([]string(nil), megaLines...), megaFirst...) {
			txt := tr(s)
			if lang == "en" && txt == s {
				t.Errorf("%q has no English", s)
			}
			lines := wrapBubble(txt, bubMaxW, func(x string) float64 { return g.textW(x, 1) }, bubLines)
			if strings.Join(lines, " ") != strings.Join(strings.Fields(txt), " ") {
				t.Errorf("%s: %q is cut: %q", lang, txt, lines)
			}
			for _, l := range lines {
				if w := g.textW(l, 1) + 1 + 12; w > bubMaxW+12 { // bold + frame
					t.Errorf("%s: line %q is %.0f px", lang, l, w)
				}
			}
		}
	}
}

// The speaker walks to his spot, 3-5 announcements in loud bubbles, the guests around come closer, face him and react;
// at the end everybody is let go. 1 and 2 players.
func TestMegaphoneAnnouncesReactsAndReleases(t *testing.T) {
	for _, two := range []bool{false, true} {
		peRetry(t, "megafono", two, func(g *Game) string {
			s := g.partyRunning("megafono").D.(*megaState)
			nGuests := len(g.guests)
			touched := map[*Guest]bool{}
			loud, maxCrowd, faced, near := false, 0, 0, 0
			reacts := map[int]bool{}
			for i := 0; i < 30*70 && g.partyRunning("megafono") != nil; i++ {
				peStep(g, 1.0/30, true)
				if i%10 == 0 {
					peDraw(g)
				}
				peTrack(g, touched)
				if s.phase == mgTalk && g.time < s.shoutEnd {
					if g.bubLoud(s.m.gu) {
						loud = true
					}
					if n := len(s.crowd.mem); n > maxCrowd {
						maxCrowd = n
					}
					for _, m := range s.crowd.mem {
						reacts[s.react[m.gu]] = true
						if m.gu.state == gIdle && m.gu.timer > 1000 {
							if m.gu.dir == faceTo(m.gu.pos, s.m.gu.pos) {
								faced++
							}
							if dist(m.gu.pos, s.m.gu.pos) < 56 {
								near++
							}
						}
					}
				}
				if g.bubLoud(s.m.gu) && (s.phase != mgTalk || g.time >= s.shoutEnd) {
					return "a loud bubble out of an announcement"
				}
			}
			if g.partyRunning("megafono") != nil {
				return "it did not end"
			}
			if s.phase == mgDrop {
				return "the megaphone fell"
			}
			if s.n < 3 || s.n > 5 {
				return "announcements: " + itoa(s.n)
			}
			if !loud {
				return "no loud bubble"
			}
			if maxCrowd < 3 || faced == 0 || near == 0 || len(reacts) < 2 {
				return "the guests around did not come, look or react"
			}
			if gu := peStale(g, touched); gu != nil {
				return "a guest is still held"
			}
			if g.bubLoud(s.m.gu) {
				return "the loud bubble stayed"
			}
			if len(g.guests) != nGuests {
				return "the number of guests changed"
			}
			return ""
		})
	}
}

// Killing the speaker: the megaphone falls, the audience runs, and it ends a few seconds later.
func TestMegaphoneKillDropsIt(t *testing.T) {
	peRetry(t, "megafono", false, func(g *Game) string {
		s := g.partyRunning("megafono").D.(*megaState)
		for i := 0; i < 30*40 && s.n < 1 && g.partyRunning("megafono") != nil; i++ {
			peStep(g, 1.0/30, true)
		}
		if s.n < 1 {
			return "no announcement"
		}
		watchers := append([]*pevMem(nil), s.crowd.mem...)
		g.killGuest(s.m.gu, true)
		peStep(g, 1.0/30, false)
		if s.phase != mgDrop {
			return "the megaphone did not fall"
		}
		for _, m := range watchers {
			if !m.gu.out() && m.gu.state != gScared {
				return "a listener did not run"
			}
		}
		peDraw(g)
		for i := 0; i < 30*8 && g.partyRunning("megafono") != nil; i++ {
			peStep(g, 1.0/30, true)
		}
		if g.partyRunning("megafono") != nil {
			return "it did not end after the fall"
		}
		return ""
	})
}

// The loud bubble of a speaker standing right under the HUD never covers it (1 and 2 players, ES and EN).
func TestMegaphoneBubbleKeepsOffHUD(t *testing.T) {
	for _, two := range []bool{false, true} {
		g := hudClearGame(t, 0, two)
		prevMgr, prevPev := bubbleMgr, g.pev
		t.Cleanup(func() { bubbleMgr, g.pev = prevMgr, prevPev })
		bubbleMgr = bubMgr{}
		for _, gu := range g.guests {
			gu.icon = ""
		}
		talker := g.guests[0]
		said := megaLines[2] // the longest one
		hr0 := Rect{hudX, hudY, hudW, 100}
		talker.pos = screenToWorld(g, hr0.X+70, hr0.Y+hr0.H+34)
		talker.icon, talker.iconT = said, 60
		s := &megaState{m: &pevMem{gu: talker, want: gIdle}, phase: mgTalk, said: said, shoutEnd: 1e9, react: map[*Guest]int{}, side: DirRight}
		g.pev = pevState{ready: true, act: []*partyAct{{ev: partyByID("megafono"), D: s}}}
		if !g.bubLoud(talker) {
			t.Fatal("the speaker's bubble is not loud")
		}
		setup := func(g *Game) { g.lvl.graded = true }
		for _, lang := range []string{"es", "en"} {
			for frame := 0; frame < 3; frame++ {
				bubbleMgr.frame += 60
				rec, err := auditRender(g, auditShot{name: "megafono", setup: setup}, lang)
				if err != "" {
					t.Fatal(err)
				}
				hr, ok := g.hudBlockRect()
				if !ok {
					t.Fatal("the HUD is not on screen")
				}
				words := strings.Fields(tr2(lang, said))
				for _, tx := range rec.texts {
					if len(words) == 0 || !strings.HasPrefix(tx.s, words[0]) && !strings.HasSuffix(tx.s, words[len(words)-1]) {
						continue
					}
					if r := (Rect{tx.x, tx.y, tx.w, tx.h}); rectsOverlap(r, hr) {
						t.Errorf("2P=%v %s: the loud bubble %q %v covers the HUD %v", two, lang, tx.s, r, hr)
					}
				}
			}
		}
	}
}

// tr2: the text in a language (tr reads uiLang).
func tr2(lang, s string) string {
	prev := uiLang
	uiLang = lang
	defer func() { uiLang = prev }()
	return tr(s)
}

// Both events really start (without the debug flag) in every campaign party they fit, in at least one of a few parties.
func TestEatAndMegaphoneStartInTheirLevels(t *testing.T) {
	for _, id := range []string{"comer", "megafono"} {
		probe := partyGame(t)
		for _, idx := range peEligible(probe, id) {
			ok := false
			for seed := int64(1); seed <= 4 && !ok; seed++ {
				peLoadSeed = seed
				g := peLoad(t, idx, false)
				peLoadSeed = 0
				partyRng = rand.New(rand.NewSource(seed*7919 + int64(idx)))
				peForce(g, id)
				dbgPartyEvent = "" // the real path: a spot anywhere in the party, the guests around it
				for s := 0; s < 30*40 && g.partyRunning(id) == nil && len(g.pev.plan) > 0; s++ {
					peStep(g, 1.0/30, true)
				}
				if a := g.partyRunning(id); a != nil {
					ok = true
					switch st := a.D.(type) {
					case *eatState:
						eatEnd(g, st)
					case *megaState:
						megaEnd(g, st)
					}
				}
			}
			if !ok {
				t.Errorf("%s never starts in campaign party %d (%s)", id, idx, levelDefs[idx].Icon)
			}
		}
	}
}
