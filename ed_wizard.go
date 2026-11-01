package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ---------------------------------------------------------------------------
// Step-by-step character wizard: front view -> auto side -> auto back.
// It is a thin layer over CharEd's existing canvas/painter/palette — every
// step is just "paint this *pixBuf with the usual tools", plus one button
// to synthesize the next view from the front art.
// ---------------------------------------------------------------------------

type wizStep int

const (
	wizFront wizStep = iota
	wizSide
	wizBack
)

var wizStepTitles = []string{
	"Paso 1/3: dibuja la vista frontal",
	"Paso 2/3: revisa/edita el lateral generado",
	"Paso 3/3: revisa/edita la espalda generada",
}

type CharWizard struct {
	ce   *CharEd
	ca   *CharAsset
	step wizStep
	name string
	// gender picked on the back step; only affects the AnimPee pose later.
	gender string
}

func newCharWizard(ce *CharEd, cat string) *CharWizard {
	w := &CharWizard{ce: ce, name: "nuevo"}
	w.ca = charAssetFromSpec(genericSpec(rand.New(rand.NewSource(rand.Int63()))), "nuevo", cat, "")
	w.ca.Meta.Name = "nuevo"
	for _, v := range w.ca.Views {
		if v.pid == nil {
			v.pid = make([]uint8, v.w*v.h)
		}
	}
	return w
}

// reroll replaces the front template with a fresh random one — "can modify a sample".
func (w *CharWizard) reroll() {
	cat := w.ca.Meta.Category
	w.ca = charAssetFromSpec(genericSpec(rand.New(rand.NewSource(rand.Int63()))), "nuevo", cat, "")
	w.ca.Meta.Name = "nuevo"
}

func (w *CharWizard) curView() *pixBuf {
	switch w.step {
	case wizSide:
		return w.ca.Views[1]
	case wizBack:
		return w.ca.Views[2]
	}
	return w.ca.Views[0]
}

func (w *CharWizard) openCanvas() {
	ce := w.ce
	ce.ed.pt.clearHistory()
	buf := w.curView()
	ce.cv.fit(buf.w, buf.h)
	ce.cv.setPix(buf)
}

func (w *CharWizard) cancel() {
	ce := w.ce
	ce.wiz = nil
	ce.ed.pt.clearHistory()
	ce.cv.fit(ce.ca.Views[ce.view].w, ce.ca.Views[ce.view].h)
	ce.cv.setPix(ce.ca.Views[ce.view])
}

func (w *CharWizard) finish() {
	ce := w.ce
	ca := w.ca
	name := safeName(w.name)
	if name == "" {
		name = "nuevo"
	}
	ca.Meta.Name = name
	ca.Meta.Category = ce.category()
	ca.Meta.Gender = w.gender
	for _, v := range ca.Views {
		autoTag(v)
	}
	if err := saveCharAsset(ce.ed.root, ca); err != nil {
		ce.ed.say("Error al guardar: " + err.Error())
		return
	}
	ce.wiz = nil
	ce.ca = ca
	ce.name = ca.Meta.Name
	ce.reloadList()
	ce.afterLoad()
	ce.ed.say("Personaje creado en " + ca.Dir)
}

