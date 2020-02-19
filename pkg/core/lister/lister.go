package lister

import (
	"runtime"
	"sort"
	"sync"

	"github.com/oneconcern/datamon/pkg/errors"

	"github.com/oneconcern/datamon/pkg/core/status"
	storagestatus "github.com/oneconcern/datamon/pkg/storage/status"
)

var (
	defaultConcurrency = 2 * runtime.NumCPU()
)

// Listable knows how to list things
type Listable interface{}

// Lister provides support to efficiently list collections of metadata objects from a store.
// Items are sorted according to the lexicographic key order.
//
// Lister works with any Listable object provided some required options are set:
//   * Iterator(): knows how to fetch all keys on store for this object
//   * Downloader(): knows how to fetch a descriptor for this object and how to unmarshal it from yaml
//
// The Check() option is not required.
type Lister struct {
	checker    func() error                           // checks prerequisites
	iterator   func(string) ([]string, string, error) // iterates over keys
	downloader func(string) (Listable, error)         // download and unmarshals metadata
	doneChan   chan struct{}                          // optional interruption channel available to stop listing
	concurrent int
	typical    int
}

func defaultLister() *Lister {
	return &Lister{
		concurrent: defaultConcurrency,
		typical:    1000,
	}
}

// New builds a new Lister
func New(opts ...Option) *Lister {
	l := defaultLister()
	for _, apply := range opts {
		apply(l)
	}
	if l.iterator == nil || l.downloader == nil {
		panic("dev error: missing required options to Lister")
	}
	return l
}

// ApplyListableFunc is a function to be applied on a listable
type ApplyListableFunc func(Listable) error

// iterListablesFunc iterates a collection of Listable
type iterListablesFunc func([]Listable)

// listableEvent catches a single listable with possible retrieval error
type listableEvent struct {
	listable Listable
	key      string
	err      error
}

type keyedListable []listableEvent

func (k keyedListable) Swap(i, j int) {
	k[i], k[j] = k[j], k[i]
}
func (k keyedListable) Len() int {
	return len(k)
}
func (k keyedListable) Less(i, j int) bool {
	return k[i].key < k[j].key
}

// listablesEvent catches a collection of listables with possible retrieval error
type listablesEvent struct {
	listables []Listable
	err       error
}

// doSelectListables is a helper function to listen on a channel of batches of listable descriptors.
//
// It applies some function on the received batches and returns upon completion or error.
//
// Example usage:
//
//		err := doSelectListables(listablesChan, func(listableBatch []Listable) {
//			listables = append(listables, listableBatch...)
//		})
func (l *Lister) doSelectListables(input <-chan listablesEvent, do iterListablesFunc) error {
	// consume batches of ordered listable metadata
	for batch := range input {
		if batch.err != nil {
			return batch.err
		}
		do(batch.listables)
	}
	return nil
}

