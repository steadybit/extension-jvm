package plugin_tracking

import (
  "github.com/rs/zerolog/log"
  "path/filepath"
  "strconv"
  "sync"
)

var (
	plugins = sync.Map{} //map[int][]string (plugin path)
)

func Add(pid int32, plugin string) {
  key := strconv.Itoa(int(pid))
  plugin = filepath.Base(plugin)
	value, ok := plugins.Load(key)
	if !ok {
		value = []string{}
	}
	value = append(value.([]string), plugin)
	plugins.Store(key, value)
}

func Remove(pid int32, plugin string) {
  key := strconv.Itoa(int(pid))
  plugin = filepath.Base(plugin)
	value, ok := plugins.Load(key)
	if !ok {
		return
	}
	value = remove(value.([]string), plugin)
	plugins.Store(key, value)
}

func RemoveAll(pid int32) {
  key := strconv.Itoa(int(pid))
	plugins.Delete(key)
}

func Has(pid int32, plugin string) bool {
  plugin = filepath.Base(plugin)
  key := strconv.Itoa(int(pid))
	value, ok := plugins.Load(key)
	if !ok {
    log.Trace().Msgf("No plugins found for pid %d", pid)
		return false
	}
	for _, v := range value.([]string) {
		if v == plugin {
			return true
		}
	}
  log.Debug().Msgf("Plugin %s not found in %v", plugin, value)
	return false
}

func remove(s []string, r string) []string {
	for i, v := range s {
		if v == r {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}
