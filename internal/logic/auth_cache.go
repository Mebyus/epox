package logic

import (
	"sync"

	"github.com/mebyus/epox/internal/base"
)

// *
// session cache capacity
const scap = 1 << 18
const debug = false

//*/

/*
const scap = 1 << 4
const debug = true

//*/

// Session cache node index encoded with valid bit.
//
// Highest bit is set to 1 if link is valid.
// Highest bit is set to 0 if link is empty/invalid.
//
// To get node index mask must be applied to zero
// highest bit.
//
// Zero value is empty/invalid link.
type slink uint32

// Decode index and valid flag stored in link.
//
// Returns (i, true) if index is valid.
// Returns (0, false) otherwise.
func (l slink) dec() (uint32, bool) {
	const mask uint32 = 1 << 31
	const mask2 = ^mask
	ok := uint32(l)&mask != 0
	i := uint32(l) & (^mask)
	return i, ok
}

// encode index as valid slink
func encx(i uint32) slink {
	const mask uint32 = 1 << 31
	return slink(mask | i)
}

// session node for doubly-linked list
type snode struct {
	sent

	// session token, always not empty for occupied node
	tok string

	// Link to next node.
	//
	// For list head (entry marked newest) this field is 0.
	next slink

	// Link to prev node.
	//
	// For list tail (entry marked oldest) this field is 0.
	prev slink
}

// session entry in cache
type sent struct {
	// equals 0 if entry is empty
	user base.UserID

	// session expire timestamp in microseconds
	expts uint64
}

// Session LRU cache.
//
// Implementation maintains usage order inside doubly-linked list.
// List nodes are stored in continuos array of nodes and linked
// together using their indices.
//
// Algorithm for bumping entries recent usage (and thus rearranging
// linked list) may choose to do nothing if number of occupied nodes
// is low and thus no eviction will be needed for many potential
// future inserts.
type scache struct {
	nodes [scap]snode

	// Contains indices of freed nodes that are ready for reuse.
	//
	// Note that buffer capacity must be the same (or greater)
	// as cache capacity for safe usage.
	ready cubuf

	mu sync.Mutex

	// Maps session token to its index inside slots.
	//
	// Always contains only valid entries. If token is found
	// inside this map that means corresponding node contains
	// valid (before expiration check) session entry.
	m map[ /* token */ string]uint32

	// Number of occupied nodes.
	//
	// Should be always equal to number of map entries.
	num uint32

	// Minimal index of node that was never in use previously.
	//
	// This is used for acquiring empty nodes when ready buffer
	// is empty.
	//
	// TODO:
	// When node is marked as empty (released) we can check
	// if it is just under current top and reduce the top index
	// instead of placing it into reuse buffer.
	top uint32

	// Index that points to list head node.
	//
	// Only valid when number of stored entries is not 0.
	head uint32

	// Index that points to list tail node.
	//
	// Only valid when number of stored entries is not 0.
	tail uint32
}

// session cache entry status code
type ssc uint8

const (
	// invalid/not found
	sinv ssc = 0

	// found and valid
	sval ssc = 1

	// found and expired
	sexp ssc = 2
)

// Init cache internal buffers and store supplied entries.
//
// Slice must be sorted by expiration timestamp in descending
// order. Oldest elements will be discarded if number of supplied
// entries exceeds maximum cache capacity.
//
// Empty or nil slice is a valid argument. In which case
// small initial size will be preallocated for future use.
//
// Must only be used once before all other operations.
// Must only be called outside of concurrent environment.
func (c *scache) init(list []base.SessionEntry) {
	const defsize = min(1<<10, scap)
	initsize := min(len(list), scap)
	if initsize < defsize {
		initsize = defsize
	}

	c.m = make(map[string]uint32, initsize)
	n := min(uint32(len(list)), scap)
	for i := range n {
		e := list[i]
		c.nodes[i] = snode{
			user:  e.UserID,
			expts: e.Expire,
			tok:   e.Token,
			next:  encx(i - 1),
			prev:  encx(i + 1),
		}
		c.m[e.Token] = i
	}
	if n != 0 {
		c.nodes[0].next = 0
		c.nodes[n-1].prev = 0

		c.head = 0
		c.tail = n - 1
		c.num = n
		c.top = n
	}
}

