package main

// clean_temp and disk_report: the tools of the CLEANUP role.
//
// clean_temp can delete, so it is the most restricted tool of the server:
//   - Only an ALLOWLIST is ever touched:
//       * temp entries named ph_* or phmcp-* directly inside /tmp or the system temp dir
//       * processes whose executable is /tmp/ph_*, <temp>/phmcp-*/game or partyhard.test
//       * the Go build and test caches (only with go_cache=true)
//     Nothing else: never assets/, imports/, intros/, music/, progress.json, the user's
//     game, the temp folders of other Claude projects or any path outside the allowlist.
//   - It lists first. dry_run is true by default. To delete, the call needs
//     dry_run=false AND confirm="DELETE".
//   - Symbolic links are removed as links (their target is never followed or touched).
//   - No shell is used: every command has fixed arguments.
// Test hook: PHMCP_TMP (colon separated folders) replaces the temp roots.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func init() {
	tools = append(tools,
		&tool{
			Name: "clean_temp",
			Description: "CLEANUP role. Lists (default, dry_run=true) or deletes ONLY: temp entries ph_* / phmcp-* in /tmp and the system temp dir, stuck test processes (/tmp/ph_*, phmcp-*/game, partyhard.test) and, with go_cache=true, the Go build and test caches. " +
				"To delete you must pass dry_run=false AND confirm=\"DELETE\". It never touches assets, imports, intros, music, progress.json, the user's game or other projects' temp folders. Reports free disk before and after.",
			InputSchema: obj(map[string]any{
				"dry_run":  map[string]any{"type": "boolean", "description": "default true: only list"},
				"confirm":  map[string]any{"type": "string", "description": "must be DELETE to really delete"},
				"go_cache": map[string]any{"type": "boolean", "description": "also run go clean -cache -testcache"},
			}),
			run: cleanTemp,
		},
		&tool{
			Name:        "disk_report",
			Description: "CLEANUP / BUILD role. Free and total disk space, and the size of the things that grow: Go cache, ph_* temp entries, dist/, the zips, imports/, intros/, music/. Read-only.",
			InputSchema: obj(map[string]any{}),
			run:         diskReport,
		},
	)
}

var reTempName = regexp.MustCompile(`^(ph_|phmcp-)[A-Za-z0-9_.-]*$`)

// tempRoots: the folders where the allowlist applies (canonical paths, no duplicates).
func tempRoots() []string {
	raw := []string{"/tmp", os.TempDir()}
	if v := os.Getenv("PHMCP_TMP"); v != "" {
		raw = strings.Split(v, ":")
	}
	seen := map[string]bool{}
	var out []string
	for _, r := range raw {
		c, err := filepath.EvalSymlinks(r)
		if err != nil {
			continue
		}
		c, err = filepath.Abs(c)
		if err != nil || seen[c] || c == "/" {
			continue
		}
		seen[c] = true
		out = append(out, c)
	}
	return out
}

func inRoots(roots []string, dir string) bool {
	for _, r := range roots {
		if dir == r {
			return true
		}
	}
	return false
}

type tempEntry struct {
	path string
	size int64
	link bool
}

// tempCandidates: the entries of the allowlist that exist right now.
func tempCandidates(roots []string) []tempEntry {
	var out []tempEntry
	for _, root := range roots {
		ents, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range ents {
			if !reTempName.MatchString(e.Name()) {
				continue
			}
			p := filepath.Join(root, e.Name())
			st, err := os.Lstat(p)
			if err != nil {
				continue
			}
			te := tempEntry{path: p, link: st.Mode()&os.ModeSymlink != 0}
			if !te.link {
				te.size = treeSize(p)
			}
			out = append(out, te)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].path < out[j].path })
	return out
}

// treeSize adds the sizes of the regular files below p (links are not followed).
func treeSize(p string) int64 {
	var n int64
	deadline := time.Now().Add(10 * time.Second)
	filepath.WalkDir(p, func(_ string, d os.DirEntry, err error) error {
		if err != nil || time.Now().After(deadline) {
			return filepath.SkipAll
		}
		if d.Type().IsRegular() {
			if info, err := d.Info(); err == nil {
				n += info.Size()
			}
		}
		return nil
	})
	return n
}

type procInfo struct {
	pid int
	cmd string
}

// allowProc says whether a process command line may be killed by clean_temp.
func allowProc(cmd string) bool {
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return false
	}
	exe := fields[0]
	if strings.Contains(exe, "..") {
		return false
	}
	switch {
	case strings.HasPrefix(exe, "/tmp/ph_"), strings.HasPrefix(exe, "/private/tmp/ph_"):
		return true
	case filepath.Base(exe) == "game" && strings.Contains(filepath.Dir(exe), "/phmcp-"):
		return true
	case filepath.Base(exe) == "partyhard.test":
		return true
	}
	return false
}

// parsePS reads the output of `ps -axo pid=,command=`.
func parsePS(out string) []procInfo {
	var res []procInfo
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		sp := strings.IndexByte(line, ' ')
		if sp < 0 {
			continue
		}
		pid, err := strconv.Atoi(line[:sp])
		if err != nil || pid <= 1 || pid == os.Getpid() {
			continue
		}
		res = append(res, procInfo{pid, strings.TrimSpace(line[sp+1:])})
	}
	return res
}

// listProcs returns the output of ps; the tests replace it so that they never see (or kill)
// real processes of the machine.
var listProcs = func() string {
	out, err := runLimited("", 15*time.Second, nil, "ps", "-axo", "pid=,command=")
	if err != nil {
		return ""
	}
	return out
}

