// Package window shows the world in a desktop window drawn with Ebitengine: pixel art sprites over
// a painted ground, walked with the keyboard and the mouse.
package window

import (
	"errors"
	"fmt"
	"image"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"lapresregne/internal/soundtrack"
	"lapresregne/internal/world"
)

// ticksPerSecond is how often the world advances, a tick at a time.
const ticksPerSecond = 60

// controls are what the player asks for in one tick.
type controls struct {
	// dx, dz is the held walking direction in the world, each from -1 to 1.
	dx, dz float64
	sprint bool
	// cursorX, cursorY is the pointer on the screen, in art pixels; pointing tells whether it is
	// over the window.
	cursorX, cursorY int
	pointing         bool
	click, quit      bool
	// track is 1 to skip to the next track, -1 for the previous one, and 0 to keep playing.
	track int
}

// readControls reads the keyboard and mouse. Up the screen is north, toward lower z.
func readControls() controls {
	var c controls
	held := func(keys ...ebiten.Key) bool {
		for _, key := range keys {
			if ebiten.IsKeyPressed(key) {
				return true
			}
		}
		return false
	}
	if held(ebiten.KeyArrowLeft, ebiten.KeyA, ebiten.KeyJ) {
		c.dx--
	}
	if held(ebiten.KeyArrowRight, ebiten.KeyD, ebiten.KeyL) {
		c.dx++
	}
	if held(ebiten.KeyArrowUp, ebiten.KeyW, ebiten.KeyI) {
		c.dz--
	}
	if held(ebiten.KeyArrowDown, ebiten.KeyS, ebiten.KeyK) {
		c.dz++
	}
	c.sprint = held(ebiten.KeyShift)
	c.cursorX, c.cursorY = ebiten.CursorPosition()
	c.pointing = image.Pt(c.cursorX, c.cursorY).In(image.Rect(0, 0, screenWidth, screenHeight))
	c.click = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	c.quit = inpututil.IsKeyJustPressed(ebiten.KeyQ) || inpututil.IsKeyJustPressed(ebiten.KeyEscape)
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		c.track = 1
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		c.track = -1
	}
	return c
}

// game is the world shown in the window.
type game struct {
	world *world.World
	music soundtrack.Player
	view  camera
	art   *sprites
	scene []placed
	// drawn holds each sprite's art once it has been sent to the graphics card, and chunks the
	// painted ground near the camera.
	drawn  map[*image.RGBA]*ebiten.Image
	chunks map[image.Point]*ebiten.Image
}

func newGame(w *world.World, music soundtrack.Player) *game {
	x, z := w.Hero().Position()
	return &game{
		world:  w,
		music:  music,
		view:   camera{x, z},
		art:    newSprites(),
		drawn:  map[*image.RGBA]*ebiten.Image{},
		chunks: map[image.Point]*ebiten.Image{},
	}
}

// step carries out one tick of the player's controls: a click walks to the ground clicked or to a
// tree to chop it, held keys walk, and the hero faces the pointer.
func (g *game) step(c controls, dt time.Duration) error {
	if g.music != nil {
		if err := g.music.Err(); err != nil {
			return fmt.Errorf("soundtrack (use --mute for silent play): %w", err)
		}
		if c.track != 0 {
			g.music.Skip(c.track)
		}
	}
	if c.click && c.pointing {
		if tree, ok := treeAt(g.scene, c.cursorX, c.cursorY); ok {
			g.world.Approach(tree)
		} else {
			g.world.WalkTo(g.view.snapped().toWorld(float64(c.cursorX)+0.5, float64(c.cursorY)+0.5))
		}
	}
	g.world.Advance(c.dx, c.dz, dt, c.sprint)
	if c.pointing {
		g.world.AimAt(g.view.snapped().toWorld(float64(c.cursorX)+0.5, float64(c.cursorY)+0.5))
	}
	x, z := g.world.Hero().Position()
	g.view.follow(x, z, dt.Seconds())
	g.scene = arrange(g.world, g.view.snapped(), g.art)
	return nil
}

func (g *game) Update() error {
	c := readControls()
	if c.quit {
		return ebiten.Termination
	}
	return g.step(c, time.Second/ticksPerSecond)
}

func (g *game) Draw(screen *ebiten.Image) {
	screen.Fill(voidColor)
	g.drawGround(screen)
	for _, p := range g.scene {
		sprite, ok := g.drawn[p.art]
		if !ok {
			sprite = ebiten.NewImageFromImage(p.art)
			g.drawn[p.art] = sprite
		}
		options := &ebiten.DrawImageOptions{}
		options.GeoM.Translate(float64(p.x), float64(p.y))
		screen.DrawImage(sprite, options)
	}
}

// drawGround draws the chunks of ground the camera shows, painting any not painted yet, and lets go
// of those it has left behind.
func (g *game) drawGround(screen *ebiten.Image) {
	originX, originY := g.view.snapped().origin()
	first := image.Pt(floorDiv(originX, chunkSize), floorDiv(originY, chunkSize))
	last := image.Pt(floorDiv(originX+screenWidth, chunkSize), floorDiv(originY+screenHeight, chunkSize))
	for at, chunk := range g.chunks {
		if at.X < first.X-1 || at.X > last.X+1 || at.Y < first.Y-1 || at.Y > last.Y+1 {
			chunk.Deallocate()
			delete(g.chunks, at)
		}
	}
	for cy := first.Y; cy <= last.Y; cy++ {
		for cx := first.X; cx <= last.X; cx++ {
			at := image.Pt(cx, cy)
			chunk, ok := g.chunks[at]
			if !ok {
				chunk = ebiten.NewImageFromImage(paintChunk(g.world, cx, cy))
				g.chunks[at] = chunk
			}
			options := &ebiten.DrawImageOptions{}
			options.GeoM.Translate(float64(cx*chunkSize-originX), float64(cy*chunkSize-originY))
			screen.DrawImage(chunk, options)
		}
	}
}

func (g *game) Layout(int, int) (int, int) { return screenWidth, screenHeight }

// Run opens a window on the world and lets the traveler explore it until they quit or close the
// window. Music is nil when muted.
func Run(w *world.World, music soundtrack.Player, title string) error {
	ebiten.SetWindowTitle(title)
	ebiten.SetWindowSize(screenWidth*3, screenHeight*3)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetTPS(ticksPerSecond)
	err := ebiten.RunGame(newGame(w, music))
	if errors.Is(err, ebiten.Termination) {
		return nil
	}
	return err
}
