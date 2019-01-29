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

const PageSize = 1000

type gcs struct {
	client         *gcsStorage.Client
	readOnlyClient *gcsStorage.Client
	bucket         string
}

func New(bucket string) (storage.Store, error) {
	googleStore := new(gcs)
	googleStore.bucket = bucket
	var err error
	googleStore.readOnlyClient, err = gcsStorage.NewClient(context.TODO(), option.WithScopes(gcsStorage.ScopeReadOnly))
	if err != nil {
		return nil, err
	}
	googleStore.client, err = gcsStorage.NewClient(context.TODO(), option.WithScopes(gcsStorage.ScopeFullControl))
	if err != nil {
		return nil, err
	}
	return googleStore, err
}

func (g *gcs) String() string {
	return "gcs://" + g.bucket
}

func (g *gcs) Has(ctx context.Context, objectName string) (bool, error) {

	_, err := g.readOnlyClient.Bucket(g.bucket).Object(objectName).Attrs(ctx)
	if err != nil {
		if err != gcsStorage.ErrObjectNotExist {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (g *gcs) Get(ctx context.Context, objectName string) (io.ReadCloser, error) {
	objectReader, err := g.readOnlyClient.Bucket(g.bucket).Object(objectName).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	return objectReader, nil
}

func (g *gcs) Put(ctx context.Context, objectName string, reader io.Reader) error {
	// Put if not present
	writer := g.client.Bucket(g.bucket).Object(objectName).If(gcsStorage.Conditions{DoesNotExist: true}).NewWriter(ctx)
	_, err := io.Copy(writer, reader)
	if err != nil {
		return err
	}
	return writer.Close()
}

func (g *gcs) Delete(ctx context.Context, objectName string) error {
	return g.client.Bucket(g.bucket).Object(objectName).Delete(ctx)
}

func (g *gcs) Keys(ctx context.Context, token string) ([]string, string, error) {
	keys, token, err := g.KeysPrefix(ctx, token, "", "")
	if err != nil {
		return nil, "", err
	}

	return keys, token, nil
}

func (g *gcs) KeysPrefix(ctx context.Context, pageToken, prefix, delimiter string) ([]string, string, error) {

	itr := g.readOnlyClient.Bucket(g.bucket).Objects(ctx, &gcsStorage.Query{Prefix: prefix, Delimiter: delimiter})

	var objects []*gcsStorage.ObjectAttrs

	var keys []string
	pageToken, err := iterator.NewPager(itr, PageSize, pageToken).NextPage(&objects)
	if err != nil {
		return nil, "", err
	}

	for _, objAttrs := range objects {
		keys = append(keys, objAttrs.Name)
	}

	return keys, pageToken, nil

}

func (g *gcs) Clear(context.Context) error {
	return errors.New("unimplemented")
}

func (g *gcs) GetAt(ctx context.Context, objectName string) (io.ReaderAt, error) {
	return nil, errors.New("unimplemented")
}
