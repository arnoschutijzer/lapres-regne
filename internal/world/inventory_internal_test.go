package world

import "testing"

func TestSlotsStayPutAndNewLogsFillTheFirstGap(t *testing.T) {
	var inventory Inventory
	for _, kind := range []TreeKind{Oak, Willow, Yew} {
		inventory.add(kind)
	}
	if kind, ok := inventory.drop(1); !ok || kind != Willow {
		t.Fatalf("dropped %v, %v; want the willow log", kind, ok)
	}
	slots := inventory.Slots()
	if slots[0].Kind != Oak || slots[1].Filled || slots[2].Kind != Yew || inventory.Len() != 2 {
		t.Fatalf("slots = %v, want the oak and yew logs to stay in their slots around a gap", slots[:3])
	}
	inventory.add(NormalTree)
	if slots := inventory.Slots(); slots[1] != (Slot{Kind: NormalTree, Filled: true}) {
		t.Fatalf("slot 1 = %v, want the new log in the first gap", slots[1])
	}
	if first, ok := inventory.FirstFilled(); !ok || first != 0 {
		t.Fatalf("first filled = %d, %v; want slot 0", first, ok)
	}
}
