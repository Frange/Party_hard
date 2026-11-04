package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/rand"
	"os"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// charBase is the image row just below the soles of a character sprite.
const charBase = charFeet + 1

// world (design) coordinates -> native screen pixels
func (g *Game) wx(x float64) float64 { return math.Round(x*wk) + g.camI.X }
func (g *Game) wy(y float64) float64 { return math.Round(y*wk) + g.camI.Y }

// sr converts a design rectangle to native pixels with consistent rounded edges.
func sr(r Rect) Rect {
	x0, y0 := math.Round(r.X*wk), math.Round(r.Y*wk)
	x1, y1 := math.Round((r.X+r.W)*wk), math.Round((r.Y+r.H)*wk)
	return Rect{x0, y0, x1 - x0, y1 - y0}
}

func drawTiled(dst *ebiten.Image, texs []*ebiten.Image, r Rect) {
	const t = 32
	for ty := math.Floor(r.Y/t) * t; ty < r.Y+r.H; ty += t {
		for tx := math.Floor(r.X/t) * t; tx < r.X+r.W; tx += t {
			x0, y0 := math.Max(tx, r.X), math.Max(ty, r.Y)
			x1, y1 := math.Min(tx+t, r.X+r.W), math.Min(ty+t, r.Y+r.H)
			if x1 <= x0 || y1 <= y0 {
				continue
			}
			ix, iy := int(tx/t), int(ty/t)
			tex := texs[int((uint32(ix)*73856093^uint32(iy)*19349663)>>7)%len(texs)]
			sub := tex.SubImage(image.Rect(int(x0-tx), int(y0-ty), int(x1-tx), int(y1-ty))).(*ebiten.Image)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(x0, y0)
			dst.DrawImage(sub, op)
		}
	}
}

// filledEllipse draws a solid ellipse directly into a screen-space image
// (unlike pixBuf.ellipse, which works on the CPU sprite buffers).
func filledEllipse(dst *ebiten.Image, cx, cy, rx, ry float64, col color.RGBA) {
	for y := -ry; y <= ry; y++ {
		dyr := y / ry
		w := rx * math.Sqrt(math.Max(0, 1-dyr*dyr))
		fillRect(dst, cx-w, cy+y, 2*w, 1, col)
	}
}

// ringAt draws a thin ring (a circle outline), scanline by scanline, so it
// only touches the pixels of the ring itself and leaves whatever is behind
// it — window, bed, text — untouched everywhere else.
func ringAt(dst *ebiten.Image, cx, cy, r, thick float64, col color.RGBA) {
	ir := r - thick
	for yy := -r; yy <= r; yy++ {
		outer := math.Sqrt(math.Max(0, r*r-yy*yy))
		if math.Abs(yy) < ir {
			inner := math.Sqrt(ir*ir - yy*yy)
			fillRect(dst, cx-outer, cy+yy, outer-inner, 1, col)
			fillRect(dst, cx+inner, cy+yy, outer-inner, 1, col)
		} else {
			fillRect(dst, cx-outer, cy+yy, 2*outer, 1, col)
		}
	}
}

// gradText draws text with a left-to-right colour gradient, one glyph
// at a time, for a title treatment closer to the original game's logo.
func (g *Game) gradText(dst *ebiten.Image, s string, cx, y float64, colA, colB, outline color.RGBA, scale float64) {
	runes := []rune(s)
	w := g.textW(s, scale)
	cur := math.Round(cx - w/2)
	n := len(runes)
	for i, r := range runes {
		ch := string(r)
		t := 0.0
		if n > 1 {
			t = float64(i) / float64(n-1)
		}
		col := mixc(colA, colB, t)
		for _, o := range [][2]float64{{-1, 0}, {1, 0}, {0, -1}, {0, 1}, {2, 2}} {
			op := &text.DrawOptions{}
			op.GeoM.Scale(scale, scale)
			op.GeoM.Translate(cur+o[0]*scale*0.5, y+o[1]*scale*0.5)
			op.ColorScale.ScaleWithColor(outline)
			text.Draw(dst, ch, g.face, op)
		}
		op := &text.DrawOptions{}
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(cur, y)
		op.ColorScale.ScaleWithColor(col)
		text.Draw(dst, ch, g.face, op)
		cw, _ := text.Measure(ch, g.face, 0)
		cur += cw * scale
	}
}

func (g *Game) bigText(dst *ebiten.Image, s string, cx, y float64, col, outline color.RGBA, scale float64) {
	s = tr(s)
	w := g.textW(s, scale)
	x := math.Round(cx - w/2)
	for _, o := range [][2]float64{{-1, 0}, {1, 0}, {0, -1}, {0, 1}, {2, 2}} {
		op := &text.DrawOptions{}
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(x+o[0]*scale*0.5, y+o[1]*scale*0.5)
		op.ColorScale.ScaleWithColor(outline)
		text.Draw(dst, s, g.face, op)
	}
	op := &text.DrawOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(col)
	text.Draw(dst, s, g.face, op)
}

func (g *Game) renderBG() {
	l := g.lvl
	def := l.Def
	if def.BGImg == nil && def.PaintBG != nil {
		def.BGImg = def.PaintBG()
	}
	if def.BGImg != nil {
		l.bg = def.BGImg
		if !def.roadsDrawn {
			def.roadsDrawn = true
			drawRoads(def.BGImg, def)
		}
		l.graded = true
		return
	}
	bg := ebiten.NewImage(screenW, screenH)
	bg.Fill(rgb(0x0a0912))
	rng := rand.New(rand.NewSource(int64(len(def.Title)*131 + 7)))
	W, H := float64(screenW), float64(screenH)

	switch def.Sky {
	case "city":
		for y := 0.0; y < H; y += 4 {
			t := y / H
			fillRect(bg, 0, y, W, 4, mixc(rgb(0x070a1c), rgb(0x141c3c), 1-math.Abs(t-0.5)*1.4))
		}
		for x := 0.0; x < W; x += 6 + rng.Float64()*18 {
			h := 18 + rng.Float64()*44
			w := 14 + rng.Float64()*24
			fillRect(bg, x, 50-h, w, h, rgb(0x0c1226))
			for wy := 50 - h + 4; wy < 45; wy += 6 {
				for wx := x + 3; wx < x+w-3; wx += 6 {
					if rng.Intn(3) == 0 {
						fillRect(bg, wx, wy, 2, 3, rgb(0xf0c850))
					}
				}
			}
		}
		for x := 0.0; x < W; x += 8 + rng.Float64()*20 {
			fillRect(bg, x, 504, 12+rng.Float64()*18, 8+rng.Float64()*20, rgb(0x0c1226))
		}
		for x := 40.0; x < 900; x += 150 {
			fillRect(bg, x, 506, 100, 28, rgb(0x14304a))
			fillRect(bg, x, 506, 100, 3, rgb(0x2a5a8a))
		}
	case "sea":
		for y := 0.0; y < 40; y += 2 {
			fillRect(bg, 0, y, W, 2, mixc(rgb(0x3a1a5a), rgb(0xb0406a), y/40))
		}
		for y := 40.0; y < 92; y += 2 {
			fillRect(bg, 0, y, W, 2, mixc(rgb(0x241a48), rgb(0x1a1236), (y-40)/52))
		}
		for yy := 0.0; yy < 36; yy++ {
			w := math.Sqrt(math.Max(0, 36*36-(36-yy)*(36-yy)))
			fillRect(bg, 780-w, yy, 2*w, 1, rgb(0xf08a30))
		}
		for i := 0; i < 20; i++ {
			fillRect(bg, 700+float64(i%5)*10, 46+float64(i)*2.4, 90-float64(i)*3, 1, rgb(0xe8884a))
		}
		fillRect(bg, 0, 38, W, 2, rgb(0xc85a7a))
		fillRect(bg, 420, 28, 40, 8, rgb(0x14101e))
		fillRect(bg, 432, 22, 10, 6, rgb(0x14101e))
	case "space":
		bg.Fill(rgb(0x02030a))
		for i := 0; i < 520; i++ {
			x, y := rng.Float64()*W, rng.Float64()*H
			s := 1.0
			if rng.Float64() < 0.2 {
				s = 2
			}
			fillRect(bg, x, y, s, s, color.RGBA{220, 220, 240, uint8(140 + rng.Intn(100))})
		}
		for i, pl := range [][3]float64{{110, 96, 58}, {850, 450, 66}} {
			px, py, pr := pl[0], pl[1], pl[2]
			pcol := []uint32{0xc06a40, 0x3a6ab8}[i%2]
			for yy := -pr; yy < pr; yy++ {
				w := math.Sqrt(math.Max(0, pr*pr-yy*yy))
				fillRect(bg, px-w, py+yy, 2*w, 1, shade(rgb(pcol), 1-yy/pr*0.35))
			}
			for k := 0; k < 14; k++ {
				a, b := rng.Float64()*2*math.Pi, rng.Float64()*pr*0.8
				cx, cy := px+math.Cos(a)*b, py+math.Sin(a)*b
				fillRect(bg, cx-5, cy-1, 12, 3, shade(rgb(pcol), 1.25))
			}
		}
	}

	for _, f := range def.Floors {
		r := sr(f.R)
		switch f.Kind {
		case "dance":
			fillRect(bg, r.X, r.Y, r.W, r.H, rgb(0x15121f))
		case "dash":
			for x := r.X; x < r.X+r.W; x += 27 {
				fillRect(bg, x, r.Y, 15, math.Max(2, r.H*0.7), rgb(0xd8c860))
			}
		case "rug":
			col := shade(desat(rgb(f.Color), 0.3), 0.85)
			fillRect(bg, r.X, r.Y, r.W, r.H, shade(col, 0.7))
			fillRect(bg, r.X+2, r.Y+2, r.W-4, r.H-4, col)
			fillRect(bg, r.X+5, r.Y+5, r.W-10, r.H-10, shade(col, 0.85))
			for x := r.X; x < r.X+r.W; x += 4 {
				fillRect(bg, x, r.Y-1, 1, 1, shade(col, 1.2))
				fillRect(bg, x, r.Y+r.H, 1, 1, shade(col, 1.2))
			}
		case "helipad":
			cx, cy := r.Center().X, r.Center().Y
			rad := r.W / 2
			col := rgb(f.Color)
			fillRect(bg, r.X, r.Y, r.W, r.H, rgb(0x1a2030))
			for a := 0.0; a < 2*math.Pi; a += 0.06 {
				for k := 0.0; k < 2; k++ {
					fillRect(bg, math.Round(cx+math.Cos(a)*(rad-3-k)), math.Round(cy+math.Sin(a)*(rad-3-k)), 1, 1, col)
				}
			}
			s := rad * 0.42
			fillRect(bg, cx-s, cy-s, 3, 2*s, col)
			fillRect(bg, cx+s-3, cy-s, 3, 2*s, col)
			fillRect(bg, cx-s, cy-1, 2*s, 3, col)
		default:
			drawTiled(bg, g.as.floors[f.Kind], r)
			g.groundDetail(bg, rng, f)
		}
	}
	g.litter(bg, rng)
	g.floorShading(bg, rng)
	for _, p := range l.Pools {
		pr := sr(p)
		c := Rect{pr.X - 8, pr.Y - 8, pr.W + 16, pr.H + 16}
		fillRect(bg, c.X, c.Y, c.W, c.H, rgb(0x1e4aa8))
		for y := c.Y; y < c.Y+c.H; y += 4 {
			for x := c.X; x < c.X+c.W; x += 4 {
				if pr.Has(x+1, y+1) {
					continue
				}
				col := rgb(0x2a5ac8)
				if (int((x-c.X)/4)+int((y-c.Y)/4))%2 == 0 {
					col = rgb(0x8ab8e8)
				}
				fillRect(bg, x, y, 4, 4, col)
			}
		}
		for i := 0.0; i < 4; i++ {
			fillRect(bg, pr.X+pr.W/2-16+i*8, pr.Y-8, 8, 4, rgb(0xeef4fa))
		}
		fillRect(bg, pr.X-1, pr.Y-1, pr.W+2, pr.H+2, rgb(0x16306a))
		fillRect(bg, pr.X, pr.Y, pr.W, pr.H, rgb(0x2f8fcf))
	}
	for _, w := range l.Walls {
		if w.Style != WallVoid {
			wallShadow(bg, sr(w.R), w.Style)
		}
	}
	for _, w := range l.Walls {
		if w.Style != WallVoid {
			drawWall(bg, sr(w.R), w.Style)
		}
	}
	for _, dr := range def.Doors {
		drawDoor(bg, dr)
	}
	drawRoads(bg, def)
	g.drawHull(bg, def)
	g.drawSign(bg, def)
	l.bg = bg
	l.graded = false
}

