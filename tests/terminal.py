"""Exercise the actual CLI in a pseudo-terminal using Python's standard library."""

import fcntl
import os
from pathlib import Path
import pty
import math
import re
import select
import signal
import struct
import subprocess
import tempfile
import termios
import time


ROOT = Path(__file__).resolve().parents[1]


class Terminal:
    def __init__(self, binary, *args):
        self.master, self.slave = pty.openpty()
        self.resize(100, 30)
        self.original = termios.tcgetattr(self.slave)
        self.process = subprocess.Popen(
            [binary, *args], stdin=self.slave, stdout=self.slave, stderr=self.slave
        )

    def resize(self, width, height):
        fcntl.ioctl(
            self.slave, termios.TIOCSWINSZ, struct.pack("HHHH", height, width, 0, 0)
        )

        if hasattr(self, "process"):
            self.process.send_signal(signal.SIGWINCH)

    def read(self, until=None, timeout=3):
        data = b""
        deadline = time.monotonic() + timeout
        while time.monotonic() < deadline:
            if select.select([self.master], [], [], 0.05)[0]:
                data += os.read(self.master, 1_000_000)
                if until and until in data:
                    return data
        if until:
            raise AssertionError(f"missing terminal output: {until!r}")
        return data

    def send(self, keys):
        os.write(self.master, keys)

    def assert_restored(self):
        cleanup = self.read(until=b"\x1b[?1049l")
        assert self.process.wait(timeout=3) == 0
        assert b"\x1b[?25h" in cleanup, "cursor must be visible after quitting"
        assert b"\x1b[<u" in cleanup, "keyboard protocol must be restored after quitting"
        assert termios.tcgetattr(self.slave) == self.original

    def close(self):
        if self.process.poll() is None:
            self.process.terminate()
            self.process.wait(timeout=3)
        os.close(self.master)
        os.close(self.slave)


def tree_pixels(output):
    """Compare visible tree glyphs, excluding the header and slider cursor."""
    frame = output.split(b"\x1b[H")[-1].decode(errors="ignore")
    return "".join(c for c in frame if "\u2800" <= c <= "\u28ff")


def check_interaction(binary):
    terminal = Terminal(binary, "--tree")
    try:
        initial = terminal.read(until=b"\x1b[51G\x1b[38;2;225;239;208m_")
        assert len(tree_pixels(initial)) > 100, "tree must be visible"
        terminal.send(b"j" * 12)
        moved = terminal.read(until=b"\x1b[39G\x1b[38;2;225;239;208m_")
        moved += terminal.read(timeout=1)
        assert tree_pixels(moved) != tree_pixels(initial), "camera must rotate the tree"

        # Arrow escape sequences may arrive in separate reads.
        terminal.send(b"\x1b")
        time.sleep(0.03)
        terminal.send(b"[C")
        terminal.read(until=b"\x1b[40G\x1b[38;2;225;239;208m_")
        terminal.send(b"\x1b[D")
        terminal.read(until=b"\x1b[39G\x1b[38;2;225;239;208m_")
        terminal.send(b"L")
        terminal.read(until=b"\x1b[40G\x1b[38;2;225;239;208m_")

        terminal.send(b"J" * 150)
        terminal.read(until=b"\x1b[1G\x1b[38;2;225;239;208m_")
        terminal.send(b"l" * 150)
        terminal.read(until=b"\x1b[100G\x1b[38;2;225;239;208m_")

        terminal.resize(60, 20)
        terminal.read(until=b"\x1b[20;1H")
        terminal.resize(20, 8)
        terminal.read(until=b"Resize to 40 x 14.")
        terminal.resize(100, 30)
        terminal.read(until=b"\x1b[30;1H")
        terminal.send(b"q")
        terminal.assert_restored()
    finally:
        terminal.close()


def check_exit(binary, action):
    terminal = Terminal(binary)
    try:
        terminal.read(until=b"\x1b[30;1H")
        if isinstance(action, bytes):
            terminal.send(action)
        else:
            terminal.process.send_signal(action)
        terminal.assert_restored()
    finally:
        terminal.close()


def check_forest(binary):
    terminal = Terminal(binary)
    try:
        initial = terminal.read(until=b"steps 0")
        assert b"W I L D S" in initial
        assert b"236;185;105m" in initial, "traveler must be visible"
        terminal.send(b"d")
        terminal.read(until=b"x 0.6  z 0.0  |  steps 1")
        terminal.send(b"\x1b[A")
        terminal.read(until=b"x 0.6  z -0.6  |  steps 2")
        terminal.send(b"js")
        terminal.read(until=b"x 0.0  z 0.0  |  steps 4")
        regions = b""
        for step in range(5, 65):
            terminal.send(b"d")
            regions += terminal.read(until=f"steps {step}  |".encode())
        assert b"river / wading" in regions, "walk must reach and cross a procedural river"
        assert tree_pixels(regions) != tree_pixels(initial), "walking must reveal new terrain"
        terminal.send(b"d" * 180)
        terminal.read(until=b"steps 244  |")
        terminal.resize(20, 8)
        terminal.read(until=b"Resize to 40 x 14.")
        terminal.resize(100, 30)
        terminal.read(until=b"\x1b[30;1H")
        terminal.send(b"q")
        terminal.assert_restored()
    finally:
        terminal.close()


