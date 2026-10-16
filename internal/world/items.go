package world

// Item is a kind of thing the hero can carry in the inventory or leave on the ground.
type Item byte

const (
	Logs Item = iota
	OakLogs
	WillowLogs
	YewLogs
)

// Landing is the sound an item makes as it hits the ground.
type Landing byte

const (
	WoodenThud Landing = iota
)

// itemDetails describes one kind of item.
type itemDetails struct {
	name    string
	landing Landing
}

var items = [...]itemDetails{
	Logs:       {name: "logs", landing: WoodenThud},
	OakLogs:    {name: "oak logs", landing: WoodenThud},
	WillowLogs: {name: "willow logs", landing: WoodenThud},
	YewLogs:    {name: "yew logs", landing: WoodenThud},
}

func (i Item) String() string { return items[i].name }

// Landing is the sound the item makes when dropped.
func (i Item) Landing() Landing { return items[i].landing }
