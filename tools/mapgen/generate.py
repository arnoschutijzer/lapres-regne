"""Generate maps/lumbridge.map one to one from Old School RuneScape's own map data.

    python3 tools/mapgen/generate.py

Reads the game cache from the OpenRS2 archive (kept in tools/mapgen/.cache after the first run) and
writes the map: ground, water, bridges, trees, graves, walls along the edges between tiles,
heights, roofs and furniture, all where the game has them. Only the names of the map's areas are
kept by hand: they are read from the map file being replaced.
"""

import argparse
import collections
import os
import statistics
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import decode  # noqa: E402
import models  # noqa: E402
from cache import Cache  # noqa: E402

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(os.path.dirname(HERE))

# Lumbridge: world x 3150 to 3283 west to east, y 3291 to 3184 north to south.
X0, Y0, W, H = 3150, 3291, 134, 108
SQUARES = [(sx, sy) for sx in range(49, 52) for sy in range(49, 52)]
# Where every adventurer starts, in the castle courtyard.
START = (3222, 3218)

# Floor overlays and underlays, by id, as their colours and textures show.
WATER = {6, 7}  # the river's water, and the swamp's pools
STONE = {10, 2, 3, 102}  # flagstones and paving, the keep's checkered floor
BOARDS = {5}
EARTH = {14, 22, 89, 81, 83, 88, 26}  # earth tracks and the desert road
EARTHY_UNDERLAYS = {62, 63, 64}  # desert sand and bare earth

TREES = {"Tree": "T", "Oak tree": "O", "Willow tree": "W", "Yew tree": "Y", "Dead tree": "X"}

# Most wall pieces are unnamed. The castle's stone walls are battlemented where they face the open
# air, and these ids are wooden fences. Pieces whose models are empty are invisible barriers, such
# as those keeping players out of the river and out of the bushes by the castle's door: they are
# left out, so the river can be waded. Everything else is a stone wall.
CASTLE = {1911}
FENCES = {814, 979, 980, 981, 997, 1007, 1008, 1415, 1624, 1625, 1626, 1631, 4325, 5633,
          12983, 12990, 12991}

# Roof pieces share their models and differ by texture: red clay tiles, thatch or slate.
ROOFING = {}
for _id in (1924, 1790, 5662, 5663, 5647, 5648, 1927, 1642, 1606):
    ROOFING[_id] = "c"
for _id in (1926, 1792, 1929, 54134, *range(12992, 13001)):
    ROOFING[_id] = "t"
for _id in (1925, 1643, 1791, 5756, 5757):
    ROOFING[_id] = "s"

# The letters of the objects grid, by the names of the game's objects they stand for.
FIXTURES = {
    "t": ["Table", "Workbench"], "c": ["Chair", "Stool", "Barstool", "Rocking chair", "Bench"],
    "T": ["Throne"], "a": ["Suit of armour"], "r": ["Cooking range", "Range", "Fireplace", "Furnace", "Spit roast"],
    "o": ["Barrel", "Barrels", "Stacked barrels", "Vat"], "x": ["Crate", "Boxes", "Closed chest", "Open chest"],
    "q": ["Sacks", "Sack"], "s": ["Shelves", "Cooking shelves", "Bookcase", "Axe cabinet", "Grandfather Clock"],
    "S": ["Staircase"], "L": ["Ladder", "Stepladder"], "i": ["Standing torch"], "l": ["Candles"], "f": ["Standard"],
    "A": ["Altar"], "p": ["Church pew"], "O": ["Church organ"], "k": ["Coffin"], "F": ["Fountain"], "u": ["Statue"],
    "w": ["Well"], "h": ["Haystack", "Hay bales"], "n": ["Signpost", "Noticeboard"], "C": ["Cart", "Wheelbarrow"],
    "g": ["Trough", "Food trough", "Sink"], "y": ["Scarecrow"], "v": ["Potato", "Onion", "Cabbage"],
    "K": ["Counter"], "b": ["Bed"], "N": ["Rusted anvil"],
}
LETTER = {name: letter for letter, names in FIXTURES.items() for name in names}

