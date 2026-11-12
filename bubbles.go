package main

import (
	"math"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Speech bubble manager. Everything that talks (guests, guards, cops, the
// slasher, animals, event actors) asks for a bubble every frame it wants one;
// the manager decides which ones are really drawn:
//   - at most bubMax at a time (a roar must not cover the whole screen),
//   - the important ones first (alarms of guards and cops), then the nearest
//     to the player; filler chatter only when there is room,
//   - a bubble that was admitted stays for a while and then rests, so the set
//     does not flicker,
//   - new ones come in one at a time, at least bubGap seconds apart,
//   - a bubble that would overlap another one is moved up, or dropped,
//   - and they never cover the player.
// ---------------------------------------------------------------------------

const (
	bubMax  = 4
	bubGap  = 0.4 // seconds between two new bubbles
	bubLife = 3.5 // seconds a bubble may stay
	bubCool = 2.0 // rest after it
	bubH    = 15.0

	bubLines   = 2     // a bubble never has more lines than this
	bubMaxW    = 170.0 // widest line, in screen pixels
	bubLineH   = 11.0
	bubBoxBase = 13.0 // height of a one-line box
)

// wrapBubble splits s into at most maxLines lines no wider than maxW (greedy, by words).
// What does not fit is cut with "..." so a bubble never grows past maxLines lines.
func wrapBubble(s string, maxW float64, width func(string) float64, maxLines int) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	cur := ""
	for i, w := range words {
		t := w
		if cur != "" {
			t = cur + " " + w
		}
		if cur != "" && width(t) > maxW {
			if len(lines) == maxLines-1 { // last line: everything left goes here, then it is cut
				cur = strings.Join(words[i-len(strings.Fields(cur)):], " ")
				break
			}
			lines = append(lines, cur)
			cur = w
			continue
		}
		cur = t
	}
	lines = append(lines, cur)
	last := lines[len(lines)-1]
	if width(last) > maxW {
		rs := []rune(last)
		for len(rs) > 1 && width(string(rs)+"...") > maxW {
			rs = rs[:len(rs)-1]
		}
		lines[len(lines)-1] = string(rs) + "..."
	}
	return lines
}

// priorities
const (
	bubFiller = 0 // animals, idle chatter
	bubGuest  = 1 // guests
	bubAlarm  = 2 // somebody screaming
	bubAuth   = 3 // guards, cops, the slasher
)

type bubReq struct {
	key      any
	pri      int
	score    float64  // higher wins
	cx, y, w float64  // screen position of the bubble box (y = top)
	lines    []string // the text, wrapped in at most bubLines lines
	h        float64  // height of the box
	red      bool
	up       float64
	shift    float64 // moved up to avoid overlaps (output)
	loud     bool    // shouted through a megaphone: yellow box, bold text (pe_megaphone.go)
}

type bubSt struct {
	since, seen, cool float64
	admitted          bool
	txt               string // what it was admitted to say: a different text is a new bubble
}

type bubMgr struct {
	st        map[any]*bubSt
	lastAdmit float64
	frame     int
	shownTxt  map[string]float64 // exact text -> last time it was on screen
	keyLast   map[any]keyPhrase  // what each character said last
	extra     []bubRect          // more rectangles no bubble may cover (the second player)
}

type keyPhrase struct {
	txt string
	t   float64
}

const (
	bubTextCool = 13.0 // the same sentence is not shown again (by anybody) for this long
	bubKeyCool  = 60.0 // and the same character does not repeat itself for this long
)

// isEmote: a symbol or a shout without words ("!?", "...", "!!!") may repeat; sentences may not.
func isEmote(s string) bool {
	n := 0
	for _, r := range s {
		if unicode.IsLetter(r) {
			n++
		}
	}
	return n < 3
}

// avoid is a screen rectangle no bubble may cover (the player).
type bubRect struct{ x0, y0, x1, y1 float64 }

func (r bubRect) hit(o bubRect) bool { return r.x0 < o.x1 && o.x0 < r.x1 && r.y0 < o.y1 && o.y0 < r.y1 }

func (r bubReq) text() string { return strings.Join(r.lines, " ") }

// pick chooses and lays out the bubbles of one frame.
func (m *bubMgr) pick(reqs []bubReq, now float64, avoid *bubRect, screenW float64) []bubReq {
	if m.st == nil {
		m.st = map[any]*bubSt{}
		m.shownTxt = map[string]float64{}
		m.keyLast = map[any]keyPhrase{}
	}
	sort.SliceStable(reqs, func(i, j int) bool { return reqs[i].score > reqs[j].score })
	var sel []bubReq
	var fresh []bubReq
	for _, r := range reqs {
		s := m.st[r.key]
		if s == nil {
			s = &bubSt{}
			m.st[r.key] = s
		}
		s.seen = now
		if s.admitted && s.txt != r.text() { // the character changed what it says: it has to be admitted again
			s.admitted, s.cool = false, 0
		}
		if s.admitted {
			if now-s.since > bubLife {
				s.admitted, s.cool = false, now+bubCool
				continue
			}
			if len(sel) >= bubMax { // too many: the weakest rest for a moment
				s.admitted, s.cool = false, now+1
				continue
			}
			sel = append(sel, r)
			continue
		}
		if now >= s.cool && m.mayShow(r, now) {
			fresh = append(fresh, r)
		}
	}
	// one newcomer at a time
	if now-m.lastAdmit >= bubGap {
		for _, r := range fresh {
			full := len(sel) >= bubMax
			if full && r.pri < bubAlarm {
				continue
			}
			if !full && r.pri == bubFiller && len(sel) >= bubMax-1 {
				continue
			}
			if full { // an alarm may push out the weakest lower-priority bubble
				vi := -1
				for i := len(sel) - 1; i >= 0; i-- {
					if sel[i].pri < r.pri {
						vi = i
						break
					}
				}
				if vi < 0 {
					continue
				}
				m.st[sel[vi].key].admitted = false
				m.st[sel[vi].key].cool = now + 1
				sel = append(sel[:vi], sel[vi+1:]...)
			}
			s := m.st[r.key]
			s.admitted, s.since, s.txt = true, now, r.text()
			m.lastAdmit = now
			sel = append(sel, r)
			if t := r.text(); !isEmote(t) {
				m.keyLast[r.key] = keyPhrase{t, now}
			}
			break
		}
	}
	for _, r := range sel { // while it is on screen its text is "in use"
		if t := r.text(); !isEmote(t) {
			m.shownTxt[t] = now
		}
	}
	for t, at := range m.shownTxt {
		if now-at > 2*bubTextCool {
			delete(m.shownTxt, t)
		}
	}
	for k, kp := range m.keyLast {
		if now-kp.t > bubKeyCool {
			delete(m.keyLast, k)
		}
	}
	for k, s := range m.st { // forget the ones not asked for any more
		if now-s.seen > 2.5 && now >= s.cool {
			delete(m.st, k)
		}
	}
	// layout, most important first: move up to avoid overlaps, drop if there is no room
	sort.SliceStable(sel, func(i, j int) bool { return sel[i].score > sel[j].score })
	var placed []bubRect
	out := sel[:0]
	for _, r := range sel {
		x := math.Min(math.Max(r.cx-math.Round(r.w/2), 2), screenW-r.w-2)
		ok := false
		var box bubRect
		bh := r.h
		if bh == 0 {
			bh = bubH
		}
		for step := 0; step < 4; step++ {
			y := r.y - float64(step)*(bh+2)
			box = bubRect{x, y, x + r.w, y + bh}
			hit := avoid != nil && box.hit(*avoid)
			for _, e := range m.extra {
				if box.hit(e) {
					hit = true
				}
			}
			for _, p := range placed {
				if box.hit(p) {
					hit = true
					break
				}
			}
			if !hit {
				r.shift = float64(step) * (bh + 2)
				ok = true
				break
			}
		}
		if !ok {
			s := m.st[r.key]
			s.admitted, s.cool = false, now+1
			continue
		}
		placed = append(placed, box)
		out = append(out, r)
	}
	return out
}

// mayShow: a sentence may not be on screen twice, nor come back within bubTextCool; the same
// character does not repeat its last sentence.
func (m *bubMgr) mayShow(r bubReq, now float64) bool {
	t := r.text()
	if isEmote(t) {
		return true
	}
	if at, ok := m.shownTxt[t]; ok && now-at < bubTextCool {
		return false
	}
	if kp, ok := m.keyLast[r.key]; ok && kp.txt == t && now-kp.t < bubKeyCool {
		return false
	}
	return true
}

var (
	bubbleMgr  bubMgr
	bubbleReqs []bubReq
)

// bubble asks for a speech bubble this frame. key identifies who is talking.
func (g *Game) bubble(key any, pri int, pos Vec, txt string, red bool, up float64) {
	if len(bubbleReqs) > 400 {
		bubbleReqs = bubbleReqs[:0]
	}
	txt = tr(txt)
	lines := wrapBubble(txt, bubMaxW, func(s string) float64 { return g.textW(s, 1) }, bubLines)
	w := 0.0
	for _, l := range lines {
		w = math.Max(w, math.Round(g.textW(l, 1)))
	}
	w += 8
	h := bubBoxBase + bubLineH*float64(len(lines)-1)
	loud := g.bubLoud(key)
	if loud { // a frame and bold letters: a little more room
		w, h = w+6, h+4
	}
	bk := 1.0
	if g.charSc > 0 {
		bk = math.Max(0.6, g.charSc)
	}
	d := 0.0
	if g.p != nil {
		d = dist(pos, g.p.pos)
		for _, pl := range g.players() { // the closest of the players counts
			if pl != nil && !pl.dead {
				d = math.Min(d, dist(pos, pl.pos))
			}
		}
	}
	bubbleReqs = append(bubbleReqs, bubReq{
		key: key, pri: pri, score: float64(pri)*10000 - d,
		cx: g.wx(pos.X), y: g.wy(pos.Y) - 52*bk - up - (h - bubBoxBase), w: w, h: h, lines: lines, red: red, up: up, loud: loud,
	})
}

// flushBubbles draws the bubbles the manager lets through. Call once per frame, after everything that asks for one.
func (g *Game) flushBubbles(dst *ebiten.Image) {
	reqs := bubbleReqs
	bubbleReqs = bubbleReqs[:0]
	bubbleMgr.frame++
	now := float64(bubbleMgr.frame) / 60
	var avoid *bubRect
	bubbleMgr.extra = bubbleMgr.extra[:0]
	bk := 1.0
	if g.charSc > 0 {
		bk = math.Max(0.6, g.charSc)
	}
	for _, pl := range g.players() { // no bubble may cover any of the players
		if pl == nil || pl.dead {
			continue
		}
		cx, fy := g.wx(pl.pos.X), g.wy(pl.pos.Y)
		r := bubRect{cx - 9, fy - 40*bk, cx + 9, fy + 2}
		if avoid == nil {
			avoid = &r
		} else {
			bubbleMgr.extra = append(bubbleMgr.extra, r)
		}
	}
	// the use prompts ("[E] ...") and the HUD go first: a bubble moves up off them or waits (hudclear.go)
	bubbleMgr.extra = append(bubbleMgr.extra, g.usePromptRects()...)
	if hr, ok := g.hudBlockRect(); ok {
		bubbleMgr.extra = append(bubbleMgr.extra, bubRect{hr.X - 4, hr.Y - 4, hr.X + hr.W + 4, hr.Y + hr.H + 4})
	}
	pos := bubbleMgr.pick(reqs, now, avoid, screenW)
	g.drawEmotes(dst, pos) // icons over the guests' heads, never over a bubble (emote.go)
	for _, r := range pos {
		g.drawBubbleBox(dst, r)
	}
}

// ---- picking phrases that were not said a moment ago ------------------------------

const (
	phraseCool    = 15.0 // seconds before anybody repeats a sentence
	phraseCatCool = 45.0 // seconds before the same kind of character repeats it
)

var phraseMem = map[string]float64{}

// freshPhrase picks a random line that has not been said lately (by anybody, nor by this kind of
// character). With nothing fresh it returns "" and the caller says nothing, instead of repeating.
func (g *Game) freshPhrase(cat string, lines []string, intn func(int) int) string {
	recent := func(k string, cool float64) bool {
		t, ok := phraseMem[k]
		return ok && t <= g.time && g.time-t < cool
	}
	var ok []string
	for _, l := range lines {
		if isEmote(l) || (!recent(l, phraseCool) && !recent(cat+"|"+l, phraseCatCool)) {
			ok = append(ok, l)
		}
	}
	if len(ok) == 0 {
		return ""
	}
	l := ok[intn(len(ok))]
	if !isEmote(l) {
		phraseMem[l], phraseMem[cat+"|"+l] = g.time, g.time
	}
	if len(phraseMem) > 600 {
		phraseMem = map[string]float64{}
	}
	return l
}

// ---- the news line under the stars (g.say) ---------------------------------------
//
// The same text is not shown twice within sayCool seconds, nor twice in a row; texts that differ
// in a number ("3 víctimas", "4 víctimas") are different texts. When two different messages arrive
// together the second one waits its turn instead of pushing the first one away.

const (
	sayCool   = 20.0 // seconds before the exact same message may come back
	sayGap    = 0.7  // seconds between two messages appearing
	sayStale  = 8.0  // a message that waited this long is no longer news
	sayShown  = 2    // lines on screen at once
	sayQueued = 6
)

type sayItem struct {
	text string
	at   float64
}

var sayState struct {
	mem     map[string]float64
	last    string
	lastAt  float64
	shownAt float64
	queue   []sayItem
}

// sayClock is real time, so that it keeps running while the game is paused. Tests replace it.
var sayClock = func() float64 { return float64(time.Now().UnixNano()) / 1e9 }

func sayReset() {
	sayState.queue = nil
	sayState.shownAt = 0
}

// sayRecent: would this exact message be dropped right now?
func sayRecent(s string) bool {
	now := sayClock()
	if t, ok := sayState.mem[s]; ok && now-t < sayCool {
		return true
	}
	return s == sayState.last && now-sayState.lastAt < 60
}

func (g *Game) sayShow(s string) {
	g.msgs = append(g.msgs, HudMsg{s, 4.5})
	sayState.shownAt = sayClock()
}

// sayOffer is what g.say does: drop repeats, show now or queue.
func (g *Game) sayOffer(s string) {
	if sayState.mem == nil {
		sayState.mem = map[string]float64{}
	}
	if sayRecent(s) {
		return
	}
	now := sayClock()
	sayState.mem[s], sayState.last, sayState.lastAt = now, s, now
	if len(sayState.mem) > 300 {
		for k, t := range sayState.mem {
			if now-t > sayCool {
				delete(sayState.mem, k)
			}
		}
	}
	if len(g.msgs) < sayShown && len(sayState.queue) == 0 && now-sayState.shownAt >= sayGap {
		g.sayShow(s)
		return
	}
	sayState.queue = append(sayState.queue, sayItem{s, now})
	if len(sayState.queue) > sayQueued {
		sayState.queue = sayState.queue[len(sayState.queue)-sayQueued:]
	}
}

// sayPromote moves waiting messages onto the screen as room appears. Call every update.
func (g *Game) sayPromote() {
	now := sayClock()
	for len(sayState.queue) > 0 && now-sayState.queue[0].at > sayStale {
		sayState.queue = sayState.queue[1:]
	}
	if len(sayState.queue) > 0 && len(g.msgs) < sayShown && now-sayState.shownAt >= sayGap {
		g.sayShow(sayState.queue[0].text)
		sayState.queue = sayState.queue[1:]
	}
}
