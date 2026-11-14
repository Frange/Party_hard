package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ALCALÁ DE HENARES is the 7th row of the list (index alcalaIdx, right after PARÍS, the old TORRE EIFFEL); the space station is
// still the last row and the island stays out of the list.
func TestAlcalaSeventhOfTheList(t *testing.T) {
	_ = layerGame(t, false)
	ai := campaignIdx("alcala")
	if ai != alcalaIdx || alcalaIdx != 6 || campaignLevels[ai-1].icon != "eiffel" || campaignLevels[ai+1].icon != "boat" {
		t.Fatalf("ALCALÁ at %d (alcalaIdx %d): it must come right after PARÍS", ai, alcalaIdx)
	}
	if campaignSlotOf(ai) != 6 || campaignNumber(ai) != 7 || campaignAtSlot(6) != ai {
		t.Fatalf("ALCALÁ is row %d of the list, want 7", campaignSlotOf(ai)+1)
	}
	if len(campaignLevels) != 28 || campaignN != 28 || campaignShown() != 26 { // (GUADALAJARA joined in v1.4.0 too)
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
	if italyIdx != campaignIdx("italy") || pirateIdx != campaignIdx("pirate") || campaignIdx("disco") != 10 || campaignIdx("cinema") != 14 {
		t.Errorf("italyIdx %d / pirateIdx %d do not follow the layout", italyIdx, pirateIdx)
	}
	d := levelDefs[ai]
	if d.Title != alcalaTitle || d.Icon != "alcala" {
		t.Fatalf("level %d is %q (%s)", ai, d.Title, d.Icon)
	}
	// not retired, it gives a medal, and the random mode rolls it too
	if campaignRetired(ai) || !medalLevel("alcala") {
		t.Error("ALCALÁ must be played and give a medal")
	}
	rolled := false
	for _, x := range levelDefs[campaignN:] {
		if x.Title == alcalaTitle {
			rolled = true
		}
	}
	if !rolled {
		t.Error("the random mode does not roll ALCALÁ")
	}
	// the VIP zone fits (the RETO DE HOY needs it too)
	if z, _ := vipPlace(d, ""); z == nil {
		t.Error("the VIP zone does not fit in ALCALÁ")
	}
	// the RETO DE HOY can be ALCALÁ (some day of the year) once it is open
	hit := false
	for day := 0; day < 365 && !hit; day++ {
		dc := dailyPick(fmt.Sprintf("2027-%02d-%02d", 1+day/31%12, 1+day%28), campaignN, nil)
		hit = dc.level == ai
	}
	if !hit {
		t.Error("ALCALÁ is never the RETO DE HOY")
	}
}

// the layout builds, is fully walkable from the start (flood fill), nothing stands on anything, every landmark is
// there, and the backdrop band is solid scenery (several seeds: the random mode rolls it with another seed)
func TestAlcalaWalkableAndLandmarks(t *testing.T) {
	e := campaignLevels[alcalaIdx]
	for _, seed := range []int64{e.seed, e.seed + 1013, 1, 2, 3, 77} {
		d := genNamed("alcala", seed, "")
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
		for k, min := range map[string]int{"alc_kiosk": 1, "alc_kioskroof": 1, "alc_monument": 1, "alc_bronzebench": 1,
			"alc_stall": 2, "alc_pottree": 4, "alc_stork": 3, "alc_bin": 3, "cafetable": 4, "bench": 4, "lamp": 6,
			"dj": 1, "punch": 1, "bar": 1, "phone": 1} {
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
		pts = append(pts, Vec{alcKiosk.X - 40, alcKiosk.Y + 6}, Vec{alcKiosk.X + 40, alcKiosk.Y + 6}) // up on the bandstand
		for _, p := range pts {
			q := l.NearestFree(p)
			if dist(p, q) > 24 || l.Path(d.Start, q) == nil {
				t.Errorf("seed %d: %v cannot be reached from the start", seed, p)
			}
		}
		// every free cell of the square is reachable (no pocket closed by the furniture)
		cs := d.CellSize
		for y := alcBackH + cs/2; y < 440; y += cs {
			for x := cs / 2; x < 960; x += cs {
				p := Vec{x, y}
				if l.Free(p) && l.Path(d.Start, p) == nil {
					t.Errorf("seed %d: free spot %v cut off from the start", seed, p)
				}
			}
		}
		// the backdrop is scenery: nobody walks on it
		for x := 8.0; x < 960; x += 24 {
			for y := 8.0; y < alcBackH-8; y += 24 {
				if l.Free(Vec{x, y}) {
					t.Fatalf("seed %d: the painted facades are walkable at (%.0f,%.0f)", seed, x, y)
				}
			}
		}
		// the square is wide open: most of it is free floor
		free, all := 0, 0
		for y := alcBackH + 8; y < 440; y += 16 {
			for x := 8.0; x < 952; x += 16 {
				all++
				if l.Free(Vec{x, y}) {
					free++
				}
			}
		}
		if free*100 < all*80 {
			t.Errorf("seed %d: only %d%% of the square is free floor", seed, free*100/all)
		}
	}
}

// the storks animate: still most of the time, and every cycle they clatter, bow and flap (every frame is used and
// every frame is a different picture)
func TestAlcalaStorksMove(t *testing.T) {
	a := propAnims["alc_stork"]
	if a == nil || a.n != 5 {
		t.Fatal("the stork is not an animated prop")
	}
	seen := map[int]int{}
	for i := 0; i < 900; i++ {
		seen[propAnimFrameAt(a, float64(i)*0.01*alcStorkCycle, 0, false)]++
	}
	for f := 0; f < a.n; f++ {
		if seen[f] == 0 {
			t.Errorf("frame %d never shows", f)
		}
	}
	if seen[0]*100 < 900*60 {
		t.Errorf("the stork moves too much: still only %d of 900", seen[0])
	}
	k := propKinds["alc_stork"]
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
	// two storks of the level do not move in step (their phase comes from their place)
	ph := map[float64]bool{}
	for _, p := range alcStorkSpots {
		ph[propAnimPhase(&Prop{X: p.X, Y: p.Y})] = true
	}
	if len(ph) < 2 {
		t.Error("the storks all move at once")
	}
}

// texts: Spanish and English for everything new, short bubbles, the title is the real name in both languages, the
// music is a pasodoble and differs from its neighbours
func TestAlcalaTexts(t *testing.T) {
	for _, l := range venueLines["alcala"] {
		if len([]rune(l.es)) > 48 || len([]rune(l.en)) > 48 {
			t.Errorf("bubble too long: %q / %q", l.es, l.en)
		}
		if langDataEN[l.es] == "" {
			t.Errorf("no English for %q", l.es)
		}
	}
	if len(pVenue["alcala"]) < 3 || len(venueLines["alcala"]) < 5 {
		t.Error("ALCALÁ needs its newspaper briefs and its guest lines")
	}
	for _, s := range []string{"Cigüeñas en los tejados y fiesta mayor en la plaza.", "Fiesta mayor con cigüeñas",
		hintThemes["alcala"].goal, hintThemes["alcala"].tip} {
		if en := langDataEN[s]; en == "" || en == s {
			t.Errorf("no English for %q", s)
		}
	}
	if langDataEN[alcalaTitle] != alcalaTitle {
		t.Errorf("the title is a proper name: %q in English", langDataEN[alcalaTitle])
	}
	if genreForTitle(alcalaTitle).id != "flamenco" {
		t.Errorf("the music of ALCALÁ is %q", genreForTitle(alcalaTitle).id)
	}
	for _, nb := range []int{alcalaIdx - 1, alcalaIdx + 1} {
		if genreForTitle(campaignLevels[nb].title).id == "flamenco" {
			t.Errorf("ALCALÁ sounds like its neighbour %s", campaignLevels[nb].title)
		}
	}
	if levelIconArt["alcala"] == nil || levelIconColors["alcala"] == 0 || seatThemes["alcala"].kits[kitCafe] == 0 {
		t.Error("ALCALÁ needs its list icon, its swatch and its café tables")
	}
	// no real person in the texts of the party
	all := alcalaTitle
	for _, l := range venueLines["alcala"] {
		all += " " + l.es + " " + l.en
	}
	for _, p := range pVenue["alcala"] {
		all += " " + p.es + " " + p.en
	}
	for _, w := range []string{"Cervantes", "Cisneros", "Quijote", "Azaña"} {
		if strings.Contains(all, w) {
			t.Errorf("the texts of ALCALÁ name %s", w)
		}
	}
}

// ---------------------------------------------------------------------------------------------------------------
// saves of the 26-party layout (rev 3 / 4) and older: ALCALÁ is inserted at index 6 without losing anything
// ---------------------------------------------------------------------------------------------------------------

func revSave(rev, unlocked int, won []int) Progress {
	p := Progress{Unlocked: unlocked, CampaignRev: rev, Stats: map[string]int{"played": 60, "distinct": len(won)},
		Medals: map[string]int{"house": 3, "eiffel": 2, "boat": 1, "station": 3}}
	for _, i := range won {
		p.Stats[fmt.Sprintf("won_%d", i)] = 1
	}
	raw, _ := json.Marshal(p) // through JSON like the real file
	var q Progress
	json.Unmarshal(raw, &q)
	return q
}

func TestAlcalaMigrationRev4(t *testing.T) {
	var all []int
	for i := 0; i < 26; i++ {
		all = append(all, i)
	}
	cases := []struct {
		name     string
		p        Progress
		unlocked int
		won      []int // won_N after the migration
		notWon   []int
	}{
		{"new game", revSave(4, 1, nil), 1, nil, []int{6}},
		{"eiffel next", revSave(4, 6, []int{0, 1, 2, 3, 4}), 6, []int{0, 4}, []int{5, 6}},
		{"eiffel beaten, cruise next", revSave(4, 7, []int{0, 1, 2, 3, 4, 5}), 8, []int{0, 5}, []int{6, 7}},
		{"cruise beaten", revSave(4, 8, []int{0, 1, 2, 3, 4, 5, 6}), 9, []int{0, 5, 7}, []int{6, 8}},
		// (the saves go on to rev 6: GUADALAJARA, inserted at 15, moves the parties from 15 on once more)
		{"italy next (rev 3)", revSave(3, 20, []int{18}), 22, []int{20}, []int{18, 19, 21, 6, 15}},
		{"everything (rev 4)", revSave(4, 26, all), 28, []int{0, 5, 7, 10, 14, 16, 21, 26, 27}, []int{6, 15}},
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
		wins := 0
		for k, v := range p.Stats {
			if strings.HasPrefix(k, "won_") && v > 0 {
				wins++
			}
		}
		bw := 0
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
		if again.Unlocked != p.Unlocked || fmt.Sprint(again.Stats) != fmt.Sprint(p.Stats) {
			t.Errorf("%s: not idempotent", c.name)
		}
	}
}

// the real tester saves of v1.0.0 (copies in the scratchpad; the test reads ALCALA_SAVES, a folder of copies, and is
// skipped without it): every win keeps its party, the frontier its party, nothing is won for free
func TestAlcalaMigrationTesterSaves(t *testing.T) {
	dir := os.Getenv("ALCALA_SAVES")
	if dir == "" {
		t.Skip("ALCALA_SAVES not set (folder with COPIES of old saves)")
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	if len(files) == 0 {
		t.Fatal("no saves in ALCALA_SAVES")
	}
	_ = layerGame(t, false)
	for _, f := range files {
		p, ok := readProgressFile(f)
		if !ok {
			t.Fatalf("%s does not load", f)
		}
		old := cloneProgress(p)
		// the party each win belonged to, by title, in its own layout (v1.0.0 saves are rev 2: 25 parties)
		titles := oldLayoutTitles(old.CampaignRev)
		migrateProgress(&p)
		for k, v := range old.Stats {
			var i int
			if _, err := fmt.Sscanf(k, "won_%d", &i); err != nil || v == 0 {
				continue
			}
			want := titles[i]
			got := ""
			for j := range campaignLevels {
				if p.Stats[fmt.Sprintf("won_%d", j)] > 0 && levelDefs[j].Title == want {
					got = want
				}
			}
			if got == "" {
				t.Errorf("%s: the win of %s (old won_%d) is lost", filepath.Base(f), want, i)
			}
		}
		if p.Stats[fmt.Sprintf("won_%d", alcalaIdx)] != 0 {
			t.Errorf("%s: ALCALÁ won for free", filepath.Base(f))
		}
		// the frontier: the newest party open before is still open
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
		// (the album may only lose "cop:N" cards: the POLICÍAS reconciliation, CopRev, not the campaign layout)
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
		t.Logf("%s: rev %d unlocked %d -> rev %d unlocked %d (%s)", filepath.Base(f), old.CampaignRev, old.Unlocked,
			p.CampaignRev, p.Unlocked, levelDefs[campaignFrontier(p.Unlocked)-1].Title)
	}
}

// oldLayoutTitles: the titles of the campaign in an old layout (rev 2: 25 parties without the pirate ship, ALCALÁ and
// GUADALAJARA; rev 3/4: 26 without ALCALÁ and GUADALAJARA; rev 5: 27 without GUADALAJARA).
func oldLayoutTitles(rev int) []string {
	var out []string
	for i, e := range campaignLevels {
		if rev <= 4 && i == alcalaIdx || rev <= 5 && i == guadalajaraIdx || rev <= 2 && i == pirateIdx {
			continue
		}
		out = append(out, levelDefs[i].Title)
		_ = e
	}
	return out
}

// an editor correction saved for a party behind ALCALÁ with the old number keeps working
func TestAlcalaOldOverrideKeys(t *testing.T) {
	if campaignKey(alcalaIdx) != "c07_alcala" || campaignKeyRev3(alcalaIdx) != "" || campaignKeyRev2(alcalaIdx) != "" || campaignKeyRev1(alcalaIdx) != "" {
		t.Errorf("ALCALÁ keys: %q %q", campaignKey(alcalaIdx), campaignKeyRev3(alcalaIdx))
	}
	boat := campaignIdx("boat")
	if campaignKey(boat) != "c08_boat" || campaignKeyRev3(boat) != "c07_boat" {
		t.Errorf("the cruise: %q (old %q)", campaignKey(boat), campaignKeyRev3(boat))
	}
	if campaignKeyRev3(5) != "" || campaignKey(5) != "c06_eiffel" {
		t.Error("the parties before ALCALÁ keep their key")
	}
	st := campaignIdx("station")
	if campaignKey(st) != "c28_station" || campaignKeyRev3(st) != "c26_station" || campaignKeyRev2(st) != "c25_station" ||
		campaignKeyRev1(st) != "c24_station" {
		t.Errorf("the station keys: %q %q %q %q", campaignKey(st), campaignKeyRev3(st), campaignKeyRev2(st), campaignKeyRev1(st))
	}
	if campaignKeyRev2(pirateIdx) != "" || campaignKeyRev3(pirateIdx) != "c25_pirate" || campaignKeyRev1(italyIdx) != "" ||
		campaignKeyRev3(italyIdx) != "c20_italy" || campaignKeyRev1(italyIdx+1) != "c20_train" {
		t.Errorf("pirate %q / %q, italy %q, train %q", campaignKeyRev2(pirateIdx), campaignKeyRev3(pirateIdx),
			campaignKeyRev1(italyIdx), campaignKeyRev1(italyIdx+1))
	}
}
