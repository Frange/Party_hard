package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"math"
	"testing"
)

// ITALIA sits right after the AUTOPISTA; the last two parties are still the space station and the secret island.
func TestItalyRightAfterHighway(t *testing.T) {
	builtinLevels()
	hw, it := campaignIdx("highway"), campaignIdx("italy")
	if hw < 0 || it != hw+1 || it != italyIdx {
		t.Fatalf("highway at %d, ITALIA at %d (italyIdx %d): ITALIA must come right after the highway", hw, it, italyIdx)
	}
	if campaignLevels[it+1].kind != "trainstation" {
		t.Errorf("after ITALIA comes %q, want the train station (the parties behind keep their order)", campaignLevels[it+1].kind)
	}
	if last := campaignLevels[len(campaignLevels)-1]; last.icon != "station" || campaignN != len(campaignLevels) {
		t.Fatalf("RULE: the space station is the last listed party (%q); every party builds (%d/%d)", last.icon, campaignN, len(campaignLevels))
	}
	if specialJurassic < campaignN {
		t.Error("the island must stay outside the campaign list (after the station)")
	}
	if levelDefs := builtinLevels(); levelDefs[it].Title != "ITALIA" || levelDefs[it].Icon != "italy" {
		t.Errorf("party %d is %q/%q", it+1, levelDefs[it].Title, levelDefs[it].Icon)
	}
}

