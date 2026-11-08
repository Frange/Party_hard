package main

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/bitmapfont/v3"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// A tiny software canvas for the newspaper front page. The whole page is painted in pixels here (an *image.RGBA of
// screenW x screenH) and uploaded as ONE texture drawn 1:1, so every letter lands on whole pixels at a whole scale:
// nothing is filtered, scaled by a fraction or anti-aliased. The font is the game's own 1-bit bitmap font.

const (
	pAsc    = 12 // ascent of the bitmap font inside its 16 px cell
	pInkH   = 12 // rows of ink of a line of text (accents to descenders)
	pInkTop = 2  // blank rows at the top of the cell
	pRowH   = 14 // line pitch of the small text
)

type ptext struct {
	s                 string
	x, y, w, h, scale int
	col               color.RGBA
	dark              bool // drawn on a dark background (the banner)
	bold, title       bool
}

type prect struct {
	kind       string
	x, y, w, h int
}

// paperLayout: everything the page drew, for the tests and the text audit.
type paperLayout struct {
	texts  []ptext
	blocks []prect
	banner prect
	briefs []string // the small news of the page, in order
}

type pcv struct {
	im  *image.RGBA
	lay *paperLayout
}

func paperTextW(s string) int { return font.MeasureString(bitmapfont.Face, s).Ceil() }

func (c *pcv) put(x, y int, col color.RGBA) {
	if x < 0 || y < 0 || x >= c.im.Rect.Dx() || y >= c.im.Rect.Dy() {
		return
	}
	i := c.im.PixOffset(x, y)
	c.im.Pix[i], c.im.Pix[i+1], c.im.Pix[i+2], c.im.Pix[i+3] = col.R, col.G, col.B, 255
}

func (c *pcv) at(x, y int) color.RGBA {
	if x < 0 || y < 0 || x >= c.im.Rect.Dx() || y >= c.im.Rect.Dy() {
		return color.RGBA{}
	}
	return c.im.RGBAAt(x, y)
}

func (c *pcv) fill(x, y, w, h int, col color.RGBA) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			c.put(xx, yy, col)
		}
	}
}

// shade adds d to the three channels (negative = darker): a tint that keeps the paper grain.
func (c *pcv) shade(x, y, w, h, d int) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			if xx < 0 || yy < 0 || xx >= c.im.Rect.Dx() || yy >= c.im.Rect.Dy() {
				continue
			}
			i := c.im.PixOffset(xx, yy)
			for k := 0; k < 3; k++ {
				c.im.Pix[i+k] = pclamp(int(c.im.Pix[i+k]) + d)
			}
		}
	}
}

