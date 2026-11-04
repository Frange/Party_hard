package main

import (
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// The kind of party of every level, with a silly name, and the music that goes
// with it (genres.go decides the genre). The level select and the title card
// of every party show it: "1 FIESTA EN CASA / Botellón de salón · House".
// ---------------------------------------------------------------------------

// partyTypeOfTitle: the kind of party, by the title of the level (the same keys as genreOfTitle).
var partyTypeOfTitle = map[string]string{
	"HOUSE PARTY":           "Botellón de salón",
	"MIAMI BEACH":           "Chiringuito sin ley",
	"GLOW NEON":             "Rave de neón",
	"BULL'S EYE":            "Hoguera con botas",
	"PLAZA DE TOROS":        "Fiesta de capotes",
	"TORRE EIFFEL":          "Cena de gala con vals",
	"CRUCERO NOCTURNO":      "Fiesta del capitán",
	"PARQUE DE ATRACCIONES": "Feria sin cinturón",
	"ESTADIO":               "Final de copa",
	"DISCOTECA":             "Sesión de madrugada",
	"FESTIVAL":              "Festival de barro",
	"SKATE PARK":            "Jam en el bowl",
	"CEMENTERIO":            "Velatorio con riffs",
	"CINE":                  "Sesión golfa",
	"CIRCUITO":              "Carrera sin freno",
	"ESTACIÓN DE ESQUÍ":     "Après-ski con chocolate",
	"ROOFTOP PARTY":         "Terraza con piscina",
	"CASINO":                "Noche de corbata",
	"AUTOPISTA":             "Atasco con altavoces",
	"ESTACIÓN DE TREN":      "Fiesta sin billete",
	"ESTACIÓN ESPACIAL":     "Rave de gravedad cero",
	"ISLA NUBLAR":           "Última fiesta jurásica",
	"PUB":                   "Noche de dardos",
	"PLAZA EUROPEA":         "Verbena europea",
	"CENTRO COMERCIAL":      "Rebajas a todo ritmo",
}

// genreShortName is the style of the music in one word.
var genreShortName = map[string]string{
	"house": "House", "electro": "Electrónica", "techno": "Techno", "disco": "Disco", "reggaeton": "Reggaetón",
	"punk": "Punk", "pop": "Pop", "metal": "Metal", "rap": "Rap", "hiphop": "Hip-hop", "clasica": "Clásica",
	"jazz": "Jazz", "country": "Country", "flamenco": "Pasodoble", "rock": "Rock", "synthwave": "Synthwave",
	"orquestal": "Orquestal", "epico": "Épica", "isla": "Épica", "chiptune": "Chiptune",
}

// partyTypeFor: the kind of party of a level ("" for the levels of the user, which get none).
func partyTypeFor(title string) string { return partyTypeOfTitle[strings.ToUpper(title)] }

// genreName: the genre of a party in one word, translated.
func genreName(title string) string {
	return tr(genreShortName[genreForTitle(title).id])
}

// partySubtitle: "Botellón de salón · House": what the level select says under the name of a party.
func partySubtitle(title string) string {
	g := genreName(title)
	if t := partyTypeFor(title); t != "" {
		return fmt.Sprintf("%s · %s", tr(t), g)
	}
	return g
}

// partyCardLine: the small line above the title when a party starts: the kind of party and, when the music of the
// game is the one playing (or an mp3 named after the style), its genre.
func (g *Game) partyCardLine() string {
	if g.lvl == nil {
		return ""
	}
	t := tr(partyTypeFor(g.lvl.Def.Title))
	lab := tr(g.levelGenreLabel())
	switch {
	case t != "" && lab != "":
		return t + " · " + lab
	case t != "":
		return t
	}
	return lab
}
