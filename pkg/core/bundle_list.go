package core

import (
	"context"
	"fmt"
	"io/ioutil"
	"sort"
	"sync"

	"errors"

	"gopkg.in/yaml.v2"

	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/oneconcern/datamon/pkg/storage/status"
)

const (
	maxMetaFilesToProcess = 1000000
	typicalBundlesNum     = 1000 // default number of allocated slots for bundles in a repo
)

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// BundleEvent catches a single bundle with possible retrieval error
type BundleEvent = func() (model.BundleDescriptor, error)

func bundleEvent(b model.BundleDescriptor, e error) BundleEvent {
	return func() (model.BundleDescriptor, error) {
		return b, e
	}
}

func bundleEventError(e error) BundleEvent {
	return func() (model.BundleDescriptor, error) {
		return model.BundleDescriptor{}, e
	}
}

// BundlesEvent catches a collection of bundles with possible retrieval error
type BundlesEvent = func() (model.BundleDescriptors, error)

func bundlesEvent(bs model.BundleDescriptors, e error) BundlesEvent {
	return func() (model.BundleDescriptors, error) {
		return bs, e
	}
}

// BundleKeyBatchEvent catches a collection of bundle keys with possible retrieval error
type BundleKeyBatchEvent = func() ([]string, error)

func bundleKeyBatchEvent(keys []string, e error) BundleKeyBatchEvent {
	return func() ([]string, error) {
		return keys, e
	}
}

// ErrInterrupted signals that the current background processing has been interrupted
var ErrInterrupted = errors.New("background processing interrupted")

// DoSelectBundles is a helper function to listen on a channel of batches of bundle descriptors.
//
// It applies some function on the received batches and returns upon completion or error.
//
// Example usage:
//
//		err := DoSelectBundles(bundlesChan, func(bundleBatch model.BundleDescriptors) {
//			bundles = append(bundles, bundleBatch...)
//		})
func DoSelectBundles(bundlesChan <-chan BundlesEvent, do func(model.BundleDescriptors)) error {
	// consume batches of ordered bundle metadata
	for bundleBatch := range bundlesChan {
		bundles, err := bundleBatch()
		if err != nil {
			return err
		}
		do(bundles)
	}
	return nil
}

// ApplyBundleFunc is a function to be applied on a bundle
type ApplyBundleFunc func(model.BundleDescriptor) error

// ListBundlesApply applies some function to the retrieved bundles, in lexicographic order of keys.
//
// The execution of the applied function does not block background retrieval of more keys and bundle descriptors.
//
// Example usage: printing bundle descriptors as they come
//
//   err := core.ListBundlesApply(repo, store, func(bundle model.BundleDescriptorOption) error {
//				fmt.Fprintf(os.Stderr, "%v\n", bundle)
//				return nil
//			})
func ListBundlesApply(repo string, store storage.Store, apply ApplyBundleFunc, opts ...BundleListOption) error {
	var (
		err, applyErr error
		wg            sync.WaitGroup
		workDone      bool
		mutex         sync.Mutex
	)

	bundleChan := make(chan model.BundleDescriptor)
	doneChan := make(chan struct{}, 1)

	// collect bundle metadata asynchronously
	wg.Add(1)
	go func(bundleChan chan<- model.BundleDescriptor, doneChan chan struct{}, wg *sync.WaitGroup) {
		defer func() {
			close(bundleChan)
			wg.Done()
		}()

		bundlesChan, workers := ListBundlesChan(repo, store, append(opts, WithBundleDoneChan(doneChan))...)

		err = DoSelectBundles(bundlesChan, func(bundleBatch model.BundleDescriptors) {
			for _, bundle := range bundleBatch {
				bundleChan <- bundle // transfer a batch of metadata to the applied func
			}
		})

		mutex.Lock()
		workDone = true // nothing left to interrupt
		close(doneChan)
		mutex.Unlock()

		workers.Wait()
	}(bundleChan, doneChan, &wg)

	// apply function on collected metadata
	for bundle := range bundleChan {
		if applyErr = apply(bundle); applyErr != nil {
			// wind down goroutines, but when nothing is left to be interrupted
			mutex.Lock()
			if !workDone {
				doneChan <- struct{}{}
			}
			mutex.Unlock()
			for range bundleChan {
			} // wait for close
			break
		}
	}
	wg.Wait()
	// collect errors
	switch {
	case err == ErrInterrupted && applyErr != nil:
		return applyErr
	case err != nil:
		return err
	case applyErr != nil:
		return applyErr
	default:
		return nil
	}
}

