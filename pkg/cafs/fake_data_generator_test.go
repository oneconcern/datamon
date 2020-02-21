package cafs

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type generateFixture struct {
	target   string
	size     int
	leafSize uint32
}

func TestGenerateFile(t *testing.T) {
	tmp, erd := ioutil.TempDir("", "faker-")
	require.NoError(t, erd)

	defer func() {
		_ = os.RemoveAll(tmp)
	}()

	for _, gen := range []generateFixture{
		{target: "file1", size: 20},
		{target: "file2", size: 2100, leafSize: 200},
	} {
		file := filepath.Join(tmp, gen.target)
		assert.NoError(t, GenerateFile(file, gen.size, gen.leafSize))
		info, err := os.Stat(file)
		require.NoError(t, err)
		assert.Equal(t, gen.size, int(info.Size()))
	}
}
