package main

import (
	"image/color"
	"math"
	"strings"
)

// ---------------------------------------------------------------------------
// TREES AND BUSHES (v1.4.0, "árboles y arbustos distintos"): the "tree" and the "bush" props get more kinds, drawn
// as variants of the same prop so that everything that knows a tree or a bush (the footprint and the solids, the
// see-through fade when somebody walks behind, the wind, the fire, hiding in a bush, the editor) keeps working:
//
//	tree 0 classic · 1 round oak in lumps with lights · 2 big shaded oak · 3 green fir in tiers · 4 tall dark fir ·
//	     5 dead fir · 6 cherry in blossom · 7 twisted pine (bonsai style)
//	bush 0 classic · 1 round light shrub · 2 flowering shrub · 3 dark berry bush · 4 dry shrub · 5 pink blossom shrub
//
// Every variant stands on the same foot as variant 0 (trunk centred at the anchor, same base): the footprint is
// fitted on variant 0 (stuck.go) and does not change, so no collision changes.
//
// A tree or a bush placed with variant 0 (every level places them like that) gets its kind from the THEME of the
// level and its place (treeVarFor, called by drawProp): firs and dead trees in the graveyard, firs in the snow,
// cherries and twisted pines in Japanese places, oaks and the odd cherry in the parks... An explicit variant (>0)
// placed in the editor is kept.
// ---------------------------------------------------------------------------

const (
	treeVars = 8
	bushVars = 6
)

// lumpPal: dark (shadow), mid, light, highlight.
type lumpPal [4]color.RGBA

// lump paints one rounded clump of leaves lit from the top left.
func lump(b *pixBuf, cx, cy, rx, ry float64, p lumpPal, seed int) {
	b.ellipse(cx, cy+1, rx, ry, p[0])
	b.ellipse(cx-0.5, cy-0.5, rx-1, ry-1.2, p[1])
	b.ellipse(cx-rx*0.28, cy-ry*0.32, rx*0.55, ry*0.45, p[2])
	for i := 0; i < int(rx*ry/10)+1; i++ { // sparkles of light on the lit side
		h := fireHash(seed, i, 91)
		x := cx - rx*0.6 + float64(h%int(rx+1))
		y := cy - ry*0.7 + float64((h/97)%int(ry*0.9+1))
		b.set(int(x), int(y), p[3])
	}
	for i := 0; i < int(rx)/2+1; i++ { // dark holes in the shadow side
		h := fireHash(seed, i, 37)
		b.set(int(cx+float64(h%int(rx+1))-rx*0.2), int(cy+ry*0.35+float64((h/89)%3)), p[0])
	}
}

// shadedCrown paints a big crown as one mass lit from the top right, with the tones dithered (the shaded oak, the
// cherry): every pixel of the ellipse gets a tone by its light plus a lumpy noise.
func shadedCrown(b *pixBuf, cx, cy, rx, ry float64, tones []color.RGBA, seed int) {
	n := len(tones)
	for y := int(cy - ry - 2); y <= int(cy+ry+2); y++ {
		for x := int(cx - rx - 2); x <= int(cx+rx+2); x++ {
			dx, dy := (float64(x)-cx)/rx, (float64(y)-cy)/ry
			bump := 0.13*math.Sin(float64(x)*0.7+float64(seed)) + 0.13*math.Cos(float64(y)*0.8+float64(x)*0.25) // lumpy outline
			if dx*dx+dy*dy > 1+bump {
				continue
			}
			l := 0.55 + 0.4*dx*0.6 - 0.55*dy                      // light from the top right
			l += 0.22 * math.Sin(float64(x)*0.55+float64(y)*0.35) // clumps
			l += 0.18 * math.Cos(float64(x)*0.3-float64(y)*0.6+float64(seed))
			l += float64(fireHash(x, y, seed)%100)/100*0.25 - 0.12 // dithering
			i := int(l * float64(n))
			if i < 0 {
				i = 0
			}
			if i >= n {
				i = n - 1
			}
			b.set(x, y, tones[i])
		}
	}
}

// trunk paints a straight trunk with light and dark sides and the roots spreading at the foot.
func trunk(b *pixBuf, x, y0, w, y1 int, mid, light, dark color.RGBA) {
	b.rect(x, y0, w, y1-y0, mid)
	b.rect(x, y0, 2, y1-y0, light)
	b.rect(x+w-2, y0, 2, y1-y0, dark)
	b.rect(x-3, y1-3, w+6, 2, mid)
	b.rect(x-5, y1-1, w+10, 2, dark)
	b.rect(x-3, y1-3, 3, 1, light)
}

