package main

import "strings"

// English texts of the hints that name the "interact" key / button (input.go: actionHint, actionHintFor): they
// used to say a fixed "(E)", now the key or the pad button (O) of the player. Merged into langDataEN without
// overwriting anything.
func init() {
	for k, v := range map[string]string{
		"Suelta a tu compañero primero (%s)":                        "Put your partner down first (%s)",
		"Suelta el cuerpo primero (%s)":                             "Drop the body first (%s)",
		"Ve a la cabina del DJ (%s) para cambiar la canción":        "Go to the DJ booth (%s) to change the song",
		"Escondido. Pulsa %s para salir":                            "Hidden. Press %s to exit",
		"{a} detenido: ¡que {b} lo saque ({k})!":                    "{a} busted: {b}, get them out ({k})!",
		"Ayuda a un VIP que da propina (%s).":                       "Help a VIP who tips (%s).",                 // trophy how-to (howHint)
		"Se lleva puesto en la fiesta. %s: ponértelo / quitártelo.": "Worn at the party. %s: put on / take off.", // shop (shopDescHint)
		"Triángulo": "Triangle", // pad buttons (input.go: padLabels)
		"Cuadrado":  "Square",
	} {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
}

// howHint: the "(E)" of a trophy's how-to names the interact key or pad button of the player ("(O)" on the pad).
func howHint(how string) string {
	if !strings.Contains(how, "(E)") {
		return how
	}
	return strings.Replace(how, "(E)", "("+actionHint(actInteract)+")", 1)
}

// shopDescHint: the "Enter:" of a shop description names the pad's accept button while the player is on a pad.
func shopDescHint(desc string) string {
	if !hintOnPad(0) || !strings.Contains(desc, "Enter:") {
		return desc
	}
	return strings.Replace(desc, "Enter:", padShort(binds[actConfirm].Pad)+":", 1)
}
