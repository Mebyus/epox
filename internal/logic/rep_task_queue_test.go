package logic

import "testing"

func TestTheap(t *testing.T) {
	var h theap
	h.init(5) // []

	n := h.num()
	if n != 0 {
		t.Errorf("want num() = %d, got %d", 0, n)
		return
	}

	_, ok := h.pop() // []
	if ok {
		t.Error("want empty heap")
		return
	}
	_, ok = h.pop() // []
	if ok {
		t.Error("want empty heap")
		return
	}

	ent1 := tent{id: 1, next: 1}
	h.push(ent1) // [1]
	n = h.num()
	if n != 1 {
		t.Errorf("want num() = %d, got %d", 1, n)
		return
	}

	ent, ok := h.pop() // []
	if !ok || ent != ent1 {
		t.Errorf("want pop() = %v, got %v", ent1, ent)
		return
	}
	n = h.num()
	if n != 0 {
		t.Errorf("want num() = %d, got %d", 0, n)
		return
	}

	_, ok = h.pop() // []
	if ok {
		t.Error("want empty heap")
		return
	}

	ent2 := tent{id: 2, next: 2}
	h.push(ent2) // [2]
	ent, ok = h.peek()
	if !ok || ent != ent2 {
		t.Errorf("want peek() = %v, got %v", ent2, ent)
		return
	}
	h.push(ent1) // [1, 2]
	n = h.num()
	if n != 2 {
		t.Errorf("want num() = %d, got %d", 2, n)
		return
	}

	ent, ok = h.pop() // [2]
	if !ok || ent != ent1 {
		t.Errorf("want pop() = %v, got %v", ent1, ent)
		return
	}
	ent, ok = h.pop() // []
	if !ok || ent != ent2 {
		t.Errorf("want pop() = %v, got %v", ent2, ent)
		return
	}

	ent3 := tent{id: 3, next: 3}
	ent4 := tent{id: 4, next: 4}
	ent5 := tent{id: 5, next: 5}
	h.push(ent4) // [4]
	h.push(ent5) // [4, 5]
	h.push(ent4) // [4, 4, 5]
	h.push(ent2) // [2, 4, 4, 5]
	h.push(ent1) // [1, 2, 4, 4, 5]
	h.push(ent1) // [1, 1, 2, 4, 4, 5]
	h.push(ent3) // [1, 1, 2, 3, 4, 4, 5]
	n = h.num()
	if n != 7 {
		t.Errorf("want num() = %d, got %d", 7, n)
		return
	}

	ent, ok = h.pop() // [1, 2, 3, 4, 4, 5]
	if !ok || ent != ent1 {
		t.Errorf("want pop() = %v, got %v", ent1, ent)
		return
	}
	ent, ok = h.pop() // [2, 3, 4, 4, 5]
	if !ok || ent != ent1 {
		t.Errorf("want pop() = %v, got %v", ent1, ent)
		return
	}
	ent, ok = h.pop() // [3, 4, 4, 5]
	if !ok || ent != ent2 {
		t.Errorf("want pop() = %v, got %v", ent2, ent)
		return
	}
	ent, ok = h.pop() // [4, 4, 5]
	if !ok || ent != ent3 {
		t.Errorf("want pop() = %v, got %v", ent3, ent)
		return
	}
	ent, ok = h.peek() // [4, 4, 5]
	if !ok || ent != ent4 {
		t.Errorf("want peek() = %v, got %v", ent4, ent)
		return
	}
	n = h.num()
	if n != 3 {
		t.Errorf("want num() = %d, got %d", 3, n)
		return
	}
	ent, ok = h.pop() // [4, 5]
	if !ok || ent != ent4 {
		t.Errorf("want pop() = %v, got %v", ent4, ent)
		return
	}
	ent, ok = h.pop() // [5]
	if !ok || ent != ent4 {
		t.Errorf("want pop() = %v, got %v", ent4, ent)
		return
	}
	ent, ok = h.pop() // []
	if !ok || ent != ent5 {
		t.Errorf("want pop() = %v, got %v", ent5, ent)
		return
	}
	_, ok = h.pop() // []
	if ok {
		t.Error("want empty heap")
		return
	}
}
