package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"path/filepath"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ---------------------------------------------------------------------------
// The scenario editor: a list on the left, the big picture in the middle and
// the tools of the chosen drawing option on the right.
// ---------------------------------------------------------------------------

const (
	lvLX = 8
	lvLW = 240
	lvCX = 256
	lvCW = 724
	lvRX = 992
	lvRW = 280
)

var lvModes = []segItem{{"image", "Fondo"}, {"wall", "Muros"}, {"sofa", "Objetos"}, {"flag", "Puntos"}, {"slider", "Ajustes"}}

var lvZoomSteps = []float64{0.5, 0.75, 1, 1.5, 2, 3, 4}

var genLabels = map[string]string{
	"house": "Casa", "club": "Discoteca", "pub": "Pub", "cinema": "Cine", "mall": "Centro comercial", "market": "Supermercado",
	"plaza": "Plaza", "paris": "París", "bullring": "Plaza de toros", "stadium": "Estadio", "festival": "Festival", "funfair": "Feria",
	"roof": "Azotea", "beach": "Playa", "skatepark": "Skate park", "circuit": "Circuito", "station": "Estación espacial", "highway": "Autopista",
	"trainstation": "Estación de tren", "graveyard": "Cementerio", "casino": "Casino", "ski": "Estación de esquí", "yacht": "Crucero",
}

func (l *LevEd) fitZoom() {
	best := lvZoomSteps[0]
	for _, z := range lvZoomSteps {
		if screenW*z <= float64(l.cv.vw) && screenH*z <= float64(l.cv.vh) {
			best = z
		}
	}
	if l.prefZoom > 0 {
		best = l.prefZoom
	}
	l.cv.fz = best
	l.cv.px = int((float64(l.cv.vw) - screenW*best) / 2)
	l.cv.py = int((float64(l.cv.vh) - screenH*best) / 2)
}

func (l *LevEd) stepZoom(dir int, mx, my int) {
	cv := &l.cv
	cur := cv.zf()
	idx := 0
	for i, z := range lvZoomSteps {
		if z <= cur+0.001 {
			idx = i
		}
	}
	idx = clampI(idx+dir, 0, len(lvZoomSteps)-1)
	nz := lvZoomSteps[idx]
	if nz == cur {
		return
	}
	ix, iy := float64(mx-cv.x-cv.px)/cur, float64(my-cv.y-cv.py)/cur
	cv.fz = nz
	l.prefZoom = nz
	cv.px = mx - cv.x - int(ix*nz)
	cv.py = my - cv.y - int(iy*nz)
}

func (l *LevEd) thumbFor(name string) *ebiten.Image {
	if im, ok := l.thumbs[name]; ok {
		return im
	}
	b, err := loadPNG(filepath.Join(backgroundDirOf(l.ed.root, name), "background.png"))
	if err != nil {
		l.thumbs[name] = nil
		return nil
	}
	tw, th := 120, 68
	t := newPix(tw, th)
	for y := 0; y < th; y++ {
		for x := 0; x < tw; x++ {
			sx, sy := x*b.w/tw, y*b.h/th
			copy(t.p[(y*tw+x)*4:(y*tw+x)*4+4], b.p[(sy*b.w+sx)*4:(sy*b.w+sx)*4+4])
		}
	}
	im := ebiten.NewImage(tw, th)
	im.WritePixels(t.p)
	l.thumbs[name] = im
	return im
}

// newDefault builds a ready-to-use scenario from a generator, so a new one is never an empty field.
func (l *LevEd) newFromGenerator(kind string) {
	var def *LevelDef
	for try := 0; try < 6 && def == nil; try++ {
		def = genNamed(kind, rand.Int63n(100000), genLabels[kind])
	}
	if def == nil {
		l.ed.say("No se pudo generar ese escenario, prueba otra vez")
		return
	}
	name := "nuevo_" + kind
	la, err := exportLevelDef(l.ed.g, def, name)
	if err != nil {
		l.ed.say("No se pudo crear: " + err.Error())
		return
	}
	l.setLevel(la)
	l.name = name
	l.sel = -1
	l.ed.say("Escenario nuevo: " + genLabels[kind] + ". Cámbialo como quieras")
}

func (l *LevEd) update(u *UI) {
	if !l.started {
		// the editor never opens on an empty field: start from a generated scenario
		l.started = true
		if l.la != nil && len(l.la.Meta.Props) == 0 {
			l.newFromGenerator("house")
		}
	}
	pt := l.ed.pt
	pt.usePart, pt.partOnly, pt.mirror, pt.brush = false, false, false, 1
	if l.importCh != nil {
		select {
		case d, ok := <-l.importCh:
			if d != nil {
				l.importDrops([]droppedImage{*d})
			}
			if !ok || d != nil {
				l.importCh = nil
			}
		default:
		}
	}
	l.leftColumn(u)
	l.centre(u)
	m := l.mode
	switch m {
	case 0:
		l.fondoPanel(u)
	case 1:
		l.murosPanel(u)
	case 2:
		l.objetosPanel(u)
	case 3:
		l.puntosPanel(u)
	default:
		l.ajustesPanel(u)
	}
}

