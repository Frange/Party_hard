# Contenido del juego en carpetas (assets)

El juego lee todo de la carpeta `assets`, que está **junto al ejecutable** (en Mac, junto a `PartyHardy.app`).
La primera vez que arranca la crea con el contenido por defecto. Si borras o cambias algo, el juego lo usa
en el siguiente arranque, sin tocar código. Con `-noassets` ignora la carpeta y usa el contenido interno.

```
assets/
  sprites/
    main_character/<nombre>/   el protagonista (se usa el primero por orden alfabético)
    people/<nombre>/           invitados aleatorios
    special_people/<nombre>/   famosos y personajes (salen con más rareza, un 40 %)
    enemies/<nombre>/          guardias de seguridad
    special_enemies/<nombre>/  guardias especiales (ven más lejos y corren más)
    police/<nombre>/           policías (role "cop") y sanitarios (role "medic")
    objetos/<nombre>/          muebles, vehículos, animales, trampas...
  backgrounds/<nombre>/        escenarios
  text/
    news.json                  avisos y noticias de pantalla (opcional, ver abajo)
    people.json                frases de los invitados normales (opcional, ver abajo)
    special_people.json        frases de personajes famosos/temáticos (opcional)
    police.json                frases de policía/militares/SWAT (opcional)
    fbi.json                   frases del FBI (opcional)
    power_rangers.json         frases de los Pawer Rangers (opcional)
    killers.json               frases del asesino especial (opcional)
```

Si una categoría está vacía el juego usa el generador interno para esa categoría.

## Personajes (`sprites/<categoría>/<nombre>/`)

| fichero | contenido |
|---|---|
| `meta.json` | `name`, `category`, `role` (`cop`, `medic`, `guard`), `carry` (objeto en la mano: knife, cup, mic, phone, baguette, tray, camera, bigcup, saber, beer, flag, pizza, sign), `display_name` (opcional: cómo se llama en los mensajes de pantalla, p.ej. `"Zonic"`; si se omite se usa el nombre de la carpeta con mayúsculas, p.ej. `special_people/tio_paco` → "Tio Paco"). Solo los personajes de `special_people` salen con nombre propio; los de `people` siempre son "un invitado"/"una invitada" |
| `front.png`, `side.png`, `back.png` | el muñeco de pie, de frente, de perfil mirando a la **derecha**, y de espaldas (24x44 por defecto, cualquier tamaño). Proporciones base del juego: figura de unos 10 px de ancho y 33-35 de alto, cabeza de 5-6 px, tronco de 9-10 px (3-6 de ancho), brazos de 1-2 px de ancho y largos como hasta la cintura, y unos 15 px de cintura a los pies |
| `front_parts.png`, `side_parts.png`, `back_parts.png` | mapa de partes: gris = id × 32. 1 cabeza, 2 torso, 3 brazo izquierdo (el de la izquierda de la imagen), 4 brazo derecho, 5 pierna izquierda, 6 pierna derecha, 0 nada |
| `anim/<animación>_<vista>_<n>.png` | opcional: fotograma pintado a mano que sustituye al generado. Ejemplo: `walk_side_2.png` |

Las animaciones (andar, correr, bailar x4, beber, charlar, saludar, pegar, apuñalar, caer, vomitar, besar, DJ, batería,
guitarra, cantar, cargar, cansado, llamar, pánico, agacharse) se generan **solas**: cada parte se gira alrededor de
su articulación (hombro, cadera) o se desplaza (cabeza, torso). Nombres de animación y vista:
`idle walk run dance0..dance3 drink talk wave punch stab fall vomit kiss dj drum guitar sing carry tired call panic crouch` ·
`front side back`. En la vista de perfil, si el brazo/pierna lejanos están tapados, se copian de los cercanos.

Si no usas el editor puedes pintar los PNG con cualquier programa (Aseprite, Piskel, GIMP...): lo importante es que
el mapa de partes marque qué píxel es cada parte.

## Objetos (`sprites/objetos/<nombre>/`)

`frame0.png`, `frame1.png`... (a tamaño final de juego, sin escalar) y `meta.json`:

| campo | significado |
|---|---|
| `anchor_x`, `anchor_y` | punto del sprite donde "pisa" el objeto (se coloca en el escenario por este punto) |
| `foot_x`, `foot_y`, `foot_w`, `foot_h` | huella de colisión en píxeles del sprite |
| `solid` | bloquea el paso |
| `hide` | el jugador puede esconderse / tirar cuerpos |
| `phone` | teléfono que se puede cortar |
| `poison` | ponche o buffet envenenable (el fotograma 1 es el aspecto envenenado) |
| `trap` | `""`, `explode` o `electric` |
| `light` | farola: emite luz |
| `variants` | los fotogramas son variantes elegibles en el escenario (si no, son una animación) |
| `fps` | animación (fotogramas por segundo) |
| `sort_off` | desplazamiento del orden de dibujo (negativo = detrás de los personajes) |
| `use_kind`, `use_x`, `use_y` | puntos de uso relativos al ancla: `drink`, `toilet`, `lounge` |
| `ambient_speed` | si es > 0 cruza la calle de vez en cuando (sprite mirando a la izquierda) |

