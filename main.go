package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"lapresregne/internal/adapters/mapfile"
	"lapresregne/internal/adapters/progressfile"
	"lapresregne/internal/adapters/songfiles"
	"lapresregne/internal/adapters/speaker"
	"lapresregne/internal/adapters/terminal"
	"lapresregne/internal/adapters/viewfile"
	"lapresregne/internal/soundtrack"
	"lapresregne/internal/world"
)

const appTitle = "L’Après-Règne"

func main() {
	seed := flag.Uint64("seed", 7, "procedural forest seed")
	mapPath := flag.String("map", "", "map file to explore (default the endless forest)")
	mute := flag.Bool("mute", false, "disable the soundtrack and sound effects")
	musicDir := flag.String("music-dir", "music", "folder containing music (missing or empty means no soundtrack)")
	savePath := flag.String("save", "", "file keeping progress between sessions (default in the user config folder)")
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), appTitle+"\n\nUsage: lapresregne [--seed number] [--map file] [--mute] [--save file]\n\nWalk through an endless dithered forest, or a hand-made map.\n\n  Arrows / WASD / IJKL   walk (I up, J left, K down, L right)\n  Shift                 run at twice the walking speed\n  Click                 walk to the ground clicked, to a tree to chop it, to a door to open or close it, or to stairs to climb them\n  E                     act on a tree, door, staircase or ladder within reach (1 or 2 picks the action)\n  Tab                   show or hide the inventory\n  [ / ]                 highlight the previous / next inventory slot (or click one)\n  X                     drop the highlighted log, or the first one (deposit it while the bank is open)\n  B                     open or close the bank, and click an inventory slot to deposit its item or a bank stack to withdraw one\n  Shift-[ / Shift-]     highlight the previous / next bank stack (opens the bank)\n  Shift-X               withdraw one item from the highlighted bank stack, or the first one\n  N / P                 next / previous music track\n  Q / Ctrl-C            quit")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}
	if *savePath == "" {
		config, err := os.UserConfigDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, "lapresregne: find a place to save progress (use --save):", err)
			os.Exit(1)
		}
		*savePath = filepath.Join(config, "lapresregne", "progress.json")
	}
	if err := run(*seed, *mapPath, *mute, *musicDir, *savePath); err != nil {
		fmt.Fprintln(os.Stderr, "lapresregne:", err)
		os.Exit(1)
	}
}

func startMusic(device *speaker.Device, directory string) (*speaker.Playback, error) {
	songs, err := songfiles.List(directory)
	if err != nil || len(songs) == 0 {
		return nil, err
	}
	stream, err := soundtrack.Shuffle(songs, songfiles.Decoder{})
	if err != nil {
		return nil, err
	}
	return device.Play(stream), nil
}

// run is the composition root: it builds every adapter and wires it to the domain.
func run(seed uint64, mapPath string, mute bool, musicDir, savePath string) error {
	progress := progressfile.File{Path: savePath}
	// The view is kept beside the progress, in a file of its own.
	views := viewfile.File{Path: filepath.Join(filepath.Dir(savePath), "view.json")}
	view, err := views.Load()
	if err != nil {
		return fmt.Errorf("%w (move or remove it to start afresh)", err)
	}
	var forest *world.World
	if mapPath == "" {
		forest, err = world.Resume(seed, progress, time.Now())
	} else {
		explored, loadErr := mapfile.Load(mapPath)
		if loadErr != nil {
			return loadErr
		}
		forest, err = world.ResumeOn(explored, seed, progress, time.Now())
	}
	if err != nil {
		return fmt.Errorf("%w (move or remove %s to start afresh)", err, savePath)
	}
	session, err := terminal.Open()
	if err != nil {
		return err
	}
	defer session.Close()
	var music soundtrack.Player
	if !mute {
		device, err := speaker.Open()
		if err != nil {
			return fmt.Errorf("open audio device (use --mute for silent play): %w", err)
		}
		defer device.Close()
		forest.SetSounds(device.Effects())
		playback, err := startMusic(device, musicDir)
		if err != nil {
			return fmt.Errorf("start soundtrack (use --mute for silent play): %w", err)
		}
		if playback != nil {
			defer playback.Close()
			music = playback
		}
	}
	camera := terminal.NewCamera(view.Zoom, view.Yaw)
	// Progress and the view are kept however the session ends, even when it ends with an error.
	played := session.Run(forest, camera, music, appTitle)
	kept := views.Save(viewfile.View{Zoom: camera.Zoom(), Yaw: camera.Yaw()})
	return errors.Join(played, forest.Keep(progress, time.Now()), kept)
}
