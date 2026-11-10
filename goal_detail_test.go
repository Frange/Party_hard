package main

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// goalDetailGame: the shared headless game (albumTestGame) with the detail window closed afterwards.
func goalDetailGame(t *testing.T) *Game {
	g := albumTestGame(t)
	prevSel, prevScroll, prevLang, prevTr := g.trophySel, g.trophyScroll, uiLang, g.progress.Trophies
	t.Cleanup(func() {
		goalDet = goalDetailState{}
		g.trophySel, g.trophyScroll, uiLang, g.progress.Trophies = prevSel, prevScroll, prevLang, prevTr
	})
	g.mode = ModeTrophies
	return g
}

func TestGoalDetailStepCycles(t *testing.T) {
	cases := []struct{ i, n, d, want int }{
		{0, 5, 1, 1}, {4, 5, 1, 0}, {0, 5, -1, 4}, {2, 5, -1, 1}, {3, 5, 0, 3}, {0, 1, 1, 0}, {0, 1, -1, 0}, {0, 0, 1, 0}, {7, 3, 1, 2},
	}
	for _, c := range cases {
		if got := goalDetailStep(c.i, c.n, c.d); got != c.want {
			t.Errorf("goalDetailStep(%d,%d,%d) = %d, want %d", c.i, c.n, c.d, got, c.want)
		}
	}
}

// every page: next walks through every element once (none skipped, none repeated) and comes back to the start,
// never leaving the page; previous from the first goes to the last; the list under the window follows the cursor.
func TestGoalDetailNavigationStaysInItsPage(t *testing.T) {
	g := goalDetailGame(t)
	g.albumDemo(1)
	for _, tab := range []int{tabTrophies, tabChars, tabVIPs, tabKillers, tabCops} {
		g.goTab(tab)
		n := g.goalDetailCount()
		if n < 2 {
			t.Fatalf("tab %d: %d elements", tab, n)
		}
		g.openGoalDetail()
		if !goalDet.open {
			t.Fatalf("tab %d: the window did not open", tab)
		}
		seen := map[int]bool{}
		for k := 0; k < n; k++ {
			i := g.goalDetailSel()
			if seen[i] {
				t.Fatalf("tab %d: element %d visited twice", tab, i)
			}
			seen[i] = true
			if tab == tabTrophies {
				if i < g.trophyScroll || i >= g.trophyScroll+trophyRows {
					t.Errorf("tab %d: trophy %d out of the list's view (scroll %d)", tab, i, g.trophyScroll)
				}
			} else if row := i / albumCols; row < g.albumScroll || row >= g.albumScroll+albumRows {
				t.Errorf("tab %d: card %d out of the grid's view (scroll %d)", tab, i, g.albumScroll)
			}
			g.goalDetailMove(1)
			if g.trophyTab != tab {
				t.Fatalf("the arrows changed the page: %d -> %d", tab, g.trophyTab)
			}
		}
		if len(seen) != n || g.goalDetailSel() != 0 {
			t.Errorf("tab %d: visited %d of %d, back at %d", tab, len(seen), n, g.goalDetailSel())
		}
		g.goalDetailMove(-1)
		if g.goalDetailSel() != n-1 {
			t.Errorf("tab %d: previous of the first = %d, want %d", tab, g.goalDetailSel(), n-1)
		}
		g.closeGoalDetail()
		if goalDet.open || g.goalDetailSel() != n-1 {
			t.Errorf("tab %d: closing must keep the element seen selected (%d)", tab, g.goalDetailSel())
		}
	}
	// CARRERA has no elements: no window
	g.goTab(tabCareer)
	g.openGoalDetail()
	if goalDet.open {
		t.Error("the window opened on CARRERA")
	}
}

