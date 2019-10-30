package google

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrincipal(t *testing.T) {
	p, err := New().Principal("")
	require.NoError(t, err)
	require.NotEmpty(t, p.Email)
	require.NotEmpty(t, p.Name)
	t.Logf("Tested principal: %#v", p)
}
