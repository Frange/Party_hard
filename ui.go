package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// ---------------------------------------------------------------------------
// A tiny immediate-mode UI for the editor. Widgets handle input while the
// editor lays itself out in Update and queue draw commands that Draw replays.
// ---------------------------------------------------------------------------

const (
	edW   = 1280
	edH   = 720
	edTop = 36 // height of the top bar
)

var (
	uiBG       = rgb(0x13131a)
	uiPanel    = rgb(0x1b1b26)
	uiCard     = rgb(0x242434)
	uiBtn      = rgb(0x2d2d42)
	uiBtnHot   = rgb(0x3d3d5a)
	uiBtnOn    = rgb(0x6a50e8)
	uiBtnOnHot = rgb(0x7e66f5)
	uiText     = rgb(0xeeeef8)
	uiDim      = rgb(0x8c8cab)
	uiAccent   = rgb(0xffb347)
	uiLine     = rgb(0x34344c)
	uiGood     = rgb(0x44d492)
	uiRed      = rgb(0xff5a7a)
)

type UI struct {
	g        *Game
	mx, my   int
	down     bool
	pressed  bool
	released bool
	rpressed bool
	rdown    bool
	mdown    bool
	wheel    float64
	typed    []rune
	back     bool
	enter    bool
	esc      bool
	cmds     []func(*ebiten.Image)
	focus    string
	blink    int
	tip      string
	grab     string // widget currently dragged
	menu     menuState

	filterCache map[string]*filterEntry
}

func (u *UI) begin() {
	u.mx, u.my = ebiten.CursorPosition()
	u.down = ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	u.pressed = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	u.released = inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)
	u.rdown = ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)
	u.rpressed = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)
	u.mdown = ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle)
	_, u.wheel = ebiten.Wheel()
	u.typed = ebiten.AppendInputChars(u.typed[:0])
	u.back = inpututil.IsKeyJustPressed(ebiten.KeyBackspace) || (ebiten.IsKeyPressed(ebiten.KeyBackspace) && inpututil.KeyPressDuration(ebiten.KeyBackspace) > 20 && inpututil.KeyPressDuration(ebiten.KeyBackspace)%3 == 0)
	u.enter = inpututil.IsKeyJustPressed(ebiten.KeyEnter)
	u.esc = inpututil.IsKeyJustPressed(ebiten.KeyEscape)
	u.menu.pressedRaw = false
	if u.menu.id != "" {
		// an open pop-up swallows the clicks: only it reacts to them
		u.menu.pressedRaw = u.pressed
		u.pressed, u.rpressed, u.released = false, false, false
	}
	u.cmds = u.cmds[:0]
	u.tip = ""
	u.blink++
	if !u.down {
		u.grab = ""
	}
}

func (u *UI) in(x, y, w, h int) bool { return u.mx >= x && u.my >= y && u.mx < x+w && u.my < y+h }

func (u *UI) add(f func(*ebiten.Image)) { u.cmds = append(u.cmds, f) }

func (u *UI) rect(x, y, w, h int, c color.RGBA) {
	u.add(func(d *ebiten.Image) { fillRect(d, float64(x), float64(y), float64(w), float64(h), c) })
}

// round paints a filled rectangle with the corners cut, which is all the roundness a pixel UI needs.
func (u *UI) round(x, y, w, h int, c color.RGBA) {
	u.add(func(d *ebiten.Image) {
		fillRect(d, float64(x+2), float64(y), float64(w-4), float64(h), c)
		fillRect(d, float64(x), float64(y+2), float64(w), float64(h-4), c)
		fillRect(d, float64(x+1), float64(y+1), float64(w-2), float64(h-2), c)
	})
}

// roundFrame is the 1px outline of round().
func (u *UI) roundFrame(x, y, w, h int, c color.RGBA) {
	u.add(func(d *ebiten.Image) {
		fillRect(d, float64(x+2), float64(y), float64(w-4), 1, c)
		fillRect(d, float64(x+2), float64(y+h-1), float64(w-4), 1, c)
		fillRect(d, float64(x), float64(y+2), 1, float64(h-4), c)
		fillRect(d, float64(x+w-1), float64(y+2), 1, float64(h-4), c)
		fillRect(d, float64(x+1), float64(y+1), 1, 1, c)
		fillRect(d, float64(x+w-2), float64(y+1), 1, 1, c)
		fillRect(d, float64(x+1), float64(y+h-2), 1, 1, c)
		fillRect(d, float64(x+w-2), float64(y+h-2), 1, 1, c)
	})
}

