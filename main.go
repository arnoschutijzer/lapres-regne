package main

import (
	"flag"
	"fmt"
	"os"

	"dither-cli/internal/adapters/songfiles"
	"dither-cli/internal/adapters/speaker"
	"dither-cli/internal/adapters/terminal"
	"dither-cli/internal/soundtrack"
	"dither-cli/internal/world"
)

const appTitle = "L’Après-Règne"

func main() {
	seed := flag.Uint64("seed", 7, "procedural forest seed")
	mute := flag.Bool("mute", false, "disable the soundtrack")
	musicDir := flag.String("music-dir", "music", "folder containing music (missing or empty means no soundtrack)")
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), appTitle+"\n\nUsage: dither-cli [--seed number] [--mute]\n\nWalk through an endless dithered forest.\n\n  Arrows / WASD / IJKL   walk (I up, J left, K down, L right)\n  Shift                 run at twice the walking speed\n  E                     act on the tree in front (1 chops it down)\n  N / P                 next / previous music track\n  Q / Ctrl-C            quit")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(*seed, *mute, *musicDir); err != nil {
		fmt.Fprintln(os.Stderr, "dither-cli:", err)
		os.Exit(1)
	}
}

func startMusic(directory string) (*speaker.Playback, error) {
	songs, err := songfiles.List(directory)
	if err != nil || len(songs) == 0 {
		return nil, err
	}
	stream, err := soundtrack.Shuffle(songs, songfiles.Decoder{})
	if err != nil {
		return nil, err
	}
	return speaker.Start(stream)
}

// run is the composition root: it builds every adapter and wires it to the domain.
func run(seed uint64, mute bool, musicDir string) error {
	session, err := terminal.Open()
	if err != nil {
		return err
	}
	defer session.Close()
	var music soundtrack.Player
	if !mute {
		playback, err := startMusic(musicDir)
		if err != nil {
			return fmt.Errorf("start soundtrack (use --mute for silent play): %w", err)
		}
		if playback != nil {
			defer playback.Close()
			music = playback
		}
	}
	return session.Run(world.New(seed), music, appTitle)
}