# The fixtures drawn as the game models them, by their letters.
MODELLED = set(FIXTURES)
# How many colours the models may paint in, beside the renderer's own inks.
MODEL_COLOURS = 160
# Wall decorations are modelled with their backs on a tile's west edge; the map's walls stand
# thicker than the game's, so each decoration is set this many of the game's units clear of the
# wall it hangs on.
DECORATION_CLEARANCE = 24
# The game's people stand about one and a half tiles tall and the hero under one: model heights
# are scaled down so furniture keeps its proportion beside the hero.
HEIGHT_SCALE = 0.6


def col_row(x, y):
    return x - X0, Y0 - y


def inside(c, r):
    return 0 <= c < W and 0 <= r < H


def read_game(cache):
    """The terrain, the objects placed at level 0, and the definitions of those objects."""
    maps = cache.table(decode.MAPS)
    keys = cache.keys()
    terrain, placed = decode.Terrain(), []
    for sx, sy in SQUARES:
        terrain.read(cache.group(decode.MAPS, maps.find(f"m{sx}_{sy}")), sx, sy)
        name = f"l{sx}_{sy}"
        placed += decode.locations(cache.group(decode.MAPS, maps.find(name), keys[name]), sx, sy)
    definitions = cache.files(decode.CONFIGS, decode.OBJECTS)
    wanted = {p["id"] for p in placed}
    names = {i: decode.object_name(definitions[i]) for i in wanted}
    sizes = {i: decode.object_size(definitions[i]) for i in wanted}
    return terrain, placed, names, sizes, definitions


def read_areas(path):
    """The hand-named areas of the map being replaced: its area settings and its areas grid."""
    with open(path) as f:
        lines = f.read().split("\n")
    settings = [line for line in lines if line.startswith("area ")]
    grid, reading = [], False
    for line in lines:
        if line == "areas":
            reading = True
        elif reading and (not line or line.startswith("//")):
            break
        elif reading:
            grid.append(line)
    return settings, grid


