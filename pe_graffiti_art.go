package main

import (
	"image"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// THE GRAFFITI LIBRARY of the street painter (pe_graffiti.go): small pixel-art pieces, drawn by hand as letter maps
// and turned into pixels once at start-up. Every design has a lively palette of 4-6 colours with highlights, and a dark
// outline that is ADDED around the shape (and fills the 1-pixel holes: eyes, counters of the letters). The walls of
// the game show a face of 9-11 pixels under their cornice (walls_look.go), so the pieces are 7-9 rows tall (9-11
// with the outline) and 13-44 columns wide: they fit on the wall face, never over the cornice.
// TestGraffitiDesignsQuality checks every one (colours, outline, not empty, not noise, fits its box).
// ---------------------------------------------------------------------------

// gfMaxW, gfMaxH: the box every design (outline included) must fit in.
const (
	gfMaxW = 48
	gfMaxH = 11
)

var gfInk = color.RGBA{0x1a, 0x10, 0x22, 255}

type gfDesign struct {
	name string
	rows []string
	pal  map[byte]uint32
}

// gfArt: a finished design: w x h pixels (alpha 0 = bare wall); line = only the outline (the first pass of the spray).
type gfArt struct {
	name       string
	w, h       int
	pix        []color.RGBA
	full, line *ebiten.Image
}

func (a *gfArt) at(x, y int) color.RGBA { return a.pix[y*a.w+x] }

// gfWord builds the rows of a word in 5x7 bubble letters (top half pink 'p', bottom half magenta 'm', a white shine).
func gfWord(word string) []string {
	glyph := map[rune][7]string{
		'P': {"pppp.", "pwppp", "pp.pp", "ppppp", "mmmm.", "mm...", "mm..."},
		'A': {".ppp.", "pwppp", "pp.pp", "ppppp", "mmmmm", "mm.mm", "mm.mm"},
		'R': {"pppp.", "pwppp", "pp.pp", "pppp.", "mmmm.", "mm.mm", "mm.mm"},
		'T': {"ppppp", "pwppp", ".ppp.", ".ppp.", ".mmm.", ".mmm.", ".mmm."},
		'Y': {"pp.pp", "pw.pp", ".ppp.", ".ppp.", ".mmm.", ".mmm.", ".mmm."},
	}
	rows := make([]string, 7)
	for i, ch := range word {
		for r := 0; r < 7; r++ {
			if i > 0 {
				rows[r] += "."
			}
			rows[r] += glyph[ch][r]
		}
	}
	rows[0] = "y.." + rows[0] + "..c"
	for r := 1; r < 7; r++ {
		rows[r] = "..." + rows[r] + "..."
	}
	rows[6] = "c" + rows[6][1:len(rows[6])-1] + "y"
	return rows
}

var gfDesigns = []gfDesign{
	{"calavera", []string{
		"y......wwwww......y",
		".....wwwwwwwwg.....",
		"....wwwwwwwwwwg....",
		"....wkkwwwwkkwg....",
		".c..wkpwwwwkpwg..c.",
		"....wwwwwkwwwgg....",
		".....gwwkkkwgg.....",
		"......wkwkwkg......",
		".......gwgwg.......",
	}, map[byte]uint32{'w': 0xfff8f0, 'g': 0xb8b0d8, 'p': 0xff5aa8, 'y': 0xffd84a, 'c': 0x48d8f0}},
	{"corona", []string{
		"y........w........y",
		".....o...o...o.....",
		"....ooo.ooo.ooo....",
		"....ooooooooooo....",
		"....owooroobooo....",
		"....ooooooooooo....",
		"....hhhhhhhhhhh....",
		"..y.............y..",
	}, map[byte]uint32{'o': 0xffc830, 'h': 0xc87818, 'r': 0xff3050, 'b': 0x40a0ff, 'w': 0xfff8f0, 'y': 0xfff070}},
	{"corazon", []string{
		"ww.....rrr.rrr.....ww",
		"cwww..rrrrrrrrr..wwwc",
		".cwww.rprrrrrrr.wwwc.",
		"..cwwwrprrrrrrrwwwc..",
		"...cc.rrrrrrrrr.cc...",
		".......drrrrrd.......",
		"........drrrd........",
		".........drd.........",
		"..........d..........",
	}, map[byte]uint32{'r': 0xf03a5a, 'p': 0xff9ab8, 'd': 0xa01838, 'w': 0xfff8f0, 'c': 0x7ad0ff}},
	{"notas", []string{
		".....mmmmmmmmmmmm..y",
		".....mppppppppppm...",
		".....m..........m...",
		".c...m..........m...",
		".....m..........m.c.",
		"..mmmm.......mmmm...",
		".mmwmm......mmwmm...",
		".mmmmm......mmmmm...",
		"..mmm........mmm..y.",
	}, map[byte]uint32{'m': 0xe040c0, 'p': 0xff90e0, 'w': 0xfff8f0, 'y': 0xffe040, 'c': 0x48e0f0}},
	{"gato", []string{
		"y..o.........o..c",
		"...op.......po...",
		"...ooooooooooo...",
		"..oookkkkokkkkoo.",
		"..ookbkkkokbkkko.",
		"..ooooooooooooooo",
		"...ooodpwpdoooo..",
		"....oooodooooo...",
		"c....ooooooooo..y",
	}, map[byte]uint32{'o': 0xff9a30, 'd': 0xc85a18, 'p': 0xff8ab0, 'b': 0x60e8ff, 'w': 0xfff8f0, 'y': 0xffe040, 'c': 0x48e0f0}},
	{"pulpo", []string{
		"....vvvvvvv....y",
		"...vlllvvvvvv...",
		"..vlvvvvvvvvvv..",
		"..vvwkvvvvwkvv..",
		"..vvvvvvpvvvvv..",
		"...vvvvvvvvvv...",
		"..vv.vv.vv.vv.vv",
		".vv..v..v..v..v.",
		".v..vv..vv.vv...",
	}, map[byte]uint32{'v': 0x9a5aff, 'l': 0xc8a8ff, 'w': 0xfff8f0, 'p': 0xff6aa8, 'y': 0xffe040}},
	{"rayo", []string{
		"c......yyyyyy..",
		".....yyyyyyo...",
		"....yywyyyo....",
		"...yyyyyyo...c.",
		"..yyyyyyyyyyy..",
		"......yyyyyo...",
		".r...yyyyo.....",
		"....yyyo.......",
		"...yyo.......r.",
	}, map[byte]uint32{'y': 0xffe040, 'o': 0xff9020, 'w': 0xfffbe0, 'c': 0x40e0ff, 'r': 0xff4060}},
	{"estrella", []string{
		".c.....yy.......",
		"......yyyo......",
		"yyyyyyywyyyyyyyo",
		".yyyykyyyykyyyo.",
		"..yypyyyyyypyo..",
		"...yyyykkyyyo...",
		"...yyyyyyyyyo...",
		"..yyyyo..yyyyo..",
		".yyyo......yyyo.",
	}, map[byte]uint32{'y': 0xffd838, 'o': 0xf08a20, 'w': 0xfffbe0, 'p': 0xff7aa8, 'c': 0x48e0f0}},
	{"party", gfWord("PARTY"), map[byte]uint32{'p': 0xff70c0, 'm': 0xc02890, 'w': 0xfff8f0, 'y': 0xffe040, 'c': 0x48e0f0}},
	{"monstruo", []string{
		"..h.......h..",
		"..hh.....hh..",
		"..ggggggggg..",
		".ggwwwwwwwgg.",
		".ggwwwkkwwgd.",
		".gggwwwwwggd.",
		".gwgwgwgwgwd.",
		".gggrrrrgggd.",
		"..ddd...ddd..",
	}, map[byte]uint32{'g': 0x60d050, 'd': 0x2a8a3a, 'h': 0xffd040, 'w': 0xfff8f0, 'r': 0xff4060}},
	{"cohete", []string{
		".......rrr..........",
		".......rrrr.........",
		"oo.yyrrwwwwwwwwr....",
		"ooyyyrwwwbbwwwwwrr..",
		"ooyyyrwwwbbwwwwwrrr.",
		"oo.yyrrggggggggr....",
		".......rrrr.........",
		".......rrr..........",
	}, map[byte]uint32{'r': 0xff4050, 'w': 0xf0f0f8, 'g': 0xa8b0c8, 'b': 0x50b8ff, 'o': 0xff8a20, 'y': 0xffe040}},
	{"tarta", []string{
		"....y...y...y....",
		"....r...r...r....",
		"....w...w...w....",
		"..ppppppppppppp..",
		"..pwpppwpppwppp..",
		"..ccccccccccccc..",
		"..cbcccbcccbccc..",
		"..bbbbbbbbbbbbb..",
	}, map[byte]uint32{'y': 0xffe040, 'r': 0xff4060, 'w': 0xfff8f0, 'p': 0xff9ac8, 'c': 0xffe0a8, 'b': 0x9a5a30}},
	{"gafas", []string{
		"y.................y",
		"mmmmmmmmmmmmmmmmmmm",
		"mkkkkkkkm.mkkkkkkkm",
		"mkcwkkkkm.mkcwkkkkm",
		"mkkckkkkmmmkkckkkkm",
		".mkkkkkm...mkkkkkm.",
		"..mmmmm.....mmmmm..",
	}, map[byte]uint32{'m': 0xff40a0, 'c': 0x48e0f0, 'w': 0xfff8f0, 'y': 0xffe040}},
	{"pez", []string{
		".......bbbb......o",
		"....bbbbbbbbb...t.",
		"..bbwkbbbbbbbb.tt.",
		".bbbbbblblblbbbttt",
		"bbbbbbblblblbbbtt.",
		".blllllllllllb.t..",
		"...lllllll........",
		"......y.......o...",
	}, map[byte]uint32{'b': 0x3a90ff, 'l': 0x90d0ff, 'w': 0xfff8f0, 't': 0x30d0b0, 'y': 0xffd040, 'o': 0xc8f0ff}},
	{"planta", []string{
		"....p.......",
		"...pwp..l...",
		".l..p..ll...",
		".lll..lgl...",
		"..lgllgl.l..",
		"...gggg.....",
		"..oooooo....",
		"..owoooo....",
		"...dddd.....",
	}, map[byte]uint32{'l': 0x8ae860, 'g': 0x3a9a40, 'o': 0xd06a30, 'd': 0x8a3a18, 'w': 0xfff8f0, 'p': 0xff70b0}},
}

// gfLib: the designs, built once.
var gfLib = func() []*gfArt {
	out := make([]*gfArt, 0, len(gfDesigns))
	for _, d := range gfDesigns {
		out = append(out, gfBuild(d))
	}
	return out
}()

// gfBuild turns a letter map into pixels: 'k' and the added ring around the shape are the outline; '.' is bare wall.
func gfBuild(d gfDesign) *gfArt {
	w := 0
	for _, r := range d.rows {
		if len(r) > w {
			w = len(r)
		}
	}
	h := len(d.rows)
	W, H := w+2, h+2
	a := &gfArt{name: d.name, w: W, h: H, pix: make([]color.RGBA, W*H)}
	solid := make([]bool, W*H)
	for y, r := range d.rows {
		r += strings.Repeat(".", w-len(r))
		for x := 0; x < w; x++ {
			ch := r[x]
			if ch == '.' {
				continue
			}
			i := (y+1)*W + x + 1
			solid[i] = true
			if ch == 'k' {
				a.pix[i] = gfInk
			} else if c, ok := d.pal[ch]; ok {
				a.pix[i] = rgb(c)
			} else {
				panic("graffiti " + d.name + ": no colour for " + string(ch))
			}
		}
	}
	// the outline: every bare pixel touching the shape (8 neighbours) is ink
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			if solid[y*W+x] {
				continue
			}
			touch := false
			for dy := -1; dy <= 1 && !touch; dy++ {
				for dx := -1; dx <= 1; dx++ {
					xx, yy := x+dx, y+dy
					if xx >= 0 && yy >= 0 && xx < W && yy < H && solid[yy*W+xx] {
						touch = true
						break
					}
				}
			}
			if touch {
				a.pix[y*W+x] = gfInk
			}
		}
	}
	return a
}

