package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"math/rand"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

func mkReqs(n, pri int) []bubReq {
	var r []bubReq
	for i := 0; i < n; i++ {
		r = append(r, bubReq{key: i + pri*1000, pri: pri, score: float64(pri)*10000 - float64(i), cx: 100 + float64(i)*90, y: 100, w: 60})
	}
	return r
}

// a dozen guests screaming at once: never more than bubMax on screen, and they come in one by one
func TestBubbleLimitAndStagger(t *testing.T) {
	var m bubMgr
	shown := map[any]bool{}
	firstAt := []float64{}
	for f := 1; f <= 240; f++ {
		now := float64(f) / 60
		got := m.pick(mkReqs(12, bubGuest), now, nil, 960)
		if len(got) > bubMax {
			t.Fatalf("frame %d: %d bubbles on screen", f, len(got))
		}
		for _, r := range got {
			if !shown[r.key] {
				shown[r.key] = true
				firstAt = append(firstAt, now)
			}
		}
	}
	if len(shown) < bubMax {
		t.Fatalf("only %d different bubbles were ever shown", len(shown))
	}
	for i := 1; i < len(firstAt); i++ {
		if firstAt[i]-firstAt[i-1] < bubGap-0.02 {
			t.Fatalf("bubbles %d and %d came %.2fs apart", i-1, i, firstAt[i]-firstAt[i-1])
		}
	}
}

// alarms of the authorities push out filler chatter, and chatter never gets in when it is crowded
func TestBubblePriority(t *testing.T) {
	var m bubMgr
	filler := mkReqs(6, bubGuest)
	for f := 1; f <= 120; f++ {
		m.pick(append([]bubReq(nil), filler...), float64(f)/60, nil, 960)
	}
	auth := []bubReq{{key: "cop", pri: bubAuth, score: 30000, cx: 400, y: 40, w: 50}}
	var got []bubReq
	for f := 121; f <= 160; f++ {
		got = m.pick(append(append([]bubReq(nil), filler...), auth...), float64(f)/60, nil, 960)
	}
	found := false
	for _, r := range got {
		if r.key == "cop" {
			found = true
		}
	}
	if !found || len(got) > bubMax {
		t.Fatalf("authority bubble missing or too many (%d): %v", len(got), found)
	}
}

// bubbles do not overlap each other and do not cover the player
func TestBubbleNoOverlapNoPlayer(t *testing.T) {
	var m bubMgr
	var reqs []bubReq
	for i := 0; i < 4; i++ {
		reqs = append(reqs, bubReq{key: i, pri: bubGuest, score: float64(-i), cx: 300, y: 120, w: 80})
	}
	player := &bubRect{270, 100, 330, 140}
	var got []bubReq
	for f := 1; f <= 200; f++ {
		got = m.pick(append([]bubReq(nil), reqs...), float64(f)/60, player, 960)
	}
	if len(got) == 0 {
		t.Fatal("nothing shown")
	}
	for i, a := range got {
		x := a.cx - a.w/2
		ra := bubRect{x, a.y - a.shift, x + a.w, a.y - a.shift + bubH}
		if ra.hit(*player) {
			t.Fatalf("bubble %d covers the player", i)
		}
		for j := i + 1; j < len(got); j++ {
			b := got[j]
			xb := b.cx - b.w/2
			rb := bubRect{xb, b.y - b.shift, xb + b.w, b.y - b.shift + bubH}
			if ra.hit(rb) {
				t.Fatalf("bubbles %d and %d overlap", i, j)
			}
		}
	}
}

// one bubble does not stay forever: it rests after bubLife
func TestBubbleExpires(t *testing.T) {
	var m bubMgr
	one := []bubReq{{key: "a", pri: bubGuest, score: 1, cx: 100, y: 100, w: 40}}
	shownLate := false
	for f := 1; f <= 60*8; f++ {
		now := float64(f) / 60
		got := m.pick(append([]bubReq(nil), one...), now, nil, 960)
		if now > bubGap+bubLife+0.2 && now < bubGap+bubLife+bubCool-0.3 && len(got) > 0 {
			shownLate = true
		}
	}
	if shownLate {
		t.Fatal("bubble stayed beyond its life")
	}
}

func fakeW(s string) float64 { return float64(len([]rune(s))) * 6 }

// a bubble never has more than 2 lines, none wider than the limit, and long text is cut with "..."
func TestWrapBubble(t *testing.T) {
	cases := []string{
		"¡Hola!",
		"Dicen que hay un asesino, pero yo vengo por el catering y no me voy",
		"Esta frase es muy muy larga y además sigue y sigue sin parar nunca jamás de los jamases del mundo entero",
		"SupercalifragilisticoespialidosoSupercalifragilisticoespialidosoSupercalifragilistico",
		"a b c d e f g h i j k l m n o p q r s t u v w x y z a b c d e f g h i j k l m",
	}
	for _, c := range cases {
		lines := wrapBubble(c, bubMaxW, fakeW, bubLines)
		if len(lines) < 1 || len(lines) > bubLines {
			t.Fatalf("%q: %d lines", c, len(lines))
		}
		for _, l := range lines {
			if fakeW(l) > bubMaxW+0.1 {
				t.Fatalf("%q: line %q too wide", c, l)
			}
		}
	}
	short := wrapBubble("Qué marcha", bubMaxW, fakeW, bubLines)
	if len(short) != 1 || short[0] != "Qué marcha" {
		t.Fatalf("short text changed: %v", short)
	}
	long := wrapBubble(cases[2], bubMaxW, fakeW, bubLines)
	if len(long) != 2 || long[1][len(long[1])-3:] != "..." {
		t.Fatalf("long text should end with ...: %v", long)
	}
}

