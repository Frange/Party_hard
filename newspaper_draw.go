package main

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// The end-of-level NEWSPAPER is the ONLY end-of-level screen (ModeLevelDone): the front page holds the news story, the
// statistics of J1 (always on the left) and of J2 (right; without J2 that column is filled with small news), the
// points table, a pixel-art photo, small news filling the rest and the banner "PULSA PARA CONTINUAR".
// Rules: every text is drawn at a whole scale on whole pixels by the software canvas (newspaper_canvas.go); the page is
// ONE texture shown 1:1, so nothing can be blurry. The body is tiled by blocks that never overlap (tests check it).

var (
	paperInk   = color.RGBA{0x20, 0x18, 0x0e, 255}
	paperRed   = color.RGBA{0x8a, 0x1c, 0x10, 255}
	paperCream = color.RGBA{0xf4, 0xea, 0xd0, 255}
	paperShade = color.RGBA{0x8d, 0x7e, 0x60, 255}
)

func lsFmtTime(t float64) string { return fmt.Sprintf("%d:%02d", int(t)/60, int(t)%60) }

// paperStatBox: the statistics of one player (rows already translated).
type paperStatBox struct {
	head, sub string
	rows      [][2]string
}

// paperData: everything the page shows, already translated. Built from the game (paperDataNow) or by hand in tests.
// In the briefs, .head and .es hold the TRANSLATED head and text.
type paperData struct {
	seed                                 int64
	two                                  bool
	title                                string
	earL1, earL2, earR1, earR2           string
	date, motto, dateR                   string
	head, dek                            string
	story                                []string
	storyHead, quote, quoteHead, quoteBy string
	photoVar                             int
	caption                              string
	stats                                []paperStatBox
	scoreHead, scoreTag                  string
	score                                [][2]string
	totalK, total, walletK, wallet       string
	briefs                               []paperBrief
	hint, hint2, foot, banner            string
}

const (
	pbX0, pbX1 = 14, 946  // the page content
	pbY0, pbY1 = 156, 480 // the body between the headline and the footer
	pcLX, pcLW = 14, 200  // left column (J1)
	pcCX, pcCW = 224, 512 // centre
	pcRX, pcRW = 746, 200 // right column (J2 / small news)
	pbRowA     = 164
	pbPhotoW   = 196
	pbPhotoH   = 116
	pbBannerW  = 320
	pbBannerH  = 38
	pbBannerY  = 490
	pbBannerX  = screenW/2 - pbBannerW/2
)

type pbox struct {
	kind string
	h    int
	draw func(x, y, w, h int)
}

type prender struct {
	c    *pcv
	d    *paperData
	used []bool
}

func (r *prender) block(kind string, x, y, w, h int) {
	r.c.lay.blocks = append(r.c.lay.blocks, prect{kind, x, y, w, h})
}

func briefLines(b paperBrief, w int) []string { return wrapPx(b.es, w-10) }

func briefH(b paperBrief, w int) int { return 4 + pRowH + len(briefLines(b, w))*pRowH + 5 }

// takeBriefs picks briefs (in the shuffled order, none twice on the page) whose heights sum to at most avail.
func (r *prender) takeBriefs(w, avail int) []pbox {
	var out []pbox
	rem := avail
	for i, b := range r.d.briefs {
		if r.used[i] || rem < briefH(paperBrief{es: "x"}, w) {
			continue
		}
		h := briefH(b, w)
		if h > rem {
			continue
		}
		r.used[i] = true
		r.c.lay.briefs = append(r.c.lay.briefs, b.es)
		rem -= h
		b := b
		out = append(out, pbox{"brief", h, func(x, y, w, h int) {
			r.c.bold(b.head, x+5, y+4, paperRed)
			for k, l := range briefLines(b, w) {
				r.c.text(l, x+5, y+4+pRowH+k*pRowH, 1, paperInk)
			}
		}})
	}
	return out
}

