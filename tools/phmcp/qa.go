package main

// qa_run_tests: the tool of the QA role. It runs go vet and go test -json and returns a
// structured summary (counts, failed tests with their last output, SKIPPED tests, slow
// tests) so the verdict is based on numbers and not on a person's memory.
//
// Safety: it only runs go (with a time limit, the process group is killed on timeout);
// it writes nothing in the project.

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

func init() {
	tools = append(tools, &tool{
		Name: "qa_run_tests",
		Description: "QA role. Runs go vet and go test -count=1 -json on the game package (time limit default 600 s, max 1200) and returns a structured report: totals, FAILED tests with their last output lines, SKIPPED tests, slow tests, and a warning when no test ran. " +
			"Use it to check the claims in an agent's report. Writes nothing in the project.",
		InputSchema: obj(map[string]any{
			"filter":  map[string]any{"type": "string", "description": "optional go test -run regular expression"},
			"seconds": map[string]any{"type": "integer", "description": "time limit, default 600, max 1200"},
		}),
		run: qaRunTests,
	})
}

type testEvent struct {
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Output  string  `json:"Output"`
	Elapsed float64 `json:"Elapsed"`
}

type testInfo struct {
	name, status string
	elapsed      float64
	out          []string
}

type testSummary struct {
	tests                        map[string]*testInfo
	passed, failed, skipped, ran int
	pkgResult                    map[string]string
	other                        []string // lines that are not JSON (build errors...)
	noTests                      bool
}

// summarizeTestJSON reads the output of `go test -json`.
func summarizeTestJSON(out string) *testSummary {
	s := &testSummary{tests: map[string]*testInfo{}, pkgResult: map[string]string{}}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var ev testEvent
		if !strings.HasPrefix(line, "{") || json.Unmarshal([]byte(line), &ev) != nil || ev.Action == "" {
			s.other = append(s.other, line)
			continue
		}
		if strings.Contains(ev.Output, "no tests to run") {
			s.noTests = true
		}
		if ev.Test == "" {
			switch ev.Action {
			case "pass", "fail", "skip":
				s.pkgResult[ev.Package] = ev.Action
			}
			continue
		}
		ti := s.tests[ev.Test]
		if ti == nil {
			ti = &testInfo{name: ev.Test}
			s.tests[ev.Test] = ti
		}
		switch ev.Action {
		case "output":
			ti.out = append(ti.out, strings.TrimRight(ev.Output, "\n"))
			if len(ti.out) > 60 {
				ti.out = ti.out[len(ti.out)-60:]
			}
		case "pass", "fail", "skip":
			ti.status, ti.elapsed = ev.Action, ev.Elapsed
		}
	}
	for _, ti := range s.tests {
		switch ti.status {
		case "pass":
			s.passed++
		case "fail":
			s.failed++
		case "skip":
			s.skipped++
		}
	}
	s.ran = s.passed + s.failed + s.skipped
	return s
}

func (s *testSummary) format() string {
	var sb strings.Builder
	verdict := "PASS"
	for _, r := range s.pkgResult {
		if r == "fail" {
			verdict = "FAIL"
		}
	}
	if len(s.pkgResult) == 0 {
		verdict = "NO RESULT (build error or timeout)"
	}
	fmt.Fprintf(&sb, "go test: %s\n", verdict)
	fmt.Fprintf(&sb, "tests (subtests included): %d ran, %d passed, %d failed, %d skipped\n", s.ran, s.passed, s.failed, s.skipped)
	if s.ran == 0 || s.noTests {
		sb.WriteString("WARNING: no test ran (the filter matched nothing or the package did not build). This proves NOTHING.\n")
	}
	var failed, skipped, slow []*testInfo
	for _, ti := range s.tests {
		switch ti.status {
		case "fail":
			failed = append(failed, ti)
		case "skip":
			skipped = append(skipped, ti)
		}
		if ti.elapsed > 20 && !strings.Contains(ti.name, "/") {
			slow = append(slow, ti)
		}
	}
	byName := func(l []*testInfo) { sort.Slice(l, func(i, j int) bool { return l[i].name < l[j].name }) }
	byName(failed)
	byName(skipped)
	sort.Slice(slow, func(i, j int) bool { return slow[i].elapsed > slow[j].elapsed })
	for _, ti := range failed {
		fmt.Fprintf(&sb, "\nFAILED %s (%.1fs)\n%s\n", ti.name, ti.elapsed, indentLast(ti.out, 12))
	}
	if len(skipped) > 0 {
		sb.WriteString("\nSKIPPED (a skipped test proves nothing; check that the skip is justified):\n")
		for i, ti := range skipped {
			if i == 25 {
				fmt.Fprintf(&sb, "  ... and %d more\n", len(skipped)-25)
				break
			}
			reason := ""
			for _, l := range ti.out {
				if strings.Contains(l, "SKIP") || strings.Contains(l, "skip") || strings.Contains(l, ".go:") {
					reason = strings.TrimSpace(l)
				}
			}
			fmt.Fprintf(&sb, "  %s %s\n", ti.name, reason)
		}
	}
	if len(slow) > 0 {
		sb.WriteString("\nSLOW (>20 s):\n")
		for i, ti := range slow {
			if i == 8 {
				break
			}
			fmt.Fprintf(&sb, "  %s %.0fs\n", ti.name, ti.elapsed)
		}
	}
	if len(s.other) > 0 {
		fmt.Fprintf(&sb, "\nNON-TEST OUTPUT (build errors?):\n%s\n", tail(strings.Join(s.other, "\n"), 20))
	}
	return sb.String()
}

func indentLast(lines []string, n int) string {
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	for i, l := range lines {
		lines[i] = "    " + l
	}
	return strings.Join(lines, "\n")
}

func qaRunTests(root string, a map[string]any) (string, error) {
	secs := argInt(a, "seconds", 600)
	if secs < 30 {
		secs = 30
	}
	if secs > 1200 {
		secs = 1200
	}
	limit := time.Duration(secs) * time.Second
	start := time.Now()
	var sb strings.Builder
	sb.WriteString("QA TEST REPORT\n")
	vet, err := runLimited(root, limit, nil, "go", "vet", ".")
	if err != nil {
		fmt.Fprintf(&sb, "go vet: FAILED (%v)\n%s\n", err, tail(vet, 20))
		return sb.String(), nil
	}
	sb.WriteString("go vet: ok\n")
	args := []string{"test", "-count=1", "-json"}
	if f := argStr(a, "filter"); f != "" {
		args = append(args, "-run", f)
	}
	args = append(args, ".")
	rest := limit - time.Since(start)
	if rest < 10*time.Second {
		rest = 10 * time.Second
	}
	out, err := runLimited(root, rest, nil, "go", args...)
	sum := summarizeTestJSON(out)
	sb.WriteString(sum.format())
	if err != nil && strings.Contains(err.Error(), "time limit") {
		fmt.Fprintf(&sb, "\nTIMEOUT: %v\n", err)
	}
	fmt.Fprintf(&sb, "\n(total %s)", time.Since(start).Round(time.Second))
	return sb.String(), nil
}
