package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------------------------------------------
// BARCO PIRATA (pirate.go): place in the campaign, layout, sea, boats, look, texts, guests, hiding, treasure, cap,
// two players, and the migration of the v1.0.0 saves (CampaignRev 2 -> 3).
// ---------------------------------------------------------------------------------------------------------------

// RULE: new parties go BEFORE the space station; the station and then the secret island are the last two.
func TestPirateRightBeforeStation(t *testing.T) {
	defs := builtinLevels()
	pi, st := campaignIdx("pirate"), campaignIdx("station")
	if pi != pirateIdx || pirateIdx != 24 || st != pi+1 {
		t.Fatalf("pirate ship at %d (pirateIdx %d), station at %d: the pirate ship is party 25, the station 26", pi, pirateIdx, st)
	}
	if len(campaignLevels) != 26 || campaignN != 26 {
		t.Fatalf("the campaign has %d parties (%d built), want 26", len(campaignLevels), campaignN)
	}
	if last := campaignLevels[len(campaignLevels)-1]; last.icon != "station" {
		t.Fatalf("RULE: the space station is the last listed party (got %q)", last.icon)
	}
	if campaignLevels[pi-1].icon != "disco70" {
		t.Errorf("before the pirate ship comes %q, want the 70s disco (the parties in front keep their order)", campaignLevels[pi-1].icon)
	}
	if specialJurassic < campaignN {
		t.Error("the island must stay outside the campaign list (after the station)")
	}
	if defs[pi].Title != "BARCO PIRATA" || defs[pi].Icon != "pirate" || defs[st].Title != "ESTACIÓN ESPACIAL" {
		t.Errorf("party 25 is %q/%q, party 26 %q", defs[pi].Title, defs[pi].Icon, defs[st].Title)
	}
	// the story still ends on the station: the rocket leaves after the party before it (the pirate ship)
	if interludePlaces(interRocket)[4] != pi {
		t.Errorf("the rocket's memories end on %d, want the pirate ship %d", interludePlaces(interRocket)[4], pi)
	}
	g := testGame()
	if k, ok := g.interludeAfter(pi); !ok || k != interRocket {
		t.Error("the rocket interlude does not come after the pirate ship (the party before the station)")
	}
}

// the layout builds with every piece, is fully walkable and nothing stands on anything (several seeds: the random
// mode rolls it with another seed)
func TestPirateWalkableAndProps(t *testing.T) {
	e := campaignLevels[pirateIdx]
	for _, seed := range []int64{e.seed, e.seed + 1013, 1, 2, 3, 77} {
		d := genNamed("pirate", seed, "")
		if d == nil {
			t.Fatalf("seed %d: no valid layout", seed)
		}
		if bad := genBad(d); len(bad) > 0 {
			t.Errorf("seed %d: unreachable %v", seed, bad)
		}
		l := buildLevel(d)
		if iss := levelOverlaps(l); len(iss) > 0 {
			t.Errorf("seed %d: %d overlaps, first %v", seed, len(iss), iss)
		}
		n := map[string]int{}
		for _, p := range d.Props {
			n[p.Kind]++
		}
		for k, min := range map[string]int{"pmast": 3, "pcannon": 3, "pcannonr": 4, "prum": 1, "pbarrel": 3, "pchest": 2, "ptreasure": 1,
			"phelm": 1, "pcaptable": 1, "plantern": 5, "panchor": 1, "pparrot": 1, "dj": 1, "pcoil": 2} {
			if n[k] < min {
				t.Errorf("seed %d: %s x%d, want at least %d", seed, k, n[k], min)
			}
		}
		// flood fill from the start: every roam area, dance floor, point of interest and performer
		var pts []Vec
		for _, r := range d.Roam {
			pts = append(pts, r.Center())
		}
		for _, r := range d.Dance {
			pts = append(pts, r.Center())
		}
		for _, ps := range l.POIs {
			pts = append(pts, ps...)
		}
		for _, pf := range d.Performers {
			pts = append(pts, pf.Pos)
		}
		seen, w, _ := playerReach(l, []Vec{d.Start})
		for _, p := range pts {
			q := l.NearestFree(p)
			if dist(p, q) > 24 || l.Path(d.Start, q) == nil {
				t.Errorf("seed %d: %v cannot be reached from the start", seed, p)
			}
			if c := int(q.Y/2)*w + int(q.X/2); c < 0 || c >= len(seen) || !seen[c] {
				t.Errorf("seed %d: the player cannot walk to %v", seed, q)
			}
		}
		// every prop of the party stands on the ship or the dock, never on the water
		for _, p := range l.Props {
			if !d.Shape(p.X, p.Y-2, 0) {
				t.Errorf("seed %d: %s at %.0f,%.0f stands on the water", seed, p.Name, p.X, p.Y)
			}
		}
	}
}

