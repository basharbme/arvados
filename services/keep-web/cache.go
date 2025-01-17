// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"sync"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"github.com/hashicorp/golang-lru"
	"github.com/prometheus/client_golang/prometheus"
)

const metricsUpdateInterval = time.Second / 10

type cache struct {
	config      *arvados.WebDAVCacheConfig
	registry    *prometheus.Registry
	metrics     cacheMetrics
	pdhs        *lru.TwoQueueCache
	collections *lru.TwoQueueCache
	permissions *lru.TwoQueueCache
	setupOnce   sync.Once
}

type cacheMetrics struct {
	requests          prometheus.Counter
	collectionBytes   prometheus.Gauge
	collectionEntries prometheus.Gauge
	collectionHits    prometheus.Counter
	pdhHits           prometheus.Counter
	permissionHits    prometheus.Counter
	apiCalls          prometheus.Counter
}

func (m *cacheMetrics) setup(reg *prometheus.Registry) {
	m.requests = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "arvados",
		Subsystem: "keepweb_collectioncache",
		Name:      "requests",
		Help:      "Number of targetID-to-manifest lookups handled.",
	})
	reg.MustRegister(m.requests)
	m.collectionHits = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "arvados",
		Subsystem: "keepweb_collectioncache",
		Name:      "hits",
		Help:      "Number of pdh-to-manifest cache hits.",
	})
	reg.MustRegister(m.collectionHits)
	m.pdhHits = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "arvados",
		Subsystem: "keepweb_collectioncache",
		Name:      "pdh_hits",
		Help:      "Number of uuid-to-pdh cache hits.",
	})
	reg.MustRegister(m.pdhHits)
	m.permissionHits = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "arvados",
		Subsystem: "keepweb_collectioncache",
		Name:      "permission_hits",
		Help:      "Number of targetID-to-permission cache hits.",
	})
	reg.MustRegister(m.permissionHits)
	m.apiCalls = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "arvados",
		Subsystem: "keepweb_collectioncache",
		Name:      "api_calls",
		Help:      "Number of outgoing API calls made by cache.",
	})
	reg.MustRegister(m.apiCalls)
	m.collectionBytes = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "arvados",
		Subsystem: "keepweb_collectioncache",
		Name:      "cached_manifest_bytes",
		Help:      "Total size of all manifests in cache.",
	})
	reg.MustRegister(m.collectionBytes)
	m.collectionEntries = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "arvados",
		Subsystem: "keepweb_collectioncache",
		Name:      "cached_manifests",
		Help:      "Number of manifests in cache.",
	})
	reg.MustRegister(m.collectionEntries)
}

type cachedPDH struct {
	expire time.Time
	pdh    string
}

type cachedCollection struct {
	expire     time.Time
	collection *arvados.Collection
}

type cachedPermission struct {
	expire time.Time
}

func (c *cache) setup() {
	var err error
	c.pdhs, err = lru.New2Q(c.config.MaxUUIDEntries)
	if err != nil {
		panic(err)
	}
	c.collections, err = lru.New2Q(c.config.MaxCollectionEntries)
	if err != nil {
		panic(err)
	}
	c.permissions, err = lru.New2Q(c.config.MaxPermissionEntries)
	if err != nil {
		panic(err)
	}

	reg := c.registry
	if reg == nil {
		reg = prometheus.NewRegistry()
	}
	c.metrics.setup(reg)
	go func() {
		for range time.Tick(metricsUpdateInterval) {
			c.updateGauges()
		}
	}()
}

func (c *cache) updateGauges() {
	c.metrics.collectionBytes.Set(float64(c.collectionBytes()))
	c.metrics.collectionEntries.Set(float64(c.collections.Len()))
}

var selectPDH = map[string]interface{}{
	"select": []string{"portable_data_hash"},
}

// Update saves a modified version (fs) to an existing collection
// (coll) and, if successful, updates the relevant cache entries so
// subsequent calls to Get() reflect the modifications.
func (c *cache) Update(client *arvados.Client, coll arvados.Collection, fs arvados.CollectionFileSystem) error {
	c.setupOnce.Do(c.setup)

	if m, err := fs.MarshalManifest("."); err != nil || m == coll.ManifestText {
		return err
	} else {
		coll.ManifestText = m
	}
	var updated arvados.Collection
	defer c.pdhs.Remove(coll.UUID)
	err := client.RequestAndDecode(&updated, "PATCH", "arvados/v1/collections/"+coll.UUID, nil, map[string]interface{}{
		"collection": map[string]string{
			"manifest_text": coll.ManifestText,
		},
	})
	if err == nil {
		c.collections.Add(client.AuthToken+"\000"+coll.PortableDataHash, &cachedCollection{
			expire:     time.Now().Add(time.Duration(c.config.TTL)),
			collection: &updated,
		})
	}
	return err
}

