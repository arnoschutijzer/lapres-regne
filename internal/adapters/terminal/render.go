package terminal

import (
	"fmt"
	"math"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"lapresregne/internal/world"
)

// Bayer thresholds turn smooth lighting into a stable, ordered dot pattern.
var bayer = [4][4]float64{
	{0, 8, 2, 10}, {12, 4, 14, 6}, {3, 11, 1, 9}, {15, 7, 13, 5},
}

// renderer draws the forest as braille cells and sends only changed cells.
type renderer struct {
	world                     *world.World
	camera                    *Camera
	title                     string
	canvas                    forestCanvas
	trees                     []world.Tree
	rivers                    riverSamples
	cells                     []forestCell
	screenWidth, screenHeight int
	lastStatus, lastHint      string
	lastSkill                 string
	// inventoryOpen shows the inventory panel; panelShown and lastPanel remember the last one drawn.
	inventoryOpen, panelShown bool
	lastPanel                 string
	cursor                    slotCursor
	// bankOpen shows the bank panel; bankShown and lastBank remember the last one drawn.
	bankOpen, bankShown bool
	lastBank            string
	bankCursor          bankCursor
	// bankTop is the first row of stacks the bank panel shows.
	bankTop int
	// built surveys a map's structures within the drawn area, each frame, and upstairs those of
	// each floor above the ground up to the hero's, the first floor first.
	built    builtArea
	upstairs []builtArea
	// menu lists the actions offered for the tree, doorway, staircase or ladder in reach, named by
	// menuTitle; nil when closed. menuFor is what it offers them for.
	menu       []string
	menuTitle  string
	menuFor    menuFor
	menuTarget int
	zoom       zoom
	turn       turning
	gait       gait
	// lastSeconds is when the last frame was rendered, so the gait can follow the time between frames.
	lastSeconds float64
	// palette is the color of each ink: the forest's own, then the colors of the map's models.
	palette [][3]int
	// modelInks are the inks of each model's faces, in their order.
	modelInks map[*world.Model][]int
	// bands are the canvases the structures and models are drawn on at once, a band of rows each.
	bands []forestCanvas
	// revealed is how far the clearing around the hero is open, from 0 shut to 1 open, and
	// revealing marks that a frame has set it, so the first frame shows it settled.
	revealed  float64
	revealing bool
	// hiddenFrom is what the hero was last judged hidden or not from, and hidden the judgment.
	hiddenFrom hidingView
	hidden     bool
	// style turns the canvas under each cell into the cell drawn.
	style Style
}

// newRenderer opens the view on the hero at the camera's zoom and yaw, so a resumed session starts
// where it was left.
func newRenderer(forest *world.World, camera *Camera, title string) *renderer {
	camera.lookAt(forest.Hero().Position())
	r := &renderer{world: forest, camera: camera, title: title, zoom: zoom{current: float64(camera.Zoom())}, turn: turning{current: camera.Yaw()}, style: Dithered{}}
	r.palette = append([][3]int(nil), forestPalette[:]...)
	r.modelInks = make(map[*world.Model][]int)
	for storey := range forest.Storeys() {
		for _, placed := range forest.ModelsOn(storey) {
			if _, known := r.modelInks[placed.Model]; known {
				continue
			}
			inks := make([]int, len(placed.Model.Faces))
			for i, face := range placed.Model.Faces {
				inks[i] = r.inkOf(face.Color)
			}
			r.modelInks[placed.Model] = inks
		}
	}
	return r
}

// maxInks is how many inks a braille cell's ink can name.
const maxInks = 256

// inkOf is the ink of a color a model is painted in: one of its own, added to the palette the
// first time, or the nearest model ink once the palette is full.
func (r *renderer) inkOf(color world.Color) int {
	rgb := [3]int{int(color.R), int(color.G), int(color.B)}
	nearest, distance := -1, math.MaxInt
	for ink := len(forestPalette); ink < len(r.palette); ink++ {
		known := r.palette[ink]
		d := (known[0]-rgb[0])*(known[0]-rgb[0]) + (known[1]-rgb[1])*(known[1]-rgb[1]) + (known[2]-rgb[2])*(known[2]-rgb[2])
		if d == 0 {
			return ink
		}
		if d < distance {
			nearest, distance = ink, d
		}
	}
	if len(r.palette) < maxInks {
		r.palette = append(r.palette, rgb)
		return len(r.palette) - 1
	}
	return nearest
}

type forestPixel struct {
	depth, light float64
	ink          int
	// tree is the standing tree drawn here, counting from 1 into the frame's trees; 0 for anything else.
	tree int
	// doorway is the doorway drawn here, counting from 1 into the frame's doorways; 0 for anything
	// else.
	doorway int
}

type forestCell struct {
	bits, ink uint8
	// solid uses filled quadrants for the figure; the scenery keeps its braille dithering.
	solid bool
	// inked draws the braille dots in color rather than in the palette's ink.
	inked        bool
	color, paper [3]int
	// veil dims the forest under an overlay, in sixteenths; an overlay's glyph replaces the forest's
	// braille and shows its ink at veil sixteenths of full strength.
	veil  uint8
	glyph rune
}

// background is the terminal color behind the forest.
var background = [3]int{9, 15, 14}

// brightness lifts every color the terminal shows, keeping each color's hue.
const brightness = 1.4

// textInk is the color of the title, hints, status and panel text.
var textInk = [3]int{147, 166, 130}

// brighten lifts a color by brightness; a channel that would pass full strength scales the
// others down with it, so the hue holds.
func brighten(color [3]int) [3]int {
	scale := brightness
	for _, channel := range color {
		if channel > 0 {
			scale = min(scale, 255/float64(channel))
		}
	}
	var lifted [3]int
	for i, channel := range color {
		lifted[i] = int(math.Round(float64(channel) * scale))
	}
	return lifted
}

// foreground is the escape that draws text in a color, brightened.
func foreground(color [3]int) string {
	lifted := brighten(color)
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", lifted[0], lifted[1], lifted[2])
}

// paperOf is the escape that fills cells behind text with a color, brightened.
func paperOf(color [3]int) string {
	lifted := brighten(color)
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", lifted[0], lifted[1], lifted[2])
}

// brailleDot numbers the braille dots by their row and column in a cell.
var brailleDot = [4][2]int{{0, 3}, {1, 4}, {2, 5}, {6, 7}}