func (l *LevEd) leftColumn(u *UI) {
	x, w := lvLX, lvLW
	u.panel(x-4, edTop+4, w+8, edH-edTop-8)
	y := edTop + 12
	u.heading(x+4, y, w-8, fmt.Sprintf("MIS ESCENARIOS (%d)", len(l.list)))
	y += 20
	gridH := 250
	budget := 3
	mk := func(i int) thumb {
		n := l.list[i]
		return thumb{label: n, lazy: func() *ebiten.Image {
			if _, ok := l.thumbs[n]; !ok {
				if budget <= 0 {
					return nil
				}
				budget--
			}
			return l.thumbFor(n)
		}}
	}
	if len(l.list) == 0 {
		u.round(x, y, w, gridH, rgb(0x101018))
		u.roundFrame(x, y, w, gridH, uiLine)
		u.text(x+12, y+16, "Aún no has guardado ninguno.", uiDim)
		u.text(x+12, y+34, "Empieza con Nuevo, una", uiDim)
		u.text(x+12, y+50, "plantilla o un generador.", uiDim)
	} else if i := u.searchGrid("lvfind", x, y, w, gridH, 2, &l.find, l.list, mk, l.sel, &l.scroll); i >= 0 {
		l.sel = i
		l.loadByName(l.list[i])
	}
	y += gridH + 10
	u.text(x+4, y+4, "Nombre", uiDim)
	u.input("lvname", x+54, y, w-54, &l.name, 24)
	y += 30
	bw := (w - 4) / 2
	if u.iconBtn(x, y, bw, 34, "save", "Guardar", false, "Guardar este escenario (Ctrl+S)") {
		l.save()
	}
	if u.iconBtn(x+bw+4, y, bw, 34, "new", "Nuevo", false, "Un escenario nuevo, ya con paredes, objetos y puntos") {
		kinds := []string{"house", "club", "pub", "plaza", "roof", "beach"}
		l.newFromGenerator(kinds[rand.Intn(len(kinds))])
	}
	y += 40
	tplList := templateEntries()
	var tpl []segItem
	for _, e := range tplList {
		tpl = append(tpl, segItem{"world", e.title})
	}
	if i := u.menuBtn(x, y, bw, 34, "folder", "Plantilla", "lvtpl", tpl, "Copia uno de los escenarios del juego para cambiarlo"); i >= 0 && i < len(tplList) {
		def := tplList[i].def()
		if def == nil {
			l.ed.say("No se pudo crear la plantilla")
		} else if la, err := exportLevelDef(l.ed.g, def, "copia_"+safeName(def.Title)); err == nil {
			l.setLevel(la)
			l.name = la.Meta.Name
			l.sel = -1
			l.ed.say("Plantilla cargada: " + def.Title)
		}
	}
	var gens []segItem
	var genK []string
	for _, k := range genKinds {
		if genLabels[k] != "" {
			gens = append(gens, segItem{"dice", genLabels[k]})
			genK = append(genK, k)
		}
	}
	if i := u.menuBtn(x+bw+4, y, bw, 34, "dice", "Generador", "lvgen", gens, "Crea un escenario distinto cada vez"); i >= 0 && i < len(genK) {
		l.newFromGenerator(genK[i])
	}
	y += 40
	if u.iconBtn(x, y, w, 34, "image", "Importar imagen de fondo", false, "PNG o JPG (o arrástrala a la ventana)") && l.importCh == nil {
		l.importCh = pickImageAsync()
	}
	y += 44
	if u.iconBtn(x, y, w, 44, "play", "Probar escenario", false, "Guarda y lo juega; con Esc vuelves al editor") {
		l.save()
		l.ed.g.testLevel(safeName(l.name))
	}
	if l.ed.debug {
		l.debugBlock(u, x, y+54, w)
	}
}