func (c *cache) Get(arv *arvadosclient.ArvadosClient, targetID string, forceReload bool) (*arvados.Collection, error) {
	c.setupOnce.Do(c.setup)
	c.metrics.requests.Inc()

	permOK := false
	permKey := arv.ApiToken + "\000" + targetID
	if forceReload {
	} else if ent, cached := c.permissions.Get(permKey); cached {
		ent := ent.(*cachedPermission)
		if ent.expire.Before(time.Now()) {
			c.permissions.Remove(permKey)
		} else {
			permOK = true
			c.metrics.permissionHits.Inc()
		}
	}

	var pdh string
	if arvadosclient.PDHMatch(targetID) {
		pdh = targetID
	} else if ent, cached := c.pdhs.Get(targetID); cached {
		ent := ent.(*cachedPDH)
		if ent.expire.Before(time.Now()) {
			c.pdhs.Remove(targetID)
		} else {
			pdh = ent.pdh
			c.metrics.pdhHits.Inc()
		}
	}

	var collection *arvados.Collection
	if pdh != "" {
		collection = c.lookupCollection(arv.ApiToken + "\000" + pdh)
	}

	if collection != nil && permOK {
		return collection, nil
	} else if collection != nil {
		// Ask API for current PDH for this targetID. Most
		// likely, the cached PDH is still correct; if so,
		// _and_ the current token has permission, we can
		// use our cached manifest.
		c.metrics.apiCalls.Inc()
		var current arvados.Collection
		err := arv.Get("collections", targetID, selectPDH, &current)
		if err != nil {
			return nil, err
		}
		if current.PortableDataHash == pdh {
			c.permissions.Add(permKey, &cachedPermission{
				expire: time.Now().Add(time.Duration(c.config.TTL)),
			})
			if pdh != targetID {
				c.pdhs.Add(targetID, &cachedPDH{
					expire: time.Now().Add(time.Duration(c.config.UUIDTTL)),
					pdh:    pdh,
				})
			}
			return collection, err
		} else {
			// PDH changed, but now we know we have
			// permission -- and maybe we already have the
			// new PDH in the cache.
			if coll := c.lookupCollection(arv.ApiToken + "\000" + current.PortableDataHash); coll != nil {
				return coll, nil
			}
		}
	}

	// Collection manifest is not cached.
	c.metrics.apiCalls.Inc()
	err := arv.Get("collections", targetID, nil, &collection)
	if err != nil {
		return nil, err
	}
	exp := time.Now().Add(time.Duration(c.config.TTL))
	c.permissions.Add(permKey, &cachedPermission{
		expire: exp,
	})
	c.pdhs.Add(targetID, &cachedPDH{
		expire: time.Now().Add(time.Duration(c.config.UUIDTTL)),
		pdh:    collection.PortableDataHash,
	})
	c.collections.Add(arv.ApiToken+"\000"+collection.PortableDataHash, &cachedCollection{
		expire:     exp,
		collection: collection,
	})
	if int64(len(collection.ManifestText)) > c.config.MaxCollectionBytes/int64(c.config.MaxCollectionEntries) {
		go c.pruneCollections()
	}
	return collection, nil
}

// pruneCollections checks the total bytes occupied by manifest_text
// in the collection cache and removes old entries as needed to bring
// the total size down to CollectionBytes. It also deletes all expired
// entries.
//
// pruneCollections does not aim to be perfectly correct when there is
// concurrent cache activity.
func (c *cache) pruneCollections() {
	var size int64
	now := time.Now()
	keys := c.collections.Keys()
	entsize := make([]int, len(keys))
	expired := make([]bool, len(keys))
	for i, k := range keys {
		v, ok := c.collections.Peek(k)
		if !ok {
			continue
		}
		ent := v.(*cachedCollection)
		n := len(ent.collection.ManifestText)
		size += int64(n)
		entsize[i] = n
		expired[i] = ent.expire.Before(now)
	}
	for i, k := range keys {
		if expired[i] {
			c.collections.Remove(k)
			size -= int64(entsize[i])
		}
	}
	for i, k := range keys {
		if size <= c.config.MaxCollectionBytes {
			break
		}
		if expired[i] {
			// already removed this entry in the previous loop
			continue
		}
		c.collections.Remove(k)
		size -= int64(entsize[i])
	}
}

// collectionBytes returns the approximate memory size of the
// collection cache.
func (c *cache) collectionBytes() uint64 {
	var size uint64
	for _, k := range c.collections.Keys() {
		v, ok := c.collections.Peek(k)
		if !ok {
			continue
		}
		size += uint64(len(v.(*cachedCollection).collection.ManifestText))
	}
	return size
}

func (c *cache) lookupCollection(key string) *arvados.Collection {
	e, cached := c.collections.Get(key)
	if !cached {
		return nil
	}
	ent := e.(*cachedCollection)
	if ent.expire.Before(time.Now()) {
		c.collections.Remove(key)
		return nil
	}
	c.metrics.collectionHits.Inc()
	return ent.collection
}
