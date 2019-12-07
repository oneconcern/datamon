package errors

import (
	"fmt"
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

func TestStringer(t *testing.T) {
	e1 := New("cause1")
	e2 := New("cause2").Wrap(e1)
	e3 := New("cause3").Wrap(e2)
	expected := "cause3: cause2: cause1"
	assert.Equal(t, expected, e3.Error())
	assert.Equal(t, expected, e3.String())
	assert.Equal(t, expected, fmt.Sprintf("%v", e3))

	e4 := e3.Wrap(fmt.Errorf("std error"))
	expected = "cause3: std error"
	assert.Equal(t, expected, fmt.Sprintf("%v", e4))
}
