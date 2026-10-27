# Contenido del juego en carpetas (assets)

El juego lee todo de la carpeta `assets`, que está **junto al ejecutable** (en Mac, junto a `PartyHard.app`).
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
```

Si una categoría está vacía el juego usa el generador interno para esa categoría.

## Personajes (`sprites/<categoría>/<nombre>/`)

| fichero | contenido |
|---|---|
| `meta.json` | `name`, `category`, `role` (`cop`, `medic`, `guard`), `carry` (objeto en la mano: knife, cup, mic, phone, baguette, tray, camera, bigcup, saber, beer, flag, pizza, sign) |
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
