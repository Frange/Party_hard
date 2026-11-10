package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// bsbSlugs: the members of Billy Talent, Lost Prophets (six), Despistaos and Punk Sailor drawn in
// bandsb_art.go.
var bsbSlugs = []string{
	"bt_cantante", "bt_guitarra", "bt_bajo", "bt_bateria",
	"lp_cantante", "lp_guitarra", "lp_bajo", "lp_bateria", "lp_ritmo", "lp_corista",
	"des_cantante", "des_guitarra", "des_bajo", "des_bateria",
	"ps_cantante", "ps_guitarra", "ps_bajo", "ps_bateria",
}

// BSB_SHEET=dir [BSB_TAG=x] go test -run TestBsbSheet writes an enlarged sheet: one row per band,
// each member in front, side, back and a walk frame.
func TestBsbSheet(t *testing.T) {
	dir := os.Getenv("BSB_SHEET")
	if dir == "" {
		t.Skip("set BSB_SHEET")
	}
	rows := [][]string{bsbSlugs[0:4], bsbSlugs[4:10], bsbSlugs[10:14], bsbSlugs[14:18]}
	z := 4
	if os.Getenv("BSB_BIG") != "" { // only one band, twice as big
		z = 8
		n := int(os.Getenv("BSB_BIG")[0] - '0')
		rows = rows[n : n+1]
	}
	memW := 4*charW*z + 12
	cellH := charH*z + 8
	most := 0
	for _, r := range rows {
		most = maxI(most, len(r))
	}
	big := image.NewRGBA(image.Rect(0, 0, most*memW, len(rows)*cellH))
	for i := 0; i < len(big.Pix); i += 4 {
		big.Pix[i], big.Pix[i+1], big.Pix[i+2], big.Pix[i+3] = 0x70, 0x70, 0x78, 255
	}
	for ri, row := range rows {
		for mi, slug := range row {
			sp := charimpASpec(t, slug)
			for vi, vw := range []int{vFront, vSide, vBack, vFront} {
				pp := poseFor(AnimIdle, 0)
				if vi == 3 {
					pp = poseFor(AnimWalk, 1)
					pp.Anim, pp.F = AnimWalk, 1
				}
				b := renderChar(&sp, vw, pp)
				for y := 0; y < b.h; y++ {
					for x := 0; x < b.w; x++ {
						i := (y*b.w + x) * 4
						if b.p[i+3] == 0 {
							continue
						}
						c := color.RGBA{b.p[i], b.p[i+1], b.p[i+2], 255}
						for dy := 0; dy < z; dy++ {
							for dx := 0; dx < z; dx++ {
								big.SetRGBA(mi*memW+(vi*charW+x)*z+dx, ri*cellH+y*z+dy, c)
							}
						}
					}
				}
			}
		}
	}
	f, err := os.Create(filepath.Join(dir, "bsb_"+os.Getenv("BSB_TAG")+".png"))
	if err != nil {
		t.Fatal(err)
	}
	png.Encode(f, big)
	f.Close()
}

// bsbSpecs: the members' specs by slug.
func bsbSpecs(t *testing.T) map[string]CharSpec {
	out := map[string]CharSpec{}
	for _, s := range bsbSlugs {
		out[s] = charimpASpec(t, s)
	}
	return out
}

