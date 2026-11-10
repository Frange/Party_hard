package main

// ---------------------------------------------------------------------------
// POLICE UNIFORMS (VESTUARIO > POLICÍAS). The 108 uniforms of the old shop are now 36 sets, and they are not bought:
// they are won by ESCAPING. An escape = the cuffs mark of a player (isSuspectP, the one levelscore.go counts as
// "cuffed") goes away because the search ended (g.wanted <= 0: the witnesses are gone, the case was closed on somebody
// else, the police gave up) while the player is neither dead nor in custody. Only the end of a search counts, not
// taking a disguise on, and two players freed in the same frame count once. It is kept in Progress.Escapes
// (omitempty, never decreases).
//
// The sets come in a fixed order (copSetOrder: two local ones, then one of each other force, six times) and set n
// opens when Escapes >= escapeNeed(n): 3 escapes each for the first 24 sets, 4 for the last 12 (3, 6 ... 72, 76 ... 120).
//
// Nothing the player already had is lost: the 36 sets keep their old ids (a set bought before is "owned" through
// Progress.Owned as always) and the 72 that left the catalogue are still worn, and still listed in the wardrobe, when
// they were bought (shopOwned, wearFx and shopCatalog[shopUniforms] keep them: trophy "retired", not buyable). The
// four trophy uniforms (trophies.go) are unchanged. Opened sets are derived from Escapes, never written to Owned.
// ---------------------------------------------------------------------------

// escapeStep / escapeStepLate: escapes per set (the first 24 sets / the last 12).
const (
	escapeStep     = 3
	escapeStepLate = 4
	escapeEarly    = 24 // sets that cost escapeStep
)

// copForce: one police type. slot = the Equip key of its uniform and the slot of its uniforms.
type copForce struct {
	slot, es string
	unit     int
}

var copForces = []copForce{
	{"cop", "Policía local", unitPolice}, {"fbi", "FBI", unitFBI}, {"army", "Ejército", unitMilitary},
	{"swat", "SWAT", unitSWAT}, {"rangers", "Power Rangers", unitRangers},
}

// copSetPick: the 36 uniforms kept, per force, in the order they open inside their force.
var copSetPick = map[string][]string{
	"cop":     {"cop:guardia", "cop:bobby", "cop:miami", "cop:retro", "cop:montada", "cop:vaquero", "cop:col1", "cop:col2", "cop:payaso", "cop:pirata", "cop:ninja", "cop:rey"},
	"fbi":     {"fbi:negro", "fbi:gabardina", "fbi:hawaii", "fbi:alien", "fbi:rosa", "fbi:c5"},
	"army":    {"army:desierto", "army:nieve", "army:boinas", "army:rosa", "army:c0", "army:c5"},
	"swat":    {"swat:azul", "swat:blanco", "swat:rojo", "swat:dorado", "swat:c0", "swat:c1"},
	"rangers": {"rangers:c0", "rangers:c2", "rangers:c3", "rangers:c4", "rangers:negros", "rangers:dorados"},
}

var copSetOrderList []string

// copSetOrder: the 36 ids in the order they open (two local ones, then FBI, army, SWAT and rangers, six rounds).
func copSetOrder() []string {
	if copSetOrderList != nil {
		return copSetOrderList
	}
	next := map[string]int{}
	take := func(slot string) {
		l := copSetPick[slot]
		if next[slot] < len(l) {
			copSetOrderList = append(copSetOrderList, l[next[slot]])
			next[slot]++
		}
	}
	for r := 0; r < 6; r++ {
		take("cop")
		take("cop")
		for _, f := range copForces[1:] {
			take(f.slot)
		}
	}
	return copSetOrderList
}

// escapeNeed: the escapes the n-th set (1-based) asks for in total.
func escapeNeed(n int) int {
	if n <= escapeEarly {
		return n * escapeStep
	}
	return escapeEarly*escapeStep + (n-escapeEarly)*escapeStepLate
}

// copSetRank: the 1-based position of a uniform in the order it opens (0 = it is not one of the 36).
func copSetRank(id string) int {
	for i, s := range copSetOrder() {
		if s == id {
			return i + 1
		}
	}
	return 0
}

// copSetsOpen: how many of the 36 sets the escapes have opened.
func copSetsOpen(escapes int) int {
	n := 0
	for n < len(copSetOrder()) && escapes >= escapeNeed(n+1) {
		n++
	}
	return n
}

// copSetNext: the total of escapes the next set asks for (ok false: all are open).
func copSetNext(escapes int) (need int, ok bool) {
	n := copSetsOpen(escapes)
	if n >= len(copSetOrder()) {
		return 0, false
	}
	return escapeNeed(n + 1), true
}

// copSetSync lets the wardrobe see the sets the escapes opened (derived on every syncShop; nothing is written).
func copSetSync(escapes int) {
	for i, id := range copSetOrder() {
		if i < copSetsOpen(escapes) {
			shopOwned[id] = true
		}
	}
}

// copSetName: the name of a uniform (its id when the catalogue does not know it).
func copSetName(id string) string {
	for _, it := range shopCatalog[shopUniforms] {
		if it.id == id {
			return it.name
		}
	}
	return id
}

// cuffEscaped: the player got away. Counts it, opens the set(s) it earned and says so.
func (g *Game) cuffEscaped() {
	before := copSetsOpen(g.progress.Escapes)
	g.progress.Escapes++
	g.stat("escapes", 1)
	if after := copSetsOpen(g.progress.Escapes); after > before {
		copSetSync(g.progress.Escapes)
		g.say(tr("Uniforme nuevo de policía") + ": " + tr(copSetName(copSetOrder()[after-1])))
	} else {
		g.say("¡Has escapado de la policía!")
	}
	saveProgress(g.progress)
}

// escapeTick runs from trackCuffs for every player: wasSus/now are the cuffs mark before and after this frame.
func (g *Game) escapeTick(pl *Player, wasSus, now bool) bool {
	return wasSus && !now && pl != nil && !pl.dead && !pl.jail.held && g.wanted <= 0
}