// dotContrast is how far a dot's color is pushed away from the tinted paper under it.
const dotContrast = 0.3

// standOut pushes a dot's color away from the paper under it, so dots keep their texture on paper
// tinted in their own material's color.
func standOut(color, paper [3]int) [3]int {
	var pushed [3]int
	for i := range pushed {
		pushed[i] = min(255, max(0, color[i]+int(math.Round(float64(color[i]-paper[i])*dotContrast))))
	}
	return pushed
}

// mix blends from one color toward another by amount, from 0 to 1.
func mix(from, to [3]int, amount float64) [3]int {
	var blended [3]int
	for i := range blended {
		blended[i] = from[i] + int(math.Round(float64(to[i]-from[i])*amount))
	}
	return blended
}

const maxForestColumns = 220

type forestCanvas struct {
	width, height int
	pixels        []forestPixel
	x, z, scale   float64
	// yaw turns the world about the upright axis through x, z; at 0 north is up the canvas.
	// yawSin and yawCos keep its sine and cosine for the many points projected each frame.
	yaw, yawSin, yawCos float64
	// drawing is the standing tree being drawn, counting from 1 into the frame's trees; 0 for
	// anything else.
	drawing int
	// doorways lists the doorways drawn this frame, and drawingDoorway is the one being drawn,
	// counting from 1 into them; 0 for anything else.
	doorways       []world.Doorway
	drawingDoorway int
	// bandFrom and bandTo limit what is plotted to the rows from bandFrom up to bandTo, so bands of
	// one canvas can be drawn at once; while bandTo is 0 every row is plotted.
	bandFrom, bandTo int
	// elevation is how high the ground lies at a world position; nil for flat ground. highest is
	// the highest it lies anywhere, and eye how high it lies under the camera, which looks down on
	// the ground from the same height wherever it goes.
	elevation    func(x, z float64) float64
	highest, eye float64
	// surface is how high whatever the ground shows lies at a world position, the ground or a
	// floor above it; nil for the ground. floor is how high the floor being drawn on lies; nil for
	// the ground.
	surface, floor func(x, z float64) float64
	// lift raises whatever is being drawn onto the ground it stands on.
	lift float64
	// under is the world position of the ground shown at each pixel, row by row.
	under []groundPoint
	// clearing is the space around the hero that faces in front of them leave open.
	clearing clearing
}

// clearing is an ellipse on the canvas around the hero, through which they show past whatever
// stands in front of them, as through a tree's leaves.
type clearing struct {
	active bool
	// x, y is the ellipse's centre and across, up its radii, in pixels; toward is how far toward
	// the viewer the hero stands.
	x, y, across, up, toward float64
}

// around clears the ellipse around the hero standing at x, z.
func (c *forestCanvas) around(x, z float64) clearing {
	c.standOn(x, z)
	centreX, centreY, _ := c.project(x, 0.9, z)
	_, toward := c.toView(x, z)
	return clearing{active: true, x: centreX, y: centreY, across: 7 * c.scale / forestScale, up: 9 * c.scale / forestScale, toward: toward}
}

// hides tells whether the clearing leaves open a pixel showing a point of the world: inside it,
// and standing between the viewer and the hero.
func (c *forestCanvas) hides(px, py int, at vec) bool {
	k := c.clearing
	if !k.active {
		return false
	}
	if _, toward := c.toView(at.x, at.z); toward <= k.toward {
		return false
	}
	dx, dy := (float64(px)-k.x)/k.across, (float64(py)-k.y)/k.up
	return dx*dx+dy*dy < 1
}

// groundPoint is a position on the ground.
type groundPoint struct{ x, z float64 }

// heightAt is how high the ground lies at a world position.
func (c *forestCanvas) heightAt(x, z float64) float64 {
	if c.elevation == nil {
		return 0
	}
	return c.elevation(x, z)
}

// surfaceAt is how high whatever the ground shows lies at a world position: the ground, or a floor
// above it.
func (c *forestCanvas) surfaceAt(x, z float64) float64 {
	if c.surface == nil {
		return c.heightAt(x, z)
	}
	return c.surface(x, z)
}

// standOn raises whatever is drawn next onto the floor being drawn on at a world position, the
// ground unless a floor above it is.
func (c *forestCanvas) standOn(x, z float64) {
	if c.floor != nil {
		c.lift = c.floor(x, z)
		return
	}
	c.lift = c.heightAt(x, z)
}

// turnTo sets the canvas's yaw.
func (c *forestCanvas) turnTo(yaw float64) {
	c.yaw = yaw
	c.yawSin, c.yawCos = math.Sincos(yaw)
}

var forestPalette = [...][3]int{
	{59, 78, 58},    // ground
	{102, 131, 71},  // grass
	{76, 100, 62},   // shaded leaves
	{147, 166, 91},  // sunlit leaves
	{143, 118, 80},  // bark
	{69, 130, 144},  // river
	{156, 195, 179}, // ripples
	{198, 187, 120}, // sunlight
	{236, 185, 105}, // character
	{205, 214, 218}, // axe head
	{214, 184, 136}, // wood chips
	{140, 150, 100}, // shaded willow leaves
	{206, 208, 150}, // sunlit willow leaves
	{48, 104, 86},   // shaded yew needles
	{84, 140, 112},  // sunlit yew needles
	{112, 104, 54},  // shaded oak leaves
	{176, 158, 78},  // sunlit oak leaves
	{212, 172, 140}, // skin
	{112, 82, 60},   // hair
	{176, 172, 160}, // undershirt
	{96, 100, 112},  // shorts
	{30, 26, 24},    // eyes
	{246, 212, 104}, // gold fireworks
	{124, 232, 168}, // green fireworks
	{238, 236, 226}, // white fireworks
	{238, 116, 60},  // ember fireworks
	{96, 206, 238},  // cyan fireworks
	{226, 104, 214}, // magenta fireworks
	{226, 52, 44},   // red fireworks
	{214, 206, 186}, // stone
	{150, 122, 84},  // earth path
	{96, 100, 108},  // flagstones
	{150, 162, 176}, // iron
	{164, 118, 70},  // planks
	{132, 130, 136}, // shaded stone
	{40, 66, 60},    // grass in shadow
	{96, 80, 64},    // earth path in shadow
	{60, 64, 78},    // flagstones in shadow
	{232, 206, 92},  // yellow wildflowers
	{226, 226, 214}, // white wildflowers
	{176, 136, 214}, // violet wildflowers
	{150, 146, 136}, // stones
	{74, 112, 60},   // bushes
	{110, 92, 62},   // muddy bank
	{150, 160, 96},  // reeds
	{92, 120, 64},   // moss
	{70, 96, 124},   // window glass
	{150, 104, 62},  // wooden boards
	{143, 118, 80},  // axe handle
	{176, 148, 88},  // thatch
	{120, 98, 60},   // shaded thatch
	{96, 104, 120},  // slate
	{64, 70, 86},    // shaded slate
	{56, 86, 156},   // blue cloth
	{152, 52, 48},   // red cloth
	{224, 222, 212}, // white cloth
	{208, 170, 76},  // gold
	{216, 94, 42},   // embers
	{252, 206, 108}, // flames
	{232, 224, 198}, // candle wax
	{170, 144, 100}, // burlap
	{160, 74, 56},   // clay roof tiles
	{104, 52, 46},   // shaded clay roof tiles
}

