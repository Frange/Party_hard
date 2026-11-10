package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Phone technicians. When somebody sees a cut phone line he cannot call the
// police any more: he runs to the yellow phone (there is one in every party,
// it cannot be cut) and calls the technicians. A yellow van brings one or two
// of them, who walk to the cut phones and repair them.
// ---------------------------------------------------------------------------

func init() {
	propKinds["techphone"] = &PropKind{W: 24, H: 46, FW: 16, FH: 8, TechPhone: true, Draw: func(b *pixBuf, v int) {
		yel, dark := c(0xf0c820), c(0xa8800c)
		b.rect(1, 0, 22, 4, dark) // roof
		b.rect(2, 0, 20, 1, c(0xfce070))
		b.rect(6, 1, 12, 2, c(0x2a2a34))  // sign
		b.rect(8, 1, 8, 1, c(0xf0c820))   // sign light
		b.rect(2, 4, 20, 38, yel)         // frame
		b.rect(4, 6, 16, 32, c(0x86b8d0)) // glass
		b.rect(5, 7, 4, 28, c(0xa8d4e8))
		b.rect(11, 4, 2, 38, yel) // mullions
		b.rect(4, 17, 16, 2, yel)
		b.rect(4, 28, 16, 2, yel)
		b.rect(7, 20, 10, 7, c(0x2a2a34)) // the phone inside
		b.rect(8, 21, 8, 2, c(0xd8d8e0))
		b.rect(9, 24, 2, 2, c(0x8a8a9a))
		b.rect(13, 24, 2, 2, c(0x8a8a9a))
		// a small spanner painted on the lower glass
		b.rect(8, 31, 8, 1, c(0x2a2a34))
		b.rect(7, 30, 2, 3, c(0x2a2a34))
		b.rect(1, 40, 22, 5, dark) // base
	}}
	propKinds["techvan"] = &PropKind{W: 86, H: 40, FW: 80, FH: 12, NoSolid: true, Vars: 1, Category: "vehicle", Draw: func(b *pixBuf, v int) { vaTechVanSide(b) }}
}

type Tech struct {
	pos    Vec
	dir    Dir
	cs     *CharSet
	path   []Vec
	state  int // 0 to the phone, 1 repairing, 2 back to the van, 3 gone
	timer  float64
	anim   float64
	moving bool
	curSpd float64
	van    *Vehicle
	target *Prop
}

func techSpec() CharSpec {
	return CharSpec{Name: "tecnico", DisplayName: "Técnico", Skin: hx(0xe8b48a), Hair: hx(0x2b1d12), Shirt: hx(0xf08a20), Shirt2: hx(0xf0c820),
		Pants: hx(0x2a3a5a), Shoes: hx(0x14141a), Top: TopVest, Hat: HatHardhat, HatCol: hx(0xf0c820), Accent: hx(0xf0c820), Hair2: HairShort, Body: BodyBig,
		Carry: HoldBriefcase, CarryCol: hx(0xc03a2a)}
}

// cutPhones lists the phones with a cut line that no technician is heading for yet.
func (g *Game) cutPhones() []*Prop {
	var out []*Prop
	for _, pr := range g.lvl.Props {
		if pr.K.Phone && pr.used && pr.techBy == nil {
			out = append(out, pr)
		}
	}
	return out
}

func (g *Game) techPhoneProp() *Prop {
	for _, pr := range g.lvl.Props {
		if pr.K.TechPhone {
			return pr
		}
	}
	return nil
}

func (g *Game) vehLive(v *Vehicle) bool {
	for _, o := range g.veh {
		if o == v && !o.dead {
			return true
		}
	}
	return false
}

// techBusy: technicians are called, on their way or at work.
func (g *Game) techBusy() bool {
	return g.techVan != nil || g.techWait > 0 || len(g.techCrew) > 0
}

// detachPartner breaks up a chat or a kiss before the guest goes off.
func (g *Game) detachPartner(gu *Guest) {
	if gu.partner != nil && gu.partner.partner == gu {
		gu.partner.partner = nil
		if gu.partner.state == gChat || gu.partner.state == gKiss || gu.partner.state == gWaitP {
			gu.partner.state, gu.partner.timer = gIdle, 1
		}
	}
	gu.partner = nil
}

