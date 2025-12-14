// Package consistenthash provides an implementation of a ring hash
package consitenthash

import (
	"hash/crc32"
	"strconv"
	"sort"
)

// You can specify your own hash function
type Hash func([]byte) uint32

type Map struct {
	// 存放哈希值
	keys 		[]int
	// 建立哈希值和value 之间的映射
	hashMap 	map[int]string
	hash 		Hash
	replicas	int
}

func New(replicas int, fn Hash) *Map {
	m := &Map{
		hashMap: 	make(map[int]string),
		replicas: 	replicas,
		hash:		fn,
	}

	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// IsEmpty return true if there are no item available
func (m *Map) IsEmpty() bool {
	return len(m.keys) == 0
}

// Add some keys to the hash
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			hashInput := strconv.Itoa(i) + key
			hash := int(m.hash([]byte(hashInput)))
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}
	// sort the keys 
	// In the Get function we can use binary search
	sort.Ints(m.keys)
}

// Get gets the closest item in the hash to the provided key
// Check if a key exists in the buffer
func (m *Map) Get(key string) string {
	if m.IsEmpty() {
		return ""
	}

	hash := int(m.hash([]byte(key)))

	// Binary search for appropriate replica
	// 找到最小的大于等于 hash 的节点
	idx := sort.Search(len(m.keys), func(i int) bool {return m.keys[i] >= hash})

	if idx == len(m.keys) {
		idx = 0
	}

	return m.hashMap[m.keys[idx]]
}
