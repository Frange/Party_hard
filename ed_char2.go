package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"math/rand"
	"os"
	"path/filepath"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ---------------------------------------------------------------------------
// The character editor: pick the kind of character first, then draw it
// (DIBUJO) and tell the game which pixel is which body part (PARTES).
// ---------------------------------------------------------------------------

type charSub struct {
	label, icon string
	folder      string // assets/sprites/<folder>
	role        string // what the game uses it as ("" = nothing special)
}

type charType struct {
	label, icon string
	subs        []charSub
}

var charTypes = []charType{
	{"Protagonista", "star", []charSub{{"Protagonista", "star", "main_character", ""}}},
	{"Persona", "person", []charSub{{"Persona", "person", "people", ""}}},
	{"Persona especial", "magic", []charSub{{"Persona especial", "magic", "special_people", ""}}},
	{"Asesino", "knife", []charSub{{"Asesino", "knife", "special_enemies", "killer"}}},
	{"Tipo especial", "wrench", []charSub{
		{"Guardia", "guard", "enemies", "guard"},
		{"Guardia duro", "guard", "special_enemies", "guard"},
		{"Médico", "cross", "police", "medic"},
		{"Bombero", "flame", "police", "firefighter"},
		{"Técnico", "wrench", "police", "technician"},
	}},
	{"Policía", "badge", []charSub{
		{"Policía", "badge", "police", "cop"},
		{"FBI", "eye", "police", "fbi"},
		{"Militar", "guard", "police", "military"},
		{"SWAT", "trap", "police", "swat"},
		{"Ranger", "star", "police", "ranger"},
	}},
}

// matches tells whether a stored character (with this role) belongs to the sub-type.
func (s charSub) matches(role string) bool {
	switch s.folder {
	case "special_enemies":
		if s.role == "killer" {
			return role == "killer"
		}
		return role != "killer"
	case "police":
		if s.role == "cop" {
			return role == "" || role == "cop"
		}
		return role == s.role
	case "enemies":
		return true
	}
	return true
}

// charRoleOf reads just the role out of a character's meta.json.
func charRoleOf(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "meta.json"))
	if err != nil {
		return ""
	}
	var m struct {
		Role string `json:"role"`
	}
	json.Unmarshal(data, &m)
	return m.Role
}

func (c *CharEd) sub() charSub {
	t := charTypes[clampI(c.typeIdx, 0, len(charTypes)-1)]
	return t.subs[clampI(c.subIdx, 0, len(t.subs)-1)]
}

func (c *CharEd) setType(ti, si int) {
	c.typeIdx, c.subIdx = ti, si
	c.reloadList()
	c.listScroll = 0
	if len(c.list) > 0 {
		c.listSel = 0
		c.load(c.list[0])
	} else {
		c.listSel = -1
		c.newBlank()
	}
}

var carryLabels = map[string]string{
	"": "Sin objeto", "knife": "Cuchillo", "cup": "Vaso", "mic": "Micrófono", "phone": "Móvil", "baguette": "Barra de pan", "tray": "Bandeja",
	"camera": "Cámara", "bigcup": "Vaso grande", "saber": "Sable", "beer": "Cerveza", "flag": "Bandera", "pizza": "Pizza", "sign": "Cartel",
	"machete": "Machete", "chainsaw": "Motosierra", "claw": "Garras", "broom": "Escoba", "vacuum": "Aspiradora", "selfie": "Palo selfie",
	"plant": "Planta", "sax": "Saxofón", "trophy": "Trofeo", "balloons": "Globos", "megaphone": "Megáfono", "briefcase": "Maletín", "torch": "Antorcha",
}

var carryOrder = []string{"", "beer", "cup", "bigcup", "pizza", "phone", "mic", "camera", "sign", "flag", "trophy", "balloons", "megaphone", "briefcase", "baguette", "tray", "plant", "knife", "machete", "saber", "torch"}

