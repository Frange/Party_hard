package main

import (
	"fmt"
	"strings"
	"testing"
)

// BARCELONA is the 4th row of the list: it took the place (index, icon "ranch", editor key, won_N, medal) of the old
// ranch party, so the saves keep everything without a migration.
func TestBarcelonaFourthOfTheList(t *testing.T) {
	_ = layerGame(t, false)
	bi := campaignIdx(bcnIcon)
	if bi != 3 || campaignSlotOf(bi) != 3 || campaignNumber(bi) != 4 || campaignAtSlot(3) != bi {
		t.Fatalf("BARCELONA at index %d, row %d: want index 3, row 4", bi, campaignSlotOf(bi)+1)
	}
	e := campaignLevels[bi]
	if e.kind != "barcelona" || e.title != barcelonaTitle || e.level != nil || e.retired {
		t.Fatalf("party 4 is %+v", e)
	}
	if campaignKey(bi) != "c04_ranch" {
		t.Errorf("the editor key moved: %q", campaignKey(bi))
	}
	if len(campaignLevels) != 28 || campaignN != 28 {
		t.Fatalf("%d parties, %d built", len(campaignLevels), campaignN)
	}
	d := levelDefs[bi]
	if d.Title != barcelonaTitle || d.Icon != bcnIcon {
		t.Fatalf("level %d is %q (%s)", bi, d.Title, d.Icon)
	}
	if !medalLevel(bcnIcon) {
		t.Error("BARCELONA must give a medal")
	}
	// the old ranch is gone: no title, no garage, no generator left behind
	for _, x := range levelDefs {
		if x.Title == "BULL'S EYE" {
			t.Errorf("the old ranch is still in the game (%d props)", len(x.Props))
		}
	}
	rolled := false
	for _, x := range levelDefs[campaignN:] {
		rolled = rolled || x.Title == barcelonaTitle
	}
	if !rolled {
		t.Error("the random mode does not roll BARCELONA")
	}
	if z, _ := vipPlace(d, ""); z == nil {
		t.Error("the VIP zone does not fit in BARCELONA")
	}
	hit := false
	for day := 0; day < 365 && !hit; day++ {
		dc := dailyPick(fmt.Sprintf("2027-%02d-%02d", 1+day/31%12, 1+day%28), campaignN, nil)
		hit = dc.level == bi
	}
	if !hit {
		t.Error("BARCELONA is never the RETO DE HOY")
	}
	if d.Police != "car" {
		t.Errorf("police %q, want the ordinary car", d.Police)
	}
}

// a save that had won the ranch (won_3, medal "ranch", frontier past it) keeps it all on BARCELONA
func TestBarcelonaKeepsTheRanchSave(t *testing.T) {
	p := Progress{Unlocked: 6, CampaignRev: campaignRev, Stats: map[string]int{"won_3": 1, "won_0": 1},
		Medals: map[string]int{"ranch": 2}}
	before := cloneProgress(p)
	migrateProgress(&p)
	if p.Unlocked != 6 || p.Stats["won_3"] != 1 || p.Medals[bcnIcon] != 2 || fmt.Sprint(p.Stats) != fmt.Sprint(before.Stats) {
		t.Errorf("the save changed: %+v -> %+v", before, p)
	}
}