class Lumbridge:
    def __init__(self, terrain, placed, names, sizes, definitions):
        self.terrain_data, self.names, self.sizes, self.definitions = terrain, names, sizes, definitions
        self.cache, self.model_points = None, {}
        # shaped lists the furniture drawn as the game models it: object, type, orientation and
        # its footprint's north-west tile and size; hung, the wall decorations and their tiles.
        self.shaped, self.hung = [], []
        self.placed = [p for p in placed if p["z"] == 0]
        self.roof_pieces = [p for p in placed if 12 <= p["type"] <= 21]
        self.ground = [["."] * W for _ in range(H)]
        self.west = [[" "] * (W + 1) for _ in range(H + 1)]
        self.north = [[" "] * (W + 1) for _ in range(H + 1)]
        self.rising = [[" "] * W for _ in range(H)]
        self.falling = [[" "] * W for _ in range(H)]
        self.objects = [["."] * W for _ in range(H)]
        self.roofs = [["."] * W for _ in range(H)]

    def key(self, c, r):
        return X0 + c, Y0 - r

    # --- Ground ------------------------------------------------------------------------------

    def lay_ground(self):
        t = self.terrain_data
        for r in range(H):
            for c in range(W):
                at = self.key(c, r)
                overlay = t.overlay.get(at, (0, 0, 0))[0]
                if t.settings.get((1, *at), 0) & 2:
                    self.ground[r][c] = "H"  # a bridge's deck, raised from the level above
                elif overlay in WATER:
                    self.ground[r][c] = "~"
                elif overlay in STONE:
                    self.ground[r][c] = "_"
                elif overlay in BOARDS:
                    self.ground[r][c] = "b"
                elif overlay in EARTH or t.underlay.get(at) in EARTHY_UNDERLAYS:
                    self.ground[r][c] = ":"
        for p in self.placed:
            c, r = col_row(p["x"], p["y"])
            name = self.names.get(p["id"])
            if not inside(c, r) or p["type"] not in (10, 11):
                continue
            if name in TREES and self.ground[r][c] in ".:":
                self.ground[r][c] = TREES[name]
            elif name == "Gravestone" and self.ground[r][c] in ".:_":
                self.ground[r][c] = "+"

    def indoor(self, c, r):
        return inside(c, r) and self.ground[r][c] in "_b"

    def crossed(self, c, r):
        return inside(c, r) and (self.rising[r][c] != " " or self.falling[r][c] != " ")

    # --- Walls -------------------------------------------------------------------------------

    def wall_models(self, oid, kind):
        definition = models.ObjectDefinition(self.definitions[oid])
        return tuple(definition.models_for(0 if kind == 2 else kind) or definition.models_for(0))

    def visible(self, oid, kind):
        """Whether a wall piece placed as a type has a model with any points to draw."""
        return any(self.points_in(model) for model in self.wall_models(oid, kind))

    def points_in(self, model):
        if model not in self.model_points:
            self.model_points[model] = len(models.Model(self.cache.group(models.MODELS, model)).x)
        return self.model_points[model]

    def kind_of(self, p):
        name = self.names.get(p["id"]) or ""
        if "oor" in name or "ate" in name or self.wall_models(p["id"], 0) in self.door_models:
            return "D"
        if p["id"] in CASTLE:
            return "C"
        return "=" if p["id"] in FENCES else "#"

    def stand_walls(self):
        def put(grid, c, r, kind):
            if 0 <= r < len(grid) and 0 <= c < len(grid[r]) and grid[r][c] != "D":
                grid[r][c] = kind

        # Doors are told by their names, or by sharing a named door's model, as the castle's own
        # large doors do.
        self.door_models = {self.wall_models(p["id"], 0) for p in self.placed if p["type"] == 0
                            and any(word in (self.names.get(p["id"]) or "") for word in ("oor", "ate"))}
        for p in self.placed:
            if p["type"] not in (0, 2, 9) or not self.visible(p["id"], p["type"]):
                continue
            c, r = col_row(p["x"], p["y"])
            kind = self.kind_of(p)
            if p["type"] == 9:
                if inside(c, r):
                    (self.rising if p["rot"] % 2 == 0 else self.falling)[r][c] = kind
                continue
            for rot in [p["rot"]] + ([(p["rot"] + 1) % 4] if p["type"] == 2 else []):
                if rot == 0:
                    put(self.west, c, r, kind)
                elif rot == 1:
                    put(self.north, c, r, kind)
                elif rot == 2:
                    put(self.west, c + 1, r, kind)
                else:
                    put(self.north, c, r + 1, kind)
        # The castle's walls are battlemented wherever they face the open air, plain within the keep.
        for r in range(H + 1):
            for c in range(W + 1):
                if self.west[r][c] == "C":
                    self.west[r][c] = "#" if self.indoor(c - 1, r) and self.indoor(c, r) else "B"
                if self.north[r][c] == "C":
                    self.north[r][c] = "#" if self.indoor(c, r - 1) and self.indoor(c, r) else "B"
        for grid in (self.rising, self.falling):
            for r in range(H):
                for c in range(W):
                    if grid[r][c] == "C":
                        grid[r][c] = "B"
        self.trim_partial_floors()
        self.floor_holes()
        self.open_doors()
        for r in range(H):
            for c in range(W):
                if self.crossed(c, r) and self.ground[r][c] in "TOWYX+":
                    self.ground[r][c] = "."

    def trim_partial_floors(self):
        """Beside a wall running diagonally across a tile, a floor the game lays over only part of
        a tile lies outside the wall: the ground there is open."""
        t = self.terrain_data
        for r in range(H):
            for c in range(W):
                shape = t.overlay.get(self.key(c, r), (0, 0, 0))[1]
                if self.ground[r][c] in "_b" and shape != 0 and not self.crossed(c, r):
                    if any(self.crossed(c + dc, r + dr) for dc in (-1, 0, 1) for dr in (-1, 0, 1)):
                        self.ground[r][c] = ":" if t.underlay.get(self.key(c, r)) in EARTHY_UNDERLAYS else "."

    def floor_holes(self):
        """A tile the game leaves unfloored in the middle of a room, such as under the castle
        kitchen's trapdoor, takes the floor round it."""
        for r in range(H):
            for c in range(W):
                if self.ground[r][c] not in ".:":
                    continue
                around = [self.ground[r + dr][c + dc] if inside(c + dc, r + dr) else "." for dc, dr in ((1, 0), (-1, 0), (0, 1), (0, -1))]
                if all(g in "_b" for g in around):
                    self.ground[r][c] = max(set(around), key=around.count)

    def edge(self, kind, c, r):
        grid = self.west if kind == "w" else self.north
        return grid[r][c] if 0 <= r < len(grid) and 0 <= c < len(grid[r]) else " "

    def open_doors(self):
        """A door standing open hangs from its hinge on the edge square to its doorway: set the
        doorway in the gap between walls that edge's end opens on."""
        doors = [("w", c, r) for r in range(H + 1) for c in range(W + 1) if self.west[r][c] == "D"]
        doors += [("n", c, r) for r in range(H + 1) for c in range(W + 1) if self.north[r][c] == "D"]
        for kind, c, r in doors:
            ahead = [("w", c, r - 1), ("w", c, r + 1)] if kind == "w" else [("n", c - 1, r), ("n", c + 1, r)]
            if all(self.edge(*e) in "#BC" for e in ahead):
                continue  # shut in its doorway already, in line with the walls
            ends = ((c, r), (c, r + 1)) if kind == "w" else ((c, r), (c + 1, r))
            for cx, cz in ends:
                moved = False
                for other, oc, orr, beyond, before in (
                    ("w", cx, cz, ("w", cx, cz + 1), ("w", cx, cz - 1)),
                    ("w", cx, cz - 1, ("w", cx, cz - 2), ("w", cx, cz)),
                    ("n", cx, cz, ("n", cx + 1, cz), ("n", cx - 1, cz)),
                    ("n", cx - 1, cz, ("n", cx - 2, cz), ("n", cx, cz)),
                ):
                    if other == kind or self.edge(other, oc, orr) != " ":
                        continue
                    # A doorway two tiles wide, a leaf of a pair of doors in each, runs on through
                    # the other leaf's edge to the wall beyond.
                    step_c, step_r = beyond[1] - oc, beyond[2] - orr
                    further = (other, beyond[1] + step_c, beyond[2] + step_r)
                    closed = self.edge(*beyond) in "#BC" or self.edge(*beyond) in " D" and self.edge(*further) in "#BC"
                    if not closed or self.edge(*before) not in "#BC":
                        continue
                    (self.west if other == "w" else self.north)[orr][oc] = "D"
                    (self.west if kind == "w" else self.north)[r][c] = " "
                    moved = True
                    break
                if moved:
                    break

    def between(self, a, b):
        (x0, y0), (x1, y1) = a, b
        if x1 == x0 + 1:
            return self.west[y1][x1]
        if x1 == x0 - 1:
            return self.west[y0][x0]
        if y1 == y0 + 1:
            return self.north[y1][x1]
        return self.north[y0][x0]

    # --- Rooms, heights and roofs ------------------------------------------------------------

    def find_rooms(self):
        self.rooms, done = [], set()
        for r in range(H):
            for c in range(W):
                if self.indoor(c, r) and (c, r) not in done and not self.crossed(c, r):
                    room, stack = {(c, r)}, [(c, r)]
                    while stack:
                        p = stack.pop()
                        for dc, dr in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                            n = (p[0] + dc, p[1] + dr)
                            if n in room or not self.indoor(*n) or self.crossed(*n) or self.between(p, n) in "#BD=":
                                continue
                            room.add(n)
                            stack.append(n)
                    done |= room
                    self.rooms.append(room)

    def gaps(self, room):
        found = []
        for x, y in room:
            for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                n = (x + dx, y + dy)
                if n in room or self.crossed(*n):
                    continue
                if self.between((x, y), n) not in "#BD":
                    found.append(((x, y), n))
        return found

    def roof_rooms(self):
        """Each room roofed as the game roofs it; one the game leaves unroofed stays open to the
        sky. A roofed room open on a few sides has archways there: doorways without doors."""
        roof_at = collections.defaultdict(collections.Counter)
        for p in self.roof_pieces:
            if p["id"] in ROOFING:
                roof_at[col_row(p["x"], p["y"])][ROOFING[p["id"]]] += 1

        def votes(room):
            counted = collections.Counter()
            for x, y in room:
                for dc in (-1, 0, 1):
                    for dr in (-1, 0, 1):
                        counted.update(roof_at.get((x + dc, y + dr), {}))
            return counted

        for room in self.rooms:
            open_sides = self.gaps(room)
            if votes(room) and 0 < len(open_sides) <= 16:
                for (x, y), (nx, ny) in open_sides:
                    if nx == x + 1:
                        self.west[y][nx] = "D"
                    elif nx == x - 1:
                        self.west[y][x] = "D"
                    elif ny == y + 1:
                        self.north[ny][x] = "D"
                    else:
                        self.north[y][x] = "D"
        for room in self.rooms:
            if self.gaps(room):
                continue
            counted = votes(room)
            roofing = counted.most_common(1)[0][0] if counted else "-"
            for x, y in room:
                self.roofs[y][x] = roofing

    def levels(self):
        """The game's heights scaled into levels 0 to 9, with each body of water, each room and each
        bridge deck lying level."""
        heights = self.terrain_data.height
        top = max(heights[self.key(c, r)] for r in range(H) for c in range(W))
        level = [[min(9, round(heights[self.key(c, r)] * 9 / top)) for c in range(W)] for r in range(H)]

        def flood(start, member):
            seen, stack = {start}, [start]
            while stack:
                c, r = stack.pop()
                for dc, dr in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                    n = (c + dc, r + dr)
                    if inside(*n) and n not in seen and member(*n):
                        seen.add(n)
                        stack.append(n)
            return seen

        done = set()
        for r in range(H):
            for c in range(W):
                if self.ground[r][c] == "~" and (c, r) not in done:
                    body = flood((c, r), lambda x, y: self.ground[y][x] == "~")
                    done |= body
                    low = min(level[y][x] for x, y in body)
                    for x, y in body:
                        level[y][x] = low
        for room in self.rooms:
            flat = statistics.mode(level[y][x] for x, y in room)
            for x, y in room:
                level[y][x] = flat
        done = set()
        for r in range(H):
            for c in range(W):
                if self.ground[r][c] == "H" and (c, r) not in done:
                    deck = flood((c, r), lambda x, y: self.ground[y][x] == "H")
                    done |= deck
                    ends = [level[y][x + dc] for x, y in deck for dc in (-1, 1)
                            if inside(x + dc, y) and self.ground[y][x + dc] not in "H~"]
                    for x, y in deck:
                        level[y][x] = max(ends)
        return level

    # --- Objects -----------------------------------------------------------------------------

    def furnish(self):
        for p in self.placed:
            name = self.names.get(p["id"])
            if p["type"] not in (10, 11) or name not in LETTER:
                continue
            across, up = self.sizes.get(p["id"], (1, 1))
            if p["rot"] in (1, 3):
                across, up = up, across
            c, r = col_row(p["x"], p["y"])
            tiles = [(c + i, r - j) for i in range(across) for j in range(up)]
            if not all(inside(*t) for t in tiles):
                continue
            if any(self.ground[y][x] in "~H+TOWYX" or self.objects[y][x] != "." for x, y in tiles):
                continue
            for x, y in tiles:
                self.objects[y][x] = LETTER[name]
            if LETTER[name] in MODELLED:
                self.shaped.append((p["id"], p["type"], p["rot"], c, r - up + 1, across, up))

    def hang_decorations(self):
        """The banners, tapestries, paintings, torches and the like hung on walls."""
        for p in self.placed:
            c, r = col_row(p["x"], p["y"])
            if 4 <= p["type"] <= 8 and inside(c, r):
                self.hung.append((p["id"], p["type"], p["rot"], c, r))

    @staticmethod
    def decoration_turns(kind, orientation):
        """How a wall decoration's model is turned, in degrees, and pushed off its tile's west edge:
        on a straight wall by its orientation, on a diagonal one a further eighth of a turn, the
        inside face of a diagonal wall half a turn round from the outside."""
        straight = (orientation * 90, DECORATION_CLEARANCE)
        across = 64 - DECORATION_CLEARANCE
        outside, inside = (orientation * 90 + 45, across), ((orientation + 2) * 90 + 45, across)
        return {4: [straight], 5: [straight], 6: [outside], 7: [inside], 8: [outside, inside]}[kind]

    def models_file(self, cache):
        """The models of the furniture drawn as the game models it, and where each stands."""
        palette = models.Palette(cache)
        lines = ["// The game's own models of Lumbridge's furniture, and of what hangs on its walls.",
                 "// Generated by tools/mapgen/generate.py."]
        meshes = {}
        for oid, kind, orientation, *_ in self.shaped:
            name = f"{oid}.{orientation}" + (".diagonal" if kind == 11 else "")
            if name not in meshes:
                definition = models.ObjectDefinition(self.definitions[oid])
                degrees = orientation * 90 + (45 if kind == 11 else 0)
                meshes[name] = models.placed_mesh(cache, palette, definition, 10, degrees, HEIGHT_SCALE)
        for oid, kind, orientation, *_ in self.hung:
            name = f"{oid}.{kind}.{orientation}"
            if name not in meshes:
                definition = models.ObjectDefinition(self.definitions[oid])
                points, faces = [], []
                for degrees, push in self.decoration_turns(kind, orientation):
                    more_points, more_faces = models.placed_mesh(cache, palette, definition, 4, degrees, HEIGHT_SCALE, push)
                    faces += [(a + len(points), b + len(points), c + len(points), rgb) for a, b, c, rgb in more_faces]
                    points += more_points
                meshes[name] = (points, faces)
        # The renderer has inks for only so many colours: the closest few are merged to fit.
        colour = models.reduce_colours([rgb for _, faces in meshes.values() for *_, rgb in faces], MODEL_COLOURS)
        for name, (points, faces) in meshes.items():
            lines += ["", f"model {name}"]
            lines += [f"point {x:.3f} {y:.3f} {z:.3f}".replace("-0.000", "0.000") for x, y, z in points]
            lines += ["face %d %d %d %02x%02x%02x" % (a, b, c, *colour[rgb]) for a, b, c, rgb in faces]
        lines.append("")
        for oid, kind, orientation, col, row, cols, rows in self.shaped:
            name = f"{oid}.{orientation}" + (".diagonal" if kind == 11 else "")
            lines.append(f"place {name} {col} {row} {cols} {rows}")
        for oid, kind, orientation, col, row in self.hung:
            lines.append(f"hang {oid}.{kind}.{orientation} {col} {row}")
        return lines

    # --- Writing -----------------------------------------------------------------------------

    def corner(self, c, r):
        meeting = [self.west[r - 1][c] if r > 0 else " ", self.west[r][c],
                   self.north[r][c - 1] if c > 0 else " ", self.north[r][c]]
        if inside(c - 1, r):
            meeting.append(self.rising[r][c - 1])
        if inside(c, r - 1):
            meeting.append(self.rising[r - 1][c])
        if inside(c, r):
            meeting.append(self.falling[r][c])
        if inside(c - 1, r - 1):
            meeting.append(self.falling[r - 1][c - 1])
        for kind in ("B", "=", "#", "D"):
            if kind in meeting:
                return "+" if kind in "#D" else kind
        return "."

    def walls_grid(self):
        lines = []
        for r in range(H + 1):
            lines.append("".join(self.corner(c, r) + (self.north[r][c] if c < W else "") for c in range(W + 1)).rstrip() or ".")
            if r < H:
                line = ""
                for c in range(W + 1):
                    line += self.west[r][c]
                    if c < W:
                        line += "/" if self.rising[r][c] != " " else ("\\" if self.falling[r][c] != " " else ".")
                lines.append(line.rstrip())
        return lines


