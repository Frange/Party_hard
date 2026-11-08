// phmcp is a tiny MCP (Model Context Protocol) server for the Party Hard project.
//
// It speaks JSON-RPC 2.0 over stdio (one JSON message per line) and exposes helpers:
//   - read-only: list_levels, list_characters, check_phrases, asset_report, disk_report
//   - checking (they start processes with a time limit): capture_level, run_tests,
//     qa_run_tests (QA role)
//   - release (writes only dist/ and the two zips, see release.go): build_release (BUILD role)
//   - cleanup (can delete, but only a strict allowlist, dry run by default, see clean.go):
//     clean_temp (CLEANUP role)
//
// Safety: no tool edits the game code and none touches assets/, imports/, intros/,
// music/ or assets/progress.json. Child processes always have a time limit and their
// whole process group is killed when it passes. Only the Go standard library is used.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
)

const maxPhrase = 48 // longest allowed speech bubble sentence (characters)

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	run         func(root string, args map[string]any) (string, error)
}

func obj(props map[string]any) map[string]any {
	return map[string]any{"type": "object", "properties": props}
}

var tools = []*tool{
	{Name: "list_levels", Description: "Lists the 21 campaign levels (in order) and the extra levels used only by random mode. Read-only.",
		InputSchema: obj(map[string]any{}), run: listLevels},
	{Name: "list_characters", Description: "Lists the character presets defined in the code (famous, kids, Monkey Island, skeletons...) with the file they live in, plus the folders found in assets/sprites and imports/sprites. Read-only.",
		InputSchema: obj(map[string]any{"filter": map[string]any{"type": "string", "description": "optional text the slug must contain"}}), run: listCharacters},
	{Name: "check_phrases", Description: "Finds speech bubble sentences longer than 48 characters in the phrase tables (jokes, dialogue, greetings, duets, Murray, Monkey Island, skeletons). Read-only.",
		InputSchema: obj(map[string]any{"max": map[string]any{"type": "integer", "description": "maximum length, default 48"}}), run: checkPhrases},
	{Name: "capture_level", Description: "Builds the game in a temporary folder and takes a screenshot of a level (1-based index) with -level N -shot. Returns the PNG path. The game is killed after 60 seconds at most. Never writes progress.",
		InputSchema: obj(map[string]any{
			"level": map[string]any{"type": "integer", "description": "campaign level, 1 to 21"},
			"tick":  map[string]any{"type": "integer", "description": "frame of the screenshot, default 200"},
			"with":  map[string]any{"type": "string", "description": "optional comma separated character slugs (-with)"},
		}), run: captureLevel},
	{Name: "run_tests", Description: "Runs go vet and go test -count=1 on the game package with a time limit (default 300 s) and returns the summary. Optional -run filter.",
		InputSchema: obj(map[string]any{
			"filter":  map[string]any{"type": "string", "description": "optional go test -run regular expression"},
			"seconds": map[string]any{"type": "integer", "description": "time limit, default 300, max 900"},
		}), run: runTests},
	{Name: "asset_report", Description: "Counts characters, objects and backgrounds in assets and in imports. Read-only, does not open progress.json.",
		InputSchema: obj(map[string]any{}), run: assetReport},
}

// ---- project root ---------------------------------------------------------------

func findRoot() string {
	if r := os.Getenv("PH_ROOT"); r != "" {
		if abs, err := filepath.Abs(r); err == nil {
			return abs
		}
	}
	dir, _ := os.Getwd()
	for d := dir; ; d = filepath.Dir(d) {
		if b, err := os.ReadFile(filepath.Join(d, "go.mod")); err == nil && strings.HasPrefix(string(b), "module partyhardy") {
			return d
		}
		if filepath.Dir(d) == d {
			return dir
		}
	}
}

// ---- JSON-RPC loop --------------------------------------------------------------

func main() {
	serve(os.Stdin, os.Stdout, findRoot())
}

