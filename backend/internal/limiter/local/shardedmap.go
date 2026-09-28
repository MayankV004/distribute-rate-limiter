package local

import (
	"hash/fnv"
	"sync"
	"time"
)

// shardCount balances memory overhead against lock contention.
// A power of two is preferred for potential modulo optimization,
// though FNV distribution over 256 shards is sufficient for gateway workloads.
const shardCount = 256

type ShardedMap struct {
	shards [shardCount]*shard
}

type shard struct {
	sync.Mutex
	items map[string]item
}

type item struct {
	value       interface{}
	lastTouched time.Time
}

func NewShardedMap(cleanupInterval time.Duration, ttl time.Duration) *ShardedMap {
	sm := &ShardedMap{}
	for i := 0; i < shardCount; i++ {
		sm.shards[i] = &shard{
			items: make(map[string]item),
		}
	}

	go sm.janitor(cleanupInterval, ttl)

	return sm
}

func (sm *ShardedMap) getShard(key string) *shard {
	hasher := fnv.New32a()
	hasher.Write([]byte(key))
	return sm.shards[hasher.Sum32()%shardCount]
}

func (sm *ShardedMap) GetOrInsert(key string, factory func() interface{}) interface{} {
	shard := sm.getShard(key)

	shard.Lock()
	defer shard.Unlock()

	i, exists := shard.items[key]
	if !exists {
		i = item{
			value:       factory(),
			lastTouched: time.Now(),
		}
	} else {
		i.lastTouched = time.Now()
	}

	shard.items[key] = i
	return i.value
}

func (sm *ShardedMap) janitor(interval time.Duration, ttl time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		for i := 0; i < shardCount; i++ {
			shard := sm.shards[i]

			shard.Lock()
			for key, item := range shard.items {
				if now.Sub(item.lastTouched) > ttl {
					delete(shard.items, key)
				}
			}
			shard.Unlock()
		}
	}
}
