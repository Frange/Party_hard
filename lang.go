package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ---------------------------------------------------------------------------
// Languages. All the text in the code is written in Spanish (a few HUD labels
// and titles in English); tr() translates whatever is about to be drawn into
// the language in force. The dictionary (lang_data.go) maps whole strings, and
// templates with %s / %d / {name} placeholders are matched by pattern so that
// formatted messages are translated too. The game and the editor each have
// their own language.
// ---------------------------------------------------------------------------

var (
	gameLang   = "es"
	editorLang = "es"
	uiLang     = "es" // the language of whatever is being drawn right now
)

type langFile struct {
	Game   string `json:"game"`
	Editor string `json:"editor"`
}

func langPath() string {
	root := assetsRoot(true)
	if root == "" {
		return ""
	}
	return filepath.Join(root, "language.json")
}

func loadLanguages() {
	p := langPath()
	if p == "" || noAssets {
		return
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return
	}
	var lf langFile
	if json.Unmarshal(data, &lf) != nil {
		return
	}
	if lf.Game == "en" || lf.Game == "es" {
		gameLang = lf.Game
	}
	if lf.Editor == "en" || lf.Editor == "es" {
		editorLang = lf.Editor
	}
}

func saveLanguages() {
	p := langPath()
	if p == "" || noAssets || progressFrozen {
		return
	}
	data, _ := json.MarshalIndent(langFile{gameLang, editorLang}, "", "  ")
	os.WriteFile(p, data, 0o644)
}

func otherLang(l string) string {
	if l == "en" {
		return "es"
	}
	return "en"
}

// ---- translation ------------------------------------------------------------------

type trTemplate struct {
	re   *regexp.Regexp
	to   string
	keep []bool // true: the capture is a number/format that is copied as is
}

type trTable struct {
	exact    map[string]string
	tpls     []trTemplate
	replacer *strings.Replacer
	cache    map[string]string
}

var phRe = regexp.MustCompile(`%[-+#0-9.]*[sdvqfgxXtc%]|\{[a-z_]+\}`)

func buildTable(src map[string]string) *trTable {
	t := &trTable{exact: src, cache: map[string]string{}}
	var keys []string
	for k := range src {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	var pairs []string
	frag := map[string]string{}
	for _, k := range keys {
		v := src[k]
		if phRe.MatchString(k) {
			if tp, ok := makeTemplate(k, v); ok {
				t.tpls = append(t.tpls, tp)
			}
			// the text between the placeholders also translates on its own, which
			// catches the same sentence with something added at the end
			ks, vs := phRe.Split(k, -1), phRe.Split(v, -1)
			if len(ks) == len(vs) {
				for i := range ks {
					if len(strings.TrimSpace(ks[i])) >= 5 && ks[i] != vs[i] && containsLetter(ks[i]) {
						frag[ks[i]] = vs[i]
					}
				}
			}
			continue
		}
		if len([]rune(k)) >= 7 {
			pairs = append(pairs, k, v)
		}
	}
	var fk []string
	for k := range frag {
		if _, dup := src[k]; !dup {
			fk = append(fk, k)
		}
	}
	sort.Slice(fk, func(i, j int) bool { return len(fk[i]) > len(fk[j]) })
	for _, k := range fk {
		pairs = append(pairs, k, frag[k])
	}
	// longest first, so that the most specific piece wins
	type kv struct{ k, v string }
	var all []kv
	for i := 0; i+1 < len(pairs); i += 2 {
		all = append(all, kv{pairs[i], pairs[i+1]})
	}
	sort.SliceStable(all, func(i, j int) bool { return len(all[i].k) > len(all[j].k) })
	pairs = pairs[:0]
	for _, e := range all {
		pairs = append(pairs, e.k, e.v)
	}
	if len(pairs) > 0 {
		t.replacer = strings.NewReplacer(pairs...)
	}
	return t
}

func makeTemplate(key, val string) (trTemplate, bool) {
	var sb strings.Builder
	sb.WriteString("^")
	var keep []bool
	last := 0
	for _, loc := range phRe.FindAllStringIndex(key, -1) {
		sb.WriteString(regexp.QuoteMeta(key[last:loc[0]]))
		tok := key[loc[0]:loc[1]]
		switch {
		case tok == "%%":
			sb.WriteString("%")
		case strings.HasSuffix(tok, "d"), strings.HasSuffix(tok, "f"), strings.HasSuffix(tok, "g"):
			sb.WriteString(`(-?[\d.,]+)`)
			keep = append(keep, true)
		default:
			sb.WriteString(`(.+?)`)
			keep = append(keep, false)
		}
		last = loc[1]
	}
	sb.WriteString(regexp.QuoteMeta(key[last:]))
	sb.WriteString("$")
	re, err := regexp.Compile(sb.String())
	if err != nil {
		return trTemplate{}, false
	}
	return trTemplate{re: re, to: val, keep: keep}, true
}

var trEN, trES *trTable

func initTables() {
	if trEN == nil {
		trEN = buildTable(langDataEN)
		trES = buildTable(langDataES)
	}
}

// tr translates s into the language in force (uiLang).
func tr(s string) string {
	if s == "" {
		return s
	}
	initTables()
	t := trES
	if uiLang == "en" {
		t = trEN
	}
	if r, ok := t.cache[s]; ok {
		return r
	}
	r := t.translate(s, 0)
	if len(t.cache) > 20000 {
		t.cache = map[string]string{}
	}
	t.cache[s] = r
	return r
}

func (t *trTable) translate(s string, depth int) string {
	if r, ok := t.exact[s]; ok {
		return r
	}
	if depth < 2 {
		for _, tp := range t.tpls {
			m := tp.re.FindStringSubmatch(s)
			if m == nil {
				continue
			}
			i := 0
			out := phRe.ReplaceAllStringFunc(tp.to, func(tok string) string {
				if tok == "%%" {
					return "%"
				}
				if i+1 >= len(m) {
					return tok
				}
				cap := m[i+1]
				k := i
				i++
				if k < len(tp.keep) && tp.keep[k] {
					return cap
				}
				return t.translate(cap, depth+1)
			})
			return out
		}
	}
	if t.replacer != nil {
		return t.replacer.Replace(s)
	}
	return s
}

// pickLanguage sets the language used while the game (or the editor) updates and draws.
func (g *Game) pickLanguage() {
	if g.mode == ModeEditor {
		uiLang = editorLang
	} else {
		uiLang = gameLang
	}
}

func containsLetter(s string) bool {
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r > 127 {
			return true
		}
	}
	return false
}