// Every member draws in the three views inside its box, the details of bandsb_art.go really
// show (against the same body without the hooks) and walking changes the frames.
func TestBsbDrawsInsideAndWalks(t *testing.T) {
	anims := []int{AnimIdle, AnimWalk, AnimRun, AnimStab, AnimDance3, AnimPanic, AnimGuitar, AnimSing, AnimDrum, AnimHop}
	for _, slug := range bsbSlugs {
		sp := charimpASpec(t, slug)
		if bsbKind(&sp) == bsbNone {
			t.Fatalf("%s: no details registered", slug)
		}
		twin := sp
		twin.Name = "twin"
		for vw := 0; vw < 3; vw++ {
			differs := false
			for _, a := range anims {
				for f := 0; f < animFrameCount(a); f++ {
					pp := poseFor(a, f)
					if a >= AnimCount {
						pp = gaitPoseFor(a, f)
					}
					pp.Anim, pp.F = a, f
					b, tb := renderChar(&sp, vw, pp), renderChar(&twin, vw, pp)
					n, edge := charimpAPix(b)
					tn, tEdge := charimpAPix(tb)
					if edge > tEdge+2 {
						t.Errorf("%s vw %d anim %d f %d: %d pixels on the edge (plain %d)", slug, vw, a, f, edge, tEdge)
					}
					if n < 60 || n < tn*9/10 {
						t.Errorf("%s vw %d anim %d f %d: only %d pixels (plain %d)", slug, vw, a, f, n, tn)
					}
					if string(b.p) != string(tb.p) {
						differs = true
					}
				}
			}
			if !differs && !bsbPlainView[slug][vw] {
				t.Errorf("%s vw %d: the details never show", slug, vw)
			}
			w0 := charimpAWalkFrame(&sp, vw, 0)
			changed := false
			for f := 1; f < animFrameCount(AnimWalk); f++ {
				if string(charimpAWalkFrame(&sp, vw, f)) != string(w0) {
					changed = true
				}
			}
			if !changed {
				t.Errorf("%s vw %d: walking does not change the frames", slug, vw)
			}
		}
	}
}

// bsbPlainView: views where a member has nothing of bandsb_art.go (his look comes from the spec alone):
// Despistaos' singer has his dark beard on the face, the bassist his print on the chest; the guitarist of
// Despistaos and the drummer of Punk Sailor are drawn by their spec only.
var bsbPlainView = map[string][3]bool{
	"des_cantante": {vBack: true}, "des_bajo": {vBack: true, vSide: true},
	"des_guitarra": {true, true, true}, "ps_bateria": {true, true, true},
}

func bsbAt(b *pixBuf, x, y int) color.RGBA {
	if x < 0 || y < 0 || x >= b.w || y >= b.h {
		return color.RGBA{}
	}
	i := (y*b.w + x) * 4
	return color.RGBA{b.p[i], b.p[i+1], b.p[i+2], b.p[i+3]}
}

// bsbCount: pixels of the picture that pass the test, in rows y0..y1.
func bsbCount(b *pixBuf, y0, y1 int, ok func(c color.RGBA) bool) int {
	n := 0
	for y := y0; y <= y1; y++ {
		for x := 0; x < b.w; x++ {
			if c := bsbAt(b, x, y); c.A > 0 && ok(c) {
				n++
			}
		}
	}
	return n
}

// bsbRows: first and last row with any pixel.
func bsbRows(b *pixBuf) (int, int) {
	top, bot := -1, -1
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			if bsbAt(b, x, y).A > 0 {
				if top < 0 {
					top = y
				}
				bot = y
				break
			}
		}
	}
	return top, bot
}