// ListableApply applies some function to the retrieved listables, in lexicographic order of keys.
//
// The execution of the applied function does not block background retrieval of more keys and listable descriptors.
//
// Example usage: printing listable descriptors as they come
//
//   err := lister.ListableApply(func(listable Listable) error {
//				fmt.Fprintf(os.Stderr, "%v\n", listable)
//				return nil
//			})
func (l *Lister) ListableApply(apply ApplyListableFunc) error {
	var (
		err, applyErr error
		once          sync.Once
	)

	listChan := make(chan Listable)
	l.doneChan = make(chan struct{}, 1)

	clean := func() {
		close(l.doneChan)
	}
	interruptAndClean := func() {
		l.doneChan <- struct{}{}
		close(l.doneChan)
	}

	// collect listable metadata asynchronously
	go func(listChan chan<- Listable, doneChan chan struct{}) {
		defer close(listChan)

		listablesChan, workers := l.listablesChan()

		err = l.doSelectListables(listablesChan, func(batch []Listable) {
			for _, listable := range batch {
				listChan <- listable // transfer a batch of metadata to the applied func
			}
		})
		once.Do(clean)

		workers.Wait()
	}(listChan, l.doneChan)

	// apply function on collected metadata
	for listable := range listChan {
		if applyErr = apply(listable); applyErr != nil {
			// wind down goroutines, but when nothing is left to be interrupted
			once.Do(interruptAndClean)
			for range listChan {
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

// List returns a list of listable descriptors from a repo. It collects all listables until completion.
//
// NOTE: this func could become deprecated. At this moment, however, it is used by pkg/web.
func (l *Lister) List() ([]Listable, error) {
	listables := make([]Listable, 0, l.typical)
	listablesChan, workers := l.listablesChan()

	// consume batches of ordered listables
	err := l.doSelectListables(listablesChan, func(batch []Listable) {
		listables = append(listables, batch...)
	})

	workers.Wait()

	return listables, err // we may have some batches resolved before the error occurred
}

// listListablesChan returns a list of listable descriptors from a repo. Each batch of returned descriptors
// is sent on the output channel, following key lexicographic order.
//
// Simple use cases of this helper are wrapped in ListListables (block until completion) and ListListablesApply
// (apply function while retrieving metadata).
//
// A signaling channel may be given as option to interrupt background processing (e.g. on error).
//
// The sync.WaitGroup for internal goroutines is returned if caller wants to wait and avoid any leaked goroutines.
func (l *Lister) listablesChan() (chan listablesEvent, *sync.WaitGroup) {
	var wg sync.WaitGroup

	batchChan := make(chan listablesEvent, 1) // buffered to 1 to avoid blocking on early errors

	if l.checker != nil {
		if err := l.checker(); err != nil {
			batchChan <- listablesEvent{err: err}
			close(batchChan)
			return batchChan, &wg
		}
	}

	// internal signaling channels
	doneWithKeysChan := make(chan struct{}, 1)
	doneWithListablesChan := make(chan struct{}, 1)

	if l.doneChan != nil {
		// watch for an interruption signal requested by caller
		wg.Add(1)
		go watchForInterrupts(l.doneChan, &wg, doneWithKeysChan, doneWithListablesChan)
	}

	keysChan := make(chan keyBatchEvent, 1)

	// starting keys retrieval
	wg.Add(1)
	go fetchKeys(l.iterator, keysChan, doneWithKeysChan, &wg) // scan for key batches

	// start listable metadata retrieval
	wg.Add(1)
	go l.fetchListables(keysChan, batchChan, doneWithKeysChan, doneWithListablesChan, &wg)

	// let the gc clean up internal signaling channels left open after wg goroutines are done.

	// return at once. Caller may chose to wait on returned WaitGroup
	return batchChan, &wg
}

// fetchListables waits on a channel of key batches and outputs batches of descriptors corresponding to these keys
func (l *Lister) fetchListables(keysChan <-chan keyBatchEvent, batchChan chan<- listablesEvent, doneWithKeysChan chan<- struct{}, doneChan <-chan struct{}, wg *sync.WaitGroup) {
	defer func() {
		close(batchChan)
		wg.Done()
	}()

	for {
		select {
		case <-doneChan:
			batchChan <- listablesEvent{err: status.ErrInterrupted}
			return
		case keyBatch, isOpen := <-keysChan:
			if !isOpen {
				return
			}
			if keyBatch.err != nil {
				batchChan <- listablesEvent{err: keyBatch.err}
				return
			}
			batch, err := l.fetchBatch(keyBatch.keys)
			if err != nil {
				doneWithKeysChan <- struct{}{} // stop co-worker
				batchChan <- listablesEvent{err: err}
				return
			}
			// send out a single batch of (ordered) listable descriptors
			batchChan <- listablesEvent{listables: batch}
		}
	}
}

// fetchBatch performs a parallel fetch for a batch of listables identified by their keys,
// then reorders the result by key.
//
// TODO: this performs a parallel fetch for a batch of keys. However, we wait until completion of this batch to start
// a new one. In addition, for every new batch of key, we spin up a new pool of workers.
// We could improve this further by streaming batches of keys then stashing looked-ahead results and directly obtain
// a sorted output.
func (l *Lister) fetchBatch(keys []string) ([]Listable, error) {
	var (
		workers, wg sync.WaitGroup
		werr        error
	)

	listableChan := make(chan listableEvent)
	keyChan := make(chan string)
	doneChan := make(chan struct{}, 1)
	defer close(doneChan)

	// spin up workers pool
	// TODO(fred): use x/errgroup pattern here
	for i := 0; i < minInt(l.concurrent, len(keys)); i++ {
		workers.Add(1)
		go l.getListableAsync(keyChan, listableChan, &workers)
	}

	bds := make(keyedListable, 0, len(keys))

	// distribute work. Stop immediately on first error reported by a worker
	wg.Add(1)
	go distributeKeys(keys)(keyChan, doneChan, &wg)

	// wait for workers to complete
	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		workers.Wait()
		close(listableChan)
	}(&wg)

	// watch for results and coalesce
	for bd := range listableChan {
		if bd.err != nil && werr == nil {
			werr = bd.err
			doneChan <- struct{}{} // interrupts key distribution (non-blocking)
			for range listableChan {
			} // wait for close
			break
		}
		bds = append(bds, bd)
	}

	wg.Wait()

	if werr != nil {
		return nil, werr
	}

	// sort a single batch according to the key order
	sort.Sort(bds)
	result := make([]Listable, 0, len(keys))
	for _, bd := range bds {
		result = append(result, bd.listable)
	}
	return result, nil
}

// getListableAsync fetches and unmarshalls the listable descriptor for each single key submitted as input
func (l *Lister) getListableAsync(input <-chan string, output chan<- listableEvent, wg *sync.WaitGroup) {
	defer wg.Done()
	for k := range input {
		bd, err := l.downloader(k)
		if err != nil {
			if errors.Is(err, storagestatus.ErrNotExists) {
				continue
			}
			output <- listableEvent{err: err}
			continue
		}

		output <- listableEvent{listable: bd, key: k}
	}
}

// watchForInterrupts broadcasts a done signal to several output channels
func watchForInterrupts(doneChan <-chan struct{}, wg *sync.WaitGroup, outputChans ...chan<- struct{}) {
	defer wg.Done()

	if _, interrupt := <-doneChan; interrupt {
		for _, outputChan := range outputChans {
			outputChan <- struct{}{}
		}
	}
}
