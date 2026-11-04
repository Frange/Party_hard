package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// every trophy counts a stat; if nothing ever writes that stat the trophy can never be won
func TestTrophyStatsAreWritten(t *testing.T) {
	files, _ := filepath.Glob("*.go")
	var src strings.Builder
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") || f == "trophies.go" {
			continue
		}
		b, _ := os.ReadFile(f)
		src.Write(b)
	}
	all := src.String()
	for _, tr := range trophyList {
		if tr.id == legendTrophy {
			continue // won by having every other trophy
		}
		s := tr.stat
		switch {
		case strings.HasPrefix(s, "kill_"):
			if !strings.Contains(all, `g.stat("kill_"+`) {
				t.Errorf("%s: kill_* stats are never written", tr.id)
			}
		case strings.HasPrefix(s, "ev_"):
			if !strings.Contains(all, `"ev_%d"`) {
				t.Errorf("%s: ev_* stats are never written", tr.id)
			}
		default:
			if !strings.Contains(all, `stat("`+s+`"`) && !strings.Contains(all, `statMax("`+s+`"`) {
				t.Errorf("trophy %s (%s) can never be won: nothing writes the stat %q", tr.id, tr.name, s)
			}
		}
	}
}

// the animals a trophy asks for must be able to appear
func TestPigsAppearOnceBought(t *testing.T) {
	b, _ := os.ReadFile("animals3.go")
	if !strings.Contains(string(b), "animalAllowed(animPig)") {
		t.Fatal("pigs are never placed in the parties")
	}
}