// ListBundles returns a list of bundle descriptors from a repo. It collects all bundles until completion.
func ListBundles(repo string, store storage.Store, opts ...BundleListOption) (model.BundleDescriptors, error) {
	bundles := make(model.BundleDescriptors, 0, typicalBundlesNum)

	bundlesChan, workers := ListBundlesChan(repo, store, opts...)

	// consume batches of ordered bundles
	err := DoSelectBundles(bundlesChan, func(bundleBatch model.BundleDescriptors) {
		bundles = append(bundles, bundleBatch...)
	})

	workers.Wait()

	return bundles, err // we may have some batches resolved before the error occurred
}

// ListBundlesChan returns a list of bundle descriptors from a repo. Each batch of returned descriptors
// is sent on the output channel, following key lexicographic order.
//
// Simple use cases of this helper are wrapped in ListBundles (block until completion) and ListBundlesApply
// (apply function while retrieving metadata).
//
// A signaling channel may be given as option to interrupt background processing (e.g. on error).
//
// The sync.WaitGroup for internal goroutines is returned if caller wants to wait and avoid any leaked goroutines.
func ListBundlesChan(repo string, store storage.Store, opts ...BundleListOption) (chan BundlesEvent, *sync.WaitGroup) {
	var (
		wg, wg2 sync.WaitGroup
	)

	settings := defaultCoreSettings()
	for _, bApply := range opts {
		bApply(&settings)
	}

	batchChan := make(chan BundlesEvent, 1) // buffered to 1 to avoid blocking on early errors

	if err := RepoExists(repo, store); err != nil {
		batchChan <- bundlesEvent(nil, err)
		close(batchChan)
		return batchChan, &wg
	}

	// internal signaling channels
	doneWithKeysChan := make(chan struct{}, 1)
	doneWithBundlesChan := make(chan struct{}, 1)

	if settings.doneChannel != nil {
		// watch for an interruption signal requested by caller
		wg.Add(1)
		go watchForInterrupts(settings.doneChannel, &wg, doneWithKeysChan, doneWithBundlesChan)
	}

	keysChan := make(chan BundleKeyBatchEvent, 1)

	// starting keys retrieval
	wg.Add(1)
	go fetchKeys(repo, store, settings, keysChan, doneWithKeysChan, &wg) // scan for key batches

	// start bundle metadata retrieval
	wg.Add(1)
	go fetchBundles(repo, store, settings, keysChan, batchChan, doneWithKeysChan, doneWithBundlesChan, &wg)

	// cleanup internal signaling channels
	wg2.Add(1)
	go func(wg, wg2 *sync.WaitGroup, inputChans ...chan<- struct{}) {
		defer wg2.Done()
		wg.Wait()
		for _, ch := range inputChans {
			close(ch)
		}
	}(&wg, &wg2, doneWithKeysChan, doneWithBundlesChan)

	// return at once. Caller may chose to wait on returned WaitGroup
	return batchChan, &wg2
}

// watchForInterrupts broadcasts a done signal to several output channels
func watchForInterrupts(doneChan <-chan struct{}, wg *sync.WaitGroup, outputChans ...chan<- struct{}) {
	defer func() {
		wg.Done()
	}()

	if _, interrupt := <-doneChan; interrupt {
		for _, outputChan := range outputChans {
			outputChan <- struct{}{}
		}
	}
}

// fetchBundles waits on a channel of key batches and outputs batches of descriptors corresponding to these keys
func fetchBundles(repo string, store storage.Store, settings CoreSettings,
	keysChan <-chan BundleKeyBatchEvent, batchChan chan<- BundlesEvent,
	doneWithKeysChan chan<- struct{}, doneChan <-chan struct{}, wg *sync.WaitGroup) {
	defer func() {
		close(batchChan)
		wg.Done()
	}()

	for {
		select {
		case <-doneChan:
			batchChan <- bundlesEvent(nil, ErrInterrupted)
			return
		case keyBatch, isOpen := <-keysChan:
			if !isOpen {
				return
			}
			keys, err := keyBatch()
			if err != nil {
				batchChan <- bundlesEvent(nil, err)
				return
			}
			batch, err := fetchBundleBatch(repo, store, settings, keys)
			if err != nil {
				doneWithKeysChan <- struct{}{} // stop co-worker
				batchChan <- bundlesEvent(nil, err)
				return
			}
			// send out a single batch of (ordered) bundle descriptors
			batchChan <- bundlesEvent(batch, nil)
		}
	}
}

