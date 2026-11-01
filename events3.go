package main

import "fmt"

// ---------------------------------------------------------------------------
// Rare events on demand. Some trophies give the power to set off a rare event
// with a key (rebindable, keyboard and pad): at most two per party, thirty
// seconds apart.
// ---------------------------------------------------------------------------

const (
	manualEvents   = 2
	manualEventGap = 30.0
)

// evPowerTrophies: winning any of these gives the power (it is also named in the reward text).
var evPowerTrophies = map[string]bool{
	"t_sam": true, "t_cesar": true, "t_diego": true, "t_w5": true, "t_cut10": true, "t_e5": true, "t_ufo": true, "t_rex": true, "t_ghost": true, "t_tech3": true,
}

func init() {
	for i := range trophyList {
		if evPowerTrophies[trophyList[i].id] {
			trophyList[i].reward.desc += " Además te da el poder de provocar sucesos raros (tecla en AJUSTES)."
		}
	}
}

func (g *Game) eventPower() bool {
	for id := range evPowerTrophies {
		if g.progress.Trophies[id] {
			return true
		}
	}
	return false
}

// manualEvent: the player calls for a rare event.
func (g *Game) manualEvent() {
	if g.mode != ModePlay || g.p.dead {
		return
	}
	if !g.eventPower() {
		g.say("Aún no puedes provocar sucesos: sale de ciertos trofeos")
		return
	}
	if g.evManual >= manualEvents {
		g.say("Ya has provocado 2 sucesos en esta fiesta")
		return
	}
	if wait := manualEventGap - (g.time - g.evManualAt); g.evManual > 0 && wait > 0 {
		g.say(fmt.Sprintf("Espera %d s para provocar otro suceso", int(wait)+1))
		return
	}
	before := len(g.events)
	var allowed []int
	for k := 0; k < evKinds; k++ {
		if k < evScooters || k > evPrank || ownedID(kidEventID[k]) {
			allowed = append(allowed, k)
		}
	}
	for try := 0; try < 12 && len(g.events) == before; try++ {
		g.forceEv = 1 + allowed[g.rng.Intn(len(allowed))]
		g.startEvent()
	}
	g.forceEv = 0
	if len(g.events) == before {
		g.say("No ha pasado nada... inténtalo otra vez")
		return
	}
	g.evManual++
	g.evManualAt = g.time
}