const maxPhrase = 48 // characters: longer lines are not funnier, they just cover the screen

// every phrase that can end up in a speech bubble is short
func TestBubblePhrasesAreShort(t *testing.T) {
	long := func(where, s string) {
		if n := utf8.RuneCountInString(s); n > maxPhrase {
			t.Errorf("%s: %d chars: %q", where, n, s)
		}
	}
	for cat, sits := range defaultDialogueBank() {
		for sit, lines := range sits {
			for _, l := range lines {
				long("dialogue "+cat+"/"+sit, l)
			}
		}
	}
	for _, l := range greetLines {
		long("greetLines", l)
	}
	for k, ls := range catchphrases {
		for _, l := range ls {
			long("catchphrase "+k, l)
		}
	}
	for _, l := range deadpoolLines {
		long("deadpoolLines", l)
	}
	for _, d := range duetDefs {
		for _, st := range d.steps {
			long("duet", st.line)
		}
	}
	for _, sc := range duetGeneric {
		for _, st := range sc {
			long("duet generic", st.line)
		}
	}
	// literals handed straight to a bubble: x.icon = "..", bubbleAt(.., "..", ..), alarm(g, "..")
	files, _ := filepath.Glob("*.go")
	fset := token.NewFileSet()
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		af, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			continue
		}
		lit := func(e ast.Expr) (string, bool) {
			bl, ok := e.(*ast.BasicLit)
			if !ok || bl.Kind != token.STRING {
				return "", false
			}
			s, err := strconv.Unquote(bl.Value)
			return s, err == nil
		}
		ast.Inspect(af, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.AssignStmt:
				for i, l := range x.Lhs {
					if sel, ok := l.(*ast.SelectorExpr); ok && sel.Sel.Name == "icon" && i < len(x.Rhs) {
						if s, ok := lit(x.Rhs[i]); ok {
							long(fset.Position(x.Pos()).String(), s)
						}
					}
				}
			case *ast.CallExpr:
				name := ""
				switch fn := x.Fun.(type) {
				case *ast.Ident:
					name = fn.Name
				case *ast.SelectorExpr:
					name = fn.Sel.Name
				}
				if name == "bubbleAt" || name == "alarm" || name == "ds" {
					for _, a := range x.Args {
						if s, ok := lit(a); ok {
							long(fset.Position(x.Pos()).String(), s)
						}
					}
				}
			}
			return true
		})
	}
}

func sentence(key any, txt string, x float64) bubReq {
	return bubReq{key: key, pri: bubGuest, score: -x, cx: x, y: 100, w: 60, lines: []string{txt}}
}

// the same sentence is never on screen twice, and does not come back within the cooldown
func TestBubbleSameTextCooldown(t *testing.T) {
	var m bubMgr
	const txt = "Qué raro, huele a quemado"
	lastShown := -100.0
	for f := 1; f <= 60*40; f++ {
		now := float64(f) / 60
		reqs := []bubReq{sentence("a", txt, 100), sentence("b", txt, 300), sentence("c", txt, 500)}
		got := m.pick(reqs, now, nil, 960)
		if len(got) > 1 {
			t.Fatalf("t=%.1f: %d identical bubbles on screen", now, len(got))
		}
		if len(got) == 1 {
			if now-lastShown > 0.5 && now-lastShown < bubTextCool && lastShown > 0 {
				t.Fatalf("t=%.1f: the sentence came back only %.1fs after it was last shown", now, now-lastShown)
			}
			lastShown = now
		}
	}
	// shouts without words may repeat
	var m2 bubMgr
	both := 0
	for f := 1; f <= 60*6; f++ {
		got := m2.pick([]bubReq{sentence("a", "!?", 100), sentence("b", "!?", 300)}, float64(f)/60, nil, 960)
		if len(got) == 2 {
			both++
		}
	}
	if both == 0 {
		t.Fatal("emotes should be allowed to repeat")
	}
}

// the same character does not repeat what it just said
func TestBubbleSameCharacterNoRepeat(t *testing.T) {
	var m bubMgr
	shown := 0
	for f := 1; f <= 60*30; f++ {
		got := m.pick([]bubReq{sentence("a", "Mañana empiezo la dieta", 100)}, float64(f)/60, nil, 960)
		if len(got) > 0 {
			shown++
		}
	}
	if shown == 0 || shown > int(bubLife*60)+60 {
		t.Fatalf("one character repeated itself: shown on %d frames", shown)
	}
}

