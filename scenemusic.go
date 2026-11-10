package main

import (
	"bytes"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
)

// ---------------------------------------------------------------------------
// The music of the story scenes.
//
//	1 the first intro (the room)        "pum pum" of the neighbours' party with a catchy hook, in time with the speakers
//	2 interlude after party 5           a cartoon march of an angry man walking (xylophone and pizzicato); the memories
//	                                    cut to a sneaky, funny minor version of the same tune
//	3 interlude after party 10          the same, as a jaunty sea march (chiptune lead)
//	4 the ending                        a cheerful, tender lullaby (music box and flute) that ends calm
//	5 the secret ending                 a festive fanfare
//	6 the invitation to the island      an adventure-with-a-little-mystery theme (marimba and flute)
// None of them has a bass drum (songs.go); the written tunes are in songs.go.
//
// Every one can be replaced with a real song: put it in the intros/ folder next to
// "assets" as N_name.mp3 (see intros/README.md). Without a file the synthesised
// music is used. Only the first intro keeps the bass drum, and its picture follows
// the BPM of the song (intros/1_intro.bpm, 124 by default).
// ---------------------------------------------------------------------------

const (
	sceneIntroN = iota + 1
	sceneInterA
	sceneInterB
	sceneEndN
	sceneSecretN
	sceneInviteN
	sceneRocketN    // the rocket interlude (after party 20); it comes last so the file numbers of the others stay
	sceneMusicCount = sceneRocketN
)

const defaultIntroBPM = 124.0

// introsDirOverride lets the tests look somewhere else.
var introsDirOverride string

func introsDir() string {
	if introsDirOverride != "" {
		return introsDirOverride
	}
	return filepath.Join(musicBase(), "intros")
}

// sceneFileNumber: the scene a file belongs to (its name starts with the number: 2_interludio_A.mp3), 0 if none.
func sceneFileNumber(name string) int {
	name = strings.ToLower(filepath.Base(name))
	if !strings.HasSuffix(name, ".mp3") || strings.HasPrefix(name, ".") {
		return 0
	}
	i := 0
	for i < len(name) && name[i] >= '0' && name[i] <= '9' {
		i++
	}
	if i == 0 || i == len(name) || !strings.ContainsRune("_- .", rune(name[i])) {
		return 0
	}
	n, err := strconv.Atoi(name[:i])
	if err != nil || n < 1 || n > sceneMusicCount {
		return 0
	}
	return n
}

// loadSceneSongs lists the songs of a folder per scene (several files with the same number: one is picked at random).
func loadSceneSongs(dir string) map[int][]string {
	out := map[int][]string{}
	for _, f := range mp3Files(dir) {
		if n := sceneFileNumber(f); n > 0 {
			out[n] = append(out[n], f)
		}
	}
	return out
}

// sceneSong returns a song of the folder for the scene (read every time: it is a handful of files),
// or "" when there is none and the synthesised music takes over.
func sceneSong(n int) string {
	list := loadSceneSongs(introsDir())[n]
	rand.Shuffle(len(list), func(i, j int) { list[i], list[j] = list[j], list[i] })
	for _, f := range list { // one that cannot be decoded is skipped (the music of the game takes over)
		if mp3Decodable(f) {
			return f
		}
	}
	return ""
}

func mp3Decodable(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	st, err := mp3.DecodeWithSampleRate(sampleRate, f)
	return err == nil && st.Length() > 0
}

// introBPM reads the optional intros/1_intro.bpm (or bpm.txt): the tempo of the song of the first intro.
func introBPM() float64 {
	for _, name := range []string{"1_intro.bpm", "bpm.txt"} {
		b, err := os.ReadFile(filepath.Join(introsDir(), name))
		if err != nil {
			continue
		}
		if v, err := strconv.ParseFloat(strings.TrimSpace(string(b)), 64); err == nil && v >= 40 && v <= 240 {
			return v
		}
	}
	return defaultIntroBPM
}

// setThumpBPM makes the picture of the first intro follow a song of another tempo.
func setThumpBPM(bpm float64) { thumpBeat = 60 / bpm }

// resetThumpBeat goes back to the beat of the synthesised loop.
func resetThumpBeat() { thumpBeat = float64(thumpBeatSamples) / sampleRate }

