package main

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// PARTY EVENTS: the common FRAME for the "sometimes" things of a party (toast, emergency lights, and the many that
// will come: conga, magician, stray dog, waiters, skaters, mime, TV reporter...). Each event lives in ITS OWN file
// and only registers itself; this file decides WHEN, WHERE and HOW OFTEN.
//
// THE RULE OF THE USER: they happen "sometimes": every event has a chance PER PARTY (10-40 %), only in the parties
// where it makes sense (Eligible), at a moment of a time window, and the last two parties never repeat the same
// event. Never all of them everywhere.
//
// ADD AN EVENT IN 5 LINES (own file, e.g. conga.go):
//
//	func init() { registerPartyEvent(&partyEvent{ID: "conga", Name: "Conga", Group: "dance", Big: true,
//	    Prob: 0.20, AtMin: 60, AtMax: 240, Dur: 20, Cooldown: 120, MaxRuns: 2,
//	    Eligible: func(g *Game, t partyTraits) bool { return t.dance },
//	    Start:    func(g *Game, a *partyAct) bool { ...; return true },  // false = no way now (retried every few seconds)
//	    Update:   func(g *Game, a *partyAct, dt float64) bool { ...; return done },
//	    Items:    func(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem { return append(items, drawItem{y, fn}) } }) }
//
// Rules of the frame: use partyDice() (own dice) and NEVER g.rng for the dice; keep the state in a.D (a.D = &myState{});
// leave the guests as you found them when Update returns true (see toastEnd); texts in ES + English in your own
// lang_*.go; new ids appear in `-pevents` and can be forced with `-pevent id`.
//
// WHAT THE FRAME DOES
//   - partyReset (loadLevel, after the blackout and the daily challenge are planned): rolls the dice of every
//     eligible event (partyPlan), at most partyMaxPlanned (3) per party, with the time of each one in its window.
//     Minor events (the fist fight) come on top of those three, are not remembered and may come back often.
//   - updateParty (updatePlay): starts the ones whose time has come (if the group is free, at most ONE Big event, at
//     most partyMaxActive running), updates the running ones, and re-plans the ones with Cooldown (MaxRuns).
//   - partyItems / partyOver (draw.go): the Items go into the y-sorted list next to lifeItems; Over is drawn after
//     the blackout darkness (emergency lights).
//   - memory of the session: the events of the last 2 parties are not rolled again (partyHist).
//   - NEVER in STEALTH or on the Jurassic island. RETO DE HOY: none, except events with DailyOK (their own effect
//     does not touch the modifiers: time limit, sight, sure blackout...). Reactive events follow a condition
//     (emergency lights = the blackout), they do not roll dice and do not count for the maximum of three.
//   - partylife.go (selfies, dance contest, pizza boy) keeps its own dice and frequencies: here they are only
//     ADAPTERS (External): they are listed, and the frame knows whether the contest (a Big group, group "dance") is
//     running so that nothing of ours starts on top of it. They are not rolled or scheduled here.
// ---------------------------------------------------------------------------

const (
	partyMaxPlanned = 3 // programmed events per party
	partyMaxActive  = 2 // running at the same time
	partyRetryGap   = 4.0
	partyRetryMax   = 8
	partyMemory     = 2 // parties remembered
)

// partyTraits: what a level has, to decide which events make sense in it.
type partyTraits struct {
	idx                                 int
	icon                                string
	indoor, outdoor                     bool // outdoor = known outdoor venue; both false = unknown (random mode)
	dance, bar, toilet, stage, dj, pool bool
	bbq, roam                           bool
	cook                                bool // a stove, a barbecue or a pizza oven (pe_cookfire.go)
	guests                              int
}

// Campaign venues that are INDOORS (the rest of the icons are outdoors).
var partyIndoorIcons = map[string]bool{"house": true, "neon": true, "disco": true, "club": true, "cinema": true,
	"casino": true, "station": true, "trainstation": true, "stadium": true, "bull": true}

