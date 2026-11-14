package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// ---------------------------------------------------------------------------
// Display names: every recognizable character (famous/themed presets, or a
// custom sprite under sprites/special_people/<name>/) gets a proper name
// used in on-screen kill/news messages. Ordinary random guests stay generic
// ("un invitado"/"una invitada"). A sprite can set its own name by adding
// "display_name" to its meta.json; otherwise the folder name is title-cased.
// ---------------------------------------------------------------------------

// displayNameOverrides fixes the handful of slugs that look wrong simply
// title-cased (acronyms, abbreviations). Add to this as new ones come up.
var displayNameOverrides = map[string]string{
	"et":         "ETT",
	"kpop_fan":   "Fan del K-pop",
	"napoleon_d": "Napoleón Dinamito",
}

// autoDisplayName turns a slug like "nathan_drake" into "Nathan Drake".
func autoDisplayName(slug string) string {
	if dn, ok := displayNameOverrides[slug]; ok {
		return dn
	}
	words := strings.Split(slug, "_")
	for i, w := range words {
		if w == "" {
			continue
		}
		r := []rune(w)
		r[0] = unicode.ToUpper(r[0])
		words[i] = string(r)
	}
	return strings.Join(words, " ")
}

// nameFromSpec resolves the display name for a built-in CharSpec, or "" if
// it is an ordinary unnamed guest (genericSpec never sets Name).
func nameFromSpec(spec *CharSpec) string {
	if spec.DisplayName != "" {
		return spec.DisplayName
	}
	if spec.Name == "" {
		return ""
	}
	return autoDisplayName(spec.Name)
}

// nameFromAsset resolves the display name for a character loaded from the
// assets folder (always named: it only applies to special_people).
func nameFromAsset(ca *CharAsset) string {
	if ca.Meta.DisplayName != "" {
		return ca.Meta.DisplayName
	}
	return autoDisplayName(ca.Meta.Name)
}

// slasherActorName is how the special-enemy killer is credited in messages.
func slasherActorName(s *Slasher) string {
	if s.name == "" {
		return "Un asesino desconocido"
	}
	return autoDisplayName(s.name)
}

// guestLabel is how a guest is referred to in messages: its proper name if
// it has one, otherwise a generic label.
func guestLabel(gu *Guest) string {
	if n := ensureNick(gu); n != "" {
		return n
	}
	if gu.cs != nil && gu.cs.Spec != nil && gu.cs.Spec.Female {
		return "una invitada"
	}
	return "un invitado"
}

// ---------------------------------------------------------------------------
// News bank: short, varied status lines with {placeholder} substitution,
// organized by category so they can be added to or rewritten without
// touching the code (assets/text/news.json overrides/extends these).
// ---------------------------------------------------------------------------

type NewsBank map[string][]string