// ---- the layers ----------------------------------------------------------------------

type sceneReq struct {
	key   string // what is playing, to tell a repeated request from a new one
	build func() []byte
	file  string
	loop  bool
}

const sceneGain = 0.7

// StartScene starts (or keeps) a layer of the music of a scene: the song `file` if it can be played, otherwise
// the synthesised loop. Nothing starts while the game is muted (it starts if the sound comes back on).
func (s *Sounds) StartScene(layer, key string, build func() []byte, file string, loop bool) {
	if s.sceneWant == nil {
		s.sceneWant, s.sceneP, s.sceneF, s.sceneVol = map[string]sceneReq{}, map[string]*audio.Player{}, map[string]*os.File{}, map[string]float64{}
	}
	if w, ok := s.sceneWant[layer]; ok && w.key == key {
		return
	}
	s.stopLayer(layer)
	s.sceneWant[layer] = sceneReq{key: key, build: build, file: file, loop: loop}
	s.ensureScene(layer)
}

func (s *Sounds) ensureScene(layer string) {
	w, ok := s.sceneWant[layer]
	if !ok || s.sceneP[layer] != nil || s.muted || s.ctx == nil {
		return
	}
	var p *audio.Player
	if w.file != "" {
		if f, err := os.Open(w.file); err == nil {
			if st, err := mp3.DecodeWithSampleRate(sampleRate, f); err == nil {
				if w.loop {
					p, err = s.ctx.NewPlayer(audio.NewInfiniteLoop(st, st.Length()))
				} else {
					p, err = s.ctx.NewPlayer(st)
				}
				if err == nil {
					s.sceneF[layer] = f
				} else {
					p = nil
				}
			}
			if p == nil {
				f.Close()
			}
		}
	}
	if p == nil && w.build != nil { // no song (or it cannot be decoded): the synthesised music
		k := "scene:" + w.key
		data, ok := s.tracks[k]
		if !ok {
			data = w.build()
			s.tracks[k] = data
		}
		var err error
		p, err = s.ctx.NewPlayer(audio.NewInfiniteLoop(bytes.NewReader(data), int64(len(data))))
		if err != nil {
			return
		}
	}
	if p == nil {
		return
	}
	p.SetVolume(0)
	p.Play()
	s.sceneP[layer] = p
}

// SetSceneVolume sets how loud a layer is (0..1); mute silences it.
func (s *Sounds) SetSceneVolume(layer string, v float64) {
	if _, ok := s.sceneWant[layer]; !ok {
		return
	}
	s.ensureScene(layer)
	p := s.sceneP[layer]
	if p == nil {
		return
	}
	if s.muted {
		v = 0
	}
	p.SetVolume(math.Max(0, math.Min(1, v)) * sceneGain)
}

// SceneActive: the layer has been asked for (it may be silent because of mute).
func (s *Sounds) SceneActive(layer string) bool {
	_, ok := s.sceneWant[layer]
	return ok
}

// SceneIsSong: the layer was asked to play a real song of the intros folder.
func (s *Sounds) SceneIsSong(layer string) bool {
	w, ok := s.sceneWant[layer]
	return ok && w.file != ""
}

func (s *Sounds) stopLayer(layer string) {
	if p := s.sceneP[layer]; p != nil {
		p.Pause()
		p.Close()
	}
	if f := s.sceneF[layer]; f != nil {
		f.Close()
	}
	delete(s.sceneP, layer)
	delete(s.sceneF, layer)
	delete(s.sceneWant, layer)
	delete(s.sceneVol, layer)
}

// StopScene ends all the layers of the music of the story scenes.
func (s *Sounds) StopScene() {
	for l := range s.sceneWant {
		s.stopLayer(l)
	}
	for l := range s.sceneP {
		s.stopLayer(l)
	}
}

// ---- small helpers for the songs --------------------------------------------------------------------

func smooth01(x float64) float64 {
	x = math.Max(0, math.Min(1, x))
	return x * x * (3 - 2*x)
}