// thickLine: a line w pixels thick (branches).
func treeLine(b *pixBuf, x0, y0, x1, y1, w int, col color.RGBA) {
	for i := 0; i < w; i++ {
		b.line(x0+i, y0, x1+i, y1, col)
	}
}

// grassTuft: a few blades of grass around the foot.
func grassTuft(b *pixBuf, cx, y, w int, a, d color.RGBA) {
	for x := cx - w/2; x <= cx+w/2; x++ {
		h := fireHash(x, y, 5) % 4
		col := a
		if h%2 == 0 {
			col = d
		}
		b.rect(x, y-h, 1, h+1, col)
	}
}

// drawTreeVar draws tree variant v (1..treeVars-1) in the 56x72 box of the tree (foot centred at x=28, y=71).
func drawTreeVar(b *pixBuf, v int) {
	brown, brownL, brownD := c(0x8a5030), c(0xb07040), c(0x5a3020)
	switch v {
	case 1: // round oak in lumps, light green, with its branches showing between the lumps
		trunk(b, 25, 42, 7, 71, brown, brownL, brownD)
		treeLine(b, 27, 52, 15, 40, 2, brown)
		treeLine(b, 29, 52, 40, 40, 2, brownD)
		g := lumpPal{c(0x1e6a2a), c(0x34a040), c(0x62c850), c(0xb0e878)}
		for i, l := range [][4]float64{{28, 12, 11, 9}, {14, 18, 10, 8}, {42, 18, 10, 8}, {8, 30, 8, 8}, {48, 30, 8, 8},
			{28, 27, 12, 10}, {14, 40, 9, 7}, {42, 40, 9, 7}} {
			lump(b, l[0], l[1], l[2], l[3], g, i*13+1)
		}
	case 2: // big oak shaded as one mass, lit from the top right, thick trunk
		trunk(b, 24, 40, 9, 71, c(0x8a6040), c(0xb08a5a), c(0x4a3020))
		treeLine(b, 26, 50, 14, 38, 3, c(0x7a5434))
		treeLine(b, 30, 48, 42, 36, 3, c(0x6a4428))
		b.rect(24, 44, 9, 3, c(0x3a2414)) // the shadow of the crown on the trunk
		shadedCrown(b, 28, 25, 26, 21, []color.RGBA{c(0x1a3a14), c(0x2c5a1c), c(0x427a24), c(0x62982c), c(0x94c03a), c(0xd0e868)}, 3)
	case 3: // green fir in three drooping tiers
		trunk(b, 26, 46, 5, 71, c(0x7a4a24), c(0xa06a34), c(0x4a2a14))
		firTiers(b, []firTier{{3, 17, 9}, {15, 19, 15}, {29, 22, 22}}, lumpPal{c(0x1a4a34), c(0x2e8a5a), c(0x5ac08a), c(0x9ae0b4)}, false)
		grassTuft(b, 28, 71, 16, c(0x5ab040), c(0x2e7a2a))
	case 4: // tall dark fir, many tiers with drooping tips
		trunk(b, 26, 54, 5, 71, c(0x8a4a3a), c(0xa86a52), c(0x5a2a20))
		firTiers(b, []firTier{{2, 9, 5}, {8, 10, 8}, {15, 11, 11}, {23, 12, 14}, {32, 12, 17}, {41, 14, 20}},
			lumpPal{c(0x12302c), c(0x1e5a50), c(0x2e7a6a), c(0x58a48e)}, true)
	case 5: // dead fir: a bare crooked trunk with broken branches
		for y := 6; y <= 71; y++ {
			t := float64(y-6) / 65
			w := int(1 + t*4.5)
			x := 28 - w/2 + int(math.Round(math.Sin(t*5)*1.5))
			b.rect(x, y, w, 1, brown)
			b.set(x, y, brownL)
			if w > 2 {
				b.set(x+w-1, y, brownD)
			}
		}
		for i, y := 0, 10; y < 58; i, y = i+1, y+4 {
			side := 1 - 2*(i%2)
			l := 3 + (y-10)/4
			x0 := 28 + int(math.Round(math.Sin(float64(y-6)/65*5)*1.5))
			x1, y1 := x0+side*l, y-2+fireHash(i, y, 3)%3
			b.line(x0, y, x1, y1, c(0x5a3420))
			if l > 7 { // the low branches are thicker
				b.line(x0, y+1, x1-side, y1+1, c(0x4a2a18))
			}
			b.set(x1-side, y1-1, c(0x4a2a18)) // a twig
			if l > 6 {
				b.line(x0+side*l/2, y-1, x0+side*(l/2+2), y-4, c(0x4a2a18))
			}
		}
		grassTuft(b, 28, 71, 14, c(0x8a8a40), c(0x4a6a2a))
	case 6: // cherry in blossom: a pink crown, purple in the shade, and fallen petals
		trunk(b, 25, 42, 7, 71, c(0x6a3a22), c(0x8a5432), c(0x3e2014))
		treeLine(b, 29, 54, 40, 42, 2, c(0x5a3020))
		shadedCrown(b, 28, 24, 26, 18, []color.RGBA{c(0x6a2c58), c(0x9a4878), c(0xc86a8e), c(0xe8909e), c(0xf6b4b8), c(0xfcd8d0)}, 7)
		for i := 0; i < 26; i++ { // petals on the ground and a few in the air
			h := fireHash(i, 6, 66)
			x, y := 12+h%33, 67+(h/37)%5
			if i > 20 {
				x, y = 8+h%40, 46+(h/41)%18
			}
			b.set(x, y, []color.RGBA{c(0xe8909e), c(0xf6b4b8), c(0xc86a8e)}[i%3])
		}
	case 7: // twisted pine: a thick crooked trunk and flat clouds of needles (bonsai style)
		pts := [][2]int{{28, 71}, {33, 62}, {24, 52}, {31, 42}, {26, 31}, {29, 20}}
		for i := 0; i+1 < len(pts); i++ {
			w := 8 - i
			treeLine(b, pts[i][0]-w/2, pts[i][1], pts[i+1][0]-w/2, pts[i+1][1], w, c(0x7a3a22))
			b.line(pts[i][0]-w/2, pts[i][1], pts[i+1][0]-w/2, pts[i+1][1], c(0xa0583a))
			b.line(pts[i][0]-w/2+1, pts[i][1], pts[i+1][0]-w/2+1, pts[i+1][1], c(0x9a5034))
			b.line(pts[i][0]+w/2-1, pts[i][1], pts[i+1][0]+w/2-1, pts[i+1][1], c(0x3e1a10))
		}
		b.rect(18, 69, 20, 2, c(0x6a3020))
		b.rect(16, 70, 6, 1, c(0x3e1a10))
		b.rect(34, 70, 7, 1, c(0x3e1a10))
		for _, p := range [][4]int{{24, 52, 10, 44}, {30, 42, 45, 36}, {26, 33, 11, 28}, {28, 24, 44, 21}, {32, 62, 46, 54}} {
			treeLine(b, p[0]-1, p[1], p[2], p[3]+2, 2, c(0x5a2a18)) // the branches to the clouds
		}
		g := lumpPal{c(0x163e1a), c(0x2a7a30), c(0x48a444), c(0x8ad06a)}
		for i, p := range [][4]float64{{10, 42, 10, 5.5}, {46, 34, 10, 5.5}, {12, 26, 10, 5}, {43, 19, 11, 5.5}, {28, 11, 13, 6.5},
			{46, 52, 8, 4.5}, {20, 18, 7, 4}, {36, 28, 7, 4}} {
			b.ellipse(p[0], p[1]+1.5, p[2], p[3], g[0])
			b.ellipse(p[0], p[1], p[2]-1, p[3]-1.2, g[1])
			b.ellipse(p[0]-1.5, p[1]-1.5, p[2]-3.5, p[3]-2.8, g[2])
			for k := 0; k < int(p[2]); k++ { // needles catching the light, and dark tufts under the cloud
				h := fireHash(i, k, 12)
				b.set(int(p[0])-int(p[2])+2+h%int(p[2]*2-3), int(p[1]-p[3]*0.6)+h/7%2, g[3])
				b.set(int(p[0])-int(p[2])+2+h/3%int(p[2]*2-3), int(p[1]+p[3]*0.6)+1, g[0])
			}
		}
	}
}

