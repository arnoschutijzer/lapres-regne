"""Decode the game's object definitions and 3D models, ported from RuneLite's cache loaders
(github.com/runelite/runelite, cache/.../definitions/loaders), and turn a placed object's model
into coloured triangles in the map's world units."""

import math
import struct

# Index 7 holds the models.
MODELS = 7


class Reader:
    def __init__(self, data, pos=0):
        self.data, self.pos = data, pos

    def u8(self):
        self.pos += 1
        return self.data[self.pos - 1]

    def s8(self):
        v = self.u8()
        return v - 256 if v >= 128 else v

    def u16(self):
        self.pos += 2
        return struct.unpack(">H", self.data[self.pos - 2:self.pos])[0]

    def s16(self):
        self.pos += 2
        return struct.unpack(">h", self.data[self.pos - 2:self.pos])[0]

    def u24(self):
        self.pos += 3
        return int.from_bytes(self.data[self.pos - 3:self.pos], "big")

    def s32(self):
        self.pos += 4
        return struct.unpack(">i", self.data[self.pos - 4:self.pos])[0]

    def string(self):
        end = self.data.index(0, self.pos)
        text = self.data[self.pos:end].decode("latin-1")
        self.pos = end + 1
        return text

    def short_smart(self):
        if self.data[self.pos] < 128:
            return self.u8() - 64
        return self.u16() - 0xC000


class ObjectDefinition:
    def __init__(self, data):
        self.name, self.models, self.types = None, [], None
        self.size_x = self.size_y = 1
        self.recolor, self.retexture = [], []
        self.scale = [128, 128, 128]
        self.offset = [0, 0, 0]
        self.mirrored = False
        r = Reader(data)
        while True:
            op = r.u8()
            if op == 0:
                break
            self.read(op, r)

    def read(self, op, r):
        if op in (1, 6):
            n = r.u8()
            pairs = [((r.u16() if op == 1 else r.s32()), r.u8()) for _ in range(n)]
            self.models, self.types = [m for m, _ in pairs], [t for _, t in pairs]
        elif op == 2:
            self.name = r.string()
        elif op in (5, 7):
            n = r.u8()
            self.models, self.types = [r.u16() if op == 5 else r.s32() for _ in range(n)], None
        elif op == 14:
            self.size_x = r.u8()
        elif op == 15:
            self.size_y = r.u8()
        elif op in (17, 18, 21, 22, 23, 27, 62, 64, 73, 74, 89, 90, 94):
            if op == 62:
                self.mirrored = True
        elif op in (19, 28, 75, 81, 91, 95, 96):
            r.u8()
        elif op in (29, 39, 69):
            r.s8()
        elif op in (24, 42, 61, 68, 82):
            r.u16()
        elif 30 <= op < 35:
            r.string()
        elif op in (40, 41):
            n = r.u8()
            pairs = [(r.s16() & 0xFFFF, r.s16() & 0xFFFF) for _ in range(n)]
            (self.recolor if op == 40 else self.retexture).extend(pairs)
        elif op in (65, 66, 67):
            self.scale[op - 65] = r.u16()
        elif op in (70, 71, 72):
            self.offset[op - 70] = r.s16()
        elif op in (77, 92):
            r.u16(); r.u16()
            if op == 92:
                r.u16()
            for _ in range(r.u8() + 1):
                r.u16()
        elif op == 78:
            r.u16(); r.u8(); r.u8()
        elif op == 79:
            r.u16(); r.u16(); r.u8(); r.u8()
            for _ in range(r.u8()):
                r.u16()
        elif op == 93:
            r.u8(); r.u16(); r.u8(); r.u16()
        elif op == 100:
            r.u8(); r.u8(); r.string()
        elif op == 101:
            r.u8(); r.u16(); r.u16(); r.s32(); r.s32(); r.string()
        elif op == 102:
            r.u8(); r.u16(); r.u16(); r.u16(); r.s32(); r.s32(); r.string()
        elif op == 249:
            for _ in range(r.u8()):
                kind = r.u8()
                r.u24()
                if kind == 1:
                    r.string()
                elif kind == 2:
                    r.pos += 8
                else:
                    r.s32()
        else:
            raise ValueError(f"unknown object opcode {op}")

    def models_for(self, kind):
        """The models an object placed as a type draws."""
        if self.types is None:
            return self.models if kind == 10 else []
        return [m for m, t in zip(self.models, self.types) if t == kind]


