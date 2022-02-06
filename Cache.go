package cache

import (
	"context"
	"sync"
	"time"
)

const (
	defaultCleanUpInterval = time.Second * 30
)

type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, expiredInterval time.Duration)
	Delete(key string)
}

type inMemoryCache struct {
	storage       sync.Map
	cleanUpTicker *time.Ticker
}

type cacheItem struct {
	validThrough time.Time
	value        interface{}
}

func NewInMemoryCache(ctx context.Context, options ...func(cache *inMemoryCache)) Cache {
	cleanUpTicker := time.NewTicker(defaultCleanUpInterval)
	cache := &inMemoryCache{cleanUpTicker: cleanUpTicker}

	for _, optionFn := range options {
		optionFn(cache)
	}

	go cache.cleanUpCache(ctx)

	return cache
}

func WithCleanUpInterval(duration time.Duration) func(*inMemoryCache) {
	return func(cache *inMemoryCache) {
		cache.cleanUpTicker.Reset(duration)
	}
}

func (c *inMemoryCache) Get(key string) (interface{}, bool) {
	storageValue, found := c.storage.Load(key)
	if !found {
		return nil, false
	}

	item := storageValue.(cacheItem)
	if time.Now().UnixNano() > item.validThrough.UnixNano() {
		return nil, false
	}

	return item.value, true
}

func (c *inMemoryCache) Set(key string, value interface{}, expiredInterval time.Duration) {
	item := cacheItem{value: value, validThrough: time.Now().Add(expiredInterval)}
	c.storage.Store(key, item)
}

func (c *inMemoryCache) Delete(key string) {
	c.storage.Delete(key)
}

func (c *inMemoryCache) cleanUpCache(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.cleanUpTicker.C:
			itemsToDelete := c.getCacheItemsToDelete()
			for _, itemKey := range itemsToDelete {
				c.storage.Delete(itemKey)
			}
		}
	}
}

func (c *inMemoryCache) getCacheItemsToDelete() []interface{} {
	var itemsToDelete []interface{}
	c.storage.Range(func(key, value interface{}) bool {
		item := value.(cacheItem)
		if time.Now().UnixNano() > item.validThrough.UnixNano() {
			itemsToDelete = append(itemsToDelete, key)
		}

		return true
	})

	return itemsToDelete
}
