package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// GUADALAJARA is the 14th row of the list (index guadalajaraIdx = 15, behind the retired CINE, before the CIRCUITO);
// the BARCO PIRATA and the space station are still the last two rows and the island stays out of the list.
func TestGuadalajaraFourteenthOfTheList(t *testing.T) {
	_ = layerGame(t, false)
	gi := campaignIdx("guadalajara")
	if gi != guadalajaraIdx || guadalajaraIdx != 15 || campaignLevels[gi-1].icon != "cinema" || campaignLevels[gi+1].icon != "circuit" {
		t.Fatalf("GUADALAJARA at %d (guadalajaraIdx %d): it must come right after the CINE, before the CIRCUITO", gi, guadalajaraIdx)
	}
	if campaignSlotOf(gi) != 13 || campaignNumber(gi) != 14 || campaignAtSlot(13) != gi {
		t.Fatalf("GUADALAJARA is row %d of the list, want 14", campaignSlotOf(gi)+1)
	}
	if campaignLevels[campaignAtSlot(12)].icon != "grave" || campaignLevels[campaignAtSlot(14)].icon != "circuit" {
		t.Error("rows 13 and 15 must be the CEMENTERIO and the CIRCUITO")
	}
	if len(campaignLevels) != 28 || campaignN != 28 || campaignShown() != 26 {
		t.Fatalf("%d parties, %d built, %d shown: want 28 / 28 / 26", len(campaignLevels), campaignN, campaignShown())
	}
	slots := campaignSlots()
	if campaignLevels[slots[len(slots)-1]].icon != "station" || campaignLevels[slots[len(slots)-2]].icon != "pirate" {
		t.Error("the last two rows must be the BARCO PIRATA and the ESTACIÓN ESPACIAL")
	}
	for _, e := range campaignLevels {
		if e.icon == "jurassic" {
			t.Error("the secret island is in the list")
		}
	}
	if italyIdx != campaignIdx("italy") || pirateIdx != campaignIdx("pirate") || alcalaIdx != campaignIdx("alcala") ||
		campaignIdx("disco") != 10 || campaignIdx("cinema") != 14 || campaignIdx("station") != 27 {
		t.Errorf("italyIdx %d / pirateIdx %d do not follow the layout", italyIdx, pirateIdx)
	}
	d := levelDefs[gi]
	if d.Title != guadalajaraTitle || d.Icon != "guadalajara" {
		t.Fatalf("level %d is %q (%s)", gi, d.Title, d.Icon)
	}
	if campaignRetired(gi) || !medalLevel("guadalajara") {
		t.Error("GUADALAJARA must be played and give a medal")
	}
	rolled := false
	for _, x := range levelDefs[campaignN:] {
		if x.Title == guadalajaraTitle {
			rolled = true
		}
	}
	if !rolled {
		t.Error("the random mode does not roll GUADALAJARA")
	}
	if nd, z := vipPlace(d, ""); z == nil {
		t.Error("the VIP zone does not fit in GUADALAJARA")
	} else { // ...and it never sweeps a landmark away
		n := map[string]int{}
		for _, p := range nd.Props {
			n[p.Kind]++
		}
		if n["gdl_fountain"] != 1 || n["gdl_park"] != 1 || n["gdl_pergola"] != 2 || n["alc_kiosk"] != 1 || n["dj"] != 1 {
			t.Errorf("the VIP carpet swept a landmark away: %v", n)
		}
	}
	hit := false
	for day := 0; day < 365 && !hit; day++ {
		dc := dailyPick(fmt.Sprintf("2027-%02d-%02d", 1+day/31%12, 1+day%28), campaignN, nil)
		hit = dc.level == gi
	}
	if !hit {
		t.Error("GUADALAJARA is never the RETO DE HOY")
	}
	// the police and the ambulance come by car along the street at the bottom
	if d.Police != "car" || d.CarFrom.X == 0 {
		t.Errorf("police %q / cars from %v: want the normal car on the street", d.Police, d.CarFrom)
	}
}

