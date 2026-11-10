package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
)

// Real songs: drop mp3 files in these folders next to the game (beside the "assets" folder):
//
//	music/N_name.mp3   main song of level N (1..15)
//	more/*.mp3         random songs when the level song ends
//	DJ/*.mp3           random songs once the DJ is dead
//
// Anything missing falls back to the built-in synthesised music.
type musicLib struct {
	base  string
	level map[int]string
	more  []string
	dj    []string
	bands map[string][]string // bands/<band>_*.mp3, keyed by the band name without spaces or symbols
}

func musicBase() string {
	if r := assetsRoot(false); r != "" {
		return filepath.Dir(r)
	}
	if exe, err := os.Executable(); err == nil {
		d := filepath.Dir(exe)
		if strings.Contains(d, ".app/Contents/MacOS") {
			return filepath.Clean(filepath.Join(d, "..", "..", ".."))
		}
		return d
	}
	wd, _ := os.Getwd()
	return wd
}

func mp3Files(dir string) []string {
	es, _ := os.ReadDir(dir)
	var out []string
	for _, e := range es {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".mp3") && !strings.HasPrefix(e.Name(), ".") {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(out)
	return out
}

// loadMusicLib scans the folders (creating them the first time so people see where to drop songs).
func loadMusicLib() *musicLib {
	m := &musicLib{base: musicBase(), level: map[int]string{}}
	for _, d := range []string{"music", "more", "DJ", "bands", "intros"} {
		os.MkdirAll(filepath.Join(m.base, d), 0o755)
	}
	for _, f := range mp3Files(filepath.Join(m.base, "music")) {
		name := filepath.Base(f)
		if i := strings.IndexAny(name, "_- "); i > 0 {
			if n, err := strconv.Atoi(name[:i]); err == nil && n >= 1 {
				if _, dup := m.level[n]; !dup {
					m.level[n] = f
				}
			}
		}
	}
	m.more = mp3Files(filepath.Join(m.base, "more"))
	m.dj = mp3Files(filepath.Join(m.base, "DJ"))
	m.bands = map[string][]string{}
	for _, f := range mp3Files(filepath.Join(m.base, "bands")) {
		name := strings.ToLower(filepath.Base(f))
		name = strings.TrimSuffix(name, ".mp3")
		for _, b := range bandList { // file names start with the band: sum41_song.mp3, "The Offspring - song.mp3"
			if k := bandFileKey(b.name); strings.HasPrefix(bandFileKey(name), k) || strings.HasPrefix(bandFileKey(name), strings.TrimPrefix(k, "the")) {
				m.bands[b.slug] = append(m.bands[b.slug], f)
				break
			}
		}
	}
	return m
}

func pickOther(rng *rand.Rand, list []string, not string) string {
	if len(list) == 0 {
		return ""
	}
	if len(list) == 1 {
		return list[0]
	}
	for {
		f := list[rng.Intn(len(list))]
		if f != not {
			return f
		}
	}
}

// SetFile plays an mp3 (once, or looped). It reports false when the file cannot be decoded.
func (s *Sounds) SetFile(path string, loop bool) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	st, err := mp3.DecodeWithSampleRate(sampleRate, f)
	if err != nil {
		f.Close()
		return false
	}
	var p *audio.Player
	if loop {
		p, err = s.ctx.NewPlayer(audio.NewInfiniteLoop(st, st.Length()))
	} else {
		p, err = s.ctx.NewPlayer(st)
	}
	if err != nil {
		f.Close()
		return false
	}
	if s.music != nil {
		s.music.Pause()
		s.music.Close()
	}
	if s.file != nil {
		s.file.Close()
	}
	s.file, s.music, s.cur, s.isFile, s.loopFile = f, p, "file:"+path, true, loop
	p.SetVolume(0.6)
	s.paused = s.muted
	if !s.muted {
		p.Play()
	}
	return true
}

// FileEnded: the mp3 that was playing has finished by itself.
func (s *Sounds) FileEnded() bool {
	return s.isFile && !s.loopFile && s.music != nil && !s.paused && !s.muted && !s.music.IsPlaying()
}

