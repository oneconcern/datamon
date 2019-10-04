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

// ErrInterrupted signals that the current background processing is interrupted
var ErrInterrupted = errors.New("background processing interrupted")

// DoSelectBundles is a helper function to listen on a channel of batches of bundle descriptors and an error channel.
//
// It applies some function on the received batches and returns an error if one appears on the error channel.
//
// Example usage:
//
//		err := DoSelectBundles(bundlesChan, errChan, func(bundleBatch model.BundleDescriptors) {
//			bundles = append(bundles, bundleBatch...)
//		})
func DoSelectBundles(bundlesChan <-chan model.BundleDescriptors, errorChan <-chan error, do func(model.BundleDescriptors)) error {
	var err error

LOOP:
	// consume batches of ordered bundle metadata
	for {
		select {
		case bundleBatch, ok := <-bundlesChan:
			if !ok {
				// last attempt to catch an error
				// (i.e. bundlesChan has closed, but an error was also present on errorChan and hasn't been selected)
				if err == nil {
					if e, ok := <-errorChan; ok {
						err = e
					}
				}
				break LOOP
			}
			do(bundleBatch)
		case e, ok := <-errorChan: // do not exit here: the only natural exit occurs when bundlesChan is closed
			if ok && err == nil { // retain first error
				err = e
			}
		}
	}

	return err
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

		bundlesChan, errChan, workers := ListBundlesChan(repo, store, append(opts, WithBundleDoneChan(doneChan))...)

		err = DoSelectBundles(bundlesChan, errChan, func(bundleBatch model.BundleDescriptors) {
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
			// wind down goroutines, but if interrupt channel is already closed (i.e nothing left to be interrupted)
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

	bundlesChan, errChan, workers := ListBundlesChan(repo, store, opts...)

	// consume batches of ordered bundles
	err := DoSelectBundles(bundlesChan, errChan, func(bundleBatch model.BundleDescriptors) {
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
// An optional signaling channel may be given as option to interrupt background processing (e.g. on error).
//
// The sync.WaitGroup for internal goroutines is returned if caller wants to wait and avoid any leaked goroutines.
func ListBundlesChan(repo string, store storage.Store, opts ...BundleListOption) (chan model.BundleDescriptors, chan error, *sync.WaitGroup) {
	var (
		wg, wg2 sync.WaitGroup
	)

	settings := defaultCoreSettings()
	for _, bApply := range opts {
		bApply(&settings)
	}

	batchChan := make(chan model.BundleDescriptors)
	errorChan := make(chan error, 1) // buffering early errors avoids blocking without dropping the message

	if err := RepoExists(repo, store); err != nil {
		errorChan <- err
		close(batchChan)
		close(errorChan)
		return batchChan, errorChan, &wg
	}

	// signaling channels
	doneWithKeysChan := make(chan struct{}, 1)
	doneWithBundlesChan := make(chan struct{}, 1)

	if settings.doneChannel != nil {
		// watch for an interruption signal requested by caller
		wg.Add(1)
		go watchForInterrupts(settings.doneChannel, &wg, doneWithKeysChan, doneWithBundlesChan)
	}

	keysChan := make(chan []string)
	keyErrorChan := make(chan error)
	bundleErrorChan := make(chan error)

	// listening to errors from slave goroutines
	wg.Add(1)
	go watchForErrors(keyErrorChan, bundleErrorChan, errorChan, doneWithKeysChan, doneWithBundlesChan, &wg)

	// starting keys retrieval
	wg.Add(1)
	go fetchKeys(repo, store, settings, keysChan, keyErrorChan, doneWithKeysChan, &wg) // scan for key batches

	// start bundle metadata retrieval
	wg.Add(1)
	go fetchBundles(repo, store, settings, keysChan, batchChan, bundleErrorChan, doneWithBundlesChan, &wg)

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
	return batchChan, errorChan, &wg2
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

// watchForErrors listens on 2 independent errors channels and report back to a main error channel.
//
// NOTE: this one does not need signaling, since any interruption bubbles up as error.
func watchForErrors(
	errorChan1, errorChan2 <-chan error,
	outputChan chan<- error,
	doneChan1, doneChan2 chan<- struct{}, wg *sync.WaitGroup) {
	defer func() {
		close(outputChan)
		wg.Done()
	}()

	var (
		err                            error
		errorChan1Open, errorChan2Open bool = true, true
	)
	for errorChan1Open || errorChan2Open { // wait on both channels to close
		select {
		case err, errorChan1Open = <-errorChan1:
			if errorChan1Open {
				outputChan <- err
				doneChan2 <- struct{}{}
			} else {
				errorChan1 = nil // no more selected case on this one
				break
			}
		case err, errorChan2Open = <-errorChan2:
			if errorChan2Open {
				doneChan1 <- struct{}{}
				outputChan <- err
			} else {
				errorChan2 = nil // no more selected case on this one
				break
			}
		}
	}
}

// fetchBundles waits on a channel of key batches and outputs batches of descriptors corresponding to these keys
func fetchBundles(repo string, store storage.Store, settings CoreSettings,
	keysChan <-chan []string,
	batchChan chan<- model.BundleDescriptors, errorChan chan<- error,
	doneChan <-chan struct{}, wg *sync.WaitGroup) {
	defer func() {
		close(batchChan)
		close(errorChan)
		wg.Done()
	}()

	for {
		select {
		case <-doneChan:
			errorChan <- ErrInterrupted
			return
		case keyBatch, ok := <-keysChan:
			if !ok {
				return
			}
			batch, err := fetchBundleBatch(repo, store, settings, keyBatch)
			if err != nil {
				errorChan <- err
				return
			}
			// send out a single batch of (ordered) bundle descriptors
			batchChan <- batch
		}
	}
}

// fetchKeys fetches keys for bundles in batches, then close the keyBatchChan channel upon completion or error.
func fetchKeys(repo string, store storage.Store, settings CoreSettings,
	keyBatchChan chan<- []string, errorChan chan<- error,
	doneChan <-chan struct{}, wg *sync.WaitGroup) {
	defer func() {
		close(keyBatchChan)
		close(errorChan)
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
			case errorChan <- err:
				break
			case <-doneChan:
				errorChan <- ErrInterrupted
				break
			}
			return
		}

		if len(ks) == 0 {
			break
		}

		select {
		case keyBatchChan <- ks:
			break
		case <-doneChan:
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

	bundleChan := make(chan model.BundleDescriptor)
	keyChan := make(chan string)
	errorChan := make(chan error)

	// spin workers pool
	for i := 0; i < minInt(settings.concurrentBundleList, len(keys)); i++ {
		workers.Add(1)
		go getBundleAsync(repo, store, keyChan, bundleChan, errorChan, &workers)
	}

	bds := make(model.BundleDescriptors, 0, settings.bundleBatchSize)

	// watch for results and coalesce
	wg.Add(1)
	go func(bundleChan <-chan model.BundleDescriptor, wg *sync.WaitGroup) {
		defer wg.Done()
		for bd := range bundleChan {
			bds = append(bds, bd)
		}
	}(bundleChan, &wg)

	// watch for errors
	wg.Add(1)
	go func(errorChan <-chan error, wg *sync.WaitGroup) {
		defer wg.Done()
		for err := range errorChan {
			werr = err
		}
	}(errorChan, &wg)

	// distribute work
	for _, k := range keys {
		keyChan <- k
	}

	close(keyChan)

	// wait for workers to complete
	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		workers.Wait()
		close(bundleChan)
		close(errorChan)
	}(&wg)

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
	input <-chan string,
	output chan<- model.BundleDescriptor,
	errorChan chan<- error, wg *sync.WaitGroup) {
	// fetch descriptors
	defer wg.Done()
	for k := range input {
		apc, err := model.GetArchivePathComponents(k)
		if err != nil {
			errorChan <- err
			continue
		}
		r, err := store.Get(context.Background(), model.GetArchivePathToBundle(repo, apc.BundleID))
		if err != nil {
			if errors.Is(err, status.ErrNotExists) {
				continue
			}
			errorChan <- err
			continue
		}
		o, err := ioutil.ReadAll(r)
		if err != nil {
			errorChan <- err
			continue
		}
		var bd model.BundleDescriptor
		err = yaml.Unmarshal(o, &bd)
		if err != nil {
			errorChan <- err
			continue
		}
		if bd.ID != apc.BundleID {
			err = fmt.Errorf("bundle IDs in descriptor '%v' and archive path '%v' don't match", bd.ID, apc.BundleID)
			errorChan <- err
			continue
		}

		output <- bd
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
