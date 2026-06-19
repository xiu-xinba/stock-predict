package cache

import (
	"testing"
	"time"
)

func TestDetailCacheGetWithMaxAgeLeavesExpiredEntryAvailableAsStale(t *testing.T) {
	cache := NewDetailCache(10, time.Minute)
	cache.Set("quote", "cached")

	cache.mu.Lock()
	cache.items["quote"].cachedAt = time.Now().Add(-5 * time.Second)
	cache.mu.Unlock()

	if value, ok := cache.GetWithMaxAge("quote", time.Second); ok {
		t.Fatalf("expected max-age lookup to miss, got %v", value)
	}
	value, ok := cache.GetStale("quote")
	if !ok || value != "cached" {
		t.Fatalf("expected stale value to remain available, got value=%v ok=%v", value, ok)
	}
}