// the layout builds, is fully walkable from the start (flood fill), nothing stands on anything, every landmark is
// there, and the backdrop band is solid scenery (several seeds: the random mode rolls it with another seed)
func TestGuadalajaraWalkableAndLandmarks(t *testing.T) {
	e := campaignLevels[guadalajaraIdx]
	for _, seed := range []int64{e.seed, e.seed + 1013, 1, 2, 3, 77} {
		d := genNamed("guadalajara", seed, "")
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
		for k, min := range map[string]int{"gdl_fountain": 1, "alc_kiosk": 1, "alc_kioskroof": 1, "gdl_park": 1,
			"gdl_pergola": 2, "gdl_honeystall": 1, "gdl_newskiosk": 1, "alc_pottree": 4, "alc_bin": 3, "cafetable": 4,
			"bench": 3, "lamp": 6, "dj": 1, "punch": 1, "phone": 1} {
			if n[k] < min {
				t.Errorf("seed %d: %s x%d, want at least %d", seed, k, n[k], min)
			}
		}
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
		pts = append(pts, Vec{gdlKiosk.X - 40, gdlKiosk.Y + 6}, Vec{gdlKiosk.X + 40, gdlKiosk.Y + 6}) // up on the bandstand
		for _, pr := range l.Props {                                                                  // every seat and counter (pergolas, honey stall, punch, benches)
			for _, u := range pr.K.Use {
				pts = append(pts, Vec{pr.X + u.X, pr.Y + u.Y})
			}
		}
		for _, p := range pts {
			q := l.NearestFree(p)
			if dist(p, q) > 24 || l.Path(d.Start, q) == nil {
				t.Errorf("seed %d: %v cannot be reached from the start", seed, p)
			}
		}
		cs := d.CellSize
		for y := gdlBackH + cs/2; y < 440; y += cs {
			for x := cs / 2; x < 960; x += cs {
				p := Vec{x, y}
				if l.Free(p) && l.Path(d.Start, p) == nil {
					t.Errorf("seed %d: free spot %v cut off from the start", seed, p)
				}
			}
		}
		for x := 8.0; x < 960; x += 24 {
			for y := 8.0; y < gdlBackH-8; y += 24 {
				if l.Free(Vec{x, y}) {
					t.Fatalf("seed %d: the painted facades are walkable at (%.0f,%.0f)", seed, x, y)
				}
			}
		}
		free, all := 0, 0
		for y := gdlBackH + 8; y < 440; y += 16 {
			for x := 8.0; x < 952; x += 16 {
				all++
				if l.Free(Vec{x, y}) {
					free++
				}
			}
		}
		if free*100 < all*75 {
			t.Errorf("seed %d: only %d%% of the square is free floor", seed, free*100/all)
		}
		// the fountain can be walked round: a point on each side of it is reachable and close to its basin
		fk := propKinds["gdl_fountain"]
		for _, p := range []Vec{{gdlFountain.X, gdlFountain.Y + 14}, {gdlFountain.X, gdlFountain.Y - fk.FH - 14},
			{gdlFountain.X - fk.FW/2 - 14, gdlFountain.Y - fk.FH/2}, {gdlFountain.X + fk.FW/2 + 14, gdlFountain.Y - fk.FH/2}} {
			q := l.NearestFree(p)
			if dist(p, q) > 20 || l.Path(d.Start, q) == nil {
				t.Errorf("seed %d: the side %v of the fountain cannot be reached", seed, p)
			}
		}
	}
}

// the fountain animates (every frame is used and every frame is a different picture)
func TestGuadalajaraFountainMoves(t *testing.T) {
	a := propAnims["gdl_fountain"]
	if a == nil || a.n != gdlFountFrames {
		t.Fatal("the fountain is not an animated prop")
	}
	k := propKinds["gdl_fountain"]
	pics := map[string]int{}
	for f := 0; f < a.n; f++ {
		b := newPix(k.W+2, k.H+2)
		b.ox, b.oy = 1, 1
		a.draw(b, k, 0, f)
		pics[string(b.p)] = f
	}
	if len(pics) != a.n {
		t.Errorf("only %d different pictures for %d frames", len(pics), a.n)
	}
	seen := map[int]bool{}
	for i := 0; i < 200; i++ {
		seen[propAnimFrameAt(a, float64(i)*0.05, 0, false)] = true
	}
	if len(seen) != a.n {
		t.Errorf("only %d of %d frames show", len(seen), a.n)
	}
}

