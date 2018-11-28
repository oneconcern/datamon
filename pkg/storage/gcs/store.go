// Copyright © 2018 One Concern

package gcs

import (
	gcsStorage "cloud.google.com/go/storage"

	"context"
	"errors"

	"github.com/oneconcern/datamon/pkg/storage"
	"google.golang.org/api/option"

	"io"
)

type gcs struct {
	bucket string
}

func New(bucket string) storage.Store {
	googleStore := new(gcs)
	googleStore.bucket = bucket
	return googleStore
}

func (g *gcs) String() string {
	return ""
}

func (g *gcs) Has(context.Context, string) (bool, error) {
	return false, errors.New("unimplemented")
}

func (g *gcs) Get(ctx context.Context, objectName string) (io.ReadCloser, error) {
	client, err := gcsStorage.NewClient(ctx, option.WithScopes(gcsStorage.ScopeReadOnly))
	if err != nil {
		return nil, err
	}
	objectReader, err := client.Bucket(g.bucket).Object(objectName).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	return objectReader, nil
}

func (g *gcs) Put(context.Context, string, io.Reader) error {
	return errors.New("unimplemented")
}

func (g *gcs) Delete(context.Context, string) error {
	return errors.New("unimplemented")
}

func (g *gcs) Keys(context.Context) ([]string, error) {
	return nil, errors.New("unimplemented")
}

func (g *gcs) Clear(context.Context) error {
	return errors.New("unimplemented")
}