// a card never met says nothing of who he is: no name, no role, no description, a silhouette that does not move.
func TestGoalDetailLockedDoesNotLeakName(t *testing.T) {
	g := goalDetailGame(t)
	for _, lang := range []string{"es", "en"} {
		uiLang = lang
		g.progress.Album = map[string]int{}
		g.progress.Trophies = map[string]bool{}
		for _, tab := range []int{tabChars, tabVIPs, tabKillers, tabCops} {
			g.goTab(tab)
			for i, c := range g.albumView() {
				g.albumSel = i
				ct := g.goalDetailContent()
				if ct.name != "???" || ct.sub != "???" || ct.ok || ct.seen || ct.seal != "" {
					t.Fatalf("[%s] locked %s shows %q / %q (ok=%v seen=%v seal=%q)", lang, c.key, ct.name, ct.sub, ct.ok, ct.seen, ct.seal)
				}
				for _, tx := range g.goalDetailTexts(ct) {
					for _, secret := range []string{tr(c.nameES), tr(c.cardName()), tr(c.roleES)} {
						if len([]rune(secret)) >= 4 && strings.Contains(tx.s, secret) && !strings.Contains(tr(c.lockedHint()), secret) {
							t.Errorf("[%s] locked %s leaks %q in %q", lang, c.key, secret, tx.s)
						}
					}
					if d := charDesc(c.slug); c.kind() == cardChar && d != "" && strings.Contains(tx.s, tr(d)) {
						t.Errorf("[%s] locked %s leaks his description", lang, c.key)
					}
				}
			}
		}
		// the secret trophies not won
		g.goTab(tabTrophies)
		for i, tp := range g.trophyView() {
			if !secretTrophy[tp.id] {
				continue
			}
			g.trophySel = i
			ct := g.goalDetailContent()
			for _, tx := range g.goalDetailTexts(ct) {
				if strings.Contains(tx.s, tr(tp.name)) || strings.Contains(tx.s, tr(tp.how)) {
					t.Errorf("[%s] secret trophy %s leaks %q", lang, tp.id, tx.s)
				}
			}
		}
	}
	// a character seen, or killed fewer times than needed, shows his face and name (as the card) but does not move
	uiLang = "es"
	c := charCard(t, "yoda")
	g.progress.Album = map[string]int{c.key: albumSeen | albumElim}
	g.progress.CharKills = map[string]int{c.key: 2}
	g.goTab(tabChars)
	for i, v := range g.albumView() {
		if v.key == c.key {
			g.albumSel = i
		}
	}
	ct := g.goalDetailContent()
	if ct.ok || !ct.seen || ct.name != c.nameES || !strings.HasPrefix(ct.seal, "ELIMINADO 2/") {
		t.Errorf("seen character: ok=%v seen=%v name=%q seal=%q", ct.ok, ct.seen, ct.name, ct.seal)
	}
	// won: he moves, with his own gait
	g.progress.Album[c.key] = albumKilled
	ct = g.goalDetailContent()
	if !ct.ok || ct.seal != "CONSEGUIDO" {
		t.Errorf("won character: ok=%v seal=%q", ct.ok, ct.seal)
	}
	if a, _ := goalDetailPose("michael_jackson", 1); a != AnimMoon {
		t.Errorf("a character with a gait walks with it: anim %d", a)
	}
	if a, _ := goalDetailPose("yoda", 1); a != AnimWalk {
		t.Errorf("the plain walk: anim %d", a)
	}
}

func TestGoalDetailHitTest(t *testing.T) {
	cases := []struct {
		x, y float64
		want int
		what string
	}{
		{10, 10, gdHitOutside, "top left of the screen"},
		{gdX - 1, gdY + 100, gdHitOutside, "just left of the window"},
		{gdX + gdW + 1, gdY + 100, gdHitOutside, "just right"},
		{gdX + 300, gdY + gdH + 1, gdHitOutside, "just below"},
		{60, 515, gdHitOutside, "the ATRÁS button"},
		{gdCloseX + 11, gdCloseY + 11, gdHitClose, "the X"},
		{gdPrevX + gdArrowW/2, gdArrowY + gdArrowH/2, gdHitPrev, "the left arrow"},
		{gdNextX + gdArrowW/2, gdArrowY + gdArrowH/2, gdHitNext, "the right arrow"},
		{gdPicX + 50, gdPicY + 50, gdHitInside, "the picture"},
		{gdTextX + 20, gdTextTop + 5, gdHitInside, "the text"},
	}
	for _, c := range cases {
		if got := goalDetailHit(c.x, c.y); got != c.want {
			t.Errorf("%s (%.0f,%.0f): hit %d, want %d", c.what, c.x, c.y, got, c.want)
		}
	}
}

