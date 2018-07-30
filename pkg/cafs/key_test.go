package cafs

import (
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testKey = "911bc2b07dd96c21ef3ab8b56ffeca4e0b8d1b74ea7667dd67eb2d037c1b4880d3b2533035d90f84ceb326ca9f0c47bb75e0ed3e86c959ab8d687b1739677278"

func TestKey_FailsOnIncorrectSize(t *testing.T) {

	data1 := make([]byte, 63)
	data2 := make([]byte, 65)

	_, err := rand.Read(data1)
	require.NoError(t, err)
	_, err = rand.Read(data2)
	require.NoError(t, err)

	_, err = NewKey(data1)
	require.Error(t, err)

	k, err := NewKey(data2)
	require.NoError(t, err)
	assert.Len(t, k, 64)

	assert.Panics(t, func() { MustNewKey(data1) })
	assert.NotPanics(t, func() { MustNewKey(data2) })
}

func TestKey_Succeeds(t *testing.T) {
	data, err := hex.DecodeString(testKey)
	require.NoError(t, err)

	key, err := NewKey(data)
	require.NoError(t, err)
	assert.Equal(t, testKey, key.String())
}
