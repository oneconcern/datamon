package core

import (
	"sync"

	"github.com/oneconcern/datamon/pkg/model"
)

// indexIterator knows how to iterate over a list of index files,
// containing BundleEntries referring to hashes on the blob storage.
type indexIterator interface {
	Next() string
}

// patherIterator is a super-iterator capable of rendering index files from
// a collection of objects (e.g. splits).
type patherIterator interface {
	// Next yields the next series of index files to retrieve, with some reusable identifier
	// and an iterator to walk through individual numbered index files.
	//
	// When completed, it returns ("", 0,nil).
	//
	// This iterator may be called concurrently.
	Next() (string, indexIterator)
}

// downloadIndexIterator implements indexIterator with a known max number of index files
// and a path function normally provided by the metadata model package
// (e.g. model.GetArchivePathToBundleFileList).
type downloadIndexIterator struct {
	i, max    uint64
	exclusive sync.Mutex
	pather    func(uint64) string
}

func (di *downloadIndexIterator) Next() string {
	di.exclusive.Lock()
	defer di.exclusive.Unlock()
	if di.i >= di.max {
		return ""
	}
	di.i++
	return di.pather(di.i - 1)
}

func newDownloadIndexIterator(max uint64, pather func(uint64) string) *downloadIndexIterator {
	return &downloadIndexIterator{
		max:    max,
		pather: pather,
	}
}

// uploadIndexIterator implements indexIterator with an unlimited number of index files
// and a path function normally provided by the metadata model package
// (e.g. model.GetArchivePathToBundleFileList).
type uploadIndexIterator struct {
	i         uint64
	exclusive sync.Mutex
	pather    func(uint64) string
}

func (ui *uploadIndexIterator) Next() string {
	ui.exclusive.Lock()
	defer ui.exclusive.Unlock()
	ui.i++
	return ui.pather(ui.i - 1)
}

func newUploadIndexIterator(pather func(uint64) string) *uploadIndexIterator {
	return &uploadIndexIterator{
		pather: pather,
	}
}

// downloadBundleIterator walks all file lists for a single bundle
type downloadBundleIterator struct {
	repoID    string
	bundle    model.BundleDescriptor
	iterated  bool
	exclusive sync.Mutex
}

func (bp *downloadBundleIterator) Next() (string, indexIterator) {
	bp.exclusive.Lock()
	defer bp.exclusive.Unlock()

	if bp.iterated {
		return "", nil
	}

	bp.iterated = true
	return bp.bundle.ID, newDownloadIndexIterator(
		bp.bundle.BundleEntriesFileCount,
		func(index uint64) string {
			return model.GetArchivePathToBundleFileList(bp.repoID, bp.bundle.ID, index)
		},
	)
}

func newDownloadBundleIterator(repo string, bundle model.BundleDescriptor) *downloadBundleIterator {
	return &downloadBundleIterator{
		repoID: repo,
		bundle: bundle,
	}
}

// uploadBundleIterator walks all file lists for a single bundle, with unlimited number of index files
// (used when uploading a bundle)
type uploadBundleIterator struct {
	repoID    string
	bundle    model.BundleDescriptor
	iterated  bool
	exclusive sync.Mutex
}

func (ub *uploadBundleIterator) Next() (string, indexIterator) {
	ub.exclusive.Lock()
	defer ub.exclusive.Unlock()

	if ub.iterated {
		return "", nil
	}

	ub.iterated = true
	return ub.bundle.ID, newUploadIndexIterator(
		func(index uint64) string {
			return model.GetArchivePathToBundleFileList(ub.repoID, ub.bundle.ID, index)
		},
	)
}

func newUploadBundleIterator(repo string, bundle model.BundleDescriptor) *uploadBundleIterator {
	return &uploadBundleIterator{
		repoID: repo,
		bundle: bundle,
	}
}

// downloadAllSplitsIterator walks all file lists from all splits of an uploaded diamond
type downloadAllSplitsIterator struct {
	repoID, diamondID string
	splits            []model.SplitDescriptor
	i                 int
	exclusive         sync.Mutex
}

func (sp *downloadAllSplitsIterator) Next() (string, indexIterator) {
	sp.exclusive.Lock()
	defer sp.exclusive.Unlock()

	if sp.i >= len(sp.splits) {
		return "", nil
	}
	sp.i++
	return sp.splits[sp.i-1].SplitID, newDownloadIndexIterator(
		sp.splits[sp.i-1].SplitEntriesFileCount,
		func(index uint64) string {
			return model.GetArchivePathToSplitFileList(sp.repoID, sp.diamondID, sp.splits[sp.i-1].SplitID, sp.splits[sp.i-1].GenerationID, index)
		},
	)
}

func newDownloadAllSplitsIterator(repo, diamondID string, splits []model.SplitDescriptor) *downloadAllSplitsIterator {
	return &downloadAllSplitsIterator{
		repoID:    repo,
		diamondID: diamondID,
		splits:    splits,
	}
}

// downloadSplitIterator walks all file lists from one single split, up to the known number of index files
type downloadSplitIterator struct {
	repoID, diamondID string
	split             model.SplitDescriptor
	iterated          bool
	exclusive         sync.Mutex
}

func (op *downloadSplitIterator) Next() (string, indexIterator) {
	op.exclusive.Lock()
	defer op.exclusive.Unlock()

	if op.iterated {
		return "", nil
	}

	op.iterated = true
	return op.split.SplitID, newDownloadIndexIterator(
		op.split.SplitEntriesFileCount,
		func(index uint64) string {
			return model.GetArchivePathToSplitFileList(op.repoID, op.diamondID, op.split.SplitID, op.split.GenerationID, index)
		},
	)
}

func newDownloadSplitIterator(repo, diamondID string, split model.SplitDescriptor) *downloadSplitIterator {
	return &downloadSplitIterator{
		repoID:    repo,
		diamondID: diamondID,
		split:     split,
	}
}

// uploadSplitIterator walks all file lists for one single split, with no limit on the number of indexes
// (used when uploading index files).
type uploadSplitIterator struct {
	repoID, diamondID string
	split             model.SplitDescriptor
	iterated          bool
	exclusive         sync.Mutex
}

func (up *uploadSplitIterator) Next() (string, indexIterator) {
	up.exclusive.Lock()
	defer up.exclusive.Unlock()

	if up.iterated {
		return "", nil
	}

	up.iterated = true
	return up.split.SplitID, newUploadIndexIterator(
		func(index uint64) string {
			return model.GetArchivePathToSplitFileList(up.repoID, up.diamondID, up.split.SplitID, up.split.GenerationID, index)
		},
	)
}

func newUploadSplitIterator(repo, diamondID string, split model.SplitDescriptor) *uploadSplitIterator {
	return &uploadSplitIterator{
		repoID:    repo,
		diamondID: diamondID,
		split:     split,
	}
}
