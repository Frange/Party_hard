package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---- qa_run_tests: the summary of `go test -json` -------------------------------

const sampleJSON = `{"Action":"run","Package":"partyhard","Test":"TestA"}
{"Action":"output","Package":"partyhard","Test":"TestA","Output":"=== RUN   TestA\\n"}
{"Action":"pass","Package":"partyhard","Test":"TestA","Elapsed":0.2}
{"Action":"run","Package":"partyhard","Test":"TestB"}
{"Action":"output","Package":"partyhard","Test":"TestB","Output":"    b_test.go:12: expected 3, got 4\\n"}
{"Action":"fail","Package":"partyhard","Test":"TestB","Elapsed":1.5}
{"Action":"run","Package":"partyhard","Test":"TestC"}
{"Action":"output","Package":"partyhard","Test":"TestC","Output":"    c_test.go:5: SKIP no assets here\\n"}
{"Action":"skip","Package":"partyhard","Test":"TestC","Elapsed":0}
{"Action":"run","Package":"partyhard","Test":"TestSlow"}
{"Action":"pass","Package":"partyhard","Test":"TestSlow","Elapsed":42}
{"Action":"fail","Package":"partyhard","Elapsed":44}
`

func TestSummarizeTestJSON(t *testing.T) {
	s := summarizeTestJSON(sampleJSON)
	if s.ran != 4 || s.passed != 2 || s.failed != 1 || s.skipped != 1 {
		t.Fatalf("counts: ran %d passed %d failed %d skipped %d", s.ran, s.passed, s.failed, s.skipped)
	}
	got := s.format()
	for _, want := range []string{"go test: FAIL", "FAILED TestB", "expected 3, got 4", "SKIPPED", "TestC", "SKIP no assets here", "SLOW", "TestSlow"} {
		if !strings.Contains(got, want) {
			t.Fatalf("report lacks %q:\n%s", want, got)
		}
	}
}

func TestSummaryWarnsWhenNothingRan(t *testing.T) {
	out := `{"Action":"output","Package":"partyhard","Output":"testing: warning: no tests to run\n"}
{"Action":"pass","Package":"partyhard","Elapsed":0.1}
`
	got := summarizeTestJSON(out).format()
	if !strings.Contains(got, "no test ran") || !strings.Contains(got, "proves NOTHING") {
		t.Fatalf("an empty run must be flagged:\n%s", got)
	}
	// a build error is not a result
	got = summarizeTestJSON("# partyhard\nfoo.go:3:1: syntax error\n").format()
	if !strings.Contains(got, "NO RESULT") || !strings.Contains(got, "syntax error") {
		t.Fatalf("a build error must be shown:\n%s", got)
	}
}

// ---- build_release: verifying the packages --------------------------------------

func makeZip(t *testing.T, path string, names ...string) {
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for _, n := range names {
		w, _ := zw.Create(n)
		w.Write([]byte("x"))
	}
	zw.Close()
	f.Close()
}

func TestVerifyZip(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.zip")
	makeZip(t, good, "PartyHard.exe", "PartyHardEditor.exe", "assets/sprites/people/a/front.png", "music/x.mp3")
	r := verifyZip(good, []string{"PartyHard.exe", "PartyHardEditor.exe"})
	if len(r.issues) != 0 || r.mp3 != 1 {
		t.Fatalf("a clean zip must pass: %v", r.issues)
	}
	bad := filepath.Join(dir, "bad.zip")
	makeZip(t, bad, "PartyHard.exe", "assets/progress.json", "assets/editor_prefs.json", "imports/sprites/people/amigo/front.png", "mac/assets/Settings.json")
	r = verifyZip(bad, []string{"PartyHard.exe", "PartyHardEditor.exe"})
	joined := strings.Join(r.issues, "\n")
	for _, want := range []string{"progress.json", "editor_prefs.json", "imports/sprites", "Settings.json", "PartyHardEditor.exe"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("the problem %q was not reported:\n%s", want, joined)
		}
	}
	if r := verifyZip(filepath.Join(dir, "nope.zip"), nil); len(r.issues) == 0 {
		t.Fatal("a missing zip must be a problem")
	}
}