// foremost marks the inks of the hero, their axe, flying chips and fireworks.
func foremost(ink int) bool {
	return ink >= 8 && ink <= 10 || ink >= defaultLook.skin && ink < stoneInk || ink == axeHandleInk
}

// toView turns a world position into the view's frame: across the canvas, and toward the viewer.
func (c *forestCanvas) toView(x, z float64) (across, toward float64) {
	sin, cos := c.yawSin, c.yawCos
	x, z = x-c.x, z-c.z
	return x*cos + z*sin, -x*sin + z*cos
}

// fromView turns an offset in the view's frame back into a world position.
func (c *forestCanvas) fromView(across, toward float64) (x, z float64) {
	sin, cos := c.yawSin, c.yawCos
	return c.x + across*cos - toward*sin, c.z + across*sin + toward*cos
}

func (c *forestCanvas) project(x, y, z float64) (float64, float64, float64) {
	across, toward := c.toView(x, z)
	y += c.lift - c.eye
	return float64(c.width)/2 + across*c.scale,
		float64(c.height)*0.57 + (toward*0.72-y*0.69)*c.scale,
		-toward*0.69 - y*0.72
}

// screen is the canvas pixel showing a point of the world at a height, wherever is being drawn.
func (c *forestCanvas) screen(x, y, z float64) (float64, float64) {
	across, toward := c.toView(x, z)
	return float64(c.width)/2 + across*c.scale, float64(c.height)*0.57 + (toward*0.72-(y-c.eye)*0.69)*c.scale
}

// ground is the world position of the ground shown at the canvas pixel x, y.
func (c *forestCanvas) ground(x, y float64) (float64, float64) {
	point := c.under[int(y)*c.width+int(x)]
	return point.x, point.z
}

// viewer is the direction from the ground toward the viewer, in the world.
func (c *forestCanvas) viewer() vec {
	sin, cos := c.yawSin, c.yawCos
	return vec{-0.69 * sin, 0.72, 0.69 * cos}
}

// mayShow tells whether any part of a tree can reach the canvas, judged by generous bounds around
// its trunk and every species' crowns.
func (c *forestCanvas) mayShow(tree world.Tree) bool {
	reach := (2*tree.Radius + 1) * c.scale
	baseX, baseY, _ := c.project(tree.X, 0, tree.Z)
	_, topY, _ := c.project(tree.X, tree.Height*1.4+tree.Radius, tree.Z)
	return baseX+reach >= 0 && baseX-reach < float64(c.width) && baseY+reach >= 0 && topY-reach < float64(c.height)
}

func (c *forestCanvas) plot(x, y int, depth, light float64, ink int) {
	if x < 0 || y < 0 || x >= c.width || y >= c.height || c.bandTo > 0 && (y < c.bandFrom || y >= c.bandTo) {
		return
	}
	i := y*c.width + x
	if depth <= c.pixels[i].depth {
		c.pixels[i] = forestPixel{depth, light, ink, c.drawing, c.drawingDoorway}
	}
}

func (c *forestCanvas) line(x1, y1, z1, x2, y2, z2, light float64, ink int) {
	ax, ay, ad := c.project(x1, y1, z1)
	bx, by, bd := c.project(x2, y2, z2)
	steps := max(1, int(math.Ceil(max(math.Abs(bx-ax), math.Abs(by-ay)))))
	for i := 0; i <= steps; i++ {
		u := float64(i) / float64(steps)
		c.plot(int(math.Round(ax+(bx-ax)*u)), int(math.Round(ay+(by-ay)*u)), ad+(bd-ad)*u-0.02, light, ink)
	}
}

