package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fgSpec(t *testing.T, slug string) CharSpec {
	t.Helper()
	f := presetByName(slug)
	if f == nil {
		t.Fatalf("%s: no preset", slug)
	}
	return f(newRand(5))
}

// The nine beans are famous guests with a name and a description in ES and EN, short translated
// catchphrases, a card in PERSONAJES with "Flan Guys" under the name, the wardrobe lists them, the
// drawing does not depend on chance and the slugs collide with nobody.
func TestFallGuysRegistered(t *testing.T) {
	if len(beanSlugs) != 9 || len(beanKinds) != 9 || len(beanNames) != 9 {
		t.Fatalf("%d slugs, %d kinds, %d names", len(beanSlugs), len(beanKinds), len(beanNames))
	}
	cards := map[string]albumCard{}
	for _, c := range charCards() {
		cards[c.slug] = c
	}
	wardrobe := map[string]bool{}
	for _, o := range wardCharacters() {
		wardrobe[o.id] = true
	}
	presets := map[string]int{}
	probe := rand.New(rand.NewSource(1))
	for _, f := range famousPresets {
		presets[f(probe).Name]++
	}
	esNames, enNames := map[string]bool{}, map[string]bool{}
	for _, slug := range beanSlugs {
		if presets[slug] != 1 {
			t.Errorf("%s: %d presets with this slug", slug, presets[slug])
		}
		n := beanNames[slug]
		if esNames[n[0]] || enNames[n[1]] {
			t.Errorf("%s: name repeated", slug)
		}
		esNames[n[0]], enNames[n[1]] = true, true
		if n[0] != n[1] || !strings.HasPrefix(n[0], "Flan ") { // a proper name: never translated (owner, 2026-11-11)
			t.Errorf("%s: %q / %q must be the same Flan name in both languages", slug, n[0], n[1])
		}
		if displayNameOverrides[slug] != n[0] || langDataEN[n[0]] != n[1] {
			t.Errorf("%s: name %q / %q", slug, displayNameOverrides[slug], langDataEN[n[0]])
		}
		d, ok := charDescs[slug]
		if !ok || d[0] == "" || d[1] == "" || langDataEN[d[0]] != d[1] {
			t.Errorf("%s: description %v", slug, d)
		}
		if f := funnyDesc["skin:"+slug]; f == "" || langDataEN[f] == "" {
			t.Errorf("%s: shop line %q", slug, f)
		}
		if len(catchphrases[slug]) < 2 {
			t.Errorf("%s: %d catchphrases", slug, len(catchphrases[slug]))
		}
		for _, l := range catchphrases[slug] {
			if len([]rune(l)) > 48 || len([]rune(langDataEN[l])) > 48 || langDataEN[l] == "" {
				t.Errorf("%s line %q: too long or not translated", slug, l)
			}
		}
		c, ok := cards[slug]
		if !ok || c.nameES != n[0] || c.roleCard != beanGame || c.trophyID != "" {
			t.Errorf("%s: card %q / %q", slug, c.nameES, c.roleCard)
		}
		if !wardrobe[slug] {
			t.Errorf("%s: not in the wardrobe list", slug)
		}
		if gaitOf(slug, true, 0) != "bean" {
			t.Errorf("%s: gait %q", slug, gaitOf(slug, true, 0))
		}
		if !(&Game{}).isFamous(slug) {
			t.Errorf("%s: not one-per-party famous", slug)
		}
		inPool := false
		for _, s := range specialPool {
			inPool = inPool || s == slug
		}
		if !inPool {
			t.Errorf("%s: not in the special pool", slug)
		}
		b := builtinChar{fgSpec(t, slug), slug, "special_people", ""}
		if !charimpFixed(b) {
			t.Errorf("%s: the drawing depends on chance", slug)
		}
	}
	if beanGame != "Flan Guys" || langDataEN[beanGame] != beanGame {
		t.Errorf("%q has no English", beanGame)
	}
	if beanRole("homer") != "" || beanRole("judia_gallina") != beanGame {
		t.Error("beanRole")
	}
}

