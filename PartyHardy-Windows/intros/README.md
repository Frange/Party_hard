# intros/ — songs of the story scenes

Drop **MP3** files here (only `.mp3` is read) to replace the built-in music of the cinematics.
The number at the start of the name says which scene it is for; the rest of the name is free:

| File name                 | Scene                                              |
|---------------------------|----------------------------------------------------|
| `1_intro.mp3`             | The first intro (the room, the party next door)    |
| `2_interludio_A.mp3`      | Interlude after party 5 (by car)                   |
| `3_interludio_B.mp3`      | Interlude after party 10 (by boat)                 |
| `4_final.mp3`             | The ending                                         |
| `5_final_secreto.mp3`     | The secret ending                                  |
| `6_invitacion.mp3`        | The invitation card to the island                  |
| `7_cohete.mp3`            | Interlude after party 20 (the home-made rocket)    |

* Several files with the same number are allowed (`2_interludio_A.mp3`, `2_interludio_A_2.mp3`...): one is picked at random each time.
* A song is looped while the scene lasts. Without a file the built-in synthesised music is used: the first intro
  has the bass drum ("pum pum") of the neighbours' party with a catchy party hook on top; the other scenes have cheerful
  tunes with no bass drum (a cartoon march for the interludes, a lullaby for the ending, a fanfare for the secret ending,
  an adventure theme for the invitation).
* **First intro and its beat**: the speakers, the shaking room and the clinking glasses follow the beat. With your own
  `1_intro.mp3` they follow **124 BPM** unless you say otherwise: create a text file `1_intro.bpm` with just the number
  (for example `128`). The first beat of the song should fall at the very start of the file.
* The mute key (`M`) and the music volume apply as usual.

The songs are not included in the repository (they may be copyrighted); add your own.
