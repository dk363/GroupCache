package cachepolicy

import "container/list"

// Cache is an LRU cache. It is not safe for concurrent access.

type LRUCache struct {
	// zero means no limit
	capacity int

	// TODO
	OnEvcted func(key Key, value interface{})

	ll    *list.List
	cache map[interface{}]*list.Element
}

type LRUEntry struct {
	key   Key
	value interface{}
}

func LRUNew(max_entries int) *LRUCache {
	if max_entries < 0 {
		panic("capacity must > 0")
	}

	return &LRUCache{
		capacity: max_entries,
		ll:       list.New(),
		cache:    make(map[interface{}]*list.Element),
	}
}

// Add adds a value to the cache.
// If the key exists, update the value
func (c *LRUCache) Add(key Key, value interface{}) {
	// Go 的结构体可以创建为 零值
	// var c Cache
	// 这个时候 c.ll c.cache == nil
	if c.cache == nil {
		c.cache = make(map[interface{}]*list.Element)
		c.ll = list.New()
	}

	// cache contains the key
	// update the value
	if ee, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ee)
		ee.Value.(*LRUEntry).value = value
		return
	}
	// cache not contains the key
	// add the new node
	ele := c.ll.PushFront(&LRUEntry{key, value})
	c.cache[key] = ele
	if c.capacity != 0 && c.ll.Len() > c.capacity {
		c.RemoveOldest()
	}
}

func (c *LRUCache) Get(key Key) (value interface{}, ok bool) {
	if c.cache == nil {
		return
	}

	if ele, hit := c.cache[key]; hit {
		c.ll.MoveToFront(ele)
		return ele.Value.(*LRUEntry).value, true
	}
	return
}

func (c *LRUCache) Remove(key Key) {
	if c.cache == nil {
		return
	}

	if ele, hit := c.cache[key]; hit {
		c.removeElement(ele)
	}
}

func (c *LRUCache) RemoveOldest() {
	if c.cache == nil {
		return
	}

	ele := c.ll.Back()
	if ele != nil {
		c.removeElement(ele)
	}
}

func (c *LRUCache) removeElement(ele *list.Element) {
	c.ll.Remove(ele)
	kv := ele.Value.(*LRUEntry)
	delete(c.cache, kv.key)
	if c.OnEvcted != nil {
		c.OnEvcted(kv.key, kv.value)
	}
}

func (c *LRUCache) Len() int {
	if c.cache == nil {
		return 0
	}
	return c.ll.Len()
}

func (c *LRUCache) Clear() {
	if c.OnEvcted != nil {
		for _, e := range c.cache {
			kv := e.Value.(*LRUEntry)
			c.OnEvcted(kv.key, kv.value)
		}
	}
	c.ll = nil
	c.cache = nil
}
