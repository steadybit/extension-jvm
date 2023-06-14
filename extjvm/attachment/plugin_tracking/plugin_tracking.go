package plugin_tracking

import (
  "path/filepath"
  "sync"
)

var (
	plugins = sync.Map{} //map[int][]string (plugin path)
)

func Add(pid int32, plugin string) {
  plugin = filepath.Base(plugin)
	value, ok := plugins.Load(pid)
	if !ok {
		value = []string{}
	}
	value = append(value.([]string), plugin)
	plugins.Store(pid, value)
}

func Remove(pid int32, plugin string) {
  plugin = filepath.Base(plugin)
	value, ok := plugins.Load(pid)
	if !ok {
		return
	}
	value = remove(value.([]string), plugin)
	plugins.Store(pid, value)
}

func RemoveAll(pid int32) {
	plugins.Delete(pid)
}

func Has(pid int32, plugin string) bool {
  plugin = filepath.Base(plugin)
	value, ok := plugins.Load(pid)
	if !ok {
		return false
	}
	for _, v := range value.([]string) {
		if v == plugin {
			return true
		}
	}
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
