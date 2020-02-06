package core

import (
	"context"
	"fmt"
	"io/ioutil"
	"sort"
	"sync"

	context2 "github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/core/status"
	"github.com/oneconcern/datamon/pkg/errors"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	storagestatus "github.com/oneconcern/datamon/pkg/storage/status"
	"gopkg.in/yaml.v2"
)

const (
	typicalSplitsNum = 100 // default number of allocated slots for splits in a diamond
)

// splitEvent catches a single split with possible retrieval error
type splitEvent struct {
	split model.SplitDescriptor
	err   error
}

// splitsEvent catches a collection of splits with possible retrieval error
type splitsEvent struct {
	splits model.SplitDescriptors
	err    error
}

// doSelectSplits is a helper function to listen on a channel of batches of split descriptors.
//
// It applies some function on the received batches and returns upon completion or error.
//
// Example usage:
//
//		err := doSelectSplits(splitsChan, func(splitBatch model.SplitDescriptors) {
//			splits = append(splits, splitBatch...)
//		})
func doSelectSplits(splitsChan <-chan splitsEvent, do func(model.SplitDescriptors)) error {
	// consume batches of ordered split metadata
	for splitBatch := range splitsChan {
		if splitBatch.err != nil {
			return splitBatch.err
		}
		do(splitBatch.splits)
	}
	return nil
}

// ApplySplitFunc is a function to be applied on a split
type ApplySplitFunc func(model.SplitDescriptor) error

// ListSplitsApply applies some function to the retrieved splits, ordered by completion time.
//
// The execution of the applied function does not block background retrieval of more keys and split descriptors.
//
// Example usage: printing split descriptors as they come
//
//   err := core.ListSplitsApply(repo, store, func(split model.SplitDescriptor) error {
//				fmt.Fprintf(os.Stderr, "%v\n", split)
//				return nil
//			})
func ListSplitsApply(repo, diamondID string, stores context2.Stores, apply ApplySplitFunc, opts ...Option) error {
	var (
		err, applyErr error
		once          sync.Once
	)

	splitChan := make(chan model.SplitDescriptor)
	doneChan := make(chan struct{}, 1)

	clean := func() {
		close(doneChan)
	}
	interruptAndClean := func() {
		doneChan <- struct{}{}
		close(doneChan)
	}

	// collect split metadata asynchronously
	go func(splitChan chan<- model.SplitDescriptor, doneChan chan struct{}) {
		defer close(splitChan)

		splitsChan, workers := listSplitsChan(repo, diamondID, stores, append(opts, WithDoneChan(doneChan))...)

		err = doSelectSplits(splitsChan, func(splitBatch model.SplitDescriptors) {
			for _, split := range splitBatch {
				splitChan <- split // transfer a batch of metadata to the applied func
			}
		})
		once.Do(clean)

		workers.Wait()
	}(splitChan, doneChan)

	// apply function on collected metadata
	for split := range splitChan {
		if applyErr = apply(split); applyErr != nil {
			// wind down goroutines, but when nothing is left to be interrupted
			once.Do(interruptAndClean)
			for range splitChan {
			} // wait for close
			break
		}
	}
	// collect errors
	switch {
	case err == status.ErrInterrupted && applyErr != nil:
		return applyErr
	case err != nil:
		return err
	case applyErr != nil:
		return applyErr
	default:
		return nil
	}
}

// ListSplits yields all ongoing splits on a repo and a given diamond
func ListSplits(repo, diamondID string, stores context2.Stores, opts ...Option) (model.SplitDescriptors, error) {
	splits := make(model.SplitDescriptors, 0, typicalSplitsNum)

	splitsChan, workers := listSplitsChan(repo, diamondID, stores, opts...)

	// consume batches of ordered splits
	err := doSelectSplits(splitsChan, func(splitBatch model.SplitDescriptors) {
		splits = append(splits, splitBatch...)
	})

	workers.Wait()

	return splits, err // we may have some batches resolved before the error occurred
}

func listSplitsChan(repo, diamondID string, stores context2.Stores, opts ...Option) (chan splitsEvent, *sync.WaitGroup) {
	var wg sync.WaitGroup

	settings := defaultSettings()
	for _, bApply := range opts {
		bApply(&settings)
	}

	batchChan := make(chan splitsEvent, 1) // buffered to 1 to avoid blocking on early errors

	if err := RepoExists(repo, stores); err != nil {
		batchChan <- splitsEvent{err: err}
		close(batchChan)
		return batchChan, &wg
	}

	if err := DiamondExists(repo, diamondID, stores); err != nil {
		batchChan <- splitsEvent{err: err}
		close(batchChan)
		return batchChan, &wg
	}

	// internal signaling channels
	doneWithKeysChan := make(chan struct{}, 1)
	doneWithSplitsChan := make(chan struct{}, 1)

	if settings.doneChannel != nil {
		// watch for an interruption signal requested by caller
		wg.Add(1)
		go watchForInterrupts(settings.doneChannel, &wg, doneWithKeysChan, doneWithSplitsChan)
	}

	unfilteredKeysChan := make(chan keyBatchEvent, 1)
	keysChan := make(chan keyBatchEvent, 1)

	iterator := func(next string) ([]string, string, error) {
		return basenameKeyFilter("split-")(
			// restrain result to split descriptors (in any state)
			GetSplitStore(stores).KeysPrefix(context.Background(), next, model.GetArchivePathPrefixToSplits(repo, diamondID), "", settings.batchSize),
		)
	}

	// starting keys retrieval
	wg.Add(1)
	go fetchKeys(iterator, unfilteredKeysChan, doneWithKeysChan, &wg) // scan for key batches

	// keys state filtering & merging
	wg.Add(1)
	go mergeKeys(unfilteredKeysChan, keysChan, settings, &wg)

	// start split metadata retrieval
	wg.Add(1)
	go fetchSplits(repo, GetSplitStore(stores), settings, keysChan, batchChan, doneWithKeysChan, doneWithSplitsChan, &wg)

	// let the gc clean up internal signaling channels left open after wg goroutines are done.

	// return at once. Caller may chose to wait on returned WaitGroup
	return batchChan, &wg
}

