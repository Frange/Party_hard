package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ---------------------------------------------------------------------------
// Asset folders. Everything the game shows can live in plain files:
//
//   assets/sprites/main_character/<name>/   the player
//   assets/sprites/people/<name>/           random party guests
//   assets/sprites/special_people/<name>/   famous / themed guests
//   assets/sprites/enemies/<name>/          security guards
//   assets/sprites/special_enemies/<name>/  tougher guards
//   assets/sprites/police/<name>/           police officers and paramedics (meta role)
//   assets/sprites/objetos/<name>/          props, vehicles, animals, traps
//   assets/backgrounds/<name>/              scenarios (painted background + logic)
//
// A character folder holds front.png, side.png, back.png (colours) and
// front_parts.png, side_parts.png, back_parts.png (which body part each pixel is),
// plus meta.json. Animations are generated from those; hand drawn frames can be
// dropped into anim/<animation>_<view>_<frame>.png to override generated ones.
// ---------------------------------------------------------------------------

var charCategories = []string{"main_character", "people", "special_people", "enemies", "special_enemies", "police"}

const (
	PartHead  = 1
	PartTorso = 2
	PartArmL  = 3 // screen-left arm (far arm in profile)
	PartArmR  = 4 // screen-right arm (near arm in profile)
	PartLegL  = 5
	PartLegR  = 6
)

var partNames = []string{"Ninguna", "Cabeza", "Torso", "Brazo I", "Brazo D", "Pierna I", "Pierna D"}

var partColors = []color.RGBA{{0, 0, 0, 0}, {255, 80, 80, 255}, {80, 200, 80, 255}, {80, 140, 255, 255}, {255, 220, 60, 255}, {220, 90, 255, 255}, {60, 230, 230, 255}}

// assetsRoot returns the assets directory. It is looked up next to the executable (next to the
// .app bundle on macOS) and in the working directory. With create it is made next to the executable.
func assetsRoot(create bool) string {
	var cands []string
	if exe, err := os.Executable(); err == nil {
		d := filepath.Dir(exe)
		if strings.Contains(d, ".app/Contents/MacOS") {
			cands = append(cands, filepath.Join(d, "..", "..", "..", "assets"))
		}
		cands = append(cands, filepath.Join(d, "assets"))
	}
	if wd, err := os.Getwd(); err == nil {
		cands = append(cands, filepath.Join(wd, "assets"))
	}
	for _, c := range cands {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			return filepath.Clean(c)
		}
	}
	if !create || len(cands) == 0 {
		return ""
	}
	os.MkdirAll(cands[0], 0o755)
	return filepath.Clean(cands[0])
}

func spritesDir(root, cat string) string { return filepath.Join(root, "sprites", cat) }
func backgroundsDir(root string) string  { return filepath.Join(root, "backgrounds") }

func listDirs(path string) []string {
	ents, err := os.ReadDir(path)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range ents {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

// ---------------------------------------------------------------------------
// PNG <-> pixBuf
// ---------------------------------------------------------------------------

func loadPNG(path string) (*pixBuf, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	im, err := png.Decode(f)
	if err != nil {
		return nil, err
	}
	return pixBufFromImage(im), nil
}

func pixBufFromImage(im image.Image) *pixBuf {
	bnd := im.Bounds()
	b := newPix(bnd.Dx(), bnd.Dy())
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			r, g, bl, a := im.At(bnd.Min.X+x, bnd.Min.Y+y).RGBA()
			i := (y*b.w + x) * 4
			if a == 0 {
				continue
			}
			// store premultiplied-corrected straight colour, opaque only if alpha is high
			b.p[i], b.p[i+1], b.p[i+2], b.p[i+3] = uint8(r*255/a), uint8(g*255/a), uint8(bl*255/a), uint8(a>>8)
		}
	}
	return b
}

func savePNG(path string, b *pixBuf) error {
	os.MkdirAll(filepath.Dir(path), 0o755)
	im := image.NewNRGBA(image.Rect(0, 0, b.w, b.h))
	for i := 0; i < b.w*b.h; i++ {
		im.Pix[i*4], im.Pix[i*4+1], im.Pix[i*4+2], im.Pix[i*4+3] = b.p[i*4], b.p[i*4+1], b.p[i*4+2], b.p[i*4+3]
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, im)
}