Objetos con nombre especial: `body_bag`, `heli`, `tank`, `ufo` sustituyen a esos elementos del juego.

## Escenarios (`backgrounds/<nombre>/`)

`background.png` (640x360, el dibujo del escenario) y `level.json`:

* `solid`: 90 filas de 160 caracteres, una celda = 4x4 píxeles. `.` libre, `#` muro (bloquea paso y visión),
  `=` valla baja (bloquea paso), `~` agua.
* `zone`: igual de grande. `.` nada, `r` donde pasean los invitados, `d` pista de baile.
* `props`: objetos colocados `{kind, x, y, var}` (píxeles, punto de ancla).
* `animals`: `{kind, x, y, radius}` con kind `horse`, `hen` o `dog` y la zona por la que se mueven.
* `start`, `car_from`, `car_to`, `pads` (helipuertos), `guard_routes` (una ruta por guardia), `performers` (DJ y músicos con `role`
  dj/drum/guitar/sing), `lights` (luces de colores).
* `police` (`car` o `heli`), `guests`, `swim`, `disco`, `dance_floor`, `dance_cell`, `tint` (RGBA de la oscuridad), `title`, `intro`.

## v6: enemigo especial

Un personaje en `sprites/special_enemies` con `"role": "killer"` en su `meta.json` es candidato a enemigo especial
(aparece como máximo uno por partida). El campo `carry` indica el arma: `chainsaw` (motosierra con chispas),
`machete`, `knife`, `claw`, `torch`... Los que no tienen rol `killer` siguen siendo guardias duros.
Objetos que se pueden llevar (`carry`): beer, tray, phone, camera, flag, mic, sign, pizza, balloons, megaphone,
briefcase, torch, broom, vacuum, selfie, plant, trophy, sax, machete, chainsaw, claw.

## v7: animales y escenarios

Animales disponibles en `animals` de los escenarios: horse, hen, dog, cat, chick, bull, elephant, giraffe, kangaroo.
Nuevos objetos: fountain, eiffel, carousel, ferris, bumpercar, coaster, tent, foodtruck, shelf, freezer, checkout, cart,
screen, popcorn, goal, bench, cafetable, statue, jukebox, pooltable, stands, ticketbooth, cottoncandy, balloons, hedge,
planter, obelisk, cage, truss, posters. Suelos nuevos: cobble, pitch, white, arena, mall.
# Modelos de referencia (generación única)

Puedes dejar PNGs de personajes en `assets/sprites/togenerate/` o, desde el
proyecto, en `sprites/togenerate/`. Al iniciar el juego, cada PNG que aún no
tenga su marca en `togenerate/generated/` se convierte en un personaje editable
en `assets/sprites/special_people/<nombre>/`: se centra el modelo, se generan
vistas iniciales y se etiquetan cabeza, torso, brazos y piernas. Después abre el
editor para pintar las vistas o animaciones a mano. El PNG fuente no se borra ni
se vuelve a importar mientras exista esa marca.

## Noticias y avisos (`text/news.json`)

Opcional. Un JSON con listas de frases por categoría, p.ej.:

```json
{
  "kill": ["{actor} ha matado a {victim}", "Han visto a {actor} matando a {victim}"],
  "fbi_arrive": ["¡El FBI se hace cargo del caso!"]
}
```

Cada categoría sustituye por completo a la del juego (no se mezclan frases); una categoría que no aparezca en el
fichero conserva las frases por defecto. Categorías actuales: `police_call`, `ambulance_call`, `backup_call`,
`fbi_arrive`, `military_arrive`, `swat_arrive`, `rangers_arrive`, `shoot_suspect` (`{actor}`, `{victim}`),
`suspect_taken` (`{victim}`), `killer_catch`, `kill` (`{actor}`, `{victim}`), `kill_player` (`{victim}`), `dj_dead`,
`vehicle_wreck_crew`, `vehicle_wreck_empty`. El juego evita repetir la misma frase dos veces seguidas dentro de la
misma categoría cuando hay alternativas.

## Diálogos (`text/people.json`, `special_people.json`, `police.json`, `fbi.json`, `power_rangers.json`, `killers.json`)

Las frases cortas que salen en el globo encima de la cabeza. Mismo formato que `news.json` pero un fichero por tipo
de personaje, y cada fichero solo necesita las situaciones que quiera cambiar: lo que no mencione conserva la frase
por defecto de esa categoría, y si una categoría no tiene nada para una situación, usa la de `people.json`. Por
ejemplo, para que los invitados normales reaccionen distinto al verte bailar:

```json
{
  "dance_join": ["¡Qué pasada!", "¡Bailas genial!"],
  "dance_annoyed": ["¡Serás plasta!"]
}
```

Situaciones de invitados (`people.json`/`special_people.json`): `dance_join`, `dance_leave`, `dance_annoyed`,
`greet`, `scared`, `saw_body`, `drink`, `dance_with`, `calling_police`, `accused`. Situaciones de policía/FBI/Power
Rangers (`police.json` también vale para militares y SWAT): `understood`, `halt`, `arrest`, `tired`. Situaciones del
asesino especial (`killers.json`): `pain`, `caught`.

