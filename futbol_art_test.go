package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func fbSpec(t *testing.T, slug string) CharSpec {
	t.Helper()
	f := presetByName(slug)
	if f == nil {
		t.Fatalf("%s: no preset", slug)
	}
	return f(newRand(5))
}

func fbHas(b *pixBuf, c color.RGBA) int {
	n := 0
	for i := 0; i+3 < len(b.p); i += 4 {
		if b.p[i+3] != 0 && b.p[i] == c.R && b.p[i+1] == c.G && b.p[i+2] == c.B {
			n++
		}
	}
	return n
}

// The four footballers are famous guests with their real name, team, description (ES and EN), short
// translated catchphrases, a card in PERSONAJES (so the wardrobe lists them) and a fixed drawing.
func TestFootballersRegistered(t *testing.T) {
	cards := map[string]albumCard{}
	for _, c := range charCards() {
		cards[c.slug] = c
	}
	wardrobe := map[string]bool{}
	for _, o := range wardCharacters() {
		wardrobe[o.id] = true
	}
	for _, slug := range footballSlugs {
		m := footballRealNames[slug]
		if presetByName(slug) == nil {
			t.Fatalf("%s: no preset", slug)
		}
		if displayNameOverrides[slug] != m.name {
			t.Errorf("%s: display name %q, want %q", slug, displayNameOverrides[slug], m.name)
		}
		d, ok := charDescs[slug]
		if !ok || d[0] == "" || langDataEN[d[0]] != d[1] {
			t.Errorf("%s: description %v", slug, d)
		}
		if f := funnyDesc["skin:"+slug]; f == "" || langDataEN[f] == "" {
			t.Errorf("%s: shop line %q", slug, f)
		}
		if len(catchphrases[slug]) == 0 {
			t.Errorf("%s: no catchphrases", slug)
		}
		for _, l := range catchphrases[slug] {
			if len([]rune(l)) > 48 || langDataEN[l] == "" {
				t.Errorf("%s line %q: too long or not translated", slug, l)
			}
		}
		c, ok := cards[slug]
		if !ok || c.cardName() != m.name || c.roleCard != m.team || c.trophyID != "" {
			t.Errorf("%s: card %q / %q", slug, c.cardName(), c.roleCard)
		}
		if !wardrobe[slug] {
			t.Errorf("%s: not in the wardrobe list", slug)
		}
		if langDataEN[m.team] == "" {
			t.Errorf("%s: team %q has no English", slug, m.team)
		}
		b := builtinChar{fbSpec(t, slug), slug, "special_people", ""}
		if !charimpFixed(b) {
			t.Errorf("%s: the drawing depends on chance", slug)
		}
	}
	if gaitSlugs["cristiano_ronaldo"] != "spin" {
		t.Errorf("Cristiano: gait %q, want the jumping spin", gaitSlugs["cristiano_ronaldo"])
	}
}

// Adding them does not move the shared random stream of builtinChars: every character keeps the
// spec it had (compared with the list built without the four footballers).
func TestFootballersKeepOthersSpecs(t *testing.T) {
	all := builtinChars()
	saved := famousPresets
	t.Cleanup(func() { famousPresets = saved })
	famousPresets = nil
	for _, f := range saved {
		if fbKinds[f(newRand(1)).Name] != fbNone {
			continue
		}
		famousPresets = append(famousPresets, f)
	}
	base := builtinChars()
	if len(all) != len(base)+len(footballSlugs) {
		t.Fatalf("%d characters with them, %d without", len(all), len(base))
	}
	byName := map[string]CharSpec{}
	for _, b := range all {
		byName[b.cat+"/"+b.name] = b.spec
	}
	for _, b := range base {
		if fmt.Sprint(byName[b.cat+"/"+b.name]) != fmt.Sprint(b.spec) {
			t.Errorf("%s/%s: spec changed by the footballers", b.cat, b.name)
		}
	}
}

