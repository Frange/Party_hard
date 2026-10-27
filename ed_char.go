package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type CharEd struct {
	ed         *Editor
	ca         *CharAsset
	catIdx     int
	list       []string
	listSel    int
	listScroll int
	name       string
	view       int
	cv         Canvas
	showParts  bool
	tplIdx     int
	// preview
	prevAnim int
	prevDir  int
	prevT    float64
	prevCS   *CharSet
	dirty    bool
	animList []int
	animScr  int
	// hand drawn frame overrides
	ovMode   bool
	ovFrame  int
	ovBuf    *pixBuf
	ovExists bool
	zoomSet  bool
	lastMX   int
	lastMY   int
}

var prevDirs = []Dir{DirDown, DirRight, DirUp, DirLeft}
var prevDirNames = []string{"Frente", "Perfil", "Espalda", "Perfil ◄"}

func newCharEd(ed *Editor) *CharEd {
	c := &CharEd{ed: ed, name: "nuevo", prevAnim: AnimWalk}
	for a := 0; a < AnimCount; a++ {
		if a != AnimLying && a != AnimSleep {
			c.animList = append(c.animList, a)
		}
	}
	c.ca = newCharAsset("nuevo", charCategories[1], charW, charH)
	c.cv = Canvas{x: 244, y: 112, vw: 290, vh: 500, zoom: 10, grid: true}
	c.cv.fit(charW, charH)
	c.dirty = true
	return c
}

func (c *CharEd) category() string { return charCategories[c.catIdx] }

func (c *CharEd) reloadList() {
	c.list = listAssetNames(spritesDir(c.ed.root, c.category()))
}

func (c *CharEd) load(name string) {
	ca, err := loadCharAsset(filepath.Join(spritesDir(c.ed.root, c.category()), name))
	if err != nil {
		c.ed.say("No se pudo cargar: " + err.Error())
		return
	}
	for _, v := range ca.Views {
		if v.pid == nil {
			v.pid = make([]uint8, v.w*v.h)
		}
	}
	c.ca = ca
	c.name = name
	c.ca.Meta.Category = c.category()
	c.afterLoad()
}

func (c *CharEd) afterLoad() {
	c.ed.pt.clearHistory()
	c.ovMode = false
	c.cv.fit(c.ca.Views[0].w, c.ca.Views[0].h)
	c.cv.setPix(c.ca.Views[c.view])
	c.dirty = true
}

func (c *CharEd) newBlank() {
	c.ca = newCharAsset("nuevo", c.category(), charW, charH)
	c.name = "nuevo"
	c.afterLoad()
}

func (c *CharEd) fromSpec(spec CharSpec) {
	role := ""
	switch c.category() {
	case "police":
		role = "cop"
	case "enemies", "special_enemies":
		role = "guard"
	}
	c.ca = charAssetFromSpec(spec, safeName(spec.Name+"_copia"), c.category(), role)
	if spec.Name == "" {
		c.ca.Meta.Name = "plantilla"
	}
	c.name = c.ca.Meta.Name
	c.afterLoad()
}

// autoTag gives every untagged painted pixel the part of its nearest tagged neighbour.
func autoTag(b *pixBuf) int {
	n := 0
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			if b.p[(y*b.w+x)*4+3] == 0 || b.pid[y*b.w+x] != 0 {
				continue
			}
			best, bd := uint8(0), 99
			for r := 1; r <= 8 && best == 0; r++ {
				for dy := -r; dy <= r; dy++ {
					for dx := -r; dx <= r; dx++ {
						xx, yy := x+dx, y+dy
						if xx < 0 || yy < 0 || xx >= b.w || yy >= b.h {
							continue
						}
						if id := b.pid[yy*b.w+xx]; id != 0 && abs(dx)+abs(dy) < bd {
							best, bd = id, abs(dx)+abs(dy)
						}
					}
				}
			}
			if best == 0 {
				best = PartTorso
			}
			b.pid[y*b.w+x] = best
			n++
		}
	}
	return n
}

