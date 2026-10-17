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
	path := filepath.Join(t.TempDir(), "nested", "progress.json")
	file := &File{Path: path, View: View{Zoom: 5, Yaw: -0.375}}
	snapshot := world.Snapshot{
		Seed: 7, Map: "Lumbridge", X: 4.95, Z: -1.65, Storey: 2, Skills: map[string]float64{world.Woodcutting: 112.5},
		Inventory: []world.Slot{{Item: world.OakLogs, Filled: true}, {}, {Item: world.WillowLogs, Filled: true}, {Item: world.YewLogs, Filled: true}},
		Stumps:    []world.Stump{{GX: 1, GZ: 0, Regrow: 41*time.Second + 250*time.Millisecond}, {GX: -2, GZ: 3, Regrow: 8400 * time.Millisecond}},
		SavedAt:   time.Date(2026, 10, 16, 21, 0, 0, 0, time.UTC),
	}
	if err := file.Save(snapshot); err != nil {
		t.Fatal(err)
	}
	reopened := &File{Path: path}
	loaded, saved, err := reopened.Load()
	if err != nil || !saved {
		t.Fatalf("load = %v, %v; want the saved snapshot", saved, err)
	}
	if reopened.View != file.View {
		t.Fatalf("view = %+v, want the kept %+v", reopened.View, file.View)
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
	file := &File{Path: filepath.Join(directory, "progress.json")}
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
	file := &File{Path: filepath.Join(t.TempDir(), "progress.json")}
	if _, saved, err := file.Load(); saved || err != nil {
		t.Fatalf("load = %v, %v; want nothing saved and no error", saved, err)
	}
}

func TestAnUnreadableFileIsReportedAndKept(t *testing.T) {
	path := filepath.Join(t.TempDir(), "progress.json")
	os.WriteFile(path, []byte("{not json"), 0o600)
	file := &File{Path: path}
	if _, _, err := file.Load(); err == nil {
		t.Fatal("a corrupt file must be reported")
	}
	if data, _ := os.ReadFile(path); string(data) != "{not json" {
		t.Fatal("loading must never change the file")
	}
}

func TestAnUnknownItemIsReported(t *testing.T) {
	path := filepath.Join(t.TempDir(), "progress.json")
	os.WriteFile(path, []byte(`{"version": 1, "seed": 7, "inventory": ["oak logs", "magic logs"]}`), 0o600)
	if _, _, err := (&File{Path: path}).Load(); err == nil {
		t.Fatal("an unknown item must be reported")
	}
}

func TestAnUnknownSkillIsReported(t *testing.T) {
	path := filepath.Join(t.TempDir(), "progress.json")
	os.WriteFile(path, []byte(`{"version": 1, "seed": 7, "skills": {"woodcutting": {"xp": 50}, "juggling": {"xp": 10}}}`), 0o600)
	if _, _, err := (&File{Path: path}).Load(); err == nil {
		t.Fatal("an unknown skill must be reported")
	}
}

func TestASkillWithoutASpellingIsRefusedRatherThanSaved(t *testing.T) {
	path := filepath.Join(t.TempDir(), "progress.json")
	file := &File{Path: path}
	if err := file.Save(world.Snapshot{Seed: 7, Skills: map[string]float64{"Juggling": 10}}); err == nil {
		t.Fatal("a skill the file cannot spell must be reported")
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatal("a refused save must leave no file behind")
	}
}

func TestASaveOfAnotherLayoutIsReported(t *testing.T) {
	path := filepath.Join(t.TempDir(), "progress.json")
	os.WriteFile(path, []byte(`{"version": 2, "seed": 7}`), 0o600)
	if _, _, err := (&File{Path: path}).Load(); err == nil {
		t.Fatal("a save of another layout must be reported")
	}
}