var animLabels = map[int]string{
	AnimIdle: "Quieto", AnimWalk: "Andar", AnimRun: "Correr", AnimDrink: "Beber", AnimTalk: "Hablar", AnimWave: "Saludar", AnimPunch: "Puñetazo",
	AnimStab: "Apuñalar", AnimFall: "Caer", AnimVomit: "Vomitar", AnimKiss: "Besar", AnimDJ: "DJ", AnimDrum: "Batería", AnimGuitar: "Guitarra",
	AnimSing: "Cantar", AnimCarry: "Cargar", AnimTired: "Cansado", AnimCall: "Llamar", AnimPanic: "Pánico", AnimCrouch: "Agacharse",
	AnimPoint: "Señalar", AnimLove: "Enamorado", AnimLaugh: "Reír", AnimPirouette: "Pirueta", AnimPee: "Pis", AnimZombie: "Zombi", AnimLean: "Inclinarse",
	AnimCheer: "Animar",
}

func animLabel(a int) string {
	if s, ok := animLabels[a]; ok {
		return s
	}
	if a >= AnimDance0 && a <= AnimDance11 {
		return fmt.Sprintf("Baile %d", a-AnimDance0+1)
	}
	return animNames[a]
}

// thumbFor loads (once) the front picture of a character for the list.
func (c *CharEd) thumbFor(folder, name string) *ebiten.Image {
	dir := spriteDirOf(c.ed.root, folder, name)
	if im, ok := c.thumbs[dir]; ok {
		return im
	}
	b, err := loadPNG(filepath.Join(dir, "front.png"))
	if err != nil {
		c.thumbs[dir] = nil
		return nil
	}
	im := ebiten.NewImage(b.w, b.h)
	im.WritePixels(b.p)
	c.thumbs[dir] = im
	return im
}

const (
	chColX   = 8
	chColW   = 268
	chRightX = 796
)

func (c *CharEd) update(u *UI) {
	pt := c.ed.pt
	// import dialog result
	if c.importCh != nil {
		select {
		case d, ok := <-c.importCh:
			if d != nil {
				c.ed.requestImport([]droppedImage{*d})
			}
			if !ok || d != nil {
				c.importCh = nil
			}
		default:
		}
	}
	pt.usePart = !c.ovMode
	pt.partOnly = c.mode == 1 && !c.ovMode
	pt.mirror = false
	pt.brush = 1
	c.leftColumn(u)
	c.canvasArea(u)
	c.rightColumn(u)
}

// ---- left: kind of character, list, save/new/import -------------------------

func (c *CharEd) leftColumn(u *UI) {
	x, w := chColX, chColW
	u.panel(x-4, edTop+4, w+8, edH-edTop-8)
	y := edTop + 12
	u.heading(x+4, y, w-8, "TIPO DE PERSONAJE")
	y += 20
	bw := (w - 6) / 2
	for i, t := range charTypes {
		bx, by := x+(i%2)*(bw+6), y+(i/2)*34
		if u.iconBtn(bx, by, bw, 30, t.icon, t.label, c.typeIdx == i, "") && c.typeIdx != i {
			c.setType(i, 0)
		}
	}
	y += 3*34 + 2
	if t := charTypes[c.typeIdx]; len(t.subs) > 1 {
		cols := 3
		bw3 := (w - 4*(cols-1)) / cols
		for j, sb := range t.subs {
			bx, by := x+(j%cols)*(bw3+4), y+(j/cols)*28
			if u.iconBtn(bx, by, bw3, 24, "", sb.label, c.subIdx == j, "") && c.subIdx != j {
				c.setType(c.typeIdx, j)
			}
		}
		y += ((len(t.subs)+cols-1)/cols)*28 + 4
	}
	u.heading(x+4, y, w-8, fmt.Sprintf("%s (%d)", c.sub().label, len(c.list)))
	y += 20
	gridH := edH - y - 150
	budget := 6
	folder := c.sub().folder
	mk := func(i int) thumb {
		n := c.list[i]
		t := thumb{label: n}
		dir := spriteDirOf(c.ed.root, folder, n)
		if _, ok := c.thumbs[dir]; ok || budget > 0 {
			if !ok {
				budget--
			}
			t.img = c.thumbFor(folder, n)
		}
		return t
	}
	if i := u.searchGrid("charfind", x, y, w, gridH, 4, &c.find, c.list, mk, c.listSel, &c.listScroll); i >= 0 {
		c.listSel = i
		c.load(c.list[i])
	}
	y += gridH + 8
	u.text(x+4, y+4, "Nombre", uiDim)
	u.input("charname", x+54, y, w-54, &c.name, 24)
	y += 28
	bw3 := (w - 8) / 3
	if u.iconBtn(x, y, bw3, 34, "save", "Guardar", false, "Guardar este personaje (Ctrl+S)") {
		c.save()
	}
	if i := u.menuBtn(x+bw3+4, y, bw3, 34, "new", "Nuevo", "newchar", []segItem{
		{"person", "Persona al azar"}, {"star", "Como el protagonista"}, {"magic", "Personaje famoso al azar"}, {"new", "En blanco"},
	}, "Empezar un personaje nuevo"); i >= 0 {
		switch i {
		case 0:
			c.fromSpec(genericSpec(rand.New(rand.NewSource(rand.Int63()))))
		case 1:
			c.fromSpec(playerSpec())
		case 2:
			c.fromSpec(famousPresets[c.tplIdx%len(famousPresets)](rand.New(rand.NewSource(int64(c.tplIdx) + 11))))
			c.tplIdx += 1 + rand.Intn(7)
		default:
			c.newBlank()
		}
		c.listSel = -1
	}
	if u.iconBtn(x+2*(bw3+4), y, bw3, 34, "import", "Importar", false, "Cargar una imagen PNG, JPG o PDF (o arrástrala a la ventana)") && c.importCh == nil {
		c.importCh = pickImageAsync()
	}
	y += 40
	if c.ca.Dir == "" {
		u.text(x+4, y, "Todavía sin guardar", uiDim)
	} else {
		u.text(x+4, y, "Guardado en la carpeta del juego", uiDim)
	}
}