// gfImages makes (once) the ebiten images of a design: the whole piece and the outline alone.
func (a *gfArt) images() (*ebiten.Image, *ebiten.Image) {
	if a.full == nil {
		full := image.NewRGBA(image.Rect(0, 0, a.w, a.h))
		line := image.NewRGBA(image.Rect(0, 0, a.w, a.h))
		for y := 0; y < a.h; y++ {
			for x := 0; x < a.w; x++ {
				c := a.at(x, y)
				if c.A == 0 {
					continue
				}
				full.SetRGBA(x, y, c)
				if c == gfInk {
					line.SetRGBA(x, y, c)
				}
			}
		}
		a.full, a.line = ebiten.NewImageFromImage(full), ebiten.NewImageFromImage(line)
	}
	return a.full, a.line
}

// gfContactSheet: every design side by side at scale k on a wall colour (for the review of the user).
func gfContactSheet(k int) *image.RGBA {
	const cols = 4
	cw, ch := (gfMaxW+8)*k, (gfMaxH+8)*k
	rows := (len(gfLib) + cols - 1) / cols
	im := image.NewRGBA(image.Rect(0, 0, cols*cw+8*k, rows*ch+8*k))
	bg := rgb(0x6a5646)
	for i := range im.Pix {
		im.Pix[i] = []uint8{bg.R, bg.G, bg.B, 255}[i%4]
	}
	for i, a := range gfLib {
		ox, oy := 8*k+(i%cols)*cw, 8*k+(i/cols)*ch
		for y := 0; y < a.h; y++ {
			for x := 0; x < a.w; x++ {
				c := a.at(x, y)
				if c.A == 0 {
					continue
				}
				for dy := 0; dy < k; dy++ {
					for dx := 0; dx < k; dx++ {
						im.SetRGBA(ox+x*k+dx, oy+y*k+dy, c)
					}
				}
			}
		}
	}
	return im
}