func (r *renderer) render(cols, rows int, seconds float64) *forestCanvas {
	f := r.world
	c := &r.canvas
	cameraX, cameraZ := r.camera.x, r.camera.z
	c.width, c.height, c.x, c.z, c.scale = cols*2, rows*4, cameraX, cameraZ, r.zoom.scale()
	c.turnTo(r.turn.current)
	storey, shown := f.Storey(), r.shownStoreys()
	c.elevation, c.highest, c.lift = f.Elevation, f.Highest()+float64(shown)*world.StoreyHeight, 0
	c.surface, c.floor, c.clearing = nil, nil, clearing{}
	if !f.Mapped() {
		// The endless forest's hills level out by the rivers, which are sampled below.
		c.elevation = func(x, z float64) float64 { return f.ElevationAmid(r.rivers.at(z), x, z) }
	}
	if shown > 0 {
		// The ground shows the floors drawn, each over the floors below it.
		c.surface = func(x, z float64) float64 {
			if shown, ok := r.shownFloor(x, z); ok {
				return f.ElevationOn(shown, x, z)
			}
			return c.heightAt(x, z)
		}
	}
	// The camera looks down on the floor the hero stands on.
	c.eye = f.ElevationOn(storey, cameraX, cameraZ)
	if len(c.pixels) != c.width*c.height {
		c.pixels = make([]forestPixel, c.width*c.height)
		c.under = make([]groundPoint, c.width*c.height)
	}
	// The world area to draw holds the view, with room for trees and grass that reach into it,
	// reaching farther off where the ground falls away below the camera and nearer where it rises.
	halfAcross := float64(c.width)/c.scale/2 + 5
	far, near := -float64(c.height)/c.scale-5-c.eye, float64(c.height)/c.scale+12+c.highest-c.eye
	left, right, back, front := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	for _, corner := range [4][2]float64{{-halfAcross, far}, {halfAcross, far}, {-halfAcross, near}, {halfAcross, near}} {
		x, z := c.fromView(corner[0], corner[1])
		left, right = math.Min(left, x), math.Max(right, x)
		back, front = math.Min(back, z), math.Max(front, z)
	}
	trees := r.trees[:0]
	spacing := f.TreeGrid()
	for gz := int(math.Floor(back / spacing)); gz <= int(math.Ceil(front/spacing)); gz++ {
		for gx := int(math.Floor(left / spacing)); gx <= int(math.Ceil(right/spacing)); gx++ {
			if tree, exists := f.TreeAt(gx, gz); exists {
				trees = append(trees, tree)
			}
		}
	}
	r.trees = trees
	// Front-to-back drawing lets hidden crown pixels skip lighting calculations.
	sort.Slice(trees, func(i, j int) bool {
		_, towardI := c.toView(trees[i].X, trees[i].Z)
		_, towardJ := c.toView(trees[j].X, trees[j].Z)
		return towardI*0.69+trees[i].Height*0.72 > towardJ*0.69+trees[j].Height*0.72
	})
	// A map's water lies in its tiles, so only the endless forest samples its rivers.
	if f.Mapped() {
		r.built.survey(f, 0, left, right, back, front)
		for len(r.upstairs) < shown {
			r.upstairs = append(r.upstairs, builtArea{})
		}
		r.upstairs = r.upstairs[:shown]
		for floor := 1; floor <= shown; floor++ {
			r.upstairs[floor-1].survey(f, floor, left, right, back, front)
		}
		// Whatever stands on a floor under another drawn above it is hidden.
		r.built.cover(r.upstairs)
		for i := range r.upstairs {
			r.upstairs[i].cover(r.upstairs[i+1:])
		}
		r.reveal(c, seconds)
	} else {
		r.rivers.sample(f, back, front, left, right, 1/c.scale)
	}
	// Each column of ground depends only on the world, so the columns are shared out across the cores.
	shareOut(c.width, func(x int) { r.groundColumn(c, x, seconds) })
	c.doorways = c.doorways[:0]
	if f.Mapped() {
		r.drawBuilt(c)
		r.drawRoofs(c)
		c.clearing = clearing{}
	}
	r.drawScatter(c, left, right, back, front)
	for _, tree := range trees {
		// Only a crown of leaves throws a shadow; a stump or a dead tree's bare limbs barely do.
		if !tree.Stump && tree.Kind != world.DeadTree {
			c.shadow(tree)
		}
	}
	chopped, chopping := f.ChoppedTree()
	strike, sinceStrike, _ := f.LastStrike()
	sinceHit := sinceStrike.Seconds()
	for i, tree := range trees {
		shake := 0.0
		if chopping && tree == chopped {
			shake = jolt(sinceHit)
		}
		c.standOn(tree.X, tree.Z)
		if c.mayShow(tree) {
			if !tree.Stump {
				c.drawing = i + 1
			}
			r.drawTree(c, tree, shake)
			c.drawing = 0
		}
	}
	if chopping {
		c.standOn(chopped.X, chopped.Z)
		drawChips(c, f, chopped, strike, sinceHit)
	}
	for _, item := range f.GroundItems() {
		c.standOn(item.X, item.Z)
		drawGroundItem(c, item)
	}
	c.floor = r.floorOf(storey)
	r.drawCharacter(c, seconds)
	c.floor = nil
	r.drawGrass(c, left, right, back, front, seconds)
	c.floor = r.floorOf(storey)
	c.halo(r.world.Hero().Position())
	c.floor = nil
	r.drawFireworks(c)
	return c
}

func (r *renderer) drawCharacter(c *forestCanvas, seconds float64) {
	x, z := r.world.Hero().Position()
	c.standOn(x, z)
	r.gait.update(x, z, r.world.Hero().Facing(), min(0.1, max(0, seconds-r.lastSeconds)))
	r.lastSeconds = seconds
	drawDetailedHero(c, x, z, r.gait, defaultLook)
	if _, sinceStrike, chopping := r.world.LastStrike(); chopping {
		drawAxe(c, x, z, r.world.Hero().Facing(), sinceStrike.Seconds())
	}
}

// canvasPoint finds the canvas pixel at the centre of a 1-based terminal cell of the last frame.
func (r *renderer) canvasPoint(column, row int) (x, y float64, ok bool) {
	c := &r.canvas
	if r.screenWidth == 0 || c.width == 0 {
		return 0, 0, false
	}
	cols, rows := c.width/2, c.height/4
	col, canvasRow := column-1-(r.screenWidth-cols)/2, row-2
	if col < 0 || col >= cols || canvasRow < 0 || canvasRow >= rows {
		return 0, 0, false
	}
	// The centre of the cell's 2 x 4 braille dots.
	return float64(col*2) + 1, float64(canvasRow*4) + 2, true
}

// groundAt finds the world position under a 1-based terminal cell of the last frame.
func (r *renderer) groundAt(column, row int) (x, z float64, ok bool) {
	px, py, ok := r.canvasPoint(column, row)
	if !ok {
		return 0, 0, false
	}
	x, z = r.canvas.ground(px, py)
	return x, z, true
}

// treeAt finds the frontmost standing tree drawn under a 1-based terminal cell of the last frame.
// Only the tree's own pixels count, so leaves cut away up close leave the trunk to be clicked.
func (r *renderer) treeAt(column, row int) (world.Tree, bool) {
	front := r.frontmostAt(column, row, func(p forestPixel) int { return p.tree })
	if front == 0 {
		return world.Tree{}, false
	}
	return r.trees[front-1], true
}

// doorwayAt finds the frontmost doorway drawn under a 1-based terminal cell of the last frame: its
// door or gate, or the stone framing it.
func (r *renderer) doorwayAt(column, row int) (world.Doorway, bool) {
	front := r.frontmostAt(column, row, func(p forestPixel) int { return p.doorway })
	if front == 0 {
		return world.Doorway{}, false
	}
	return r.canvas.doorways[front-1], true
}

