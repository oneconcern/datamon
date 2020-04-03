package core

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	context2 "github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/core/status"
	"github.com/oneconcern/datamon/pkg/storage"

	"github.com/oneconcern/datamon/pkg/model"
)

const (
	typicalLabelsNum = 1000 // default number of allocated slots for labels
)

// ListLabels returns all labels from a repo
func ListLabels(repo string, stores context2.Stores, opts ...Option) ([]model.LabelDescriptor, error) {
	labels := make(model.LabelDescriptors, 0, typicalLabelsNum)

	labelsChan, workers := listLabelsChan(repo, stores, opts...)

	// consume batches of ordered labels
	err := doSelectLabels(labelsChan, func(labelBatch model.LabelDescriptors) {
		labels = append(labels, labelBatch...)
	})

	workers.Wait()

	return labels, err // we may have some batches resolved before the error occurred
}

// ApplyLabelFunc is a function to be applied on a label
type ApplyLabelFunc func(model.LabelDescriptor) error

// ListLabelsApply applies some function to the retrieved labels, in lexicographic order of keys.
func ListLabelsApply(repo string, store context2.Stores, apply ApplyLabelFunc, opts ...Option) error {
	var (
		err, applyErr error
		once          sync.Once
	)

	labelChan := make(chan model.LabelDescriptor)
	doneChan := make(chan struct{}, 1)

	clean := func() {
		close(doneChan)
	}
	interruptAndClean := func() {
		doneChan <- struct{}{}
		close(doneChan)
	}

	// collect label metadata asynchronously
	go func(labelChan chan<- model.LabelDescriptor, doneChan chan struct{}) {
		defer close(labelChan)

		labelsChan, workers := listLabelsChan(repo, store, append(opts, WithDoneChan(doneChan))...)

		err = doSelectLabels(labelsChan, func(labelBatch model.LabelDescriptors) {
			for _, label := range labelBatch {
				labelChan <- label // transfer a batch of metadata to the applied func
			}
		})
		once.Do(clean)

		workers.Wait()
	}(labelChan, doneChan)

	// apply function on collected metadata
	for label := range labelChan {
		if applyErr = apply(label); applyErr != nil {
			// wind down goroutines, but when nothing is left to be interrupted
			once.Do(interruptAndClean)
			for range labelChan {
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

// labelEvent catches a single label with possible retrieval error
type labelEvent struct {
	label model.LabelDescriptor
	err   error
}

// labelsEvent catches a collection of labels with possible retrieval error
type labelsEvent struct {
	labels model.LabelDescriptors
	err    error
}

func listLabelsChan(repo string, stores context2.Stores, opts ...Option) (chan labelsEvent, *sync.WaitGroup) {
	var wg sync.WaitGroup

	settings := defaultSettings()
	for _, bApply := range opts {
		bApply(&settings)
	}
	prefix := settings.labelPrefix

	batchChan := make(chan labelsEvent, 1) // buffered to 1 to avoid blocking on early errors

	if settings.labelVersions {
		vstore, ok := getLabelStore(stores).(storage.VersionedStore)
		if !ok {
			batchChan <- labelsEvent{err: status.ErrVersionedStoreRequired.WrapMessage("%T does not implement VersionedStore", vstore)}
			close(batchChan)
			return batchChan, &wg
		}
		if ok, err := vstore.IsVersioned(context.Background()); !ok || err != nil {
			batchChan <- labelsEvent{err: status.ErrVersionedStoreRequired.WrapMessage("bucket does not enable versioning")}
			close(batchChan)
			return batchChan, &wg
		}
	}

	if err := RepoExists(repo, stores); err != nil {
		batchChan <- labelsEvent{err: err}
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
		return getLabelStore(stores).KeysPrefix(context.Background(), next, model.GetArchivePathPrefixToLabels(repo, prefix), "", settings.batchSize)
	}

	if settings.labelVersions {
		unversionedKeysChan := make(chan keyBatchEvent, 1)

		// starting keys retrieval
		wg.Add(1)
		go fetchKeys(iterator, unversionedKeysChan, doneWithKeysChan, &wg) // scan for key batches

		// keys versions expansion
		wg.Add(1)
		go versionedKeys(getLabelStore(stores).(storage.VersionedStore), unversionedKeysChan, keysChan, settings, &wg)
	} else {
		// starting keys retrieval, no versioning
		wg.Add(1)
		go fetchKeys(iterator, keysChan, doneWithKeysChan, &wg) // scan for key batches
	}
	// start repo metadata retrieval
	wg.Add(1)
	go fetchLabels(repo, stores, settings, keysChan, batchChan, doneWithKeysChan, doneWithBundlesChan, &wg)

	// let the gc clean up internal signaling channels left open after wg goroutines are done.

	// return at once. Caller may chose to wait on returned WaitGroup
	return batchChan, &wg
}

func doSelectLabels(labelsChan <-chan labelsEvent, do func(model.LabelDescriptors)) error {
	// consume batches of ordered label metadata
	for labelBatch := range labelsChan {
		if labelBatch.err != nil {
			return labelBatch.err
		}
		do(labelBatch.labels)
	}
	return nil
}

// fetchLabels waits on a channel of key batches and outputs batches of descriptors corresponding to these keys
func fetchLabels(repo string, stores context2.Stores, settings Settings,
	keysChan <-chan keyBatchEvent, batchChan chan<- labelsEvent,
	doneWithKeysChan chan<- struct{}, doneChan <-chan struct{}, wg *sync.WaitGroup) {
	defer func() {
		close(batchChan)
		wg.Done()
	}()

	for {
		select {
		case <-doneChan:
			batchChan <- labelsEvent{err: status.ErrInterrupted}
			return
		case keyBatch, isOpen := <-keysChan:
			if !isOpen {
				return
			}
			if keyBatch.err != nil {
				batchChan <- labelsEvent{err: keyBatch.err}
				return
			}
			batch, err := fetchLabelBatch(repo, stores, settings, keyBatch.keys)
			if err != nil {
				doneWithKeysChan <- struct{}{} // stop co-worker
				batchChan <- labelsEvent{err: err}
				return
			}
			// send out a single batch of (ordered) bundle descriptors
			batchChan <- labelsEvent{labels: batch}
		}
	}
}

// fetchLabelBatch performs a parallel fetch for a batch of labels identified by their keys,
// then reorders the result by key.
func fetchLabelBatch(repo string, stores context2.Stores, settings Settings, keys []string) (model.LabelDescriptors, error) {
	var (
		workers, wg sync.WaitGroup
		werr        error
	)

	labelChan := make(chan labelEvent)
	keyChan := make(chan string)
	doneChan := make(chan struct{}, 1)
	defer close(doneChan)

	// spin up workers pool
	for i := 0; i < minInt(settings.concurrentList, len(keys)); i++ {
		workers.Add(1)
		go getLabelAsync(repo, stores, keyChan, labelChan, settings.labelVersions, &workers)
	}

	lbs := make(model.LabelDescriptors, 0, len(keys))

	// distribute work. Stop immediately on first error reported by a worker
	wg.Add(1)
	go distributeKeys(keys)(keyChan, doneChan, &wg)

	// wait for workers to complete
	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		workers.Wait()
		close(labelChan)
	}(&wg)

	// watch for results and coalesce
	for lb := range labelChan {
		if lb.err != nil && werr == nil {
			werr = lb.err
			doneChan <- struct{}{} // interrupts key distribution (non-blocking)
			for range labelChan {
			} // wait for close
			break
		}
		lbs = append(lbs, lb.label)
	}

	wg.Wait()

	if werr != nil {
		return nil, werr
	}

	// sort result batch
	sort.Sort(lbs)
	return lbs, nil
}

// getLabelAsync fetches and unmarshalls the label descriptor for each single key submitted as input
func getLabelAsync(repo string, stores context2.Stores, input <-chan string, output chan<- labelEvent, withVersions bool, wg *sync.WaitGroup) {
	defer wg.Done()
	var version string
	for k := range input {
		if withVersions {
			// internally, we follow the {key}#{version} convention. This applies to metadata keys holding labels,
			// and is only part of the internal key fetching / versions expension pipeline (not exposed to caller).
			parts := strings.Split(k, "#")
			if len(parts) != 2 {
				panic("dev error: unexpected internal format for versioned keys")
			}
			k = parts[0]
			version = parts[1]
		}
		apc, err := model.GetArchivePathComponents(k)
		if err != nil {
			output <- labelEvent{err: err}
			continue
		}
		bundle := NewBundle(Repo(repo), ContextStores(stores))
		labelName := apc.LabelName
		label := NewLabel(LabelDescriptor(model.NewLabelDescriptor(model.LabelName(labelName))), LabelWithVersion(version))
		err = label.DownloadDescriptor(context.Background(), bundle, false)
		if err != nil {
			output <- labelEvent{err: err}
			continue
		}
		if label.Descriptor.Name == "" {
			label.Descriptor.Name = apc.LabelName
		} else if label.Descriptor.Name != apc.LabelName {
			output <- labelEvent{err: fmt.Errorf("label names in descriptor '%v' and archive path '%v' don't match", label.Descriptor.Name, apc.LabelName)}
			continue
		}
		output <- labelEvent{label: label.Descriptor}
	}
}
