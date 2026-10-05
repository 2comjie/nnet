// Copyright 2022 CloudWeGo Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package nnet

import (
	"runtime"
	"sync/atomic"
	"unsafe"
)

type operatorCache struct {
	first      *FDOperator
	cache      []*FDOperator
	locked     int32
	freeLocked int32
	freelist   []int32
}

func newOperatorCache() *operatorCache {
	return &operatorCache{
		cache:    make([]*FDOperator, 0, 1024),
		freelist: make([]int32, 0, 1024),
	}
}

func (c *operatorCache) alloc() *FDOperator {
	lock(&c.locked)
	if c.first == nil {
		const opSize = unsafe.Sizeof(FDOperator{})
		n := 4 * 1024 * 1024 / opSize
		if n == 0 {
			n = 1
		}
		index := int32(len(c.cache))
		for i := uintptr(0); i < n; i++ {
			op := &FDOperator{
				index: index,
			}
			op.next = c.first
			c.first = op
			c.cache = append(c.cache, op)
			index++
		}
	}

	op := c.first
	c.first = op.next
	unlock(&c.locked)
	return op
}

func (c *operatorCache) freeable(op *FDOperator) {
	op.unused()
	op.reset()
	lock(&c.freeLocked)
	c.freelist = append(c.freelist, op.index)
	unlock(&c.freeLocked)
}

func (c *operatorCache) free() {
	lock(&c.freeLocked)
	defer unlock(&c.freeLocked)
	if len(c.freelist) == 0 {
		return
	}

	// 头插
	lock(&c.locked)
	for _, idx := range c.freelist {
		op := c.cache[idx]
		op.next = c.first
		c.first = op
	}
	c.freelist = c.freelist[:0]
	unlock(&c.locked)
}

func lock(locked *int32) {
	for !atomic.CompareAndSwapInt32(locked, 0, 1) {
		runtime.Gosched()
	}
}
func unlock(locked *int32) {
	atomic.StoreInt32(locked, 0)
}