func (g *Game) updateTech(dt float64) {
	if g.lvl == nil || g.mode != ModePlay {
		return
	}
	if g.dbgCut && g.time > 3 {
		g.dbgCut = false
		for _, pr := range g.lvl.Props {
			if pr.K.Phone || pr.K.TechPhone {
				tlog("prop %s at %.0f,%.0f", pr.Name, pr.X, pr.Y)
			}
			if pr.K.Phone {
				pr.used = true
			}
		}
	}
	if g.techVan != nil && !g.vehLive(g.techVan) {
		g.techVan = nil
	}
	if g.techWait > 0 {
		g.techWait -= dt
		if g.techWait <= 0 {
			g.techWait = 0
			g.techsArrive(nil)
		}
	}
	alive := g.techCrew[:0]
	for _, t := range g.techCrew {
		g.updateTechnician(t, dt)
		if t.state != 3 {
			alive = append(alive, t)
		}
	}
	g.techCrew = alive
	if v := g.techVan; v != nil && v.spawned && len(g.techCrew) == 0 && v.state == 1 {
		v.state = 2
		g.techVan = nil
	}
	cuts := g.cutPhones()
	if g.techBusy() {
		return
	}
	if len(cuts) == 0 {
		if c := g.techCaller; c != nil {
			c.tech = false
			if c.state == gToPhone || c.state == gCall {
				c.state, c.timer, c.path = gIdle, 1, nil
			}
			g.techCaller = nil
		}
		return
	}
	if c := g.techCaller; c != nil {
		if c.out() || !c.tech || g.time > g.techCallT {
			c.tech = false
			if !c.out() && (c.state == gToPhone || c.state == gCall) {
				c.state, c.timer, c.path = gIdle, 1, nil
			}
			g.techCaller = nil
		} else {
			return
		}
	}
	g.techScanT -= dt
	if g.techScanT > 0 {
		return
	}
	g.techScanT = 0.5
	tp := g.techPhoneProp()
	for _, pr := range cuts {
		spot := Vec{pr.X, pr.Y - 12}
		for _, gu := range g.guests {
			if !g.canWatch(gu) || gu.perf != "" || gu.state == gZombie || gu.state == gLift || gu.state == gKiller || gu.state == gPunch || gu.state == gFun {
				continue
			}
			if !g.sees(gu.pos, gu.dir, spot, 130, guestDot) {
				continue
			}
			g.detachPartner(gu)
			gu.tech = true
			gu.state = gToPhone
			gu.path = nil
			if tp != nil {
				gu.path = g.lvl.PathG(gu.pos, Vec{tp.X, tp.Y + 14})
			}
			gu.icon, gu.iconT = "¡Está cortado!", 2.2
			g.snd.Play("alert")
			g.techCaller, g.techCallT = gu, g.time+45
			tlog("%.1f caller %s goes to the yellow phone (%d cut)", g.time, gu.name, len(cuts))
			return
		}
	}
}

// dispatchTech: the call went through, the technicians come.
func (g *Game) dispatchTech() {
	def := g.lvl.Def
	if def.Police == "car" {
		tx, ly := g.parkSlot()
		v := &Vehicle{kind: 8, x: def.CarFrom.X + 60, y: ly, tx: tx, ty: ly}
		g.veh = append(g.veh, v)
		g.techVan = v
	} else {
		g.techWait = 9 // no road: they simply turn up
	}
	g.snd.Play("ring")
	g.say("Han avisado a los técnicos del teléfono")
	tlog("%.1f technicians called (van=%v)", g.time, g.techVan != nil)
}

// techsArrive: the van has parked (or, with no road, they walk in).
func (g *Game) techsArrive(v *Vehicle) {
	cuts := g.cutPhones()
	n := 1
	if len(cuts) > 1 {
		n = 2
	}
	spec := techSpec()
	var origin Vec
	if v != nil {
		origin = Vec{v.x - 14, v.y - 24}
	} else if tp := g.techPhoneProp(); tp != nil {
		origin = Vec{tp.X, tp.Y + 18}
	} else {
		origin = g.lvl.Def.Start
	}
	for i := 0; i < n; i++ {
		sp := spec
		sp.Hair = hx(pick(g.rng, hairTones))
		sp.Name = "tecnico"
		cs := newCharSet(&sp)
		if len(g.as.techCS) > 0 {
			cs = g.as.techCS[i%len(g.as.techCS)]
		}
		pos := g.lvl.NearestFree(Vec{origin.X + float64(i)*10, origin.Y - float64(i%2)*6})
		t := &Tech{pos: pos, dir: DirDown, cs: cs, van: v}
		g.techCrew = append(g.techCrew, t)
		g.techPick(t)
	}
	g.say("Llegan los técnicos")
	tlog("%.1f %d technicians arrive", g.time, n)
}

// techPick sends a technician to the nearest cut phone nobody is on yet.
func (g *Game) techPick(t *Tech) {
	var best *Prop
	bd := 1e9
	for _, pr := range g.cutPhones() {
		if d := dist(t.pos, Vec{pr.X, pr.Y}); d < bd {
			best, bd = pr, d
		}
	}
	if best == nil {
		t.state, t.target = 2, nil
		t.path = g.lvl.Path(t.pos, g.techHome(t))
		return
	}
	best.techBy = t
	t.target, t.state = best, 0
	t.path = g.lvl.Path(t.pos, g.lvl.NearestFree(Vec{best.X, best.Y + 16}))
}

// techHome is where a technician returns to: the van, or the yellow phone when there is none.
func (g *Game) techHome(t *Tech) Vec {
	if t.van != nil {
		return g.lvl.NearestFree(Vec{t.van.x - 14, t.van.y - 24})
	}
	if tp := g.techPhoneProp(); tp != nil {
		return g.lvl.NearestFree(Vec{tp.X, tp.Y + 18})
	}
	return g.lvl.Def.Start
}

