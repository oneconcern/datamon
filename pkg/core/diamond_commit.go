package core

import (
	"sync"
	"time"

	iradix "github.com/hashicorp/go-immutable-radix"
	"github.com/oneconcern/datamon/pkg/core/status"
	"github.com/oneconcern/datamon/pkg/errors"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
)

// bufferingFactor determines the size of the channel buffers as a multiple of entriesPerFile
//
// NOTE: we can significantly speed up the process by allocating a larger buffer here. However, this
// stresses the implementation of Store, e.g. not closing descriptors properly (localfs) makes a high
// buferred process rapidly overcome system limits (i.e. max open files).
const bufferingFactor = 100

// Commit a diamond
func (d *Diamond) Commit(opts ...Option) error {
	return d.implCommit(opts...)
}

// Cancel a diamond
func (d *Diamond) Cancel() (err error) {
	defer func(t0 time.Time) {
		if d.MetricsEnabled() {
			d.m.Usage.UsedAll(t0, "DiamondCancel")(err)
		}
	}(time.Now())

	err = d.downloadDescriptor()
	if err != nil {
		return err
	}

	switch d.DiamondDescriptor.State {
	case model.DiamondCanceled, model.DiamondDone:
		return errors.New("diamond is already terminated").
			WrapMessage("diamond state: %v", d.DiamondDescriptor.State)
	default:
		err = d.WithState(model.DiamondCanceled).uploadDescriptor()
		if err != nil {
			return err
		}
	}
	return nil
}

// WithState returns the diamond with its state modified and transition timestamp updated
func (d *Diamond) WithState(state model.DiamondState) *Diamond {
	if state == d.DiamondDescriptor.State {
		return d
	}
	d.DiamondDescriptor.State = state

	switch state {
	case model.DiamondDone, model.DiamondCanceled:
		// writes down the time when reaching a terminal state
		d.DiamondDescriptor.EndTime = model.GetBundleTimeStamp()
	}
	return d
}

// collectSplits which are done, sorted by completion time
func (d *Diamond) collectSplits(opts ...Option) (model.SplitDescriptors, error) {
	splits := make(model.SplitDescriptors, 0, typicalSplitsNum)

	err := ListSplitsApply(
		d.RepoID,
		d.DiamondDescriptor.DiamondID,
		d.contextStores,
		func(sd model.SplitDescriptor) error {
			if sd.State == model.SplitDone {
				splits = append(splits, sd)
			}
			return nil
		},
		opts...,
	)
	if err != nil {
		return nil, errors.New("couldn't collect splits").Wrap(err)
	}
	return splits, nil
}

func (d *Diamond) checkBundleID() error {
	if !enableBundlePreserve {
		// feature guard: require specific build flag
		return nil
	}
	if d.BundleID == "" {
		// regular case: bundle ID is generated on the fly
		return nil
	}

	// case of bundleID preservation (e.g. data migration use case)
	id, err := ksuid.Parse(d.BundleID)
	if err != nil {
		return status.ErrInvalidKsuid.Wrap(err)
	}
	d.setBundleID(id.String())
	exists, err := d.Exists(d.contexter())
	if err != nil {
		return errors.New("could not check for bundle existence").Wrap(err)
	}
	if exists {
		return status.ErrBundleIDExists.WrapMessage("bundleID: %v", id)
	}
	return nil
}

