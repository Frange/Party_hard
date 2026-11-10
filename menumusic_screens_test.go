package main

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/bitmapfont/v3"
	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// ---------------------------------------------------------------------------
// The screens that hang from the main menu keep the menu's song (same player, no cut, no restart), in the
// four menu tiers; the ones opened from the pause window keep the song of the party; and the title never
// waits for the menu's track: the picture runs and the song comes in as soon as it is built.
// ---------------------------------------------------------------------------

var menuTierNames = []string{"normal", "bronce", "plata", "oro"}

// musicWatch: what the music does frame after frame (the track asked for, the player, the crossfade).
type musicWatch struct {
	t      *testing.T
	g      *Game
	key    string
	player any
	log    []string
}

func newMusicWatch(t *testing.T, g *Game) *musicWatch {
	return &musicWatch{t: t, g: g, key: g.snd.cur, player: g.snd.music}
}

// frames runs n frames of syncMusic (the first thing Game.Update does) after `step` (the screen's own frame);
// the track, the player and the crossfade must stay as they were.
func (w *musicWatch) frames(where string, n int, step func()) {
	w.t.Helper()
	s := w.g.snd
	for i := 0; i < n; i++ {
		w.g.syncMusic()
		if step != nil {
			step()
		}
		if s.cur != w.key || any(s.music) != w.player {
			w.t.Fatalf("%s, frame %d: the music changed to %q (was %q; player changed: %v)", where, i, s.cur, w.key, any(s.music) != w.player)
		}
		if s.fadePend != "" || s.fadeDip > 1e-9 {
			w.t.Fatalf("%s, frame %d: the song is fading (pending %q, dip %.3f)", where, i, s.fadePend, s.fadeDip)
		}
	}
	w.log = append(w.log, where+"="+s.cur)
}

// every screen reachable from the main menu, in every tier: the menu's track on the same player, all along
func TestMenuScreensKeepTheMenuMusic(t *testing.T) {
	g := menuFlowGame(t)
	g.progress.Stats = map[string]int{"played": 3, "seen_intro": 1}
	t.Cleanup(func() {
		g.dbg.tier, g.pause, g.mode, g.helpOv, g.trophyOv, g.sceneReplay = 0, pauseState{}, ModeMenu, false, false, false
	})
	partyKey := ""
	if sp := playlistFor(g.lvl.Def.Title)[0]; sp.style != "" {
		partyKey = trackKey(sp.style, int(sp.seed%1000)*10+sp.variant, g.lvl.Def.Title)
	}
	type screen struct {
		name       string
		set, unset func()
	}
	mode := func(m Mode) func() { return func() { g.mode = m } }
	back := func() { g.mode = ModeMenu }
	screens := []screen{
		{"Tienda", mode(ModeShop), back},
		{"Trofeos/Objetivos/Album", mode(ModeTrophies), back},
		{"Ajustes", mode(ModeSettings), back},
		{"Elige la fiesta", mode(ModeLevelSelect), back},
		{"Escenas", mode(ModeScenes), back},
		{"Confirmar", mode(ModeConfirm), back},
		{"DEBUG", mode(ModeDebug), back},
		{"Editor", mode(ModeEditor), back},
		{"Ayuda", func() { g.helpOv = true }, func() { g.helpOv = false }},
		{"Trofeos (ventana)", func() { g.trophyOv = true }, func() { g.trophyOv = false }},
	}
	for tier := tierNormal; tier <= tierGold; tier++ {
		g.dbg.tier = tier + 1
		key, kind := menuMusicFor(tier)
		ambientPCM(kind)
		g.mode = ModeMenu
		g.snd.StartMenuMusicNow(tier)
		for i := 0; i < 120; i++ { // the fade in of the menu's song
			g.syncMusic()
		}
		w := newMusicWatch(t, g)
		if w.key != key || w.player == nil {
			t.Fatalf("%s: the menu plays %q, not %q", menuTierNames[tier], g.snd.cur, key)
		}
		w.frames(menuTierNames[tier]+" menu", 30, nil)
		for _, sc := range screens {
			sc.set()
			w.frames(menuTierNames[tier]+" "+sc.name, 150, nil)
			sc.unset()
			w.frames(menuTierNames[tier]+" back to the menu from "+sc.name, 90, nil)
		}
		// a cinematic watched again from the scenes list: its own music; the menu's song waits paused (leaveScene
		// resumes it) and the party song never starts
		for _, m := range []Mode{ModeIntro, ModeInterlude, ModeEnding} {
			g.mode, g.sceneReplay = ModeScenes, false
			w.frames(menuTierNames[tier]+" Escenas", 10, nil)
			g.mode, g.sceneReplay = m, true
			w.frames(fmt.Sprintf("%s replayed scene (mode %d)", menuTierNames[tier], m), 150, nil)
			g.mode, g.sceneReplay = ModeScenes, false
			g.snd.StartMusic() // leaveScene
			w.frames(menuTierNames[tier]+" Escenas after the scene", 90, nil)
			g.mode = ModeMenu
		}
		w.frames(menuTierNames[tier]+" menu at the end", 60, nil)
		if partyKey != "" && strings.Contains(strings.Join(w.log, " "), partyKey) {
			t.Fatalf("%s: the song of the first party was asked for: %v", menuTierNames[tier], w.log)
		}
		t.Logf("%s: %d screens, one track %q, one player", menuTierNames[tier], len(w.log), key)
	}
}