// the layout builds with every landmark, is fully walkable and nothing stands on anything (several seeds: the random
// mode rolls it with another seed)
func TestItalyWalkableAndLandmarks(t *testing.T) {
	e := campaignLevels[italyIdx]
	for _, seed := range []int64{e.seed, e.seed + 1013, 1, 2, 3, 77} {
		d := genNamed("italy", seed, "")
		if d == nil {
			t.Fatalf("seed %d: no valid layout", seed)
		}
		if bad := genBad(d); len(bad) > 0 {
			t.Errorf("seed %d: unreachable %v", seed, bad)
		}
		l := buildLevel(d)
		if iss := levelOverlaps(l); len(iss) > 0 {
			t.Errorf("seed %d: %d overlaps, first %v", seed, len(iss), iss[0])
		}
		n := map[string]int{}
		for _, p := range d.Props {
			n[p.Kind]++
		}
		for _, p := range colosParts {
			if n[p.Kind] != 1 {
				t.Errorf("seed %d: amphitheatre piece %s x%d", seed, p.Kind, n[p.Kind])
			}
		}
		for k, min := range map[string]int{"pisa": 1, "colosfloor": 1, "gondola": 2, "venpalazzo": 4, "palina": 4, "gelato": 1,
			"pizzaoven": 1, "vespa": 2, "cypress": 6, "fountain": 1, "cafetable": 3, "dj": 1} {
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
		for _, p := range pts {
			q := l.NearestFree(p)
			if dist(p, q) > 24 || l.Path(d.Start, q) == nil {
				t.Errorf("seed %d: %v cannot be reached from the start", seed, p)
			}
		}
	}
}

// the canal is water (nobody walks on it), the wooden landing is walkable, and the arena of the amphitheatre is
// reached through the big arch AND through the collapsed gap (each one alone is enough)
func TestItalyCanalAndArches(t *testing.T) {
	d := genNamed("italy", campaignLevels[italyIdx].seed, "")
	l := buildLevel(d)
	for x := italyCanal.X + 8; x < italyCanal.X+italyCanal.W-8; x += 8 {
		for y := italyCanal.Y + 6; y < italyCanal.Y+italyCanal.H-4; y += 8 {
			if italyPier.Grow(6).Has(x, y) {
				continue
			}
			if l.Free(Vec{x, y}) {
				t.Fatalf("the canal is walkable at (%.0f,%.0f)", x, y)
			}
		}
	}
	pier := Vec{italyPier.X + italyPier.W/2, italyPier.Y + italyPier.H - 12}
	if !l.Free(pier) || l.Path(d.Start, pier) == nil {
		t.Error("the wooden landing cannot be reached")
	}
	arena := Vec{italyColos.X - 40, italyColos.Y + 10}
	if l.Path(d.Start, arena) == nil {
		t.Fatal("the arena cannot be reached")
	}
	// close one way in at a time: the other must still lead inside
	gate := Rect{italyColos.X - 30, italyColos.Y + colosRY - 20, 60, 30}
	gap := Rect{italyColos.X + 95, italyColos.Y + 15, 50, 40}
	for name, block := range map[string]Rect{"the arch": gate, "the gap": gap} {
		l2 := buildLevel(d)
		l2.solidsN = append(l2.solidsN, block)
		l2.initNav()
		if l2.Path(d.Start, arena) == nil {
			t.Errorf("with %s closed the arena cannot be reached: the other way in does not work", name)
		}
	}
	both := buildLevel(d)
	both.solidsN = append(both.solidsN, gate, gap)
	both.initNav()
	if both.Path(d.Start, arena) != nil {
		t.Error("the ring of the amphitheatre is not closed: there is a third way in through the wall")
	}
	// the tower blocks only its base
	for _, p := range l.Props {
		if p.Name == "pisa" && (p.Foot.W > 44 || p.Foot.H > 14) {
			t.Errorf("the tower blocks %v: only the base should be solid", p.Foot)
		}
	}
}

// ---------------------------------------------------------------------------------------------------------------
// saves of the 24-party layout (rev 1): ITALIA is inserted at index 19 without losing anything
// ---------------------------------------------------------------------------------------------------------------

func rev1Save(unlocked int, won []int) Progress {
	p := Progress{Unlocked: unlocked, CampaignRev: 1, Stats: map[string]int{"played": 60, "distinct": len(won)}}
	for _, i := range won {
		p.Stats[fmt.Sprintf("won_%d", i)] = 1
	}
	raw, _ := json.Marshal(p) // through JSON like the real file
	var q Progress
	json.Unmarshal(raw, &q)
	return q
}

func TestItalyMigrationKeepsEveryWin(t *testing.T) {
	var all []int
	for i := 0; i < 24; i++ {
		all = append(all, i)
	}
	p := rev1Save(24, all) // the whole 24-party campaign won
	migrateProgress(&p)
	if p.CampaignRev != campaignRev || p.Unlocked != 26 {
		t.Fatalf("rev %d unlocked %d, want %d and 26 (ITALIA and the pirate ship open too)", p.CampaignRev, p.Unlocked, campaignRev)
	}
	// the same parties (by name) are still won
	titles := []string{"HOUSE PARTY"} // only checked through the icons below
	_ = titles
	old := []string{"house", "beach", "neon", "ranch", "bull", "eiffel", "boat", "funfair", "stadium", "disco", "festival", "skate",
		"grave", "cinema", "circuit", "snow", "roof", "casino", "highway", "train", "cake", "hockey", "disco70", "station"}
	for oldIdx, icon := range old {
		ni := campaignIdx(icon)
		if ni < 0 {
			t.Fatalf("%s is gone from the campaign", icon)
		}
		if p.Stats[fmt.Sprintf("won_%d", ni)] != 1 {
			t.Errorf("%s (old won_%d) lost its win: won_%d = %d", icon, oldIdx, ni, p.Stats[fmt.Sprintf("won_%d", ni)])
		}
	}
	if _, ok := p.Stats[fmt.Sprintf("won_%d", italyIdx)]; ok {
		t.Error("ITALIA was never played: it cannot be won")
	}
	if p.Stats["distinct"] != 24 || p.Stats["played"] != 60 {
		t.Errorf("counters changed: distinct %d played %d", p.Stats["distinct"], p.Stats["played"])
	}
	g := testGame()
	g.progress = p
	if got := g.careerLevelsDone(); got != 24 {
		t.Errorf("parties done %d, want 24 (all but ITALIA)", got)
	}
	// once saved (rev 3) it is never moved again
	raw, _ := json.Marshal(p)
	var back Progress
	json.Unmarshal(raw, &back)
	migrateProgress(&back)
	if back.Unlocked != 26 || back.Stats["won_25"] != 1 || back.Stats["won_24"] != 0 || back.Stats["won_20"] != 1 {
		t.Errorf("a migrated save was migrated again: %+v", back)
	}
}

func TestItalyMigrationFrontier(t *testing.T) {
	cases := []struct{ un, want int }{
		{1, 1}, {10, 10},
		{19, 19},           // the highway is the frontier: beating it opens ITALIA
		{20, 21},           // the highway was beaten (train station open): ITALIA and the train station are open
		{22, 23}, {24, 26}, // the station was open: the pirate ship (rev 3) opens too
	}
	for _, c := range cases {
		var won []int
		for i := 0; i < c.un-1; i++ {
			won = append(won, i)
		}
		p := rev1Save(c.un, won)
		migrateProgress(&p)
		if p.Unlocked != c.want {
			t.Errorf("frontier %d became %d, want %d", c.un, p.Unlocked, c.want)
		}
		for _, i := range won { // every win keeps its party
			ni := campaignIdx(campaignLevelsRev1Icon(i))
			if p.Stats[fmt.Sprintf("won_%d", ni)] != 1 {
				t.Errorf("frontier %d: old won_%d did not land on won_%d", c.un, i, ni)
			}
		}
		// the frontier party itself is the same one as before
		if c.un-1 >= 0 && c.un < 25 {
			oldFront := campaignLevelsRev1Icon(c.un - 1)
			if newFront := campaignLevels[p.Unlocked-1].icon; newFront != oldFront {
				t.Errorf("frontier %d: the next party to beat was %s and now is %s", c.un, oldFront, newFront)
			}
		}
	}
}

// campaignLevelsRev1Icon: the icon of party i in the 24-party layout (the campaign without ITALIA and the pirate ship)
func campaignLevelsRev1Icon(i int) string {
	var icons []string
	for _, e := range campaignLevels {
		if e.icon != "italy" && e.icon != "pirate" {
			icons = append(icons, e.icon)
		}
	}
	return icons[i]
}

// editor corrections of the originals saved with the 24-party numbers still find their party
func TestItalyOldOverrideKeys(t *testing.T) {
	train := campaignIdx("train")
	if campaignKey(train) != "c21_train" || campaignKeyRev1(train) != "c20_train" {
		t.Errorf("train station keys %q / %q", campaignKey(train), campaignKeyRev1(train))
	}
	if campaignKeyRev1(italyIdx) != "" || campaignKeyRev1(italyIdx-1) != "" || campaignKeyRev1(0) != "" {
		t.Error("ITALIA and the parties before it did not move: they have no old key")
	}
	saved := levelOverrides
	t.Cleanup(func() { levelOverrides = saved })
	d := genNamed("trainstation", campaignLevels[train].seed, "ESTACIÓN DE TREN")
	la := levelAssetFromDef(d, "c20_train")
	la.Meta.Name = "tren viejo"
	levelOverrides = map[string]*LevelAsset{"c20_train": la}
	defs := builtinLevels()
	if defs[train].Title != "ESTACIÓN DE TREN" {
		t.Fatalf("party %d is %q", train+1, defs[train].Title)
	}
	levelOverrides = saved
	builtinLevels()
}

// texts in Spanish and English, and the row of the party list fits its column
func TestItalyTextsAndListRow(t *testing.T) {
	for _, s := range []string{"ITALIA", "Pasta, amore y caos", "Pizza, góndolas y una torre que ya se cae sola."} {
		if en := langDataEN[s]; en == "" || en == s {
			t.Errorf("no English for %q", s)
		}
	}
	th := hintThemes["italy"]
	for _, s := range []string{th.goal, th.tip} {
		if len([]rune(s)) > 48 || langDataEN[s] == "" || len([]rune(langDataEN[s])) > 48 {
			t.Errorf("hint %q too long or untranslated", s)
		}
	}
	if len(venueLines["italy"]) != 6 || len(pVenue["italy"]) < 5 {
		t.Errorf("venue lines %d, newspaper briefs %d", len(venueLines["italy"]), len(pVenue["italy"]))
	}
	for _, l := range venueLines["italy"] {
		if len([]rune(l.es)) > 48 || len([]rune(l.en)) > 48 || langDataEN[l.es] != l.en {
			t.Errorf("venue line %q", l.es)
		}
	}
	if genreForTitle("ITALIA").id != "clasica" || levelIconColors["italy"] == 0 || partyIndoorIcons["italy"] {
		t.Error("genre, menu colour or open air missing")
	}
	pg, pe, pu := gameLang, editorLang, uiLang
	t.Cleanup(func() { gameLang, editorLang, uiLang = pg, pe, pu })
	g := auditGame(t)
	for _, lang := range []string{"es", "en"} {
		gameLang, editorLang, uiLang = lang, lang, lang
		label := "> " + fmt.Sprintf("%d ", italyIdx+1) + tr("ITALIA")
		if want := map[string]string{"es": "ITALIA", "en": "ITALY"}[lang]; tr(label) != "> 20 "+want {
			t.Errorf("[%s] the list row reads %q", lang, tr(label))
		}
		w := g.textW(label, 1.3) + 18 // the icon on the left
		sw := g.textW(partySubtitle("ITALIA"), 1)
		half := (levelColX[1] - levelColX[0]) / 2
		if w/2 > half-12 || sw/2 > half-12 {
			t.Errorf("[%s] the row of ITALIA is too wide for its column: %.0f / %.0f (max %.0f)", lang, w, sw, 2*(half-12))
		}
	}
}

// ---------------------------------------------------------------------------------------------------------------
// brightness and colour cast: ITALIA is a DAYTIME party (soft neutral sunlight) and must not be blown out nor
// yellow (the user rejected a "too bright and yellowish" golden wash). The game cannot read its own screen in a
// test, so the frame is rebuilt on the CPU with the same pieces: the stylized floor tiles, the stylized prop sprites
// in depth order and the night / day wash of lighting (without the lamp halos, the same for every party). Mean
// luminance and mean R-B (the yellow cast) are compared.
// ---------------------------------------------------------------------------------------------------------------

func levelLook(d *LevelDef) (lum, warm float64) {
	const step = 3
	W, H := int(d.W), int(d.H)
	img := make([][3]float64, (W/step+1)*(H/step+1))
	at := func(x, y int) int { return (y/step)*(W/step+1) + x/step }
	floorCol := map[string][3]float64{}
	avg := func(b *pixBuf) [3]float64 {
		var s [3]float64
		n := 0.0
		for i := 0; i+3 < len(b.p); i += 4 {
			if b.p[i+3] > 0 {
				s[0], s[1], s[2] = s[0]+float64(b.p[i]), s[1]+float64(b.p[i+1]), s[2]+float64(b.p[i+2])
				n++
			}
		}
		if n > 0 {
			s[0], s[1], s[2] = s[0]/n, s[1]/n, s[2]/n
		}
		return s
	}
	colOf := func(kind string) [3]float64 {
		if c, ok := floorCol[kind]; ok {
			return c
		}
		var c [3]float64
		switch kind {
		case "water":
			w := rgb(0x2f7f80)
			c = [3]float64{float64(w.R), float64(w.G), float64(w.B)}
		case "dash", "dance", "rug", "helipad", "rink", "disco70", "edge", "rails":
			c = [3]float64{70, 70, 76}
		default:
			c = avg(stylizeFloor(kind, genFloor(kind, 0)))
		}
		floorCol[kind] = c
		return c
	}
	for y := 0; y < H; y += step {
		for x := 0; x < W; x += step {
			c := [3]float64{10, 9, 18}
			for _, f := range d.Floors {
				if f.R.Has(float64(x), float64(y)) {
					c = colOf(f.Kind)
				}
			}
			img[at(x, y)] = c
		}
	}
	props := append([]PropInst(nil), d.Props...)
	sortKey := func(p PropInst) float64 { return p.Y + propKinds[p.Kind].SortOff }
	for i := 1; i < len(props); i++ { // depth order (insertion sort: few props)
		for j := i; j > 0 && sortKey(props[j]) < sortKey(props[j-1]); j-- {
			props[j], props[j-1] = props[j-1], props[j]
		}
	}
	for _, p := range props {
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
	var a float64
	var base [3]float64
	day := d.Day
	if day {
		// drawLighting fills the screen with Tint*0.75 as a premultiplied colour: dst = wash + dst*(1-a)
		a = float64(t.A) * 0.75 / 255
		base = [3]float64{float64(t.R) * 0.75, float64(t.G) * 0.75, float64(t.B) * 0.75}
	} else {
		a = clampF(float64(t.A)*1.2, 70, 112) / 255
		bc := mixc(rgb(0x14142a), color.RGBA{t.R, t.G, t.B, 255}, 0.5)
		base = [3]float64{float64(bc.R), float64(bc.G), float64(bc.B)}
	}
	n := 0.0
	for y := 0; y < H; y += step {
		for x := 0; x < W; x += step {
			c := img[at(x, y)]
			for i := range c {
				if day {
					c[i] = math.Min(255, c[i]*(1-a)+base[i])
				} else {
					c[i] = c[i]*(1-a) + base[i]*a
				}
			}
			lum += 0.299*c[0] + 0.587*c[1] + 0.114*c[2]
			warm += c[0] - c[2]
			n++
		}
	}
	return lum / n, warm / n
}

func TestItalyDaylightNotBlownOutNorYellow(t *testing.T) {
	defs := builtinLevels()
	it := defs[italyIdx]
	if !it.Day {
		t.Fatal("ITALIA must be a daytime party (Day: no night layer)")
	}
	if it.Tint.R > it.Tint.B {
		t.Errorf("the daylight wash of ITALIA is warm (golden): %v", it.Tint)
	}
	for _, p := range it.Props { // by day no street lamp is lit (their halo would glow over the sunlit piazza)
		if k := propKinds[p.Kind]; k != nil && k.Light {
			t.Errorf("ITALIA is daytime but has a lit %q at %.0f,%.0f", p.Kind, p.X, p.Y)
		}
	}
	lum, warm := levelLook(it)
	maxNight, maxWarm := 0.0, -1000.0
	for _, icon := range []string{"eiffel", "highway", "festival", "bull"} {
		l, w := levelLook(defs[campaignIdx(icon)])
		t.Logf("%-8s luminance %.1f  R-B %.1f", icon, l, w)
		maxNight, maxWarm = math.Max(maxNight, l), math.Max(maxWarm, w)
	}
	jl, jw := levelLook(jurassicLevel())
	t.Logf("jurassic luminance %.1f  R-B %.1f (the other daytime level)", jl, jw)
	t.Logf("italy    luminance %.1f  R-B %.1f", lum, warm)
	// daylight: clearly lighter than the night parties, but a soft day, not a burnt-out frame
	const maxDayLum, maxDayWarm = 118.0, 12.0
	if lum < maxNight+10 {
		t.Errorf("ITALIA does not read as daytime: mean luminance %.1f, the night open-air parties reach %.1f", lum, maxNight)
	}
	if lum > maxDayLum {
		t.Errorf("ITALIA is too bright: mean luminance %.1f (max %.0f for a soft day)", lum, maxDayLum)
	}
	if warm > maxDayWarm {
		t.Errorf("ITALIA is too yellow: mean R-B %.1f (max %.0f; night parties reach %.1f)", warm, maxDayWarm, maxWarm)
	}
	// the check notices the golden sunset wash the user rejected (Day + warm tint)
	gold := *it
	gold.Tint = color.RGBA{150, 80, 40, 30}
	if l, w := levelLook(&gold); l <= maxDayLum && w <= maxDayWarm {
		t.Errorf("the check does not catch the old golden wash (luminance %.1f, R-B %.1f)", l, w)
	} else {
		t.Logf("golden wash luminance %.1f  R-B %.1f (rejected)", l, w)
	}
	// and the old blue night layer would no longer pass as daytime
	night := *it
	night.Day, night.Tint = false, color.RGBA{20, 28, 56, 88}
	if l, _ := levelLook(&night); l >= maxNight+10 {
		t.Errorf("the check takes the old night layer for daylight (luminance %.1f)", l)
	}
}
