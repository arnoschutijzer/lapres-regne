package terminal

import (
	"fmt"
	"math"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	"lapresregne/internal/world"
)

// Bayer thresholds turn smooth lighting into a stable, ordered dot pattern.
var bayer = [4][4]float64{
	{0, 8, 2, 10}, {12, 4, 14, 6}, {3, 11, 1, 9}, {15, 7, 13, 5},
}

// renderer draws the forest as braille cells and sends only changed cells.
type renderer struct {
	world                     *world.World
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
	// built surveys a map's structures within the drawn area, each frame.
	built builtArea
	// menu lists the actions offered for the tree in reach, named by menuTitle; nil when closed.
	menu      []string
	menuTitle string
	zoom      zoom
	turn      turning
	gait      gait
	// lastSeconds is when the last frame was rendered, so the gait can follow the time between frames.
	lastSeconds float64
}

// newRenderer opens the view at the camera's zoom and yaw, so a resumed session starts where it was left.
func newRenderer(world *world.World, title string) *renderer {
	return &renderer{world: world, title: title, zoom: zoom{current: float64(world.Zoom())}, turn: turning{current: world.Yaw()}}
}

type forestPixel struct {
	depth, light float64
	ink          int
	// tree is the standing tree drawn here, counting from 1 into the frame's trees; 0 for anything else.
	tree int
}

type forestCell struct {
	bits, ink uint8
	// solid uses filled quadrants for the figure; the scenery keeps its braille dithering.
	solid        bool
	color, paper [3]int
	// veil dims the forest under an overlay, in sixteenths; an overlay's glyph replaces the forest's
	// braille and shows its ink at veil sixteenths of full strength.
	veil  uint8
	glyph rune
}

// background is the terminal color behind the forest.
var background = [3]int{9, 15, 14}

// brailleDot numbers the braille dots by their row and column in a cell.
var brailleDot = [4][2]int{{0, 3}, {1, 4}, {2, 5}, {6, 7}}

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
	// elevation is how high the ground lies at a world position; nil for flat ground. highest is
	// the highest it lies anywhere, and eye how high it lies under the camera, which looks down on
	// the ground from the same height wherever it goes.
	elevation    func(x, z float64) float64
	highest, eye float64
	// lift raises whatever is being drawn onto the ground it stands on.
	lift float64
	// under is the world position of the ground shown at each pixel, row by row.
	under []groundPoint
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

