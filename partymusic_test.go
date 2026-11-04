package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/bitmapfont/v3"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

var allPartyTitles = func() []string {
	var t []string
	for k := range genreOfTitle {
		t = append(t, k)
	}
	return t
}()

// every party names its kind and its style, and the two fit under its name in the level select
func TestEveryPartyHasATypeAndFits(t *testing.T) {
	g := &Game{face: text.NewGoXFace(bitmapfont.Face)}
	defer func(l string) { uiLang = l }(uiLang)
	for _, title := range allPartyTitles {
		if partyTypeFor(title) == "" {
			t.Errorf("%s: no kind of party", title)
		}
		for _, lang := range []string{"es", "en"} {
			uiLang = lang
			sub := partySubtitle(title)
			if !strings.Contains(sub, " · ") {
				t.Errorf("%s: subtitle %q", title, sub)
			}
			if w := g.textW(sub, 1); w > 222 { // the columns of the level select are 235 px apart
				t.Errorf("%s (%s): %q is %.0f px wide", title, lang, sub, w)
			}
		}
		uiLang = "en"
		if tr(partyTypeFor(title)) == partyTypeFor(title) {
			t.Errorf("%s: the kind of party is not translated", title)
		}
		uiLang = "es"
	}
	// the title card says both things, and only claims a style that is really playing
	g.lvl = &Level{Def: &LevelDef{Title: "SKATE PARK"}}
	if line := g.partyCardLine(); line != "Jam en el bowl · Música: Punk" {
		t.Errorf("card: %q", line)
	}
	g.djDead = true
	if line := g.partyCardLine(); line != "Jam en el bowl" {
		t.Errorf("card with the DJ dead: %q", line)
	}
}

func fakeLib(level map[int]string, more, dj []string) *musicLib {
	return &musicLib{level: level, more: more, dj: dj, bands: map[string][]string{}}
}

func TestGenreFromFileName(t *testing.T) {
	for name, want := range map[string]string{
		"13_cementerio_metal.mp3": "metal", "1_house_party_reggaeton.mp3": "reggaeton", "extra_01_house.mp3": "house",
		"6_torre_eiffel_clasica.mp3": "clasica", "5_plaza_de_toros_flamenco.mp3": "flamenco", "5_fiesta.mp3": "",
		"3 Neón Synthwave.mp3": "synthwave", "7_crucero_Disco.MP3": "disco", "punk_1.mp3": "punk", "song.mp3": "",
	} {
		if got := genreFromFileName(name); got != want {
			t.Errorf("%s: %q, want %q", name, got, want)
		}
	}
}

// the genre announced is the genre that plays, whatever mp3 there is in the folders
func TestMusicPlanIsCoherent(t *testing.T) {
	defs := builtinLevels()
	for i, d := range defs {
		title := d.Title
		lg := genreForTitle(title).id
		// no mp3: our own music, in the style of the party
		p := planMusic(fakeLib(map[int]string{}, nil, nil), title, i, false, "")
		if p.source != "synth" || p.genre != lg || p.label == "" {
			t.Errorf("%s: no mp3 -> %+v", title, p)
		}
		for _, sp := range playlistFor(title) {
			if sp.style != lg {
				t.Errorf("%s: the synthesised playlist has a %s song, the party is %s", title, sp.style, lg)
			}
		}
		// a left-over mp3 named after another style is ignored, a matching one is used
		other := "reggaeton"
		if lg == other {
			other = "metal"
		}
		old := filepath.Join("music", "x_"+other+".mp3")
		p = planMusic(fakeLib(map[int]string{i + 1: old}, nil, nil), title, i, false, "")
		if p.source != "synth" || p.genre != lg {
			t.Errorf("%s: an mp3 of %s must not play: %+v", title, other, p)
		}
		good := filepath.Join("music", "x_"+lg+".mp3")
		p = planMusic(fakeLib(map[int]string{i + 1: good}, nil, nil), title, i, false, "")
		if p.source != "level-mp3" || p.genre != lg || p.label == "" {
			t.Errorf("%s: an mp3 of its own style must play: %+v", title, p)
		}
		// an mp3 that names no style plays but the card claims nothing
		p = planMusic(fakeLib(map[int]string{i + 1: "music/x_nombre.mp3"}, nil, nil), title, i, false, "")
		if p.source != "level-mp3" || p.label != "" {
			t.Errorf("%s: unnamed mp3 -> %+v", title, p)
		}
		// more/ with songs of every style: only the ones of the party are used
		var more []string
		for id := range genreTable {
			more = append(more, filepath.Join("more", "extra_"+id+".mp3"))
		}
		p = planMusic(fakeLib(map[int]string{}, more, nil), title, i, false, "")
		if p.source != "more-mp3" || p.genre != lg || p.label == "" {
			t.Errorf("%s: more/ -> %+v", title, p)
		}
		// more/ with no song of the style: our music
		p = planMusic(fakeLib(map[int]string{}, []string{"more/extra_nope_" + other + ".mp3"}, nil), title, i, false, "")
		if p.source != "synth" || p.genre != lg {
			t.Errorf("%s: more/ without the style -> %+v", title, p)
		}
		// the DJ dead: punk, and the card says nothing about the style of the party
		p = planMusic(fakeLib(map[int]string{}, nil, nil), title, i, true, "")
		if p.genre != "punk" || p.label != "" {
			t.Errorf("%s: DJ dead -> %+v", title, p)
		}
		// a famous band of its own style
		p = planMusic(fakeLib(map[int]string{}, nil, nil), title, i, false, "metallica")
		if p.genre != "metal" || p.label != "" {
			t.Errorf("%s: Metallica -> %+v", title, p)
		}
		// the invariant: whatever the card claims is what plays
		if p.label != "" && p.genre != lg {
			t.Errorf("%s: the card says %q but %s plays", title, p.label, p.genre)
		}
	}
}

// the game itself (levelGenreLabel) agrees with the plan, for the real states of the music
func TestGameLabelFollowsThePlayingSong(t *testing.T) {
	g := &Game{lvl: &Level{Def: &LevelDef{Title: "CEMENTERIO"}}}
	if g.levelGenreLabel() != "Música: Metal" {
		t.Errorf("synth: %q", g.levelGenreLabel())
	}
	g.musicMode, g.songFile = "level", "music/13_cementerio_metal.mp3"
	if g.levelGenreLabel() != "Música: Metal" {
		t.Errorf("mp3 of its style: %q", g.levelGenreLabel())
	}
	g.songFile = "music/13_cine_electro.mp3"
	if g.levelGenreLabel() != "" {
		t.Errorf("an mp3 of another style must not be announced: %q", g.levelGenreLabel())
	}
	g.songFile = "music/13_algo.mp3"
	if g.levelGenreLabel() != "" {
		t.Error("an mp3 that names no style is not announced")
	}
}

// the real library: files of the user are never touched, but they are filtered by style
func TestLibFiltersFiles(t *testing.T) {
	m := fakeLib(map[int]string{13: "music/13_cine_electro.mp3", 12: "music/12_skate_park_punk.mp3"}, []string{"more/a_house.mp3", "more/b_punk.mp3", "more/c.mp3"}, nil)
	if _, ok := m.levelFile(13, "metal", false); ok {
		t.Error("the electro mp3 must not play in the cemetery")
	}
	if f, ok := m.levelFile(12, "punk", false); !ok || !strings.HasSuffix(f, "punk.mp3") {
		t.Error("the punk mp3 plays in the skate park")
	}
	got := m.moreFor("punk", false)
	if len(got) != 2 {
		t.Errorf("more for punk: %v", got)
	}
}