func (u *UI) frame(x, y, w, h int, c color.RGBA) {
	u.add(func(d *ebiten.Image) {
		fillRect(d, float64(x), float64(y), float64(w), 1, c)
		fillRect(d, float64(x), float64(y+h-1), float64(w), 1, c)
		fillRect(d, float64(x), float64(y), 1, float64(h), c)
		fillRect(d, float64(x+w-1), float64(y), 1, float64(h), c)
	})
}

// plainText draws UI text without the drop shadow the game's text has: crisper on flat panels.
func (g *Game) plainText(d *ebiten.Image, s string, x, y float64, c color.RGBA) {
	s = tr(s)
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	text.Draw(d, s, g.face, op)
}

func (u *UI) text(x, y int, s string, c color.RGBA) {
	u.add(func(d *ebiten.Image) { u.g.plainText(d, s, float64(x), float64(y), c) })
}

func (u *UI) textC(cx, y int, s string, c color.RGBA) {
	u.add(func(d *ebiten.Image) { u.g.plainText(d, s, float64(cx)-u.g.textW(s, 1)/2, float64(y), c) })
}

func (u *UI) panel(x, y, w, h int) {
	u.round(x, y, w, h, uiPanel)
	u.roundFrame(x, y, w, h, uiLine)
}

// button returns true when clicked.
func (u *UI) button(x, y, w, h int, label string, on bool) bool {
	hot := u.in(x, y, w, h) && u.grab == ""
	col := uiBtn
	if on {
		col = uiBtnOn
	}
	if hot {
		col = uiBtnHot
		if on {
			col = uiBtnOnHot
		}
	}
	u.round(x, y, w, h, col)
	u.textC(x+w/2, y+(h-12)/2, label, uiText)
	if hot && u.pressed {
		u.focus = ""
		return true
	}
	return false
}

func (u *UI) toggle(x, y, w int, label string, val *bool) bool {
	if u.button(x, y, w, 20, label, *val) {
		*val = !*val
		return true
	}
	return false
}

// input edits a single-line string. Returns true when the text changed.
func (u *UI) input(id string, x, y, w int, val *string, maxLen int) bool {
	focused := u.focus == id
	hot := u.in(x, y, w, 20)
	if u.pressed {
		if hot {
			u.focus = id
			focused = true
		} else if focused {
			u.focus = ""
			focused = false
		}
	}
	changed := false
	if focused {
		for _, r := range u.typed {
			if r >= 32 && r != 127 && len([]rune(*val)) < maxLen {
				*val += string(r)
				changed = true
			}
		}
		if u.back && len(*val) > 0 {
			rs := []rune(*val)
			*val = string(rs[:len(rs)-1])
			changed = true
		}
		if u.enter {
			u.focus = ""
		}
	}
	col := rgb(0x14141c)
	if focused {
		col = rgb(0x22222e)
	}
	u.round(x, y, w, 20, col)
	bc := uiLine
	if focused {
		bc = uiAccent
	}
	u.roundFrame(x, y, w, 20, bc)
	s := *val
	if focused && (u.blink/30)%2 == 0 {
		s += "_"
	}
	for u.g.textW(s, 1) > float64(w-8) && len(s) > 1 {
		s = s[1:]
	}
	u.text(x+4, y+4, s, uiText)
	return changed
}

func (u *UI) slider(id string, x, y, w int, val *int, lo, hi int, label string) bool {
	hot := u.in(x, y, w, 14)
	changed := false
	if hot && u.pressed {
		u.grab = id
	}
	if u.grab == id && u.down {
		v := lo + int(math.Round(float64(u.mx-x)/float64(w)*float64(hi-lo)))
		v = clampI(v, lo, hi)
		if v != *val {
			*val = v
			changed = true
		}
	}
	u.round(x, y+5, w, 4, rgb(0x0e0e14))
	frac := float64(*val-lo) / float64(hi-lo)
	u.round(x, y+5, int(frac*float64(w))+1, 4, uiBtnOn)
	u.round(x+int(frac*float64(w))-4, y+1, 9, 12, uiAccent)
	u.text(x+w+8, y, fmt.Sprintf("%s %d", label, *val), uiText)
	return changed
}

