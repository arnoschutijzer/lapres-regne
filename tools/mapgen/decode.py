"""Decode the game's map squares and object definitions."""

import math
import re
import struct

# Index 5 holds the map squares, index 2 the configs: group 6 the object definitions.
MAPS, CONFIGS, OBJECTS = 5, 2, 6

COS = [int(65536 * math.cos(i * 0.0030679615)) for i in range(2048)]


def _noise(x, y):
    n = x + y * 57
    n ^= (n << 13) & 0xFFFFFFFF
    n &= 0xFFFFFFFF
    n = n - (1 << 32) if n >= 1 << 31 else n
    return (((n * (n * n * 15731 + 789221) + 1376312589) & 0x7FFFFFFF) >> 19) & 0xFF


def _smooth(x, y):
    corners = _noise(x - 1, y - 1) + _noise(x + 1, y - 1) + _noise(x - 1, y + 1) + _noise(x + 1, y + 1)
    sides = _noise(x - 1, y) + _noise(x + 1, y) + _noise(x, y - 1) + _noise(x, y + 1)
    return corners // 16 + sides // 8 + _noise(x, y) // 4


def _cosine(a, b, x, scale):
    f = (65536 - COS[x * 1024 // scale]) >> 1
    return ((65536 - f) * a >> 16) + (f * b >> 16)


def _interpolate(x, y, scale):
    ix, fx, iy, fy = x // scale, x & (scale - 1), y // scale, y & (scale - 1)
    a, b, c, d = _smooth(ix, iy), _smooth(ix + 1, iy), _smooth(ix, iy + 1), _smooth(ix + 1, iy + 1)
    return _cosine(_cosine(a, b, fx, scale), _cosine(c, d, fx, scale), fy, scale)


def procedural_height(x, y):
    """The height the game gives a tile its map square leaves to chance, as the client does."""
    n = (_interpolate(x + 45365, y + 91923, 4) - 128
         + (_interpolate(x + 10294, y + 37821, 2) - 128 >> 1)
         + (_interpolate(x, y, 1) - 128 >> 2))
    return max(10, min(60, int(n * 0.3) + 35))


class Terrain:
    """The ground of map squares, by world tile x, y: heights at level 0, floors and settings."""

    def __init__(self):
        self.height = {}
        # overlays holds each level's floors, (id, shape, rotation), and overlay those at level 0;
        # underlay is an id, at level 0.
        self.overlays = {z: {} for z in range(4)}
        self.overlay, self.underlay = self.overlays[0], {}
        # settings holds each tile's flags at levels 0 and 1, by (level, x, y).
        self.settings = {}

    def read(self, data, square_x, square_y):
        pos = 0

        def u8():
            nonlocal pos
            pos += 1
            return data[pos - 1]

        def u16():
            nonlocal pos
            pos += 2
            return struct.unpack(">H", data[pos - 2:pos])[0]

        for z in range(4):
            for lx in range(64):
                for ly in range(64):
                    x, y = square_x * 64 + lx, square_y * 64 + ly
                    while True:
                        op = u16()
                        if op == 0:
                            if z == 0:
                                self.height[(x, y)] = procedural_height(x + 932731, y + 556238) * 8
                            break
                        if op == 1:
                            h = u8()
                            if z == 0:
                                self.height[(x, y)] = (0 if h == 1 else h) * 8
                            break
                        if op <= 49:
                            overlay = u16()
                            self.overlays[z][(x, y)] = (overlay, (op - 2) // 4, (op - 2) & 3)
                        elif op <= 81:
                            if z <= 1:
                                self.settings[(z, x, y)] = op - 49
                        elif z == 0:
                            self.underlay[(x, y)] = op - 81


def _smart(data, pos):
    if data[pos] < 128:
        return data[pos], pos + 1
    return struct.unpack(">H", data[pos:pos + 2])[0] - 32768, pos + 2


def _big_smart(data, pos):
    total = 0
    value, pos = _smart(data, pos)
    while value == 32767:
        total += 32767
        value, pos = _smart(data, pos)
    return total + value, pos


def locations(data, square_x, square_y):
    """The objects a map square places: id, world x and y, level, type and rotation."""
    placed, pos, oid = [], 0, -1
    while True:
        step, pos = _big_smart(data, pos)
        if step == 0:
            return placed
        oid += step
        position = 0
        while True:
            offset, pos = _smart(data, pos)
            if offset == 0:
                break
            position += offset - 1
            attributes = data[pos]
            pos += 1
            placed.append({
                "id": oid, "x": square_x * 64 + (position >> 6 & 63), "y": square_y * 64 + (position & 63),
                "z": position >> 12 & 3, "type": attributes >> 2, "rot": attributes & 3,
            })


# Object definition opcodes and how many bytes follow each, for those read past on the way to the
# size; 0 takes none.
_U8 = {19, 28, 29, 39, 69, 75, 81}
_NONE = {17, 18, 21, 22, 23, 62, 64, 73, 74, 89}
_U16 = {24, 61, 65, 66, 67, 68, 70, 71, 72, 82}


def object_size(data):
    """An object's footprint in tiles across and up, read from its definition's first opcodes."""
    pos, across, up = 0, 1, 1
    try:
        while pos < len(data):
            op = data[pos]
            pos += 1
            if op == 0:
                break
            if op == 14:
                across = data[pos]
                pos += 1
            elif op == 15:
                up = data[pos]
                pos += 1
            elif op in _U8:
                pos += 1
            elif op in _NONE:
                pass
            elif op in _U16:
                pos += 2
            elif op == 2 or 30 <= op <= 34:
                pos = data.index(0, pos) + 1
            elif op in (40, 41):
                pos += 1 + 4 * data[pos]
            elif op == 1:
                pos += 1 + 3 * data[pos]
            elif op == 5:
                pos += 1 + 2 * data[pos]
            else:
                break
    except (IndexError, ValueError):
        pass
    return across, up


def object_name(data):
    """An object's name: the last name string in its definition, or None for an unnamed one."""
    names = re.findall(rb"\x02([A-Za-z][ -~]*?)\x00", data)
    return names[-1].decode() if names else None
