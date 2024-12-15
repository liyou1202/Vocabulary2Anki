package cache

import (
	"anki-tool/model"
	"sync"
)

type Cache struct {
	data map[string][]model.VocabularyInfo
	mu   sync.RWMutex
}

func NewCache() *Cache {
	return &Cache{
		data: make(map[string][]model.VocabularyInfo),
	}
}

func (c *Cache) Get(key string) ([]model.VocabularyInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result, exists := c.data[key]

	return result, exists
}

func (c *Cache) Set(key string, val []model.VocabularyInfo) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = val
	return nil
}
