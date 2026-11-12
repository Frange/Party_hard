package main

import (
	"archive/zip"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The old generated DJ/punk_1.mp3 ... punk_6.mp3 are gone from the game folder (the songs after the DJ are
// synthetic now, songs_afterdj.go). The manifests of the versions already handed out still list them: a patch
// from any of those versions must put them in REMOVIDOS.txt (to delete) and must not fail because they are
// missing from the new build.
var djPunk = regexp.MustCompile(`^DJ/punk_\d+\.mp3$`)

func TestOldDJPunkFilesBecomeRemovedInPatches(t *testing.T) {
	files, _ := filepath.Glob(filepath.Join("..", "..", "releases", "manifest-*.json"))
	if len(files) == 0 {
		t.Fatal("no manifests in releases/")
	}
	withPunk := 0
	for _, f := range files {
		base, err := loadManifest(f)
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		var punk []string
		next := base
		next.Files = nil
		for _, e := range base.Files {
			if djPunk.MatchString(e.Path) {
				punk = append(punk, e.Path)
				continue
			}
			next.Files = append(next.Files, e)
		}
		if len(punk) == 0 {
			continue
		}
		withPunk++
		d := computeDiff(base, next)
		if strings.Join(d.Removed, ",") != strings.Join(punk, ",") || len(d.Added)+len(d.Changed) != 0 {
			t.Errorf("%s: removed %v, want %v (added %v, changed %v)", filepath.Base(f), d.Removed, punk, d.Added, d.Changed)
		}
	}
	if withPunk == 0 {
		t.Fatal("no old manifest lists DJ/punk_N.mp3: the check would prove nothing")
	}

	// a full patch: the base had the six songs, the new build has none (and an empty DJ/ folder)
	dir := t.TempDir()
	z0, z1 := filepath.Join(dir, "v0.zip"), filepath.Join(dir, "v1.zip")
	old := baseFiles("1.0.0")
	for i := 1; i <= 6; i++ {
		old["DJ/punk_"+string(rune('0'+i))+".mp3"] = tf{body: "punk"}
	}
	mkzip(t, z0, "", old)
	mkzip(t, z1, "", baseFiles("1.0.1"), "DJ")
	c0, err := openRelease(z0, "win")
	if err != nil {
		t.Fatal(err)
	}
	c0.Reader.Close()
	c1, err := openRelease(z1, "win")
	if err != nil {
		t.Fatal(err)
	}
	defer c1.Reader.Close()
	p := filepath.Join(dir, "p.zip")
	if _, err := writePatch(p, c0.Manifest, c1); err != nil {
		t.Fatalf("patch without the old DJ songs failed: %v", err)
	}
	zr, err := zip.OpenReader(p)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	removed := ""
	for _, f := range zr.File {
		if djPunk.MatchString(f.Name) {
			t.Errorf("the patch carries %s", f.Name)
		}
		if f.Name == "REMOVIDOS.txt" {
			b, _ := readZipFile(f)
			removed = string(b)
		}
	}
	for i := 1; i <= 6; i++ {
		if n := "DJ/punk_" + string(rune('0'+i)) + ".mp3"; !strings.Contains(removed, n+"\r\n") {
			t.Errorf("REMOVIDOS.txt lacks %s: %q", n, removed)
		}
	}
}