// ---- centre: views, DIBUJO / PARTES, canvas and tools ------------------------------

func (c *CharEd) canvasArea(u *UI) {
	pt := c.ed.pt
	x0 := chColX + chColW + 12
	u.panel(x0-4, edTop+4, chRightX-x0-4, edH-edTop-8)
	if !c.ovMode {
		if i := u.seg(x0+4, edTop+10, 290, 28, []segItem{{"", "Frente"}, {"", "Perfil"}, {"", "Espalda"}}, c.view); i >= 0 {
			c.view = i
			c.cv.setPix(c.ca.Views[i])
		}
		if i := u.seg(x0+304, edTop+10, 180, 28, []segItem{{"", "DIBUJO"}, {"", "PARTES"}}, c.mode); i >= 0 && i != c.mode {
			c.mode = i
			pt.tool = ToolPencil
			if i == 1 {
				pt.part = PartHead
			}
		}
	} else {
		u.text(x0+8, edTop+14, fmt.Sprintf("Fotograma a mano: %s · %s · %d", animLabel(c.prevAnim), prevDirNames[c.prevDir], c.ovFrame+1), uiAccent)
		u.text(x0+8, edTop+30, "Pinta encima del fotograma generado. Al terminar, guárdalo.", uiDim)
	}
	buf := c.curBuf()
	if c.cv.img == nil || c.cv.img.Bounds().Dx() != buf.w || c.cv.img.Bounds().Dy() != buf.h {
		c.cv.fit(buf.w, buf.h)
		c.cv.setPix(buf)
	}
	c.cv.x, c.cv.y = x0+4, edTop+46
	c.cv.vw = 290
	// zoom, pan
	if c.cv.inView(u.mx, u.my) {
		if u.wheel > 0 {
			c.cv.zoomAt(u.mx, u.my, c.cv.zoom+1)
		} else if u.wheel < 0 {
			c.cv.zoomAt(u.mx, u.my, c.cv.zoom-1)
		}
	}
	panning := u.mdown || (ebiten.IsKeyPressed(ebiten.KeySpace) && u.down)
	if panning && c.cv.inView(u.mx, u.my) && (c.lastMX != 0 || c.lastMY != 0) {
		c.cv.px += u.mx - c.lastMX
		c.cv.py += u.my - c.lastMY
	}
	c.lastMX, c.lastMY = u.mx, u.my
	if u.focus == "" {
		ctrl := ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta)
		if ctrl && inpututil.IsKeyJustPressed(ebiten.KeyZ) {
			if ebiten.IsKeyPressed(ebiten.KeyShift) {
				pt.doRedo(buf)
			} else {
				pt.doUndo(buf)
			}
			c.cv.dirty, c.dirty = true, true
		}
		if ctrl && inpututil.IsKeyJustPressed(ebiten.KeyY) {
			pt.doRedo(buf)
			c.cv.dirty, c.dirty = true, true
		}
		dx, dy := 0, 0
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
			dx = -1
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
			dx = 1
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
			dy = -1
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
			dy = 1
		}
		if (dx != 0 || dy != 0) && c.mode == 0 {
			pt.pushUndo(buf)
			shiftView(buf, dx, dy)
			c.cv.dirty, c.dirty = true, true
		}
	}
	if !panning && pt.handle(u, &c.cv, buf, true) {
		c.cv.dirty, c.dirty = true, true
	}
	if pt.drawing || u.released {
		c.cv.dirty = true
	}
	if c.cv.dirty {
		c.cv.refresh(buf)
	}
	cv := c.cv
	showParts := c.mode == 1 && !c.ovMode
	u.add(func(d *ebiten.Image) {
		cv.draw(d, buf.w, buf.h, true)
		vp := d.SubImage(imageRect(cv.x, cv.y, cv.vw, cv.vh)).(*ebiten.Image)
		if showParts && buf.pid != nil {
			for yy := 0; yy < buf.h; yy++ {
				for xx := 0; xx < buf.w; xx++ {
					id := buf.pid[yy*buf.w+xx]
					if buf.p[(yy*buf.w+xx)*4+3] == 0 {
						continue
					}
					pc := partColors[id]
					if id == 0 {
						pc = color.RGBA{255, 0, 255, 255}
					}
					fillRect(vp, float64(cv.x+cv.px+xx*cv.zoom), float64(cv.y+cv.py+yy*cv.zoom), float64(cv.zoom), float64(cv.zoom), color.RGBA{pc.R / 2, pc.G / 2, pc.B / 2, 150})
				}
			}
		}
		if !showParts {
			pt.previewShape(func(x, y int) {
				fillRect(vp, float64(cv.x+cv.px+x*cv.zoom), float64(cv.y+cv.py+y*cv.zoom), float64(cv.zoom), float64(cv.zoom), color.RGBA{pt.col.R, pt.col.G, pt.col.B, 160})
			})
		}
		hx, hy := cv.at(u.mx, u.my)
		if cv.inView(u.mx, u.my) && hx >= 0 && hy >= 0 && hx < buf.w && hy < buf.h {
			fillRect(vp, float64(cv.x+cv.px+hx*cv.zoom), float64(cv.y+cv.py+hy*cv.zoom), float64(cv.zoom), 1, uiAccent)
			fillRect(vp, float64(cv.x+cv.px+hx*cv.zoom), float64(cv.y+cv.py+(hy+1)*cv.zoom-1), float64(cv.zoom), 1, uiAccent)
		}
	})
	u.roundFrame(cv.x-1, cv.y-1, cv.vw+2, cv.vh+2, uiLine)
	u.text(cv.x, cv.y+cv.vh+6, "Rueda: zoom · Espacio+arrastrar: mover · Ctrl+Z: deshacer", uiDim)
	// tools to the right of the canvas
	tx := cv.x + cv.vw + 10
	if c.mode == 0 || c.ovMode {
		c.drawTools(u, tx, edTop+46)
	} else {
		c.partTools(u, tx, edTop+46, buf)
	}
}