// the layout builds, is fully walkable from the start (flood fill), nothing stands on anything, every landmark is
// there, the backdrop band is solid scenery and the square is wide open (several seeds: the random mode too)
func TestBarcelonaWalkableAndLandmarks(t *testing.T) {
	e := campaignLevels[campaignIdx(bcnIcon)]
	for _, seed := range []int64{e.seed, e.seed + 1013, 1, 2, 3, 77} {
		d := genNamed("barcelona", seed, "")
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
		fl := map[int]int{}
		for _, p := range d.Props {
			n[p.Kind]++
			if p.Kind == "flagpole" || p.Kind == "flagmini" {
				fl[p.Var]++
			}
		}
		for k, min := range map[string]int{"bcn_mosaic": 1, "bcn_fountain": 2, "bcn_pavilion": 1, "bcn_bench": 1, "bcn_kiosk": 1,
			"bcn_cablecar": 1, "flagpole": 2, "flagmini": 2, "palm": 4, "tree": 2, "lamp": 6, "bench": 3, "cafetable": 6,
			"alc_bin": 3, "dj": 1, "speaker": 2, "punch": 1, "phone": 1} {
			if n[k] < min {
				t.Errorf("seed %d: %s x%d, want at least %d", seed, k, n[k], min)
			}
		}
		if fl[flagSpain] < 2 || fl[flagCatalonia] < 2 {
			t.Errorf("seed %d: flags of Spain %d, of Catalonia %d: want both on the balcony and on a pole", seed, fl[flagSpain], fl[flagCatalonia])
		}
		// the DJ plays on the mosaic, the dance floor is the mosaic
		if len(d.Dance) != 1 || !d.Dance[0].Has(bcnMosaic.X, bcnMosaic.Y+10) || dist(d.Performers[0].Pos, bcnMosaic) > 60 {
			t.Errorf("seed %d: the dance floor %v / the DJ %v are not on the mosaic", seed, d.Dance, d.Performers)
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
		for _, p := range pts {
			q := l.NearestFree(p)
			if dist(p, q) > 24 || l.Path(d.Start, q) == nil {
				t.Errorf("seed %d: %v cannot be reached from the start", seed, p)
			}
		}
		cs := d.CellSize
		for y := bcnBackH + cs/2; y < 440; y += cs {
			for x := cs / 2; x < 960; x += cs {
				p := Vec{x, y}
				if l.Free(p) && l.Path(d.Start, p) == nil {
					t.Errorf("seed %d: free spot %v cut off from the start", seed, p)
				}
			}
		}
		for x := 8.0; x < 960; x += 24 {
			for y := 8.0; y < bcnBackH-8; y += 24 {
				if l.Free(Vec{x, y}) {
					t.Fatalf("seed %d: the painted backdrop is walkable at (%.0f,%.0f)", seed, x, y)
				}
			}
		}
		free, all := 0, 0
		for y := bcnBackH + 8; y < 440; y += 16 {
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
		// the middle of the square (around the mosaic) is clear: the mosaic itself is free floor
		for _, p := range []Vec{bcnMosaic, {bcnMosaic.X - 60, bcnMosaic.Y + 20}, {bcnMosaic.X + 60, bcnMosaic.Y + 20}} {
			if !l.Free(p) {
				t.Errorf("seed %d: the mosaic is blocked at %v", seed, p)
			}
		}
	}
}

// the flags wave: every frame is a different picture, the cloth has the right colours (Spain: red, yellow, red;
// Catalonia: four red stripes on yellow; France: blue, white, red) and the free end moves more than the pole side
func TestFlagsWaveAndHaveTheirColours(t *testing.T) {
	for _, name := range []string{"flagpole", "flagmini"} {
		a, k := propAnims[name], propKinds[name]
		if a == nil || k == nil || a.n < 4 || k.Vars != flagVars {
			t.Fatalf("%s is not an animated flag with %d variants", name, flagVars)
		}
		for v := 0; v < flagVars; v++ {
			pics := map[string]bool{}
			for f := 0; f < a.n; f++ {
				b := newPix(k.W+2, k.H+2)
				b.ox, b.oy = 1, 1
				a.draw(b, k, v, f)
				pics[string(b.p)] = true
			}
			if len(pics) != a.n {
				t.Errorf("%s v%d: %d different pictures for %d frames", name, v, len(pics), a.n)
			}
		}
	}
	// the stripes of the flat cloth
	stripes := func(v, w, h int, x int) []string {
		var out []string
		for y := 0; y < h; y++ {
			c := flagColourAt(v, x, y, w, h)
			n := map[[3]uint8]string{{flagRed.R, flagRed.G, flagRed.B}: "R", {flagYellow.R, flagYellow.G, flagYellow.B}: "Y",
				{flagBlue.R, flagBlue.G, flagBlue.B}: "B", {flagWhite.R, flagWhite.G, flagWhite.B}: "W"}[[3]uint8{c.R, c.G, c.B}]
			if len(out) == 0 || out[len(out)-1] != n {
				out = append(out, n)
			}
		}
		return out
	}
	for _, w := range []int{flagClothW - 1, flagMiniW} {
		if s := strings.Join(stripes(flagSpain, w, flagClothH(flagSpain, w), 3), ""); s != "RYR" {
			t.Errorf("Spain (w %d): %s", w, s)
		}
		if s := strings.Join(stripes(flagCatalonia, w, flagClothH(flagCatalonia, w), 3), ""); s != "YRYRYRYRY" {
			t.Errorf("Catalonia (w %d): %s, want four red stripes on yellow", w, s)
		}
		var fr []string
		for x := 0; x < w; x++ {
			c := flagColourAt(flagFrance, x, 2, w, flagClothH(flagFrance, w))
			k := map[[3]uint8]string{{flagRed.R, flagRed.G, flagRed.B}: "R", {flagBlue.R, flagBlue.G, flagBlue.B}: "B",
				{flagWhite.R, flagWhite.G, flagWhite.B}: "W"}[[3]uint8{c.R, c.G, c.B}]
			if len(fr) == 0 || fr[len(fr)-1] != k {
				fr = append(fr, k)
			}
		}
		if s := strings.Join(fr, ""); s != "BWR" {
			t.Errorf("France (w %d): %s, want blue, white, red from the pole", w, s)
		}
	}
	// the wave: the free end moves, the side by the pole hardly does
	moveAt := func(x int) int {
		lo, hi := 99, -99
		for f := 0; f < flagFrames; f++ {
			dy, _ := flagWave(x, flagClothW, f)
			lo, hi = minI(lo, dy), maxI(hi, dy)
		}
		return hi - lo
	}
	if moveAt(0) > 1 || moveAt(flagClothW-1) < 3 {
		t.Errorf("the wave: %d px at the pole, %d px at the free end", moveAt(0), moveAt(flagClothW-1))
	}
}

// the fountains and the cable car move; every frame is a different picture
func TestBarcelonaAnimatedProps(t *testing.T) {
	for _, name := range []string{"bcn_fountain", "bcn_cablecar"} {
		a, k := propAnims[name], propKinds[name]
		if a == nil || k == nil {
			t.Fatalf("%s is not animated", name)
		}
		pics := map[string]bool{}
		for f := 0; f < a.n; f++ {
			b := newPix(k.W+2, k.H+2)
			b.ox, b.oy = 1, 1
			a.draw(b, k, 0, f)
			pics[string(b.p)] = true
		}
		if len(pics) != a.n {
			t.Errorf("%s: %d different pictures for %d frames", name, len(pics), a.n)
		}
	}
	// the backdrop is made once and has its landmarks' colours (brick of the arch and the market, the sea, the sky)
	b := barcelonaBackdrop()
	if b.w != 960 || b.h != int(bcnBackH) {
		t.Fatalf("backdrop %dx%d", b.w, b.h)
	}
	for _, c := range []struct {
		x, y int
		what string
	}{{820, 140, "the Arc de Triomf"}, {800, 70, "the sea"}, {480, 4, "the sky"}, {665, 120, "the Sagrada Família"}} {
		px := b.get(c.x, c.y)
		if px.A == 0 {
			t.Errorf("%s is not painted at %d,%d", c.what, c.x, c.y)
		}
	}
}

// texts: Spanish and English for everything new, short bubbles, the title is the real name in both languages, the
// music differs from its neighbours; no real person is named
func TestBarcelonaTexts(t *testing.T) {
	for _, l := range venueLines[bcnIcon] {
		if len([]rune(l.es)) > 48 || len([]rune(l.en)) > 48 {
			t.Errorf("bubble too long: %q / %q", l.es, l.en)
		}
		if langDataEN[l.es] == "" {
			t.Errorf("no English for %q", l.es)
		}
	}
	if len(pVenue[bcnIcon]) < 3 || len(venueLines[bcnIcon]) < 5 {
		t.Error("BARCELONA needs its newspaper briefs and its guest lines")
	}
	for _, s := range []string{"Sol, mar y mosaico: la plaza baila a la catalana.", "Rumba en la plaza",
		hintThemes[bcnIcon].goal, hintThemes[bcnIcon].tip} {
		if en := langDataEN[s]; en == "" || en == s {
			t.Errorf("no English for %q", s)
		}
	}
	if langDataEN[barcelonaTitle] != barcelonaTitle {
		t.Errorf("the title is a proper name: %q in English", langDataEN[barcelonaTitle])
	}
	bi := campaignIdx(bcnIcon)
	_ = layerGame(t, false)
	for _, nb := range []int{bi - 1, bi + 1} {
		if levelDefs[nb].Title == "" || genreForTitle(levelDefs[nb].Title).id == genreForTitle(barcelonaTitle).id {
			t.Errorf("BARCELONA sounds like its neighbour %q", levelDefs[nb].Title)
		}
	}
	if levelIconArt[bcnIcon] == nil || levelIconColors[bcnIcon] == 0 || seatThemes[bcnIcon].kits[kitCafe] == 0 {
		t.Error("BARCELONA needs its list icon, its swatch and its café tables")
	}
	// nothing of the ranch is left in the tables of the icon
	all := barcelonaTitle + " " + hintThemes[bcnIcon].goal + " " + hintThemes[bcnIcon].tip
	for _, l := range venueLines[bcnIcon] {
		all += " " + l.es + " " + l.en
	}
	for _, p := range pVenue[bcnIcon] {
		all += " " + p.es + " " + p.en
	}
	for _, w := range []string{"vaquero", "rancho", "heno", "Gaudí", "Gaudi", "Güell", "Colau", "Messi"} {
		if strings.Contains(all, w) {
			t.Errorf("the texts of BARCELONA say %q", w)
		}
	}
	for _, m := range []map[string]bool{toroIcons, painterIcons, flashIcons, pokerIcons} {
		if m[bcnIcon] {
			t.Error("a ranch-only event is still on in BARCELONA")
		}
	}
	if murrayLevels["BULL'S EYE"] || murrayLevels[barcelonaTitle] {
		t.Error("the skull on its stake belonged to the ranch")
	}
}