func (u *UI) tooltip(x, y, w, h int, s string) {
	if u.in(x, y, w, h) {
		u.tip = s
	}
}

// list draws a scrollable list of names and returns the clicked index (or -1).
func (u *UI) list(x, y, w, rows int, items []string, sel int, scroll *int) int {
	u.round(x, y, w, rows*18+2, rgb(0x101018))
	u.roundFrame(x, y, w, rows*18+2, uiLine)
	if u.in(x, y, w, rows*18+2) && u.wheel != 0 {
		*scroll -= int(math.Round(u.wheel))
	}
	maxS := len(items) - rows
	if maxS < 0 {
		maxS = 0
	}
	*scroll = clampI(*scroll, 0, maxS)
	clicked := -1
	for i := 0; i < rows; i++ {
		idx := i + *scroll
		if idx >= len(items) {
			break
		}
		ry := y + 1 + i*18
		hot := u.in(x, ry, w, 18)
		if idx == sel {
			u.rect(x+1, ry, w-2, 18, uiBtnOn)
		} else if hot {
			u.rect(x+1, ry, w-2, 18, uiBtn)
		}
		s := items[idx]
		for u.g.textW(s, 1) > float64(w-10) && len(s) > 1 {
			s = s[:len(s)-1]
		}
		u.text(x+5, ry+3, s, uiText)
		if hot && u.pressed {
			clicked = idx
		}
	}
	return clicked
}

func hexOf(c color.RGBA) string { return fmt.Sprintf("%02X%02X%02X", c.R, c.G, c.B) }

func parseHex(s string) (color.RGBA, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 {
		return color.RGBA{}, false
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return color.RGBA{}, false
	}
	return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 255}, true
}

// ---------------------------------------------------------------------------
// Pixel canvas widget with zoom and pan
// ---------------------------------------------------------------------------

type Canvas struct {
	x, y   int // top-left of the viewport on screen
	vw, vh int // viewport size
	zoom   int
	fz     float64 // fractional zoom (the level editor); 0 = use the integer zoom
	px, py int     // pan: offset of the image inside the viewport, in screen pixels
	grid   bool
	img    *ebiten.Image
	dirty  bool

	bgImg, gridImg *ebiten.Image // cached checker and grid pictures
	bgKey, gridKey [4]int
}

func (c *Canvas) setPix(b *pixBuf) {
	if c.img == nil || c.img.Bounds().Dx() != b.w || c.img.Bounds().Dy() != b.h {
		c.img = ebiten.NewImage(b.w, b.h)
	}
	c.dirty = true
}

func (c *Canvas) refresh(b *pixBuf) {
	if c.img == nil || c.img.Bounds().Dx() != b.w || c.img.Bounds().Dy() != b.h {
		c.img = ebiten.NewImage(b.w, b.h)
	}
	c.img.WritePixels(b.p)
	c.dirty = false
}

func (c *Canvas) fit(w, h int) {
	z := minI(c.vw/w, c.vh/h)
	if z < 1 {
		z = 1
	}
	c.zoom = z
	c.px, c.py = (c.vw-w*z)/2, (c.vh-h*z)/2
}

// at converts a screen position to image pixel coordinates.
func (c *Canvas) at(mx, my int) (int, int) {
	z := c.zf()
	x := int(math.Floor(float64(mx-c.x-c.px) / z))
	y := int(math.Floor(float64(my-c.y-c.py) / z))
	return x, y
}

// zf is the zoom actually in use.
func (c *Canvas) zf() float64 {
	if c.fz > 0 {
		return c.fz
	}
	return float64(c.zoom)
}

func (c *Canvas) inView(mx, my int) bool {
	return mx >= c.x && my >= c.y && mx < c.x+c.vw && my < c.y+c.vh
}

// zoomAt changes the zoom keeping the pixel under the mouse in place.
func (c *Canvas) zoomAt(mx, my, nz int) {
	if nz < 1 || nz > 24 || nz == c.zoom {
		return
	}
	ix, iy := float64(mx-c.x-c.px)/float64(c.zoom), float64(my-c.y-c.py)/float64(c.zoom)
	c.zoom = nz
	c.px = mx - c.x - int(ix*float64(nz))
	c.py = my - c.y - int(iy*float64(nz))
}

// what the transparent pixels of a drawing look like while painting (shared by the editors)
type canvasBG struct {
	a, b  uint32 // the two checker colours (equal = flat colour)
	light bool
	name  string
}