func defaultNewsBank() NewsBank {
	return NewsBank{
		"fbi_arrive":          {"¡El FBI se hace cargo del caso!", "El FBI toma el control de la investigación", "Agentes del FBI llegan a la escena"},
		"military_arrive":     {"Entra en acción el ejército", "Llegan los militares a poner orden", "El ejército toma posiciones en la zona"},
		"swat_arrive":         {"El SWAT toma el control", "Un furgón del SWAT irrumpe en la escena", "El SWAT se despliega por la zona"},
		"rangers_arrive":      {"¡Llaman a los Pawer Rangers!", "¡Es morphling time! Llegan los Pawer Rangers", "Los Pawer Rangers aparecen para salvar el día"},
		"police_call":         {"¡Alguien ha llamado a la policía!", "La policía ha sido avisada", "Suenan las sirenas: viene la policía"},
		"ambulance_call":      {"Llega una ambulancia a por los cuerpos", "Avisan a una ambulancia, hay demasiados cuerpos", "Una ambulancia viene de camino"},
		"backup_call":         {"Piden refuerzos: llega otro agente", "La policía pide refuerzos", "Un segundo agente viene de camino"},
		"shoot_suspect":       {"{actor} ha disparado a un sospechoso", "{actor} abre fuego contra un sospechoso", "{actor} dispara a {victim}"},
		"suspect_taken":       {"La policía se lleva a un sospechoso", "Se llevan esposado a {victim}", "{victim} queda detenido"},
		"killer_catch":        {"¡La policía ha detenido al asesino!", "Han atrapado al asesino", "El asesino queda bajo custodia"},
		"kill":                {"{actor} ha matado a {victim}", "{actor} ha acabado con {victim}", "Han visto a {actor} matando a {victim}", "{actor} ha asesinado a {victim}"},
		"kill_player":         {"Has matado a {victim}", "{victim} no lo vio venir", "Un sospechoso menos: {victim}", "{victim} ha caído"},
		"jail_held":           {"{victim} acaba entre rejas... de furgón", "La policía se lleva esposado a {victim}", "{victim} duerme hoy en comisaría... o eso cree"},
		"jail_break":          {"¡Fuga! {victim} escapa arrastrando una bola", "{victim} se fuga de la cárcel ruidosamente", "Un preso con bola y cadena corre por la fiesta"},
		"jail_rescue":         {"{victim} se libra de la poli con un poco de ayuda", "¡Rescate! {victim} vuelve a la fiesta", "Alguien ha abierto el furgón: {victim} libre"},
		"electrocuted":        {"{victim} se ha electrocutado: queda en los huesos", "¡Zas! {victim} se ha quedado en los huesos", "Una descarga deja a {victim} en el esqueleto", "{victim} se ha puesto muy eléctrico"},
		"arson_arrive":        {"Un pirómano anda suelto por la fiesta", "Alguien huele a chamusquina: hay un pirómano", "Un loco con un mechero ha entrado en la fiesta"},
		"arson_fire":          {"¡Fuego en la fiesta!", "Algo arde y no es la pista de baile", "El pirómano ha prendido algo"},
		"fire_call":           {"Piden bomberos: hay un incendio en la fiesta", "Alguien ha llamado a los bomberos por el fuego"},
		"fire_death":          {"{victim} se ha chamuscado", "{victim} no ha salido a tiempo del fuego", "El fuego se ha llevado a {victim}"},
		"dj_dead":             {"¡Han matado al DJ! Suena funk, ska o salsa", "El DJ ha muerto. ¿Quién pondrá la música ahora?"},
		"vehicle_wreck_crew":  {"¡El tanque ha aplastado un vehículo de emergencias con su tripulación!", "¡El tanque ha hecho pedazos un vehículo de emergencias, con todos dentro!"},
		"vehicle_wreck_empty": {"¡El tanque ha aplastado el vehículo de emergencias!", "El tanque deja el vehículo de emergencias hecho chatarra"},
	}
}

// loadNewsBank reads assets/text/news.json if present and merges it over
// the built-in defaults (a category in the file replaces the built-in one;
// categories it does not mention keep their defaults). Missing file or
// -noassets just keeps the defaults, same as every other asset folder.
func loadNewsBank() NewsBank {
	bank := defaultNewsBank()
	if noAssets {
		return bank
	}
	root := assetsRoot(false)
	if root == "" {
		return bank
	}
	data, err := os.ReadFile(filepath.Join(root, "text", "news.json"))
	if err != nil {
		return bank
	}
	var custom map[string][]string
	if json.Unmarshal(data, &custom) != nil {
		return bank
	}
	for k, v := range custom {
		if len(v) > 0 {
			bank[k] = v
		}
	}
	return bank
}

// sayNews queues a line from category, substituting {name} placeholders
// from vars, and avoids immediately repeating the same line for that
// category when an alternative exists.
func (g *Game) sayNews(category string, vars map[string]string) {
	lines := g.news[category]
	if len(lines) == 0 {
		return
	}
	// pick a variant that has not been on screen lately; with none, say nothing
	start := g.rng.Intn(len(lines))
	for n := 0; n < len(lines); n++ {
		i := (start + n) % len(lines)
		s := lines[i]
		for k, v := range vars {
			s = strings.ReplaceAll(s, "{"+k+"}", v)
		}
		if sayRecent(s) {
			continue
		}
		g.newsLast[category] = i
		g.say(s)
		return
	}
}