def check_responsive_rendering(binary):
    terminal = Terminal(binary)
    try:
        terminal.read(until=b"seed 7")
        terminal.resize(220, 90)
        initial = terminal.read(until=b"seed 7")
        idle = terminal.read(timeout=0.35)
        frames = idle.count(b"\x1b[H")
        assert frames >= 15, "animation must use the new 60 FPS cadence"
        assert b"\x1b[2J" not in idle, "steady animation must not clear the screen"
        assert len(idle) / frames < len(initial) / 2, "idle frames must send incremental updates"

        terminal.send(b"d" * 40)
        moved = terminal.read(until=b"steps 40  |")
        assert moved.count(b"\x1b[H") <= 3, "queued input must not trigger one redraw per key"
        terminal.send(b"q")
        terminal.assert_restored()
    finally:
        terminal.close()


def positions(output):
    return [
        (float(x), float(z))
        for x, z in re.findall(rb"  x (-?[\d.]+)  z (-?[\d.]+)  \|", output)
    ]


def check_held_movement(binary):
    terminal = Terminal(binary)
    try:
        startup = terminal.read(until=b"seed 7")
        assert b"\x1b[>11u\x1b[?u" in startup, "request key press/release reporting"
        terminal.send(b"\x1b[?11u")  # Simulate a supporting terminal's capability reply.
        terminal.send(b"\x1b[1;1D\x1b[1;1A")  # Hold LEFT + UP without repeats.
        start = time.monotonic()
        diagonal = positions(terminal.read(timeout=0.35))
        elapsed = time.monotonic() - start
        assert diagonal and diagonal[-1][0] < -1 and diagonal[-1][1] < -1
        assert abs(diagonal[-1][0] - diagonal[-1][1]) < 0.15
        assert 0.8 < math.hypot(*diagonal[-1]) / (9 * elapsed) < 1.2, "diagonal speed must be normalized"

        # Repeats do not add movement impulses, and releasing LEFT preserves UP.
        terminal.send(b"\x1b[1;1:2A" * 20 + b"\x1b[1;1:3D")
        vertical = positions(terminal.read(timeout=0.35))
        assert vertical and abs(vertical[-1][0] - diagonal[-1][0]) < 0.2
        assert 2 < diagonal[-1][1] - vertical[-1][1] < 4.3, "repeats must not add movement impulses"

        # Independent physical aliases: release the arrow while keeping W held.
        terminal.send(b"\x1b[119;1u\x1b[1;1:3A")
        letter = positions(terminal.read(timeout=0.2))
        assert letter and letter[-1][1] < vertical[-1][1] - 1
        terminal.send(b"\x1b[119;1:")
        time.sleep(0.01)
        terminal.send(b"3u")
        terminal.read(timeout=0.1)
        assert not positions(terminal.read(timeout=0.2)), "releasing all movement keys must stop immediately"

        terminal.send(b"\x1b[100;1u")
        assert positions(terminal.read(timeout=0.15))
        terminal.send(b"\x1b[O")  # Losing focus clears held keys.
        terminal.read(timeout=0.1)
        assert not positions(terminal.read(timeout=0.2)), "focus loss must stop movement"
        terminal.send(b"\x1b[113;1:3u")  # Releasing Q must not quit.
        terminal.read(timeout=0.05)
        assert terminal.process.poll() is None
        terminal.send(b"\x1b[113;1u")
        terminal.assert_restored()
    finally:
        terminal.close()


with tempfile.TemporaryDirectory(prefix="dither-test-") as directory:
    binary = str(Path(directory) / "dither-cli")
    subprocess.run(["go", "build", "-o", binary, "."], cwd=ROOT, check=True)
    help_result = subprocess.run([binary, "-h"], capture_output=True)
    assert help_result.returncode == 0 and b"Usage: dither-cli" in help_result.stderr
    non_terminal = subprocess.run([binary], stdin=subprocess.DEVNULL, capture_output=True)
    assert non_terminal.returncode == 1 and b"interactive terminal" in non_terminal.stderr
    check_interaction(binary)
    check_forest(binary)
    check_responsive_rendering(binary)
    check_held_movement(binary)
    check_exit(binary, b"\x03")
    check_exit(binary, b"\x1b[1;\x03")
    check_exit(binary, b"\x1b[99;5u")
    check_exit(binary, signal.SIGTERM)
    print("Passed: held diagonals, independent releases, normalized speed, focus loss, 60 FPS cadence, legacy controls, exploration, incremental redraws, cleanup")