// frontmostAt finds what is drawn frontmost under a 1-based terminal cell of the last frame, as
// one of a pixel's numbers picks it out; 0 where nothing is.
func (r *renderer) frontmostAt(column, row int, number func(forestPixel) int) int {
	c := &r.canvas
	centreX, centreY, ok := r.canvasPoint(column, row)
	if !ok {
		return 0
	}
	// The cell covers 2 x 4 braille dots around its centre.
	left, top := int(centreX)-1, int(centreY)-2
	front, nearest := 0, math.Inf(1)
	for y := top; y < top+4; y++ {
		for x := left; x < left+2; x++ {
			p := c.pixels[y*c.width+x]
			if n := number(p); n != 0 && p.depth < nearest {
				front, nearest = n, p.depth
			}
		}
	}
	return front
}

// slotAt finds the inventory slot under a 1-based terminal cell of the open panel.
func (r *renderer) slotAt(column, row int) (int, bool) {
	if !r.inventoryOpen || r.screenWidth == 0 || r.canvas.width == 0 {
		return 0, false
	}
	cols := r.canvas.width / 2
	panelColumn := column - 1 - (r.screenWidth-cols)/2 - max(0, cols-panelWidth)
	line := row - 2 - slotLine
	slotRow := line / iconLines
	// A short terminal cuts the panel off at the forest's last row.
	if row-2 >= r.canvas.height/4 {
		return 0, false
	}
	if panelColumn < slotIndent || panelColumn >= slotIndent+panelColumns*slotPitch || line < 0 || slotRow >= world.InventorySize/panelColumns {
		return 0, false
	}
	return slotRow*panelColumns + (panelColumn-slotIndent)/slotPitch, true
}

// synchronizedStart and synchronizedEnd bracket a frame so the terminal shows it only once it has
// arrived whole, never half drawn over the last; terminals that cannot ignore them.
const (
	synchronizedStart = "\x1b[?2026h"
	synchronizedEnd   = "\x1b[?2026l"
)

func (r *renderer) frame(width, height int, seconds float64) string {
	var out strings.Builder
	out.WriteString(synchronizedStart + "\x1b[H\x1b[0m" + paperOf(background) + foreground(textInk))
	if width < 40 || height < 14 {
		r.screenWidth = 0
		message := "Resize to 40 x 14. Q quits."
		out.WriteString("\x1b[2J" + message[:min(width, len(message))])
		return out.String() + synchronizedEnd
	}
	cols, rows := min(width, maxForestColumns), min(height-3, 90)
	c := r.render(cols, rows, seconds)
	margin := (width - cols) / 2
	full := width != r.screenWidth || height != r.screenHeight
	if full {
		r.screenWidth, r.screenHeight = width, height
		if len(r.cells) != cols*rows {
			r.cells = make([]forestCell, cols*rows)
		}
		out.WriteString("\x1b[2J\x1b[1;1H  " + r.title)
	}
	hint := "  arrows/WASD/IJKL · shift run · E act · Tab bag · B bank · X drop · Q quit"
	if width >= 81 {
		hint += " · N/P tracks · V style: " + r.style.Name()
	}
	if message := r.world.Message(); message != "" {
		hint = "  " + message
	}
	if r.menu != nil {
		hint = "  " + r.menuTitle + ":"
		for i, entry := range r.menu {
			hint += fmt.Sprintf("  %d %s", i+1, entry)
		}
	}
	if full || hint != r.lastHint {
		controls := []rune(hint)
		fmt.Fprintf(&out, "\x1b[%d;1H\x1b[2K%s%s", height-1, foreground(textInk), string(controls[:min(width-1, len(controls))]))
		r.lastHint = hint
	}
	skill := ""
	for _, name := range world.SkillNames() {
		trained := r.world.Hero().Skill(name)
		skill += fmt.Sprintf("%s %d · %s xp  ", strings.ToLower(name), trained.Level(), strconv.FormatFloat(trained.XP(), 'f', -1, 64))
	}
	if full || skill != r.lastSkill {
		// Right-align the skill on the title row without ever covering the title. The row is cleared
		// from the end of the title, so a shorter skill leaves nothing of a longer one behind.
		skillRunes := []rune(skill)
		titleEnd := len([]rune(r.title)) + 2
		start := max(titleEnd+3, width-len(skillRunes))
		if start < width {
			fmt.Fprintf(&out, "\x1b[1;%dH\x1b[K\x1b[1;%dH%s%s", titleEnd+1, start+1, foreground(textInk), string(skillRunes[:min(width-start, len(skillRunes))]))
		}
		r.lastSkill = skill
	}
	// The panel covers the forest's top right corner; when it closes, those cells are drawn afresh.
	panelLeft := max(0, cols-panelWidth)
	if r.panelShown && !r.inventoryOpen {
		for row := range min(panelHeight, rows) {
			for col := panelLeft; col < cols; col++ {
				r.cells[row*cols+col] = forestCell{bits: 0xff, ink: 0xff}
			}
		}
	}
	// The bank covers the top left corner in the same way.
	bankRight := min(bankWidth(bankColumns(cols)), cols)
	if r.bankShown && !r.bankOpen {
		for row := range min(bankHeight, rows) {
			for col := range bankRight {
				r.cells[row*cols+col] = forestCell{bits: 0xff, ink: 0xff}
			}
		}
	}
	banner := levelUpBanner(r.world, cols)
	lastColor := [3]int{-1, -1, -1}
	lastPaper := [3]int{-1, -1, -1}
	for row := range rows {
		var current [maxForestColumns]forestCell
		for col := range cols {
			current[col] = r.style.cell(c, r.palette, col, row)
			if cover, covered := banner.at(row, col); covered {
				if cover.glyph != 0 {
					current[col] = cover
				} else {
					current[col].veil = cover.veil
				}
			}
		}
		previous := r.cells[row*cols : (row+1)*cols]
		start, visible := 0, cols
		if r.bankOpen && row < bankHeight {
			start = bankRight
		}
		if r.inventoryOpen && row < panelHeight {
			visible = panelLeft
		}
		for col := start; col < visible; {
			if !full && current[col] == previous[col] {
				col++
				continue
			}
			end := col + 1
			// Bridge tiny gaps when that costs less than another cursor escape.
			for scan := end; scan < visible; scan++ {
				if full || current[scan] != previous[scan] {
					if scan-end > 3 {
						break
					}
					end = scan + 1
				}
			}
			fmt.Fprintf(&out, "\x1b[%d;%dH", row+2, margin+col+1)
			for ; col < end; col++ {
				cell := current[col]
				previous[col] = cell
				color, glyph := r.palette[cell.ink], rune(0x2800+int(cell.bits))
				paper := cell.paper
				if paper != background {
					color = standOut(color, paper)
				}
				if cell.solid {
					color, glyph = cell.color, quadrants[cell.bits]
				}
				if cell.inked {
					color = cell.color
				}
				if cell.glyph != 0 {
					color, paper, glyph = mix(background, color, float64(cell.veil)/16), background, cell.glyph
				} else if cell.veil != 0 {
					color = mix(color, background, veilDepth*float64(cell.veil)/16)
					paper = mix(paper, background, veilDepth*float64(cell.veil)/16)
				}
				if paper != lastPaper {
					out.WriteString(paperOf(paper))
					lastPaper = paper
				}
				if color != lastColor {
					out.WriteString(foreground(color))
					lastColor = color
				}
				out.WriteRune(glyph)
			}
		}
	}
	out.WriteString(paperOf(background))
	if r.inventoryOpen {
		lines := inventoryPanel(r.world.Hero().Inventory(), r.cursor)
		panel := strings.Join(lines, "\n")
		if full || !r.panelShown || panel != r.lastPanel {
			for i, line := range lines[:min(len(lines), rows)] {
				fmt.Fprintf(&out, "\x1b[%d;%dH%s", i+2, margin+panelLeft+1, line)
			}
			r.lastPanel = panel
		}
	}
	r.panelShown = r.inventoryOpen
	if r.bankOpen {
		columns := bankColumns(cols)
		// A narrower forest or a smaller bank may leave fewer rows to scroll through.
		r.bankTop = scrolled(r.bankTop, bankCursor{}, columns, len(r.world.Bank().Stacks()))
		lines := bankPanel(r.world.Bank(), columns, r.bankCursor, r.bankTop)
		panel := strings.Join(lines, "\n")
		if full || !r.bankShown || panel != r.lastBank {
			for i, line := range lines[:min(len(lines), rows)] {
				fmt.Fprintf(&out, "\x1b[%d;%dH%s", i+2, margin+1, line)
			}
			r.lastBank = panel
		}
	}
	r.bankShown = r.bankOpen
	x, z := r.world.Hero().Position()
	status := fmt.Sprintf("  x %.1f  z %.1f  |  steps %d  |  facing %s  |  %s", x, z, r.world.Hero().Steps(), r.world.Hero().Facing(), r.world.Region())
	if r.world.Chopping() {
		status += "  |  chopping"
	}
	if r.world.Mapped() {
		status += "  |  " + r.world.MapName()
	} else {
		status += fmt.Sprintf("  |  seed %d", r.world.Seed())
	}
	if full || status != r.lastStatus {
		fmt.Fprintf(&out, "\x1b[%d;1H\x1b[2K%s%s", height, foreground(textInk), status[:min(width-1, len(status))])
		r.lastStatus = status
	}
	return out.String() + synchronizedEnd
}