class Model:
    """A model's points (x, y down, z) and triangles, each with a colour or a texture."""

    def __init__(self, data):
        if data[-2] == 0xFF and data[-1] == 0xFD:
            self.decode_new(data, 26, version=3)
        elif data[-2] == 0xFF and data[-1] == 0xFE:
            self.decode_old(data, 23, extended=True)
        elif data[-2] == 0xFF and data[-1] == 0xFF:
            self.decode_new(data, 23, version=1)
        else:
            self.decode_old(data, 18, extended=False)

    def read_points(self, flags, xs, ys, zs, count):
        x = y = z = 0
        self.x, self.y, self.z = [], [], []
        for _ in range(count):
            f = flags.u8()
            x += xs.short_smart() if f & 1 else 0
            y += ys.short_smart() if f & 2 else 0
            z += zs.short_smart() if f & 4 else 0
            self.x.append(x)
            self.y.append(y)
            self.z.append(z)

    def read_faces(self, indices, kinds, count):
        a = b = c = last = 0
        self.faces = []
        for _ in range(count):
            kind = kinds.u8()
            if kind == 1:
                a = indices.short_smart() + last
                b = indices.short_smart() + a
                c = indices.short_smart() + b
                last = c
            elif kind == 2:
                b = c
                c = indices.short_smart() + last
                last = c
            elif kind == 3:
                a = c
                c = indices.short_smart() + last
                last = c
            elif kind == 4:
                a, b = b, a
                c = indices.short_smart() + last
                last = c
            self.faces.append((a, b, c))

    def decode_old(self, data, footer, extended):
        """The original format (footer 18), and the type 2 format (footer 23), which adds
        animaya groups but lays out its faces alike."""
        r = Reader(data, len(data) - footer)
        vertices, faces, textures = r.u16(), r.u16(), r.u8()
        textured, priority, transparent, transparency_groups = r.u8(), r.u8(), r.u8(), r.u8()
        if extended:
            vertex_groups, _animaya = r.u8(), r.u8()
        else:
            vertex_groups = r.u8()
        x_len, y_len, z_len, index_len = r.u16(), r.u16(), r.u16(), r.u16()
        extra = r.u16() if extended else 0
        pos = 0
        flags_at = pos; pos += vertices
        kinds_at = pos; pos += faces
        pos += faces if priority == 255 else 0
        pos += faces if transparency_groups == 1 else 0
        texture_flags_at = pos; pos += faces if textured == 1 else 0
        if extended:
            pos += extra  # vertex groups and animaya data
        else:
            pos += vertices if vertex_groups == 1 else 0
        alpha_at = pos; pos += faces if transparent == 1 else 0
        indices_at = pos; pos += index_len
        colors_at = pos; pos += faces * 2
        pos += textures * 6
        x_at = pos; pos += x_len
        y_at = pos; pos += y_len
        z_at = pos
        self.read_points(Reader(data, flags_at), Reader(data, x_at), Reader(data, y_at), Reader(data, z_at), vertices)
        colors, flags, alpha = Reader(data, colors_at), Reader(data, texture_flags_at), Reader(data, alpha_at)
        self.colors, self.textures, self.alphas = [], [], []
        for _ in range(faces):
            color = colors.u16()
            texture = -1
            if textured == 1 and flags.u8() & 2:
                texture, color = color, 127
            self.colors.append(color)
            self.textures.append(texture)
            self.alphas.append(alpha.u8() if transparent == 1 else 0)
        self.read_faces(Reader(data, indices_at), Reader(data, kinds_at), faces)

    def decode_new(self, data, footer, version):
        """The type 1 (footer 23) and type 3 (footer 26) formats."""
        r = Reader(data, len(data) - footer)
        vertices, faces, textures = r.u16(), r.u16(), r.u8()
        render_types, priority, transparent, transparency_groups = r.u8(), r.u8(), r.u8(), r.u8()
        textured, vertex_groups = r.u8(), r.u8()
        animaya = r.u8() if version == 3 else 0
        x_len, y_len, z_len, index_len, texture_index_len = r.u16(), r.u16(), r.u16(), r.u16(), r.u16()
        extra = r.u16() if version == 3 else 0
        pos = textures + vertices
        flags_at = textures
        pos_render = pos; pos += faces if render_types == 1 else 0
        kinds_at = pos; pos += faces
        pos += faces if priority == 255 else 0
        pos += faces if transparency_groups == 1 else 0
        if version == 1:
            pos += vertices if vertex_groups == 1 else 0
        else:
            pos += extra  # vertex groups and animaya data
        alpha_at = pos; pos += faces if transparent == 1 else 0
        indices_at = pos; pos += index_len
        texture_ids_at = pos; pos += faces * 2 if textured == 1 else 0
        texture_coords_at = pos; pos += texture_index_len
        colors_at = pos; pos += faces * 2
        x_at = pos; pos += x_len
        y_at = pos; pos += y_len
        z_at = pos
        self.read_points(Reader(data, flags_at), Reader(data, x_at), Reader(data, y_at), Reader(data, z_at), vertices)
        colors, alpha, texture_ids = Reader(data, colors_at), Reader(data, alpha_at), Reader(data, texture_ids_at)
        self.colors, self.textures, self.alphas = [], [], []
        for _ in range(faces):
            self.colors.append(colors.u16())
            self.alphas.append(alpha.u8() if transparent == 1 else 0)
            self.textures.append(texture_ids.u16() - 1 if textured == 1 else -1)
        self.read_faces(Reader(data, indices_at), Reader(data, kinds_at), faces)