func (c *CharEd) drawTools(u *UI, x, y int) {
	pt := c.ed.pt
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
	buf := c.curBuf()
	if u.iconBtn(x, y, 88, 26, "undo", "Deshacer", false, "Ctrl+Z") {
		pt.doUndo(buf)
		c.cv.dirty, c.dirty = true, true
	}
	if u.iconBtn(x+92, y, 88, 26, "redo", "Rehacer", false, "Ctrl+Y") {
		pt.doRedo(buf)
		c.cv.dirty, c.dirty = true, true
	}
	y += 36
	u.heading(x, y, w, "COLOR")
	y += 20
	y = u.simplePalette(pt, x, y, w)
	u.canvasOptions(x, y+8, w)
}

// simplePalette: swatches, recent colours and a hex box. Returns the next free y.
func (u *UI) simplePalette(pt *Painter, x, y, w int) int {
	cols := w / 17
	for i, c := range defaultPalette {
		cx, cy := x+(i%cols)*17, y+(i/cols)*17
		col := rgb(c)
		u.rect(cx, cy, 15, 15, col)
		if pt.col == col {
			u.frame(cx-1, cy-1, 17, 17, uiAccent)
		}
		if u.in(cx, cy, 15, 15) && u.pressed {
			pt.setColor(col)
		}
	}
	y += ((len(defaultPalette)+cols-1)/cols)*17 + 6
	if len(pt.recent) > 0 {
		u.text(x, y, "Recientes", uiDim)
		y += 14
		for i, c := range pt.recent {
			if i >= cols*2 {
				break
			}
			cx, cy := x+(i%cols)*17, y+(i/cols)*17
			u.rect(cx, cy, 15, 15, c)
			if u.in(cx, cy, 15, 15) && u.pressed {
				pt.setColor(c)
			}
		}
		y += 17*minI(2, (len(pt.recent)+cols-1)/cols) + 6
	}
	u.swatch(x, y, 30, 26, pt.col)
	if u.input("hex", x+38, y+3, 74, &pt.hexText, 7) {
		if c, ok := parseHex(pt.hexText); ok {
			pt.col = c
		}
	}
	return y + 34
}