func (w *CharWizard) update(u *UI) {
	ce := w.ce
	pt := ce.ed.pt
	pt.usePart = false

	u.panel(4, 32, 236, 684)
	y := 40
	u.text(12, y, "NUEVO PERSONAJE (PASO A PASO)", uiAccent)
	y += 20
	u.text(12, y, wizStepTitles[w.step], uiDim)
	y += 30
	u.text(12, y, "Nombre (carpeta)", uiDim)
	u.input("wizname", 12, y+14, 218, &w.name, 24)
	y += 46

	switch w.step {
	case wizFront:
		if u.button(12, y, 218, 22, "Nueva plantilla aleatoria", false) {
			w.reroll()
			w.openCanvas()
		}
		y += 28
		if u.button(12, y, 218, 22, "Siguiente: generar lateral ▶", false) {
			w.ca.Views[1] = synthesizeSide(w.ca.Views[0])
			w.step = wizSide
			w.openCanvas()
		}
	case wizSide:
		if u.button(12, y, 218, 22, "◄ Atrás (frontal)", false) {
			w.step = wizFront
			w.openCanvas()
		}
		y += 28
		if u.button(12, y, 218, 22, "Regenerar desde el frontal", false) {
			w.ca.Views[1] = synthesizeSide(w.ca.Views[0])
			w.openCanvas()
		}
		y += 28
		if u.button(12, y, 218, 22, "Siguiente: generar espalda ▶", false) {
			w.ca.Views[2] = synthesizeBack(w.ca.Views[0])
			w.step = wizBack
			w.openCanvas()
		}
	case wizBack:
		if u.button(12, y, 218, 22, "◄ Atrás (lateral)", false) {
			w.step = wizSide
			w.openCanvas()
		}
		y += 28
		if u.button(12, y, 218, 22, "Regenerar desde el frontal", false) {
			w.ca.Views[2] = synthesizeBack(w.ca.Views[0])
			w.openCanvas()
		}
		y += 30
		u.text(12, y, "Sexo", uiDim)
		if u.button(60, y-4, 170, 20, "sexo: "+orDash(w.gender), false) {
			w.gender = cycleStr(w.gender, []string{"", "m", "f"})
		}
		u.tooltip(60, y-4, 170, 20, "Solo afecta a la animación de hacer pis")
		y += 30
		if u.button(12, y, 218, 26, "HECHO", false) {
			w.finish()
			return
		}
	}
	if u.button(12, y+36, 218, 22, "Cancelar", false) {
		w.cancel()
		return
	}

	buf := w.curView()
	if ce.cv.img == nil || ce.cv.img.Bounds().Dx() != buf.w || ce.cv.img.Bounds().Dy() != buf.h {
		ce.cv.fit(buf.w, buf.h)
		ce.cv.setPix(buf)
	}
	// zoom, pan, paint — identical handling to the free-form view tab.
	if ce.cv.inView(u.mx, u.my) {
		if u.wheel > 0 {
			ce.cv.zoomAt(u.mx, u.my, ce.cv.zoom+1)
		} else if u.wheel < 0 {
			ce.cv.zoomAt(u.mx, u.my, ce.cv.zoom-1)
		}
	}
	if (u.mdown || (ebiten.IsKeyPressed(ebiten.KeySpace) && u.down)) && ce.cv.inView(u.mx, u.my) {
		if ce.lastMX != 0 || ce.lastMY != 0 {
			ce.cv.px += u.mx - ce.lastMX
			ce.cv.py += u.my - ce.lastMY
		}
	}
	ce.lastMX, ce.lastMY = u.mx, u.my
	if u.focus == "" {
		ctrl := ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta)
		if ctrl && inpututil.IsKeyJustPressed(ebiten.KeyZ) {
			if ebiten.IsKeyPressed(ebiten.KeyShift) {
				pt.doRedo(buf)
			} else {
				pt.doUndo(buf)
			}
			ce.cv.dirty = true
		}
		if ctrl && inpututil.IsKeyJustPressed(ebiten.KeyY) {
			pt.doRedo(buf)
			ce.cv.dirty = true
		}
	}
	panning := u.mdown || (ebiten.IsKeyPressed(ebiten.KeySpace) && u.down)
	if !panning && pt.handle(u, &ce.cv, buf, true) {
		ce.cv.dirty = true
	}
	if pt.drawing || u.released {
		ce.cv.dirty = true
	}
	if ce.cv.dirty {
		ce.cv.refresh(buf)
	}
	cv := ce.cv
	u.add(func(d *ebiten.Image) {
		cv.draw(d, buf.w, buf.h, true)
		vp := d.SubImage(imageRect(cv.x, cv.y, cv.vw, cv.vh)).(*ebiten.Image)
		pt.previewShape(func(x, y int) {
			fillRect(vp, float64(cv.x+cv.px+x*cv.zoom), float64(cv.y+cv.py+y*cv.zoom), float64(cv.zoom), float64(cv.zoom), color.RGBA{pt.col.R, pt.col.G, pt.col.B, 160})
		})
		hx, hy := cv.at(u.mx, u.my)
		if cv.inView(u.mx, u.my) && hx >= 0 && hy >= 0 && hx < buf.w && hy < buf.h {
			fillRect(vp, float64(cv.x+cv.px+hx*cv.zoom), float64(cv.y+cv.py+hy*cv.zoom), float64(cv.zoom), 1, uiAccent)
			fillRect(vp, float64(cv.x+cv.px+hx*cv.zoom), float64(cv.y+cv.py+(hy+1)*cv.zoom-1), float64(cv.zoom), 1, uiAccent)
		}
	})
	u.frame(cv.x-1, cv.y-1, cv.vw+2, cv.vh+2, uiLine)
	u.text(cv.x, cv.y+cv.vh+4, "Ctrl+Z deshacer · rueda: zoom · igual que la pestaña normal del personaje", uiDim)

	u.panel(542, 32, 246, 684)
	u.text(550, 38, "HERRAMIENTAS", uiAccent)
	u.paletteUI(pt, 550, 56, false)
}

// ---------------------------------------------------------------------------
// Front -> side / back synthesis. Deliberately only a starting template —
// there is no 3D model here, just a silhouette heuristic the user then
// hand-edits (same honesty as the mirror fallback in processToGenerate).
// ---------------------------------------------------------------------------

// partBBox returns the bounding box of every opaque pixel tagged with id.
func partBBox(b *pixBuf, id uint8) (x0, y0, x1, y1 int, ok bool) {
	x0, y0, x1, y1 = b.w, b.h, -1, -1
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			if b.pid[y*b.w+x] == id && b.p[(y*b.w+x)*4+3] > 0 {
				x0, y0 = minI(x0, x), minI(y0, y)
				x1, y1 = maxI(x1, x), maxI(y1, y)
			}
		}
	}
	return x0, y0, x1, y1, x1 >= 0
}

