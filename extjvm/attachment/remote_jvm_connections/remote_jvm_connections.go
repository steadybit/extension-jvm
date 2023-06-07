package remote_jvm_connections

import (
  "github.com/rs/zerolog/log"
  "github.com/steadybit/extension-kit/extutil"
  "sync"
  "time"
)


type InetSocketAddress struct {
  Host string
  Port int
}

func (a InetSocketAddress) Address() string {
  return a.Host + ":" + extutil.ToString(a.Port)
}
var (
  connections = sync.Map{} //map[int32]InetSocketAddress (IP address)
)

func WaitForConnection(pid int32, timeout time.Duration) bool {
  //TODO: implement
  return false
}

func GetConnection(pid int32) *InetSocketAddress {
  ipaddress, ok := connections.Load(pid)
  if !ok {
    log.Warn().Msgf("No connection found for pid %d", pid)
    return nil
  }
  return extutil.Ptr(ipaddress.(InetSocketAddress))
}

func AddConnection(pid int32, host string, port int) {
  connection := InetSocketAddress{Host: host, Port: port}
  existingConnection := GetConnection(pid)
  if existingConnection != nil && existingConnection.Host == host && existingConnection.Port == port {
    log.Debug().Msgf("JVM connection with  with PID %d on %+v exists already. Skipping registration", pid, connection)
    return
  }
  connectionIsNew := false
  if existingConnection == nil {
    connectionIsNew = true
  }
  connections.Store(pid, connection)
  if connectionIsNew {
    log.Debug().Msgf("JVM connection with PID %d on %+v registered", pid, connection)
  } else {
    log.Debug().Msgf("JVM connection with PID %d on %+v updated", pid, connection)
  }
}

func RemoveConnection(pid int32) {
  connections.Delete(pid)
  log.Trace().Msgf("JVM connection with PID %d removed", pid)
}

func ClearConnections() {
  connections = sync.Map{}
  log.Trace().Msg("All JVM connections removed")
}

func Size() int {
  size := 0
  connections.Range(func(key, value interface{}) bool {
    size++
    return true
  })
  return size
}
