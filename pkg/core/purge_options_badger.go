package core

type kvOptions struct {
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

func defaultKVOptions() kvOptions {
	return kvOptions{
		kvIndexCacheSize:          200 << 20, // 200MB, badger default: 0
		kvBaseLevelSize:           10 * MB,   // badger default: 10MB
		kvBaseTableSize:           2 * MB,    // badger default: 2MB (or ~ 8k 128-bytes keys)
		kvLevelSizeMultiplier:     10,        // badger default: 10 [governs KV compaction process trigger]
		kvMaxLevels:               7,         // badger default: 7
		kvMemTableSize:            64 * MB,   // badger default: 64MB (or ~ 500k keys)
		kvNumLevelZeroTables:      5,         // badger default: 5
		kvNumLevelZeroTablesStall: 1024,      // badger default: 15 (-> ~ 1024 * 2MB = 8m keys)
		kvNumMemTables:            5,         // badger default: 5
		kvBlockCacheSize:          0,         // badger default: 256MB
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
