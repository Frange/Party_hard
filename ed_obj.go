package main

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Objects and animals. First choose objects or animals and the kind of object;
// then paint it (PINTAR) and say where it stands and how it is used (LÓGICA).
// ---------------------------------------------------------------------------

type objType struct {
	key, label, icon string
	w, h             int
	tip              string
}

var objTypes = []objType{
	{"furniture", "Muebles", "sofa", 40, 40, "Mesas, sofás, barras..."},
	{"decor", "Decoración", "star", 32, 32, "Cosas para adornar, sin más"},
	{"nature", "Naturaleza", "leaf", 40, 48, "Árboles, plantas, arbustos"},
	{"vehicle", "Vehículos", "car", 64, 34, "Pasan por la carretera"},
	{"trap", "Trampas", "trap", 24, 32, "Se arman con E y explotan o electrocutan"},
	{"usable", "Usables", "hand", 32, 40, "Se pueden beber, usar, esconderse..."},
	{"light", "Luces", "bulb", 16, 44, "Farolas y focos"},
}

var animalType = objType{"animal", "Animales", "paw", 32, 24, "Animales"}

// objTypeIndex infers which kind an object is from its properties, so that the
// objects exported long ago sort into the right list.
func objTypeIndex(m *ObjMeta) int {
	switch {
	case m.Category == "animal":
		return -1
	case m.Ambient > 0 || m.Category == "vehicle":
		return 3
	case m.Trap != "":
		return 4
	case m.Light:
		return 6
	case m.Hide || m.Phone || m.Poison || m.UseKind != "" || m.Category == "usable" || m.Category == "special":
		if m.UseKind != "" && !m.Hide && !m.Phone && !m.Poison && m.Category == "furniture" {
			return 0 // a sofa you can sit on is still furniture
		}
		return 5
	case m.Category == "nature":
		return 2
	case m.Category == "decor":
		return 1
	}
	return 0
}

type objEntry struct {
	name string
	typ  int
}

type ObjEd struct {
	ed       *Editor
	meta     ObjMeta
	frames   []*pixBuf
	dir      string
	entries  []objEntry
	list     []string // names of the current filter
	sel      int
	scroll   int
	find     string
	name     string
	frame    int
	cv       Canvas
	view     int // 0 objects, 1 animals
	typeIdx  int
	mode     int // 0 PINTAR, 1 LÓGICA
	ltool    int // logic tool: 0 anchor, 1 use zone, 2 collision
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
	thumbs   map[string]*ebiten.Image
	importCh chan *droppedImage
	imported bool // came from a picture: saved in the imports folder
}

func newObjEd(ed *Editor) *ObjEd {
	o := &ObjEd{ed: ed, sel: -1, thumbs: map[string]*ebiten.Image{}}
	o.cv = Canvas{x: 292, y: 84, vw: 290, vh: 560, zoom: 8, grid: true}
	o.newBlank()
	return o
}

func (o *ObjEd) curType() objType {
	if o.view == 1 {
		return animalType
	}
	return objTypes[clampI(o.typeIdx, 0, len(objTypes)-1)]
}

func (o *ObjEd) reloadList() {
	o.ed.dirCache = nil
	o.entries = nil
	for _, n := range spriteNames(o.ed.root, "objetos") {
		data, err := os.ReadFile(filepath.Join(spriteDirOf(o.ed.root, "objetos", n), "meta.json"))
		if err != nil {
			continue
		}
		var m ObjMeta
		if json_unmarshal(data, &m) != nil {
			continue
		}
		o.entries = append(o.entries, objEntry{n, objTypeIndex(&m)})
	}
	o.filterList()
}

func (o *ObjEd) filterList() {
	o.list = o.list[:0]
	want := o.typeIdx
	if o.view == 1 {
		want = -1
	}
	for _, e := range o.entries {
		if e.typ == want {
			o.list = append(o.list, e.name)
		}
	}
	sort.Strings(o.list)
}

