package cachepolicy

import (
	"container/list"
	"sync"
)

type ListType int

const (
	LRU ListType = iota
	RG
	LFU
	FG
)

type ARCEntry struct {
	key       Key
	value     interface{}
	// O(1) 快速定位
	ListType ListType
}

type ARCCache struct {
	mu sync.Mutex

	// capacity is the max entries 
	// lru + lfu == capacity
	// zero mean no limit. But I'm not sure no limit will be a good approach
	// So we will panic when you input 0
	capacity int
	// Adaptive parameters
	// 也就是lru期望的大小
	p        int

	// OnEvicted optionally specifies a callback function to be
	// executed when an entry is purged from the cache.
	OnEvcted func(key Key, value interface{})

	lru       *list.List
	lru_ghost *list.List
	lfu       *list.List
	lfu_ghost *list.List

	cache map[interface{}]*list.Element
}

func ARCNew(capacity int) *ARCCache {
	if capacity <= 0 {
		panic("capacity must be > 0")
	}

	return &ARCCache{
		capacity:  capacity,
		p:         0,
		lru:       list.New(),
		lru_ghost: list.New(),
		lfu:       list.New(),
		lfu_ghost: list.New(),
		cache:     make(map[interface{}]*list.Element),
	}
}

// 将key value 放入ARC 中
// 如果缓存中已经有了 那么就 update
func (c *ARCCache) Add(key Key, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cache == nil {
		c.cache = make(map[interface{}]*list.Element)
		c.lru = list.New()
		c.lru_ghost = list.New()
		c.lfu = list.New()
		c.lfu_ghost = list.New()
	}

	if elem, ok := c.cache[key]; ok {
		entry := elem.Value.(*ARCEntry)

		switch entry.ListType {
		case LRU:
			entry.value = value
			c.lru.Remove(elem)
			entry.ListType = LFU
			c.lfu.PushFront(entry) // 因为 PushFront 是接受一个新元素 所以这里要用 entry
			c.cache[key] = c.lfu.Front()
		
		case LFU:
			c.lfu.MoveToFront(elem)

		case RG:
			delta := 1
			if c.lru_ghost.Len() < c.lfu_ghost.Len() {
				delta = c.lfu_ghost.Len() / c.lru_ghost.Len()
			}

			c.p = c.min(c.p+delta, c.capacity)
			c.replace(key)

			entry := elem.Value.(*ARCEntry)
			entry.value = value
			entry.ListType = LRU
			c.lru_ghost.Remove(elem)
			c.lru.PushFront(entry)
			c.cache[key] = c.lru.Front()
		
		case FG:
			delta := 1
			if c.lru_ghost.Len() > c.lfu_ghost.Len() {
				delta = c.lru_ghost.Len() / c.lfu_ghost.Len()
			}

			c.p = c.max(c.p-delta, 0)

			c.replace(key)

			entry := elem.Value.(*ARCEntry)
			entry.value = value
			entry.ListType = LFU
			c.lfu_ghost.Remove(elem)
			c.lru.PushFront(entry)
			c.cache[key] = c.lru.Front()
		}
	}

	// 未命中 这是一个新元素

	// 驱逐 lru 中的元素
	if c.lru.Len()+c.lru_ghost.Len() == c.capacity {
		if c.lru.Len() < c.capacity {
			// 从 ghost 中淘汰
			c.removeLRU(c.lru_ghost)
			c.replace(key)
		} else {
			// c.lru == c.capacity
			// 从 lru 中淘汰
			c.removeLRU(c.lru)
		}
	} else {
		// lru + lru_ghost < capacity
		// 也就是说这个时候不需要驱逐LRU中的元素
		// 但是需要维护链表
		// 也就是如果总长度是两倍的capacity
		// 那么就需要驱逐lfu中的元素
		totalLen := c.lru.Len() + c.lru_ghost.Len() + c.lfu.Len() + c.lfu_ghost.Len()
		if totalLen >= c.capacity {
			if totalLen == 2*c.capacity {
				c.removeLRU(c.lfu_ghost)
			}
			c.replace(key)
		}
	}

	newEntry := &ARCEntry{
		key: key,
		value: value,
		ListType: LRU,
	}

	c.lru.PushFront(newEntry)
	c.cache[key] = c.lru.Front()
}

// get 函数是由副作用的 
// 如果命中在lru 中 那么将其放到lfu中
// 如果已经在lfu中 那么将其提前
// 这里不考虑按照命中次数排序
// 有点太复杂了
// 而且我不确定如果按照命中次数排序性能表现会不会更好
func (c *ARCCache) Get(key Key) (value interface{}, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.cache[key]; ok {
		entry := elem.Value.(*ARCEntry)

		if entry.ListType == LRU {
			c.lru.Remove(elem)
			c.lfu.PushFront(entry)
			c.cache[key] = c.lfu.Front()
		} else {
			c.lfu.MoveToFront(elem)
		}
		
		return entry.value, true
	}

	// 不在缓存之内 
	return
}

// 驱逐末尾的元素
func (c *ARCCache) replace(key Key) {
	lruLen := c.lru.Len()

	inLFU := false
	if elem, ok := c.cache[key]; ok && elem.Value.(*ARCEntry).ListType == LFU {
		inLFU = true
	}

	if lruLen > 0 && (lruLen > c.p || (inLFU && lruLen == c.p)) {
		lru := c.lru.Back()
		entry := lru.Value.(*ARCEntry)
		entry.value = nil
		entry.ListType = RG

		c.lru.Remove(lru)
		c.lru_ghost.PushFront(entry)
		c.cache[entry.key] = c.lru_ghost.Front()
	} else {
		lru := c.lfu.Back()
		entry := lru.Value.(*ARCEntry)
		entry.value = nil
		entry.ListType = FG

		c.lfu.Remove(lru)
		c.lfu_ghost.PushFront(entry)
		c.cache[entry.key] = c.lfu_ghost.Front()
	}
}

// 淘汰末尾的元素
func (c *ARCCache) removeLRU(l *list.List) {
	if l.Len() == 0 {
		return
	}

	lru := l.Back()
	entry := lru.Value.(*ARCEntry)
	delete(c.cache, entry.key)
	l.Remove(lru)
}

func (c *ARCCache) min(i int, j int) int {
	if i < j {
		return i
	} else {
		return j
	}
}

func (c *ARCCache) max(i, j int) int {
	if i > j {
		return i
	} else {
		return j
	}
}
