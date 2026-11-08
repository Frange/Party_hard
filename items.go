package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Things to use: drink bowls (poisonable), poison bottles and disguise bags to
// pick up, chairs to sit on, three more traps (gas, fireworks, falling piano)
// and the markers that show what can be used. Also the floating point texts.
// ---------------------------------------------------------------------------

func init() {
	pk := propKinds
	pk["bowl"] = &PropKind{W: 30, H: 20, FW: 26, FH: 8, Poison: true, UseKind: "drink", Use: []Vec{{0, 12}}, Vars: 2, Draw: func(b *pixBuf, v int) {
		liq := c([]uint32{0xe03a60, 0x40c060}[v%2])
		b.ellipse(15, 18, 13, 2.5, color.RGBA{0, 0, 0, 80})
		b.ellipse(15, 12, 13, 7, c(0xc8ccd8))
		b.ellipse(15, 11, 11, 5.4, c(0x8a8e9c))
		b.ellipse(15, 11, 10, 4.6, liq)
		b.ellipse(12, 10, 3, 1.4, shade(liq, 1.3))
		b.rect(2, 12, 26, 1, c(0xe8ecf4))
	}}
	pk["chair"] = &PropKind{W: 18, H: 26, FW: 14, FH: 6, UseKind: "lounge", Use: []Vec{{0, 12}}, Vars: 2, Draw: func(b *pixBuf, v int) {
		wood := c([]uint32{0x8a5a30, 0x5a3a24}[v%2])
		b.ellipse(9, 24, 7, 1.8, color.RGBA{0, 0, 0, 70})
		b.rect(3, 2, 12, 11, shade(wood, 0.9)) // backrest
		b.rect(3, 2, 12, 1, shade(wood, 1.3))
		b.rect(2, 13, 14, 4, wood) // seat
		b.rect(2, 13, 14, 1, shade(wood, 1.3))
		b.rect(3, 17, 2, 7, shade(wood, 0.7))
		b.rect(13, 17, 2, 7, shade(wood, 0.7))
	}}
	pk["poisonbottle"] = &PropKind{W: 12, H: 22, FW: 8, FH: 4, NoSolid: true, Item: "poison", Draw: func(b *pixBuf, v int) {
		b.ellipse(6, 20, 5, 1.5, color.RGBA{0, 0, 0, 70})
		b.rect(2, 8, 8, 12, c(0x2aa840))
		b.rect(2, 8, 8, 1, c(0x70e088))
		b.rect(4, 3, 4, 6, c(0x2aa840))
		b.rect(4, 1, 4, 3, c(0x6a4a2a))  // cork
		b.rect(3, 12, 6, 5, c(0xf4f4e8)) // label
		b.set(5, 13, c(0x14140e))
		b.set(7, 13, c(0x14140e))
		b.rect(5, 15, 2, 1, c(0x14140e))
	}}
	pk["disguisebag"] = &PropKind{W: 24, H: 26, FW: 18, FH: 6, NoSolid: true, Item: "disguise", Draw: func(b *pixBuf, v int) {
		b.ellipse(12, 24, 9, 2, color.RGBA{0, 0, 0, 70})
		b.rect(3, 8, 18, 15, c(0x8a3ac8))
		b.rect(3, 8, 18, 2, c(0xb070f0))
		b.rect(8, 3, 8, 6, c(0x8a3ac8)) // handles
		b.rect(10, 5, 4, 3, c(0x1a1a22))
		b.rect(9, 13, 6, 6, c(0xf4f4e8)) // a "?"
		b.rect(10, 13, 4, 1, c(0x2a2a30))
		b.rect(13, 14, 1, 2, c(0x2a2a30))
		b.rect(11, 16, 2, 1, c(0x2a2a30))
		b.set(11, 18, c(0x2a2a30))
	}}
	pk["gastank"] = &PropKind{W: 22, H: 34, FW: 18, FH: 8, Trap: "gas", Draw: func(b *pixBuf, v int) {
		b.ellipse(11, 32, 9, 2, color.RGBA{0, 0, 0, 80})
		b.rect(3, 8, 16, 24, c(0x9aa0a8))
		b.ellipse(11, 8, 8, 4, c(0xb8bec6))
		b.rect(3, 8, 3, 24, c(0xc8ced6))
		b.rect(8, 2, 6, 6, c(0x4a4a54))
		b.rect(3, 18, 16, 6, c(0xe8c020)) // hazard band
		for x := 4; x < 18; x += 4 {
			b.rect(x, 18, 2, 6, c(0x2a2a30))
		}
		b.rect(9, 12, 4, 4, c(0x3aa860))
	}}
	pk["fireworksbox"] = &PropKind{W: 28, H: 22, FW: 24, FH: 8, Trap: "fireworks", Draw: func(b *pixBuf, v int) {
		b.ellipse(14, 20, 12, 2, color.RGBA{0, 0, 0, 80})
		b.rect(2, 8, 24, 12, c(0xc8a060))
		b.rect(2, 8, 24, 2, c(0xe0c080))
		b.rect(2, 8, 24, 12, c(0xc8a060))
		for i, col := range []uint32{0xe03040, 0x3070e0, 0xf0c030, 0x30c070} {
			b.rect(4+i*6, 3, 4, 7, c(col)) // rocket tubes
			b.rect(5+i*6, 1, 2, 3, c(0xf4f4f0))
		}
		b.rect(4, 12, 20, 5, c(0xe8e0c8))
		b.rect(6, 14, 16, 1, c(0xd02030))
	}}
	pk["lever"] = &PropKind{W: 20, H: 30, FW: 16, FH: 8, Trap: "piano", Draw: func(b *pixBuf, v int) {
		b.ellipse(10, 28, 8, 2, color.RGBA{0, 0, 0, 80})
		b.rect(2, 20, 16, 8, c(0x5a5a64))
		b.rect(2, 20, 16, 2, c(0x8a8a96))
		b.line(10, 22, 15, 6, c(0xb0b0bc))
		b.line(11, 22, 16, 6, c(0x8a8a96))
		b.ellipse(15, 5, 3, 3, c(0xe03030))
		b.rect(6, 16, 8, 3, c(0xf0c030)) // warning plate
	}}
	// stool and tv can be sat on too
	if k := pk["stool"]; k != nil && k.UseKind == "" {
		k.UseKind, k.Use = "lounge", []Vec{{0, 10}}
	}
}