// stack lays boxes one under another in the column (x, y, w, h); the spare height is shared out as padding, so the
// boxes tile the column exactly.
func (r *prender) stack(x, w, y, h int, bs []pbox) {
	if len(bs) == 0 {
		return
	}
	sum := 0
	for _, b := range bs {
		sum += b.h
	}
	slack := h - sum
	n := len(bs)
	yy := y
	for i, b := range bs {
		bh := b.h + slack/n
		if i == n-1 {
			bh = y + h - yy
		}
		if i > 0 {
			r.c.fill(x, yy, w, 1, paperInk)
		}
		b.draw(x, yy, w, bh)
		r.block(b.kind, x, yy, w, bh)
		yy += bh
	}
}

const (
	statRowH   = 15
	statHeadH  = 22
	statSubH   = 14
	scoreRowH  = 15
	scoreHeadH = 24
	scoreTotH  = 38
)

func statBoxH(sb paperStatBox) int {
	h := statHeadH + len(sb.rows)*statRowH + 6
	if sb.sub != "" {
		h += statSubH
	}
	return h
}

func (r *prender) statBox(sb paperStatBox) pbox {
	return pbox{"stats", statBoxH(sb), func(x, y, w, h int) {
		c := r.c
		c.shade(x, y, w, h, -9)
		c.bold(sb.head, x+6, y+6, paperInk)
		c.fill(x+4, y+19, w-8, 1, paperInk)
		ry := y + statHeadH
		if sb.sub != "" {
			c.text(sb.sub, x+6, ry, 1, paperRed)
			ry += statSubH
		}
		for _, row := range sb.rows {
			c.text(row[0], x+6, ry, 1, paperInk)
			c.bold(row[1], x+w-7-paperTextW(row[1]), ry, paperRed)
			c.dotted(x+4, ry+13, w-8, paperShade)
			ry += statRowH
		}
	}}
}

func scoreBoxH(n int) int { return scoreHeadH + ((n+1)/2)*scoreRowH + 6 + scoreTotH + 6 }

func (r *prender) scoreBox() pbox {
	d := r.d
	return pbox{"score", scoreBoxH(len(d.score)), func(x, y, w, h int) {
		c := r.c
		c.shade(x, y, w, h, -9)
		c.fill(x+4, y+3, w-8, 2, paperInk)
		c.bold(d.scoreHead, x+w/2-paperTextW(d.scoreHead)/2, y+8, paperInk)
		if d.scoreTag != "" {
			c.textR(d.scoreTag, x+w-8, y+8, 1, paperRed)
		}
		c.fill(x+4, y+22, w-8, 1, paperInk)
		half := (len(d.score) + 1) / 2
		colW := (w - 24) / 2
		for i, row := range d.score {
			cx := x + 8
			k := i
			if i >= half {
				cx, k = x+w/2+4, i-half
			}
			ry := y + scoreHeadH + 2 + k*scoreRowH
			c.text(row[0], cx, ry, 1, paperInk)
			c.bold(row[1], cx+colW-1-paperTextW(row[1]), ry, paperRed)
			c.dotted(cx, ry+13, colW, paperShade)
		}
		c.fill(x+w/2-1, y+scoreHeadH+2, 1, half*scoreRowH-2, paperShade)
		ty := y + scoreHeadH + half*scoreRowH + 6
		c.fill(x+4, ty-3, w-8, 2, paperInk)
		c.text(d.totalK, x+10, ty+4, 2, paperInk)
		c.text(d.total, x+w-10-paperTextW(d.total)*2, ty+4, 2, paperRed)
		c.textR(d.walletK+" "+d.wallet, x+w-10, ty+4+24+2, 1, paperInk)
	}}
}

// storyLines: the sentences that fit in maxLines lines of width w (the first and the last are kept longest).
func storyLines(sent []string, w, maxLines int) []string {
	sel := append([]string{}, sent...)
	for {
		para := ""
		for i, s := range sel {
			if i > 0 {
				para += " "
			}
			para += s
		}
		lines := wrapPx(para, w)
		if len(lines) <= maxLines || len(sel) <= 2 {
			return lines
		}
		sel = append(sel[:len(sel)-2], sel[len(sel)-1])
	}
}