// floorShading is what keeps a floor from reading as a flat fill: every room
// gets ambient occlusion along its edges, a soft centre, and large patches of
// wear and dirt, so the tiles underneath stop looking like a repeated
// pattern. Outdoor ground gets the patches only.
func (g *Game) floorShading(bg *ebiten.Image, rng *rand.Rand) {
	def := g.lvl.Def
	clip := func(r Rect, x, y, w, h float64, c color.RGBA) {
		x0, y0 := math.Max(x, r.X), math.Max(y, r.Y)
		x1, y1 := math.Min(x+w, r.X+r.W), math.Min(y+h, r.Y+r.H)
		if x1 > x0 && y1 > y0 {
			fillRect(bg, x0, y0, x1-x0, y1-y0, c)
		}
	}
	blob := func(r Rect, cx, cy, rx, ry float64, c color.RGBA) {
		for y := -ry; y <= ry; y++ {
			d := y / ry
			w := rx * math.Sqrt(math.Max(0, 1-d*d))
			clip(r, cx-w, cy+y, 2*w, 1, c)
		}
	}
	for _, f := range def.Floors {
		switch f.Kind {
		case "dance", "dash", "rug", "helipad":
			continue
		}
		r := sr(f.R)
		indoor := floorIndoor(f.Kind)
		if indoor {
			// edge occlusion: 16px fading bands on all four sides
			for i := 0.0; i < 16; i++ {
				a := uint8(72 * (1 - i/16) * (1 - i/16))
				c := color.RGBA{0, 0, 0, a}
				clip(r, r.X, r.Y+i, r.W, 1, c)
				clip(r, r.X, r.Y+r.H-1-i, r.W, 1, c)
				clip(r, r.X+i, r.Y, 1, r.H, c)
				clip(r, r.X+r.W-1-i, r.Y, 1, r.H, c)
			}
			// a soft lit centre
			blob(r, r.X+r.W/2, r.Y+r.H/2, r.W*0.42, r.H*0.42, color.RGBA{14, 12, 8, 14})
		}
		// wear and dirt: a handful of wide, faint patches per rect
		n := int(r.W*r.H/14000) + 2
		for i := 0; i < n; i++ {
			cx, cy := r.X+rng.Float64()*r.W, r.Y+rng.Float64()*r.H
			rx, ry := 18+rng.Float64()*40, 10+rng.Float64()*24
			if rng.Intn(3) == 0 {
				a := uint8(6 + rng.Intn(6))
				blob(r, cx, cy, rx, ry, color.RGBA{a, a, a, a})
			} else {
				blob(r, cx, cy, rx, ry, color.RGBA{0, 0, 0, uint8(12 + rng.Intn(12))})
			}
		}
	}
}

// gradeBG is a final colour grade on the baked background: a touch more
// contrast and a fine grain, so flat areas stop looking painted-on.
func gradeBG(bg *ebiten.Image, rng *rand.Rand) {
	w, h := bg.Bounds().Dx(), bg.Bounds().Dy()
	pix := make([]byte, w*h*4)
	bg.ReadPixels(pix)
	var lut [256]byte
	for i := range lut {
		v := float64(i)/255 - 0.5
		v = v * 1.14
		lut[i] = byte(clampF((v+0.5)*255-6, 0, 255))
	}
	for i := 0; i < len(pix); i += 4 {
		if pix[i+3] == 0 {
			continue
		}
		n := rng.Intn(9) - 4
		for k := 0; k < 3; k++ {
			v := int(lut[pix[i+k]]) + n
			if v < 0 {
				v = 0
			} else if v > 255 {
				v = 255
			}
			pix[i+k] = byte(v)
		}
	}
	bg.WritePixels(pix)
}

// drawSign paints a small neon wall sign (a glow backing plus gradient
// text), the same gradient treatment as the main menu's title, baked into
// the static background so guests walk in front of it like any other wall
// decoration.
func (g *Game) drawSign(dst *ebiten.Image, def *LevelDef) {
	if def.Sign == "" {
		return
	}
	p := def.SignPos
	w := g.textW(def.Sign, 2) + 16
	fillRect(dst, p.X-w/2, p.Y-12, w, 24, rgba(0x1a0a2a, 150))
	fillRect(dst, p.X-w/2, p.Y-12, w, 2, rgba(0xff3070, 90))
	fillRect(dst, p.X-w/2, p.Y+10, w, 2, rgba(0x9040e0, 90))
	g.gradText(dst, def.Sign, p.X, p.Y-8, rgb(0xff3070), rgb(0x9040e0), rgb(0x300050), 2)
}

// drawHull paints a dark hull band with lit portholes around the outside of
// HullRect (a ship-shaped venue's railing boundary), so it reads as a boat
// seen from above instead of a floating platform.
func (g *Game) drawHull(dst *ebiten.Image, def *LevelDef) {
	if !def.Hull {
		return
	}
	if def.Shape != nil {
		if def.Sky == "sea" {
			g.paintSea(dst, def.Shape)
		}
		g.drawShapeHull(dst, def)
		return
	}
	r := def.HullRect
	const band = 16.0
	hull := rgb(0x0e2440)
	fillRect(dst, r.X-band, r.Y-band, r.W+2*band, band, hull)
	fillRect(dst, r.X-band, r.Y+r.H, r.W+2*band, band, hull)
	fillRect(dst, r.X-band, r.Y, band, r.H, hull)
	fillRect(dst, r.X+r.W, r.Y, band, r.H, hull)
	fillRect(dst, r.X-band, r.Y-band, r.W+2*band, 2, rgb(0x1c4068))
	fillRect(dst, r.X-band, r.Y+r.H+band-2, r.W+2*band, 2, rgb(0x081830))
	fillRect(dst, r.X-band, r.Y-band, 2, r.H+2*band, rgb(0x1c4068))
	fillRect(dst, r.X+r.W+band-2, r.Y-band, 2, r.H+2*band, rgb(0x081830))
	pcol := rgb(0xf0d060)
	for x := r.X + 24; x < r.X+r.W-20; x += 36 {
		fillRect(dst, x, r.Y-band/2-2, 5, 5, pcol)
		fillRect(dst, x, r.Y+r.H+band/2-2, 5, 5, pcol)
	}
	if def.HullName != "" {
		g.drawTextC(dst, def.HullName, r.X+r.W/2, r.Y+r.H+band+3, rgb(0x6a86a8), 1)
	}
}

func drawWall(dst *ebiten.Image, r Rect, style int) {
	horiz := r.W >= r.H
	var capC, faceC color.RGBA
	switch style {
	case WallBrick:
		capC, faceC = rgb(0x1c1416), rgb(0x6a3e34)
	case WallClub:
		capC, faceC = rgb(0x14101c), rgb(0x2c2142)
	case WallHedge:
		capC, faceC = rgb(0x1e5028), rgb(0x2e7a3a)
	case WallFence:
		capC, faceC = rgb(0x6a4e2a), rgb(0x9c7a46)
	case WallRoof:
		capC, faceC = rgb(0x3a3c4a), rgb(0x161820)
	case WallWood:
		capC, faceC = rgb(0x3a2410), rgb(0x8a5e30)
	case WallRail:
		capC, faceC = rgb(0x7a7e88), rgb(0x3a3c44)
	default:
		capC, faceC = rgb(0x1e1a1e), rgb(0x4a3a30)
	}
	if horiz {
		capH := math.Min(5, math.Max(2, r.H*0.35))
		if style == WallFence || style == WallHedge {
			capH = 2
		}
		fillRect(dst, r.X, r.Y, r.W, capH, shade(capC, 1.25))
		fillRect(dst, r.X, r.Y, r.W, 1, shade(capC, 1.7))
		fillRect(dst, r.X, r.Y+capH, r.W, r.H-capH, shade(faceC, 0.9))
		fh := r.H - capH
		switch style {
		case WallBrick:
			for y := 0.0; y < fh; y += 3 {
				fillRect(dst, r.X, r.Y+capH+y, r.W, 1, shade(faceC, 0.8))
				off := 0.0
				if int(y/3)%2 == 1 {
					off = 3
				}
				for x := off; x < r.W; x += 6 {
					fillRect(dst, r.X+x, r.Y+capH+y, 1, 3, shade(faceC, 0.8))
				}
			}
		case WallClub:
			fillRect(dst, r.X, r.Y+r.H-2, r.W, 1, rgb(0xff2090))
			fillRect(dst, r.X, r.Y+r.H-1, r.W, 1, rgb(0x20c0ff))
		case WallRoof:
			fillRect(dst, r.X, r.Y+capH, r.W, 1, rgb(0x6a6e80))
			fillRect(dst, r.X, r.Y+r.H-1, r.W, 1, rgb(0x2a2c3a))
		case WallRail:
			// A thin, pale railing: two rails and evenly spaced posts, instead
			// of the solid parapet look, so it reads as something you can see
			// past rather than a wall.
			fillRect(dst, r.X, r.Y+capH, r.W, 1, rgb(0xd8dce4))
			fillRect(dst, r.X, r.Y+r.H-1, r.W, 1, rgb(0xb0b4bc))
			for x := 0.0; x < r.W; x += 9 {
				fillRect(dst, r.X+x, r.Y+capH, 2, fh, rgb(0xc4c8d0))
			}
		case WallWood:
			for x := 0.0; x < r.W; x += 4 {
				fillRect(dst, r.X+x, r.Y+capH, 1, fh, shade(faceC, 0.7))
			}
			fillRect(dst, r.X, r.Y+r.H-2, r.W, 2, shade(faceC, 0.55))
			fillRect(dst, r.X, r.Y+capH, r.W, 1, shade(faceC, 1.3))
		case WallFence:
			for x := 0.0; x < r.W; x += 4 {
				fillRect(dst, r.X+x, r.Y+capH, 1, fh, shade(faceC, 0.72))
			}
			fillRect(dst, r.X, r.Y+capH+1, r.W, 1, shade(faceC, 0.6))
		case WallHedge:
			for x := 0.0; x < r.W; x += 4 {
				for y := 0.0; y < fh; y += 3 {
					if int(x/4+y/3)%2 == 0 {
						fillRect(dst, r.X+x, r.Y+capH+y, 1, 1, shade(faceC, 1.3))
					}
				}
			}
		default:
			for x := 0.0; x < r.W; x += 4 {
				fillRect(dst, r.X+x, r.Y+capH, 1, fh, shade(faceC, 0.8))
			}
			fillRect(dst, r.X, r.Y+r.H-2, r.W, 2, shade(faceC, 0.6))
			fillRect(dst, r.X, r.Y+capH, r.W, 1, shade(faceC, 1.3))
		}
		if (style == WallPlaster || style == WallWood) && fh >= 9 {
			drawPictures(dst, r, capH, style)
		}
		if style == WallBrick && r.H > 26 {
			for x := 7.0; x+9 < r.W; x += 26 {
				fillRect(dst, r.X+x, r.Y+capH+7, 9, 10, rgb(0x2a3a58))
				fillRect(dst, r.X+x, r.Y+capH+7, 9, 2, rgb(0x6a86b0))
			}
		}
		if style != WallFence && style != WallHedge && style != WallRail {
			fillRect(dst, r.X, r.Y+r.H-2, r.W, 2, rgba(0x000000, 70))
		}
		return
	}
	fillRect(dst, r.X, r.Y, r.W, r.H, capC)
	fillRect(dst, r.X, r.Y, 1, r.H, shade(capC, 1.45))
	fillRect(dst, r.X+r.W-1, r.Y, 1, r.H, shade(capC, 0.7))
	switch style {
	case WallFence:
		fillRect(dst, r.X+1, r.Y, r.W-2, r.H, faceC)
		for y := 0.0; y < r.H; y += 4 {
			fillRect(dst, r.X+1, r.Y+y, r.W-2, 1, shade(faceC, 0.7))
		}
	case WallHedge:
		fillRect(dst, r.X, r.Y, r.W, r.H, faceC)
		for y := 0.0; y < r.H; y += 3 {
			fillRect(dst, r.X+1, r.Y+y, 1, 1, shade(faceC, 1.3))
		}
	case WallClub:
		fillRect(dst, r.X+r.W/2, r.Y, 1, r.H, rgb(0xff2090))
	case WallBrick:
		fillRect(dst, r.X+1, r.Y, r.W-2, r.H, faceC)
		// lit windows along the run, matching the horizontal case above —
		// this is what exterior house walls were missing them on, since
		// their side walls are built as vertical segments.
		if r.H > 60 {
			for y := 10.0; y+12 < r.H; y += 34 {
				fillRect(dst, r.X+1, r.Y+y, r.W-2, 12, rgb(0x2a3a58))
				fillRect(dst, r.X+1, r.Y+y, r.W-2, 2, rgb(0x6a86b0))
			}
		}
	case WallRoof:
		fillRect(dst, r.X+1, r.Y, r.W-2, r.H, rgb(0x2a2c3a))
		fillRect(dst, r.X, r.Y, 1, r.H, rgb(0x6a6e80))
	case WallRail:
		fillRect(dst, r.X+1, r.Y, r.W-2, r.H, rgb(0x3a3c44))
		fillRect(dst, r.X, r.Y, 1, r.H, rgb(0xd8dce4))
		fillRect(dst, r.X+r.W-1, r.Y, 1, r.H, rgb(0xb0b4bc))
		for y := 0.0; y < r.H; y += 9 {
			fillRect(dst, r.X+1, r.Y+y, r.W-2, 2, rgb(0xc4c8d0))
		}
	case WallWood:
		fillRect(dst, r.X+1, r.Y, r.W-2, r.H, faceC)
		for y := 0.0; y < r.H; y += 5 {
			fillRect(dst, r.X+1, r.Y+y, r.W-2, 1, shade(faceC, 0.7))
		}
	}
	fillRect(dst, r.X+r.W, r.Y, 2, r.H, color.RGBA{0, 0, 0, 60})
}

// ---------------------------------------------------------------------------

type drawItem struct {
	y  float64
	fn func()
}

func (g *Game) Layout(w, h int) (int, int) {
	if g.mode == ModeEditor {
		return edW, edH
	}
	return screenW, screenH
}

func (g *Game) drawImg(dst, im *ebiten.Image, x, y float64, ox, oy float64) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(math.Round(x-ox), math.Round(y-oy))
	dst.DrawImage(im, op)
}

