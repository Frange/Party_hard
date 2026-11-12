package main

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The scenes (cinematics) you can watch again: ESCENAS in the main menu.
// Each one stays locked, with a mysterious name and a funny hint, until it has
// been seen once for real (the "seen_*" counters of the progress), or while the
// DEBUG session is on, where everything is open.
// ---------------------------------------------------------------------------

// the order of the story, as you meet it in the game: this is the order and the numbering of the list,
// of the DEBUG menu and of the -scenes flag
const (
	sceneIntro   = iota // 1. the night it all began
	sceneInter1         // 2. after party 5 (by car)
	sceneInter2         // 3. after party 10 (by boat)
	sceneInter15        // 4. after party 15 (by hot air balloon): new, so the story order renumbers what follows
	sceneInter3         // 5. after party 20 (by rocket)
	sceneEnd1           // 6. "¿ FINAL ?"
	sceneInvite         // 7. the letter of the island
	sceneArrival        // 8. the helicopter ride to the island (after the letter blows up)
	sceneEnd2           // 9. the secret ending
	sceneCount
)

type sceneInfo struct {
	key             string // progress stat that says "seen"
	title, desc     string // once unlocked
	lockTitle, hint string // while locked
}

var sceneList = [sceneCount]sceneInfo{
	sceneIntro: {"seen_intro", "INTRO: NADIE TE DEJÓ DORMIR", "Las 3:47, el bajo del vecino, ovejitas con gafas de sol y un cuchillo. Así empezó todo.",
		"NO ME DEJAN DORMIR...", "Se ve al empezar de cero: borra tus datos."},
	sceneEnd1: {"seen_end1", "¿ FINAL ?", "Una alarma sin oxígeno, una cápsula de escape y un paseo a casa con todos los fantasmas de la noche. Corto y callado.",
		"¿ FINAL ?", "Todo tiene un final. Este, tras la última fiesta."},
	sceneEnd2: {"seen_end2", "FINAL SECRETO", "Raptores con sombrero, confeti, conga, fuegos artificiales y foto de familia. El final de quienes lo hicieron todo.",
		"FINAL: ???", "Dicen que lo ven quienes lo tienen todo... todo todo."},
	sceneInvite: {"seen_invite", "INVITACIÓN A ISLA NUBLADA", "Una carta brilla en el suelo de la estación, sin gravedad ni remitente. Huele a selva y a problemas con dientes.",
		"UNA CARTA MISTERIOSA", "Alguien te espera en una isla."},
	sceneArrival: {"seen_arrival", "LLEGADA A ISLA NUBLADA", "Una nave desde la estación, un helicóptero alquilado en la Tierra, un mareo y más fuego del que dice el contrato.",
		"¿VUELO SIN RETORNO?", "Tras la carta, alguien te lleva volando. Y no vuelve."},
	sceneInter1: {"seen_interlude5", "INTERLUDIO: OTRA FIESTA MÁS", "Cabreado y con humo en las orejas, repasa las fiestas ya vencidas y se va en coche a por la siguiente.",
		"¿QUIÉN CONDUCE?", "El viaje continúa... tras la fiesta número 5."},
	sceneInter2: {"seen_interlude10", "INTERLUDIO: A TODA VELA", "Cinco fiestas más a la espalda, mala cara y una lancha que no hace preguntas.",
		"¿PUEDO IR EN BARCO?", "Pasada la discoteca hay agua. Y mal humor."},
	sceneInter15: {"seen_interlude15", "INTERLUDIO: GLOBO A LA DERIVA", "Un globo aerostático con cesta, sacos de lastre y un pinchazo muy mal momento. Baja más rápido de lo que sube.",
		"¿ESO FLOTA?", "Más allá de la fiesta número 15 hay cielo. Y poco aire."},
	sceneInter3: {"seen_interlude20", "INTERLUDIO: CORREOS ESPACIALES", "Un cohete casero con váter, petardos y un guardia que pide el billete. Falla, claro, y despega igual.",
		"¿ESTO ES UN BARRIL?", "La última fiesta queda muy lejos... tras la fiesta número 20."},
}

