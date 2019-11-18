package core

import (
	"sync"

	"github.com/oneconcern/datamon/pkg/core/status"
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
