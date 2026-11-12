package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// LEVEL HINT (idea 56). Right after the start marker (the blinking floor) a plate of one or two lines appears at the
// bottom centre for lhDur seconds: the goal of the party plus a tip. Line 1 comes from a table by theme of the level
// (its icon key, or its title for the second rolls of the random mode); line 2 is taken from REAL data of the level, in
// this order: VIP zone, security cameras, traps, poison bottle, disguise bag; with none of those the theme's own tip.
//
//   - It closes with any action key (hit / use / dance, either player) or by time. Never in STEALTH, never in the
//     endings, interludes or menus (those are other game modes; the plate is drawn from drawHUD in ModePlay only).
//   - It sits above the control strips of the first seconds (y = screenH-70 bottom) and in the middle, so it does not
//     touch the HUD panel (top left), the "?" button (top right), the cameras of the level or the minimap (bottom right).
//   - levelHintOn=false (flag -nohint) turns it off. There is no DEBUG menu row: that menu is full (27 rows).
// ---------------------------------------------------------------------------

const (
	lhDur    = 4.8 // seconds on screen, counted from the end of the start marker
	lhBottom = 70.0
	lhLineH  = 12.0
)

// lhActionFn: an action key was just pressed (hit, use or dance, either player); the tests replace it.
var lhActionFn = func() bool {
	return actAttack.justEither() || actInteract.justEither() || actDance.justEither()
}

// levelHintOn: the switch (true by default). The flag -nohint clears it.
var levelHintOn = true

type levelHint struct {
	lines  []string
	t      float64
	closed bool
	// what the last frame drew (tests)
	drawn bool
	rect  [4]float64
}

type hintTheme struct{ goal, tip string }

// hintThemes: by icon key (campaign.go / levelIconColors). Every line is at most 48 characters.
var hintThemes = map[string]hintTheme{
	"house":    {"Fiesta en casa: que no quede ni uno en pie", "Los armarios y arbustos son buenos escondites"},
	"beach":    {"Fiesta en la playa: la arena no cuenta nada", "Cuidado con los socorristas, siempre miran"},
	"neon":     {"Noche de neón: apaga a todos los invitados", "Con tanta luz, mejor ve por las sombras"},
	"ranch":    {"Rancho en fiesta: vacíalo antes del amanecer", "Los animales no te delatan, los vaqueros sí"},
	"bull":     {"Plaza de toros: hoy el toro eres tú", "El toro no distingue entre invitados y tú"},
	"eiffel":   {"París de noche: sin testigos ni baguettes", "Los mimos callan, pero los gendarmes no"},
	"boat":     {"Crucero nocturno: aquí nadie oye nada", "Los guardias hacen ronda: vigila la cubierta"},
	"funfair":  {"Feria: ni noria ni algodón, solo tú y ellos", "Entre atracciones hay buenos rincones ciegos"},
	"stadium":  {"Estadio: deja el campo sin hinchas", "El público grita, pero nadie mira lo bajo"},
	"disco":    {"Discoteca: haz callar la pista", "El DJ aguanta el ritmo; tú, el cuchillo"},
	"festival": {"Festival: que la última canción sea la tuya", "Entre el público es fácil pasar desapercibido"},
	"skate":    {"Skate park: nadie baja de la rampa vivo", "Los patinadores corren más: no los persigas"},
	"grave":    {"Cementerio: aquí ya hay sitio de sobra", "Los muertos no hablan; los vivos, sí"},
	"cinema":   {"Cine: la última sesión no tiene final feliz", "En la oscuridad nadie ve, salvo las cámaras"},
	"circuit":  {"Circuito: ni una vuelta más para nadie", "Los coches cruzan la pista: no te atropellen"},
	"snow":     {"Estación de esquí: la nieve lo tapa todo", "El hielo y la gente no se llevan bien contigo"},
	"roof":     {"Azotea: vacía la fiesta de las alturas", "Desde aquí arriba se ve quién te mira"},
	"casino":   {"Casino: la banca siempre gana, y tú también", "Cuidado con las cámaras y los de seguridad"},
	"highway":  {"Autopista: cuidado con el tráfico, y con ellos", "Cruza solo cuando no pase ningún coche"},
	"train":    {"Estación de tren: nadie coge el último tren", "Los vigilantes hacen ronda: mide tus pasos"},
	"station":  {"Estación espacial: en el espacio nadie te oye", "Con poca gravedad los nervios, no la puntería"},
	"pub":      {"Pub: la última ronda corre de tu cuenta", "El camarero lo ve todo: cuida tus pasos"},
	"plaza":    {"Plaza europea: vacía el paseo sin hacer ruido", "La gente cae bien en las esquinas de la plaza"},
	"market":   {"Mercado: aquí todo está a buen precio", "Los puestos tapan la vista: úsalos"},
	"mall":     {"Centro comercial: liquidación de invitados", "Muchas cámaras y mucha gente: mide bien"},
}

