package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ---------------------------------------------------------------------------
// MINIMAP (ideas 57 and 61), in the style of the map of Pro Evolution Soccer 6: a small see-through panel in the
// bottom right corner with the level to scale, the walls and solid objects in WHITE and every character as one
// pixel (two for the players) of a colour per kind:
//
//	player 1 blue (blinks), player 2 red, guests light grey, guards green, police dark blue / red flashing,
//	VIP gold, security cameras yellow (with a faint cone), armed traps orange, special killers purple.
//
// OFF by default (so the base look does not change); TAB or the □ button of a gamepad switch it for the rest of the
// session (a variable in memory: nothing is written to progress.json). The flag -minimap starts it on (debug, the
// progress stays frozen like with every flag). The help window ("?") lists the key.
//
// COST: the walls are drawn once per level into a pixel buffer (base); each frame copies it, sets the dots and uploads
// the whole 150x84 picture with a single WritePixels: no per-dot draw calls, no allocations.
//   - STEALTH: only the walls and the players are shown (the guards would be an unfair radar).
//   - BLACKOUT: the picture flickers and carries noise, and the characters blink out.
// ---------------------------------------------------------------------------

const (
	mmW, mmH = 150, 84
	mmMargin = 6.0
)

// minimapOn: the session preference (TAB / □). The flag -minimap sets it.
var minimapOn = false

// hardware access for the tests
var (
	mmKeyFn = inpututil.IsKeyJustPressed
	mmPadFn = func() bool {
		if !minimapPadFree() {
			return false
		}
		for _, id := range padList() {
			if padJustFn(id, minimapPadButton) {
				return true
			}
		}
		return false
	}
)

// minimapPadButton: □ switches the minimap on a pad (START is the pause since 2026-11-11, and with START the
// pause window opened first anyway). It only works while no action of either player is bound to it.
const minimapPadButton = ebiten.StandardGamepadButtonRightLeft

func minimapPadFree() bool { return !padButtonUsed(minimapPadButton) }

// Colours of the dots (RGBA premultiplied, ready to copy to the buffer).
var (
	mmBG     = [4]uint8{4, 3, 8, 115} // dark, alpha 0.45
	mmEdge   = [4]uint8{70, 60, 120, 200}
	mmWall   = [4]uint8{235, 235, 235, 235} // white
	mmP1     = [4]uint8{60, 150, 255, 255}
	mmP1b    = [4]uint8{200, 225, 255, 255} // the blink of player 1
	mmP2     = [4]uint8{255, 70, 70, 255}
	mmLastA  = [4]uint8{255, 214, 64, 255} // last guests blink gold / white
	mmLastB  = [4]uint8{255, 250, 220, 255}
	mmGuest  = [4]uint8{190, 190, 200, 255}
	mmGuard  = [4]uint8{50, 210, 90, 255}
	mmCopA   = [4]uint8{40, 70, 220, 255}
	mmCopB   = [4]uint8{255, 50, 50, 255}
	mmVIP    = [4]uint8{255, 205, 40, 255}
	mmCam    = [4]uint8{255, 235, 60, 255}
	mmCone   = [4]uint8{110, 100, 25, 110} // premultiplied: a faint yellow
	mmTrap   = [4]uint8{255, 130, 30, 255}
	mmKiller = [4]uint8{190, 80, 255, 255}
	mmNoise  = [4]uint8{100, 100, 105, 120}
)

type miniMap struct {
	lvl       *Level
	nsol      int
	sc        float64 // world -> minimap pixels
	ox, oy    float64 // where the level starts inside the picture
	base      []uint8 // walls and background
	buf       []uint8 // the picture of this frame
	img       *ebiten.Image
	dots      int // dots set in the last frame (tests)
	built     int // how many times the base was rebuilt (tests: it must be once per level)
	drawn     bool
	noiseSeed uint32
}

func (m *miniMap) reset() { m.lvl = nil }

func mmPut(buf []uint8, x, y int, c [4]uint8) {
	if x < 0 || y < 0 || x >= mmW || y >= mmH {
		return
	}
	i := (y*mmW + x) * 4
	buf[i], buf[i+1], buf[i+2], buf[i+3] = c[0], c[1], c[2], c[3]
}

// build draws the background and the walls of the level into base (once per level).
func (m *miniMap) build(l *Level) {
	m.lvl, m.nsol = l, len(l.solidsP)
	if m.base == nil {
		m.base = make([]uint8, mmW*mmH*4)
		m.buf = make([]uint8, mmW*mmH*4)
	}
	m.built++
	for i := 0; i < mmW*mmH; i++ {
		c := mmBG
		x, y := i%mmW, i/mmW
		if x == 0 || y == 0 || x == mmW-1 || y == mmH-1 {
			c = mmEdge
		}
		m.base[i*4], m.base[i*4+1], m.base[i*4+2], m.base[i*4+3] = c[0], c[1], c[2], c[3]
	}
	w, h := math.Max(l.W, 1), math.Max(l.H, 1)
	m.sc = math.Min(float64(mmW-4)/w, float64(mmH-4)/h)
	m.ox = math.Round((float64(mmW) - w*m.sc) / 2)
	m.oy = math.Round((float64(mmH) - h*m.sc) / 2)
	for _, r := range l.solidsP {
		x0 := int(math.Floor(m.ox + math.Max(r.X, 0)*m.sc))
		y0 := int(math.Floor(m.oy + math.Max(r.Y, 0)*m.sc))
		x1 := int(math.Ceil(m.ox + math.Min(r.X+r.W, w)*m.sc))
		y1 := int(math.Ceil(m.oy + math.Min(r.Y+r.H, h)*m.sc))
		if r.W <= 0 || r.H <= 0 || r.X+r.W <= 0 || r.Y+r.H <= 0 || r.X >= w || r.Y >= h {
			continue
		}
		if x1 <= x0 {
			x1 = x0 + 1
		}
		if y1 <= y0 {
			y1 = y0 + 1
		}
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				if x > 0 && y > 0 && x < mmW-1 && y < mmH-1 {
					mmPut(m.base, x, y, mmWall)
				}
			}
		}
	}
	if m.img == nil {
		m.img = ebiten.NewImage(mmW, mmH)
	}
}