// groundColumn lights the ground seen down one canvas column: grass, canopy shade, pools of
// sunlight and rivers. It marches from the nearest ground toward the horizon a pixel's depth at a
// time, and each step draws only the ground that rises on screen above what nearer steps drew, so
// nearer hills hide whatever lies behind them.
func (r *renderer) groundColumn(c *forestCanvas, x int, seconds float64) {
	quiet := c.quiet()
	across := (float64(x) - float64(c.width)/2) / c.scale
	step := 1 / (0.72 * c.scale)
	// The nearest ground that can show lies below the bottom of the canvas even where it lies
	// highest, and the farthest lies above the top even where it lies lowest.
	near := (0.43*float64(c.height)/c.scale+0.69*(c.highest-c.eye))/0.72 + step
	far := (-0.57*float64(c.height)/c.scale-0.69*c.eye)/0.72 - step
	// top is the highest row drawn so far; the march draws upward from it.
	top := c.height
	type sample struct{ x, z, height, toward, y float64 }
	at := func(toward float64) sample {
		wx, wz := c.fromView(across, toward)
		height := c.surfaceAt(wx, wz) - c.eye
		return sample{wx, wz, height, toward, float64(c.height)*0.57 + (toward*0.72-height*0.69)*c.scale}
	}
	previous := at(near)
	for toward := near - step; toward >= far && top > 0; toward -= step {
		current := at(toward)
		// Each row the ground rises past between the two steps shows the ground in between.
		for ; top > 0 && float64(top-1) >= current.y; top-- {
			y := top - 1
			u := min(1, max(0, (previous.y-float64(y))/(previous.y-current.y)))
			point := sample{
				previous.x + (current.x-previous.x)*u,
				previous.z + (current.z-previous.z)*u,
				previous.height + (current.height-previous.height)*u,
				previous.toward + (current.toward-previous.toward)*u,
				float64(y),
			}
			ink, light := r.groundLight(c, point.x, point.z, point.height+c.eye, seconds)
			c.pixels[y*c.width+x] = forestPixel{-point.toward*0.69 - point.height*0.72, light * quiet, ink, 0, 0}
			c.under[y*c.width+x] = groundPoint{point.x, point.z}
		}
		previous = current
	}
}

// shownStoreys is how many floors above the ground are drawn: every one while the hero stands in
// the open, but only up to the hero's own while they stand indoors or under a floor above, which
// would hide them.
func (r *renderer) shownStoreys() int {
	f := r.world
	if !f.Mapped() {
		return 0
	}
	if f.Indoors() {
		return f.Storey()
	}
	x, z := f.Hero().Position()
	for above := f.Storey() + 1; above < f.Storeys(); above++ {
		if _, ok := f.GroundOn(above, x, z); ok {
			return f.Storey()
		}
	}
	return f.Storeys() - 1
}

// reveal opens the clearing around the hero while the walls, roofs and floors in front of them hide
// most of them, so they show through, and shuts it once they don't. It eases open and shut.
func (r *renderer) reveal(c *forestCanvas, seconds float64) {
	target := 0.0
	if r.heroHidden(c) {
		target = 1
	}
	dt := min(0.1, max(0, seconds-r.lastSeconds))
	r.revealed += (target - r.revealed) * (1 - math.Exp(-6*dt))
	if !r.revealing || math.Abs(target-r.revealed) < 0.005 {
		r.revealed, r.revealing = target, true
	}
	if r.revealed == 0 {
		return
	}
	c.floor = r.floorOf(r.world.Storey())
	c.clearing = c.around(r.world.Hero().Position())
	c.floor = nil
	c.clearing.across *= r.revealed
	c.clearing.up *= r.revealed
}

