package main

// English texts of the COMPLETADO block (completion.go), the menu titles (menutier.go) and the DEBUG rows added with
// them. Merged into langDataEN without overwriting anything.
func init() {
	m := map[string]string{
		"COMPLETADO %d%%": "COMPLETE %d%%",
		"Faltan:":         "Missing:",
		"1 fiesta":        "1 party",
		"%d fiestas":      "%d parties",
		"1 personaje":     "1 character",
		"%d personajes":   "%d characters",
		"1 VIP":           "1 VIP",
		"%d VIPs":         "%d VIPs",
		"1 asesino":       "1 killer",
		"%d asesinos":     "%d killers",
		"1 policía":       "1 cop",
		"%d policías":     "%d cops",
		"1 trofeo":        "1 trophy",
		"%d trofeos":      "%d trophies",
		"1 de tienda":     "1 from the shop",
		"%d de tienda":    "%d from the shop",

		"★ PESADILLA DEL VECINDARIO ★": "★ NEIGHBOURHOOD NIGHTMARE ★",
		"★ TERROR DE LA FIESTA ★":      "★ TERROR OF THE PARTY ★",

		"Grupos de policía desbloqueados":                 "Police groups unlocked",
		"como están":                                      "as saved",
		"Reto de hoy siempre libre (no gasta el intento)": "Daily challenge always open (no attempt spent)",
		"Estado del menú (colores y títulos)":             "Menu state (colours and titles)",
		"Intro":                                           "Intro",
		"Interludio: coche":                               "Interlude: car",
		"Interludio: barco":                               "Interlude: boat",
		"Interludio: globo":                               "Interlude: balloon",
		"Interludio: cohete":                              "Interlude: rocket",
		"Final":                                           "Ending",
		"Invitación a la isla":                            "Island invitation",
		"Llegada a la isla":                               "Island arrival",
		"Final secreto":                                   "Secret ending",
		"Álbum de OBJETIVOS (personajes, VIPs, asesinos, policías)": "GOALS album (characters, VIPs, killers, cops)",
		"todo visto":                      "all seen",
		"todo eliminado":                  "all eliminated",
		"todo conseguido":                 "all won",
		"nada":                            "nothing",
		"Dar 100.000 puntos a la cartera": "Give 100,000 points to the wallet",
		"Ver escena":                      "Watch scene",
		"como está":                       "as saved",
		"normal":                          "normal",
		"bronce":                          "bronze",
		"plata":                           "silver",
		"oro":                             "gold",
	}
	for k, v := range m {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
}