func serve(in io.Reader, out io.Writer, root string) {
	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 1<<20), 8<<20)
	enc := json.NewEncoder(out)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var req request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			enc.Encode(response{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &rpcError{-32700, "parse error"}})
			continue
		}
		if len(req.ID) == 0 { // notification: no answer
			continue
		}
		res := response{JSONRPC: "2.0", ID: req.ID}
		switch req.Method {
		case "initialize":
			var p struct {
				ProtocolVersion string `json:"protocolVersion"`
			}
			json.Unmarshal(req.Params, &p)
			if p.ProtocolVersion == "" {
				p.ProtocolVersion = "2025-03-26"
			}
			res.Result = map[string]any{
				"protocolVersion": p.ProtocolVersion,
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]any{"name": "party-hard", "version": "1.0.0"},
			}
		case "ping":
			res.Result = map[string]any{}
		case "tools/list":
			res.Result = map[string]any{"tools": tools}
		case "tools/call":
			res.Result = callTool(root, req.Params)
		default:
			res.Error = &rpcError{-32601, "method not found: " + req.Method}
		}
		enc.Encode(res)
	}
}

func callTool(root string, raw json.RawMessage) map[string]any {
	var p struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	json.Unmarshal(raw, &p)
	for _, t := range tools {
		if t.Name != p.Name {
			continue
		}
		text, err := t.run(root, p.Arguments)
		if err != nil {
			return map[string]any{"isError": true, "content": []any{map[string]any{"type": "text", "text": err.Error()}}}
		}
		return map[string]any{"content": []any{map[string]any{"type": "text", "text": text}}}
	}
	return map[string]any{"isError": true, "content": []any{map[string]any{"type": "text", "text": "unknown tool: " + p.Name}}}
}

