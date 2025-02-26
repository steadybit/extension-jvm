// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2024 Steadybit GmbH

package jvm

import (
	"github.com/steadybit/extension-jvm/extjvm/jvm/test"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func Test_should_track_jvm(t *testing.T) {
	sleep := test.NewSleep()
	defer sleep.Stop()

	jvms := newJavaVms()
	added := sink[JavaVm]{}
	removed := sink[JavaVm]{}
	go added.drain(jvms.Added)
	go removed.drain(jvms.Removed)

	javaVm := newJavaVm(sleep.Process(), "test")
	jvms.addJvm(javaVm)
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.Contains(c, added.list(), javaVm)
	}, 10*time.Second, 10*time.Millisecond)

	sleepPid := sleep.Pid()
	assert.NotNil(t, jvms.getJvm(sleepPid))

	sleep.Stop()
	jvms.getJvms()
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.Contains(c, removed.list(), javaVm)
	}, 10*time.Second, 10*time.Millisecond)

	assert.Nil(t, jvms.getJvm(sleepPid))
}

type sink[T any] struct {
	m sync.Mutex
	l []T
}

func (s *sink[T]) drain(ch <-chan T) {
	for j := range ch {
		func() {
			s.m.Lock()
			defer s.m.Unlock()
			s.l = append(s.l, j)
		}()
	}
}

func (s *sink[T]) list() []T {
	s.m.Lock()
	defer s.m.Unlock()
	c := make([]T, len(s.l))
	copy(c, s.l)
	return c
}
