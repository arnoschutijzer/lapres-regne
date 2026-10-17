package terminal

// iconCells and iconLines size an item's icon in a slot: 4 cells across and 2 lines down. Each cell
// shows two square pixels, one above the other, so an icon is 4 by 4 pixels of any colour.
const (
	iconCells = 4
	iconLines = 2
)

// sprite is pixel art for an icon, a row of letters to a row of pixels. Each letter names a colour
// in the item's palette; any other letter leaves the background.
type sprite []string

// pixels colours a sprite from a palette.
func (s sprite) pixels(palette map[byte][3]int) [][][3]int {
	pixels := make([][][3]int, len(s))
	for y, row := range s {
		pixels[y] = make([][3]int, len(row))
		for x := range len(row) {
			color, ok := palette[row[x]]
			if !ok {
				color = background
			}
			pixels[y][x] = color
		}
	}
	return pixels
}

// logSprite is a log lying from the lower left up to its cut end at the upper right, lit along its
// upper edge, as Old School RuneScape draws it: h is lit bark, b bark, d bark in shadow, e the cut
// end's pale wood and r its darker heart.
var logSprite = sprite{
	"..ee",
	".her",
	"hbd.",
	"bd..",
}

// logPixels colours the log sprite from a log's tint.
func logPixels(tint [3]int) [][][3]int {
	end := cutEnd(tint)
	return logSprite.pixels(map[byte][3]int{
		'h': mix(tint, [3]int{255, 250, 235}, 0.2),
		'b': tint,
		'd': mix(background, tint, 0.5),
		'e': end,
		'r': mix(end, tint, 0.7),
	})
}

// renderIcon draws pixels line by line and cell by cell as upper half blocks, the upper pixel in
// the foreground and the lower in the paper.
func renderIcon(pixels [][][3]int) [][]string {
	rendered := make([][]string, len(pixels)/2)
	for line := range rendered {
		rendered[line] = make([]string, len(pixels[line*2]))
		for column := range rendered[line] {
			rendered[line][column] = foreground(pixels[line*2][column]) + paperOf(pixels[line*2+1][column]) + "▀"
		}
	}
	return rendered
}
