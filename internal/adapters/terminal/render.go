package terminal

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

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
	cells                     []forestCell
	screenWidth, screenHeight int
	lastStatus, lastHint      string
	lastSkill                 string
	// inventoryOpen shows the inventory panel; panelShown and lastPanel remember the last one drawn.
	inventoryOpen, panelShown bool
	lastPanel                 string
	cursor                    slotCursor
	// menu lists the actions offered for the tree in reach, named by menuTitle; nil when closed.
	menu      []string
	menuTitle string
	zoom      zoom
	gait      gait
	// lastSeconds is when the last frame was rendered, so the gait can follow the time between frames.
	lastSeconds float64
}

// newRenderer opens the view at the camera's zoom, so a resumed session starts where it was left.
func newRenderer(world *world.World, title string) *renderer {
	return &renderer{world: world, title: title, zoom: zoom{current: float64(world.Zoom())}}
}

type forestPixel struct {
	depth, light float64
	ink          int
}

type forestCell struct {
	bits, ink uint8
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
}

// foremost marks the inks of the hero, their axe and flying chips.
func foremost(ink int) bool {
	return ink >= 8 && ink <= 10 || ink >= defaultLook.skin
}

func (c *forestCanvas) project(x, y, z float64) (float64, float64, float64) {
	return float64(c.width)/2 + (x-c.x)*c.scale,
		float64(c.height)*0.57 + ((z-c.z)*0.72-y*0.69)*c.scale,
		-(z-c.z)*0.69 - y*0.72
}

// ground inverts the projection at ground level for the canvas pixel x, y.
func (c *forestCanvas) ground(x, y float64) (float64, float64) {
	return c.x + (x-float64(c.width)/2)/c.scale,
		c.z + (y-float64(c.height)*0.57)/(0.72*c.scale)
}

func (c *forestCanvas) plot(x, y int, depth, light float64, ink int) {
	if x < 0 || y < 0 || x >= c.width || y >= c.height {
		return
	}
	i := y*c.width + x
	if depth <= c.pixels[i].depth {
		c.pixels[i] = forestPixel{depth, light, ink}
	}
}

func (c *forestCanvas) line(x1, y1, z1, x2, y2, z2, light float64, ink int) {
	ax, ay, ad := c.project(x1, y1, z1)
	bx, by, bd := c.project(x2, y2, z2)
	steps := max(1, int(math.Ceil(math.Max(math.Abs(bx-ax), math.Abs(by-ay)))))
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
	if len(c.pixels) != c.width*c.height {
		c.pixels = make([]forestPixel, c.width*c.height)
	}
	left := c.x - float64(c.width)/c.scale/2 - 5
	right := c.x + float64(c.width)/c.scale/2 + 5
	back := c.z - float64(c.height)/c.scale - 5
	front := c.z + float64(c.height)/c.scale + 12
	trees := r.trees[:0]
	for gz := int(math.Floor(back / world.TreeSpacing)); gz <= int(math.Ceil(front/world.TreeSpacing)); gz++ {
		for gx := int(math.Floor(left / world.TreeSpacing)); gx <= int(math.Ceil(right/world.TreeSpacing)); gx++ {
			if tree, exists := f.TreeAt(gx, gz); exists {
				trees = append(trees, tree)
			}
		}
	}
	r.trees = trees
	// Front-to-back drawing lets hidden crown pixels skip lighting calculations.
	sort.Slice(trees, func(i, j int) bool {
		return trees[i].Z*0.69+trees[i].Height*0.72 > trees[j].Z*0.69+trees[j].Height*0.72
	})
	channels := make([]world.RiverChannel, 0, 6)
	for y := range c.height {
		wz := (float64(y)-float64(c.height)*0.57)/(c.scale*0.72) + c.z
		channels = f.RiverChannels(channels[:0], wz, left, right)
		for x := range c.width {
			wx := (float64(x)-float64(c.width)/2)/c.scale + c.x
			n := f.Noise(wx*0.65, wz*0.65, 19)
			cover := f.Cover(wx, wz)
			ink, light := 0, 0.13+n*0.23
			if cover < 0.38 {
				ink, light = 1, 0.21+n*0.24
			}
			// Diagonal pools of sunlight survive only where the canopy opens.
			beam := math.Max(0, math.Sin(wx*0.28-wz*0.19+f.Noise(wx/9, wz/9, 81)*2)-0.45)
			sun := beam * math.Max(0, 1-cover*1.3)
			light += sun * 0.8
			if sun > 0.24 {
				ink = 7
			}
			distance, width := world.NearestRiver(channels, wx)
			if distance < width {
				ink, light = 5, 0.28+n*0.3
				if math.Sin(wz*2.6+wx*0.8-seconds*1.8+n*1.3) > 0.91 {
					ink, light = 6, 0.8
				}
			} else if distance < width+0.65 {
				ink, light = 7, 0.32+n*0.25
			}
			c.pixels[y*c.width+x] = forestPixel{-(wz - c.z) * 0.69, light, ink}
		}
	}
	for _, tree := range trees {
		if !tree.Stump {
			c.shadow(tree)
		}
	}
	chopped, chopping := f.ChoppedTree()
	strike, sinceStrike, _ := f.LastStrike()
	sinceHit := sinceStrike.Seconds()
	for _, tree := range trees {
		shake := 0.0
		if chopping && tree == chopped {
			shake = jolt(sinceHit)
		}
		r.drawTree(c, tree, shake)
	}
	if chopping {
		drawChips(c, f, chopped, strike, sinceHit)
	}
	for _, item := range f.GroundItems() {
		drawGroundItem(c, item)
	}
	r.drawCharacter(c, seconds)
	r.drawGrass(c, left, right, back, front, seconds)
	return c
}

