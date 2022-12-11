package core

import (
	"time"

	context2 "github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/dlogger"
	"go.uber.org/zap"
)

const MB = 1024 * 1024

type (
	// PurgeOption modifies the behavior of the purge operations.
	PurgeOption func(*purgeOptions)

	purgeOptions struct {
		force            bool
		dryRun           bool
		localStorePath   string
		l                *zap.Logger
		extraStores      []context2.Stores
		maxParallel      int
		indexChunkSize   uint64 // in # of keys
		uploaderInterval time.Duration
		monitorInterval  time.Duration

		kvOptions
	}

	kvOptions struct {
		kvIndexCacheSize          int64
		kvBaseLevelSize           int64
		kvBaseTableSize           int64
		kvLevelSizeMultiplier     int
		kvMaxLevels               int
		kvMemTableSize            int64
		kvNumLevelZeroTables      int
		kvNumLevelZeroTablesStall int
		kvNumMemTables            int
		kvBlockCacheSize          int64
	}
)

func defaultKVOptions() kvOptions {
	return kvOptions{
		kvIndexCacheSize:          200 << 20, // 200MB, badger default: 0
		kvBaseLevelSize:           10 * MB,   // badger default: 10MB
		kvBaseTableSize:           2 * MB,    // badger default: 2MB (or ~ 8k 128-bytes keys)
		kvLevelSizeMultiplier:     10,        // badger default: 10 [governs KV compaction process trigger]
		kvMaxLevels:               7,         // badger default: 7
		kvMemTableSize:            64 * MB,   // badger default: 64MB (or ~ 500k keys)
		kvNumLevelZeroTables:      5,         // badger default: 5
		kvNumLevelZeroTablesStall: 512,       // badger default: 15 (-> ~ 512 * 2MB = 4m keys)
		kvNumMemTables:            5,         // badger default: 5
		kvBlockCacheSize:          256 * MB,  // badger default: 256MB
	}
}

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

func WithPurgeKVIndexCacheSize(size int64) PurgeOption {
	return func(o *purgeOptions) {
		o.kvIndexCacheSize = size
	}
}

func WithPurgeKVBaseLevelSize(size int64) PurgeOption {
	return func(o *purgeOptions) {
		o.kvBaseLevelSize = size
	}
}

func WithPurgeKVBaseTableSize(size int64) PurgeOption {
	return func(o *purgeOptions) {
		o.kvBaseTableSize = size
	}
}

func WithPurgeKVLevelSizeMultiplier(mult int) PurgeOption {
	return func(o *purgeOptions) {
		o.kvLevelSizeMultiplier = mult
	}
}

func WithPurgeKVMaxLevels(levels int) PurgeOption {
	return func(o *purgeOptions) {
		o.kvMaxLevels = levels
	}
}

func WithPurgeKVNumLevelZeroTables(tables int) PurgeOption {
	return func(o *purgeOptions) {
		o.kvNumLevelZeroTables = tables
	}
}

func WithPurgeKVNumLevelZeroTablesStall(tables int) PurgeOption {
	return func(o *purgeOptions) {
		o.kvNumLevelZeroTablesStall = tables
	}
}

func WithPurgeKVNumMemTables(tables int) PurgeOption {
	return func(o *purgeOptions) {
		o.kvNumMemTables = tables
	}
}

func WithPurgeKVMemGTableSize(size int64) PurgeOption {
	return func(o *purgeOptions) {
		o.kvMemTableSize = size
	}
}

func WithPurgeKVBlockCacheSize(size int64) PurgeOption {
	return func(o *purgeOptions) {
		o.kvBlockCacheSize = size
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
		kvOptions:        defaultKVOptions(),
	}

	for _, apply := range opts {
		apply(o)
	}

	return o
}
