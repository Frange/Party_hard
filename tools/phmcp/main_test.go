package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// a tiny fake project so the tests do not depend on the real one
func fakeProject(t *testing.T) string {
	root := t.TempDir()
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.MkdirAll(filepath.Join(root, "assets", "sprites", "people", "abogado"), 0o755))
	must(os.MkdirAll(filepath.Join(root, "assets", "sprites", "people", "becario"), 0o755))
	must(os.MkdirAll(filepath.Join(root, "imports", "sprites", "people", "amigo"), 0o755))
	must(os.MkdirAll(filepath.Join(root, "assets", "backgrounds"), 0o755))
	must(os.WriteFile(filepath.Join(root, "campaign.go"), []byte(`package main
var campaignLevels = []campaignEntry{
	{level: level1, icon: "house"},
	{kind: "bullring", seed: 11, title: "PLAZA DE TOROS", icon: "bull"},
}
`), 0o644))
	must(os.WriteFile(filepath.Join(root, "jokes.go"), []byte(`package main
var a = []string{"Corta", "Esta frase es demasiado larga para un bocadillo de la fiesta, de verdad"}
`), 0o644))
	must(os.WriteFile(filepath.Join(root, "presets.go"), []byte(`package main
func f() { def("sonic", x); def("shrek", y) }
`), 0o644))
	return root
}

func rpc(t *testing.T, root string, lines ...string) []map[string]any {
	var out bytes.Buffer
	serve(strings.NewReader(strings.Join(lines, "\n")+"\n"), &out, root)
	var res []map[string]any
	dec := json.NewDecoder(&out)
	for dec.More() {
		var m map[string]any
		if err := dec.Decode(&m); err != nil {
			t.Fatal(err)
		}
		res = append(res, m)
	}
	return res
}

func text(m map[string]any) string {
	r := m["result"].(map[string]any)
	return r["content"].([]any)[0].(map[string]any)["text"].(string)
}

func TestInitializeAndList(t *testing.T) {
	root := fakeProject(t)
	res := rpc(t, root,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26"}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"nope"}`)
	if len(res) != 3 { // the notification gets no answer
		t.Fatalf("expected 3 answers, got %d", len(res))
	}
	info := res[0]["result"].(map[string]any)["serverInfo"].(map[string]any)
	if info["name"] != "party-hard" {
		t.Fatal("serverInfo")
	}
	list := res[1]["result"].(map[string]any)["tools"].([]any)
	if len(list) != 10 {
		t.Fatalf("expected 10 tools, got %d", len(list))
	}
	names := map[string]bool{}
	for _, it := range list {
		names[it.(map[string]any)["name"].(string)] = true
	}
	for _, want := range []string{"qa_run_tests", "build_release", "clean_temp", "disk_report"} {
		if !names[want] {
			t.Fatalf("tool %s missing", want)
		}
	}
	if res[2]["error"] == nil {
		t.Fatal("unknown method must be an error")
	}
}

func TestCheapTools(t *testing.T) {
	root := fakeProject(t)
	res := rpc(t, root,
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"asset_report","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"list_levels","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"check_phrases","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"list_characters","arguments":{"filter":"son"}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"capture_level","arguments":{"level":99}}}`)
	if got := text(res[0]); !strings.Contains(got, "sprites/people: 2") || !strings.Contains(got, "imports") {
		t.Fatalf("asset_report: %s", got)
	}
	if got := text(res[1]); !strings.Contains(got, "PLAZA DE TOROS") {
		t.Fatalf("list_levels: %s", got)
	}
	if got := text(res[2]); !strings.Contains(got, "demasiado larga") || strings.Contains(got, "Corta") {
		t.Fatalf("check_phrases: %s", got)
	}
	if got := text(res[3]); !strings.Contains(got, "sonic") || strings.Contains(got, "shrek") {
		t.Fatalf("list_characters: %s", got)
	}
	if res[4]["result"].(map[string]any)["isError"] != true {
		t.Fatal("level out of range must be an error")
	}
}

// the tools must never write inside the project (checked on the fake one)
func TestReadOnlyTools(t *testing.T) {
	root := fakeProject(t)
	before := snapshot(t, root)
	rpc(t, root,
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"asset_report","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"list_levels","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"check_phrases","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"list_characters","arguments":{}}}`)
	if after := snapshot(t, root); after != before {
		t.Fatalf("the project changed:\n%s\n---\n%s", before, after)
	}
}

func snapshot(t *testing.T, root string) string {
	var sb strings.Builder
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err == nil {
			rel, _ := filepath.Rel(root, p)
			sb.WriteString(rel + " " + info.ModTime().String() + "\n")
		}
		return nil
	})
	return sb.String()
}
