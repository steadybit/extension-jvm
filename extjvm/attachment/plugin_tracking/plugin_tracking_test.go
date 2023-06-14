package plugin_tracking

import (
  "github.com/stretchr/testify/assert"
  "testing"
)

func TestAdd(t *testing.T) {
  Add(1, "/tmp/plugin1")
  Add(1, "/tmp/plugin2")

  has := Has(1, "/var/plugin1")
  assert.True(t, has)
  has = Has(1, "/var/plugin2")
  assert.True(t, has)

  Remove(1, "/lala/plugin1")
  has = Has(1, "/lala/plugin1")
  assert.False(t, has)
  has = Has(1, "/var/plugin2")
  assert.True(t, has)
}
