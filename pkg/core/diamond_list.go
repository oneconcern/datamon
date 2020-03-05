package core

import (
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
	typicalDiamondsNum = 100 // default number of allocated slots for diamonds in a diamond
)

// diamondEvent catches a single diamond with possible retrieval error
type diamondEvent struct {
	diamond model.DiamondDescriptor
	err     error
}

// diamondsEvent catches a collection of diamonds with possible retrieval error
type diamondsEvent struct {
	diamonds model.DiamondDescriptors
	err      error
}

// doSelectDiamonds is a helper function to listen on a channel of batches of diamond descriptors.
//
// It applies some function on the received batches and returns upon completion or error.
//
// Example usage:
//
//		err := doSelectDiamonds(diamondsChan, func(diamondBatch model.DiamondDescriptors) {
//			diamonds = append(diamonds, diamondBatch...)
//		})
func doSelectDiamonds(diamondsChan <-chan diamondsEvent, do func(model.DiamondDescriptors)) error {
	// consume batches of ordered diamond metadata
	for diamondBatch := range diamondsChan {
		if diamondBatch.err != nil {
			return diamondBatch.err
		}
		do(diamondBatch.diamonds)
	}
	return nil
}

// ApplyDiamondFunc is a function to be applied on a diamond
type ApplyDiamondFunc func(model.DiamondDescriptor) error