// hidingView is everything that decides whether the structures hide the hero: where they stand,
// how the view looks on them and which part of the map is surveyed. A map's structures never
// move, and a door hides the hero no more shut than open, so nothing else changes the judgment.
type hidingView struct {
	judged                        bool
	x, z, yawSin, yawCos, highest float64
	storey, floors, roofs         int
	col, row, cols, rows          int
}

// heroHidden tells whether the walls, doors, fences, roofs and floors above hide most of the hero
// from the viewer. The rays are cast afresh only once the hero, the view or the surveyed part of
// the map changes.
func (r *renderer) heroHidden(c *forestCanvas) bool {
	x, z := r.world.Hero().Position()
	view := hidingView{
		judged: true, x: x, z: z, yawSin: c.yawSin, yawCos: c.yawCos, highest: c.highest,
		storey: r.world.Storey(), floors: len(r.upstairs), roofs: len(r.built.roofs),
		col: r.built.col, row: r.built.row, cols: r.built.cols, rows: r.built.rows,
	}
	if view != r.hiddenFrom {
		r.hiddenFrom, r.hidden = view, r.castHidden(c)
	}
	return r.hidden
}

// castHidden judges whether the hero is hidden by rays cast toward the viewer from points across
// the hero's body. Furniture and models never count: the hero is not shown through them.
func (r *renderer) castHidden(c *forestCanvas) bool {
	f := r.world
	x, z := f.Hero().Position()
	base := f.ElevationOn(f.Storey(), x, z)
	// Nothing stands higher than the walls of the highest floor shown, or the highest roof's ridge.
	top := c.highest + wallHeight + copingHeight + merlonHeight
	for _, roof := range r.built.roofs {
		top = max(top, f.Highest()+roof.ridge())
	}
	hidden := 0
	for _, height := range [3]float64{0.3, 0.9, 1.5} {
		for _, across := range [3]float64{-0.25, 0, 0.25} {
			if r.blocked(c, x+c.yawCos*across, base+height, z+c.yawSin*across, top) {
				hidden++
			}
		}
	}
	return hidden >= 5
}

// blocked tells whether a wall, door, fence, roof or floor above stands between the viewer and the
// point x, y, z of the world, looking no higher than top.
func (r *renderer) blocked(c *forestCanvas, x, y, z, top float64) bool {
	f := r.world
	const step = 0.15
	toViewer := c.viewer()
	for travelled := step; ; travelled += step {
		px, py, pz := x+toViewer.x*travelled, y+toViewer.y*travelled, z+toViewer.z*travelled
		if py > top {
			return false
		}
		for floor := f.Storey(); floor <= len(r.upstairs); floor++ {
			area := &r.built
			if floor > 0 {
				area = &r.upstairs[floor-1]
			}
			height := py - f.ElevationOn(floor, px, pz)
			if height >= 0 && area.occupied(px, height, pz) {
				return true
			}
			// A floor above blocks where the ray rises through it.
			if _, floored := f.GroundOn(floor, px, pz); floored && floor > f.Storey() && height >= 0 && height < toViewer.y*step {
				return true
			}
		}
		ground := c.heightAt(px, pz)
		for _, roof := range r.built.roofs {
			if roof.covers(px, py-ground, pz) {
				return true
			}
		}
	}
}

// shownFloor is the highest floor above the ground drawn floored at a world position, or false
// where the ground shows. A floor above the hero's is left out round the hero where it stands in
// front of them, so they show through it.
func (r *renderer) shownFloor(x, z float64) (int, bool) {
	c := &r.canvas
	for floor := len(r.upstairs); floor > 0; floor-- {
		if _, ok := r.world.GroundOn(floor, x, z); !ok {
			continue
		}
		height := r.world.ElevationOn(floor, x, z)
		if px, py := c.screen(x, height, z); floor > r.world.Storey() && c.hides(int(px), int(py), vec{x, height, z}) {
			continue
		}
		return floor, true
	}
	return 0, false
}

// shareOut works through count pieces of work at once across the cores, each core taking the next
// piece as soon as it is done with its last, so faster cores take on more and none waits long on a
// slower one.
func shareOut(count int, work func(i int)) {
	var next atomic.Int64
	var done sync.WaitGroup
	for range min(runtime.GOMAXPROCS(0), count) {
		done.Add(1)
		go func() {
			defer done.Done()
			for i := int(next.Add(1)) - 1; i < count; i = int(next.Add(1)) - 1 {
				work(i)
			}
		}()
	}
	done.Wait()
}

// drawBuilt draws the structures and models of the ground and every floor shown above it. The
// canvas is cut into bands of rows drawn at once across the cores: each band draws everything but
// plots only its own rows, so every pixel is drawn in the same order as on a single core.
func (r *renderer) drawBuilt(c *forestCanvas) {
	f := r.world
	count := min(runtime.GOMAXPROCS(0), c.height)
	for len(r.bands) < count {
		r.bands = append(r.bands, forestCanvas{})
	}
	var done sync.WaitGroup
	for i := range count {
		band := &r.bands[i]
		// Each band lists the doorways it draws on its own, all in the same order.
		doorways := band.doorways[:0]
		*band = *c
		band.doorways = doorways
		band.bandFrom, band.bandTo = c.height*i/count, c.height*(i+1)/count
		done.Add(1)
		go func() {
			defer done.Done()
			r.drawStructures(band, &r.built)
			r.drawModels(band, &r.built, f.Models())
			for floor := range r.upstairs {
				band.floor = r.floorOf(floor + 1)
				r.drawStructures(band, &r.upstairs[floor])
				r.drawModels(band, &r.upstairs[floor], f.ModelsOn(floor+1))
			}
		}()
	}
	done.Wait()
	c.doorways = append(c.doorways, r.bands[0].doorways...)
}

// floorOf is how high a floor lies, for whatever is drawn on it; nil for the ground.
func (r *renderer) floorOf(storey int) func(x, z float64) float64 {
	if storey == 0 {
		return nil
	}
	return func(x, z float64) float64 { return r.world.ElevationOn(storey, x, z) }
}

// flightAt finds the staircase or ladder on the hero's floor shown under a 1-based terminal cell
// of the last frame.
func (r *renderer) flightAt(column, row int) (world.Flight, bool) {
	x, z, ok := r.groundAt(column, row)
	if !ok {
		return world.Flight{}, false
	}
	fixture, stands := r.world.FixtureOn(r.world.Storey(), x, z)
	if !stands || fixture != world.Staircase && fixture != world.Ladder {
		return world.Flight{}, false
	}
	col, tileRow, _, _ := tileAt(x, z)
	return world.Flight{Col: col, Row: tileRow, Kind: fixture}, true
}

