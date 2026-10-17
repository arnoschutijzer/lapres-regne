package progressfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"lapresregne/internal/world"
)

// File keeps progress as JSON at Path, along with the View the terminal last showed it from.
type File struct {
	Path string
	// View is read by Load and written by Save.
	View View
}

// View is the camera's zoom, in steps in from the forest view, and its yaw about the upright axis,
// in radians.
type View struct {
	Zoom int
	Yaw  float64
}

// version marks the file layout; a file of any other layout is refused.
const version = 1

// itemNames spell each item in the file, independent of how the domain names them.
var itemNames = map[world.Item]string{
	world.Logs: "logs", world.OakLogs: "oak logs", world.WillowLogs: "willow logs", world.YewLogs: "yew logs",
}

// skillNames spell each skill in the file, independent of how the domain names them.
var skillNames = map[string]string{world.Woodcutting: "woodcutting"}

type progressJSON struct {
	Version int    `json:"version"`
	Seed    uint64 `json:"seed"`
	// Map names the map the hero and stumps belong to; absent for the endless forest.
	Map     string               `json:"map,omitempty"`
	SavedAt time.Time            `json:"savedAt"`
	Hero    heroJSON             `json:"hero"`
	Skills  map[string]skillJSON `json:"skills"`
	Camera  cameraJSON           `json:"camera"`
	// Inventory lists the inventory slot by slot, naming each item and leaving "" for an empty slot.
	Inventory []string    `json:"inventory"`
	Stumps    []stumpJSON `json:"stumps"`
}

type heroJSON struct {
	X float64 `json:"x"`
	Z float64 `json:"z"`
	// Storey is absent while the hero stands on the ground floor.
	Storey int `json:"storey,omitempty"`
}

type cameraJSON struct {
	Zoom int     `json:"zoom"`
	Yaw  float64 `json:"yaw"`
}

type skillJSON struct {
	XP float64 `json:"xp"`
}

type stumpJSON struct {
	GX           int   `json:"gx"`
	GZ           int   `json:"gz"`
	RegrowMillis int64 `json:"regrowMillis"`
}

func (f *File) Load() (world.Snapshot, bool, error) {
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
		Seed: saved.Seed, Map: saved.Map, X: saved.Hero.X, Z: saved.Hero.Z, Storey: saved.Hero.Storey,
		SavedAt: saved.SavedAt,
	}
	f.View = View{Zoom: saved.Camera.Zoom, Yaw: saved.Camera.Yaw}
	for name, skill := range saved.Skills {
		known, ok := skillNamed(name)
		if !ok {
			return world.Snapshot{}, false, fmt.Errorf("read %s: unknown skill %q", f.Path, name)
		}
		if snapshot.Skills == nil {
			snapshot.Skills = make(map[string]float64)
		}
		snapshot.Skills[known] = skill.XP
	}
	for _, name := range saved.Inventory {
		if name == "" {
			snapshot.Inventory = append(snapshot.Inventory, world.Slot{})
			continue
		}
		item, known := itemNamed(name)
		if !known {
			return world.Snapshot{}, false, fmt.Errorf("read %s: unknown item %q", f.Path, name)
		}
		snapshot.Inventory = append(snapshot.Inventory, world.Slot{Item: item, Filled: true})
	}
	for _, stump := range saved.Stumps {
		snapshot.Stumps = append(snapshot.Stumps, world.Stump{GX: stump.GX, GZ: stump.GZ, Regrow: time.Duration(stump.RegrowMillis) * time.Millisecond})
	}
	return snapshot, true, nil
}

// Save writes a temporary file beside the save and renames it over the old one, so an
// interrupted save never leaves a half-written file behind.
func (f *File) Save(snapshot world.Snapshot) error {
	saved := progressJSON{
		Version: version, Seed: snapshot.Seed, Map: snapshot.Map, SavedAt: snapshot.SavedAt,
		Hero:      heroJSON{X: snapshot.X, Z: snapshot.Z, Storey: snapshot.Storey},
		Skills:    map[string]skillJSON{},
		Camera:    cameraJSON{Zoom: f.View.Zoom, Yaw: f.View.Yaw},
		Inventory: []string{},
		Stumps:    []stumpJSON{},
	}
	for skill, xp := range snapshot.Skills {
		name, known := skillNames[skill]
		if !known {
			return fmt.Errorf("save %s: no spelling for skill %q", f.Path, skill)
		}
		saved.Skills[name] = skillJSON{XP: xp}
	}
	// Empty slots are saved as "" so items keep their places; trailing empty slots are left off.
	last := -1
	for index, slot := range snapshot.Inventory {
		if slot.Filled {
			last = index
		}
	}
	for _, slot := range snapshot.Inventory[:last+1] {
		name := ""
		if slot.Filled {
			name = itemNames[slot.Item]
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

func skillNamed(name string) (string, bool) {
	for skill, spelled := range skillNames {
		if spelled == name {
			return skill, true
		}
	}
	return "", false
}

func itemNamed(name string) (world.Item, bool) {
	for item, spelled := range itemNames {
		if spelled == name {
			return item, true
		}
	}
	return 0, false
}