// The key looks of the photos: shocking pink hair over a dark beard, black and white vertical
// stripes, braces on a white tee, a shaved head, a captain's cap with gold, a tall crest, a navy
// and white striped tee under the leather jacket, orange-brown jeans.
func TestBsbKeyLooks(t *testing.T) {
	sp := bsbSpecs(t)
	idle := poseFor(AnimIdle, 0)
	front := func(slug string) *pixBuf { s := sp[slug]; return renderChar(&s, vFront, idle) }
	plain := func(slug string) *pixBuf { s := sp[slug]; s.Name = "twin"; return renderChar(&s, vFront, idle) }
	pink := func(c color.RGBA) bool { return c.R > 220 && c.G < 110 && c.B > 140 }
	// pink hair, and a dark beard under it (the plain body would have a pink beard)
	d := front("des_cantante")
	top, _ := bsbRows(d)
	if n := bsbCount(d, top, top+4, pink); n < 15 {
		t.Errorf("des_cantante: %d shocking pink pixels on the head", n)
	}
	if n := bsbCount(d, top, top+8, func(c color.RGBA) bool { return c == bsbBeard }); n < 6 {
		t.Errorf("des_cantante: no dark beard (%d)", n)
	}
	if bsbCount(d, top+4, top+8, pink) >= bsbCount(plain("des_cantante"), top+4, top+8, pink) {
		t.Errorf("des_cantante: the beard is still pink")
	}
	for _, s := range bsbSlugs {
		if s != "des_cantante" && bsbCount(front(s), 0, charH-1, pink) > 0 {
			t.Errorf("%s: pink pixels, only the singer of Despistaos has pink hair", s)
		}
	}
	// vertical stripes: on the chest of ps_guitarra, columns alternate black and white in most rows
	g := front("ps_guitarra")
	rows := 0
	for y := 18; y < 28; y++ {
		swaps := 0
		for x := 1; x < g.w; x++ {
			a, b := bsbAt(g, x-1, y), bsbAt(g, x, y)
			if a.A > 0 && b.A > 0 && (a.R > 180) != (b.R > 180) && (a.R < 40 || b.R < 40) {
				swaps++
			}
		}
		if swaps >= 4 {
			rows++
		}
	}
	if rows < 4 {
		t.Errorf("ps_guitarra: vertical stripes in only %d rows", rows)
	}
	// the captain: gold on the cap above the face, a navy and white striped tee
	c := front("ps_cantante")
	top, _ = bsbRows(c)
	gold := func(c color.RGBA) bool { return c.R > 200 && c.G > 160 && c.B < 120 }
	if n := bsbCount(c, top, top+6, gold); n < 6 {
		t.Errorf("ps_cantante: %d gold pixels on the cap", n)
	}
	if sp["ps_cantante"].Hat != HatPolice || bsbCount(c, top, top+3, func(c color.RGBA) bool { return c.B > c.R && c.B < 90 }) < 6 {
		t.Errorf("ps_cantante: no dark peaked cap")
	}
	navy, white := bsbCount(c, 18, 28, func(c color.RGBA) bool { return c.B > 70 && c.R < 60 }), bsbCount(c, 18, 28, func(c color.RGBA) bool { return c.R > 200 && c.G > 200 && c.B > 200 })
	if navy < 6 || white < 6 {
		t.Errorf("ps_cantante: striped tee %d navy / %d white", navy, white)
	}
	// braces on a white tee
	bt := sp["bt_bateria"]
	if bt.Susp.A == 0 || bt.Susp.R > 40 || bt.Shirt.R < 230 || bt.Top != TopTee {
		t.Errorf("bt_bateria: no black braces on a white tee: %+v", bt)
	}
	bb := front("bt_bateria")
	dark := 0
	for y := 18; y < 26; y++ {
		for x := 0; x < bb.w; x++ {
			if c := bsbAt(bb, x, y); c.A > 0 && c.R < 30 && c.G < 30 && bsbAt(bb, x-1, y).R > 180 {
				dark++
			}
		}
	}
	if dark < 8 {
		t.Errorf("bt_bateria: the braces do not show (%d)", dark)
	}
	// shaved head: no hair colour at all, the scalp is skin (with stubble shade)
	sh := sp["bt_bajo"]
	if sh.Hair2 != HairBald {
		t.Errorf("bt_bajo: hair style %d, want shaved", sh.Hair2)
	}
	for vw := 0; vw < 3; vw++ {
		b := renderChar(&sh, vw, idle)
		top, _ := bsbRows(b)
		if n := bsbCount(b, top, top+4, func(c color.RGBA) bool { return c == sh.Hair }); n > 0 {
			t.Errorf("bt_bajo vw %d: %d hair pixels on the shaved head", vw, n)
		}
		if n := bsbCount(b, top, top+2, func(c color.RGBA) bool { return c == shade(sh.Skin, 0.78) }); n < 3 {
			t.Errorf("bt_bajo vw %d: no shaved stubble (%d)", vw, n)
		}
	}
	// the crest stands well above the plain head
	ct, _ := bsbRows(front("bt_guitarra"))
	pt, _ := bsbRows(plain("bt_guitarra"))
	if pt-ct < 4 {
		t.Errorf("bt_guitarra: the crest rises only %d rows", pt-ct)
	}
	// the fringe of the Billy Talent singer covers one eye: fewer skin pixels on one half of the face
	f, pf := front("bt_cantante"), plain("bt_cantante")
	skin := sp["bt_cantante"].Skin
	isSkin := func(c color.RGBA) bool { return c == skin }
	if bsbCount(f, 0, 14, isSkin) >= bsbCount(pf, 0, 14, isSkin) {
		t.Errorf("bt_cantante: the fringe hides nothing")
	}
	// Lost Prophets: worn orange-brown jeans for all six
	for _, s := range bsbSlugs[4:10] {
		p := sp[s].Pants
		if !(p.R > p.G+20 && p.G > p.B+10 && p.R > 0x70) {
			t.Errorf("%s: jeans %v are not orange-brown", s, p)
		}
		if sp[s].Shirt.R > 40 {
			t.Errorf("%s: not a black tee (%v)", s, sp[s].Shirt)
		}
	}
	// Billy Talent: all in black except the drummer's white tee
	for _, s := range []string{"bt_cantante", "bt_guitarra", "bt_bajo"} {
		if sp[s].Shirt.R > 60 {
			t.Errorf("%s: shirt %v is not black", s, sp[s].Shirt)
		}
	}
}

