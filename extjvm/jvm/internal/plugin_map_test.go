package internal

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAdd(t *testing.T) {
	m := PluginMap{}

	m.Add(1, "/tmp/plugin1")
	m.Add(1, "/tmp/plugin2")

	has := m.Has(1, "/var/plugin1")
	assert.True(t, has)
	has = m.Has(1, "/var/plugin2")
	assert.True(t, has)

	m.Remove(1, "/lala/plugin1")
	has = m.Has(1, "/lala/plugin1")
	assert.False(t, has)
	has = m.Has(1, "/var/plugin2")
	assert.True(t, has)
}
