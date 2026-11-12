package main

import (
	"fmt"
	"math/rand"
)

// ---------------------------------------------------------------------------
// RESUMEN DE RACHA: one line on the newspaper (inside the statistics box of each player) that tells the
// oddest thing the player did in the party, read from the statistics. Every rule has a rarity; the rarest
// rule that applies wins, ties are broken by a generator of its own seeded with the paper (never g.rng), so
// the same party prints the same line on every frame. Texts go in both languages here (no tr key needed).
// ---------------------------------------------------------------------------

type streakIn struct {
	k, knife, poison, traps, blast int
	chain, clean, seen, cuff, secs int
	hits, floor, best              int
	f                              paperFacts
	two                            bool
}

type streakRule struct {
	rare   int // 1 common ... 5 very odd
	ok     func(s *streakIn) bool
	es, en string                  // Sprintf formats
	args   func(s *streakIn) []any // nil = no arguments
}

func sN(f func(s *streakIn) int) func(s *streakIn) []any {
	return func(s *streakIn) []any { return []any{f(s)} }
}

var streakRules = []streakRule{
	{5, func(s *streakIn) bool { return s.k == 0 && s.seen == 0 && s.secs > 20 }, "Cero bajas, cero testigos: ¿fiesta o retiro?", "Zero kills, zero witnesses: party or retreat?", nil},
	{5, func(s *streakIn) bool { return s.k >= 5 && s.seen == 0 }, "%d bajas sin que nadie te mirara", "%d kills and nobody even looked", sN(func(s *streakIn) int { return s.k })},
	{5, func(s *streakIn) bool { return s.k >= 4 && s.clean >= s.k }, "Todas tus bajas, limpias: ni una mancha", "Every kill clean: not a single stain", nil},
	{5, func(s *streakIn) bool { return s.k >= 3 && s.knife == 0 }, "Ni un solo cuchillo: todo con estilo", "Not a single knife: all style", nil},
	{5, func(s *streakIn) bool { return s.k >= 6 && s.poison == s.k }, "Todo veneno: cocinero con mala fe", "All poison: a chef with bad intentions", nil},
	{4, func(s *streakIn) bool { return s.best >= 8 }, "Racha de %d bajas limpias: nadie sospechó", "A streak of %d clean kills: nobody suspected", sN(func(s *streakIn) int { return s.best })},
	{4, func(s *streakIn) bool { return s.chain >= 8 }, "Cadena de %d: parecía un baile de bajas", "A chain of %d: it looked like a dance of kills", sN(func(s *streakIn) int { return s.chain })},
	{4, func(s *streakIn) bool { return s.k >= 3 && s.traps*2 >= s.k }, "Las trampas hicieron el trabajo, tú mirabas", "The traps did the work, you just watched", nil},
	{4, func(s *streakIn) bool { return s.blast >= 4 }, "%d explosiones: ¿vecinos? ¿qué vecinos?", "%d explosions: neighbours? what neighbours?", sN(func(s *streakIn) int { return s.blast })},
	{4, func(s *streakIn) bool { return s.cuff >= 3 }, "Esposado %d veces: ya sabes el nombre del poli", "Cuffed %d times: you know the cop by name", sN(func(s *streakIn) int { return s.cuff })},
	{4, func(s *streakIn) bool { return s.k >= 1 && s.secs <= 120 }, "Fiesta liquidada en %d segundos: ¿prisa?", "Party wrapped up in %d seconds: in a hurry?", sN(func(s *streakIn) int { return s.secs })},
	{4, func(s *streakIn) bool { return s.k == 1 && s.secs > 60 }, "Una sola baja: puntería o timidez", "A single kill: fine aim or plain shyness", nil},
	{4, func(s *streakIn) bool { return s.k >= 1 && s.seen == 0 }, "Nadie te vio: ni tu sombra te delató", "Nobody saw you: not even your shadow told", nil},
	{4, func(s *streakIn) bool { return s.f.animalsDead >= 3 }, "%d animales menos: el zoo pide explicaciones", "%d animals fewer: the zoo wants answers", func(s *streakIn) []any { return []any{s.f.animalsDead} }},
	{4, func(s *streakIn) bool { return s.two && s.hits >= 5 }, "%d golpes a tu compañero: amor con mano dura", "%d blows to your partner: tough love", sN(func(s *streakIn) int { return s.hits })},
	{3, func(s *streakIn) bool { return s.best >= 5 }, "Racha de %d sin testigos: el sigilo da puntos", "A streak of %d unseen: stealth pays", sN(func(s *streakIn) int { return s.best })},
	{3, func(s *streakIn) bool { return s.chain >= 5 }, "Cadena de %d bajas seguidas: ¡qué ritmo!", "A chain of %d kills in a row: what a rhythm!", sN(func(s *streakIn) int { return s.chain })},
	{3, func(s *streakIn) bool { return s.clean >= 5 }, "%d bajas limpias: la lejía te adora", "%d clean kills: bleach adores you", sN(func(s *streakIn) int { return s.clean })},
	{3, func(s *streakIn) bool { return s.seen >= 10 }, "Te vieron %d veces: discreción, esa desconocida", "Seen %d times: discretion, that stranger", sN(func(s *streakIn) int { return s.seen })},
	{3, func(s *streakIn) bool { return s.k >= 20 }, "%d bajas: la fiesta se queda sin gente", "%d kills: the party is running out of guests", sN(func(s *streakIn) int { return s.k })},
	{3, func(s *streakIn) bool { return s.poison >= 5 }, "%d envenenados: revisad el ponche", "%d poisoned: somebody check the punch", sN(func(s *streakIn) int { return s.poison })},
	{3, func(s *streakIn) bool { return s.traps >= 5 }, "%d víctimas de trampas: el suelo juega a tu favor", "%d trap victims: the floor plays on your side", sN(func(s *streakIn) int { return s.traps })},
	{3, func(s *streakIn) bool { return s.k >= 2 && s.seen > s.k }, "Más testigos que bajas: tu fama te precede", "More witnesses than kills: your fame precedes you", nil},
	{3, func(s *streakIn) bool { return s.f.slasher }, "Otro asesino en la fiesta, y tú tan fresco", "Another killer at the party and you stayed cool", nil},
	{3, func(s *streakIn) bool { return s.f.camera }, "Una cámara te grabó: sonríe, sales en la tele", "A camera filmed you: smile, you are on TV", nil},
	{3, func(s *streakIn) bool { return s.two && s.floor >= 20 }, "%d s en el suelo: siesta reglamentaria", "%d s on the floor: a proper nap", sN(func(s *streakIn) int { return s.floor })},
	{3, func(s *streakIn) bool { return s.f.jailed }, "Pasaste por el calabozo y volviste a la fiesta", "You visited the cell and came back to the party", nil},
	{2, func(s *streakIn) bool { return s.k >= 10 && s.knife == s.k }, "%d cuchilladas: el cuchillo pide vacaciones", "%d stabs: the knife wants a holiday", sN(func(s *streakIn) int { return s.k })},
	{2, func(s *streakIn) bool { return s.secs >= 600 }, "%d minutos de fiesta: casi un turno de oficina", "%d minutes of party: almost an office shift", sN(func(s *streakIn) int { return s.secs / 60 })},
	{2, func(s *streakIn) bool { return s.f.medic }, "Vino la ambulancia y no era para ti", "The ambulance came and it was not for you", nil},
	{2, func(s *streakIn) bool { return s.k >= 5 && s.f.animalsAlive > 0 && s.f.animalsDead == 0 }, "Ni un animal herido: el lado tierno del crimen", "Not one animal hurt: the tender side of crime", nil},
	{2, func(s *streakIn) bool { return s.f.challenge && s.k >= 3 }, "Con el RETO puesto y sin pestañear", "With the CHALLENGE on and without blinking", nil},
	{1, func(s *streakIn) bool { return s.k >= 1 && s.knife == s.k }, "Clásico: cuchillo, sigilo y buen humor", "A classic: knife, stealth and good humour", nil},
	{0, func(s *streakIn) bool { return true }, "Una noche más en la oficina del crimen", "Another night at the crime office", nil},
}