// hintGeneric: for a level without an entry in the table.
var hintGeneric = hintTheme{"Acaba con todos los invitados de la fiesta", "Que nadie te vea y no habrá policía"}

// Tips from the data of the level (line 2).
const (
	hintVIP      = "Hay un VIP en la sala del fondo"
	hintCams     = "¡Ojo con las cámaras! Graban tus crímenes"
	hintTraps    = "Hay trampas por la sala: arma una y espera"
	hintPoison   = "Hay veneno cerca: un trago y adiós"
	hintDisguise = "Un disfraz te hace pasar por uno más"
)

// hintThemeOf finds the theme key of a level: its icon, or the one of the campaign level of the same title.
func hintThemeOf(def *LevelDef) string {
	if def == nil {
		return ""
	}
	if def.Icon != "" {
		return def.Icon
	}
	for _, e := range campaignLevels {
		if e.kind != "" && e.title == def.Title {
			return e.icon
		}
	}
	switch def.Title {
	case "PUB":
		return "pub"
	case "PLAZA EUROPEA":
		return "plaza"
	case "CENTRO COMERCIAL":
		return "mall"
	case "HOUSE PARTY":
		return "house"
	case "MIAMI BEACH":
		return "beach"
	}
	return ""
}

// levelHintLines picks the lines of a level (1 or 2). It reads the real data: VIP zone, cameras, traps, items.
func (g *Game) levelHintLines() []string {
	if g.lvl == nil || g.lvl.Def == nil {
		return nil
	}
	th, ok := hintThemes[hintThemeOf(g.lvl.Def)]
	if !ok {
		th = hintGeneric
	}
	tip := th.tip
	var traps, poison, disguise bool
	for _, pr := range g.lvl.Props {
		if pr == nil || pr.K == nil || pr.gone {
			continue
		}
		switch {
		case trapProp(pr):
			traps = true
		case pr.K.Item == "poison":
			poison = true
		case pr.K.Item == "disguise":
			disguise = true
		}
	}
	switch {
	case g.lvl.vip != nil:
		tip = hintVIP
	case len(g.cams) > 0:
		tip = hintCams
	case traps:
		tip = hintTraps
	case poison:
		tip = hintPoison
	case disguise:
		tip = hintDisguise
	}
	return []string{th.goal, tip}
}

func (h *levelHint) reset(g *Game) {
	*h = levelHint{lines: g.levelHintLines()}
	if m := g.medalHintLine(); m != "" && len(h.lines) > 0 { // the points of the medals (medals.go), a third line
		h.lines = append(h.lines, m)
	}
}

// active: the plate is on screen right now.
func (g *Game) levelHintActive() bool {
	h := &g.lh
	return levelHintOn && !h.closed && len(h.lines) > 0 && h.t < lhDur &&
		g.mode == ModePlay && !g.stealth && g.lvl != nil && !g.sb.active()
}

