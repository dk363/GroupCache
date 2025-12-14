package consitenthash

import (
	"fmt"
	"strconv"
	"testing"
)

func TestHashing(t *testing.T) {

	hash := New(3, func(key []byte) uint32 {
		i, err := strconv.Atoi(string(key))
		if err != nil {
			panic(err)
		}

		return uint32(i)
	})

	// Given the above hash function, this will give relicas with "hashes"
	// 2, 12, 22, 4, 14, 24, 6, 16, 26
	// And sort for the "hashes"
	hash.Add("6", "4", "2")

	testCases := map[string]string {
		"2": "2",
		"11": "2",
		"23": "4",
		"27": "2",
	}
	
	for k, v := range testCases {
		if hash.Get(k) != v {
			t.Errorf("Asking for %s, shoule have yielded %s", k, v)
		}
	}

	hash.Add("8")

	// Adds 8, 18, 28
	testCases["27"] = "8"

	for k, v := range testCases {
		if hash.Get(k) != v {
			t.Errorf("Asking for %s, should have yielded %s", k, v)
		}
	}

}


func TestConsistency(t *testing.T) {
    hash1 := New(1, nil)
    hash1.Add("NodeA", "NodeB", "NodeC")  
    

	keys := []string{"Ben", "Alice", "Tom", "Data123"}
    
    for _, key := range keys {
        node := hash1.Get(key)
        fmt.Printf("Key '%s' → Node '%s'\n", key, node)
    }

	hash2 := New(1, nil)
	hash2.Add("NodeA", "NodeB", "NodeC")

	if hash1.Get("Ben") != hash2.Get("Ben") ||
		hash1.Get("Bob") != hash2.Get("Bob") ||
		hash1.Get("Bonny") != hash2.Get("Bonny") {
		t.Errorf("Direct matches should always return the same entry")
	}
}

func BenchmarkGet8(b *testing.B)   { benchmarkGet(b, 8) }
func BenchmarkGet32(b *testing.B)  { benchmarkGet(b, 32) }
func BenchmarkGet128(b *testing.B) { benchmarkGet(b, 128) }
func BenchmarkGet512(b *testing.B) { benchmarkGet(b, 512) }

// Get 的性能测试
func benchmarkGet(b *testing.B, shards int) {
    hash := New(50, nil)
    
    var buckets []string
    for i := 0; i < shards; i++ {
        buckets = append(buckets, fmt.Sprintf("shard-%d", i))
    }
    
    hash.Add(buckets...)
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        hash.Get(buckets[i&(shards-1)])
    }
}