package core

import (
	"github.com/oneconcern/datamon/pkg/storage"
	"go.uber.org/zap"
)

type fileIndexOption func(*fileIndex)

func fileIndexPather(iterator patherIterator) func(*fileIndex) {
	return func(f *fileIndex) {
		f.indexPather = iterator
	}
}

func fileIndexMeta(store storage.Store) func(*fileIndex) {
	return func(f *fileIndex) {
		if store != nil {
			f.meta = store
		}
	}
}

func fileIndexLogger(l *zap.Logger) func(*fileIndex) {
	return func(f *fileIndex) {
		if l != nil {
			f.l = l
		}
	}
}