func argInt(a map[string]any, k string, def int) int {
	switch v := a[k].(type) {
	case float64:
		return int(v)
	case string:
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func argStr(a map[string]any, k string) string {
	s, _ := a[k].(string)
	return s
}

// ---- list_levels ----------------------------------------------------------------

var (
	reHand  = regexp.MustCompile(`\{level:\s*(\w+),\s*icon:\s*"(\w+)"\}`)
	reGen   = regexp.MustCompile(`\{kind:\s*"(\w+)",\s*seed:\s*(\d+),\s*title:\s*"([^"]+)",\s*icon:\s*"(\w+)"\}`)
	reExtra = regexp.MustCompile(`\{"(\w+)",\s*(\d+),\s*"([^"]+)"\}`)
)

func listLevels(root string, _ map[string]any) (string, error) {
	b, err := os.ReadFile(filepath.Join(root, "campaign.go"))
	if err != nil {
		return "", err
	}
	type entry struct {
		pos  int
		text string
	}
	var es []entry
	for _, m := range reHand.FindAllStringSubmatchIndex(string(b), -1) {
		s := string(b)
		es = append(es, entry{m[0], fmt.Sprintf("hand-made level (%s), icon %s", s[m[2]:m[3]], s[m[4]:m[5]])})
	}
	for _, m := range reGen.FindAllStringSubmatchIndex(string(b), -1) {
		s := string(b)
		es = append(es, entry{m[0], fmt.Sprintf("%s (generator %q, seed %s)", s[m[6]:m[7]], s[m[2]:m[3]], s[m[4]:m[5]])})
	}
	sort.Slice(es, func(i, j int) bool { return es[i].pos < es[j].pos })
	var sb strings.Builder
	sb.WriteString("CAMPAIGN (use these numbers with -level and capture_level):\n")
	for i, e := range es {
		fmt.Fprintf(&sb, "%2d  %s\n", i+1, e.text)
	}
	if lb, err := os.ReadFile(filepath.Join(root, "levels.go")); err == nil {
		s := string(lb)
		if i := strings.Index(s, "var extraLevels"); i >= 0 {
			if j := strings.Index(s[i:], "\n}\n"); j > 0 {
				sb.WriteString("\nEXTRA (random mode only):\n")
				for _, m := range reExtra.FindAllStringSubmatch(s[i:i+j], -1) {
					fmt.Fprintf(&sb, "    %s (generator %q, seed %s)\n", m[3], m[1], m[2])
				}
			}
		}
	}
	sb.WriteString("\nThe Jurassic Park island is not in the campaign: use the -jurassic flag.\n")
	return sb.String(), nil
}

// ---- list_characters ------------------------------------------------------------

var reDef = regexp.MustCompile(`\bdef\("([a-z0-9_]+)"`)

func listCharacters(root string, a map[string]any) (string, error) {
	filter := strings.ToLower(argStr(a, "filter"))
	files, _ := filepath.Glob(filepath.Join(root, "*.go"))
	sort.Strings(files)
	var sb strings.Builder
	total := 0
	sb.WriteString("PRESETS IN THE CODE (def(\"slug\", ...)):\n")
	for _, f := range files {
		base := filepath.Base(f)
		if strings.HasSuffix(base, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var slugs []string
		for _, m := range reDef.FindAllStringSubmatch(string(b), -1) {
			if filter == "" || strings.Contains(m[1], filter) {
				slugs = append(slugs, m[1])
			}
		}
		if len(slugs) > 0 {
			total += len(slugs)
			fmt.Fprintf(&sb, "\n%s (%d): %s\n", base, len(slugs), strings.Join(slugs, ", "))
		}
	}
	fmt.Fprintf(&sb, "\nTotal presets: %d\n\nFOLDERS WITH DRAWN SPRITES:\n", total)
	for _, base := range []string{"assets", "imports"} {
		cats, _ := os.ReadDir(filepath.Join(root, base, "sprites"))
		for _, c := range cats {
			if !c.IsDir() {
				continue
			}
			names := subdirs(filepath.Join(root, base, "sprites", c.Name()))
			var keep []string
			for _, n := range names {
				if filter == "" || strings.Contains(n, filter) {
					keep = append(keep, n)
				}
			}
			if len(keep) > 0 {
				fmt.Fprintf(&sb, "%s/%s (%d): %s\n", base, c.Name(), len(keep), strings.Join(keep, ", "))
			}
		}
	}
	return sb.String(), nil
}

func subdirs(dir string) []string {
	es, _ := os.ReadDir(dir)
	var out []string
	for _, e := range es {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

// ---- check_phrases --------------------------------------------------------------

var phraseFiles = []string{"jokes.go", "dialogue.go", "greetings.go", "duets.go", "murray.go", "monkey_lines.go", "skelcast.go"}

func checkPhrases(root string, a map[string]any) (string, error) {
	limit := argInt(a, "max", maxPhrase)
	var sb strings.Builder
	n := 0
	for _, f := range phraseFiles {
		path := filepath.Join(root, f)
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			continue
		}
		ast.Inspect(file, func(nd ast.Node) bool {
			bl, ok := nd.(*ast.BasicLit)
			if !ok || bl.Kind != token.STRING {
				return true
			}
			s, err := strconv.Unquote(bl.Value)
			if err != nil || strings.ContainsAny(s, "%{") && utf8.RuneCountInString(s) < 3 {
				return true
			}
			if utf8.RuneCountInString(s) > limit && strings.Contains(s, " ") {
				n++
				if n <= 80 {
					fmt.Fprintf(&sb, "%s:%d (%d) %s\n", f, fset.Position(bl.Pos()).Line, utf8.RuneCountInString(s), s)
				}
			}
			return true
		})
	}
	if n == 0 {
		return fmt.Sprintf("No sentence is longer than %d characters in %s.", limit, strings.Join(phraseFiles, ", ")), nil
	}
	return fmt.Sprintf("%d sentences longer than %d characters (first 80 shown). Some may be descriptions and not bubbles.\n%s", n, limit, sb.String()), nil
}

// ---- processes with a time limit ------------------------------------------------

// runLimited runs a command and kills the whole process group when the time is up.
func runLimited(dir string, limit time.Duration, env []string, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return string(out), fmt.Errorf("time limit of %s reached: process killed", limit)
	}
	return string(out), err
}

func tail(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

// ---- capture_level --------------------------------------------------------------

func captureLevel(root string, a map[string]any) (string, error) {
	level := argInt(a, "level", 1)
	if level < 1 || level > 21 {
		return "", fmt.Errorf("level must be between 1 and 21")
	}
	tick := argInt(a, "tick", 200)
	if tick < 1 || tick > 3000 {
		return "", fmt.Errorf("tick must be between 1 and 3000")
	}
	tmp, err := os.MkdirTemp("", "phmcp-")
	if err != nil {
		return "", err
	}
	bin := filepath.Join(tmp, "game")
	if out, err := runLimited(root, 180*time.Second, nil, "go", "build", "-o", bin, "."); err != nil {
		return "", fmt.Errorf("build failed: %v\n%s", err, tail(out, 15))
	}
	shot := filepath.Join(tmp, fmt.Sprintf("level%02d.png", level))
	args := []string{"-level", strconv.Itoa(level), "-shottick", strconv.Itoa(tick), "-shot", shot}
	if w := argStr(a, "with"); w != "" {
		if !regexp.MustCompile(`^[a-z0-9_,]+$`).MatchString(w) {
			return "", fmt.Errorf("with: only slugs separated by commas")
		}
		args = append(args, "-with", w)
	}
	out, err := runLimited(root, 60*time.Second, nil, bin, args...)
	os.Remove(bin)
	if _, statErr := os.Stat(shot); statErr != nil {
		return "", fmt.Errorf("no screenshot was produced (%v)\n%s", err, tail(out, 10))
	}
	return "Screenshot saved at " + shot + " (the temporary game binary was removed).", nil
}

// ---- run_tests ------------------------------------------------------------------

func runTests(root string, a map[string]any) (string, error) {
	secs := argInt(a, "seconds", 300)
	if secs < 10 {
		secs = 10
	}
	if secs > 900 {
		secs = 900
	}
	limit := time.Duration(secs) * time.Second
	start := time.Now()
	vet, err := runLimited(root, limit, nil, "go", "vet", ".")
	var sb strings.Builder
	if err != nil {
		fmt.Fprintf(&sb, "go vet FAILED: %v\n%s\n", err, tail(vet, 20))
		return sb.String(), nil
	}
	sb.WriteString("go vet: ok\n")
	args := []string{"test", "-count=1"}
	if f := argStr(a, "filter"); f != "" {
		args = append(args, "-run", f)
	}
	args = append(args, ".")
	rest := limit - time.Since(start)
	if rest < 5*time.Second {
		rest = 5 * time.Second
	}
	out, err := runLimited(root, rest, nil, "go", args...)
	if err != nil {
		fmt.Fprintf(&sb, "go test FAILED: %v\n%s\n", err, tail(out, 40))
	} else {
		fmt.Fprintf(&sb, "go test: %s\n", tail(out, 3))
	}
	fmt.Fprintf(&sb, "(%s)", time.Since(start).Round(time.Second))
	return sb.String(), nil
}

// ---- asset_report ---------------------------------------------------------------

func assetReport(root string, _ map[string]any) (string, error) {
	var sb strings.Builder
	for _, base := range []string{"assets", "imports"} {
		dir := filepath.Join(root, base)
		if !dirExists(dir) {
			fmt.Fprintf(&sb, "%s: folder not found\n", base)
			continue
		}
		fmt.Fprintf(&sb, "%s:\n", base)
		cats, _ := os.ReadDir(filepath.Join(dir, "sprites"))
		for _, c := range cats {
			if c.IsDir() {
				fmt.Fprintf(&sb, "  sprites/%s: %d\n", c.Name(), len(subdirs(filepath.Join(dir, "sprites", c.Name()))))
			}
		}
		fmt.Fprintf(&sb, "  backgrounds: %d\n", len(subdirs(filepath.Join(dir, "backgrounds"))))
	}
	return sb.String(), nil
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}