// syncRealMusic drives the mp3 playlist; it returns true when real music is handling the moment.
func (g *Game) syncRealMusic() bool {
	if g.lib == nil {
		g.lib = loadMusicLib()
		intros := 0
		for _, l := range loadSceneSongs(introsDir()) {
			intros += len(l)
		}
		fmt.Printf("music: %d level songs, %d more, %d DJ, %d intros (%s)\n", len(g.lib.level), len(g.lib.more), len(g.lib.dj), intros, g.lib.base)
	}
	lib := g.lib
	want := "level"
	if g.djDead && len(lib.dj) > 0 {
		want = "dj"
	} else if g.band != nil && !g.djDead && len(lib.bands[g.band.slug]) > 0 {
		want = "band" // the band of the party plays its own songs
	}
	lg := g.partyGenreID()
	moreFn := func() []string { return lib.moreFor(lg, true) } // the songs of more/ that fit the style of this party
	list := lib.dj
	if want == "band" {
		list = lib.bands[g.band.slug]
	}
	play := func(path string, loop bool) bool {
		if path != "" && g.snd.SetFile(path, loop) {
			g.songFile = path
			return true
		}
		if path != "" {
			fmt.Println("music: no se pudo reproducir", path) // (a file that cannot be decoded: the game falls back to its own music)
		}
		return false
	}
	// (musicMode is "<want>-synth" once the synthesised music has taken over: that is not a change of mode)
	if strings.TrimSuffix(g.musicMode, "-synth") != want {
		g.musicMode = want
		ok := false
		more := moreFn()
		if want == "dj" || want == "band" {
			ok = play(pickOther(g.rng, list, g.songFile), false)
		} else if f, has := lib.levelFile(g.levelIdx+1, lg, true); has {
			ok = play(f, len(more) == 0)
		} else if len(more) > 0 {
			ok = play(pickOther(g.rng, more, g.songFile), false)
		}
		if !ok {
			g.musicMode = want + "-synth"
		}
		if g.lvl != nil {
			what := "música sintetizada de " + genreShortName[lg]
			if ok {
				what = g.songFile
			}
			fmt.Printf("music: %s (%s) -> %s [%s]\n", g.lvl.Def.Title, lg, what, want)
		}
		return ok
	}
	if g.musicMode == want+"-synth" {
		return false
	}
	if g.snd.FileEnded() || g.skipSong {
		g.lifeSongChanged() // some guests go dancing (partylife.go)
		var ok bool
		more := moreFn()
		if want == "dj" || want == "band" {
			ok = play(pickOther(g.rng, list, g.songFile), false)
		} else if len(more) > 0 {
			ok = play(pickOther(g.rng, more, g.songFile), false)
			if ok && g.mode == ModePlay {
				g.say("El DJ cambia de canción")
			}
		} else {
			ok = play(g.songFile, false)
		}
		if !ok {
			g.musicMode = want + "-synth"
		}
	}
	return true
}

// freshSong builds song number n of a new batch: its own seed, tempo, key and style.
func freshSong(batch, n int, styles []string) songSpec {
	r := rand.New(rand.NewSource(int64(batch*100003 + n*7919 + 17)))
	st := styles[n%len(styles)]
	bpm := map[string]float64{"house": 122, "electro": 128, "techno": 138, "disco": 116, "reggaeton": 94, "punk": 172}[st] + float64(r.Intn(15)-7)
	roots := []float64{55, 58.27, 61.74, 65.41, 69.3, 73.42, 77.78, 82.41, 49, 51.91}
	scales := [][]int{scaleMinor, scalePhryg, scaleMinor, scaleMajor}
	sc := scales[r.Intn(len(scales))]
	if st == "disco" || st == "reggaeton" {
		sc = scaleMinor
	}
	if st == "punk" {
		sc = scaleMajor
		roots = []float64{82.41, 73.42, 98.0, 110.0, 87.31}
	}
	return songSpec{style: st, bpm: bpm, root: roots[r.Intn(len(roots))], scale: sc, seed: int64(batch*5000 + n*131 + 3), bars: 48}
}

// exportMusic renders a fresh batch of songs to mp3 (needs ffmpeg): 15 level songs,
// 10 for more/ and 6 punk for DJ/. A different batch number gives different songs.
func exportMusic(base string, batch int) {
	write := func(dir, name string, sp songSpec) {
		pcm := renderSong(sp)
		if _, ok := genreSongs[sp.style]; ok { // the tunes of the new genres are short loops: play them for a minute or so
			one := pcm
			for len(pcm) < 60*sampleRate*4 {
				pcm = append(pcm, one...)
			}
		}
		os.MkdirAll(filepath.Join(base, dir), 0o755)
		out := filepath.Join(base, dir, name+".mp3")
		cmd := exec.Command("ffmpeg", "-loglevel", "error", "-y", "-f", "s16le", "-ar", strconv.Itoa(sampleRate), "-ac", "2", "-i", "-", "-b:a", "160k", out)
		cmd.Stdin = bytes.NewReader(pcm)
		if err := cmd.Run(); err != nil {
			fmt.Println("ffmpeg failed:", err)
			return
		}
		fmt.Println(out)
	}
	titles := []string{}
	for _, d := range builtinLevels() {
		titles = append(titles, d.Title)
	}
	slug := func(t string) string {
		return strings.ToLower(strings.Join(strings.Fields(strings.Map(func(r rune) rune {
			if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
				return r
			}
			return ' '
		}, t)), "_"))
	}
	mix := []string{"techno", "disco", "house", "reggaeton", "electro"}
	rand.New(rand.NewSource(int64(batch))).Shuffle(len(mix), func(i, j int) { mix[i], mix[j] = mix[j], mix[i] })
	for i := 0; i < 15; i++ {
		t := fmt.Sprintf("NIVEL %d", i+1)
		if i < len(titles) {
			t = titles[i]
		}
		sp := freshSong(batch, i, mix)
		if gi := genreForTitle(t); gi.legacy {
			sp = freshSong(batch, i, []string{gi.id}) // every party with the genre of its own
		} else {
			sp = genrePlaylist(t)[batch%3]
		}
		write("music", fmt.Sprintf("%d_%s_%s", i+1, slug(t), sp.style), sp)
	}
	for i := 0; i < 10; i++ {
		sp := freshSong(batch, 20+i, mix)
		write("more", fmt.Sprintf("extra_%02d_%s", i+1, sp.style), sp)
	}
	for i := 0; i < 6; i++ {
		sp := freshSong(batch, 40+i, []string{"punk"})
		write("DJ", fmt.Sprintf("punk_%d", i+1), sp)
	}
}
