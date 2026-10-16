package terminal

import (
	"math"
	"strconv"
	"strings"

	"dither-cli/internal/world"
)

// bannerFadeIn, bannerRest and bannerFadeOut pace the level-up banner, in seconds, to the jingle's six.
const bannerFadeIn, bannerRest, bannerFadeOut = 0.8, 4.0, 1.2

const (
	// bannerHeight is how many forest rows the banner covers, a quiet row above and below included.
	bannerHeight = 6
	// bannerPadding is the space between the banner's edges and its icon or text.
	bannerPadding = 2
	// bannerEdge is how many cells the veil takes to fade out toward the banner's sides.
	bannerEdge = 3
	// veilDepth is how far a fully veiled cell is darkened toward the background.
	veilDepth = 0.8
)

// woodcuttingIcon is a tree with a sunlit crown, after Old School RuneScape's skill icon:
// S marks shaded leaves, L sunlit leaves and T bark. Every 2 x 4 pixels make one braille cell.
var woodcuttingIcon = []string{
	"....SSLL....",
	"..SSSLLLLL..",
	".SSSSSLLLLL.",
	"SSSSSSSLLLLL",
	"SSSSSSSSLLLL",
	"SSSSSSSSSSSS",
	".SSSSSSSSSS.",
	"..SS.TT.SS..",
	".....TT.....",
	".....TTT....",
	"....TTTT....",
	"....TTTT....",
	"....TTTT....",
	"...TTTTTT...",
	"..TT.TT.TT..",
	"............",
}

// iconInk colors the icon's pixels with the forest's shaded leaves, sunlit leaves and bark.
var iconInk = map[byte]uint8{'S': 2, 'L': 3, 'T': 4}

// The banner's lines take the forest's sunlight, wood chip and sunlit willow colors.
const titleInk, levelInk, unlockInk = 7, 10, 12

// bannerAlpha is how visible the banner is some seconds after the level-up: it eases in, rests and
// eases out.
func bannerAlpha(since float64) float64 {
	ease := func(u float64) float64 { return u * u * (3 - 2*u) }
	switch {
	case since <= 0 || since >= bannerFadeIn+bannerRest+bannerFadeOut:
		return 0
	case since < bannerFadeIn:
		return ease(since / bannerFadeIn)
	case since <= bannerFadeIn+bannerRest:
		return 1
	default:
		return ease(1 - (since-bannerFadeIn-bannerRest)/bannerFadeOut)
	}
}

// overlay is drawn over the forest's top rows, centered: glyphs replace forest cells, and the
// forest elsewhere under it is veiled.
type overlay struct {
	left, width int
	// cells holds bannerHeight rows of width cells; veil is in sixteenths of full strength.
	cells []forestCell
}

// at is the overlay's cell over a forest cell, if the overlay covers it.
func (o *overlay) at(row, col int) (forestCell, bool) {
	if o == nil || row >= bannerHeight || col < o.left || col >= o.left+o.width {
		return forestCell{}, false
	}
	return o.cells[row*o.width+col-o.left], true
}

// levelUpBanner lays out the banner for the latest level-up across a forest cols wide, or nil
// when no banner shows.
func levelUpBanner(f *world.World, cols int) *overlay {
	levelUp, since, ok := f.LatestLevelUp()
	if !ok {
		return nil
	}
	alpha := bannerAlpha(since.Seconds())
	strength := uint8(math.Round(alpha * 16))
	if strength == 0 {
		return nil
	}
	type line struct {
		text string
		ink  uint8
	}
	lines := []line{
		{"Congratulations!", titleInk},
		{levelUp.Skill + " level " + strconv.Itoa(levelUp.Level), levelInk},
	}
	if len(levelUp.Unlocked) > 0 {
		names := make([]string, len(levelUp.Unlocked))
		for i, kind := range levelUp.Unlocked {
			names[i] = kind.String() + "s"
		}
		lines = append(lines, line{"You can now chop " + strings.Join(names, " and ") + ".", unlockInk})
	}
	icon := brailleArt(woodcuttingIcon, iconInk)
	iconWidth := len(woodcuttingIcon[0]) / 2
	textLeft := bannerPadding + iconWidth + 2
	width := 0
	for _, l := range lines {
		width = max(width, textLeft+len([]rune(l.text))+bannerPadding)
	}
	width = min(width, cols)
	o := &overlay{left: (cols - width) / 2, width: width, cells: make([]forestCell, bannerHeight*width)}
	// The veil softens toward the sides and the bottom row so the banner melts into the forest.
	for row := range bannerHeight {
		for col := range width {
			edge := math.Min(1, float64(min(col, width-1-col)+1)/float64(bannerEdge+1))
			if row == bannerHeight-1 {
				edge /= 2
			}
			o.cells[row*width+col].veil = uint8(math.Round(alpha * edge * 16))
		}
	}
	put := func(row, col int, glyph rune, ink uint8) {
		if col >= 0 && col < width {
			o.cells[row*width+col] = forestCell{ink: ink, veil: strength, glyph: glyph}
		}
	}
	drawSparks(o, since.Seconds(), levelUp.Level, strength)
	for row, cells := range icon {
		for col, cell := range cells {
			if cell.bits != 0 {
				put(1+row, bannerPadding+col, rune(0x2800+int(cell.bits)), cell.ink)
			}
		}
	}
	// The lines sit beside the icon, centered on its four rows.
	top := 1 + (len(icon)-len(lines))/2
	for i, l := range lines {
		for col, letter := range []rune(l.text) {
			put(top+i, textLeft+col, letter, l.ink)
		}
	}
	return o
}

// drawSparks lets a few warm sparks drift slowly up through the banner, one braille dot each.
func drawSparks(o *overlay, since float64, seed int, strength uint8) {
	const sparks, rise = 5, 2.5 // dots per second
	height := float64(bannerHeight * 4)
	for i := range sparks {
		spread := math.Mod(float64(i)*0.613+float64(seed)*0.271, 1)
		x := int(spread * float64(o.width*2))
		y := int(height - 1 - math.Mod(since*rise+float64(i)*height/sparks, height))
		row, col := y/4, x/2
		cell := &o.cells[row*o.width+col]
		cell.glyph |= 0x2800 | 1<<brailleDot[y%4][x%2]
		cell.ink, cell.veil = titleInk, strength
	}
}

// brailleArt turns pixel art into braille cells, each colored by the ink most of its pixels have.
func brailleArt(art []string, inks map[byte]uint8) [][]forestCell {
	rows := make([][]forestCell, (len(art)+3)/4)
	for row := range rows {
		rows[row] = make([]forestCell, (len(art[0])+1)/2)
		for col := range rows[row] {
			votes := map[uint8]int{}
			cell := &rows[row][col]
			for dy := range 4 {
				for dx := range 2 {
					y, x := row*4+dy, col*2+dx
					if y >= len(art) || x >= len(art[y]) {
						continue
					}
					if ink, ok := inks[art[y][x]]; ok {
						cell.bits |= 1 << brailleDot[dy][dx]
						votes[ink]++
						if votes[ink] > votes[cell.ink] {
							cell.ink = ink
						}
					}
				}
			}
		}
	}
	return rows
}
