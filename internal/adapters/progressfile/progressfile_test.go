package progressfile

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"lapresregne/internal/world"
)

func TestSavedProgressLoadsBackUnchanged(t *testing.T) {
	file := File{Path: filepath.Join(t.TempDir(), "nested", "progress.json")}
	snapshot := world.Snapshot{
		Seed: 7, X: 4.95, Z: -1.65, WoodcuttingXP: 112.5,
		Inventory: []world.Slot{{Kind: world.Oak, Filled: true}, {}, {Kind: world.Willow, Filled: true}, {Kind: world.Yew, Filled: true}},
		Stumps:    []world.Stump{{GX: 1, GZ: 0, Regrow: 41*time.Second + 250*time.Millisecond}, {GX: -2, GZ: 3, Regrow: 8400 * time.Millisecond}},
		SavedAt:   time.Date(2026, 10, 16, 21, 0, 0, 0, time.UTC),
	}
	if err := file.Save(snapshot); err != nil {
		t.Fatal(err)
	}
	loaded, saved, err := file.Load()
	if err != nil || !saved {
		t.Fatalf("load = %v, %v; want the saved snapshot", saved, err)
	}
	if !loaded.SavedAt.Equal(snapshot.SavedAt) {
		t.Fatalf("saved at = %v, want %v", loaded.SavedAt, snapshot.SavedAt)
	}
	loaded.SavedAt = snapshot.SavedAt
	if !reflect.DeepEqual(loaded, snapshot) {
		t.Fatalf("loaded = %+v, want %+v", loaded, snapshot)
	}
}

func TestSavingReplacesTheFileWithoutLeavingTemporaryFiles(t *testing.T) {
	directory := t.TempDir()
	file := File{Path: filepath.Join(directory, "progress.json")}
	file.Save(world.Snapshot{Seed: 1})
	if err := file.Save(world.Snapshot{Seed: 2}); err != nil {
		t.Fatal(err)
	}
	if loaded, _, _ := file.Load(); loaded.Seed != 2 {
		t.Fatalf("seed = %d, want the latest save", loaded.Seed)
	}
	entries, _ := os.ReadDir(directory)
	if len(entries) != 1 {
		t.Fatalf("files = %v, want only progress.json", entries)
	}
}

func TestAMissingFileMeansNothingWasSaved(t *testing.T) {
	file := File{Path: filepath.Join(t.TempDir(), "progress.json")}
	if _, saved, err := file.Load(); saved || err != nil {
		t.Fatalf("load = %v, %v; want nothing saved and no error", saved, err)
	}
}

func TestAnUnreadableFileIsReportedAndKept(t *testing.T) {
	path := filepath.Join(t.TempDir(), "progress.json")
	os.WriteFile(path, []byte("{not json"), 0o600)
	file := File{Path: path}
	if _, _, err := file.Load(); err == nil {
		t.Fatal("a corrupt file must be reported")
	}
	if data, _ := os.ReadFile(path); string(data) != "{not json" {
		t.Fatal("loading must never change the file")
	}
}

func TestAVersionOneSaveCarriesItsLogCountAsNormalLogs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "progress.json")
	os.WriteFile(path, []byte(`{"version": 1, "seed": 7, "savedAt": "2026-10-16T21:00:00Z", "hero": {"x": 2.75, "z": 0},
		"woodcutting": {"xp": 50}, "logs": 2, "stumps": [{"gx": 1, "gz": 0, "regrowMillis": 4000}]}`), 0o600)
	loaded, saved, err := File{Path: path}.Load()
	if err != nil || !saved {
		t.Fatalf("load = %v, %v; want the old save", saved, err)
	}
	want := []world.Slot{{Kind: world.NormalTree, Filled: true}, {Kind: world.NormalTree, Filled: true}}
	if !reflect.DeepEqual(loaded.Inventory, want) || loaded.WoodcuttingXP != 50 || loaded.X != 2.75 || len(loaded.Stumps) != 1 {
		t.Fatalf("loaded = %+v, want two normal logs and the rest of the old save", loaded)
	}
}

func TestAnUnknownLogIsReported(t *testing.T) {
	path := filepath.Join(t.TempDir(), "progress.json")
	os.WriteFile(path, []byte(`{"version": 2, "seed": 7, "inventory": ["oak", "magic"]}`), 0o600)
	if _, _, err := (File{Path: path}).Load(); err == nil {
		t.Fatal("an unknown log must be reported")
	}
}
