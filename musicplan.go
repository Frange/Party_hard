package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// Which music plays in a party, and does it fit the genre that the party
// announces? A real mp3 of the user wins over the synthesised music, but only
// if its file name does not say another style: "13_cementerio_metal.mp3" is
// fine in the cemetery, "13_cine_electro.mp3" left over from an older batch is
// ignored (with a warning in the console) and the music of the game plays.
// ---------------------------------------------------------------------------

// genreAliases: the words that name a genre in a file name.
var genreAliases = map[string]string{
	"house": "house", "electro": "electro", "electronica": "electro", "electronic": "electro", "edm": "electro",
	"techno": "techno", "disco": "disco", "reggaeton": "reggaeton", "reggaetón": "reggaeton", "punk": "punk",
	"pop": "pop", "metal": "metal", "rap": "rap", "hiphop": "hiphop", "hip-hop": "hiphop", "lofi": "hiphop",
	"clasica": "clasica", "clásica": "clasica", "classical": "clasica", "jazz": "jazz", "country": "country",
	"flamenco": "flamenco", "pasodoble": "flamenco", "rock": "rock", "synthwave": "synthwave",
	"orquestal": "orquestal", "orchestral": "orquestal",
	// "epico" is the genre of the BARCO PIRATA (pirate_text.go); the secret island has its own word, so an epic mp3
	// left in the island's number is not taken for the island's score and vice versa
	"epico": "epico", "épico": "epico", "epica": "epico", "épica": "epico", "epic": "epico", "isla": "isla", "jurassic": "isla",
	"chiptune": "chiptune",
}

// genreFromFileName: the style a song file declares in the last word of its name (not counting a trailing number) ("" if it declares none).
func genreFromFileName(path string) string {
	name := strings.ToLower(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	words := strings.FieldsFunc(name, func(r rune) bool { return r == '_' || r == ' ' })
	for len(words) > 0 { // "13_cementerio_metal_2.mp3": a trailing number is only a version
		if _, err := strconv.Atoi(words[len(words)-1]); err != nil {
			break
		}
		words = words[:len(words)-1]
	}
	if len(words) == 0 {
		return ""
	}
	return genreAliases[words[len(words)-1]]
}

var musicWarned = map[string]bool{}

// fitsGenre reports whether a song file may play in a party of that genre (files that name no genre always may).
func fitsGenre(path, genre string, warn bool) bool {
	fg := genreFromFileName(path)
	if fg == "" || fg == genre {
		return true
	}
	if warn && !musicWarned[path] {
		musicWarned[path] = true
		fmt.Printf("music: %s es de estilo %q pero esta fiesta es de %q: se usa la música del juego\n", filepath.Base(path), fg, genre)
	}
	return false
}

// levelFile is the mp3 of party number n (1-based) if it exists and fits the genre of the party.
func (m *musicLib) levelFile(n int, genre string, warn bool) (string, bool) {
	f, ok := m.level[n]
	if !ok || !fitsGenre(f, genre, warn) {
		return "", false
	}
	return f, true
}

// moreFor: the songs of more/ that fit the genre of the party.
func (m *musicLib) moreFor(genre string, warn bool) []string {
	var out []string
	for _, f := range m.more {
		if fitsGenre(f, genre, false) {
			out = append(out, f)
		}
	}
	if warn && len(out) < len(m.more) && !musicWarned["more:"+genre] {
		musicWarned["more:"+genre] = true // one line for the whole folder, not one per song
		fmt.Printf("music: more/ tiene %d canciones de otro estilo que se ignoran en esta fiesta (%s); suenan %d\n", len(m.more)-len(out), genre, len(out))
	}
	return out
}

// partyGenreID: the genre id of the party being played ("" outside a party).
func (g *Game) partyGenreID() string {
	if g.lvl == nil {
		return ""
	}
	return genreForTitle(g.lvl.Def.Title).id
}

// musicPlan says what plays in a party and what the title card may claim about it. It is the single
// decision used by the tests; the game takes the same steps in syncRealMusic and syncMusic.
type musicPlan struct {
	source string // level-mp3, more-mp3, dj-mp3, band-mp3 or synth
	genre  string // the genre that is heard ("" when an mp3 does not say)
	file   string
	label  string // what the title card says about the music ("" = nothing)
}

func planMusic(lib *musicLib, title string, levelIdx int, djDead bool, band string) musicPlan {
	lg := genreForTitle(title).id
	own := band != "" && bandGenre(band) != "punk"
	switch {
	case djDead && lib != nil && len(lib.dj) > 0:
		return musicPlan{source: "dj-mp3", genre: genreFromFileName(lib.dj[0]), file: lib.dj[0]}
	case !djDead && band != "" && lib != nil && len(lib.bands[band]) > 0:
		return musicPlan{source: "band-mp3", genre: genreFromFileName(lib.bands[band][0]), file: lib.bands[band][0]}
	case djDead:
		return musicPlan{source: "synth", genre: "afterdj"} // the songs after the DJ (songs_afterdj.go)
	case own:
		return musicPlan{source: "synth", genre: bandGenre(band)}
	}
	if lib != nil {
		if f, ok := lib.levelFile(levelIdx+1, lg, false); ok {
			return musicPlan{source: "level-mp3", genre: genreFromFileName(f), file: f, label: labelIf(genreFromFileName(f) == lg, title)}
		}
		if more := lib.moreFor(lg, false); len(more) > 0 {
			return musicPlan{source: "more-mp3", genre: genreFromFileName(more[0]), file: more[0], label: labelIf(genreFromFileName(more[0]) == lg, title)}
		}
	}
	return musicPlan{source: "synth", genre: lg, label: genreForTitle(title).label}
}

func labelIf(ok bool, title string) string {
	if ok {
		return genreForTitle(title).label
	}
	return ""
}
