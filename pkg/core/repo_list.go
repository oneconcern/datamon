package core

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"sort"
	"sync"

	"github.com/oneconcern/datamon/pkg/core/status"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	storagestatus "github.com/oneconcern/datamon/pkg/storage/status"
	yaml "gopkg.in/yaml.v2"
)

const (
	maxReposToList  = 1000000
	typicalReposNum = 1000 // default number of allocated slots for repos
)

// GetRepoDescriptorByRepoName returns the descriptor of a named repo
func GetRepoDescriptorByRepoName(store storage.Store, repoName string) (model.RepoDescriptor, error) {
	return getRepoDescriptorByRepoName(store, repoName)
}

func getRepoDescriptorByRepoName(store storage.Store, repoName string) (model.RepoDescriptor, error) {
	var rd model.RepoDescriptor
	archivePathToRepoDescriptor := model.GetArchivePathToRepoDescriptor(repoName)
	has, err := store.Has(context.Background(), archivePathToRepoDescriptor)
	if err != nil {
		return rd, err
	}
	if !has {
		return rd, status.ErrNotFound
	}
	r, err := store.Get(context.Background(), archivePathToRepoDescriptor)
	if err != nil {
		return rd, err
	}
	o, err := ioutil.ReadAll(r)
	if err != nil {
		return rd, err
	}
	err = yaml.Unmarshal(o, &rd)
	if err != nil {
		return rd, err
	}
	if rd.Name != repoName {
		return rd, fmt.Errorf("repo names in descriptor '%v' and archive path '%v' don't match",
			rd.Name, repoName)
	}
	return rd, nil
}

// ListRepos returns all repos from a store
func ListRepos(store storage.Store, opts ...ListOption) ([]model.RepoDescriptor, error) {
	repos := make(model.RepoDescriptors, 0, typicalReposNum)

	reposChan, workers := listReposChan(store, opts...)

	// consume batches of ordered repos
	err := doSelectRepos(reposChan, func(repoBatch model.RepoDescriptors) {
		repos = append(repos, repoBatch...)
	})

	workers.Wait()

	return repos, err // we may have some batches resolved before the error occurred
}

// ListReposPaginated is at this moment only used by the CSI package.
// Question: shall we deprecate this?
func ListReposPaginated(store storage.Store, token string) ([]model.RepoDescriptor, error) {
	// Get a list
	ks, _, _ := store.KeysPrefix(context.Background(), "", model.GetArchivePathPrefixToRepos(), "", maxReposToList)
	var repos = make([]model.RepoDescriptor, 0)
	tokenHit := false
	for _, k := range ks {
		apc, err := model.GetArchivePathComponents(k)
		if err != nil {
			return nil, err
		}
		if apc.Repo == token {
			tokenHit = true
		}
		if !tokenHit {
			continue
		}

		var rd model.RepoDescriptor
		rd, err = GetRepoDescriptorByRepoName(store, apc.Repo)
		if err != nil {
			return nil, err
		}
		repos = append(repos, rd)
	}
	return repos, nil
}

// ApplyRepoFunc is a function to be applied on a repo
type ApplyRepoFunc func(model.RepoDescriptor) error