// drawActor draws a character animation frame with feet at logic position pos.
func (g *Game) drawActor(dst *ebiten.Image, cs *CharSet, anim int, phase float64, pos Vec, dir Dir, yoff float64, swim bool) {
	cx, cy := g.wx(pos.X), g.wy(pos.Y)
	sc := g.charSc
	if sc <= 0 {
		sc = 1
	}
	if !swim {
		fillShadow(dst, cx, cy, sc)
	}
	img, flip := cs.img(anim, dir, phase)
	op := &ebiten.DrawImageOptions{}
	if sc != 1 {
		op.Filter = ebiten.FilterLinear // smaller people: smooth the shrinking
	}
	if g.zombieTint {
		op.ColorScale.Scale(0.62, 1, 0.66, 1)
	}
	if swim {
		img = img.SubImage(image.Rect(0, 0, charW, 30)).(*ebiten.Image)
		yoff += 12
	}
	if flip {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(charW, 0)
	}
	op.GeoM.Translate(-charCX, -charBase+yoff)
	if sc != 1 {
		op.GeoM.Scale(sc, sc)
	}
	op.GeoM.Translate(cx, cy)
	dst.DrawImage(img, op)
	if swim {
		for i := 0; i < 2; i++ {
			a := uint8(110 - 40*i)
			w := 16 + float64(i)*5 + math.Sin(g.time*4+pos.X)*1.5
			fillRect(dst, cx-w/2, cy-10+float64(i)*2, w, 1, color.RGBA{a, a, a, a})
		}
	}
}

func (g *Game) drawLying(dst *ebiten.Image, cs *CharSet, sleep bool, pos Vec) {
	im := cs.lying(sleep)
	w, h := im.Bounds().Dx(), im.Bounds().Dy()
	cx, cy := g.wx(pos.X), g.wy(pos.Y)
	fillRect(dst, cx-float64(w)/2+3, cy-2, float64(w)-6, 3, color.RGBA{0, 0, 0, 50})
	g.drawImg(dst, im, cx, cy, float64(w)/2, float64(h)-2)
}

func fillShadow(dst *ebiten.Image, cx, cy float64, k ...float64) {
	s := 1.0
	if len(k) > 0 {
		s = k[0]
	}
	for i, w := range []float64{15 * s, 10 * s} {
		a := uint8(40 + i*30)
		fillRect(dst, cx-w/2, cy-1+float64(i), w, 1, color.RGBA{0, 0, 0, a})
		fillRect(dst, cx-w/2+1, cy-2+float64(i)*2, w-2, 1, color.RGBA{0, 0, 0, a})
	}
}

func (g *Game) drawRing(dst *ebiten.Image, pos Vec) {
	cx, cy := g.wx(pos.X), g.wy(pos.Y)-1
	for a := 0.0; a < 2*math.Pi; a += 0.2 {
		fillRect(dst, math.Round(cx+math.Cos(a)*13), math.Round(cy+math.Sin(a)*5), 1, 1, color.RGBA{235, 235, 245, 235})
	}
}

// drawBubbleBox paints one bubble chosen by the manager (see bubbles.go).
func (g *Game) drawBubbleBox(dst *ebiten.Image, r bubReq) {
	w := r.w
	cx := r.cx
	x := clampF(cx-math.Round(w/2), 2, screenW-w-2)
	y := r.y - r.shift
	col := rgb(0x8a3ae0)
	if r.red {
		col = rgb(0xc42a2a)
	}
	h := r.h
	if h == 0 {
		h = 13
	}
	fillRect(dst, x+1, y, w-2, h, col)
	fillRect(dst, x, y+1, w, h-2, col)
	fillRect(dst, x+2, y+1, w-4, 1, shade(col, 1.35))
	if r.shift == 0 {
		fillRect(dst, cx-2, y+h, 4, 1, col)
		fillRect(dst, cx-1, y+h+1, 2, 1, col)
	}
	for i, l := range r.lines {
		lw := math.Round(g.textW(l, 1))
		g.drawText(dst, l, x+math.Round((w-lw)/2), y+float64(i)*bubLineH, rgb(0xffffff), 1)
	}
}

func (g *Game) glowAt(dst *ebiten.Image, wp Vec, r float64, col color.RGBA) {
	op := &ebiten.DrawImageOptions{}
	s := r * wk / 48
	op.GeoM.Translate(-48, -48)
	op.GeoM.Scale(s, s)
	op.GeoM.Translate(g.wx(wp.X), g.wy(wp.Y))
	op.Blend = ebiten.BlendLighter
	op.ColorScale.ScaleWithColor(col)
	dst.DrawImage(g.as.glow, op)
}

// --- animation selection --------------------------------------------------------

func (g *Game) guestPose(gu *Guest) (int, float64) {
	if gu.quirk != "" && !gu.moving && (gu.state == gIdle || gu.state == gDance || gu.state == gChat || gu.state == gDanceWith) {
		if a, ph, ok := g.quirkPose(gu); ok {
			return a, ph
		}
	}
	t := g.time
	switch gu.state {
	case gZombie:
		return AnimZombie, gu.anim
	case gLift:
		return AnimPanic, t*6 + gu.off
	}
	if gu.stabT > 0 {
		return AnimStab, math.Min(2, (0.25-gu.stabT)/0.25*3)
	}
	if gu.poison > 0 && !gu.moving && (gu.state == gIdle || gu.state == gLounge) {
		return AnimTired, t*2 + gu.off // dizzy, leaning on nothing
	}
	if gu.moving {
		switch {
		case gu.state == gFlee || gu.state == gRun || gu.state == gScared:
			return AnimPanic, gu.anim
		case gu.curSpd > 60:
			return AnimRun, gu.anim
		}
		return AnimWalk, gu.anim
	}
	switch gu.state {
	case gDance, gDanceWith:
		return AnimDance0 + gu.style%4, t*7 + gu.off
	case gDrink:
		return AnimDrink, (gu.dur-gu.timer)*2.2 + 0.01
	case gLounge:
		return AnimDrink, t*1.1 + gu.off
	case gChat:
		if int(t/2.2)%2 == gu.role {
			return AnimTalk, t*6 + gu.off
		}
	case gKiss:
		return AnimKiss, t * 1.5
	case gVomit:
		return AnimVomit, t * 5
	case gAlarm:
		return AnimWave, t * 9
	case gCall:
		return AnimCall, t * 1.5
	case gFun:
		switch gu.fun {
		case 0:
			return AnimWave, t*3 + gu.off
		case 1:
			return AnimGuitar, t*7 + gu.off
		case 2:
			return AnimDrum, t*9 + gu.off
		case 4:
			return AnimDance0 + gu.style%4, t*9 + gu.off
		case 5:
			return AnimLaugh, t*8 + gu.off
		case 6:
			return AnimPirouette, t*7 + gu.off
		case 7:
			return AnimPoint, t*4 + gu.off
		case 8:
			return AnimDance4, t*9 + gu.off
		case 9:
			return AnimDance5, t*8 + gu.off
		case 10:
			return AnimLove, t*5 + gu.off
		case 11:
			return AnimPee, t*4 + gu.off
		case 12:
			return AnimWave, t*7 + gu.off
		case 13:
			return AnimCheer, t * 5
		case 14:
			return AnimPanic, t * 7
		case 17, 18:
			return gu.dAnim, t*6 + gu.off
		case 15:
			if gu.rcv {
				return AnimWave, t * 6
			}
			return AnimLaugh, t*5 + gu.off
		}
	case gToilet:
		if gu.inside == nil && gu.queue == nil {
			return AnimPee, t*3 + gu.off
		}
	case gPerform:
		switch gu.perf {
		case "dj":
			return AnimDJ, gu.anim
		case "drum":
			return AnimDrum, gu.anim * 1.4
		case "guitar":
			return AnimGuitar, gu.anim * 0.8
		case "sing":
			return AnimSing, gu.anim * 0.8
		}
	}
	return AnimIdle, t*1.3 + gu.off
}

func (g *Game) guardPose(gd *Guard) (int, float64) {
	t := g.time
	switch {
	case gd.state == gdBeat:
		return AnimPunch, t * 9
	case gd.callT > 0:
		return AnimCall, t * 1.5
	case gd.moving && gd.state == gdChase:
		return AnimRun, gd.anim
	case gd.moving:
		return AnimWalk, gd.anim
	}
	return AnimIdle, t*1.2 + gd.pos.X
}

func (g *Game) copPose(c *Cop) (int, float64) {
	t := g.time
	switch {
	case c.state == cBag || (c.state == cBagHere && !c.moving):
		return AnimCrouch, t * 2
	case c.state == cTired:
		return AnimTired, t * 2
	case c.moving && c.state == cChase:
		return AnimRun, c.anim
	case c.moving && c.carry != nil:
		return AnimCarry, c.anim
	case c.moving:
		return AnimWalk, c.anim
	}
	return AnimIdle, t*1.2 + c.pos.X
}

func (g *Game) playerPose() (int, float64) {
	p := g.p
	t := g.time
	switch {
	case p.stabT > 0:
		return AnimStab, math.Min(2, (0.18-p.stabT)/0.18*3)
	case p.danceT > 0:
		return AnimDance0 + p.style%4, t * 7
	case p.moving && p.carrying != nil:
		return AnimCarry, p.anim
	case p.moving && p.running:
		return AnimRun, p.anim
	case p.moving:
		return AnimWalk, p.anim
	case p.carrying != nil:
		return AnimCarry, 0
	}
	return AnimIdle, t * 1.3
}

func (g *Game) bodyLyingImg(b *Body) *ebiten.Image {
	if b.bagged {
		return g.as.bag
	}
	return b.cs.lying(false)
}

func (g *Game) drawBody(dst *ebiten.Image, b *Body) {
	if b.bagged {
		w, h := g.as.bag.Bounds().Dx(), g.as.bag.Bounds().Dy()
		g.drawImg(dst, g.as.bag, g.wx(b.pos.X), g.wy(b.pos.Y), float64(w)/2, float64(h)-1)
		return
	}
	if b.skeleton {
		g.drawSkeletonBody(dst, b)
		return
	}
	if b.fallT > 0 {
		f := math.Min(2, (0.45-b.fallT)/0.45*3)
		g.drawActor(dst, b.cs, AnimFall, f, b.pos, b.dir, 0, false)
		return
	}
	g.drawLying(dst, b.cs, false, b.pos)
}

func (g *Game) drawSiren(dst *ebiten.Image, pos Vec) {
	x, y := g.wx(pos.X), g.wy(pos.Y)-22+math.Round(math.Sin(g.time*6))
	red := int(g.time*6)%2 == 0
	fillRect(dst, x-3, y+2, 6, 2, rgb(0x2a2a34))
	c1, c2 := rgb(0xff3030), rgb(0x3060ff)
	if red {
		c1, c2 = c2, c1
	}
	fillRect(dst, x-2, y, 2, 3, c1)
	fillRect(dst, x, y, 2, 3, c2)
}