// Adding them does not move the shared random stream of builtinChars: every other character keeps
// the spec it had (compared with the list built without the beans).
func TestFallGuysKeepOthersSpecs(t *testing.T) {
	all := builtinChars()
	saved := famousPresets
	t.Cleanup(func() { famousPresets = saved })
	famousPresets = nil
	for _, f := range saved {
		if beanKinds[f(newRand(1)).Name] != fgNone {
			continue
		}
		famousPresets = append(famousPresets, f)
	}
	base := builtinChars()
	if len(all) != len(base)+len(beanSlugs) {
		t.Fatalf("%d characters with them, %d without", len(all), len(base))
	}
	byName := map[string]CharSpec{}
	for _, b := range all {
		byName[b.cat+"/"+b.name] = b.spec
	}
	for _, b := range base {
		if fmt.Sprint(byName[b.cat+"/"+b.name]) != fmt.Sprint(b.spec) {
			t.Errorf("%s/%s: spec changed by the beans", b.cat, b.name)
		}
	}
}

// Every view and animation (the gaits included) draws inside the sprite box (1 px margin, held
// objects apart), the body parts for the rig are all tagged, and walking, running and dancing change
// the frames in the three views.
func TestFallGuysDrawInBoxAndAnimate(t *testing.T) {
	for _, slug := range beanSlugs {
		sp := fgSpec(t, slug)
		if !isBean(&sp) {
			t.Fatalf("%s: not drawn as a bean", slug)
		}
		for vw := 0; vw < 3; vw++ {
			for a := 0; a < animGaitEnd; a++ {
				for f := 0; f < animFrameCount(a); f++ {
					pp := ownPose(a, f)
					pp.Hold = HoldNone
					minX, minY, maxX, maxY := bbox(genChar(&sp, vw, pp))
					if maxX < 0 {
						t.Fatalf("%s: empty frame vw=%d anim=%d f=%d", slug, vw, a, f)
					}
					if minX < 1 || minY < 1 || maxX > charW-2 || maxY > charH-1 {
						t.Errorf("%s vw=%d anim=%d f=%d outside the box: %d,%d-%d,%d", slug, vw, a, f, minX, minY, maxX, maxY)
					}
				}
			}
			for _, a := range []int{AnimWalk, AnimRun, AnimLumber, AnimHop, AnimDance0} {
				seen := map[string]bool{}
				for f := 0; f < animFrameCount(a); f++ {
					seen[string(genChar(&sp, vw, ownPose(a, f)).p)] = true
				}
				if len(seen) < 2 {
					t.Errorf("%s vw %d anim %d: the frames do not change", slug, vw, a)
				}
			}
			trackParts = true
			b := genChar(&sp, vw, ownPose(AnimIdle, 0))
			trackParts = false
			parts := map[uint8]bool{}
			for i, id := range b.pid {
				if b.p[i*4+3] != 0 {
					parts[id] = true
				}
			}
			for id := uint8(PartHead); id <= PartLegR; id++ {
				if !parts[id] && !(vw == vSide && (id == PartArmL || id == PartLegL)) {
					t.Errorf("%s vw %d: part %d missing", slug, vw, id)
				}
			}
		}
	}
}