// the pause window: AJUSTES keeps the song of the party, SALIR A ELEGIR FIESTA keeps it in the party list (the
// first party loaded behind it does not start its song); back in the menu the menu's song takes over
func TestPauseScreensKeepThePartyMusic(t *testing.T) {
	g, _ := pauseGame(t, false)
	oldSnd, oldLib, oldNext, oldIdx := g.snd, g.lib, g.nextSong, g.songIdx
	t.Cleanup(func() {
		g.snd.StopMusic()
		g.snd, g.lib, g.nextSong, g.songIdx, g.dbg.tier, g.musicMode = oldSnd, oldLib, oldNext, oldIdx, 0, ""
		g.loadLevel(0)
		g.mode = ModeMenu
	})
	g.snd = newSounds(true)
	g.lib = &musicLib{level: map[int]string{}, bands: map[string][]string{}}
	first := g.lvl.Def.Title
	lvl := -1
	for i := 1; i < campaignN; i++ {
		if genreOfTitle[levelDefs[i].Title] != genreOfTitle[first] {
			lvl = i
			break
		}
	}
	if lvl < 0 {
		t.Fatal("no party with a style other than the first one's")
	}
	for tier := tierNormal; tier <= tierGold; tier++ {
		g.dbg.tier = tier + 1
		g.loadLevel(lvl)
		g.lvl.graded = true
		g.mode, g.pause = ModePlay, pauseState{}
		g.nextSong, g.songIdx = 1e9, 0
		partySong := func() bool { // its style, or the band that came to the party
			return g.snd.cur != "" && !strings.HasPrefix(g.snd.cur, menuKeyAnthem)
		}
		for i := 0; i < 800 && !partySong(); i++ { // the party's song is built by a thread
			g.syncMusic()
			time.Sleep(10 * time.Millisecond)
		}
		if !partySong() || strings.Contains(g.snd.cur, first) {
			t.Fatalf("%s: the party %q never got its song (%q)", menuTierNames[tier], g.lvl.Def.Title, g.snd.cur)
		}
		w := newMusicWatch(t, g)
		name := menuTierNames[tier]
		w.frames(name+" party", 30, nil)
		g.pause.open = true
		w.frames(name+" pause window", 60, nil)
		g.pauseChoose(pauseRowSettings)
		if g.mode != ModeSettings {
			t.Fatal("AJUSTES did not open the settings")
		}
		w.frames(name+" Ajustes from the pause", 150, g.pauseAfterSettings)
		g.mode = ModeMenu // ATRÁS of the settings, in the same frame pauseAfterSettings takes it back to the pause
		g.pauseAfterSettings()
		if g.mode != ModePlay || !g.pause.open {
			t.Fatal("leaving the settings did not come back to the pause window")
		}
		w.frames(name+" pause window again", 30, nil)
		g.pauseChoose(pauseRowQuit)
		if g.mode != ModeLevelSelect || g.lvl.Def.Title != first {
			t.Fatalf("SALIR A ELEGIR FIESTA: mode %d, level %q", g.mode, g.lvl.Def.Title)
		}
		w.frames(name+" Elige la fiesta from the pause", 180, nil)
		if strings.Contains(g.snd.cur, first) {
			t.Fatalf("%s: the first party's song started in the list", name)
		}
		g.mode = ModeMenu // Esc in the list: the main menu, with its own song (a crossfade)
		key, kind := menuMusicFor(tier)
		ambientPCM(kind)
		for i := 0; i < 60*3; i++ {
			g.syncMusic()
		}
		if g.snd.cur != key || g.pause.keepSong {
			t.Fatalf("%s: back in the menu the music is %q (keep %v), not %q", name, g.snd.cur, g.pause.keepSong, key)
		}
		t.Logf("%s: %v, then the menu %q", name, w.log, key)
	}
}

