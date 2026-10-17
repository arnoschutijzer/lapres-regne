"""Read Old School RuneScape's game cache from the OpenRS2 archive, one group at a time.

Groups are downloaded from archive.openrs2.org once and kept in a local folder, so later runs work
offline. Only the Python standard library is used.
"""

import bz2
import gzip
import json
import os
import struct
import urllib.request

# The cache this tool reads: March 2026, build 236. Newer caches leave out the names of the map
# groups, which this tool finds them by.
CACHE_ID = 2499
ARCHIVE = "https://archive.openrs2.org/caches/runescape"


class Cache:
    """One game cache, its groups kept in a local folder."""

    def __init__(self, folder, cache_id=CACHE_ID):
        self.folder = folder
        self.cache_id = cache_id
        os.makedirs(folder, exist_ok=True)

    def download(self, name, url):
        path = os.path.join(self.folder, name)
        if not os.path.exists(path):
            request = urllib.request.Request(url, headers={"User-Agent": "lapresregne-mapgen"})
            with urllib.request.urlopen(request) as response, open(path + ".part", "wb") as out:
                out.write(response.read())
            os.replace(path + ".part", path)
        with open(path, "rb") as f:
            return f.read()

    def raw(self, archive, group):
        """The container of a group, still compressed and perhaps encrypted."""
        return self.download(f"{self.cache_id}_{archive}_{group}.dat", f"{ARCHIVE}/{self.cache_id}/archives/{archive}/groups/{group}.dat")

    def group(self, archive, group, key=None):
        """A group's data, decrypted with its key when it has one and decompressed."""
        raw = self.raw(archive, group)
        if key:
            kind, length = raw[0], struct.unpack(">I", raw[1:5])[0]
            body = raw[5:5 + length + (4 if kind else 0)]
            raw = raw[:5] + xtea_decrypt(body, key)
        return unpack(raw)

    def keys(self):
        """The keys to the map squares' location groups, by group name."""
        data = self.download(f"{self.cache_id}_keys.json", f"{ARCHIVE}/{self.cache_id}/keys.json")
        return {entry["name"]: entry["key"] for entry in json.loads(data) if entry.get("name")}

    def table(self, archive):
        """The reference table describing an archive's groups."""
        return ReferenceTable(self.group(255, archive))

    def files(self, archive, group):
        """The files packed in a group, by file id."""
        ids = self.table(archive).files[group]
        return dict(zip(ids, split(self.group(archive, group), len(ids))))


def unpack(raw):
    """Decompress a container: none, bzip2 or gzip."""
    kind, length = raw[0], struct.unpack(">I", raw[1:5])[0]
    if kind == 0:
        return raw[5:5 + length]
    data = raw[9:9 + length]
    if kind == 2:
        return gzip.decompress(data)
    return bz2.decompress(b"BZh1" + data)


def xtea_decrypt(data, key):
    """Decrypt data with the 128-bit XTEA key, 32 rounds, block by block."""
    out = bytearray(data)
    k = [x & 0xFFFFFFFF for x in key]
    for i in range(0, len(data) - len(data) % 8, 8):
        v0, v1 = struct.unpack(">II", data[i:i + 8])
        delta, total = 0x9E3779B9, (0x9E3779B9 * 32) & 0xFFFFFFFF
        for _ in range(32):
            v1 = (v1 - ((((v0 << 4) ^ (v0 >> 5)) + v0) ^ (total + k[(total >> 11) & 3]))) & 0xFFFFFFFF
            total = (total - delta) & 0xFFFFFFFF
            v0 = (v0 - ((((v1 << 4) ^ (v1 >> 5)) + v1) ^ (total + k[total & 3]))) & 0xFFFFFFFF
        out[i:i + 8] = struct.pack(">II", v0, v1)
    return bytes(out)


def name_hash(name):
    """The hash a group's name is stored as."""
    h = 0
    for ch in name:
        h = (h * 31 + ord(ch)) & 0xFFFFFFFF
    return h - (1 << 32) if h >= 1 << 31 else h


class ReferenceTable:
    """An archive's groups: their ids by name hash, and the file ids packed in each."""

    def __init__(self, data):
        pos = 0
        protocol = data[pos]
        pos += 1
        if protocol >= 6:
            pos += 4
        flags = data[pos]
        pos += 1

        def count():
            nonlocal pos
            if protocol >= 7 and data[pos] & 0x80:
                value = struct.unpack(">I", data[pos:pos + 4])[0] & 0x7FFFFFFF
                pos += 4
            else:
                value = struct.unpack(">H", data[pos:pos + 2])[0]
                pos += 2
            return value

        n = count()
        ids, last = [], 0
        for _ in range(n):
            last += count()
            ids.append(last)
        self.named = {}
        if flags & 1:
            for gid in ids:
                self.named[struct.unpack(">i", data[pos:pos + 4])[0]] = gid
                pos += 4
        pos += 4 * n  # checksums
        if flags & 8:
            pos += 4 * n
        if flags & 2:
            pos += 64 * n
        if flags & 4:
            pos += 8 * n
        pos += 4 * n  # versions
        counts = [count() for _ in range(n)]
        self.files = {}
        for gid, c in zip(ids, counts):
            fids, last = [], 0
            for _ in range(c):
                last += count()
                fids.append(last)
            self.files[gid] = fids

    def find(self, name):
        return self.named[name_hash(name)]


def split(data, n):
    """Split a group's data into its n files, stored in chunks."""
    chunks = data[-1]
    pos = len(data) - 1 - chunks * n * 4
    sizes = []
    for _ in range(chunks):
        size, row = 0, []
        for _ in range(n):
            size += struct.unpack(">i", data[pos:pos + 4])[0]
            pos += 4
            row.append(size)
        sizes.append(row)
    files, offset = [b""] * n, 0
    for row in sizes:
        for f in range(n):
            files[f] += data[offset:offset + row[f]]
            offset += row[f]
    return files