func (d *Diamond) implCommit(opts ...Option) (err error) {
	defer func(t0 time.Time) {
		if d.MetricsEnabled() {
			d.m.Usage.UsedAll(t0, "DiamondCommit")(err)
		}
	}(time.Now())

	// check if repo exists
	if err = RepoExists(d.RepoID, d.contextStores); err != nil {
		return err
	}

	// check when bundle is already specified (e.g. data migration use case, with bundleID preservation)
	if err = d.checkBundleID(); err != nil {
		return err
	}

	logger := d.l.With(zap.String("diamond_id", d.DiamondDescriptor.DiamondID))

	// critical section #1

	// check that the diamond is ready to accept splits (e.g. not already done or canceled)
	if err = diamondReady(d.RepoID, d.DiamondDescriptor.DiamondID, d.contextStores); err != nil {
		return errors.New("cannot proceed with diamond commit").WrapWithLog(logger, err)
	}

	// before exiting, save the diamond state either as done or rollbacked to initialized
	defer func() {
		if err != nil {
			// TODO(fred): nice - last ditch check done automatically with nooverwrite: problem is error qualification...
			logger.Error("diamond commit failed", zap.Error(err))
			return
		}
		err = d.WithState(model.DiamondDone).uploadDescriptor()
		if err != nil {
			logger.Warn("failed to complete diamond commit, but bundle is available", zap.String("bundle_id", d.BundleID), zap.Error(err))
		}
	}()

	// walk all completed splits
	// TODO(fred): nice - performances - should be piped to next stage asynchronously - at the moment, we start collecting index files
	// only after metadata about all splits have been collected.
	splits, err := d.collectSplits(opts...)
	if err != nil {
		return err
	}

	if len(splits) == 0 {
		return errors.New("no split to commit")
	}

	// capture splits summarized metadata in the final diamond descriptor
	d.DiamondDescriptor.Splits = splits
	d.l.Info("splits for this commit", zap.Int("num_splits", len(splits)))

	// prepare indexer to walk over all index files for all splits
	d.splitIndexer = d.makeDownloadIndexer()

	// merge contributors
	d.BundleDescriptor.Contributors = d.BundleDescriptor.Contributors[:0]
	for _, split := range d.DiamondDescriptor.Splits {
		d.BundleDescriptor.Contributors = append(d.BundleDescriptor.Contributors, split.Contributors...)
	}

	if d.BundleID == "" {
		err = d.InitializeBundleID()
		if err != nil {
			return err
		}
	}

	// prepare index to upload bundle index files
	d.bundleIndexer = d.makeUploadIndexer()

	// upload file lists to the target bundle
	filePackedC := make(chan filePacked, bufferingFactor*d.bundleIndexer.entriesPerFile)
	errorC := make(chan errorHit)
	doneOkC := make(chan struct{})

	// start merging split file lists
	var wg sync.WaitGroup
	wg.Add(1)
	go d.mergeSplits(filePackedC, errorC, doneOkC, &wg)
	defer wg.Wait()

	// hand out all files to merger goroutine
	count, err := d.bundleIndexer.Upload(filePackedC, errorC, doneOkC)
	if err != nil {
		return err
	}

	// finalize bundle metadata
	d.BundleDescriptor.BundleEntriesFileCount = count

	err = uploadBundleDescriptor(d.contexter(), d.Bundle)
	if err != nil {
		return err
	}

	d.l.Info("uploaded bundle id", zap.String("BundleID", d.BundleID))
	d.DiamondDescriptor.BundleID = d.BundleID
	return nil
}

type mergeEntry struct {
	model.BundleEntry        // single file
	ID                string // split ID which uploaded this file
}

