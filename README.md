# Party Hardy (a fan-made party-game clone) with a built-in sprite and level editor

A fan-made homage to the original *Party Hard* (Pinokl Games / tinyBuild), written in Go with [Ebiten](https://ebitengine.org/). Not affiliated with them.
All art is procedural pixel art; everything else (characters, objects, animals, levels) can be replaced from the `assets/` folder.
Made by **Frange**. Credits and the full homage notice are in the in-game *Settings > Credits* screen.

- Windows: `PartyHardy.exe` (game) and `PartyHardyEditor.exe` (editor).
- macOS: `PartyHardy.app` and `PartyHardyEditor.app` (first time: right-click > Open).
- The `assets` folder must sit next to them (next to the apps on macOS). If you delete it, it is recreated with the defaults.
- Inside the game, **F2** opens the editor; "Back to game (F2)" in the editor returns.

## The game

You are a killer at a party. Kill everybody (guests, guards, DJ and band) without getting caught.

- **Move** WASD / arrows, **run** Shift, **stab** Space, **dance** F, **use** E (carry bodies, hide, traps, drinks, phones, items). All keys and pad buttons can be rebound in Settings; the mouse works in every menu.
- **Gamepad** (PlayStation-style by default): left stick / D-pad move, **R1** run, **X** stab / accept, **O** use / back, **Triangle** dance, **R2** immunity, **L1** event, **START** pause (O or START close it), **SELECT** help, **Square** minimap; in the wardrobe and GOALS, Square switches P1/P2 and opens a card.
- Everybody only sees in front of them: sneak up from behind. A stab in the back of the special killer cannot be answered at once.
- Witnesses phone the police from the **red booths** (at least two per party). Cutting a booth's cable (once each) stops that phone; a guest who sees it cut goes to the **yellow technician phone** and a yellow van brings technicians to repair the line.
- Police escalate with your criminality stars: police > FBI > military > SWAT > Pawer Rangers. They bag bodies, an ambulance comes for many bodies, and if you are hunted you will see handcuffs over your head (it means the police were called on you, not that you are arrested). Places without a road (cruise ship, space station) get boats or ships in each vehicle's colour.
- **Items**: poison bottles (to poison drink bowls, punch and buffets), disguise bags (an automatic outfit change that makes the police lose you), chairs, and more traps (explosive, electric, gas leak, fireworks, falling piano). Anything you can use is marked with a bobbing diamond and its name when you get close.
- **Points** for kills, animals, explosions and traps, hidden bodies, cut phone lines, killing cops with traps, and killing the special killer.
- **Trophies**: 50 trophies, each giving something (clothes, characters, events, powers). The last one, *Leyenda de la fiesta*, needs all the others and gives powers (immunity with golden halo and cop killing, summon famous guests, send a killer, rare events), turns the main menu golden, and unlocks golden items.
- **Campaign**: 26 parties, from the house party to the space station (always last, right after the pirate ship), including a six-lane motorway, a railway station, a casino, a ski resort and a graveyard, plus a secret island after the end. Every party gives a bronze, silver or gold medal. A free random mode and generated parties are also available.
- **Rare events**: UFO abductions, Souperman, Satan and God, zombies, T-Rex, ghosts, F1 drivers tearing through the party, escaped herds, skydivers, ball games, Halloween / Christmas / summer groups, taxis, monster trucks, rocket-football cars, trains, cheerleaders, snakes... Some trophies let you trigger rare events on demand.
- Famous guests have catchphrases, chat with each other in short scenes and do their own special moves.
- Hidden **DEBUG mode** (up, up, down, down, left, right, left, right in the main menu): toggles for immunity, cop killing, powers, unlocking everything and more; nothing is saved while it is on.

## Editor

Three tabs, each with big icon buttons and the same layout: what you have on the left, the drawing in the middle, the options on the right.

- **Characters**: choose the kind first (protagonist, person, special person, killer, special type — guard, medic, firefighter, phone technician — and police — police, FBI, military, SWAT, ranger), then pick one from the thumbnails. Draw it in **DRAWING** (pencil, eraser, picker and a palette) and tell the game which pixel is which body part in **PARTS** (with an automatic assignment button). Sex, held object and size are set on the right, with an animated preview.
- **Objects and animals**: choose *objects* or *animals* and the kind (furniture, decoration, nature, vehicles, traps, usables, lights). The options change with the kind: vehicle speed, trap effect, what a usable does (drink, toilet, sit, hide, phone, poisonable), light distance / strength / colour, animation.
  **PAINT** draws the picture; **LOGIC** places the anchor point, the use zone and the collision box.
- **Scenarios**: list of your scenarios with thumbnails, name, save, new (always from a generated scenario, never empty), template (copy a game scenario), generator, import a PNG/JPG background and test. Drawing options: Background, Walls, Objects, Points and Settings.
- **Roads**: in *Walls* paint a one-lane road and choose its direction (left, right, up, down); traffic drives along it.
- **Languages**: Spanish or English, in the game's Settings and in the editor's top bar (all texts, messages and funny names are translated).
- Import buttons open the system file dialog; you can also drag a PNG, JPG or PDF (first page; macOS and Linux) onto the window.
- **Import window**: a big picture (a photo of a friend, a drawing...) opens a window to crop what you want, remove the background, choose the pixel size, reduce the colours and see the result before sending it to the editor to be retouched. Several pictures (front, side, back) can be queued.
- **`imports/` folder**: whatever you save after importing a picture (characters, objects, animals, scenario backgrounds) goes to `imports/` (next to `assets/`), not to `assets/`. The game and the editor read both. `imports/` is in `.gitignore`, so your own pictures are never uploaded to the repository.
- **`intros/` folder**: optional MP3 songs for the story scenes (`1_intro.mp3`, `2_interludio_A.mp3`, `3_interludio_B.mp3`, `4_final.mp3`, `5_final_secreto.mp3`, `6_invitacion.mp3`, `7_cohete.mp3` for the rocket interlude after party 25). Without them the game synthesises its own: the first intro has the bass drum of the party next door in time with the speakers, the other scenes use cheerful drumless tunes (a cartoon march, a lullaby, a fanfare, an adventure theme). `1_intro.bpm` sets the beat of your own first intro (124 BPM by default). The folder is not uploaded to the repository (see `.gitignore`).
- The content format is documented in [`ASSETS.md`](ASSETS.md).

## Music genres

Each party has its own musical style, played by the built-in synthesiser when you have not put an mp3 for it in `music/`:
house, reggaeton, synthwave, country, pasodoble, classical waltz, disco, chiptune, pop, techno, electronic, punk, metal,
orchestral, hip-hop, jazz, rock, rap and, for the secret island, an original epic orchestral piece (a noble horn theme over strings and harp, a dark danger section with stomping timpani, and the theme returning). Every party also has a silly *kind of party*
("Living-room kegger · House", "Wake with riffs · Metal"...), shown under its name in the level select and on the title card.
Your own mp3s whose file name ends with a genre (`13_cementerio_metal.mp3`) only play in parties of that genre; files of another
style are ignored with a console warning. The full table and the naming rule are in [`music/README.md`](music/README.md).

## Guests

Each party starts with 45 to 60 guests (random; *Settings > minimum / maximum guests*). The maximum is a strict cap on the guests present at once, arrivals during the party included. Famous characters appear only once per party.

## Building

Requires Go. `./build.sh` builds the game and the editor for Windows and macOS into `dist/` and zips them (the editor is the same code built with `-tags editor`).

Debug flags: `-level N`, `-event N`, `-mute`, `-check`, `-selftest N`, `-shot f.png`, `-noassets`, `-editor`, `-immune`, `-fastevents`, `-with slug,slug`.

Tests: `go test .`

The version lives in `version.go` (`gameVersion`) and must match the first entry of [`CHANGELOG.md`](CHANGELOG.md) (`build.sh` refuses to build otherwise).
`build.sh` also writes `releases/manifest-<version>-<mac|win>.json`; after it, `./mkpatch.sh <base version>` (e.g. `./mkpatch.sh 1.0.0`) makes update patches with only the files that changed (`PartyHardy-Parche-<Mac|Windows>-<base>-a-<version>.zip`). A patch never carries the player's save (`assets/progress.json`) or settings; how to apply it is in `LEEME.txt` and in the `LEEME_ACTUALIZAR.txt` inside the patch.

### One or two players

Main menu > **PLAYERS** switches between 1 and 2 players (default 1, saved in `settings.json`). With two players each
one picks the device: both on the keyboard (P1 W A S D · Space · E · F · left Shift; P2 arrow keys · right Shift ·
`.` stab · `,` use · `/` dance), one on the keyboard and one on a gamepad, or both on gamepads. The default is automatic
(no pad: shared keyboard; one pad: P1 keyboard, P2 pad; two pads: one each) and pads are noticed when plugged in or
pulled out. Settings > PLAYERS and > PLAYER 2 CONTROLS rebind everything.
