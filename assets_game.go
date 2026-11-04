package main

import (
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Turning asset files into game content
// ---------------------------------------------------------------------------

// applyObjects registers the objects found on disk as prop kinds (they replace built-ins of the same name).
func applyObjects(objs []*ObjAsset) {
	for _, oa := range objs {
		m := oa.Meta
		if m.Name == "phone" {
			continue // the red booth is always the built-in one (older exports kept a different drawing)
		}
		f0 := oa.Frames[0]
		k := &PropKind{
			W: int(float64(f0.w) / wk), H: int(float64(f0.h) / wk),
			FW: float64(m.FootW) / wk, FH: float64(m.FootH) / wk,
			NoSolid: !m.Solid, Hide: m.Hide, Phone: m.Phone, Poison: m.Poison, Trap: m.Trap, Light: m.Light,
			SortOff: float64(m.SortOff) / wk, UseKind: m.UseKind,
			Native: oa.Frames, NAX: float64(m.AnchorX), NAY: float64(m.AnchorY), FPS: m.FPS,
			FootDX: float64(m.FootX-m.AnchorX) / wk, FootDY: float64(m.FootY-m.AnchorY) / wk, hasFoot: true,
		}
		if m.Variants {
			k.Vars = len(oa.Frames)
		}
		for i := range m.UseX {
			if i < len(m.UseY) {
				k.Use = append(k.Use, Vec{float64(m.UseX[i]) / wk, float64(m.UseY[i]) / wk})
			}
		}
		k.Ambient = m.Ambient
		if m.Light && m.LightR > 0 {
			k.LightR = float64(m.LightR)
			a := m.LightA
			if a <= 0 {
				a = 120
			}
			k.LightC = color.RGBA{uint8(clampI(m.LightCol[0], 0, 255)), uint8(clampI(m.LightCol[1], 0, 255)), uint8(clampI(m.LightCol[2], 0, 255)), uint8(clampI(a, 0, 255))}
		}
		k.Category = m.Category
		if old := propKinds[m.Name]; old != nil && len(old.Solids) > 0 {
			// a built-in walkable prop (the stage): keep its built-in solid parts
			k.Solids, k.NoSolid = old.Solids, old.NoSolid
		}
		propKinds[m.Name] = k
	}
}

// rigFor returns (and caches) the animated character set for an asset.
func (g *Game) rigFor(ca *CharAsset) *CharSet {
	if cs, ok := g.rigCache[ca]; ok {
		return cs
	}
	cs := newRigCharSet(ca)
	g.rigCache[ca] = cs
	return cs
}

func (g *Game) pickAsset(cat string) *CharAsset {
	if g.store == nil {
		return nil
	}
	l := g.store.Chars[cat]
	if len(l) == 0 {
		return nil
	}
	return l[g.rng.Intn(len(l))]
}

// newGuestCS picks a guest from the asset folders, or falls back to the
// built-in generator. The second return value is the guest's proper display
// name for on-screen messages, or "" for an ordinary unnamed guest.
func (g *Game) newGuestCS() (*CharSet, string) {
	if g.store != nil {
		people, special := g.store.Chars["people"], g.store.Chars["special_people"]
		if len(people)+len(special) > 0 {
			var ca *CharAsset
			isSpecial := len(special) > 0 && (len(people) == 0 || g.rng.Float64() < 0.4)
			for try := 0; try < 24; try++ {
				if isSpecial {
					ca = special[g.rng.Intn(len(special))]
				} else {
					ca = people[g.rng.Intn(len(people))]
				}
				// those are only there to be edited; and a famous face turns up once per party
				if !ca.Meta.ShopOnly && !(isSpecial && g.famousPick[ca.Meta.Name]) {
					break
				}
			}
			if isSpecial && (ca.Meta.ShopOnly || g.famousPick[ca.Meta.Name]) && len(people) > 0 {
				isSpecial, ca = false, people[g.rng.Intn(len(people))] // the famous ones are all here already
			}
			name := ""
			if isSpecial {
				name = nameFromAsset(ca)
				if g.famousPick != nil {
					g.famousPick[ca.Meta.Name] = true
				}
			}
			return g.rigFor(ca), name
		}
	}
	spec := randomGuestSpec(g.rng)
	return newCharSet(&spec), nameFromSpec(&spec)
}

// setupCast chooses the characters for the player, police and guards (assets first, built-ins as fallback).
func (g *Game) setupCast() {
	g.loadOverrides()
	as := g.as
	ps, cs, ms := playerSpec(), copSpec(), medicSpec()
	as.playerCS, as.copCS, as.medicCS = newCharSet(&ps), newCharSet(&cs), newCharSet(&ms)
	as.guardCS, as.specialGuardCS = nil, nil
	{
		u := unitSpecs()
		var alt []*CharSet
		for i := range u.copAlt {
			alt = append(alt, newCharSet(&u.copAlt[i]))
		}
		as.copAlt = alt
		as.swatCS = newCharSet(&u.swat)
		as.fbiCS, as.militaryCS, as.rangerCS = nil, nil, nil
		for i := range u.fbi {
			as.fbiCS = append(as.fbiCS, newCharSet(&u.fbi[i]))
		}
		for i := range u.military {
			as.militaryCS = append(as.militaryCS, newCharSet(&u.military[i]))
		}
		for i := range u.rangers {
			as.rangerCS = append(as.rangerCS, newCharSet(&u.rangers[i]))
		}
	}
	as.killerCS = nil
	for i := 0; i < 4; i++ {
		gs := guardSpec(g.rng)
		as.guardCS = append(as.guardCS, newCharSet(&gs))
	}
	// the protagonist wears what player 1 put on in the wardrobe, unless the user drew him by hand in the editor
	as.playerCS = g.playerCS(0)
	if g.store == nil {
		return
	}
	if l := g.store.Chars["main_character"]; len(l) > 0 && l[0].Meta.Edited {
		as.playerCS = g.rigFor(l[0])
	}
	var cops, medics, fbis, militars, swats, rangers []*CharAsset
	as.fireCS, as.techCS = nil, nil
	for _, ca := range g.store.Chars["police"] {
		switch ca.Meta.Role {
		case "medic":
			medics = append(medics, ca)
		case "firefighter":
			if ca.Meta.Edited {
				as.fireCS = append(as.fireCS, g.rigFor(ca))
			}
		case "technician":
			if ca.Meta.Edited {
				as.techCS = append(as.techCS, g.rigFor(ca))
			}
		case "fbi":
			if ca.Meta.Edited {
				fbis = append(fbis, ca)
			}
		case "military":
			if ca.Meta.Edited {
				militars = append(militars, ca)
			}
		case "swat":
			if ca.Meta.Edited {
				swats = append(swats, ca)
			}
		case "ranger":
			if ca.Meta.Edited {
				rangers = append(rangers, ca)
			}
		default:
			cops = append(cops, ca)
		}
	}
	rigs := func(l []*CharAsset) []*CharSet {
		var out []*CharSet
		for _, ca := range l {
			out = append(out, g.rigFor(ca))
		}
		return out
	}
	if len(fbis) > 0 {
		as.fbiCS = rigs(fbis)
	}
	if len(militars) > 0 {
		as.militaryCS = rigs(militars)
	}
	if len(swats) > 0 {
		as.swatCS = g.rigFor(swats[0])
	}
	if len(rangers) > 0 {
		as.rangerCS = rigs(rangers)
	}
	if len(cops) > 0 {
		as.copCS = g.rigFor(cops[0])
	}
	if len(medics) > 0 {
		as.medicCS = g.rigFor(medics[0])
	}
	if l := g.store.Chars["enemies"]; len(l) > 0 {
		as.guardCS = nil
		for _, ca := range l {
			as.guardCS = append(as.guardCS, g.rigFor(ca))
		}
	}
	as.killerCS = nil
	for _, ca := range g.store.Chars["special_enemies"] {
		if ca.Meta.Role == "killer" {
			as.killerCS = append(as.killerCS, g.rigFor(ca))
		} else {
			as.specialGuardCS = append(as.specialGuardCS, g.rigFor(ca))
		}
	}
}

// ---------------------------------------------------------------------------
// Levels
// ---------------------------------------------------------------------------

// mergeCells turns cells matching a predicate into as few rectangles as possible (cell units).
func mergeCells(rows []string, match func(byte) bool) []Rect {
	var out []Rect
	type run struct{ x0, x1 int }
	prev := map[run]int{} // run -> index in out of the open rectangle
	for y, row := range rows {
		cur := map[run]int{}
		for x := 0; x < len(row); {
			if !match(row[x]) {
				x++
				continue
			}
			x0 := x
			for x < len(row) && match(row[x]) {
				x++
			}
			r := run{x0, x}
			if idx, ok := prev[r]; ok {
				out[idx].H++
				cur[r] = idx
			} else {
				out = append(out, Rect{float64(x0), float64(y), float64(x - x0), 1})
				cur[r] = len(out) - 1
			}
		}
		prev = cur
	}
	return out
}

func cellRectToDesign(r Rect) Rect {
	k := float64(logicCell) / wk
	return Rect{r.X * k, r.Y * k, r.W * k, r.H * k}
}

func levelDefFromAsset(la *LevelAsset) *LevelDef {
	m := la.Meta
	toV := func(p LevelPoint) Vec { return Vec{float64(p.X) / wk, float64(p.Y) / wk} }
	d := &LevelDef{
		Title: m.Title, Intro: m.Intro, W: 960, H: 540, Police: m.Police, Swim: m.Swim, Disco: m.Disco, Custom: true,
		Tint: color.RGBA{m.Tint[0], m.Tint[1], m.Tint[2], m.Tint[3]}, Guests: m.Guests,
		Start: toV(m.Start), CarFrom: toV(m.CarFrom), CarTo: toV(m.CarTo), Strobe: m.Strobe,
		CellSize: math.Max(6, float64(m.DanceCell)/wk),
	}
	if d.Title == "" {
		d.Title = m.Name
	}
	for _, p := range m.Pads {
		d.Pads = append(d.Pads, toV(p))
	}
	if d.Police == "heli" && len(d.Pads) == 0 {
		d.Police = "car"
	}
	for _, r := range m.Guards {
		var route []Vec
		for _, p := range r {
			route = append(route, toV(p))
		}
		if len(route) > 0 {
			d.Guards = append(d.Guards, route)
		}
	}
	for _, p := range m.Performers {
		d.Performers = append(d.Performers, Perf{Pos: Vec{float64(p.X) / wk, float64(p.Y) / wk}, Role: p.Role, Dir: DirDown})
	}
	for _, l := range m.Lights {
		d.Glows = append(d.Glows, Glow{Pos: Vec{float64(l.X) / wk, float64(l.Y) / wk}, Col: color.RGBA{l.Color[0], l.Color[1], l.Color[2], l.Alpha}, R: float64(l.R) / wk, Pulse: l.Pulse})
	}
	for _, an := range m.Animals {
		r := float64(an.Radius) / wk
		if r < 20 {
			r = 60
		}
		c := Vec{float64(an.X) / wk, float64(an.Y) / wk}
		d.Animals = append(d.Animals, AnimalSpawn{Kind: an.Kind, Pos: c, Area: Rect{c.X - r, c.Y - r*0.7, 2 * r, 1.4 * r}})
	}
	for _, p := range m.Props {
		if _, ok := propKinds[p.Kind]; !ok {
			fmt.Println("assets: level", m.Name, "uses unknown object", p.Kind)
			continue
		}
		d.Props = append(d.Props, PropInst{Kind: p.Kind, X: float64(p.X) / wk, Y: float64(p.Y) / wk, Var: p.Var})
	}
	for _, r := range mergeCells(m.Solid, func(c byte) bool { return c == '#' }) {
		d.Walls = append(d.Walls, Wall{R: cellRectToDesign(r), Style: WallPlaster})
	}
	for _, r := range mergeCells(m.Solid, func(c byte) bool { return c == '=' }) {
		d.Walls = append(d.Walls, Wall{R: cellRectToDesign(r), Style: WallFence})
	}
	for _, r := range mergeCells(m.Solid, func(c byte) bool { return c == '~' }) {
		d.Pools = append(d.Pools, cellRectToDesign(r))
	}
	for _, r := range mergeCells(m.Solid, func(c byte) bool { return c == ',' }) {
		d.Slow = append(d.Slow, cellRectToDesign(r))
	}
	for dir, ch := range []byte{'>', '<', '^', 'v'} {
		ch := ch
		for _, r := range mergeCells(m.Solid, func(c byte) bool { return c == ch }) {
			d.Roads = append(d.Roads, RoadSeg{R: cellRectToDesign(r), Dir: dir})
		}
	}
	for _, r := range mergeCells(m.Solid, func(c byte) bool { return c == 'w' }) {
		wr := cellRectToDesign(r)
		d.Windows = append(d.Windows, wr)
		d.Walls = append(d.Walls, Wall{R: wr, Style: WallFence})
	}
	switch m.TimeOfDay {
	case "day":
		d.Day = true
		d.Tint = color.RGBA{0, 0, 0, 0}
	case "dusk":
		d.Day = true
		d.Tint = color.RGBA{70, 24, 8, 70}
	}
	if m.RoadH > 0 {
		d.LaneOff = float64(m.RoadH) / wk / 4
	}
	d.Traffic = m.Traffic
	d.ParkX0, d.ParkX1 = float64(m.ParkX0)/wk, float64(m.ParkX1)/wk
	for _, r := range mergeCells(m.Zone, func(c byte) bool { return c == 'r' || c == 'd' }) {
		d.Roam = append(d.Roam, cellRectToDesign(r))
	}
	for _, r := range mergeCells(m.Zone, func(c byte) bool { return c == 'd' }) {
		dr := cellRectToDesign(r)
		d.Dance = append(d.Dance, dr)
		if m.DanceFloor {
			d.Floors = append(d.Floors, Floor{R: dr, Kind: "dance"})
		}
	}
	if len(d.Roam) == 0 {
		d.Roam = []Rect{{40, 40, 880, 460}}
	}
	bg := ebiten.NewImage(la.BG.w, la.BG.h)
	bg.WritePixels(la.BG.p)
	d.BGImg = bg
	return d
}

// rebuildLevels replaces the level list with the ones on disk (or the built-ins if there are none).
func (g *Game) rebuildLevels() {
	levelOverrides = map[string]*LevelAsset{}
	var custom []*LevelAsset
	if g.store != nil {
		custom, levelOverrides = splitLevels(g.store.Levels)
	}
	if len(custom) > 0 {
		var defs []*LevelDef
		for _, la := range custom {
			defs = append(defs, levelDefFromAsset(la))
		}
		levelDefs = defs
		campaignN = len(defs) // custom background sets replace the campaign wholesale
	} else {
		levelDefs = builtinLevels()
	}
	g.baseN = len(levelDefs)
}

// ---------------------------------------------------------------------------
// Exporting the built-in levels as editable backgrounds (needs a running game to read pixels)
// ---------------------------------------------------------------------------

// renderDefBG paints the background of a scenario definition and returns its pixels.
func renderDefBG(g *Game, def *LevelDef) []byte {
	save := g.lvl
	g.lvl = buildLevel(def)
	g.renderBG()
	bgImg := g.lvl.bg
	g.lvl = save
	pix := make([]byte, 4*screenW*screenH)
	bgImg.ReadPixels(pix)
	return pix
}

func exportLevelDef(g *Game, def *LevelDef, name string) (*LevelAsset, error) {
	pix := renderDefBG(g, def)
	la := levelAssetFromDef(def, name)
	copy(la.BG.p, pix)
	return la, nil
}

// levelAssetFromDef turns a level definition into the editable form: the points, the objects
// and the logic grids. The background picture is left blank (exportLevelDef paints it).
func levelAssetFromDef(def *LevelDef, name string) *LevelAsset {
	la := newLevelAsset(name)
	la.BG = newPix(screenW, screenH)
	m := &la.Meta
	m.Title, m.Intro, m.Police, m.Guests, m.Swim, m.Disco = def.Title, def.Intro, def.Police, def.Guests, def.Swim, def.Disco
	m.Tint = [4]uint8{def.Tint.R, def.Tint.G, def.Tint.B, def.Tint.A}
	m.Strobe = def.Strobe
	toP := func(v Vec) LevelPoint { return LevelPoint{int(math.Round(v.X * wk)), int(math.Round(v.Y * wk))} }
	m.Start, m.CarFrom, m.CarTo = toP(def.Start), toP(def.CarFrom), toP(def.CarTo)
	m.Pads = nil
	for _, p := range def.Pads {
		m.Pads = append(m.Pads, toP(p))
	}
	m.Guards = nil
	for _, r := range def.Guards {
		var route []LevelPoint
		for _, p := range r {
			route = append(route, toP(p))
		}
		m.Guards = append(m.Guards, route)
	}
	for _, p := range def.Performers {
		q := toP(p.Pos)
		m.Performers = append(m.Performers, LevelPerf{X: q.X, Y: q.Y, Role: p.Role})
	}
	for _, gl := range def.Glows {
		q := toP(gl.Pos)
		m.Lights = append(m.Lights, LevelLight{X: q.X, Y: q.Y, R: int(gl.R * wk), Color: [3]uint8{gl.Col.R, gl.Col.G, gl.Col.B}, Alpha: gl.Col.A, Pulse: gl.Pulse})
	}
	for _, an := range def.Animals {
		q := toP(an.Pos)
		r := int(math.Max(an.Area.W, an.Area.H) / 2 * wk)
		m.Animals = append(m.Animals, LevelAnimal{Kind: an.Kind, X: q.X, Y: q.Y, Radius: r})
	}
	for _, p := range def.Props {
		q := toP(Vec{p.X, p.Y})
		m.Props = append(m.Props, LevelProp{Kind: p.Kind, X: q.X, Y: q.Y, Var: p.Var})
	}
	for _, f := range def.Floors {
		m.Floors = append(m.Floors, LevelFloor{X: int(math.Round(f.R.X * wk)), Y: int(math.Round(f.R.Y * wk)), W: int(math.Round(f.R.W * wk)), H: int(math.Round(f.R.H * wk)), Kind: f.Kind, Color: f.Color})
	}
	for _, p := range def.Pools {
		m.Pools = append(m.Pools, LevelRect{X: int(math.Round(p.X * wk)), Y: int(math.Round(p.Y * wk)), W: int(math.Round(p.W * wk)), H: int(math.Round(p.H * wk))})
	}
	var danceRects []Rect
	for _, f := range def.Floors {
		if f.Kind == "dance" {
			danceRects = append(danceRects, f.R)
			m.DanceFloor = true
			m.DanceCell = int(def.CellSize * wk)
		}
	}
	if m.DanceCell == 0 {
		m.DanceCell = 10
	}
	cw, ch := screenW/logicCell, screenH/logicCell
	solid, zone := make([][]byte, ch), make([][]byte, ch)
	k := float64(logicCell) / wk
	for cy := 0; cy < ch; cy++ {
		solid[cy], zone[cy] = make([]byte, cw), make([]byte, cw)
		for cx := 0; cx < cw; cx++ {
			cell := Rect{float64(cx) * k, float64(cy) * k, k, k}
			cxm, cym := cell.X+k/2, cell.Y+k/2
			solid[cy][cx], zone[cy][cx] = '.', '.'
			for _, w := range def.Walls {
				ov := overlapArea(cell, w.R)
				if ov >= 0.3*k*k {
					if w.Style == WallFence {
						if solid[cy][cx] == '.' {
							solid[cy][cx] = '='
						}
					} else {
						solid[cy][cx] = '#'
					}
				}
			}
			for _, p := range def.Pools {
				if p.Has(cxm, cym) && solid[cy][cx] == '.' {
					solid[cy][cx] = '~'
				}
			}
			if solid[cy][cx] == '.' || solid[cy][cx] == '~' {
				for _, r := range def.Roam {
					if r.Has(cxm, cym) {
						zone[cy][cx] = 'r'
					}
				}
				for _, r := range danceRects {
					if r.Has(cxm, cym) {
						zone[cy][cx] = 'd'
					}
				}
				if len(danceRects) == 0 {
					for _, r := range def.Dance {
						if r.Has(cxm, cym) {
							zone[cy][cx] = 'd'
						}
					}
				}
			}
		}
	}
	m.Solid, m.Zone = make([]string, ch), make([]string, ch)
	for y := 0; y < ch; y++ {
		m.Solid[y], m.Zone[y] = string(solid[y]), string(zone[y])
	}
	return la
}

func overlapArea(a, b Rect) float64 {
	w := math.Min(a.X+a.W, b.X+b.W) - math.Max(a.X, b.X)
	h := math.Min(a.Y+a.H, b.Y+b.H) - math.Max(a.Y, b.Y)
	if w <= 0 || h <= 0 {
		return 0
	}
	return w * h
}

func (g *Game) exportBuiltinLevels(root string) {
	for i, def := range builtinLevels() {
		name := fmt.Sprintf("%02d_%s", i+1, safeName(def.Title))
		la, err := exportLevelDef(g, def, name)
		if err != nil {
			fmt.Println("export:", err)
			continue
		}
		if err := saveLevelAsset(root, la); err != nil {
			fmt.Println("export:", err)
		}
	}
}

func removeAll(path string) { os.RemoveAll(path) }

var _ = filepath.Join