def generate(cache_folder, areas_path, out_path):
    cache = Cache(cache_folder)
    lumbridge = Lumbridge(*read_game(cache))
    lumbridge.cache = cache
    lumbridge.lay_ground()
    lumbridge.stand_walls()
    lumbridge.find_rooms()
    level = lumbridge.levels()
    lumbridge.roof_rooms()
    lumbridge.furnish()
    lumbridge.hang_decorations()
    area_settings, areas = read_areas(areas_path)
    start = col_row(*START)
    text = [
        "// Lumbridge as in Old School RuneScape, one to one from the game's own map data (cache of",
        "// March 2026): x 3150 to 3283 west to east, y 3291 to 3184 north to south, one character a game",
        "// tile. Paths, floors, water, trees, graves, walls, fences, doorways, heights, roofs and objects",
        "// all stand where the game has them; only the areas are named by hand.",
        "// Generated by tools/mapgen/generate.py.",
        "name Lumbridge",
        f"start {start[0]} {start[1]}",
        f"models {os.path.splitext(os.path.basename(out_path))[0]}.models",
        "",
    ] + area_settings + ["", "terrain"] + ["".join(row) for row in lumbridge.ground] + [
        "", "// Walls, battlements, fences and doorways along the edges between tiles, at twice the",
        "// terrain's resolution.", "walls"] + lumbridge.walls_grid() + [
        "", "areas"] + areas + [
        "", "// The game's heights scaled into levels 0 to 9; each body of water, each room and each bridge",
        "// deck lies level.", "heights"] + ["".join(str(v) for v in row) for row in level] + [
        "", "// Each room roofed as the game roofs it: c clay tiles, t thatch, s slate, - open to the sky.",
        "roofs"] + ["".join(row) for row in lumbridge.roofs] + [
        "", "// Furniture and landmarks where the game stands them, each filling its footprint.",
        "objects"] + ["".join(row) for row in lumbridge.objects]
    with open(out_path, "w") as f:
        f.write("\n".join(text) + "\n")
    with open(os.path.splitext(out_path)[0] + ".models", "w") as f:
        f.write("\n".join(lumbridge.models_file(cache)) + "\n")


def main():
    parser = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    lumbridge_map = os.path.join(ROOT, "maps", "lumbridge.map")
    parser.add_argument("--out", default=lumbridge_map, help="the map file to write")
    parser.add_argument("--areas", default=lumbridge_map, help="the map file whose named areas to keep")
    parser.add_argument("--cache", default=os.path.join(HERE, ".cache"), help="where to keep the downloaded cache")
    args = parser.parse_args()
    generate(args.cache, args.areas, args.out)


if __name__ == "__main__":
    main()
