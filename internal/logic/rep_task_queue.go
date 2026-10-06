package logic

import (
	"time"

	"github.com/mebyus/epox/internal/base"
)

// Time queue for repeatable task triggers.
//
// Keeps track of next trigger time for each repeatable task.
// When time for next nearest trigger comes queue wakes up
// and creates a new task, then places next trigger time into
// queue.
type tqueue struct {
	entries theap

	// fires when next trigger comes
	timer *time.Timer

	enq chan tent
	deq chan tent
}

// trigger entry for repeatable task
type tent struct {
	id base.RepTaskID

	// time of next trigger
	next base.MicroTime
}

func (q *tqueue) init() {
	q.entries.init(64)
	q.timer = time.NewTimer(0)
	q.enq = make(chan tent, 16)
	q.deq = make(chan tent, 1)
	go q.watch()
}

func (q *tqueue) put(ent tent) {
	q.enq <- ent
}

// blocks and waits for next nearest trigger.
func (q *tqueue) next() tent {
	return <-q.deq
}

func (q *tqueue) watch() {
	for {
		select {
		case <-q.timer.C:
			ent, ok1 := q.entries.pop()
			top, ok2 := q.entries.peek()
			if ok2 {
				q.reset(top)
			}
			if ok1 {
				q.deq <- ent
			}
			// TODO: do something if heap was empty on pop, situation is abnormal
		case ent := <-q.enq:
			// TODO: check that new entry is not expired
			q.entries.push(ent)
			top, _ := q.entries.peek()
			if top == ent {
				q.reset(top)
			}
		}
	}
}

func (q *tqueue) reset(ent tent) {
	nowts := time.Now().UnixMicro()
	if ent.next > base.MicroTime(nowts) {
		delay := ent.next - base.MicroTime(nowts)
		d := time.Microsecond * time.Duration(delay)
		q.timer.Reset(d)
	}
}

// Implements heap data structure for repeatable task triggers.
//
// Heap complete state is when each parent has next trigger
// time less or equal then both of its children.
type theap struct {
	items []tent
}

func (h *theap) init(initsize int) {
	h.items = make([]tent, 0, initsize)
}

// Returns number of stored items.
//
// Guaranteed to have the same exact number of
// non-empty pops before heap becomes empty.
func (h *theap) num() int {
	return len(h.items)
}

// Add item to heap, while maintaining complete state.
func (h *theap) push(ent tent) {
	i := len(h.items) // starting index of newly pushed item
	h.items = append(h.items, ent)
	h.up(i)
}

func (h *theap) peek() (tent, bool) {
	if len(h.items) == 0 {
		return tent{}, false
	}
	return h.items[0], true
}

// Take top item from the heap. In complete heap top item has
// minimal time of next trigger.
//
// Returns ({...}, true) when heap had at least one item prior
// to this operation and thus the operation was successful.
//
// Returns ({}, false) when heap was empty and thus no items
// can be taken from heap. The operation does nothing in this
// case.
func (h *theap) pop() (tent, bool) {
	n := len(h.items)
	if n == 0 {
		return tent{}, false
	}

	top := h.items[0]
	h.items[0] = h.items[n-1]
	h.items = h.items[:n-1]
	if len(h.items) != 0 {
		h.down(0)
	}
	return top, true
}

// moves item at specified index until complete state of the heap.
func (h *theap) up(i int) {
	for i > 0 {
		p := (i - 1) / 2 // parent's index
		if h.items[p].next <= h.items[i].next {
			// move item up until parent is less or equal
			break
		}

		h.items[i], h.items[p] = h.items[p], h.items[i]
		i = p
	}
}

// moves item at specified index down until complete state of the heap.
func (h *theap) down(i int) {
	n := len(h.items)
	for {
		// keep in mind that child may not exist
		// if its index is outside of items list
		l := 2*i + 1 // index of left child
		r := 2*i + 2 // index of right child

		// chosen index where item will be moved
		//
		// we must find index with the smallest item among
		// the three items: parent (current item) and up to
		// two of its children
		s := i

		if l < n && h.items[l].next < h.items[s].next {
			s = l
		}
		if r < n && h.items[r].next < h.items[s].next {
			s = r
		}

		if s == i {
			// already the smallest item, stop here
			return
		}
		h.items[i], h.items[s] = h.items[s], h.items[i]
		i = s
	}
}
