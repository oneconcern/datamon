package core

import (
	"context"
	"fmt"
	"io/ioutil"
	"sort"
	"sync"

	context2 "github.com/oneconcern/datamon/pkg/context"

	"github.com/oneconcern/datamon/pkg/errors"

	"gopkg.in/yaml.v2"

	"github.com/oneconcern/datamon/pkg/core/status"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	storagestatus "github.com/oneconcern/datamon/pkg/storage/status"
)

const (
	maxMetaFilesToProcess = 1000000
	typicalBundlesNum     = 1000 // default number of allocated slots for bundles in a repo
)

// bundleEvent catches a single bundle with possible retrieval error
type bundleEvent struct {
	bundle model.BundleDescriptor
	err    error
}

// bundlesEvent catches a collection of bundles with possible retrieval error
type bundlesEvent struct {
	bundles model.BundleDescriptors
	err     error
}

// doSelectBundles is a helper function to listen on a channel of batches of bundle descriptors.
//
// It applies some function on the received batches and returns upon completion or error.
//
// Example usage:
//
//	err := doSelectBundles(bundlesChan, func(bundleBatch model.BundleDescriptors) {
//		bundles = append(bundles, bundleBatch...)
//	})
func doSelectBundles(bundlesChan <-chan bundlesEvent, do func(model.BundleDescriptors)) error {
	// consume batches of ordered bundle metadata
	for bundleBatch := range bundlesChan {
		if bundleBatch.err != nil {
			return bundleBatch.err
		}
		do(bundleBatch.bundles)
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
//	  err := core.ListBundlesApply(repo, store, func(bundle model.BundleDescriptor) error {
//					fmt.Fprintf(os.Stderr, "%v\n", bundle)
//					return nil
//				})
func ListBundlesApply(repo string, stores context2.Stores, apply ApplyBundleFunc, opts ...Option) error {
	var (
		err, applyErr error
		once          sync.Once
	)

	bundleChan := make(chan model.BundleDescriptor)
	doneChan := make(chan struct{}, 1)

	clean := func() {
		close(doneChan)
	}
	interruptAndClean := func() {
		doneChan <- struct{}{}
		close(doneChan)
	}

	// collect bundle metadata asynchronously
	go func(bundleChan chan<- model.BundleDescriptor, doneChan chan struct{}) {
		defer close(bundleChan)

		bundlesChan, workers := listBundlesChan(repo, stores, append(opts, WithDoneChan(doneChan))...)

		err = doSelectBundles(bundlesChan, func(bundleBatch model.BundleDescriptors) {
			for _, bundle := range bundleBatch {
				bundleChan <- bundle // transfer a batch of metadata to the applied func
			}
		})
		once.Do(clean)

		workers.Wait()
	}(bundleChan, doneChan)

	// apply function on collected metadata
	for bundle := range bundleChan {
		if applyErr = apply(bundle); applyErr != nil {
			// wind down goroutines, but when nothing is left to be interrupted
			once.Do(interruptAndClean)
			for range bundleChan {
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

// ListBundles returns a list of bundle descriptors from a repo. It collects all bundles until completion.
//
// NOTE: this func could become deprecated. At this moment, however, it is used by pkg/web.
func ListBundles(repo string, stores context2.Stores, opts ...Option) (model.BundleDescriptors, error) {
	bundles := make(model.BundleDescriptors, 0, typicalBundlesNum)

	bundlesChan, workers := listBundlesChan(repo, stores, opts...)

	// consume batches of ordered bundles
	err := doSelectBundles(bundlesChan, func(bundleBatch model.BundleDescriptors) {
		bundles = append(bundles, bundleBatch...)
	})

	workers.Wait()

	return bundles, err // we may have some batches resolved before the error occurred
}

// listBundlesChan returns a list of bundle descriptors from a repo. Each batch of returned descriptors
// is sent on the output channel, following key lexicographic order.
//
// Simple use cases of this helper are wrapped in ListBundles (block until completion) and ListBundlesApply
// (apply function while retrieving metadata).
//
// A signaling channel may be given as option to interrupt background processing (e.g. on error).
//
// The sync.WaitGroup for internal goroutines is returned if caller wants to wait and avoid any leaked goroutines.
func listBundlesChan(repo string, stores context2.Stores, opts ...Option) (chan bundlesEvent, *sync.WaitGroup) {
	var wg sync.WaitGroup

	settings := defaultSettings()
	for _, bApply := range opts {
		bApply(&settings)
	}

	batchChan := make(chan bundlesEvent, 1) // buffered to 1 to avoid blocking on early errors

	if err := RepoExists(repo, stores); err != nil {
		batchChan <- bundlesEvent{err: err}
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

	keysChan := make(chan keyBatchEvent, 1)

	iterator := func(next string) ([]string, string, error) {
		return GetBundleStore(stores).KeysPrefix(context.Background(), next, model.GetArchivePathPrefixToBundles(repo), "/", settings.batchSize)
	}
	// starting keys retrieval
	wg.Add(1)
	go fetchKeys(iterator, keysChan, doneWithKeysChan, &wg) // scan for key batches

	// start bundle metadata retrieval
	wg.Add(1)
	go fetchBundles(repo, getMetaStore(stores), settings, keysChan, batchChan, doneWithKeysChan, doneWithBundlesChan, &wg)

	// let the gc clean up internal signaling channels left open after wg goroutines are done.

	// return at once. Caller may chose to wait on returned WaitGroup
	return batchChan, &wg
}

// fetchBundles waits on a channel of key batches and outputs batches of descriptors corresponding to these keys
func fetchBundles(repo string, store storage.Store, settings Settings,
	keysChan <-chan keyBatchEvent, batchChan chan<- bundlesEvent,
	doneWithKeysChan chan<- struct{}, doneChan <-chan struct{}, wg *sync.WaitGroup) {
	defer func() {
		close(batchChan)
		wg.Done()
	}()

	for {
		select {
		case <-doneChan:
			batchChan <- bundlesEvent{err: status.ErrInterrupted}
			return
		case keyBatch, isOpen := <-keysChan:
			if !isOpen {
				return
			}
			if keyBatch.err != nil {
				batchChan <- bundlesEvent{err: keyBatch.err}
				return
			}
			batch, err := fetchBundleBatch(repo, store, settings, keyBatch.keys)
			if err != nil {
				doneWithKeysChan <- struct{}{} // stop co-worker
				batchChan <- bundlesEvent{err: err}
				return
			}
			// send out a single batch of (ordered) bundle descriptors
			batchChan <- bundlesEvent{bundles: batch}
		}
	}
}

// fetchBundleBatch performs a parallel fetch for a batch of bundles identified by their keys,
// then reorders the result by key.
//
// TODO: this performs a parallel fetch for a batch of keys. However, we wait until completion of this batch to start
// a new one. In addition, for every new batch of key, we spin up a new pool of workers.
// We could improve this further by streaming batches of keys then stashing looked-ahead results and directly obtain
// a sorted output.
func fetchBundleBatch(repo string, store storage.Store, settings Settings, keys []string) (model.BundleDescriptors, error) {
	var (
		workers, wg sync.WaitGroup
		werr        error
	)

	bundleChan := make(chan bundleEvent)
	keyChan := make(chan string)
	doneChan := make(chan struct{}, 1)
	defer close(doneChan)

	// spin up workers pool
	for i := 0; i < minInt(settings.concurrentList, len(keys)); i++ {
		workers.Add(1)
		go getBundleAsync(repo, store, keyChan, bundleChan, &workers, settings)
	}

	bds := make(model.BundleDescriptors, 0, len(keys))

	// distribute work. Stop immediately on first error reported by a worker
	wg.Add(1)
	go distributeKeys(keys)(keyChan, doneChan, &wg)

	// wait for workers to complete
	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		workers.Wait()
		close(bundleChan)
	}(&wg)

	// watch for results and coalesce
	for bd := range bundleChan {
		if bd.err != nil && werr == nil {
			werr = bd.err
			doneChan <- struct{}{} // interrupts key distribution (non-blocking)
			for range bundleChan {
			} // wait for close
			break
		}
		bds = append(bds, bd.bundle)
	}

	wg.Wait()

	if werr != nil {
		return nil, werr
	}

	// sort result batch
	sort.Sort(bds)
	return bds, nil
}

func downloadBundleDescriptor(store storage.Store, repo, key string, settings Settings) (model.BundleDescriptor, error) {
	apc, err := model.GetArchivePathComponents(key)
	if err != nil {
		return model.BundleDescriptor{}, err
	}

	if settings.withMinimalBundle {
		// in this configuration, don't fetch the bundle descriptor: we are only interested about the key
		return model.BundleDescriptor{
			ID: apc.BundleID,
		}, nil
	}

	r, err := store.Get(context.Background(), model.GetArchivePathToBundle(repo, apc.BundleID))
	if err != nil {
		return model.BundleDescriptor{}, err
	}

	o, err := ioutil.ReadAll(r)
	if err != nil {
		return model.BundleDescriptor{}, err
	}

	var bd model.BundleDescriptor
	err = yaml.Unmarshal(o, &bd)
	if err != nil {
		return model.BundleDescriptor{}, err
	}

	if bd.ID != apc.BundleID {
		if !settings.ignoreCorruptedMetadata {
			err = fmt.Errorf("bundle IDs in descriptor '%v' and archive path '%v' don't match", bd.ID, apc.BundleID)

			return model.BundleDescriptor{}, err
		} else {
			bd.ID = apc.BundleID // other metadata might be missing (case of a corrupted bundle.yaml file, should not be blocking)
		}
	}

	return bd, nil
}

// getBundleAsync fetches and unmarshalls the bundle descriptor for each single key submitted as input
func getBundleAsync(repo string, store storage.Store, input <-chan string, output chan<- bundleEvent,
	wg *sync.WaitGroup,
	settings Settings,
) {
	defer wg.Done()
	for k := range input {
		bd, err := downloadBundleDescriptor(store, repo, k, settings)

		if err != nil {
			if errors.Is(err, storagestatus.ErrNotExists) {
				continue
			}
			output <- bundleEvent{err: err}
			continue
		}

		output <- bundleEvent{bundle: bd}
	}
}

// GetLatestBundle returns the latest bundle descriptor from a repo
func GetLatestBundle(repo string, stores context2.Stores) (string, error) {
	e := RepoExists(repo, stores)
	if e != nil {
		return "", e
	}
	ks, _, err := getMetaStore(stores).KeysPrefix(context.Background(), "", model.GetArchivePathPrefixToBundles(repo), "", maxMetaFilesToProcess)
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
