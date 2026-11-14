package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ---------------------------------------------------------------------------
// Dialogue bank: short speech-bubble lines (the ones shown above a head via
// gu.icon/c.icon), organized by character category and situation, so they
// can be rewritten or extended per category without touching the code.
// One JSON file per category under assets/text/: people.json,
// special_people.json, police.json, killers.json, fbi.json,
// power_rangers.json. A file only needs the situations it wants to change;
// anything it does not mention keeps the built-in lines for that category,
// and a category/situation with nothing at all falls back to "people".
// ---------------------------------------------------------------------------

type DialogueBank map[string]map[string][]string

var dialogueCategories = []string{"people", "special_people", "police", "killers", "fbi", "power_rangers"}

func defaultDialogueBank() DialogueBank {
	b := baseDialogueBank()
	mergeJokes(b)
	return b
}

func baseDialogueBank() DialogueBank {
	return DialogueBank{
		"people": {
			"dance_join":     {"¡Vamos a bailar!", "¡Qué marcha!", "¡Esto es vida!", "¡Dale caña!"},
			"dance_leave":    {"Paso de esto", "Qué cutre bailas", "Me voy de aquí", "Esto no es lo mío"},
			"dance_annoyed":  {"¡Ey, cuidado!", "¡Serás...!", "¡Te voy a dar!", "¡A mí no me empujas!"},
			"greet":          {"¡Hola!", "¡Qué pasa!", "¡Buenas!", "¡Eh, qué tal!"},
			"scared":         {"¡Ahhh!", "¡Socorro!", "¡Que alguien me ayude!", "¡No, por favor!"},
			"saw_body":       {"¡Un cuerpo!", "¡Hay un muerto!", "¡Dios mío, sangre!", "¡Alguien ha muerto aquí!"},
			"drink":          {"¡Salud!", "Mmm, rico", "Una más...", "Esto anima la fiesta"},
			"dance_with":     {"¡Contigo sí!", "¡Qué ritmo!", "¡Así, así!", "¡No pares!"},
			"calling_police": {"¡Llamando a la policía!", "¡Voy a avisar!", "¡Esto hay que denunciarlo!", "¡La policía tiene que saberlo!"},
			"accused":        {"¡Fue él!", "¡Yo le vi!", "¡Ese es el asesino!", "¡Qué alguien lo detenga!"},
		},
		"police": {
			"understood": {"ENTENDIDO", "RECIBIDO", "VOY PARA ALLÁ"},
			"halt":       {"¡ALTO! ¡POLICÍA!", "¡QUIETO AHÍ!", "¡NO TE MUEVAS!"},
			"arrest":     {"¡QUEDA DETENIDO!", "¡ESTÁS ARRESTADO!", "SE VIENE CONMIGO"},
			"tired":      {"PUFF... PUFF...", "NECESITO UN DESCANSO", "ESTO ES AGOTADOR"},
		},
		"fbi": {
			"understood": {"COMPRENDIDO", "AGENTE EN CAMINO", "RECIBIDO, INVESTIGANDO"},
			"halt":       {"¡FBI, QUIETO!", "¡ALTO AHÍ!", "¡NO DÉ UN PASO MÁS!"},
			"arrest":     {"QUEDA BAJO CUSTODIA FEDERAL", "EL CASO ESTÁ CERRADO", "TIENE DERECHO A GUARDAR SILENCIO"},
			"tired":      {"ESTO NO ENCAJA...", "SEGUIMOS INVESTIGANDO", "NECESITAMOS MÁS PRUEBAS"},
		},
		"power_rangers": {
			"understood": {"¡RECIBIDO!", "¡VAMOS ALLÁ!", "¡EN CAMINO, EQUIPO!"},
			"halt":       {"¡SE ACABÓ EL JUEGO!", "¡NO TE SALDRÁS CON LA TUYA!", "¡ES MORPHLING TIME!"},
			"arrest":     {"¡JUSTICIA SERVIDA!", "¡OTRO VILLANO MENOS!", "¡EL POR EL PODER DEL GREYSKULL!"},
			"tired":      {"¡AÚN QUEDA PELEA!", "¡NO NOS RENDIMOS!"},
		},
		"killers": {
			"pain":   {"!!!", "¡AAAGH!", "GRRR..."},
			"caught": {"...", "MALDICIÓN...", "ESTO NO HA TERMINADO"},
		},
	}
}

// loadDialogueBank reads assets/text/<category>.json for each category (if
// present) and merges its situations over the built-in defaults.
func loadDialogueBank() DialogueBank {
	bank := defaultDialogueBank()
	if noAssets {
		return bank
	}
	root := assetsRoot(false)
	if root == "" {
		return bank
	}
	for _, cat := range dialogueCategories {
		data, err := os.ReadFile(filepath.Join(root, "text", cat+".json"))
		if err != nil {
			continue
		}
		var custom map[string][]string
		if json.Unmarshal(data, &custom) != nil {
			continue
		}
		if bank[cat] == nil {
			bank[cat] = map[string][]string{}
		}
		for situation, lines := range custom {
			if len(lines) > 0 {
				bank[cat][situation] = lines
			}
		}
	}
	return bank
}

// line picks a random phrase for (category, situation); a category with
// nothing for that situation falls back to "people", then a neutral filler.
func (g *Game) line(category, situation string) string {
	lines := g.dialogue[category][situation]
	if len(lines) == 0 {
		lines = g.dialogue["people"][situation]
	}
	if len(lines) == 0 {
		return "..."
	}
	return g.freshPhrase(category+"/"+situation, lines, g.rng.Intn) // "" = nothing new to say
}

// guestCategory: "special_people" for a named/recognizable guest, "people" otherwise.
func guestCategory(gu *Guest) string {
	if gu.name != "" {
		return "special_people"
	}
	return "people"
}

// copCategory: which dialogue bank a responder's own voice lines use.
// Military and SWAT share the plain "police" bank (the request only asked
// for dedicated police/killers/fbi/power_rangers files).
func copCategory(kind int) string {
	switch kind {
	case unitFBI:
		return "fbi"
	case unitRangers:
		return "power_rangers"
	default:
		return "police"
	}
}