// standOn raises whatever is drawn next onto the ground at a world position.
func (c *forestCanvas) standOn(x, z float64) {
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
	if x < 0 || y < 0 || x >= c.width || y >= c.height {
		return
	}
	i := y*c.width + x
	if depth <= c.pixels[i].depth {
		c.pixels[i] = forestPixel{depth, light, ink, c.drawing}
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
	cameraX, cameraZ := f.Camera()
	c.width, c.height, c.x, c.z, c.scale = cols*2, rows*4, cameraX, cameraZ, r.zoom.scale()
	c.turnTo(r.turn.current)
	c.elevation, c.highest, c.lift = f.Elevation, f.Highest(), 0
	if !f.Mapped() {
		// The endless forest's hills level out by the rivers, which are sampled below.
		c.elevation = func(x, z float64) float64 { return f.ElevationAmid(r.rivers.at(z), x, z) }
	}
	c.eye = f.Elevation(cameraX, cameraZ)
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
		r.built.survey(f, left, right, back, front)
	} else {
		r.rivers.sample(f, back, front, left, right, 1/c.scale)
	}
	// Each column of ground depends only on the world, so the columns are shared out across the
	// cores, interleaved so columns crossing a river don't all land on one core.
	var ground sync.WaitGroup
	cores := runtime.GOMAXPROCS(0)
	for core := range cores {
		ground.Add(1)
		go func() {
			defer ground.Done()
			for x := core; x < c.width; x += cores {
				r.groundColumn(c, x, seconds)
			}
		}()
	}
	ground.Wait()
	if f.Mapped() {
		r.drawStructures(c)
		r.drawRoofs(c)
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
	r.drawCharacter(c, seconds)
	r.drawGrass(c, left, right, back, front, seconds)
	c.halo(r.world.Hero().Position())
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
	c := &r.canvas
	centreX, centreY, ok := r.canvasPoint(column, row)
	if !ok {
		return world.Tree{}, false
	}
	// The cell covers 2 x 4 braille dots around its centre.
	left, top := int(centreX)-1, int(centreY)-2
	front, nearest := 0, math.Inf(1)
	for y := top; y < top+4; y++ {
		for x := left; x < left+2; x++ {
			if p := c.pixels[y*c.width+x]; p.tree != 0 && p.depth < nearest {
				front, nearest = p.tree, p.depth
			}
		}
	}
	if front == 0 {
		return world.Tree{}, false
	}
	return r.trees[front-1], true
}

// slotAt finds the inventory slot under a 1-based terminal cell of the open panel.
func (r *renderer) slotAt(column, row int) (int, bool) {
	if !r.inventoryOpen || r.screenWidth == 0 || r.canvas.width == 0 {
		return 0, false
	}
	cols := r.canvas.width / 2
	panelColumn := column - 1 - (r.screenWidth-cols)/2 - max(0, cols-panelWidth)
	slotRow := row - 2 - slotLine
	if panelColumn < slotIndent || panelColumn >= slotIndent+panelColumns*slotPitch || slotRow < 0 || slotRow >= world.InventorySize/panelColumns {
		return 0, false
	}
	return slotRow*panelColumns + (panelColumn-slotIndent)/slotPitch, true
}

func (r *renderer) frame(width, height int, seconds float64) string {
	var out strings.Builder
	out.WriteString("\x1b[H\x1b[0m\x1b[48;2;9;15;14m\x1b[38;2;147;166;130m")
	if width < 40 || height < 14 {
		r.screenWidth = 0
		message := "Resize to 40 x 14. Q quits."
		out.WriteString("\x1b[2J" + message[:min(width, len(message))])
		return out.String()
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
	hint := "  arrows/WASD/IJKL · shift run · E act · Tab bag · X drop · Q quit"
	if width >= 81 {
		hint += " · N/P tracks"
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
		fmt.Fprintf(&out, "\x1b[%d;1H\x1b[2K\x1b[38;2;147;166;130m%s", height-1, string(controls[:min(width-1, len(controls))]))
		r.lastHint = hint
	}
	woodcutting := r.world.Hero().Woodcutting()
	skill := fmt.Sprintf("woodcutting %d · %s xp  ", woodcutting.Level(), strconv.FormatFloat(woodcutting.XP(), 'f', -1, 64))
	if full || skill != r.lastSkill {
		// Right-align the skill on the title row without ever covering the title. The row is cleared
		// from the end of the title, so a shorter skill leaves nothing of a longer one behind.
		skillRunes := []rune(skill)
		titleEnd := len([]rune(r.title)) + 2
		start := max(titleEnd+3, width-len(skillRunes))
		if start < width {
			fmt.Fprintf(&out, "\x1b[1;%dH\x1b[K\x1b[1;%dH\x1b[38;2;147;166;130m%s", titleEnd+1, start+1, string(skillRunes[:min(width-start, len(skillRunes))]))
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
	banner := levelUpBanner(r.world, cols)
	lastColor := [3]int{-1, -1, -1}
	lastPaper := [3]int{-1, -1, -1}
	for row := range rows {
		var current [maxForestColumns]forestCell
		for col := range cols {
			bits, ink := 0, 0
			var votes [len(forestPalette)]int
			var coverage [len(forestPalette)]int
			var light [len(forestPalette)]float64
			figure := false
			for dy := range 4 {
				for dx := range 2 {
					x, y := col*2+dx, row*4+dy
					p := c.pixels[y*c.width+x]
					if p.light > 0 {
						coverage[p.ink]++
						light[p.ink] += p.light
						figure = figure || figureInk(p.ink)
					}
					if p.light > (bayer[y%4][x%4]+0.5)/16 {
						bits |= 1 << brailleDot[dy][dx]
						votes[p.ink]++
					}
				}
			}
			best := 0
			for i, count := range votes {
				// The hero, their axe and flying chips stay visible over anything else in the cell.
				score := count
				if foremost(i) && count > 0 {
					score += 64
				}
				if prominent(i) {
					score *= 2
				}
				if score > best {
					ink, best = i, score
				}
			}
			current[col] = forestCell{bits: uint8(bits), ink: uint8(ink), paper: background}
			// A faint material color reinforces broad surfaces while the ordered dots carry their shading.
			for material, count := range coverage {
				if count >= 6 {
					strength := 0.18
					if prominent(material) || material == 2 || material == 3 || material == 4 || material >= 11 && material <= 16 {
						strength = 0.35
					}
					current[col].paper = mix(background, forestPalette[material], strength*light[material]/float64(count))
					break
				}
			}
			if figure {
				current[col] = c.figureCell(col, row)
			}
			if cover, covered := banner.at(row, col); covered {
				if cover.glyph != 0 {
					current[col] = cover
				} else {
					current[col].veil = cover.veil
				}
			}
		}
		previous := r.cells[row*cols : (row+1)*cols]
		visible := cols
		if r.inventoryOpen && row < panelHeight {
			visible = panelLeft
		}
		for col := 0; col < visible; {
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
				color, glyph := forestPalette[cell.ink], rune(0x2800+int(cell.bits))
				paper := cell.paper
				if cell.solid {
					color, glyph = cell.color, quadrants[cell.bits]
				}
				if cell.glyph != 0 {
					color, paper, glyph = mix(background, color, float64(cell.veil)/16), background, cell.glyph
				} else if cell.veil != 0 {
					color = mix(color, background, veilDepth*float64(cell.veil)/16)
					paper = mix(paper, background, veilDepth*float64(cell.veil)/16)
				}
				if paper != lastPaper {
					fmt.Fprintf(&out, "\x1b[48;2;%d;%d;%dm", paper[0], paper[1], paper[2])
					lastPaper = paper
				}
				if color != lastColor {
					fmt.Fprintf(&out, "\x1b[38;2;%d;%d;%dm", color[0], color[1], color[2])
					lastColor = color
				}
				out.WriteRune(glyph)
			}
		}
	}
	out.WriteString("\x1b[48;2;9;15;14m")
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
		fmt.Fprintf(&out, "\x1b[%d;1H\x1b[2K\x1b[38;2;147;166;130m%s", height, status[:min(width-1, len(status))])
		r.lastStatus = status
	}
	return out.String()
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
		height := c.heightAt(wx, wz) - c.eye
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
			c.pixels[y*c.width+x] = forestPixel{-point.toward*0.69 - point.height*0.72, light * quiet, ink, 0}
			c.under[y*c.width+x] = groundPoint{point.x, point.z}
		}
		previous = current
	}
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
		ground = r.built.underfoot(wx, wz, ground)
		switch {
		case !onMap:
			ink, light = 0, 0
		case ground == world.Path:
			ink, light = pathInk, 0.24+n*0.22
			// Pebbles catch the light here and there along the track.
			if f.Noise(wx*3, wz*3, 23) > 0.78 {
				light += 0.3
			}
		case ground == world.Floor || isWall(ground) || ground == world.Door:
			ink, light = floorInk, flagstones(wx, wz, n)
		case ground == world.Boards:
			ink, light = boardsInk, boards(wx, wz, n)
		case ground == world.Bridge:
			// The deck's boards lie across the way, as an east-west bridge carries it.
			ink, light = boardsInk, boards(wz, wx, n)
		}
		water = ground == world.Water
		if ground == world.Grass && r.built.banked(wx, wz) {
			ink, light = bankInk, 0.26+n*0.2
		}
		if onMap {
			shade := r.built.shade(wx, wz)
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
	normal := vec{height - c.elevation(x+reach, z), reach, height - c.elevation(x, z+reach)}.unit()
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