// the title never waits for the menu's track: with the track still being built every frame (update and draw)
// takes far less than 100 ms, the picture runs, and the song starts as soon as it is there, where the picture
// is, and goes on into the menu on the same player
func TestTitleDoesNotWaitForTheAnthem(t *testing.T) {
	for tier := tierNormal; tier <= tierGold; tier++ {
		g := menuFlowGame(t)
		g.progress.Stats = map[string]int{"played": 3, "seen_intro": 1}
		g.dbg.tier = tier + 1
		g.face = text.NewGoXFace(bitmapfont.Face)
		_, kind := menuMusicFor(tier)
		real := ambientPCM(kind)
		// the track is not built yet: its build waits until `release` (the slow machine, the first second)
		release := make(chan struct{})
		var once bool
		oldFn := ambBuildFn
		ambBuildFn = func(k string) []byte {
			if k == kind {
				<-release
				return real
			}
			return oldFn(k)
		}
		ambMu.Lock()
		delete(ambCache, kind)
		ambMu.Unlock()
		ambReady.Delete(kind)
		ambWarming.Delete(kind)
		t.Cleanup(func() {
			if !once {
				close(release)
			}
			ambBuildFn = oldFn
			ambientPCM(kind)
		})
		g.snd = newSounds(true)
		g.rng = rand.New(rand.NewSource(1))
		screen := ebiten.NewImage(screenW, screenH)
		if px == nil { // the white pixel every rectangle is drawn with (newGame makes it)
			wb := newPix(1, 1)
			wb.p[0], wb.p[1], wb.p[2], wb.p[3] = 255, 255, 255, 255
			px = wb.img()
		}
		g.startTitle()
		const wait = 40 // frames before the track is there (0.67 s)
		// the frame of the game: Update (syncMusic, then the title) must never wait for the track. Draw is called too
		// (it never touches the music); its time in a test without a window is the software queue of ebiten, only logged
		worst, worstDraw := time.Duration(0), time.Duration(0)
		frame := func() {
			t0 := time.Now()
			g.syncMusic()
			g.updateTitle()
			t1 := time.Now()
			if d := t1.Sub(t0); d > worst {
				worst = d
			}
			if g.mode == ModeTitle {
				g.drawTitle(screen)
			}
			if d := time.Since(t1); d > worstDraw {
				worstDraw = d
			}
		}
		for i := 0; i < wait; i++ {
			frame()
			if g.snd.music != nil || g.title.musicOn {
				t.Fatalf("%s: the music started before its track was built", menuTierNames[tier])
			}
		}
		if worst > 100*time.Millisecond {
			t.Fatalf("%s: a title update took %v while the track was being built", menuTierNames[tier], worst)
		}
		if g.title.t < float64(wait-1)/60 {
			t.Fatalf("%s: the picture of the title did not run (t = %.2f)", menuTierNames[tier], g.title.t)
		}
		waitWorst, waitDraw := worst, worstDraw
		worst = 0
		once = true
		close(release)
		for i := 0; i < 200 && ambientReady(kind) == nil; i++ {
			time.Sleep(5 * time.Millisecond)
		}
		frame()
		key, _ := menuMusicFor(tier)
		p := g.snd.music
		if p == nil || g.snd.cur != key || !g.title.musicOn {
			t.Fatalf("%s: the track is built but the music did not start (%q)", menuTierNames[tier], g.snd.cur)
		}
		at := p.Position().Seconds()
		if want := float64(wait) / 60; at < want-0.05 || at > want+0.05 {
			t.Errorf("%s: the song starts at %.3f s, not where the picture is (%.3f s)", menuTierNames[tier], at, want)
		}
		for i := 0; g.mode == ModeTitle && i < 60*12; i++ {
			frame()
			if g.snd.music != p || g.snd.cur != key {
				t.Fatalf("%s: the title switched the music (%q)", menuTierNames[tier], g.snd.cur)
			}
		}
		if g.mode != ModeMenu {
			t.Fatalf("%s: the title did not lead to the menu (mode %d)", menuTierNames[tier], g.mode)
		}
		for i := 0; i < 60*4; i++ {
			g.syncMusic()
			if g.snd.music != p || g.snd.cur != key || g.snd.fadePend != "" {
				t.Fatalf("%s: menu, %.2f s: the music changed (%q)", menuTierNames[tier], float64(i)/60, g.snd.cur)
			}
		}
		if worst > 100*time.Millisecond {
			t.Errorf("%s: a title update took %v after the track was built", menuTierNames[tier], worst)
		}
		t.Logf("%s: slowest update while the track was being built %v (draw %v), after it %v; the song came in at %.3f s on the same player into the menu", menuTierNames[tier], waitWorst, waitDraw, worst, at)
	}
}

// the kinds build each in its own thread now (the menu's track must not queue behind the scene songs): built
// at the same time they come out exactly as built one after the other (run it with -race too)
func TestAmbientKindsBuildInParallel(t *testing.T) {
	kinds := []string{ambCalm, ambSleep, ambientKinds[0]}
	want := map[string][]byte{}
	for _, k := range kinds {
		want[k] = ambientPCM(k)
	}
	for _, k := range kinds {
		ambMu.Lock()
		delete(ambCache, k)
		ambMu.Unlock()
		ambReady.Delete(k)
	}
	done := make(chan string)
	for _, k := range kinds {
		go func(k string) { ambientPCM(k); done <- k }(k)
	}
	for range kinds {
		<-done
	}
	for _, k := range kinds {
		got := ambientReady(k)
		if string(got) != string(want[k]) {
			t.Errorf("%s: built in parallel it is not the same song (%d vs %d bytes)", k, len(got), len(want[k]))
		}
	}
}
