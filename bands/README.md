# bands/ — songs played by the bands

Some parties have a band playing on a stage. If you drop **MP3** files here whose name **starts with the band name**,
that band plays your songs while it is on stage (only `.mp3` is read). Without a file the built-in music is used.

Letters are compared ignoring case, spaces and symbols, so all of these work for Sum 41:
`sum41_song.mp3`, `Sum 41 - Song.mp3`, `SUM41-Fat Lip.mp3`. A leading "The" may be omitted (`offspring_x.mp3`).
Several files per band are fine: one is picked at random each time.

Names the game knows:

| Band | File name starts with |
|------|-----------------------|
| The Beatles | `thebeatles` or `beatles` |
| Sum 41 | `sum41` |
| Blink-182 | `blink182` |
| Simple Plan | `simpleplan` |
| The Offspring | `theoffspring` or `offspring` |
| Metallica | `metallica` |
| Punk Sailor | `punksailor` |
| Billy Talent | `billytalent` |
| Lost Prophets | `lostprophets` |
| Despistaos | `despistaos` |
| Elvis | `elvis` |

Examples: `beatles_hey_jude.mp3`, `Metallica - Enter Sandman.mp3`, `blink-182 - All The Small Things.mp3`.

The songs are not included in the repository (they may be copyrighted); add your own.
