package remote_jvm_connections

import (
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/utils"
	"strconv"
	"sync"
	"time"
)

type SocketConnection struct {
	Host  string
	Port  int
	Mutex sync.Mutex
}

func (a *SocketConnection) Address() string {
	return a.Host + ":" + strconv.Itoa(a.Port)
}

var (
	connections = sync.Map{} //map[int32]SocketConnection (IP address)
)

func WaitForConnection(pid int32, timeout time.Duration) bool {
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		for {
			if GetConnection(pid) != nil {
				wg.Done()
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()
	if utils.WaitTimeout(&wg, timeout) {
		return false
	} else {
		return true
	}
}

func GetConnection(pid int32) *SocketConnection {
	ipaddress, ok := connections.Load(pid)
	if !ok {
		log.Trace().Msgf("No connection found for pid %d", pid)
		return nil
	}
	return ipaddress.(*SocketConnection)
}

func AddConnection(pid int32, host string, port int) {
	log.Info().Msgf("Adding connection for PID %d on %s:%d", pid, host, port)
	connection := SocketConnection{Host: host, Port: port}
	existingConnection := GetConnection(pid)
	if existingConnection != nil && existingConnection.Host == host && existingConnection.Port == port {
		log.Debug().Msgf("JVM connection with with PID %d on %s exists already. Skipping registration", pid, connection.Address())
		return
	}
	connectionIsNew := false
	if existingConnection == nil {
		connectionIsNew = true
	}
	connections.Store(pid, &connection)
	if connectionIsNew {
		log.Debug().Msgf("JVM connection with PID %d on %s registered", pid, connection.Address())
	} else {
		log.Debug().Msgf("JVM connection with PID %d on %s updated", pid, connection.Address())
	}
}

func RemoveConnection(pid int32) {
	connections.Delete(pid)
	log.Trace().Msgf("JVM connection with PID %d removed", pid)
}

func ClearConnections() {
	connections = sync.Map{}
	log.Debug().Msg("All JVM connections removed")
}

func Size() int {
	size := 0
	connections.Range(func(key, value interface{}) bool {
		size++
		return true
	})
	return size
}