var canvasBGs = []canvasBG{
	{0x2c2c38, 0x34343f, false, "Oscuro (cuadros)"},
	{0xc8c8d4, 0xdcdce6, true, "Claro (cuadros)"},
	{0xffffff, 0xffffff, true, "Blanco"},
	{0x808890, 0x808890, true, "Gris"},
	{0x4a9a5a, 0x4a9a5a, false, "Verde"},
	{0x5a8ad0, 0x5a8ad0, false, "Azul"},
	{0xd06a9a, 0xd06a9a, false, "Rosa"},
}

var (
	cvGrid = true
	cvBG   = 0
)

// canvasOptions: grid on/off and the colour shown behind transparent pixels. Returns the next free y.
func (u *UI) canvasOptions(x, y, w int) int {
	u.heading(x, y, w, "VISTA DEL DIBUJO")
	y += 20
	if u.iconBtn(x, y, w, 26, "grid", "Cuadrícula", cvGrid, "Mostrar u ocultar la cuadrícula mientras pintas") {
		cvGrid = !cvGrid
	}
	y += 32
	u.text(x, y, "Fondo transparente", uiDim)
	y += 16
	for i, b := range canvasBGs {
		cx := x + i*(w/len(canvasBGs))
		sw := w/len(canvasBGs) - 3
		u.rect(cx, y, sw, 18, rgb(b.a))
		if b.a != b.b {
			u.rect(cx, y, sw/2, 9, rgb(b.b))
			u.rect(cx+sw/2, y+9, sw-sw/2, 9, rgb(b.b))
		}
		if i == cvBG {
			u.frame(cx-1, y-1, sw+2, 20, uiAccent)
		}
		if u.in(cx, y, sw, 18) {
			u.tip = b.name
			if u.pressed {
				cvBG = i
			}
		}
	}
	return y + 28
}