// the signs of the shops: every letter of their names has a glyph (nothing is drawn as a hole)
func TestGuadalajaraSignsHaveGlyphs(t *testing.T) {
	for _, s := range []string{"PASTELERÍA", "MARIAM", "MIEL DE LA ALCARRIA", "MIEL", "PRENSA"} {
		for _, r := range s {
			if r == 'Í' {
				r = 'I'
			}
			if _, ok := gdlGlyphs[r]; !ok {
				t.Errorf("%q: no glyph for %q", s, r)
			}
		}
	}
	if w := gdlTextW("MIEL DE LA ALCARRIA"); w > 118-20 {
		t.Errorf("the honey sign is %d px wide: it leaves its board", w)
	}
	if w := gdlTextW("PASTELERÍA"); w > 86 {
		t.Errorf("the cake shop sign is %d px wide: it leaves its board", w)
	}
}

// texts: Spanish and English for everything new, short bubbles, the title and the shops keep their real names in both
// languages, the music is a pasodoble and differs from its neighbours
func TestGuadalajaraTexts(t *testing.T) {
	for _, l := range venueLines["guadalajara"] {
		if len([]rune(l.es)) > 48 || len([]rune(l.en)) > 48 {
			t.Errorf("bubble too long: %q / %q", l.es, l.en)
		}
		if langDataEN[l.es] == "" {
			t.Errorf("no English for %q", l.es)
		}
	}
	if len(pVenue["guadalajara"]) < 3 || len(venueLines["guadalajara"]) < 5 {
		t.Error("GUADALAJARA needs its newspaper briefs and its guest lines")
	}
	for _, s := range []string{"Fuente, templete y miel de la Alcarria: fiesta en la plaza.", "Verbena con miel y templete",
		hintThemes["guadalajara"].goal, hintThemes["guadalajara"].tip} {
		if en := langDataEN[s]; en == "" || en == s {
			t.Errorf("no English for %q", s)
		}
		if len([]rune(s)) > 60 {
			t.Errorf("too long: %q", s)
		}
	}
	for _, s := range []string{hintThemes["guadalajara"].goal, hintThemes["guadalajara"].tip} {
		if len([]rune(s)) > 48 || len([]rune(langDataEN[s])) > 48 {
			t.Errorf("hint too long: %q", s)
		}
	}
	if langDataEN[guadalajaraTitle] != guadalajaraTitle {
		t.Errorf("the title is a proper name: %q in English", langDataEN[guadalajaraTitle])
	}
	for _, l := range append(venueLines["guadalajara"], venueLine{pVenue["guadalajara"][4].es, pVenue["guadalajara"][4].en}) {
		if strings.Contains(l.es, "Pastelería Mariam") && !strings.Contains(l.en, "Pastelería Mariam") {
			t.Errorf("the shop's name is translated: %q", l.en)
		}
	}
	if genreForTitle(guadalajaraTitle).id != "flamenco" {
		t.Errorf("the music of GUADALAJARA is %q", genreForTitle(guadalajaraTitle).id)
	}
	for _, nb := range []int{campaignAtSlot(12), campaignAtSlot(14)} { // its neighbours in the list
		if genreForTitle(campaignLevels[nb].title).id == "flamenco" {
			t.Errorf("GUADALAJARA sounds like its neighbour %s", campaignLevels[nb].title)
		}
	}
	if levelIconArt["guadalajara"] == nil || levelIconColors["guadalajara"] == 0 || seatThemes["guadalajara"].kits[kitCafe] == 0 {
		t.Error("GUADALAJARA needs its list icon, its swatch and its café tables")
	}
}

// ---------------------------------------------------------------------------------------------------------------
// saves of the 27-party layout (rev 5) and older: GUADALAJARA is inserted at index 15 without losing anything
// ---------------------------------------------------------------------------------------------------------------