// updateLevelHint counts the time once the start marker is gone and closes the plate with any action key.
func (g *Game) updateLevelHint(dt float64) {
	h := &g.lh
	if h.closed || g.stealth || !levelHintOn || g.sb.active() {
		return
	}
	h.t += dt
	if lhActionFn() {
		h.closed = true
	}
}

// levelHintShown: the lines on the plate now. The plate stays two lines tall (taller, it covered the players at the
// start): with the medals line of a campaign party (medals.go) the second line shows the tip for the first half of
// the time and the points of the medals for the second half.
func (g *Game) levelHintShown() []string {
	h := &g.lh
	if len(h.lines) <= 2 {
		return h.lines
	}
	if h.t < lhDur/2 {
		return h.lines[:2]
	}
	return []string{h.lines[0], h.lines[2]}
}

// drawLevelHint paints the plate. Called from drawHUD (ModePlay only).
func (g *Game) drawLevelHint(dst *ebiten.Image) {
	h := &g.lh
	h.drawn = false
	if !g.levelHintActive() {
		return
	}
	// it slides in and out by whole pixels
	slide := 0.0
	if h.t < 0.25 {
		slide = math.Round((0.25 - h.t) / 0.25 * 24)
	} else if left := lhDur - h.t; left < 0.25 {
		slide = math.Round((0.25 - left) / 0.25 * 24)
	}
	w := 0.0
	for _, l := range h.lines {
		w = math.Max(w, g.textW(l, 1))
	}
	w = math.Round(w) + 24 // the widest of ALL the lines: the plate keeps its size when line 2 changes
	shown := g.levelHintShown()
	ph := lhLineH*float64(len(shown)) + 8
	x, y := math.Round(screenW/2-w/2), screenH-lhBottom-ph+slide
	g.panel(dst, x, y, w, ph)
	for i, l := range shown {
		col := rgb(0xf0e060)
		if i > 0 {
			col = rgb(0xe8e4f4)
		}
		if i == 1 && len(h.lines) > 2 && l == h.lines[2] { // the medals line
			col = rgb(0xe8b860)
		}
		g.drawTextC(dst, l, screenW/2, y+4+lhLineH*float64(i), col, 1)
	}
	h.drawn, h.rect = true, [4]float64{x, y, w, ph}
}