// Find stored session entry in cache by token and bump
// its usage if necessary. Accepts unix microseconds timestamp
// for deciding if session is expired.
//
// If found entry is expired this operation also removes
// the entry from cache.
//
// Returns empty (with user=0) entry if not found.
//
// Safe for concurrent usage.
func (c *scache) get(token string, nowts uint64) (sent, ssc) {
	c.mu.Lock()

	var s sent
	var r ssc

	i, ok := c.m[token]
	if !ok {
		r = sinv
	} else if c.nodes[i].expts <= nowts {
		// entry expired, need to remove it
		s = c.remove(i)
		r = sexp
	} else {
		s = c.bump(i)
		r = sval
	}

	c.mu.Unlock()
	return s, r
}

// Remove entry from cache while maintaining consistency
// of linked list and other internal structures.
//
// Returns previously stored entry.
//
// Can only be used if target node is occupied.
//
// Must only be used with acquired lock.
func (c *scache) remove(i uint32) sent {
	node := c.nodes[i]
	s := node.sent

	c.nodes[i] = snode{}
	delete(c.m, node.tok)
	c.num -= 1
	c.release(i)
	if c.num == 0 {
		// removed last node, linked list is empty now
		return s
	}

	// for code below tail != head is true
	// since it could only happen when there was
	// one occupied node and such case was already
	// handled previously

	ni, ok := node.next.dec()
	if !ok {
		// removed head node, need to move list head to
		// prev link, it is guaranteed to be valid
		// for head node since there was at least
		// 2 nodes in linked list
		pi, _ := node.prev.dec()
		c.nodes[pi].next = 0
		c.head = pi
	} else {
		pi, ok := node.prev.dec()
		if !ok {
			// removed tail node, need to move list tail to
			// next link
			c.nodes[ni].prev = 0
			c.tail = ni
		} else {
			// no need to move head or tail
			c.nodes[ni].prev = node.prev
			c.nodes[pi].next = node.next
		}
	}

	return s
}

// Mark node with specified index as newest and return stored entry.
//
// Bumped node moves to list head, previous head node becomes
// prev node for new head.
//
// Can only be used if target node is occupied.
//
// Must only be used with acquired lock.
func (c *scache) bump(i uint32) sent {
	node := c.nodes[i]
	s := node.sent

	// threshold for high/low number of entries
	// inside cache for switching on/off bumping
	// algorithm
	const threshold = scap>>1 + scap>>2
	if c.num < threshold {
		// number of stored entries is too low
		// to bother with bumping nodes in linked list
		return s
	}

	ni, ok := node.next.dec()
	if !ok {
		// already at the head and thus no actions needed
		return s
	}

	// update links for next and prev nodes
	c.nodes[ni].prev = node.prev
	pi, ok := node.prev.dec()
	if ok {
		c.nodes[pi].next = node.next
	} else {
		// this node was a tail
		// move tail link to next node
		c.tail = ni
	}

	// current head index
	// guaranteed to be valid, since this method can only
	// be used if there is at least one occupied node
	hi := c.head
	c.head = i
	c.nodes[hi].next = encx(i)

	// make previous head as prev node to new head
	// and erase next pointer to mark it as head node
	c.nodes[i].prev = encx(hi)
	c.nodes[i].next = 0

	return s
}

// Place session entry into cache with specified token.
// Token must uniquely identify session.
//
// It is caller responsibility to check that session is
// not expired before placing it into cache.
//
// Does nothing if token already stored in cache.
//
// Safe for concurrent usage.
func (c *scache) put(token string, s sent) {
	c.mu.Lock()

	_, ok := c.m[token]
	if ok {
		// already stored in cache and thus no actions needed
		//
		// we assume here that all tokens are unique and sessions
		// are immutable once created

		c.mu.Unlock()
		return
	}

	i, ok := c.acquire()
	if ok {
		// found empty node where new entry can be stored
		c.m[token] = i

		// rearrange linked list to place node with new entry at the head
		if c.num == 0 {
			// when cache is empty linked list has no head yet
			// and thus such case must be handled separately
			c.nodes[i] = snode{
				tok:  token,
				sent: s,
				next: 0,
				prev: 0,
			}
			c.tail = i
		} else {
			hi := c.head
			c.nodes[i] = snode{
				tok:  token,
				sent: s,
				next: 0,
				prev: encx(hi),
			}
			c.nodes[hi].next = encx(i)
		}
		c.head = i
		c.num += 1

		c.mu.Unlock()
		return
	}

	// Failed to acquire empty node. That means cache is full
	// and thus we need to evict tail entry.
	//
	// To do that we change tail into head and place
	// new entry into it.
	//
	// Since cache is already full tail is guaranteed
	// to exist as well as its next node. Head also
	// guaranteed to exist.
	i = c.tail
	hi := c.head
	ni, _ := c.nodes[i].next.dec()

	// replace token inside index map
	oldtok := c.nodes[i].tok
	delete(c.m, oldtok)
	c.m[token] = i

	// make next (to current tail) node into new tail
	c.nodes[ni].prev = 0
	c.tail = ni

	c.nodes[i] = snode{
		tok:  token,
		sent: s,
		next: 0,
		prev: encx(hi),
	}
	c.nodes[hi].next = encx(i)
	c.head = i

	// Number of stored entries did not change since
	// we evicted old entry and replaced it with new one.

	c.mu.Unlock()
}