func (c *Canvas) draw(d *ebiten.Image, w, h int, checker bool) {
	if c.img == nil {
		return
	}
	// clip drawing to the viewport using a sub-image
	vp := d.SubImage(imageRect(c.x, c.y, c.vw, c.vh)).(*ebiten.Image)
	fillRect(vp, float64(c.x), float64(c.y), float64(c.vw), float64(c.vh), rgb(0x101018))
	ix, iy := float64(c.x+c.px), float64(c.y+c.py)
	bg := canvasBGs[clampI(cvBG, 0, len(canvasBGs)-1)]
	if checker {
		// one cached picture scaled up instead of a rectangle per pixel
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(float64(c.zoom), float64(c.zoom))
		op.GeoM.Translate(ix, iy)
		vp.DrawImage(c.bgImage(w, h, bg), op)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(c.zf(), c.zf())
	op.GeoM.Translate(ix, iy)
	vp.DrawImage(c.img, op)
	if c.grid && cvGrid && c.zoom >= 6 && c.fz == 0 {
		vp.DrawImage(c.gridImage(w, h, c.zoom, bg.light), &ebiten.DrawImageOptions{GeoM: translateGeo(ix, iy)})
	}
}

func translateGeo(x, y float64) ebiten.GeoM {
	var g ebiten.GeoM
	g.Translate(x, y)
	return g
}

// bgImage is the picture of the transparent background (w x h pixels, drawn scaled).
func (c *Canvas) bgImage(w, h int, bg canvasBG) *ebiten.Image {
	key := [4]int{w, h, cvBG, 1}
	if c.bgImg != nil && c.bgKey == key {
		return c.bgImg
	}
	pix := make([]byte, w*h*4)
	for yy := 0; yy < h; yy++ {
		for xx := 0; xx < w; xx++ {
			col := rgb(bg.a)
			if bg.a != bg.b && (xx+yy)%2 == 0 {
				col = rgb(bg.b)
			}
			i := (yy*w + xx) * 4
			pix[i], pix[i+1], pix[i+2], pix[i+3] = col.R, col.G, col.B, 255
		}
	}
	if c.bgImg != nil {
		c.bgImg.Deallocate()
	}
	c.bgImg = ebiten.NewImage(w, h)
	c.bgImg.WritePixels(pix)
	c.bgKey = key
	return c.bgImg
}

// gridImage is the pixel grid for a w x h picture at this zoom, as one transparent picture.
func (c *Canvas) gridImage(w, h, zoom int, light bool) *ebiten.Image {
	l := 0
	if light {
		l = 1
	}
	key := [4]int{w, h, zoom, l}
	if c.gridImg != nil && c.gridKey == key {
		return c.gridImg
	}
	gw, gh := w*zoom+1, h*zoom+1
	pix := make([]byte, gw*gh*4)
	// premultiplied: white at alpha 22, or black at alpha 40
	r, a := byte(22), byte(22)
	if light {
		r, a = 0, 40
	}
	set := func(x, y int) {
		i := (y*gw + x) * 4
		pix[i], pix[i+1], pix[i+2], pix[i+3] = r, r, r, a
	}
	for xx := 0; xx <= w; xx++ {
		for y := 0; y < gh; y++ {
			set(xx*zoom, y)
		}
	}
	for yy := 0; yy <= h; yy++ {
		for x := 0; x < gw; x++ {
			set(x, yy*zoom)
		}
	}
	if c.gridImg != nil {
		c.gridImg.Deallocate()
	}
	c.gridImg = ebiten.NewImage(gw, gh)
	c.gridImg.WritePixels(pix)
	c.gridKey = key
	return c.gridImg
}

// ---------------------------------------------------------------------------
// Painting tools shared by every editor
// ---------------------------------------------------------------------------

const (
	ToolPencil = iota
	ToolEraser
	ToolLine
	ToolRect
	ToolRectFill
	ToolEllipse
	ToolFill
	ToolPicker
	ToolMove
	ToolCount
)

var toolNames = []string{"Lápiz", "Goma", "Línea", "Rect.", "Rect.▇", "Elipse", "Cubo", "Cuentagotas", "Mover"}

type snapshot struct {
	p   []byte
	pid []uint8
}

type Painter struct {
	tool     int
	col      color.RGBA
	part     uint8
	usePart  bool // paint the part id too (character editor)
	partOnly bool // PARTES mode: only the part map changes, the colours stay
	mirror   bool
	brush    int
	undo     []snapshot
	redo     []snapshot
	drawing  bool
	sx, sy   int
	lx, ly   int
	pvx, pvy int // preview end
	recent   []color.RGBA
	hexText  string
}

func newPainter() *Painter {
	return &Painter{col: rgb(0x2c3c78), brush: 1, part: PartTorso, hexText: "2C3C78"}
}

func (pt *Painter) snap(b *pixBuf) snapshot {
	s := snapshot{p: append([]byte(nil), b.p...)}
	if b.pid != nil {
		s.pid = append([]uint8(nil), b.pid...)
	}
	return s
}

func (pt *Painter) pushUndo(b *pixBuf) {
	pt.undo = append(pt.undo, pt.snap(b))
	if len(pt.undo) > 80 {
		pt.undo = pt.undo[1:]
	}
	pt.redo = nil
}

func (pt *Painter) restore(b *pixBuf, s snapshot) {
	if len(s.p) != len(b.p) {
		return
	}
	copy(b.p, s.p)
	if b.pid != nil && s.pid != nil {
		copy(b.pid, s.pid)
	}
}

func (pt *Painter) doUndo(b *pixBuf) bool {
	if len(pt.undo) == 0 {
		return false
	}
	pt.redo = append(pt.redo, pt.snap(b))
	pt.restore(b, pt.undo[len(pt.undo)-1])
	pt.undo = pt.undo[:len(pt.undo)-1]
	return true
}

func (pt *Painter) doRedo(b *pixBuf) bool {
	if len(pt.redo) == 0 {
		return false
	}
	pt.undo = append(pt.undo, pt.snap(b))
	pt.restore(b, pt.redo[len(pt.redo)-1])
	pt.redo = pt.redo[:len(pt.redo)-1]
	return true
}

func (pt *Painter) clearHistory() { pt.undo, pt.redo = nil, nil }

func swapPart(id uint8) uint8 {
	switch id {
	case PartArmL:
		return PartArmR
	case PartArmR:
		return PartArmL
	case PartLegL:
		return PartLegR
	case PartLegR:
		return PartLegL
	}
	return id
}

func (pt *Painter) setPx(b *pixBuf, x, y int, erase bool, mirrorParts bool) {
	put := func(x, y int, part uint8) {
		if x < 0 || y < 0 || x >= b.w || y >= b.h {
			return
		}
		i := (y*b.w + x) * 4
		if pt.partOnly && b.pid != nil {
			// assigning parts: only pixels that are painted take a part; the colours are left alone
			if b.p[i+3] > 0 {
				if erase {
					b.pid[y*b.w+x] = 0
				} else {
					b.pid[y*b.w+x] = part
				}
			}
			return
		}
		if erase {
			b.p[i], b.p[i+1], b.p[i+2], b.p[i+3] = 0, 0, 0, 0
			if b.pid != nil {
				b.pid[y*b.w+x] = 0
			}
			return
		}
		b.p[i], b.p[i+1], b.p[i+2], b.p[i+3] = pt.col.R, pt.col.G, pt.col.B, 255
		if b.pid != nil && pt.usePart {
			b.pid[y*b.w+x] = part
		}
	}
	for dy := 0; dy < pt.brush; dy++ {
		for dx := 0; dx < pt.brush; dx++ {
			put(x+dx, y+dy, pt.part)
			if pt.mirror {
				mp := pt.part
				if mirrorParts {
					mp = swapPart(pt.part)
				}
				put(b.w-1-(x+dx), y+dy, mp)
			}
		}
	}
}

func bresenham(x0, y0, x1, y1 int, f func(x, y int)) {
	dx, dy := abs(x1-x0), -abs(y1-y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	e := dx + dy
	for {
		f(x0, y0)
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * e
		if e2 >= dy {
			e += dy
			x0 += sx
		}
		if e2 <= dx {
			e += dx
			y0 += sy
		}
	}
}

func shapePoints(tool, x0, y0, x1, y1 int, f func(x, y int)) {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	switch tool {
	case ToolRect:
		for x := x0; x <= x1; x++ {
			f(x, y0)
			f(x, y1)
		}
		for y := y0; y <= y1; y++ {
			f(x0, y)
			f(x1, y)
		}
	case ToolRectFill:
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				f(x, y)
			}
		}
	case ToolEllipse:
		cx, cy := float64(x0+x1)/2, float64(y0+y1)/2
		rx, ry := float64(x1-x0)/2+0.5, float64(y1-y0)/2+0.5
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				dx, dy := (float64(x)-cx)/rx, (float64(y)-cy)/ry
				d := dx*dx + dy*dy
				if d <= 1 && d >= 1-2.2/math.Max(rx, ry) {
					f(x, y)
				}
			}
		}
	}
}