// newBlank starts a new object of the current kind with sensible defaults.
func (o *ObjEd) newBlank() {
	o.imported = false
	t := o.curType()
	o.meta = ObjMeta{Name: "nuevo", Category: t.key}
	o.name = "nuevo"
	w, h := t.w, t.h
	o.frames = []*pixBuf{newPix(w, h)}
	o.dir = ""
	o.frame = 0
	o.anchor = [2]int{w / 2, h - 2}
	o.foot = [4]int{w / 6, h - 10, w - w/3, 8}
	o.usePts = nil
	o.mode, o.ltool = 0, 0
	o.applyTypeDefaults()
	o.afterLoad()
}

// applyTypeDefaults gives a fresh object the properties that fit its kind.
func (o *ObjEd) applyTypeDefaults() {
	m := &o.meta
	t := o.curType()
	m.Category = t.key
	m.Solid, m.Hide, m.Phone, m.Poison, m.Light = true, false, false, false, false
	m.Trap, m.UseKind, m.Ambient = "", "", 0
	switch t.key {
	case "decor":
		m.Solid = false
	case "vehicle":
		m.Solid = false
		m.Ambient = 90
		o.foot = [4]int{4, o.frames[0].h - 6, o.frames[0].w - 8, 5}
	case "trap":
		m.Trap = "explode"
	case "usable":
		m.UseKind = "lounge"
	case "light":
		m.Light = true
		m.LightR, m.LightA = 110, 120
		m.LightCol = [3]int{150, 118, 52}
	case "animal":
		m.Solid = false
		m.FPS = 6
	}
}

func (o *ObjEd) setView(v, ti int) {
	o.view, o.typeIdx = v, ti
	o.filterList()
	o.scroll = 0
	if len(o.list) > 0 {
		o.sel = 0
		o.loadByName(o.list[0])
	} else {
		o.sel = -1
		o.newBlank()
	}
}

func (o *ObjEd) afterLoad() {
	o.ed.pt.clearHistory()
	o.ed.pt.usePart = false
	o.cv.fit(o.frames[0].w, o.frames[0].h)
	o.cv.setPix(o.frames[o.frame])
}

func (o *ObjEd) thumbFor(name string) *ebiten.Image {
	dir := o.ed.spriteDir("objetos", name)
	if im, ok := o.thumbs[dir]; ok {
		return im
	}
	b, err := loadPNG(filepath.Join(dir, "frame0.png"))
	if err != nil {
		o.thumbs[dir] = nil
		return nil
	}
	im := ebiten.NewImage(b.w, b.h)
	im.WritePixels(b.p)
	o.thumbs[dir] = im
	return im
}

func (o *ObjEd) loadByName(name string) {
	dir := spriteDirOf(o.ed.root, "objetos", name)
	o.imported = inImports(o.ed.root, dir)
	oa, err := loadObjAsset(dir)
	if err != nil {
		o.ed.say("No se pudo cargar: " + err.Error())
		return
	}
	o.meta = oa.Meta
	if ti := objTypeIndex(&o.meta); ti < 0 {
		o.view = 1
	} else {
		o.view, o.typeIdx = 0, ti
	}
	o.filterList()
	for i, n := range o.list {
		if n == name {
			o.sel = i
			o.scroll = maxI(0, i/4-1)
		}
	}
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
	m.Category = o.curType().key
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
	saveRoot, nm := importSaveRoot(o.ed.root, o.imported, "objetos", m.Name)
	oa.Meta.Name = nm
	if err := saveObjAsset(saveRoot, oa); err != nil {
		o.ed.say("Error al guardar: " + err.Error())
		return
	}
	o.dir = oa.Dir
	o.name = nm
	o.meta = oa.Meta
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

func json_unmarshal(data []byte, v any) error { return jsonUnmarshal(data, v) }

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
	o.imported = true
	o.ed.pt.clearHistory()
	o.cv.fit(o.frames[0].w, o.frames[0].h)
	o.cv.setPix(o.frames[o.frame])
	o.ed.say("Imagen importada. Marca el ancla y la huella de colisión")
}