func TestGuadalajaraMigrationRev5(t *testing.T) {
	var all []int
	for i := 0; i < 27; i++ {
		all = append(all, i)
	}
	cases := []struct {
		name     string
		p        Progress
		unlocked int
		won      []int // won_N after the migration
		notWon   []int
	}{
		{"new game", revSave(5, 1, nil), 1, nil, []int{15}},
		{"graveyard next", revSave(5, 14, []int{0, 12}), 14, []int{0, 12}, []int{13, 15}},
		{"frontier on the retired CINE", revSave(5, 15, []int{0, 13}), 17, []int{0, 13}, []int{15, 16}},
		{"graveyard beaten, circuit next", revSave(5, 16, []int{0, 13}), 17, []int{0, 13}, []int{15, 16}},
		{"circuit beaten", revSave(5, 17, []int{0, 13, 15}), 18, []int{0, 13, 16}, []int{15, 17}},
		{"everything (rev 5)", revSave(5, 27, all), 28, []int{0, 6, 10, 13, 14, 16, 21, 26, 27}, []int{15}},
		{"alcala era (rev 4)", revSave(4, 26, []int{0, 5, 13, 19, 25}), 28, []int{0, 5, 14, 21, 27}, []int{6, 15}},
		{"v1.0.0 (rev 2)", revSave(2, 25, []int{0, 18, 19, 24}), 28, []int{0, 20, 21, 27}, []int{6, 15, 26}},
	}
	for _, c := range cases {
		before := cloneProgress(c.p)
		p := c.p
		migrateProgress(&p)
		if p.CampaignRev != campaignRev || campaignRev != 6 {
			t.Errorf("%s: rev %d", c.name, p.CampaignRev)
		}
		if p.Unlocked != c.unlocked {
			t.Errorf("%s: unlocked %d, want %d", c.name, p.Unlocked, c.unlocked)
		}
		for _, i := range c.won {
			if p.Stats[fmt.Sprintf("won_%d", i)] != 1 {
				t.Errorf("%s: won_%d lost", c.name, i)
			}
		}
		for _, i := range c.notWon {
			if p.Stats[fmt.Sprintf("won_%d", i)] != 0 {
				t.Errorf("%s: won_%d given for free", c.name, i)
			}
		}
		wins, bw := 0, 0
		for k, v := range p.Stats {
			if strings.HasPrefix(k, "won_") && v > 0 {
				wins++
			}
		}
		for k, v := range before.Stats {
			if strings.HasPrefix(k, "won_") && v > 0 {
				bw++
			}
		}
		if wins != bw || p.Stats["distinct"] != before.Stats["distinct"] || fmt.Sprint(p.Medals) != fmt.Sprint(before.Medals) {
			t.Errorf("%s: wins %d -> %d, or distinct / medals touched", c.name, bw, wins)
		}
		again := cloneProgress(p)
		migrateProgress(&again)
		if again.Unlocked != p.Unlocked || fmt.Sprint(again.Stats) != fmt.Sprint(p.Stats) || again.CampaignRev != p.CampaignRev {
			t.Errorf("%s: not idempotent", c.name)
		}
	}
}

