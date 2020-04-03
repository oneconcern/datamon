package gcs

import (
	"context"
	"io"
	"strconv"

	gcsStorage "cloud.google.com/go/storage"
	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/oneconcern/datamon/pkg/storage/status"
	"go.uber.org/zap"
	"google.golang.org/api/iterator"
)

var _ storage.VersionedStore = &gcs{}

func (g *gcs) IsVersioned(ctx context.Context) (bool, error) {
	attr, err := g.readOnlyClient.Bucket(g.bucket).Attrs(ctx)
	if err != nil {
		return false, err
	}
	return attr.VersioningEnabled, nil
}

// KeyVersions returns all versions of a given key
func (g *gcs) KeyVersions(ctx context.Context, key string) ([]string, error) {
	//      var err error
	logger := g.l.With(zap.String("key", key))
	logger.Debug("start KeyVersions")

	versionsPrefix := func(pageToken string) ([]string, string, error) {
		const versionsPerPage = 1024
		itr := g.readOnlyClient.Bucket(g.bucket).Objects(ctx, &gcsStorage.Query{Prefix: key, Versions: true})
		var objects []*gcsStorage.ObjectAttrs
		nextPageToken, err := iterator.NewPager(itr, versionsPerPage, pageToken).NextPage(&objects)
		if err != nil {
			return nil, "", toSentinelErrors(err)
		}
		versions := make([]string, 0, len(objects))
		for _, objAttrs := range objects {
			versions = append(versions, strconv.FormatInt(objAttrs.Generation, 10))
		}
		return versions, nextPageToken, nil
	}

	var (
		pageToken              string
		err                    error
		versions, versionsPage []string
	)

	for {
		versionsPage, pageToken, err = versionsPrefix(pageToken)
		if err != nil {
			return nil, toSentinelErrors(err)
		}
		versions = append(versions, versionsPage...)
		if pageToken == "" {
			break
		}
	}

	return versions, nil
}

// GetVersion returns a reader pointing to a specific version of a stored object
func (g *gcs) GetVersion(ctx context.Context, objectName, version string) (io.ReadCloser, error) {
	g.l.Debug("Start GetVersion", zap.String("objectName", objectName))
	var err error
	defer g.l.Debug("End GetVersion", zap.String("objectName", objectName), zap.Error(err))

	gen, err := strconv.ParseInt(version, 10, 64)
	if err != nil {
		return nil, status.ErrInvalidVersion.Wrap(err)
	}

	objectReader, err := g.readOnlyClient.Bucket(g.bucket).Object(objectName).Generation(gen).NewReader(ctx)
	if err != nil {
		return nil, toSentinelErrors(err)
	}
	return &gcsReader{
		g:            g,
		objectReader: objectReader,
		l:            g.l,
	}, nil
}
