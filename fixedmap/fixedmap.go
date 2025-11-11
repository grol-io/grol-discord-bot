package fixedmap

import (
	"sync"
	"time"
)

// Node is a double linked list node.
// Note: wrote this before I saw container/list but then that one isn't using generics so this is better anyway.
type Node[K, V any] struct {
	Key      K
	Value    V
	LastUsed time.Time
	Next     *Node[K, V]
	Prev     *Node[K, V]
}

type FixedMap[K comparable, V any] struct {
	Max  int
	Map  map[K]*Node[K, V]
	Head *Node[K, V]
	Tail *Node[K, V]
	lock sync.Mutex
}

// NewFixedMap initializes a new FixedMap with a given maximum size.
func NewFixedMap[K comparable, V any](maxV int) *FixedMap[K, V] {
	if maxV < 2 {
		panic("max must be at least 2")
	}
	return &FixedMap[K, V]{
		Max: maxV,
		Map: make(map[K]*Node[K, V]),
	}
}

// Add adds a new key to the FixedMap, evicting the least recently used if necessary
// Returns the evicted node and a boolean indicating if the key was new.
func (fs *FixedMap[K, V]) Add(key K, value V) (*Node[K, V], bool) {
	fs.lock.Lock()
	defer fs.lock.Unlock()

	if node, exists := fs.Map[key]; exists {
		node.Value = value
		fs.moveToHead(node)
		return nil, false
	}
	// Create a new node
	node := &Node[K, V]{
		Key:   key,
		Value: value,
	}
	// Add to map and linked list
	fs.Map[key] = node
	fs.addToFront(node)
	// Check if we need to evict
	if len(fs.Map) > fs.Max {
		return fs.evict(), true
	}
	return nil, true
}

// Get retrieves a key from the FixedMap and updates its position.
func (fs *FixedMap[K, V]) Get(key K) (v V, found bool) {
	fs.lock.Lock()
	defer fs.lock.Unlock()
	if node, exists := fs.Map[key]; exists {
		fs.moveToHead(node)
		return node.Value, true
	}
	return v, found // zero values, ie not found
}

func (fs *FixedMap[K, V]) addToFront(node *Node[K, V]) {
	node.LastUsed = time.Now()
	if fs.Head == nil { // first add ever
		fs.Head = node
		fs.Tail = node
		return
	}
	fs.setNodeToHead(node)
}

func (fs *FixedMap[K, V]) setNodeToHead(node *Node[K, V]) {
	node.Next = fs.Head
	fs.Head.Prev = node
	fs.Head = node
}

// moveToHead moves a given node to the head of the list
// and updates the LastUsed time.
func (fs *FixedMap[K, V]) moveToHead(node *Node[K, V]) {
	node.LastUsed = time.Now()
	if fs.Head == node {
		return // already at head, most recently used
	}
	// Check if it's the last (note max is >= 2)
	if fs.Tail == node {
		fs.Tail = node.Prev
	} else {
		// not last, next non nil, unlink/link Next to prev.
		node.Next.Prev = node.Prev
	}
	// (finish) unlink (not first so safe to deref Prev)
	node.Prev.Next = node.Next
	node.Prev = nil
	// put to head
	fs.setNodeToHead(node)
}

// evict removes the least recently used (tail) node from the list and map.
func (fs *FixedMap[K, V]) evict() *Node[K, V] {
	if fs.Tail == nil {
		panic("evict called on empty list")
	}
	evictedNode := fs.Tail
	fs.Tail = evictedNode.Prev
	if fs.Tail != nil {
		fs.Tail.Next = nil
	}
	delete(fs.Map, evictedNode.Key)
	evictedNode.Next = nil // don't leak
	evictedNode.Prev = nil
	return evictedNode
}
