package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func jsonUnmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }

var usableActions = []segItem{{"", "Nada"}, {"", "Beber"}, {"", "Baño"}, {"sofa", "Sentarse"}, {"", "Esconderse"}, {"", "Teléfono"}, {"", "Envenenable"}}

func (o *ObjEd) actionIndex() int {
	m := &o.meta
	switch {
	case m.Phone:
		return 5
	case m.Hide:
		return 4
	case m.Poison:
		return 6
	case m.UseKind == "drink":
		return 1
	case m.UseKind == "toilet":
		return 2
	case m.UseKind == "lounge":
		return 3
	}
	return 0
}

func (o *ObjEd) setAction(i int) {
	m := &o.meta
	m.Hide, m.Phone, m.Poison, m.UseKind = false, false, false, ""
	switch i {
	case 1:
		m.UseKind = "drink"
	case 2:
		m.UseKind = "toilet"
	case 3:
		m.UseKind = "lounge"
	case 4:
		m.Hide = true
	case 5:
		m.Phone = true
	case 6:
		m.Poison, m.UseKind = true, "drink"
	}
}

var lightColors = []uint32{0xffb85a, 0xfff0a0, 0xffffff, 0x70c8ff, 0xff6aa8, 0xa070ff, 0x60e090, 0xff5a4a}

const (
	obColX = 8
	obColW = 268
	obRX   = 796
)

func (o *ObjEd) update(u *UI) {
	pt := o.ed.pt
	pt.usePart, pt.partOnly, pt.mirror, pt.brush = false, false, false, 1
	if o.importCh != nil {
		select {
		case d, ok := <-o.importCh:
			if d != nil {
				o.ed.requestImport([]droppedImage{*d})
			}
			if !ok || d != nil {
				o.importCh = nil
			}
		default:
		}
	}
	o.leftColumn(u)
	o.canvasArea(u)
	o.rightColumn(u)
}

func (o *ObjEd) leftColumn(u *UI) {
	x, w := obColX, obColW
	u.panel(x-4, edTop+4, w+8, edH-edTop-8)
	y := edTop + 12
	if i := u.seg(x+2, y, w-4, 34, []segItem{{"sofa", "Objetos"}, {"paw", "Animales"}}, o.view); i >= 0 && i != o.view {
		if i == 1 {
			o.setView(1, 0)
		} else {
			o.setView(0, 0)
		}
	}
	y += 44
	if o.view == 0 {
		u.heading(x+4, y, w-8, "TIPO DE OBJETO")
		y += 20
		bw := (w - 6) / 2
		for i, t := range objTypes {
			bx, by := x+(i%2)*(bw+6), y+(i/2)*34
			if u.iconBtn(bx, by, bw, 30, t.icon, t.label, o.typeIdx == i, t.tip) && o.typeIdx != i {
				o.setView(0, i)
			}
		}
		y += ((len(objTypes)+1)/2)*34 + 4
	}
	t := o.curType()
	u.heading(x+4, y, w-8, fmt.Sprintf("%s (%d)", t.label, len(o.list)))
	y += 20
	gridH := edH - y - 150
	budget := 8
	mk := func(i int) thumb {
		n := o.list[i]
		return thumb{label: n, lazy: func() *ebiten.Image {
			if _, ok := o.thumbs[o.ed.spriteDir("objetos", n)]; !ok {
				if budget <= 0 {
					return nil
				}
				budget--
			}
			return o.thumbFor(n)
		}}
	}
	if i := u.searchGrid("objfind", x, y, w, gridH, 4, &o.find, o.list, mk, o.sel, &o.scroll); i >= 0 {
		o.sel = i
		o.loadByName(o.list[i])
	}
	y += gridH + 8
	u.text(x+4, y+4, "Nombre", uiDim)
	u.input("objname", x+54, y, w-54, &o.name, 24)
	y += 28
	bw3 := (w - 8) / 3
	if u.iconBtn(x, y, bw3, 34, "save", "Guardar", false, "Guardar este objeto (Ctrl+S). Cambia el nombre para guardar una copia") {
		o.save()
	}
	if u.iconBtn(x+bw3+4, y, bw3, 34, "new", "Nuevo", false, "Empezar un objeto nuevo con ajustes de ejemplo para este tipo") {
		o.newBlank()
		o.sel = -1
	}
	if u.iconBtn(x+2*(bw3+4), y, bw3, 34, "import", "Importar", false, "Cargar una imagen PNG, JPG o PDF (o arrástrala a la ventana)") && o.importCh == nil {
		o.importCh = pickImageAsync()
	}
	y += 40
	if o.dir == "" {
		u.text(x+4, y, "Todavía sin guardar", uiDim)
	} else {
		u.text(x+4, y, "Guardado en la carpeta del juego", uiDim)
	}
}

