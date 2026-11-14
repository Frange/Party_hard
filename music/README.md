# music/ — main song of each party

Drop **MP3** files here (only `.mp3` is read; other formats are ignored).

The file name must **start with the party number**, followed by `_`, `-` or a space and anything else:

```
1_house_party.mp3
2 - rooftop.mp3
7_cruise.mp3
```

- `1` is the first party in the campaign list, `2` the second, and so on (the number shown in the level menu).
- If two files have the same number, the first one in alphabetical order is used.
- A party without a file uses the built-in synthesised music.

The songs are not included in the repository (they may be copyrighted); add your own.

## Which style does each party have?

Every party has its own genre. When there is no mp3 for a party the game plays its own synthesised music in that style
(a short title at the start of the party tells you which). Use this table to choose which song goes where:

| # | Party | Kind of party | Genre |
|---|-------|---------------|-------|
| 1 | House party | Living-room kegger | House |
| 2 | Miami Beach | Lawless beach shack | Reggaeton |
| 3 | Glow Neon | Neon rave | Synthwave |
| 4 | Barcelona | Rumba in the square | Reggaeton (Latin, for the rumba) |
| 5 | Bullring | Cape and capers bash | Pasodoble / flamenco |
| 6 | Paris (was Eiffel Tower) | Black-tie waltz dinner | Classical (waltz) |
| 7 | Alcalá de Henares | Town fiesta with storks | Pasodoble / flamenco |
| 8 | Night cruise | The captain's bash | Disco |
| 9 | Funfair | Fairground, no seatbelt | Chiptune |
| 10 | Stadium | Cup final blowout | Pop |
| 11 | Disco club | Sunrise session | Techno |
| 12 | Festival | Mud festival | Electronic |
| 13 | Skate park | Bowl jam | Punk |
| 14 | Cemetery | Wake with riffs | Metal |
| 15 | Cinema | Late-night screening | Orchestral |
| 16 | Guadalajara | Street party with honey and bandstand | Pasodoble / flamenco |
| 17 | Circuit | Brakeless race | Electronic |
| 18 | Ski resort | Après-ski with cocoa | Pop |
| 19 | Rooftop | Rooftop pool bash | Hip-hop (lo-fi) |
| 20 | Casino | Black-tie night | Jazz (swing) |
| 21 | Motorway | Traffic jam with speakers | Rock |
| 22 | Italy | Pasta, amore and chaos | Classical (opera) |
| 23 | Train station | Ticketless party | Rap (boom bap) |
| 24 | Birthday party | Candle-blowing mayhem | Pop |
| 25 | Hockey rink | Overtime without a referee | Rock |
| 26 | 70s disco | Bell-bottom fever | Disco |
| 27 | Pirate ship | Boarding party with rum | Epic |
| 28 | Space station | Zero-gravity rave | Electronic |
| – | Isla Nublada (secret) | Last Cretaceous party | Epic orchestral (wonder and danger) |

Italy (now 22) was added right after the Motorway, Alcalá de Henares (7, v1.4.0) right after the Eiffel Tower (now Paris) and
Guadalajara (16, v1.4.0, the 14th party of the list) right after the Cemetery: files numbered 1 to 6 keep their party; a
file you had numbered 7 to 14 belongs one number later now (+1), and one numbered 15 or more two numbers later (+2) if
it was named before Alcalá; if you had already renumbered it for Alcalá, only the files numbered 16 or more move (+1). A file of
another style is ignored anyway (see below). The number is the place of the party in the campaign, counting the two parties
that are no longer played (Disco club 11, Cinema 15).

The level select shows "kind of party · genre" under the name of every party, and the title card of the party shows the
kind and the music (`Música: ...`) for a few seconds.

## Name your files after the style (so they cannot clash)

If the **last word** of a file name is a genre (`13_cementerio_metal.mp3`, `7_cruise_disco.mp3`, `extra_03_house_2.mp3`;
a trailing version number is ignored) the game checks it against the genre of the party:

- the same genre: the file plays and the title card says so;
- another genre (old batches such as `13_cine_electro.mp3` playing in the Cemetery): the file is **ignored** with a
  line in the console and the music of the game plays instead;
- no genre in the name (`5_fiesta.mp3`): it always plays, and the card says nothing about the style.

Genre words: house, electro/electronica, techno, disco, reggaeton, punk, pop, metal, rap, hiphop/hip-hop/lofi,
clasica/classical, jazz, country, flamenco/pasodoble, rock, synthwave, orquestal/orchestral, epico/epica/epic (the pirate ship), isla (the secret island), chiptune.

`more/*.mp3` (the songs that follow the main one) are filtered the same way: only the files of the style of the party
play; if none fits, the synthesised music of the party's genre does. The files in `DJ/` and `bands/` are not filtered,
except the old `DJ/punk_1.mp3 ... punk_6.mp3` of earlier versions: they are no longer used or shipped (an update
patch lists them in REMOVIDOS.txt) and, if a copy is still there, it is skipped.
`./PartyHardy -exportmusic music -musicbatch 3` writes files already named with their style.

The random-mode venues use Rock (pub), Pasodoble (European square) and Pop (mall). When the DJ dies one of six songs written
for it takes over (fast funk, big beat, ska, salsa, disco funk, surf rock: `songs_afterdj.go`, about 60-66 s each, with
sections, a tune and drums that change; never the same song twice in a row; an mp3 of yours in `DJ/` wins);
a band of its own style (Metalica = metal, The Beetles = pop, Elviz = country, Lost Profets and Despistaus = rock) plays
its genre, and the punk bands play the style of the party.
