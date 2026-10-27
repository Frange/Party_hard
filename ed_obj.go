package main

import (
	"fmt"
	"image/color"
	"path/filepath"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var objCategories = []string{"furniture", "decor", "vehicle", "animal", "trap", "nature", "special"}
var useKinds = []string{"", "drink", "toilet", "lounge"}

type ObjEd struct {
	ed       *Editor
	meta     ObjMeta
	frames   []*pixBuf
	dir      string
	list     []string
	sel      int
	scroll   int
	name     string
	frame    int
	cv       Canvas
	mode     int // 0 paint, 1 anchor, 2 footprint, 3 use points
	usePts   [][2]int
	foot     [4]int // x, y, w, h
	anchor   [2]int
	dragFoot bool
	fx0, fy0 int
	prevT    float64
	refCS    *CharSet
	prevImg  *ebiten.Image
	lastMX   int
	lastMY   int
}

func newObjEd(ed *Editor) *ObjEd {
	o := &ObjEd{ed: ed, sel: -1}
	o.cv = Canvas{x: 244, y: 90, vw: 520, vh: 520, zoom: 8, grid: true}
	o.newBlank()
	return o
}

func (o *ObjEd) reloadList() {
	o.list = listAssetNames(spritesDir(o.ed.root, "objetos"))
}

func (o *ObjEd) newBlank() {
	o.meta = ObjMeta{Name: "nuevo_objeto", Category: "furniture", Solid: true}
	o.name = "nuevo_objeto"
	o.frames = []*pixBuf{newPix(48, 48)}
	o.dir = ""
	o.frame = 0
	o.anchor = [2]int{24, 46}
	o.foot = [4]int{6, 40, 36, 7}
	o.usePts = nil
	o.afterLoad()
}

func (o *ObjEd) afterLoad() {
	o.ed.pt.clearHistory()
	o.ed.pt.usePart = false
	o.cv.fit(o.frames[0].w, o.frames[0].h)
	o.cv.setPix(o.frames[o.frame])
}

func (o *ObjEd) loadByName(name string) {
	oa, err := loadObjAsset(filepath.Join(spritesDir(o.ed.root, "objetos"), name))
	if err != nil {
		o.ed.say("No se pudo cargar: " + err.Error())
		return
	}
	o.meta = oa.Meta
	o.frames = oa.Frames
	o.dir = oa.Dir
	o.name = name
	o.frame = 0
	o.anchor = [2]int{oa.Meta.AnchorX, oa.Meta.AnchorY}
	o.foot = [4]int{oa.Meta.FootX, oa.Meta.FootY, oa.Meta.FootW, oa.Meta.FootH}
	o.usePts = nil
	for i := range oa.Meta.UseX {
		if i < len(oa.Meta.UseY) {
			o.usePts = append(o.usePts, [2]int{oa.Meta.UseX[i] + oa.Meta.AnchorX, oa.Meta.UseY[i] + oa.Meta.AnchorY})
		}
	}
	o.afterLoad()
}

func (o *ObjEd) save() {
	name := safeName(o.name)
	if name == "" {
		o.ed.say("Pon un nombre al objeto")
		return
	}
	m := o.meta
	m.Name = name
	m.AnchorX, m.AnchorY = o.anchor[0], o.anchor[1]
	m.FootX, m.FootY, m.FootW, m.FootH = o.foot[0], o.foot[1], o.foot[2], o.foot[3]
	m.UseX, m.UseY = nil, nil
	for _, p := range o.usePts {
		m.UseX = append(m.UseX, p[0]-m.AnchorX)
		m.UseY = append(m.UseY, p[1]-m.AnchorY)
	}
	if m.UseKind == "" {
		m.UseX, m.UseY = nil, nil
	}
	oa := &ObjAsset{Meta: m, Frames: o.frames}
	if err := saveObjAsset(o.ed.root, oa); err != nil {
		o.ed.say("Error al guardar: " + err.Error())
		return
	}
	o.dir = oa.Dir
	o.name = name
	o.meta = m
	o.reloadList()
	o.ed.refreshObjects()
	o.ed.say("Guardado en " + oa.Dir)
}

func (ed *Editor) refreshObjects() {
	g := ed.g
	st := loadAssetStore()
	if st != nil {
		applyObjects(st.Objs)
	}
	g.as.props = buildPropImages()
}

func (o *ObjEd) update(u *UI) {
	pt := o.ed.pt
	pt.usePart = false
	cur := o.frames[o.frame]
	// ---- left panel
	u.panel(4, 32, 236, 684)
	y := 40
	u.text(12, y, "Objetos guardados", uiDim)
	if i := u.list(12, y+14, 218, 10, o.list, o.sel, &o.scroll); i >= 0 {
		o.sel = i
		o.loadByName(o.list[i])
	}
	y += 14 + 10*18 + 10
	u.text(12, y, "Nombre (carpeta)", uiDim)
	u.input("objname", 12, y+14, 218, &o.name, 24)
	y += 42
	if u.button(12, y, 106, 22, "Guardar", false) {
		o.save()
	}
	if u.button(124, y, 106, 22, "Nuevo en blanco", false) {
		o.newBlank()
		o.sel = -1
	}
	y += 28
	u.text(12, y, "Categoría", uiDim)
	if u.button(84, y-4, 146, 20, o.meta.Category, false) {
		o.meta.Category = cycleStr(o.meta.Category, objCategories)
	}
	y += 24
	w, h := cur.w, cur.h
	u.text(12, y, fmt.Sprintf("Tamaño %dx%d", w, h), uiDim)
	rs := func(nw, nh int) {
		for i, f := range o.frames {
			nb := newPix(nw, nh)
			for yy := 0; yy < f.h && yy < nh; yy++ {
				for xx := 0; xx < f.w && xx < nw; xx++ {
					copy(nb.p[(yy*nw+xx)*4:(yy*nw+xx)*4+4], f.p[(yy*f.w+xx)*4:(yy*f.w+xx)*4+4])
				}
			}
			o.frames[i] = nb
		}
		o.ed.pt.clearHistory()
		o.cv.fit(nw, nh)
		o.cv.setPix(o.frames[o.frame])
	}
	if u.button(110, y-4, 24, 20, "W-", false) && w > 8 {
		rs(w-4, h)
	}
	if u.button(136, y-4, 24, 20, "W+", false) && w < 200 {
		rs(w+4, h)
	}
	if u.button(166, y-4, 24, 20, "H-", false) && h > 8 {
		rs(w, h-4)
	}
	if u.button(192, y-4, 24, 20, "H+", false) && h < 200 {
		rs(w, h+4)
	}
	y += 26
	u.text(12, y, fmt.Sprintf("Fotograma %d/%d", o.frame+1, len(o.frames)), uiDim)
	if u.button(120, y-4, 26, 20, "◄", false) {
		o.frame = (o.frame + len(o.frames) - 1) % len(o.frames)
		o.cv.setPix(o.frames[o.frame])
		pt.clearHistory()
	}
	if u.button(148, y-4, 26, 20, "►", false) {
		o.frame = (o.frame + 1) % len(o.frames)
		o.cv.setPix(o.frames[o.frame])
		pt.clearHistory()
	}
	if u.button(176, y-4, 26, 20, "+", false) {
		cp := newPix(cur.w, cur.h)
		copy(cp.p, cur.p)
		o.frames = append(o.frames, cp)
		o.frame = len(o.frames) - 1
		o.cv.setPix(cp)
		pt.clearHistory()
	}
	if u.button(204, y-4, 26, 20, "-", false) && len(o.frames) > 1 {
		o.frames = append(o.frames[:o.frame], o.frames[o.frame+1:]...)
		o.frame = clampI(o.frame, 0, len(o.frames)-1)
		o.cv.setPix(o.frames[o.frame])
		pt.clearHistory()
	}
	y += 30
	u.text(12, y, "PROPIEDADES", uiAccent)
	y += 18
	u.toggle(12, y, 106, "Sólido", &o.meta.Solid)
	u.toggle(124, y, 106, "Esconderse", &o.meta.Hide)
	u.tooltip(124, y, 106, 20, "El jugador puede esconderse o tirar cuerpos aquí")
	y += 24
	u.toggle(12, y, 106, "Teléfono", &o.meta.Phone)
	u.toggle(124, y, 106, "Ponche/buffet", &o.meta.Poison)
	y += 24
	u.toggle(12, y, 106, "Farola (luz)", &o.meta.Light)
	if u.button(124, y, 106, 20, "Trampa: "+orDash(o.meta.Trap), o.meta.Trap != "") {
		o.meta.Trap = cycleStr(o.meta.Trap, []string{"", "explode", "electric"})
	}
	y += 24
	u.toggle(12, y, 218, "Fotogramas = variantes", &o.meta.Variants)
	y += 24
	u.text(12, y, fmt.Sprintf("Animación: %.1f fps", o.meta.FPS), uiDim)
	if u.button(150, y-4, 36, 20, "-", false) && o.meta.FPS > 0 {
		o.meta.FPS -= 0.5
	}
	if u.button(190, y-4, 36, 20, "+", false) {
		o.meta.FPS += 0.5
	}
	y += 24
	u.text(12, y, fmt.Sprintf("Orden de dibujo: %d", o.meta.SortOff), uiDim)
	if u.button(150, y-4, 36, 20, "-", false) {
		o.meta.SortOff -= 4
	}
	if u.button(190, y-4, 36, 20, "+", false) {
		o.meta.SortOff += 4
	}
	u.tooltip(12, y-4, 214, 20, "Negativo: se dibuja detrás de los personajes (alfombras, escenarios)")
	y += 24
	u.text(12, y, fmt.Sprintf("Cruza la calle (vel): %.0f", o.meta.Ambient), uiDim)
	if u.button(168, y-4, 28, 20, "-", false) && o.meta.Ambient > 0 {
		o.meta.Ambient -= 10
	}
	if u.button(198, y-4, 28, 20, "+", false) {
		o.meta.Ambient += 10
	}
	y += 24
	u.text(12, y, "Uso:", uiDim)
	if u.button(50, y-4, 180, 20, "puntos de uso: "+orDash(o.meta.UseKind), o.meta.UseKind != "") {
		o.meta.UseKind = cycleStr(o.meta.UseKind, useKinds)
	}
	// ---- mode buttons
	modes := []string{"Pintar", "Ancla (pies)", "Huella colisión", "Puntos de uso"}
	for i, m := range modes {
		if u.button(248+i*128, 36, 124, 22, m, o.mode == i) {
			o.mode = i
		}
	}
	hint := []string{"Pinta el sprite. Rueda: zoom.", "Clic: coloca el ancla (punto donde se apoya el objeto).", "Arrastra: dibuja el rectángulo que bloquea el paso.", "Clic: añade un punto donde se colocan los invitados. Clic derecho: quitar."}[o.mode]
	u.text(248, 64, hint, uiDim)
	// ---- canvas
	if o.cv.img == nil {
		o.cv.setPix(cur)
	}
	if o.cv.inView(u.mx, u.my) {
		if u.wheel > 0 {
			o.cv.zoomAt(u.mx, u.my, o.cv.zoom+1)
		} else if u.wheel < 0 {
			o.cv.zoomAt(u.mx, u.my, o.cv.zoom-1)
		}
	}
	panning := u.mdown || (ebiten.IsKeyPressed(ebiten.KeySpace) && u.down)
	if panning && o.cv.inView(u.mx, u.my) && (o.lastMX != 0 || o.lastMY != 0) {
		o.cv.px += u.mx - o.lastMX
		o.cv.py += u.my - o.lastMY
	}
	o.lastMX, o.lastMY = u.mx, u.my
	if u.focus == "" {
		ctrl := ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta)
		if ctrl && inpututil.IsKeyJustPressed(ebiten.KeyZ) {
			pt.doUndo(cur)
			o.cv.dirty = true
		}
		if ctrl && inpututil.IsKeyJustPressed(ebiten.KeyY) {
			pt.doRedo(cur)
			o.cv.dirty = true
		}
	}
	x, yy := o.cv.at(u.mx, u.my)
	inside := o.cv.inView(u.mx, u.my) && !panning
	switch o.mode {
	case 0:
		if !panning && pt.handle(u, &o.cv, cur, false) {
			o.cv.dirty = true
		}
	case 1:
		if inside && u.down {
			o.anchor = [2]int{clampI(x, 0, cur.w), clampI(yy, 0, cur.h)}
		}
	case 2:
		if inside && u.pressed {
			o.dragFoot, o.fx0, o.fy0 = true, x, yy
		}
		if o.dragFoot && u.down {
			x0, y0 := minI(o.fx0, x), minI(o.fy0, yy)
			x1, y1 := maxI(o.fx0, x), maxI(o.fy0, yy)
			o.foot = [4]int{x0, y0, x1 - x0 + 1, y1 - y0 + 1}
		}
		if u.released {
			o.dragFoot = false
		}
	case 3:
		if inside && u.pressed {
			o.usePts = append(o.usePts, [2]int{x, yy})
		}
		if inside && u.rpressed && len(o.usePts) > 0 {
			best, bd := 0, 1<<30
			for i, p := range o.usePts {
				if d := abs(p[0]-x) + abs(p[1]-yy); d < bd {
					best, bd = i, d
				}
			}
			o.usePts = append(o.usePts[:best], o.usePts[best+1:]...)
		}
	}
	if pt.drawing || u.released {
		o.cv.dirty = true
	}
	if o.cv.dirty {
		o.cv.refresh(cur)
	}
	cv := o.cv
	anchor, foot, use := o.anchor, o.foot, append([][2]int(nil), o.usePts...)
	u.add(func(d *ebiten.Image) {
		cv.draw(d, cur.w, cur.h, true)
		vp := d.SubImage(imageRect(cv.x, cv.y, cv.vw, cv.vh)).(*ebiten.Image)
		sx := func(px int) float64 { return float64(cv.x + cv.px + px*cv.zoom) }
		sy := func(py int) float64 { return float64(cv.y + cv.py + py*cv.zoom) }
		pt.previewShape(func(px, py int) {
			fillRect(vp, sx(px), sy(py), float64(cv.zoom), float64(cv.zoom), color.RGBA{pt.col.R, pt.col.G, pt.col.B, 160})
		})
		// footprint
		fc := color.RGBA{80, 255, 120, 255}
		fillRect(vp, sx(foot[0]), sy(foot[1]), float64(foot[2]*cv.zoom), 2, fc)
		fillRect(vp, sx(foot[0]), sy(foot[1]+foot[3])-2, float64(foot[2]*cv.zoom), 2, fc)
		fillRect(vp, sx(foot[0]), sy(foot[1]), 2, float64(foot[3]*cv.zoom), fc)
		fillRect(vp, sx(foot[0]+foot[2])-2, sy(foot[1]), 2, float64(foot[3]*cv.zoom), fc)
		fillRect(vp, sx(foot[0]), sy(foot[1]), float64(foot[2]*cv.zoom), float64(foot[3]*cv.zoom), color.RGBA{80, 255, 120, 30})
		// anchor
		ac := color.RGBA{255, 60, 220, 255}
		fillRect(vp, sx(anchor[0])-8, sy(anchor[1])-1, 17, 2, ac)
		fillRect(vp, sx(anchor[0])-1, sy(anchor[1])-8, 2, 17, ac)
		for _, p := range use {
			fillRect(vp, sx(p[0])-3, sy(p[1])-3, 7, 7, color.RGBA{255, 220, 40, 255})
		}
	})
	u.frame(cv.x-1, cv.y-1, cv.vw+2, cv.vh+2, uiLine)
	u.text(cv.x, cv.y+cv.vh+4, "Verde: huella de colisión · Magenta: ancla · Amarillo: puntos de uso", uiDim)
	// ---- right: palette + preview
	u.panel(776, 32, 500, 684)
	u.text(784, 38, "HERRAMIENTAS", uiAccent)
	u.paletteUI(pt, 784, 56, false)
	o.preview(u, 1022, 56)
}