// Acquire index of empty node from cache pool for storing
// new entry.
//
// Returns (i, true) if empty node was found.
// Returns (0, false) if there are no empty nodes left.
func (c *scache) acquire() (uint32, bool) {
	i, ok := c.ready.pop()
	if ok {
		return i, true
	}

	if c.top < scap {
		i = c.top
		c.top += 1
		return i, true
	}

	return 0, false
}

// Mark node with specified index as empty and thus ready
// for reuse by new nodes.
func (c *scache) release(i uint32) {
	if c.num == 0 {
		c.ready.reset()
		c.top = 0
		return
	}

	if i+1 >= c.top {
		c.top = i
		if i != 0 {
			c.shrink()
		}
		return
	}

	c.ready.push(i)
}

// Try to shrink top by removing empty nodes from reuse
// buffer if previous (relative to top) nodes are empty.
func (c *scache) shrink() {
	i := c.top - 1
	for {
		if c.nodes[i].user != 0 {
			// end of empty nodes streak

			// number of empty nodes to shrink
			n := c.top - i - 1
			if n != 0 {
				c.top = c.top - n
				c.ready.cull(c.top)
			}
			return
		}
		// +1 empty node in streak

		if i == 0 {
			// TODO: should we even need this case?
			// it is only possible when number of occupied
			// nodes is already 0, and we do not call shrink()
			// in such case
			c.ready.reset()
			c.top = 0
			return
		}
		i -= 1
	}
}

// Circular buffer for unsigned (32-bit) integers.
// Implements FIFO logic for putting and taking elements.
//
// Implementation is deliberately unsafe. It does not check
// for capacity overflow.
//
// Zero value is ready to use empty buffer.
type cubuf struct {
	slots [scap]uint32

	// slot index where next element should be stored
	p uint32

	// number of elements stored
	n uint32
}

// Store element into buffer, placing it at the head.
func (c *cubuf) push(x uint32) {
	i := c.next()
	c.slots[i] = x
}

// Returns slot index for placing new element.
// Handles internal logic for slot index wrapping and
// stored elements number.
func (c *cubuf) next() uint32 {
	// assumes that capacity is a power of 2 for wrapping logic
	const mask = scap - 1

	p := c.p
	c.p += (p + 1) & mask
	c.n += 1
	return p
}

// Take element from buffer tail.
//
// Returns (x, true) if operation was successful.
// Returns (0, false) if there are no elements left in buffer.
func (c *cubuf) pop() (uint32, bool) {
	if c.n == 0 {
		return 0, false
	}

	// assumes that capacity is a power of 2 for wrapping logic
	const mask = scap - 1

	i := (c.p - c.n) & mask
	c.n -= 1
	return c.slots[i], true
}

// remove all stored integers which are greater
// or equal to the given integer
func (c *cubuf) cull(x uint32) {
	if x == 0 {
		c.reset()
		return
	}

	const mask = scap - 1

	s := (c.p - c.n) & mask // starting index
	i := s                  // insert index
	j := i                  // scan index
	var n uint32            // counter of inspected elements
	var k uint32            // counter of accepted elements
	for n < c.n {
		if c.slots[j] < x {
			c.slots[i] = c.slots[j]
			i = (i + 1) & mask
			k += 1
		}

		j = (j + 1) & mask
		n += 1
	}

	c.n = k
	c.p = (s + k) & mask
}

func (c *cubuf) reset() {
	c.p = 0
	c.n = 0
}