func (c *CharEd) save() {
	name := safeName(c.name)
	if name == "" {
		c.ed.say("Pon un nombre al personaje")
		return
	}
	tagged := 0
	for _, v := range c.ca.Views {
		tagged += autoTag(v)
	}
	c.ca.Meta.Name = name
	c.ca.Meta.Category = c.category()
	if err := saveCharAsset(c.ed.root, c.ca); err != nil {
		c.ed.say("Error al guardar: " + err.Error())
		return
	}
	c.name = name
	c.reloadList()
	c.dirty = true
	msg := "Guardado en " + c.ca.Dir
	if tagged > 0 {
		msg += fmt.Sprintf(" (%d píxeles sin parte asignados solos)", tagged)
	}
	c.ed.say(msg)
}

func (c *CharEd) viewIndexForDir(d Dir) int {
	switch d {
	case DirUp:
		return 2
	case DirLeft, DirRight:
		return 1
	}
	return 0
}

func (c *CharEd) rebuildPreview() {
	for _, v := range c.ca.Views {
		if v.pid == nil {
			v.pid = make([]uint8, v.w*v.h)
		}
	}
	// work on a copy so that preview tagging does not alter the art being edited
	tmp := &CharAsset{Dir: c.ca.Dir, Meta: c.ca.Meta}
	for i, v := range c.ca.Views {
		cp := &pixBuf{w: v.w, h: v.h, p: append([]byte(nil), v.p...), pid: append([]uint8(nil), v.pid...)}
		autoTag(cp)
		tmp.Views[i] = cp
	}
	c.prevCS = newRigCharSet(tmp)
	c.dirty = false
}

func (c *CharEd) curBuf() *pixBuf {
	if c.ovMode && c.ovBuf != nil {
		return c.ovBuf
	}
	return c.ca.Views[c.view]
}

func (c *CharEd) loadOverrideFrame() {
	dir := prevDirs[c.prevDir]
	vw := vFront
	switch dir {
	case DirUp:
		vw = vBack
	case DirLeft, DirRight:
		vw = vSide
	}
	n := animFrames[c.prevAnim]
	c.ovFrame = clampI(c.ovFrame, 0, n-1)
	if c.dirty || c.prevCS == nil {
		c.rebuildPreview()
	}
	key := overrideKey(c.prevAnim, vw, c.ovFrame)
	path := filepath.Join(c.ca.Dir, "anim", key+".png")
	c.ovExists = false
	if c.ca.Dir != "" {
		if b, err := loadPNG(path); err == nil {
			c.ovBuf, c.ovExists = b, true
		}
	}
	if !c.ovExists {
		fb := c.prevCS.rig.render(vw, poseFor(c.prevAnim, c.ovFrame))
		c.ovBuf = fb
	}
	c.ovBuf.pid = nil
	c.ed.pt.clearHistory()
	c.cv.fit(c.ovBuf.w, c.ovBuf.h)
	c.cv.setPix(c.ovBuf)
}

func (c *CharEd) overridePath() string {
	dir := prevDirs[c.prevDir]
	vw := vFront
	switch dir {
	case DirUp:
		vw = vBack
	case DirLeft, DirRight:
		vw = vSide
	}
	return filepath.Join(c.ca.Dir, "anim", overrideKey(c.prevAnim, vw, c.ovFrame)+".png")
}

