package main

import (
	"fmt"
	"image/color"
	"math"
	"path/filepath"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type logicBrush struct {
	label string
	layer int // 0 solid, 1 zone
	ch    byte
	col   color.RGBA
}

var logicBrushes = []logicBrush{
	{"Muro (opaco)", 0, '#', color.RGBA{230, 60, 60, 150}},
	{"Valla baja", 0, '=', color.RGBA{235, 170, 50, 150}},
	{"Agua", 0, '~', color.RGBA{60, 140, 255, 140}},
	{"Borrar sólido", 0, '.', color.RGBA{}},
	{"Zona de paseo", 1, 'r', color.RGBA{70, 220, 100, 90}},
	{"Zona de baile", 1, 'd', color.RGBA{235, 80, 235, 120}},
	{"Borrar zona", 1, '.', color.RGBA{}},
}

var perfRoles = []string{"dj", "drum", "guitar", "sing"}
var perfLabels = map[string]string{"dj": "DJ", "drum": "Batería", "guitar": "Guitarra", "sing": "Cantante"}

type LevEd struct {
	ed                           *Editor
	la                           *LevelAsset
	list                         []string
	sel                          int
	scroll                       int
	name                         string
	mode                         int
	cv                           Canvas
	solid                        [][]byte
	zone                         [][]byte
	brush                        int
	bsize                        int
	rectMode                     bool
	rdrag                        bool
	rx0, ry0                     int
	undoL                        [][2][][]byte
	showLogic, showObjs, showPts bool
	kinds                        []string
	kindSel                      int
	kindScr                      int
	moveMode                     bool
	selInst                      int
	dragInst                     bool
	curVar                       int
	ptTool                       int
	routeIdx                     int
	role                         string
	lcol                         [3]int
	lalpha                       int
	lrad                         int
	lpulse                       bool
	tplIdx                       int
	lastMX                       int
	lastMY                       int
	titleTxt                     string
	animKind                     int
	animRad                      int
	introTxt                     string
}

func newLevEd(ed *Editor) *LevEd {
	l := &LevEd{ed: ed, sel: -1, showLogic: false, showObjs: true, showPts: true, bsize: 2, role: "dj", lcol: [3]int{255, 200, 120}, lalpha: 140, lrad: 70, selInst: -1, animRad: 50}
	l.cv = Canvas{x: 248, y: 64, vw: 720, vh: 648, zoom: 1}
	l.newBlank()
	return l
}

func (l *LevEd) reloadList() {
	l.list = listAssetNames(backgroundsDir(l.ed.root))
	l.refreshKinds()
}

func (l *LevEd) refreshKinds() {
	l.kinds = l.kinds[:0]
	for k := range propKinds {
		l.kinds = append(l.kinds, k)
	}
	sort.Strings(l.kinds)
}

func (l *LevEd) newBlank() {
	l.setLevel(newLevelAsset("nuevo_escenario"))
	l.name = "nuevo_escenario"
}

func (l *LevEd) setLevel(la *LevelAsset) {
	l.la = la
	l.solid, l.zone = toGrid(la.Meta.Solid), toGrid(la.Meta.Zone)
	l.titleTxt, l.introTxt = la.Meta.Title, la.Meta.Intro
	l.selInst, l.routeIdx = -1, 0
	l.undoL = nil
	l.ed.pt.clearHistory()
	l.cv.zoom = 1
	l.cv.px, l.cv.py = (l.cv.vw-screenW)/2, (l.cv.vh-screenH)/2
	l.cv.setPix(la.BG)
}

func toGrid(rows []string) [][]byte {
	g := make([][]byte, len(rows))
	for i, r := range rows {
		g[i] = []byte(r)
	}
	return g
}

func fromGrid(g [][]byte) []string {
	out := make([]string, len(g))
	for i, r := range g {
		out[i] = string(r)
	}
	return out
}

func (l *LevEd) loadByName(name string) {
	la, err := loadLevelAsset(filepath.Join(backgroundsDir(l.ed.root), name))
	if err != nil {
		l.ed.say("No se pudo cargar: " + err.Error())
		return
	}
	l.setLevel(la)
	l.name = name
}

func (l *LevEd) syncMeta() {
	l.la.Meta.Solid = fromGrid(l.solid)
	l.la.Meta.Zone = fromGrid(l.zone)
	l.la.Meta.Title = l.titleTxt
	l.la.Meta.Intro = l.introTxt
}

func (l *LevEd) save() {
	name := safeName(l.name)
	if name == "" {
		l.ed.say("Pon un nombre al escenario")
		return
	}
	l.syncMeta()
	l.la.Meta.Name = name
	if l.la.Meta.Title == "" {
		l.la.Meta.Title = name
	}
	if err := saveLevelAsset(l.ed.root, l.la); err != nil {
		l.ed.say("Error al guardar: " + err.Error())
		return
	}
	l.name = name
	l.reloadList()
	l.ed.say("Guardado en " + l.la.Dir)
}

func (l *LevEd) pushLogicUndo() {
	cp := func(g [][]byte) [][]byte {
		o := make([][]byte, len(g))
		for i := range g {
			o[i] = append([]byte(nil), g[i]...)
		}
		return o
	}
	l.undoL = append(l.undoL, [2][][]byte{cp(l.solid), cp(l.zone)})
	if len(l.undoL) > 40 {
		l.undoL = l.undoL[1:]
	}
}

func (l *LevEd) paintCells(x, y int, erase bool) {
	b := logicBrushes[l.brush]
	cx, cy := x/logicCell, y/logicCell
	for dy := 0; dy < l.bsize; dy++ {
		for dx := 0; dx < l.bsize; dx++ {
			xx, yy := cx+dx, cy+dy
			if yy < 0 || yy >= len(l.solid) || xx < 0 || xx >= len(l.solid[0]) {
				continue
			}
			ch := b.ch
			if erase {
				ch = '.'
			}
			if b.layer == 0 {
				l.solid[yy][xx] = ch
			} else {
				l.zone[yy][xx] = ch
			}
		}
	}
}

func (l *LevEd) fillCells(x0, y0, x1, y1 int, erase bool) {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	b := logicBrushes[l.brush]
	for cy := y0 / logicCell; cy <= y1/logicCell; cy++ {
		for cx := x0 / logicCell; cx <= x1/logicCell; cx++ {
			if cy < 0 || cy >= len(l.solid) || cx < 0 || cx >= len(l.solid[0]) {
				continue
			}
			ch := b.ch
			if erase {
				ch = '.'
			}
			if b.layer == 0 {
				l.solid[cy][cx] = ch
			} else {
				l.zone[cy][cx] = ch
			}
		}
	}
}

func nearestIdx(n int, dist func(i int) float64) (int, float64) {
	best, bd := -1, 1e9
	for i := 0; i < n; i++ {
		if d := dist(i); d < bd {
			best, bd = i, d
		}
	}
	return best, bd
}

func (l *LevEd) update(u *UI) {
	m := &l.la.Meta
	pt := l.ed.pt
	pt.usePart = false
	// ---- left panel
	u.panel(4, 32, 240, 684)
	y := 40
	u.text(12, y, "Escenarios guardados", uiDim)
	if i := u.list(12, y+14, 224, 9, l.list, l.sel, &l.scroll); i >= 0 {
		l.sel = i
		l.loadByName(l.list[i])
	}
	y += 14 + 9*18 + 12
	u.text(12, y, "Nombre (carpeta)", uiDim)
	u.input("lvname", 12, y+14, 224, &l.name, 24)
	y += 42
	if u.button(12, y, 110, 22, "Guardar", false) {
		l.save()
	}
	if u.button(126, y, 110, 22, "Nuevo en blanco", false) {
		l.newBlank()
		l.sel = -1
	}
	y += 28
	if u.button(12, y, 224, 22, "Plantilla: nivel por defecto ▶", false) {
		defs := builtinLevels()
		def := defs[l.tplIdx%len(defs)]
		la, err := exportLevelDef(l.ed.g, def, "copia_"+safeName(def.Title))
		if err == nil {
			l.setLevel(la)
			l.name = la.Meta.Name
			l.sel = -1
			l.ed.say("Cargada la plantilla: " + def.Title)
		}
		l.tplIdx++
	}
	y += 28
	if u.button(12, y, 224, 24, "▶ Probar este escenario", false) {
		l.save()
		l.ed.g.testLevel(safeName(l.name))
		return
	}
	y += 34
	u.text(12, y, "CAPAS VISIBLES", uiAccent)
	y += 18
	u.toggle(12, y, 224, "Lógica (paredes, agua, zonas)", &l.showLogic)
	u.toggle(12, y+24, 224, "Objetos", &l.showObjs)
	u.toggle(12, y+48, 224, "Puntos (inicio, rutas, luces)", &l.showPts)
	y += 80
	u.text(12, y, "Rueda del ratón: zoom", uiDim)
	u.text(12, y+16, "Botón central / espacio+arrastrar: mover", uiDim)
	u.text(12, y+32, "Ctrl+S guardar · Ctrl+Z deshacer", uiDim)
	// ---- mode buttons
	for i, n := range []string{"1 Fondo", "2 Lógica", "3 Objetos", "4 Puntos", "5 Propiedades"} {
		if u.button(250+i*118, 36, 114, 22, n, l.mode == i) {
			l.mode = i
		}
	}
	// ---- canvas
	cv := &l.cv
	if cv.img == nil {
		cv.setPix(l.la.BG)
	}
	panning := u.mdown || (ebiten.IsKeyPressed(ebiten.KeySpace) && u.down)
	if cv.inView(u.mx, u.my) {
		if u.wheel > 0 {
			cv.zoomAt(u.mx, u.my, cv.zoom+1)
		} else if u.wheel < 0 {
			cv.zoomAt(u.mx, u.my, cv.zoom-1)
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
			if l.mode == 0 {
				pt.doUndo(l.la.BG)
				cv.dirty = true
			} else if len(l.undoL) > 0 {
				s := l.undoL[len(l.undoL)-1]
				l.undoL = l.undoL[:len(l.undoL)-1]
				l.solid, l.zone = s[0], s[1]
			}
		}
		if ctrl && inpututil.IsKeyJustPressed(ebiten.KeyY) && l.mode == 0 {
			pt.doRedo(l.la.BG)
			cv.dirty = true
		}
		for i := 0; i < 5; i++ {
			if inpututil.IsKeyJustPressed(ebiten.Key1 + ebiten.Key(i)) {
				l.mode = i
			}
		}
		if (inpututil.IsKeyJustPressed(ebiten.KeyDelete) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace)) && l.mode == 2 && l.selInst >= 0 && l.selInst < len(m.Props) {
			m.Props = append(m.Props[:l.selInst], m.Props[l.selInst+1:]...)
			l.selInst = -1
		}
	}
	switch l.mode {
	case 0:
		u.text(250, 40+26, "Pinta el fondo del escenario píxel a píxel (640x360). Los objetos se colocan aparte.", uiDim)
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
	if cv.dirty {
		cv.refresh(l.la.BG)
	}
	// ---- canvas drawing
	view := *cv
	logic := l.showLogic || l.mode == 1
	solid, zone := l.solid, l.zone
	showObjs, showPts := l.showObjs, l.showPts || l.mode == 3
	props := append([]LevelProp(nil), m.Props...)
	meta := *m
	selInst := l.selInst
	mode := l.mode
	rd, rx0, ry0 := l.rdrag, l.rx0, l.ry0
	g := l.ed.g
	u.add(func(d *ebiten.Image) {
		view.draw(d, screenW, screenH, false)
		vp := d.SubImage(imageRect(view.x, view.y, view.vw, view.vh)).(*ebiten.Image)
		sx := func(px int) float64 { return float64(view.x + view.px + px*view.zoom) }
		sy := func(py int) float64 { return float64(view.y + view.py + py*view.zoom) }
		z := float64(view.zoom)
		if showObjs {
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
		}
		if logic {
			for cy, row := range solid {
				for cx, ch := range row {
					for _, b := range logicBrushes {
						if b.layer == 0 && b.ch == ch && ch != '.' {
							fillRect(vp, sx(cx*logicCell), sy(cy*logicCell), z*logicCell, z*logicCell, b.col)
						}
					}
				}
			}
			for cy, row := range zone {
				for cx, ch := range row {
					for _, b := range logicBrushes {
						if b.layer == 1 && b.ch == ch && ch != '.' {
							fillRect(vp, sx(cx*logicCell), sy(cy*logicCell), z*logicCell, z*logicCell, b.col)
						}
					}
				}
			}
		}
		if showPts {
			drawLevelPoints(vp, &meta, sx, sy, z, mode == 3)
		}
		view.dirty = false
		if mode == 0 {
			pt.previewShape(func(px, py int) {
				fillRect(vp, sx(px), sy(py), z, z, color.RGBA{pt.col.R, pt.col.G, pt.col.B, 160})
			})
		}
		if rd && mode == 1 {
			x0, y0, x1, y1 := minI(rx0, 0), 0, 0, 0
			_ = x0
			_ = y0
			_ = x1
			_ = y1
			fillRect(vp, sx(rx0), sy(ry0), 2, 2, uiAccent)
		}
		// 4x4 hover cell in logic mode
		if mode == 1 {
			mx, my := view.at(ebiten.CursorPosition())
			fillRect(vp, sx(mx/logicCell*logicCell), sy(my/logicCell*logicCell), z*logicCell, 1, uiAccent)
			fillRect(vp, sx(mx/logicCell*logicCell), sy(my/logicCell*logicCell)+z*logicCell, z*logicCell, 1, uiAccent)
		}
	})
	u.frame(cv.x-1, cv.y-1, cv.vw+2, cv.vh+2, uiLine)
	if inCanvas && cv.inView(u.mx, u.my) {
		u.text(cv.x+4, cv.y+cv.vh-16, fmt.Sprintf("x:%d y:%d", x, yy), uiText)
	}
	// ---- right panel
	u.panel(976, 32, 300, 684)
	rx := 984
	switch l.mode {
	case 0:
		u.text(rx, 38, "PINTAR FONDO", uiAccent)
		u.paletteUI(pt, rx, 56, false)
	case 1:
		l.logicPanel(u, rx)
	case 2:
		l.objectsPanel(u, rx)
	case 3:
		l.pointsPanel(u, rx)
	case 4:
		l.propsPanel(u, rx)
	}
}

func drawLevelPoints(vp *ebiten.Image, m *LevelMeta, sx, sy func(int) float64, z float64, active bool) {
	dot := func(x, y int, r float64, c color.RGBA) {
		fillRect(vp, sx(x)-r, sy(y)-r, 2*r, 2*r, c)
	}
	line := func(x0, y0, x1, y1 int, c color.RGBA) {
		bresenham(x0, y0, x1, y1, func(px, py int) { fillRect(vp, sx(px), sy(py), math.Max(1, z*0.6), math.Max(1, z*0.6), c) })
	}
	// start
	dot(m.Start.X, m.Start.Y, 5, color.RGBA{60, 230, 255, 255})
	fillRect(vp, sx(m.Start.X)-1, sy(m.Start.Y)-16, 2, 16, color.RGBA{60, 230, 255, 255})
	fillRect(vp, sx(m.Start.X)+1, sy(m.Start.Y)-16, 8, 5, color.RGBA{60, 230, 255, 255})
	if m.Police == "car" {
		line(m.CarFrom.X, m.CarFrom.Y, m.CarTo.X, m.CarTo.Y, color.RGBA{90, 120, 255, 255})
		dot(m.CarFrom.X, m.CarFrom.Y, 4, color.RGBA{90, 120, 255, 255})
		dot(m.CarTo.X, m.CarTo.Y, 5, color.RGBA{255, 255, 255, 255})
		fillRect(vp, sx(m.CarTo.X)-14*z, sy(m.CarTo.Y)-8*z, 28*z, 8*z, color.RGBA{90, 120, 255, 90})
	}
	for _, p := range m.Pads {
		dot(p.X, p.Y, 6, color.RGBA{255, 255, 255, 255})
		dot(p.X, p.Y, 4, color.RGBA{60, 60, 100, 255})
	}
	routeCols := []color.RGBA{{255, 120, 60, 255}, {255, 230, 60, 255}, {120, 255, 120, 255}, {255, 120, 255, 255}, {120, 200, 255, 255}}
	for ri, r := range m.Guards {
		c := routeCols[ri%len(routeCols)]
		for i, p := range r {
			dot(p.X, p.Y, 3, c)
			if i > 0 {
				line(r[i-1].X, r[i-1].Y, p.X, p.Y, color.RGBA{c.R, c.G, c.B, 160})
			}
		}
		if len(r) > 1 {
			line(r[len(r)-1].X, r[len(r)-1].Y, r[0].X, r[0].Y, color.RGBA{c.R, c.G, c.B, 70})
		}
	}
	for _, p := range m.Performers {
		dot(p.X, p.Y, 5, color.RGBA{190, 90, 255, 255})
	}
	for _, an := range m.Animals {
		c := color.RGBA{255, 180, 90, 255}
		dot(an.X, an.Y, 4, c)
		r := float64(an.Radius)
		for a := 0.0; a < 2*math.Pi; a += 0.12 {
			fillRect(vp, sx(an.X)+math.Cos(a)*r*z, sy(an.Y)+math.Sin(a)*r*z*0.7, 1, 1, color.RGBA{255, 180, 90, 130})
		}
	}
	for _, lt := range m.Lights {
		c := color.RGBA{lt.Color[0], lt.Color[1], lt.Color[2], 255}
		dot(lt.X, lt.Y, 3, c)
		r := float64(lt.R)
		for a := 0.0; a < 2*math.Pi; a += 0.08 {
			fillRect(vp, sx(lt.X)+math.Cos(a)*r*z, sy(lt.Y)+math.Sin(a)*r*z, 1, 1, color.RGBA{c.R, c.G, c.B, 160})
		}
	}
}

func (l *LevEd) logicPanel(u *UI, rx int) {
	u.text(rx, 38, "LÓGICA DEL ESCENARIO", uiAccent)
	u.text(rx, 56, "Clic izq.: pintar · Clic der.: borrar", uiDim)
	y := 78
	for i, b := range logicBrushes {
		col := b.col
		if u.button(rx, y+i*26, 220, 22, b.label, l.brush == i) {
			l.brush = i
		}
		if b.ch != '.' {
			u.rect(rx+226, y+i*26+2, 16, 16, color.RGBA{col.R, col.G, col.B, 255})
		}
	}
	y += len(logicBrushes)*26 + 8
	u.text(rx, y, fmt.Sprintf("Pincel: %dx%d celdas", l.bsize, l.bsize), uiText)
	if u.button(rx+170, y-4, 28, 20, "-", false) && l.bsize > 1 {
		l.bsize--
	}
	if u.button(rx+202, y-4, 28, 20, "+", false) && l.bsize < 10 {
		l.bsize++
	}
	y += 26
	u.toggle(rx, y, 220, "Modo rectángulo (arrastrar)", &l.rectMode)
	y += 30
	lines := []string{
		"Muro opaco: bloquea paso y visión.",
		"Valla baja: bloquea el paso, se ve a través.",
		"Agua: el jugador no entra; los invitados nadan",
		"  si el escenario permite nadar.",
		"Zona de paseo: donde se mueven los invitados.",
		"Zona de baile: donde bailan (y se puede animar",
		"  con cuadros de neón en Propiedades).",
		"Cada celda son 4x4 píxeles.",
	}
	for i, s := range lines {
		u.text(rx, y+i*15, s, uiDim)
	}
	y += len(lines)*15 + 14
	if u.button(rx, y, 220, 22, "Borrar toda la lógica", false) {
		l.pushLogicUndo()
		l.solid, l.zone = toGrid(blankRows()), toGrid(blankRows())
	}
}

func blankRows() []string {
	s, _ := blankLogic()
	return s
}

func (l *LevEd) updateObjects(u *UI, x, y int, inside bool) {
	m := &l.la.Meta
	if inside {
		if l.moveMode {
			if u.pressed {
				l.selInst, _ = nearestIdx(len(m.Props), func(i int) float64 { return math.Hypot(float64(m.Props[i].X-x), float64(m.Props[i].Y-y)) })
				if l.selInst >= 0 {
					if d := math.Hypot(float64(m.Props[l.selInst].X-x), float64(m.Props[l.selInst].Y-y)); d > 20 {
						l.selInst = -1
					} else {
						l.dragInst = true
					}
				}
			}
			if l.dragInst && u.down && l.selInst >= 0 && l.selInst < len(m.Props) {
				m.Props[l.selInst].X, m.Props[l.selInst].Y = x, y
			}
		} else if u.pressed && l.kindSel >= 0 && l.kindSel < len(l.kinds) {
			m.Props = append(m.Props, LevelProp{Kind: l.kinds[l.kindSel], X: x, Y: y, Var: l.curVar})
			l.selInst = len(m.Props) - 1
		}
		if u.rpressed {
			if i, d := nearestIdx(len(m.Props), func(i int) float64 { return math.Hypot(float64(m.Props[i].X-x), float64(m.Props[i].Y-y)) }); i >= 0 && d < 24 {
				m.Props = append(m.Props[:i], m.Props[i+1:]...)
				l.selInst = -1
			}
		}
	}
	if u.released {
		l.dragInst = false
	}
}

func (l *LevEd) objectsPanel(u *UI, rx int) {
	m := &l.la.Meta
	u.text(rx, 38, "OBJETOS", uiAccent)
	if u.button(rx, 56, 144, 22, "Colocar", !l.moveMode) {
		l.moveMode = false
	}
	if u.button(rx+148, 56, 144, 22, "Mover", l.moveMode) {
		l.moveMode = true
	}
	if i := u.list(rx, 86, 292, 22, l.kinds, l.kindSel, &l.kindScr); i >= 0 {
		l.kindSel = i
		l.curVar = 0
	}
	y := 86 + 22*18 + 12
	u.text(rx, y, fmt.Sprintf("Variante: %d", l.curVar), uiText)
	if u.button(rx+110, y-4, 28, 20, "-", false) && l.curVar > 0 {
		l.curVar--
		if l.selInst >= 0 && l.selInst < len(m.Props) {
			m.Props[l.selInst].Var = l.curVar
		}
	}
	if u.button(rx+142, y-4, 28, 20, "+", false) && l.curVar < 9 {
		l.curVar++
		if l.selInst >= 0 && l.selInst < len(m.Props) {
			m.Props[l.selInst].Var = l.curVar
		}
	}
	y += 24
	u.text(rx, y, fmt.Sprintf("Objetos en el escenario: %d", len(m.Props)), uiDim)
	u.text(rx, y+18, "Clic derecho: borrar el más cercano", uiDim)
	u.text(rx, y+34, "Supr: borrar el seleccionado", uiDim)
}

func (l *LevEd) updatePoints(u *UI, x, y int, inside bool) {
	m := &l.la.Meta
	if !inside {
		return
	}
	nearest := func(n int, at func(i int) (int, int)) int {
		i, d := nearestIdx(n, func(i int) float64 { ax, ay := at(i); return math.Hypot(float64(ax-x), float64(ay-y)) })
		if d > 24 {
			return -1
		}
		return i
	}
	switch l.ptTool {
	case 0:
		if u.pressed {
			m.Start = LevelPoint{x, y}
		}
	case 1:
		if u.pressed {
			m.CarFrom = LevelPoint{x, y}
		}
	case 2:
		if u.pressed {
			m.CarTo = LevelPoint{x, y}
		}
	case 3:
		if u.pressed && len(m.Pads) < 3 {
			m.Pads = append(m.Pads, LevelPoint{x, y})
		}
		if u.rpressed {
			if i := nearest(len(m.Pads), func(i int) (int, int) { return m.Pads[i].X, m.Pads[i].Y }); i >= 0 {
				m.Pads = append(m.Pads[:i], m.Pads[i+1:]...)
			}
		}
	case 4:
		if u.pressed {
			if len(m.Guards) == 0 {
				m.Guards = append(m.Guards, nil)
				l.routeIdx = 0
			}
			l.routeIdx = clampI(l.routeIdx, 0, len(m.Guards)-1)
			m.Guards[l.routeIdx] = append(m.Guards[l.routeIdx], LevelPoint{x, y})
		}
		if u.rpressed && len(m.Guards) > 0 {
			r := m.Guards[l.routeIdx]
			if len(r) > 0 {
				m.Guards[l.routeIdx] = r[:len(r)-1]
			}
		}
	case 5:
		if u.pressed {
			m.Performers = append(m.Performers, LevelPerf{X: x, Y: y, Role: l.role})
		}
		if u.rpressed {
			if i := nearest(len(m.Performers), func(i int) (int, int) { return m.Performers[i].X, m.Performers[i].Y }); i >= 0 {
				m.Performers = append(m.Performers[:i], m.Performers[i+1:]...)
			}
		}
	case 7:
		if u.pressed {
			m.Animals = append(m.Animals, LevelAnimal{Kind: animalKinds[l.animKind], X: x, Y: y, Radius: l.animRad})
		}
		if u.rpressed {
			if i := nearest(len(m.Animals), func(i int) (int, int) { return m.Animals[i].X, m.Animals[i].Y }); i >= 0 {
				m.Animals = append(m.Animals[:i], m.Animals[i+1:]...)
			}
		}
	case 6:
		if u.pressed {
			m.Lights = append(m.Lights, LevelLight{X: x, Y: y, R: l.lrad, Color: [3]uint8{uint8(l.lcol[0]), uint8(l.lcol[1]), uint8(l.lcol[2])}, Alpha: uint8(l.lalpha), Pulse: l.lpulse})
		}
		if u.rpressed {
			if i := nearest(len(m.Lights), func(i int) (int, int) { return m.Lights[i].X, m.Lights[i].Y }); i >= 0 {
				m.Lights = append(m.Lights[:i], m.Lights[i+1:]...)
			}
		}
	}
}

func (l *LevEd) pointsPanel(u *UI, rx int) {
	m := &l.la.Meta
	u.text(rx, 38, "PUNTOS DEL ESCENARIO", uiAccent)
	tools := []string{"Inicio del jugador", "Policía: coche llega desde", "Policía: coche aparca en", "Helipuertos (hasta 3)", "Ruta de guardia", "DJ / músicos", "Luces", "Animales"}
	for i, t := range tools {
		if u.button(rx, 58+i*26, 292, 22, t, l.ptTool == i) {
			l.ptTool = i
		}
	}
	y := 58 + len(tools)*26 + 10
	switch l.ptTool {
	case 0:
		u.text(rx, y, "Clic: dónde aparece el jugador.", uiDim)
	case 1, 2:
		u.text(rx, y, "El coche entra por 'desde' y aparca en 'aparca en'.", uiDim)
		u.text(rx, y+16, "Pon la ruta sobre la carretera.", uiDim)
		if u.button(rx, y+40, 292, 22, "Policía por: "+m.Police, false) {
			m.Police = cycleStr(m.Police, []string{"car", "heli"})
		}
	case 3:
		u.text(rx, y, "Si la policía llega en helicóptero, aterriza aquí.", uiDim)
		u.text(rx, y+16, "El segundo helipuerto es para la ambulancia.", uiDim)
		if u.button(rx, y+40, 292, 22, "Policía por: "+m.Police, false) {
			m.Police = cycleStr(m.Police, []string{"car", "heli"})
		}
	case 4:
		u.text(rx, y, fmt.Sprintf("Rutas: %d   (seleccionada: %d)", len(m.Guards), l.routeIdx+1), uiText)
		if u.button(rx, y+20, 144, 22, "Nueva ruta", false) {
			m.Guards = append(m.Guards, nil)
			l.routeIdx = len(m.Guards) - 1
		}
		if u.button(rx+148, y+20, 144, 22, "Borrar ruta", false) && len(m.Guards) > 0 {
			m.Guards = append(m.Guards[:l.routeIdx], m.Guards[l.routeIdx+1:]...)
			l.routeIdx = clampI(l.routeIdx, 0, maxI(0, len(m.Guards)-1))
		}
		if u.button(rx, y+46, 144, 22, "◄ ruta anterior", false) && len(m.Guards) > 0 {
			l.routeIdx = (l.routeIdx + len(m.Guards) - 1) % len(m.Guards)
		}
		if u.button(rx+148, y+46, 144, 22, "ruta siguiente ►", false) && len(m.Guards) > 0 {
			l.routeIdx = (l.routeIdx + 1) % len(m.Guards)
		}
		u.text(rx, y+76, "Clic: añade punto a la ruta (un guardia por ruta).", uiDim)
		u.text(rx, y+92, "Clic derecho: quita el último punto.", uiDim)
	case 5:
		if u.button(rx, y, 292, 22, "Rol: "+perfLabels[l.role], false) {
			l.role = cycleStr(l.role, perfRoles)
		}
		u.text(rx, y+30, "Clic: coloca. Clic derecho: borra el más cercano.", uiDim)
		u.text(rx, y+46, "Se quedan quietos y se animan (disco, batería...).", uiDim)
	case 7:
		if u.button(rx, y, 292, 22, "Animal: "+animalKinds[l.animKind], false) {
			l.animKind = (l.animKind + 1) % len(animalKinds)
		}
		u.slider("arad", rx, y+28, 150, &l.animRad, 20, 150, "Zona")
		u.text(rx, y+54, "Clic: coloca. Clic derecho: borra el más cercano.", uiDim)
		u.text(rx, y+70, "El caballo da coces a quien se queda detrás.", uiDim)
		u.text(rx, y+86, "Las gallinas huyen; el perro merodea.", uiDim)
	case 6:
		u.slider("lr", rx, y, 150, &l.lcol[0], 0, 255, "R")
		u.slider("lg", rx, y+18, 150, &l.lcol[1], 0, 255, "G")
		u.slider("lb", rx, y+36, 150, &l.lcol[2], 0, 255, "B")
		u.slider("la", rx, y+54, 150, &l.lalpha, 20, 255, "Fuerza")
		u.slider("lrad", rx, y+72, 150, &l.lrad, 10, 200, "Radio")
		u.toggle(rx, y+94, 292, "Parpadea", &l.lpulse)
		u.rect(rx+230, y, 40, 40, color.RGBA{uint8(l.lcol[0]), uint8(l.lcol[1]), uint8(l.lcol[2]), 255})
		u.text(rx, y+122, "Clic: coloca. Clic derecho: borra la más cercana.", uiDim)
	}
}

func (l *LevEd) propsPanel(u *UI, rx int) {
	m := &l.la.Meta
	u.text(rx, 38, "PROPIEDADES", uiAccent)
	y := 58
	u.text(rx, y, "Título que se muestra", uiDim)
	u.input("lvtitle", rx, y+14, 292, &l.titleTxt, 28)
	y += 44
	u.text(rx, y, "Texto de introducción", uiDim)
	u.input("lvintro", rx, y+14, 292, &l.introTxt, 80)
	y += 48
	u.text(rx, y, fmt.Sprintf("Invitados: %d", m.Guests), uiText)
	if u.button(rx+150, y-4, 30, 20, "-", false) && m.Guests > 4 {
		m.Guests -= 2
	}
	if u.button(rx+184, y-4, 30, 20, "+", false) && m.Guests < 80 {
		m.Guests += 2
	}
	y += 28
	if u.button(rx, y, 292, 22, "La policía llega en: "+m.Police, false) {
		m.Police = cycleStr(m.Police, []string{"car", "heli"})
	}
	y += 28
	u.toggle(rx, y, 292, "Los invitados pueden nadar (agua)", &m.Swim)
	u.toggle(rx, y+24, 292, "Luces de discoteca sobre la zona de baile", &m.Disco)
	u.toggle(rx, y+48, 292, "Cuadros de neón animados en la pista", &m.DanceFloor)
	y += 76
	u.text(rx, y, fmt.Sprintf("Tamaño de los cuadros: %d px", m.DanceCell), uiDim)
	if u.button(rx+212, y-4, 30, 20, "-", false) && m.DanceCell > 4 {
		m.DanceCell--
	}
	if u.button(rx+246, y-4, 30, 20, "+", false) && m.DanceCell < 40 {
		m.DanceCell++
	}
	y += 26
	u.text(rx, y, "Oscuridad / color ambiente", uiDim)
	t := [4]int{int(m.Tint[0]), int(m.Tint[1]), int(m.Tint[2]), int(m.Tint[3])}
	u.slider("tr", rx, y+18, 150, &t[0], 0, 120, "R")
	u.slider("tg", rx, y+36, 150, &t[1], 0, 120, "G")
	u.slider("tb", rx, y+54, 150, &t[2], 0, 120, "B")
	u.slider("ta", rx, y+72, 150, &t[3], 0, 200, "Oscuridad")
	m.Tint = [4]uint8{uint8(t[0]), uint8(t[1]), uint8(t[2]), uint8(t[3])}
	u.rect(rx+230, y+18, 40, 40, color.RGBA{uint8(minI(255, t[0]*2)), uint8(minI(255, t[1]*2)), uint8(minI(255, t[2]*2)), 255})
	u.text(rx, y+104, "Consejo: pinta el fondo con colores claros;", uiDim)
	u.text(rx, y+120, "el juego lo oscurece según la oscuridad.", uiDim)
}

// importDrops: a dropped PNG becomes the background (resized to 640x360 if needed).
func (l *LevEd) importDrops(drops []droppedImage) {
	b := drops[0].b
	nb := newPix(screenW, screenH)
	for y := 0; y < screenH; y++ {
		for x := 0; x < screenW; x++ {
			sx, sy := x*b.w/screenW, y*b.h/screenH
			copy(nb.p[(y*screenW+x)*4:(y*screenW+x)*4+4], b.p[(sy*b.w+sx)*4:(sy*b.w+sx)*4+4])
		}
	}
	l.la.BG = nb
	l.ed.pt.clearHistory()
	l.cv.setPix(nb)
	l.cv.dirty = true
	l.ed.say("Fondo importado. Dibuja ahora la lógica (paredes, agua, zonas)")
}
