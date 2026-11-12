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
	album                    int  // 0 as saved, 1 all seen, 2 all eliminated, 3 all won, 4 none (debugApply)
	dailyFree                bool // RETO DE HOY always open (daily.go dailyState); nothing is saved
	tier                     int  // menu colour state shown: 0 as the save says, 1..4 = normal, bronze, silver, gold (menutier.go)
	copMemory                bool // police with memory: a scared witness tells on you when he meets an officer (copmemory.go)
}

var debugSnap *Progress

// debugPrevFrozen: progressFrozen as it was when the DEBUG menu mode began (restored by debugEnd).
var debugPrevFrozen bool

// debugPrevGroups: dbgPoliceGroups (the -unlocked flag) as it was when the session began (restored by debugEnd).
var debugPrevGroups int

func (g *Game) debugBegin() {
	if debugSnap == nil {
		c := cloneProgress(g.progress)
		debugSnap = &c
		debugPrevFrozen = progressFrozen
		debugPrevGroups = dbgPoliceGroups
		progressFrozen = true
	}
}

// debugApply rebuilds the in-memory progress from the real one plus the active switches.
func (g *Game) debugApply() {
	g.debugBegin()
	p := cloneProgress(*debugSnap)
	p.MenuTier = 0 // the DEBUG switches show the tier they give, not the one kept by the real save (medals.go)
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
	debugAlbum(&p, d.album)
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

// debugAlbumNames: the states of the row "Álbum" (OBJETIVOS: PERSONAJES, VIPS, ASESINOS, POLICÍAS).
var debugAlbumNames = []string{"como está", "todo visto", "todo eliminado", "todo conseguido", "nada"}

// debugAlbum rewrites the cards of a COPY of the progress (never the real one): 1 seen, 2 eliminated once,
// 3 won (characters at charKillsNeed kills, VIPs killed, killers and cops beaten), 4 nothing.
func debugAlbum(p *Progress, mode int) {
	if mode == 0 {
		return
	}
	p.Album = map[string]int{}
	p.CharKills = map[string]int{}
	if mode == 4 {
		return
	}
	for _, c := range albumCards() {
		switch mode {
		case 1:
			p.Album[c.key] = albumSeen
		case 2:
			p.Album[c.key] = albumSeen | albumElim
			if c.kind() == cardChar {
				p.CharKills[c.key] = 1
			} else {
				p.Album[c.key] = albumSeen // only the characters have an ELIMINADO step
			}
		case 3:
			p.Album[c.key] = albumSeen | albumElim | albumKilled
			if c.kind() == cardVIP {
				p.Album[c.key] |= albumHelped
			}
			if c.kind() == cardChar {
				p.CharKills[c.key] = charKillsNeed
			}
			if c.kind() == cardCop && c.roleKind > unitPolice { // a police card needs its type open (goalcops.go)
				p.Owned = append(p.Owned, fmt.Sprintf("police:%d", c.roleKind))
			}
		}
	}
}

// debugEnd puts the real progress back and closes the session.
func (g *Game) debugEnd() {
	if debugSnap != nil {
		g.progress = *debugSnap
		debugSnap = nil
	}
	progressFrozen = debugPrevFrozen
	debugPrevFrozen = false
	dbgPoliceGroups = debugPrevGroups
	debugSceneSel = 0
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
	l := []int{menuContinue, menuNewGame, menuPlayers, menuShop, menuTrophies, menuScenes, menuSettings, menuCredits, menuQuit}
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
		return "OBJETIVOS"
	case menuScenes:
		return "ESCENAS"
	case menuSettings:
		return "AJUSTES"
	case menuCredits:
		return "CRÉDITOS"
	}
	return "SALIR"
}

