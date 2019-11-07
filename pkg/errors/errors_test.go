package errors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestError(t *testing.T) {
	e1 := New("cause1")
	e2 := New("cause2").Wrap(e1)
	e := New("dummy").Wrap(e2)
	e3 := e.Unwrap()
	assert.True(t, Is(e, e1))
	assert.True(t, Is(e, e2))
	assert.True(t, e3 == e2)
}