func (o *ObjEd) canvasArea(u *UI) {
	pt := o.ed.pt
	x0 := obColX + obColW + 12
	u.panel(x0-4, edTop+4, obRX-x0-4, edH-edTop-8)
	if i := u.seg(x0+4, edTop+10, 170, 28, []segItem{{"pencil", "Pintar"}, {"anchor", "Lógica"}}, o.mode); i >= 0 {
		o.mode = i
		if i == 1 {
			o.ltool = 0
		}
	}
	// frames
	cur := o.frames[o.frame]
	fx := x0 + 184
	u.textC(fx+66, edTop+17, fmt.Sprintf("Imagen %d de %d", o.frame+1, len(o.frames)), uiText)
	if u.iconBtn(fx+130, edTop+10, 28, 28, "left", "", false, "Imagen anterior") {
		o.frame = (o.frame + len(o.frames) - 1) % len(o.frames)
		o.cv.setPix(o.frames[o.frame])
		pt.clearHistory()
	}
	if u.iconBtn(fx+160, edTop+10, 28, 28, "right", "", false, "Imagen siguiente") {
		o.frame = (o.frame + 1) % len(o.frames)
		o.cv.setPix(o.frames[o.frame])
		pt.clearHistory()
	}
	if u.iconBtn(fx+192, edTop+10, 28, 28, "plus", "", false, "Añadir una imagen (copia de esta): para animar o para variantes") {
		cp := newPix(cur.w, cur.h)
		copy(cp.p, cur.p)
		o.frames = append(o.frames, cp)
		o.frame = len(o.frames) - 1
		o.cv.setPix(cp)
		pt.clearHistory()
	}
	if u.iconBtn(fx+222, edTop+10, 28, 28, "trash", "", false, "Quitar esta imagen") && len(o.frames) > 1 {
		o.frames = append(o.frames[:o.frame], o.frames[o.frame+1:]...)
		o.frame = clampI(o.frame, 0, len(o.frames)-1)
		o.cv.setPix(o.frames[o.frame])
		pt.clearHistory()
	}
	cur = o.frames[o.frame]
	o.cv.x, o.cv.y, o.cv.vw, o.cv.vh = x0+4, edTop+46, 290, 560
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
	inside := o.cv.inView(u.mx, u.my) && !panning && u.grab == ""
	if o.mode == 0 {
		if !panning && pt.handle(u, &o.cv, cur, false) {
			o.cv.dirty = true
		}
	} else {
		switch o.ltool {
		case 0:
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
		case 1:
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
	}
	if pt.drawing || u.released {
		o.cv.dirty = true
	}
	if o.cv.dirty {
		o.cv.refresh(cur)
	}
	cv := o.cv
	anchor, foot, use := o.anchor, o.foot, append([][2]int(nil), o.usePts...)
	logic := o.mode == 1
	ltool := o.ltool
	u.add(func(d *ebiten.Image) {
		cv.draw(d, cur.w, cur.h, true)
		vp := d.SubImage(imageRect(cv.x, cv.y, cv.vw, cv.vh)).(*ebiten.Image)
		sx := func(px int) float64 { return float64(cv.x + cv.px + px*cv.zoom) }
		sy := func(py int) float64 { return float64(cv.y + cv.py + py*cv.zoom) }
		if !logic {
			pt.previewShape(func(px, py int) {
				fillRect(vp, sx(px), sy(py), float64(cv.zoom), float64(cv.zoom), color.RGBA{pt.col.R, pt.col.G, pt.col.B, 160})
			})
		}
		a := uint8(70)
		if logic {
			a = 255
		}
		fc := color.RGBA{80, 255, 120, a}
		fillRect(vp, sx(foot[0]), sy(foot[1]), float64(foot[2]*cv.zoom), 2, fc)
		fillRect(vp, sx(foot[0]), sy(foot[1]+foot[3])-2, float64(foot[2]*cv.zoom), 2, fc)
		fillRect(vp, sx(foot[0]), sy(foot[1]), 2, float64(foot[3]*cv.zoom), fc)
		fillRect(vp, sx(foot[0]+foot[2])-2, sy(foot[1]), 2, float64(foot[3]*cv.zoom), fc)
		if logic {
			fillRect(vp, sx(foot[0]), sy(foot[1]), float64(foot[2]*cv.zoom), float64(foot[3]*cv.zoom), color.RGBA{80, 255, 120, 40})
		}
		ac := color.RGBA{255, 60, 220, a}
		fillRect(vp, sx(anchor[0])-8, sy(anchor[1])-1, 17, 2, ac)
		fillRect(vp, sx(anchor[0])-1, sy(anchor[1])-8, 2, 17, ac)
		for _, p := range use {
			fillRect(vp, sx(p[0])-4, sy(p[1])-4, 9, 9, color.RGBA{255, 220, 40, a})
		}
		hx, hy := cv.at(ebiten.CursorPosition())
		if !logic && cv.inView(ebiten.CursorPosition()) && hx >= 0 && hy >= 0 && hx < cur.w && hy < cur.h {
			fillRect(vp, sx(hx), sy(hy), float64(cv.zoom), 1, uiAccent)
			fillRect(vp, sx(hx), sy(hy+1)-1, float64(cv.zoom), 1, uiAccent)
		}
		_ = ltool
	})
	u.roundFrame(cv.x-1, cv.y-1, cv.vw+2, cv.vh+2, uiLine)
	u.text(cv.x, cv.y+cv.vh+6, "Rueda: zoom · Espacio+arrastrar: mover · Ctrl+Z: deshacer", uiDim)
	tx := cv.x + cv.vw + 10
	if o.mode == 0 {
		o.paintTools(u, tx, edTop+46)
	} else {
		o.logicTools(u, tx, edTop+46)
	}
}

func (o *ObjEd) paintTools(u *UI, x, y int) {
	pt := o.ed.pt
	w := 180
	u.heading(x, y, w, "HERRAMIENTAS")
	y += 20
	for i, t := range []struct {
		tool       int
		icon, name string
	}{{ToolPencil, "pencil", "Lápiz (P)"}, {ToolEraser, "eraser", "Goma (E)"}, {ToolPicker, "picker", "Cuentagotas (I)"}} {
		if u.iconBtn(x+i*61, y, 57, 44, t.icon, "", pt.tool == t.tool, t.name) {
			pt.tool = t.tool
		}
	}
	y += 50
	cur := o.frames[o.frame]
	if u.iconBtn(x, y, 88, 26, "undo", "Deshacer", false, "Ctrl+Z") {
		pt.doUndo(cur)
		o.cv.dirty = true
	}
	if u.iconBtn(x+92, y, 88, 26, "redo", "Rehacer", false, "Ctrl+Y") {
		pt.doRedo(cur)
		o.cv.dirty = true
	}
	y += 36
	u.heading(x, y, w, "COLOR")
	y += 20
	y = u.simplePalette(pt, x, y, w)
	u.canvasOptions(x, y+8, w)
}

func (o *ObjEd) logicTools(u *UI, x, y int) {
	w := 180
	u.heading(x, y, w, "DÓNDE Y CÓMO")
	y += 22
	tools := []struct{ icon, label, tip string }{
		{"anchor", "Punto de apoyo", "Marca dónde toca el suelo (los pies del objeto)"},
		{"target", "Zona de uso", "Marca dónde se colocan los invitados al usarlo"},
		{"foot", "Colisión", "Dibuja el rectángulo que bloquea el paso"},
	}
	for i, t := range tools {
		if u.iconBtn(x, y, w, 40, t.icon, t.label, o.ltool == i, t.tip) {
			o.ltool = i
		}
		y += 46
	}
	y += 6
	hints := [][]string{
		{"Haz clic donde apoya el", "objeto en el suelo."},
		{"Clic: añade un punto.", "Clic derecho: quita el", "más cercano."},
		{"Arrastra para dibujar el", "rectángulo que bloquea", "el paso."},
	}
	for i, l := range hints[o.ltool] {
		u.text(x, y+i*15, l, uiDim)
	}
	y += 60
	if o.ltool == 1 {
		if u.iconBtn(x, y, w, 28, "trash", "Quitar los puntos", false, "") {
			o.usePts = nil
		}
		y += 34
		if o.meta.UseKind == "" && !o.meta.Hide && !o.meta.Phone {
			u.text(x, y, "Este objeto no se usa:", uiRed)
			u.text(x, y+14, "elígelo en Propiedades.", uiRed)
		}
	}
	y += 50
	u.text(x, y, "Verde: colisión", uiGood)
	u.text(x, y+15, "Magenta: apoyo", rgb(0xff6adc))
	u.text(x, y+30, "Amarillo: uso", uiAccent)
}

func (o *ObjEd) toggleRow(u *UI, x, y, w int, label string, v *bool, tip string) bool {
	icon := ""
	if *v {
		icon = "check"
	}
	if u.iconBtn(x, y, w, 26, icon, label, *v, tip) {
		*v = !*v
		return true
	}
	return false
}

func (o *ObjEd) rightColumn(u *UI) {
	x := obRX
	w := edW - x - 8
	m := &o.meta
	t := o.curType()
	// ---- properties
	ph := 110
	switch t.key {
	case "usable":
		ph = 156
	case "light":
		ph = 190
	}
	if len(o.frames) > 1 {
		ph += 62
	} else if t.key == "animal" || t.key == "vehicle" {
		ph += 22
	}
	u.panel(x-4, edTop+4, w+8, ph)
	u.heading(x+4, edTop+12, w-8, "PROPIEDADES · "+t.label)
	y := edTop + 34
	switch t.key {
	case "vehicle":
		v := int(m.Ambient)
		u.text(x+6, y+4, "Velocidad", uiDim)
		if u.slider("vel", x+80, y+2, 220, &v, 20, 200, "") {
			m.Ambient = float64(v)
		}
		u.text(x+360, y+2, fmt.Sprintf("%d", v), uiText)
		y += 30
		u.text(x+6, y, "Pasa por la carretera de los escenarios que tienen calle.", uiDim)
		y += 24
	case "trap":
		u.text(x+6, y+8, "Efecto", uiDim)
		i := 0
		if m.Trap == "electric" {
			i = 1
		}
		if c := u.seg(x+60, y, 300, 28, []segItem{{"flame", "Explota"}, {"", "Electrocuta"}}, i); c >= 0 {
			m.Trap = []string{"explode", "electric"}[c]
		}
		y += 36
		u.text(x+6, y, "Explota: mata en un radio. Electrocuta: mata a quien nade.", uiDim)
		y += 24
	case "usable":
		u.text(x+6, y+2, "Qué se puede hacer con él", uiDim)
		y += 20
		cur := o.actionIndex()
		if c := u.seg(x+4, y, w-8, 26, usableActions[:4], map[bool]int{true: cur, false: -1}[cur < 4]); c >= 0 {
			o.setAction(c)
		}
		y += 30
		if c := u.seg(x+4, y, w-8, 26, usableActions[4:], map[bool]int{true: cur - 4, false: -1}[cur >= 4]); c >= 0 {
			o.setAction(c + 4)
		}
		y += 30
		switch cur {
		case 1, 2, 3, 6:
			u.text(x+6, y, "Marca en LÓGICA > Zona de uso dónde se ponen los invitados.", uiDim)
		case 4:
			u.text(x+6, y, "El jugador se esconde dentro y puede tirar cuerpos.", uiDim)
		case 5:
			u.text(x+6, y, "Cabina: se puede cortar el cable.", uiDim)
		}
		y += 22
	case "light":
		o.toggleRow(u, x+4, y, 200, "Emite luz", &m.Light, "")
		y += 32
		r := m.LightR
		if r == 0 {
			r = 110
		}
		a := m.LightA
		if a == 0 {
			a = 120
		}
		u.text(x+6, y+2, "Distancia", uiDim)
		if u.slider("lr", x+90, y, 200, &r, 30, 220, "") {
			m.LightR = r
		}
		y += 24
		u.text(x+6, y+2, "Intensidad", uiDim)
		if u.slider("la", x+90, y, 200, &a, 30, 255, "") {
			m.LightA = a
		}
		y += 24
		u.text(x+6, y+6, "Color", uiDim)
		for i, c := range lightColors {
			cx := x + 90 + i*28
			col := rgb(c)
			on := m.LightCol == [3]int{int(col.R), int(col.G), int(col.B)}
			if on {
				u.round(cx-2, y-1, 26, 26, uiAccent)
			}
			u.swatch(cx, y+1, 22, 22, col)
			if u.in(cx, y+1, 22, 22) && u.pressed {
				m.LightCol = [3]int{int(col.R), int(col.G), int(col.B)}
				if m.LightR == 0 {
					m.LightR = 110
				}
				if m.LightA == 0 {
					m.LightA = 120
				}
			}
		}
		y += 32
	case "animal":
		u.text(x+6, y, "Los animales no bloquean el paso.", uiDim)
		y += 22
	default:
		o.toggleRow(u, x+4, y, 200, "Bloquea el paso", &m.Solid, "Si no, se puede pasar por encima")
		behind := m.SortOff < 0
		if o.toggleRow(u, x+212, y, w-220, "Siempre al fondo", &behind, "Se dibuja detrás de las personas (alfombras)") {
			m.SortOff = 0
			if behind {
				m.SortOff = -200
			}
		}
		y += 34
	}
	// animation
	if len(o.frames) > 1 {
		mode := 0
		if m.Variants {
			mode = 1
		}
		if c := u.seg(x+4, y, 300, 26, []segItem{{"play", "Animación"}, {"image", "Variantes"}}, mode); c >= 0 {
			m.Variants = c == 1
		}
		y += 30
		if !m.Variants {
			f := int(m.FPS)
			if f == 0 {
				f = 4
				m.FPS = 4
			}
			u.text(x+6, y+2, "Velocidad", uiDim)
			if u.slider("fps", x+90, y, 200, &f, 1, 12, "") {
				m.FPS = float64(f)
			}
			u.text(x+320, y+2, fmt.Sprintf("%d imágenes/s", f), uiText)
			y += 24
		} else {
			u.text(x+6, y, "Cada imagen es una versión distinta del objeto.", uiDim)
			y += 20
		}
	} else if t.key == "animal" || t.key == "vehicle" {
		u.text(x+6, y, "Añade más imágenes (+) para animarlo.", uiDim)
	}
	// ---- preview
	py := edTop + 4 + ph + 8
	u.panel(x-4, py, w+8, edH-py-4)
	u.heading(x+4, py+8, w-8, "VISTA PREVIA")
	o.preview(u, x+4, py+30, w-8, edH-py-40)
}

func (o *ObjEd) preview(u *UI, x, y, w, h int) {
	if o.meta.FPS > 0 {
		o.prevT += 1.0 / 60
	}
	fi := o.frame
	if len(o.frames) > 1 && o.meta.FPS > 0 && !o.meta.Variants {
		fi = int(o.prevT*o.meta.FPS) % len(o.frames)
	}
	img := o.frames[fi]
	if o.refCS == nil {
		spec := playerSpec()
		ca := charAssetFromSpec(spec, "ref", "main_character", "")
		o.refCS = newRigCharSet(ca)
	}
	ref, _ := o.refCS.img(AnimIdle, DirDown, 0)
	sc := 4
	for (img.w*sc+40+charW*sc > w || img.h*sc > h-40) && sc > 1 {
		sc--
	}
	ground := y + h - 36
	if o.prevImg == nil || o.prevImg.Bounds().Dx() != img.w || o.prevImg.Bounds().Dy() != img.h {
		o.prevImg = ebiten.NewImage(img.w, img.h)
	}
	o.prevImg.WritePixels(img.p)
	im := o.prevImg
	ox := x + 16
	cx := ox + img.w*sc + 30
	anchor := o.anchor
	m := o.meta
	u.add(func(d *ebiten.Image) {
		fillRect(d, float64(x), float64(y), float64(w), float64(h), rgb(0x1c2a24))
		fillRect(d, float64(x), float64(ground), float64(w), float64(h-(ground-y)), rgb(0x24362c))
		fillRect(d, float64(x), float64(ground), float64(w), 1, rgb(0x6a8a70))
		if m.Light {
			r := float64(m.LightR)
			if r == 0 {
				r = 110
			}
			a := float64(m.LightA)
			if a == 0 {
				a = 120
			}
			lc := color.RGBA{uint8(m.LightCol[0]), uint8(m.LightCol[1]), uint8(m.LightCol[2]), 255}
			if m.LightCol == [3]int{} {
				lc = color.RGBA{150, 118, 52, 255}
			}
			cxl, cyl := float64(ox)+float64(img.w*sc)/2, float64(ground)-float64(anchor[1]*sc)
			for rr := r * 0.9; rr > 4; rr -= 6 {
				k := (1 - rr/r) * a / 255 * 0.9
				filledEllipse(d, cxl, cyl, rr*math.Min(1, float64(w)/(2*r)), rr*0.7*math.Min(1, float64(w)/(2*r)), color.RGBA{uint8(float64(lc.R) * k), uint8(float64(lc.G) * k), uint8(float64(lc.B) * k), uint8(40 * k)})
			}
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(float64(sc), float64(sc))
		op.GeoM.Translate(float64(ox), float64(ground-anchor[1]*sc))
		d.DrawImage(im, op)
		op2 := &ebiten.DrawImageOptions{}
		op2.GeoM.Scale(float64(sc), float64(sc))
		op2.GeoM.Translate(float64(cx), float64(ground-charBase*sc+sc))
		d.DrawImage(ref, op2)
	})
	u.text(x+4, y+h-16, "Al lado, el protagonista: así se ve el tamaño.", uiDim)
}
