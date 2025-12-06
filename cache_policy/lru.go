package cachepolicy

import "container/list"

type LRUCache struct {
	// zero means no limit
	MaxEntries int

	// TODO
	OnEvcted func (key Key, value interface{})

	ll *list.List
	cache map[interface{}]*list.Element
}

type Key interface{}

type entry struct {
	key Key
	value interface{}
}

func LRUNew(max_entries int) *LRUCache {
	return &LRUCache {
		MaxEntries: 	max_entries,
		ll:				list.New(),
		cache:			make(map[interface{}]*list.Element),
	}
}

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
		ee.Value.(*entry).value = value
		return 
	}
	// cache not contains the key
	// add the new node
	ele := c.ll.PushFront(&entry{key, value})
	c.cache[key] = ele
	if c.MaxEntries != 0 && c.ll.Len() > c.MaxEntries {
		c.RemoveOldest()
	}
}

func (c *LRUCache) Get(key Key) (value interface{}, ok bool) {
	if c.cache == nil {
		return
	}

	if ele, hit := c.cache[key]; hit {
		c.ll.MoveToFront(ele)
		return ele.Value.(*entry).value, true
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
	kv := ele.Value.(*entry)
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
			kv := e.Value.(*entry)
			c.OnEvcted(kv.key, kv.value)
		}
	}
	c.ll = nil
	c.cache = nil
}