// the sea all around is solid; the gangplank and the dock are walkable; each castle is reached by either of its two
// stairs and its rail is closed everywhere else
func TestPirateSeaSolidAndStairs(t *testing.T) {
	d := genNamed("pirate", campaignLevels[pirateIdx].seed, "")
	l := buildLevel(d)
	for y := pirSea + 4; y < d.H-4; y += 6 {
		for x := 4.0; x < d.W-4; x += 6 {
			if d.Shape(x, y, 6) {
				continue // the deck, the dock and their edges
			}
			if l.Free(Vec{x, y}) {
				t.Fatalf("the sea is walkable at (%.0f,%.0f)", x, y)
			}
		}
	}
	// the real movement of the player cannot carry him off the deck, the gangplank or the dock
	for _, from := range []Vec{{300, 170}, {300, 390}, {120, 276}, {860, 276}, {480, 440}, {340, 496}, {640, 496}} {
		p := l.NearestFree(from)
		for _, dir := range [][2]float64{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {-1, -1}} {
			q := walkPlayer(l, p, dir[0], dir[1], 240)
			if !d.Shape(q.X, q.Y, 3) {
				t.Errorf("walking %v from %v ends in the sea at %v", dir, p, q)
			}
		}
	}
	dock := Vec{pirDock.X + 60, pirDock.Y + pirDock.H/2}
	plank := Vec{pirPlank.X + pirPlank.W/2, pirPlank.Y + pirPlank.H/2}
	for name, p := range map[string]Vec{"the dock": dock, "the gangplank": plank} {
		if !l.Free(p) || l.Path(d.Start, p) == nil {
			t.Errorf("%s cannot be reached", name)
		}
	}
	quarter, fore := Vec{170, 276}, Vec{760, 300}
	for _, c := range []struct {
		name   string
		in     Vec
		x      float64
		stairs [2][2]float64
	}{{"quarterdeck", quarter, pirQD, pirQDStairs}, {"forecastle", fore, pirFC, pirFCStairs}} {
		if l.Path(d.Start, l.NearestFree(c.in)) == nil {
			t.Fatalf("the %s cannot be reached", c.name)
		}
		block := func(s [2]float64) Rect { return Rect{c.x - 10, s[0] - 2, pirRail + 20, s[1] - s[0] + 4} }
		for i, s := range c.stairs { // close one stair: the other one still leads up
			l2 := buildLevel(d)
			l2.solidsN = append(l2.solidsN, block(s))
			l2.initNav()
			if l2.Path(d.Start, l2.NearestFree(c.in)) == nil {
				t.Errorf("with stair %d of the %s closed it cannot be reached: the other stair does not work", i, c.name)
			}
		}
		both := buildLevel(d)
		both.solidsN = append(both.solidsN, block(c.stairs[0]), block(c.stairs[1]))
		both.initNav()
		if both.Path(d.Start, both.NearestFree(c.in)) != nil {
			t.Errorf("the rail of the %s is not closed: there is a third way up", c.name)
		}
	}
}

// boats: every arrival docks at the little dock, and whoever gets off (police, paramedics) lands on the dock,
// reachable from the start, and is still on board a few seconds later (nobody vanishes or walks on the water)
func TestPirateBoatArrivals(t *testing.T) {
	d := genNamed("pirate", campaignLevels[pirateIdx].seed, "")
	l := buildLevel(d)
	if d.Transit != "sea" || d.Police != "car" || laneVertical(d) {
		t.Fatalf("transit %q police %q: the boats must sail along the dock", d.Transit, d.Police)
	}
	ly := d.CarFrom.Y - laneOffOf(d)
	if d.Shape(d.CarTo.X, ly, 0) || d.Shape(d.ParkX0, ly, 0) {
		t.Fatal("the boat lane crosses the dock")
	}
	for n := 0; n < 5; n++ { // the parking slots of up to five boats at once (dispatch / parkSlot)
		tx := math.Max(d.ParkX0, d.ParkX1-float64(n)*100)
		out := l.NearestFree(Vec{tx - 12, ly - 24})
		if !pirDock.Grow(2).Has(out.X, out.Y) {
			t.Errorf("slot %d: the crew gets off at %v, not on the dock", n, out)
		}
		if dist(out, Vec{tx - 12, ly - 24}) > 30 || l.Path(out, d.Start) == nil {
			t.Errorf("slot %d: the crew lands at %v, far or cut off from the deck", n, out)
		}
	}
	g := playGame(t, pirateIdx, 11)
	if !pirateBay(g.lvl.Def) {
		t.Fatalf("level %d is %q", pirateIdx, g.lvl.Def.Title)
	}
	before := len(g.cops)
	// a reported body on the main deck: the paramedics have somebody to go to (with nothing to do they sail away)
	g.bodies = append(g.bodies, &Body{pos: g.lvl.NearestFree(Vec{420, 330}), reported: true, cs: g.guests[0].cs})
	g.dispatch(false) // a police launch
	g.dispatch(true)  // and an ambulance launch
	for i := 0; i < 60*25 && len(g.cops) < before+2; i++ {
		g.updateVehicles(1.0 / 60)
	}
	if len(g.cops) < before+2 {
		t.Fatalf("after 25 s only %d people got off the boats", len(g.cops)-before)
	}
	medics := 0
	var crew []*Cop
	for _, c := range g.cops[before:] {
		crew = append(crew, c)
		if c.medic {
			medics++
		}
		// 2026-11-11: the ambulance is a rescue launch alongside the hull; its paramedics come up a rope
		// ladder and land on the deck at the top of it (ambulance.go), the police still tie up at the dock
		if c.medic {
			if c.veh == nil || !c.veh.ladder || !pirateDeck(c.pos.X, c.pos.Y, 0) || g.lvl.Path(c.pos, g.lvl.Def.Start) == nil {
				t.Errorf("a paramedic got off at %v (not on deck at the rope ladder, or cut off)", c.pos)
			}
			continue
		}
		if !pirDock.Grow(4).Has(c.pos.X, c.pos.Y) || g.lvl.Path(c.pos, g.lvl.Def.Start) == nil {
			t.Errorf("a %s got off at %v (not on the dock or cut off)", map[bool]string{true: "paramedic", false: "police officer"}[c.medic], c.pos)
		}
	}
	if medics == 0 || medics == len(crew) {
		t.Errorf("crew %d with %d paramedics: the police and the ambulance both landed", len(crew), medics)
	}
	for i := 0; i < 60*4; i++ {
		g.updatePlay(1.0 / 60)
	}
	for _, c := range crew {
		found := false
		for _, o := range g.cops {
			found = found || o == c
		}
		if !found {
			t.Errorf("a crew member vanished within 4 s of landing")
			continue
		}
		if !g.lvl.Def.Shape(c.pos.X, c.pos.Y, 3) {
			t.Errorf("a crew member is on the water at %v", c.pos)
		}
	}
}