// fetchSplits waits on a channel of key batches and outputs batches of descriptors corresponding to these keys
func fetchSplits(repo string, store storage.Store, settings Settings,
	keysChan <-chan keyBatchEvent, batchChan chan<- splitsEvent,
	doneWithKeysChan chan<- struct{}, doneChan <-chan struct{}, wg *sync.WaitGroup) {
	defer func() {
		close(batchChan)
		wg.Done()
	}()

	for {
		select {
		case <-doneChan:
			batchChan <- splitsEvent{err: status.ErrInterrupted}
			return
		case keyBatch, isOpen := <-keysChan:
			if !isOpen {
				return
			}
			if keyBatch.err != nil {
				batchChan <- splitsEvent{err: keyBatch.err}
				return
			}
			batch, err := fetchSplitBatch(repo, store, settings, keyBatch.keys)
			if err != nil {
				doneWithKeysChan <- struct{}{} // stop co-worker
				batchChan <- splitsEvent{err: err}
				return
			}
			// send out a single batch of (ordered) split descriptors
			batchChan <- splitsEvent{splits: batch}
		}
	}
}

// fetchSplitBatch performs a parallel fetch for a batch of splits identified by their keys,
// then reorders the result by key.
//
// TODO: this performs a parallel fetch for a batch of keys. However, we wait until completion of this batch to start
// a new one. In addition, for every new batch of key, we spin up a new pool of workers.
// We could improve this further by streaming batches of keys then stashing looked-ahead results and directly obtain
// a sorted output.
func fetchSplitBatch(repo string, store storage.Store, settings Settings, keys []string) (model.SplitDescriptors, error) {
	var (
		workers, wg sync.WaitGroup
		werr        error
	)

	splitChan := make(chan splitEvent)
	keyChan := make(chan string)
	doneChan := make(chan struct{}, 1)
	defer close(doneChan)

	// spin up workers pool
	for i := 0; i < minInt(settings.concurrentList, len(keys)); i++ {
		workers.Add(1)
		go getSplitAsync(repo, store, keyChan, splitChan, &workers)
	}

	bds := make(model.SplitDescriptors, 0, len(keys))

	// distribute work. Stop immediately on first error reported by a worker
	wg.Add(1)
	go distributeKeys(keys)(keyChan, doneChan, &wg)

	// wait for workers to complete
	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		workers.Wait()
		close(splitChan)
	}(&wg)

	// watch for results and coalesce
	for bd := range splitChan {
		if bd.err != nil && werr == nil {
			werr = bd.err
			doneChan <- struct{}{} // interrupts key distribution (non-blocking)
			for range splitChan {
			} // wait for close
			break
		}
		bds = append(bds, bd.split)
	}

	wg.Wait()

	if werr != nil {
		return nil, werr
	}

	// sort result batch
	sort.Sort(bds)
	return bds, nil
}

// getSplitAsync fetches and unmarshalls the split descriptor for each single key submitted as input
func getSplitAsync(repo string, store storage.Store, input <-chan string, output chan<- splitEvent,
	wg *sync.WaitGroup) {
	defer wg.Done()

	for k := range input {
		bd, err := readSplit(repo, k, store)
		if err != nil {
			if errors.Is(err, storagestatus.ErrNotExists) {
				continue
			}
			output <- splitEvent{err: err}
			continue
		}

		output <- splitEvent{split: bd}
	}
}

func readSplit(repo, k string, store storage.Store) (model.SplitDescriptor, error) {
	apc, err := model.GetArchivePathComponents(k)
	if err != nil {
		return model.SplitDescriptor{}, err
	}

	var src string
	if apc.IsFinalState {
		src = model.GetArchivePathToFinalSplit(repo, apc.DiamondID, apc.SplitID)
	} else {
		src = model.GetArchivePathToInitialSplit(repo, apc.DiamondID, apc.SplitID)
	}

	r, err := store.Get(context.Background(), src)
	if err != nil {
		return model.SplitDescriptor{}, err
	}

	o, err := ioutil.ReadAll(r)
	if err != nil {
		return model.SplitDescriptor{}, err
	}

	var sd model.SplitDescriptor
	err = yaml.Unmarshal(o, &sd)
	if err != nil {
		return model.SplitDescriptor{}, err
	}

	if sd.SplitID != apc.SplitID {
		err = fmt.Errorf("split IDs in descriptor '%v' and archive path '%v' don't match", sd.SplitID, apc.SplitID)
		return model.SplitDescriptor{}, err
	}

	return sd, nil
}
