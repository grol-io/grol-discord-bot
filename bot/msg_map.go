package bot

import (
	"sync"
	"time"

	"fortio.org/log"
)

// Double Linked list node
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

// NewFixedMap initializes a new FixedMap with a given maximum size
func NewFixedMap[K comparable, V any](max int) *FixedMap[K, V] {
	if max < 2 {
		panic("max must be at least 2")
	}
	return &FixedMap[K, V]{
		Max: max,
		Map: make(map[K]*Node[K, V]),
	}
}

// Add adds a new key to the FixedMap, evicting the least recently used if necessary
func (fs *FixedMap[K, V]) Add(key K, value V) {
	fs.lock.Lock()
	defer fs.lock.Unlock()

	if node, exists := fs.Map[key]; exists {
		fs.moveToHead(node)
		return
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
		fs.evict()
	}
}

// Get retrieves a key from the FixedMap and updates its position
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
	log.Infof("Adding key %v val %v", node.Key, node.Value)
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
// and updates the LastUsed time
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

// evict removes the least recently used (tail) node from the list and map
func (fs *FixedMap[K, V]) evict() {
	if fs.Tail == nil {
		panic("evict called on empty list")
	}
	evictedNode := fs.Tail
	log.Infof("Evicting key %v val %v", evictedNode.Key, evictedNode.Value)
	fs.Tail = evictedNode.Prev
	if fs.Tail != nil {
		fs.Tail.Next = nil
	}
	delete(fs.Map, evictedNode.Key)
}