func laneOffOf(d *LevelDef) float64 {
	if d.LaneOff > 0 {
		return d.LaneOff
	}
	return laneOffset
}

// hiding in a barrel or a chest: the player comes out exactly where he got in (hideexit.go)
func TestPirateHideInBarrelsAndChests(t *testing.T) {
	g := playGame(t, pirateIdx, 5)
	n := 0
	for _, pr := range g.lvl.Props {
		if pr.Name != "pbarrel" && pr.Name != "pchest" {
			continue
		}
		if !pr.K.Hide {
			t.Fatalf("%s is not a hiding place", pr.Name)
		}
		in := g.lvl.NearestFree(Vec{pr.X, pr.Foot.Y + pr.Foot.H + 6})
		p := g.p
		p.pos, p.hidden = in, false
		if lab, _ := g.useLabel(pr); lab != "Esconderse" {
			t.Errorf("%s at %.0f,%.0f: the prompt reads %q", pr.Name, pr.X, pr.Y, lab)
		}
		g.hideEnter(p, pr)
		g.hideLeave(p)
		if dist(p.pos, in) > 0.01 {
			t.Errorf("%s at %.0f,%.0f: in at %v, out at %v (must be the same spot)", pr.Name, pr.X, pr.Y, in, p.pos)
		}
		n++
	}
	if n < 5 {
		t.Errorf("only %d barrels and chests to hide in", n)
	}
}

// the treasure of the bow: one press gives the points and it is gone; it never takes the poison / disguise slot
func TestPirateTreasure(t *testing.T) {
	g := playGame(t, pirateIdx, 5)
	var tr *Prop
	for _, pr := range g.lvl.Props {
		if pr.Name == "ptreasure" {
			tr = pr
		}
	}
	if tr == nil {
		t.Fatal("no treasure on board")
	}
	p := g.p
	p.pos = g.lvl.NearestFree(Vec{tr.X, tr.Y + 8})
	p.item, p.spare = "poison", ""
	if l, _ := g.useLabel(tr); l != "Coger el tesoro" || langDataEN[l] == "" {
		t.Fatalf("the treasure prompt reads %q", l)
	}
	before := g.points
	if !g.pickupInteract(p) {
		t.Fatal("the treasure cannot be picked up")
	}
	if g.points != before+pirateTreasurePts || !tr.gone || p.item != "poison" || p.spare != "" {
		t.Errorf("points %d -> %d, gone %v, slots %q/%q", before, g.points, tr.gone, p.item, p.spare)
	}
	if g.pickupInteract(p) {
		t.Error("the treasure was picked up twice")
	}
}

// ---------------------------------------------------------------------------------------------------------------
// look: a sunset, warm accents but not golden nor blown out (the user rejected "muy brillante y amarillento" in
// ITALIA). The frame is rebuilt on the CPU like levelLook (italy_test.go) but with the painted bay of the party.
// ---------------------------------------------------------------------------------------------------------------

