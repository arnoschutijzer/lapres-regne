package progressfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"dither-cli/internal/world"
)

// File keeps progress as JSON at Path.
type File struct {
	Path string
}

// version marks the file layout so later layouts can still read older saves. Version 1
// counted logs; version 2 lists the inventory slot by slot, naming each log's tree and
// leaving "" for an empty slot.
const version = 2

// logNames spell each kind of log in the file, independent of how the domain names them.
var logNames = map[world.TreeKind]string{
	world.NormalTree: "normal", world.Oak: "oak", world.Willow: "willow", world.Yew: "yew",
}

type progressJSON struct {
	Version     int       `json:"version"`
	Seed        uint64    `json:"seed"`
	SavedAt     time.Time `json:"savedAt"`
	Hero        heroJSON  `json:"hero"`
	Woodcutting skillJSON `json:"woodcutting"`
	// Logs is the log count of version 1 saves.
	Logs      int         `json:"logs,omitempty"`
	Inventory []string    `json:"inventory"`
	Stumps    []stumpJSON `json:"stumps"`
}

type heroJSON struct {
	X float64 `json:"x"`
	Z float64 `json:"z"`
}

type skillJSON struct {
	XP float64 `json:"xp"`
}

type stumpJSON struct {
	GX           int   `json:"gx"`
	GZ           int   `json:"gz"`
	RegrowMillis int64 `json:"regrowMillis"`
}

func (f File) Load() (world.Snapshot, bool, error) {
	data, err := os.ReadFile(f.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return world.Snapshot{}, false, nil
	}
	if err != nil {
		return world.Snapshot{}, false, err
	}
	var saved progressJSON
	if err := json.Unmarshal(data, &saved); err != nil {
		return world.Snapshot{}, false, fmt.Errorf("read %s: %w", f.Path, err)
	}
	if saved.Version != 1 && saved.Version != version {
		return world.Snapshot{}, false, fmt.Errorf("read %s: unknown version %d", f.Path, saved.Version)
	}
	snapshot := world.Snapshot{
		Seed: saved.Seed, X: saved.Hero.X, Z: saved.Hero.Z,
		WoodcuttingXP: saved.Woodcutting.XP, SavedAt: saved.SavedAt,
	}
	// Version 1 counted logs before they had kinds; they were all normal logs.
	for range saved.Logs {
		snapshot.Inventory = append(snapshot.Inventory, world.Slot{Kind: world.NormalTree, Filled: true})
	}
	for _, name := range saved.Inventory {
		if name == "" {
			snapshot.Inventory = append(snapshot.Inventory, world.Slot{})
			continue
		}
		kind, known := kindNamed(name)
		if !known {
			return world.Snapshot{}, false, fmt.Errorf("read %s: unknown log %q", f.Path, name)
		}
		snapshot.Inventory = append(snapshot.Inventory, world.Slot{Kind: kind, Filled: true})
	}
	for _, stump := range saved.Stumps {
		snapshot.Stumps = append(snapshot.Stumps, world.Stump{GX: stump.GX, GZ: stump.GZ, Regrow: time.Duration(stump.RegrowMillis) * time.Millisecond})
	}
	return snapshot, true, nil
}

// Save writes a temporary file beside the save and renames it over the old one, so an
// interrupted save never leaves a half-written file behind.
func (f File) Save(snapshot world.Snapshot) error {
	saved := progressJSON{
		Version: version, Seed: snapshot.Seed, SavedAt: snapshot.SavedAt,
		Hero:        heroJSON{X: snapshot.X, Z: snapshot.Z},
		Woodcutting: skillJSON{XP: snapshot.WoodcuttingXP},
		Inventory:   []string{},
		Stumps:      []stumpJSON{},
	}
	// Empty slots are saved as "" so logs keep their places; trailing empty slots are left off.
	last := -1
	for index, slot := range snapshot.Inventory {
		if slot.Filled {
			last = index
		}
	}
	for _, slot := range snapshot.Inventory[:last+1] {
		name := ""
		if slot.Filled {
			name = logNames[slot.Kind]
		}
		saved.Inventory = append(saved.Inventory, name)
	}
	for _, stump := range snapshot.Stumps {
		saved.Stumps = append(saved.Stumps, stumpJSON{GX: stump.GX, GZ: stump.GZ, RegrowMillis: stump.Regrow.Milliseconds()})
	}
	data, err := json.MarshalIndent(saved, "", "  ")
	if err != nil {
		return err
	}
	directory := filepath.Dir(f.Path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".progress-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(temporary.Name())
	if _, err := temporary.Write(append(data, '\n')); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporary.Name(), f.Path)
}

func kindNamed(name string) (world.TreeKind, bool) {
	for kind, spelled := range logNames {
		if spelled == name {
			return kind, true
		}
	}
	return 0, false
}
