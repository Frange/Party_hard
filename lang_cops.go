package main

import "fmt"

// English texts of the POLICÍAS page of OBJETIVOS (goalcops.go), the escapes that open the police uniforms
// (copsets.go) and the three pages of VESTUARIO (wardrobe_kinds.go). Merged into langDataEN without overwriting anything.
func init() {
	m := map[string]string{
		"POLICÍAS":  "POLICE",
		"PERSONAJE": "CHARACTER",

		"1 estrella":  "1 star",
		"2 estrellas": "2 stars",
		"3 estrellas": "3 stars",
		"4 estrellas": "4 stars",
		"5 estrellas": "5 stars",

		"Mata a un policía local":                                 "Kill a local cop",
		"Mata a un agente de este tipo (llega con %d asesinatos)": "Kill an officer of this kind (comes at %d kills)",
		"Escapes: %d (próximo uniforme: %d)":                      "Escapes: %d (next uniform: %d)",
		"Escapes: %d (todos los uniformes)":                       "Escapes: %d (all the uniforms)",

		"Uniforme nuevo de policía":                            "New police uniform",
		"¡Has escapado de la policía!":                         "You got away from the police!",
		"Uniforme de la policía local. Se elige en VESTUARIO.": "Local police uniform. Pick it in the WARDROBE.",

		"Original":         "Original",
		"Color de la ropa": "Clothes colour",
		"Arma de todos":    "Everyone's weapon",
		"/Tab otro jugador   RePág/AvPág sección   ": "/Tab other player   PgUp/PgDn page   ",
		"RePág/AvPág sección   ":                     "PgUp/PgDn page   ",

		"Los asesinos que ya has derrotado (mira OBJETIVOS).":                                    "The killers you have already beaten (see GOALS).",
		"Tiñe la ropa de este asesino cuando salga a la fiesta.":                                 "Tints this killer's clothes when he comes to the party.",
		"El arma que llevan todos los asesinos. Solo las que ya son tuyas.":                      "The weapon every killer carries. Only the ones you already own.",
		"Cómo viste esta policía en la fiesta. Los uniformes nuevos se ganan escapando de ella.": "How this police force dresses at the party. New uniforms are won by escaping from it.",
	}
	for i := 1; i <= 4; i++ { // the milestone numbers are written into the hint of the locked cards
		m[fmt.Sprintf("Mata a un agente de este tipo (llega con %d asesinatos)", i*500)] = fmt.Sprintf("Kill an officer of this kind (comes at %d kills)", i*500)
	}
	for k, v := range m {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
}