// groundLight is the ink and light of the ground at a world position, lying at a height.
func (r *renderer) groundLight(c *forestCanvas, wx, wz, height, seconds float64) (ink int, light float64) {
	f := r.world
	cover, pools := f.Cover(wx, wz), f.Noise(wx/9, wz/9, 81)
	n := f.Noise(wx*0.65, wz*0.65, 19)
	ink, light = 0, 0.13+n*0.23
	if cover < 0.38 {
		ink, light = 1, 0.21+n*0.24
	}
	// Diagonal pools of sunlight survive only where the canopy opens.
	beam := max(0, wave(wx*0.28-wz*0.19+pools*2)-0.45)
	sun := beam * max(0, 1-cover*1.3)
	light += sun * 0.8
	if sun > 0.24 {
		ink = 7
	}
	// Wildflowers bloom in the open, and patches of grass wear down to bare earth.
	if speck, speckLight, ok := groundSpeck(f, wx, wz, n, c.scale); ok && (cover < 0.38 || speck == pathInk) {
		ink, light = speck, speckLight
	}
	water, bank := false, false
	if f.Mapped() {
		ground, onMap := f.GroundAt(wx, wz)
		built := &r.built
		if shown, ok := r.shownFloor(wx, wz); ok {
			ground, _ = f.GroundOn(shown, wx, wz)
			built = &r.upstairs[shown-1]
		}
		ground = built.underfoot(wx, wz, ground)
		switch {
		case !onMap:
			ink, light = 0, 0
		case ground == world.Path:
			ink, light = pathInk, 0.24+n*0.22
			// Pebbles catch the light here and there along the track.
			if f.Noise(wx*3, wz*3, 23) > 0.78 {
				light += 0.3
			}
		case ground == world.Floor || isWall(ground) || ground == world.Door || ground == world.Steps:
			ink, light = floorInk, flagstones(wx, wz, n)
		case ground == world.Boards:
			ink, light = boardsInk, boards(wx, wz, n)
		case ground == world.Bridge:
			// The deck's boards lie across the way, as an east-west bridge carries it.
			ink, light = boardsInk, boards(wz, wx, n)
		}
		water = ground == world.Water
		if ground == world.Grass && built.banked(wx, wz) {
			ink, light = bankInk, 0.26+n*0.2
		}
		if onMap {
			shade := built.shade(wx, wz)
			light *= shade
			if shade == castShadow {
				ink = inShadow(ink)
			}
		}
	} else {
		distance, width := world.NearestRiver(r.rivers.at(wz), wx)
		water, bank = distance < width, distance < width+0.65
	}
	if water {
		ink, light = 5, 0.28+n*0.3
		if wave(wz*2.6+wx*0.8-seconds*1.8+n*1.3) > 0.91 {
			ink, light = 6, 0.8
		}
	} else if bank {
		ink, light = 7, 0.32+n*0.25
	}
	return ink, light * c.slopeLight(wx, wz, height)
}

// slopeLight brightens the ground lying at a height where it slopes toward the sun and darkens it
// where it slopes away, so the hills read by their shading; flat ground keeps its light.
func (c *forestCanvas) slopeLight(x, z, height float64) float64 {
	if c.elevation == nil {
		return 1
	}
	const reach = 0.3
	normal := vec{height - c.surfaceAt(x+reach, z), reach, height - c.surfaceAt(x, z+reach)}.unit()
	return max(0.2, 1+2*(sunlight(normal)/sunlight(vec{0, 1, 0})-1))
}

// riverSamples holds the river channels at depths a step apart across the drawn area, so each
// ground pixel looks up its rivers instead of working them out again.
type riverSamples struct {
	back, step float64
	// bands is how many channels each depth holds.
	bands    int
	channels []world.RiverChannel
}

func (s *riverSamples) sample(f *world.World, back, front, left, right, step float64) {
	s.back, s.step = back, step
	s.channels = s.channels[:0]
	for z := back; z <= front+step; z += step {
		before := len(s.channels)
		s.channels = f.RiverChannels(s.channels, z, left, right)
		s.bands = len(s.channels) - before
	}
}

// at finds the river channels at the sampled depth nearest z.
func (s *riverSamples) at(z float64) []world.RiverChannel {
	depths := len(s.channels) / s.bands
	i := min(depths-1, max(0, int(math.Round((z-s.back)/s.step))))
	return s.channels[i*s.bands : (i+1)*s.bands]
}

// underlay is how much the material color under the dots is strengthened: twice as much when
// zoomed all the way out, where dots alone scatter into noise, easing to none from the zoom where
// trees show their detail.
func (c *forestCanvas) underlay() float64 {
	return 2 - min(1, max(0, (c.scale-forestScale)/(detailScale-forestScale)))
}

// quiet is how strongly the ground shows, from half when zoomed all the way out up to full from
// the zoom where trees show their detail, so figures and structures stand out of a calm ground
// when the view takes in a lot.
func (c *forestCanvas) quiet() float64 {
	return 0.5 + 0.5*min(1, max(0, (c.scale-forestScale)/(detailScale-forestScale)))
}

// halo leaves a dark ring a pixel wide around the figure standing at x, z, so their silhouette
// reads against whatever lies behind them.
func (c *forestCanvas) halo(x, z float64) {
	c.standOn(x, z)
	footX, footY, _ := c.project(x, 0, z)
	_, headY, _ := c.project(x, 2.4, z)
	reach := 1.5 * c.scale
	left, right := max(1, int(footX-reach)), min(c.width-2, int(footX+reach))
	top, bottom := max(1, int(headY-reach)), min(c.height-2, int(footY+reach))
	if left > right || top > bottom {
		return
	}
	figure := func(px, py int) bool {
		p := c.pixels[py*c.width+px]
		return foremost(p.ink) && p.light > 0
	}
	var ring []int
	for py := top; py <= bottom; py++ {
		for px := left; px <= right; px++ {
			if figure(px, py) {
				continue
			}
			for _, step := range [8][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
				if figure(px+step[0], py+step[1]) {
					ring = append(ring, py*c.width+px)
					break
				}
			}
		}
	}
	for _, i := range ring {
		c.pixels[i].light = 0
	}
}