func (o *ObjEd) preview(u *UI, x, y int) {
	u.text(x, y-18, "VISTA PREVIA", uiAccent)
	if o.meta.FPS > 0 {
		o.prevT += 1.0 / 60
	}
	fi := o.frame
	if o.meta.FPS > 0 && !o.meta.Variants {
		fi = int(o.prevT*o.meta.FPS) % len(o.frames)
	}
	img := o.frames[fi]
	if o.refCS == nil {
		spec := playerSpec()
		ca := charAssetFromSpec(spec, "ref", "main_character", "")
		o.refCS = newRigCharSet(ca)
	}
	ref, _ := o.refCS.img(AnimIdle, DirDown, 0)
	sc := 3
	if img.w > 80 {
		sc = 2
	}
	if img.w > 130 {
		sc = 1
	}
	ground := y + 300
	if o.prevImg == nil || o.prevImg.Bounds().Dx() != img.w || o.prevImg.Bounds().Dy() != img.h {
		o.prevImg = ebiten.NewImage(img.w, img.h)
	}
	o.prevImg.WritePixels(img.p)
	im := o.prevImg
	ox := x + 10
	cx := x + 10 + img.w*sc + 24
	if cx+charW*sc > x+250 {
		cx = x + 250 - charW*sc
	}
	anchor := o.anchor
	u.add(func(d *ebiten.Image) {
		fillRect(d, float64(x), float64(y), 250, 340, rgb(0x2a3a30))
		fillRect(d, float64(x), float64(ground), 250, 1, rgb(0x6a8a70))
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(float64(sc), float64(sc))
		op.GeoM.Translate(float64(ox), float64(ground-anchor[1]*sc))
		d.DrawImage(im, op)
		op2 := &ebiten.DrawImageOptions{}
		op2.GeoM.Scale(float64(sc), float64(sc))
		op2.GeoM.Translate(float64(cx), float64(ground-charBase*sc+sc))
		d.DrawImage(ref, op2)
	})
	u.text(x, y+346, "El muñeco azul es el protagonista:", uiDim)
	u.text(x, y+360, "compara el tamaño.", uiDim)
	u.text(x, y+380, fmt.Sprintf("Ampliación x%d", sc), uiDim)
}