// ---- floating point texts -----------------------------------------------------------

type floatText struct {
	pos Vec
	txt string
	t   float64
	col color.RGBA
}

// award gives points and shows where they came from.
func (g *Game) award(pos Vec, n int, why string) {
	g.points += n
	txt := "+" + itoa(n)
	if why != "" {
		txt += " " + why
	}
	g.floats = append(g.floats, floatText{pos: Vec{pos.X, pos.Y - 30}, txt: txt, t: 1.8, col: rgb(0xffe060)})
}

func (g *Game) updateFloats(dt float64) {
	out := g.floats[:0]
	for _, f := range g.floats {
		f.t -= dt
		f.pos.Y -= 14 * dt
		if f.t > 0 {
			out = append(out, f)
		}
	}
	g.floats = out
}

func (g *Game) drawFloats(dst *ebiten.Image) {
	for _, f := range g.floats {
		a := math.Min(1, f.t)
		col := f.col
		col.A = uint8(255 * a)
		g.drawTextC(dst, f.txt, g.wx(f.pos.X), g.wy(f.pos.Y), col, 1.2)
	}
}

// ---- what the player can use ------------------------------------------------------

// useLabel: what pressing the use key would do on this prop right now, or "".
func (g *Game) useLabel(pr *Prop) (string, color.RGBA) {
	k, p := pr.K, g.p
	if pr.gone {
		return "", color.RGBA{}
	}
	switch {
	case k.Hide:
		if p.carrying != nil {
			return "Esconder cuerpo", rgb(0x70e0ff)
		}
		return "Esconderse", rgb(0x70e0ff)
	case k.Phone && !pr.used:
		if g.canCallAmbulance(p) {
			return "Llamar a la ambulancia", rgb(0x60ffa0)
		}
		return "Cortar cable", rgb(0xff6060)
	case k.Item != "" && !canTake(p, k.Item):
		return "", color.RGBA{}
	case k.Item == "poison":
		return "Coger veneno", rgb(0x60ff80)
	case k.Item == "disguise":
		return "Coger disfraz", rgb(0xd090ff)
	case k.Item == "suit":
		return "Ponerte el traje dorado", rgb(0xffd860)
	case k.Poison && pr.armT <= 0:
		if p.item == "poison" {
			return "Envenenar", rgb(0x60ff80)
		}
		return "Necesita veneno", rgb(0x70a080)
	case (pr.Name == "closet" || pr.Name == "wardrobe") && g.wanted > 0:
		return "Cambiarse de ropa", rgb(0xd090ff)
	case k.Trap != "" && !pr.used && pr.armT == 0 && !pr.armed:
		return "Armar trampa", rgb(0xffa040)
	}
	return "", color.RGBA{}
}

