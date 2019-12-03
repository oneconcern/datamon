/*
 * Copyright © 2019 One Concern
 *
 */

package context

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"

	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
)

// Stores defines a complete context for datamon objects
type Stores struct {
	wal       storage.Store
	readLog   storage.Store
	blob      storage.Store
	metadata  storage.Store
	vMetadata storage.Store
	_         struct{}
}

// NewStores creates a new instance of context stores
func NewStores(wal, readLog, blob, metadata, vMetadata storage.Store) Stores {
	return Stores{wal: wal, readLog: readLog, blob: blob, metadata: metadata, vMetadata: vMetadata}
}

// ReadLog yields the Read Log storage for a context
func (c *Stores) ReadLog() storage.Store {
	return c.readLog
}

// SetReadLog sets the context storage for Read Log
func (c *Stores) SetReadLog(readLog storage.Store) {
	c.readLog = readLog
}

// SetVMetadata sets the context storage for versioning metadata
func (c *Stores) SetVMetadata(vMetadata storage.Store) {
	c.vMetadata = vMetadata
}

// SetMetadata sets the context storage for metadata, other than versioning metadata
func (c *Stores) SetMetadata(metadata storage.Store) {
	c.metadata = metadata
}

// SetBlob sets the context storage for blobs
func (c *Stores) SetBlob(blob storage.Store) {
	c.blob = blob
}

// SetWal sets the context storage for Write Ahead Log
func (c *Stores) SetWal(wal storage.Store) {
	c.wal = wal
}

// Metadata yields the metadata storage for a context
func (c *Stores) Metadata() storage.Store {
	return c.metadata
}

// VMetadata yields the version metadata storage for a context
func (c *Stores) VMetadata() storage.Store {
	return c.vMetadata
}

// Blob yields the Blob storage for a context
func (c *Stores) Blob() storage.Store {
	return c.blob
}

// Wal yields the Write Ahead Log storage for a context
func (c *Stores) Wal() storage.Store {
	return c.wal
}

// CreateContext marshals and persists a context
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

// GetContext downloads and unmarshals a context
func GetContext(ctx context.Context, configStore storage.Store, contextName string,
) (context *model.Context, err error) {
	rdr, err := configStore.Get(ctx, model.GetPathToContext(contextName))
	if err != nil {
		return context, err
	}
	bytes, err := ioutil.ReadAll(rdr)
	if err != nil {
		return
	}
	context, err = model.UnmarshalContext(bytes)
	if err != nil {
		return
	}
	err = model.ValidateContext(*context)
	if err != nil {
		return
	}
	return context, nil
}
