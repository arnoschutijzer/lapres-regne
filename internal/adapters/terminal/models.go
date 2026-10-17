package terminal

import "lapresregne/internal/world"

// drawModels draws each of a floor's models standing within its surveyed area, but those under a
// floor drawn above it, triangle by triangle in their own colours, each lit as it faces the sun,
// on the floor at the middle of the tiles it stands over.
func (r *renderer) drawModels(c *forestCanvas, a *builtArea, models []world.PlacedModel) {
	// The hero is shown through what hangs on a wall, as through the wall, but never through
	// furniture standing on the floor.
	open := c.clearing
	defer func() { c.clearing = open }()
	for _, placed := range models {
		c.clearing.active = open.active && placed.Hangs
		if placed.Col+placed.Cols < a.col || placed.Col > a.col+a.cols || placed.Row+placed.Rows < a.row || placed.Row > a.row+a.rows {
			continue
		}
		if a.at(placed.Col+placed.Cols/2, placed.Row+placed.Rows/2).covered {
			continue
		}
		middleX := (float64(placed.Col) + float64(placed.Cols)/2) * world.TileSize
		middleZ := (float64(placed.Row) + float64(placed.Rows)/2) * world.TileSize
		c.standOn(middleX, middleZ)
		inks := r.modelInks[placed.Model]
		points := placed.Model.Points
		at := func(p world.Point) vec { return vec{middleX + p.X, p.Y, middleZ + p.Z} }
		for i, triangle := range placed.Model.Faces {
			origin, b, d := at(points[triangle.A]), at(points[triangle.B]), at(points[triangle.C])
			u, v := b.sub(origin), d.sub(origin)
			normal := u.cross(v)
			if normal.dot(normal) == 0 {
				continue
			}
			c.fill(face{origin: origin, u: u, v: v, normal: normal.unit(), triangle: true, seamless: true}, inks[i], plain)
		}
	}
}

// modelled marks the tiles of the surveyed area a model stands over, whose fixtures it shapes.
func (a *builtArea) modelled(models []world.PlacedModel) {
	for _, placed := range models {
		if placed.Hangs {
			continue
		}
		for row := placed.Row; row < placed.Row+placed.Rows; row++ {
			for col := placed.Col; col < placed.Col+placed.Cols; col++ {
				if col-a.col >= 0 && row-a.row >= 0 && col-a.col < a.cols && row-a.row < a.rows {
					a.tiles[(row-a.row)*a.cols+col-a.col].shaped = true
				}
			}
		}
	}
}