// The look: a bean is wider than tall-ish people (12 px body plus the flippers), as tall as a short
// guest, with the white face and two black eyes in front and in profile but not from behind, and
// the nine costumes all look different from each other in the three views.
func TestFallGuysLook(t *testing.T) {
	idle := ownPose(AnimIdle, 0)
	has := func(b *pixBuf, c color.RGBA) int {
		n := 0
		for y := 0; y < b.h; y++ {
			for x := 0; x < b.w; x++ {
				if b.get(x, y) == c {
					n++
				}
			}
		}
		return n
	}
	frames := map[string]string{}
	for _, slug := range beanSlugs {
		sp := fgSpec(t, slug)
		front, side, back := genChar(&sp, vFront, idle), genChar(&sp, vSide, idle), genChar(&sp, vBack, idle)
		if has(front, fgWhite) < 30 || has(front, fgEye) < 4 {
			t.Errorf("%s: no face in front (%d white, %d eye)", slug, has(front, fgWhite), has(front, fgEye))
		}
		if has(side, fgWhite) < 10 || has(side, fgEye) < 2 {
			t.Errorf("%s: no face in profile", slug)
		}
		if has(back, fgEye) != 0 {
			t.Errorf("%s: eyes on the back", slug)
		}
		minX, minY, maxX, maxY := bbox(front)
		if w, h := maxX-minX+1, maxY-minY+1; w < 16 || h < 24 || h > 32 {
			t.Errorf("%s: front %dx%d", slug, w, h)
		}
		for vw, b := range []*pixBuf{front, side, back} {
			key := fmt.Sprint(vw, string(b.p))
			if other, dup := frames[key]; dup {
				t.Errorf("%s and %s draw the same in view %d", slug, other, vw)
			}
			frames[key] = slug
		}
	}
}

// Card and detail window in ES and EN: the bean's name (the same in both), "Flan Guys" under it and in the
// window, his description, and the longest name keeps >= 6 px of padding on the card (margins rule).
func TestFallGuysCardsAndDetail(t *testing.T) {
	g := goalDetailGame(t)
	prev := uiLang
	t.Cleanup(func() { uiLang = prev })
	for _, lg := range []string{"es", "en"} {
		uiLang = lg
		for _, slug := range beanSlugs {
			for _, s := range []string{tr(beanNames[slug][0]), tr(beanGame)} {
				if w := g.textW(s, 1); w > albumCardW-12 {
					t.Errorf("%s %s: %q is %.0f px wide on a %.0f px card", lg, slug, s, w, albumCardW)
				}
			}
		}
	}
	g.progress.Album, g.progress.CharKills = map[string]int{}, map[string]int{}
	for _, c := range albumCards() {
		g.progress.Album[c.key] = albumKilled
	}
	g.goTab(tabChars)
	found := 0
	for li, lg := range []string{"es", "en"} {
		uiLang = lg
		for i, c := range g.albumView() {
			n, ok := beanNames[c.slug]
			if !ok {
				continue
			}
			found++
			if c.cardName() != n[0] || tr(c.cardName()) != n[li] {
				t.Errorf("%s %s: card name %q", lg, c.slug, tr(c.cardName()))
			}
			g.albumSel = i
			ct := g.goalDetailContent()
			if ct.name != n[li] || ct.sub != beanGame {
				t.Errorf("%s %s: window %q / %q", lg, c.slug, ct.name, ct.sub)
			}
			if len(ct.paras) == 0 || ct.paras[0].s != tr(charDesc(c.slug)) || ct.paras[0].s == "" {
				t.Errorf("%s %s: window without his description", lg, c.slug)
			}
		}
	}
	if found != 2*len(beanSlugs) {
		t.Errorf("found %d of %d bean cards in PERSONAJES", found, 2*len(beanSlugs))
	}
}