// No two members look the same in any view, and all of them always draw the same pictures (so the
// exported ones can follow the code: TestCharimpReexportedPictures checks them).
func TestBsbDistinctFixedAndListed(t *testing.T) {
	sp := bsbSpecs(t)
	for vw := 0; vw < 3; vw++ {
		seen := map[string]string{}
		for _, s := range bsbSlugs {
			x := sp[s]
			key := string(renderChar(&x, vw, poseFor(AnimIdle, 0)).p)
			if o, dup := seen[key]; dup {
				t.Errorf("vw %d: %s and %s look the same", vw, s, o)
			}
			seen[key] = s
		}
	}
	found := map[string]bool{}
	for _, b := range builtinChars() {
		for _, s := range bsbSlugs {
			if b.name == s {
				found[s] = true
				if b.cat != "special_people" || !charimpFixed(b) {
					t.Errorf("%s: cat %s fixed %v", s, b.cat, charimpFixed(b))
				}
			}
		}
	}
	renewed := map[string]bool{}
	for _, s := range charimpRenewed {
		renewed[s] = true
	}
	for _, s := range bsbSlugs {
		if !found[s] {
			t.Errorf("%s: not a built-in famous guest", s)
		}
		if !renewed[s] {
			t.Errorf("%s: not in charimpRenewed", s)
		}
	}
}

