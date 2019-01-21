// Copyright © 2018 One Concern

package gcs

import (
	gcsStorage "cloud.google.com/go/storage"
	"context"
	"errors"
	"google.golang.org/api/iterator"

	"github.com/oneconcern/datamon/pkg/storage"
	"google.golang.org/api/option"

	"io"
)

const PageSize  = 1000

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

	object := client.Bucket(g.bucket).Object(objectName)
	_, err = object.Attrs(ctx)
	if err == gcsStorage.ErrObjectNotExist {
		return false, nil
	}

	if err != nil {
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

func (g *gcs) Put(ctx context.Context, key string, rdr io.Reader) error {
	client, err := gcsStorage.NewClient(ctx, option.WithScopes(gcsStorage.ScopeReadWrite))
	if err != nil{
		return err
	}

	wc := client.Bucket(g.bucket).Object(key).NewWriter(ctx)
	if _, err = io.Copy(wc, rdr); err != nil {
		return err
	}

	if err := wc.Close(); err != nil {
		return err
	}

	return nil
}

func (g *gcs) Delete(ctx context.Context, key string) error {
	client, err := gcsStorage.NewClient(ctx, option.WithScopes(gcsStorage.ScopeReadWrite))
	if err != nil{
		return err
	}
	object := client.Bucket(g.bucket).Object(key)
	if err := object.Delete(ctx); err != nil {
		return err
	}

	return nil
}

func (g *gcs) Keys(ctx context.Context) ([]string, error) {
	return nil, errors.New("unimplemented")
}

func (g *gcs) KeysPrefix(ctx context.Context , pageToken, prefix, delimiter string) ([]string, string, error) {
	client, err := gcsStorage.NewClient(ctx, option.WithScopes(gcsStorage.ScopeReadOnly))
	if err != nil{
		return nil, "", err
	}

	itr := client.Bucket(g.bucket).Objects(ctx, &gcsStorage.Query{Prefix: prefix, Delimiter: delimiter})

	var objects []*gcsStorage.ObjectAttrs

	var buckets []string
	pageToken, err = iterator.NewPager(itr, PageSize, pageToken).NextPage(&objects)

	for _, objAttrs := range  objects {
		buckets = append(buckets, objAttrs.Name)
	}

	return buckets, pageToken,  err

}

func (g *gcs) Clear(context.Context) error {
	return errors.New("unimplemented")
}