// keepSeen copies the "seen" marks of an old progress into a fresh one (a new game keeps the cinematics seen).
func keepSeen(dst *Progress, src Progress) {
	for k, v := range src.Stats {
		if strings.HasPrefix(k, "seen_") || k == "jp_unlocked" {
			if dst.Stats == nil {
				dst.Stats = map[string]int{}
			}
			dst.Stats[k] = v
		}
	}
}

// seen: has this cinematic been watched for real?
func (g *Game) seen(key string) bool { return g.progress.Stats[key] > 0 }

// markSeen records it (and saves, unless the DEBUG session is freezing the progress).
func (g *Game) markSeen(key string) {
	if g.progress.Stats == nil {
		g.progress.Stats = map[string]int{}
	}
	if g.progress.Stats[key] > 0 {
		return
	}
	g.progress.Stats[key] = 1
	saveProgress(g.progress)
}

// sceneUnlocked: seen once, or everything is open in DEBUG. Somebody who has already played
// before the intro existed gets it too: there is no other way of seeing it without wiping the game.
func (g *Game) sceneUnlocked(i int) bool {
	if debugSnap != nil {
		return true
	}
	if g.seen(sceneList[i].key) {
		return true
	}
	return i == sceneIntro && (g.progress.Unlocked > 1 || g.progress.Stats["played"] > 0)
}

// sceneMission: the number of missions (campaign parties) done when the scene comes up; 0 = no counter
// (the intro).
func sceneMission(i int) int {
	switch i {
	case sceneInter1:
		return 5
	case sceneInter2:
		return 10
	case sceneInter15:
		return 15
	case sceneInter3:
		return len(campaignLevels) - 1 // the rocket leaves after the last party before the space station
	case sceneEnd1, sceneInvite, sceneArrival, sceneEnd2:
		return campaignN // they come after the last party (the island ones after the last story party and its trophies)
	}
	return 0
}

// sceneRowName: the left text of a row of the list, "N.- NAME" or "N.- ¿?" while locked (the name is not revealed).
func (g *Game) sceneRowName(i int) string {
	if !g.sceneUnlocked(i) {
		return fmt.Sprintf("%d.- ", i+1) + tr("¿?")
	}
	return fmt.Sprintf("%d.- ", i+1) + tr(sceneList[i].title)
}

// sceneRowCount: the "(x/21)" of a seen scene, or "".
func (g *Game) sceneRowCount(i int) string {
	if m := sceneMission(i); m > 0 && g.sceneUnlocked(i) {
		return fmt.Sprintf("(%d/%d)", m, campaignN)
	}
	return ""
}

// sceneName is what the list shows: the real title or the mysterious one.
func (g *Game) sceneName(i int) string {
	if g.sceneUnlocked(i) {
		return sceneList[i].title
	}
	return sceneList[i].lockTitle
}

func (g *Game) sceneText(i int) string {
	if g.sceneUnlocked(i) {
		return sceneList[i].desc
	}
	return sceneList[i].hint
}

// playScene starts a cinematic as a replay: when it ends (or is skipped) it comes back to `back`.
func (g *Game) playScene(i int, back Mode) {
	switch i {
	case sceneIntro:
		g.startIntro()
	case sceneInter1:
		g.startInterlude(interCar, 5)
	case sceneInter2:
		g.startInterlude(interBoat, 10)
	case sceneInter15:
		g.startInterlude(interBalloon, 15)
	case sceneInter3:
		g.startInterlude(interRocket, len(campaignLevels)-1)
	case sceneEnd1:
		g.startEndingPre(false)
	case sceneInvite:
		g.startInvitePre()
	case sceneArrival:
		g.startArrival()
	default:
		g.startEndingPre(true)
	}
	g.sceneReplay, g.sceneBack = true, back
}