type firTier struct{ top, h, half int }

// firTiers paints the tiers of a fir from the top: each one a wide triangle whose lower edge hangs in tips.
func firTiers(b *pixBuf, tiers []firTier, p lumpPal, drips bool) {
	for i, t := range tiers {
		for r := 0; r < t.h; r++ {
			hw := int(math.Round(float64(t.half) * math.Pow(float64(r+1)/float64(t.h), 0.8)))
			y := t.top + r
			for x := -hw; x <= hw; x++ {
				col := p[1]
				switch {
				case r >= t.h-2 || x > hw*2/3: // the underside and the shaded right side
					col = p[0]
				case x < -hw/3 && r < t.h-4:
					col = p[2]
				}
				if fireHash(x, y, i)%23 == 0 {
					col = p[3]
				}
				b.set(28+x, y, col)
			}
		}
		// the tips hanging under the tier
		y := t.top + t.h
		for x := -t.half; x <= t.half; x += 3 {
			l := 1 + fireHash(x, y, i)%2
			if drips {
				l += fireHash(x, y, 7) % 3
			}
			b.rect(28+x, y, 1, l, p[0])
		}
	}
}

// drawBushVar draws bush variant v (1..bushVars-1) in the 34x28 box of the bush (foot at the bottom).
func drawBushVar(b *pixBuf, v int) {
	switch v {
	case 1: // round light shrub in lumps
		g := lumpPal{c(0x1e6a2a), c(0x3aa444), c(0x6ccc58), c(0xb4ec80)}
		lump(b, 17, 19, 16, 8, g, 1)
		lump(b, 9, 15, 8, 7, g, 2)
		lump(b, 25, 15, 8, 7, g, 3)
		lump(b, 17, 10, 9, 7, g, 4)
	case 2: // flowering shrub: dark leaves full of pink and white flowers
		g := lumpPal{c(0x173f1e), c(0x24602c), c(0x387a3a), c(0x5a9a4a)}
		lump(b, 17, 19, 16, 8, g, 5)
		lump(b, 10, 14, 9, 7, g, 6)
		lump(b, 24, 14, 9, 7, g, 7)
		lump(b, 17, 9, 9, 6, g, 8)
		for i := 0; i < 14; i++ {
			h := fireHash(i, 2, 22)
			x, y := 5+h%24, 5+(h/31)%17
			col := []color.RGBA{c(0xf06a9a), c(0xf8b4cc), c(0xfcf0f4)}[i%3]
			b.rect(x, y, 2, 2, col)
			b.set(x, y, c(0xfce8a0)) // the centre of the flower
		}
	case 3: // dark bush with red berries
		g := lumpPal{c(0x10301a), c(0x1c4a28), c(0x2c6a36), c(0x4a8a4a)}
		lump(b, 17, 19, 16, 8, g, 9)
		lump(b, 10, 14, 9, 8, g, 10)
		lump(b, 24, 14, 9, 8, g, 11)
		lump(b, 17, 9, 8, 6, g, 12)
		for i := 0; i < 10; i++ {
			h := fireHash(i, 3, 33)
			x, y := 6+h%22, 7+(h/29)%15
			b.set(x, y, c(0xd02838))
			b.set(x+1, y, c(0x8a1424))
			b.set(x, y-1, c(0xff8a8a))
		}
	case 4: // dry shrub: a tangle of brown twigs with a few olive leaves
		for i := 0; i < 16; i++ {
			h := fireHash(i, 4, 44)
			x1, y1 := 3+h%28, 3+(h/23)%14
			b.line(17+(i%5)-2, 26, x1, y1, []color.RGBA{c(0x6a4a2a), c(0x4a3018), c(0x8a6a3a)}[i%3])
		}
		for i := 0; i < 22; i++ {
			h := fireHash(i, 5, 55)
			b.set(3+h%28, 3+(h/19)%18, []color.RGBA{c(0x7a7a3a), c(0x9a8a4a), c(0x5a5a2a)}[i%3])
		}
		b.rect(9, 25, 16, 2, c(0x4a3018))
	case 5: // pink blossom shrub (the small brother of the cherry)
		shadedCrown(b, 17, 15, 16, 11, []color.RGBA{c(0x6a2c58), c(0x9a4878), c(0xc86a8e), c(0xe8909e), c(0xf6b4b8), c(0xfcd8d0)}, 11)
		b.rect(15, 24, 4, 3, c(0x5a3020))
	}
}

