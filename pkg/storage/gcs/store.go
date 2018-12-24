// Copyright © 2018 One Concern

package gcs

import (
	"context"
	"errors"

	gcsStorage "cloud.google.com/go/storage"

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

func (g *gcs) Has(ctx context.Context, objectName string) (bool, error) {
	client, err := gcsStorage.NewClient(ctx, option.WithScopes(gcsStorage.ScopeReadOnly))
	if err != nil {
		return false, err
	}
	_, err = client.Bucket(g.bucket).Object(objectName).Attrs(ctx)
	if err != nil {
		if err != gcsStorage.ErrObjectNotExist {
			return false, nil
		}
		return false, err
	}
	return true, nil
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

func (g *gcs) Put(ctx context.Context, objectName string, reader io.Reader) error {
	client, err := gcsStorage.NewClient(ctx, option.WithScopes(gcsStorage.ScopeReadWrite))
	if err != nil {
		return err
	}

	// Put if not present
	writer := client.Bucket(g.bucket).Object(objectName).Generation(0).NewWriter(ctx)
	_, err = io.Copy(writer, reader)
	if err != nil {
		return err
	}
	return writer.Close()
}

func (g *gcs) Delete(ctx context.Context, objectName string) error {
	client, err := gcsStorage.NewClient(ctx, option.WithScopes(gcsStorage.DeleteAction))
	if err != nil {
		return err
	}
	return client.Bucket(g.bucket).Object(objectName).Delete(ctx)
}

func (g *gcs) Keys(ctx context.Context) ([]string, error) {
	client, err := gcsStorage.NewClient(ctx, option.WithScopes(gcsStorage.ScopeReadOnly))
	if err != nil {
		return nil, err
	}
	objectsIterator := client.Bucket(g.bucket).Objects(ctx, nil)
	objectsIterator.Next()
	return nil, nil
}

func (g *gcs) Clear(context.Context) error {
	return errors.New("unimplemented")
}