// The gang: 2-4 different beans arrive together (the first leads), never a bean already at the party,
// and spawnStars takes them from the famous places; a whole party with the gang keeps every famous
// face once and no more people than without it.
func TestFallGuysGang(t *testing.T) {
	g := partyGame(t)
	prevForce, prevRng := fgForce, fgRng
	t.Cleanup(func() { fgForce, fgRng = prevForce, prevRng })
	fgForce = true
	for seed := int64(1); seed <= 30; seed++ {
		fgRng = rand.New(rand.NewSource(seed))
		g.guests = nil
		if seed%3 == 0 { // one already at the party never comes twice
			spec := fgSpec(t, "judia_azul")
			g.guests = append(g.guests, &Guest{cs: newCharSet(&spec), keep: true})
		}
		before := len(g.guests)
		n := g.spawnBeanGang()
		if n < 2 || n > 4 || len(g.guests)-before != n {
			t.Fatalf("seed %d: gang of %d (%d guests added)", seed, n, len(g.guests)-before)
		}
		seen := map[string]int{}
		for i, gu := range g.guests {
			seen[gu.cs.Spec.Name]++
			if i > before && gu.leader != g.guests[before] {
				t.Errorf("seed %d: guest %d does not follow the first of the gang", seed, i)
			}
		}
		for s, c := range seen {
			if c > 1 {
				t.Errorf("seed %d: %s %d times", seed, s, c)
			}
		}
	}
	// without forcing it, only some parties get a gang
	fgForce = false
	fgRng = rand.New(rand.NewSource(9))
	got := 0
	for i := 0; i < 400; i++ {
		g.guests = nil
		if g.spawnBeanGang() > 0 {
			got++
		}
	}
	if got < 40 || got > 130 {
		t.Errorf("a gang in %d of 400 parties, want about 1 in %d", got, beanGangChance)
	}
	// a whole party with the gang: every famous face once, the head count within the settings
	fgForce = true
	for _, lvl := range []int{0, 3} {
		fgRng = rand.New(rand.NewSource(int64(40 + lvl)))
		g.loadLevel(lvl)
		count := map[string]int{}
		beans := 0
		for _, gu := range g.guests {
			if gu.cs == nil || gu.cs.Spec == nil {
				continue
			}
			n := gu.cs.Spec.Name
			if g.isFamous(n) && gu.perf == "" && !gu.famousOK {
				count[n]++
			}
			if beanKinds[n] != fgNone {
				beans++
			}
		}
		for n, c := range count {
			if c > 1 {
				t.Errorf("level %d: %s %d times", lvl, n, c)
			}
		}
		// the gang comes inside the cap: only the cap of cap.go may trim one of them
		if beans < 2 && g.guestLive() < g.guestCap() {
			t.Errorf("level %d: %d beans with the gang forced", lvl, beans)
		}
		if g.guestLive() > g.guestCap() {
			t.Errorf("level %d: %d guests over the cap %d", lvl, g.guestLive(), g.guestCap())
		}
	}
}

// BEAN_EXPORT=assets-root go test -run TestFallGuysExport writes the pictures of the nine from the
// code (same method as TestFootballersExport: an existing meta.json is kept; a new folder gets the
// display name and shop_only, so they only come through the special pool and the gang).
func TestFallGuysExport(t *testing.T) {
	root := os.Getenv("BEAN_EXPORT")
	if root == "" {
		t.Skip("set BEAN_EXPORT to an assets folder")
	}
	n := 0
	for _, b := range builtinChars() {
		if beanKinds[b.name] == fgNone {
			continue
		}
		ca := charimpFresh(b)
		if old, err := loadCharAsset(filepath.Join(spritesDir(root, b.cat), b.name)); err == nil {
			if old.Meta.Edited {
				t.Logf("%s: edited by hand, skipped", b.name)
				continue
			}
			carry := ca.Meta.Carry
			ca.Meta = old.Meta
			ca.Meta.Carry = carry
		} else {
			spec := b.spec
			ca.Meta.ShopOnly = true
			ca.Meta.DisplayName = nameFromSpec(&spec)
			ca.Meta.Gender = "m"
		}
		if err := saveCharAsset(root, ca); err != nil {
			t.Fatal(err)
		}
		n++
	}
	if n != len(beanSlugs) {
		t.Errorf("exported %d of %d", n, len(beanSlugs))
	}
}