func (l *LevEd) centre(u *UI) {
	m := &l.la.Meta
	pt := l.ed.pt
	u.panel(lvCX-4, edTop+4, lvCW+16, edH-edTop-8)
	if i := u.seg(lvCX+4, edTop+10, 470, 30, lvModes, l.mode); i >= 0 {
		l.mode = i
	}
	// zoom and "show walls"
	zx := lvCX + 486
	if u.iconBtn(zx, edTop+10, 28, 30, "minus", "", false, "Alejar") {
		l.stepZoom(-1, l.cv.x+l.cv.vw/2, l.cv.y+l.cv.vh/2)
	}
	u.textC(zx+56, edTop+19, fmt.Sprintf("%d%%", int(l.cv.zf()*100+0.5)), uiText)
	if u.iconBtn(zx+86, edTop+10, 28, 30, "plus", "", false, "Acercar") {
		l.stepZoom(1, l.cv.x+l.cv.vw/2, l.cv.y+l.cv.vh/2)
	}
	if l.mode != 1 {
		on := l.showLogic
		if u.iconBtn(zx+120, edTop+10, 90, 30, "eye", "Muros", on, "Ver los muros y las zonas encima") {
			l.showLogic = !l.showLogic
		}
	}
	cv := &l.cv
	cv.x, cv.y, cv.vw, cv.vh = lvCX+4, edTop+48, lvCW, 430
	if cv.fz == 0 || l.needFit {
		l.fitZoom()
		l.needFit = false
	}
	if cv.img == nil {
		cv.setPix(l.la.BG)
	}
	panning := u.mdown || (ebiten.IsKeyPressed(ebiten.KeySpace) && u.down)
	if cv.inView(u.mx, u.my) {
		if u.wheel > 0 {
			l.stepZoom(1, u.mx, u.my)
		} else if u.wheel < 0 {
			l.stepZoom(-1, u.mx, u.my)
		}
	}
	if panning && cv.inView(u.mx, u.my) && (l.lastMX != 0 || l.lastMY != 0) {
		cv.px += u.mx - l.lastMX
		cv.py += u.my - l.lastMY
	}
	l.lastMX, l.lastMY = u.mx, u.my
	x, yy := cv.at(u.mx, u.my)
	inside := cv.inView(u.mx, u.my) && !panning && u.grab == ""
	inCanvas := x >= 0 && yy >= 0 && x < screenW && yy < screenH
	if u.focus == "" {
		ctrl := ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta)
		if ctrl && inpututil.IsKeyJustPressed(ebiten.KeyZ) {
			if l.mode == 2 || l.mode == 3 {
				l.moveUndo()
			} else if l.mode == 0 {
				pt.doUndo(l.la.BG)
				cv.dirty = true
			} else if len(l.undoL) > 0 {
				s := l.undoL[len(l.undoL)-1]
				l.undoL = l.undoL[:len(l.undoL)-1]
				l.solid, l.zone = s[0], s[1]
			}
		}
		if ctrl && inpututil.IsKeyJustPressed(ebiten.KeyY) {
			if l.mode == 2 || l.mode == 3 {
				l.moveRedo()
			} else if l.mode == 0 {
				pt.doRedo(l.la.BG)
				cv.dirty = true
			}
		}
		l.moveKeys(u)
		for i := 0; i < 5; i++ {
			if inpututil.IsKeyJustPressed(ebiten.Key1 + ebiten.Key(i)) {
				l.mode = i
			}
		}
		if (inpututil.IsKeyJustPressed(ebiten.KeyDelete) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace)) && l.mode == 2 && l.objTool == 0 && l.selInst >= 0 && l.selInst < len(m.Props) {
			m.Props = append(m.Props[:l.selInst], m.Props[l.selInst+1:]...)
			l.selInst = -1
		}
	}
	switch l.mode {
	case 0:
		if !panning && pt.handle(u, cv, l.la.BG, false) {
			cv.dirty = true
		}
		if pt.drawing || u.released {
			cv.dirty = true
		}
	case 1:
		if inside && inCanvas {
			if u.pressed || u.rpressed {
				l.pushLogicUndo()
				if l.rectMode {
					l.rdrag, l.rx0, l.ry0 = true, x, yy
				}
			}
			if !l.rectMode && (u.down || u.rdown) {
				l.paintCells(x, yy, u.rdown && !u.down)
			}
		}
		if l.rdrag && (u.released || (!u.down && !u.rdown)) {
			l.fillCells(l.rx0, l.ry0, x, yy, !u.down && u.rdown)
			l.rdrag = false
		}
	case 2:
		l.updateObjects(u, x, yy, inside && inCanvas)
	case 3:
		l.updatePoints(u, x, yy, inside && inCanvas)
	}
	if l.mv.needDraw && !l.mv.drag && l.mv.zmode != 2 && !u.down {
		l.mv.needDraw = false
		l.redrawOriginal()
	}
	if cv.dirty {
		cv.refresh(l.la.BG)
	}
	l.drawCanvas(u)
	u.roundFrame(cv.x-1, cv.y-1, cv.vw+2, cv.vh+2, uiLine)
	// under the canvas: where the mouse is and what to do
	iy := cv.y + cv.vh + 10
	if inCanvas && cv.inView(u.mx, u.my) {
		u.text(cv.x+4, iy, fmt.Sprintf("x:%d  y:%d", x, yy), uiDim)
	}
	u.text(cv.x+130, iy, "Rueda: zoom · Espacio+arrastrar: mover · Ctrl+Z: deshacer · Teclas 1-5: cambiar de opción", uiDim)
	help := map[int][]string{
		0: {"FONDO: pinta el suelo y las paredes como quieras. Los objetos se colocan aparte (opción Objetos)."},
		1: {"MUROS: dibuja lo que bloquea el paso (muros, vallas, agua) y las zonas donde pasea y baila la gente.", "Clic izquierdo pinta, clic derecho borra."},
		2: {"OBJETOS: Colocar (clic), Mover (arrastrar objetos y suelos) o Zona (mover un rectángulo entero). Elige arriba a la derecha."},
		3: {"PUNTOS: dónde aparece el jugador, por dónde llega la policía, rutas de guardias, DJ, luces y animales. «Mover» los arrastra."},
		4: {"AJUSTES: título, invitados, hora del día y carretera."},
	}[l.mode]
	for i, s := range help {
		u.text(cv.x+4, iy+26+i*16, s, uiText)
	}
}