// the real tester saves of v1.0.0 (COPIES; the test reads GUADALAJARA_SAVES, or ALCALA_SAVES, a folder of copies, and
// is skipped without it): every win keeps its party, the frontier its party, nothing is won for free, and migrating
// twice changes nothing
func TestGuadalajaraMigrationTesterSaves(t *testing.T) {
	dir := os.Getenv("GUADALAJARA_SAVES")
	if dir == "" {
		dir = os.Getenv("ALCALA_SAVES")
	}
	if dir == "" {
		t.Skip("GUADALAJARA_SAVES not set (folder with COPIES of old saves)")
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	if len(files) == 0 {
		t.Fatal("no saves in GUADALAJARA_SAVES")
	}
	_ = layerGame(t, false)
	for _, f := range files {
		p, ok := readProgressFile(f)
		if !ok {
			t.Fatalf("%s does not load", f)
		}
		old := cloneProgress(p)
		titles := oldLayoutTitles(old.CampaignRev)
		migrateProgress(&p)
		for k, v := range old.Stats {
			var i int
			if _, err := fmt.Sscanf(k, "won_%d", &i); err != nil || v == 0 {
				continue
			}
			want, got := titles[i], ""
			for j := range campaignLevels {
				if p.Stats[fmt.Sprintf("won_%d", j)] > 0 && levelDefs[j].Title == want {
					got = want
				}
			}
			if got == "" {
				t.Errorf("%s: the win of %s (old won_%d) is lost", filepath.Base(f), want, i)
			}
		}
		if p.Stats[fmt.Sprintf("won_%d", guadalajaraIdx)] != 0 || p.Stats[fmt.Sprintf("won_%d", alcalaIdx)] != 0 {
			t.Errorf("%s: GUADALAJARA or ALCALÁ won for free", filepath.Base(f))
		}
		if old.Unlocked >= 1 {
			was := titles[minI(old.Unlocked, len(titles))-1]
			open := false
			for j := 0; j < p.Unlocked; j++ {
				if levelDefs[j].Title == was {
					open = true
				}
			}
			if !open {
				t.Errorf("%s: %s was open and is locked now", filepath.Base(f), was)
			}
		}
		album := 0
		for k := range old.Album {
			if _, ok := p.Album[k]; ok || strings.HasPrefix(k, "cop:") {
				album++
			}
		}
		if fmt.Sprint(p.Medals) != fmt.Sprint(old.Medals) || album != len(old.Album) || p.Wallet != old.Wallet ||
			len(p.Trophies) != len(old.Trophies) {
			t.Errorf("%s: medals, album, wallet or trophies touched", filepath.Base(f))
		}
		again := cloneProgress(p)
		migrateProgress(&again)
		if again.Unlocked != p.Unlocked || fmt.Sprint(again.Stats) != fmt.Sprint(p.Stats) {
			t.Errorf("%s: migrating twice changes the save", filepath.Base(f))
		}
		won := []string{}
		for j := range campaignLevels {
			if p.Stats[fmt.Sprintf("won_%d", j)] > 0 {
				won = append(won, campaignLevels[j].icon)
			}
		}
		t.Logf("%s: rev %d unlocked %d -> rev %d unlocked %d (%s); won %v", filepath.Base(f), old.CampaignRev, old.Unlocked,
			p.CampaignRev, p.Unlocked, levelDefs[campaignFrontier(p.Unlocked)-1].Title, won)
	}
}

// an editor correction saved for a party behind GUADALAJARA with the old number keeps working
func TestGuadalajaraOldOverrideKeys(t *testing.T) {
	gi := guadalajaraIdx
	if campaignKey(gi) != "c16_guadalajara" || campaignKeyRev5(gi) != "" || campaignKeyRev3(gi) != "" ||
		campaignKeyRev2(gi) != "" || campaignKeyRev1(gi) != "" {
		t.Errorf("GUADALAJARA keys: %q %q %q", campaignKey(gi), campaignKeyRev5(gi), campaignKeyRev3(gi))
	}
	circ := campaignIdx("circuit")
	if campaignKey(circ) != "c17_circuit" || campaignKeyRev5(circ) != "c16_circuit" || campaignKeyRev3(circ) != "c15_circuit" {
		t.Errorf("the circuit: %q (old %q / %q)", campaignKey(circ), campaignKeyRev5(circ), campaignKeyRev3(circ))
	}
	cine := campaignIdx("cinema")
	if campaignKeyRev5(cine) != "" || campaignKey(cine) != "c15_cinema" || campaignKeyRev3(cine) != "c14_cinema" {
		t.Errorf("the parties before GUADALAJARA keep their key: %q %q", campaignKey(cine), campaignKeyRev3(cine))
	}
	st := campaignIdx("station")
	if campaignKey(st) != "c28_station" || campaignKeyRev5(st) != "c27_station" || campaignKeyRev3(st) != "c26_station" ||
		campaignKeyRev2(st) != "c25_station" || campaignKeyRev1(st) != "c24_station" {
		t.Errorf("the station keys: %q %q %q %q %q", campaignKey(st), campaignKeyRev5(st), campaignKeyRev3(st),
			campaignKeyRev2(st), campaignKeyRev1(st))
	}
	train := campaignIdx("train")
	if campaignKey(train) != "c23_train" || campaignKeyRev5(train) != "c22_train" || campaignKeyRev3(train) != "c21_train" ||
		campaignKeyRev1(train) != "c20_train" {
		t.Errorf("the train station keys: %q %q %q %q", campaignKey(train), campaignKeyRev5(train), campaignKeyRev3(train),
			campaignKeyRev1(train))
	}
	// an override file saved with the 27-party key of the circuit is applied to the circuit
	if rev5Index(circ) != 15 || rev5Index(gi) != -1 || rev3Index(gi) != -1 || rev3Index(circ) != 14 {
		t.Errorf("rev5Index %d / rev3Index %d", rev5Index(circ), rev3Index(circ))
	}
}