func (r *prender) rowA() pbox {
	d := r.d
	const sw = pcCW - pbPhotoW - 12
	quoteLines := wrapPx(d.quote, sw-10)
	quoteH := 4 + pRowH + len(quoteLines)*pRowH + pRowH + 5
	maxStory := (pbRowA - quoteH - 4 - pRowH - 5) / pRowH
	sl := storyLines(d.story, sw-10, maxStory)
	story := pbox{"story", 4 + pRowH + len(sl)*pRowH + 5, func(x, y, w, h int) {
		r.c.bold(d.storyHead, x+5, y+4, paperRed)
		for k, l := range sl {
			r.c.text(l, x+5, y+4+pRowH+k*pRowH, 1, paperInk)
		}
	}}
	quote := pbox{"quote", quoteH, func(x, y, w, h int) {
		c := r.c
		c.shade(x, y, w, h, -9)
		c.bold(d.quoteHead, x+5, y+4, paperInk)
		for k, l := range quoteLines {
			c.text(l, x+5, y+4+pRowH+k*pRowH, 1, paperRed)
		}
		c.textR(d.quoteBy, x+w-6, y+4+pRowH+len(quoteLines)*pRowH, 1, paperInk)
	}}
	return pbox{"rowA", pbRowA, func(x, y, w, h int) {
		c := r.c
		c.photo(x, y+1, pbPhotoW, pbPhotoH, d.photoVar, d.two, d.seed)
		for k, l := range wrapPx(d.caption, pbPhotoW) {
			c.text(l, x, y+pbPhotoH+6+k*pRowH, 1, paperInk)
		}
		r.block("photo", x, y, pbPhotoW, h)
		for yy := y; yy < y+h; yy += 2 {
			c.put(x+pbPhotoW+5, yy, paperShade)
		}
		r.stack(x+pbPhotoW+12, sw, y, h, []pbox{story, quote})
	}}
}

// body lays the three columns.
func (r *prender) body() {
	c, d := r.c, r.d
	c.fill(pcLX+pcLW+5, pbY0, 1, pbY1-pbY0, paperInk)
	c.fill(pcCX+pcCW+4, pbY0, 1, pbY1-pbY0, paperInk)
	H := pbY1 - pbY0
	// left: J1
	left := []pbox{r.statBox(d.stats[0])}
	left = append(left, r.takeBriefs(pcLW, H-left[0].h)...)
	// right: J2, or only small news
	var right []pbox
	if len(d.stats) > 1 {
		right = []pbox{r.statBox(d.stats[1])}
	}
	rh := 0
	for _, b := range right {
		rh += b.h
	}
	right = append(right, r.takeBriefs(pcRW, H-rh)...)
	// centre: photo + story, points table, then small news
	a, s := r.rowA(), r.scoreBox()
	centre := []pbox{a, s}
	centre = append(centre, r.takeBriefs(pcCW, H-a.h-s.h)...)
	r.stack(pcLX, pcLW, pbY0, H, left)
	r.stack(pcCX, pcCW, pbY0, H, centre)
	r.stack(pcRX, pcRW, pbY0, H, right)
}

func (r *prender) masthead() {
	c, d := r.c, r.d
	c.fill(pbX0, 14, pbX1-pbX0, 2, paperInk)
	ear := func(x int, a, b string) {
		c.frame(x, 20, 150, 34, paperInk)
		c.shade(x+1, 21, 148, 32, -8)
		c.textC(a, x+75, 25, 1, paperInk)
		c.textC(b, x+75, 39, 1, paperInk)
	}
	ear(20, d.earL1, d.earL2)
	ear(790, d.earR1, d.earR2)
	if !c.title(d.title, 480, 23, 4, paperInk, paperShade) {
		panic("newspaper title letter without a glyph: " + d.title)
	}
	c.fill(pbX0, 58, pbX1-pbX0, 2, paperInk)
	c.fill(pbX0, 62, pbX1-pbX0, 1, paperInk)
	c.text(d.date, pbX0, 66, 1, paperInk)
	c.textC(d.motto, 480, 66, 1, paperRed)
	c.textR(d.dateR, pbX1, 66, 1, paperInk)
	c.fill(pbX0, 80, pbX1-pbX0, 1, paperInk)
	c.fill(pbX0, 82, pbX1-pbX0, 2, paperInk)
	r.block("masthead", pbX0, 14, pbX1-pbX0, 70)
}