// pos converts a world position to picture pixels.
func (m *miniMap) pos(v Vec) (int, int) {
	return int(m.ox + v.X*m.sc), int(m.oy + v.Y*m.sc)
}

func (m *miniMap) dot(v Vec, c [4]uint8, size int) {
	x, y := m.pos(v)
	for dy := 0; dy < size; dy++ {
		for dx := 0; dx < size; dx++ {
			mmPut(m.buf, x+dx, y+dy, c)
		}
	}
	m.dots++
}

// frame builds the picture of this frame in buf (no drawing yet): the tests read it.
func (g *Game) minimapFrame() {
	m := &g.mm
	l := g.lvl
	if m.lvl != l || m.nsol != len(l.solidsP) {
		m.build(l)
	}
	copy(m.buf, m.base)
	m.dots = 0
	dark := g.blackoutOn()
	show := !dark || int(g.time*8)%3 != 0 // blackout: the characters blink out
	if dark {
		m.noiseSeed = m.noiseSeed*1664525 + 1013904223 + uint32(g.tick)
		s := m.noiseSeed
		for i := 0; i < 14; i++ {
			s = s*1664525 + 1013904223
			mmPut(m.buf, int(s>>8)%mmW, int(s>>16)%mmH, mmNoise)
		}
	}
	// armed traps and cameras first, characters over them
	if !g.stealth && show {
		for _, pr := range l.Props {
			if pr != nil && !pr.gone && trapProp(pr) && pr.armed {
				m.dot(Vec{pr.X, pr.Y}, mmTrap, 1)
			}
		}
		for _, c := range g.cams {
			for _, i := range [...]int{0, camRays / 2, camRays - 1} {
				r := c.rays[i]
				if r <= 0 {
					continue
				}
				a := c.ang - camHalf + 2*camHalf*float64(i)/float64(camRays-1)
				dx, dy := math.Cos(a), math.Sin(a)
				for d := 10.0; d < r; d += 9 {
					x, y := m.pos(Vec{c.eye.X + dx*d, c.eye.Y + dy*d})
					mmPut(m.buf, x, y, mmCone)
				}
			}
			m.dot(c.eye, mmCam, 2)
		}
		for _, gu := range g.guests {
			if !gu.out() && gu.py == nil {
				m.dot(gu.pos, mmGuest, 1)
			}
		}
		for _, gd := range g.guards {
			if gd.dead {
				continue
			}
			if gd.vip == vipBoss || gd.vip == vipInner {
				m.dot(gd.pos, mmVIP, 2)
			} else {
				m.dot(gd.pos, mmGuard, 1)
			}
		}
		for _, p := range g.lastTargets() { // the last guests: 2x2, blinking gold / white
			c := mmLastA
			if int(g.time*6)%2 == 0 {
				c = mmLastB
			}
			m.dot(p, c, 2)
		}
		flash := int(g.time*4)%2 == 0
		for _, cp := range g.cops {
			if cp.medic || cp.state == cLeave {
				continue
			}
			c := mmCopA
			if flash {
				c = mmCopB
			}
			m.dot(cp.pos, c, 1)
		}
		for _, s := range g.allSlashers() {
			if s.state != slGone {
				m.dot(s.pos, mmKiller, 1)
			}
		}
	}
	// the players last, 2 px; player 1 blinks between two blues
	if p := g.first(); p != nil && !p.dead && !p.hidden {
		c := mmP1
		if int(g.time*3)%2 == 0 {
			c = mmP1b
		}
		m.dot(p.pos, c, 2)
	}
	if p := g.p2; p != nil && !p.dead && !p.hidden {
		m.dot(p.pos, mmP2, 2)
	}
}

// minimapShown: the panel exists now.
func (g *Game) minimapShown() bool {
	return minimapOn && g.mode == ModePlay && g.lvl != nil && !g.helpOv && !g.dc.has(dmNoMap)
}

// updateMinimapInput: TAB or □ switch the minimap (session only).
func (g *Game) updateMinimapInput() {
	if mmKeyFn(ebiten.KeyTab) || mmPadFn() {
		minimapOn = !minimapOn
	}
}

// drawMinimap paints the panel in the bottom right corner (called from drawHUD, ModePlay only).
func (g *Game) drawMinimap(dst *ebiten.Image) {
	g.mm.drawn = false
	if !g.minimapShown() {
		return
	}
	g.minimapFrame()
	m := &g.mm
	m.img.WritePixels(m.buf)
	op := &mmOp
	op.GeoM.Reset()
	op.GeoM.Translate(screenW-mmW-mmMargin, screenH-mmH-mmMargin)
	if g.blackoutOn() && int(g.time*10)%4 == 0 {
		op.GeoM.Translate(1, 0) // a tremor of one pixel
	}
	dst.DrawImage(m.img, op)
	m.drawn = true
}

var mmOp ebiten.DrawImageOptions

// mmRect: where the panel is on the screen.
func mmRect() (x, y, w, h float64) {
	return screenW - mmW - mmMargin, screenH - mmH - mmMargin, mmW, mmH
}