func (g *Game) drawCuffs(dst *ebiten.Image, pos Vec) {
	x, y := g.wx(pos.X), g.wy(pos.Y)-54+math.Round(math.Sin(g.time*5))
	g.drawImg(dst, g.as.cuffs, x, y, 7, 4)
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.pickLanguage()
	if g.mode == ModeEditor {
		g.drawEditor(screen)
		if g.shot != "" && g.tick == g.shotTick {
			g.saveShot(screen)
		}
		return
	}
	screen.Fill(rgb(0x08060c))
	if g.mode == ModeEnding || g.mode == ModeIntro || g.mode == ModeInterlude {
		if g.mode == ModeIntro {
			g.drawIntro(screen)
		} else if g.mode == ModeInterlude {
			g.drawInterlude(screen)
		} else {
			g.drawEnding(screen)
		}
		if g.shot != "" && g.tick == g.shotTick {
			g.saveShot(screen)
		}
		return
	}
	g.camI = Vec{}
	if g.shake > 0 {
		k := math.Min(1, g.shake*3) * 3
		g.camI = Vec{math.Round((g.rng.Float64() - 0.5) * k), math.Round((g.rng.Float64() - 0.5) * k)}
	}
	l := g.lvl
	if !l.graded && l.Def.BGImg == nil {
		gradeBG(l.bg, rand.New(rand.NewSource(int64(len(l.Def.Title)*131+7))))
		l.graded = true
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(g.camI.X, g.camI.Y)
	screen.DrawImage(l.bg, op)
	g.drawAnimatedFloors(screen)

	for _, d := range g.decals {
		x, y := g.wx(d.pos.X), g.wy(d.pos.Y)
		col := color.RGBA{110, 14, 14, 200}
		if d.kind == 1 {
			col = color.RGBA{10, 10, 12, 190}
		}
		if d.kind == 2 {
			col = color.RGBA{90, 130, 30, 200}
		}
		if d.kind == 3 {
			col = color.RGBA{214, 200, 60, 150}
		}
		if d.kind == 4 {
			col = color.RGBA{96, 58, 24, 230}
		}
		r := d.r * wk
		for i := 0; i < 5; i++ {
			a := float64(i) * 1.7
			rr := r * (0.45 + 0.12*float64(i))
			fillRect(screen, x+math.Cos(a)*r*0.35-rr/2, y-2+math.Sin(a)*r*0.2-rr*0.2, rr, rr*0.45, col)
		}
	}

	var items []drawItem
	for _, pr := range l.Props {
		pr := pr
		items = append(items, drawItem{pr.Sort, func() { g.drawProp(screen, pr) }})
	}
	for _, b := range g.bodies {
		b := b
		if b.carried {
			continue
		}
		items = append(items, drawItem{b.pos.Y - 4, func() { g.drawBody(screen, b) }})
	}
	for _, gu := range g.guests {
		gu := gu
		if gu.out() {
			continue
		}
		items = append(items, drawItem{gu.pos.Y, func() {
			if gu.state == gToilet && gu.inside != nil {
				return // behind the door
			}
			if gu.state == gSleep || (gu.state == gFun && (gu.fun == 16 || gu.fun == 3 && gu.timer < 2.4)) {
				g.drawLying(screen, gu.cs, true, gu.pos)
				return
			}
			if kidRiding(gu) {
				g.drawSkater(screen, gu)
				return
			}
			anim, phase := g.guestPose(gu)
			swim := g.lvl.InPool(gu.pos)
			pos := gu.pos
			yoff := 0.0
			if swim {
				yoff = math.Round(math.Sin(g.time*2 + gu.off))
			}
			if gu.state == gLift {
				yoff = -gu.lift
			}
			g.zombieTint = gu.state == gZombie
			defer func() { g.zombieTint = false }()
			if gu.stabT > 0 {
				v := dirVec(gu.dir)
				pos.X += v.X * 3
				pos.Y += v.Y * 3
			}
			pos.X += poisonedStagger(gu, g.time)
			g.drawActor(screen, gu.cs, anim, phase, pos, gu.dir, yoff, swim)
		}})
	}
	for _, gd := range g.guards {
		gd := gd
		items = append(items, drawItem{gd.pos.Y, func() {
			if gd.dead {
				return
			}
			// A faint halo drawn just before the body, so the opaque sprite
			// covers its center and only a subtle rim of color shows through.
			g.glowAt(screen, Vec{gd.pos.X, gd.pos.Y - 20}, 16, color.RGBA{230, 140, 30, 42})
			anim, phase := g.guardPose(gd)
			g.drawActor(screen, gd.cs, anim, phase, gd.pos, gd.dir, 0, false)
		}})
	}
	for _, e := range g.elevators {
		e := e
		items = append(items, drawItem{e.pos.Y - 1, func() { g.drawElevator(screen, e) }})
	}
	if s := g.slasher; s != nil && (s.state == slHunt || s.state == slCuffed) {
		items = append(items, drawItem{s.pos.Y, func() { g.drawSlasher(screen, s) }})
	}
	for _, c := range g.cops {
		c := c
		items = append(items, drawItem{c.pos.Y, func() {
			if c.state != cLeave {
				g.glowAt(screen, Vec{c.pos.X, c.pos.Y - 20}, 17, g.copHaloCol(c))
			}
			cs := g.copCSFor(c)
			anim, phase := g.copPose(c)
			g.drawActor(screen, cs, anim, phase, c.pos, c.dir, 0, g.lvl.InPool(c.pos) && c.carry == nil)
			if c.carry != nil {
				bi := g.bodyLyingImg(c.carry)
				w, h := bi.Bounds().Dx(), bi.Bounds().Dy()
				g.drawImg(screen, bi, g.wx(c.pos.X), g.wy(c.pos.Y)-32, float64(w)/2, float64(h))
			}
		}})
	}
	for _, rc := range g.roadCars {
		rc := rc
		s := g.lvl.Def.Roads[rc.seg]
		y := rc.at(s).Y
		if s.horizontal() {
			y += 8
		}
		items = append(items, drawItem{y, func() { g.drawRoadCar(screen, rc) }})
	}
	if tr := g.train; tr != nil {
		items = append(items, drawItem{tr.trackY + 12, func() { g.drawTrain(screen) }})
	}
	for _, w := range g.wrecks {
		w := w
		items = append(items, drawItem{w.Y, func() {
			pi := g.as.props["car"][4]
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(math.Round(g.wx(w.X)-pi.ox), math.Round(g.wy(w.Y)-pi.oy))
			op.ColorScale.Scale(0.15, 0.13, 0.13, 1)
			screen.DrawImage(pi.img, op)
			if !g.wreckOut[w] {
				g.drawFlames(screen, g.wx(w.X), g.wy(w.Y)-14, 22)
			}
		}})
	}
	for _, v := range g.veh {
		v := v
		if v.isHeli() {
			continue
		}
		items = append(items, drawItem{v.y, func() {
			if g.lvl.Def.Transit != "" {
				g.drawTransit(screen, v)
				return
			}
			if v.kind == 7 {
				g.drawFireTruck(screen, v)
				return
			}
			if v.kind == 8 {
				g.drawTechVan(screen, v)
				return
			}
			k := 4
			if v.kind == 1 {
				k = 5
			}
			pi := g.as.props["car"][k]
			if v.kind == 4 {
				g.drawHearse(screen, pi, v)
				return
			}
			if v.kind == 6 {
				g.drawLimo(screen, pi, v)
				return
			}
			if v.unit > 0 {
				g.drawUnitCar(screen, pi, v)
				return
			}
			g.drawImg(screen, pi.img, g.wx(v.x), g.wy(v.y), pi.ox, pi.oy)
		}})
	}
	for _, a := range g.amb {
		a := a
		items = append(items, drawItem{a.y, func() { g.drawAmbient(screen, a) }})
	}
	for _, fm := range g.fireCrew {
		fm := fm
		items = append(items, drawItem{fm.pos.Y, func() { g.drawFireman(screen, fm) }})
	}
	for _, tc := range g.techCrew {
		tc := tc
		items = append(items, drawItem{tc.pos.Y, func() { g.drawTechnician(screen, tc) }})
	}
	for _, an := range g.animals {
		an := an
		items = append(items, drawItem{an.pos.Y, func() { g.drawAnimal(screen, an) }})
	}
	items = g.jurassicItems(screen, items)
	p := g.p
	if g.mode != ModeMenu && g.mode != ModeLevelSelect && g.mode != ModeSettings && g.mode != ModeShop && g.mode != ModeConfirm && g.mode != ModeTrophies && g.mode != ModeScenes && g.mode != ModeDebug && !p.hidden {
		items = append(items, drawItem{p.pos.Y + 0.2, func() {
			if p.dead || p.knockT > 0 {
				g.drawLying(screen, p.cs, false, p.pos)
				if p.knockT > 0 && !p.dead {
					for i := 0; i < 3; i++ {
						a := g.time*6 + float64(i)*2.1
						fillRect(screen, g.wx(p.pos.X)+math.Cos(a)*8-1, g.wy(p.pos.Y)-18+math.Sin(a)*3, 3, 3, rgb(0xf8e040))
					}
				}
				return
			}
			g.drawRing(screen, p.pos)
			anim, phase := g.playerPose()
			pos := p.pos
			if p.stabT > 0 {
				v := dirVec(p.dir)
				pos.X += v.X * 3
				pos.Y += v.Y * 3
			}
			g.drawActor(screen, p.cs, anim, phase, pos, p.dir, 0, l.InPool(p.pos) && p.carrying == nil)
			g.drawHalo(screen)
			if p.carrying != nil {
				bi := g.bodyLyingImg(p.carrying)
				w, h := bi.Bounds().Dx(), bi.Bounds().Dy()
				g.drawImg(screen, bi, g.wx(p.pos.X), g.wy(p.pos.Y)-38, float64(w)/2, float64(h))
			}
		}})
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].y < items[j].y })
	for _, it := range items {
		it.fn()
	}
	for _, b := range g.bodies {
		if b.reported && !b.carried && !b.bagged {
			g.drawSiren(screen, b.pos)
		}
	}
	for _, v := range g.veh {
		if v.isHeli() {
			g.drawHeli(screen, v)
		}
	}
	for _, a := range g.amb {
		if a.kind == ambUFO {
			g.drawAmbient(screen, a)
		}
	}
	g.drawEvents(screen)
	if g.mode == ModePlay && g.wanted > 0 && !p.hidden && !p.dead && !g.otherCuffed() {
		g.drawCuffs(screen, p.pos)
	}

	for _, gu := range g.guests {
		if !gu.out() && gu.icon != "" {
			pri := bubGuest
			if gu.icon == "!!!" {
				pri = bubAlarm
			}
			g.bubble(gu, pri, gu.pos, gu.icon, gu.icon == "!!!", float64(gu.cs.Spec.Stature)*-0.6)
		}
	}
	for _, gd := range g.guards {
		if !gd.dead && gd.icon != "" {
			g.bubble(gd, bubAuth, gd.pos, gd.icon, true, 0)
		}
	}
	for _, c := range g.cops {
		if c.icon != "" {
			g.bubble(c, bubAuth, c.pos, c.icon, true, 0)
		}
	}
	if s := g.slasher; s != nil && s.icon != "" {
		g.bubble(s, bubAuth, s.pos, s.icon, true, -6)
	}
	g.flushBubbles(screen)
	for _, pt := range g.parts {
		a := pt.life / pt.max
		c := pt.col
		c = color.RGBA{uint8(float64(c.R) * a), uint8(float64(c.G) * a), uint8(float64(c.B) * a), uint8(255 * a)}
		fillRect(screen, g.wx(pt.pos.X), g.wy(pt.pos.Y), math.Max(1, math.Round(pt.size*0.7)), math.Max(1, math.Round(pt.size*0.7)), c)
	}

	g.drawLighting(screen)
	g.drawCrossingLights(screen)
	g.drawEventsTop(screen)
	g.drawPhoneMarkers(screen)
	g.drawTechMarker(screen)
	g.drawBandSign(screen)
	g.drawHoverName(screen)
	g.drawHUD(screen)
	switch g.mode {
	case ModeMenu:
		g.drawMenu(screen)
	case ModeLevelSelect:
		g.drawLevelSelect(screen)
	case ModeTrophies:
		g.drawTrophies(screen)
	case ModeScenes:
		g.drawScenes(screen)
	case ModeConfirm:
		g.drawConfirm(screen)
	case ModeShop:
		g.drawShop(screen)
	case ModeSettings:
		g.drawSettings(screen)
	case ModeDebug:
		g.drawDebug(screen)
	case ModeLevelDone:
		g.drawPanel(screen, "¡NIVEL COMPLETADO!", "No ha quedado nadie con vida en la fiesta...", rgb(0x60ff80),
			fmt.Sprintf("Puntos: %d   Tiempo: %d:%02d", g.points, int(g.endTime)/60, int(g.endTime)%60), "ENTER: siguiente fiesta")
	case ModeFinal:
		g.drawPanel(screen, "¡HAS ACABADO CON TODAS LAS FIESTAS!", "La ciudad por fin está en silencio.", rgb(0xf0e060),
			fmt.Sprintf("Puntos del último nivel: %d", g.points), "ENTER: volver al menú")
	case ModeLose:
		if g.time-g.endAt > 0.6 {
			g.drawPanel(screen, "FIN DEL JUEGO", g.loseReason, rgb(0xff5050),
				fmt.Sprintf("Víctimas: %d de %d   Puntos: %d", g.total-g.aliveCount(), g.total, g.points), "R / ENTER: reintentar    ESC: menú")
		}
	}
	g.drawTrophyHud(screen)
	g.drawToast(screen)
	if g.shot != "" && g.tick == g.shotTick {
		g.saveShot(screen)
	}
}

func (g *Game) drawAmbient(dst *ebiten.Image, a *Ambient) {
	x, y := g.wx(a.x), g.wy(a.y)
	flip := a.vx > 0
	if a.v0 != 0 {
		flip = a.v0 > 0
	}
	switch a.kind {
	case ambCar, ambBike:
		name, v := "car", a.col
		if a.kind == ambBike {
			name, v = "motorbike", a.col%2
		}
		pi := g.as.props[name][v%len(g.as.props[name])]
		op := &ebiten.DrawImageOptions{}
		if vehMirror(name, flip) {
			op.GeoM.Scale(-1, 1)
			op.GeoM.Translate(float64(pi.img.Bounds().Dx()), 0)
			op.GeoM.Translate(math.Round(x-(float64(pi.img.Bounds().Dx())-pi.ox)), math.Round(y-pi.oy))
		} else {
			op.GeoM.Translate(math.Round(x-pi.ox), math.Round(y-pi.oy))
		}
		dst.DrawImage(pi.img, op)
	case ambTank:
		im := g.as.tank
		w, h := float64(im.Bounds().Dx()), float64(im.Bounds().Dy())
		op := &ebiten.DrawImageOptions{}
		if flip {
			op.GeoM.Scale(-1, 1)
			op.GeoM.Translate(w, 0)
		}
		op.GeoM.Translate(math.Round(x-w/2), math.Round(y-h))
		dst.DrawImage(im, op)
	case ambHorse:
		pi := g.as.props["horse"][0]
		bob := math.Round(math.Sin(a.phase * 12))
		op := &ebiten.DrawImageOptions{}
		w := float64(pi.img.Bounds().Dx())
		if !flip {
			op.GeoM.Scale(-1, 1)
			op.GeoM.Translate(w, 0)
		}
		op.GeoM.Translate(math.Round(x-w/2), math.Round(y-pi.oy+bob))
		dst.DrawImage(pi.img, op)
		dir := DirLeft
		if flip {
			dir = DirRight
		}
		g.drawActor(dst, g.as.cowCS, AnimIdle, 0, Vec{a.x, a.y - 16}, dir, bob, false)
	case ambObj:
		k := propKinds[a.name]
		imgs := g.as.props[a.name]
		if k == nil || len(imgs) == 0 {
			return
		}
		v := 0
		if k.FPS > 0 {
			v = int(a.phase*k.FPS) % len(imgs)
		} else if len(imgs) > 1 {
			v = a.col % len(imgs)
		}
		pi := imgs[v]
		op := &ebiten.DrawImageOptions{}
		w := float64(pi.img.Bounds().Dx())
		y -= a.hop
		if vehMirror(a.name, flip) {
			op.GeoM.Scale(-1, 1)
			op.GeoM.Translate(w, 0)
			op.GeoM.Translate(math.Round(x-(w-pi.ox)), math.Round(y-pi.oy))
		} else {
			op.GeoM.Translate(math.Round(x-pi.ox), math.Round(y-pi.oy))
		}
		dst.DrawImage(pi.img, op)
	case ambMeteor:
		dir := -1.0
		if flip {
			dir = 1
		}
		for i := 0; i < 6; i++ {
			fx := x - dir*float64(i)*5
			fy := y - float64(i)*2
			a := uint8(230 - i*32)
			fillRect(dst, fx-1, fy-1, 3, 3, color.RGBA{255, 220, 140, a})
		}
		fillRect(dst, x-1, y-1, 3, 3, color.RGBA{255, 255, 240, 255})
	case ambUFO:
		im := g.as.ufo
		w, h := float64(im.Bounds().Dx()), float64(im.Bounds().Dy())
		yy := y + math.Round(math.Sin(a.phase*2)*3)
		g.drawImg(dst, im, x, yy, w/2, h/2)
		fillRect(dst, x-5, yy+h/2, 10, 3, color.RGBA{60, 90, 70, 60})
	case ambTow:
		pi := towSide()
		op := &ebiten.DrawImageOptions{}
		if a.loaded {
			// the wrecked car rides on the flat bed, at the back of the truck (opposite its travel)
			bx := x + 12
			if flip {
				bx = x - 12
			}
			wi := g.as.props["car"][4]
			bop := &ebiten.DrawImageOptions{}
			bop.ColorScale.Scale(0.3, 0.26, 0.26, 1)
			bop.GeoM.Translate(math.Round(bx-wi.ox), math.Round(y-wi.oy-9))
			dst.DrawImage(wi.img, bop)
		}
		if flip {
			op.GeoM.Scale(-1, 1)
			op.GeoM.Translate(float64(pi.img.Bounds().Dx()), 0)
			op.GeoM.Translate(math.Round(x-(float64(pi.img.Bounds().Dx())-pi.ox)), math.Round(y-pi.oy))
		} else {
			op.GeoM.Translate(math.Round(x-pi.ox), math.Round(y-pi.oy))
		}
		dst.DrawImage(pi.img, op)
	}
}

