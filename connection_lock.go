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
)

type who = int32

const (
	none who = iota
	user
	poller
)

type key int32

const (
	closing    key = iota // 0: 关闭锁 —— 谁在执行关闭流程
	connecting            // 1: 建连锁 —— dial 建连过程（服务端连接用不到）
	processing            // 2: 处理锁 —— 锁住 OnRequest 回调的执行
	flushing              // 3: 发送锁 —— 锁住 outputBuffer（写缓冲）
	total                 // 4: 哨兵，用于声明数组长度 [total]int32
)

type locker struct {
	// keychain used for lock/unlock/stop operation by who.
	// 0 means unlock, 1 means locked, 2 means stop.
	keychain [total]int32
}

func (l *locker) closeBy(w who) (success bool) {
	return atomic.CompareAndSwapInt32(&l.keychain[closing], 0, w)
}

func (l *locker) isCloseBy(w who) bool {
	return atomic.LoadInt32(&l.keychain[closing]) == w
}

func (l *locker) status(k key) int32 {
	return atomic.LoadInt32(&l.keychain[k])
}

func (l *locker) force(k key, v int32) {
	atomic.StoreInt32(&l.keychain[k], v)
}

func (l *locker) lock(k key) (success bool) {
	return atomic.CompareAndSwapInt32(&l.keychain[k], 0, 1)
}

func (l *locker) unlock(k key) {
	atomic.StoreInt32(&l.keychain[k], 0)
}

func (l *locker) stop(k key) {
	for !atomic.CompareAndSwapInt32(&l.keychain[k], 0, 2) && atomic.LoadInt32(&l.keychain[k]) != 2 {
		runtime.Gosched()
	}
}

func (l *locker) isUnlock(k key) bool {
	return atomic.LoadInt32(&l.keychain[k]) == 0
}