def hsl_to_rgb(hsl, brightness=0.8):
    """A 16-bit colour of the game's, its hue, saturation and luminance packed, as red, green, blue."""
    hue = (hsl >> 10 & 63) / 64 + 0.5 / 64
    saturation = (hsl >> 7 & 7) / 8 + 0.5 / 8
    luminance = (hsl & 127) / 128
    chroma = (1 - abs(2 * luminance - 1)) * saturation
    x = chroma * (1 - abs((hue * 6) % 2 - 1))
    lightness = luminance - chroma / 2
    r = g = b = lightness
    sector = int(hue * 6)
    if sector == 0: r, g = r + chroma, g + x
    elif sector == 1: g, r = g + chroma, r + x
    elif sector == 2: g, b = g + chroma, b + x
    elif sector == 3: b, g = b + chroma, g + x
    elif sector == 4: b, r = b + chroma, r + x
    else: r, b = r + chroma, b + x
    return tuple(min(255, int(((c * 256) / 256) ** brightness * 256)) for c in (r, g, b))


# Textures live in index 9, group 0; each names the sprite it is painted from, in index 8.
TEXTURES, SPRITES = 9, 8


class Palette:
    """The colours of the game's textures, each the average of its sprite."""

    def __init__(self, cache):
        self.cache = cache
        self.textures = cache.files(TEXTURES, 0)
        self.known = {}

    def texture(self, texture):
        if texture not in self.known:
            sprite = struct.unpack(">H", self.textures[texture][:2])[0]
            self.known[texture] = sprite_average(self.cache.group(SPRITES, sprite))
        return self.known[texture]


def sprite_average(data):
    """The average colour of a sprite's first frame, its transparent pixels left out."""
    frames = struct.unpack(">H", data[-2:])[0]
    footer = len(data) - 7 - frames * 8
    width, height = struct.unpack(">HH", data[footer:footer + 4])
    colours = data[footer + 4] + 1
    widths_at = footer + 5 + frames * 4
    frame_width = struct.unpack(">H", data[widths_at:widths_at + 2])[0]
    frame_height = struct.unpack(">H", data[widths_at + frames * 2:widths_at + frames * 2 + 2])[0]
    table = footer - (colours - 1) * 3
    palette = [(0, 0, 0)] + [tuple(data[table + 3 * i:table + 3 * i + 3]) for i in range(colours - 1)]
    pixels = [palette[b] for b in data[1:1 + frame_width * frame_height] if b]
    if not pixels:
        return (128, 128, 128)
    return tuple(sum(p[i] for p in pixels) // len(pixels) for i in range(3))


def placed_mesh(cache, palette, definition, kind, degrees, height_scale, push=0):
    """The triangles an object draws with its models of a type, turned by degrees and first pushed
    push of the game's units east: points in world units about the middle of its footprint (x east,
    y up, z south), each triangle's colour as red, green, blue. A tile is 128 of the game's units
    across and two world units; heights are scaled by height_scale besides, to keep models in
    proportion with the hero."""
    angle = math.radians(degrees)
    cos, sin = math.cos(angle), math.sin(angle)
    recolor = dict(definition.recolor)
    retexture = dict(definition.retexture)
    points, faces = [], []
    for model_id in definition.models_for(kind):
        model = Model(cache.group(MODELS, model_id))
        base = len(points)
        for x, y, z in zip(model.x, model.y, model.z):
            if definition.mirrored:
                z = -z
            x += push
            # Each quarter turn takes x, z to z, -x.
            x, z = x * cos + z * sin, -x * sin + z * cos
            x = x * definition.scale[0] / 128 + definition.offset[0]
            y = y * definition.scale[1] / 128 - definition.offset[1]
            z = z * definition.scale[2] / 128 + definition.offset[2]
            points.append((x / 64, -y / 64 * height_scale, -z / 64))
        for (a, b, c), colour, texture, alpha in zip(model.faces, model.colors, model.textures, model.alphas):
            if alpha >= 254:
                continue
            if texture >= 0:
                rgb = palette.texture(retexture.get(texture, texture))
            else:
                rgb = hsl_to_rgb(recolor.get(colour, colour))
            if definition.mirrored:
                b, c = c, b
            faces.append((base + a, base + b, base + c, rgb))
    return points, faces


def reduce_colours(colours, limit):
    """Merge the closest of a set of colours until at most limit remain, the commonest kept as they
    are: each colour maps to the one standing for it."""
    counts = {}
    for colour in colours:
        counts[colour] = counts.get(colour, 0) + 1
    order = sorted(counts, key=lambda c: -counts[c])
    reach = 0
    while True:
        kept, mapping = [], {}
        for colour in order:
            near = min(kept, key=lambda k: sum((a - b) ** 2 for a, b in zip(k, colour)), default=None)
            if near is not None and sum((a - b) ** 2 for a, b in zip(near, colour)) <= reach:
                mapping[colour] = near
            else:
                kept.append(colour)
                mapping[colour] = colour
        if len(kept) <= limit:
            return mapping
        reach = max(1, reach * 2)