// ---- which kind goes where ---------------------------------------------------------------------------------------

const (
	treeThemePark = iota
	treeThemeGrave
	treeThemeSnow
	treeThemeJapan
	treeThemeForest
	treeThemeEurope
	treeThemeBeach
)

var treePools = [...][]int{
	treeThemePark:   {0, 1, 2, 3, 1, 2, 6},
	treeThemeGrave:  {5, 4, 5, 3},
	treeThemeSnow:   {3, 4, 3, 5},
	treeThemeJapan:  {6, 7, 6, 1, 7},
	treeThemeForest: {3, 4, 2, 1, 5, 3},
	treeThemeEurope: {1, 2, 6, 0},
	treeThemeBeach:  {1, 0, 2},
}

var bushPools = [...][]int{
	treeThemePark:   {0, 1, 2, 3},
	treeThemeGrave:  {4, 3, 4},
	treeThemeSnow:   {3, 1},
	treeThemeJapan:  {5, 2, 1, 5},
	treeThemeForest: {0, 1, 3, 4},
	treeThemeEurope: {1, 2, 0},
	treeThemeBeach:  {1, 0, 2},
}

var treeThemeIcons = map[string]int{
	"grave": treeThemeGrave, "snow": treeThemeSnow, "hockey": treeThemeSnow,
	"italy": treeThemeEurope, "eiffel": treeThemeEurope,
	"beach": treeThemeBeach, "pirate": treeThemeBeach, "boat": treeThemeBeach, "jurassic": treeThemeBeach,
	"japan": treeThemeJapan, "zen": treeThemeJapan, "tokyo": treeThemeJapan, "kyoto": treeThemeJapan, "dojo": treeThemeJapan,
	"forest": treeThemeForest, "camp": treeThemeForest, "camping": treeThemeForest, "woods": treeThemeForest,
}

