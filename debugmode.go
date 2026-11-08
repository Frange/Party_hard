package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// DEBUG mode: opened with the Konami code in the main menu (up, up, down, down,
// left, right, left, right). Everything here is temporary: the real progress is
// copied when the session starts, nothing is written to disk while it lasts,
// and "Salir del DEBUG" puts the real progress back.
// ---------------------------------------------------------------------------

type debugState struct {
	immune, killCops, powers bool
	levels, shop             bool
	trophyMode               int // 0 as they are, 1 all won, 2 none
	bonus                    int // points added to the wallet (temporary)
	speed, noPolice, fastEv  bool
	guestMin, guestMax       int  // fewest / most guests (0 = what is saved)
	copMemory                bool // police with memory: a scared witness tells on you when he meets an officer (copmemory.go)
}

var debugSnap *Progress

// debugPrevFrozen: progressFrozen as it was when the DEBUG menu mode began (restored by debugEnd).
var debugPrevFrozen bool

func (g *Game) debugBegin() {
	if debugSnap == nil {
		c := cloneProgress(g.progress)
		debugSnap = &c
		debugPrevFrozen = progressFrozen
		progressFrozen = true
	}
}

// debugApply rebuilds the in-memory progress from the real one plus the active switches.
func (g *Game) debugApply() {
	g.debugBegin()
	p := cloneProgress(*debugSnap)
	d := g.dbg
	own := map[string]bool{}
	for _, id := range p.Owned {
		own[id] = true
	}
	add := func(id string) {
		if !own[id] {
			own[id] = true
			p.Owned = append(p.Owned, id)
		}
	}
	if d.levels {
		p.Unlocked = campaignN
	}
	if d.shop {
		for _, list := range shopCatalog {
			for _, it := range list {
				add(it.id)
			}
		}
	}
	switch d.trophyMode {
	case 1:
		for _, t := range trophyList {
			p.Trophies[t.id] = true
			add(t.reward.id)
		}
		for _, r := range legendExtras() {
			add(r.id)
		}
	case 2:
		p.Trophies = map[string]bool{}
		drop := map[string]bool{}
		for _, t := range trophyList {
			drop[t.reward.id] = true
		}
		for _, r := range legendExtras() {
			drop[r.id] = true
		}
		var keep []string
		for _, id := range p.Owned {
			if !drop[id] {
				keep = append(keep, id)
			}
		}
		p.Owned = keep
	}
	p.Wallet += d.bonus
	if d.guestMin > 0 {
		p.GuestMin = d.guestMin
	}
	if d.guestMax > 0 {
		p.GuestMax = d.guestMax
	}
	g.progress = p
	g.syncShop()
	g.applyShop()
	fastEv = d.fastEv
}

// debugEnd puts the real progress back and closes the session.
func (g *Game) debugEnd() {
	if debugSnap != nil {
		g.progress = *debugSnap
		debugSnap = nil
	}
	progressFrozen = debugPrevFrozen
	debugPrevFrozen = false
	g.dbg = debugState{}
	g.debugMenu = false
	g.powerImmune = false
	fastEv = false
	g.syncShop()
	g.applyShop()
}

// konamiStep watches the main menu for the secret sequence.
func (g *Game) konamiStep() {
	seq := []Action{actUp, actUp, actDown, actDown, actLeft, actRight, actLeft, actRight}
	for _, a := range []Action{actUp, actDown, actLeft, actRight} {
		if !a.just() {
			continue
		}
		switch {
		case a == seq[g.konami]:
			g.konami++
		case a == actUp:
			g.konami = 1
		default:
			g.konami = 0
		}
		break
	}
	if g.konami >= len(seq) {
		g.konami = 0
		if !g.debugMenu {
			g.debugMenu = true
			g.mainSel = 0
			g.menuMsg, g.menuMsgT = "MODO DEBUG DESBLOQUEADO", 3
			g.snd.Play("win")
		}
	}
}

// ---- the main menu rows, with the hidden one on top when it is unlocked ----------

func (g *Game) menuList() []int {
	l := []int{menuContinue, menuNewGame, menuPlayers, menuShop, menuTrophies, menuScenes, menuSettings, menuQuit}
	if !g.hasSave() {
		l = l[1:] // the first time: there is nothing to continue
	}
	if g.debugMenu {
		l = append([]int{menuDebug}, l...)
	}
	return l
}

