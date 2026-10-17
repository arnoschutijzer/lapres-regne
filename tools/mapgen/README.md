# mapgen

Generates `maps/lumbridge.map` one to one from Old School RuneScape's own map
data, read from the game cache in the [OpenRS2 archive](https://archive.openrs2.org).

```sh
python3 tools/mapgen/generate.py
```

The first run downloads the parts of the cache it needs into
`tools/mapgen/.cache`; later runs work offline. It needs Python 3.10 or newer and
nothing beyond the standard library.

The map's ground, water, bridges, trees, graves, walls, fences, doorways,
heights, roofs and furniture all come from the game. Only the names of the
map's areas are kept by hand: the tool reads the `area` settings and the
`areas` grid from the map it replaces (`--areas` picks another file). `--out`
writes somewhere else, and `--cache` keeps the download elsewhere.

## How the game's data becomes the map

- **Cache.** Cache 2499 (March 2026, build 236), the newest that still names its
  map groups. Map squares 49 to 51 east and north cover Lumbridge, x 3150 to
  3283 and y 3184 to 3291. `cache.py` downloads, decrypts and unpacks groups;
  `decode.py` reads terrain, placed objects and object definitions.
- **Ground.** Floor overlays give water (the river and the swamp's pools),
  stone, boards and earth; the sand and bare-earth underlays give earth. Tiles
  the game raises a bridge's deck over become bridge.
- **Walls.** Wall objects stand along tile edges (rotation 0 to 3: west, north,
  east, south), corners along two, and diagonal walls across a tile (even
  rotations rising, odd falling). Most are unnamed, so `generate.py` lists the
  castle's stone walls and the wooden fences by id. A piece whose model is
  empty is an invisible barrier, as along the riverbanks and round the bushes by
  the castle's door, and is left out. Doors are told by their names, or by
  sharing a named door's model, as the castle's own large doors do. A door that
  stands open in the game hangs on the edge square to its doorway; it moves into
  the gap in the wall at its hinge, each leaf of a double door into its half. A
  tile the game leaves unfloored inside a room, as under the castle kitchen's
  trapdoor, takes the floor round it.
- **Roofs.** Roof pieces share their models and differ in texture: red clay
  tiles, thatch or slate. A room under none stays open to the sky; a roofed room
  open on a few sides gets doorways in its archways.
- **Heights.** The game's level-0 heights scaled into levels 0 to 9, with each
  body of water, room and bridge deck lying level.
- **Steps.** The flights of stone steps up to the keep's doors are unnamed
  objects laid flat on the ground (`STEPS` lists them by id). Their tiles become
  steps and keep the game's heights, as do the landing at the top and the
  ground at the foot, so the way climbs from the courtyard to the door.
- **Objects.** Furniture and landmarks fill their footprints, turned with their
  rotation.
- **Floors.** The game's levels 1 to 3 become the map's floors above the
  ground (`terrain 1`, `walls 1`, `objects 1` and so on), floored in stone and
  boards where the game floors them and open to the air elsewhere; a bridge's
  deck stays on the ground. Their walls and furniture stand as on the ground.
  The game leaves the floor out under a staircase or ladder above the ground,
  so those tiles take the floor beside them. A staircase or ladder leads up
  where the game stands its top on the same tiles a level higher.
  Furniture and wall decorations upstairs are modelled too, each floor's
  after a `floor` line in the models file.
  The game also places walls on its upper levels where they have no floor,
  under roofs and over the bridge; a wall with open air on both sides is left
  out, with any decoration hung on it.
- **Models.** Every piece of furniture and every landmark is drawn as the game
  models it, written to `maps/lumbridge.models`; one the game stands
  diagonally is turned a further eighth of a turn. `models.py` ports RuneLite's object and model
  loaders; each object's models are recoloured, retextured, mirrored, turned,
  scaled and offset as its definition says, and a textured face takes its
  texture's average colour. Heights are scaled by 0.6, as the game's people
  stand about a tile and a half tall and the hero under one; footprints stay
  exact. The renderer has inks for a limited number of colours, so the closest
  of the models' colours are merged until 160 remain. `MODELLED` in
  `generate.py` lists the fixtures drawn this way.
- **Wall decorations.** Banners, tapestries, paintings, torches and the like
  hang on the walls as the game models them (`hang` lines). The game models
  each with its back on a tile's west edge and turns it to its wall; one on a
  diagonal wall turns a further eighth of a turn, its inside face half a turn
  round from the outside. The map's walls stand thicker than the game's, so
  every decoration is set 24 of the game's units clear of its wall.
