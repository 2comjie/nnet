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

//go:build !windows

package nnet

import (
	"fmt"
	"runtime"
	"sync/atomic"

	"github.com/2comjie/ntool/logx"
)

const (
	managerUninitialized = iota
	managerInitializing
	managerInitialized
)

func newManager(numLoops int) *manager {
	m := new(manager)
	m.SetLoadBalance(RoundRobin)
	m.SetNumLoops(numLoops)
	return m
}

type manager struct {
	numLoops int32
	status   int32       // 0: uninitialized, 1: initializing, 2: initialized
	balance  loadbalance // load balancing method
	polls    []Poll      // all the polls
}

func (m *manager) SetNumLoops(numLoops int) (err error) {
	if numLoops < 1 {
		return fmt.Errorf("set invalid numLoops[%d]", numLoops)
	}
	atomic.StoreInt32(&m.numLoops, int32(numLoops))
	atomic.StoreInt32(&m.status, managerUninitialized)
	return nil
}

func (m *manager) SetLoadBalance(lb LoadBalance) error {
	if m.balance != nil && m.balance.LoadBalance() == lb {
		return nil
	}
	m.balance = newLoadbalance(lb, m.polls)
	return nil
}

func (m *manager) Close() (err error) {
	for _, poll := range m.polls {
		err = poll.Close()
	}
	m.numLoops = 0
	m.balance = nil
	m.polls = nil
	return err
}

func (m *manager) Run() (err error) {
	defer func() {
		if err != nil {
			_ = m.Close()
		}
	}()

	numLoops := int(atomic.LoadInt32(&m.numLoops))
	if numLoops == len(m.polls) {
		return nil
	}
	polls := make([]Poll, numLoops)
	if numLoops < len(m.polls) {
		// shrink polls
		copy(polls, m.polls[:numLoops])
		for idx := numLoops; idx < len(m.polls); idx++ {
			// close redundant polls
			if err = m.polls[idx].Close(); err != nil {
				logx.Errorf("NETPOLL: poller close failed: %v\n", err)
			}
		}
	} else {
		// growth polls
		copy(polls, m.polls)
		for idx := len(m.polls); idx < numLoops; idx++ {
			var poll Poll
			poll, err = openPoll()
			if err != nil {
				return err
			}
			polls[idx] = poll
			go func() {
				werr := poll.Wait()
				if werr != nil {
					logx.Errorf("NETPOLL: poller exited with error: %v\n", werr)
				}
			}()
		}
	}
	m.polls = polls

	m.balance.Rebalance(m.polls)
	return nil
}

func (m *manager) Reset() error {
	for _, poll := range m.polls {
		cerr := poll.Close()
		if cerr != nil {
			logx.Errorf("NETPOLL: poller close failed: %v\n", cerr)
		}
	}
	m.polls = nil
	return m.Run()
}

func (m *manager) Pick() Poll {
START:
	if atomic.LoadInt32(&m.status) == managerInitialized {
		return m.balance.Pick()
	}
	// slow path
	// try to get initializing lock failed, wait others finished the init work, and try again
	if !atomic.CompareAndSwapInt32(&m.status, managerUninitialized, managerInitializing) {
		runtime.Gosched()
		goto START
	}
	// adjust polls
	// m.Run() will finish very quickly, so will not many goroutines block on Pick.
	_ = m.Run()

	//nolint:staticcheck // SA9003: empty branch
	if !atomic.CompareAndSwapInt32(&m.status, managerInitializing, managerInitialized) {
		// SetNumLoops called during m.Run() which cause CAS failed
		// The polls will be adjusted next Pick
	}
	return m.balance.Pick()
}