func (c *CharEd) update(u *UI) {
	pt := c.ed.pt
	pt.usePart = !c.ovMode
	// ---- left panel
	u.panel(4, 32, 236, 684)
	y := 40
	u.text(12, y, "Categoría", uiDim)
	if u.button(12, y+14, 218, 20, c.category(), false) {
		c.catIdx = (c.catIdx + 1) % len(charCategories)
		c.reloadList()
		c.listSel, c.listScroll = -1, 0
	}
	y += 40
	if i := u.list(12, y, 218, 12, c.list, c.listSel, &c.listScroll); i >= 0 {
		c.listSel = i
		c.load(c.list[i])
	}
	y += 12*18 + 12
	u.text(12, y, "Nombre (carpeta)", uiDim)
	u.input("charname", 12, y+14, 218, &c.name, 24)
	y += 42
	if u.button(12, y, 106, 22, "Guardar", false) {
		c.save()
	}
	if u.button(124, y, 106, 22, "Nuevo en blanco", false) {
		c.newBlank()
	}
	y += 28
	if u.button(12, y, 218, 22, "Plantilla: famoso (siguiente) ▶", false) {
		c.fromSpec(famousPresets[c.tplIdx%len(famousPresets)](rand.New(rand.NewSource(int64(c.tplIdx) + 11))))
		c.tplIdx++
	}
	y += 26
	if u.button(12, y, 106, 22, "Genérica", false) {
		c.fromSpec(genericSpec(rand.New(rand.NewSource(rand.Int63()))))
	}
	if u.button(124, y, 106, 22, "Protagonista", false) {
		c.fromSpec(playerSpec())
	}
	y += 30
	u.text(12, y, "Rol", uiDim)
	if u.button(40, y-4, 80, 20, "rol: "+orDash(c.ca.Meta.Role), false) {
		c.ca.Meta.Role = cycleStr(c.ca.Meta.Role, []string{"", "cop", "medic", "guard"})
	}
	u.text(130, y, "Objeto", uiDim)
	if u.button(172, y-4, 58, 20, orDash(c.ca.Meta.Carry), false) {
		c.ca.Meta.Carry = cycleStr(c.ca.Meta.Carry, []string{"", "knife", "cup", "mic", "baguette", "tray", "camera", "bigcup", "saber", "beer", "flag", "pizza", "sign"})
		c.dirty = true
	}
	u.tooltip(172, y-4, 58, 20, "Objeto que lleva en la mano")
	y += 26
	w, h := c.ca.Views[0].w, c.ca.Views[0].h
	u.text(12, y, fmt.Sprintf("Tamaño %dx%d", w, h), uiDim)
	if u.button(110, y-4, 24, 20, "W-", false) && w > 12 {
		c.resize(w-1, h)
	}
	if u.button(136, y-4, 24, 20, "W+", false) && w < 40 {
		c.resize(w+1, h)
	}
	if u.button(166, y-4, 24, 20, "H-", false) && h > 20 {
		c.resize(w, h-1)
	}
	if u.button(192, y-4, 24, 20, "H+", false) && h < 60 {
		c.resize(w, h+1)
	}
	y += 28
	if u.button(12, y, 218, 22, "Exportar hoja de sprites (PNG)", false) {
		c.exportSheet()
	}
	// ---- view tabs and canvas
	if !c.ovMode {
		if u.button(248, 36, 70, 22, "Frente", c.view == 0) {
			c.view = 0
			c.cv.setPix(c.ca.Views[0])
		}
		if u.button(322, 36, 120, 22, "Perfil (→)", c.view == 1) {
			c.view = 1
			c.cv.setPix(c.ca.Views[1])
		}
		if u.button(446, 36, 80, 22, "Espalda", c.view == 2) {
			c.view = 2
			c.cv.setPix(c.ca.Views[2])
		}
		if u.button(248, 62, 130, 20, "Frente → Espalda", false) {
			pt.pushUndo(c.ca.Views[2])
			copyView(c.ca.Views[0], c.ca.Views[2], false)
			c.view = 2
			c.cv.setPix(c.ca.Views[2])
			c.dirty = true
		}
		if u.button(382, 62, 144, 20, "Reflejar esta vista", false) {
			b := c.ca.Views[c.view]
			pt.pushUndo(b)
			flipView(b)
			c.cv.setPix(b)
			c.dirty = true
		}
		u.toggle(248, 84, 100, "Ver partes", &c.showParts)
		if u.button(352, 84, 176, 20, "Auto-asignar partes", false) {
			b := c.ca.Views[c.view]
			pt.pushUndo(b)
			autoParts(b, c.view == 1)
			c.cv.dirty, c.dirty = true, true
			c.ed.say("Partes asignadas por la silueta")
		}
	} else {
		u.text(248, 40, fmt.Sprintf("Editando fotograma a mano: %s · %s · %d", animNames[c.prevAnim], prevDirNames[c.prevDir], c.ovFrame), uiAccent)
		if c.ovExists {
			u.text(248, 58, "(ya existe un fotograma pintado a mano)", uiDim)
		} else {
			u.text(248, 58, "(partes del cuerpo no se usan aquí)", uiDim)
		}
	}
	buf := c.curBuf()
	if c.cv.img == nil || c.cv.img.Bounds().Dx() != buf.w || c.cv.img.Bounds().Dy() != buf.h {
		c.cv.fit(buf.w, buf.h)
		c.cv.setPix(buf)
	}
	// zoom, pan
	if c.cv.inView(u.mx, u.my) {
		if u.wheel > 0 {
			c.cv.zoomAt(u.mx, u.my, c.cv.zoom+1)
		} else if u.wheel < 0 {
			c.cv.zoomAt(u.mx, u.my, c.cv.zoom-1)
		}
	}
	if (u.mdown || (ebiten.IsKeyPressed(ebiten.KeySpace) && u.down)) && c.cv.inView(u.mx, u.my) {
		if c.lastMX != 0 || c.lastMY != 0 {
			c.cv.px += u.mx - c.lastMX
			c.cv.py += u.my - c.lastMY
		}
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
		if dx != 0 || dy != 0 {
			pt.pushUndo(buf)
			shiftView(buf, dx, dy)
			c.cv.dirty, c.dirty = true, true
		}
	}
	panning := u.mdown || (ebiten.IsKeyPressed(ebiten.KeySpace) && u.down)
	if !panning && pt.handle(u, &c.cv, buf, true) {
		c.cv.dirty, c.dirty = true, true
	}
	if pt.drawing || u.released {
		c.cv.dirty = true
	}
	if c.cv.dirty {
		c.cv.refresh(buf)
	}
	// ---- canvas drawing
	cv := c.cv
	showParts := c.showParts && !c.ovMode
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
					fillRect(vp, float64(cv.x+cv.px+xx*cv.zoom), float64(cv.y+cv.py+yy*cv.zoom), float64(cv.zoom), float64(cv.zoom), color.RGBA{pc.R / 2, pc.G / 2, pc.B / 2, 130})
				}
			}
		}
		pt.previewShape(func(x, y int) {
			fillRect(vp, float64(cv.x+cv.px+x*cv.zoom), float64(cv.y+cv.py+y*cv.zoom), float64(cv.zoom), float64(cv.zoom), color.RGBA{pt.col.R, pt.col.G, pt.col.B, 160})
		})
		// hover pixel
		hx, hy := cv.at(u.mx, u.my)
		if cv.inView(u.mx, u.my) && hx >= 0 && hy >= 0 && hx < buf.w && hy < buf.h {
			fillRect(vp, float64(cv.x+cv.px+hx*cv.zoom), float64(cv.y+cv.py+hy*cv.zoom), float64(cv.zoom), 1, uiAccent)
			fillRect(vp, float64(cv.x+cv.px+hx*cv.zoom), float64(cv.y+cv.py+(hy+1)*cv.zoom-1), float64(cv.zoom), 1, uiAccent)
		}
	})
	u.frame(cv.x-1, cv.y-1, cv.vw+2, cv.vh+2, uiLine)
	u.text(cv.x, cv.y+cv.vh+4, "Ctrl+Z deshacer · rueda: zoom · flechas: mover · arrastra un PNG para importarlo", uiDim)
	if c.ca.Dir == "" && !c.ovMode {
		u.text(cv.x, cv.y+cv.vh+18, "(sin guardar)", uiDim)
	}
	// ---- tools column
	u.panel(542, 32, 246, 684)
	u.text(550, 38, "HERRAMIENTAS", uiAccent)
	pt2 := u.paletteUI(pt, 550, 56, !c.ovMode)
	_ = pt2
	// ---- preview column
	c.previewPanel(u)
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func (c *CharEd) resize(w, h int) {
	for i, v := range c.ca.Views {
		nb := newPix(w, h)
		nb.pid = make([]uint8, w*h)
		// keep the art anchored to the bottom centre
		ox, oy := (w-v.w)/2, h-v.h
		for y := 0; y < v.h; y++ {
			for x := 0; x < v.w; x++ {
				nx, ny := x+ox, y+oy
				if nx < 0 || ny < 0 || nx >= w || ny >= h {
					continue
				}
				copy(nb.p[(ny*w+nx)*4:(ny*w+nx)*4+4], v.p[(y*v.w+x)*4:(y*v.w+x)*4+4])
				if v.pid != nil {
					nb.pid[ny*w+nx] = v.pid[y*v.w+x]
				}
			}
		}
		c.ca.Views[i] = nb
	}
	c.ed.pt.clearHistory()
	c.cv.fit(w, h)
	c.cv.setPix(c.ca.Views[c.view])
	c.dirty = true
}

