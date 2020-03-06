package core

import (
	"sort"
	"sync"

	"testing"

	"github.com/oneconcern/datamon/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeKeys(t *testing.T) {
	unfilteredKeysChan := make(chan keyBatchEvent, 1)
	keysChan := make(chan keyBatchEvent, 1)
	var wg sync.WaitGroup
	settings := Settings{batchSize: 5}

	wg.Add(1)
	go mergeKeys(unfilteredKeysChan, keysChan, settings, &wg)

	wg.Add(1)
	go func(output chan<- keyBatchEvent) {
		defer func() {
			close(output)
			wg.Done()
		}()
		for _, batch := range mergeFixture() {
			sort.Strings(batch) // ensure fixture is sorted
			unfilteredKeysChan <- keyBatchEvent{keys: batch}
		}
	}(unfilteredKeysChan)

	results := make(map[string]bool, 100)
	for input := range keysChan {
		for _, k := range input.keys {
			assert.NoError(t, input.err)
			assert.NotContains(t, results, k) // unique output
			results[k] = true
		}
	}
	wg.Wait()
	assertMergeFixture(t, results)
}

func mergeFixture() [][]string {
	return [][]string{
		{
			"diamonds/myrepo/0ujssxh0cECutqzMgbtXSGnjorm/diamond-done.yaml",
			"diamonds/myrepo/0ujssxh0cECutqzMgbtXSGnjorm/diamond-running.yaml",
			"diamonds/myrepo/0ujsszgFvbiEr7CDgE3z8MAUPFt/diamond-running.yaml",
			"diamonds/myrepo/0ujsszwN8NRY24YaXiTIE2VWDTS/diamond-running.yaml",
			"diamonds/myrepo/0ujsszwN8NRY24YaXiTIE2VWDTS/diamond-done.yaml",
			"diamonds/myrepo/0ujsswThIGTUYm2K8FjOOfXtY1K/diamond-done.yaml",
			"diamonds/myrepo/0ujssxh0cECutqzMgbtXSGnjorm/splits/0ujtsYcgvSTl8PAuAdqWYSMnLOv/split-done.yaml",
			"diamonds/myrepo/0ujssxh0cECutqzMgbtXSGnjorm/splits/0ujtsYcgvSTl8PAuAdqWYSMnLOv/split-running.yaml",
			"diamonds/myrepo/0ujssxh0cECutqzMgbtXSGnjorm/splits/i0uk1Ha7hGJ1Q9Xbnkt0yZgNwg3g/split-done.yaml",
		},
		{
			"diamonds/myrepo/0ujsswThIGTUYm2K8FjOOfXtY1K/diamond-running.yaml",                                   // important: key "running" comes AFTER key "done"
			"diamonds/myrepo/0ujssxh0cECutqzMgbtXSGnjorm/splits/i0uk1Ha7hGJ1Q9Xbnkt0yZgNwg3g/split-running.yaml", // important: key "running" comes AFTER key "done"
			"diamonds/myrepo/0ujssxh0cECutqzMgbtXSGnjorm/splits/0uk1Hbc9dQ9pxyTqJ93IUrfhdGq/split-running.yaml",
		},
		{
			"diamonds/myrepo/0ujsswThIGTUYm2K8FjOOfXtY1K/splits/0uk1HdCJ6hUZKDgcxhpJwUl5ZEI/split-running.yaml",
		},
	}
}

func assertMergeFixture(t testing.TB, results map[string]bool) {
	require.Len(t, results, 4+3+1)
	for k := range results {
		apc, err := model.GetArchivePathComponents(k)
		require.NoError(t, err)
		switch apc.DiamondID {
		case "0ujssxh0cECutqzMgbtXSGnjorm":
			switch apc.SplitID {
			case "0ujtsYcgvSTl8PAuAdqWYSMnLOv", "i0uk1Ha7hGJ1Q9Xbnkt0yZgNwg3g":
				assert.True(t, apc.IsFinalState, "unexpected merged state for SplitID: %s, diamondID: %s", apc.SplitID, apc.DiamondID)
			case "0uk1Hbc9dQ9pxyTqJ93IUrfhdGq":
				assert.False(t, apc.IsFinalState, "unexpected merged state for SplitID: %s, diamondID: %s", apc.SplitID, apc.DiamondID)
			case "":
				assert.True(t, apc.IsFinalState, "unexpected merged state for diamondID: %s", apc.DiamondID)
			default:
				t.Logf("unexpected SplitID %s for DiamondID: %s", apc.SplitID, apc.DiamondID)
				t.Fail()
			}
		case "0ujsszwN8NRY24YaXiTIE2VWDTS":
			assert.True(t, apc.IsFinalState, "unexpected merged state for diamondID: %s", apc.DiamondID)
			assert.Empty(t, apc.SplitID)
		case "0ujsswThIGTUYm2K8FjOOfXtY1K":
			switch apc.SplitID {
			case "0uk1HdCJ6hUZKDgcxhpJwUl5ZEI":
				assert.False(t, apc.IsFinalState, "unexpected merged state for SplitID: %s, diamondID: %s", apc.SplitID, apc.DiamondID)
			case "":
				assert.True(t, apc.IsFinalState, "unexpected merged state for diamondID: %s", apc.DiamondID)
			default:
				t.Logf("unexpected SplitID %s for DiamondID: %s", apc.SplitID, apc.DiamondID)
				t.Fail()
			}
		case "0ujsszgFvbiEr7CDgE3z8MAUPFt":
			assert.False(t, apc.IsFinalState)
			assert.Empty(t, apc.SplitID)
		default:
			t.Logf("unexpected DiamondID: %s", apc.DiamondID)
			t.Fail()
		}
	}
}
