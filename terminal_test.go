//go:build darwin || linux

package main

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
)

// Exercise the compiled application through a real pseudo-terminal.
type testTerminal struct {
	t             *testing.T
	master, slave *os.File
	cmd           *exec.Cmd
	done          chan error
	original      string
}

func openTestTerminal(t *testing.T, binary string, args ...string) *testTerminal {
	t.Helper()
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { master.Close(); slave.Close() })
	s := &testTerminal{t: t, master: master, slave: slave, done: make(chan error, 1)}
	s.resize(100, 30)
	s.original = s.settings()
	s.cmd = exec.Command(binary, args...)
	s.cmd.Stdin, s.cmd.Stdout, s.cmd.Stderr = slave, slave, slave
	s.cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := s.cmd.Start(); err != nil {
		t.Fatal(err)
	}
	go func() { s.done <- s.cmd.Wait(); close(s.done) }()
	t.Cleanup(func() {
		s.cmd.Process.Kill()
		select {
		case <-s.done:
		case <-time.After(3 * time.Second):
			t.Error("terminal process did not exit during cleanup")
		}
	})
	return s
}

func (s *testTerminal) settings() string {
	s.t.Helper()
	cmd := exec.Command("stty", "-g")
	cmd.Stdin = s.slave
	data, err := cmd.Output()
	if err != nil {
		s.t.Fatal(err)
	}
	return string(data)
}

func (s *testTerminal) resize(width, height uint16) {
	s.t.Helper()
	if err := pty.Setsize(s.slave, &pty.Winsize{Rows: height, Cols: width}); err != nil {
		s.t.Fatal(err)
	}
	if s.cmd != nil {
		if err := s.cmd.Process.Signal(syscall.SIGWINCH); err != nil {
			s.t.Fatal(err)
		}
	}
}

func (s *testTerminal) send(keys string) {
	s.t.Helper()
	if _, err := s.master.WriteString(keys); err != nil {
		s.t.Fatal(err)
	}
}

func (s *testTerminal) readFor(duration time.Duration) string {
	s.t.Helper()
	return s.read(duration, "")
}

func (s *testTerminal) until(marker string) string {
	s.t.Helper()
	return s.read(3*time.Second, marker)
}

func (s *testTerminal) read(duration time.Duration, marker string) string {
	s.t.Helper()
	var output bytes.Buffer
	buffer := make([]byte, 32768)
	deadline := time.Now().Add(duration)
	fds := []unix.PollFd{{Fd: int32(s.master.Fd()), Events: unix.POLLIN}}
	for time.Now().Before(deadline) {
		n, err := unix.Poll(fds, max(1, int(time.Until(deadline).Milliseconds())))
		if err == syscall.EINTR {
			continue
		}
		if err != nil {
			s.t.Fatal(err)
		}
		if n == 0 {
			continue
		}
		n, err = s.master.Read(buffer)
		if err != nil {
			s.t.Fatalf("read terminal: %v", err)
		}
		output.Write(buffer[:n])
		if marker != "" && bytes.Contains(output.Bytes(), []byte(marker)) {
			return output.String()
		}
	}
	if marker != "" {
		tail := output.Bytes()[max(0, output.Len()-512):]
		s.t.Fatalf("missing terminal output %q; final output: %q", marker, tail)
	}
	return output.String()
}

func (s *testTerminal) restored() {
	s.t.Helper()
	cleanup := s.until("\x1b[?1049l")
	select {
	case err := <-s.done:
		if err != nil {
			s.t.Fatalf("terminal exited with error: %v", err)
		}
	case <-time.After(3 * time.Second):
		s.t.Fatal("terminal did not quit")
	}
	terminalCheck(s.t, strings.Contains(cleanup, "\x1b[?25h"), "cursor must be restored")
	terminalCheck(s.t, strings.Contains(cleanup, "\x1b[<u"), "keyboard mode must be restored")
	terminalCheck(s.t, strings.Contains(cleanup, "\x1b[?1006l\x1b[?1003l"), "mouse reporting must be restored")
	terminalCheck(s.t, s.settings() == s.original, "terminal settings must be restored")
}

func terminalCheck(t *testing.T, condition bool, message string) {
	t.Helper()
	if !condition {
		t.Fatal(message)
	}
}

func treePixels(output string) string {
	frames := strings.Split(output, "\x1b[H")
	return strings.Map(func(r rune) rune {
		if r >= 0x2800 && r <= 0x28ff {
			return r
		}
		return -1
	}, frames[len(frames)-1])
}

var positionPattern = regexp.MustCompile(`  x (-?[\d.]+)  z (-?[\d.]+)  \|`)

func positions(output string) [][2]float64 {
	var result [][2]float64
	for _, match := range positionPattern.FindAllStringSubmatch(output, -1) {
		x, _ := strconv.ParseFloat(match[1], 64)
		z, _ := strconv.ParseFloat(match[2], 64)
		result = append(result, [2]float64{x, z})
	}
	return result
}