// drawUseMarks: a bobbing diamond over everything that can be used; close up
// it turns into the key and the name of the action.
func (g *Game) drawUseMarks(dst *ebiten.Image) {
	if g.mode != ModePlay {
		return
	}
	// with two players each one gets his own marks (the key and the colour are his): the diamond
	// over a prop is drawn once, the text only when that player is close
	drawn := map[*Prop]bool{}
	for _, pl := range g.players() {
		if pl == nil || pl.dead || pl.hidden {
			continue
		}
		pl := pl
		g.withPlayer(pl, func() { g.drawUseMarksFor(dst, pl, drawn) })
	}
}

func (g *Game) drawUseMarksFor(dst *ebiten.Image, pl *Player, drawn map[*Prop]bool) {
	hint := actionHintFor(actInteract, pl.idx)
	for _, pr := range g.lvl.Props {
		lab, col := g.useLabel(pr)
		if lab == "" {
			continue
		}
		d := distRect(g.p.pos, pr.Foot)
		if d > 260 {
			continue
		}
		top := float64(pr.K.H) * 0.62
		if pr.K.NoSolid && pr.K.H < 30 {
			top = float64(pr.K.H) + 4
		}
		x, y := g.wx(pr.X), g.wy(pr.Y)-top-8+math.Sin(g.time*4+pr.X)*2
		a := 0.45 + 0.35*math.Sin(g.time*3+pr.Y)
		if d < 56 {
			a = 1
		}
		c := col
		c.A = uint8(255 * a)
		if !drawn[pr] {
			drawn[pr] = true
			for i := 0.0; i < 4; i++ {
				fillRect(dst, x-i, y-4+i, 1+2*i, 1, c)
				fillRect(dst, x-3+i, y+i, 7-2*i, 1, c)
			}
		}
		if d < 30 {
			txt := "[" + hint + "] " + lab
			ty := y - 16
			if g.is2P() {
				ty = g.clearOfTags(x, ty, g.textW(txt, 1)+10)
			}
			g.drawTextC(dst, txt, x, ty, col, 1)
			if g.is2P() { // a small square in the player's colour says whose key it is
				pc := playerColor(pl.idx)
				fillRect(dst, math.Round(x-g.textW(txt, 1)/2)-7, ty+2, 4, 4, pc)
			}
		}
	}
	// with poison in the pocket the guest within reach can be offered a drink
	if gu := g.poisonTarget(); gu != nil {
		col := rgb(0x60ff80)
		x, y := g.wx(gu.pos.X), g.wy(gu.pos.Y)-48+math.Sin(g.time*4)*2
		for i := 0.0; i < 4; i++ {
			fillRect(dst, x-i, y-4+i, 1+2*i, 1, col)
			fillRect(dst, x-3+i, y+i, 7-2*i, 1, col)
		}
		g.drawTextC(dst, "["+hint+"] Ofrecer copa envenenada", x, y-16, col, 1)
	}
}

// clearOfTags (two players): the y for a text of width w centred on x whose top is at ty so that it does not
// sit on the J1/J2 tag of a player, nor on his cry or his "!" alert (they stack up to 46 px above the tag).
func (g *Game) clearOfTags(x, ty, w float64) float64 {
	for k := 0; k < 3; k++ { // moving up for one tag can land on the other
		moved := false
		for _, p := range g.players() {
			if p == nil || p.hidden {
				continue
			}
			sc := g.charSc
			if sc <= 0 {
				sc = 1
			}
			px, py := math.Round(g.wx(p.pos.X)), math.Round(g.wy(p.pos.Y))-40*sc-6
			top := py - 30 // the tag is ~10 px tall and the arrow marker of the player floats above it; the cry sits over both
			if p.msgT > 0 && p.msg != "" {
				top -= 11
			}
			if g.playerAlert(p) {
				top -= 28 // the badge and its pulse
			}
			if math.Abs(x-px) < w/2+14 && ty < py+10 && ty+11 > top {
				ty = top - 12
				moved = true
			}
		}
		if !moved {
			break
		}
	}
	return ty
}