func (c *CharEd) previewPanel(u *UI) {
	u.panel(794, 32, 482, 684)
	u.text(802, 38, "VISTA PREVIA (animaciones generadas)", uiAccent)
	if c.dirty || c.prevCS == nil {
		c.rebuildPreview()
	}
	names := make([]string, len(c.animList))
	sel := 0
	for i, a := range c.animList {
		names[i] = animNames[a]
		if a == c.prevAnim {
			sel = i
		}
	}
	if i := u.list(802, 58, 130, 22, names, sel, &c.animScr); i >= 0 {
		c.prevAnim = c.animList[i]
		c.prevT = 0
		if c.ovMode {
			c.loadOverrideFrame()
		}
	}
	for i, n := range prevDirNames[:3] {
		if u.button(942+i*80, 58, 76, 20, n, c.prevDir == i) {
			c.prevDir = i
			if c.ovMode {
				c.loadOverrideFrame()
			}
		}
	}
	if u.button(942+3*80, 58, 76, 20, "Perfil ◄", c.prevDir == 3) {
		c.prevDir = 3
		if c.ovMode {
			c.loadOverrideFrame()
		}
	}
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
	u.text(942, 84, fmt.Sprintf("fotograma %d/%d", frame+1, n), uiDim)
	px0, py0 := 950, 110
	zoom := 7
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	u.add(func(d *ebiten.Image) {
		fillRect(d, float64(px0-4), float64(py0-4), float64(w*zoom+8), float64(h*zoom+8), rgb(0x34404a))
		fillRect(d, float64(px0-4), float64(py0+h*zoom-14), float64(w*zoom+8), 18, rgb(0x2a3a2a))
		op := &ebiten.DrawImageOptions{}
		if flip {
			op.GeoM.Scale(-1, 1)
			op.GeoM.Translate(float64(w), 0)
		}
		op.GeoM.Scale(float64(zoom), float64(zoom))
		op.GeoM.Translate(float64(px0), float64(py0))
		d.DrawImage(img, op)
		// real size
		op2 := &ebiten.DrawImageOptions{}
		if flip {
			op2.GeoM.Scale(-1, 1)
			op2.GeoM.Translate(float64(w), 0)
		}
		op2.GeoM.Scale(2, 2)
		op2.GeoM.Translate(1210, 130)
		d.DrawImage(img, op2)
	})
	u.text(1190, 112, "Tamaño real x2", uiDim)
	// all frames strip
	sy := 470
	u.text(802, sy, "Todos los fotogramas", uiDim)
	sw := minI(n, 8)
	for f := 0; f < n; f++ {
		fi, fl := c.prevCS.img(c.prevAnim, prevDirs[c.prevDir], float64(f))
		if c.prevDir == 3 {
			fl = true
		} else if c.prevDir == 1 {
			fl = false
		}
		ox := 806 + (f%sw)*60
		oy := sy + 16 + (f/sw)*100
		fw, fh := fi.Bounds().Dx(), fi.Bounds().Dy()
		frameSel := c.ovMode && f == c.ovFrame
		u.add(func(d *ebiten.Image) {
			op := &ebiten.DrawImageOptions{}
			if fl {
				op.GeoM.Scale(-1, 1)
				op.GeoM.Translate(float64(fw), 0)
			}
			op.GeoM.Scale(2, 2)
			op.GeoM.Translate(float64(ox), float64(oy))
			d.DrawImage(fi, op)
			if frameSel {
				fillRect(d, float64(ox-2), float64(oy-2), float64(fw*2+4), 1, uiAccent)
				fillRect(d, float64(ox-2), float64(oy+fh*2+2), float64(fw*2+4), 1, uiAccent)
			}
		})
		if c.ovMode && u.in(ox, oy, fw*2, fh*2) && u.pressed {
			c.ovFrame = f
			c.loadOverrideFrame()
		}
	}
	by := 640
	if !c.ovMode {
		if u.button(802, by, 250, 22, "Editar este fotograma a mano", false) {
			if c.ca.Dir == "" {
				c.ed.say("Guarda primero el personaje")
			} else {
				c.ovMode = true
				c.ovFrame = int(c.prevT) % n
				c.loadOverrideFrame()
			}
		}
		u.text(802, by+28, "Los fotogramas pintados a mano sustituyen a los generados.", uiDim)
	} else {
		if u.button(802, by, 100, 22, "◄ anterior", false) {
			c.ovFrame = (c.ovFrame + n - 1) % n
			c.loadOverrideFrame()
		}
		if u.button(906, by, 100, 22, "siguiente ►", false) {
			c.ovFrame = (c.ovFrame + 1) % n
			c.loadOverrideFrame()
		}
		if u.button(1010, by, 130, 22, "Guardar fotograma", false) {
			ensureDir(filepath.Dir(c.overridePath()))
			if err := savePNG(c.overridePath(), c.ovBuf); err != nil {
				c.ed.say("Error: " + err.Error())
			} else {
				c.ed.say("Fotograma guardado")
				c.ovExists = true
				c.dirty = true
			}
		}
		if u.button(1144, by, 130, 22, "Quitar (volver)", false) {
			os.Remove(c.overridePath())
			c.ed.say("Fotograma restaurado al generado")
			c.dirty = true
			c.loadOverrideFrame()
		}
		if u.button(802, by+28, 250, 22, "Terminar y volver al dibujo base", false) {
			c.ovMode = false
			c.ed.pt.clearHistory()
			c.cv.fit(c.ca.Views[c.view].w, c.ca.Views[c.view].h)
			c.cv.setPix(c.ca.Views[c.view])
		}
	}
}

