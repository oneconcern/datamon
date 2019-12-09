package gcs

import (
	"context"

	context2 "github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/context/status"
	"github.com/oneconcern/datamon/pkg/model"
	gcsstore "github.com/oneconcern/datamon/pkg/storage/gcs"
)

// MakeContext initializes all gcs stores in a context described by its model, with some gcs credentials
func MakeContext(ctx context.Context, descriptor model.Context, creds string) (context2.Stores, error) {
	stores := context2.New()

	meta, err := gcsstore.New(ctx, descriptor.Metadata, creds)
	if err != nil {
		return nil, status.ErrInitMetadata.Wrap(err)
	}
	stores.SetMetadata(meta)

	blob, err := gcsstore.New(ctx, descriptor.Blob, creds)
	if err != nil {
		return nil, status.ErrInitBlob.Wrap(err)
	}
	stores.SetBlob(blob)

	v, err := gcsstore.New(ctx, descriptor.VMetadata, creds)
	if err != nil {
		return nil, status.ErrInitVMetadata.Wrap(err)
	}
	stores.SetVMetadata(v)

	w, err := gcsstore.New(ctx, descriptor.WAL, creds)
	if err != nil {
		return nil, status.ErrInitWAL.Wrap(err)
	}
	stores.SetWal(w)

	r, err := gcsstore.New(ctx, descriptor.ReadLog, creds)
	if err != nil {
		return nil, status.ErrInitRLog.Wrap(err)
	}
	stores.SetReadLog(r)

	return stores, nil
}
