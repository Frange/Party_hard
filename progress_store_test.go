package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// progressTempDir: a temp folder with assets/, used as the working dir (never the real progress.json).
func progressTempDir(t *testing.T) string {
	t.Helper()
	oldFrozen, oldNo := progressFrozen, noAssets
	wd, _ := os.Getwd()
	t.Cleanup(func() { progressFrozen, noAssets = oldFrozen, oldNo; os.Chdir(wd) })
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, "assets"), 0o755)
	os.Chdir(dir)
	progressFrozen, noAssets = false, false
	path := progressPath()
	want, _ := filepath.EvalSymlinks(filepath.Join(dir, "assets"))
	if got, _ := filepath.EvalSymlinks(filepath.Dir(path)); got != want {
		t.Fatalf("progressPath outside the temp dir: %q", path)
	}
	return path
}

func sampleProgress() Progress {
	return Progress{Unlocked: 7, Wallet: 1234, Stats: map[string]int{"played": 9, "won_3": 1},
		Trophies: map[string]bool{"first": true}, Owned: []string{"hat_a"}, Equip: map[string]string{"hat": "hat_a"},
		CharKills: map[string]int{"yoda": 2}, Started: 1794158654}
}

func TestProgressRoundTripObfuscated(t *testing.T) {
	path := progressTempDir(t)
	saveProgress(sampleProgress())
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte(progressMagic)) || bytes.Contains(data, []byte("wallet")) || bytes.Contains(data, []byte("1234")) {
		t.Fatalf("the save is readable: %q", data)
	}
	p := loadProgress()
	if p.Unlocked != 7 || p.Wallet != 1234 || p.Stats["played"] != 9 || !p.Trophies["first"] || p.CharKills["yoda"] != 2 ||
		len(p.Owned) != 1 || p.Equip["hat"] != "hat_a" {
		t.Fatalf("round trip lost data: %+v", p)
	}
	if p.Started != 1794158654 {
		t.Fatalf("Started not kept: %d", p.Started)
	}
	c := cloneProgress(p)
	if c.Started != p.Started {
		t.Fatal("cloneProgress lost Started")
	}
}

func TestProgressLegacyPlainJSONLoadsAndIsRewritten(t *testing.T) {
	path := progressTempDir(t)
	old := map[string]any{"unlocked": 5, "wallet": 77, "stats": map[string]int{"played": 3}, "campaignRev": 1}
	js, _ := json.MarshalIndent(old, "", "  ")
	os.WriteFile(path, js, 0o644)
	mt := time.Unix(1794158654, 0)
	os.Chtimes(path, mt, mt)
	p := loadProgress()
	if p.Unlocked != 5 || p.Wallet != 77 || p.Stats["played"] != 3 {
		t.Fatalf("old plain save not loaded: %+v", p)
	}
	if p.Started <= 0 || p.Started > 1794158654 {
		t.Fatalf("Started of an old save should come from the file age: %d", p.Started)
	}
	saveProgress(p) // the next normal save writes the new format, and the plain one becomes the .bak
	data, _ := os.ReadFile(path)
	if !bytes.HasPrefix(data, []byte(progressMagic)) {
		t.Fatal("not rewritten obfuscated")
	}
	if b, _ := os.ReadFile(path + ".bak"); !bytes.Equal(b, js) {
		t.Fatal(".bak should hold the previous good file")
	}
	if q := loadProgress(); q.Wallet != 77 || q.Started != p.Started {
		t.Fatalf("reload: %+v", q)
	}
	if _, err := os.Stat(path + ".tmp"); err == nil {
		t.Fatal("tmp file left behind")
	}
}

// A file changed by hand (one byte) is rejected: the .bak is used, or a new game; never the edited data.
func TestProgressTamperedFallsBackToBak(t *testing.T) {
	path := progressTempDir(t)
	a := sampleProgress()
	a.Wallet = 10
	saveProgress(a)
	b := sampleProgress()
	b.Wallet = 20
	saveProgress(b) // progress.json = 20, .bak = 10
	data, _ := os.ReadFile(path)
	for i := len(progressMagic) + 5; i < len(data)-2; i += 7 {
		bad := append([]byte(nil), data...)
		if bad[i] == 'A' {
			bad[i] = 'B'
		} else {
			bad[i] = 'A'
		}
		os.WriteFile(path, bad, 0o644)
		p := loadProgress()
		if p.Wallet != 10 {
			t.Fatalf("byte %d changed: wallet %d, want the .bak (10)", i, p.Wallet)
		}
	}
	// a plain JSON written by hand over a signed save (with a signed .bak) is not accepted either
	os.WriteFile(path, []byte(`{"unlocked":24,"wallet":999999}`), 0o644)
	if p := loadProgress(); p.Wallet == 999999 || p.Unlocked == 24 {
		t.Fatalf("hand-written JSON unlocked things: %+v", p)
	}
	// the .bak broken too: a new game
	os.WriteFile(path+".bak", []byte("garbage"), 0o644)
	os.WriteFile(path, data[:len(data)-10], 0o644)
	if p := loadProgress(); p.Wallet != 0 || p.Unlocked != 1 {
		t.Fatalf("both broken: want a new game, got %+v", p)
	}
	// saving over a corrupt file never puts the corrupt one in the .bak
	saveProgress(Progress{Unlocked: 2})
	if bk, _ := os.ReadFile(path + ".bak"); string(bk) != "garbage" {
		t.Fatal("a corrupt file was copied to .bak")
	}
}

func TestProgressEmptyOrGarbageNoCrash(t *testing.T) {
	path := progressTempDir(t)
	for _, junk := range [][]byte{nil, []byte(" "), []byte("{"), []byte(progressMagic), []byte(progressMagic + "!!!!"),
		[]byte(progressMagic + "QUJD"), bytes.Repeat([]byte{0xff, 0, 7}, 999), []byte(strings.Repeat("[", 5000))} {
		os.WriteFile(path, junk, 0o644)
		p := loadProgress()
		if p.Unlocked != 1 || p.Wallet != 0 {
			t.Fatalf("junk %q: got %+v", junk, p)
		}
		if p.Started == 0 {
			t.Fatal("a new game must get its start time")
		}
	}
}

func TestProgressNewGameStartedAndCareerLine(t *testing.T) {
	path := progressTempDir(t)
	before := time.Now().Unix()
	p := loadProgress()
	if p.Started < before-1 || p.Started > time.Now().Unix()+1 {
		t.Fatalf("new game Started = %d", p.Started)
	}
	saveProgress(p)
	if q := loadProgress(); q.Started != p.Started {
		t.Fatalf("Started changed across save/load: %d vs %d", q.Started, p.Started)
	}
	_ = path
	ts := time.Date(2026, 11, 9, 11, 7, 0, 0, time.Local).Unix()
	if got := careerSinceLine(ts); got != "Toda tu carrera: desde el 9/11/2026, 11:07." {
		t.Fatalf("line = %q", got)
	}
	old := uiLang
	defer func() { uiLang = old }()
	uiLang = "en"
	if got := tr(careerSinceLine(ts)); got != "Your whole career: since 9/11/2026, 11:07." {
		t.Fatalf("EN line = %q", got)
	}
	if got := careerSinceLine(0); got != "Toda tu carrera: se cuenta desde la primera fiesta." {
		t.Fatalf("no date: %q", got)
	}
}