func copyView(src, dst *pixBuf, flip bool) {
	for y := 0; y < src.h && y < dst.h; y++ {
		for x := 0; x < src.w && x < dst.w; x++ {
			sx := x
			if flip {
				sx = src.w - 1 - x
			}
			copy(dst.p[(y*dst.w+x)*4:(y*dst.w+x)*4+4], src.p[(y*src.w+sx)*4:(y*src.w+sx)*4+4])
			if dst.pid != nil && src.pid != nil {
				id := src.pid[y*src.w+sx]
				if flip {
					id = swapPart(id)
				}
				dst.pid[y*dst.w+x] = id
			}
		}
	}
}

func flipView(b *pixBuf) {
	tmp := newPix(b.w, b.h)
	if b.pid != nil {
		tmp.pid = make([]uint8, b.w*b.h)
	}
	copyView(b, tmp, true)
	copy(b.p, tmp.p)
	if b.pid != nil {
		copy(b.pid, tmp.pid)
	}
}

func shiftView(b *pixBuf, dx, dy int) {
	np := make([]byte, len(b.p))
	var nid []uint8
	if b.pid != nil {
		nid = make([]uint8, len(b.pid))
	}
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			nx, ny := x+dx, y+dy
			if nx < 0 || ny < 0 || nx >= b.w || ny >= b.h {
				continue
			}
			copy(np[(ny*b.w+nx)*4:(ny*b.w+nx)*4+4], b.p[(y*b.w+x)*4:(y*b.w+x)*4+4])
			if nid != nil {
				nid[ny*b.w+nx] = b.pid[y*b.w+x]
			}
		}
	}
	copy(b.p, np)
	if nid != nil {
		copy(b.pid, nid)
	}
}