func menuLabel(id int) string {
	switch id {
	case menuDebug:
		return "DEBUG"
	case menuContinue:
		return "CONTINUAR"
	case menuNewGame:
		return "NUEVO JUEGO"
	case menuPlayers:
		return "JUGADORES"
	case menuShop:
		return "TIENDA"
	case menuTrophies:
		return "TROFEOS"
	case menuScenes:
		return "ESCENAS"
	case menuSettings:
		return "AJUSTES"
	}
	return "SALIR"
}

func menuRowY(i, n int) float64 {
	if n > 9 { // DEBUG with the save: ten rows, the last one must stay above the footer texts (y=474)
		return 140 + float64(i)*33
	}
	if n > 7 {
		return 146 + float64(i)*36
	}
	return 178 + float64(i)*40
}

// ---- the DEBUG screen ---------------------------------------------------------------

type debugRow struct {
	label string
	val   func() string
	act   func()
}

func onOff(b bool) string {
	if b {
		return "SÍ"
	}
	return "NO"
}

func (g *Game) debugRows() []debugRow {
	flip := func(p *bool) func() {
		return func() {
			*p = !*p
			g.debugApply()
		}
	}
	d := &g.dbg
	guestStep := g.guestStep
	return []debugRow{
		{"Inmunidad (nada te mata)", func() string { return onOff(d.immune) }, flip(&d.immune)},
		{"Matar policías con la puñalada", func() string { return onOff(d.killCops) }, flip(&d.killCops)},
		{"Poderes de los botones (sucesos, famosos, asesino)", func() string { return onOff(d.powers) }, flip(&d.powers)},
		{"Todas las fiestas desbloqueadas", func() string { return onOff(d.levels) }, flip(&d.levels)},
		{"Toda la tienda desbloqueada", func() string { return onOff(d.shop) }, flip(&d.shop)},
		{"Trofeos", func() string { return []string{"como están", "TODOS ganados", "NINGUNO"}[d.trophyMode] }, func() { d.trophyMode = (d.trophyMode + 1) % 3; g.debugApply() }},
		{"Dar 10.000 puntos a la cartera", func() string { return fmt.Sprintf("+%d", d.bonus) }, func() { d.bonus += 10000; g.debugApply() }},
		{"Velocidad x2", func() string { return onOff(d.speed) }, flip(&d.speed)},
		{"Nadie llama a la policía", func() string { return onOff(d.noPolice) }, flip(&d.noPolice)},
		{"Invitados mínimo", func() string { return fmt.Sprintf("◄ %d ►", g.maxStart()) }, func() { guestStep(false)(1) }},
		{"Invitados máximo", func() string { return fmt.Sprintf("◄ %d ►", g.guestMax()) }, func() { guestStep(true)(1) }},
		{"Sucesos y llegadas muy rápidos", func() string { return onOff(d.fastEv) }, flip(&d.fastEv)},
		{"Policía con memoria (el testigo asustado delata)", func() string { return onOff(d.copMemory) }, flip(&d.copMemory)},
		{stealthMenuLabel, func() string { return "" }, func() { g.debugBegin(); g.startStealth(0) }},
		{stealth2pMenuLabel, func() string { return "" }, func() { g.debugBegin(); g.startStealth2P(0) }},
		{"Ir a la última fiesta: Isla Nublar (Jurassic Park)", func() string { return "" }, func() { g.startLevel(specialJurassic) }},
		{"1. Ver la intro", func() string { return "" }, func() { g.playScene(sceneIntro, ModeDebug) }},
		{"2. Ver el interludio 1 (coche, tras la plaza de toros)", func() string { return "" }, func() { g.playScene(sceneInter1, ModeDebug) }},
		{"3. Ver el interludio 2 (barco, tras la discoteca)", func() string { return "" }, func() { g.playScene(sceneInter2, ModeDebug) }},
		{"4. Ver el interludio del globo (tras la fiesta 15)", func() string { return "" }, func() { g.playScene(sceneInter15, ModeDebug) }},
		{"5. Ver el interludio 3 (cohete, tras la estación de tren)", func() string { return "" }, func() { g.playScene(sceneInter3, ModeDebug) }},
		{"6. Ver el final (sin todos los trofeos)", func() string { return "" }, func() { g.playScene(sceneEnd1, ModeDebug) }},
		{"7. Ver la invitación a la isla", func() string { return "" }, func() { g.playScene(sceneInvite, ModeDebug) }},
		{"8. Ver la llegada en helicóptero a la isla", func() string { return "" }, func() { g.playScene(sceneArrival, ModeDebug) }},
		{"9. Ver el final secreto (con todos los trofeos)", func() string { return "" }, func() { g.playScene(sceneEnd2, ModeDebug) }},
		{"Salir del DEBUG (recupera tu partida real)", func() string { return "" }, func() { g.debugEnd(); g.mode = ModeMenu }},
		{"Volver al menú", func() string { return "" }, func() { g.mode = ModeMenu }},
	}
}