// ---- items ----------------------------------------------------------------------------

// pickReach is how close (to the foot of the prop) a player must be to pick an item up: the same
// distance at which the "[E] Coger ..." label shows (it used to be 16 against the label's 30).
const pickReach = 30.0

// canTake: the player can pick this kind up (a second slot holds the other kind; the same kind only
// refills a bottle that is not full).
func canTake(p *Player, kind string) bool {
	switch {
	case kind == "suit":
		return p.suitT <= 0
	case p.item == kind:
		return kind == "poison" && p.doses < poisonDoses
	case p.item != "" && p.spare == kind:
		return false
	}
	return true
}

// pickupInteract: poison and clothes can ALWAYS be picked up. Nothing but being dead, in the hands
// of the police or knocked out (updatePlayer already stops those) prevents it: not the stars, not the
// alert, not the cuffs, not carrying a body, not a hiding spot, not the two-player gestures (a cop to
// distract, a high five) that used to take the key first.
func (g *Game) pickupInteract(p *Player) bool {
	if p == nil || p.dead || p.hidden || p.jail.held {
		return false
	}
	if g.vipInteract(p) { // help the VIP who tips (vip.go)
		return true
	}
	var best *Prop
	bd := pickReach
	for _, pr := range g.lvl.Props {
		if pr.gone || pr.K.Item == "" || !canTake(p, pr.K.Item) {
			continue
		}
		if d := distRect(p.pos, pr.Foot); d <= bd {
			best, bd = pr, d
		}
	}
	if best == nil {
		return false
	}
	for _, pr := range g.lvl.Props { // something else to use that is closer wins (a bush, a phone...)
		if pr != best && !pr.gone && pr.K.Item == "" && distRect(p.pos, pr.Foot) < bd {
			if lab, _ := g.useLabel(pr); lab != "" {
				return false
			}
		}
	}
	g.pickItem(best)
	return true
}

func (g *Game) pickItem(pr *Prop) {
	p := g.p
	pr.gone = true
	g.snd.Play("spark")
	kind := pr.K.Item
	if kind == "suit" {
		g.putOnSuit(p) // worn at once, one use (vip.go)
		return
	}
	if p.item != "" && p.item != kind {
		p.spare, p.spareDoses = kind, 0 // the other kind waits in the second slot
		if kind == "poison" {
			p.spareDoses = poisonDoses
			g.say("Guardas el frasco de veneno junto al disfraz")
		} else {
			g.say("Guardas un disfraz: se usa solo cuando te buscan")
		}
		return
	}
	switch kind {
	case "poison":
		p.item, p.doses = "poison", poisonDoses
		g.say("Tienes veneno: da copas a la gente o échalo en el ponche")
		g.say(dosesText(p.doses))
	case "disguise":
		p.item = "disguise"
		g.say("Tienes un disfraz: se usa solo cuando te buscan")
	}
}

// promoteSpare: when the carried item is used up, the one in the second slot comes out.
func (p *Player) promoteSpare() {
	if p.item == "" && p.spare != "" {
		p.item, p.doses, p.spare, p.spareDoses = p.spare, p.spareDoses, "", 0
	}
}

// useDisguise: a new look from head to toe; the police lose the trail and the
// stars go away.
func (g *Game) useDisguise() {
	spec := genericSpec(g.rng)
	if g.p.cs != nil && g.p.cs.Spec != nil {
		// keep the player's build so it still feels like the same person
		spec.Body = g.p.cs.Spec.Body
		spec.Stature = g.p.cs.Spec.Stature
	}
	g.p.cs = newCharSet(&spec)
	g.p.item = ""
	g.p.promoteSpare()
	g.p.disguised = true
	g.p.jail.sus = false // the cuffs mark leaves him (two players, twojail.go)
	// the police lose the trail when nobody is left that they could still be after
	if g.disguiseClosesCase() {
		g.closeCase()
	}
	g.burst(Vec{g.p.pos.X, g.p.pos.Y - 16}, 20, 80, rgb(0xd090ff), 0.6, 2, 0)
	g.snd.Play("spark")
	g.say("¡Disfraz puesto! Ya no te buscan")
	g.award(g.p.pos, 50, "disfraz")
}

