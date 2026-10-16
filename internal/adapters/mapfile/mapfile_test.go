package mapfile

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"lapresregne/internal/world"
)

// write puts a map file in a temporary folder and returns its path.
func write(t *testing.T, name, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAMapFileLoadsItsTilesTreesAndStart(t *testing.T) {
	path := write(t, "pond.map", `name The Pond

// A meadow with a pond to the east.
terrain
.T..~
.@O~~
WY...
`)
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	g, w := world.Grass, world.Water
	want := world.Map{
		Name:   "The Pond",
		Ground: [][]world.Ground{{g, g, g, g, w}, {g, g, g, w, w}, {g, g, g, g, g}},
		Trees: []world.PlantedTree{
			{Col: 1, Row: 0, Kind: world.NormalTree}, {Col: 2, Row: 1, Kind: world.Oak},
			{Col: 0, Row: 2, Kind: world.Willow}, {Col: 1, Row: 2, Kind: world.Yew},
		},
		StartCol: 1, StartRow: 1,
	}
	if !reflect.DeepEqual(loaded, want) {
		t.Fatalf("map = %+v, want %+v", loaded, want)
	}
}

func TestAMapWithoutANameIsNamedAfterItsFile(t *testing.T) {
	loaded, err := Load(write(t, "meadow.map", "terrain\n.@.\n"))
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Name != "meadow" {
		t.Fatalf("name = %q, want meadow", loaded.Name)
	}
}

func TestMistakesInAMapFileAreReportedWithTheirLine(t *testing.T) {
	for _, mistake := range []struct{ text, want string }{
		{"terrain\n.@.\n.?.\n", "line 3: unknown tile '?'"},
		{"terrain\n...\n", "no start tile '@'"},
		{"terrain\n.@.\n.@.\n", "line 3: a second start tile '@'"},
		{"height 3\nterrain\n.@.\n", "line 1: unknown setting \"height\""},
		{"name Pond\n", "no terrain"},
	} {
		_, err := Load(write(t, "broken.map", mistake.text))
		if err == nil || !strings.Contains(err.Error(), mistake.want) {
			t.Errorf("load %q = %v, want an error containing %q", mistake.text, err, mistake.want)
		}
	}
}

func TestAMissingMapFileIsReported(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "nowhere.map")); err == nil {
		t.Fatal("a missing map file must be reported")
	}
}

func TestAMapFileLoadsBuiltGroundAndStructures(t *testing.T) {
	loaded, err := Load(write(t, "yard.map", "terrain\n#_:@\n=G+.\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := [][]world.Ground{
		{world.Wall, world.Floor, world.Path, world.Grass},
		{world.Fence, world.Gate, world.Grave, world.Grass},
	}
	if !reflect.DeepEqual(loaded.Ground, want) {
		t.Fatalf("ground = %v, want %v", loaded.Ground, want)
	}
}
