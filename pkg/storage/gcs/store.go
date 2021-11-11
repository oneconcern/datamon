// Copyright © 2018 One Concern

// Package gcs implements datamon Store for Google GCS
package gcs

import (
	"context"
	"io"
	"time"

	"go.uber.org/zap"
	"google.golang.org/api/iterator"

	gcsStorage "cloud.google.com/go/storage"

	"github.com/cenkalti/backoff/v4"
	"github.com/oneconcern/datamon/pkg/dlogger"
	"github.com/oneconcern/datamon/pkg/storage"
	storagestatus "github.com/oneconcern/datamon/pkg/storage/status"
	"google.golang.org/api/option"
)

var (
	_ storage.Store    = &gcs{}
	_ storage.StoreCRC = &gcs{}
)

type gcs struct {
	client         *gcsStorage.Client
	readOnlyClient *gcsStorage.Client
	bucket         string
	keyPrefix      string
	ctx            context.Context
	l              *zap.Logger
	isReadOnly     bool
	retry          bool
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
func New(ctx context.Context, bucket string, credentialFile string, opts ...Option) (storage.Store, error) {
	googleStore := new(gcs)
	googleStore.retry = true

	for _, apply := range opts {
		apply(googleStore)
	}
	if googleStore.l == nil {
		// default logger if none provided by options
		googleStore.l, _ = dlogger.GetLogger("info")
	}
	googleStore.l = googleStore.l.With(zap.String("bucket", bucket))
	googleStore.ctx = ctx
	googleStore.bucket = bucket

	var err error
	googleStore.readOnlyClient, err = gcsStorage.NewClient(ctx, clientOpts(true, credentialFile)...)
	if err != nil {
		return nil, toSentinelErrors(err)
	}
	if !googleStore.isReadOnly {
		googleStore.client, err = gcsStorage.NewClient(ctx, clientOpts(false, credentialFile)...)
		if err != nil {
			return nil, toSentinelErrors(err)
		}
	} else {
		// if ReadOnly option, the "write" client is gained with readOnly scope:
		// write / delete operations will fail
		googleStore.client = googleStore.readOnlyClient
	}
	return googleStore, nil
}

func (g *gcs) String() string {
	return "gcs://" + g.bucket + g.keyPrefix
}

// Has this object in the store?
func (g *gcs) Has(ctx context.Context, objectName string) (bool, error) {
	client := g.readOnlyClient
	_, err := client.Bucket(g.bucket).Object(g.keyPrefix + objectName).Attrs(ctx)
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
	l            *zap.Logger
}

func (r *gcsReader) WriteTo(writer io.Writer) (n int64, err error) {
	return storage.PipeIO(writer, r.objectReader)
}

func (r *gcsReader) Close() error {
	return r.objectReader.Close()
}

func (r *gcsReader) Read(p []byte) (n int, err error) {
	r.l.Debug("Start Read", zap.Int("chunk size", len(p)))
	defer func() {
		r.l.Debug("End Read", zap.Int("chunk size", len(p)), zap.Int("bytes read", n), zap.Error(err))
	}()
	read, err := r.objectReader.Read(p)
	return read, toSentinelErrors(err)
}

func (r *gcsReader) ReadAt(p []byte, offset int64) (n int, err error) {
	r.l.Debug("Start ReadAt", zap.Int("chunk size", len(p)), zap.Int64("offset", offset))
	defer func() {
		r.l.Debug("End ReadAt", zap.Int("chunk size", len(p)), zap.Int64("offset", offset), zap.Int("bytes read", n), zap.Error(err))
	}()
	objectReader, err := r.g.readOnlyClient.Bucket(r.g.bucket).Object(r.g.keyPrefix+r.objectName).NewRangeReader(
		r.g.ctx, offset, int64(len(p)))
	if err != nil {
		return 0, toSentinelErrors(err)
	}
	return objectReader.Read(p)
}

func (g *gcs) Get(ctx context.Context, objectName string) (io.ReadCloser, error) {
	g.l.Debug("Start Get", zap.String("objectName", objectName))
	objectReader, err := g.readOnlyClient.Bucket(g.bucket).Object(g.keyPrefix + objectName).NewReader(ctx)
	g.l.Debug("End Get", zap.String("objectName", objectName), zap.Error(err))
	if err != nil {
		return nil, toSentinelErrors(err)
	}
	return &gcsReader{
		g:            g,
		objectReader: objectReader,
		l:            g.l,
	}, nil
}

func (g *gcs) GetAttr(ctx context.Context, objectName string) (storage.Attributes, error) {
	g.l.Debug("Start GetAttr", zap.String("objectName", objectName))
	attr, err := g.readOnlyClient.Bucket(g.bucket).Object(g.keyPrefix + objectName).Attrs(ctx)
	g.l.Debug("End GetAttr", zap.String("objectName", objectName), zap.Error(err))
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
		l:          g.l,
	}, nil
}

