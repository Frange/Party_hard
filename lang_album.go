package main

// English texts of the card album and the secret trophies (album.go). Merged into langDataEN without overwriting anything.
func init() {
	for k, v := range map[string]string{
		"ÁLBUM":                   "ALBUM",
		"Taz":                     "Taz",
		"Stan":                    "Stan",
		"Pantera Rosa":            "Pink Panther",
		"Juancho":                 "Lenny",
		"Peque travieso":          "Little rascal",
		"Invitado famoso":         "Famous guest",
		"VISTO":                   "SEEN",
		"AMIGO":                   "FRIEND",
		"ELIMINADO":               "ELIMINATED",
		"Cromo nuevo en el álbum": "New card in the album",
		"Flechas mover   Tab o izquierda: trofeos   ":      "Arrows move   Tab or left: trophies   ",
		"???  Trofeo secreto":                              "???  Secret trophy",
		"Trofeo secreto: se descubre jugando.":             "Secret trophy: found by playing.",
		"Premio: ???":                                      "Reward: ???",
		"¡Trofeo secreto desbloqueado!":                    "Secret trophy unlocked!",
		"Medalla secreta":                                  "Secret medal",
		"Un honor discreto. No lo cuentes.":                "A quiet honour. Don't tell anyone.",
		"↑/↓ o rueda elegir   Tab: álbum   ":               "Up/down or wheel choose   Tab: album   ",
		"Baile de gala":                                    "Gala dance",
		"Baila a menos de 40 px de un VIP.":                "Dance within 40 px of a VIP.",
		"Cazador de coronas":                               "Crown hunter",
		"Elimina a 3 VIP en total.":                        "Eliminate 3 VIPs in total.",
		"Amigo de los poderosos":                           "Friend of the powerful",
		"Ayuda a un VIP que da propina (E).":               "Help a VIP who tips (E).",
		"Con invitación, por favor":                        "Invitation, please",
		"Entra en una zona VIP con una invitación robada.": "Enter a VIP area with a stolen invitation.",
		"Brillo dorado":                                    "Golden glow",
		"Usa el traje dorado 3 veces.":                     "Wear the gold suit 3 times.",
		"Mina perfecta":                                    "Perfect mine",
		"Mata a 3 personas con una sola trampa.":           "Kill 3 people with a single trap.",
		"Cadena x3":                                        "Chain x3",
		"Encadena 3 muertes seguidas.":                     "Chain 3 kills in a row.",
		"Tarjeta al rojo":                                  "Maxed-out card",
		"Haz 75 compras en la tienda.":                     "Make 75 purchases in the shop.",
		"Luces fuera, manos quietas":                       "Lights out, hands off",
		"Aguanta un apagón sin matar a nadie.":             "Survive a blackout without killing anyone.",
		"Cromos y más cromos":                              "Cards and more cards",
		"Consigue 15 cromos del álbum.":                    "Get 15 cards of the album.",
	} {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
}