func menuRowY(i, n int) float64 {
	if n > 9 { // DEBUG with the save: ten rows, the last one must stay above the footer texts (y=474)
		return 150 + float64(i)*31
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

// debugSteps: rows with ◄ ► values (label -> step), filled by debugRows. Left / right change the value of these rows;
// Enter and click run act.
var debugSteps = map[string]func(dir int){}

func stepRow(label string, val func() string, act func(), step func(int)) debugRow {
	debugSteps[label] = step
	return debugRow{label, val, act}
}

// debugScene: one cinematic the DEBUG menu can play. ADD ONE LINE HERE for a new scene (scenes.go has the constants;
// TestDebugScenesCoverAll fails if a scene constant is missing).
type debugScene struct {
	label string
	scene int
}

var debugScenes = []debugScene{
	{"Intro", sceneIntro},
	{"Interludio: coche", sceneInter1},
	{"Interludio: barco", sceneInter2},
	{"Interludio: globo", sceneInter15},
	{"Interludio: cohete", sceneInter3},
	{"Final", sceneEnd1},
	{"Invitación a la isla", sceneInvite},
	{"Llegada a la isla", sceneArrival},
	{"Final secreto", sceneEnd2},
}

var debugSceneSel int // the scene the row "Ver escena" points at

func debugSceneStep(dir int) {
	debugSceneSel = (debugSceneSel + dir + len(debugScenes)) % len(debugScenes)
}

var debugTierNames = []string{"como está", "normal", "bronce", "plata", "oro"}

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
		{"Dar 100.000 puntos a la cartera", func() string { return "" }, func() { d.bonus += 100000; g.debugApply() }},
		{"Velocidad x2", func() string { return onOff(d.speed) }, flip(&d.speed)},
		{"Nadie llama a la policía", func() string { return onOff(d.noPolice) }, flip(&d.noPolice)},
		{"Invitados mínimo", func() string { return fmt.Sprintf("◄ %d ►", g.maxStart()) }, func() { guestStep(false)(1) }},
		{"Invitados máximo", func() string { return fmt.Sprintf("◄ %d ►", g.guestMax()) }, func() { guestStep(true)(1) }},
		{"Sucesos y llegadas muy rápidos", func() string { return onOff(d.fastEv) }, flip(&d.fastEv)},
		{"Policía con memoria (el testigo asustado delata)", func() string { return onOff(d.copMemory) }, flip(&d.copMemory)},
		stepRow("Grupos de policía desbloqueados", func() string {
			if dbgPoliceGroups == 0 {
				return "◄ " + tr("como están") + " ►"
			}
			return fmt.Sprintf("◄ %d ►", dbgPoliceGroups)
		}, func() {}, func(dir int) { // the same variable as the flag -unlocked N (police_groups.go); never the real save
			g.debugBegin()
			dbgPoliceGroups = clampI(dbgPoliceGroups+dir, 0, len(groupStat))
		}),
		{"Reto de hoy siempre libre (no gasta el intento)", func() string { return onOff(d.dailyFree) }, flip(&d.dailyFree)},
		stepRow("Álbum de OBJETIVOS (personajes, VIPs, asesinos, policías)", func() string { return "◄ " + tr(debugAlbumNames[d.album]) + " ►" }, func() {}, func(dir int) {
			d.album = (d.album + dir + len(debugAlbumNames)) % len(debugAlbumNames)
			g.debugApply()
		}),
		stepRow("Estado del menú (colores y títulos)", func() string { return "◄ " + tr(debugTierNames[d.tier]) + " ►" }, func() {}, func(dir int) {
			d.tier = (d.tier + dir + len(debugTierNames)) % len(debugTierNames)
		}),
		{stealthMenuLabel, func() string { return "" }, func() { g.debugBegin(); g.startStealth(0) }},
		{stealth2pMenuLabel, func() string { return "" }, func() { g.debugBegin(); g.startStealth2P(0) }},
		{"Ir a la última fiesta: Isla Nublada (Parque Fiestásico)", func() string { return "" }, func() { g.startLevel(specialJurassic) }},
		stepRow("Ver escena", func() string {
			return fmt.Sprintf("◄ %d/%d %s ►", debugSceneSel+1, len(debugScenes), tr(debugScenes[debugSceneSel].label))
		}, func() { g.playScene(debugScenes[debugSceneSel].scene, ModeDebug) }, debugSceneStep),
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
	debugRowH = 16.0  // the rows end >= 12 px above the status line (y=498)
	debugRow0 = 100.0 // >= 12 px under the subtitle (y=70)
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
	if st := debugSteps[rows[g.dbgSel].label]; st != nil {
		dir := 0
		switch {
		case actLeft.just():
			dir = -1
		case actRight.just():
			dir = 1
		}
		if dir != 0 {
			st(dir)
		}
		if actConfirm.just() || click {
			rows[g.dbgSel].act()
		}
	} else if lb := rows[g.dbgSel].label; lb == "Invitados mínimo" || lb == "Invitados máximo" {
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
	fillRect(dst, 132, 28, 696, 504, rgb(0x0a0c14)) // 12 px from the ATRÁS button
	fillRect(dst, 132, 28, 696, 2, rgb(0xffa030))
	fillRect(dst, 132, 530, 696, 2, rgb(0xffa030))
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
}
