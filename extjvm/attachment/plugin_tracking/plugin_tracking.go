package plugin_tracking

import (
	"github.com/steadybit/extension-jvm/extjvm"
	"sync"
)

var (
	plugins = sync.Map{} //map[int][]string (plugin path)
)

func Add(jvm *extjvm.JavaVm, plugin string) {
	value, ok := plugins.Load(jvm.Pid)
	if !ok {
		value = []string{}
	}
	value = append(value.([]string), plugin)
	plugins.Store(jvm.Pid, value)
}

func Remove(jvm *extjvm.JavaVm, plugin string) {
	value, ok := plugins.Load(jvm.Pid)
	if !ok {
		return
	}
	value = remove(value.([]string), plugin)
	plugins.Store(jvm.Pid, value)
}

func RemoveAll(jvm *extjvm.JavaVm) {
	plugins.Delete(jvm.Pid)
}

func Has(jvm *extjvm.JavaVm, plugin string) bool {
	value, ok := plugins.Load(jvm.Pid)
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