func pirateLook(d *LevelDef) (lum, warm, yellow float64) {
	const step = 3
	W, H := int(d.W), int(d.H)
	img := make([][3]float64, (W/step+1)*(H/step+1))
	at := func(x, y int) int { return (y/step)*(W/step+1) + x/step }
	sc := pirateScene()
	for y := 0; y < H; y += step {
		for x := 0; x < W; x += step {
			c := [3]float64{80, 40, 90} // the sunset sky (draw.go Sky "sea": purple to pink)
			if float64(y) >= pirSea {
				p := sc.get(x, y)
				c = [3]float64{float64(p.R), float64(p.G), float64(p.B)}
			}
			img[at(x, y)] = c
		}
	}
	for _, p := range d.Props {
		k := propKinds[p.Kind]
		if k == nil || k.Draw == nil {
			continue
		}
		b := stylizeSprite(renderPropSprite(p.Kind, k, p.Var))
		x0, y0 := int(p.X)-k.W/2-1, int(p.Y)-k.H-1
		for py := 0; py < b.h; py++ {
			for px := 0; px < b.w; px++ {
				i := (py*b.w + px) * 4
				x, y := x0+px, y0+py
				if b.p[i+3] < 128 || x < 0 || y < 0 || x >= W || y >= H || x%step != 0 || y%step != 0 {
					continue
				}
				img[at(x, y)] = [3]float64{float64(b.p[i]), float64(b.p[i+1]), float64(b.p[i+2])}
			}
		}
	}
	t := d.Tint
	a := float64(t.A) * 0.75 / 255
	base := [3]float64{float64(t.R) * 0.75, float64(t.G) * 0.75, float64(t.B) * 0.75}
	n := 0.0
	for y := 0; y < H; y += step {
		for x := 0; x < W; x += step {
			c := img[at(x, y)]
			for i := range c {
				c[i] = math.Min(255, c[i]*(1-a)+base[i])
			}
			lum += 0.299*c[0] + 0.587*c[1] + 0.114*c[2]
			warm += c[0] - c[2]
			yellow += (c[0]+c[1])/2 - c[2]
			n++
		}
	}
	return lum / n, warm / n, yellow / n
}

func TestPirateSunsetNotYellowNorBlownOut(t *testing.T) {
	defs := builtinLevels()
	pd := defs[pirateIdx]
	if !pd.Day {
		t.Fatal("the pirate party is at sunset: no night layer (Day), just a wash")
	}
	if pd.Tint.R > pd.Tint.B || pd.Tint.G > pd.Tint.B {
		t.Errorf("the sunset wash is warm (golden): %v; the warmth must come from the sky and the lanterns", pd.Tint)
	}
	lum, warm, yellow := pirateLook(pd)
	maxNight, maxWarm := 0.0, -1000.0
	for _, icon := range []string{"boat", "eiffel", "highway", "festival", "bull"} {
		l, w := levelLook(defs[campaignIdx(icon)])
		t.Logf("%-8s luminance %.1f  R-B %.1f", icon, l, w)
		maxNight, maxWarm = math.Max(maxNight, l), math.Max(maxWarm, w)
	}
	il, iw := levelLook(defs[campaignIdx("italy")])
	t.Logf("italy    luminance %.1f  R-B %.1f (daylight)", il, iw)
	t.Logf("pirate   luminance %.1f  R-B %.1f  (R+G)/2-B %.1f", lum, warm, yellow)
	const maxLum, maxWarmPirate, maxYellow = 110.0, 12.0, 6.0
	if lum < maxNight+4 {
		t.Errorf("the sunset reads as night: luminance %.1f, the night parties reach %.1f", lum, maxNight)
	}
	if lum > maxLum || lum > il {
		t.Errorf("the sunset is too bright: luminance %.1f (max %.0f, and never brighter than the daylight of ITALIA %.1f)", lum, maxLum, il)
	}
	if warm > maxWarmPirate || yellow > maxYellow {
		t.Errorf("the sunset is too yellow: R-B %.1f (max %.0f), (R+G)/2-B %.1f (max %.0f)", warm, maxWarmPirate, yellow, maxYellow)
	}
	// the check catches the golden wash the user rejected
	gold := *pd
	gold.Tint = color.RGBA{150, 80, 40, 30}
	if l, w, y := pirateLook(&gold); l <= maxLum && w <= maxWarmPirate && y <= maxYellow {
		t.Errorf("the check does not catch a golden wash (luminance %.1f, R-B %.1f, yellow %.1f)", l, w, y)
	}
}

// ---------------------------------------------------------------------------------------------------------------
// texts, list row, list icon, music, events and the guests dressed as pirates
// ---------------------------------------------------------------------------------------------------------------

