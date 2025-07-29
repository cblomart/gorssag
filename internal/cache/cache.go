package cache

import (
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
)

type Manager struct {
	cache *cache.Cache
	mu    sync.RWMutex
}

func NewManager(defaultTTL time.Duration) *Manager {
	return &Manager{
		cache: cache.New(defaultTTL, 10*time.Minute),
	}
}

func (m *Manager) Get(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cache.Get(key)
}

func (m *Manager) Set(key string, value interface{}, ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache.Set(key, value, ttl)
}

func (m *Manager) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache.Delete(key)
}

func (m *Manager) Flush() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache.Flush()
}