func (c *CharEd) partTools(u *UI, x, y int, buf *pixBuf) {
	pt := c.ed.pt
	w := 180
	u.heading(x, y, w, "PARTES DEL CUERPO")
	y += 20
	u.text(x, y, "Elige una parte y pinta", uiDim)
	u.text(x, y+14, "encima del dibujo.", uiDim)
	y += 36
	for i := 1; i <= 6; i++ {
		on := pt.part == uint8(i)
		if u.iconBtn(x, y, w, 30, "", partNames[i], on, "") {
			pt.part = uint8(i)
		}
		u.round(x+8, y+9, 12, 12, partColors[i])
		y += 34
	}
	if u.iconBtn(x, y, w, 30, "eraser", "Quitar parte", pt.part == 0, "Deja los píxeles sin parte") {
		pt.part = 0
	}
	y += 38
	for i, t := range []struct {
		tool       int
		icon, name string
	}{{ToolPencil, "pencil", "Pincel"}, {ToolFill, "target", "Rellenar toda la zona"}} {
		if u.iconBtn(x+i*94, y, 90, 30, t.icon, t.name[:minI(len(t.name), 7)], pt.tool == t.tool, t.name) {
			pt.tool = t.tool
		}
	}
	y += 40
	if u.iconBtn(x, y, w, 36, "magic", "Auto-asignar", false, "Asigna las partes solo, según la silueta") {
		pt.pushUndo(buf)
		autoParts(buf, c.view == 1)
		c.cv.dirty, c.dirty = true, true
		c.ed.say("Partes asignadas por la silueta")
	}
	y += 44
	u.text(x, y, "Los píxeles sin parte se", uiDim)
	u.text(x, y+14, "asignan solos al guardar.", uiDim)
}

// ---- right: properties and animated preview ----------------------------------------

func (c *CharEd) rightColumn(u *UI) {
	x := chRightX
	w := edW - x - 8
	// properties
	u.panel(x-4, edTop+4, w+8, 118)
	u.heading(x+4, edTop+12, w-8, "PROPIEDADES")
	y := edTop + 34
	u.text(x+6, y+8, "Sexo", uiDim)
	g := 0
	if c.ca.Meta.Gender == "f" {
		g = 1
	}
	if i := u.seg(x+52, y, 170, 28, []segItem{{"male", "Hombre"}, {"female", "Mujer"}}, g); i >= 0 {
		c.ca.Meta.Gender = []string{"m", "f"}[i]
	}
	u.text(x+240, y+8, "Objeto", uiDim)
	cur := c.ca.Meta.Carry
	lab := carryLabels[cur]
	if lab == "" {
		lab = cur
	}
	var items []segItem
	for _, k := range carryOrder {
		items = append(items, segItem{"hand", carryLabels[k]})
	}
	if i := u.menuBtn(x+292, y, w-292, 28, "hand", lab, "carry", items, "Lo que lleva en la mano"); i >= 0 {
		c.ca.Meta.Carry = carryOrder[i]
		c.dirty = true
	}
	y += 36
	cw, ch := c.ca.Views[0].w, c.ca.Views[0].h
	u.text(x+6, y+4, "Tamaño", uiDim)
	if u.iconBtn(x+52, y, 22, 20, "minus", "", false, "Más estrecho") && cw > 12 {
		c.resize(cw-1, ch)
	}
	u.textC(x+92, y+4, fmt.Sprintf("%d", cw), uiText)
	if u.iconBtn(x+110, y, 22, 20, "plus", "", false, "Más ancho") && cw < 40 {
		c.resize(cw+1, ch)
	}
	u.text(x+142, y+4, "x", uiDim)
	if u.iconBtn(x+156, y, 22, 20, "minus", "", false, "Más bajo") && ch > 20 {
		c.resize(cw, ch-1)
	}
	u.textC(x+196, y+4, fmt.Sprintf("%d", ch), uiText)
	if u.iconBtn(x+214, y, 22, 20, "plus", "", false, "Más alto") && ch < 60 {
		c.resize(cw, ch+1)
	}
	if u.iconBtn(x+250, y-2, w-250, 26, "export", "Exportar PNG", false, "Guarda una hoja con todas las animaciones") {
		c.exportSheet()
	}
	// preview
	py := edTop + 130
	u.panel(x-4, py, w+8, edH-py-4)
	u.heading(x+4, py+8, w-8, "VISTA PREVIA")
	c.previewPanel(u, x, py+28, w)
}

