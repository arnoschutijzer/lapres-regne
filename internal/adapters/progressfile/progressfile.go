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

// version marks the file layout so later layouts can still read older saves.
const version = 1

type progressJSON struct {
	Version     int         `json:"version"`
	Seed        uint64      `json:"seed"`
	SavedAt     time.Time   `json:"savedAt"`
	Hero        heroJSON    `json:"hero"`
	Woodcutting skillJSON   `json:"woodcutting"`
	Logs        int         `json:"logs"`
	Stumps      []stumpJSON `json:"stumps"`
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
	if saved.Version != version {
		return world.Snapshot{}, false, fmt.Errorf("read %s: unknown version %d", f.Path, saved.Version)
	}
	snapshot := world.Snapshot{
		Seed: saved.Seed, X: saved.Hero.X, Z: saved.Hero.Z,
		WoodcuttingXP: saved.Woodcutting.XP, Logs: saved.Logs, SavedAt: saved.SavedAt,
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
		Logs:        snapshot.Logs,
		Stumps:      []stumpJSON{},
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