func (g *gcs) Touch(ctx context.Context, objectName string) error {
	g.l.Debug("Start Touch", zap.String("objectName", objectName))
	_, err := g.client.Bucket(g.bucket).Object(g.keyPrefix+objectName).Update(ctx, gcsStorage.ObjectAttrsToUpdate{})
	g.l.Debug("End touch", zap.String("objectName", objectName), zap.Error(err))
	return toSentinelErrors(err)
}

type readCloser struct {
	reader io.Reader
}

func (rc readCloser) Read(p []byte) (int, error) {
	return rc.reader.Read(p)
}

func (rc readCloser) Close() error {
	return nil
}

func (g *gcs) Put(ctx context.Context, objectName string, reader io.Reader, newObject bool) (err error) {
	return g.putObject(ctx, g.keyPrefix+objectName, reader, newObject, false, 0)
}

func (g *gcs) PutCRC(ctx context.Context, objectName string, reader io.Reader, newObject bool, crc uint32) (err error) {
	return g.putObject(ctx, g.keyPrefix+objectName, reader, newObject, true, crc)
}

func (g *gcs) putObject(ctx context.Context, objectName string, reader io.Reader, newObject bool, isPutCRC bool, crc uint32) (err error) {
	var (
		retryPolicy backoff.BackOff
		writer      *gcsStorage.Writer
	)

	g.l.Debug("Start Put", zap.String("objectName", objectName))
	defer func() {
		g.l.Debug("End Put", zap.String("objectName", objectName), zap.Error(err))
	}()

	if g.retry {
		r := backoff.NewExponentialBackOff()
		r.MaxElapsedTime = 30 * time.Second
		r.Reset()
		retryPolicy = r
	} else {
		retryPolicy = &backoff.StopBackOff{}
	}

	gcsObject := g.client.Bucket(g.bucket).Object(g.keyPrefix + objectName)

	// wrapping PipeIO execution so it can be retried
	operation := func() error {
		if newObject {
			gcsObject = gcsObject.If(gcsStorage.Conditions{DoesNotExist: newObject})
		}

		writer = gcsObject.NewWriter(ctx)

		if isPutCRC {
			writer.CRC32C = crc
		}

		_, err = storage.PipeIO(writer, readCloser{reader: reader})
		if err != nil {
			g.l.Error("write error, retrying",
				zap.String("objectName", objectName),
				zap.Error(err),
			)
		}

		err = writer.Close()
		if err != nil {
			g.l.Error("write error, retrying",
				zap.String("objectName", objectName),
				zap.Error(err),
			)
		}

		return err
	}
	err = backoff.Retry(operation, retryPolicy)
	if err != nil {
		return toSentinelErrors(err)
	}

	return nil
}

func (g *gcs) Delete(ctx context.Context, objectName string) (err error) {
	g.l.Debug("Start Delete", zap.String("objectName", objectName))
	err = toSentinelErrors(g.client.Bucket(g.bucket).Object(g.keyPrefix + objectName).Delete(ctx))
	g.l.Debug("End Delete", zap.String("objectName", objectName), zap.Error(err))
	return
}

// Keys returns all the keys known to a store
//
// TODO: Send an error if more than a million keys. Use KeysPrefix API.
func (g *gcs) Keys(ctx context.Context) (keys []string, err error) {
	g.l.Debug("Start Keys")
	defer func() {
		g.l.Debug("End Keys", zap.Int("keys", len(keys)), zap.Error(err))
	}()
	const keysPerQuery = 1000000
	var pageToken string
	nextPageToken := "sentinel" /* could be any nonempty string to start */
	keys = make([]string, 0)
	for nextPageToken != "" {
		var keysCurr []string
		keysCurr, nextPageToken, err = g.KeysPrefix(ctx, pageToken, "", "", keysPerQuery)
		if err != nil {
			return nil, toSentinelErrors(err)
		}
		keys = append(keys, keysCurr...)
		pageToken = nextPageToken
	}
	return keys, nil
}

func (g *gcs) KeysPrefix(ctx context.Context, pageToken string, prefix string, delimiter string, count int) (keys []string, next string, err error) {
	g.l.Debug("Start KeysPrefix", zap.String("start", pageToken), zap.String("prefix", prefix))
	defer func() {
		g.l.Debug("End KeysPrefix", zap.String("start", pageToken), zap.String("prefix", prefix), zap.Int("keys", len(keys)), zap.Error(err))
	}()
	itr := g.readOnlyClient.Bucket(g.bucket).Objects(ctx, &gcsStorage.Query{Prefix: g.keyPrefix + prefix, Delimiter: delimiter})

	var objects []*gcsStorage.ObjectAttrs

	keys = make([]string, 0, count)
	next, err = iterator.NewPager(itr, count, pageToken).NextPage(&objects)
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
	return
}

func (g *gcs) Clear(context.Context) error {
	return storagestatus.ErrNotImplemented
}