var _ = sort.Strings

// importDrops: the first PNG replaces the current frame (the canvas takes its size); more PNGs become new frames.
func (o *ObjEd) importDrops(drops []droppedImage) {
	first := drops[0].b
	if len(o.frames) == 1 || first.w != o.frames[o.frame].w || first.h != o.frames[o.frame].h {
		if first.w != o.frames[0].w || first.h != o.frames[0].h {
			o.frames = []*pixBuf{newPix(first.w, first.h)}
			o.frame = 0
			o.anchor = [2]int{first.w / 2, first.h - 2}
			o.foot = [4]int{first.w/2 - first.w/3, first.h - 8, first.w * 2 / 3, 6}
		}
	}
	for i, d := range drops {
		var dst *pixBuf
		if i == 0 {
			dst = o.frames[o.frame]
		} else {
			dst = newPix(o.frames[0].w, o.frames[0].h)
			o.frames = append(o.frames, dst)
		}
		for k := range dst.p {
			dst.p[k] = 0
		}
		for y := 0; y < d.b.h && y < dst.h; y++ {
			for x := 0; x < d.b.w && x < dst.w; x++ {
				copy(dst.p[(y*dst.w+x)*4:(y*dst.w+x)*4+4], d.b.p[(y*d.b.w+x)*4:(y*d.b.w+x)*4+4])
			}
		}
	}
	o.ed.pt.clearHistory()
	o.cv.fit(o.frames[0].w, o.frames[0].h)
	o.cv.setPix(o.frames[o.frame])
	o.ed.say("Imagen importada. Marca el ancla y la huella de colisión")
}