func init() {
	for es, en := range map[string]string{
		"Fiesta en casa: que no quede ni uno en pie":     "House party: leave nobody standing",
		"Los armarios y arbustos son buenos escondites":  "Wardrobes and bushes make good hiding spots",
		"Fiesta en la playa: la arena no cuenta nada":    "Beach party: sand tells no tales",
		"Cuidado con los socorristas, siempre miran":     "Mind the lifeguards, they always watch",
		"Noche de neón: apaga a todos los invitados":     "Neon night: switch every guest off",
		"Con tanta luz, mejor ve por las sombras":        "With so much light, stick to the shadows",
		"Rancho en fiesta: vacíalo antes del amanecer":   "Ranch party: empty it before sunrise",
		"Los animales no te delatan, los vaqueros sí":    "Animals will not tell on you, cowboys will",
		"Plaza de toros: hoy el toro eres tú":            "Bullring: today you are the bull",
		"El toro no distingue entre invitados y tú":      "The bull cannot tell guests from you",
		"París de noche: sin testigos ni baguettes":      "Paris by night: no witnesses, no baguettes",
		"Los mimos callan, pero los gendarmes no":        "Mimes keep quiet, gendarmes do not",
		"Crucero nocturno: aquí nadie oye nada":          "Night cruise: nobody hears a thing out here",
		"Los guardias hacen ronda: vigila la cubierta":   "Guards patrol: watch the deck",
		"Feria: ni noria ni algodón, solo tú y ellos":    "Funfair: no wheel, no candy, just you and them",
		"Entre atracciones hay buenos rincones ciegos":   "The rides leave plenty of blind corners",
		"Estadio: deja el campo sin hinchas":             "Stadium: leave the pitch without fans",
		"El público grita, pero nadie mira lo bajo":      "The crowd shouts, but nobody looks down",
		"Discoteca: haz callar la pista":                 "Disco: shut the dance floor up",
		"El DJ aguanta el ritmo; tú, el cuchillo":        "The DJ holds the beat; you hold the knife",
		"Festival: que la última canción sea la tuya":    "Festival: make the last song yours",
		"Entre el público es fácil pasar desapercibido":  "In the crowd it is easy to go unnoticed",
		"Skate park: nadie baja de la rampa vivo":        "Skate park: nobody rolls down alive",
		"Los patinadores corren más: no los persigas":    "Skaters are faster: do not chase them",
		"Cementerio: aquí ya hay sitio de sobra":         "Graveyard: plenty of room already",
		"Los muertos no hablan; los vivos, sí":           "The dead do not talk; the living do",
		"Cine: la última sesión no tiene final feliz":    "Cinema: the last show has no happy ending",
		"En la oscuridad nadie ve, salvo las cámaras":    "Nobody sees in the dark, except the cameras",
		"Circuito: ni una vuelta más para nadie":         "Circuit: not one more lap for anybody",
		"Los coches cruzan la pista: no te atropellen":   "Cars cross the track: do not get run over",
		"Estación de esquí: la nieve lo tapa todo":       "Ski resort: snow covers everything",
		"El hielo y la gente no se llevan bien contigo":  "Ice and people do not get along with you",
		"Azotea: vacía la fiesta de las alturas":         "Rooftop: clear the party up high",
		"Desde aquí arriba se ve quién te mira":          "From up here you see who is watching",
		"Casino: la banca siempre gana, y tú también":    "Casino: the house always wins, and so do you",
		"Cuidado con las cámaras y los de seguridad":     "Mind the cameras and the security guys",
		"Autopista: cuidado con el tráfico, y con ellos": "Motorway: watch the traffic, and them",
		"Cruza solo cuando no pase ningún coche":         "Cross only when no car is coming",
		"Estación de tren: nadie coge el último tren":    "Train station: nobody catches the last train",
		"Los vigilantes hacen ronda: mide tus pasos":     "Watchmen patrol: count your steps",
		"Estación espacial: en el espacio nadie te oye":  "Space station: in space nobody hears you",
		"Con poca gravedad los nervios, no la puntería":  "Low gravity shakes the nerves, not the aim",
		"Pub: la última ronda corre de tu cuenta":        "Pub: the last round is on you",
		"El camarero lo ve todo: cuida tus pasos":        "The barman sees everything: watch your steps",
		"Plaza europea: vacía el paseo sin hacer ruido":  "European square: empty the promenade quietly",
		"La gente cae bien en las esquinas de la plaza":  "People drop nicely in the corners of the square",
		"Mercado: aquí todo está a buen precio":          "Market: everything is a bargain here",
		"Los puestos tapan la vista: úsalos":             "Stalls block the view: use them",
		"Centro comercial: liquidación de invitados":     "Mall: guest clearance sale",
		"Muchas cámaras y mucha gente: mide bien":        "Many cameras and many people: be careful",
		"Acaba con todos los invitados de la fiesta":     "Finish off every guest at the party",
		"Que nadie te vea y no habrá policía":            "Let nobody see you and there will be no police",
		hintVIP:                                          "There is a VIP in the back room",
		hintCams:                                         "Watch the cameras! They record your crimes",
		hintTraps:                                        "There are traps in the room: arm one and wait",
		hintPoison:                                       "Poison nearby: one sip and goodbye",
		hintDisguise:                                     "A disguise makes you look like one of them",
	} {
		if _, ok := langDataEN[es]; !ok {
			langDataEN[es] = en
		}
	}
}