// mergeSplits merges all files from splits and resolves conflicts
// then produces the output to the given channels.
//
// TODO(fred): nice - ok so the channel stuff here might look irrelevant for that use case.
// I am reusing Download, which is asynchronous, but everything is buffered in the loop below.
// I am also using buffered channels here...
// I am thinking channel might be useful when we need to offload part of this to disk (asynchronously).
//
// TODO(fred): scalability - at the moment, the merger is carried out in memory. Use local storage to merge very large file lists
// TODO(fred): nice - a more careful choice of type in lieu of filePacked could avoid some extra data copy => will tend to that when refactoring with bundle
func (d *Diamond) mergeSplits(filePackedC chan<- filePacked, errorC chan<- errorHit, doneOkC chan<- struct{}, wgg *sync.WaitGroup) {
	defer wgg.Done()

	var (
		wg sync.WaitGroup

		// metrics
		conflicts, merged, bundleEntries uint64
	)

	mode := d.DiamondDescriptor.Mode
	t0 := time.Now()
	interrupt := make(chan struct{}, 1)

	// proceed with merger
	wg.Add(1)
	go func(input <-chan bundleEntriesRes, output chan<- filePacked, interrupt <-chan struct{}, wg *sync.WaitGroup) {
		defer wg.Done()

		mergeIndex := iradix.New()
		for res := range input {
			splitID := res.id
			d.l.Debug("merge received batch", zap.String("from split", splitID), zap.Int("num_entries", len(res.bundleEntries.BundleEntries)))
			for _, file := range res.bundleEntries.BundleEntries {
				select {
				case <-interrupt:
					return
				default:
				}
				merged++
				d.l.Debug("merge received file entry", zap.String("from split", splitID), zap.String("entry", file.NameWithPath))
				key := []byte(file.NameWithPath)
				obj, found := mergeIndex.Get(key)
				if !found {
					mergeIndex, _, _ = mergeIndex.Insert(key, mergeEntry{BundleEntry: file, ID: splitID})
					continue
				}

				existing := obj.(mergeEntry)
				if file.Hash == existing.Hash {
					continue
				}

				if file.Timestamp.IsZero() {
					d.l.Error("dev error: expecting files processed by diamond commit to have a timestamp", zap.Any("file", file))
					panic("dev error: files should have a timing") // internal safeguard
				}
				if file.Timestamp.After(existing.Timestamp) {
					// got a more recent file

					switch {
					case mode == model.IgnoreConflicts || splitID == existing.ID:
						// ignore conflict: replace existing entry with newer version
						// or: self-inflicted conflict, which is ignored
						mergeIndex, _, _ = mergeIndex.Insert(key, mergeEntry{BundleEntry: file, ID: splitID})

					case mode == model.ForbidConflicts:
						conflicts++
						errorC <- errorHit{
							error: status.ErrCommitGivenUp.
								WrapWithLog(d.l, status.ErrForbiddenConflict, zap.String("entry", file.NameWithPath)),
						}
						return

					default:
						// report conflict/checkpoint: add conflicting file to the bundle in some special location
						// (e.g. .conflicts/{splitID}/{path}) and update the key with the newer file
						existing.NameWithPath = d.deconflicter(splitID, existing.NameWithPath)
						d.l.Debug("deconflicting", zap.String("from", file.NameWithPath), zap.String("to", existing.NameWithPath))
						mergeIndex, _, _ = mergeIndex.Insert([]byte(existing.NameWithPath), existing)
						// overwrite with new version
						mergeIndex, _, _ = mergeIndex.Insert(key, mergeEntry{BundleEntry: file, ID: splitID})
						conflicts++
					}
				} else {
					// got an older file

					if splitID == existing.ID {
						// ignored self-inflicted conflict
						continue
					}

					switch mode {
					case model.EnableConflicts, model.EnableCheckpoints:
						newEntry := file
						newEntry.NameWithPath = d.deconflicter(splitID, existing.NameWithPath)
						d.l.Debug("deconflicting", zap.String("from", file.NameWithPath), zap.String("to", newEntry.NameWithPath))
						mergeIndex, _, _ = mergeIndex.Insert([]byte(d.deconflicter(splitID, file.NameWithPath)), mergeEntry{BundleEntry: newEntry, ID: splitID})
						conflicts++

					case model.ForbidConflicts:
						conflicts++
						errorC <- errorHit{
							error: status.ErrCommitGivenUp.
								WrapWithLog(d.l, status.ErrForbiddenConflict, zap.String("entry", file.NameWithPath)),
						}
						return
					}
				}
			}
		}

		d.l.Info("merge input completed", zap.Uint64("entries processed", merged), zap.Duration("elapsed", time.Since(t0)))
		if conflicts > 0 {
			d.l.Warn("conflicts detected", zap.Uint64("conflicts", conflicts))
			d.DiamondDescriptor.HasConflicts = d.DiamondDescriptor.Mode != model.EnableCheckpoints
			d.DiamondDescriptor.HasCheckpoints = d.DiamondDescriptor.Mode == model.EnableCheckpoints
		}

		// now dump the merged index as output
		t0 = time.Now()
		iterator := mergeIndex.Root().Iterator()
		for _, obj, ok := iterator.Next(); ok; _, obj, ok = iterator.Next() {
			bundleEntries++
			existing := obj.(mergeEntry)
			d.l.Debug("merge sending", zap.String("entry", existing.NameWithPath))
			output <- mergeEntryToFilePacked(existing)
		}
	}(d.splitIndexer.OutputChan(), filePackedC, interrupt, &wg)

	// feed the process with some split input, collecting all splits to be merged
	if err := d.splitIndexer.Download(); err != nil {
		d.l.Error("failed downloading list files for diamond", zap.Error(err))
		interrupt <- struct{}{}
		errorC <- errorHit{error: err}
		wg.Wait()
		return
	}

	// signals the recipient to end with channel close rather than done signal
	wg.Wait()

	d.l.Info("merge output completed", zap.Uint64("entries to bundle", bundleEntries), zap.Duration("elapsed", time.Since(t0)))
	close(filePackedC)
}

func mergeEntryToFilePacked(in mergeEntry) filePacked {
	return filePacked{
		hash: in.Hash,
		name: in.NameWithPath,
		size: in.Size,
		// remove snapshot time from bundle file index
	}
}

func (d *Diamond) makeDownloadIndexer() *fileIndex {
	return newFileIndex(
		d.contextStores,
		fileIndexMeta(GetDiamondStore(d.contextStores)),
		fileIndexPather(
			newDownloadAllSplitsIterator(
				d.RepoID,
				d.DiamondDescriptor.DiamondID,
				d.DiamondDescriptor.Splits,
			)),
		fileIndexLogger(d.l.With(
			zap.String("diamond_id", d.DiamondDescriptor.DiamondID),
		)),
	)
}

func (d *Diamond) makeUploadIndexer() *fileIndex {
	return newFileIndex(
		d.contextStores,
		fileIndexMeta(d.contextStores.Metadata()), // bundle metadata store
		fileIndexPather(
			newUploadBundleIterator(
				d.RepoID,
				d.BundleDescriptor,
			)),
		fileIndexLogger(d.l.With(
			zap.String("diamond_id", d.DiamondDescriptor.DiamondID),
			zap.String("bundle", d.BundleID),
		)),
	)
}