// The exported folders in assets/ are the pictures the code draws, meta.json has the name, and the
// rig built from them walks in the three views (the game draws the famous from these files).
func TestFallGuysExportedPictures(t *testing.T) {
	seen := 0
	for _, b := range builtinChars() {
		if beanKinds[b.name] == fgNone {
			continue
		}
		dir := filepath.Join(spritesDir("assets", b.cat), b.name)
		ca, err := loadCharAsset(dir)
		if err != nil {
			t.Fatalf("%s: %v", b.name, err)
		}
		seen++
		if !charimpSame(ca, charimpFresh(b)) {
			t.Errorf("%s: the exported pictures are not the ones the code draws", b.name)
		}
		raw, _ := os.ReadFile(filepath.Join(dir, "meta.json"))
		var meta struct {
			Name, Category string
			DisplayName    string `json:"display_name"`
			ShopOnly       bool   `json:"shop_only"`
		}
		if json.Unmarshal(raw, &meta) != nil || meta.Name != b.name || meta.DisplayName != beanNames[b.name][0] || meta.Category != "special_people" || !meta.ShopOnly {
			t.Errorf("%s: meta.json %s", b.name, raw)
		}
		rig := buildRig(ca)
		for vw := 0; vw < 3; vw++ {
			f0 := rig.render(vw, poseFor(AnimWalk, 0))
			if n, _ := charimpAPix(f0); n < 60 {
				t.Errorf("%s rig vw %d: only %d pixels", b.name, vw, n)
			}
			if string(rig.render(vw, poseFor(AnimWalk, 1)).p) == string(f0.p) {
				t.Errorf("%s rig vw %d: walking does not change the frames", b.name, vw)
			}
		}
	}
	if seen != len(beanSlugs) {
		t.Errorf("checked %d of %d", seen, len(beanSlugs))
	}
}

// BEAN_SHEET=dir go test -run TestFallGuysSheet writes an enlarged sheet: idle, walk, dance and
// lying (front, side, back) of each bean, from the code and from the rig of the exported files.
func TestFallGuysSheet(t *testing.T) {
	dir := os.Getenv("BEAN_SHEET")
	if dir == "" {
		t.Skip("set BEAN_SHEET to a directory")
	}
	os.MkdirAll(dir, 0o755)
	z := 5
	poses := []Pose{ownPose(AnimIdle, 0), ownPose(AnimLumber, 1), ownPose(AnimHop, 1), ownPose(AnimCheer, 0)}
	cellW, cellH := 3*charW*z+12, charH*z+6
	big := image.NewRGBA(image.Rect(0, 0, (len(poses)+1)*cellW, len(beanSlugs)*cellH))
	for i := 0; i < len(big.Pix); i += 4 {
		big.Pix[i], big.Pix[i+1], big.Pix[i+2], big.Pix[i+3] = 0x6a, 0xb4, 0xe8, 255
	}
	blit := func(b *pixBuf, ox, oy int) {
		for y := 0; y < b.h; y++ {
			for x := 0; x < b.w; x++ {
				i := (y*b.w + x) * 4
				if b.p[i+3] == 0 {
					continue
				}
				c := color.RGBA{b.p[i], b.p[i+1], b.p[i+2], 255}
				for dy := 0; dy < z; dy++ {
					for dx := 0; dx < z; dx++ {
						big.SetRGBA(ox+x*z+dx, oy+y*z+dy, c)
					}
				}
			}
		}
	}
	for si, slug := range beanSlugs {
		sp := fgSpec(t, slug)
		for pi, p := range poses {
			for vi, vw := range []int{vFront, vSide, vBack} {
				blit(renderChar(&sp, vw, p), pi*cellW+vi*charW*z, si*cellH)
			}
		}
		cs := newCharSet(&sp)
		if cs.rig != nil { // in-game look (from the exported files): lying dead and walking
			blit(cs.rigFrame(AnimLying, vFront, 0), len(poses)*cellW, si*cellH)
			blit(cs.rigFrame(AnimWalk, vSide, 1), len(poses)*cellW+2*charW*z, si*cellH)
		}
	}
	f, err := os.Create(filepath.Join(dir, "fallguys.png"))
	if err != nil {
		t.Fatal(err)
	}
	png.Encode(f, big)
	f.Close()
}