// part maps are stored as grayscale PNGs: value = part id * 32.
func loadParts(path string, w, h int) []uint8 {
	out := make([]uint8, w*h)
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()
	im, err := png.Decode(f)
	if err != nil {
		return out
	}
	bnd := im.Bounds()
	for y := 0; y < h && y < bnd.Dy(); y++ {
		for x := 0; x < w && x < bnd.Dx(); x++ {
			r, _, _, _ := im.At(bnd.Min.X+x, bnd.Min.Y+y).RGBA()
			id := uint8((r >> 8) / 32)
			if id > 6 {
				id = 0
			}
			out[y*w+x] = id
		}
	}
	return out
}

func saveParts(path string, pid []uint8, w, h int) error {
	os.MkdirAll(filepath.Dir(path), 0o755)
	im := image.NewGray(image.Rect(0, 0, w, h))
	for i, v := range pid {
		im.Pix[i] = v * 32
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, im)
}

// ---------------------------------------------------------------------------
// Characters
// ---------------------------------------------------------------------------

type CharMeta struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name,omitempty"` // proper name shown in on-screen messages, e.g. "Sonic"
	Category    string `json:"category"`
	Role        string `json:"role,omitempty"`   // cop | medic | guard
	Gender      string `json:"gender,omitempty"` // "" | m | f — only affects the AnimPee pose
	Carry       string `json:"carry,omitempty"`
	Source      string `json:"source,omitempty"` // source PNG imported from sprites/togenerate
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Weight      int    `json:"weight,omitempty"`
	Edited      bool   `json:"edited,omitempty"`    // saved from the editor: the game uses this drawing for the character of the same name
	ShopOnly    bool   `json:"shop_only,omitempty"` // exported later, to be edited: never picked as a random guest
}

type CharAsset struct {
	Dir   string
	Meta  CharMeta
	Views [3]*pixBuf // front, side, back (colours). pid holds the part map.
}

func newCharAsset(name, cat string, w, h int) *CharAsset {
	ca := &CharAsset{Meta: CharMeta{Name: name, Category: cat, Width: w, Height: h}}
	for i := range ca.Views {
		ca.Views[i] = newPix(w, h)
		ca.Views[i].pid = make([]uint8, w*h)
	}
	return ca
}

var viewNames = []string{"front", "side", "back"}

func loadCharAsset(dir string) (*CharAsset, error) {
	data, err := os.ReadFile(filepath.Join(dir, "meta.json"))
	if err != nil {
		return nil, err
	}
	ca := &CharAsset{Dir: dir}
	if err := json.Unmarshal(data, &ca.Meta); err != nil {
		return nil, err
	}
	for i, vn := range viewNames {
		b, err := loadPNG(filepath.Join(dir, vn+".png"))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", vn, err)
		}
		b.pid = loadParts(filepath.Join(dir, vn+"_parts.png"), b.w, b.h)
		ca.Views[i] = b
	}
	ca.Meta.Width, ca.Meta.Height = ca.Views[0].w, ca.Views[0].h
	return ca, nil
}

func saveCharAsset(root string, ca *CharAsset) error {
	dir := filepath.Join(spritesDir(root, ca.Meta.Category), ca.Meta.Name)
	os.MkdirAll(dir, 0o755)
	ca.Dir = dir
	ca.Meta.Width, ca.Meta.Height = ca.Views[0].w, ca.Views[0].h
	data, _ := json.MarshalIndent(ca.Meta, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "meta.json"), data, 0o644); err != nil {
		return err
	}
	for i, vn := range viewNames {
		if err := savePNG(filepath.Join(dir, vn+".png"), ca.Views[i]); err != nil {
			return err
		}
		if err := saveParts(filepath.Join(dir, vn+"_parts.png"), ca.Views[i].pid, ca.Views[i].w, ca.Views[i].h); err != nil {
			return err
		}
	}
	return nil
}

