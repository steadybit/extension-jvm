// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2024 Steadybit GmbH

package internal

import (
	"path/filepath"
	"slices"
	"sync"
)

type PluginMap struct {
	plugins sync.Map //map[int][]string (plugin path)
}

func (p *PluginMap) Add(pid int32, plugin string) {
	value, ok := p.plugins.Load(pid)
	if !ok {
		value = []string{}
	}
	normalized := filepath.Base(plugin)
	value = append(value.([]string), normalized)
	p.plugins.Store(pid, value)
}

func (p *PluginMap) Remove(pid int32, plugin string) {
	value, ok := p.plugins.Load(pid)
	if !ok {
		return
	}
	normalized := filepath.Base(plugin)
	value = slices.DeleteFunc(value.([]string), func(s string) bool {
		return s == normalized
	})
	p.plugins.Store(pid, value)
}

func (p *PluginMap) RemoveAll(pid int32) {
	p.plugins.Delete(pid)
}

func (p *PluginMap) Has(pid int32, plugin string) bool {
	normalized := filepath.Base(plugin)
	if value, ok := p.plugins.Load(pid); ok {
		return slices.Contains(value.([]string), normalized)
	}
	return false
}