var shotZoom int
var shotCrop [4]int

func (g *Game) saveShot(screen *ebiten.Image) {
	b := screen.Bounds()
	pix := make([]byte, 4*b.Dx()*b.Dy())
	screen.ReadPixels(pix)
	im := image.NewRGBA(b)
	copy(im.Pix, pix)
	var out image.Image = im
	if shotZoom > 1 {
		x0, y0, w, h := shotCrop[0], shotCrop[1], shotCrop[2], shotCrop[3]
		z := shotZoom
		zi := image.NewRGBA(image.Rect(0, 0, w*z, h*z))
		for y := 0; y < h*z; y++ {
			for x := 0; x < w*z; x++ {
				zi.SetRGBA(x, y, im.RGBAAt(x0+x/z, y0+y/z))
			}
		}
		out = zi
	}
	f, err := os.Create(g.shot)
	if err == nil {
		png.Encode(f, out)
		f.Close()
	}
	os.Exit(0)
}

func (g *Game) drawHeli(dst *ebiten.Image, v *Vehicle) {
	cx, cy := g.wx(v.x), g.wy(v.y)
	alt := math.Max(0, v.ty-v.y)
	sw := (40 - math.Min(alt, 120)*0.15) * wk * 1.4
	fillRect(dst, cx-sw/2, g.wy(v.ty)+4, sw, 5, color.RGBA{0, 0, 0, 70})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(math.Round(cx-g.as.heliOX), math.Round(cy-g.as.heliOY))
	img := g.as.heli
	view, front := false, false
	if g.as.heliU[0] != nil { // the stock helicopter: coming down or going up it is seen from the front / the rear
		if fl, fr := heliFlying(v); fl {
			view, front = true, fr
			img = heliViewImg(heliLiveryIdx(v), front)
			op.GeoM.Reset()
			op.GeoM.Translate(math.Round(cx-(heliViewCX*wk+1)), math.Round(cy-(heliViewCY*wk+1)))
		}
	}
	if view {
		// livery already in the view
	} else if u := g.heliImg(v); u != nil {
		img = u
	} else if v.kind == 3 {
		op.ColorScale.Scale(1, 0.92, 0.92, 1)
	}
	if v.kind == 5 {
		op.ColorScale.Scale(0.3, 0.26, 0.34, 1)
	}
	if v.kind == 2 && g.heliImg(v) == nil && !view {
		switch v.unit {
		case unitFBI:
			op.ColorScale.Scale(0.2, 0.22, 0.3, 1)
		case unitMilitary:
			op.ColorScale.Scale(0.22, 0.32, 0.2, 1)
		case unitSWAT:
			op.ColorScale.Scale(0.35, 0.37, 0.42, 1)
		case unitRangers:
			op.ColorScale.Scale(0.4, 0.22, 0.3, 1)
		}
	}
	dst.DrawImage(img, op)
	if v.kind == 3 && g.heliImg(v) == nil && !view {
		fillRect(dst, math.Round(cx-5), math.Round(cy-5), 7, 2, rgb(0xd02828))
		fillRect(dst, math.Round(cx-3), math.Round(cy-7), 3, 6, rgb(0xd02828))
	}
	rc := Vec{cx, cy - 25*wk}
	for _, a := range []float64{0, 1.6, 3.2} {
		for k := 0; k < 2; k++ {
			ang := v.rot*0.6 + a + float64(k)*math.Pi/2
			rop := &ebiten.DrawImageOptions{}
			rop.GeoM.Scale(58*wk, 2)
			rop.GeoM.Translate(-29*wk, -1)
			rop.GeoM.Rotate(ang)
			rop.GeoM.Translate(rc.X, rc.Y)
			rop.ColorScale.ScaleWithColor(color.RGBA{40, 40, 46, 70})
			dst.DrawImage(px, rop)
		}
	}
	tail := Vec{cx + 48*wk, cy - 16*wk}
	if view {
		tail = Vec{cx, cy - 18*wk} // from the rear the tail rotor hangs at the top of the fin
	}
	tp := &ebiten.DrawImageOptions{}
	tp.GeoM.Scale(1.5, 9)
	tp.GeoM.Translate(-0.75, -4.5)
	tp.GeoM.Rotate(v.rot * 0.9)
	tp.GeoM.Translate(tail.X, tail.Y)
	tp.ColorScale.ScaleWithColor(color.RGBA{60, 60, 70, 110})
	dst.DrawImage(px, tp)
	if v.state != 1 || int(g.time*6)%2 == 0 {
		g.glowAt(dst, Vec{v.x + 20, v.y - 6}, 26, color.RGBA{190, 20, 20, 130})
	} else {
		g.glowAt(dst, Vec{v.x - 20, v.y - 6}, 26, color.RGBA{30, 60, 210, 130})
	}
}

func (g *Game) drawProp(dst *ebiten.Image, pr *Prop) {
	k := pr.K
	if pr.gone {
		return
	}
	if k.Hide && !g.p.hidden && dist(g.p.pos, Vec{pr.X, pr.Y}) < 46 {
		// A faint cue that this is somewhere you can duck into, visible only
		// when you are close enough for it to matter.
		g.glowAt(dst, Vec{pr.X, pr.Y - 10}, 16, color.RGBA{120, 220, 230, 55})
	}
	v := pr.Var
	if k.Poison && pr.armT > 0 {
		v = 1
	}
	if pr.Name == "murray" {
		v = g.murrayTalk(pr)
	}
	if pr.Name == "hen" && k.Native == nil {
		v = (int(g.time*2.5) + int(pr.X)) % 2
	}
	if k.FPS > 0 && k.Native != nil {
		v = (int(g.time*k.FPS) + int(pr.X)) % len(k.Native)
	}
	imgs := g.as.props[pr.Name]
	pi := imgs[v%len(imgs)]
	x, y := g.wx(pr.X), g.wy(pr.Y)
	if !k.NoSolid && k.FW > 0 {
		// contact shadow: a soft dark ellipse under the footprint
		rx, ry := k.FW/2+4, math.Max(3, k.FH*0.55+2)
		filledEllipse(dst, x, y-1, rx+2, ry+1, color.RGBA{0, 0, 0, 28})
		filledEllipse(dst, x, y-1, rx, ry, color.RGBA{0, 0, 0, 55})
	}
	op := &ebiten.DrawImageOptions{}
	sx := 0.0
	if p := g.p; p.hidden && p.hideProp == pr {
		sx = float64(g.tick/3%2)*2 - 1
	}
	if pr.shake > g.time {
		sx = float64(g.tick/2%2)*2 - 1
	}
	if pr.scale > 0 && pr.scale != 1 { // a smaller object (the island): shrink around the anchor
		op.Filter = ebiten.FilterLinear
		op.GeoM.Translate(-pi.ox, -pi.oy)
		op.GeoM.Scale(pr.scale, pr.scale)
		op.GeoM.Translate(math.Round(x+sx), math.Round(y))
	} else {
		op.GeoM.Translate(math.Round(x-pi.ox+sx), math.Round(y-pi.oy))
	}
	if pr.used && k.Trap != "" {
		op.ColorScale.Scale(0.35, 0.3, 0.3, 1)
	}
	if pr.used && k.Phone {
		op.ColorScale.Scale(0.6, 0.6, 0.6, 1)
	}
	// tall things turn see-through while somebody is standing behind them
	if pi.img.Bounds().Dy() >= 34 && g.mode == ModePlay {
		tgt, rate, amp := 0.0, 0.2, 0.55
		if pr.Name == "jp_gate" { // the gate of the island stays the main thing: only a faint see-through, gone at once
			rate, amp = 0.2, 0.22
		}
		if g.actorBehind(pr, pi) {
			tgt = 1
		} else if pr.Name == "jp_gate" {
			rate = 0.6
		}
		pr.fade += (tgt - pr.fade) * rate
		if pr.fade > 0.02 {
			op.ColorScale.ScaleAlpha(float32(1 - amp*pr.fade))
		}
	}
	if pr.wreck {
		op.ColorScale.Scale(0.16, 0.14, 0.14, 1)
	}
	if pr.armT > 0 && k.Trap != "" && int(pr.armT*6)%2 == 0 {
		op.ColorScale.Scale(1.6, 0.7, 0.7, 1)
	}
	if !(k.NoSolid && pr.Name != "tree") && pr.Name != "slide" && pr.Name != "stage" {
		fw, fh := pr.Foot.W*wk+3, math.Max(3, pr.Foot.H*wk*0.7)
		cxs, cys := g.wx(pr.Foot.X+pr.Foot.W/2)+1, g.wy(pr.Foot.Y+pr.Foot.H)
		switch pr.Name {
		case "tree":
			fw, fh, cxs = 30, 8, cxs+6
		case "palm":
			fw, fh, cxs = 18, 5, cxs+5
		case "umbrella":
			fw, fh, cxs = 24, 6, cxs+5
		}
		softShadow(dst, cxs, cys-fh/2+1, fw, fh, 85)
	}
	dst.DrawImage(pi.img, op)
	if pr.wreck && !pr.fireOut {
		g.drawFlames(dst, x, y-pi.oy*0.5, 18)
	}
	if k.Trap != "" && !pr.used && pr.armT == 0 && g.mode == ModePlay && distRect(g.p.pos, pr.Foot) < 40 {
		fillRect(dst, math.Round(x)-1, math.Round(y-pi.oy)-3+math.Sin(g.time*5), 3, 2, rgb(0xff4040))
	}
}

func (g *Game) drawAnimatedFloors(dst *ebiten.Image) {
	l := g.lvl
	def := l.Def
	beat := int(g.time * 4)
	// muted tones: a club floor under coloured light, not a rainbow
	pal := []uint32{0x8a2f5a, 0x2f4e8a, 0x2f7a62, 0x8a7a2f, 0x5e2f8a}
	if len(def.Strobe) > 0 {
		pal = def.Strobe
	}
	cell := def.CellSize
	if cell == 0 {
		cell = 16
	}
	for _, f := range def.Floors {
		if f.Kind != "dance" {
			continue
		}
		for cy := 0.0; cy < f.R.H; cy += cell {
			for cx := 0.0; cx < f.R.W; cx += cell {
				r := sr(Rect{f.R.X + cx, f.R.Y + cy, math.Min(cell, f.R.W-cx), math.Min(cell, f.R.H-cy)})
				i := (int(cx/cell) + int(cy/cell) + beat) % len(pal)
				col := rgb(pal[i])
				fillRect(dst, r.X+g.camI.X, r.Y+g.camI.Y, r.W, r.H, shade(col, 0.4))
				fillRect(dst, r.X+g.camI.X+1, r.Y+g.camI.Y+1, r.W-2, r.H-2, shade(col, 0.72))
				fillRect(dst, r.X+g.camI.X+1, r.Y+g.camI.Y+1, r.W-2, 1, shade(col, 0.95))
			}
		}
	}
	for _, p := range l.Pools {
		r := sr(p)
		x, y := r.X+g.camI.X, r.Y+g.camI.Y
		if !def.Custom {
			fillRect(dst, x, y, r.W, r.H, rgb(0x2a86c6))
			fillRect(dst, x, y, r.W, 4, rgb(0x1f6fa8))
		}
		for yy := 3.0; yy < r.H; yy += 5 {
			for xx := 0.0; xx < r.W; xx += 11 {
				x0 := xx + 4*math.Sin(g.time*1.4+yy*0.6+xx*0.1)
				if x0 < 0 || x0+5 > r.W {
					continue
				}
				fillRect(dst, x+x0, y+yy, 5, 1, rgb(0x6cc4f0))
				fillRect(dst, x+x0+1, y+yy+1, 3, 1, rgb(0x4aa8e0))
			}
		}
		if !def.Custom {
			fillRect(dst, x+r.W*0.62, y+r.H*0.3+math.Round(math.Sin(g.time*1.3)), 11, 3, rgb(0xe8c840))
			fillRect(dst, x+r.W*0.2, y+r.H*0.7+math.Round(math.Sin(g.time*1.1)), 9, 9, rgb(0xe04a4a))
			fillRect(dst, x+r.W*0.2+3, y+r.H*0.7+math.Round(math.Sin(g.time*1.1))+3, 3, 3, rgb(0x2a86c6))
		}
	}
}

