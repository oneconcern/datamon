// +build influxdbintegration

package influxdb

import (
	"context"
	"testing"
	"time"

	influxdb "github.com/influxdata/influxdb/client/v2"
	"github.com/stretchr/testify/require"
)

func TestClient(t *testing.T) {
	store, err := NewStore()
	require.NoError(t, err)

	require.NoError(t, store.Ping(context.Background(), 1*time.Second))

	client := store.GetClient()

	bp, err := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
		Database:  store.Database(),
		Precision: "s",
	})
	require.NoError(t, err)

	pt, err := influxdb.NewPoint("myview", map[string]string{"mytag": "myvalue"}, map[string]interface{}{"counter": int64(34)})
	require.NoError(t, err)

	bp.AddPoint(pt)

	require.NoError(t, client.Write(bp))
}
