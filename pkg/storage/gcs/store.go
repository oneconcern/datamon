// Copyright © 2018 One Concern

// Package gcs implements datamon Store for Google GCS
package gcs

import (
	"context"
	"io"

	"go.uber.org/zap"
	"google.golang.org/api/iterator"

	gcsStorage "cloud.google.com/go/storage"

	"github.com/oneconcern/datamon/pkg/dlogger"
	"github.com/oneconcern/datamon/pkg/errors"
	"github.com/oneconcern/datamon/pkg/storage"
	storagestatus "github.com/oneconcern/datamon/pkg/storage/status"
	"google.golang.org/api/option"
)

type gcs struct {
	client         *gcsStorage.Client
	readOnlyClient *gcsStorage.Client
	bucket         string
	ctx            context.Context
	l              *zap.Logger
	s              storage.Settings
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

// versionedObject wraps *BucketHandle.Object
// towards canonicalization of object names.
func versionedObject(
	bucketHandle *gcsStorage.BucketHandle,
	objectName string,
) *gcsStorage.ObjectHandle {
	return bucketHandle.Object(objectName)
}

// implNew provides stricter typing than New for package internal tests.
// Akin to the impl* functions in pkg/core/bundle.go, this function is in place
// in order to be extended with additional parameters and/or return values
// independently of the public interface provided by New.
func implNew(ctx context.Context, bucket string, credentialFile string, opts ...Option) (*gcs, error) {
	googleStore := new(gcs)
	for _, apply := range opts {
		apply(googleStore)
	}
	{
		generationNumber := googleStore.s.Version.GcsVersion()
		if generationNumber < 0 && generationNumber != storage.GcsSentinelVersion {
			return nil, errors.New("invalid version")
		}
	}
	if googleStore.l == nil {
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
	googleStore.client, err = gcsStorage.NewClient(ctx, clientOpts(false, credentialFile)...)
	if err != nil {
		return nil, toSentinelErrors(err)
	}
	return googleStore, nil
}

// New builds a new storage object from a bucket string
func New(ctx context.Context, bucket string, credentialFile string, opts ...Option) (storage.Store, error) {
	return implNew(ctx, bucket, credentialFile, opts...)
}

func (g *gcs) String() string {
	return "gcs://" + g.bucket
}

// Has this object in the store?
func (g *gcs) Has(ctx context.Context, objectName string) (bool, error) {
	client := g.readOnlyClient
	_, err := versionedObject(client.Bucket(g.bucket), objectName).Attrs(ctx)
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
	objectReader, err := versionedObject(r.g.readOnlyClient.Bucket(r.g.bucket), r.objectName).
		NewRangeReader(r.g.ctx, offset, int64(len(p)))
	if err != nil {
		return 0, toSentinelErrors(err)
	}
	return objectReader.Read(p)
}

func (g *gcs) Get(ctx context.Context, objectName string) (io.ReadCloser, error) {
	g.l.Debug("Start Get", zap.String("objectName", objectName))
	objectReader, err := versionedObject(g.readOnlyClient.Bucket(g.bucket), objectName).NewReader(ctx)
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
	attr, err := versionedObject(g.readOnlyClient.Bucket(g.bucket), objectName).Attrs(ctx)
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
	_, err := versionedObject(g.client.Bucket(g.bucket), objectName).
		Update(ctx, gcsStorage.ObjectAttrsToUpdate{})
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
	g.l.Debug("Start Put", zap.String("objectName", objectName))
	defer func() {
		g.l.Debug("End Put", zap.String("objectName", objectName), zap.Error(err))
	}()
	// Put if not present
	var writer *gcsStorage.Writer
	b := false
	if newObject {
		b = true
	}
	if newObject {
		writer = versionedObject(g.client.Bucket(g.bucket), objectName).If(gcsStorage.Conditions{
			DoesNotExist: b,
		}).NewWriter(ctx)
	} else {
		writer = versionedObject(g.client.Bucket(g.bucket), objectName).NewWriter(ctx)
	}
	g.l.Debug("Start Put PipeIO", zap.String("objectName", objectName))
	_, err = storage.PipeIO(writer, readCloser{reader: reader})
	g.l.Debug("End Put PipeIO", zap.String("objectName", objectName), zap.Error(err))
	if err != nil {
		return toSentinelErrors(err)
	}
	err = writer.Close()
	return toSentinelErrors(err)
}

func (g *gcs) PutCRC(ctx context.Context, objectName string, reader io.Reader, doesNotExist bool, crc uint32) (err error) {
	g.l.Debug("Start PutCRC", zap.String("objectName", objectName))
	defer func() {
		g.l.Debug("End PutCRC", zap.String("objectName", objectName), zap.Error(err))
	}()
	// Put if not present
	var writer *gcsStorage.Writer
	if doesNotExist {
		writer = versionedObject(g.client.Bucket(g.bucket), objectName).If(gcsStorage.Conditions{
			DoesNotExist: doesNotExist,
		}).NewWriter(ctx)
	} else {
		writer = versionedObject(g.client.Bucket(g.bucket), objectName).NewWriter(ctx)
	}
	writer.CRC32C = crc
	g.l.Debug("Start PutCRC PipeIO", zap.String("objectName", objectName))
	_, err = storage.PipeIO(writer, readCloser{reader: reader})
	g.l.Debug("End PutCRC PipeIO", zap.String("objectName", objectName), zap.Error(err))
	if err != nil {
		return toSentinelErrors(err)
	}
	err = writer.Close()
	return toSentinelErrors(err)
}

func (g *gcs) Delete(ctx context.Context, objectName string) (err error) {
	g.l.Debug("Start Delete", zap.String("objectName", objectName))
	obj := versionedObject(g.client.Bucket(g.bucket), objectName)
	if ver := g.s.Version.GcsVersion(); ver != storage.GcsSentinelVersion {
		obj = obj.Generation(ver)
	}
	err = toSentinelErrors(obj.Delete(ctx))
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

func (g *gcs) KeysPrefix(
	ctx context.Context,
	pageToken string,
	prefix string,
	delimiter string,
	count int,
) (keys []string, next string, err error) {
	logger := g.l.With(
		zap.String("start", pageToken),
		zap.String("prefix", prefix),
		zap.Int("keys", len(keys)),
		zap.Error(err))
	logger.Debug("Start KeysPrefix")
	defer func() {
		logger.Debug("End KeysPrefix")
	}()

	itr := g.readOnlyClient.Bucket(g.bucket).Objects(ctx, &gcsStorage.Query{Prefix: prefix, Delimiter: delimiter})
	var objects []*gcsStorage.ObjectAttrs
	next, err = iterator.NewPager(itr, count, pageToken).NextPage(&objects)
	if err != nil {
		return nil, "", toSentinelErrors(err)
	}

	keys = make([]string, 0, count)
	for _, objAttrs := range objects {
		if objAttrs.Prefix != "" {
			keys = append(keys, objAttrs.Prefix)
		} else {
			keys = append(keys, objAttrs.Name)
		}
	}
	return
}

func (g *gcs) KeyVersions(ctx context.Context, key string) ([]storage.Version, error) {
	//	var err error
	logger := g.l.With(zap.String("key", key))
	logger.Debug("start KeyVersions")

	versionsPrefix := func(pageToken string) ([]storage.Version, string, error) {
		const versionsPerPage = 100
		itr := g.readOnlyClient.Bucket(g.bucket).Objects(ctx, &gcsStorage.Query{Prefix: key, Versions: true})
		var objects []*gcsStorage.ObjectAttrs
		nextPageToken, err := iterator.NewPager(itr, versionsPerPage, pageToken).NextPage(&objects)
		if err != nil {
			return nil, "", toSentinelErrors(err)
		}
		versions := make([]storage.Version, 0, versionsPerPage)
		for _, objAttrs := range objects {
			versions = append(versions, storage.NewVersionGcs(objAttrs.Generation))
		}
		return versions, nextPageToken, nil
	}

	pageToken := ""
	versions := make([]storage.Version, 0)
	for {
		versionsCurr, pageToken, err := versionsPrefix(pageToken)
		if err != nil {
			return nil, toSentinelErrors(err)
		}
		versions = append(versions, versionsCurr...)
		if pageToken == "" {
			break
		}
	}

	return versions, nil
}

func (g *gcs) Clear(context.Context) error {
	return storagestatus.ErrNotImplemented
}