// floorIndoor tells the lighting pass which floors are rooms: they stay in a
// visible half-light, while everything else (lawns, streets, sand) falls to
// near-black away from the lamps.
func floorIndoor(kind string) bool {
	switch kind {
	case "wood", "tile", "carpet", "parquet", "marble", "mall", "white", "concrete", "rug", "dance", "deck":
		return true
	}
	return false
}

// lightSrc is one light for the frame: it cuts a hole in the darkness layer
// and then adds its colour on top.
type lightSrc struct {
	pos Vec
	r   float64
	col color.RGBA
	cut float64 // how much darkness it removes at its centre, 0..1
}

// cutAt removes darkness from the layer with the same radial falloff as glowAt.
func (g *Game) cutAt(dark *ebiten.Image, wp Vec, r, k float64) {
	op := &ebiten.DrawImageOptions{}
	s := r * wk / 48
	op.GeoM.Translate(-48, -48)
	op.GeoM.Scale(s, s)
	op.GeoM.Translate(g.wx(wp.X), g.wy(wp.Y))
	op.Blend = ebiten.BlendDestinationOut
	op.ColorScale.Scale(float32(k), float32(k), float32(k), float32(k))
	dark.DrawImage(g.as.glow, op)
}

func cutRect(dark *ebiten.Image, x, y, w, h, k float64) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(w, h)
	op.GeoM.Translate(x, y)
	op.Blend = ebiten.BlendDestinationOut
	op.ColorScale.Scale(float32(k), float32(k), float32(k), float32(k))
	dark.DrawImage(px, op)
}

func (g *Game) drawLighting(dst *ebiten.Image) {
	def := g.lvl.Def
	t := def.Tint
	var lights []lightSrc
	for i, lp := range g.lvl.Lights {
		r, col := 110.0, color.RGBA{150, 118, 52, 120}
		if i < len(g.lvl.LightSrc) {
			if k := g.lvl.LightSrc[i]; k != nil && k.LightR > 0 {
				r, col = k.LightR, k.LightC
			}
		}
		lights = append(lights, lightSrc{lp, r, col, 0.95})
	}
	for _, gl := range def.Glows {
		k := 1.0
		if gl.Pulse {
			k = 0.75 + 0.25*math.Sin(g.time*3+gl.Pos.X)
		}
		c := gl.Col
		lights = append(lights, lightSrc{gl.Pos, gl.R * (0.9 + 0.1*k), color.RGBA{uint8(float64(c.R) * k), uint8(float64(c.G) * k), uint8(float64(c.B) * k), uint8(float64(c.A) * k)}, 0.8 * k})
	}
	if def.Disco {
		cols := []uint32{0xff2090, 0x20c0ff, 0xa040ff, 0x20ff90}
		for _, d := range def.Dance {
			c := d.Center()
			for i := 0; i < 3; i++ {
				a := g.time*0.8 + float64(i)*2.1
				pos := Vec{c.X + math.Cos(a)*d.W*0.35, c.Y + math.Sin(a*1.3)*d.H*0.35}
				col := rgb(cols[(i+int(g.time*0.7))%len(cols)])
				lights = append(lights, lightSrc{pos, 70, color.RGBA{col.R / 3, col.G / 3, col.B / 3, 90}, 0.5})
			}
		}
	}
	for _, v := range g.veh {
		if v.isHeli() || v.kind == 6 || v.kind == 8 || v.kind == 9 {
			continue
		}
		if v.kind == 4 {
			lights = append(lights, lightSrc{Vec{v.x, v.y - 20}, 60, color.RGBA{60, 200, 60, 120}, 0.6})
			continue
		}
		if int(g.time*6)%2 == 0 {
			lights = append(lights, lightSrc{Vec{v.x - 7, v.y - 32}, 50, color.RGBA{200, 20, 20, 160}, 0.5})
		} else {
			lights = append(lights, lightSrc{Vec{v.x + 7, v.y - 32}, 50, color.RGBA{30, 60, 220, 160}, 0.5})
		}
	}
	for _, b := range g.bodies {
		if b.reported && !b.carried && !b.bagged {
			col := color.RGBA{150, 20, 20, 120}
			if int(g.time*6)%2 == 1 {
				col = color.RGBA{30, 50, 170, 120}
			}
			lights = append(lights, lightSrc{Vec{b.pos.X, b.pos.Y - 18}, 34, col, 0.4})
		}
	}
	for _, pr := range g.lvl.Props {
		if pr.armT > 0 && pr.K.Trap != "" {
			lights = append(lights, lightSrc{Vec{pr.X, pr.Y - 14}, 30, color.RGBA{140, 60, 20, 120}, 0.4})
		}
	}

	// wider but softer lights: they reach further from their source and no longer blow out
	for i := range lights {
		ls := &lights[i]
		ls.r *= 1.6
		ls.col = color.RGBA{uint8(float64(ls.col.R) * 0.6), uint8(float64(ls.col.G) * 0.6), uint8(float64(ls.col.B) * 0.6), uint8(float64(ls.col.A) * 0.6)}
	}
	// 1. darkness: outdoors near-black, rooms in half-light, sky left alone,
	//    every light cutting a soft hole. Tint.A (already tuned per level)
	//    sets how deep the night is.
	if def.Day {
		// daylight / dusk: just the colour wash and the lights' glow, no night layer
		fillRect(dst, 0, 0, screenW, screenH, color.RGBA{uint8(float64(t.R) * 0.75), uint8(float64(t.G) * 0.75), uint8(float64(t.B) * 0.75), uint8(float64(t.A) * 0.75)})
		for _, ls := range lights {
			g.glowAt(dst, ls.pos, ls.r, ls.col)
		}
		g.finishLighting(dst)
		return
	}
	dark := g.as.dark
	aOut := clampF(float64(t.A)*1.2, 70, 112) // night, but still playable everywhere
	aIn := aOut * 0.3
	base := mixc(rgb(0x14142a), color.RGBA{t.R, t.G, t.B, 255}, 0.5)
	a := uint8(aOut)
	dark.Fill(color.RGBA{uint8(float64(base.R) * aOut / 255), uint8(float64(base.G) * aOut / 255), uint8(float64(base.B) * aOut / 255), a})
	skyTop := 1e9
	for _, f := range def.Floors {
		if f.R.Y < skyTop {
			skyTop = f.R.Y
		}
	}
	if skyTop > 0 && skyTop < 1e9 {
		cutRect(dark, 0, 0, screenW, g.wy(skyTop), 1-aIn*0.5/aOut)
	}
	for _, f := range def.Floors {
		if !floorIndoor(f.Kind) {
			continue
		}
		r := sr(f.R)
		cutRect(dark, r.X+g.camI.X, r.Y+g.camI.Y, r.W, r.H, 1-aIn/aOut)
	}
	for _, dr := range def.Doors {
		if dr.Style == WallFence || dr.Style == WallHedge {
			continue
		}
		c := dr.R.Center()
		g.cutAt(dark, c, 46, 0.55)
	}
	for _, ls := range lights {
		g.cutAt(dark, ls.pos, ls.r*1.25, ls.cut)
	}
	dst.DrawImage(dark, nil)
	// 2. colour: the additive glows as before, on top of the darkness
	for _, ls := range lights {
		g.glowAt(dst, ls.pos, ls.r, ls.col)
	}
	g.finishLighting(dst)
}

// finishLighting is the part of the lighting pass shared by night and day levels.
func (g *Game) finishLighting(dst *ebiten.Image) {
	dst.DrawImage(g.as.vignette, nil)
	if g.flash > 0 {
		a := uint8(math.Min(1, g.flash*5) * 180)
		fillRect(dst, 0, 0, screenW, screenH, color.RGBA{a, a, a, a})
	}
	if g.mode == ModeLose {
		a := uint8(math.Min(1, (g.time-g.endAt)*1.5) * 120)
		fillRect(dst, 0, 0, screenW, screenH, color.RGBA{a, 0, 0, a})
	}
}

// ---------------------------------------------------------------------------
// HUD & menus
// ---------------------------------------------------------------------------

func (g *Game) textW(s string, scale float64) float64 {
	s = tr(s)
	w, _ := text.Measure(s, g.face, 0)
	return w * scale
}

func (g *Game) drawText(dst *ebiten.Image, s string, x, y float64, c color.RGBA, scale float64) {
	s = tr(s)
	op := &text.DrawOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(x+1, y+1)
	op.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, 255})
	text.Draw(dst, s, g.face, op)
	op = &text.DrawOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, s, g.face, op)
}

func (g *Game) drawTextC(dst *ebiten.Image, s string, cx, y float64, c color.RGBA, scale float64) {
	g.drawText(dst, s, math.Round(cx-g.textW(s, scale)/2), y, c, scale)
}

func (g *Game) panel(dst *ebiten.Image, x, y, w, h float64) {
	fillRect(dst, x, y, w, h, rgba(0x0a0812, 190))
	fillRect(dst, x, y, w, 1, rgb(0x6a5a9a))
	fillRect(dst, x, y+h-1, w, 1, rgb(0x6a5a9a))
}

// drawStars shows the 0-5 GTA-style criminality rating centered at the top
// of the screen, clear of the scenario below it.
func (g *Game) drawStars(dst *ebiten.Image) {
	if g.mode != ModePlay || g.stars <= 0 {
		return
	}
	const n, size, gap = 5, 10.0, 3.0
	totalW := n*size + (n-1)*gap
	x0 := screenW/2 - totalW/2
	for i := 0; i < n; i++ {
		g.drawStarIcon(dst, x0+float64(i)*(size+gap), 4, size, i < g.stars)
	}
}

func (g *Game) drawStarIcon(dst *ebiten.Image, x, y, size float64, on bool) {
	col := color.RGBA{90, 80, 70, 170}
	if on {
		col = color.RGBA{255, 210, 40, 255}
	}
	mid := x + size/2
	fillRect(dst, mid-1, y, 2, 2, col)
	fillRect(dst, x, y+2, size, 2, col)
	fillRect(dst, mid-2, y+4, 4, 2, col)
	fillRect(dst, mid-3.5, y+6, 2, 2, col)
	fillRect(dst, mid+1.5, y+6, 2, 2, col)
}

// rangerHaloCols: the classic 5 Power Ranger colors, one per squad member (cop.alt).
var rangerHaloCols = []color.RGBA{
	{220, 30, 30, 46},  // red
	{30, 90, 220, 46},  // blue
	{235, 210, 30, 46}, // yellow
	{40, 180, 60, 46},  // green
	{235, 90, 190, 46}, // pink
}

// copHaloCol: a faint, kept-subtle color per responder tier so the halo reads
// as a soft rim behind the body rather than a bright glow on top of it.
func (g *Game) copHaloCol(c *Cop) color.RGBA {
	col := color.RGBA{35, 100, 255, 40} // police: blue
	switch c.kind {
	case unitFBI:
		col = color.RGBA{184, 66, 245, 44} // purple
	case unitMilitary:
		col = color.RGBA{60, 200, 70, 44} // green
	case unitSWAT:
		col = color.RGBA{255, 202, 38, 44} // yellow
	case unitRangers:
		col = rangerHaloCols[((c.alt%5)+5)%5]
	}
	if c.medic {
		col = color.RGBA{90, 230, 150, 28}
	}
	return col
}

// starLabel/starHalo give each police-hierarchy tier its name and glow color
// for the HUD counters and the halos in drawLighting.
func starLabel(kind int) string {
	switch kind {
	case unitFBI:
		return "FBI"
	case unitMilitary:
		return "MILITAR"
	case unitSWAT:
		return "SWAT"
	case unitRangers:
		return "POWER RANGERS"
	default:
		return "POLICIA"
	}
}