func TestPirateTextsAndListRow(t *testing.T) {
	d := genNamed("pirate", campaignLevels[pirateIdx].seed, "")
	for _, s := range []string{"BARCO PIRATA", d.Intro, partyTypeOfTitle["BARCO PIRATA"], "Coger el tesoro", "¡Un tesoro! Doblones para el bolsillo"} {
		if en := langDataEN[s]; en == "" || en == s {
			t.Errorf("no English for %q", s)
		}
	}
	if len([]rune(d.Intro)) > 48 || len([]rune(langDataEN[d.Intro])) > 48 {
		t.Errorf("the intro line is too long for a bubble: %q", d.Intro)
	}
	th := hintThemes["pirate"]
	for _, s := range []string{th.goal, th.tip} {
		if len([]rune(s)) > 48 || langDataEN[s] == "" || len([]rune(langDataEN[s])) > 48 {
			t.Errorf("hint %q too long or untranslated", s)
		}
	}
	if len(venueLines["pirate"]) != 6 || len(pVenue["pirate"]) < 5 {
		t.Errorf("venue lines %d, newspaper briefs %d", len(venueLines["pirate"]), len(pVenue["pirate"]))
	}
	for _, l := range venueLines["pirate"] {
		if len([]rune(l.es)) > 48 || len([]rune(l.en)) > 48 || langDataEN[l.es] != l.en {
			t.Errorf("venue line %q", l.es)
		}
	}
	for _, p := range pVenue["pirate"] {
		if langDataEN[p.es] != p.en {
			t.Errorf("newspaper brief %q untranslated", p.es)
		}
	}
	places := 0
	for _, p := range ptPlaces {
		if p.theme == "pirate" {
			places++
		}
	}
	if places < 2 {
		t.Errorf("%d newspaper places of the pirate ship", places)
	}
	// music: one of the existing styles, not the one of its neighbours in the list
	gid := genreForTitle("BARCO PIRATA").id
	if gid != "epico" || gid == genreForTitle("DISCO SETENTERA").id || gid == genreForTitle("ESTACIÓN ESPACIAL").id {
		t.Errorf("genre %q", gid)
	}
	if levelIconColors["pirate"] == 0 || partyIndoorIcons["pirate"] || levelIconArt["pirate"] == nil {
		t.Error("menu colour, open air or list picture missing")
	}
	if !congaIcons["pirate"] || !dogNoIcons["pirate"] || buskerIcons["pirate"] || karaokeIcons["pirate"] || tiroIcons["pirate"] {
		t.Error("party events: conga on deck yes; no dog act, busker, karaoke or shoot-out on a ship")
	}
	if cameraPlanHas("BARCO PIRATA") {
		t.Error("a pirate ship has no security cameras")
	}
	if bigCrowd["pirate"] != 0 {
		t.Error("the pirate ship is not one of the three big-crowd parties")
	}
	pg, pe, pu := gameLang, editorLang, uiLang
	t.Cleanup(func() { gameLang, editorLang, uiLang = pg, pe, pu })
	g := auditGame(t)
	for _, lang := range []string{"es", "en"} {
		gameLang, editorLang, uiLang = lang, lang, lang
		label := "> " + fmt.Sprintf("%d ", pirateIdx+1) + tr("BARCO PIRATA")
		if want := map[string]string{"es": "BARCO PIRATA", "en": "PIRATE SHIP"}[lang]; tr(label) != "> 25 "+want {
			t.Errorf("[%s] the list row reads %q", lang, tr(label))
		}
		w := g.textW(label, 1.3) + 18 // the icon on the left
		sw := g.textW(partySubtitle("BARCO PIRATA"), 1)
		half := (levelColX[1] - levelColX[0]) / 2
		if w/2 > half-12 || sw/2 > half-12 {
			t.Errorf("[%s] the row of the pirate ship is too wide for its column: %.0f / %.0f (max %.0f)", lang, w, sw, 2*(half-12))
		}
	}
}

// cameraPlanHas: does the plan of security cameras name this party?
func cameraPlanHas(title string) bool {
	_, ok := cameraPlanNormal[title]
	return ok
}

// the list picture is a picture (several colours, the black flag, the hull), dimmed when locked
func TestPirateListIconIsAPicture(t *testing.T) {
	cols := map[uint32]bool{}
	for _, row := range pirateIconRows {
		if len(row) != 12 {
			t.Fatalf("row %q is not 12 px", row)
		}
		for i := 0; i < len(row); i++ {
			c, ok := pirateIconCols[row[i]]
			if !ok {
				t.Fatalf("unknown colour %q", row[i])
			}
			cols[c] = true
		}
	}
	if len(cols) < 7 {
		t.Errorf("only %d colours: a flat swatch, not a picture", len(cols))
	}
}

// the flags of the masts and the parrot move: two frames that swap with the clock, the style stays
func TestPirateFlagsWave(t *testing.T) {
	for _, name := range []string{"pmast", "pparrot"} {
		k := propKinds[name]
		seen := map[int]bool{}
		for i := 0; i < 20; i++ {
			v := pirateAnimVar(name, 0, float64(i)*0.13, 480)
			if v/2 != 0 || v >= k.Vars {
				t.Fatalf("%s: variant %d out of its style", name, v)
			}
			seen[v] = true
		}
		if len(seen) != 2 {
			t.Errorf("%s does not move (%v)", name, seen)
		}
		a := renderPropSprite(name, k, 0)
		b := renderPropSprite(name, k, 1)
		same := true
		for i := range a.p {
			same = same && a.p[i] == b.p[i]
		}
		if same {
			t.Errorf("%s: the two frames are the same picture", name)
		}
	}
	if pirateAnimVar("speaker", 3, 1.7, 100) != 3 {
		t.Error("other props must keep their variant")
	}
}

// many guests come dressed as pirates (tricorns, striped shirts, bandanas, long coats), with their own dice
func TestPirateGuestsDressed(t *testing.T) {
	r := rand.New(rand.NewSource(9))
	tricorn, stripes, beards := 0, 0, 0
	for i := 0; i < 400; i++ {
		s := crowdSpec("pirate", r)
		if s.Name != "" || s.DisplayName != "" {
			t.Fatal("a dressed guest got a name")
		}
		if s.Hat == HatPirate {
			tricorn++
		}
		if s.Pattern == 2 {
			stripes++
		}
		if s.Beard != BeardNone {
			beards++
		}
	}
	if tricorn < 120 || stripes < 120 || beards < 40 {
		t.Errorf("tricorns %d, stripes %d, beards %d of 400", tricorn, stripes, beards)
	}
	if crowdKind(&LevelDef{Icon: "pirate"}) != "pirate" || crowdChance["pirate"] < 0.3 {
		t.Error("the pirate ship does not dress its guests")
	}
}