// leaveScene returns from a replayed cinematic.
func (g *Game) leaveScene() {
	g.snd.StopThump()
	g.snd.StopScene()
	resetThumpBeat()
	g.intro, g.end, g.inter = nil, nil, nil
	g.sceneReplay = false
	g.bedQuiet, g.bedEmpty, g.bedNoTalk = false, false, false
	g.mode = g.sceneBack
	g.snd.StartMusic()
}

func (g *Game) updateScenes() {
	n := sceneCount
	click := g.mouseMenu(func(mx, my float64) int {
		for i := 0; i < n; i++ {
			if y := sceneRowY(i); mx >= 250 && mx <= 910 && my >= y-4 && my < y+22 {
				return i
			}
		}
		return -1
	}, &g.sceneSel)
	if actDown.just() {
		g.sceneSel = (g.sceneSel + 1) % n
	}
	if actUp.just() {
		g.sceneSel = (g.sceneSel + n - 1) % n
	}
	if actConfirm.just() || click {
		if g.sceneUnlocked(g.sceneSel) {
			g.playScene(g.sceneSel, ModeScenes)
		} else {
			g.snd.Play("thud")
		}
	}
	if actBack.just() {
		g.mode = ModeMenu
	}
}

func sceneRowY(i int) float64 { return 100 + float64(i)*26 } // nine rows fit above the card at y=340

func (g *Game) drawScenes(dst *ebiten.Image) {
	g.drawBedroom(dst)
	g.drawLogo(dst, logoSideCX, 30)
	fillRect(dst, 230, 60, 700, 440, rgba(0x05070e, 228))
	fillRect(dst, 230, 60, 700, 1, rgba(0x6a4ab0, 160))
	fillRect(dst, 230, 499, 700, 1, rgba(0x6a4ab0, 160))
	g.drawText(dst, "ESCENAS", 250, 72, rgb(0xff6a8a), 1.8)
	seen := 0
	for i := 0; i < sceneCount; i++ {
		if g.sceneUnlocked(i) {
			seen++
		}
	}
	cnt := fmt.Sprintf("%d / %d", seen, sceneCount)
	g.drawText(dst, cnt, 910-g.textW(cnt, 1.6), 72, rgb(0xf0e060), 1.6)
	for i := 0; i < sceneCount; i++ {
		y := sceneRowY(i)
		open := g.sceneUnlocked(i)
		col := rgb(0x3fb59a)
		if !open {
			col = rgb(0x8a86a0)
		}
		pre := "  "
		if i == g.sceneSel {
			col, pre = rgb(0xff6a8a), "> "
			if !open {
				col = rgb(0xffb070)
			}
			fillRect(dst, 244, y-4, 672, 26, rgba(0x2a1a4a, 160))
		}
		g.drawText(dst, pre+g.sceneRowName(i), 256, y+2, col, 1.5)
		if c := g.sceneRowCount(i); c != "" {
			g.drawText(dst, c, 836-g.textW(c, 1.5), y+2, col, 1.5)
		}
		tag, tc := "BLOQUEADA", rgb(0x6a6680)
		if open {
			tag, tc = "VISTA", rgb(0x70e090)
		}
		g.drawText(dst, tag, 904-g.textW(tag, 1.1), y+6, tc, 1.1)
	}
	// the card of the selected scene: what it is, or the hint of how to get it
	fillRect(dst, 250, 340, 660, 118, rgba(0x120a24, 230))
	fillRect(dst, 250, 340, 660, 1, rgba(0x6a4ab0, 200))
	open := g.sceneUnlocked(g.sceneSel)
	head, hc := "PISTA", rgb(0xffb070)
	if open {
		head, hc = sceneList[g.sceneSel].title, rgb(0xf0e060)
	}
	g.drawText(dst, head, 266, 352, hc, 1.4)
	g.drawWrapped(dst, g.sceneText(g.sceneSel), 266, 382, 628, rgb(0xe0d8f0))
}