func (r *renderer) drawCharacter(c *forestCanvas, seconds float64) {
	x, z := r.world.Hero().Position()
	stride := 0.0
	if time.Since(r.world.Hero().LastWalk()) < 200*time.Millisecond {
		stride = math.Sin(seconds*18) * 0.24
	}
	strideX, strideZ := 0.0, stride
	if facing := r.world.Hero().Facing(); facing == world.FacingLeft || facing == world.FacingRight {
		strideX, strideZ = stride, 0
	}
	r.gait.update(x, z, r.world.Hero().Facing(), min(0.1, max(0, seconds-r.lastSeconds)))
	r.lastSeconds = seconds
	if c.scale >= heroDetailScale {
		drawDetailedHero(c, x, z, r.gait, defaultLook)
	} else {
		// A small warm-colored traveler: boots, coat, arms, and a round hat.
		c.line(x-0.16+strideX, 0.05, z+strideZ, x-0.12, 0.7, z, 1, 8)
		c.line(x+0.16-strideX, 0.05, z-strideZ, x+0.12, 0.7, z, 1, 8)
		c.line(x, 0.55, z, x, 1.15, z, 1, 8)
		c.line(x-0.34, 0.65, z-stride, x+0.34, 1.0, z+stride, 0.95, 8)
		for i := range 8 {
			a := float64(i) * math.Pi / 4
			c.line(x, 1.48, z, x+math.Cos(a)*0.28, 1.48, z+math.Sin(a)*0.28, 1, 8)
		}
	}
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
func (r *renderer) treeAt(column, row int) (world.Tree, bool) {
	c := &r.canvas
	x, y, ok := r.canvasPoint(column, row)
	if !ok {
		return world.Tree{}, false
	}
	// Trees are kept front to back from the last render, so the first outline hit is in front.
	for _, tree := range r.trees {
		if tree.Stump {
			continue
		}
		baseX, baseY, _ := c.project(tree.X, 0, tree.Z)
		_, topY, _ := c.project(tree.X, tree.Height+0.3+tree.Radius*0.7, tree.Z)
		halfWidth := tree.Radius * 1.3 * c.scale
		if x >= baseX-halfWidth && x <= baseX+halfWidth && y >= topY && y <= baseY {
			return tree, true
		}
	}
	return world.Tree{}, false
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
		// Right-align the skill on the title row without ever covering the title.
		skillRunes := []rune(skill)
		start := max(len([]rune(r.title))+5, width-len(skillRunes))
		if start < width {
			fmt.Fprintf(&out, "\x1b[1;%dH\x1b[K\x1b[38;2;147;166;130m%s", start+1, string(skillRunes[:min(width-start, len(skillRunes))]))
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
	for row := range rows {
		var current [maxForestColumns]forestCell
		for col := range cols {
			bits, ink := 0, 0
			var votes [len(forestPalette)]int
			for dy := range 4 {
				for dx := range 2 {
					x, y := col*2+dx, row*4+dy
					p := c.pixels[y*c.width+x]
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
				if score > best {
					ink, best = i, score
				}
			}
			current[col] = forestCell{bits: uint8(bits), ink: uint8(ink)}
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
				if cell.bits == 0 && cell.glyph == 0 {
					out.WriteByte(' ')
					continue
				}
				color, glyph := forestPalette[cell.ink], rune(0x2800+int(cell.bits))
				if cell.glyph != 0 {
					color, glyph = mix(background, color, float64(cell.veil)/16), cell.glyph
				} else if cell.veil != 0 {
					color = mix(color, background, veilDepth*float64(cell.veil)/16)
				}
				if color != lastColor {
					fmt.Fprintf(&out, "\x1b[38;2;%d;%d;%dm", color[0], color[1], color[2])
					lastColor = color
				}
				out.WriteRune(glyph)
			}
		}
	}
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
	status += fmt.Sprintf("  |  seed %d", r.world.Seed())
	if full || status != r.lastStatus {
		fmt.Fprintf(&out, "\x1b[%d;1H\x1b[2K\x1b[38;2;147;166;130m%s", height, status[:min(width-1, len(status))])
		r.lastStatus = status
	}
	return out.String()
}