// the cap of the kills is the normal 65 (not a big-crowd party), arrivals never pass it and killing everybody of
// the target ends the party (N/N)
func TestPirateKillTargetCapped(t *testing.T) {
	if len(levelDefs) < len(campaignLevels) {
		// the full campaign is what newGame builds too (arrival_test.go does the same): it is NOT put back. Putting
		// back the 5 classic levels undid the shared audit game built meanwhile (playGame) and left campaignN = 26
		// with 5 levels: the medal tests saw "no campaign" and TestPirateBoatArrivals indexed out of range (-shuffle)
		levelDefs = builtinLevels()
	}
	g := playGame(t, pirateIdx, 21)
	if g.goalMax() != 65 {
		t.Fatalf("cap %d, want 65", g.goalMax())
	}
	if mx := killTargetStress(t, g, 60); mx > 65 {
		t.Errorf("the target reached %d", mx)
	}
	total := g.total
	killAll(g)
	if g.aliveCount() != 0 || g.total != total {
		t.Errorf("%d alive, total %d -> %d", g.aliveCount(), total, g.total)
	}
	if g.mode == ModePlay && !g.slow.active {
		t.Errorf("everybody of the target is out but the party did not end")
	}
}

// two players: both start on the deck, can reach each other and everything, and the cap grows by the 2P extra only
func TestPirateTwoPlayers(t *testing.T) {
	d := genNamed("pirate", campaignLevels[pirateIdx].seed, "")
	l := buildLevel(d)
	starts := playerStarts(l, d)
	for _, set := range starts {
		seen, w, _ := playerReach(l, set)
		for _, p := range set {
			if !d.Shape(p.X, p.Y, 0) || !l.Free(p) {
				t.Errorf("a player starts at %v, not on the deck", p)
			}
		}
		for _, r := range d.Roam {
			q := l.NearestFree(r.Center())
			if !seen[int(q.Y/2)*w+int(q.X/2)] {
				t.Errorf("with %d player(s) %v cannot be reached", len(set), q)
			}
		}
	}
	old, oldDbg := twoPlayerMode, dbgTwoPlayers
	twoPlayerMode, dbgTwoPlayers = true, true
	t.Cleanup(func() { twoPlayerMode, dbgTwoPlayers = old, oldDbg })
	g := capGame(d, 5)
	g.pickStartTarget()
	if g.twoExtra < 20 || g.twoExtra > 30 || g.goalMax() != 65+g.twoExtra || g.crowdExtra != 0 {
		t.Errorf("two players: extra %d, crowd %d, cap %d", g.twoExtra, g.crowdExtra, g.goalMax())
	}
}

// ---------------------------------------------------------------------------------------------------------------
// the saves of v1.0.0 (CampaignRev 2: 25 parties, the station was won_24 and frontier 25): the pirate ship is
// inserted at 24 without losing anything, once, and the real progress.json is never touched
// ---------------------------------------------------------------------------------------------------------------

// v100Save: a progress as v1.0.0 wrote it (rev 2), through JSON like the real file.
func v100Save(unlocked int, won []int, extra map[string]int) Progress {
	p := Progress{Unlocked: unlocked, CampaignRev: 2, Wallet: 4321, Stats: map[string]int{"played": 77, "distinct": len(won), "kills": 900},
		Trophies: map[string]bool{"first_blood": true}, Owned: []string{"hat:pirate"}, Album: map[string]int{"elvis": 3},
		Started: 1700000000}
	for _, i := range won {
		p.Stats[fmt.Sprintf("won_%d", i)] = 1
	}
	for k, v := range extra {
		p.Stats[k] = v
	}
	raw, _ := json.Marshal(p)
	var q Progress
	json.Unmarshal(raw, &q)
	return q
}

func upTo(n int) []int {
	var out []int
	for i := 0; i < n; i++ {
		out = append(out, i)
	}
	return out
}

// rev2Icon: the icon of party i in the 25-party layout of v1.0.0
func rev2Icon(i int) string {
	if i >= pirateIdx {
		return campaignLevels[i+1].icon
	}
	return campaignLevels[i].icon
}

