package config

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type CacheStore interface {
	Set(key string, value interface{}, ttl time.Duration) error
	Get(key string) (interface{}, bool)
	Delete(key string) error
}

type memoryItem struct {
	value      interface{}
	expiration time.Time
}

type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]memoryItem
}

var Cache CacheStore
var SessionCache CacheStore

func NewMemoryCache() *MemoryCache {
	mc := &MemoryCache{
		items: make(map[string]memoryItem),
	}
	// Jalankan cleanup goroutine tiap 1 menit
	go mc.cleanup()
	return mc
}

func (m *MemoryCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for key, item := range m.items {
			if now.After(item.expiration) {
				delete(m.items, key)
			}
		}
		m.mu.Unlock()
	}
}

func (m *MemoryCache) Set(key string, value interface{}, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[key] = memoryItem{
		value:      value,
		expiration: time.Now().Add(ttl),
	}
	return nil
}

func (m *MemoryCache) Get(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	item, ok := m.items[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(item.expiration) {
		return nil, false
	}
	return item.value, true
}

func (m *MemoryCache) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, key)
	return nil
}

type RedisCache struct {
	client *redis.Client
}

func (r *RedisCache) Set(key string, value interface{}, ttl time.Duration) error {
	if RedisClient == nil {
		return nil
	}
	ctx := context.Background()
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisCache) Get(key string) (interface{}, bool) {
	if RedisClient == nil {
		return nil, false
	}
	ctx := context.Background()
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return nil, false
	}
	return val, true
}

func (r *RedisCache) Delete(key string) error {
	if RedisClient == nil {
		return nil
	}
	ctx := context.Background()
	return r.client.Del(ctx, key).Err()
}

func InitCache() {
	if RedisClient != nil {
		Cache = &RedisCache{client: RedisClient}
	} else {
		Cache = NewMemoryCache()
	}

	// Session cache terpisah
	if RedisSessionClient != nil {
		SessionCache = &RedisCache{client: RedisSessionClient}
	} else if Cache != nil {
		SessionCache = Cache // fallback ke cache utama
	} else {
		SessionCache = NewMemoryCache()
	}
}