func (c *CharEd) previewPanel(u *UI, x, y, w int) {
	if c.dirty || c.prevCS == nil {
		c.rebuildPreview()
	}
	// animation chooser: arrows + name + the common ones
	idx := 0
	for i, a := range c.animList {
		if a == c.prevAnim {
			idx = i
		}
	}
	setAnim := func(i int) {
		c.prevAnim = c.animList[(i+len(c.animList))%len(c.animList)]
		c.prevT = 0
		if c.ovMode {
			c.loadOverrideFrame()
		}
	}
	if u.iconBtn(x+4, y, 28, 26, "left", "", false, "Animación anterior") {
		setAnim(idx - 1)
	}
	u.round(x+36, y, 150, 26, uiCard)
	u.textC(x+36+75, y+7, animLabel(c.prevAnim), uiText)
	if u.iconBtn(x+190, y, 28, 26, "right", "", false, "Animación siguiente") {
		setAnim(idx + 1)
	}
	for i, a := range []int{AnimIdle, AnimWalk, AnimRun, AnimDance0} {
		if u.iconBtn(x+232+i*58, y, 56, 26, "", []string{"Quieto", "Andar", "Correr", "Bailar"}[i], c.prevAnim == a, "") {
			c.prevAnim, c.prevT = a, 0
			if c.ovMode {
				c.loadOverrideFrame()
			}
		}
	}
	y += 34
	if i := u.seg(x+4, y, 300, 26, []segItem{{"", "Frente"}, {"", "Perfil"}, {"", "Espalda"}}, minI(c.prevDir, 2)); i >= 0 {
		c.prevDir = i
		if c.ovMode {
			c.loadOverrideFrame()
		}
	}
	if u.iconBtn(x+308, y, 110, 26, "", "Perfil ◄", c.prevDir == 3, "") {
		c.prevDir = 3
		if c.ovMode {
			c.loadOverrideFrame()
		}
	}
	y += 34
	if !c.ovMode {
		c.prevT += 1.0 / 60 * 8
	}
	n := animFrames[c.prevAnim]
	frame := int(c.prevT) % n
	if c.ovMode {
		frame = c.ovFrame
	}
	img, flip := c.prevCS.img(c.prevAnim, prevDirs[c.prevDir], float64(frame))
	if c.prevDir == 3 {
		flip = true
	} else if c.prevDir == 1 {
		flip = false
	}
	zoom := 7
	iw, ih := img.Bounds().Dx(), img.Bounds().Dy()
	for ih*zoom > 270 && zoom > 2 {
		zoom--
	}
	px0, py0 := x+14, y+4
	u.add(func(d *ebiten.Image) {
		fillRect(d, float64(px0-6), float64(py0-4), float64(iw*zoom+12), float64(ih*zoom+8), rgb(0x2a3340))
		fillRect(d, float64(px0-6), float64(py0+ih*zoom-14), float64(iw*zoom+12), 18, rgb(0x27382b))
		op := &ebiten.DrawImageOptions{}
		if flip {
			op.GeoM.Scale(-1, 1)
			op.GeoM.Translate(float64(iw), 0)
		}
		op.GeoM.Scale(float64(zoom), float64(zoom))
		op.GeoM.Translate(float64(px0), float64(py0))
		d.DrawImage(img, op)
		op2 := &ebiten.DrawImageOptions{}
		if flip {
			op2.GeoM.Scale(-1, 1)
			op2.GeoM.Translate(float64(iw), 0)
		}
		op2.GeoM.Scale(2, 2)
		op2.GeoM.Translate(float64(x+iw*zoom+60), float64(py0+30))
		d.DrawImage(img, op2)
	})
	u.text(x+iw*zoom+50, py0+8, fmt.Sprintf("Fotograma %d/%d", frame+1, n), uiDim)
	u.text(x+iw*zoom+50, py0+ih*0+120, "Tamaño real", uiDim)
	// edit-by-hand controls, below the big preview and the frame strip
	sy := y + ih*zoom + 22
	u.text(x+4, sy, "Todos los fotogramas", uiDim)
	sw := minI(n, 9)
	for f := 0; f < n; f++ {
		fi, fl := c.prevCS.img(c.prevAnim, prevDirs[c.prevDir], float64(f))
		if c.prevDir == 3 {
			fl = true
		} else if c.prevDir == 1 {
			fl = false
		}
		ox := x + 8 + (f%sw)*50
		oy := sy + 16 + (f/sw)*84
		fw, fh := fi.Bounds().Dx(), fi.Bounds().Dy()
		frameSel := c.ovMode && f == c.ovFrame
		u.add(func(d *ebiten.Image) {
			op := &ebiten.DrawImageOptions{}
			if fl {
				op.GeoM.Scale(-1, 1)
				op.GeoM.Translate(float64(fw), 0)
			}
			op.GeoM.Scale(1.6, 1.6)
			op.GeoM.Translate(float64(ox), float64(oy))
			d.DrawImage(fi, op)
			if frameSel {
				fillRect(d, float64(ox-2), float64(oy-2), float64(fw)*1.6+4, 1, uiAccent)
				fillRect(d, float64(ox-2), float64(oy)+float64(fh)*1.6+2, float64(fw)*1.6+4, 1, uiAccent)
			}
		})
		if c.ovMode && u.in(ox, oy, int(float64(fw)*1.6), int(float64(fh)*1.6)) && u.pressed {
			c.ovFrame = f
			c.loadOverrideFrame()
		}
	}
	by := edH - 78
	if !c.ovMode {
		if u.iconBtn(x+4, by, 300, 30, "pencil", "Retocar este fotograma a mano", false, "Los fotogramas pintados a mano sustituyen a los generados") {
			if c.ca.Dir == "" {
				c.ed.say("Guarda primero el personaje")
			} else {
				c.ovMode = true
				c.ovFrame = int(c.prevT) % n
				c.loadOverrideFrame()
			}
		}
	} else {
		if u.iconBtn(x+4, by, 56, 30, "left", "", false, "Fotograma anterior") {
			c.ovFrame = (c.ovFrame + n - 1) % n
			c.loadOverrideFrame()
		}
		if u.iconBtn(x+64, by, 56, 30, "right", "", false, "Fotograma siguiente") {
			c.ovFrame = (c.ovFrame + 1) % n
			c.loadOverrideFrame()
		}
		if u.iconBtn(x+126, by, 130, 30, "save", "Guardar", false, "Guarda este fotograma") {
			ensureDir(filepath.Dir(c.overridePath()))
			if err := savePNG(c.overridePath(), c.ovBuf); err != nil {
				c.ed.say("Error: " + err.Error())
			} else {
				c.ed.say("Fotograma guardado")
				c.ovExists = true
				c.dirty = true
			}
		}
		if u.iconBtn(x+260, by, 100, 30, "undo", "Quitar", false, "Vuelve al fotograma generado") {
			os.Remove(c.overridePath())
			c.ed.say("Fotograma restaurado al generado")
			c.dirty = true
			c.loadOverrideFrame()
		}
		if u.iconBtn(x+4, by+36, 200, 28, "check", "Terminar", false, "Vuelve al dibujo base") {
			c.ovMode = false
			c.ed.pt.clearHistory()
			c.cv.fit(c.ca.Views[c.view].w, c.ca.Views[c.view].h)
			c.cv.setPix(c.ca.Views[c.view])
		}
	}
}

var _ = sort.Strings
