package viewfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestASavedViewLoadsBackUnchanged(t *testing.T) {
	file := File{Path: filepath.Join(t.TempDir(), "nested", "view.json")}
	kept := View{Zoom: 5, Yaw: -0.375}
	if err := file.Save(kept); err != nil {
		t.Fatal(err)
	}
	if loaded, err := file.Load(); err != nil || loaded != kept {
		t.Fatalf("load = %+v, %v; want the kept %+v", loaded, err, kept)
	}
}

func TestAMissingFileOpensOnTheForestView(t *testing.T) {
	file := File{Path: filepath.Join(t.TempDir(), "view.json")}
	if loaded, err := file.Load(); err != nil || loaded != (View{}) {
		t.Fatalf("load = %+v, %v; want the forest view facing north", loaded, err)
	}
}

func TestAnUnreadableFileIsReported(t *testing.T) {
	path := filepath.Join(t.TempDir(), "view.json")
	os.WriteFile(path, []byte("{not json"), 0o600)
	if _, err := (File{Path: path}).Load(); err == nil {
		t.Fatal("a corrupt file must be reported")
	}
}