func TestBuildReleaseVerifyOnlyBuildsNothing(t *testing.T) {
	root := fakeProject(t)
	os.WriteFile(filepath.Join(root, "go.mod"), []byte("module partyhard\n"), 0o644)
	makeZip(t, filepath.Join(root, "PartyHard-Mac.zip"), "PartyHard.app/Contents/MacOS/PartyHard", "PartyHardEditor.app/Contents/MacOS/PartyHardEditor")
	makeZip(t, filepath.Join(root, "PartyHard-Windows.zip"), "PartyHard.exe", "PartyHardEditor.exe", "assets/progress.json")
	res := rpc(t, root, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"build_release","arguments":{"verify_only":true}}}`)
	got := text(res[0])
	if !strings.Contains(got, "PartyHard-Mac.zip: ok") || !strings.Contains(got, "PartyHard-Windows.zip: PROBLEMS") || !strings.Contains(got, "progress.json") {
		t.Fatalf("report: %s", got)
	}
	if !strings.Contains(got, "ROOT BINARIES STALE") || !strings.Contains(got, "RESULT: PROBLEMS") {
		t.Fatalf("the stale root binaries and the final result must be said:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(root, "dist")); err == nil {
		t.Fatal("verify_only must not create dist/")
	}
	// a folder that is not the project is refused
	other := t.TempDir()
	res = rpc(t, other, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"build_release","arguments":{"verify_only":true}}}`)
	if res[0]["result"].(map[string]any)["isError"] != true {
		t.Fatal("build_release must refuse a folder that is not the project")
	}
}

func TestRootBinariesComparedWithDist(t *testing.T) {
	root := t.TempDir()
	for _, p := range rootBinaryPairs {
		for _, f := range p {
			os.MkdirAll(filepath.Dir(filepath.Join(root, f)), 0o755)
			os.WriteFile(filepath.Join(root, f), []byte("same"), 0o755)
		}
	}
	if s := staleRootBinaries(root); len(s) != 0 {
		t.Fatalf("identical files are not stale: %v", s)
	}
	os.WriteFile(filepath.Join(root, "PartyHard.exe"), []byte("old"), 0o755)
	if s := staleRootBinaries(root); len(s) != 1 || s[0] != "PartyHard.exe" {
		t.Fatalf("stale: %v", s)
	}
}

// ---- clean_temp: the allowlist ---------------------------------------------------

func TestAllowProc(t *testing.T) {
	yes := []string{"/tmp/ph_chk -level 1 -shot /tmp/x.png", "/private/tmp/ph_g -intro", "/var/folders/ab/T/phmcp-123/game -level 3", "/var/folders/x/go-build/b001/partyhard.test -test.run X"}
	no := []string{"/Users/someone/Proyectos/Party Hard JM/PartyHard", "/Applications/PartyHard.app/Contents/MacOS/PartyHard", "/usr/bin/vim /tmp/ph_notes.txt",
		"/tmp/other", "/tmp/../Users/x/ph_evil", "/tmp/phmcpx/game", "go test .", ""}
	for _, c := range yes {
		if !allowProc(c) {
			t.Errorf("should be allowed: %s", c)
		}
	}
	for _, c := range no {
		if allowProc(c) {
			t.Errorf("must NOT be allowed: %s", c)
		}
	}
	procs := parsePS("  101 /tmp/ph_chk -level 1\n 202 /usr/bin/vim x\nbad line\n    1 /sbin/launchd\n")
	if len(procs) != 2 || procs[0].pid != 101 {
		t.Fatalf("parsePS: %v", procs)
	}
}

