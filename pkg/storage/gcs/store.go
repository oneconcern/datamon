// Copyright © 2018 One Concern

package gcs

import (
	"context"
	"errors"

	"google.golang.org/api/iterator"

	gcsStorage "cloud.google.com/go/storage"

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

func New(bucket string, credentialFile string) (storage.Store, error) {
	googleStore := new(gcs)
	googleStore.bucket = bucket
	var err error
	if credentialFile != "" {
		c := option.WithCredentialsFile(credentialFile)
		googleStore.readOnlyClient, err = gcsStorage.NewClient(context.TODO(), option.WithScopes(gcsStorage.ScopeReadOnly), c)
	} else {
		googleStore.readOnlyClient, err = gcsStorage.NewClient(context.TODO(), option.WithScopes(gcsStorage.ScopeReadOnly))
	}
	if err != nil {
		return nil, err
	}
	if credentialFile != "" {
		c := option.WithCredentialsFile(credentialFile)
		googleStore.client, err = gcsStorage.NewClient(context.TODO(), option.WithScopes(gcsStorage.ScopeFullControl), c)
	} else {
		googleStore.client, err = gcsStorage.NewClient(context.TODO(), option.WithScopes(gcsStorage.ScopeFullControl))
	}
	if err != nil {
		return nil, err
	}
	return googleStore, err
}

func (g *gcs) String() string {
	return "gcs://" + g.bucket
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

type gcsReader struct {
	objectReader io.ReadCloser
}

func (r *gcsReader) WriteTo(writer io.Writer) (n int64, err error) {
	return storage.WriteTo(writer, r.objectReader)
}

func (r gcsReader) Close() error {
	return r.objectReader.Close()
}

func (r gcsReader) Read(p []byte) (n int, err error) {
	return r.objectReader.Read(p)
}

func (g *gcs) Get(ctx context.Context, objectName string) (io.ReadCloser, error) {
	objectReader, err := g.readOnlyClient.Bucket(g.bucket).Object(objectName).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	return gcsReader{
		objectReader: objectReader,
	}, nil
}

func (g *gcs) Put(ctx context.Context, objectName string, reader io.Reader, doesNotExist bool) error {
	// Put if not present
	var writer *gcsStorage.Writer
	if doesNotExist {
		writer = g.client.Bucket(g.bucket).Object(objectName).If(gcsStorage.Conditions{DoesNotExist: doesNotExist}).NewWriter(ctx)
	} else {
		writer = g.client.Bucket(g.bucket).Object(objectName).NewWriter(ctx)

	}
	_, err := io.Copy(writer, reader)
	if err != nil {
		return err
	}
	return writer.Close()
}

func (g *gcs) Delete(ctx context.Context, objectName string) error {
	return g.client.Bucket(g.bucket).Object(objectName).Delete(ctx)
}

func (g *gcs) Keys(ctx context.Context) ([]string, error) {
	keys, _, err := g.KeysPrefix(ctx, "", "", "", 0)
	if err != nil {
		return nil, err
	}

	return keys, nil
}

func (g *gcs) KeysPrefix(ctx context.Context, pageToken string, prefix string, delimiter string, count int) ([]string, string, error) {

	itr := g.readOnlyClient.Bucket(g.bucket).Objects(ctx, &gcsStorage.Query{Prefix: prefix, Delimiter: delimiter})

	var objects []*gcsStorage.ObjectAttrs

	keys := make([]string, 0, PageSize)
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
