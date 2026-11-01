package main

import (
	"image/color"
	"math"
	"math/rand"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Bands: now and then a famous group plays in the party, on the stage of the
// level when it has one or in an open spot with its own dance floor.
// ---------------------------------------------------------------------------

type bandMember struct {
	slug, role string
	dx, dy     float64
}

type Band struct {
	slug, name string
	members    []bandMember
}

func row(slugs ...string) []bandMember {
	roles := []string{"sing", "guitar", "guitar", "drum"}
	pos := [][2]float64{{0, 0}, {-34, 0}, {34, 0}, {0, -24}}
	var out []bandMember
	for i, s := range slugs {
		out = append(out, bandMember{s, roles[i], pos[i][0], pos[i][1]})
	}
	return out
}

var bandList = []Band{
	{"beatles", "The Beatles", []bandMember{{"lennon", "guitar", -34, 0}, {"mccartney", "sing", 0, 0}, {"harrison", "guitar", 34, 0}, {"ringo", "drum", 0, -24}}},
	{"sum41", "Sum 41", row("deryck", "dave_baksh", "jason_mc", "frank_zummo")},
	{"blink182", "Blink-182", row("mark_hoppus", "tom_delonge", "travis_barker")},
	{"simpleplan", "Simple Plan", row("sp_cantante", "sp_guitarra", "sp_bajo", "sp_bateria")},
	{"offspring", "The Offspring", row("off_cantante", "off_guitarra", "off_bajo", "off_bateria")},
	{"metallica", "Metallica", row("met_cantante", "met_guitarra", "met_bajo", "met_bateria")},
	{"punksailor", "Punk Sailor", row("ps_cantante", "ps_guitarra", "ps_bajo", "ps_bateria")},
	{"billytalent", "Billy Talent", row("bt_cantante", "bt_guitarra", "bt_bajo", "bt_bateria")},
	{"lostprophets", "Lost Prophets", row("lp_cantante", "lp_guitarra", "lp_bajo", "lp_bateria")},
	{"despistaos", "Despistaos", row("des_cantante", "des_guitarra", "des_bajo", "des_bateria")},
	{"elvis", "Elvis", []bandMember{{"elvis", "sing", 0, 0}}},
}

func init() {
	mk := func(prefix, band string, fem bool, opts [4][]po) {
		for i, role := range []string{"cantante", "guitarra", "bajo", "bateria"} {
			slug := prefix + "_" + role
			starPresets = append(starPresets, def(slug, opts[i]...))
			displayNameOverrides[slug] = band + " · " + role
		}
	}
	mk("sp", "Simple Plan", false, [4][]po{
		{hairc(0x3a2a1a, HairMessy), tops(TopTee, 0xf4f4f4, 0x14141a), pantsc(0x2a3a5a), shoesc(0xf4f4f4)},
		{hatc(HatCap, 0x14141a), tops(TopHoodie, 0xc02a2a, 0xc02a2a), pantsc(0x2a2a30), hairc(0x14101a, HairShort)},
		{hairc(0xe8c860, HairSpiky), tops(TopTank, 0x14141a, 0x14141a), pantsc(0x4a4a56), suspc(0xd02828)},
		{hairc(0x2a1a14, HairShort), tops(TopTee, 0x2a60d0, 0x2a60d0), pantsc(0x14141a), glass(GlassRound)},
	})
	mk("off", "The Offspring", false, [4][]po{
		{hairc(0xe8c860, HairDreads), tops(TopTee, 0x14141a, 0x14141a), pantsc(0x3a4a6a), shoesc(0x14141a)},
		{hairc(0x2a1a14, HairLong), beardc(BeardFull), tops(TopTank, 0x8a2a2a, 0x8a2a2a), pantsc(0x14141a)},
		{hairc(0x14101a, HairShort), glass(GlassRound), tops(TopTee, 0xf4f4f4, 0x14141a), patn(2), pantsc(0x4a4a56)},
		{hairc(0x4a3018, HairMessy), tops(TopTank, 0x2a8a4a, 0x2a8a4a), pantsc(0x14141a), bottomc(BotShorts, 0x14141a)},
	})
	mk("met", "Metallica", false, [4][]po{
		{hairc(0x14101a, HairShort), beardc(BeardGoatee), tops(TopTee, 0x14141a, 0x14141a), pantsc(0x14141a), shoesc(0x14141a), skinc(0xe0b090)},
		{hairc(0x14101a, HairLong), tops(TopTank, 0x14141a, 0x14141a), pantsc(0x2a2a30), shoesc(0x14141a)},
		{hairc(0x2a1a14, HairLong), beardc(BeardFull), tops(TopTank, 0x14141a, 0x8a8a94), pantsc(0x14141a), bodyc(BodyBig)},
		{hairc(0x4a3a2a, HairShort), tops(TopTee, 0x14141a, 0x14141a), pantsc(0x3a4a6a), shoesc(0x14141a)},
	})
	mk("ps", "Punk Sailor", false, [4][]po{
		{hatc(HatCap, 0xf4f4f4), hairc(0xe03a8a, HairMohawk), tops(TopTee, 0xf4f4f4, 0x2a4ab8), patn(2), pantsc(0xf4f4f4)},
		{hairc(0x3ad0a8, HairMohawk), hatc(HatCap, 0xf4f4f4), tops(TopTee, 0xf4f4f4, 0x2a4ab8), patn(2), bottomc(BotShorts, 0x2a4ab8)},
		{hairc(0xf0c030, HairSpiky), tops(TopTank, 0xf4f4f4, 0xf4f4f4), pantsc(0x2a4ab8), suspc(0xd02828)},
		{hatc(HatCap, 0x2a4ab8), hairc(0x14101a, HairShort), tops(TopTee, 0xf4f4f4, 0x2a4ab8), patn(2), pantsc(0xf4f4f4)},
	})
	mk("bt", "Billy Talent", false, [4][]po{
		{hairc(0x14101a, HairCurly), tops(TopSuit, 0x14141a, 0xf4f4f4), pantsc(0x14141a), accentc(0xc82a2a)},
		{hairc(0x14101a, HairSpiky), tops(TopLong, 0x6a2a2a, 0x6a2a2a), pantsc(0x2a2a30)},
		{hatc(HatBeanie, 0x14141a), tops(TopTee, 0xe8e8f0, 0xe8e8f0), pantsc(0x3a4a6a), beardc(BeardStubble)},
		{hairc(0x4a3018, HairComb), tops(TopTank, 0x14141a, 0x14141a), pantsc(0x14141a)},
	})
	mk("lp", "Lost Prophets", false, [4][]po{
		{hairs(HairBald), beardc(BeardGoatee), tops(TopHoodie, 0x14141a, 0x14141a), pantsc(0x3a4a6a)},
		{hairc(0x14101a, HairMessy), tops(TopHoodie, 0xc02a2a, 0xc02a2a), pantsc(0x14141a)},
		{hairc(0xe8c860, HairSpiky), tops(TopTee, 0x2a4ab8, 0x2a4ab8), pantsc(0x4a4a56)},
		{hatc(HatCap, 0xf4f4f4), tops(TopTank, 0x14141a, 0x14141a), pantsc(0x14141a), bottomc(BotShorts, 0x14141a)},
	})
	mk("des", "Despistaos", false, [4][]po{
		{hairc(0x3a2414, HairMessy), beardc(BeardStubble), tops(TopLong, 0x8a6a4a, 0x8a6a4a), pantsc(0x3a4a6a)},
		{hairc(0x14101a, HairShort), tops(TopTee, 0xe8e8f0, 0x2a8a8a), patn(2), pantsc(0x2a2a30)},
		{hairc(0x5a3a22, HairCurly), tops(TopTank, 0x2a2a30, 0x2a2a30), pantsc(0x6a5a3a)},
		{hatc(HatBeanie, 0x6a6a74), tops(TopTee, 0xc8a840, 0xc8a840), pantsc(0x2a2a30), beardc(BeardMustache)},
	})
}

// planBand decides whether a band plays in this party and where. It returns a
// modified copy of the level (a dance floor next to the band) when it needs to.
func (g *Game) planBand(def *LevelDef) *LevelDef {
	g.band, g.bandStage, g.bandSpot, g.bandUsed = nil, false, Vec{}, nil
	if g.rng.Float64() > 0.7 && g.forceBand == 0 {
		return nil
	}
	b := bandList[g.rng.Intn(len(bandList))]
	if g.forceBand > 0 {
		b = bandList[(g.forceBand-1)%len(bandList)]
	}
	for _, bd := range bandList { // a band with bought members is called first
		for _, m := range bd.members {
			for _, cslug := range castSlugs {
				if cslug == m.slug && g.rng.Float64() < 0.7 {
					b = bd
				}
			}
		}
	}
	stage := false
	for _, pf := range def.Performers {
		if pf.Role != "dj" {
			stage = true
		}
	}
	if stage {
		g.band, g.bandStage = &b, true
		return nil
	}
	var spot Vec
	ok := false
	bw := 200.0
	for _, sz := range [][2]float64{{200, 130}, {160, 110}, {120, 96}} {
		if spot, ok = g.openSpot(sz[0], sz[1]); ok {
			bw = sz[0]
			break
		}
	}
	if !ok {
		return nil
	}
	g.band, g.bandSpot = &b, spot
	d2 := *def
	d2.Floors = append(append([]Floor(nil), def.Floors...),
		Floor{R: Rect{spot.X - bw/2 - 6, spot.Y - 44, bw + 12, 46}, Kind: "boardwalk"},
		Floor{R: Rect{spot.X - bw/2, spot.Y + 8, bw, 74}, Kind: "dance"})
	d2.Dance = append(append([]Rect(nil), def.Dance...), Rect{spot.X - bw/2 + 8, spot.Y + 12, bw - 16, 66})
	d2.Glows = append(append([]Glow(nil), def.Glows...), Glow{Vec{spot.X, spot.Y + 40}, partyGlow(g.rng), 120, true})
	return &d2
}

func partyGlow(r *rand.Rand) color.RGBA { return color.RGBA{120, 40, 170, 150} }

// openSpot looks for a free rectangle (w x h, centred at the band row) away
// from the start, the pools and the existing dance floors.
func (g *Game) openSpot(w, h float64) (Vec, bool) {
	l := g.lvl
	var cands []Vec
	for cy := 70.0; cy < l.H-h; cy += 24 {
		for cx := w/2 + 20; cx < l.W-w/2-20; cx += 24 {
			ok := true
			for y := cy - 46; y <= cy+h-40 && ok; y += 12 {
				for x := cx - w/2; x <= cx+w/2; x += 12 {
					if !l.Free(Vec{x, y}) || l.InPool(Vec{x, y}) {
						ok = false
						break
					}
				}
			}
			if !ok || dist(Vec{cx, cy}, l.Def.Start) < 120 {
				continue
			}
			// the boardwalk and the dance floor must not sit on top of any wall (thin walls slip between samples)
			area := Rect{cx - w/2 - 8, cy - 46, w + 16, h + 46}
			for _, wl := range l.Walls {
				if wl.Style != WallVoid && rectsOverlap(area, wl.R) {
					ok = false
					break
				}
			}
			if !ok {
				continue
			}
			if !ok || !g.spotClear(area) {
				continue
			}
			for _, d := range l.Def.Dance {
				if dist(d.Center(), Vec{cx, cy + 40}) < 150 {
					ok = false
				}
			}
			for _, pf := range l.Def.Performers {
				if dist(pf.Pos, Vec{cx, cy}) < 130 {
					ok = false
				}
			}
			if ok {
				cands = append(cands, Vec{cx, cy})
			}
		}
	}
	if len(cands) == 0 {
		return Vec{}, false
	}
	return cands[g.rng.Intn(len(cands))], true
}

// stageMember: the band member who plays this role on the stage of the level.
func (g *Game) stageMember(role string) *bandMember {
	if g.band == nil || !g.bandStage {
		return nil
	}
	for i := range g.band.members {
		if len(g.bandUsed) < len(g.band.members) {
			g.bandUsed = make([]bool, len(g.band.members))
		}
		if m := &g.band.members[i]; m.role == role && !g.bandUsed[i] {
			g.bandUsed[i] = true
			return m
		}
	}
	return &g.band.members[g.rng.Intn(len(g.band.members))]
}

// spawnBand puts the band on its spot (levels without a stage).
func (g *Game) spawnBand() {
	if g.band == nil || g.bandStage {
		return
	}
	for _, m := range g.band.members {
		f := presetByName(m.slug)
		if f == nil {
			continue
		}
		spec := f(g.rng)
		pos := g.lvl.NearestFree(Vec{g.bandSpot.X + m.dx, g.bandSpot.Y + m.dy - 14})
		g.guests = append(g.guests, &Guest{pos: pos, dir: DirDown, cs: newCharSet(&spec), name: nameFromSpec(&spec), perf: m.role, state: gPerform, role: g.rng.Intn(2), keep: true})
	}
	g.say(g.band.name + " toca en la fiesta")
}

// bandSlug is the folder-less key for mp3 files of the band in "bands/".
func bandFileKey(name string) string {
	name = strings.ToLower(name)
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// drawBandSign hangs a banner with the name of the band over the musicians.
func (g *Game) drawBandSign(dst *ebiten.Image) {
	if g.band == nil || g.mode != ModePlay {
		return
	}
	var sx, n float64
	minY := 1e9
	for _, gu := range g.guests {
		if gu.perf != "" && gu.perf != "dj" && !gu.out() {
			sx += gu.pos.X
			n++
			minY = math.Min(minY, gu.pos.Y)
		}
	}
	if n == 0 {
		return
	}
	cx, by := g.wx(sx/n), g.wy(minY)-58
	w := g.textW(g.band.name, 1.5) + 22
	fillRect(dst, cx-w/2-2, by-24, w+4, 26, rgb(0x14101e))
	fillRect(dst, cx-w/2, by-22, w, 22, rgb(0x2a1a4a))
	fillRect(dst, cx-w/2, by-22, w, 2, rgb(0xff3070))
	fillRect(dst, cx-w/2, by-2, w, 2, rgb(0x60d0f0))
	fillRect(dst, cx-w/2+4, by, 2, 14, rgb(0x6a6a76)) // the poles holding it
	fillRect(dst, cx+w/2-6, by, 2, 14, rgb(0x6a6a76))
	g.gradText(dst, g.band.name, cx, by-19, rgb(0xff3070), rgb(0x60d0f0), rgb(0x14101e), 1.5)
}

func rectsOverlap(a, b Rect) bool {
	return a.X < b.X+b.W && b.X < a.X+a.W && a.Y < b.Y+b.H && b.Y < a.Y+a.H
}

// spotBadFloors: ground a band or dance floor must never be built on.
var spotBadFloors = map[string]bool{"asphalt": true, "dash": true, "helipad": true, "pitch": true, "arena": true,
	"water": true, "rail": true, "rails": true, "edge": true, "track": true, "lava": true, "ice": true}

// spotClear: the band area must not cover roads, helipads, pools, doorways,
// props (solid or not), the starting point, the car park or anything the
// game needs to keep reachable.
func (g *Game) spotClear(area Rect) bool {
	l := g.lvl
	def := l.Def
	for _, f := range def.Floors {
		if spotBadFloors[f.Kind] && rectsOverlap(area, f.R) {
			return false
		}
	}
	for _, pl := range def.Pools {
		if rectsOverlap(area, pl.Grow(10)) {
			return false
		}
	}
	for _, pd := range def.Pads {
		if rectsOverlap(area, Rect{pd.X - 56, pd.Y - 56, 112, 112}) {
			return false
		}
	}
	for _, d := range def.Doors {
		if rectsOverlap(area, d.R.Grow(22)) {
			return false
		}
	}
	for _, w := range def.Windows {
		if rectsOverlap(area, w.Grow(12)) {
			return false
		}
	}
	for _, sl := range def.Slow {
		if rectsOverlap(area, sl) {
			return false
		}
	}
	for _, d := range def.Dance {
		if rectsOverlap(area, d.Grow(14)) {
			return false
		}
	}
	for _, pr := range l.Props {
		if pr.gone || pr.K == nil {
			continue
		}
		// the whole picture of the prop, not just its foot
		w, h := float64(pr.K.W), float64(pr.K.H)
		if rectsOverlap(area, Rect{pr.X - w/2 - 4, pr.Y - h - 4, w + 8, h + 8}) {
			return false
		}
	}
	for _, pf := range def.Performers {
		if rectsOverlap(area, Rect{pf.Pos.X - 40, pf.Pos.Y - 40, 80, 80}) {
			return false
		}
	}
	if def.Police == "car" && def.Transit == "" {
		// keep the whole street and the vehicle parking strip free
		if rectsOverlap(area, Rect{0, def.CarFrom.Y - 50, l.W, 100}) && def.CarFrom.Y > 0 && def.CarFrom.Y < l.H {
			return false
		}
	}
	return true
}
