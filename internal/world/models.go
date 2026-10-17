package world

// Model is the shape of a fixture as the game models it: triangles between points, each in its own
// colour, the points in world units about the middle of the ground the model stands on, y upward.
type Model struct {
	Points []Point
	Faces  []ModelFace
}

// Point is a point of a model.
type Point struct{ X, Y, Z float64 }

// ModelFace is a triangle of a model between three of its points, by their index.
type ModelFace struct {
	A, B, C int
	Color   Color
}

// PlacedModel is a model standing over a block of tiles, from Col, Row across Cols and Rows.
type PlacedModel struct {
	Col, Row, Cols, Rows int
	Model                *Model
	// Hangs marks a model hung on a wall, such as a banner, which shapes none of the fixtures on
	// its tile.
	Hangs bool
}

// Models lists the models standing on the map; none in the endless forest.
func (w *World) Models() []PlacedModel {
	if w.terrain == nil {
		return nil
	}
	return w.terrain.models
}
