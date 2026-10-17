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

// File keeps progress as JSON at Path.
type File struct {
	Path string
}

// version marks the file layout so later layouts can still read older saves. Version 1
// counted logs; version 2 lists the inventory slot by slot, naming each item and leaving ""
// for an empty slot; version 3 keeps the experience of every skill by name.
const version = 3

// itemNames spell each item in the file, independent of how the domain names them. Logs keep
// the names of their trees from before items were generic.
var itemNames = map[world.Item]string{
	world.Logs: "normal", world.OakLogs: "oak", world.WillowLogs: "willow", world.YewLogs: "yew",
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
	// Woodcutting is the woodcutting experience of version 1 and 2 saves.
	Woodcutting *skillJSON `json:"woodcutting,omitempty"`
	// Camera is absent from saves made before the view could zoom, which open on the forest view.
	Camera cameraJSON `json:"camera"`
	// Logs is the log count of version 1 saves.
	Logs      int         `json:"logs,omitempty"`
	Inventory []string    `json:"inventory"`
	Stumps    []stumpJSON `json:"stumps"`
}

type heroJSON struct {
	X float64 `json:"x"`
	Z float64 `json:"z"`
	// Storey is absent while the hero stands on the ground floor, as in saves made before maps had
	// floors.
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
	if saved.Version < 1 || saved.Version > version {
		return world.Snapshot{}, false, fmt.Errorf("read %s: unknown version %d", f.Path, saved.Version)
	}
	snapshot := world.Snapshot{
		Seed: saved.Seed, Map: saved.Map, X: saved.Hero.X, Z: saved.Hero.Z, Storey: saved.Hero.Storey,
		Zoom: saved.Camera.Zoom, Yaw: saved.Camera.Yaw, SavedAt: saved.SavedAt,
	}
	// Versions 1 and 2 kept woodcutting, the only skill then, on its own.
	if saved.Woodcutting != nil {
		snapshot.Skills = map[string]float64{world.Woodcutting: saved.Woodcutting.XP}
	}
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
	// Version 1 counted logs before they had kinds; they were all normal logs.
	for range saved.Logs {
		snapshot.Inventory = append(snapshot.Inventory, world.Slot{Item: world.Logs, Filled: true})
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
func (f File) Save(snapshot world.Snapshot) error {
	saved := progressJSON{
		Version: version, Seed: snapshot.Seed, Map: snapshot.Map, SavedAt: snapshot.SavedAt,
		Hero:      heroJSON{X: snapshot.X, Z: snapshot.Z, Storey: snapshot.Storey},
		Skills:    map[string]skillJSON{},
		Camera:    cameraJSON{Zoom: snapshot.Zoom, Yaw: snapshot.Yaw},
		Inventory: []string{},
		Stumps:    []stumpJSON{},
	}
	for skill, xp := range snapshot.Skills {
		saved.Skills[skillNames[skill]] = skillJSON{XP: xp}
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
