package world

import (
	"math"
	"time"
)

// TreeKind is a species of tree, each with its own woodcutting requirements and rewards.
type TreeKind byte

const (
	NormalTree TreeKind = iota
	Oak
	Willow
	Yew
)

func (k TreeKind) String() string {
	return [...]string{"tree", "oak", "willow", "yew"}[k]
}

// treeSpecies holds Old School RuneScape's woodcutting values for one kind of tree.
type treeSpecies struct {
	level int
	xp    float64
	// low and high are the bronze-axe success rolls out of 256 at levels 1 and 99.
	low, high float64
	// fallsWithFirstLog marks trees that never give a second log; the others fall with the
	// first log after lifetime of chopping.
	fallsWithFirstLog bool
	lifetime          time.Duration
	// respawn is how long a stump takes to regrow, up to respawnSpread longer at random.
	respawn, respawnSpread time.Duration
}

var species = [...]treeSpecies{
	NormalTree: {level: 1, xp: 25, low: 64, high: 200, fallsWithFirstLog: true, respawn: 36 * time.Second, respawnSpread: 24 * time.Second},
	Oak:        {level: 15, xp: 37.5, low: 32, high: 100, lifetime: 45 * gameTick, respawn: 14 * gameTick},
	Willow:     {level: 30, xp: 67.5, low: 16, high: 50, lifetime: 50 * gameTick, respawn: 14 * gameTick},
	Yew:        {level: 60, xp: 175, low: 4, high: 12, lifetime: 190 * gameTick, respawn: 99 * gameTick},
}

// chance is the chance one tick cuts a log at a level, rolled as Old School RuneScape does.
func (s treeSpecies) chance(level int) float64 {
	l := float64(level)
	return (1 + math.Floor(s.low*(maxLevel-l)/(maxLevel-1)+s.high*(l-1)/(maxLevel-1)+0.5)) / 256
}

// treeKind picks the species growing at a spot: willows by rivers, oaks and rare yews in
// deep woods, and normal trees everywhere else.
func (w *World) treeKind(gx, gz int, x, z float64) TreeKind {
	if distance, width := w.River(x, z); distance < width+4 {
		return Willow
	}
	if w.Cover(x, z) > 0.6 {
		if w.Random(gx, gz, 8) < 0.04 {
			return Yew
		}
		return Oak
	}
	return NormalTree
}
