// Copyright © 2019 One Concern

package context

import (
	"bytes"
	"context"
	"fmt"

	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
)

// Stores for datamon.
type Stores struct {
	wal       storage.Store
	readLog   storage.Store
	blob      storage.Store
	metadata  storage.Store
	vMetadata storage.Store
	_         struct{}
}

func NewStores(wal, readLog, blob, metadata, vMetadata storage.Store) Stores {
	return Stores{wal: wal, readLog: readLog, blob: blob, metadata: metadata, vMetadata: vMetadata}
}

func (c *Stores) ReadLog() storage.Store {
	return c.readLog
}

func (c *Stores) SetReadLog(readLog storage.Store) {
	c.readLog = readLog
}

func (c *Stores) SetVMetadata(vMetadata storage.Store) {
	c.vMetadata = vMetadata
}

func (c *Stores) SetMetadata(metadata storage.Store) {
	c.metadata = metadata
}

func (c *Stores) SetBlob(blob storage.Store) {
	c.blob = blob
}

func (c *Stores) SetWal(wal storage.Store) {
	c.wal = wal
}

func (c *Stores) Metadata() storage.Store {
	return c.metadata
}

func (c *Stores) VMetadata() storage.Store {
	return c.vMetadata
}

func (c *Stores) Blob() storage.Store {
	return c.blob
}

func (c *Stores) Wal() storage.Store {
	return c.wal
}

func CreateContext(ctx context.Context, configStore storage.Store, context model.Context) error {
	// 1. Validate
	err := model.ValidateContext(context)
	if err != nil {
		return fmt.Errorf("validation for new context %s failed, err: %v", context.Name, err)
	}
	// 2. Serialize
	b, err := model.MarshalContext(&context)
	if err != nil {
		return fmt.Errorf("failed to serialize context: %v", err)
	}
	// 3. Create only
	path := model.GetPathToContext(context.Name)
	err = configStore.Put(ctx, path, bytes.NewReader(b), storage.NoOverWrite)
	if err != nil {
		return fmt.Errorf("failed to write context %v: %v", context, err)
	}
	return nil
}