func partyTraitsOf(d *LevelDef, idx int) partyTraits {
	t := partyTraits{idx: idx}
	if d == nil {
		return t
	}
	t.icon, t.guests = d.Icon, d.Guests
	if t.icon != "" {
		t.indoor = partyIndoorIcons[t.icon]
		t.outdoor = !t.indoor
	}
	t.dance, t.pool, t.roam = len(d.Dance) > 0, len(d.Pools) > 0, len(d.Roam) > 0
	for _, p := range d.Props {
		switch p.Kind {
		case "bar", "minibar":
			t.bar = true
		case "toilet":
			t.toilet = true
		case "stage":
			t.stage = true
		case "dj", "booth":
			t.dj = true
		case "bbq":
			t.bbq, t.cook = true, true
		case "stove", "pizzaoven":
			t.cook = true
		}
	}
	return t
}

type partyEvent struct {
	ID, Name string
	Group    string // mutual exclusion: two events of the same group never run together ("dance", "speech", "light"...)
	Big      bool   // big group of guests: only one Big event at a time
	Reactive bool   // follows a condition (Eligible = "it holds now"; Update must not finish while it holds), no dice
	DailyOK  bool   // may run in the RETO DE HOY
	// Minor: a small, frequent thing (a fist fight): it does not count for the three planned events of a party nor for the
	// two running at once, it is not kept out by the memory of the last parties, and its chance may go up to 95 %.
	// It is never Big. (Feedback: "cada x tiempo tiene que haber alguna pelea".)
	Minor    bool
	Prob     float64
	AtMin    float64 // window of the start, seconds of the party
	AtMax    float64
	Dur      float64 // seconds it should last (the frame ends it at 2*Dur at most)
	Cooldown float64 // > 0: it may come again this many seconds after it ends (MaxRuns total, default 1)
	MaxRuns  int
	Prio     int // higher wins when more than partyMaxPlanned want to happen

	// External: managed by another system (partylife.go). Listed, never scheduled; reports whether this party has it.
	External func(g *Game) bool
	Eligible func(g *Game, t partyTraits) bool
	Start    func(g *Game, a *partyAct) bool
	Update   func(g *Game, a *partyAct, dt float64) bool // true = finished
	Items    func(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem
	Over     func(g *Game, dst *ebiten.Image, a *partyAct)
}

var partyRegistry []*partyEvent

func registerPartyEvent(e *partyEvent) { partyRegistry = append(partyRegistry, e) }

// partyEvents: the registry in a fixed order (by id): the same seed always gives the same parties.
func partyEvents() []*partyEvent {
	out := append([]*partyEvent(nil), partyRegistry...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func partyByID(id string) *partyEvent {
	id, _, _ = strings.Cut(id, ":") // -pevent id:stage (a capture stage, pe_treats.go pevDemo)
	for _, e := range partyRegistry {
		if e.ID == id {
			return e
		}
	}
	return nil
}

// partyAct: one running event.
type partyAct struct {
	ev    *partyEvent
	t0    float64
	runs  int
	D     any // the event's own state
	tries int
}

type partyPlanned struct {
	ev    *partyEvent
	at    float64
	runs  int
	tries int
}

type pevState struct {
	ready   bool
	plan    []*partyPlanned
	act     []*partyAct
	history []string // ids rolled for this party (for the memory)
	traits  partyTraits
	ran     map[string]bool // ids that started in this party (the newspaper tells about them; newspaper_events.go)
}

var (
	partyRng  *rand.Rand
	partySeed int64      // tests: non-zero = deterministic dice
	partyHist [][]string // ids of the last partyMemory parties (this session only)
	// dbgPartyEvent: -pevent id forces that event (3 s into the party) in the level given, ignoring chance and level.
	dbgPartyEvent string
)

// partyDice: the OWN dice of the frame, seeded per session (g.rng is never moved).
func partyDice() *rand.Rand {
	if partyRng == nil {
		sd := partySeed
		if sd == 0 {
			sd = time.Now().UnixNano() ^ 0x5eed7a11
		}
		partyRng = rand.New(rand.NewSource(sd))
	}
	return partyRng
}

func partyRecent(hist [][]string, id string) bool {
	for _, h := range hist {
		for _, x := range h {
			if x == id {
				return true
			}
		}
	}
	return false
}

// partyPlan rolls the dice for one party. Pure with respect to the game (only reads g), so it can be tested by the
// thousand. Reactive and External events are not rolled.
func partyPlan(r *rand.Rand, g *Game, t partyTraits, hist [][]string, fast bool) []*partyPlanned {
	var cand, minor []*partyPlanned
	for _, e := range partyEvents() {
		if e.Reactive || e.External != nil || e.Eligible == nil || !e.Eligible(g, t) {
			continue
		}
		roll := r.Float64() // always consumed: the plan of one event does not shift the dice of the next
		at := e.AtMin + r.Float64()*(e.AtMax-e.AtMin)
		if roll >= e.Prob || !e.Minor && partyRecent(hist, e.ID) {
			continue
		}
		if fast {
			at = math.Max(4, at*0.25)
		}
		if e.Minor {
			minor = append(minor, &partyPlanned{ev: e, at: at})
			continue
		}
		cand = append(cand, &partyPlanned{ev: e, at: at})
	}
	sort.SliceStable(cand, func(i, j int) bool { return cand[i].ev.Prio > cand[j].ev.Prio })
	if len(cand) > partyMaxPlanned {
		cand = cand[:partyMaxPlanned]
	}
	cand = append(cand, minor...) // the small ones come on top of the three
	sort.SliceStable(cand, func(i, j int) bool { return cand[i].at < cand[j].at })
	// events of the same group or Big ones must not overlap in time: the later one waits for the earlier
	for i := range cand {
		for j := 0; j < i; j++ {
			a, b := cand[j], cand[i]
			if (a.ev.Group != "" && a.ev.Group == b.ev.Group || a.ev.Big && b.ev.Big) && b.at < a.at+a.ev.Dur*2+6 {
				b.at = a.at + a.ev.Dur*2 + 6
			}
		}
	}
	return cand
}

// partyBlocked: why this party has no events ("" = it can).
func partyBlocked(g *Game) string {
	switch {
	case g.lvl == nil:
		return "no level"
	case g.stealth:
		return "stealth"
	case g.levelIdx == specialJurassic:
		return "jurassic"
	}
	return ""
}

// partyReset: new party. Call after blackoutLevelReset and dailyApplyEvents (emergency lights read g.blk).
func (g *Game) partyReset() {
	g.pev = pevState{ready: true}
	if partyBlocked(g) != "" {
		return
	}
	t := partyTraitsOf(g.lvl.Def, g.levelIdx)
	g.pev.traits = t
	var plan []*partyPlanned
	if dbgPartyEvent != "" {
		if e := partyByID(dbgPartyEvent); e != nil && e.External == nil && !e.Reactive {
			plan = []*partyPlanned{{ev: e, at: 3}}
		}
	} else if g.dc == nil { // RETO DE HOY: only DailyOK events (below), planned by the same dice
		plan = partyPlan(partyDice(), g, t, partyHist, fastEv)
	} else {
		for _, p := range partyPlan(partyDice(), g, t, partyHist, fastEv) {
			if p.ev.DailyOK {
				plan = append(plan, p)
			}
		}
	}
	g.pev.plan = plan
	if dbgPartyEvent == "" {
		for _, p := range plan {
			if p.ev.Minor {
				continue // the small ones are not remembered (they may come every party)
			}
			g.pev.history = append(g.pev.history, p.ev.ID)
		}
		partyHist = append(partyHist, g.pev.history)
		if len(partyHist) > partyMemory {
			partyHist = partyHist[len(partyHist)-partyMemory:]
		}
	}
}

func (g *Game) partyActive() bool { return g.mode == ModePlay && g.lvl != nil && partyBlocked(g) == "" }

func (g *Game) partyRunning(id string) *partyAct {
	for _, a := range g.pev.act {
		if a.ev.ID == id {
			return a
		}
	}
	return nil
}

// partyGroupBusy: something of this group runs (including the dance contest of partylife.go for "dance").
func (g *Game) partyGroupBusy(e *partyEvent) bool {
	for _, a := range g.pev.act {
		if e.Group != "" && a.ev.Group == e.Group {
			return true
		}
	}
	if e.Group == "dance" && g.life.contest != nil {
		return true
	}
	return false
}

// partyMajorRunning: running events that are not Minor (the small ones never take a place of the two).
func (g *Game) partyMajorRunning() int {
	n := 0
	for _, a := range g.pev.act {
		if !a.ev.Minor {
			n++
		}
	}
	return n
}

// partyBigBusy: a Big event runs (or the dance contest, which is one).
func (g *Game) partyBigBusy() bool {
	for _, a := range g.pev.act {
		if a.ev.Big {
			return true
		}
	}
	return g.life.contest != nil
}

func (g *Game) updateParty(dt float64) {
	if !g.partyActive() {
		return
	}
	if !g.pev.ready {
		g.partyReset()
	}
	p := &g.pev
	// running events
	for i := 0; i < len(p.act); i++ {
		a := p.act[i]
		done := a.ev.Update == nil || a.ev.Update(g, a, dt) || g.time-a.t0 > a.ev.Dur*2+1
		if !done {
			continue
		}
		p.act = append(p.act[:i], p.act[i+1:]...)
		i--
		mr := a.ev.MaxRuns
		if mr < 1 {
			mr = 1
		}
		if a.ev.Cooldown > 0 && a.runs < mr && dbgPartyEvent == "" && !a.ev.Reactive {
			p.plan = append(p.plan, &partyPlanned{ev: a.ev, runs: a.runs, at: g.time + a.ev.Cooldown*(0.8+0.4*partyDice().Float64())})
		}
	}
	// reactive ones: they follow a condition
	for _, e := range partyEvents() {
		if !e.Reactive || e.Eligible == nil || g.partyRunning(e.ID) != nil || g.dc != nil && !e.DailyOK {
			continue
		}
		if e.Eligible(g, p.traits) && e.Start != nil { // Reactive: Eligible = "the condition holds now"
			a := &partyAct{ev: e, t0: g.time, runs: 1}
			if e.Start(g, a) {
				p.act = append(p.act, a)
				g.partyNoteRan(e)
			}
		}
	}
	// the ones whose time has come
	for i := 0; i < len(p.plan); i++ {
		pl := p.plan[i]
		if g.time < pl.at {
			continue
		}
		e := pl.ev
		if !e.Minor && g.partyMajorRunning() >= partyMaxActive || g.partyGroupBusy(e) || e.Big && g.partyBigBusy() {
			pl.at += partyRetryGap
			if pl.at > e.AtMax+e.Dur*4+120 { // it never found room: dropped
				p.plan = append(p.plan[:i], p.plan[i+1:]...)
				i--
			}
			continue
		}
		a := &partyAct{ev: e, t0: g.time, runs: pl.runs + 1}
		if e.Start != nil && e.Start(g, a) {
			p.act = append(p.act, a)
			g.partyNoteRan(e)
			p.plan = append(p.plan[:i], p.plan[i+1:]...)
			i--
			continue
		}
		pl.tries++
		pl.at += partyRetryGap
		if pl.tries >= partyRetryMax {
			p.plan = append(p.plan[:i], p.plan[i+1:]...)
			i--
		}
	}
}

// partyItems: the y-sorted items of the running events (next to lifeItems in draw.go).
func (g *Game) partyItems(dst *ebiten.Image, items []drawItem) []drawItem {
	if !g.partyActive() {
		return items
	}
	for _, a := range g.pev.act {
		if a.ev.Items != nil {
			items = a.ev.Items(g, dst, a, items)
		}
	}
	return items
}

// partyOver: what is drawn over the world (after the blackout darkness, before the HUD).
func (g *Game) partyOver(dst *ebiten.Image) {
	if !g.partyActive() {
		return
	}
	for _, a := range g.pev.act {
		if a.ev.Over != nil {
			a.ev.Over(g, dst, a)
		}
	}
}

// partyList: text of `-pevents` (id, name, group, chance, window...).
func partyList() string {
	var b strings.Builder
	b.WriteString("id\tname\tgroup\tchance\twindow(s)\tdur\tcooldown\tnotes\n")
	for _, e := range partyEvents() {
		notes := ""
		if e.External != nil {
			notes += "partylife.go (own dice) "
		}
		if e.Reactive {
			notes += "reactive "
		}
		if e.Big {
			notes += "big "
		}
		if e.DailyOK {
			notes += "daily-ok "
		}
		if e.Minor {
			notes += "minor "
		}
		fmt.Fprintf(&b, "%s\t%s\t%s\t%.0f%%\t%.0f-%.0f\t%.0f\t%.0f\t%s\n", e.ID, e.Name, e.Group, e.Prob*100, e.AtMin, e.AtMax, e.Dur, e.Cooldown, strings.TrimSpace(notes))
	}
	return b.String()
}