func (g *Game) drawHUD(dst *ebiten.Image) {
	if g.mode == ModeMenu || g.mode == ModeLevelSelect || g.mode == ModeSettings || g.mode == ModeShop || g.mode == ModeConfirm || g.mode == ModeTrophies || g.mode == ModeScenes || g.mode == ModeDebug {
		return
	}
	alive := g.aliveCount()
	g.drawText(dst, fmt.Sprintf("POINTS:%04d", g.points), 10, 8, rgb(0xb88cff), 2)
	g.drawText(dst, fmt.Sprintf("KILLS:%02d/%02d", g.total-alive, g.total), 10, 36, rgb(0xff6aa8), 2)
	people, guards := 0, 0
	for _, gu := range g.guests {
		if !gu.out() {
			people++
		}
	}
	for _, gd := range g.guards {
		if !gd.dead {
			guards++
		}
	}
	byKind := map[int]int{}
	for _, c := range g.cops {
		if !c.medic && c.state != cLeave {
			byKind[c.kind]++
		}
	}
	killers := 0
	if g.slasher != nil && g.slasher.state != slGone {
		killers = 1
	}
	g.drawText(dst, fmt.Sprintf("GENTE:%d  GUARDIAS:%d", people, guards), 10, 61, rgb(0xff8070), 1)
	forces := ""
	for _, k := range []int{unitPolice, unitFBI, unitMilitary, unitSWAT, unitRangers} {
		if n := byKind[k]; n > 0 {
			if forces != "" {
				forces += "  "
			}
			forces += fmt.Sprintf("%s:%d", starLabel(k), n)
		}
	}
	if killers > 0 {
		if forces != "" {
			forces += "  "
		}
		forces += fmt.Sprintf("ASESINO:%d", killers)
	}
	if forces != "" {
		g.drawText(dst, forces, 10, 74, rgb(0x90c8ff), 1)
	}
	if it := g.p.item; it != "" && g.mode == ModePlay {
		name, col := "VENENO", rgb(0x60ff80)
		if it == "disguise" {
			name, col = "DISFRAZ", rgb(0xd090ff)
		}
		if it == "poison" && g.p.doses > 0 {
			name += " x" + itoa(g.p.doses)
		}
		g.drawText(dst, "OBJETO: "+name, 10, 88, col, 1)
	}
	g.drawStars(dst)
	g.drawCallHud(dst)
	if g.mode == ModePlay && g.titleT > 0 {
		a := math.Min(1, g.titleT)
		c := color.RGBA{uint8(255 * a), uint8(150 * a), uint8(30 * a), uint8(255 * a)}
		g.bigText(dst, g.lvl.Def.Title, screenW/2, 12, c, color.RGBA{uint8(90 * a), uint8(30 * a), 0, uint8(255 * a)}, 4)
	}
	if g.mode == ModePlay {
		y := 62.0 // below the level title, which it used to overlap
		for _, m := range g.msgs {
			mw := g.textW(m.text, 1) + 12
			g.panel(dst, math.Round(screenW/2-mw/2), y, mw, 17)
			g.drawTextC(dst, m.text, screenW/2, y+1, rgb(0xffffff), 1)
			y += 19
		}
	}
	if g.mode == ModePlay && g.p.stamina < 5.9 {
		fillRect(dst, 10, 88, 70, 4, rgb(0x2a2a3a))
		fillRect(dst, 10, 88, 70*g.p.stamina/6, 4, rgb(0x60d060))
	}
	if g.mode == ModePlay && g.time < 16 {
		s := fmt.Sprintf("%s/%s/%s/%s mover · %s apuñalar · %s interactuar · %s bailar · %s correr", actionHint(actUp), actionHint(actLeft), actionHint(actDown), actionHint(actRight), actionHint(actAttack), actionHint(actInteract), actionHint(actDance), actionHint(actRun))
		if g.eventPower() && g.evManual < manualEvents {
			s += fmt.Sprintf(" · %s suceso", actionHint(actEvent))
		}
		g.panel(dst, math.Round(screenW/2-g.textW(s, 1)/2-6), screenH-26, g.textW(s, 1)+12, 17)
		g.drawTextC(dst, s, screenW/2, screenH-25, rgb(0xb8b0d0), 1)
	}
}

// levelIconColors: a flat color swatch per campaign theme, menu-only. The
// game has no painted level thumbnails, so this reuses the same primitive,
// flat-color style as the rest of the HUD (stars, panels) instead of
// inventing a new art pipeline just for the menu.
var levelIconColors = map[string]uint32{
	"house": 0xb06a40, "beach": 0x30c0d0, "neon": 0xe030e0, "ranch": 0xc08840,
	"bull": 0xa02020, "eiffel": 0xb0b0c0, "boat": 0x2060b0, "funfair": 0xe08020,
	"stadium": 0x30a040, "disco": 0x9030e0, "festival": 0xe0c020, "skate": 0x20a0a0,
	"cinema": 0x802030, "circuit": 0x606060, "station": 0x4040a0,
	"roof": 0x6080c0, "grave": 0x6a3a8a, "snow": 0xa0c8f0, "casino": 0xe0b030, "pub": 0x8a5a2a, "plaza": 0xc0a070, "market": 0x40b060, "highway": 0xd09030, "train": 0x8a7050, "mall": 0xd060a0,
}

func (g *Game) drawLevelIcon(dst *ebiten.Image, x, y float64, icon string, dim bool) {
	col := rgb(0x4a4a5a)
	if hex, ok := levelIconColors[icon]; ok {
		col = rgb(hex)
	}
	if dim {
		col = shade(col, 0.4)
	}
	fillRect(dst, x, y, 14, 14, rgb(0x0a0a10))
	fillRect(dst, x+1, y+1, 12, 12, col)
}

func (g *Game) drawLock(dst *ebiten.Image, x, y float64) {
	col := rgb(0xd8d4c0)
	fillRect(dst, x+1, y+5, 8, 6, col)
	fillRect(dst, x+2, y+1, 2, 5, col)
	fillRect(dst, x+6, y+1, 2, 5, col)
	fillRect(dst, x+2, y, 6, 2, col)
	fillRect(dst, x+4, y+7, 2, 2, rgb(0x2a2a30))
}

// drawBedroom is the killer's bedroom behind both menu screens, laid out
// like the original's: a big window with blinds and the party lights outside,
// the shelf with mask, map and radio under it, the bed on the right and a
// teal rug on a dark floor.
// drawLogo is the title badge: radar rings, a small cyan PARTY and a big
// pink-to-purple HARD, top left like the original.
func (g *Game) drawLogo(dst *ebiten.Image, cx, y float64) {
	bob := math.Sin(g.time*3) * 3
	ring := rgba(0x6a4ab0, 110)
	ringAt(dst, cx, y+34+bob, 58, 2, ring)
	ringAt(dst, cx, y+34+bob, 44, 1, rgba(0x6a4ab0, 70))
	ringAt(dst, cx, y+34+bob, 70, 1, rgba(0x6a4ab0, 50))
	if g.allTrophies() { // the golden logo of whoever has won everything
		g.gradText(dst, "PARTY", cx-36, y+bob, rgb(0xfff0a0), rgb(0xf0c030), rgb(0x302000), 2)
		g.gradText(dst, "HARD", cx, y+16+bob, rgb(0xfff0a0), rgb(0xd88a10), rgb(0x302000), 6)
		return
	}
	g.gradText(dst, "PARTY", cx-36, y+bob, rgb(0x60d0f0), rgb(0x60d0f0), rgb(0x0a1a2a), 2)
	g.gradText(dst, "HARD", cx, y+16+bob, rgb(0xff3070), rgb(0x9040e0), rgb(0x300050), 6)
}

func (g *Game) drawMenu(dst *ebiten.Image) {
	if g.iconMode {
		dst.Fill(rgb(0x0a0912))
		g.drawLogo(dst, 160, 60)
		return
	}
	g.drawBedroom(dst)
	g.drawLogo(dst, 160, 30)
	list := g.menuList()
	gold := g.allTrophies()
	for i, id := range list {
		s := menuLabel(id)
		y := menuRowY(i, len(list))
		col := rgb(0x3fb59a)
		hot := rgb(0xff6a8a)
		if gold {
			col, hot = rgb(0xe0b030), rgb(0xfff0a0)
		}
		if id == menuDebug {
			col, hot = rgb(0xff9020), rgb(0xffd060)
		}
		if i == g.mainSel {
			g.drawText(dst, s+" ◄", 110, y, hot, 2.2)
			continue
		}
		g.drawText(dst, s, 110, y, col, 2.2)
	}
	if g.menuMsgT > 0 {
		g.drawTextC(dst, g.menuMsg, 520, 262, rgb(0xffa030), 1.6)
	}
	if gold {
		g.drawText(dst, "★ LEYENDA DE LA FIESTA ★", 110, 118, rgb(0xffe070), 1.3)
	}
	g.drawTextC(dst, "Nadie te dejó dormir. Ahora nadie saldrá de la fiesta.", 520, 232, rgba(0xe0d8f0, 200), 1)
	g.drawText(dst, actionHint(actConfirm)+"  seleccionar     "+actionHint(actBack)+"  volver", screenW-330, screenH-24, rgb(0x8a86a0), 1)
	g.drawText(dst, "M: sonido   F11: pantalla completa", 8, screenH-24, rgb(0x5a5670), 1)
	g.drawText(dst, "Versión "+gameVersion+" · hecho por Frange", 8, screenH-66, rgb(0xb8b0d0), 1)
	g.drawText(dst, "Homenaje a Party Hard (Pinokl Games · tinyBuild)", 8, screenH-52, rgb(0x7a7690), 1)
	if padConnected() {
		g.drawText(dst, "Mando: D-pad elegir · A confirmar · B volver · X usar · Y bailar · RB correr", 8, screenH-38, rgb(0x5a5670), 1)
	}
}

// the level list: three columns of seven rows
var levelColX = [3]float64{345, 580, 815}

const levelRowsPerCol = 8

func (g *Game) drawLevelSelect(dst *ebiten.Image) {
	g.drawBedroom(dst)
	g.drawLogo(dst, 160, 30)
	fillRect(dst, 230, 60, 700, 420, rgba(0x05070e, 215))
	fillRect(dst, 230, 60, 700, 1, rgba(0x6a4ab0, 160))
	fillRect(dst, 230, 479, 700, 1, rgba(0x6a4ab0, 160))
	g.drawTextC(dst, "ELIGE LA FIESTA", 580, 76, rgb(0xff6a8a), 2)
	// every row (the campaign + ALEATORIO) is always visible, no scrolling:
	// three 7-row columns side by side.
	rows := campaignN + 1
	for i := 0; i < rows; i++ {
		cx := levelColX[i/levelRowsPerCol]
		y := 118 + float64(i%levelRowsPerCol)*34
		col := rgb(0x3fb59a)
		pre := "  "
		if i == g.menuSel {
			col = rgb(0xff6a8a)
			pre = "> "
		}
		if i == campaignN {
			g.drawTextC(dst, pre+"ALEATORIO", cx, y, col, 1.3)
			continue
		}
		locked := i >= g.progress.Unlocked
		title := levelDefs[i].Title
		num := fmt.Sprintf("%d ", i+1)
		if locked {
			col = rgb(0x3a4652)
		}
		label := pre + num + title
		iconX := cx - g.textW(label, 1.3)/2 - 18
		g.drawLevelIcon(dst, iconX, y-1, levelDefs[i].Icon, locked)
		if locked {
			g.drawLock(dst, iconX+2, y+2)
		}
		g.drawTextC(dst, label, cx, y, col, 1.3)
	}
	lines := []string{
		fmt.Sprintf("Mover: %s %s %s %s   %s: correr   %s: apuñalar   %s: bailar   %s: usar / esconderse / cargar", actionHint(actUp), actionHint(actLeft), actionHint(actDown), actionHint(actRight), actionHint(actRun), actionHint(actAttack), actionHint(actDance), actionHint(actInteract)),
		"Los guardias también son objetivos. Solo ven hacia delante: acércate por la espalda.",
	}
	for i, l := range lines {
		g.drawTextC(dst, l, 580, 400+float64(i)*16, rgb(0x8a86a0), 1)
	}
	if int(g.time*2)%2 == 0 {
		g.drawTextC(dst, actionHint(actConfirm)+": empezar   "+actionHint(actBack)+": volver   R: aleatoria   G: generado   F2: editor", 580, 446, rgb(0xf0e060), 1.3)
	}
}

func (g *Game) drawPanel(dst *ebiten.Image, title, sub string, c color.RGBA, l3, l4 string) {
	fillRect(dst, 0, 170, screenW, 190, rgba(0x000000, 210))
	g.bigText(dst, title, screenW/2, 188, c, color.RGBA{0, 0, 0, 255}, 3)
	g.drawTextC(dst, sub, screenW/2, 248, rgb(0xe0d8f0), 1.5)
	g.drawTextC(dst, l3, screenW/2, 282, rgb(0xf0e060), 1.5)
	g.drawTextC(dst, l4, screenW/2, 322, rgb(0xb8b0d0), 1.5)
}

// ---------------------------------------------------------------------------
// ground, walls, doors and clutter
// ---------------------------------------------------------------------------

func alphaCol(c color.RGBA, a uint8) color.RGBA {
	return color.RGBA{uint8(uint16(c.R) * uint16(a) / 255), uint8(uint16(c.G) * uint16(a) / 255), uint8(uint16(c.B) * uint16(a) / 255), a}
}

// groundDetail breaks up the flat floor textures: stains, cracks, grass tufts, pebbles, scratches.
func (g *Game) groundDetail(bg *ebiten.Image, rng *rand.Rand, f Floor) {
	{
		switch f.Kind {
		case "dance", "dash", "rug", "helipad":
			return
		}
		r := sr(f.R)
		area := r.W * r.H
		put := func(x, y, w, h float64, c color.RGBA) {
			if x < r.X {
				w -= r.X - x
				x = r.X
			}
			if y < r.Y {
				h -= r.Y - y
				y = r.Y
			}
			if x+w > r.X+r.W {
				w = r.X + r.W - x
			}
			if y+h > r.Y+r.H {
				h = r.Y + r.H - y
			}
			if w > 0 && h > 0 {
				fillRect(bg, x, y, w, h, c)
			}
		}
		for i := 0; i < int(area/650); i++ {
			w, h := 6+rng.Float64()*18, 3+rng.Float64()*9
			x, y := r.X+rng.Float64()*r.W-w/2, r.Y+rng.Float64()*r.H-h/2
			if rng.Intn(2) == 0 {
				put(x, y, w, h, color.RGBA{0, 0, 0, uint8(9 + rng.Intn(10))})
			} else {
				a := uint8(4 + rng.Intn(7))
				put(x, y, w, h, color.RGBA{a, a, a, a})
			}
		}
		switch f.Kind {
		case "asphalt", "pave", "concrete":
			for i := 0; i < int(area/2200); i++ {
				x, y := r.X+rng.Float64()*r.W, r.Y+rng.Float64()*r.H
				for s := 0; s < 4+rng.Intn(4); s++ {
					put(x, y, 1, 1, color.RGBA{0, 0, 0, 90})
					x += float64(rng.Intn(3) - 1 + 1)
					y += float64(rng.Intn(3) - 1)
				}
			}
		case "grass":
			for i := 0; i < int(area/70); i++ {
				x, y := r.X+rng.Float64()*r.W, r.Y+rng.Float64()*r.H
				col := rgb(0x4a9650)
				if rng.Intn(2) == 0 {
					col = rgb(0x1f4a28)
				}
				put(x, y, 1, 2+float64(rng.Intn(2)), col)
				put(x+1, y+1, 1, 2, shade(col, 0.9))
			}
			for i := 0; i < int(area/700); i++ {
				x, y := r.X+rng.Float64()*r.W, r.Y+rng.Float64()*r.H
				col := []uint32{0xf4f4f0, 0xf0d040, 0xf080b0, 0xb090f0}[rng.Intn(4)]
				put(x, y, 1, 1, rgb(col))
			}
		case "sand", "dirt":
			for i := 0; i < int(area/110); i++ {
				x, y := r.X+rng.Float64()*r.W, r.Y+rng.Float64()*r.H
				if rng.Intn(2) == 0 {
					put(x, y, 1, 1, color.RGBA{0, 0, 0, 40})
				} else {
					put(x, y, 1, 1, color.RGBA{20, 20, 20, 20})
				}
			}
		case "wood", "parquet", "deck", "boardwalk":
			for i := 0; i < int(area/900); i++ {
				x, y := r.X+rng.Float64()*r.W, r.Y+rng.Float64()*r.H
				put(x, y, 3+rng.Float64()*6, 1, color.RGBA{16, 12, 8, 26})
			}
		case "tile", "marble", "carpet":
			for i := 0; i < int(area/1500); i++ {
				x, y := r.X+rng.Float64()*r.W, r.Y+rng.Float64()*r.H
				put(x, y, 3+rng.Float64()*4, 2+rng.Float64()*2, color.RGBA{30, 20, 10, 30})
			}
		}
	}
}

