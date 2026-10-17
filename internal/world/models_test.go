package world_test

import (
	"reflect"
	"testing"

	"lapresregne/internal/world"
)

func TestAMapsModelsStandWhereItPlacesThem(t *testing.T) {
	block := &world.Model{
		Points: []world.Point{{X: -1, Y: 0, Z: -0.5}, {X: 1, Y: 0, Z: -0.5}, {X: 0, Y: 0.8, Z: 0.5}},
		Faces:  []world.ModelFace{{A: 0, B: 1, C: 2, Color: world.Color{120, 80, 50}}},
	}
	m := pond()
	m.Models = []world.PlacedModel{{Col: 1, Row: 1, Cols: 2, Rows: 1, Model: block}}
	if models := world.NewOnMap(7, m).Models(); !reflect.DeepEqual(models, m.Models) {
		t.Fatalf("models = %+v, want the one the map places", models)
	}
	if models := world.New(7).Models(); len(models) != 0 {
		t.Fatalf("endless forest models = %+v, want none", models)
	}
}
