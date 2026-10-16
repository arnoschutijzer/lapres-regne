package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"testing"
)

func TestPlaylistShuffle(t *testing.T) {
	playlist := &shuffledSongs{}
	last := ""
	for range 20 {
		seen := make(map[string]bool)
		for range len(forestSongs) {
			id := playlist.Next()
			if id == last || seen[id] {
				t.Fatalf("song %q repeats before completing the shuffle", id)
			}
			seen[id], last = true, id
		}
		for _, id := range forestSongs {
			if !seen[id] {
				t.Errorf("shuffle omitted %q", id)
			}
		}
	}
}

func TestSongHistoryNavigation(t *testing.T) {
	history := &songHistory{}
	first, second, third := history.Next(), history.Next(), history.Next()
	if history.Previous() != second || history.Previous() != first || history.Previous() != first {
		t.Fatal("previous must retrace history and stay on the first song at the start")
	}
	if history.Next() != second || history.Next() != third {
		t.Fatal("next must retrace forward history before selecting a new song")
	}
}

func TestManualTrackChangesCrossfade(t *testing.T) {
	index := -1
	stream, err := newMusicStream(func(direction int) (*musicTrack, error) {
		index = max(0, index+direction)
		tone := stereoTone{left: 1}
		if index%2 == 1 {
			tone = stereoTone{right: 1}
		}
		return &musicTrack{samples: tone, frames: 10000}, nil
	}, 4)
	if err != nil {
		t.Fatal(err)
	}
	read := func(frames int) []byte {
		t.Helper()
		pcm := make([]byte, frames*musicChannels*4)
		if _, err := io.ReadFull(stream, pcm); err != nil {
			t.Fatal(err)
		}
		return pcm
	}
	assertTone := func(pcm []byte, left, right float32) {
		t.Helper()
		for offset := 0; offset < len(pcm); offset += 8 {
			l := math.Float32frombits(binary.LittleEndian.Uint32(pcm[offset:]))
			r := math.Float32frombits(binary.LittleEndian.Uint32(pcm[offset+4:]))
			if l != left || r != right {
				t.Fatalf("wrong song after navigation: %f, %f", l, r)
			}
		}
	}
	assertTone(read(1024), 1, 0)
	stream.Skip(1)
	fade := read(4)
	for offset := 8; offset < len(fade); offset += 8 {
		l := math.Float32frombits(binary.LittleEndian.Uint32(fade[offset:]))
		r := math.Float32frombits(binary.LittleEndian.Uint32(fade[offset+4:]))
		if l <= 0 || r <= 0 {
			t.Fatal("manual skip must overlap the outgoing and incoming songs")
		}
	}
	assertTone(read(1024), 0, 1)
	stream.Skip(-1)
	read(4)
	assertTone(read(1024), 1, 0)

	// Preserve the current crossfade, then honor the latest queued request.
	stream.Skip(1)
	read(1)
	stream.Skip(1)
	stream.Skip(-1)
	read(3)
	read(4)
	assertTone(read(1024), 1, 0)
}

func TestEmbeddedSongsDecode(t *testing.T) {
	for _, id := range forestSongs {
		t.Run(id, func(t *testing.T) {
			track, err := openSong(id)
			if err != nil {
				t.Fatal(err)
			}
			if track.frames < 2*musicFadeFrames {
				t.Fatal("track is too short for the crossfade")
			}
			samples := make([]float32, musicSampleRate*musicChannels)
			if err := readSamples(track, samples); err != nil {
				t.Fatal(err)
			}
			peak := 0.0
			for _, sample := range samples {
				if math.IsNaN(float64(sample)) || math.IsInf(float64(sample), 0) {
					t.Fatal("invalid audio sample")
				}
				peak = math.Max(peak, math.Abs(float64(sample)))
			}
			if peak < 0.0001 {
				t.Fatal("track starts with silent audio")
			}
		})
	}
}

// Stereo fixtures isolate outgoing sound on the left and incoming on the right.
type stereoTone struct{ left, right float32 }

func (s stereoTone) Read(p []float32) (int, error) {
	for i := 0; i < len(p); i += musicChannels {
		p[i], p[i+1] = s.left, s.right
	}
	return len(p), nil
}

func crossfadeFixture(t *testing.T) *musicStream {
	t.Helper()
	next := 0
	stream, err := newMusicStream(func(_ int) (*musicTrack, error) {
		tone := stereoTone{left: 1}
		if next%2 == 1 {
			tone = stereoTone{right: 1}
		}
		next++
		return &musicTrack{samples: tone, frames: 12}, nil
	}, 4)
	if err != nil {
		t.Fatal(err)
	}
	return stream
}

func TestSoundtrackCrossfadesWithoutGaps(t *testing.T) {
	stream := crossfadeFixture(t)
	pcm := make([]byte, 20*musicChannels*4)
	if _, err := io.ReadFull(stream, pcm); err != nil {
		t.Fatal(err)
	}
	var previousLeft, previousRight float64
	for frame := range 20 {
		left := float64(math.Float32frombits(binary.LittleEndian.Uint32(pcm[frame*8:])))
		right := float64(math.Float32frombits(binary.LittleEndian.Uint32(pcm[frame*8+4:])))
		if math.Abs(left*left+right*right-1) > 0.00001 {
			t.Fatalf("audible power dips or spikes at frame %d: %f, %f", frame, left, right)
		}
		if frame < 8 && (left != 1 || right != 0) {
			t.Fatal("incoming song begins before the crossfade")
		}
		if frame > 8 && frame < 12 && (left >= previousLeft || right <= previousRight) {
			t.Fatal("outgoing song must fade down while incoming song fades up")
		}
		if frame >= 12 && frame < 16 && (left != 0 || right != 1) {
			t.Fatal("incoming song must continue after the crossfade")
		}
		if frame > 0 && (math.Abs(left-previousLeft) > 0.4 || math.Abs(right-previousRight) > 0.4) {
			t.Fatal("discontinuous sample at the track transition")
		}
		previousLeft, previousRight = left, right
	}

	// Audio backends may pull bytes in sizes that split sample boundaries.
	fragmented := make([]byte, len(pcm))
	other := crossfadeFixture(t)
	for i := range fragmented {
		if _, err := io.ReadFull(other, fragmented[i:i+1]); err != nil {
			t.Fatal(err)
		}
	}
	if !bytes.Equal(pcm, fragmented) {
		t.Fatal("audio changes with backend read sizes")
	}
}

func TestSoundtrackTransitionsThroughEmbeddedSongs(t *testing.T) {
	selected := 0
	stream, err := newMusicStream(func(_ int) (*musicTrack, error) {
		track, err := openSong(forestSongs[selected%len(forestSongs)])
		if err != nil {
			return nil, err
		}
		selected++
		// Use the actual end of each song to exercise decoder EOFs quickly.
		tailFrames := int64(2*musicFadeFrames + musicSampleRate/10)
		seeker := track.samples.(interface{ SetPosition(int64) error })
		if err := seeker.SetPosition(track.frames - tailFrames); err != nil {
			return nil, err
		}
		track.frames = tailFrames
		return track, nil
	}, musicFadeFrames)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.CopyN(io.Discard, stream, 20*musicSampleRate*musicChannels*4); err != nil {
		t.Fatal(err)
	}
	if selected < len(forestSongs)+1 {
		t.Fatal("soundtrack did not continue into a second playlist cycle")
	}
}