func TestCleanTempDryRunAndConfirmation(t *testing.T) {
	old := listProcs
	listProcs = func() string { return "" } // never touch the real processes of the machine
	defer func() { listProcs = old }()
	tmp := t.TempDir()
	t.Setenv("PHMCP_TMP", tmp)
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	os.WriteFile(secret, []byte("do not touch"), 0o644)
	mk := func(name string, dir bool) string {
		p := filepath.Join(tmp, name)
		if dir {
			os.MkdirAll(filepath.Join(p, "sub"), 0o755)
			os.WriteFile(filepath.Join(p, "sub", "a.bin"), []byte("12345"), 0o644)
		} else {
			os.WriteFile(p, []byte("x"), 0o644)
		}
		return p
	}
	a := mk("ph_chk", false)
	b := mk("phmcp-123", true)
	keep1 := mk("other.txt", false)
	keep2 := mk("photos", true) // starts with "ph" but not "ph_"
	link := filepath.Join(tmp, "ph_link")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	call := func(args string) map[string]any {
		return rpc(t, "/unused", `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"clean_temp","arguments":`+args+`}}`)[0]["result"].(map[string]any)
	}
	// default is a dry run: it lists and deletes nothing
	r := call(`{}`)
	txt := r["content"].([]any)[0].(map[string]any)["text"].(string)
	if !strings.Contains(txt, "DRY RUN") || !strings.Contains(txt, "ph_chk") || !strings.Contains(txt, "phmcp-123") || strings.Contains(txt, "other.txt") || strings.Contains(txt, "photos") {
		t.Fatalf("dry run report:\n%s", txt)
	}
	for _, p := range []string{a, b, keep1, keep2, link, secret} {
		if _, err := os.Lstat(p); err != nil {
			t.Fatalf("a dry run deleted %s", p)
		}
	}
	// deleting without the confirmation is refused
	for _, args := range []string{`{"dry_run":false}`, `{"dry_run":false,"confirm":"yes"}`} {
		if r := call(args); r["isError"] != true {
			t.Fatalf("%s must be refused", args)
		}
	}
	if _, err := os.Lstat(a); err != nil {
		t.Fatal("a refused call deleted something")
	}
	// the real deletion removes only the allowlist
	r = call(`{"dry_run":false,"confirm":"DELETE"}`)
	if r["isError"] == true {
		t.Fatalf("%v", r)
	}
	for _, p := range []string{a, b, link} {
		if _, err := os.Lstat(p); err == nil {
			t.Errorf("%s should be gone", p)
		}
	}
	for _, p := range []string{keep1, keep2, secret, outside} {
		if _, err := os.Lstat(p); err != nil {
			t.Errorf("%s must survive", p)
		}
	}
}

func TestTempRootsRefusesRoot(t *testing.T) {
	t.Setenv("PHMCP_TMP", "/")
	for _, r := range tempRoots() {
		if r == "/" {
			t.Fatal("the file system root can never be a temp root")
		}
	}
}

func TestDiskReportIsReadOnly(t *testing.T) {
	old := listProcs
	listProcs = func() string { return "" }
	defer func() { listProcs = old }()
	root := fakeProject(t)
	t.Setenv("PHMCP_TMP", t.TempDir())
	before := snapshot(t, root)
	res := rpc(t, root, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"disk_report","arguments":{}}}`)
	if got := text(res[0]); !strings.Contains(got, "DISK REPORT") || !strings.Contains(got, "free of") {
		t.Fatalf("disk_report: %s", got)
	}
	if snapshot(t, root) != before {
		t.Fatal("disk_report changed the project")
	}
}

// a stuck process of the allowlist is found through the ps hook (and nothing else is)
func TestAllowedProcsUsesTheHook(t *testing.T) {
	old := listProcs
	listProcs = func() string {
		return " 4001 /tmp/ph_g -level 2\n 4002 /Users/x/Party Hard JM/PartyHard\n 4003 /usr/bin/ssh host\n"
	}
	defer func() { listProcs = old }()
	got := allowedProcs()
	if len(got) != 1 || got[0].pid != 4001 {
		t.Fatalf("allowlist: %v", got)
	}
}

// every skill of the project has a valid header: name equal to its folder and a description
func TestSkillsHaveValidFrontmatter(t *testing.T) {
	dirs, err := filepath.Glob(filepath.Join("..", "..", ".claude", "skills", "*", "SKILL.md"))
	if err != nil || len(dirs) < 5 {
		t.Fatalf("expected at least 5 skills in .claude/skills, found %d (%v)", len(dirs), err)
	}
	for _, p := range dirs {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		s := string(b)
		if !strings.HasPrefix(s, "---\n") {
			t.Errorf("%s: no frontmatter", p)
			continue
		}
		end := strings.Index(s[4:], "\n---")
		if end < 0 {
			t.Errorf("%s: the frontmatter is not closed", p)
			continue
		}
		head := s[4 : 4+end]
		folder := filepath.Base(filepath.Dir(p))
		if !strings.Contains(head, "name: "+folder+"\n") {
			t.Errorf("%s: name must be %q", p, folder)
		}
		if i := strings.Index(head, "description: "); i < 0 || len(strings.TrimSpace(head[i+13:])) < 40 {
			t.Errorf("%s: description missing or too short", p)
		}
		if len(strings.TrimSpace(s[4+end+4:])) < 200 {
			t.Errorf("%s: the body is empty", p)
		}
	}
}
