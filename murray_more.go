package main

// More of Murray's chatter in the parties where he sits on his stake (murray.go, monkey_lines.go): demonic
// boasts, blocks and pickaxes, pirate crews and treasure. ES -> EN, <= 48 runes (the bubble).
var murrayLinesMore = [][2]string{
	{"¡Vuestras almas animarán mi fiesta eterna!", "Your souls will liven up my eternal party!"},
	{"¡Firmad aquí! Con sangre o con boli, da igual.", "Sign here! In blood or biro, I'm not fussy."},
	{"¡Muajaja! ...¿Nadie? Ni un escalofrío.", "Mwahaha! ...Nobody? Not even a shiver."},
	{"Bajo la luna mi poder crece. Mi cuerpo, no.", "Under the moon my power grows. My body doesn't."},
	{"No cavéis hacia abajo, ¿eh? Hacedme caso.", "Don't dig straight down, OK? Trust me."},
	{"¿Ese «sssss» detrás de ti? Un amigo verde.", "That \"sssss\" behind you? A green friend."},
	{"Hmm. Hmmm. Soy un aldeano demoníaco.", "Hmm. Hmmm. I'm a demonic villager."},
	{"¿Alguien ha visto mi pico de diamante?", "Has anyone seen my diamond pickaxe?"},
	{"¡Seré el Rey de las Fiestas! ¡Arrodillaos!", "I'll be King of the Parties! Kneel!"},
	{"¡Uníos a mi tripulación! Plazas: todas.", "Join my crew! Vacancies: all of them."},
	{"Mi tesoro: una estaca y un rencor eterno.", "My treasure: a stake and an eternal grudge."},
	{"Un namaka nunca deja a otro en una estaca.", "A namaka never leaves another on a stake."},
}

func init() {
	for _, p := range murrayLinesMore {
		murrayLines = append(murrayLines, p[0])
		if _, ok := langDataEN[p[0]]; !ok {
			langDataEN[p[0]] = p[1]
		}
	}
}
