package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/oneconcern/datamon/pkg/storage"

	"go.uber.org/zap"
)

type mockDestStore struct {
	name                 string
	l                    *zap.Logger
	fileListUploadPutCnt int
}

func (mds *mockDestStore) String() string {
	return "mock destination store"
}

func (mds *mockDestStore) Has(ctx context.Context, key string) (bool, error) {
	// detect and respond to RepoExists() call
	if strings.HasPrefix(key, "repos/") && strings.HasSuffix(key, "/repo.yaml") {
		return true, nil
	}
	return false, errors.New("mock destination store Has() unimpl (other than for RepoExists calls)")
}

func (mds *mockDestStore) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return nil, errors.New("mock destination store Get() unimpl")
}

func (mds *mockDestStore) GetAt(ctx context.Context, key string) (io.ReaderAt, error) {
	return nil, errors.New("mock destination store GetAt() unimpl")
}

func (mds *mockDestStore) GetAttr(ctx context.Context, objectName string) (storage.Attributes, error) {
	panic("implement me")
}

func (mds *mockDestStore) Touch(ctx context.Context, objectName string) error {
	panic("implement me")
}

var fileListRe *regexp.Regexp

func init() {
	fileListRe = regexp.MustCompile(`bundle-files-\d+.yaml$`)
}

func (mds *mockDestStore) Put(ctx context.Context, key string, source io.Reader, exclusive bool) error {
	// strings.HasPrefix(key, "bundles/") &&
	isFilelist := fileListRe.MatchString(key)
	if isFilelist {
		mds.fileListUploadPutCnt++
		fmt.Println("mock dest store Put got filelist")
	}
	mds.l.Info("mock destination store Put()",
		zap.String("key", key),
		zap.Bool("isFilelist", isFilelist),
		zap.String("store name", mds.name),
		zap.Int("fileListUploadPutCnt", mds.fileListUploadPutCnt),
	)
	return nil
}

func (mds *mockDestStore) Delete(ctx context.Context, key string) error {
	return errors.New("mock destination store Delete() unimpl")
}

func (mds *mockDestStore) Keys(ctx context.Context) ([]string, error) {
	return nil, errors.New("mock destination store Keys() unimpl")
}

func (mds *mockDestStore) KeysPrefix(ctx context.Context, token, prefix, delimiter string, count int) ([]string, string, error) {
	return nil, "", errors.New("mock destination store KeysPrefix() unimpl")
}

func (mds *mockDestStore) Clear(ctx context.Context) error {
	return errors.New("mock destination store Clear() unimpl")
}

func newMockDestStore(name string, l *zap.Logger) storage.Store {
	return &mockDestStore{
		name: name,
		l:    l,
	}
}
