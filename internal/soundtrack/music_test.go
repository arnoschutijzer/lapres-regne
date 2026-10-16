package soundtrack

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"testing"
)

func TestPlaylistShuffle(t *testing.T) {
	songs := []string{"a", "b", "c", "d", "e"}
	playlist := &shuffledSongs{songs: songs}
	last := ""
	for range 20 {
		seen := make(map[string]bool)
		for range len(songs) {
			id := playlist.Next()
			if id == last || seen[id] {
				t.Fatalf("song %q repeats before completing the shuffle", id)
			}
			seen[id], last = true, id
		}
		for _, id := range songs {
			if !seen[id] {
				t.Errorf("shuffle omitted %q", id)
			}
		}
	}
}

func TestSingleSongLoops(t *testing.T) {
	playlist := &shuffledSongs{songs: []string{"only"}}
	for range 3 {
		if playlist.Next() != "only" {
			t.Fatal("single-track folders should loop")
		}
	}
}

func TestSongHistoryNavigation(t *testing.T) {
	history := &songHistory{shuffle: shuffledSongs{songs: []string{"a", "b", "c"}}}
	first, second, third := history.Next(), history.Next(), history.Next()
	if history.Previous() != second || history.Previous() != first || history.Previous() != first {
		t.Fatal("previous must retrace history and stay on the first song at the start")
	}
	if history.Next() != second || history.Next() != third {
		t.Fatal("next must retrace forward history before selecting a new song")
	}
}

// Stereo fixtures make any overlap between different songs directly observable.
type stereoTone struct{ left, right float32 }

func (s stereoTone) Read(p []float32) (int, error) {
	for i := 0; i < len(p); i += Channels {
		p[i], p[i+1] = s.left, s.right
	}
	return len(p), nil
}

func readMusic(t *testing.T, stream io.Reader, frames int) []byte {
	t.Helper()
	pcm := make([]byte, frames*Channels*4)
	if _, err := io.ReadFull(stream, pcm); err != nil {
		t.Fatal(err)
	}
	return pcm
}

func sampleAt(pcm []byte, frame int) (float32, float32) {
	return math.Float32frombits(binary.LittleEndian.Uint32(pcm[frame*8:])),
		math.Float32frombits(binary.LittleEndian.Uint32(pcm[frame*8+4:]))
}

func musicFixture(t *testing.T, frames int64) *Stream {
	t.Helper()
	index := -1
	stream, err := newStream(func(direction int) (*Track, error) {
		index = max(0, index+direction)
		tone := stereoTone{left: 1}
		if index%2 == 1 {
			tone = stereoTone{right: 1}
		}
		return &Track{Samples: tone, Frames: frames}, nil
	}, 4, 2)
	if err != nil {
		t.Fatal(err)
	}
	return stream
}

func TestSongsFadeThroughSilenceWithoutOverlap(t *testing.T) {
	pcm := readMusic(t, musicFixture(t, 12), 28)
	for frame := range 28 {
		left, right := sampleAt(pcm, frame)
		if left != 0 && right != 0 {
			t.Fatalf("songs overlap at frame %d", frame)
		}
		if frame < 12 && right != 0 {
			t.Fatal("next song starts before outgoing fade completes")
		}
		if frame >= 14 && left != 0 {
			t.Fatal("outgoing song continues after the silence")
		}
		if frame >= 11 && frame <= 13 || frame >= 25 {
			if left != 0 || right != 0 {
				t.Fatalf("transition must reach actual silence at frame %d", frame)
			}
		}
		if frame > 0 && frame < 4 {
			previous, _ := sampleAt(pcm, frame-1)
			if left <= previous {
				t.Fatal("first song must fade in")
			}
		}
		if frame > 8 && frame < 12 {
			previous, _ := sampleAt(pcm, frame-1)
			if left >= previous {
				t.Fatal("outgoing song must fade out")
			}
		}
		if frame >= 14 && frame < 18 {
			_, previous := sampleAt(pcm, frame-1)
			if right <= previous {
				t.Fatal("next song must fade in after the silence")
			}
		}
	}
	fragmented := make([]byte, len(pcm))
	other := musicFixture(t, 12)
	for i := range fragmented {
		if _, err := io.ReadFull(other, fragmented[i:i+1]); err != nil {
			t.Fatal(err)
		}
	}
	if !bytes.Equal(pcm, fragmented) {
		t.Fatal("audio changes with backend read sizes")
	}
}

func TestManualNavigationFadesThroughSilence(t *testing.T) {
	stream := musicFixture(t, 10000)
	assertTone := func(pcm []byte, left, right float32) {
		t.Helper()
		for frame := range len(pcm) / 8 {
			l, r := sampleAt(pcm, frame)
			if l != left || r != right {
				t.Fatalf("wrong song: %f, %f", l, r)
			}
		}
	}
	readMusic(t, stream, 4)
	assertTone(readMusic(t, stream, 1024), 1, 0)
	for _, direction := range []int{1, -1} {
		stream.Skip(direction)
		out := readMusic(t, stream, 4)
		left, right := sampleAt(out, 3)
		if left != 0 || right != 0 {
			t.Fatal("manual navigation must fade to zero")
		}
		assertTone(readMusic(t, stream, 2), 0, 0)
		readMusic(t, stream, 4)
		if direction > 0 {
			assertTone(readMusic(t, stream, 1024), 0, 1)
		} else {
			assertTone(readMusic(t, stream, 1024), 1, 0)
		}
	}
	stream.Skip(1)
	readMusic(t, stream, 1)
	stream.Skip(1)
	stream.Skip(-1)
	readMusic(t, stream, 3)
	assertTone(readMusic(t, stream, 2), 0, 0)
	readMusic(t, stream, 4)
	assertTone(readMusic(t, stream, 1024), 1, 0)
}

func TestVeryShortTracksLoop(t *testing.T) {
	readMusic(t, musicFixture(t, 2), 100)
}

func TestNavigationWhileStreaming(t *testing.T) {
	stream := musicFixture(t, 10000)
	done := make(chan error, 1)
	go func() {
		_, err := io.CopyN(io.Discard, stream, 100000)
		done <- err
	}()
	for i := range 1000 {
		stream.Skip(1 - 2*(i%2))
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
