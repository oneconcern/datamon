// Copyright © 2018 One Concern

// Package gcs implements datamon Store for Google GCS
package gcs

import (
	"context"
	"io"

	"google.golang.org/api/iterator"

	gcsStorage "cloud.google.com/go/storage"

	"github.com/oneconcern/datamon/pkg/storage"
	storagestatus "github.com/oneconcern/datamon/pkg/storage/status"
	"google.golang.org/api/option"
)

type gcs struct {
	client         *gcsStorage.Client
	readOnlyClient *gcsStorage.Client
	bucket         string
	ctx            context.Context
}

func clientOpts(readOnly bool, credentialFile string) []option.ClientOption {
	opts := make([]option.ClientOption, 0, 2)
	if readOnly {
		opts = append(opts, option.WithScopes(gcsStorage.ScopeReadOnly))
	} else {
		opts = append(opts, option.WithScopes(gcsStorage.ScopeFullControl))
	}
	if credentialFile != "" {
		opts = append(opts, option.WithCredentialsFile(credentialFile))
	}
	return opts
}

// New builds a new storage object from a bucket string
func New(ctx context.Context, bucket string, credentialFile string) (storage.Store, error) {
	googleStore := new(gcs)
	googleStore.ctx = ctx
	googleStore.bucket = bucket

	var err error
	googleStore.readOnlyClient, err = gcsStorage.NewClient(ctx, clientOpts(true, credentialFile)...)
	if err != nil {
		return nil, toSentinelErrors(err)
	}
	googleStore.client, err = gcsStorage.NewClient(ctx, clientOpts(false, credentialFile)...)
	if err != nil {
		return nil, toSentinelErrors(err)
	}
	return googleStore, nil
}

func (g *gcs) String() string {
	return "gcs://" + g.bucket
}

func (g *gcs) Has(ctx context.Context, objectName string) (bool, error) {
	client := g.readOnlyClient
	_, err := client.Bucket(g.bucket).Object(objectName).Attrs(ctx)
	if err != nil {
		if err == gcsStorage.ErrObjectNotExist {
			return false, nil
		}
		return false, toSentinelErrors(err)
	}
	return true, nil
}

type gcsReader struct {
	g            *gcs
	objectName   string
	objectReader io.ReadCloser
}

func (r *gcsReader) WriteTo(writer io.Writer) (n int64, err error) {
	return storage.PipeIO(writer, r.objectReader)
}

func (r *gcsReader) Close() error {
	return r.objectReader.Close()
}

func (r *gcsReader) Read(p []byte) (n int, err error) {
	read, err := r.objectReader.Read(p)
	return read, toSentinelErrors(err)
}

func (r *gcsReader) ReadAt(p []byte, offset int64) (n int, err error) {
	objectReader, err := r.g.readOnlyClient.Bucket(r.g.bucket).Object(r.objectName).NewRangeReader(
		r.g.ctx, offset, int64(len(p)))
	if err != nil {
		return 0, toSentinelErrors(err)
	}
	return objectReader.Read(p)
}

func (g *gcs) Get(ctx context.Context, objectName string) (io.ReadCloser, error) {
	objectReader, err := g.readOnlyClient.Bucket(g.bucket).Object(objectName).NewReader(ctx)
	if err != nil {
		return nil, toSentinelErrors(err)
	}
	return &gcsReader{
		g:            g,
		objectReader: objectReader,
	}, nil
}

func (g *gcs) GetAttr(ctx context.Context, objectName string) (storage.Attributes, error) {
	attr, err := g.readOnlyClient.Bucket(g.bucket).Object(objectName).Attrs(ctx)
	if err != nil {
		return storage.Attributes{}, toSentinelErrors(err)
	}
	return storage.Attributes{
		Created: attr.Created,
		Updated: attr.Updated,
		Owner:   attr.Owner,
	}, nil
}

func (g *gcs) GetAt(ctx context.Context, objectName string) (io.ReaderAt, error) {
	return &gcsReader{
		g:          g,
		objectName: objectName,
	}, nil
}

func (g *gcs) Touch(ctx context.Context, objectName string) error {
	_, err := g.client.Bucket(g.bucket).Object(objectName).Update(ctx, gcsStorage.ObjectAttrsToUpdate{})
	return toSentinelErrors(err)
}

type readCloser struct {
	reader io.Reader
}

func (rc readCloser) Read(p []byte) (n int, err error) {
	return rc.reader.Read(p)
}
func (rc readCloser) Close() error {
	return nil
}

func (g *gcs) Put(ctx context.Context, objectName string, reader io.Reader, newObject storage.NewKey) error {
	// Put if not present
	var writer *gcsStorage.Writer
	b := false
	if newObject {
		b = true
	}
	if newObject {
		writer = g.client.Bucket(g.bucket).Object(objectName).If(gcsStorage.Conditions{
			DoesNotExist: b,
		}).NewWriter(ctx)
	} else {
		writer = g.client.Bucket(g.bucket).Object(objectName).NewWriter(ctx)
	}
	_, err := storage.PipeIO(writer, readCloser{reader: reader})
	if err != nil {
		return toSentinelErrors(err)
	}
	return toSentinelErrors(writer.Close())
}

func (g *gcs) PutCRC(ctx context.Context, objectName string, reader io.Reader, doesNotExist bool, crc uint32) error {
	// Put if not present
	var writer *gcsStorage.Writer
	if doesNotExist {
		writer = g.client.Bucket(g.bucket).Object(objectName).If(gcsStorage.Conditions{
			DoesNotExist: doesNotExist,
		}).NewWriter(ctx)
	} else {
		writer = g.client.Bucket(g.bucket).Object(objectName).NewWriter(ctx)
	}
	writer.CRC32C = crc
	_, err := storage.PipeIO(writer, readCloser{reader: reader})
	if err != nil {
		return toSentinelErrors(err)
	}
	return toSentinelErrors(writer.Close())
}

func (g *gcs) Delete(ctx context.Context, objectName string) error {
	return toSentinelErrors(g.client.Bucket(g.bucket).Object(objectName).Delete(ctx))
}

// TODO: Sent error if more than a million keys. Use KeysPrefix API.
func (g *gcs) Keys(ctx context.Context) ([]string, error) {
	const keysPerQuery = 1000000
	var pageToken string
	nextPageToken := "sentinel" /* could be any nonempty string to start */
	keys := make([]string, 0)
	for nextPageToken != "" {
		var keysCurr []string
		var err error
		keysCurr, nextPageToken, err = g.KeysPrefix(ctx, pageToken, "", "", keysPerQuery)
		if err != nil {
			return nil, toSentinelErrors(err)
		}
		keys = append(keys, keysCurr...)
		pageToken = nextPageToken
	}
	return keys, nil
}

func (g *gcs) KeysPrefix(ctx context.Context, pageToken string, prefix string, delimiter string, count int) ([]string, string, error) {
	itr := g.readOnlyClient.Bucket(g.bucket).Objects(ctx, &gcsStorage.Query{Prefix: prefix, Delimiter: delimiter})

	var objects []*gcsStorage.ObjectAttrs

	keys := make([]string, 0, count)
	pageToken, err := iterator.NewPager(itr, count, pageToken).NextPage(&objects)
	if err != nil {
		return nil, "", toSentinelErrors(err)
	}

	for _, objAttrs := range objects {
		if objAttrs.Prefix != "" {
			keys = append(keys, objAttrs.Prefix)
		} else {
			keys = append(keys, objAttrs.Name)
		}
	}

	return keys, pageToken, nil
}

func (g *gcs) Clear(context.Context) error {
	return storagestatus.ErrNotImplemented
}