func TestPirateMigrationStationWonAndIsland(t *testing.T) {
	p := v100Save(25, upTo(25), map[string]int{"jp_unlocked": 1, "seen_interlude20": 1, "seen_end1": 1})
	migrateProgress(&p)
	if p.CampaignRev != 3 || campaignRev != 3 || p.Unlocked != 26 {
		t.Fatalf("rev %d (const %d) unlocked %d, want 3 and 26: the station stays open and the pirate ship opens", p.CampaignRev, campaignRev, p.Unlocked)
	}
	for old := 0; old < 25; old++ { // every party keeps its own win, by name
		ni := campaignIdx(rev2Icon(old))
		if p.Stats[fmt.Sprintf("won_%d", ni)] != 1 {
			t.Errorf("%s (old won_%d) lost its win", rev2Icon(old), old)
		}
	}
	if _, ok := p.Stats[fmt.Sprintf("won_%d", pirateIdx)]; ok {
		t.Error("the pirate ship was never played: it cannot be won")
	}
	if p.Stats["won_25"] != 1 {
		t.Error("the station's win did not move to won_25")
	}
	for k, want := range map[string]int{"jp_unlocked": 1, "seen_interlude20": 1, "seen_end1": 1, "played": 77, "distinct": 25, "kills": 900} {
		if p.Stats[k] != want {
			t.Errorf("%s = %d, want %d", k, p.Stats[k], want)
		}
	}
	if p.Wallet != 4321 || !p.Trophies["first_blood"] || len(p.Owned) != 1 || p.Album["elvis"] != 3 || p.Started != 1700000000 {
		t.Errorf("something else changed: %+v", p)
	}
	g := testGame()
	g.progress = p
	// careerLevelsDone counts up to campaignN, which only builtinLevels fills in: run alone (or
	// first) the test saw 0 parties. The game always builds the campaign before any menu.
	prevN := campaignN
	campaignN = len(campaignLevels)
	t.Cleanup(func() { campaignN = prevN })
	if got := g.careerLevelsDone(); got != 25 {
		t.Errorf("parties done %d, want 25 (all but the pirate ship)", got)
	}
	if !g.jpListed() {
		t.Error("the island is no longer in the list")
	}
}

func TestPirateMigrationStationOpenNotWon(t *testing.T) {
	p := v100Save(25, upTo(24), nil)
	migrateProgress(&p)
	if p.Unlocked != 26 || p.Stats["won_25"] != 0 || p.Stats["won_24"] != 0 || p.Stats["won_23"] != 1 {
		t.Errorf("unlocked %d, won_23 %d won_24 %d won_25 %d: the disco keeps its win, nothing is invented",
			p.Unlocked, p.Stats["won_23"], p.Stats["won_24"], p.Stats["won_25"])
	}
}

func TestPirateMigrationFrontier(t *testing.T) {
	for _, c := range []struct{ un, want int }{{1, 1}, {7, 7}, {20, 20}, {24, 24}, {25, 26}} {
		p := v100Save(c.un, upTo(c.un-1), nil)
		before := cloneProgress(p)
		migrateProgress(&p)
		if p.Unlocked != c.want {
			t.Errorf("frontier %d became %d, want %d", c.un, p.Unlocked, c.want)
		}
		if c.un < 25 { // the station had not been reached: no stat moves at all
			for k, v := range before.Stats {
				if p.Stats[k] != v {
					t.Errorf("frontier %d: %s changed %d -> %d", c.un, k, v, p.Stats[k])
				}
			}
			// the next party to beat is the same one; at the disco (24), beating it now opens the pirate ship
			if got, want := campaignLevels[p.Unlocked-1].icon, rev2Icon(c.un-1); got != want {
				t.Errorf("frontier %d: the next party was %s and now is %s", c.un, want, got)
			}
		}
	}
	// a frontier at the disco: beating it opens the pirate ship, then the station, which stays the last
	builtinLevels()
	g := testGame()
	g.progress = v100Save(24, upTo(23), nil)
	migrateProgress(&g.progress)
	for _, idx := range []int{campaignIdx("disco70"), pirateIdx, campaignIdx("station")} {
		g.levelIdx = idx
		g.unlockNext()
	}
	if g.progress.Unlocked != 26 || campaignLevels[g.progress.Unlocked-1].icon != "station" {
		t.Errorf("after the disco and the pirate ship the frontier is %d", g.progress.Unlocked)
	}
}

func TestPirateMigrationOnceAndOlderSaves(t *testing.T) {
	p := v100Save(25, upTo(25), map[string]int{"jp_unlocked": 1})
	migrateProgress(&p)
	once := cloneProgress(p)
	migrateProgress(&p) // idempotent
	raw, _ := json.Marshal(p)
	var back Progress
	json.Unmarshal(raw, &back)
	migrateProgress(&back) // and after a save round trip (rev 3 is stamped)
	for _, q := range []Progress{p, back} {
		if q.Unlocked != once.Unlocked || q.CampaignRev != 3 || fmt.Sprint(q.Stats) != fmt.Sprint(once.Stats) {
			t.Errorf("migrated twice: %+v", q)
		}
	}
	// a save already in the new layout (rev 3) with the pirate ship won is never moved
	var fresh Progress
	json.Unmarshal([]byte(`{"unlocked":26,"campaignRev":3,"stats":{"won_24":1,"won_25":1}}`), &fresh)
	migrateProgress(&fresh)
	if fresh.Unlocked != 26 || fresh.Stats["won_24"] != 1 || fresh.Stats["won_25"] != 1 {
		t.Errorf("a rev-3 save was migrated: %+v", fresh)
	}
	// the old layouts go through the whole chain: the station's win ends at won_25
	old := oldSave(21, 20) // the 21-party game, station won
	migrateProgress(&old)
	if old.Unlocked != 26 || old.Stats["won_25"] != 1 || old.Stats["won_24"] != 0 || old.CampaignRev != 3 {
		t.Errorf("a 21-party save: unlocked %d won_24 %d won_25 %d rev %d", old.Unlocked, old.Stats["won_24"], old.Stats["won_25"], old.CampaignRev)
	}
	r1 := rev1Save(24, upTo(24)) // the 24-party game, station won
	migrateProgress(&r1)
	if r1.Unlocked != 26 || r1.Stats["won_25"] != 1 || r1.Stats["won_24"] != 0 || r1.Stats["won_19"] != 0 {
		t.Errorf("a 24-party save: %+v", r1.Stats)
	}
}