func (l *LevEd) drawCanvas(u *UI) {
	m := &l.la.Meta
	pt := l.ed.pt
	view := l.cv
	logic := l.showLogic || l.mode == 1
	solid, zone := l.solid, l.zone
	showPts := l.showPts || l.mode == 3
	props := append([]LevelProp(nil), m.Props...)
	meta := *m
	selInst := l.selInst
	mode := l.mode
	rd, rx0, ry0 := l.rdrag, l.rx0, l.ry0
	g := l.ed.g
	hls := l.highlights()
	u.add(func(d *ebiten.Image) {
		view.draw(d, screenW, screenH, false)
		vp := d.SubImage(imageRect(view.x, view.y, view.vw, view.vh)).(*ebiten.Image)
		z := view.zf()
		sx := func(px int) float64 { return float64(view.x+view.px) + float64(px)*z }
		sy := func(py int) float64 { return float64(view.y+view.py) + float64(py)*z }
		idx := make([]int, len(props))
		for i := range idx {
			idx[i] = i
		}
		sort.Slice(idx, func(a, b int) bool { return props[idx[a]].Y < props[idx[b]].Y })
		for _, i := range idx {
			p := props[i]
			imgs := g.as.props[p.Kind]
			if len(imgs) == 0 {
				fillRect(vp, sx(p.X)-4*z, sy(p.Y)-8*z, 8*z, 8*z, color.RGBA{255, 0, 255, 200})
				continue
			}
			pi := imgs[p.Var%len(imgs)]
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(z, z)
			op.GeoM.Translate(sx(p.X)-pi.ox*z, sy(p.Y)-pi.oy*z)
			vp.DrawImage(pi.img, op)
			if i == selInst && mode == 2 {
				w, h := float64(pi.img.Bounds().Dx())*z, float64(pi.img.Bounds().Dy())*z
				bx, by := sx(p.X)-pi.ox*z, sy(p.Y)-pi.oy*z
				fillRect(vp, bx, by, w, 1, uiAccent)
				fillRect(vp, bx, by+h, w, 1, uiAccent)
				fillRect(vp, bx, by, 1, h, uiAccent)
				fillRect(vp, bx+w, by, 1, h, uiAccent)
			}
			if mode == 2 {
				fillRect(vp, sx(p.X)-2, sy(p.Y)-2, 4, 4, color.RGBA{255, 60, 220, 255})
			}
		}
		if logic {
			cell := z * logicCell
			for cy, row := range solid {
				for cx, ch := range row {
					if ch == '.' {
						continue
					}
					for _, b := range logicBrushes {
						if b.layer == 0 && b.ch == ch {
							fillRect(vp, sx(cx*logicCell), sy(cy*logicCell), cell, cell, b.col)
						}
					}
				}
			}
			for cy, row := range zone {
				for cx, ch := range row {
					if ch == '.' {
						continue
					}
					for _, b := range logicBrushes {
						if b.layer == 1 && b.ch == ch {
							fillRect(vp, sx(cx*logicCell), sy(cy*logicCell), cell, cell, b.col)
						}
					}
				}
			}
		}
		if showPts {
			drawLevelPoints(vp, &meta, sx, sy, z, mode == 3)
		}
		for _, h := range hls {
			bx, by, bw, bh := sx(h.x), sy(h.y), float64(h.w)*z, float64(h.h)*z
			fillRect(vp, bx, by, bw, bh, color.RGBA{h.col.R * 38 / 255, h.col.G * 38 / 255, h.col.B * 38 / 255, 38}) // premultiplied
			fillRect(vp, bx, by, bw, 1, h.col)
			fillRect(vp, bx, by+bh-1, bw, 1, h.col)
			fillRect(vp, bx, by, 1, bh, h.col)
			fillRect(vp, bx+bw-1, by, 1, bh, h.col)
			if h.label != "" {
				g.plainText(vp, h.label, bx, by-14, h.col)
			}
		}
		if mode == 0 {
			pt.previewShape(func(px, py int) {
				fillRect(vp, sx(px), sy(py), math.Max(1, z), math.Max(1, z), color.RGBA{pt.col.R, pt.col.G, pt.col.B, 160})
			})
		}
		if rd && mode == 1 {
			fillRect(vp, sx(rx0), sy(ry0), 3, 3, uiAccent)
		}
		if mode == 1 {
			mx, my := view.at(ebiten.CursorPosition())
			fillRect(vp, sx(mx/logicCell*logicCell), sy(my/logicCell*logicCell), z*logicCell, 1, uiAccent)
			fillRect(vp, sx(mx/logicCell*logicCell), sy(my/logicCell*logicCell)+z*logicCell, z*logicCell, 1, uiAccent)
		}
		_ = ry0
	})
}

// ---- right-hand panels -----------------------------------------------------------

func (l *LevEd) fondoPanel(u *UI) {
	pt := l.ed.pt
	x := lvRX
	u.panel(x-4, edTop+4, lvRW+8, edH-edTop-8)
	y := edTop + 12
	u.heading(x+4, y, lvRW-8, "PINTAR EL FONDO")
	y += 22
	if l.overrideKey != "" {
		u.text(x+4, y, "Original del juego: este dibujo no se guarda", uiRed)
		y += 18
	}
	for i, t := range []struct {
		tool       int
		icon, name string
	}{{ToolPencil, "pencil", "Lápiz (P)"}, {ToolEraser, "eraser", "Goma (E)"}, {ToolPicker, "picker", "Cuentagotas (I)"}} {
		if u.iconBtn(x+i*92, y, 88, 48, t.icon, "", pt.tool == t.tool, t.name) {
			pt.tool = t.tool
		}
	}
	y += 56
	if u.iconBtn(x, y, 136, 28, "undo", "Deshacer", false, "Ctrl+Z") {
		pt.doUndo(l.la.BG)
		l.cv.dirty = true
	}
	if u.iconBtn(x+140, y, 136, 28, "redo", "Rehacer", false, "Ctrl+Y") {
		pt.doRedo(l.la.BG)
		l.cv.dirty = true
	}
	y += 40
	u.heading(x+4, y, lvRW-8, "COLOR")
	y += 22
	u.simplePalette(pt, x, y, lvRW-4)
}

