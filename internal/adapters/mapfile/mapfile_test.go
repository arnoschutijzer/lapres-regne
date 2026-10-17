package mapfile

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

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
	loaded, err := Load(write(t, "yard.map", "terrain\n#_:@bH\n=G+DB.\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := [][]world.Ground{
		{world.Wall, world.Floor, world.Path, world.Grass, world.Boards, world.Bridge},
		{world.Fence, world.Gate, world.Grave, world.Door, world.Battlement, world.Grass},
	}
	if !reflect.DeepEqual(loaded.Ground, want) {
		t.Fatalf("ground = %v, want %v", loaded.Ground, want)
	}
}

func TestAMapFileNamesItsAreas(t *testing.T) {
	loaded, err := Load(write(t, "village.map", `area C Lumbridge Castle
area g Graveyard

terrain
#_#..
#@#++

areas
CCC.g
CCC
`))
	if err != nil {
		t.Fatal(err)
	}
	castle, yard := "Lumbridge Castle", "Graveyard"
	want := [][]string{{castle, castle, castle, "", yard}, {castle, castle, castle}}
	if !reflect.DeepEqual(loaded.Areas, want) {
		t.Fatalf("areas = %q, want %q", loaded.Areas, want)
	}
}

func TestAnAreaMustBeNamedBeforeItIsDrawn(t *testing.T) {
	_, err := Load(write(t, "village.map", "terrain\n.@\n\nareas\n.x\n"))
	if err == nil || !strings.Contains(err.Error(), "line 5: unnamed area 'x'") {
		t.Fatalf("load = %v, want the unnamed area reported", err)
	}
}

func TestAMapFileRaisesItsTilesToTheirHeights(t *testing.T) {
	loaded, err := Load(write(t, "hill.map", "terrain\n.@..\n....\n\nheights\n0123\n9\n"))
	if err != nil {
		t.Fatal(err)
	}
	if want := [][]int{{0, 1, 2, 3}, {9}}; !reflect.DeepEqual(loaded.Heights, want) {
		t.Fatalf("heights = %v, want %v", loaded.Heights, want)
	}
}

func TestAHeightMustBeADigit(t *testing.T) {
	_, err := Load(write(t, "hill.map", "terrain\n.@\n\nheights\n0x\n"))
	if err == nil || !strings.Contains(err.Error(), "line 5: height 'x' is not a digit") {
		t.Fatalf("load = %v, want the height that is not a digit reported", err)
	}
}

func TestTheLumbridgeMapLoadsWithTheHeroInTheCastleCourtyard(t *testing.T) {
	lumbridge, err := Load(filepath.Join("..", "..", "..", "maps", "lumbridge.map"))
	if err != nil {
		t.Fatal(err)
	}
	if lumbridge.Name != "Lumbridge" {
		t.Fatalf("name = %q, want Lumbridge", lumbridge.Name)
	}
	if area := lumbridge.Areas[lumbridge.StartRow][lumbridge.StartCol]; area != "Castle courtyard" {
		t.Fatalf("start area = %q, want the castle courtyard", area)
	}
}

func TestTheHeroWalksOverTheLumbridgeBridge(t *testing.T) {
	lumbridge, err := Load(filepath.Join("..", "..", "..", "maps", "lumbridge.map"))
	if err != nil {
		t.Fatal(err)
	}
	// The bridge runs east over the River Lum from the path at tile 89, 65 to the far bank.
	lumbridge.StartCol, lumbridge.StartRow = 89, 65
	w := world.NewOnMap(7, lumbridge)
	w.Advance(1, 0, 10*time.Second, false)
	if x, _ := w.Hero().Position(); x < 102*world.TileSize {
		t.Fatalf("x = %v; want the hero over the bridge on the east bank", x)
	}
}

func TestAMapFilePlacesItsFixtures(t *testing.T) {
	loaded, err := Load(write(t, "hall.map", "terrain\n.@...\n_____\n\nobjects\n.....\n.t..t\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := []world.PlacedFixture{{Col: 1, Row: 1, Kind: world.Table}, {Col: 4, Row: 1, Kind: world.Table}}
	if !reflect.DeepEqual(loaded.Fixtures, want) {
		t.Fatalf("fixtures = %+v, want %+v", loaded.Fixtures, want)
	}
}

func TestAnUnknownFixtureIsReported(t *testing.T) {
	_, err := Load(write(t, "hall.map", "terrain\n.@.\n\nobjects\n.?.\n"))
	if err == nil || !strings.Contains(err.Error(), "line 5") {
		t.Fatalf("error = %v, want the unknown fixture reported on line 5", err)
	}
}

func TestEachObjectLetterStandsForItsFixture(t *testing.T) {
	for letter, kind := range map[string]world.Fixture{
		"t": world.Table, "c": world.Chair, "T": world.Throne, "a": world.Armour, "r": world.Range,
		"o": world.Barrel, "x": world.Crate, "q": world.Sacks, "s": world.Shelves, "S": world.Staircase,
		"L": world.Ladder, "i": world.Torch, "l": world.Candles, "f": world.Standard,
	} {
		loaded, err := Load(write(t, "room.map", "terrain\n@.\n\nobjects\n."+letter+"\n"))
		if err != nil {
			t.Fatal(err)
		}
		if want := []world.PlacedFixture{{Col: 1, Row: 0, Kind: kind}}; !reflect.DeepEqual(loaded.Fixtures, want) {
			t.Errorf("object %q = %+v, want %+v", letter, loaded.Fixtures, want)
		}
	}
}