// streakFor picks the line for one player (index i) of a party. Deterministic for the same seed.
func streakFor(f paperFacts, p paperPlayer, i int, secs float64) string {
	s := &streakIn{k: 0, knife: p.kills[kcKnife] + p.kills[kcNone], poison: p.kills[kcPoison], traps: p.kills[kcTrap], blast: p.kills[kcBlast],
		chain: p.chainMax, clean: p.clean, seen: p.seen, cuff: p.cuff, secs: int(secs), hits: p.coop.hits, floor: int(p.coop.floorT + 0.5), best: p.streakBest, f: f, two: f.two}
	s.k = s.knife + s.poison + s.traps + s.blast
	best := -1
	var cand []*streakRule
	for ri := range streakRules {
		r := &streakRules[ri]
		if !r.ok(s) {
			continue
		}
		if r.rare > best {
			best, cand = r.rare, cand[:0]
		}
		if r.rare == best {
			cand = append(cand, r)
		}
	}
	rg := rand.New(rand.NewSource(f.seed*7919 + int64(i)*104729 + 17)) // its own dice: g.rng is untouched
	r := cand[rg.Intn(len(cand))]
	format := r.es
	if uiLang == "en" {
		format = r.en
	}
	if r.args == nil {
		return format
	}
	return fmt.Sprintf(format, r.args(s)...)
}