func (pt *Painter) flood(b *pixBuf, x, y int) {
	if x < 0 || y < 0 || x >= b.w || y >= b.h {
		return
	}
	get := func(x, y int) [5]uint8 {
		i := (y*b.w + x) * 4
		var id uint8
		if b.pid != nil {
			id = b.pid[y*b.w+x]
		}
		return [5]uint8{b.p[i], b.p[i+1], b.p[i+2], b.p[i+3], id}
	}
	target := get(x, y)
	if !pt.usePart {
		target[4] = 0
	}
	same := func(x, y int) bool {
		v := get(x, y)
		if !pt.usePart {
			v[4] = 0
		}
		return v == target
	}
	want := [5]uint8{pt.col.R, pt.col.G, pt.col.B, 255, pt.part}
	if !pt.usePart {
		want[4] = 0
	}
	if get(x, y) == want || (target[3] == 255 && target[0] == want[0] && target[1] == want[1] && target[2] == want[2] && (!pt.usePart || target[4] == want[4])) {
		return
	}
	stack := [][2]int{{x, y}}
	for len(stack) > 0 {
		p := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if p[0] < 0 || p[1] < 0 || p[0] >= b.w || p[1] >= b.h || !same(p[0], p[1]) {
			continue
		}
		pt.setPx(b, p[0], p[1], false, true)
		// setPx may mirror; keep filling the original region only
		stack = append(stack, [2]int{p[0] + 1, p[1]}, [2]int{p[0] - 1, p[1]}, [2]int{p[0], p[1] + 1}, [2]int{p[0], p[1] - 1})
	}
}

func (pt *Painter) pushRecent(c color.RGBA) {
	for i, r := range pt.recent {
		if r == c {
			pt.recent = append(pt.recent[:i], pt.recent[i+1:]...)
			break
		}
	}
	pt.recent = append([]color.RGBA{c}, pt.recent...)
	if len(pt.recent) > 16 {
		pt.recent = pt.recent[:16]
	}
}

func (pt *Painter) setColor(c color.RGBA) {
	pt.col = c
	pt.hexText = hexOf(c)
}

