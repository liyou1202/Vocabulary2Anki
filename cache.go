package main

type Cache struct {
	data       map[string]string
	order      []string
	maxEntries int
}

func NewCache(maxEntries int) *Cache {
	return &Cache{
		data:       make(map[string]string),
		order:      make([]string, 0),
		maxEntries: maxEntries,
	}
}

func (c *Cache) Set(key, value string) {
	if _, exists := c.data[key]; exists {
		c.data[key] = value
		c.removeKeyFromOrder(key)
		c.order = append(c.order, key)
	} else {
		if len(c.data) >= c.maxEntries {
			oldestKey := c.order[0]
			delete(c.data, oldestKey)
			c.order = c.order[1:]
		}
		c.data[key] = value
		c.order = append(c.order, key)
	}
}

func (c *Cache) Get(key string) (string, bool) {
	value, exists := c.data[key]
	return value, exists
}

func (c *Cache) removeKeyFromOrder(key string) {
	for i, k := range c.order {
		if k == key {
			c.order = append(c.order[:i], c.order[i+1:]...)
			break
		}
	}
}
