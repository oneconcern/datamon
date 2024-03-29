package core

import (
	"time"

	context2 "github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/dlogger"
	"go.uber.org/zap"
)

const (
	MB = 1024 * 1024
)

const (
	KVTypeBadger KVType = iota
	KVTypePebble
)

type (
	KVType uint8

	// PurgeOption modifies the behavior of the purge operations.
	PurgeOption func(*purgeOptions)

	purgeOptions struct {
		force            bool
		dryRun           bool
		resume           bool
		localStorePath   string
		l                *zap.Logger
		extraStores      []context2.Stores
		maxParallel      int
		indexChunkSize   uint64 // in # of keys
		uploaderInterval time.Duration
		monitorInterval  time.Duration
		indexStart       uint64
		kvType           KVType

		kvOptions
	}
)

func WithPurgeForce(enabled bool) PurgeOption {
	return func(o *purgeOptions) {
		o.force = enabled
	}
}

func WithPurgeParallel(parallel int) PurgeOption {
	return func(o *purgeOptions) {
		if parallel > 0 {
			o.maxParallel = parallel
		}
	}
}

func WithPurgeDryRun(enabled bool) PurgeOption {
	return func(o *purgeOptions) {
		o.dryRun = enabled
	}
}

func WithPurgeLocalStore(pth string) PurgeOption {
	return func(o *purgeOptions) {
		if pth != "" {
			o.localStorePath = pth
		}
	}
}

func WithPurgeLogger(zlg *zap.Logger) PurgeOption {
	return func(o *purgeOptions) {
		if zlg != nil {
			o.l = zlg
		}
	}
}

func WithPurgeExtraContexts(extraStores []context2.Stores) PurgeOption {
	return func(o *purgeOptions) {
		o.extraStores = extraStores
	}
}

func WithPurgeIndexChunkSize(chunkSize uint64) PurgeOption {
	return func(o *purgeOptions) {
		o.indexChunkSize = chunkSize
	}
}

func WithPurgeIndexChunkStart(indexStart int) PurgeOption {
	return func(o *purgeOptions) {
		if indexStart > 0 {
			o.indexStart = uint64(indexStart)
		}
	}
}

// WithPurgeResumeIndex will resume index building by first reloading a fresh KV
// with all the already uploaded index files.
func WithPurgeResumeIndex(enabled bool) PurgeOption {
	return func(o *purgeOptions) {
		o.resume = enabled
	}
}

// WithKVType selects the KV implementation (badgerDB or pebble).
//
// The current default is pebble.
func WithKVType(kvType KVType) PurgeOption {
	return func(o *purgeOptions) {
		o.kvType = kvType
	}
}

func defaultPurgeOptions(opts []PurgeOption) *purgeOptions {
	o := &purgeOptions{
		localStorePath:   ".datamon-index",
		l:                dlogger.MustGetLogger("info"),
		maxParallel:      10,
		indexChunkSize:   500000,
		uploaderInterval: 5 * time.Minute,
		monitorInterval:  5 * time.Minute,
		kvType:           KVTypePebble, // defaults to pebble, since we have encountered issues with badger DB
		kvOptions:        defaultKVOptions(),
	}

	for _, apply := range opts {
		apply(o)
	}

	return o
}