// Names, bands, cards and texts: the real name of the known musicians, else '<band> · <role>'; six Lost Prophets on the
// stage, one card each in OBJETIVOS (5 kills to win it), own descriptions in both languages.
func TestBsbBandsNamesAndTexts(t *testing.T) {
	want := map[string]string{"bt": "Billy Talent", "lp": "Lost Prophets", "des": "Despistaos", "ps": "Punk Sailor"}
	for _, s := range bsbSlugs {
		pre := s[:strings.Index(s, "_")]
		name := displayNameOverrides[s]
		if m, named := bandRealNames[s]; named { // a musician shown by his real name (bands_names_test.go)
			if name != m.name || bandOfMember(s) != want[pre] {
				t.Errorf("%s: display name %q band %q", s, name, bandOfMember(s))
			}
		} else if !strings.HasPrefix(name, want[pre]+" · ") {
			t.Errorf("%s: display name %q", s, name)
		}
		d, ok := charDescs[s]
		if !ok || d[0] == "" || d[1] == "" || len([]rune(d[0])) > 36 || len([]rune(d[1])) > 40 {
			t.Errorf("%s: description %q", s, d)
		}
		if tr := langDataEN[d[0]]; tr != d[1] {
			t.Errorf("%s: English of %q is %q", s, d[0], tr)
		}
	}
	descs := map[string]string{}
	for _, s := range bsbSlugs {
		if o, dup := descs[charDescs[s][0]]; dup {
			t.Errorf("%s and %s share the description", s, o)
		}
		descs[charDescs[s][0]] = s
	}
	for _, b := range bandList {
		if b.slug != "lostprophets" {
			continue
		}
		if len(b.members) != 6 {
			t.Fatalf("Lost Prophets: %d members on stage", len(b.members))
		}
		pos := map[[2]float64]bool{}
		for _, m := range b.members {
			if presetByName(m.slug) == nil || !strings.HasPrefix(m.slug, "lp_") {
				t.Errorf("Lost Prophets member %q", m.slug)
			}
			if pos[[2]float64{m.dx, m.dy}] || m.dx < -60 || m.dx > 60 {
				t.Errorf("%s: spot %v,%v", m.slug, m.dx, m.dy)
			}
			pos[[2]float64{m.dx, m.dy}] = true
		}
	}
	cards := map[string]albumCard{}
	for _, c := range charCards() {
		cards[c.slug] = c
	}
	for _, s := range bsbSlugs {
		c, ok := cards[s]
		if !ok {
			t.Errorf("%s: no card in OBJETIVOS", s)
			continue
		}
		if m, named := bandRealNames[s]; named {
			if c.roleCard != want[s[:strings.Index(s, "_")]] || c.cardName() != m.name {
				t.Errorf("%s: card %q / %q", s, c.cardName(), c.roleCard)
			}
		} else if c.roleCard == "" || c.cardES == "" || langDataEN[c.cardES] == "" {
			t.Errorf("%s: card %q / %q (EN %q)", s, c.cardES, c.roleCard, langDataEN[c.cardES])
		}
	}
	if charKillsNeed != 5 {
		t.Errorf("charKillsNeed = %d", charKillsNeed)
	}
	for _, s := range []string{"bt_cantante", "des_cantante"} {
		if got, _ := gaitAnim(gaitSlugs[s], AnimDance2, 0.3); got != AnimHop {
			t.Errorf("%s: dances %d, want the pogo", s, got)
		}
	}
}

// BSB_EXPORT=assets-root go test -run TestBsbExport writes, from the code, the pictures of the
// members of these four bands only (meta.json is kept; the two new ones get a new folder).
func TestBsbExport(t *testing.T) {
	root := os.Getenv("BSB_EXPORT")
	if root == "" {
		t.Skip("set BSB_EXPORT to an assets folder")
	}
	mine := map[string]bool{}
	for _, s := range bsbSlugs {
		mine[s] = true
	}
	n := 0
	for _, b := range builtinChars() {
		if b.cat != "special_people" || !mine[b.name] {
			continue
		}
		ca := charimpFresh(b)
		old, err := loadCharAsset(filepath.Join(spritesDir(root, b.cat), b.name))
		if err == nil {
			if old.Meta.Edited {
				t.Logf("%s: edited, left alone", b.name)
				continue
			}
			if charimpSame(old, ca) {
				continue
			}
			carry := ca.Meta.Carry
			ca.Meta = old.Meta
			ca.Meta.Carry = carry
		} else { // a new member: the same flags as the other band folders
			ca.Meta.DisplayName, ca.Meta.Gender, ca.Meta.ShopOnly = displayNameOverrides[b.name], "m", true
		}
		if err := saveCharAsset(root, ca); err != nil {
			t.Fatal(err)
		}
		n++
	}
	t.Logf("written: %d", n)
}
