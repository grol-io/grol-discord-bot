package fixedmap

import (
	"testing"
)

func TestMap(t *testing.T) {
	cache := NewFixedMap[string, string](3)

	cache.Add("a", "1")
	cache.Add("b", "2")
	cache.Add("c", "3")
	cache.Add("d", "4") // This should evict "a"

	if v, found := cache.Get("a"); found {
		t.Errorf("Key a should have been evicted, but it was found, value: %s", v)
	} else {
		t.Log("Key a was correctly evicted")
	}

	if v, found := cache.Get("d"); found {
		t.Logf("Key d found in cache - value %s", v)
	} else {
		t.Error("Key d should be in the cache, but it was not found")
	}

	cache.Add("e", "5") // This should evict "b", not "d"

	if _, found := cache.Get("b"); found {
		t.Error("Key b should have been evicted, but it was found")
	} else {
		t.Log("Key b was correctly evicted")
	}
	if v, found := cache.Get("d"); found {
		t.Logf("Key d found in cache %s", v)
	} else {
		t.Error("Key d should be in the cache, but it was not found")
	}
}