func allowedProcs() []procInfo {
	var res []procInfo
	for _, p := range parsePS(listProcs()) {
		if allowProc(p.cmd) {
			res = append(res, p)
		}
	}
	return res
}

func freeDisk(path string) (free, total uint64) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, 0
	}
	return uint64(st.Bavail) * uint64(st.Bsize), uint64(st.Blocks) * uint64(st.Bsize)
}

func gb(n uint64) string { return fmt.Sprintf("%.1f GB", float64(n)/1e9) }

// duKB measures a folder with `du -sk` (fixed arguments, time limit).
func duKB(path string) int64 {
	out, err := runLimited("", 40*time.Second, nil, "du", "-sk", path)
	if err != nil && out == "" {
		return -1
	}
	f := strings.Fields(out)
	if len(f) == 0 {
		return -1
	}
	n, err := strconv.ParseInt(f[0], 10, 64)
	if err != nil {
		return -1
	}
	return n
}

func goCachePath() string {
	out, err := runLimited("", 30*time.Second, nil, "go", "env", "GOCACHE")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

func cleanTemp(_ string, a map[string]any) (string, error) {
	dry := true
	if v, ok := a["dry_run"].(bool); ok {
		dry = v
	}
	if !dry && argStr(a, "confirm") != "DELETE" {
		return "", fmt.Errorf("refused: to delete you must pass dry_run=false AND confirm=\"DELETE\"")
	}
	withCache, _ := a["go_cache"].(bool)
	roots := tempRoots()
	var sb strings.Builder
	freeBefore, _ := freeDisk("/")
	if dry {
		sb.WriteString("CLEAN (DRY RUN: nothing is deleted)\n")
	} else {
		sb.WriteString("CLEAN (DELETING the allowlist)\n")
	}
	fmt.Fprintf(&sb, "allowed temp roots: %s\n", strings.Join(roots, ", "))
	procs := allowedProcs()
	fmt.Fprintf(&sb, "\nstuck processes (%d):\n", len(procs))
	killed := 0
	for _, p := range procs {
		fmt.Fprintf(&sb, "  pid %d  %s\n", p.pid, tail(p.cmd, 1))
		if !dry {
			// look again right before killing: the command line must still be in the allowlist
			for _, q := range allowedProcs() {
				if q.pid == p.pid && q.cmd == p.cmd {
					if syscall.Kill(p.pid, syscall.SIGKILL) == nil {
						killed++
					}
					break
				}
			}
		}
	}
	ents := tempCandidates(roots)
	fmt.Fprintf(&sb, "\ntemp entries ph_* / phmcp-* (%d):\n", len(ents))
	var total int64
	removed := 0
	for _, e := range ents {
		kind := "dir/file"
		if e.link {
			kind = "symlink (only the link is removed)"
		}
		fmt.Fprintf(&sb, "  %s  %.1f MB  %s\n", e.path, float64(e.size)/1e6, kind)
		total += e.size
		if !dry {
			// the name was validated by reTempName and the parent must be one of the canonical allowed roots
			if inRoots(roots, filepath.Dir(e.path)) && reTempName.MatchString(filepath.Base(e.path)) && os.RemoveAll(e.path) == nil {
				removed++
			}
		}
	}
	fmt.Fprintf(&sb, "  total about %.1f MB\n", float64(total)/1e6)
	if withCache {
		gc := goCachePath()
		kb := int64(-1)
		if gc != "" {
			kb = duKB(gc)
		}
		fmt.Fprintf(&sb, "\nGo cache: %s (%.1f GB)\n", gc, float64(kb)/1e6)
		if !dry {
			out, err := runLimited("", 180*time.Second, nil, "go", "clean", "-cache", "-testcache")
			if err != nil {
				fmt.Fprintf(&sb, "go clean FAILED: %v %s\n", err, tail(out, 3))
			} else {
				sb.WriteString("go clean -cache -testcache: done\n")
			}
		}
	}
	if !dry {
		fmt.Fprintf(&sb, "\nkilled %d processes, removed %d temp entries\n", killed, removed)
	}
	freeAfter, tot := freeDisk("/")
	fmt.Fprintf(&sb, "\ndisk free: before %s, now %s (total %s)", gb(freeBefore), gb(freeAfter), gb(tot))
	return sb.String(), nil
}

func diskReport(root string, _ map[string]any) (string, error) {
	var sb strings.Builder
	free, total := freeDisk("/")
	fmt.Fprintf(&sb, "DISK REPORT\ndisk: %s free of %s\n", gb(free), gb(total))
	if gc := goCachePath(); gc != "" {
		fmt.Fprintf(&sb, "Go cache %s: %.2f GB\n", gc, float64(duKB(gc))/1e6)
	}
	ents := tempCandidates(tempRoots())
	var n int64
	for _, e := range ents {
		n += e.size
	}
	fmt.Fprintf(&sb, "temp ph_* / phmcp-*: %d entries, %.1f MB\n", len(ents), float64(n)/1e6)
	fmt.Fprintf(&sb, "stuck processes in the allowlist: %d\n", len(allowedProcs()))
	for _, name := range []string{"dist", "PartyHard-Mac.zip", "PartyHard-Windows.zip", "imports", "intros", "music", "more", "DJ", "assets"} {
		p := filepath.Join(root, name)
		if _, err := os.Lstat(p); err != nil {
			continue
		}
		fmt.Fprintf(&sb, "%s: %.1f MB\n", name, float64(duKB(p))/1e3)
	}
	return sb.String(), nil
}
