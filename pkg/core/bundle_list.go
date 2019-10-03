package core

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"

	"gopkg.in/yaml.v2"

	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
)

const (
	maxMetaFilesToProcess = 1000000
	typicalBundlesNum     = 1000
)

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ListBundles returns the list of bundle descriptors from a repo.
//
// NOTE: resulting bundles are returned in no particular order.
//
// TODO: return a paginated list of id<->bd (separate function in repo_list.go)
func ListBundles(repo string, store storage.Store, opts ...BundleListOption) ([]model.BundleDescriptor, error) {
	settings := CoreSettings{concurrentBundleList: defaultBundleListConcurrency}
	for _, bApply := range opts {
		bApply(&settings)
	}

	// Get a list
	e := RepoExists(repo, store)
	if e != nil {
		return nil, e
	}

	// this call is not made async yet
	ks, _, err := store.KeysPrefix(context.Background(), "", model.GetArchivePathPrefixToBundles(repo), "/", maxMetaFilesToProcess)
	if err != nil {
		return nil, err
	}

	var (
		workers, wg sync.WaitGroup
		werr        error
	)

	bundleChan := make(chan model.BundleDescriptor)
	keyChan := make(chan string)
	errorChan := make(chan error)

	for i := 0; i < minInt(settings.concurrentBundleList, len(ks)); i++ {
		workers.Add(1)
		go getBundleAsync(repo, store, keyChan, bundleChan, errorChan, &workers)
	}

	bds := make([]model.BundleDescriptor, 0, typicalBundlesNum) // preallocate some typical number of bundles, e.g. 1000

	wg.Add(1)
	go func(bundleChan <-chan model.BundleDescriptor, wg *sync.WaitGroup) {
		defer wg.Done()
		for bd := range bundleChan { // watch for results and coalesce
			bds = append(bds, bd)
		}
	}(bundleChan, &wg)

	wg.Add(1)
	go func(errorChan <-chan error, wg *sync.WaitGroup) { // watch for errors
		defer wg.Done()
		for err := range errorChan {
			werr = err
		}
	}(errorChan, &wg)

	for _, k := range ks { // distribute work
		keyChan <- k
	}

	close(keyChan)

	wg.Add(1)
	go func(wg *sync.WaitGroup) { // wait for workers to complete
		defer wg.Done()
		workers.Wait()
		close(bundleChan)
		close(errorChan)
	}(&wg)

	wg.Wait()

	if werr != nil {
		return nil, werr
	}

	return bds, nil
}

func getBundleAsync(repo string, store storage.Store,
	input <-chan string,
	output chan<- model.BundleDescriptor,
	errorChan chan<- error,
	wg *sync.WaitGroup) {

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
			// TODO: should be an error type (this creates a dependency on the actual implementation of the interface)
			if strings.Contains(err.Error(), "object doesn't exist") {
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
	ks, _, err := store.KeysPrefix(context.Background(), "", model.GetArchivePathPrefixToBundles(repo), "", 1000000)
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
