package main

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func writeWAV(t *testing.T, path string) {
	t.Helper()
	const rate, frames = 22050, 2205
	data := make([]byte, 44+frames*2)
	copy(data, "RIFF")
	binary.LittleEndian.PutUint32(data[4:], uint32(len(data)-8))
	copy(data[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(data[16:], 16)
	binary.LittleEndian.PutUint16(data[20:], 1)
	binary.LittleEndian.PutUint16(data[22:], 1)
	binary.LittleEndian.PutUint32(data[24:], rate)
	binary.LittleEndian.PutUint32(data[28:], rate*2)
	binary.LittleEndian.PutUint16(data[32:], 2)
	binary.LittleEndian.PutUint16(data[34:], 16)
	copy(data[36:], "data")
	binary.LittleEndian.PutUint32(data[40:], frames*2)
	for frame := range frames {
		sample := int16(math.Sin(float64(frame)*2*math.Pi*440/rate) * 12000)
		binary.LittleEndian.PutUint16(data[44+frame*2:], uint16(sample))
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestMusicFolder(t *testing.T) {
	directory := t.TempDir()
	for _, folder := range []string{directory, filepath.Join(directory, "missing")} {
		playback, err := startMusic(folder)
		if err != nil || playback != nil {
			t.Fatalf("missing or empty folder should be silent: %v", err)
		}
	}
	writeWAV(t, filepath.Join(directory, "My song.WAV"))
	if err := os.WriteFile(filepath.Join(directory, "notes.txt"), []byte("not audio"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(directory, "nested.wav"), 0700); err != nil {
		t.Fatal(err)
	}
	songs, err := songsInFolder(directory)
	if err != nil || len(songs) != 1 || filepath.Base(songs[0]) != "My song.WAV" {
		t.Fatalf("discover user audio files only: %v, %v", songs, err)
	}
	playlist := &shuffledSongs{songs: songs}
	for range 3 {
		if playlist.Next() != songs[0] {
			t.Fatal("single-track folders should loop")
		}
	}
}

func TestUserSongFormats(t *testing.T) {
	wav := filepath.Join(t.TempDir(), "tone.wav")
	writeWAV(t, wav)
	for _, path := range []string{wav, "testdata/music/tone.mp3", "testdata/music/tone.ogg", "testdata/music/tone.flac"} {
		t.Run(filepath.Ext(path), func(t *testing.T) {
			track, err := openSong(path)
			if err != nil {
				t.Fatal(err)
			}
			defer track.Close()
			samples := make([]float32, track.frames*musicChannels)
			if err := readSamples(track, samples); err != nil {
				t.Fatal(err)
			}
			peak := 0.0
			for _, sample := range samples {
				if math.IsNaN(float64(sample)) || math.IsInf(float64(sample), 0) {
					t.Fatal("invalid decoded sample")
				}
				peak = math.Max(peak, math.Abs(float64(sample)))
			}
			if peak < 0.0001 {
				t.Fatal("decoded song is silent")
			}
		})
	}
}