// words of a title (random-mode rolls carry no icon; new themed parties may carry new icons)
var treeThemeWords = []struct {
	w     string
	theme int
}{
	{"JAPON", treeThemeJapan}, {"JAPÓN", treeThemeJapan}, {"ZEN", treeThemeJapan}, {"TOKIO", treeThemeJapan}, {"KIOTO", treeThemeJapan},
	{"SAKURA", treeThemeJapan}, {"BOSQUE", treeThemeForest}, {"CAMPING", treeThemeForest}, {"CAMPAMENTO", treeThemeForest},
	{"CEMENTERIO", treeThemeGrave}, {"HALLOWEEN", treeThemeGrave}, {"ESQU", treeThemeSnow}, {"NIEVE", treeThemeSnow},
	{"ITALIA", treeThemeEurope}, {"EIFFEL", treeThemeEurope}, {"PLAYA", treeThemeBeach}, {"BEACH", treeThemeBeach},
}

func treeThemeOf(def *LevelDef) int {
	if def == nil {
		return treeThemePark
	}
	if t, ok := treeThemeIcons[def.Icon]; ok {
		return t
	}
	for _, k := range []string{"japan", "zen", "forest", "grave", "snow"} {
		if def.Icon != "" && strings.Contains(def.Icon, k) {
			return treeThemeIcons[k]
		}
	}
	up := strings.ToUpper(def.Title)
	for _, tw := range treeThemeWords {
		if strings.Contains(up, tw.w) {
			return tw.theme
		}
	}
	return treeThemePark
}

// a one-level cache: the theme is looked up once per level, not per frame
var (
	treeThemeDef *LevelDef
	treeThemeVal int
)

// treeVarFor: the variant to draw for a tree or a bush placed with variant 0 (its theme and its place pick one; an
// explicit variant is kept).
func treeVarFor(def *LevelDef, pr *Prop) int {
	if pr.Var != 0 {
		return pr.Var
	}
	if def != treeThemeDef {
		treeThemeDef, treeThemeVal = def, treeThemeOf(def)
	}
	pool := treePools[treeThemeVal]
	if pr.Name == "bush" {
		pool = bushPools[treeThemeVal]
	}
	return pool[fireHash(int(pr.X), int(pr.Y), 17)%len(pool)]
}

// treeArtBuiltin: the built-in kinds of the props drawn here, kept before the asset objects may replace them (the
// tests and the export compare the asset files with these).
var treeArtBuiltin = map[string]*PropKind{}

func init() { // runs after props.go / props2.go (file order): the built-in kinds are registered
	for _, n := range []string{"tree", "bush", "fireplace"} {
		treeArtBuiltin[n] = propKinds[n]
	}
}
