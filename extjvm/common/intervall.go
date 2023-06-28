package common

import (
  "os"
)

func GetDiscoveryCallInterval() string {
  interval := os.Getenv("STEADYBIT_EXTENSION_DISCOVERY_CALL_INTERVAL")
  if interval != "" {
    return interval
  }
  return "1m"
}