// dominantColor returns the most common opaque colour of a part within a row range.
func dominantColor(b *pixBuf, id uint8, y0, y1 int) color.RGBA {
	counts := map[[3]uint8]int{}
	for y := maxI(0, y0); y <= minI(b.h-1, y1); y++ {
		for x := 0; x < b.w; x++ {
			i := (y*b.w + x) * 4
			if b.pid[y*b.w+x] == id && b.p[i+3] > 0 {
				counts[[3]uint8{b.p[i], b.p[i+1], b.p[i+2]}]++
			}
		}
	}
	var best [3]uint8
	bc := 0
	for c, n := range counts {
		if n > bc {
			best, bc = c, n
		}
	}
	if bc == 0 {
		return color.RGBA{180, 180, 190, 255}
	}
	return color.RGBA{best[0], best[1], best[2], 255}
}

// synthesizeSide builds a profile template from a front view. buildRig
// already duplicates and darkens whichever of the far arm/leg is left
// untagged in a side view (rig.go), so this only has to paint ONE arm and
// ONE leg (screen-right) plus a head and torso — the rig fills in the rest.
func synthesizeSide(front *pixBuf) *pixBuf {
	if front.pid == nil {
		front.pid = make([]uint8, front.w*front.h)
	}
	src := &pixBuf{w: front.w, h: front.h, p: append([]byte(nil), front.p...), pid: append([]uint8(nil), front.pid...)}
	if isUntagged(src) {
		autoParts(src, false)
	}
	out := newPix(front.w, front.h)
	out.pid = make([]uint8, front.w*front.h)
	cx := float64(front.w) / 2

	if x0, y0, x1, y1, ok := partBBox(src, PartHead); ok {
		hw := math.Max(2, float64(x1-x0+1)*0.6)
		col := dominantColor(src, PartHead, y0, y0+(y1-y0)/3+1)
		out.cur = PartHead
		out.rect(int(cx-hw/2), y0, int(hw), y1-y0+1, col)
		out.set(int(cx+hw/2), (y0+y1)/2, shade(col, 0.8)) // nose bump, facing right
	}
	var hipY = front.h
	if x0, y0, x1, y1, ok := partBBox(src, PartTorso); ok {
		tw := math.Max(3, float64(x1-x0+1)*0.5)
		col := dominantColor(src, PartTorso, y0, y1)
		out.cur = PartTorso
		out.rect(int(cx-tw/2), y0, int(tw), y1-y0+1, col)
		hipY = y1 + 1
	}
	if _, y0, _, y1, ok := partBBox(src, PartArmR); ok {
		col := dominantColor(src, PartArmR, y0, y1)
		out.cur = PartArmR
		out.rect(int(cx), y0, 2, y1-y0+1, col)
	}
	if _, y0, _, y1, ok := partBBox(src, PartLegR); ok {
		col := dominantColor(src, PartLegR, y0, y1)
		out.cur = PartLegR
		out.rect(int(cx-1), maxI(y0, hipY), 2, y1-maxI(y0, hipY)+1, col)
	}
	return out
}

// synthesizeBack starts from an unmirrored copy of the front (clothing
// silhouette is close enough front vs. back as a template), flattens the
// head band to a hair-only colour (no face), and darkens the rest slightly
// to read as "facing away".
func synthesizeBack(front *pixBuf) *pixBuf {
	if front.pid == nil {
		front.pid = make([]uint8, front.w*front.h)
	}
	src := &pixBuf{w: front.w, h: front.h, p: append([]byte(nil), front.p...), pid: append([]uint8(nil), front.pid...)}
	if isUntagged(src) {
		autoParts(src, false)
	}
	out := &pixBuf{w: front.w, h: front.h, p: append([]byte(nil), src.p...), pid: append([]uint8(nil), src.pid...)}
	if x0, y0, x1, y1, ok := partBBox(src, PartHead); ok {
		hairCol := dominantColor(src, PartHead, y0, y0+(y1-y0)/3+1)
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				if src.pid[y*src.w+x] == PartHead && src.p[(y*src.w+x)*4+3] > 0 {
					out.set(x, y, hairCol)
				}
			}
		}
	}
	for i := 0; i < len(out.p); i += 4 {
		if out.p[i+3] == 0 {
			continue
		}
		out.p[i] = uint8(float64(out.p[i]) * 0.88)
		out.p[i+1] = uint8(float64(out.p[i+1]) * 0.88)
		out.p[i+2] = uint8(float64(out.p[i+2]) * 0.88)
	}
	return out
}

func isUntagged(b *pixBuf) bool {
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			if b.p[(y*b.w+x)*4+3] > 0 && b.pid[y*b.w+x] != 0 {
				return false
			}
		}
	}
	return true
}