func pclamp(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func (c *pcv) frame(x, y, w, h int, col color.RGBA) {
	c.fill(x, y, w, 1, col)
	c.fill(x, y+h-1, w, 1, col)
	c.fill(x, y, 1, h, col)
	c.fill(x+w-1, y, 1, h, col)
}

func (c *pcv) dotted(x, y, w int, col color.RGBA) {
	for i := 0; i < w; i += 2 {
		c.put(x+i, y, col)
	}
}

// text draws s with its ink top-left at (x, y) at a whole scale; returns the width in pixels.
func (c *pcv) text(s string, x, y, scale int, col color.RGBA) int {
	return c.textOpt(s, x, y, scale, col, false, false)
}

func (c *pcv) bold(s string, x, y int, col color.RGBA) int {
	return c.textOpt(s, x, y, 1, col, false, true)
}

func (c *pcv) textOpt(s string, x, y, scale int, col color.RGBA, dark, bold bool) int {
	if s == "" {
		return 0
	}
	w := paperTextW(s)
	const pad = 3
	mask := image.NewAlpha(image.Rect(0, 0, w+2*pad, 16))
	d := font.Drawer{Dst: mask, Src: image.Opaque, Face: bitmapfont.Face, Dot: fixed.P(pad, pAsc)}
	d.DrawString(s)
	for pass := 0; pass < 2; pass++ {
		if pass == 1 && !bold {
			break
		}
		for my := 0; my < 16; my++ {
			for mx := 0; mx < w+2*pad; mx++ {
				if mask.AlphaAt(mx, my).A >= 128 {
					c.fill(x+(mx-pad)*scale+pass, y+(my-pInkTop)*scale, scale, scale, col)
				}
			}
		}
	}
	tw := w * scale
	if bold {
		tw++
	}
	c.lay.texts = append(c.lay.texts, ptext{s: s, x: x, y: y, w: tw, h: pInkH * scale, scale: scale, col: col, dark: dark, bold: bold})
	return tw
}

func (c *pcv) textR(s string, right, y, scale int, col color.RGBA) {
	c.text(s, right-paperTextW(s)*scale, y, scale, col)
}

func (c *pcv) textC(s string, cx, y, scale int, col color.RGBA) {
	c.text(s, cx-paperTextW(s)*scale/2, y, scale, col)
}

// wrapPx breaks s at spaces into lines no wider than maxW pixels (scale 1).
func wrapPx(s string, maxW int) []string {
	var out []string
	cur := ""
	for _, w := range splitWords(s) {
		if cur != "" && paperTextW(cur+" "+w) > maxW {
			out = append(out, cur)
			cur = w
			continue
		}
		if cur != "" {
			cur += " "
		}
		cur += w
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func splitWords(s string) []string {
	var out []string
	w := ""
	for _, r := range s {
		if r == ' ' {
			if w != "" {
				out = append(out, w)
			}
			w = ""
			continue
		}
		w += string(r)
	}
	if w != "" {
		out = append(out, w)
	}
	return out
}

// ---- the newspaper title: a heavy serif alphabet drawn from 5x7 cells with serif feet -------------------------

var paperGlyphs = map[rune][7]string{
	'A': {".###.", "#...#", "#...#", "#####", "#...#", "#...#", "#...#"},
	'B': {"####.", "#...#", "#...#", "####.", "#...#", "#...#", "####."},
	'C': {".####", "#....", "#....", "#....", "#....", "#....", ".####"},
	'D': {"####.", "#...#", "#...#", "#...#", "#...#", "#...#", "####."},
	'E': {"#####", "#....", "#....", "####.", "#....", "#....", "#####"},
	'F': {"#####", "#....", "#....", "####.", "#....", "#....", "#...."},
	'G': {".####", "#....", "#....", "#.###", "#...#", "#...#", ".####"},
	'H': {"#...#", "#...#", "#...#", "#####", "#...#", "#...#", "#...#"},
	'I': {"#####", "..#..", "..#..", "..#..", "..#..", "..#..", "#####"},
	'J': {"..###", "...#.", "...#.", "...#.", "#..#.", "#..#.", ".##.."},
	'K': {"#...#", "#..#.", "#.#..", "##...", "#.#..", "#..#.", "#...#"},
	'L': {"#....", "#....", "#....", "#....", "#....", "#....", "#####"},
	'M': {"#...#", "##.##", "#.#.#", "#...#", "#...#", "#...#", "#...#"},
	'N': {"#...#", "##..#", "#.#.#", "#..##", "#...#", "#...#", "#...#"},
	'O': {".###.", "#...#", "#...#", "#...#", "#...#", "#...#", ".###."},
	'P': {"####.", "#...#", "#...#", "####.", "#....", "#....", "#...."},
	'Q': {".###.", "#...#", "#...#", "#...#", "#.#.#", "#..#.", ".##.#"},
	'R': {"####.", "#...#", "#...#", "####.", "#.#..", "#..#.", "#...#"},
	'S': {".####", "#....", "#....", ".###.", "....#", "....#", "####."},
	'T': {"#####", "..#..", "..#..", "..#..", "..#..", "..#..", "..#.."},
	'U': {"#...#", "#...#", "#...#", "#...#", "#...#", "#...#", ".###."},
	'V': {"#...#", "#...#", "#...#", "#...#", ".#.#.", ".#.#.", "..#.."},
	'W': {"#...#", "#...#", "#...#", "#.#.#", "#.#.#", "##.##", "#...#"},
	'X': {"#...#", ".#.#.", "..#..", "..#..", "..#..", ".#.#.", "#...#"},
	'Y': {"#...#", "#...#", ".#.#.", "..#..", "..#..", "..#..", "..#.."},
	'Z': {"#####", "....#", "...#.", "..#..", ".#...", "#....", "#####"},
	'Ó': {"..#..", ".###.", "#...#", "#...#", "#...#", "#...#", ".###."},
	'Á': {"..#..", ".###.", "#...#", "#####", "#...#", "#...#", "#...#"},
	'É': {"..#..", "#####", "#....", "####.", "#....", "#....", "#####"},
	'Í': {"..#..", "#####", "..#..", "..#..", "..#..", "..#..", "#####"},
	'Ú': {"..#..", "#...#", "#...#", "#...#", "#...#", "#...#", ".###."},
	'Ñ': {".#.#.", "#...#", "##..#", "#.#.#", "#..##", "#...#", "#...#"},
}

const paperGlyphAdv = 7 // cells per letter (5 + a serif cell on each side)
const paperSpaceAdv = 4

// paperGlyphCells: the lit cells of a letter in a 7 x 7 grid, serifs added to the lone stems of the top and bottom rows.
func paperGlyphCells(r rune) (cells [7][7]bool, ok bool) {
	g, ok := paperGlyphs[r]
	if !ok {
		return cells, false
	}
	for y := 0; y < 7; y++ {
		for x := 0; x < 5; x++ {
			if g[y][x] == '#' {
				cells[y][x+1] = true
			}
		}
		if y == 0 || y == 6 {
			for x := 0; x < 5; x++ {
				if g[y][x] != '#' {
					continue
				}
				l := x > 0 && g[y][x-1] == '#'
				r := x < 4 && g[y][x+1] == '#'
				if !l && !r {
					cells[y][x] = true
					cells[y][x+2] = true
				}
			}
		}
	}
	return cells, true
}

func paperTitleW(s string, sc int) int {
	w := 0
	for _, r := range s {
		if r == ' ' {
			w += paperSpaceAdv
		} else {
			w += paperGlyphAdv
		}
	}
	return w * sc
}

// title draws the newspaper name centred on cx with a shadow; returns false if a letter has no glyph.
func (c *pcv) title(s string, cx, y, sc int, ink, shadow color.RGBA) bool {
	x0 := cx - paperTitleW(s, sc)/2
	ok := true
	for pass := 0; pass < 2; pass++ {
		x := x0
		off := 0
		col := ink
		if pass == 0 {
			off, col = sc/2+1, shadow
		}
		for _, r := range s {
			if r == ' ' {
				x += paperSpaceAdv * sc
				continue
			}
			cells, has := paperGlyphCells(r)
			if !has {
				ok = false
			}
			for gy := 0; gy < 7; gy++ {
				for gx := 0; gx < 7; gx++ {
					if cells[gy][gx] {
						c.fill(x+gx*sc+off, y+gy*sc+off, sc, sc, col)
					}
				}
			}
			x += paperGlyphAdv * sc
		}
	}
	c.lay.texts = append(c.lay.texts, ptext{s: s, x: x0, y: y, w: paperTitleW(s, sc), h: 7 * sc, scale: sc, col: ink, title: true})
	return ok
}

// ---- small drawing helpers for the photo --------------------------------------------------------------------------

var bayer4 = [16]int{0, 8, 2, 10, 12, 4, 14, 6, 3, 11, 1, 9, 15, 7, 13, 5}

// dither: true for roughly frac (0..1) of the pixels, in a fixed ordered pattern.
func dither(frac float64, x, y int) bool {
	return float64(bayer4[(y&3)*4+(x&3)])+0.5 < frac*16
}

func (c *pcv) ellipse(cx, cy, rx, ry int, col color.RGBA) {
	for y := -ry; y <= ry; y++ {
		for x := -rx; x <= rx; x++ {
			if x*x*ry*ry+y*y*rx*rx <= rx*rx*ry*ry {
				c.put(cx+x, cy+y, col)
			}
		}
	}
}

func (c *pcv) thickLine(x0, y0, x1, y1, th int, colAt func(i int) color.RGBA) {
	n := int(math.Max(math.Abs(float64(x1-x0)), math.Abs(float64(y1-y0))))
	for i := 0; i <= n; i++ {
		x := x0 + (x1-x0)*i/maxI(n, 1)
		y := y0 + (y1-y0)*i/maxI(n, 1)
		col := colAt(i)
		for t := 0; t < th; t++ {
			c.put(x, y+t, col)
		}
	}
}

func (c *pcv) tri(x0, y0, x1, y1, x2, y2 int, col color.RGBA) {
	minX, maxX := minI(x0, minI(x1, x2)), maxI(x0, maxI(x1, x2))
	minY, maxY := minI(y0, minI(y1, y2)), maxI(y0, maxI(y1, y2))
	e := func(ax, ay, bx, by, px, py int) int { return (bx-ax)*(py-ay) - (by-ay)*(px-ax) }
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			a, b, d := e(x0, y0, x1, y1, x, y), e(x1, y1, x2, y2, x, y), e(x2, y2, x0, y0, x, y)
			if (a >= 0 && b >= 0 && d >= 0) || (a <= 0 && b <= 0 && d <= 0) {
				c.put(x, y, col)
			}
		}
	}
}