func (l *LevEd) murosPanel(u *UI) {
	x := lvRX
	if l.wasRoad && l.brush < 9 {
		l.bsize = 2 // back from the wide road brush
	}
	l.wasRoad = l.brush >= 9
	u.panel(x-4, edTop+4, lvRW+8, edH-edTop-8)
	y := edTop + 12
	u.heading(x+4, y, lvRW-8, "OBSTÁCULOS")
	y += 22
	type bt struct {
		idx        int
		icon, text string
		tip        string
	}
	solids := []bt{
		{0, "wall", "Muro", "Bloquea el paso y la vista"},
		{1, "wall", "Valla", "Bloquea el paso, se ve a través"},
		{2, "", "Agua", "El jugador no entra; los invitados nadan si el escenario lo permite"},
		{3, "", "Zona lenta", "Se puede pisar pero a poca velocidad (barro, gente)"},
		{4, "", "Ventana", "No se camina por ella; con E se salta al otro lado"},
		{5, "eraser", "Borrar", "Quita muros, vallas, agua..."},
	}
	bw := (lvRW - 4) / 2
	for i, b := range solids {
		bx, by := x+(i%2)*(bw+4), y+(i/2)*34
		if u.iconBtn(bx, by, bw, 30, b.icon, b.text, l.brush == b.idx, b.tip) {
			l.brush = b.idx
		}
		if lb := logicBrushes[b.idx]; lb.ch != '.' {
			u.round(bx+bw-18, by+9, 12, 12, color.RGBA{lb.col.R, lb.col.G, lb.col.B, 255})
		}
	}
	y += 3*34 + 8
	u.heading(x+4, y, lvRW-8, "CARRETERA (1 CARRIL)")
	y += 22
	roads := []struct {
		idx       int
		icon, tip string
	}{
		{9, "left", "Carretera: los coches van hacia la izquierda"},
		{10, "right", "Carretera: los coches van hacia la derecha"},
		{11, "up", "Carretera: los coches van hacia arriba"},
		{12, "down", "Carretera: los coches van hacia abajo"},
	}
	// the buttons are laid out left, right, up, down; the brushes are right, left, up, down
	order := []int{1, 0, 2, 3}
	for i, o := range order {
		rb := roads[i]
		bi := 9 + o
		if u.iconBtn(x+i*((lvRW-4)/4), y, (lvRW-4)/4-3, 34, rb.icon, "", l.brush == bi, rb.tip) {
			l.brush = bi
			l.bsize = 9 // one lane is about as wide as a car
		}
	}
	y += 40
	u.text(x+4, y, "Pinta una franja: los coches", uiDim)
	u.text(x+4, y+15, "pasan en la dirección elegida.", uiDim)
	y += 36
	u.heading(x+4, y, lvRW-8, "ZONAS DE LA GENTE")
	y += 22
	zones := []bt{
		{6, "person", "Pasear", "Dónde se mueven los invitados"},
		{7, "music", "Bailar", "Dónde bailan"},
		{8, "eraser", "Borrar", "Quita las zonas"},
	}
	for i, b := range zones {
		bx := x + i*(lvRW-8)/3
		if u.iconBtn(bx, y, (lvRW-8)/3-2, 30, b.icon, b.text, l.brush == b.idx, b.tip) {
			l.brush = b.idx
		}
	}
	y += 44
	u.heading(x+4, y, lvRW-8, "PINCEL")
	y += 22
	u.stepper(x+4, y, "Tamaño", &l.bsize, 1, 10, 1)
	y += 30
	icon := ""
	if l.rectMode {
		icon = "check"
	}
	if u.iconBtn(x, y, lvRW-4, 28, icon, "Pintar rectángulos arrastrando", l.rectMode, "") {
		l.rectMode = !l.rectMode
	}
	y += 40
	if u.iconBtn(x, y, lvRW-4, 28, "trash", "Borrar todo esto", false, "Quita todos los muros y zonas") {
		l.pushLogicUndo()
		l.solid, l.zone = toGrid(blankRows()), toGrid(blankRows())
	}
}

func (l *LevEd) objetosPanel(u *UI) {
	m := &l.la.Meta
	x := lvRX
	u.panel(x-4, edTop+4, lvRW+8, edH-edTop-8)
	y := edTop + 12
	u.heading(x+4, y, lvRW-8, "OBJETOS")
	y += 22
	if i := u.seg(x, y, lvRW-4, 30, []segItem{{"pencil", "Colocar"}, {"hand", "Mover"}, {"target", "Zona"}}, l.objTool); i >= 0 && i != l.objTool {
		l.objTool = i
		l.mv.zhas, l.mv.zmode, l.mv.drag = false, 0, false
		l.selectItem(lvItem{})
	}
	y += 38
	switch l.objTool {
	case 1:
		l.movePanel(u, x, y, maskProps|l.groundMask())
		return
	case 2:
		l.zonePanel(u, x, y)
		return
	}
	mkKind := func(i int) thumb {
		t := thumb{label: l.kinds[i]}
		if imgs := l.ed.g.as.props[l.kinds[i]]; len(imgs) > 0 {
			t.img = imgs[0].img
		}
		return t
	}
	gh := 380
	if i := u.searchGrid("kindfind", x, y, lvRW-4, gh, 4, &l.kindFind, l.kinds, mkKind, l.kindSel, &l.kindScr); i >= 0 {
		l.kindSel = i
		l.curVar = 0
	}
	y += gh + 10
	u.stepper(x+4, y, "Versión", &l.curVar, 0, 9, 1)
	if l.selInst >= 0 && l.selInst < len(m.Props) {
		m.Props[l.selInst].Var = l.curVar
	}
	y += 30
	u.text(x+4, y, fmt.Sprintf("%d objetos en el escenario", len(m.Props)), uiDim)
	u.text(x+4, y+16, "Clic derecho: borra el más cercano", uiDim)
	u.text(x+4, y+32, "Supr: borra el seleccionado", uiDim)
}