// echoLoop adds a looping echo (the tails of the loop come round to its start).
func echoLoop(x []float64, delay, fb, mixv float64) []float64 {
	n := len(x)
	d := int(delay * sampleRate)
	wet := make([]float64, n)
	for pass := 0; pass < 8; pass++ {
		for i := 0; i < n; i++ {
			j := (i - d + n) % n
			wet[i] = x[j] + fb*wet[j]
		}
	}
	out := make([]float64, n)
	for i := range out {
		out[i] = x[i] + mixv*wet[i]
	}
	return out
}

// ---- built in the background, once ------------------------------------------------------------------

// Every kind is built once, by whoever asks first; the others wait for that build (never for another kind's:
// the menu's track of the tier must not queue behind the scene songs the warm-up thread is building).
type ambEntry struct {
	once sync.Once
	b    []byte
}

var (
	ambMu    sync.Mutex
	ambCache = map[string]*ambEntry{}
	// ambBuildFn builds a kind (the tests make it slow to check that nothing waits for it)
	ambBuildFn = buildAmbient
)

// ambientPCM returns the loop of a kind, building it the first time (the warm-up thread usually has it ready).
func ambientPCM(kind string) []byte {
	ambMu.Lock()
	e := ambCache[kind]
	if e == nil {
		e = &ambEntry{}
		ambCache[kind] = e
	}
	ambMu.Unlock()
	e.once.Do(func() {
		e.b = ambBuildFn(kind)
		ambReady.Store(kind, e.b)
	})
	return e.b
}

// ambReady: the loops already built, readable without waiting for the build thread.
var ambReady sync.Map

// ambientReady returns the loop of a kind if it has been built already, nil otherwise (it never waits).
func ambientReady(kind string) []byte {
	if b, ok := ambReady.Load(kind); ok {
		return b.([]byte)
	}
	return nil
}

var warmOnce sync.Once

// warmAmbient builds the ambient loops in a background thread so no scene waits for them.
func warmAmbient() {
	warmOnce.Do(func() {
		go func() {
			for _, k := range append([]string{ambAnthem}, append(ambientKinds, ambSleep)...) { // the anthem first: the title and the menu play it
				ambientPCM(k)
			}
		}()
	})
}

// ambientWarmOne builds one kind in a background thread (once), for the loops the warm-up does not build.
var ambWarming sync.Map

func ambientWarmOne(kind string) {
	if _, busy := ambWarming.LoadOrStore(kind, true); busy {
		return
	}
	go ambientPCM(kind)
}

// ambientBuilder is what StartScene needs for a kind.
func ambientBuilder(kind string) func() []byte { return func() []byte { return ambientPCM(kind) } }

// ---- what each scene asks for ---------------------------------------------------------------------------

// sceneMusic starts the "base" layer of a scene: its song if the intros folder has one, else the ambient loop.
func (g *Game) sceneMusic(n int, kind string) {
	g.snd.StartScene("base", "base:"+kind, ambientBuilder(kind), sceneSong(n), true)
}

// introMusicStart starts the first intro: its song if there is one (and the picture follows its BPM), else the
// "pum pum" loop. It reports whether a song is playing.
func (g *Game) introMusicStart() bool {
	if f := sceneSong(sceneIntroN); f != "" {
		setThumpBPM(introBPM())
		g.snd.StartScene("base", "base:intro", nil, f, true)
		return true
	}
	resetThumpBeat()
	g.snd.StartThump()
	return false
}

// ---- how loud the synthesised party of the first intro is -------------------------------------------------

// The party next door (the synthesised loop, buildThump) used to be heard only faintly. Its level is
// introThumpVol (the story: it swells, drops when the cloud pops, the door...) times a ramp that grows
// over the scene from introRampFrom to introRampTo, and never above introMusicCap; the player then
// applies thumpGain (audio.go). A song of the intros/ folder keeps its old level (introThumpVol * sceneGain).
const (
	introRampFrom = 1.15 // the ramp at the start of the scene (already louder than before: it was 1.0)
	introRampTo   = 1.35 // and when the cloud pops (inBurst): the party is clearly there
	introMusicCap = 0.9  // the ceiling of the level: present, never on top of the other sounds
)

// introMusicVol: how loud the synthesised party of the first intro is at scene time t (0..introMusicCap).
func introMusicVol(t float64) float64 {
	ramp := introRampFrom + (introRampTo-introRampFrom)*smooth01(t/inBurst)
	return math.Min(introMusicCap, introThumpVol(t)*ramp)
}