// processToGenerate turns a supplied reference sprite into an editable rigged
// character exactly once. Put one PNG per character in either
// assets/sprites/togenerate/ or project/sprites/togenerate/. The original file
// is never overwritten; a tiny .done marker records that its editable character
// folder already exists. The editor can then refine the automatic part map.
func processToGenerate(root string) int {
	if root == "" {
		return 0
	}
	inputs := []string{spritesDir(root, "togenerate"), filepath.Join(filepath.Dir(root), "sprites", "togenerate")}
	made := 0
	for _, in := range inputs {
		ents, err := os.ReadDir(in)
		if err != nil {
			continue
		}
		for _, e := range ents {
			if e.IsDir() || strings.ToLower(filepath.Ext(e.Name())) != ".png" {
				continue
			}
			stem := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
			name := strings.ToLower(strings.TrimSpace(stem))
			name = strings.Map(func(r rune) rune {
				if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
					return r
				}
				return '-'
			}, name)
			name = strings.Trim(name, "-")
			if name == "" {
				continue
			}
			marker := filepath.Join(in, "generated", name+".done")
			out := filepath.Join(spritesDir(root, "special_people"), name, "meta.json")
			if _, err := os.Stat(marker); err == nil {
				continue
			}
			if _, err := os.Stat(out); err == nil {
				os.MkdirAll(filepath.Dir(marker), 0o755)
				os.WriteFile(marker, []byte("already present\n"), 0o644)
				continue
			}
			src, err := loadPNG(filepath.Join(in, e.Name()))
			if err != nil {
				fmt.Println("togenerate: skipping", e.Name(), err)
				continue
			}
			w, h := maxI(32, src.w+8), maxI(48, src.h+6)
			ca := newCharAsset(name, "special_people", w, h)
			ca.Meta.Source = e.Name()
			// Center the supplied front pose on a roomy canvas, feet aligned at the
			// baseline so generated walk/dance animations have a stable pivot.
			x0, y0 := (w-src.w)/2, h-src.h-2
			for y := 0; y < src.h; y++ {
				for x := 0; x < src.w; x++ {
					si, di := (y*src.w+x)*4, ((y0+y)*w+x0+x)*4
					copy(ca.Views[0].p[di:di+4], src.p[si:si+4])
				}
			}
			// A mirrored profile/back is deliberately only a starting point. It is
			// valid for the rig and immediately editable in the Character tab.
			for view := 1; view < 3; view++ {
				for y := 0; y < h; y++ {
					for x := 0; x < w; x++ {
						si, di := (y*w+(w-1-x))*4, (y*w+x)*4
						copy(ca.Views[view].p[di:di+4], ca.Views[0].p[si:si+4])
					}
				}
			}
			for i := range ca.Views {
				autoParts(ca.Views[i], i == 1)
			}
			if err := saveCharAsset(root, ca); err != nil {
				fmt.Println("togenerate:", err)
				continue
			}
			os.MkdirAll(filepath.Dir(marker), 0o755)
			os.WriteFile(marker, []byte("generated "+name+"\n"), 0o644)
			made++
		}
	}
	return made
}

// ---------------------------------------------------------------------------
// Objects (props, vehicles, animals, traps)
// ---------------------------------------------------------------------------

type ObjMeta struct {
	Name     string  `json:"name"`
	Category string  `json:"category"` // furniture | decor | vehicle | animal | trap | nature
	AnchorX  int     `json:"anchor_x"` // native pixels, bottom-centre of the footprint
	AnchorY  int     `json:"anchor_y"`
	FootX    int     `json:"foot_x"` // footprint rectangle in sprite pixels
	FootY    int     `json:"foot_y"`
	FootW    int     `json:"foot_w"`
	FootH    int     `json:"foot_h"`
	Solid    bool    `json:"solid"`
	Hide     bool    `json:"hide"`
	Phone    bool    `json:"phone"`
	Poison   bool    `json:"poison"`
	Trap     string  `json:"trap"` // "", explode, electric
	Light    bool    `json:"light"`
	Variants bool    `json:"variants"` // frames are variants (level picks one) instead of an animation
	FPS      float64 `json:"fps"`
	SortOff  int     `json:"sort_off"`
	UseKind  string  `json:"use_kind,omitempty"` // drink | toilet | lounge
	UseX     []int   `json:"use_x,omitempty"`
	UseY     []int   `json:"use_y,omitempty"`
	Ambient  float64 `json:"ambient_speed"` // >0: crosses the street now and then
	LightR   int     `json:"light_r,omitempty"`     // light radius in pixels (0 = the default)
	LightCol [3]int  `json:"light_col,omitempty"`   // light colour
	LightA   int     `json:"light_a,omitempty"`     // light strength 0..255 (0 = the default)
}

type ObjAsset struct {
	Dir    string
	Meta   ObjMeta
	Frames []*pixBuf
}