// groundMask: the ground and the water can be moved only in the game's own scenarios.
func (l *LevEd) groundMask() int {
	if l.overrideKey != "" {
		return maskGround
	}
	return 0
}

// movePanel: the list of everything that can be grabbed, and how to move it.
func (l *LevEd) movePanel(u *UI, x, y, mask int) {
	m := &l.la.Meta
	items := l.listItems(mask)
	labels := make([]string, len(items))
	sel := -1
	for i, it := range items {
		labels[i] = it.label(m)
		if it == l.mv.sel {
			sel = i
		}
	}
	u.heading(x+4, y, lvRW-8, "ELEMENTOS DEL ESCENARIO")
	y += 22
	rows := 18
	if i := u.list(x, y, lvRW-4, rows, labels, sel, &l.mv.elScroll); i >= 0 {
		l.selectItem(items[i])
	}
	y += rows*18 + 8
	if l.mv.sel.kind == itProp && l.selInst >= 0 && l.selInst < len(m.Props) {
		u.stepper(x+4, y, "Versión", &l.curVar, 0, 9, 1)
		m.Props[l.selInst].Var = l.curVar
		y += 28
	}
	if l.mv.sel.kind != itNone {
		if u.iconBtn(x, y, lvRW-4, 28, "trash", "Quitar el seleccionado", false, "Supr") {
			l.deleteSelected()
		}
		y += 34
	}
	u.text(x+4, y, "Arrastra una cosa en el escenario.", uiDim)
	u.text(x+4, y+16, "Flechas: 1 celda (Mayús: 1 píxel).", uiDim)
	u.text(x+4, y+32, "Ctrl+Z / Ctrl+Y: deshacer y rehacer.", uiDim)
	if l.overrideKey != "" {
		u.text(x+4, y+52, "Original del juego: al soltar un suelo", uiAccent)
		u.text(x+4, y+68, "o el agua se repinta el fondo.", uiAccent)
	}
}

// zonePanel: moving a rectangle of the scenario with everything in it.
func (l *LevEd) zonePanel(u *UI, x, y int) {
	mv := &l.mv
	u.heading(x+4, y, lvRW-8, "MOVER UNA ZONA")
	y += 22
	for i, s := range []string{
		"1. Arrastra sobre el escenario para",
		"    elegir un rectángulo.",
		"2. Arrastra dentro para moverlo con",
		"    todo lo que lleva dentro.",
	} {
		u.text(x+4, y+i*16, s, uiDim)
	}
	y += 76
	if u.iconBtn(x, y, lvRW-4, 30, "plus", "Copiar en vez de mover", mv.zcopy, "También con Alt apretado") {
		mv.zcopy = !mv.zcopy
	}
	y += 36
	if mv.zhas {
		if u.iconBtn(x, y, lvRW-4, 30, "trash", "Vaciar la zona", false, "Supr") {
			l.zoneDelete()
		}
		y += 36
		r := mv.zrect
		u.text(x+4, y, fmt.Sprintf("Zona: %d x %d píxeles", r[2]-r[0], r[3]-r[1]), uiText)
		zc := l.captureZone(r)
		u.text(x+4, y+16, fmt.Sprintf("Dentro: %d objetos, %d puntos", len(zc.props), len(zc.pts)), uiDim)
		y += 38
	} else {
		u.text(x+4, y, "Aún no hay zona elegida.", uiDim)
		y += 24
	}
	u.text(x+4, y, "Lleva consigo: el dibujo del fondo,", uiDim)
	u.text(x+4, y+16, "muros, agua, suelo lento, carreteras,", uiDim)
	u.text(x+4, y+32, "zonas, objetos y puntos.", uiDim)
	u.text(x+4, y+54, "Flechas: 1 celda · Supr: vacía la zona", uiDim)
	u.text(x+4, y+70, "Esc: quita la selección", uiDim)
	if l.overrideKey != "" {
		u.text(x+4, y+94, "Original del juego: el dibujo del fondo", uiAccent)
		u.text(x+4, y+110, "no se mueve; sí suelos, agua, muros,", uiAccent)
		u.text(x+4, y+126, "objetos y puntos.", uiAccent)
	}
}

