package world

import "testing"

func TestSlotsStayPutAndNewItemsFillTheFirstGap(t *testing.T) {
	var inventory Inventory
	for _, item := range []Item{OakLogs, WillowLogs, YewLogs} {
		inventory.add(item)
	}
	if item, ok := inventory.drop(1); !ok || item != WillowLogs {
		t.Fatalf("dropped %v, %v; want the willow logs", item, ok)
	}
	slots := inventory.Slots()
	if slots[0].Item != OakLogs || slots[1].Filled || slots[2].Item != YewLogs || inventory.Len() != 2 {
		t.Fatalf("slots = %v, want the oak and yew logs to stay in their slots around a gap", slots[:3])
	}
	inventory.add(Logs)
	if slots := inventory.Slots(); slots[1] != (Slot{Item: Logs, Filled: true}) {
		t.Fatalf("slot 1 = %v, want the new log in the first gap", slots[1])
	}
	if first, ok := inventory.FirstFilled(); !ok || first != 0 {
		t.Fatalf("first filled = %d, %v; want slot 0", first, ok)
	}
}
