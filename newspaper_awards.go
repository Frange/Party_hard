package main

import "sort"

// ---------------------------------------------------------------------------
// AWARDS OF THE 2P NEWSPAPER: in two players the statistics box of each player gets a dark ribbon under its title with a
// funny prize ("PREMIO: El más sigiloso"), worked out by comparing the two players' existing statistics (paperPlayer:
// kills by cause, chain, clean kills, times seen, police warnings and the coop counters). The ribbon is ONE line that
// fits the column (a test checks both languages). It takes the place of the nickname line of the box (the "sub" row,
// statSubH tall, which 2P already reserved), so the box does not grow by a pixel: a taller box cost the 2P page its
// small news (TestPaperBriefsDistinctDeterministicAndSeeded) and the text rect (12 px) fits the row (14).
//
// Every award has a priority (the odder, the higher) and a score (how much better than the partner: a plain
// difference, already scaled so that "5 more times seen" and "3 more blows" weigh alike). Greedy like coopTitleFor:
// the best (priority, score) pair that is still free is given; the two players never get the same prize. Nothing
// is random, g.rng is untouched, and the same party prints the same prize on every frame.
// ---------------------------------------------------------------------------

type paperAward struct {
	es, en string
	prio   int                            // higher goes first
	need   float64                        // the minimum score to be given
	score  func(a, b paperPlayer) float64 // how much better a is than b
}

func awKills(p paperPlayer) int {
	return p.kills[kcKnife] + p.kills[kcPoison] + p.kills[kcTrap] + p.kills[kcBlast] + p.kills[kcNone]
}

var paperAwards = []paperAward{
	{"El más sigiloso", "The sneakiest", 5, 2, func(a, b paperPlayer) float64 { return float64(b.seen - a.seen) }},
	{"El más ruidoso", "The loudest", 5, 2, func(a, b paperPlayer) float64 { return float64(a.seen - b.seen) }},
	{"Mejor bailarín", "Best dancer", 5, 4, func(a, b paperPlayer) float64 { return (a.coop.danceT - b.coop.danceT) / 3 }},
	{"El más caótico", "The most chaotic", 6, 2, func(a, b paperPlayer) float64 { return float64(a.kills[kcBlast]*2 - b.kills[kcBlast]*2) }},
	{"Rey de la cadena", "Chain king", 4, 2, func(a, b paperPlayer) float64 { return float64(a.chainMax - b.chainMax) }},
	{"Manos limpias", "Clean hands", 4, 2, func(a, b paperPlayer) float64 { return float64(a.clean - b.clean) }},
	{"El envenenador", "The poisoner", 6, 2, func(a, b paperPlayer) float64 { return float64(a.kills[kcPoison] - b.kills[kcPoison]) }},
	{"Maestro de trampas", "Trap master", 6, 2, func(a, b paperPlayer) float64 { return float64(a.kills[kcTrap] - b.kills[kcTrap]) }},
	{"Mano dura", "Heavy hand", 5, 2, func(a, b paperPlayer) float64 { return float64(a.coop.hits - b.coop.hits) }},
	{"Siesta oficial", "Official nap", 5, 4, func(a, b paperPlayer) float64 { return (a.coop.floorT - b.coop.floorT) / 3 }},
	{"Ángel de la guarda", "Guardian angel", 6, 1, func(a, b paperPlayer) float64 {
		return float64(a.coop.helps + a.coop.distracts - b.coop.helps - b.coop.distracts)
	}},
	{"Rey del choque", "High-five king", 3, 1, func(a, b paperPlayer) float64 { return float64(a.coop.fives - b.coop.fives) }},
	{"Cliente de la poli", "Police regular", 5, 1, func(a, b paperPlayer) float64 { return float64(a.cuff - b.cuff) }},
	{"El cuchillero", "The knife artist", 3, 2, func(a, b paperPlayer) float64 {
		return float64(a.kills[kcKnife] + a.kills[kcNone] - b.kills[kcKnife] - b.kills[kcNone])
	}},
	{"MVP de la fiesta", "Party MVP", 2, 30, func(a, b paperPlayer) float64 { return float64(a.coop.points-b.coop.points) / 10 }},
	{"El más pacífico", "The most peaceful", 1, 2, func(a, b paperPlayer) float64 { return float64(awKills(b) - awKills(a)) }},
}

// the prizes nobody earned (the party was too even): one each, never repeated
var paperAwardFallback = [2][2]string{{"Alma de la fiesta", "Life of the party"}, {"Invitado de honor", "Guest of honour"}}

// awardsFor: the prize of each of the two players ({es, en}); "" when there is only one player.
func awardsFor(a, b paperPlayer) [2][2]string {
	type cand struct {
		p, i  int
		score float64
	}
	var cs []cand
	pl := [2]paperPlayer{a, b}
	for i, aw := range paperAwards {
		for p := 0; p < 2; p++ {
			if s := aw.score(pl[p], pl[1-p]); s >= aw.need {
				cs = append(cs, cand{p, i, s})
			}
		}
	}
	sort.SliceStable(cs, func(x, y int) bool {
		if pa, pb := paperAwards[cs[x].i].prio, paperAwards[cs[y].i].prio; pa != pb {
			return pa > pb
		}
		return cs[x].score > cs[y].score
	})
	var out [2][2]string
	used := map[int]bool{}
	for _, c := range cs {
		if out[c.p][0] != "" || used[c.i] {
			continue
		}
		out[c.p] = [2]string{paperAwards[c.i].es, paperAwards[c.i].en}
		used[c.i] = true
	}
	for p := 0; p < 2; p++ {
		if out[p][0] == "" {
			out[p] = paperAwardFallback[p]
		}
	}
	return out
}

// awardLine: the text of the ribbon for player i, in the language on screen.
func awardLine(a, b paperPlayer, i int) string {
	aw := awardsFor(a, b)[i]
	if uiLang == "en" {
		return "PRIZE: " + aw[1]
	}
	return "PREMIO: " + aw[0]
}
