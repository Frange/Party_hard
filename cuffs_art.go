package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// THE HANDCUFFS SYMBOL: one drawing for every place the game shows them (the HUD "wanted" icon, the mark over a
// suspect / prisoner, the career page...). Two grey-blue steel rings with a light edge, a dark navy outline, a clasp
// with a lime square on each ring, a short blue chain with links and a small gold tip on top.
// cuffsPix(tier) is the ONLY function that draws them; the tier only changes the size (tiny = HUD strip, small = the mark over
// heads and the career page) and simplifies the detail (a 1 px steel rim at the small size).
// Sprites are built once per tier and always drawn at scale 1 or a whole multiple, so the pixels stay sharp.
// ---------------------------------------------------------------------------

const (
	cuffsTiny = iota // the HUD strip: 15x11, the least that stays readable
	cuffsSmall
	cuffsTiers
)

var (
	cuffNavy  = rgb(0x14204a)
	cuffSteel = [3]color.RGBA{rgb(0xdcecfb), rgb(0x9cb8d6), rgb(0x56708a)} // light, mid, shade
	cuffShine = rgb(0xffffff)
	cuffLime  = rgb(0xa8e82a)
	cuffChain = [2]color.RGBA{rgb(0x6c8cf0), rgb(0x2c4cb4)}
	cuffGold  = rgb(0xf0b830)
)

// cuffsGeom: ring outer diameter, hole diameter, clasp inner size, chain rise (px above the clasp box).
type cuffsGeom struct{ d, hole, cw, ch, rise int }

var cuffsGeoms = [cuffsTiers]cuffsGeom{
	cuffsTiny:  {8, 2, 1, 1, 0},
	cuffsSmall: {8, 2, 1, 1, 3},
}

// cuffsSize: pixel size (w, h) of a tier.
func cuffsSize(tier int) (int, int) {
	g := cuffsGeoms[tier]
	return 2*g.d - 1, g.rise + g.ch + 3 + g.d - 1
}

var cuffsCache [cuffsTiers]*pixBuf

// cuffsPix returns the shared handcuffs sprite of a tier (do not modify it).
func cuffsPix(tier int) *pixBuf {
	if tier < 0 || tier >= cuffsTiers {
		tier = cuffsSmall
	}
	if cuffsCache[tier] == nil {
		cuffsCache[tier] = buildCuffs(cuffsGeoms[tier])
	}
	return cuffsCache[tier]
}

func buildCuffs(g cuffsGeom) *pixBuf {
	w, h := 2*g.d-1, g.rise+g.ch+3+g.d-1
	b := newPix(w, h)
	y0 := h - g.d // top row of the rings
	// --- rings ---
	r := float64(g.d) / 2
	c0 := float64(g.d-1) / 2
	ring := func(x, y, ox int) (in, hole bool, dx, dy float64) {
		dx, dy = float64(x-ox)-c0, float64(y-y0)-c0
		d := math.Hypot(dx, dy)
		return d <= r-0.05, d < float64(g.hole)/2+0.01, dx, dy
	}
	isRing := func(x, y int) bool {
		for _, ox := range []int{0, g.d - 1} {
			if in, hole, _, _ := ring(x, y, ox); in && !hole {
				return true
			}
		}
		return false
	}
	for y := y0; y < h; y++ {
		for x := 0; x < w; x++ {
			if !isRing(x, y) {
				continue
			}
			edge := !isRing(x-1, y) || !isRing(x+1, y) || !isRing(x, y-1) || !isRing(x, y+1)
			ox := 0
			if x > g.d-1 {
				ox = g.d - 1
			}
			_, _, dx, dy := ring(x, y, ox)
			col := cuffSteel[1]
			switch s := dx + dy; {
			case s < -float64(g.d)*0.12:
				col = cuffSteel[0]
			case s > float64(g.d)*0.2:
				col = cuffSteel[2]
			}
			if edge {
				col = cuffNavy
			}
			b.set(x, y, col)
		}
	}
	// a bright spot on the upper left of each ring (not at the small size: the rim is only 1 px)
	if g.d >= 12 {
		for _, ox := range []int{0, g.d - 1} {
			for k := 0; k < g.d/6; k++ {
				p := b.get(ox+g.d/4+k, y0+2)
				if p.A != 0 && p != cuffNavy {
					b.set(ox+g.d/4+k, y0+2, cuffShine)
				}
			}
		}
	}
	// --- clasps (navy box, steel, lime square) ---
	cx := [2]int{(g.d - g.cw) / 2, g.d - 1 + (g.d-g.cw)/2}
	boxTop := y0 - g.ch - 1
	for _, x := range cx {
		b.rect(x-1, boxTop, g.cw+2, g.ch+3, cuffNavy)
		b.rect(x, boxTop+1, g.cw, g.ch+1, cuffSteel[1])
		b.rect(x, boxTop+1, g.cw, g.ch, cuffLime)
		if g.ch >= 2 {
			b.rect(x, boxTop+1, g.cw, 1, shade(cuffLime, 1.2))
		}
	}
	// --- chain: two blue strands rising to a gold tip in the middle ---
	mid := g.d - 1
	var chain []image2
	for i, x := range cx {
		sx := x + g.cw/2
		sy := boxTop - 1
		ex, ey := mid, 0
		n := int(math.Max(math.Abs(float64(ex-sx)), float64(sy-ey)))
		for k := 0; k <= n; k++ {
			t := float64(k) / float64(n)
			px := int(math.Round(float64(sx) + float64(ex-sx)*t))
			py := int(math.Round(float64(sy) + float64(ey-sy)*t))
			chain = append(chain, image2{px, py, (k + i) % 2})
		}
	}
	gw := 1 + g.d/10
	tips := [][2]int{}
	for k := 0; k < max(1, g.rise-1) && k < 2+g.d/8; k++ {
		for dx := 0; dx < gw; dx++ {
			tips = append(tips, [2]int{mid - gw/2 + dx, k})
		}
	}
	// outline of the chain and tip (outwards only)
	for _, c := range chain {
		b.setIfEmpty(c.x, c.y, cuffChain[c.k])
	}
	for _, t := range tips {
		b.set(t[0], t[1], cuffGold)
	}
	out := newPix(w, h)
	copy(out.p, b.p)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if b.get(x, y).A != 0 || y > y0 {
				continue
			}
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				p := b.get(x+d[0], y+d[1])
				if p.A != 0 && (p == cuffGold || p == cuffChain[0] || p == cuffChain[1]) {
					out.set(x, y, cuffNavy)
					break
				}
			}
		}
	}
	return out
}

type image2 struct{ x, y, k int }

func (b *pixBuf) setIfEmpty(x, y int, c color.RGBA) {
	if b.get(x, y).A == 0 {
		b.set(x, y, c)
	}
}

// drawCuffsIcon paints the symbol at whole-pixel (x, y), top left, with a whole scale k >= 1.
func drawCuffsIcon(dst *ebiten.Image, tier int, x, y float64, k int) {
	im := cuffsImage(tier)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(k), float64(k))
	op.GeoM.Translate(math.Round(x), math.Round(y))
	dst.DrawImage(im, op)
}

var cuffsImgs [cuffsTiers]*ebiten.Image

func cuffsImage(tier int) *ebiten.Image {
	if cuffsImgs[tier] == nil {
		cuffsImgs[tier] = cuffsPix(tier).img()
	}
	return cuffsImgs[tier]
}