// exportSheet writes every animation of the character into one PNG for reference.
func (c *CharEd) exportSheet() {
	if c.ca.Dir == "" {
		c.ed.say("Guarda primero el personaje")
		return
	}
	c.rebuildPreview()
	w, h := c.ca.Views[0].w, c.ca.Views[0].h
	cols := 0
	for _, a := range c.animList {
		cols = maxI(cols, animFrames[a])
	}
	sheet := newPix(cols*3*w, len(c.animList)*h)
	for r, a := range c.animList {
		for v, vw := range []int{vFront, vSide, vBack} {
			for f := 0; f < animFrames[a]; f++ {
				fb := c.prevCS.rigFrame(a, vw, f)
				for y := 0; y < fb.h; y++ {
					for x := 0; x < fb.w; x++ {
						i := (y*fb.w + x) * 4
						if fb.p[i+3] > 0 {
							sheet.set((v*cols+f)*w+x, r*h+y, color.RGBA{fb.p[i], fb.p[i+1], fb.p[i+2], 255})
						}
					}
				}
			}
		}
	}
	path := filepath.Join(c.ca.Dir, "sheet.png")
	if err := savePNG(path, sheet); err != nil {
		c.ed.say("Error: " + err.Error())
		return
	}
	c.ed.say("Hoja guardada: " + path)
}