func lastPosition(t *testing.T, output string) [2]float64 {
	t.Helper()
	points := positions(output)
	if len(points) == 0 {
		t.Fatal("movement must update the displayed position")
	}
	return points[len(points)-1]
}

func TestTerminal(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "dither-cli")
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	t.Run("help and noninteractive input", func(t *testing.T) {
		output, err := exec.Command(binary, "-h").CombinedOutput()
		terminalCheck(t, err == nil && bytes.Contains(output, []byte("Usage: dither-cli")) && bytes.Contains(output, []byte(appTitle)), "help must describe the CLI")
		output, err = exec.Command(binary).CombinedOutput()
		terminalCheck(t, err != nil && bytes.Contains(output, []byte("interactive terminal")), "noninteractive input must produce a clear error")
	})
	t.Run("forest exploration", func(t *testing.T) {
		s := openTestTerminal(t, binary, "--mute")
		initial := s.until("steps 0")
		terminalCheck(t, strings.Contains(initial, appTitle) && strings.Contains(initial, "236;185;105m"), "world title and traveler must be visible")
		s.send("np\x1b[110;1u\x1b[112;1:3u")
		s.readFor(50 * time.Millisecond)
		s.send("d")
		s.until("x 0.6  z 0.0  |  steps 1  |  facing right")
		s.send("\x1b[A")
		s.until("x 0.6  z -0.6  |  steps 2  |  facing up")
		s.send("js")
		s.until("x 0.0  z 0.0  |  steps 4  |  facing down")
		var regions strings.Builder
		for step := 5; step < 65; step++ {
			s.send("d")
			regions.WriteString(s.until(fmt.Sprintf("steps %d  |", step)))
		}
		terminalCheck(t, strings.Contains(regions.String(), "river / wading"), "walking must reach and cross a river")
		terminalCheck(t, treePixels(regions.String()) != treePixels(initial), "walking must reveal new terrain")
		s.send(strings.Repeat("d", 180))
		s.until("steps 244  |")
		s.resize(20, 8)
		s.until("Resize to 40 x 14.")
		s.resize(100, 30)
		s.until("\x1b[30;1H")
		s.send("q")
		s.restored()
	})
	t.Run("responsive rendering", func(t *testing.T) {
		s := openTestTerminal(t, binary, "--mute")
		s.until("seed 7")
		s.resize(220, 90)
		initial := s.until("seed 7")
		idle := s.readFor(350 * time.Millisecond)
		frames := strings.Count(idle, "\x1b[H")
		terminalCheck(t, frames >= 15, "animation must use the 60 FPS cadence")
		terminalCheck(t, !strings.Contains(idle, "\x1b[2J"), "steady animation must not clear the screen")
		terminalCheck(t, len(idle)/frames < len(initial)/2, "idle frames must send incremental updates")
		s.send(strings.Repeat("d", 40))
		moved := s.until("steps 40  |")
		terminalCheck(t, strings.Count(moved, "\x1b[H") <= 3, "queued input must not trigger one redraw per key")
		s.send("q")
		s.restored()
	})
	t.Run("held movement", func(t *testing.T) {
		s := openTestTerminal(t, binary, "--mute")
		terminalCheck(t, strings.Contains(s.until("seed 7"), "\x1b[>11u\x1b[?u"), "request key press/release reporting")
		s.send("\x1b[?11u\x1b[1;1D\x1b[1;1A")
		start := time.Now()
		diagonal := lastPosition(t, s.readFor(350*time.Millisecond))
		speed := math.Hypot(diagonal[0], diagonal[1]) / (9 * time.Since(start).Seconds())
		terminalCheck(t, diagonal[0] < -1 && diagonal[1] < -1 && math.Abs(diagonal[0]-diagonal[1]) < 0.15, "held arrows must move diagonally without repeats")
		terminalCheck(t, speed > 0.8 && speed < 1.2, "diagonal speed must be normalized")
		s.send(strings.Repeat("\x1b[1;1:2A", 20) + "\x1b[1;1:3D")
		vertical := lastPosition(t, s.readFor(350*time.Millisecond))
		terminalCheck(t, math.Abs(vertical[0]-diagonal[0]) < 0.2, "releasing LEFT must stop horizontal movement")
		distance := diagonal[1] - vertical[1]
		terminalCheck(t, distance > 2 && distance < 4.3, "UP must stay held and repeats must not add impulses")
		s.send("\x1b[119;1u\x1b[1;1:3A")
		letter := lastPosition(t, s.readFor(200*time.Millisecond))
		terminalCheck(t, letter[1] < vertical[1]-1, "releasing UP must preserve held W")
		s.send("\x1b[119;1:")
		time.Sleep(10 * time.Millisecond)
		s.send("3u")
		s.readFor(100 * time.Millisecond)
		terminalCheck(t, len(positions(s.readFor(200*time.Millisecond))) == 0, "releasing movement keys must stop immediately")
		s.send("\x1b[100;1u")
		lastPosition(t, s.readFor(150*time.Millisecond))
		s.send("\x1b[O")
		s.readFor(100 * time.Millisecond)
		terminalCheck(t, len(positions(s.readFor(200*time.Millisecond))) == 0, "focus loss must stop movement")
		s.send("\x1b[113;1:3u")
		s.readFor(50 * time.Millisecond)
		s.send("\x1b[113;1u")
		s.restored()
	})
	t.Run("sprinting", func(t *testing.T) {
		s := openTestTerminal(t, binary, "--mute")
		s.until("seed 7")
		s.send("\x1b[?11u\x1b[119;1u")
		start := time.Now()
		walked := lastPosition(t, s.readFor(250*time.Millisecond))
		walkingSpeed := -walked[1] / time.Since(start).Seconds()
		s.send("\x1b[57441;2u")
		start = time.Now()
		ran := lastPosition(t, s.readFor(250*time.Millisecond))
		runningSpeed := (walked[1] - ran[1]) / time.Since(start).Seconds()
		terminalCheck(t, runningSpeed/walkingSpeed > 1.7 && runningSpeed/walkingSpeed < 2.3, "Shift must double speed")
		s.send("\x1b[57447;2u\x1b[57441;2:3u")
		bothShifts := lastPosition(t, s.readFor(200*time.Millisecond))
		terminalCheck(t, ran[1]-bothShifts[1] > 2.7, "releasing one Shift must preserve sprinting with the other")
		s.send("\x1b[57447;1:3u")
		// Drain frames drawn before the release arrived, so no sprinting frame is measured as walking.
		released := lastPosition(t, s.readFor(50*time.Millisecond))
		start = time.Now()
		resumed := lastPosition(t, s.readFor(250*time.Millisecond))
		resumedSpeed := (released[1] - resumed[1]) / time.Since(start).Seconds()
		terminalCheck(t, resumedSpeed/walkingSpeed > 0.8 && resumedSpeed/walkingSpeed < 1.2, "releasing Shift must restore walking speed")
		s.send("\x1b[119;1:3u")
		s.readFor(100 * time.Millisecond)
		terminalCheck(t, len(positions(s.readFor(150*time.Millisecond))) == 0, "releasing W must stop after sprinting")
		s.send("q")
		s.restored()
	})
	t.Run("mouse tracking", func(t *testing.T) {
		s := openTestTerminal(t, binary, "--mute")
		initial := s.until("steps 0")
		terminalCheck(t, strings.Contains(initial, "\x1b[?1003h\x1b[?1006h"), "request mouse motion reporting")
		// Motion to the top, then left of the hero, then a click and its release elsewhere.
		s.send("\x1b[<35;1;1M\x1b[<35;10;17M\x1b[<0;60;20M\x1b[<0;60;20m")
		s.readFor(100 * time.Millisecond)
		s.send("d")
		s.until("x 0.6  z 0.0  |  steps 1  |  facing left")
		s.send("q")
		s.restored()
	})
	t.Run("woodcutting", func(t *testing.T) {
		// Sound effects stay on without music, so every chop plays through the real speaker.
		s := openTestTerminal(t, binary, "--music-dir", filepath.Join(t.TempDir(), "missing"))
		s.until("logs 0")
		s.send("e")
		terminalCheck(t, !strings.Contains(s.readFor(200*time.Millisecond), "Chop down"), "E must not offer actions without a tree in front")
		// Seed 7 grows a normal tree at x 6.0, z 1.7; stop beside its trunk, facing it.
		s.send(strings.Repeat("s", 3) + strings.Repeat("d", 9))
		s.until("x 4.9  z 1.7  |  steps 12  |  facing right")
		s.send("e")
		s.until("1 Chop down")
		s.send("1")
		chopping := s.until("chopping")
		felled := s.until("logs 1")
		terminalCheck(t, !strings.Contains(chopping+felled, "logs 2"), "one normal tree must give one log")
		s.send("e")
		terminalCheck(t, !strings.Contains(s.readFor(time.Second), "Chop down"), "a stump must not offer chopping")
		s.send("q")
		s.restored()
	})
	t.Run("missing music folder", func(t *testing.T) {
		s := openTestTerminal(t, binary, "--music-dir", filepath.Join(t.TempDir(), "missing"))
		s.until("seed 7")
		s.send("npdq")
		s.restored()
	})
	for _, exit := range []struct{ name, keys string }{
		{"ctrl-c", "\x03"}, {"ctrl-c during escape", "\x1b[1;\x03"}, {"encoded ctrl-c", "\x1b[99;5u"}, {"sigterm", ""},
	} {
		t.Run(exit.name, func(t *testing.T) {
			s := openTestTerminal(t, binary, "--mute")
			s.until("\x1b[30;1H")
			if exit.keys != "" {
				s.send(exit.keys)
			} else if err := s.cmd.Process.Signal(syscall.SIGTERM); err != nil {
				t.Fatal(err)
			}
			s.restored()
		})
	}
}