func (l *LevEd) puntosPanel(u *UI) {
	m := &l.la.Meta
	x := lvRX
	u.panel(x-4, edTop+4, lvRW+8, edH-edTop-8)
	y := edTop + 12
	u.heading(x+4, y, lvRW-8, "PUNTOS DEL ESCENARIO")
	y += 22
	tools := []struct{ icon, label, tip string }{
		{"flag", "Inicio", "Dónde aparece el jugador"},
		{"car", "Policía: llega", "Por dónde entra el coche de policía"},
		{"road", "Policía: aparca", "Dónde para el coche de policía"},
		{"star", "Helipuertos", "Si la policía llega en helicóptero (hasta 3)"},
		{"route", "Ruta de guardia", "Camino que recorre un guardia"},
		{"music", "DJ y músicos", "Dónde tocan"},
		{"bulb", "Luces", "Focos de colores"},
		{"paw", "Animales", "Dónde viven los animales"},
		{"road", "Zona de parada", "Dónde paran la policía y las ambulancias"},
		{"hand", "Mover", "Arrastra cualquier punto: inicio, policía, helipuertos, rutas, músicos, luces, animales"},
	}
	bw := (lvRW - 4) / 2
	for i, t := range tools {
		bx, by := x+(i%2)*(bw+4), y+(i/2)*34
		if u.iconBtn(bx, by, bw, 30, t.icon, t.label, l.ptTool == i, t.tip) {
			l.ptTool = i
		}
	}
	y += ((len(tools)+1)/2)*34 + 8
	u.rect(x, y-4, lvRW-4, 1, uiLine)
	switch l.ptTool {
	case ptMoveTool:
		l.movePanel(u, x, y+4, maskPoints)
	case 0:
		u.text(x+4, y, "Clic: dónde aparece el jugador.", uiDim)
	case 8:
		u.text(x+4, y, "Clic izquierdo: inicio del tramo.", uiDim)
		u.text(x+4, y+16, "Clic derecho: final del tramo.", uiDim)
		if u.iconBtn(x, y+40, lvRW-4, 28, "trash", "Quitar la zona", false, "") {
			m.ParkX0, m.ParkX1 = 0, 0
		}
	case 1, 2:
		u.text(x+4, y, "El coche entra por 'llega' y", uiDim)
		u.text(x+4, y+16, "aparca en 'aparca'. Usa la misma", uiDim)
		u.text(x+4, y+32, "altura en los dos, sobre la calle.", uiDim)
		l.policeSeg(u, x, y+54)
	case 3:
		u.text(x+4, y, "Si la policía llega en helicóptero,", uiDim)
		u.text(x+4, y+16, "aterriza aquí. El segundo es para", uiDim)
		u.text(x+4, y+32, "la ambulancia.", uiDim)
		l.policeSeg(u, x, y+54)
	case 4:
		u.text(x+4, y, fmt.Sprintf("Rutas: %d   Seleccionada: %d", len(m.Guards), l.routeIdx+1), uiText)
		if u.iconBtn(x, y+22, bw, 28, "plus", "Nueva", false, "") {
			m.Guards = append(m.Guards, nil)
			l.routeIdx = len(m.Guards) - 1
		}
		if u.iconBtn(x+bw+4, y+22, bw, 28, "trash", "Borrar", false, "") && len(m.Guards) > 0 {
			m.Guards = append(m.Guards[:l.routeIdx], m.Guards[l.routeIdx+1:]...)
			l.routeIdx = clampI(l.routeIdx, 0, maxI(0, len(m.Guards)-1))
		}
		if u.iconBtn(x, y+54, bw, 28, "left", "Anterior", false, "") && len(m.Guards) > 0 {
			l.routeIdx = (l.routeIdx + len(m.Guards) - 1) % len(m.Guards)
		}
		if u.iconBtn(x+bw+4, y+54, bw, 28, "right", "Siguiente", false, "") && len(m.Guards) > 0 {
			l.routeIdx = (l.routeIdx + 1) % len(m.Guards)
		}
		u.text(x+4, y+92, "Clic: añade un punto a la ruta.", uiDim)
		u.text(x+4, y+108, "Clic derecho: quita el último.", uiDim)
	case 5:
		cur := 0
		for i, r := range perfRoles {
			if r == l.role {
				cur = i
			}
		}
		var segs []segItem
		for _, r := range perfRoles {
			segs = append(segs, segItem{"", perfLabels[r]})
		}
		if i := u.seg(x, y, lvRW-4, 28, segs, cur); i >= 0 {
			l.role = perfRoles[i]
		}
		u.text(x+4, y+38, "Clic: coloca. Clic derecho: borra.", uiDim)
	case 7:
		if u.iconBtn(x, y, lvRW-4, 28, "paw", "Animal: "+animalKinds[l.animKind], false, "Clic para cambiar de animal") {
			l.animKind = (l.animKind + 1) % len(animalKinds)
		}
		u.stepper(x+4, y+36, "Zona", &l.animRad, 20, 150, 10)
		u.text(x+4, y+66, "Clic: coloca. Clic derecho: borra.", uiDim)
	case 6:
		u.text(x+4, y+2, "Color", uiDim)
		for i, c := range lightColors {
			cx := x + 50 + i*27
			col := rgb(c)
			if int(col.R) == l.lcol[0] && int(col.G) == l.lcol[1] && int(col.B) == l.lcol[2] {
				u.round(cx-2, y-2, 24, 24, uiAccent)
			}
			u.swatch(cx, y, 20, 20, col)
			if u.in(cx, y, 20, 20) && u.pressed {
				l.lcol = [3]int{int(col.R), int(col.G), int(col.B)}
			}
		}
		u.text(x+4, y+34, "Fuerza", uiDim)
		u.slider("lal", x+60, y+32, 160, &l.lalpha, 20, 255, "")
		u.text(x+4, y+58, "Radio", uiDim)
		u.slider("lrd", x+60, y+56, 160, &l.lrad, 10, 200, "")
		icon := ""
		if l.lpulse {
			icon = "check"
		}
		if u.iconBtn(x, y+84, lvRW-4, 28, icon, "Parpadea", l.lpulse, "") {
			l.lpulse = !l.lpulse
		}
		u.text(x+4, y+122, "Clic: coloca. Clic derecho: borra.", uiDim)
	}
}

