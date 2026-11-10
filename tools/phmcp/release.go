package main

// build_release: the tool of the BUILD role. It checks the format, runs go vet, runs
// ./build.sh and verifies the two zips: they must not carry the player's save, the
// editor preferences or the imports folder, and they must hold the game and the editor.
// It also tells whether the binaries in the project root are older than the ones in dist/.
//
// Safety: the only thing written is what build.sh itself writes (dist/ and the two zips).
// It never copies into the project root, never touches assets/, imports/, intros/,
// music/ or progress.json. Time limits everywhere. verify_only=true builds nothing.

import (
	"archive/zip"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func init() {
	tools = append(tools, &tool{
		Name: "build_release",
		Description: "BUILD role. Runs gofmt -l, go vet and ./build.sh (writes only dist/ and the two zips), then verifies the zips (no progress.json, settings.json, editor_prefs.json, imports/ inside; game and editor present), prints sizes and times, and says whether the binaries in the project root are stale compared with dist/. " +
			"With verify_only=true it only verifies the zips and the root binaries. It never copies into the project root: the BUILD agent does that with cp. Time limit default 900 s, max 1800.",
		InputSchema: obj(map[string]any{
			"verify_only": map[string]any{"type": "boolean", "description": "do not build, only verify existing zips and root binaries"},
			"seconds":     map[string]any{"type": "integer", "description": "time limit for the whole build, default 900, max 1800"},
		}),
		run: buildRelease,
	})
}

// files that must never travel inside a package
var forbiddenInZip = map[string]bool{"progress.json": true, "settings.json": true, "editor_prefs.json": true}

type zipReport struct {
	name    string
	size    int64
	mod     time.Time
	entries int
	mp3     int
	version string // the text of VERSION.txt inside the zip ("" if there is none)
	issues  []string
}

// verifyZip opens a zip and reports what must not be there and what must be there.
// want: suffixes of entries that have to exist (the game and the editor).
func verifyZip(path string, want []string) *zipReport {
	rep := &zipReport{name: filepath.Base(path)}
	st, err := os.Stat(path)
	if err != nil {
		rep.issues = append(rep.issues, "missing: "+err.Error())
		return rep
	}
	rep.size, rep.mod = st.Size(), st.ModTime()
	zr, err := zip.OpenReader(path)
	if err != nil {
		rep.issues = append(rep.issues, "not a valid zip: "+err.Error())
		return rep
	}
	defer zr.Close()
	found := map[string]bool{}
	for _, f := range zr.File {
		rep.entries++
		name := strings.ReplaceAll(f.Name, "\\", "/")
		parts := strings.Split(strings.Trim(name, "/"), "/")
		base := parts[len(parts)-1]
		if forbiddenInZip[strings.ToLower(base)] {
			rep.issues = append(rep.issues, "FORBIDDEN file inside: "+name)
		}
		for _, p := range parts[:len(parts)-1] {
			if p == "imports" || strings.HasPrefix(p, "assets_backup_") {
				rep.issues = append(rep.issues, "FORBIDDEN folder inside: "+name)
				break
			}
		}
		if strings.HasSuffix(strings.ToLower(name), ".mp3") {
			rep.mp3++
		}
		if base == "VERSION.txt" && len(parts) <= 2 {
			if rc, err := f.Open(); err == nil {
				b, _ := io.ReadAll(io.LimitReader(rc, 256))
				rc.Close()
				rep.version = strings.TrimSpace(string(b))
			}
		}
		for _, w := range want {
			if strings.HasSuffix(name, w) {
				found[w] = true
			}
		}
	}
	for _, w := range want {
		if !found[w] {
			rep.issues = append(rep.issues, "missing in the zip: "+w)
		}
	}
	if len(rep.issues) > 12 {
		rep.issues = append(rep.issues[:12], fmt.Sprintf("... and %d more", len(rep.issues)-12))
	}
	return rep
}

func (z *zipReport) line() string {
	status := "ok"
	if len(z.issues) > 0 {
		status = "PROBLEMS"
	}
	if z.size == 0 && len(z.issues) > 0 {
		return fmt.Sprintf("%s: %s\n  %s", z.name, status, strings.Join(z.issues, "\n  "))
	}
	s := fmt.Sprintf("%s: %s, %.1f MB, %d entries (%d mp3), written %s", z.name, status, float64(z.size)/1e6, z.entries, z.mp3, z.mod.Format("2006-01-02 15:04"))
	if len(z.issues) > 0 {
		s += "\n  " + strings.Join(z.issues, "\n  ")
	}
	return s
}

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// rootBinaryPairs: binary in the project root -> the same one built into dist/.
var rootBinaryPairs = [][2]string{
	{"PartyHardy.exe", "dist/win/PartyHardy.exe"},
	{"PartyHardyEditor.exe", "dist/win/PartyHardyEditor.exe"},
	{"PartyHardy.app/Contents/MacOS/PartyHardy", "dist/mac/PartyHardy.app/Contents/MacOS/PartyHardy"},
	{"PartyHardyEditor.app/Contents/MacOS/PartyHardyEditor", "dist/mac/PartyHardyEditor.app/Contents/MacOS/PartyHardyEditor"},
}

// staleRootBinaries lists the root binaries that differ from dist/ (or are missing).
func staleRootBinaries(root string) []string {
	var stale []string
	for _, p := range rootBinaryPairs {
		a, errA := fileHash(filepath.Join(root, p[0]))
		b, errB := fileHash(filepath.Join(root, p[1]))
		switch {
		case errB != nil:
			stale = append(stale, p[0]+" (no dist/ copy to compare)")
		case errA != nil || a != b:
			stale = append(stale, p[0])
		}
	}
	return stale
}

func buildRelease(root string, a map[string]any) (string, error) {
	if b, err := os.ReadFile(filepath.Join(root, "go.mod")); err != nil || !strings.HasPrefix(string(b), "module partyhardy") {
		return "", fmt.Errorf("%s does not look like the Party Hardy project", root)
	}
	verifyOnly, _ := a["verify_only"].(bool)
	secs := argInt(a, "seconds", 900)
	if secs < 60 {
		secs = 60
	}
	if secs > 1800 {
		secs = 1800
	}
	limit := time.Duration(secs) * time.Second
	start := time.Now()
	var sb strings.Builder
	sb.WriteString("BUILD REPORT\n")
	if !verifyOnly {
		out, err := runLimited(root, 120*time.Second, nil, "gofmt", "-l", ".")
		if err != nil {
			fmt.Fprintf(&sb, "gofmt: FAILED to run (%v)\n", err)
		} else {
			var bad []string
			for _, f := range strings.Fields(out) {
				if f != "assets.go" && f != "props.go" { // these two were already unformatted
					bad = append(bad, f)
				}
			}
			if len(bad) == 0 {
				sb.WriteString("gofmt: ok (assets.go and props.go were already unformatted and are ignored)\n")
			} else {
				fmt.Fprintf(&sb, "gofmt: NOT FORMATTED: %s\n", strings.Join(bad, ", "))
			}
		}
		vet, err := runLimited(root, limit, nil, "go", "vet", ".")
		if err != nil {
			fmt.Fprintf(&sb, "go vet: FAILED (%v)\n%s\nBuild not attempted.\n", err, tail(vet, 20))
			return sb.String(), nil
		}
		sb.WriteString("go vet: ok\n")
		rest := limit - time.Since(start)
		if rest < 60*time.Second {
			rest = 60 * time.Second
		}
		bout, err := runLimited(root, rest, nil, "bash", "build.sh")
		if err != nil {
			fmt.Fprintf(&sb, "build.sh: FAILED (%v)\n%s\n", err, tail(bout, 25))
			return sb.String(), nil
		}
		fmt.Fprintf(&sb, "build.sh: ok (%s)\n", tail(bout, 1))
	}
	reps := []*zipReport{
		verifyZip(filepath.Join(root, "PartyHardy-Mac.zip"), []string{"PartyHardy.app/Contents/MacOS/PartyHardy", "PartyHardyEditor.app/Contents/MacOS/PartyHardyEditor"}),
		verifyZip(filepath.Join(root, "PartyHardy-Windows.zip"), []string{"PartyHardy.exe", "PartyHardyEditor.exe"}),
	}
	bad := false
	for _, r := range reps {
		sb.WriteString(r.line() + "\n")
		if len(r.issues) > 0 {
			bad = true
		}
	}
	if line, ok := versionCheck(root, reps); ok {
		sb.WriteString(line + "\n")
	} else {
		sb.WriteString(line + "\n")
		bad = true
	}
	if stale := staleRootBinaries(root); len(stale) > 0 {
		fmt.Fprintf(&sb, "ROOT BINARIES STALE (copy them from dist/ with cp, then codesign the mac ones): %s\n", strings.Join(stale, ", "))
	} else {
		sb.WriteString("root binaries: up to date with dist/\n")
	}
	if bad {
		sb.WriteString("RESULT: PROBLEMS in the packages (see above). Do not hand them out.\n")
	} else {
		sb.WriteString("RESULT: packages ok\n")
	}
	fmt.Fprintf(&sb, "(%s)", time.Since(start).Round(time.Second))
	return sb.String(), nil
}

var _ = sort.Strings

// projectVersions reads the version constant of version.go and the first "## [vX.Y.Z]" header of CHANGELOG.md.
func projectVersions(root string) (code, changelog string) {
	if b, err := os.ReadFile(filepath.Join(root, "version.go")); err == nil {
		for _, l := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(l, "const gameVersion = \"") {
				code = strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(l), "const gameVersion = \""), "\"")
			}
		}
	}
	if b, err := os.ReadFile(filepath.Join(root, "CHANGELOG.md")); err == nil {
		for _, l := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(l, "## [v") {
				if i := strings.Index(l, "]"); i > 5 {
					changelog = l[5:i]
				}
				break
			}
		}
	}
	return
}

// versionCheck: the version constant, the CHANGELOG header and the VERSION.txt of every zip must agree.
func versionCheck(root string, reps []*zipReport) (string, bool) {
	code, cl := projectVersions(root)
	ok := code != "" && code == cl
	parts := []string{"version.go " + code, "CHANGELOG " + cl}
	for _, r := range reps {
		v := "" // VERSION.txt says "Party Hardy v1.0.0": the last word without the v
		if f := strings.Fields(r.version); len(f) > 0 {
			v = strings.TrimPrefix(f[len(f)-1], "v")
		}
		if v == "" {
			v = "(no VERSION.txt)"
		}
		if v != code {
			ok = false
		}
		parts = append(parts, r.name+" "+v)
	}
	if ok {
		return "VERSION: ok, all v" + code + " (" + strings.Join(parts, ", ") + ")", true
	}
	return "VERSION: MISMATCH (" + strings.Join(parts, ", ") + "): raise the version and the CHANGELOG together, then rebuild", false
}
