package hnc

import (
	"fmt"
)

type productCache struct {
	appCache interface{}
}

func newProductCache(in interface{}) (*productCache, error) {
	if in == nil {
		return &productCache{}, nil
	}

	cache, ok := in.(*productCache)
	if !ok {
		return nil, fmt.Errorf("input is not a product cache")
	}
	return cache.DeepCopy(), nil
}

func (c *productCache) DeepCopy() *productCache {
	appCache := c.appCache
	return &productCache{
		appCache: appCache,
	}
}