// fetchKeys fetches keys for bundles in batches, then close the keyBatchChan channel upon completion or error.
func fetchKeys(repo string, store storage.Store, settings CoreSettings,
	keyBatchChan chan<- BundleKeyBatchEvent,
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
		ks, next, err = store.KeysPrefix(context.Background(), next, model.GetArchivePathPrefixToBundles(repo), "/", settings.bundleBatchSize)
		if err != nil {
			select {
			case keyBatchChan <- bundleKeyBatchEvent(nil, err):
			case <-doneChan:
				select {
				case keyBatchChan <- bundleKeyBatchEvent(nil, ErrInterrupted):
				default:
				}
			}
			return
		}

		if len(ks) == 0 {
			break
		}

		select {
		case keyBatchChan <- bundleKeyBatchEvent(ks, nil):
		case <-doneChan:
			select {
			case keyBatchChan <- bundleKeyBatchEvent(nil, ErrInterrupted):
			default:
			}
			return
		}

		if next == "" {
			break
		}
	}
}

// fetchBundleBatch performs a parallel fetch for a batch of bundles identified by their keys, then reorders the result by key
func fetchBundleBatch(repo string, store storage.Store, settings CoreSettings, keys []string) (model.BundleDescriptors, error) {
	var (
		workers, wg sync.WaitGroup
		werr        error
	)

	bundleChan := make(chan BundleEvent)
	keyChan := make(chan string)
	doneChan := make(chan struct{}, 1)
	defer close(doneChan)

	// spin up workers pool
	for i := 0; i < minInt(settings.concurrentBundleList, len(keys)); i++ {
		workers.Add(1)
		go getBundleAsync(repo, store, keyChan, bundleChan, &workers)
	}

	bds := make(model.BundleDescriptors, 0, len(keys))

	// distribute work. Stop immediately on first error reported by a worker
	wg.Add(1)
	go func(keyChan chan<- string, doneChan <-chan struct{}, wg *sync.WaitGroup) {
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
	}(keyChan, doneChan, &wg)

	// wait for workers to complete
	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		workers.Wait()
		close(bundleChan)
	}(&wg)

	// watch for results and coalesce
	for bd := range bundleChan {
		bundle, err := bd()
		if err != nil && werr == nil {
			werr = err
			doneChan <- struct{}{} // interrupts key distribution (non-blocking)
			for range bundleChan {
			} // wait for close
			break
		}
		bds = append(bds, bundle)
	}

	wg.Wait()

	if werr != nil {
		return nil, werr
	}

	// sort result batch
	sort.Sort(bds)
	return bds, nil
}

// getBundleAsync fetches and unmarshalls the bundle descriptor for each single key submitted as input
func getBundleAsync(repo string, store storage.Store,
	input <-chan string, output chan<- BundleEvent,
	wg *sync.WaitGroup) {

	// fetch descriptors
	defer wg.Done()
	for k := range input {
		apc, err := model.GetArchivePathComponents(k)
		if err != nil {
			output <- bundleEventError(err)
			continue
		}
		r, err := store.Get(context.Background(), model.GetArchivePathToBundle(repo, apc.BundleID))
		if err != nil {
			if errors.Is(err, status.ErrNotExists) {
				continue
			}
			output <- bundleEventError(err)
			continue
		}
		o, err := ioutil.ReadAll(r)
		if err != nil {
			output <- bundleEventError(err)
			continue
		}
		var bd model.BundleDescriptor
		err = yaml.Unmarshal(o, &bd)
		if err != nil {
			output <- bundleEventError(err)
			continue
		}
		if bd.ID != apc.BundleID {
			err = fmt.Errorf("bundle IDs in descriptor '%v' and archive path '%v' don't match", bd.ID, apc.BundleID)
			output <- bundleEventError(err)
			continue
		}

		output <- bundleEvent(bd, nil)
	}
}

// GetLatestBundle returns the latest bundle descriptor from a repo
func GetLatestBundle(repo string, store storage.Store) (string, error) {
	e := RepoExists(repo, store)
	if e != nil {
		return "", e
	}
	ks, _, err := store.KeysPrefix(context.Background(), "", model.GetArchivePathPrefixToBundles(repo), "", maxMetaFilesToProcess)
	if err != nil {
		return "", err
	}
	if len(ks) == 0 {
		return "", fmt.Errorf("no bundles uploaded to repo: %s", repo)
	}

	apc, err := model.GetArchivePathComponents(ks[len(ks)-1])
	if err != nil {
		return "", err
	}

	return apc.BundleID, nil
}
