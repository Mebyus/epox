package logic

import (
	"fmt"
	"time"

	"github.com/mebyus/epox/internal/base"
)

// debug repeatable tasks queue
const tdebug = false

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

func newTimer() *time.Timer {
	timer := time.NewTimer(0)

	// drain timer since zero or negative
	// initial duration will cause it to fire almost
	// immediately
	//
	// we want "empty" (which waits for reset to fire)
	// timer for our queue, since initial queue state
	// is empty queue
	<-timer.C

	return timer
}

func (q *tqueue) init() {
	q.entries.init(64)
	q.timer = newTimer()
	q.enq = make(chan tent, 16)
	q.deq = make(chan tent, 1)
	go q.watch()

	if tdebug {
		start := time.Now()
		fmt.Printf("[tqueue] start: %v\n", start)

		q.put(tent{id: 1, next: base.FromNow(3 * time.Second)})
		q.put(tent{id: 2, next: base.FromNow(8 * time.Second)})
		q.put(tent{id: 3, next: base.FromNow(5 * time.Second)})
		q.put(tent{id: 4, next: base.FromNow(2 * time.Second)})
		q.put(tent{id: 5, next: base.FromNow(0 * time.Second)})

		// test when trigger instants are equal
		// for multiple entries
		ts := base.FromNow(5 * time.Second)
		q.put(tent{id: 6, next: ts})
		q.put(tent{id: 7, next: ts})
		q.put(tent{id: 8, next: ts})

		go func() {
			for {
				ent := q.next()
				fmt.Printf("[tqueue] %d: %v\n", ent.id, time.Since(start))
			}
		}()
	}
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
			q.tick()
		case ent := <-q.enq:
			q.handle(ent)
		}
	}
}

// handles new entries in watch loop
func (q *tqueue) handle(ent tent) {
	nowts := base.Now()
	if ent.next <= nowts {
		// trigger instantly if its target instant already passed
		q.deq <- ent
		return
	}

	q.entries.push(ent)
	top, _ := q.entries.peek() // no need for flag since at least one entry is already stored

	// here we check if top entry has changed to the new entry
	if top != ent {
		// if no change in top entry occured then
		// no need to reschedule timer
		return
	}

	// check delay again since heap push is potentially
	// a slow operation
	const margin = 2 // in microseconds
	nowts = base.Now()
	if ent.next > nowts+margin {
		delay := ent.next - nowts
		d := time.Duration(delay) * time.Microsecond
		q.timer.Reset(d)
		return
	}

	// trigger instant already passed while we were dealing
	// with heap operations, send entry now and remove it from
	// heap
	q.deq <- ent
	q.entries.pop()
}

// handles timer tick in watch loop
func (q *tqueue) tick() {
	ent, ok := q.entries.pop()
	if !ok {
		// internal queue is empty on tick
		//
		// situation is abnormal if it happens on initial
		// loop iteration because ticks must be scheduled
		// only if entries are still present in internal queue
		//
		// nothing left to do here
		//
		// TODO: should we report this or just panic?
		return
	}

	// drain internal queue until we find top element
	// that should trigger strictly after now instant
	for {
		top, ok := q.entries.peek()
		if !ok {
			// internal queue is empty after pop on a tick
			//
			// skip timer scheduling to render it inactive
			// until new entry comes from queue client
			//
			// thus we only need to send poped entry
			q.deq <- ent
			return
		}

		// algrorithm in this function may look a bit weird
		// in its logic
		//
		// it is written this way to enshure minimal time
		// between checking scheduling delay and actually
		// resetting the timer
		//
		// note that we send last popped entry only after
		// resetting the timer
		const margin = 10 // in microseconds
		nowts := base.Now()
		if top.next > nowts+margin {
			delay := top.next - nowts
			dur := time.Duration(delay) * time.Microsecond
			q.timer.Reset(dur)
			q.deq <- ent
			return
		}

		q.deq <- ent
		ent, _ = q.entries.pop() // ignore flag due to previous peek() check
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