// handle processes mouse painting on a canvas. mirrorParts swaps left/right body parts when mirroring.
// Returns true when the pixels changed.
func (pt *Painter) handle(u *UI, cv *Canvas, b *pixBuf, mirrorParts bool) bool {
	changed := false
	x, y := cv.at(u.mx, u.my)
	inside := cv.inView(u.mx, u.my) && u.grab == ""
	switch {
	case u.pressed && inside:
		switch pt.tool {
		case ToolPicker:
			if x >= 0 && y >= 0 && x < b.w && y < b.h {
				i := (y*b.w + x) * 4
				if b.p[i+3] > 0 {
					pt.setColor(color.RGBA{b.p[i], b.p[i+1], b.p[i+2], 255})
					if b.pid != nil && pt.usePart && b.pid[y*b.w+x] > 0 {
						pt.part = b.pid[y*b.w+x]
					}
					pt.tool = ToolPencil // got a colour: back to painting
				}
			}
		case ToolFill:
			pt.pushUndo(b)
			if pt.partOnly {
				pt.floodPart(b, x, y)
			} else {
				pt.flood(b, x, y)
			}
			pt.pushRecent(pt.col)
			changed = true
		default:
			pt.pushUndo(b)
			pt.drawing = true
			pt.sx, pt.sy, pt.lx, pt.ly = x, y, x, y
			pt.pvx, pt.pvy = x, y
			if pt.tool == ToolPencil || pt.tool == ToolEraser {
				pt.setPx(b, x, y, pt.tool == ToolEraser, mirrorParts)
				pt.pushRecent(pt.col)
				changed = true
			}
		}
	case u.down && pt.drawing:
		if pt.tool == ToolPencil || pt.tool == ToolEraser {
			bresenham(pt.lx, pt.ly, x, y, func(px, py int) { pt.setPx(b, px, py, pt.tool == ToolEraser, mirrorParts) })
			pt.lx, pt.ly = x, y
			changed = true
		}
		pt.pvx, pt.pvy = x, y
	case u.released && pt.drawing:
		pt.drawing = false
		switch pt.tool {
		case ToolLine:
			bresenham(pt.sx, pt.sy, x, y, func(px, py int) { pt.setPx(b, px, py, false, mirrorParts) })
			changed = true
		case ToolRect, ToolRectFill, ToolEllipse:
			shapePoints(pt.tool, pt.sx, pt.sy, x, y, func(px, py int) { pt.setPx(b, px, py, false, mirrorParts) })
			changed = true
		}
		if changed {
			pt.pushRecent(pt.col)
		}
	}
	return changed
}

// previewShape returns the pixels of the shape being dragged (for the overlay).
func (pt *Painter) previewShape(f func(x, y int)) {
	if !pt.drawing {
		return
	}
	switch pt.tool {
	case ToolLine:
		bresenham(pt.sx, pt.sy, pt.pvx, pt.pvy, f)
	case ToolRect, ToolRectFill, ToolEllipse:
		shapePoints(pt.tool, pt.sx, pt.sy, pt.pvx, pt.pvy, f)
	}
}

var defaultPalette = []uint32{
	0x000000, 0x1a1c2c, 0x2c3c78, 0x3b5dc9, 0x41a6f6, 0x73eff7, 0xf4f4f4, 0x94b0c2,
	0x566c86, 0x333c57, 0x5d275d, 0xb13e53, 0xef7d57, 0xffcd75, 0xa7f070, 0x38b764,
	0x257179, 0x29366f, 0x3a2a1a, 0x6b4630, 0x9c6a43, 0xc98a5e, 0xe8b48a, 0xf4c9a4,
	0xf7d9c0, 0xffe9d8, 0x14101a, 0x2b1d12, 0x4a2c17, 0x8a5a2b, 0xd9a441, 0xe8d6a0,
	0xe04040, 0xff8030, 0xf0c030, 0x90e040, 0x45c45a, 0x20c0b0, 0x3aa0e0, 0xb050d0,
	0xf060a0, 0xd94a9a, 0x7a2a4a, 0x4a3a2a, 0x5a5a66, 0x6a6a74, 0xc8b88a, 0xffffff,
}