// ---- the three new traps ------------------------------------------------------------------

type gasCloud struct {
	pos Vec
	t   float64
}
type fwBox struct {
	pos    Vec
	t, nxt float64
}
type pianoDrop struct {
	pos Vec
	t   float64
	hit bool
}

func (g *Game) trapKill(gu *Guest) {
	g.killCause = kcTrap
	g.killGuest(gu, false)
	g.stat("traps", 1)
	g.secretTrapKill()
}

func (g *Game) trapCops(centre Vec, r float64) {
	for _, c := range append([]*Cop(nil), g.cops...) {
		if !c.medic && dist(c.pos, centre) < r {
			g.stabCop(c)
		}
	}
}

func (g *Game) triggerNewTrap(pr *Prop) {
	centre := Vec{pr.X, pr.Y - 8}
	switch pr.K.Trap {
	case "gas":
		g.snd.Play("alert")
		g.gas = append(g.gas, gasCloud{pos: centre, t: 7})
		g.say("¡Fuga de gas!")
	case "fireworks":
		g.snd.Play("spark")
		g.fw = append(g.fw, fwBox{pos: centre, t: 4.5})
		g.say("¡Se disparan los fuegos artificiales!")
	case "piano":
		// the piano drops on the biggest crowd
		best, bn := centre, -1
		for _, gu := range g.guests {
			if gu.out() {
				continue
			}
			n := 0
			for _, o := range g.guests {
				if !o.out() && dist(o.pos, gu.pos) < 36 {
					n++
				}
			}
			if n > bn {
				best, bn = gu.pos, n
			}
		}
		g.pianos = append(g.pianos, pianoDrop{pos: best})
		g.say("Algo enorme cae del cielo...")
	}
}

func (g *Game) updateTraps2(dt float64) {
	// gas: a green cloud that chokes everybody in it
	out := g.gas[:0]
	for _, c := range g.gas {
		c.t -= dt
		if g.rng.Float64() < dt*24 {
			a := g.rng.Float64() * 2 * math.Pi
			r := g.rng.Float64() * 52
			g.burst(Vec{c.pos.X + math.Cos(a)*r, c.pos.Y - 4 + math.Sin(a)*r*0.5}, 1, 10, color.RGBA{120, 200, 60, 150}, 1.1, 4, -14)
		}
		g.credit = true
		for _, gu := range g.guests {
			if gu.out() || dist(gu.pos, c.pos) > 56 {
				continue
			}
			gu.gasT += dt
			if gu.iconT <= 0 {
				gu.icon, gu.iconT = []string{"*COF COF*", "¡GAS!", "¡NO RESPIRO!"}[g.rng.Intn(3)], 1.2
			}
			if gu.gasT > 2.2 {
				g.trapKill(gu)
			}
		}
		g.credit = false
		for _, pl := range g.players() { // every player breathes his own share
			pl := pl
			exp := &g.gasP
			if pl.idx == 1 {
				exp = &g.gasP2
			}
			if pl.dead || dist(pl.pos, c.pos) >= 50 {
				continue
			}
			*exp += dt
			if *exp > 3 {
				g.withPlayer(pl, func() { g.die("Te has asfixiado con tu propia trampa de gas") })
			}
		}
		g.trapCopsGas(c)
		if c.t > 0 {
			out = append(out, c)
		}
	}
	g.gas = out
	// fireworks: rockets in every direction
	fo := g.fw[:0]
	for _, f := range g.fw {
		f.t -= dt
		f.nxt -= dt
		if f.nxt <= 0 {
			f.nxt = 0.14
			a := g.rng.Float64() * 2 * math.Pi
			r := 18 + g.rng.Float64()*80
			at := Vec{f.pos.X + math.Cos(a)*r, f.pos.Y + math.Sin(a)*r*0.7 - 10}
			cols := []uint32{0xff4040, 0xffd040, 0x40ff80, 0x40c0ff, 0xff60e0}
			g.burst(at, 18, 90, rgb(cols[g.rng.Intn(len(cols))]), 0.7, 2, 70)
			g.snd.Play("spark")
			g.credit = true
			for _, gu := range g.guests {
				if !gu.out() && dist(gu.pos, at) < 30 {
					g.trapKill(gu)
				}
			}
			g.credit = false
			for _, gd := range g.guards {
				if !gd.dead && dist(gd.pos, at) < 30 {
					g.killGuard(gd)
				}
			}
			g.trapCops(at, 30)
			g.eachPlayer(func(p *Player) {
				if !p.dead && dist(p.pos, at) < 20 {
					g.die("Un cohete de tu trampa te ha dado")
				}
			})
			g.checkDone()
		}
		if f.t > 0 {
			fo = append(fo, f)
		}
	}
	g.fw = fo
	// the piano
	po := g.pianos[:0]
	for _, d := range g.pianos {
		d.t += dt
		if d.t >= 0.9 && !d.hit {
			d.hit = true
			g.snd.Play("boom")
			g.shake = 0.5
			g.burst(Vec{d.pos.X, d.pos.Y - 6}, 26, 110, rgb(0xd8d0c0), 0.6, 3, 90)
			g.decals = append(g.decals, Decal{pos: d.pos, kind: 1, r: 16})
			g.credit = true
			for _, gu := range g.guests {
				if !gu.out() && math.Abs(gu.pos.X-d.pos.X) < 32 && math.Abs(gu.pos.Y-d.pos.Y) < 22 {
					g.trapKill(gu)
				}
			}
			g.credit = false
			for _, gd := range g.guards {
				if !gd.dead && math.Abs(gd.pos.X-d.pos.X) < 32 && math.Abs(gd.pos.Y-d.pos.Y) < 22 {
					g.killGuard(gd)
				}
			}
			g.trapCops(d.pos, 34)
			g.eachPlayer(func(p *Player) {
				if !p.dead && math.Abs(p.pos.X-d.pos.X) < 30 && math.Abs(p.pos.Y-d.pos.Y) < 20 {
					g.die("Te ha caído un piano encima")
				}
			})
			g.checkDone()
		}
		if d.t < 4 {
			po = append(po, d)
		}
	}
	g.pianos = po
}

