// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package keepclient

import (
	"io"
	"sort"
	"sync"
	"time"
)

var DefaultBlockCache = &BlockCache{}

type BlockCache struct {
	// Maximum number of blocks to keep in the cache. If 0, a
	// default size (currently 4) is used instead.
	MaxBlocks int

	cache map[string]*cacheBlock
	mtx   sync.Mutex
}

const defaultMaxBlocks = 4

// Sweep deletes the least recently used blocks from the cache until
// there are no more than MaxBlocks left.
func (c *BlockCache) Sweep() {
	max := c.MaxBlocks
	if max == 0 {
		max = defaultMaxBlocks
	}
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if len(c.cache) <= max {
		return
	}
	lru := make([]time.Time, 0, len(c.cache))
	for _, b := range c.cache {
		lru = append(lru, b.lastUse)
	}
	sort.Sort(sort.Reverse(timeSlice(lru)))
	threshold := lru[max]
	for loc, b := range c.cache {
		if !b.lastUse.After(threshold) {
			delete(c.cache, loc)
		}
	}
}

// Get returns data from the cache, first retrieving it from Keep if
// necessary.
func (c *BlockCache) Get(kc *KeepClient, locator string) ([]byte, error) {
	cacheKey := locator[:32]
	c.mtx.Lock()
	if c.cache == nil {
		c.cache = make(map[string]*cacheBlock)
	}
	b, ok := c.cache[cacheKey]
	if !ok || b.err != nil {
		b = &cacheBlock{
			fetched: make(chan struct{}),
			lastUse: time.Now(),
		}
		c.cache[cacheKey] = b
		go func() {
			rdr, size, _, err := kc.Get(locator)
			var data []byte
			if err == nil {
				data = make([]byte, size, BLOCKSIZE)
				_, err = io.ReadFull(rdr, data)
				err2 := rdr.Close()
				if err == nil {
					err = err2
				}
			}
			c.mtx.Lock()
			b.data, b.err = data, err
			c.mtx.Unlock()
			close(b.fetched)
			go c.Sweep()
		}()
	}
	c.mtx.Unlock()

	// Wait (with mtx unlocked) for the fetch goroutine to finish,
	// in case it hasn't already.
	<-b.fetched

	c.mtx.Lock()
	b.lastUse = time.Now()
	c.mtx.Unlock()
	return b.data, b.err
}

func (c *BlockCache) Clear() {
	c.mtx.Lock()
	c.cache = nil
	c.mtx.Unlock()
}

type timeSlice []time.Time

func (ts timeSlice) Len() int { return len(ts) }

func (ts timeSlice) Less(i, j int) bool { return ts[i].Before(ts[j]) }

func (ts timeSlice) Swap(i, j int) { ts[i], ts[j] = ts[j], ts[i] }

type cacheBlock struct {
	data    []byte
	err     error
	fetched chan struct{}
	lastUse time.Time
}