// paletteUI draws tools, parts, colors for a painter. x,y is the top-left corner.
func (u *UI) paletteUI(pt *Painter, x, y int, withParts bool) int {
	// tools
	for i := 0; i < ToolCount; i++ {
		bx := x + (i%3)*76
		by := y + (i/3)*22
		if u.button(bx, by, 74, 20, toolNames[i], pt.tool == i) {
			pt.tool = i
		}
	}
	y += 3*22 + 4
	u.toggle(x, y, 74, "Simetría", &pt.mirror)
	if u.button(x+76, y, 36, 20, "-", false) && pt.brush > 1 {
		pt.brush--
	}
	u.textC(x+130, y+4, fmt.Sprintf("%dpx", pt.brush), uiText)
	if u.button(x+150, y, 36, 20, "+", false) && pt.brush < 6 {
		pt.brush++
	}
	y += 26
	if withParts {
		u.text(x, y, "Parte del cuerpo (se pinta con el color):", uiDim)
		y += 16
		for i := 0; i <= 6; i++ {
			bx := x + (i%4)*57
			by := y + (i/4)*22
			label := partNames[i]
			if u.button(bx, by, 55, 20, label, pt.part == uint8(i)) {
				pt.part = uint8(i)
			}
			if i > 0 {
				u.rect(bx+2, by+16, 51, 3, partColors[i])
			}
		}
		y += 2*22 + 6
	}
	// palette
	for i, c := range defaultPalette {
		cx := x + (i%12)*17
		cy := y + (i/12)*17
		col := rgb(c)
		u.rect(cx, cy, 15, 15, col)
		if pt.col == col {
			u.frame(cx-1, cy-1, 17, 17, uiAccent)
		}
		if u.in(cx, cy, 15, 15) && u.pressed {
			pt.setColor(col)
		}
	}
	y += 4*17 + 4
	if len(pt.recent) > 0 {
		u.text(x, y, "Recientes", uiDim)
		y += 14
		for i, c := range pt.recent {
			cx := x + (i%12)*17
			cy := y + (i/12)*17
			u.rect(cx, cy, 15, 15, c)
			if u.in(cx, cy, 15, 15) && u.pressed {
				pt.setColor(c)
			}
		}
		y += 17 * ((len(pt.recent) + 11) / 12)
	}
	y += 4
	u.rect(x, y, 30, 30, pt.col)
	u.frame(x, y, 30, 30, uiLine)
	if u.input("hex", x+38, y, 70, &pt.hexText, 7) {
		if c, ok := parseHex(pt.hexText); ok {
			pt.col = c
		}
	}
	u.text(x+114, y+4, "#HEX", uiDim)
	r, g, b := int(pt.col.R), int(pt.col.G), int(pt.col.B)
	y += 34
	ch := false
	ch = u.slider("sr", x, y, 150, &r, 0, 255, "R") || ch
	ch = u.slider("sg", x, y+16, 150, &g, 0, 255, "G") || ch
	ch = u.slider("sb", x, y+32, 150, &b, 0, 255, "B") || ch
	if ch {
		pt.setColor(color.RGBA{uint8(r), uint8(g), uint8(b), 255})
	}
	return y + 52
}

func imageRect(x, y, w, h int) image.Rectangle { return image.Rect(x, y, x+w, y+h) }

// rectFill paints a filled rectangle with the current colour and part (used by tests and tools).
func (pt *Painter) rectFill(b *pixBuf, x0, y0, x1, y1 int) {
	shapePoints(ToolRectFill, x0, y0, x1, y1, func(x, y int) { pt.setPx(b, x, y, false, false) })
}

// floodPart gives the whole connected painted region under (x,y) that shares its part the current part.
func (pt *Painter) floodPart(b *pixBuf, x, y int) {
	if b.pid == nil || x < 0 || y < 0 || x >= b.w || y >= b.h || b.p[(y*b.w+x)*4+3] == 0 {
		return
	}
	target := b.pid[y*b.w+x]
	if target == pt.part {
		return
	}
	stack := [][2]int{{x, y}}
	for len(stack) > 0 {
		p := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		px, py := p[0], p[1]
		if px < 0 || py < 0 || px >= b.w || py >= b.h || b.p[(py*b.w+px)*4+3] == 0 || b.pid[py*b.w+px] != target {
			continue
		}
		b.pid[py*b.w+px] = pt.part
		stack = append(stack, [2]int{px + 1, py}, [2]int{px - 1, py}, [2]int{px, py + 1}, [2]int{px, py - 1})
	}
}