func (l *LevEd) policeSeg(u *UI, x, y int) {
	m := &l.la.Meta
	cur := 0
	if m.Police == "heli" {
		cur = 1
	}
	u.text(x+4, y-14, "La policía llega en:", uiDim)
	if i := u.seg(x, y+2, lvRW-4, 28, []segItem{{"car", "Coche"}, {"star", "Helicóptero"}}, cur); i >= 0 {
		m.Police = []string{"car", "heli"}[i]
	}
}

func (l *LevEd) ajustesPanel(u *UI) {
	m := &l.la.Meta
	x := lvRX
	u.panel(x-4, edTop+4, lvRW+8, edH-edTop-8)
	y := edTop + 12
	u.heading(x+4, y, lvRW-8, "AJUSTES")
	y += 22
	u.text(x+4, y, "Título", uiDim)
	u.input("lvtitle", x, y+14, lvRW-4, &l.titleTxt, 28)
	y += 44
	u.text(x+4, y, "Frase de introducción", uiDim)
	u.input("lvintro", x, y+14, lvRW-4, &l.introTxt, 80)
	y += 48
	u.stepper(x+4, y, "Invitados", &m.Guests, 4, 80, 2)
	y += 32
	u.text(x+4, y+4, "Policía", uiDim)
	cur := 0
	if m.Police == "heli" {
		cur = 1
	}
	if i := u.seg(x+60, y, lvRW-64, 26, []segItem{{"", "Coche"}, {"", "Helicóptero"}}, cur); i >= 0 {
		m.Police = []string{"car", "heli"}[i]
	}
	y += 34
	tod := 0
	switch m.TimeOfDay {
	case "dusk":
		tod = 1
	case "day":
		tod = 2
	}
	u.text(x+4, y+4, "Hora", uiDim)
	if i := u.seg(x+60, y, lvRW-64, 26, []segItem{{"", "Noche"}, {"", "Tarde"}, {"", "Día"}}, tod); i >= 0 {
		m.TimeOfDay = []string{"", "dusk", "day"}[i]
	}
	y += 34
	t := [4]int{int(m.Tint[0]), int(m.Tint[1]), int(m.Tint[2]), int(m.Tint[3])}
	u.text(x+4, y+2, "Oscuridad", uiDim)
	u.slider("ta", x+80, y, lvRW-100, &t[3], 0, 200, "")
	m.Tint = [4]uint8{uint8(t[0]), uint8(t[1]), uint8(t[2]), uint8(t[3])}
	y += 30
	check := func(label string, v *bool, tip string) {
		icon := ""
		if *v {
			icon = "check"
		}
		if u.iconBtn(x, y, lvRW-4, 26, icon, label, *v, tip) {
			*v = !*v
		}
		y += 30
	}
	check("Se puede nadar en el agua", &m.Swim, "")
	check("Luces de discoteca", &m.Disco, "Luces de colores sobre la zona de baile")
	check("Pista de baile de neón", &m.DanceFloor, "Cuadros de colores animados")
	u.rect(x, y, lvRW-4, 1, uiLine)
	y += 10
	u.heading(x+4, y, lvRW-8, "CARRETERA")
	y += 22
	rh := m.RoadH
	if rh == 0 {
		rh = 44
	}
	if u.stepper(x+4, y, "Ancho", &rh, 16, 120, 4) {
		m.RoadH = rh
	}
	y += 32
	tr := 0
	switch m.Traffic {
	case "right":
		tr = 1
	case "left":
		tr = 2
	}
	if i := u.seg(x, y, lvRW-4, 26, []segItem{{"", "Ambos"}, {"right", "Derecha"}, {"left", "Izquierda"}}, tr); i >= 0 {
		m.Traffic = []string{"", "right", "left"}[i]
	}
}

// tplEntry is one game scenario that can be copied as a template. Building a scenario
// (generating it and checking that every point can be reached) takes a good fraction of a
// second, so only the title is known up front and the scenario is built when it is chosen.
type tplEntry struct {
	title string
	def   func() *LevelDef
}

var tplCache []tplEntry

func templateEntries() []tplEntry {
	if tplCache != nil {
		return tplCache
	}
	for _, e := range campaignLevels {
		e := e
		title := e.title
		if e.level != nil {
			title = e.level().Title
		}
		tplCache = append(tplCache, tplEntry{title, func() *LevelDef { return resolveCampaign(e) }})
	}
	for _, e := range extraLevels {
		e := e
		tplCache = append(tplCache, tplEntry{e.title, func() *LevelDef { return genNamed(e.kind, e.seed, e.title) }})
	}
	return tplCache
}