// guestStep: the number of guests of a party (45 to 60 by default) is only changed from here, in
// steps of 5 between 10 and 100, and the minimum never goes above the maximum.
func (g *Game) guestStep(max bool) func(int) {
	return func(dir int) {
		d := &g.dbg
		if max {
			d.guestMax = clampI(g.guestMax()+dir*5, g.maxStart(), 100)
			d.guestMin = g.maxStart()
		} else {
			d.guestMin = clampI(g.maxStart()+dir*5, 10, 100)
			d.guestMax = maxI(g.guestMax(), d.guestMin)
		}
		g.debugApply()
	}
}

const (
	debugRowH = 15.0 // 27 rows: they must end above the status line (y=498)
	debugRow0 = 86.0
)

func (g *Game) updateDebug() {
	rows := g.debugRows()
	n := len(rows)
	click := g.mouseMenu(func(mx, my float64) int {
		for i := 0; i < n; i++ {
			if y := debugRow0 + float64(i)*debugRowH; mx >= 150 && mx <= 810 && my >= y-4 && my < y+debugRowH-4 {
				return i
			}
		}
		return -1
	}, &g.dbgSel)
	if actDown.just() {
		g.dbgSel = (g.dbgSel + 1) % n
	}
	if actUp.just() {
		g.dbgSel = (g.dbgSel + n - 1) % n
	}
	if lb := rows[g.dbgSel].label; lb == "Invitados mínimo" || lb == "Invitados máximo" {
		// numbers: left and right go down and up, Enter and click go up
		dir := 0
		switch {
		case actLeft.just():
			dir = -1
		case actRight.just() || actConfirm.just() || click:
			dir = 1
		}
		if dir != 0 {
			g.guestStep(lb == "Invitados máximo")(dir)
		}
	} else if actConfirm.just() || actLeft.just() || actRight.just() || click {
		rows[g.dbgSel].act()
	}
	if actBack.just() {
		g.mode = ModeMenu
	}
}

func (g *Game) drawDebug(dst *ebiten.Image) {
	g.drawBedroom(dst)
	fillRect(dst, 0, 0, screenW, screenH, rgba(0x000000, 190))
	fillRect(dst, 130, 28, 700, 504, rgb(0x0a0c14))
	fillRect(dst, 130, 28, 700, 2, rgb(0xffa030))
	fillRect(dst, 130, 530, 700, 2, rgb(0xffa030))
	g.drawTextC(dst, "MODO DEBUG", 480, 36, rgb(0xffa030), 2.2)
	g.drawTextC(dst, "Todo es temporal: no se guarda nada y tu partida real no cambia.", 480, 70, rgb(0xb8b0d0), 1.1)
	for i, r := range g.debugRows() {
		y := debugRow0 + float64(i)*debugRowH
		col := rgb(0x3fb59a)
		if i == g.dbgSel {
			col = rgb(0xff6a8a)
			fillRect(dst, 140, y-3, 680, debugRowH-1, rgba(0x6a4ab0, 70))
		}
		g.drawText(dst, r.label, 156, y, col, 1.3)
		if v := r.val(); v != "" {
			vc := rgb(0x8a86a0)
			if v == "SÍ" || v[0] == '+' || v == "TODOS ganados" {
				vc = rgb(0x60e090)
			} else if v == "NINGUNO" {
				vc = rgb(0xff8070)
			}
			g.drawText(dst, v, 700, y, vc, 1.3)
		}
	}
	if debugSnap != nil {
		g.drawTextC(dst, "SESIÓN DEBUG ACTIVA · nada se guarda hasta que salgas", 480, 498, color.RGBA{255, 160, 48, 255}, 1.1)
	}
	g.drawTextC(dst, "↑↓ elegir   Enter / ←→ / clic: cambiar   Esc: volver", 480, 514, rgb(0xf0e060), 1.1)
}