// They draw in the three views and the usual animations inside the sprite box, their layers show,
// every body part is there for the rig, and walking changes the frames.
func TestFootballersDrawInsideAndWalk(t *testing.T) {
	anims := []int{AnimIdle, AnimWalk, AnimRun, AnimStab, AnimDance3, AnimDance7, AnimPanic, AnimHop, AnimCheer}
	for _, slug := range footballSlugs {
		sp := fbSpec(t, slug)
		twin := sp
		twin.Name = "twin"
		for _, vw := range []int{vFront, vSide, vBack} {
			for _, a := range anims {
				for f := 0; f < animFrameCount(a); f++ {
					n, edge := charimpCount(renderChar(&sp, vw, ownPose(a, f)))
					tn, tEdge := charimpCount(renderChar(&twin, vw, ownPose(a, f)))
					if edge > tEdge {
						t.Errorf("%s view %d anim %d frame %d: %d pixels on the edge (without layers: %d)", slug, vw, a, f, edge, tEdge)
					}
					if n < 60 || n < tn*9/10 {
						t.Errorf("%s view %d anim %d frame %d: only %d pixels (without layers: %d)", slug, vw, a, f, n, tn)
					}
				}
			}
			if string(renderChar(&sp, vw, ownPose(AnimIdle, 0)).p) == string(renderChar(&twin, vw, ownPose(AnimIdle, 0)).p) {
				t.Errorf("%s view %d: the own layers are not drawn", slug, vw)
			}
			seen := map[string]bool{}
			for f := 0; f < animFrameCount(AnimWalk); f++ {
				seen[string(renderChar(&sp, vw, ownPose(AnimWalk, f)).p)] = true
			}
			if len(seen) < 3 {
				t.Errorf("%s view %d: only %d different walking frames", slug, vw, len(seen))
			}
			trackParts = true
			b := genChar(&sp, vw, poseFor(AnimIdle, 0))
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

// The kits read: the number is on the back only (7, 10, 19, 10 in the number colour), Messi's
// shirt has white stripes, the socks are painted, and the four look different.
func TestFootballersKits(t *testing.T) {
	idle := ownPose(AnimIdle, 0)
	frames := map[string]string{}
	for _, slug := range footballSlugs {
		sp := fbSpec(t, slug)
		k := fbKind(&sp)
		num, ncol, _, sock := fbKit(k)
		back := renderChar(&sp, vBack, idle)
		front := renderChar(&sp, vFront, idle)
		twin := sp
		twin.Name = "twin"
		inBack := fbHas(back, ncol) - fbHas(renderChar(&twin, vBack, idle), ncol)
		want := 0
		for i := 0; i < len(num); i++ {
			for _, row := range fbDigits[num[i]] {
				for _, ch := range row {
					if ch == '#' {
						want++
					}
				}
			}
		}
		if inBack < want*8/10 {
			t.Errorf("%s: number %s shows %d of %d pixels on the back", slug, num, inBack, want)
		}
		if fbHas(back, sock) == 0 || fbHas(front, sock) == 0 {
			t.Errorf("%s: socks not painted", slug)
		}
		if fbNumberW(num) > bodyWidth(sp.Body) {
			t.Errorf("%s: number %s (%d px) wider than the back (%d px)", slug, num, fbNumberW(num), bodyWidth(sp.Body))
		}
		frames[slug] = string(front.p)
		for other, f := range frames {
			if other != slug && f == frames[slug] {
				t.Errorf("%s and %s draw the same", slug, other)
			}
		}
	}
	m := fbSpec(t, "lionel_messi")
	if fbHas(renderChar(&m, vFront, idle), fbArgWht) < 6 {
		t.Error("Messi: no white stripes on the shirt")
	}
	cr, me := fbSpec(t, "cristiano_ronaldo"), fbSpec(t, "lionel_messi")
	if tiaTop(renderChar(&me, vFront, idle)) <= tiaTop(renderChar(&cr, vFront, idle)) {
		t.Error("Messi should be shorter than Cristiano")
	}
}

// Card and detail window in ES and EN: real name (no translation), team under it, "team · Footballer"
// in the window, and the longest name keeps >= 6 px of padding on the card (margins rule).
func TestFootballersCardsAndDetail(t *testing.T) {
	g := goalDetailGame(t)
	prev := uiLang
	t.Cleanup(func() { uiLang = prev })
	for _, lg := range []string{"es", "en"} {
		uiLang = lg
		for _, slug := range footballSlugs {
			m := footballRealNames[slug]
			if got := tr(m.name); got != m.name {
				t.Errorf("%s %s: tr(%q) = %q", lg, slug, m.name, got)
			}
			for _, s := range []string{m.name, tr(m.team)} {
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
	want := map[string][2]string{
		"cristiano_ronaldo": {"Portugal · Futbolista", "Portugal · Footballer"},
		"lionel_messi":      {"Argentina · Futbolista", "Argentina · Footballer"},
		"lamine_yamal":      {"España · Futbolista", "Spain · Footballer"},
		"kylian_mbappe":     {"Francia · Futbolista", "France · Footballer"},
	}
	found := 0
	for li, lg := range []string{"es", "en"} {
		uiLang = lg
		for i, c := range g.albumView() {
			w, ok := want[c.slug]
			if !ok {
				continue
			}
			found++
			g.albumSel = i
			ct := g.goalDetailContent()
			if ct.name != footballRealNames[c.slug].name || ct.sub != w[li] {
				t.Errorf("%s %s: window %q / %q, want %q", lg, c.slug, ct.name, ct.sub, w[li])
			}
			if len(ct.paras) == 0 || ct.paras[0].s != tr(charDesc(c.slug)) {
				t.Errorf("%s %s: window without his description", lg, c.slug)
			}
		}
	}
	if found != 2*len(footballSlugs) {
		t.Errorf("found %d of %d footballer cards in PERSONAJES", found, 2*len(footballSlugs))
	}
}

// FUTBOL_EXPORT=assets-root go test -run TestFootballersExport writes the pictures of the four from the
// code (same method as TestKtmExport: an existing meta.json is kept; a new folder gets the meta of
// exportMissingCharacters). Only these four folders are touched.
func TestFootballersExport(t *testing.T) {
	root := os.Getenv("FUTBOL_EXPORT")
	if root == "" {
		t.Skip("set FUTBOL_EXPORT to an assets folder")
	}
	want := map[string]bool{}
	for _, s := range footballSlugs {
		want[s] = true
	}
	n := 0
	for _, b := range builtinChars() {
		if !want[b.name] {
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
	if n != len(footballSlugs) {
		t.Errorf("exported %d of %d", n, len(footballSlugs))
	}
}

// The exported folders in assets/ are the pictures the code draws, meta.json has the real name, and
// the rig built from them walks in the three views (the game draws the famous from these files).
func TestFootballersExportedPictures(t *testing.T) {
	seen := 0
	for _, b := range builtinChars() {
		if fbKinds[b.name] == fbNone {
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
		if json.Unmarshal(raw, &meta) != nil || meta.DisplayName != footballRealNames[b.name].name || meta.Category != "special_people" || !meta.ShopOnly {
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
	if seen != len(footballSlugs) {
		t.Errorf("checked %d of %d", seen, len(footballSlugs))
	}
}

// FUTBOL_SHEET=dir go test -run TestFootballersSheet writes an enlarged sheet: idle, walk and the
// dance (front, side, back) of each footballer.
func TestFootballersSheet(t *testing.T) {
	dir := os.Getenv("FUTBOL_SHEET")
	if dir == "" {
		t.Skip("set FUTBOL_SHEET to a directory")
	}
	os.MkdirAll(dir, 0o755)
	z := 6
	poses := []Pose{ownPose(AnimIdle, 0), ownPose(AnimWalk, 1), ownPose(AnimHop, 1)}
	if os.Getenv("FUTBOL_ZOOM") != "" { // close-up: only the standing pose
		z, poses = 12, poses[:1]
	}
	cellW, cellH := 3*charW*z+12, charH*z+6
	big := image.NewRGBA(image.Rect(0, 0, len(poses)*cellW, len(footballSlugs)*cellH))
	for i := 0; i < len(big.Pix); i += 4 {
		big.Pix[i], big.Pix[i+1], big.Pix[i+2], big.Pix[i+3] = 0x6a, 0x8a, 0x5a, 255
	}
	for si, slug := range footballSlugs {
		sp := fbSpec(t, slug)
		for pi, p := range poses {
			for vi, vw := range []int{vFront, vSide, vBack} {
				b := renderChar(&sp, vw, p)
				for y := 0; y < b.h; y++ {
					for x := 0; x < b.w; x++ {
						i := (y*b.w + x) * 4
						if b.p[i+3] == 0 {
							continue
						}
						c := color.RGBA{b.p[i], b.p[i+1], b.p[i+2], 255}
						for dy := 0; dy < z; dy++ {
							for dx := 0; dx < z; dx++ {
								big.SetRGBA(pi*cellW+(vi*charW+x)*z+dx, si*cellH+y*z+dy, c)
							}
						}
					}
				}
			}
		}
	}
	f, err := os.Create(filepath.Join(dir, "futbol.png"))
	if err != nil {
		t.Fatal(err)
	}
	png.Encode(f, big)
	f.Close()
}