// ListReposApply applies some function to the retrieved repos, in lexicographic order of keys.
func ListReposApply(store storage.Store, apply ApplyRepoFunc, opts ...ListOption) error {
	var (
		err, applyErr error
		once          sync.Once
	)

	repoChan := make(chan model.RepoDescriptor)
	doneChan := make(chan struct{}, 1)

	clean := func() {
		close(doneChan)
	}
	interruptAndClean := func() {
		doneChan <- struct{}{}
		close(doneChan)
	}

	// collect repo metadata asynchronously
	go func(repoChan chan<- model.RepoDescriptor, doneChan chan struct{}) {
		defer close(repoChan)

		reposChan, workers := listReposChan(store, append(opts, WithDoneChan(doneChan))...)

		err = doSelectRepos(reposChan, func(repoBatch model.RepoDescriptors) {
			for _, repo := range repoBatch {
				repoChan <- repo // transfer a batch of metadata to the applied func
			}
		})
		once.Do(clean)

		workers.Wait()
	}(repoChan, doneChan)

	// apply function on collected metadata
	for repo := range repoChan {
		if applyErr = apply(repo); applyErr != nil {
			// wind down goroutines, but when nothing is left to be interrupted
			once.Do(interruptAndClean)
			for range repoChan {
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

// repoEvent catches a single repo with possible retrieval error
type repoEvent struct {
	repo model.RepoDescriptor
	err  error
}

// reposEvent catches a collection of repos with possible retrieval error
type reposEvent struct {
	repos model.RepoDescriptors
	err   error
}

func listReposChan(store storage.Store, opts ...ListOption) (chan reposEvent, *sync.WaitGroup) {
	var wg sync.WaitGroup

	settings := defaultSettings()
	for _, bApply := range opts {
		bApply(&settings)
	}

	batchChan := make(chan reposEvent, 1) // buffered to 1 to avoid blocking on early errors

	// internal signaling channels
	doneWithKeysChan := make(chan struct{}, 1)
	doneWithReposChan := make(chan struct{}, 1)

	if settings.doneChannel != nil {
		// watch for an interruption signal requested by caller
		wg.Add(1)
		go watchForInterrupts(settings.doneChannel, &wg, doneWithKeysChan, doneWithReposChan)
	}

	keysChan := make(chan keyBatchEvent, 1)

	iterator := func(next string) ([]string, string, error) {
		return store.KeysPrefix(context.Background(), next, model.GetArchivePathPrefixToRepos(), "", settings.batchSize)
	}
	// starting keys retrieval
	wg.Add(1)
	go fetchKeys(store, iterator, keysChan, doneWithKeysChan, &wg) // scan for key batches

	// start repo metadata retrieval
	wg.Add(1)
	go fetchRepos(store, settings, keysChan, batchChan, doneWithKeysChan, doneWithReposChan, &wg)

	// let the gc clean up internal signaling channels left open after wg goroutines are done.

	// return at once. Caller may chose to wait on returned WaitGroup
	return batchChan, &wg
}

func doSelectRepos(reposChan <-chan reposEvent, do func(model.RepoDescriptors)) error {
	// consume batches of ordered repo metadata
	for repoBatch := range reposChan {
		if repoBatch.err != nil {
			return repoBatch.err
		}
		do(repoBatch.repos)
	}
	return nil
}

// fetchRepos waits on a channel of key batches and outputs batches of descriptors corresponding to these keys
func fetchRepos(store storage.Store, settings Settings,
	keysChan <-chan keyBatchEvent, batchChan chan<- reposEvent,
	doneWithKeysChan chan<- struct{}, doneChan <-chan struct{}, wg *sync.WaitGroup) {
	defer func() {
		close(batchChan)
		wg.Done()
	}()

	for {
		select {
		case <-doneChan:
			batchChan <- reposEvent{err: status.ErrInterrupted}
			return
		case keyBatch, isOpen := <-keysChan:
			if !isOpen {
				return
			}
			if keyBatch.err != nil {
				batchChan <- reposEvent{err: keyBatch.err}
				return
			}
			batch, err := fetchRepoBatch(store, settings, keyBatch.keys)
			if err != nil {
				doneWithKeysChan <- struct{}{} // stop co-worker
				batchChan <- reposEvent{err: err}
				return
			}
			// send out a single batch of (ordered) bundle descriptors
			batchChan <- reposEvent{repos: batch}
		}
	}
}

// fetchRepoBatch performs a parallel fetch for a batch of repos identified by their keys,
// then reorders the result by key.
func fetchRepoBatch(store storage.Store, settings Settings, keys []string) (model.RepoDescriptors, error) {
	var (
		workers, wg sync.WaitGroup
		werr        error
	)

	repoChan := make(chan repoEvent)
	keyChan := make(chan string)
	doneChan := make(chan struct{}, 1)
	defer close(doneChan)

	// spin up workers pool
	for i := 0; i < minInt(settings.concurrentList, len(keys)); i++ {
		workers.Add(1)
		go getRepoAsync(store, keyChan, repoChan, &workers)
	}

	rps := make(model.RepoDescriptors, 0, len(keys))

	// distribute work. Stop immediately on first error reported by a worker
	wg.Add(1)
	go distributeKeys(keys)(keyChan, doneChan, &wg)

	// wait for workers to complete
	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		workers.Wait()
		close(repoChan)
	}(&wg)

	// watch for results and coalesce
	for rp := range repoChan {
		if rp.err != nil && werr == nil {
			werr = rp.err
			doneChan <- struct{}{} // interrupts key distribution (non-blocking)
			for range repoChan {
			} // wait for close
			break
		}
		rps = append(rps, rp.repo)
	}

	wg.Wait()

	if werr != nil {
		return nil, werr
	}

	// sort result batch
	sort.Sort(rps)
	return rps, nil
}

// getRepoAsync fetches and unmarshalls the repo descriptor for each single key submitted as input
func getRepoAsync(store storage.Store, input <-chan string, output chan<- repoEvent, wg *sync.WaitGroup) {
	defer wg.Done()
	for k := range input {
		apc, err := model.GetArchivePathComponents(k)
		if err != nil {
			output <- repoEvent{err: err}
			continue
		}
		rd, err := getRepoDescriptorByRepoName(store, apc.Repo)
		if err != nil {
			if errors.Is(err, storagestatus.ErrNotExists) {
				continue
			}
			output <- repoEvent{err: err}
			continue
		}
		output <- repoEvent{repo: rd}
	}
}
