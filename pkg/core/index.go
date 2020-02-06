package core

import (
	"sync"
	"time"

	context2 "github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/dlogger"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

// fileIndex manages an index of files based on file lists stored in metadata.
//
// Indexing is critical to be able to retrieve files from the blob storage.
//
// File lists are split into manageable chunks. By default each single index file holds up to 1000 file entries.
//
// It knows how to store and fetch file lists.
type fileIndex struct {
	metaObject

	indexPather    patherIterator
	output         chan bundleEntriesRes
	concurrency    int
	entriesPerFile int // max number of file entries in a single index file
	l              *zap.Logger
	_              struct{}
}

func defaultFileIndex(stores context2.Stores) *fileIndex {
	return &fileIndex{
		metaObject:     defaultMetaObject(stores.VMetadata()), // TODO(fred): nice - when generalizing this to other objects, default may change
		output:         make(chan bundleEntriesRes, bufferingFactor*defaultBundleEntriesPerFile),
		concurrency:    100,
		l:              dlogger.MustGetLogger("info"),
		entriesPerFile: defaultBundleEntriesPerFile,
	}
}

func newFileIndex(stores context2.Stores, opts ...fileIndexOption) *fileIndex {
	f := defaultFileIndex(stores)
	for _, apply := range opts {
		apply(f)
	}
	return f
}

// Iterator returns the current patherIterator for this infex
func (f *fileIndex) Iterator() patherIterator {
	return f.indexPather
}

// Download bundle or splits file entries into an output channel.
//
// Download may iterate over several bundles or splits.
func (f *fileIndex) Download() error {
	return f.unpack()
}

// OutputChan provides the output channel for Download.
//
// Download will close the channel when done.
func (f *fileIndex) OutputChan() chan bundleEntriesRes {
	return f.output
}

// Upload file entries for a single bundle or split from input channel to index files.
//
// It returns the number of index files uploaded.
func (f *fileIndex) Upload(filePackedC chan filePacked, errorC chan errorHit, doneOkC chan struct{}) (uint64, error) {
	return f.pack(filePackedC, errorC, doneOkC)
}

// Reset all index files to empty files
func (f *fileIndex) Reset() error {
	return f.reset()
}

// uploadIndex uploads a single file list index file
func (f *fileIndex) uploadIndex(fileList []model.BundleEntry, pth string) error {
	buffer, err := yaml.Marshal(model.BundleEntries{
		BundleEntries: fileList,
	})
	if err != nil {
		return err
	}

	err = f.writeMetadata(pth, storage.NoOverWrite, buffer)
	if err != nil {
		return err
	}
	return nil
}

// getIndexfile retrieves a single index file and unmarshals its content
func (f *fileIndex) getIndexFile(pth string) (model.BundleEntries, error) {
	buffer, err := f.readMetadata(pth)
	if err != nil {
		return model.BundleEntries{}, err
	}

	var entries model.BundleEntries
	if err := yaml.Unmarshal(buffer, &entries); err != nil {
		return model.BundleEntries{}, err
	}
	return entries, nil
}

// downloadIndex asynchronously downloads the i-th file index located by pather
func (f *fileIndex) downloadIndex(id, pth string, index uint64, chans downloadBundleFileListChans) {
	defer func() {
		<-chans.concurrencyControl
	}()

	f.l.Info("downloading index file", zap.String("for", id), zap.String("file", pth), zap.Uint64("current entry", index))
	entries, err := f.getIndexFile(pth)
	if err != nil {
		chans.error <- err
		return
	}

	// TODO(fred): nice - diamond does not use the idx field, but bundle_unpack does,
	// we keep it for this method to be reusable by bundle_unpack at a later time.
	// Ideally, we shouldn't need that index numbering.
	chans.bundleEntries <- bundleEntriesRes{bundleEntries: entries, idx: index, id: id}
}

// downloadAll downloads all index files in parallel, given an iterator to point to the file location
func (f *fileIndex) downloadAll(chans downloadBundleFileListChans) {
	throttle := make(chan struct{}, f.concurrency)
	chans.concurrencyControl = throttle

	for id, pather := f.indexPather.Next(); pather != nil; id, pather = f.indexPather.Next() {
		i := uint64(0)
		for pth := pather.Next(); pth != ""; pth = pather.Next() {
			f.l.Debug("download file index from object", zap.String("from", id), zap.String("index", pth))
			throttle <- struct{}{}                // add up to buffer size, then block
			go f.downloadIndex(id, pth, i, chans) // download index file for split {id}, at location pth (with index num {i})
			i++
		}
	}

	// ensure all started goroutines are done
	for i := 0; i < cap(throttle); i++ {
		throttle <- struct{}{}
	}

	f.l.Debug("download done")
	chans.doneOk <- struct{}{}
}

// reset reset all index files to empty files
func (f *fileIndex) reset() error {
	throttle := make(chan struct{}, f.concurrency)
	errC := make(chan error)

	var (
		err error
		wg  sync.WaitGroup
	)

	wg.Add(1)
	go func(errC <-chan error, wg *sync.WaitGroup) {
		defer wg.Done()
		err = <-errC
	}(errC, &wg)

	for _, pather := f.indexPather.Next(); pather != nil; _, pather = f.indexPather.Next() {
		for pth := pather.Next(); pth != ""; pth = pather.Next() {
			throttle <- struct{}{} // add up to buffer size, then block
			go func(pth string, throttle <-chan struct{}, errC chan<- error) {
				defer func() {
					<-throttle
				}()
				erw := f.writeMetadata(pth, storage.NoOverWrite, []byte{})
				if erw != nil {
					errC <- erw
				}
			}(pth, throttle, errC)
		}
	}

	// ensure all started goroutines are done
	for i := 0; i < cap(throttle); i++ {
		throttle <- struct{}{}
	}
	close(errC)
	wg.Wait()

	return err
}

// unpack channels all the content of the iterated index files to output
func (f *fileIndex) unpack() error {
	f.l.Info("kicking off file list download", zap.Int("concurrent filelist downloads", f.concurrency))

	bundleEntriesC := make(chan bundleEntriesRes)
	errorC := make(chan error)
	doneC := make(chan struct{})
	doneOkC := make(chan struct{})

	defer func() {
		close(doneC)
		close(f.output)
	}()

	go f.downloadAll(downloadBundleFileListChans{
		bundleEntries: bundleEntriesC,
		error:         errorC,
		doneOk:        doneOkC,
	})

	var gotDoneSignal bool
	for !gotDoneSignal {
		select {

		case res := <-bundleEntriesC:
			f.output <- res // route unordered file lists to output channel, with index and originating object ID

		case err := <-errorC:
			f.l.Error("unpack bundle filelist failed", zap.Error(err))
			return err

		case <-doneOkC:
			gotDoneSignal = true
		}
	}
	return nil
}

// pack channels input file entries to index files
func (f *fileIndex) pack(filePackedC chan filePacked, errorC chan errorHit, doneOkC chan struct{}) (uint64, error) {
	// one single split or bundle is walked through
	_, uploadPather := f.indexPather.Next()
	if uploadPather == nil {
		return 0, nil
	}

	var (
		numFilePackedRes int
		count            uint64
	)

	fileList := make([]model.BundleEntry, 0, f.entriesPerFile)

	// wait for files on filePackedc until the input channel is closed, or a done signal is sent, or an error occurs
	//
	// NOTE: the done signal method should not be used in the case of buffered channels: in that case, a closed input
	// channel is the correct way to use pack().
	var done bool
	for !done {
		select {
		case file, ok := <-filePackedC:
			if !ok {
				f.l.Debug("file input channel now closed. Done", zap.String("file", file.name))
				done = true
				break
			}
			numFilePackedRes++

			// TODO(fred): nice - since we do not really need the full filePacked info here, could be better to simplify the input type
			entry := filePacked2BundleEntry(file)
			entry.Timestamp = time.Now() // this registers the time of upload (used to disambiguate conflicts)
			fileList = append(fileList, entry)

			// Write the bundle entry file if reached max or the last one
			if len(fileList) == f.entriesPerFile {
				f.l.Debug("Uploading filelist (max entries reached)")
				if err := f.uploadIndex(fileList, uploadPather.Next()); err != nil { // TODO(fred): performances - upload async
					f.l.Error("Split upload failed. Failed to upload split entries list.", zap.Error(err))
					return count, err
				}
				count++
				fileList = fileList[:0]
			}

		case e := <-errorC:
			f.l.Error("Split upload failed. Failed to upload file", zap.Error(e.error), zap.String("file", e.file))
			return count, e.error

		case <-doneOkC:
			f.l.Debug("Got upload done signal")
			done = true
		}
	}

	if len(fileList) != 0 {
		f.l.Debug("Uploading filelist (final)")
		if err := f.uploadIndex(fileList, uploadPather.Next()); err != nil { // TODO(fred): performances - upload async
			f.l.Error("Split upload failed. Failed to upload split entries list.", zap.Error(err))
			return count, err
		}
		count++
	}

	f.l.Info("uploaded filelists",
		zap.Uint64("actual number uploads attempted", count),
		zap.Int("approx expected number of uploads", maxInt(numFilePackedRes/f.entriesPerFile, 1)),
	)
	return count, nil
}
