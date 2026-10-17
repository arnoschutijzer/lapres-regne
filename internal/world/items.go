package world

// Item is a kind of thing the hero can carry in the inventory or leave on the ground.
type Item byte

const (
	Logs Item = iota
	OakLogs
	WillowLogs
	YewLogs
)

// Shape is the silhouette an item is drawn with, in the inventory and on the ground.
type Shape byte

const (
	LogShape Shape = iota
)

// Color is an item's tint as red, green and blue.
type Color struct{ R, G, B uint8 }

// itemDetails describes one kind of item.
type itemDetails struct {
	name    string
	shape   Shape
	color   Color
	landing Effect
}

var items = [...]itemDetails{
	Logs:       {name: "logs", shape: LogShape, color: Color{196, 170, 120}, landing: WoodenThud},
	OakLogs:    {name: "oak logs", shape: LogShape, color: Color{178, 128, 74}, landing: WoodenThud},
	WillowLogs: {name: "willow logs", shape: LogShape, color: Color{150, 158, 112}, landing: WoodenThud},
	YewLogs:    {name: "yew logs", shape: LogShape, color: Color{132, 84, 60}, landing: WoodenThud},
}

func (i Item) String() string { return items[i].name }

// Shape is the silhouette the item is drawn with.
func (i Item) Shape() Shape { return items[i].shape }

// Color is the tint the item is drawn in.
func (i Item) Color() Color { return items[i].color }

// Landing is the sound the item makes when dropped.
func (i Item) Landing() Effect { return items[i].landing }
