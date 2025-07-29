package cache

import (
	"testing"
	"time"
)

func TestCacheManager_GetSet(t *testing.T) {
	cacheManager := NewManager(15 * time.Minute)

	// Test setting and getting a value
	key := "test-key"
	value := "test-value"

	cacheManager.Set(key, value, 15*time.Minute)

	// Get the value
	if cached, found := cacheManager.Get(key); found {
		if cachedValue, ok := cached.(string); ok {
			if cachedValue != value {
				t.Errorf("Expected value %s, got %s", value, cachedValue)
			}
		} else {
			t.Error("Failed to type assert cached value")
		}
	} else {
		t.Error("Expected to find cached value")
	}
}

func TestCacheManager_Delete(t *testing.T) {
	cacheManager := NewManager(15 * time.Minute)

	key := "test-key"
	value := "test-value"

	cacheManager.Set(key, value, 15*time.Minute)

	// Verify value exists
	if _, found := cacheManager.Get(key); !found {
		t.Error("Expected to find cached value before deletion")
	}

	// Delete the value
	cacheManager.Delete(key)

	// Verify value is gone
	if _, found := cacheManager.Get(key); found {
		t.Error("Expected cached value to be deleted")
	}
}

func TestCacheManager_Flush(t *testing.T) {
	cacheManager := NewManager(15 * time.Minute)

	// Add multiple values
	cacheManager.Set("key1", "value1", 15*time.Minute)
	cacheManager.Set("key2", "value2", 15*time.Minute)

	// Verify values exist
	if _, found := cacheManager.Get("key1"); !found {
		t.Error("Expected to find key1 before flush")
	}
	if _, found := cacheManager.Get("key2"); !found {
		t.Error("Expected to find key2 before flush")
	}

	// Flush cache
	cacheManager.Flush()

	// Verify all values are gone
	if _, found := cacheManager.Get("key1"); found {
		t.Error("Expected key1 to be flushed")
	}
	if _, found := cacheManager.Get("key2"); found {
		t.Error("Expected key2 to be flushed")
	}
}
