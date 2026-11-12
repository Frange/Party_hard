package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

type tf struct {
	body string
	mode os.FileMode
}

// mkzip writes a fake full release zip (prefix "" for Windows, "mac/" for Mac).
func mkzip(t *testing.T, file, prefix string, files map[string]tf, dirs ...string) {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	when := time.Date(2026, 11, 10, 17, 22, 0, 0, time.UTC)
	if prefix != "" {
		h := &zip.FileHeader{Name: prefix, Modified: when}
		h.SetMode(os.ModeDir | 0o755)
		zw.CreateHeader(h)
	}
	for _, d := range dirs {
		h := &zip.FileHeader{Name: prefix + d + "/", Modified: when}
		h.SetMode(os.ModeDir | 0o755)
		zw.CreateHeader(h)
	}
	for _, n := range names {
		h := &zip.FileHeader{Name: prefix + n, Method: zip.Deflate, Modified: when}
		m := files[n].mode
		if m == 0 {
			m = 0o644
		}
		h.SetMode(m)
		w, err := zw.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(files[n].body))
	}
	zw.Close()
	if err := os.WriteFile(file, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

const changelog = "# Registro\n---\n\n## [v1.0.2] - x - menor\n- tres\n\n---\n\n## [v1.0.1] - x - menor\n- dos\n\n---\n\n## [v1.0.0] - x - grande\n- uno\n"

func baseFiles(ver string) map[string]tf {
	return map[string]tf{
		"VERSION.txt":       {body: "Party Hardy v" + ver + "\n"},
		"CHANGELOG.md":      {body: changelog},
		"PartyHardy.exe":    {body: "exe-" + ver, mode: 0o755},
		"assets/a.png":      {body: "A"},
		"assets/b.png":      {body: "B"},
		"assets/old.png":    {body: "OLD"},
		"intros/x.bpm":      {body: "120"},
		"music/1_house.mp3": {body: "mp3"},
	}
}

func TestDiffAddedChangedRemovedAndDeterminism(t *testing.T) {
	dir := t.TempDir()
	z0, z1 := filepath.Join(dir, "v0.zip"), filepath.Join(dir, "v1.zip")
	mkzip(t, z0, "", baseFiles("1.0.0"), "assets/backgrounds")
	nf := baseFiles("1.0.1")
	nf["assets/a.png"] = tf{body: "A2"}
	nf["assets/new.png"] = tf{body: "NEW"}
	nf["assets/b.png"] = tf{body: "B", mode: 0o755} // permissions change counts as a change
	delete(nf, "assets/old.png")
	mkzip(t, z1, "", nf, "assets/backgrounds", "assets/empty_new")

	c0, err := openRelease(z0, "win")
	if err != nil {
		t.Fatal(err)
	}
	c0.Reader.Close()
	rel := filepath.Join(dir, "releases")
	if _, err := writeManifest(rel, c0.Manifest, false); err != nil {
		t.Fatal(err)
	}
	m0, err := loadManifest(manifestPath(rel, "1.0.0", "win"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(m0.encode(), c0.Manifest.encode()) {
		t.Fatal("manifest does not round-trip")
	}
	if !reflect.DeepEqual(m0.EmptyDirs, []string{"assets/backgrounds"}) {
		t.Fatalf("empty dirs %v", m0.EmptyDirs)
	}
	c1, err := openRelease(z1, "win")
	if err != nil {
		t.Fatal(err)
	}
	defer c1.Reader.Close()
	d := computeDiff(m0, c1.Manifest)
	if !reflect.DeepEqual(d.Added, []string{"assets/new.png"}) ||
		!reflect.DeepEqual(d.Changed, []string{"PartyHardy.exe", "VERSION.txt", "assets/a.png", "assets/b.png"}) ||
		!reflect.DeepEqual(d.Removed, []string{"assets/old.png"}) ||
		!reflect.DeepEqual(d.AddedDirs, []string{"assets/empty_new"}) {
		t.Fatalf("diff %+v", d)
	}

	p1, p2 := filepath.Join(dir, "p1.zip"), filepath.Join(dir, "p2.zip")
	if _, err := writePatch(p1, m0, c1); err != nil {
		t.Fatal(err)
	}
	if _, err := writePatch(p2, m0, c1); err != nil {
		t.Fatal(err)
	}
	b1, _ := os.ReadFile(p1)
	b2, _ := os.ReadFile(p2)
	if !bytes.Equal(b1, b2) {
		t.Fatal("patch is not deterministic")
	}
	if !bytes.Equal(m0.encode(), c0.Manifest.encode()) {
		t.Fatal("manifest not deterministic")
	}
	zr, err := zip.OpenReader(p1)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	got := map[string]string{}
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
		if !f.FileInfo().IsDir() {
			b, _ := readZipFile(f)
			got[f.Name] = string(b)
		}
		if f.Name == "PartyHardy.exe" && f.Mode().Perm() != 0o755 {
			t.Fatalf("exe mode %v", f.Mode())
		}
	}
	for _, must := range []string{"assets/new.png", "assets/a.png", "VERSION.txt", "PartyHardy.exe", "REMOVIDOS.txt", "LEEME_ACTUALIZAR.txt", "VERIFICAR_BASE.txt", "assets/empty_new/"} {
		if _, ok := got[must]; !ok && !contains(names, must) {
			t.Fatalf("patch lacks %s: %v", must, names)
		}
	}
	for _, never := range []string{"music/1_house.mp3", "intros/x.bpm", "CHANGELOG.md", "assets/old.png"} {
		if contains(names, never) {
			t.Fatalf("patch carries unchanged/removed %s", never)
		}
	}
	if got["REMOVIDOS.txt"] != "assets/old.png\r\n" {
		t.Fatalf("REMOVIDOS %q", got["REMOVIDOS.txt"])
	}
	le := got["LEEME_ACTUALIZAR.txt"]
	if !strings.Contains(le, "v1.0.0 -> v1.0.1") || !strings.Contains(le, "- dos") || strings.Contains(le, "- uno") || strings.Contains(le, "- tres") {
		t.Fatalf("LEEME without the right changelog:\n%s", le)
	}
	verHash := ""
	for _, e := range m0.Files {
		if e.Path == "VERSION.txt" {
			verHash = e.SHA256
		}
	}
	if verHash == "" || !strings.Contains(got["VERIFICAR_BASE.txt"], verHash+"  VERSION.txt") {
		t.Fatal("VERIFICAR_BASE without hashes")
	}
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

func TestForbiddenFilesRejected(t *testing.T) {
	for _, p := range []string{"assets/progress.json", "assets/progress.json.bak", "assets/settings.json", "assets/editor_prefs.json",
		"imports/x.png", "AI_with_Claude/a.md", "assets_backup_1/a.png", "assets/.DS_Store", "intros/song.mp3", "__MACOSX/x", "assets/._a.png"} {
		if forbiddenReason(p) == "" {
			t.Errorf("%s should be forbidden", p)
		}
	}
	for _, p := range []string{"assets/a.png", "intros/x.bpm", "intros/README.md", "music/1.mp3", "PartyHardy.app/Contents/Info.plist"} {
		if forbiddenReason(p) != "" {
			t.Errorf("%s should be allowed", p)
		}
	}
	dir := t.TempDir()
	f := baseFiles("1.0.0")
	f["assets/progress.json"] = tf{body: "{}"}
	z := filepath.Join(dir, "bad.zip")
	mkzip(t, z, "", f)
	if _, err := openRelease(z, "win"); err == nil {
		t.Fatal("a zip with progress.json must be refused")
	}
}

func TestMacSignatureTravelsWithApp(t *testing.T) {
	files := func(ver, plist string) map[string]tf {
		return map[string]tf{
			"VERSION.txt": {body: "Party Hardy v" + ver + "\n"},
			"PartyHardy.app/Contents/MacOS/PartyHardy":             {body: "bin", mode: 0o755},
			"PartyHardy.app/Contents/Info.plist":                   {body: plist},
			"PartyHardy.app/Contents/_CodeSignature/CodeResources": {body: "seal"},
			"PartyHardy.app/Contents/Resources/AppIcon.icns":       {body: "icon"},
			"assets/a.png": {body: "A"},
		}
	}
	dir := t.TempDir()
	z0, z1 := filepath.Join(dir, "m0.zip"), filepath.Join(dir, "m1.zip")
	mkzip(t, z0, "mac/", files("1.0.0", "p"))
	mkzip(t, z1, "mac/", files("1.0.1", "p2"))
	c0, err := openRelease(z0, "mac")
	if err != nil {
		t.Fatal(err)
	}
	c0.Reader.Close()
	if c0.Manifest.ZipPrefix != "mac/" || c0.Manifest.Files[0].Path != "PartyHardy.app/Contents/Info.plist" {
		t.Fatalf("prefix not stripped: %+v", c0.Manifest)
	}
	c1, err := openRelease(z1, "mac")
	if err != nil {
		t.Fatal(err)
	}
	c1.Reader.Close()
	d := computeDiff(c0.Manifest, c1.Manifest)
	want := []string{"PartyHardy.app/Contents/Info.plist", "PartyHardy.app/Contents/MacOS/PartyHardy",
		"PartyHardy.app/Contents/_CodeSignature/CodeResources", "VERSION.txt"}
	if !reflect.DeepEqual(d.Changed, want) {
		t.Fatalf("changed %v want %v", d.Changed, want)
	}
}

func TestMissingBaseManifestAndVersions(t *testing.T) {
	dir := t.TempDir()
	z := filepath.Join(dir, "v1.zip")
	mkzip(t, z, "", baseFiles("1.0.1"))
	err := run([]string{"-from", "0.9.0", "-win", z, "-dir", filepath.Join(dir, "releases"), "-out", dir})
	if err == nil || !strings.Contains(err.Error(), "no existe el manifiesto") {
		t.Fatalf("want missing manifest error, got %v", err)
	}
	if err := run([]string{"manifest", "-win", z, "-dir", filepath.Join(dir, "releases")}); err != nil {
		t.Fatal(err)
	}
	// same version, different content: never overwritten silently
	z2 := filepath.Join(dir, "v1b.zip")
	f := baseFiles("1.0.1")
	f["assets/a.png"] = tf{body: "changed"}
	mkzip(t, z2, "", f)
	if err := run([]string{"manifest", "-win", z2, "-dir", filepath.Join(dir, "releases")}); err == nil {
		t.Fatal("a published manifest must not be overwritten without -force")
	}
	if err := run([]string{"-from", "1.0.1", "-win", z, "-dir", filepath.Join(dir, "releases"), "-out", dir}); err == nil {
		t.Fatal("from == to must fail")
	}
	if err := run([]string{"-from", "0.9.0", "-to", "2.0.0", "-win", z, "-dir", filepath.Join(dir, "releases"), "-out", dir}); err == nil {
		t.Fatal("-to different from the zip must fail")
	}
	if got := changelogBetween(changelog, "1.0.0", "1.0.2"); !strings.Contains(got, "- dos") || !strings.Contains(got, "- tres") || strings.Contains(got, "- uno") {
		t.Fatalf("changelogBetween %q", got)
	}
	if !semverLess("1.0.9", "1.0.10") || semverLess("2.0.0", "1.9.9") {
		t.Fatal("semver order")
	}
}