// ListDiamondsApply applies some function to the retrieved diamonds, ordered by completion time.
//
// The execution of the applied function does not block background retrieval of more keys and diamond descriptors.
//
// Example usage: printing diamond descriptors as they come
//
//   err := core.ListDiamondsApply(repo, store, func(diamond model.DiamondDescriptor) error {
//				fmt.Fprintf(os.Stderr, "%v\n", diamond)
//				return nil
//			})
func ListDiamondsApply(repo string, stores context2.Stores, apply ApplyDiamondFunc, opts ...Option) error {
	var (
		err, applyErr error
		once          sync.Once
	)

	diamondChan := make(chan model.DiamondDescriptor)
	doneChan := make(chan struct{}, 1)

	clean := func() {
		close(doneChan)
	}
	interruptAndClean := func() {
		doneChan <- struct{}{}
		close(doneChan)
	}

	// collect diamond metadata asynchronously
	go func(diamondChan chan<- model.DiamondDescriptor, doneChan chan struct{}) {
		defer close(diamondChan)

		diamondsChan, workers := listDiamondsChan(repo, stores, append(opts, WithDoneChan(doneChan))...)

		err = doSelectDiamonds(diamondsChan, func(diamondBatch model.DiamondDescriptors) {
			for _, diamond := range diamondBatch {
				diamondChan <- diamond // transfer a batch of metadata to the applied func
			}
		})
		once.Do(clean)

		workers.Wait()
	}(diamondChan, doneChan)

	// apply function on collected metadata
	for diamond := range diamondChan {
		if applyErr = apply(diamond); applyErr != nil {
			// wind down goroutines, but when nothing is left to be interrupted
			once.Do(interruptAndClean)
			for range diamondChan {
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

// ListDiamonds yields all ongoing diamonds on a repo
func ListDiamonds(repo string, stores context2.Stores, opts ...Option) (model.DiamondDescriptors, error) {
	diamonds := make(model.DiamondDescriptors, 0, typicalDiamondsNum)

	diamondsChan, workers := listDiamondsChan(repo, stores, opts...)

	// consume batches of ordered diamonds
	err := doSelectDiamonds(diamondsChan, func(diamondBatch model.DiamondDescriptors) {
		diamonds = append(diamonds, diamondBatch...)
	})

	workers.Wait()

	return diamonds, err // we may have some batches resolved before the error occurred
}

func listDiamondsChan(repo string, stores context2.Stores, opts ...Option) (chan diamondsEvent, *sync.WaitGroup) {
	var wg sync.WaitGroup

	settings := defaultSettings()
	for _, bApply := range opts {
		bApply(&settings)
	}

	batchChan := make(chan diamondsEvent, 1) // buffered to 1 to avoid blocking on early errors

	if err := RepoExists(repo, stores); err != nil {
		batchChan <- diamondsEvent{err: err}
		close(batchChan)
		return batchChan, &wg
	}

	// internal signaling channels
	doneWithKeysChan := make(chan struct{}, 1)
	doneWithDiamondsChan := make(chan struct{}, 1)

	if settings.doneChannel != nil {
		// watch for an interruption signal requested by caller
		wg.Add(1)
		go watchForInterrupts(settings.doneChannel, &wg, doneWithKeysChan, doneWithDiamondsChan)
	}

	unfilteredKeysChan := make(chan keyBatchEvent, 1)
	keysChan := make(chan keyBatchEvent, 1)

	iterator := func(next string) ([]string, string, error) {
		return basenameKeyFilter("diamond-")(
			// restrain result to diamond descriptors (in any state)
			GetDiamondStore(stores).KeysPrefix(backgroundContexter(), next, model.GetArchivePathPrefixToDiamonds(repo), "", settings.batchSize),
		)
	}

	// starting keys retrieval
	wg.Add(1)
	go fetchKeys(iterator, unfilteredKeysChan, doneWithKeysChan, &wg) // scan for key batches

	// keys state filtering & merging
	wg.Add(1)
	go mergeKeys(unfilteredKeysChan, keysChan, settings, &wg)

	// start diamond metadata retrieval
	wg.Add(1)
	go fetchDiamonds(repo, GetDiamondStore(stores), settings, keysChan, batchChan, doneWithKeysChan, doneWithDiamondsChan, &wg)

	// let the gc clean up internal signaling channels left open after wg goroutines are done.

	// return at once. Caller may chose to wait on returned WaitGroup
	return batchChan, &wg
}

// fetchDiamonds waits on a channel of key batches and outputs batches of descriptors corresponding to these keys
func fetchDiamonds(repo string, store storage.Store, settings Settings,
	keysChan <-chan keyBatchEvent, batchChan chan<- diamondsEvent,
	doneWithKeysChan chan<- struct{}, doneChan <-chan struct{}, wg *sync.WaitGroup) {
	defer func() {
		close(batchChan)
		wg.Done()
	}()

	for {
		select {
		case <-doneChan:
			batchChan <- diamondsEvent{err: status.ErrInterrupted}
			return
		case keyBatch, isOpen := <-keysChan:
			if !isOpen {
				return
			}
			if keyBatch.err != nil {
				batchChan <- diamondsEvent{err: keyBatch.err}
				return
			}
			batch, err := fetchDiamondBatch(repo, store, settings, keyBatch.keys)
			if err != nil {
				doneWithKeysChan <- struct{}{} // stop co-worker
				batchChan <- diamondsEvent{err: err}
				return
			}
			// send out a single batch of (ordered) diamond descriptors
			batchChan <- diamondsEvent{diamonds: batch}
		}
	}
}

// fetchDiamondBatch performs a parallel fetch for a batch of diamonds identified by their keys,
// then reorders the result by key.
//
// TODO: this performs a parallel fetch for a batch of keys. However, we wait until completion of this batch to start
// a new one. In addition, for every new batch of key, we spin up a new pool of workers.
// We could improve this further by streaming batches of keys then stashing looked-ahead results and directly obtain
// a sorted output.
func fetchDiamondBatch(repo string, store storage.Store, settings Settings, keys []string) (model.DiamondDescriptors, error) {
	var (
		workers, wg sync.WaitGroup
		werr        error
	)

	diamondChan := make(chan diamondEvent)
	keyChan := make(chan string)
	doneChan := make(chan struct{}, 1)
	defer close(doneChan)

	// spin up workers pool
	for i := 0; i < minInt(settings.concurrentList, len(keys)); i++ {
		workers.Add(1)
		go getDiamondAsync(repo, store, keyChan, diamondChan, &workers)
	}

	bds := make(model.DiamondDescriptors, 0, len(keys))

	// distribute work. Stop immediately on first error reported by a worker
	wg.Add(1)
	go distributeKeys(keys)(keyChan, doneChan, &wg)

	// wait for workers to complete
	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		workers.Wait()
		close(diamondChan)
	}(&wg)

	// watch for results and coalesce
	for bd := range diamondChan {
		if bd.err != nil && werr == nil {
			werr = bd.err
			doneChan <- struct{}{} // interrupts key distribution (non-blocking)
			for range diamondChan {
			} // wait for close
			break
		}
		bds = append(bds, bd.diamond)
	}

	wg.Wait()

	if werr != nil {
		return nil, werr
	}

	// sort result batch
	sort.Sort(bds)
	return bds, nil
}

// getDiamondAsync fetches and unmarshalls the diamond descriptor for each single key submitted as input
func getDiamondAsync(repo string, store storage.Store, input <-chan string, output chan<- diamondEvent,
	wg *sync.WaitGroup) {
	defer wg.Done()

	for k := range input {
		bd, err := readDiamond(repo, k, store)
		if err != nil {
			if errors.Is(err, storagestatus.ErrNotExists) {
				continue
			}
			output <- diamondEvent{err: err}
			continue
		}

		output <- diamondEvent{diamond: bd}
	}
}

func readDiamond(repo, k string, store storage.Store) (model.DiamondDescriptor, error) {
	apc, err := model.GetArchivePathComponents(k)
	if err != nil {
		return model.DiamondDescriptor{}, err
	}

	var src string
	if apc.IsFinalState {
		src = model.GetArchivePathToFinalDiamond(repo, apc.DiamondID)
	} else {
		src = model.GetArchivePathToInitialDiamond(repo, apc.DiamondID)
	}

	r, err := store.Get(backgroundContexter(), src)
	if err != nil {
		return model.DiamondDescriptor{}, err
	}

	o, err := ioutil.ReadAll(r)
	if err != nil {
		return model.DiamondDescriptor{}, err
	}

	var sd model.DiamondDescriptor
	err = yaml.Unmarshal(o, &sd)
	if err != nil {
		return model.DiamondDescriptor{}, err
	}

	if sd.DiamondID != apc.DiamondID {
		err = fmt.Errorf("diamond IDs in descriptor '%v' and archive path '%v' don't match", sd.DiamondID, apc.DiamondID)
		return model.DiamondDescriptor{}, err
	}

	return sd, nil
}