// autoParts tags a standing figure by its silhouette: head on top, legs at the bottom, torso in the
// middle and arms at the sides. side views get head, torso and one leg. It is a starting point to refine.
func autoParts(b *pixBuf, side bool) {
	if b.pid == nil {
		b.pid = make([]uint8, b.w*b.h)
	}
	x0, y0, x1, y1 := b.w, b.h, -1, -1
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			if b.p[(y*b.w+x)*4+3] > 0 {
				x0, y0, x1, y1 = minI(x0, x), minI(y0, y), maxI(x1, x), maxI(y1, y)
			}
		}
	}
	if x1 < 0 {
		return
	}
	H := y1 - y0 + 1
	cx := float64(x0+x1+1) / 2
	headEnd := y0 + int(0.26*float64(H))
	hipY := y0 + int(0.62*float64(H))
	// torso half width from the middle rows
	var widths []int
	for y := y0 + int(0.36*float64(H)); y < y0+int(0.5*float64(H)); y++ {
		n := 0
		for x := 0; x < b.w; x++ {
			if b.p[(y*b.w+x)*4+3] > 0 {
				n++
			}
		}
		widths = append(widths, n)
	}
	half := float64(x1-x0+1) * 0.34
	if len(widths) > 0 {
		sort.Ints(widths)
		half = float64(widths[len(widths)/2])/2 - 1.8
	}
	half = math.Max(2, half)
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			if b.p[(y*b.w+x)*4+3] == 0 {
				b.pid[y*b.w+x] = 0
				continue
			}
			id := uint8(PartTorso)
			switch {
			case y < headEnd:
				id = PartHead
			case y >= hipY:
				if side {
					id = PartLegR
				} else if float64(x)+0.5 < cx {
					id = PartLegL
				} else {
					id = PartLegR
				}
			case !side && math.Abs(float64(x)+0.5-cx) > half:
				if float64(x)+0.5 < cx {
					id = PartArmL
				} else {
					id = PartArmR
				}
			}
			b.pid[y*b.w+x] = id
		}
	}
}

// importDrops places dropped PNGs into the character: by file name (front/side/back, frente/perfil/espalda)
// or in order. The canvas takes the size of the image.
func (c *CharEd) importDrops(drops []droppedImage) {
	if c.ovMode && c.ovBuf != nil {
		d := drops[0].b
		for i := range c.ovBuf.p {
			c.ovBuf.p[i] = 0
		}
		for y := 0; y < d.h && y < c.ovBuf.h; y++ {
			for x := 0; x < d.w && x < c.ovBuf.w; x++ {
				copy(c.ovBuf.p[(y*c.ovBuf.w+x)*4:(y*c.ovBuf.w+x)*4+4], d.p[(y*d.w+x)*4:(y*d.w+x)*4+4])
			}
		}
		c.cv.dirty = true
		c.ed.say("Imagen importada al fotograma")
		return
	}
	w, h := drops[0].b.w, drops[0].b.h
	if w != c.ca.Views[0].w || h != c.ca.Views[0].h {
		c.resizeTo(w, h)
	}
	for i, d := range drops {
		vi := c.view
		if len(drops) > 1 {
			vi = i % 3
		}
		n := d.name
		switch {
		case strings.Contains(n, "side") || strings.Contains(n, "perfil"):
			vi = 1
		case strings.Contains(n, "back") || strings.Contains(n, "espalda"):
			vi = 2
		case strings.Contains(n, "front") || strings.Contains(n, "frente"):
			vi = 0
		}
		dst := c.ca.Views[vi]
		for k := range dst.p {
			dst.p[k] = 0
		}
		for y := 0; y < d.b.h && y < dst.h; y++ {
			for x := 0; x < d.b.w && x < dst.w; x++ {
				copy(dst.p[(y*dst.w+x)*4:(y*dst.w+x)*4+4], d.b.p[(y*d.b.w+x)*4:(y*d.b.w+x)*4+4])
			}
		}
		autoParts(dst, vi == 1)
		c.view = vi
	}
	c.cv.setPix(c.ca.Views[c.view])
	c.dirty = true
	c.ed.pt.clearHistory()
	c.ed.say("Imagen importada. Las partes se han asignado por la silueta: revísalas con 'Ver partes'")
}

func (c *CharEd) resizeTo(w, h int) {
	for i := range c.ca.Views {
		nb := newPix(w, h)
		nb.pid = make([]uint8, w*h)
		c.ca.Views[i] = nb
	}
	c.cv.fit(w, h)
}