// the editor corrections saved with the 25-party (c25_station) or older numbers still reach the station
func TestPirateOldOverrideKeys(t *testing.T) {
	st := campaignIdx("station")
	if campaignKey(st) != "c26_station" || campaignKeyRev2(st) != "c25_station" || campaignKeyRev1(st) != "c24_station" || legacyStationKey != "c21_station" {
		t.Fatalf("station keys %q / %q / %q / %q", campaignKey(st), campaignKeyRev2(st), campaignKeyRev1(st), legacyStationKey)
	}
	if campaignKey(pirateIdx) != "c25_pirate" || campaignKeyRev2(pirateIdx) != "" || campaignKeyRev1(pirateIdx) != "" {
		t.Error("the pirate ship is new: it has no old key")
	}
	train := campaignIdx("train")
	if campaignKey(train) != "c21_train" || campaignKeyRev1(train) != "c20_train" || campaignKeyRev2(train) != "" {
		t.Error("the parties in front of the pirate ship keep their keys")
	}
	saved := levelOverrides
	t.Cleanup(func() { levelOverrides = saved; builtinLevels() })
	for _, key := range []string{"c25_station", "c24_station"} {
		d := genNamed("station", campaignLevels[st].seed, "ESTACIÓN ESPACIAL")
		la := levelAssetFromDef(d, key)
		la.Meta.Intro = "corrección " + key
		levelOverrides = map[string]*LevelAsset{key: la}
		defs := builtinLevels()
		if defs[st].Intro != la.Meta.Intro {
			t.Errorf("the correction saved as %s does not reach the station (intro %q)", key, defs[st].Intro)
		}
		if defs[pirateIdx].Intro == la.Meta.Intro {
			t.Errorf("the correction saved as %s went to the pirate ship", key)
		}
	}
}

// the migration runs on a copy in a temporary folder; the real assets/progress.json is never read for writing
func TestPirateMigrationNeverTouchesTheRealSave(t *testing.T) {
	snapshotGlobals(t)
	real := filepath.Join("assets", "progress.json")
	sum := func() string {
		b, err := os.ReadFile(real)
		if err != nil {
			return "none"
		}
		h := sha256.Sum256(b)
		return fmt.Sprintf("%x", h)
	}
	st, _ := os.Stat(real)
	before := sum()
	dir := t.TempDir()
	path := filepath.Join(dir, "progress.json")
	if err := writeProgressFile(path, v100Save(25, upTo(25), map[string]int{"jp_unlocked": 1})); err != nil {
		t.Fatal(err)
	}
	p, ok := readProgressFile(path)
	if !ok || p.CampaignRev != 2 {
		t.Fatalf("the v1.0.0 copy did not load (%v, rev %d)", ok, p.CampaignRev)
	}
	migrateProgress(&p)
	if err := writeProgressFile(path, p); err != nil {
		t.Fatal(err)
	}
	q, _ := readProgressFile(path)
	if q.CampaignRev != 3 || q.Unlocked != 26 || q.Stats["won_25"] != 1 || q.Stats["jp_unlocked"] != 1 {
		t.Errorf("the migrated copy reads back as %+v", q)
	}
	if after := sum(); after != before {
		t.Fatal("assets/progress.json changed during the test")
	}
	if st2, _ := os.Stat(real); st != nil && (st2 == nil || !st2.ModTime().Equal(st.ModTime())) {
		t.Fatal("assets/progress.json was rewritten during the test")
	}
}

// the set pieces that cannot happen on a ship anchored in a bay stay away (cars, herds, the giant rex and the kids on
// scooters would come in over the water); the ones that fit a deck still happen
func TestPirateEventsThatFit(t *testing.T) {
	g := playGame(t, pirateIdx, 3)
	for name, start := range map[string]func(*Event) bool{
		"F1":       g.startF1,
		"stampede": g.startStampede,
		"rex":      g.startRex,
		"scooters": func(e *Event) bool { e.kind = evScooters; return g.startKidRun(e) },
		"skaters":  func(e *Event) bool { e.kind = evSkate; return g.startKidRun(e) },
	} {
		for i := 0; i < 5; i++ {
			if start(&Event{}) {
				t.Errorf("%s started on the pirate ship", name)
				break
			}
		}
	}
	if g.tankDue {
		t.Error("a tank on a pirate ship")
	}
	ok := false
	for i := 0; i < 10 && !ok; i++ {
		ok = g.startParachute(&Event{})
	}
	if !ok {
		t.Error("a parachutist can always land on the deck")
	}
}