func (g *Game) trapCopsGas(c gasCloud) {
	for _, o := range append([]*Cop(nil), g.cops...) {
		if !o.medic && dist(o.pos, c.pos) < 40 && g.rng.Float64() < 0.01 {
			g.stabCop(o)
		}
	}
}

func (g *Game) drawTraps2(dst *ebiten.Image) {
	for _, d := range g.pianos {
		im := g.as.props["piano"]
		if len(im) == 0 {
			continue
		}
		pi := im[0]
		h := math.Max(0, (0.9-d.t)/0.9) * 320
		if d.t < 0.9 {
			// the shadow grows as it comes down
			s := 8 + 26*(d.t/0.9)
			filledEllipse(dst, g.wx(d.pos.X), g.wy(d.pos.Y), s, s*0.5, color.RGBA{0, 0, 0, 110})
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(math.Round(g.wx(d.pos.X)-pi.ox), math.Round(g.wy(d.pos.Y)-pi.oy-h))
		dst.DrawImage(pi.img, op)
	}
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ensureItems puts the things to use into every level: a drink bowl, poison
// to poison it with, a disguise, somewhere to sit and a few more traps.
func (l *Level) ensureItems() bool {
	added := false
	var seeds []Vec
	for _, p := range l.Props {
		if p.K.Trap != "" || p.K.Poison || p.K.Phone || p.K.TechPhone {
			seeds = append(seeds, Vec{p.X, p.Y})
		}
	}
	place := func(name string, n int) {
		before := len(l.Props)
		if l.ensureBooth(name, n+countKind(l, name), func(p *Prop) bool { return p.Name == name }, seeds, 28) {
			added = true
		}
		for _, p := range l.Props[before:] {
			seeds = append(seeds, Vec{p.X, p.Y})
			for _, u := range p.K.Use {
				l.POIs[p.K.UseKind] = append(l.POIs[p.K.UseKind], Vec{p.X + u.X*p.sc(), p.Y + u.Y*p.sc()})
			}
		}
	}
	place("bowl", 1)
	place("poisonbottle", 2)
	place("disguisebag", 1)
	place("chair", 2)
	place("gastank", 1)
	place("fireworksbox", 1)
	place("lever", 1)
	return added
}
