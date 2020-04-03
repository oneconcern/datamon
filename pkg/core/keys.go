package core

import (
	"context"
	"sync"

	"github.com/oneconcern/datamon/pkg/core/status"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
)

// keyBatchEvent catches a collection of keys with possible retrieval error
type keyBatchEvent struct {
	keys []string
	err  error
}

// fetchKeys fetches keys for repos in batches, then close the keyBatchChan channel upon completion or error.
func fetchKeys(iterator func(string) ([]string, string, error), keyBatchChan chan<- keyBatchEvent,
	doneChan <-chan struct{}, wg *sync.WaitGroup) {
	defer func() {
		close(keyBatchChan)
		wg.Done()
	}()

	var (
		ks   []string
		next string
		err  error
	)

	for {
		// get a batch of keys
		ks, next, err = iterator(next)
		if err != nil {
			select {
			case keyBatchChan <- keyBatchEvent{err: err}:
			case <-doneChan:
				select {
				case keyBatchChan <- keyBatchEvent{err: status.ErrInterrupted}:
				default:
				}
			}
			return
		}

		if len(ks) == 0 {
			break
		}

		select {
		case keyBatchChan <- keyBatchEvent{keys: ks}:
		case <-doneChan:
			select {
			case keyBatchChan <- keyBatchEvent{err: status.ErrInterrupted}:
			default:
			}
			return
		}

		if next == "" {
			break
		}
	}
}

func distributeKeys(keys []string) func(chan<- string, <-chan struct{}, *sync.WaitGroup) {
	return func(keyChan chan<- string, doneChan <-chan struct{}, wg *sync.WaitGroup) {
		defer func() {
			close(keyChan)
			wg.Done()
		}()
		for _, k := range keys {
			select {
			case keyChan <- k:
			case <-doneChan:
				return
			}
		}
	}
}

// mergeKeys representing different states for diamonds and splits.
//
// We assume here that the iterator walks over both "running" and "done" metadata:
// if, for some diamond, we find both, only the done key is returned. This way, we list only one key per object.
//
// To achieve this without stashing previously fetched keys, we rely on the following properties of the metadata keys.
//
// * (filtered) keys for the same diamond or split in different states are adjacent: the internal map remains sparsely
//   populated under normal conditions
// * keys are sorted: ".../{diamond-id}/diamond-done.yaml" will be fetched _before_ ".../{diamond-id}/diamond-running.yaml"
func mergeKeys(inputChan <-chan keyBatchEvent, outputChan chan<- keyBatchEvent, settings Settings, wg *sync.WaitGroup) {
	defer func() {
		close(outputChan)
		wg.Done()
	}()

	type stateMerge struct {
		isFinal bool
		count   int
		key     string
	}

	states := make(map[string]stateMerge, settings.batchSize)
	for batch := range inputChan {
		var err error
		filtered := make([]string, 0, len(batch.keys))
		for _, key := range batch.keys {
			apc, erp := model.GetArchivePathComponents(key)
			if erp != nil {
				err = erp
				continue
			}
			if apc.SplitID == "" && apc.DiamondID == "" {
				continue
			}
			// merge state for splits
			keyState := states[apc.DiamondID+apc.SplitID]

			var retained string
			if keyState.isFinal && !apc.IsFinalState {
				retained = keyState.key
			} else {
				retained = key
			}

			keyState = stateMerge{isFinal: keyState.isFinal || apc.IsFinalState, count: keyState.count + 1, key: retained}
			states[apc.DiamondID+apc.SplitID] = keyState

			// case settled when either we have found more than 1 state with file or we have found a running state only
			if keyState.count > 1 || keyState.count == 1 && !keyState.isFinal {
				filtered = append(filtered, retained)
				delete(states, apc.DiamondID+apc.SplitID)
			}
		}
		outputChan <- keyBatchEvent{
			keys: filtered,
			err:  err,
		}
	}
}

func versionedKeys(vstore storage.VersionedStore, inputChan <-chan keyBatchEvent, outputChan chan<- keyBatchEvent, settings Settings, wg *sync.WaitGroup) {
	defer func() {
		close(outputChan)
		wg.Done()
	}()

	for batch := range inputChan {
		var err error
		expanded := make([]string, 0, len(batch.keys)*10)
		for _, key := range batch.keys {
			versions, erv := vstore.KeyVersions(context.Background(), key)
			if erv != nil {
				err = erv
				continue // drain chan
			}
			for _, v := range versions {
				// internally, we follow the {key}#{version} convention
				expanded = append(expanded, key+"#"+v)
			}
		}
		outputChan <- keyBatchEvent{
			keys: expanded,
			err:  err,
		}
	}
}