func (r *prender) headline() {
	c, d := r.c, r.d
	sc := 3
	if paperTextW(d.head)*3 > 920 {
		sc = 2
	}
	c.textC(d.head, 480, 86+(46-pInkH*sc)/2, sc, paperRed)
	r.block("headline", pbX0, 84, pbX1-pbX0, 46)
	c.textC(d.dek, 480, 133, 1, paperInk)
	r.block("dek", pbX0, 130, pbX1-pbX0, 20)
	c.fill(pbX0, 148, pbX1-pbX0, 1, paperInk)
	c.fill(pbX0, 151, pbX1-pbX0, 2, paperInk)
}

func (r *prender) footer(state int) {
	c, d := r.c, r.d
	c.fill(pbX0, 484, pbX1-pbX0, 2, paperInk)
	c.fill(pbX0, 487, pbX1-pbX0, 1, paperInk)
	y := 494
	c.bold(d.hint, pbX0+2, y, paperInk)
	for _, l := range wrapPx(d.hint2, 290) {
		y += pRowH
		c.text(l, pbX0+2, y, 1, paperInk)
	}
	y = 494
	for _, l := range wrapPx(d.foot, 290) {
		c.textR(l, pbX1-2, y, 1, paperInk)
		y += pRowH
	}
	bg := paperInk
	if state == 1 {
		bg = paperRed
	}
	c.fill(pbBannerX, pbBannerY, pbBannerW, pbBannerH, bg)
	c.frame(pbBannerX+3, pbBannerY+3, pbBannerW-6, pbBannerH-6, paperCream)
	for _, dx := range []int{7, pbBannerW - 9} {
		c.fill(pbBannerX+dx, pbBannerY+pbBannerH/2-1, 2, 2, paperCream)
	}
	tw := paperTextW(d.banner) * 2
	c.textOpt(d.banner, 480-tw/2, pbBannerY+(pbBannerH-24)/2, 2, paperCream, true, false)
	c.lay.banner = prect{"banner", pbBannerX, pbBannerY, pbBannerW, pbBannerH}
	r.block("footer", pbX0, 484, pbX1-pbX0, 44)
}

// renderPaper paints the whole page; state (0 / 1) is the banner's colour (it alternates twice a second).
func renderPaper(d *paperData, state int) (*image.RGBA, *paperLayout) {
	lay := &paperLayout{}
	c := newPaperCanvas(d.seed, lay)
	r := &prender{c: c, d: d, used: make([]bool, len(d.briefs))}
	r.masthead()
	r.headline()
	r.body()
	r.footer(state)
	return c.im, lay
}

// ---- the Game side ---------------------------------------------------------------------------------------------

var paperC paperCache // the page painted for the party on screen (one at a time)

type paperCache struct {
	key string
	tex [2]*ebiten.Image
	lay *paperLayout
}

func (g *Game) drawPaper(dst *ebiten.Image) {
	d := g.paperDataNow()
	key := fmt.Sprint(d.seed, d.title, d.head, d.stats, d.score, d.total, d.hint, d.two)
	pc := &paperC
	if pc.key != key {
		for _, t := range pc.tex {
			if t != nil {
				t.Deallocate()
			}
		}
		*pc = paperCache{key: key}
	}
	st := int(g.time*4) & 1 // two states, two changes a second
	if pc.tex[st] == nil {
		im, lay := renderPaper(d, st)
		pc.tex[st] = ebiten.NewImageFromImage(im)
		if pc.lay == nil {
			pc.lay = lay
		}
	}
	dst.DrawImage(pc.tex[st], nil)
	if textRec != nil {
		// the page is one opaque full-screen picture: whatever was drawn before it (the HUD of the party) is hidden
		textRec.fills = append(textRec.fills, panelRect{0, 0, screenW, screenH, true, len(textRec.texts)})
		for _, t := range pc.lay.texts {
			textRec.texts = append(textRec.texts, textRect{"drawText", t.s, float64(t.x), float64(t.y), float64(t.w), float64(t.h), float64(t.scale), screenW, screenH, dst.Bounds(), false})
		}
	}
}

// levelDoneContinue: "press to continue" on the newspaper (the only end screen) goes straight on to what came after the
// old results page: the bank / next party / interlude, or the stealth exit.
func (g *Game) levelDoneContinue() {
	if g.stealth {
		g.stealthExit()
		return
	}
	g.afterLevel()
}