// the window fits the screen and its parts do not touch; every text of every element (all cards in each state,
// all trophies won and not, both languages: the longest description included) stays in its column and above the
// bottom, at whole pixels and scales.
func TestGoalDetailGeometryAndTextsFit(t *testing.T) {
	g := goalDetailGame(t)
	type box struct {
		name       string
		x, y, w, h float64
	}
	in := func(a, b box) bool { return a.x >= b.x && a.y >= b.y && a.x+a.w <= b.x+b.w && a.y+a.h <= b.y+b.h }
	hit := func(a, b box) bool { return a.x < b.x+b.w && b.x < a.x+a.w && a.y < b.y+b.h && b.y < a.y+a.h }
	win := box{"window", gdX, gdY, gdW, gdH}
	if !in(win, box{"screen", 0, 0, screenW, screenH}) || gdY+gdH >= backBtnY {
		t.Fatal("the window leaves the screen or covers the ATRÁS button")
	}
	parts := []box{{"X", gdCloseX, gdCloseY, gdCloseS, gdCloseS}, {"prev", gdPrevX, gdArrowY, gdArrowW, gdArrowH}, {"next", gdNextX, gdArrowY, gdArrowW, gdArrowH},
		{"picture", gdPicX, gdPicY, gdPicW, gdPicH}, {"text", gdTextX, gdTextTop, gdTextW, gdTextBottom - gdTextTop}}
	for i, a := range parts {
		if !in(a, win) {
			t.Errorf("%s is not inside the window", a.name)
		}
		for _, b := range parts[i+1:] {
			if hit(a, b) {
				t.Errorf("%s and %s touch", a.name, b.name)
			}
		}
	}
	longest, longestKey := 0, ""
	check := func(label string) {
		ct := g.goalDetailContent()
		texts := g.goalDetailTexts(ct)
		for _, tx := range texts[1:] { // [0] is the head
			w := g.textW(tx.s, tx.scale)
			_, h := textMeasure(g, tx.s)
			if tx.x != float64(int(tx.x)) || tx.y != float64(int(tx.y)) || tx.scale != float64(int(tx.scale)) {
				t.Errorf("%s: %q not at whole pixels", label, tx.s)
			}
			if tx.x < gdTextX || tx.x+w > gdTextX+gdTextW || tx.y < gdTextTop || tx.y+h*tx.scale > gdTextBottom {
				t.Errorf("%s: %q (%.0f,%.0f w%.0f) leaves the text column", label, tx.s, tx.x, tx.y, w)
			}
		}
		head := texts[0]
		if head.x+g.textW(head.s, 1) >= gdCloseX || head.x < gdX {
			t.Errorf("%s: the head %q reaches the X", label, head.s)
		}
		for _, p := range ct.paras {
			if n := len([]rune(p.s)); n > longest {
				longest, longestKey = n, label
			}
		}
	}
	states := []struct {
		name string
		set  func(c albumCard)
	}{
		{"locked", func(c albumCard) {}},
		{"seen", func(c albumCard) {
			g.progress.Album[c.key] = albumSeen | albumElim
			g.progress.CharKills[c.key] = charKillsNeed - 1
		}},
		{"won", func(c albumCard) { g.progress.Album[c.key] = albumKilled | albumHelped }},
	}
	for _, lang := range []string{"es", "en"} {
		uiLang = lang
		for _, st := range states {
			g.progress.Album, g.progress.CharKills = map[string]int{}, map[string]int{}
			for _, c := range albumCards() {
				st.set(c)
			}
			for _, tab := range []int{tabChars, tabVIPs, tabKillers, tabCops} {
				g.goTab(tab)
				for i, c := range g.albumView() {
					g.albumSel = i
					check(lang + " " + st.name + " " + c.key)
				}
			}
		}
		for _, won := range []bool{false, true} {
			g.progress.Trophies = map[string]bool{}
			if won {
				for _, tp := range trophyList {
					g.progress.Trophies[tp.id] = true
				}
			}
			g.goTab(tabTrophies)
			for i, tp := range g.trophyView() {
				g.trophySel = i
				check(lang + " trophy " + tp.id)
			}
		}
	}
	if longest < 60 {
		t.Errorf("the longest paragraph checked has only %d letters (%s): the test did not see the long ones", longest, longestKey)
	}
	t.Logf("longest paragraph: %d letters (%s)", longest, longestKey)
	// it draws (every kind of picture) without panicking, and only while open
	dst := ebiten.NewImage(screenW, screenH)
	defer dst.Deallocate()
	for _, tab := range []int{tabTrophies, tabChars, tabKillers, tabCops, tabVIPs} {
		g.goTab(tab)
		g.openGoalDetail()
		goalDet.t = 2.3
		g.drawTrophies(dst)
	}
	g.closeGoalDetail()
	g.drawGoalDetail(dst)
}
