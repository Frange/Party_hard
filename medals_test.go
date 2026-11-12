package main

import (
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// medalGame: the shared test game in one-player play, with its progress put back afterwards.
func medalGame(t *testing.T) *Game {
	t.Helper()
	g := hudPlay(t, false)
	snapshotGlobals(t)
	progressFrozen = true
	if len(levelDefs) < campaignN || campaignN == 0 {
		t.Fatal("no campaign")
	}
	tl := append([]Trophy(nil), trophyList...)
	t.Cleanup(func() {
		trophyList = tl
		g.run = runStats{}
		g.dc, g.challenge, g.stealth = nil, false, false
		g.toasts = nil
		g.loadLevel(0)
	})
	return g
}

// One function, monotonic, coherent with the kill target of every party.
func TestMedalThresholdsMonotonic(t *testing.T) {
	prev := [3]int{}
	for goal := 1; goal <= 200; goal++ {
		th := medalThresholds(goal)
		if !(th[0] > 0 && th[0] < th[1] && th[1] < th[2]) {
			t.Fatalf("goal %d: %v not bronze < silver < gold", goal, th)
		}
		for i := range th {
			if th[i]%50 != 0 || th[i] < prev[i] {
				t.Fatalf("goal %d: %v (previous %v): not round or going down", goal, th, prev)
			}
		}
		if th[2] >= medalReasonableMax(goal) {
			t.Fatalf("goal %d: gold %d above the reasonable max", goal, th[2])
		}
		prev = th
	}
	g := medalGame(t)
	for i := 0; i < campaignN; i++ {
		def := levelDefs[i]
		want := goalCap + bigCrowd[def.Icon]
		if got := g.medalGoalFor(def); got != want {
			t.Errorf("%s: medal goal %d, its kill target is %d", def.Title, got, want)
		}
		if g.medalGoalFor(def) > goalCap && medalThresholds(g.medalGoalFor(def))[2] <= medalThresholds(goalCap)[2] {
			t.Errorf("%s: a bigger party should ask for more points", def.Title)
		}
	}
	// the real target of a loaded party is what the hint and the medal use
	for _, i := range []int{0, campaignIdx("funfair")} {
		g.loadLevel(i)
		if g.partyMedalGoal() != g.goalMax() || g.partyMedalGoal() != g.medalGoalFor(g.lvl.Def) {
			t.Errorf("party %d: medal goal %d, goalMax %d, list %d", i, g.partyMedalGoal(), g.goalMax(), g.medalGoalFor(g.lvl.Def))
		}
	}
}

func TestMedalForPoints(t *testing.T) {
	for _, goal := range []int{65, 95, 110} {
		th := medalThresholds(goal)
		cases := map[int]int{0: medalNone, th[0] - 1: medalNone, th[0]: medalBronze, th[1] - 1: medalBronze, th[1]: medalSilver,
			th[2] - 1: medalSilver, th[2]: medalGold, th[2] * 3: medalGold}
		for pts, want := range cases {
			if got := medalFor(pts, goal); got != want {
				t.Errorf("goal %d, %d points: medal %d, want %d", goal, pts, got, want)
			}
		}
	}
}

// The parties with a medal are the campaign list as it is (the island never), whatever is inserted later.
func TestMedalLevelsAreTheCampaign(t *testing.T) {
	ls := medalLevels()
	if len(ls) != len(campaignLevels) {
		t.Fatalf("%d medal parties for %d campaign parties", len(ls), len(campaignLevels))
	}
	for _, k := range ls {
		if k == "jurassic" || k == "" {
			t.Errorf("party %q gives a medal", k)
		}
	}
	if medalLevel("jurassic") {
		t.Error("the island gives a medal")
	}
}

// Won only when better, only campaign parties, never the RETO, the stealth mode, the random mode or the island.
func TestMedalAwardOnlyImproves(t *testing.T) {
	g := medalGame(t)
	g.progress.Medals = nil
	g.loadLevel(2)
	icon := g.lvl.Def.Icon
	th := medalThresholds(g.goalMax())
	g.points = th[1]
	g.awardMedal()
	if g.medalOf(icon) != medalSilver || !g.run.medalNew || g.run.medal != medalSilver || !g.run.medalOn {
		t.Fatalf("silver: saved %d, run %+v", g.medalOf(icon), g.run.medal)
	}
	g.points = th[0]
	g.awardMedal()
	if g.medalOf(icon) != medalSilver || g.run.medalNew || g.run.medal != medalBronze {
		t.Fatalf("a worse party changed the medal: saved %d, new %v", g.medalOf(icon), g.run.medalNew)
	}
	g.points = th[2]
	g.awardMedal()
	if g.medalOf(icon) != medalGold || !g.run.medalNew {
		t.Fatalf("gold not saved: %d", g.medalOf(icon))
	}
	if len(g.progress.Medals) != 1 {
		t.Errorf("other parties got medals: %v", g.progress.Medals)
	}
	// no medal: RETO, challenge, stealth, a random roll
	g.progress.Medals = nil
	g.points = th[2] * 2
	for name, set := range map[string]func(){
		"reto":    func() { g.dc = &dailyChallenge{} },
		"challe":  func() { g.challenge = true },
		"stealth": func() { g.stealth = true },
		"random":  func() { g.levelIdx = campaignN },
		"island":  func() { g.levelIdx = specialJurassic },
	} {
		g.loadLevel(2)
		set()
		g.awardMedal()
		if len(g.progress.Medals) != 0 || g.run.medalOn {
			t.Errorf("%s gave a medal: %v", name, g.progress.Medals)
		}
		g.dc, g.challenge, g.stealth = nil, false, false
	}
}

// The trophy wants gold in EVERY party of the list (counted from the list, so a new party joins by itself), not the island.
func TestMedalTrophyAllGold(t *testing.T) {
	g := medalGame(t)
	delete(g.progress.Trophies, medalTrophy)
	g.progress.Medals = map[string]int{"jurassic": medalGold}
	ls := medalLevels()
	for _, k := range ls[:len(ls)-1] {
		g.progress.Medals[k] = medalGold
	}
	g.progress.Medals[ls[len(ls)-1]] = medalSilver
	g.medalSync()
	g.checkTrophies()
	if g.progress.Trophies[medalTrophy] {
		t.Fatal("trophy with a silver left")
	}
	var tr Trophy
	for _, x := range trophyList {
		if x.id == medalTrophy {
			tr = x
		}
	}
	if tr.need != len(ls) || g.progress.Stats[medalStat] != len(ls)-1 || secretTrophy[medalTrophy] {
		t.Fatalf("x/N: %d/%d (want %d/%d), secret %v", g.progress.Stats[medalStat], tr.need, len(ls)-1, len(ls), secretTrophy[medalTrophy])
	}
	g.progress.Medals[ls[len(ls)-1]] = medalGold
	g.medalSync()
	g.checkTrophies()
	if !g.progress.Trophies[medalTrophy] {
		t.Fatal("no trophy with gold everywhere")
	}
	// a party inserted later (the pirate ship) is needed too
	old := campaignLevels
	t.Cleanup(func() { campaignLevels = old; g.medalSync() })
	campaignLevels = append(append([]campaignEntry(nil), old[:len(old)-1]...), campaignEntry{kind: "x", icon: "test_new"}, old[len(old)-1])
	g.medalSync()
	if g.medalAllGold() || g.progress.Stats[medalStat] != len(ls) {
		t.Error("a new party without gold still counts as all gold")
	}
	// it is one more trophy before the legend; the legend stays the last one
	if trophyList[len(trophyList)-1].id != legendTrophy {
		t.Error("the legend is not the last trophy")
	}
}

// v1.0.0 saves: no medals field, load the same, nothing lost; round trip; a game without a save does not break.
func TestMedalSaveCompatible(t *testing.T) {
	old := `{"unlocked":12,"stats":{"won_0":2,"kills":340},"trophies":{"t_first":true},"wallet":900,"owned":["hat:x"],"equip":{"hat":"hat:x"},"campaignRev":2,"started":1700000000}`
	p, _, err := decodeProgress([]byte(old), true)
	if err != nil {
		t.Fatal(err)
	}
	if p.Medals != nil || p.MenuTier != 0 || p.MedalRev != 0 || p.Unlocked != 12 || p.Wallet != 900 || p.Stats["kills"] != 340 ||
		!p.Trophies["t_first"] || p.CampaignRev != 2 {
		t.Fatalf("old save read as %+v", p)
	}
	js, _ := json.Marshal(p)
	for _, k := range []string{"medals", "menuTier", "medalRev"} {
		if strings.Contains(string(js), `"`+k+`"`) {
			t.Errorf("a save without medals writes %q", k)
		}
	}
	// round trip through the real (signed, obfuscated) file
	dir := t.TempDir()
	path := filepath.Join(dir, "progress.json")
	p.Medals = map[string]int{"house": medalGold, "station": medalBronze}
	p.MenuTier, p.MedalRev = tierGold, 1
	if err := writeProgressFile(path, p); err != nil {
		t.Fatal(err)
	}
	q, found := readProgressFile(path)
	if !found || q.Medals["house"] != medalGold || q.Medals["station"] != medalBronze || q.MenuTier != tierGold || q.MedalRev != 1 ||
		q.Unlocked != 12 || q.Wallet != 900 {
		t.Fatalf("round trip: %+v", q)
	}
	c := cloneProgress(q)
	c.Medals["house"] = 1
	if q.Medals["house"] != medalGold {
		t.Error("cloneProgress shares the medals map")
	}
	// no save at all
	if _, found := readProgressFile(filepath.Join(dir, "none.json")); found {
		t.Error("found a save that does not exist")
	}
	g := medalGame(t)
	g.progress = Progress{Unlocked: 1}
	g.medalLegacyInit()
	if g.progress.MenuTier != tierNormal || g.progress.MedalRev != 1 || g.medalOf("house") != medalNone || g.goldCount() != 0 {
		t.Errorf("new game: %+v", g.progress)
	}
	g.loadLevel(0)
	g.points = 99999
	g.awardMedal() // nil map: created
	if g.medalOf(g.lvl.Def.Icon) != medalGold {
		t.Error("no medal on a new game")
	}
}

// The safeguard: whoever had every trophy (silver/gold menu, COMPLETADO 100 %) in v1.0.0 keeps it with the new trophy.
func TestMedalStickyMenuTier(t *testing.T) {
	g := medalGame(t)
	// a real v1.0.0 file: its 25 parties won (the BARCO PIRATA did not exist), migrated as newGame does
	p := v100Save(25, upTo(25), nil)
	migrateProgress(&p)
	if p.Stats[fmt.Sprintf("won_%d", pirateIdx)] != 0 || p.CampaignRev != campaignRev {
		t.Fatal("the pirate ship came won from v1.0.0")
	}
	for _, tr := range trophyList {
		if tr.id != medalTrophy {
			p.Trophies[tr.id] = true
		}
	}
	g.progress = p
	g.dbg.tier = 0
	if g.othersDone() {
		t.Fatal("the new trophy should be missing")
	}
	g.medalLegacyInit()
	if g.progress.MenuTier < tierSilver || g.menuTier() < tierSilver || !g.allTrophies() {
		t.Fatalf("v1.0.0 save with every trophy: tier %d (kept %d), all %v", g.menuTier(), g.progress.MenuTier, g.allTrophies())
	}
	// once the save has been through the medals, the computation does not run again
	g.progress.MenuTier = 0
	g.medalLegacyInit()
	if g.progress.MenuTier != 0 {
		t.Error("the v1.0.0 computation ran twice")
	}
	// a new save with the same trophies and every one of the 26 parties won (no legacy) is bronze until the gold trophy
	q := cloneProgress(p)
	q.MedalRev = 1
	q.Stats[fmt.Sprintf("won_%d", pirateIdx)] = 1
	g.progress = q
	g.medalLegacyInit()
	if g.menuTier() != tierBronze || g.allTrophies() {
		t.Errorf("new save without the gold trophy: tier %d", g.menuTier())
	}
	// the tier never goes down once reached
	g.progress.MenuTier = tierGold
	if g.menuTier() != tierGold {
		t.Error("the kept gold tier was lost")
	}
	// COMPLETADO: the 100 % stays, the new trophy is listed as "new"
	parts := []complPart{{"1 trofeo", "%d trofeos", 59, 60}}
	lines := g.complLines(parts)
	if lines[0] != "100 %" || !strings.HasPrefix(lines[1], tr("Nuevo:")) {
		t.Errorf("COMPLETADO with the kept gold: %q", lines)
	}
	g.progress.MenuTier = tierNormal
	if lines := g.complLines(parts); lines[0] == "100 %" || strings.HasPrefix(lines[1], tr("Nuevo:")) {
		t.Errorf("COMPLETADO without the kept gold: %q", lines)
	}
}

// The party list: medals and the points line of the highlighted party, in ES and EN, with its margins.
func TestMedalLevelSelectLayout(t *testing.T) {
	pg, pe, pu := gameLang, editorLang, uiLang
	t.Cleanup(func() { gameLang, editorLang, uiLang = pg, pe, pu })
	g := medalGame(t)
	g.progress.Unlocked = campaignN
	g.progress.Medals = map[string]int{}
	for i, k := range medalLevels() {
		g.progress.Medals[k] = 1 + i%3
	}
	for _, lang := range []string{"es", "en"} {
		gameLang, editorLang, uiLang = lang, lang, lang
		for sel := 0; sel < campaignN; sel++ {
			auditReset(g)
			g.mode, g.menuSel = ModeLevelSelect, sel
			textRec = &textRecorder{}
			img := ebiten.NewImage(screenW, screenH)
			g.Draw(img)
			img.Deallocate()
			rec := textRec
			textRec = nil
			want := tr(medalLine(g.medalGoalFor(levelDefs[sel])))
			var line *textRect
			var texts []textRect
			for i, r := range rec.texts {
				if r.x >= 230 && r.s != "3:47" {
					texts = append(texts, r)
				}
				if r.s == want {
					line = &rec.texts[i]
				}
			}
			if line == nil {
				t.Fatalf("[%s] party %d: no points line %q", lang, sel, want)
			}
			if x0, _, x1, _ := line.ink(); x0 < 246 || x1 > 914 || (x1-x0) > 2*(levelColX[1]-levelColX[0])-160 {
				t.Errorf("[%s] %q leaves the panel: %.0f-%.0f", lang, line.s, x0, x1)
			}
			// every medal: 16 px inside the panel, 12 px from any text of another row or column
			for i := 0; i < campaignN; i++ {
				cx := levelColX[i/levelRowsPerCol]
				y := levelRowY0 + float64(i%levelRowsPerCol)*levelRowDY
				pre := "  "
				if i == sel {
					pre = "> "
				}
				label := pre + fmt.Sprintf("%d ", i+1) + tr(levelDefs[i].Title)
				mx := levelMedalX(cx, g.textW(label, 1.3))
				m := [4]float64{mx, y - 1, mx + medalIconW, y - 1 + medalIconH}
				if m[2] > 930-16 {
					t.Errorf("[%s] medal of %d at x %.0f: too close to the panel edge", lang, i, m[2])
				}
				for _, r := range texts {
					x0, y0, x1, y1 := r.ink()
					sameRow := math.Abs(r.y-y) < 1 && math.Abs(r.x+r.w/2-cx) < 2
					if sameRow {
						if x1 > m[0]-6 {
							t.Errorf("[%s] medal of %d touches its name %q", lang, i, r.s)
						}
						continue
					}
					dx := math.Max(x0-m[2], m[0]-x1)
					dy := math.Max(y0-m[3], m[1]-y1)
					if dx < 12 && dy < 2 {
						t.Errorf("[%s] medal of %d is %.0f px from %q", lang, i, dx, r.s)
					}
				}
			}
		}
	}
}

// The hint plate at the start carries the medals line (exact for the party) and fits; not in the random mode.
func TestMedalHintLine(t *testing.T) {
	hintRestore(t)
	pu := uiLang
	t.Cleanup(func() { uiLang = pu })
	g := medalGame(t)
	for _, lang := range []string{"es", "en"} {
		uiLang = lang
		for _, i := range []int{0, campaignIdx("festival")} {
			g.loadLevel(i)
			if n := len(g.lh.lines); n != 3 || g.lh.lines[2] != g.medalHintLine() {
				t.Fatalf("party %d: hint %q", i, g.lh.lines)
			}
			if w := g.textW(g.lh.lines[2], 1); w+24 > 520 {
				t.Errorf("[%s] medals line %.0f px wide", lang, w)
			}
			// the plate stays two lines: the tip first, the medals in the second half
			if sh := g.levelHintShown(); len(sh) != 2 || sh[1] != g.lh.lines[1] {
				t.Errorf("first half shows %q", sh)
			}
			g.lh.t = lhDur*0.5 + 0.1
			if sh := g.levelHintShown(); len(sh) != 2 || sh[1] != g.medalHintLine() {
				t.Errorf("second half shows %q", sh)
			}
		}
	}
	g.loadLevel(campaignN)
	if len(g.lh.lines) != 2 {
		t.Errorf("random roll has a medals line: %q", g.lh.lines)
	}
}

// The newspaper: the medal and "NEW MEDAL" in the head of the points table, clear of the head text, ES and EN.
func TestMedalPaper(t *testing.T) {
	pu := uiLang
	t.Cleanup(func() { uiLang = pu })
	g := medalGame(t)
	for _, lang := range []string{"es", "en"} {
		uiLang = lang
		for _, c := range []struct {
			m   int
			new bool
		}{{medalBronze, true}, {medalSilver, false}, {medalGold, true}, {medalNone, false}} {
			g.run = runStats{done: true, medalOn: true, medal: c.m, medalNew: c.new, medalGoal: 95}
			d := paperSample(paperFacts{seed: 9}, false, false, 3)
			g.paperMedal(d)
			if c.m == medalNone {
				if !strings.Contains(d.scoreTag, fmt.Sprint(medalThresholds(95)[0])) || d.medal != 0 {
					t.Errorf("[%s] no medal: %q", lang, d.scoreTag)
				}
			} else if d.medal != c.m || strings.Contains(d.scoreTag, tr("¡NUEVA MEDALLA!")) != c.new || !strings.Contains(d.scoreTag, tr(medalName(c.m))) {
				t.Errorf("[%s] medal %d new %v: %q", lang, c.m, c.new, d.scoreTag)
			}
			_, lay := renderPaper(d, 0)
			var head, tag *ptext
			for i, tx := range lay.texts {
				if tx.s == d.scoreHead {
					head = &lay.texts[i]
				}
				if tx.s == d.scoreTag {
					tag = &lay.texts[i]
				}
			}
			if head == nil || tag == nil {
				t.Fatalf("[%s] head %v tag %v", lang, head, tag)
			}
			left := tag.x
			if c.m != medalNone {
				if len(lay.medals) != 1 {
					t.Fatalf("[%s] %d medals drawn", lang, len(lay.medals))
				}
				md := lay.medals[0]
				left = md.x
				if tag.x-(md.x+md.w) < 6 || md.y < head.y-4 || md.y+md.h > head.y+head.h+4 {
					t.Errorf("[%s] medal %v vs tag %+v", lang, md, *tag)
				}
			}
			if left-(head.x+head.w) < 12 {
				t.Errorf("[%s] %q is %d px from the head", lang, d.scoreTag, left-(head.x+head.w))
			}
		}
	}
	// the RETO keeps its own tag
	g.run = runStats{done: true}
	d := paperSample(paperFacts{seed: 9}, false, true, 3)
	tag := d.scoreTag
	g.paperMedal(d)
	if d.scoreTag != tag || d.medal != 0 {
		t.Error("the RETO page got a medal")
	}
}
