package kubeless

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGlobPatternCheckWithGlob(t *testing.T)  {
	globPattern := "vendor/*"

	assert.Equal(t, "vendor", globPatternCheck(globPattern))
}

func TestGlobPatterCheckWithoutGlob(t *testing.T)  {
	globPattern := "helloget.py"

	assert.Equal(t, "helloget.py", globPatternCheck(globPattern))
}
