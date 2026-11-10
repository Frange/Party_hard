package main

import (
	"math/rand"
	"time"
)

// ---------------------------------------------------------------------------
// EXCUSAS: a witness who has seen the player do something rotten (the "!" alert of suspect.go) sometimes
// blurts out an absurd excuse in a speech bubble instead of staying silent. No new AI: it only reads
// witnessPending and writes the guest's icon, so the bubble manager (bubbles.go) keeps the limits (max 4
// bubbles, never over the HUD, never over the player, no overlaps). Anti-spam: a rest per guest, a global
// rest, one excuse at a time, and the same excuse is not repeated in a row. Own dice, g.rng untouched.
// Max 48 characters per line.
// ---------------------------------------------------------------------------

const (
	excuseGuestCool  = 40.0 // seconds before the same guest says another one
	excuseGlobalCool = 7.0  // seconds between two excuses of anybody
	excuseRate       = 0.35 // chance per second while a witness is pending and silent
)

type excuseLine struct{ es, en string }

var excuseLines = []excuseLine{
	{"Yo solo vengo por el cáterin", "I only came for the catering"},
	{"Yo no he visto nada, soy de otra fiesta", "I saw nothing, I am from another party"},
	{"Estaba mirando el techo. Qué techo", "I was looking at the ceiling. Nice ceiling"},
	{"Eso no era un cuchillo, era un abanico", "That was no knife, it was a fan"},
	{"Perdona, tengo prosopagnosia selectiva", "Sorry, I have selective face blindness"},
	{"Yo solo paso por aquí. Nunca estuve", "I am only passing by. Never been here"},
	{"Estoy ciego del ojo izquierdo. Y del derecho", "I am blind in my left eye. And the right one"},
	{"Seguro que ha sido el ponche", "I bet it was the punch"},
	{"Eso es arte moderno, ¿no?", "That is modern art, right?"},
	{"Yo he venido a por hielo, nada más", "I came for ice, nothing else"},
	{"Es que me ha entrado una pestaña en el ojo", "An eyelash got in my eye"},
	{"No es lo que parece. Bueno, sí. Pero no", "Not what it looks like. Well, it is. But no"},
	{"Eso lo hacen todos en mi pueblo", "Everybody does that in my village"},
	{"Yo soy el de los canapés. Nadie me ve", "I am the canapé guy. Nobody sees me"},
	{"Mi cerebro ha cerrado por vacaciones", "My brain closed for the holidays"},
	{"Me voy a ir a un sitio donde no pase nada", "I am going where nothing happens"},
	{"Seguro que tiene una explicación lógica", "I bet there is a logical explanation"},
	{"Ha sido el mayordomo. Siempre es el mayordomo", "The butler did it. It is always the butler"},
	{"Yo estaba en el baño. Todo el rato", "I was in the bathroom. The whole time"},
	{"Eh... ¿hay wifi por aquí?", "Uh... is there wifi around here?"},
	{"Me llamo Pepe y no he visto nada", "My name is Bob and I saw nothing"},
	{"No he sido yo, ¡he sido yo! No, tú", "It was not me, it was me! No, you"},
	{"Yo solo hago bulto en esta fiesta", "I just bulk up this party"},
	{"Es una performance. Una performance cara", "It is a performance. An expensive one"},
	{"A mí que me registren. Pero con cariño", "Search me. But gently"},
	{"Eso no pasa en mi barrio, que es muy tranquilo", "That never happens in my quiet neighbourhood"},
	{"Yo me tapo los ojos y el problema se va", "I cover my eyes and the problem goes away"},
	{"Llevo cinco copas, mi testimonio no vale", "Five drinks in, my testimony is worthless"},
	{"Qué casualidad, yo también pasaba por aquí", "What a coincidence, I was passing by too"},
	{"Me he quedado dormido de pie. Qué sueño", "I fell asleep standing up. So sleepy"},
	{"Yo soy sordo, mudo y muy discreto", "I am deaf, mute and very discreet"},
	{"Mi abogado dice que no diga nada. Hola", "My lawyer says to say nothing. Hello"},
	{"Eso ya estaba así cuando llegué", "It was already like that when I arrived"},
	{"Era una broma. Una broma muy rara", "It was a joke. A very strange joke"},
}

var (
	excuseRng   = rand.New(rand.NewSource(time.Now().UnixNano()))
	excuseGuest = map[*Guest]float64{}
	excuseAt    float64
	excuseLast  = -1
)

// excuseReset forgets the rests (a new level restarts the clock).
func excuseReset() {
	excuseGuest = map[*Guest]float64{}
	excuseAt, excuseLast = 0, -1
}

// guestExcuse: called every frame for each guest (game.go updateGuest).
func (g *Game) guestExcuse(gu *Guest, dt float64) {
	if g.mode != ModePlay || gu.icon != "" || gu.out() || !gu.sawKiller || gu.warned {
		return
	}
	now := g.time
	if excuseAt > now { // the clock went back: new level
		excuseReset()
	}
	if now-excuseAt < excuseGlobalCool {
		return
	}
	if t, ok := excuseGuest[gu]; ok && t <= now && now-t < excuseGuestCool {
		return
	}
	pending := false
	for _, p := range g.players() {
		if p != nil && g.witnessPending(gu, p) {
			pending = true
		}
	}
	if !pending || excuseRng.Float64() >= excuseRate*dt {
		return
	}
	i := excuseRng.Intn(len(excuseLines))
	if i == excuseLast {
		i = (i + 1) % len(excuseLines)
	}
	excuseLast = i
	excuseAt = now
	if len(excuseGuest) > 200 {
		excuseGuest = map[*Guest]float64{}
	}
	excuseGuest[gu] = now
	gu.icon, gu.iconT = excuseLines[i].es, 2.6
}

func init() {
	for _, l := range excuseLines {
		if _, ok := langDataEN[l.es]; !ok {
			langDataEN[l.es] = l.en
		}
	}
}
