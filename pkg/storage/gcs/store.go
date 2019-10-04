// Copyright © 2018 One Concern

package gcs

import (
	"context"
	"errors"
	"io"
	"strings"

	"google.golang.org/api/iterator"

	gcsStorage "cloud.google.com/go/storage"

	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/oneconcern/datamon/pkg/storage/status"
	"google.golang.org/api/option"
)

type gcs struct {
	client         *gcsStorage.Client
	readOnlyClient *gcsStorage.Client
	bucket         string
	ctx            context.Context
}

func New(ctx context.Context, bucket string, credentialFile string) (storage.Store, error) {
	googleStore := new(gcs)
	googleStore.ctx = ctx
	googleStore.bucket = bucket
	var err error
	if credentialFile != "" {
		c := option.WithCredentialsFile(credentialFile)
		googleStore.readOnlyClient, err = gcsStorage.NewClient(ctx, option.WithScopes(gcsStorage.ScopeReadOnly), c)
	} else {
		googleStore.readOnlyClient, err = gcsStorage.NewClient(ctx, option.WithScopes(gcsStorage.ScopeReadOnly))
	}
	if err != nil {
		return nil, err
	}
	if credentialFile != "" {
		c := option.WithCredentialsFile(credentialFile)
		googleStore.client, err = gcsStorage.NewClient(ctx, option.WithScopes(gcsStorage.ScopeFullControl), c)
	} else {
		googleStore.client, err = gcsStorage.NewClient(ctx, option.WithScopes(gcsStorage.ScopeFullControl))
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
	client := g.readOnlyClient
	_, err := client.Bucket(g.bucket).Object(objectName).Attrs(ctx)
	if err != nil {
		if err == gcsStorage.ErrObjectNotExist {
			return false, nil
		}
		return false, err
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
	return read, err
}

func (r *gcsReader) ReadAt(p []byte, offset int64) (n int, err error) {
	objectReader, err := r.g.readOnlyClient.Bucket(r.g.bucket).Object(r.objectName).NewRangeReader(
		r.g.ctx, offset, int64(len(p)))
	if err != nil {
		return 0, err
	}
	return objectReader.Read(p)
}

func toSentinelErrors(err error) error {
	// return sentinel errors defined by the status package
	if strings.Contains(err.Error(), "object doesn't exist") {
		return status.ErrNotExists
	}
	return err
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

func (g *gcs) GetAt(ctx context.Context, objectName string) (io.ReaderAt, error) {
	return &gcsReader{
		g:          g,
		objectName: objectName,
	}, nil
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

func (g *gcs) Put(ctx context.Context, objectName string, reader io.Reader, doesNotExist bool) error {
	// Put if not present
	var writer *gcsStorage.Writer
	if doesNotExist {
		writer = g.client.Bucket(g.bucket).Object(objectName).If(gcsStorage.Conditions{
			DoesNotExist: doesNotExist,
		}).NewWriter(ctx)
	} else {
		writer = g.client.Bucket(g.bucket).Object(objectName).NewWriter(ctx)
	}
	_, err := storage.PipeIO(writer, readCloser{reader: reader})
	if err != nil {
		return err
	}
	return writer.Close()
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
		return err
	}
	return writer.Close()
}

func (g *gcs) Delete(ctx context.Context, objectName string) error {
	return g.client.Bucket(g.bucket).Object(objectName).Delete(ctx)
}

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
			return nil, err
		}
		keys = append(keys, keysCurr...)
		pageToken = nextPageToken
	}
	return keys, nil
}

// todo: consider KeysPrefix returning all keys, not pages,
//   and expanding api surface to include paged results for both keys and keys according to prefix
func (g *gcs) KeysPrefix(ctx context.Context, pageToken string, prefix string, delimiter string, count int) ([]string, string, error) {
	itr := g.readOnlyClient.Bucket(g.bucket).Objects(ctx, &gcsStorage.Query{Prefix: prefix, Delimiter: delimiter})

	var objects []*gcsStorage.ObjectAttrs

	keys := make([]string, 0, count)
	pageToken, err := iterator.NewPager(itr, count, pageToken).NextPage(&objects)
	if err != nil {
		return nil, "", err
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
	return errors.New("unimplemented")
}