func fakeSay(t *testing.T) *float64 {
	now := 1000.0
	old := sayClock
	sayClock = func() float64 { return now }
	sayState.mem, sayState.last, sayState.queue, sayState.shownAt = nil, "", nil, 0
	t.Cleanup(func() { sayClock = old })
	return &now
}

func TestSayDedupe(t *testing.T) {
	now := fakeSay(t)
	g := &Game{}
	g.say("Has matado a un perro...")
	g.say("Has matado a un perro...")
	if len(g.msgs) != 1 || len(sayState.queue) != 0 {
		t.Fatalf("a repeated message got in: %d shown, %d queued", len(g.msgs), len(sayState.queue))
	}
	*now += 10
	g.say("Has matado a un perro...")
	if len(g.msgs)+len(sayState.queue) != 1 {
		t.Fatal("the same message came back inside the cooldown")
	}
	*now += sayCool
	g.msgs = nil
	sayState.queue = nil
	g.say("Has matado a un perro...")
	if len(g.msgs) != 0 {
		t.Fatal("the same message twice in a row must not show, even after the cooldown")
	}
	g.say("Otra cosa distinta")
	g.msgs = nil
	*now += 2
	g.say("Has matado a un perro...")
	if len(g.msgs) != 1 {
		t.Fatal("the message should be allowed again after the cooldown and something else in between")
	}
	// numbers make a different text
	g.msgs = nil
	*now += 5
	g.say("La manada deja 3 víctimas")
	*now += 1
	g.say("La manada deja 4 víctimas")
	if len(g.msgs)+len(sayState.queue) != 2 {
		t.Fatalf("messages that differ in a number were merged: %d+%d", len(g.msgs), len(sayState.queue))
	}
}

// several different messages at once wait their turn; never more than sayShown on screen
func TestSayQueue(t *testing.T) {
	now := fakeSay(t)
	g := &Game{}
	for i := 0; i < 5; i++ {
		g.say(string(rune('A'+i)) + " algo ha pasado en la fiesta")
	}
	if len(g.msgs) > 1 { // the first one appears, the rest wait (they come one at a time)
		t.Fatalf("%d messages popped up at once", len(g.msgs))
	}
	seen := map[string]bool{}
	for step := 0; step < 200; step++ {
		*now += 0.1
		for i := range g.msgs {
			g.msgs[i].t -= 0.1
		}
		for len(g.msgs) > 0 && g.msgs[0].t <= 0 {
			g.msgs = g.msgs[1:]
		}
		g.sayPromote()
		if len(g.msgs) > sayShown {
			t.Fatalf("%d messages on screen", len(g.msgs))
		}
		for _, m := range g.msgs {
			seen[m.text] = true
		}
	}
	if len(seen) < 3 {
		t.Fatalf("only %d of the queued messages ever appeared", len(seen))
	}
}

func TestFreshPhrase(t *testing.T) {
	phraseMem = map[string]float64{}
	g := &Game{}
	rng := rand.New(rand.NewSource(1))
	lines := []string{"uno uno uno", "dos dos dos", "tres tres tres"}
	got := map[string]bool{}
	for i := 0; i < 3; i++ {
		l := g.freshPhrase("people/scared", lines, rng.Intn)
		if l == "" || got[l] {
			t.Fatalf("pick %d: %q (repeated or empty)", i, l)
		}
		got[l] = true
	}
	if l := g.freshPhrase("people/scared", lines, rng.Intn); l != "" {
		t.Fatalf("with everything said lately it must stay quiet, got %q", l)
	}
	if l := g.freshPhrase("police/halt", lines[:1], rng.Intn); l != "" {
		t.Fatalf("another category repeated a sentence said 0s ago: %q", l)
	}
	g.time += phraseCatCool + 1
	if l := g.freshPhrase("people/scared", lines, rng.Intn); l == "" {
		t.Fatal("after the cooldown they may speak again")
	}
	// shouts always come out
	if l := g.freshPhrase("x", []string{"!!!"}, rng.Intn); l != "!!!" {
		t.Fatal("emotes must not be blocked")
	}
}

// a character that changes what it says while its bubble is up cannot make a duplicate
func TestBubbleChangedTextNoDuplicate(t *testing.T) {
	var m bubMgr
	for f := 1; f <= 60*20; f++ {
		now := float64(f) / 60
		bText := "Otra frase distinta"
		if now > 3 {
			bText = "Qué marcha tiene esta fiesta" // b copies a
		}
		got := m.pick([]bubReq{sentence("a", "Qué marcha tiene esta fiesta", 100), sentence("b", bText, 300)}, now, nil, 960)
		seen := map[string]bool{}
		for _, r := range got {
			if seen[r.text()] {
				t.Fatalf("t=%.1f: %q twice on screen", now, r.text())
			}
			seen[r.text()] = true
		}
	}
}