func loadObjAsset(dir string) (*ObjAsset, error) {
	data, err := os.ReadFile(filepath.Join(dir, "meta.json"))
	if err != nil {
		return nil, err
	}
	oa := &ObjAsset{Dir: dir}
	if err := json.Unmarshal(data, &oa.Meta); err != nil {
		return nil, err
	}
	for i := 0; i < 64; i++ {
		b, err := loadPNG(filepath.Join(dir, fmt.Sprintf("frame%d.png", i)))
		if err != nil {
			break
		}
		oa.Frames = append(oa.Frames, b)
	}
	if len(oa.Frames) == 0 {
		return nil, fmt.Errorf("no frames in %s", dir)
	}
	return oa, nil
}

func saveObjAsset(root string, oa *ObjAsset) error {
	dir := filepath.Join(spritesDir(root, "objetos"), oa.Meta.Name)
	os.MkdirAll(dir, 0o755)
	// remove stale frames
	for i := 0; i < 64; i++ {
		os.Remove(filepath.Join(dir, fmt.Sprintf("frame%d.png", i)))
	}
	oa.Dir = dir
	data, _ := json.MarshalIndent(oa.Meta, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "meta.json"), data, 0o644); err != nil {
		return err
	}
	for i, f := range oa.Frames {
		if err := savePNG(filepath.Join(dir, fmt.Sprintf("frame%d.png", i)), f); err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Levels (backgrounds)
// ---------------------------------------------------------------------------

const logicCell = 4 // native pixels per logic cell

type LevelProp struct {
	Kind string `json:"kind"`
	X    int    `json:"x"` // native pixels, anchor point
	Y    int    `json:"y"`
	Var  int    `json:"var,omitempty"`
}

// LevelFloor and LevelRect describe the ground and the water of one of the game's own scenarios
// (only used by the corrections of the originals, see level_override.go).
type LevelFloor struct {
	X     int    `json:"x"`
	Y     int    `json:"y"`
	W     int    `json:"w"`
	H     int    `json:"h"`
	Kind  string `json:"kind"`
	Color uint32 `json:"color,omitempty"`
}

type LevelRect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type LevelPoint struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type LevelPerf struct {
	X, Y int
	Role string
}

type LevelLight struct {
	X, Y  int
	R     int
	Color [3]uint8
	Alpha uint8
	Pulse bool
}

type LevelAnimal struct {
	Kind   string `json:"kind"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Radius int    `json:"radius"`
}

type LevelMeta struct {
	Animals    []LevelAnimal  `json:"animals,omitempty"`
	Name       string         `json:"name"`
	Override   string         `json:"override,omitempty"` // campaign key: this is a correction of one of the game's own scenarios
	Title      string         `json:"title"`
	Intro      string         `json:"intro"`
	Police     string         `json:"police"` // car | heli
	Tint       [4]uint8       `json:"tint"`
	Guests     int            `json:"guests"`
	Swim       bool           `json:"swim"`
	Disco      bool           `json:"disco"`
	DanceFloor bool           `json:"dance_floor"` // animate neon squares over the dance zone
	DanceCell  int            `json:"dance_cell"`
	Strobe     []uint32       `json:"strobe,omitempty"`
	Start      LevelPoint     `json:"start"`
	CarFrom    LevelPoint     `json:"car_from"`
	CarTo      LevelPoint     `json:"car_to"`
	Pads       []LevelPoint   `json:"pads"`
	Guards     [][]LevelPoint `json:"guard_routes"`
	Performers []LevelPerf    `json:"performers"`
	Lights     []LevelLight   `json:"lights"`
	Props      []LevelProp    `json:"props"`
	Floors     []LevelFloor   `json:"floors,omitempty"` // originals only: the ground (helipads, dance floors, rugs...)
	Pools      []LevelRect    `json:"pools,omitempty"`  // originals only: the water
	// Logic layers, one string per row, one char per logicCell x logicCell native pixels.
	// Solid: '.' free, '#' wall (blocks movement and sight), '=' low (blocks movement only), '~' water, ',' slow ground (walkable, 55% speed), 'w' window (blocks walking, E jumps through)
	Solid []string `json:"solid"`
	// Road and ambience, all optional (zero values keep the old behaviour).
	TimeOfDay string `json:"time_of_day,omitempty"` // night (default) | dusk | day
	RoadH     int    `json:"road_h,omitempty"`      // width of the road in native pixels; each lane sits a quarter of it from the centre line
	Traffic   string `json:"traffic,omitempty"`     // both (default) | right | left
	ParkX0    int    `json:"park_x0,omitempty"`     // x range along the road where police and ambulances stop
	ParkX1    int    `json:"park_x1,omitempty"`
	// Zone: '.' none, 'r' guests wander here, 'd' dance floor
	Zone []string `json:"zone"`
}

type LevelAsset struct {
	Dir  string
	Meta LevelMeta
	BG   *pixBuf
}

func newLevelAsset(name string) *LevelAsset {
	la := &LevelAsset{}
	la.Meta = LevelMeta{Name: name, Title: strings.ToUpper(name), Intro: "", Police: "car", Guests: 24, DanceCell: 12,
		Tint: [4]uint8{0, 6, 30, 80}, Start: LevelPoint{320, 330}, CarFrom: LevelPoint{700, 340}, CarTo: LevelPoint{480, 340}}
	la.BG = newPix(screenW, screenH)
	la.BG.rect(0, 0, screenW, screenH, rgb(0x2a3a30))
	la.Meta.Solid, la.Meta.Zone = blankLogic()
	return la
}

func blankLogic() ([]string, []string) {
	w, h := screenW/logicCell, screenH/logicCell
	row := strings.Repeat(".", w)
	s, z := make([]string, h), make([]string, h)
	for i := range s {
		s[i], z[i] = row, row
	}
	return s, z
}

func loadLevelAsset(dir string) (*LevelAsset, error) {
	data, err := os.ReadFile(filepath.Join(dir, "level.json"))
	if err != nil {
		return nil, err
	}
	la := &LevelAsset{Dir: dir}
	if err := json.Unmarshal(data, &la.Meta); err != nil {
		return nil, err
	}
	bg, err := loadPNG(filepath.Join(dir, "background.png"))
	if err != nil {
		return nil, err
	}
	la.BG = bg
	w, h := screenW/logicCell, screenH/logicCell
	fix := func(rows []string) []string {
		out := make([]string, h)
		for y := range out {
			r := ""
			if y < len(rows) {
				r = rows[y]
			}
			for len(r) < w {
				r += "."
			}
			out[y] = r[:w]
		}
		return out
	}
	la.Meta.Solid, la.Meta.Zone = fix(la.Meta.Solid), fix(la.Meta.Zone)
	return la, nil
}

func saveLevelAsset(root string, la *LevelAsset) error {
	dir := filepath.Join(backgroundsDir(root), la.Meta.Name)
	os.MkdirAll(dir, 0o755)
	la.Dir = dir
	data, _ := json.MarshalIndent(la.Meta, "", " ")
	if err := os.WriteFile(filepath.Join(dir, "level.json"), data, 0o644); err != nil {
		return err
	}
	return savePNG(filepath.Join(dir, "background.png"), la.BG)
}

// ---------------------------------------------------------------------------
// Store: everything loaded from disk
// ---------------------------------------------------------------------------

type AssetStore struct {
	Root   string
	Chars  map[string][]*CharAsset
	Objs   []*ObjAsset
	Levels []*LevelAsset
}

func loadAssetStore() *AssetStore {
	root := assetsRoot(false)
	if root == "" {
		return nil
	}
	st := &AssetStore{Root: root, Chars: map[string][]*CharAsset{}}
	for _, cat := range charCategories {
		for _, name := range spriteNames(root, cat) {
			ca, err := loadCharAsset(spriteDirOf(root, cat, name))
			if err != nil {
				fmt.Println("assets: skipping character", cat, name, ":", err)
				continue
			}
			ca.Meta.Category = cat
			if ca.Meta.Name == "" {
				ca.Meta.Name = name
			}
			st.Chars[cat] = append(st.Chars[cat], ca)
		}
	}
	for _, name := range spriteNames(root, "objetos") {
		oa, err := loadObjAsset(spriteDirOf(root, "objetos", name))
		if err != nil {
			fmt.Println("assets: skipping object", name, ":", err)
			continue
		}
		if oa.Meta.Name == "" {
			oa.Meta.Name = name
		}
		st.Objs = append(st.Objs, oa)
	}
	for _, name := range backgroundNames(root) {
		la, err := loadLevelAsset(backgroundDirOf(root, name))
		if err != nil {
			fmt.Println("assets: skipping background", name, ":", err)
			continue
		}
		if la.Meta.Name == "" {
			la.Meta.Name = name
		}
		st.Levels = append(st.Levels, la)
	}
	return st
}

func (st *AssetStore) pickChar(cat string, rnd func(int) int) *CharAsset {
	l := st.Chars[cat]
	if len(l) == 0 {
		return nil
	}
	return l[rnd(len(l))]
}

func clampI(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

var _ = math.Pi
