package jvm

import (
	"context"
	"github.com/rs/zerolog/log"
	"sync"
	"time"
)

type jvmConnections struct {
	connections sync.Map //map[int32]jvmConnection (IP address)
}

type jvmConnection struct {
	Address string
	m       sync.Mutex
}

func (c *jvmConnection) lock() {
	c.m.Lock()
}

func (c *jvmConnection) unlock() {
	c.m.Unlock()
}

func (c *jvmConnections) waitForConnection(pid int32, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(100 * time.Millisecond):
			if c.getConnection(pid) != nil {
				return true
			}
		}
	}
}

func (c *jvmConnections) getConnection(pid int32) *jvmConnection {
	ipaddress, ok := c.connections.Load(pid)
	if !ok {
		log.Trace().Msgf("No connection found for pid %d", pid)
		return nil
	}
	return ipaddress.(*jvmConnection)
}

func (c *jvmConnections) addConnection(pid int32, address string) {
	log.Info().Msgf("Adding connection for PID %d on %s", pid, address)

	existing := c.getConnection(pid)
	if existing != nil && existing.Address == address {
		log.Debug().Msgf("JVM connection with with PID %d on %s exists already. Skipping registration", pid, address)
		return
	}

	c.connections.Store(pid, &jvmConnection{Address: address})
	if existing == nil {
		log.Debug().Msgf("JVM connection with PID %d on %s registered", pid, address)
	} else {
		log.Debug().Msgf("JVM connection with PID %d on %s updated", pid, address)
	}
}

func (c *jvmConnections) removeConnection(pid int32) {
	if conn, ok := c.connections.LoadAndDelete(pid); ok {
		log.Trace().Msgf("JVM connection with PID %d an %s removed", pid, conn.(*jvmConnection).Address)
	}
}
