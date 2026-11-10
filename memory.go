package main

import "math/rand"

// ---------------------------------------------------------------------------
// MEMORY OF THE GUESTS: if somebody SAW you do something (a kill: Guest.sawKiller) and SURVIVED the party, in the NEXT
// party of the SAME level (a retry, or coming back later) two or three of the guests recognise you with a joke in a
// speech bubble ("You again!") when you come near. No game consequence at all: no fright, no blame, no cuffs.
//
// DECISION: session memory only (not in Progress): nothing is written to progress.json, so there is no save
// format to migrate and no test can touch the real game. It is forgotten when the game is closed, which is fine for
// a joke. memRemember runs at the start of loadLevel, when the guests of the party that just ended are still there:
// it needs the party to have run for 20 s (a level loaded only to show it does not erase the memory).
//
// The comments are a REACTIVE party event ("memory": no dice; Eligible = "this level is remembered and has not been
// done yet"): the frame calls Start every frame while it holds, and it only holds once per party.
// ---------------------------------------------------------------------------

const (
	memMinTime = 20.0
	memNear    = 80.0
	memGap     = 7.0
	memMaxTime = 480.0
)

var memLines = pevTexts(
	"¡Tú otra vez!", "Yo a ti te conozco...", "¿No eras tú el de la otra vez?", "Ese es el que me dio el susto.",
	"Mira quién ha vuelto.", "Esta vez no me pillas.", "Me suenas de una pesadilla.", "Tú eres el de las malas pulgas.",
	"¡Madre mía, otra vez este!", "Te voy a vigilar, ¿eh?", "Esta vez sí he visto algo.", "Aquí viene el problemático.",
	"No me mires así, te recuerdo.", "Me debes una disculpa y un sofá.", "¡Eh, tú! Sí, tú. Te vi.",
	"Cuidado con ese, ya sé cómo es.", "Qué cara más familiar y poco amistosa.", "Esta vez me quedo junto a la salida.",
	"¿Otra fiesta, otro lío? Claro.", "No te acerques a mi vaso, por si acaso.", "Te tengo fichado. Con cariño.",
	"Sabía que volverías. Qué miedo.", "¿Vienes a por la revancha?", "Yo no olvido una cara. Ni una cuchilla.")

// memState: what the session remembers (levelIdx -> a witness survived) and the plan of the party in play.
var memState struct {
	seen  map[int]bool
	lvl   *Level // the party whose comments are already planned
	pick  []*Guest
	spoke map[*Guest]bool
	last  float64
}

// memRemember: called at the start of loadLevel, while the guests of the party that ended are still there.
func (g *Game) memRemember() {
	if g.lvl == nil || len(g.guests) == 0 || g.time < memMinTime || g.levelIdx < 0 {
		return
	}
	n := 0
	for _, gu := range g.guests {
		if gu.sawKiller && !gu.out() && gu.py == nil && !gu.tech {
			n++
		}
	}
	memSet(g.levelIdx, n > 0)
}

func memSet(idx int, v bool) {
	if memState.seen == nil {
		memState.seen = map[int]bool{}
	}
	if v {
		memState.seen[idx] = true
	} else {
		delete(memState.seen, idx)
	}
}

func init() {
	registerPartyEvent(&partyEvent{ID: "memory", Name: "Guests remember you", Group: "memory", Reactive: true, DailyOK: true,
		Prob: 1, AtMin: 0, AtMax: 1, Dur: memMaxTime,
		Eligible: func(g *Game, t partyTraits) bool {
			return memState.seen[g.levelIdx] && memState.lvl != g.lvl && len(g.guests) > 0
		},
		Start: memStart, Update: memUpdate})
}

func memStart(g *Game, a *partyAct) bool {
	memState.lvl = g.lvl
	memState.spoke = map[*Guest]bool{}
	memState.pick, memState.last = nil, g.time
	r := partyDice()
	var cand []*Guest
	for _, gu := range g.guests {
		if !gu.out() && gu.perf == "" && gu.py == nil && !gu.tech && gu.quirk == "" && !starSlugs[guestSlug(gu)] {
			cand = append(cand, gu)
		}
	}
	if len(cand) < 3 {
		return false
	}
	r.Shuffle(len(cand), func(i, j int) { cand[i], cand[j] = cand[j], cand[i] })
	memState.pick = cand[:2+r.Intn(2)]
	return true
}

// memCanSay: the guest can speak now (not busy, no other bubble on him).
func memCanSay(g *Game, gu *Guest) bool {
	if gu.out() || gu.icon != "" || !g.canWatch(gu) {
		return false
	}
	switch gu.state {
	case gIdle, gGoto, gDance, gLounge, gDrink:
		return true
	}
	return false
}

func memUpdate(g *Game, a *partyAct, dt float64) bool {
	left := 0
	for _, gu := range memState.pick {
		if !gu.out() && !memState.spoke[gu] {
			left++
		}
	}
	if left == 0 || g.time-a.t0 > memMaxTime {
		return true
	}
	if g.time-memState.last < memGap {
		return false
	}
	for _, gu := range memState.pick {
		if memState.spoke[gu] || !memCanSay(g, gu) {
			continue
		}
		for _, pl := range g.players() {
			if pl != nil && !pl.dead && dist(pl.pos, gu.pos) < memNear && g.lvl.LOS(gu.pos, pl.pos) {
				memSay(gu, partyDice())
				memState.spoke[gu] = true
				memState.last = g.time
				return false
			}
		}
	}
	return false
}

func memSay(gu *Guest, r *rand.Rand) {
	gu.icon, gu.iconT = pevPick(r, memLines), 3.2
}
