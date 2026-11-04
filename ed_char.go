package main

import (
	"fmt"
	"image/color"
	"math"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

type CharEd struct {
	ed         *Editor
	ca         *CharAsset
	typeIdx    int // which kind of character is being edited (see charTypes)
	subIdx     int
	mode       int // 0 DIBUJO, 1 PARTES
	thumbs     map[string]*ebiten.Image
	importCh   chan *droppedImage
	list       []string
	listSel    int
	partsImg   *ebiten.Image // cached overlay of the part colours
	imported   bool          // came from a picture: saved in the imports folder
	find       string
	listScroll int
	name       string
	view       int
	cv         Canvas
	wiz        *CharWizard
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
	c := &CharEd{ed: ed, name: "nuevo", prevAnim: AnimWalk, typeIdx: 1, thumbs: map[string]*ebiten.Image{}}
	for a := 0; a < AnimCount; a++ {
		if a != AnimLying && a != AnimSleep {
			c.animList = append(c.animList, a)
		}
	}
	c.ca = newCharAsset("nuevo", "people", charW, charH)
	c.ca.Meta.Gender = "m"
	c.cv = Canvas{x: 288, y: 84, vw: 300, vh: 600, zoom: 10, grid: true}
	c.cv.fit(charW, charH)
	c.dirty = true
	return c
}

func (c *CharEd) category() string { return c.sub().folder }

func (c *CharEd) reloadList() {
	c.ed.dirCache = nil
	sub := c.sub()
	var out []string
	for _, n := range spriteNames(c.ed.root, sub.folder) {
		if sub.matches(c.ed.roleOf(c.ed.spriteDir(sub.folder, n))) {
			out = append(out, n)
		}
	}
	c.list = out
}

func (c *CharEd) load(name string) {
	dir := spriteDirOf(c.ed.root, c.category(), name)
	ca, err := loadCharAsset(dir)
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
	c.imported = inImports(c.ed.root, dir)
	c.ca.Meta.Category = c.category()
	if c.ca.Meta.Gender == "" {
		c.ca.Meta.Gender = "m"
	}
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
	c.imported = false
	c.ca = newCharAsset("nuevo", c.category(), charW, charH)
	c.ca.Meta.Role = c.sub().role
	c.ca.Meta.Gender = "m"
	c.name = "nuevo"
	c.afterLoad()
}

func (c *CharEd) fromSpec(spec CharSpec) {
	c.imported = false
	role := ""
	switch c.category() {
	case "police":
		role = "cop"
	case "enemies", "special_enemies":
		role = "guard"
	}
	c.ca = charAssetFromSpec(spec, safeName(spec.Name+"_copia"), c.category(), role)
	if spec.Name == "" {
		c.ca.Meta.Name = "nuevo"
	}
	c.ca.Meta.Gender = "m"
	if spec.Female {
		c.ca.Meta.Gender = "f"
	}
	c.ca.Meta.Role = c.sub().role
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
	c.ca.Meta.Role = c.sub().role
	c.ca.Meta.Edited = true
	c.ca.Meta.ShopOnly = false
	c.ed.roleCache = nil
	if c.ca.Meta.Gender == "" {
		c.ca.Meta.Gender = "m"
	}
	saveRoot, name := importSaveRoot(c.ed.root, c.imported, c.category(), name)
	c.ca.Meta.Name = name
	if err := saveCharAsset(saveRoot, c.ca); err != nil {
		c.ed.say("Error al guardar: " + err.Error())
		return
	}
	c.name = name
	delete(c.thumbs, c.ca.Dir)
	c.reloadList()
	for i, n := range c.list {
		if n == name {
			c.listSel = i
		}
	}
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
	c.imported = true
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
