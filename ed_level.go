package main

import (
	"image/color"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
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
	{"Zona lenta (barro, gente)", 0, ',', color.RGBA{150, 110, 60, 140}},
	{"Ventana (se salta con E)", 0, 'w', color.RGBA{80, 230, 230, 160}},
	{"Borrar sólido", 0, '.', color.RGBA{}},
	{"Zona de paseo", 1, 'r', color.RGBA{70, 220, 100, 90}},
	{"Zona de baile", 1, 'd', color.RGBA{235, 80, 235, 120}},
	{"Borrar zona", 1, '.', color.RGBA{}},
	// roads (indices 9..12): one lane of traffic going right, left, up or down
	{"Carretera →", 0, '>', color.RGBA{110, 130, 190, 150}},
	{"Carretera ←", 0, '<', color.RGBA{110, 130, 190, 150}},
	{"Carretera ↑", 0, '^', color.RGBA{110, 130, 190, 150}},
	{"Carretera ↓", 0, 'v', color.RGBA{110, 130, 190, 150}},
}

var perfRoles = []string{"dj", "drum", "guitar", "sing"}
var perfLabels = map[string]string{"dj": "DJ", "drum": "Batería", "guitar": "Guitarra", "sing": "Cantante"}

type LevEd struct {
	ed                           *Editor
	la                           *LevelAsset
	list                         []string
	sel                          int
	scroll                       int
	find, kindFind               string
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
	objTool                      int // Objetos tab: 0 place, 1 move, 2 move a zone
	mv                           lvMove
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
	thumbs                       map[string]*ebiten.Image
	importCh                     chan *droppedImage
	overrideKey                  string // set while a correction of one of the game's scenarios is open (debug mode)
	imported                     bool   // came from a picture: saved in the imports folder
	propThumbs                   map[string]*ebiten.Image
	wasRoad                      bool
	prefZoom                     float64 // zoom the user chose, remembered between sessions
	needFit                      bool    // centre and fit the picture at the next frame
	started                      bool    // the first scenario of the session was generated
}

func newLevEd(ed *Editor) *LevEd {
	l := &LevEd{ed: ed, thumbs: map[string]*ebiten.Image{}, sel: -1, showLogic: false, showObjs: true, showPts: true, bsize: 2, role: "dj", lcol: [3]int{255, 200, 120}, lalpha: 140, lrad: 70, selInst: -1, animRad: 50}
	l.cv = Canvas{x: 248, y: 64, vw: 720, vh: 648, zoom: 1}
	l.newBlank()
	return l
}

func (l *LevEd) reloadList() {
	l.list = nil
	for _, n := range backgroundNames(l.ed.root) {
		if !isCampaignKey(n) { // the corrections of the originals have their own list (debug mode)
			l.list = append(l.list, n)
		}
	}
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
	l.imported = false
	l.overrideKey = la.Meta.Override
	l.la = la
	l.solid, l.zone = toGrid(la.Meta.Solid), toGrid(la.Meta.Zone)
	l.titleTxt, l.introTxt = la.Meta.Title, la.Meta.Intro
	l.selInst, l.routeIdx = -1, 0
	l.undoL = nil
	l.mv = lvMove{}
	l.ed.pt.clearHistory()
	l.needFit = true
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
	dir := backgroundDirOf(l.ed.root, name)
	la, err := loadLevelAsset(dir)
	if err != nil {
		l.ed.say("No se pudo cargar: " + err.Error())
		return
	}
	l.setLevel(la)
	l.imported = inImports(l.ed.root, dir)
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
	if l.overrideKey != "" { // a correction of an original: always saved under its own key
		l.la.Meta.Name, l.la.Meta.Override = l.overrideKey, l.overrideKey
		if err := saveLevelAsset(l.ed.root, l.la); err != nil {
			l.ed.say("Error al guardar: " + err.Error())
			return
		}
		l.name = l.overrideKey
		l.ed.say("Original corregido y guardado en " + l.la.Dir)
		return
	}
	l.la.Meta.Floors, l.la.Meta.Pools = nil, nil // only the corrections of originals use them
	l.la.Meta.Name = name
	if l.la.Meta.Title == "" {
		l.la.Meta.Title = name
	}
	saveRoot, nm := importSaveRoot(l.ed.root, l.imported, "", name)
	if saveRoot != l.ed.root && dirExists(backgroundDirOf(l.ed.root, nm)) && !inImports(l.ed.root, backgroundDirOf(l.ed.root, nm)) {
		nm += "_importado"
	}
	l.la.Meta.Name = nm
	if err := saveLevelAsset(saveRoot, l.la); err != nil {
		l.ed.say("Error al guardar: " + err.Error())
		return
	}
	l.name = nm
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
		rh := m.RoadH
		if rh == 0 {
			rh = 44
		}
		fillRect(vp, sx(0), sy(m.CarFrom.Y-rh/2), 960*z, float64(rh)*z, rgba(0x5a78ff, 40))
		fillRect(vp, sx(0), sy(m.CarFrom.Y), 960*z, 1, rgba(0xffe65a, 200))
		if m.ParkX1 > m.ParkX0 {
			fillRect(vp, sx(m.ParkX0), sy(m.CarFrom.Y-rh/2), float64(m.ParkX1-m.ParkX0)*z, float64(rh)*z, rgba(0x3cff96, 90))
		}
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

func blankRows() []string {
	s, _ := blankLogic()
	return s
}

func (l *LevEd) updateObjects(u *UI, x, y int, inside bool) {
	m := &l.la.Meta
	switch l.objTool {
	case 1:
		mask := maskProps
		if l.overrideKey != "" { // ground and water only exist in the game's own scenarios
			mask |= maskGround
		}
		l.updateMoveTool(u, x, y, inside, mask)
		return
	case 2:
		l.updateZone(u, x, y, inside)
		return
	}
	if inside {
		if u.pressed && l.kindSel >= 0 && l.kindSel < len(l.kinds) {
			before := l.takeSnap(false)
			m.Props = append(m.Props, LevelProp{Kind: l.kinds[l.kindSel], X: x, Y: y, Var: l.curVar})
			l.pushMoveUndo(before)
			l.selInst = len(m.Props) - 1
		}
		if u.rpressed {
			if i, d := nearestIdx(len(m.Props), func(i int) float64 { return math.Hypot(float64(m.Props[i].X-x), float64(m.Props[i].Y-y)) }); i >= 0 && d < 24 {
				before := l.takeSnap(false)
				m.Props = append(m.Props[:i], m.Props[i+1:]...)
				l.pushMoveUndo(before)
				l.selInst = -1
			}
		}
	}
}

func (l *LevEd) updatePoints(u *UI, x, y int, inside bool) {
	m := &l.la.Meta
	if l.ptTool == ptMoveTool {
		l.updateMoveTool(u, x, y, inside, maskPoints)
		return
	}
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
	case 8:
		if u.pressed {
			m.ParkX0 = x
			if m.ParkX1 <= m.ParkX0 {
				m.ParkX1 = x + 160
			}
		}
		if u.rpressed {
			m.ParkX1 = x
			if m.ParkX0 >= m.ParkX1 {
				m.ParkX0 = x - 160
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
	l.imported = true
	l.ed.pt.clearHistory()
	l.cv.setPix(nb)
	l.cv.dirty = true
	l.ed.say("Fondo importado. Dibuja ahora la lógica (paredes, agua, zonas)")
}