func (g *Game) updateTechnician(t *Tech, dt float64) {
	t.moving = false
	switch t.state {
	case 0:
		if t.target == nil || !t.target.used {
			g.techPick(t) // somebody else repaired it, or it is gone
			break
		}
		m, arr := follow(&t.pos, &t.dir, &t.path, 62*paceK, dt)
		t.moving, t.curSpd = m, 62
		if arr || dist(t.pos, Vec{t.target.X, t.target.Y + 8}) < 20 {
			t.state, t.timer = 1, 4.5+g.rng.Float64()*2
			t.dir = DirUp
		}
	case 1:
		t.timer -= dt
		if g.rng.Float64() < dt*5 {
			g.burst(Vec{t.target.X, t.target.Y - 18}, 3, 30, rgb(0xffe060), 0.35, 2, 50)
		}
		if t.timer <= 0 {
			pr := t.target
			if pr != nil {
				pr.used, pr.techBy = false, nil
				g.snd.Play("spark")
				g.burst(Vec{pr.X, pr.Y - 18}, 12, 45, rgb(0xffe060), 0.5, 2, 60)
				g.stat("techs", 1)
				tlog("%.1f phone repaired", g.time)
				g.say("Los técnicos han arreglado el teléfono")
			}
			g.techPick(t)
		}
	case 2:
		if cuts := g.cutPhones(); len(cuts) > 0 {
			g.techPick(t) // another line was cut while they were leaving
			break
		}
		m, arr := follow(&t.pos, &t.dir, &t.path, 62*paceK, dt)
		t.moving, t.curSpd = m, 62
		if arr || dist(t.pos, g.techHome(t)) < 16 {
			t.state = 3
		}
	}
	if t.moving {
		t.anim += dt * stridePhase(t.curSpd)
	}
}

func (g *Game) drawTechnician(dst *ebiten.Image, t *Tech) {
	anim, phase := AnimIdle, g.time*1.2+t.pos.X
	switch {
	case t.moving:
		anim, phase = AnimWalk, t.anim
	case t.state == 1:
		anim, phase = AnimCrouch, g.time*2
	}
	g.drawActor(dst, t.cs, anim, phase, t.pos, t.dir, 0, false)
}

func (g *Game) drawTechVan(dst *ebiten.Image, v *Vehicle) {
	imgs := g.as.props["techvan"]
	if len(imgs) == 0 {
		return
	}
	pi := imgs[0]
	w := float64(pi.img.Bounds().Dx())
	op := &ebiten.DrawImageOptions{}
	if v.state != 2 { // the sprite faces right: mirror it while it drives in from the right
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(w, 0)
		op.GeoM.Translate(math.Round(g.wx(v.x)-(w-pi.ox)), math.Round(g.wy(v.y)-pi.oy))
	} else {
		op.GeoM.Translate(math.Round(g.wx(v.x)-pi.ox), math.Round(g.wy(v.y)-pi.oy))
	}
	dst.DrawImage(pi.img, op)
	if int(g.time*3)%2 == 0 {
		g.glowAt(dst, Vec{v.x - 8, v.y - 32}, 22, color.RGBA{255, 150, 20, 130})
	}
}

// drawTechMarker: a yellow badge over the technicians' phone while a cut line waits to be reported.
func (g *Game) drawTechMarker(dst *ebiten.Image) {
	if g.mode != ModePlay || g.lvl == nil || g.techBusy() || len(g.cutPhones()) == 0 {
		return
	}
	pr := g.techPhoneProp()
	if pr == nil {
		return
	}
	x, y := g.wx(pr.X), g.wy(pr.Y)
	if x < -20 || x > screenW+20 || y < -20 || y > screenH+60 {
		return
	}
	pulse := 0.5 + 0.5*math.Sin(g.time*3)
	g.glowAt(dst, Vec{x - g.camI.X, y - g.camI.Y - 6}, 20+pulse*5, color.RGBA{250, 210, 40, uint8(50 + pulse*40)})
	by := math.Round(y - 60 + math.Sin(g.time*3.2)*2)
	bx := math.Round(x) - 8
	fillRect(dst, bx, by, 16, 12, rgb(0x6a5008))
	fillRect(dst, bx+1, by+1, 14, 10, rgb(0xf0c820))
	k := rgb(0x2a2a34) // a spanner
	fillRect(dst, bx+3, by+7, 8, 2, k)
	fillRect(dst, bx+9, by+3, 3, 5, k)
	fillRect(dst, bx+10, by+3, 1, 2, rgb(0xf0c820))
	fillRect(dst, bx+7, by+12, 2, 2, rgb(0xf0c820))
}

// techLog prints the technicians' steps when the -cutphones debug flag is on.
var techLog bool

func tlog(f string, a ...any) {
	if techLog {
		fmt.Printf(f+"\n", a...)
	}
}
