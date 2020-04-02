package core

import (
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	"go.uber.org/zap"
)

// KeyFilter is a function to filter the files to upload
type KeyFilter func(string) bool

// KeyIterator retrieves all keys from a store
type KeyIterator func(string) ([]string, error)

// SplitOption defines an option for a Split
type SplitOption func(*Split)

// SplitDescriptor sets the model descriptor for a Split
func SplitDescriptor(d *model.SplitDescriptor) SplitOption {
	return func(s *Split) {
		if d != nil {
			s.SplitDescriptor = *d
		}
	}
}

// SplitMustExist sets a split which must already be existing on metadata storage
// when created.
func SplitMustExist(d bool) SplitOption {
	return func(s *Split) {
		s.mustExist = d
	}
}

// SplitLogger sets a logger on the split object
func SplitLogger(l *zap.Logger) SplitOption {
	return func(s *Split) {
		if l != nil {
			s.l = l
		}
	}
}

// SplitConsumableStore defines the consumable storage for a split
func SplitConsumableStore(store storage.Store) SplitOption {
	return func(s *Split) {
		s.ConsumableStore = store
	}
}

// SplitKeyFilter defines a filter on the keys to be uploaded.
func SplitKeyFilter(f KeyFilter) SplitOption {
	return func(s *Split) {
		if f != nil {
			s.filter = f
		}
	}
}

// SplitSkipMissing indicates that file retrieval errors should be ignored
func SplitSkipMissing(skip bool) SplitOption {
	return func(s *Split) {
		s.SkipOnError = skip
	}
}

// SplitConcurrentFileUploads tunes the level of concurrency when uploading files
func SplitConcurrentFileUploads(concurrentFileUploads int) SplitOption {
	return func(s *Split) {
		s.concurrentFileUploads = concurrentFileUploads
	}
}

// SplitKeyIterator defines a custom key iterator function to upload files.
// KeyIterator may be used independently from KeyFilter.
func SplitKeyIterator(iterator KeyIterator) SplitOption {
	return func(s *Split) {
		if iterator != nil {
			s.getKeys = iterator
		}
	}
}

// SplitWithMetrics toggles metrics on a core Split object
func SplitWithMetrics(enabled bool) SplitOption {
	return func(s *Split) {
		s.EnableMetrics(enabled)
	}
}