// litter: the mess a party leaves behind.
func (g *Game) litter(bg *ebiten.Image, rng *rand.Rand) {
	l := g.lvl
	for i := 0; i < 220; i++ {
		p := l.RandomRoam(rng)
		x, y := math.Round(p.X*wk), math.Round(p.Y*wk)
		switch rng.Intn(10) {
		case 0:
			fillRect(bg, x, y, 3, 2, rgb(0xe8e8f0))
			fillRect(bg, x+1, y, 1, 1, rgb(0xb8b8c8))
		case 1:
			fillRect(bg, x, y, 2, 3, rgb([]uint32{0xd03030, 0x3060d0, 0xe0c030}[rng.Intn(3)]))
			fillRect(bg, x, y, 2, 1, rgb(0xf4f4f4))
		case 2:
			fillRect(bg, x, y, 1, 4, rgb(0x2a8a40))
			fillRect(bg, x, y, 1, 1, rgb(0x90e0a0))
		case 3:
			fillRect(bg, x, y, 1, 4, rgb(0x7a4a22))
			fillRect(bg, x, y, 1, 1, rgb(0xd0a060))
		case 4:
			fillRect(bg, x, y, 2, 2, rgb(0xb8bcc8))
		case 5:
			for k := 0; k < 3; k++ {
				fillRect(bg, x+float64(rng.Intn(5)), y+float64(rng.Intn(3)), 1, 1, rgb([]uint32{0xff4090, 0x40e0ff, 0xf0e040, 0x80ff80}[rng.Intn(4)]))
			}
		case 6:
			fillRect(bg, x, y, 2, 1, rgb(0xf0f0f0))
			fillRect(bg, x+2, y, 1, 1, rgb(0xe08030))
		case 7:
			fillRect(bg, x, y, 5, 2, color.RGBA{20, 10, 0, 50})
		case 8:
			fillRect(bg, x, y, 3, 3, rgb(0xf4f4f0))
			fillRect(bg, x+1, y+1, 1, 1, rgb(0xd8c8a0))
		default:
			fillRect(bg, x, y, 4, 2, color.RGBA{60, 90, 140, 40})
		}
	}
}

// wallShadow: soft ambient occlusion on the floor next to walls.
func wallShadow(bg *ebiten.Image, r Rect, style int) {
	if style == WallHedge || style == WallFence {
		if r.W >= r.H {
			fillRect(bg, r.X, r.Y+r.H, r.W, 2, color.RGBA{0, 0, 0, 40})
		} else {
			fillRect(bg, r.X+r.W, r.Y, 2, r.H, color.RGBA{0, 0, 0, 40})
		}
		return
	}
	// a long, soft occlusion gradient on the floor side, and a short one behind
	if r.W >= r.H {
		for i, a := range []uint8{120, 100, 82, 66, 52, 40, 30, 21, 14, 8, 4} {
			fillRect(bg, r.X, r.Y+r.H+float64(i), r.W, 1, color.RGBA{0, 0, 0, a})
		}
		for i, a := range []uint8{40, 24, 12, 5} {
			fillRect(bg, r.X, r.Y-1-float64(i), r.W, 1, color.RGBA{0, 0, 0, a})
		}
	} else {
		for i, a := range []uint8{100, 80, 62, 46, 32, 21, 13, 7, 3} {
			fillRect(bg, r.X+r.W+float64(i), r.Y, 1, r.H, color.RGBA{0, 0, 0, a})
		}
		for i, a := range []uint8{40, 24, 12, 5} {
			fillRect(bg, r.X-1-float64(i), r.Y, 1, r.H, color.RGBA{0, 0, 0, a})
		}
	}
}

// drawDoor draws a door frame, threshold and an open door leaf in a wall gap.
func drawDoor(bg *ebiten.Image, d Door) {
	r := sr(d.R)
	frame, hi := rgb(0x2e1c10), rgb(0x6a4a2c)
	leaf, leafHi := rgb(0x8a5a2e), rgb(0xb88040)
	if d.Style == WallFence || d.Style == WallHedge {
		if d.Horiz {
			fillRect(bg, r.X, r.Y, 3, r.H, rgb(0x6a4a2a))
			fillRect(bg, r.X+r.W-3, r.Y, 3, r.H, rgb(0x6a4a2a))
			fillRect(bg, r.X, r.Y, 3, 1, hi)
			fillRect(bg, r.X+r.W-3, r.Y, 3, 1, hi)
		} else {
			fillRect(bg, r.X, r.Y, r.W, 3, rgb(0x6a4a2a))
			fillRect(bg, r.X, r.Y+r.H-3, r.W, 3, rgb(0x6a4a2a))
		}
		return
	}
	if d.Horiz {
		fillRect(bg, r.X, r.Y, r.W, 2, color.RGBA{0, 0, 0, 90})
		fillRect(bg, r.X, r.Y, 3, r.H, frame)
		fillRect(bg, r.X+r.W-3, r.Y, 3, r.H, frame)
		fillRect(bg, r.X, r.Y, 1, r.H, hi)
		fillRect(bg, r.X+r.W-3, r.Y, 1, r.H, hi)
		fillRect(bg, r.X+3, r.Y+r.H-2, r.W-6, 2, rgb(0x7a6a58))
		fillRect(bg, r.X+3, r.Y+r.H-2, r.W-6, 1, rgb(0xa89880))
		ll := math.Min(r.H-2, r.W/2)
		fillRect(bg, r.X+3, r.Y+1, 2, ll, leaf)
		fillRect(bg, r.X+3, r.Y+1, 1, ll, leafHi)
		fillRect(bg, r.X+4, r.Y+ll/2, 1, 1, rgb(0xf0d060))
	} else {
		fillRect(bg, r.X, r.Y, r.W, 3, frame)
		fillRect(bg, r.X, r.Y+r.H-3, r.W, 3, frame)
		fillRect(bg, r.X, r.Y, r.W, 1, hi)
		fillRect(bg, r.X+r.W-2, r.Y+3, 2, r.H-6, rgb(0x7a6a58))
		ll := math.Min(r.W-2, r.H/2)
		fillRect(bg, r.X+1, r.Y+3, ll, 2, leaf)
		fillRect(bg, r.X+1, r.Y+3, ll, 1, leafHi)
	}
}

// drawPictures hangs framed pictures and shelves on interior walls.
func drawPictures(dst *ebiten.Image, r Rect, capH float64, style int) {
	for x := 8.0; x+10 < r.W; x += 36 {
		h := uint32(int(r.X+x)*2654435761) >> 8
		if h%3 != 0 {
			continue
		}
		pw := 6 + float64(h%5)
		px, py := r.X+x, r.Y+capH+2
		fillRect(dst, px-1, py-1, pw+2, 7, rgb(0x2a1a10))
		art := []uint32{0x6a8ab8, 0xb86a6a, 0x6ab88a, 0xd8c070, 0x9a6ac0}[h%5]
		fillRect(dst, px, py, pw, 5, rgb(art))
		fillRect(dst, px, py, pw, 1, shade(rgb(art), 1.3))
		fillRect(dst, px+1, py+3, pw-2, 1, shade(rgb(art), 0.7))
	}
}

func genVignette() *pixBuf {
	b := newPix(screenW, screenH)
	for y := 0; y < screenH; y++ {
		for x := 0; x < screenW; x++ {
			dx := (float64(x) - screenW/2) / (screenW / 2)
			dy := (float64(y) - screenH/2) / (screenH / 2)
			d := math.Sqrt(dx*dx*0.8 + dy*dy*1.1)
			a := math.Max(0, d-0.62) / 0.7
			v := uint8(math.Min(1, a*a) * 70)
			i := (y*screenW + x) * 4
			b.p[i+3] = v
		}
	}
	return b
}

func softShadow(dst *ebiten.Image, cx, cy, w, h float64, alpha uint8) {
	rows := int(h)
	if rows < 1 {
		rows = 1
	}
	for i := 0; i < rows; i++ {
		t := (float64(i)+0.5)/float64(rows)*2 - 1
		ww := w * math.Sqrt(math.Max(0, 1-t*t))
		fillRect(dst, math.Round(cx-ww/2), math.Round(cy-h/2)+float64(i), math.Round(ww), 1, color.RGBA{0, 0, 0, alpha})
	}
}

// heliImg: the livery of this helicopter (nil when a custom helicopter sprite is in use).
func (g *Game) heliImg(v *Vehicle) *ebiten.Image {
	switch {
	case v.kind == 3:
		return g.as.heliU[5]
	case v.kind == 2 && v.unit >= 0 && v.unit < 5:
		return g.as.heliU[v.unit]
	}
	return nil
}

// actorBehind: the player, a guest, a guard or an officer is hidden behind this prop's sprite.
func (g *Game) actorBehind(pr *Prop, pi PropImg) bool {
	w, h := float64(pi.img.Bounds().Dx()), float64(pi.img.Bounds().Dy())
	x0, y0 := pr.X-pi.ox, pr.Y-pi.oy
	hit := func(p Vec) bool {
		if p.Y >= pr.Sort-1 || p.Y < y0 {
			return false
		}
		return p.X > x0+2 && p.X < x0+w-2 && p.Y-22 < y0+h
	}
	if !g.p.dead && !g.p.hidden && hit(g.p.pos) {
		return true
	}
	for _, gu := range g.guests {
		if !gu.out() && hit(gu.pos) {
			return true
		}
	}
	for _, gd := range g.guards {
		if !gd.dead && hit(gd.pos) {
			return true
		}
	}
	for _, c := range g.cops {
		if hit(c.pos) {
			return true
		}
	}
	return false
}

// drawPhoneMarkers: a bobbing green badge and a soft ring over every working phone booth,
// drawn above the lighting so they can be found in the dark.
func (g *Game) drawPhoneMarkers(dst *ebiten.Image) {
	if g.mode != ModePlay || g.lvl == nil {
		return
	}
	for _, pr := range g.lvl.Props {
		if !pr.K.Phone || pr.used {
			continue
		}
		x, y := g.wx(pr.X), g.wy(pr.Y)
		if x < -20 || x > screenW+20 || y < -20 || y > screenH+60 {
			continue
		}
		pulse := 0.5 + 0.5*math.Sin(g.time*3+pr.X)
		g.glowAt(dst, Vec{x - g.camI.X, y - g.camI.Y - 6}, 22+pulse*6, color.RGBA{80, 230, 140, uint8(50 + pulse*40)})
		by := math.Round(y - 60 + math.Sin(g.time*3.2+pr.X)*2)
		bx := math.Round(x) - 8
		fillRect(dst, bx, by, 16, 12, rgb(0x14502c))
		fillRect(dst, bx+1, by+1, 14, 10, rgb(0x30c060))
		// handset
		w := rgb(0xffffff)
		fillRect(dst, bx+3, by+3, 3, 6, w)
		fillRect(dst, bx+3, by+3, 10, 2, w)
		fillRect(dst, bx+10, by+3, 3, 6, w)
		fillRect(dst, bx+7, by+12, 2, 2, rgb(0x30c060))
	}
}

// copCSFor is the look of one responder: its unit's outfit, the medic suit or
// the police alternates. Bodies reuse it so the dead keep their own clothes.
func (g *Game) copCSFor(c *Cop) *CharSet {
	cs := g.as.copCS
	if c.alt > 0 && len(g.as.copAlt) > 0 && c.kind == unitPolice {
		cs = g.as.copAlt[(c.alt-1)%len(g.as.copAlt)]
	}
	switch c.kind {
	case unitFBI:
		if len(g.as.fbiCS) > 0 {
			cs = g.as.fbiCS[c.alt%len(g.as.fbiCS)]
		}
	case unitMilitary:
		if len(g.as.militaryCS) > 0 {
			cs = g.as.militaryCS[c.alt%len(g.as.militaryCS)]
		}
	case unitSWAT:
		cs = g.as.swatCS
	case unitRangers:
		if len(g.as.rangerCS) > 0 {
			cs = g.as.rangerCS[c.alt%len(g.as.rangerCS)]
		}
	}
	if c.medic {
		cs = g.as.medicCS
	}
	return cs
}
