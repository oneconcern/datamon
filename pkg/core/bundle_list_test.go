package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/oneconcern/datamon/pkg/storage/mockstorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type bundleFixture struct {
	name          string
	repo          string
	wantError     bool
	expected      []model.BundleDescriptor
	errorContains []string
}

type testReadCloserWithErr struct {
}

func (testReadCloserWithErr) Read(_ []byte) (int, error) {
	return 0, errors.New("io error")
}
func (testReadCloserWithErr) Close() error {
	return nil
}

const tokStr = "token"

var bundleTestCases = []bundleFixture{
	{
		name: "happy path",
		repo: "happy/repo.json",
		expected: []model.BundleDescriptor{
			{
				ID:       "myID1",
				LeafSize: 16,
				Message:  "this is a message",
				Version:  4,
			},
			{
				ID:       "myID2",
				LeafSize: 16,
				Message:  "this is a message",
				Version:  4,
			},
			{
				ID:       "myID3",
				LeafSize: 16,
				Message:  "this is a message",
				Version:  4,
			},
		},
	},
	// error cases
	{
		name:          "no repo",
		repo:          "norepo/repo.json",
		wantError:     true,
		errorContains: []string{"repo validation: Repo", "does not exist"},
	},
	{
		name:          "no key",
		repo:          "nokey/repo.json",
		wantError:     true,
		errorContains: []string{"storage error"},
	},
	{
		name:          "invalid file name",
		repo:          "invalid/repo.json",
		wantError:     true,
		errorContains: []string{"expected label"},
	},
	{
		name:          "no archive path",
		repo:          "noarchive/repo.json",
		wantError:     true,
		errorContains: []string{"get store error"},
	},
	{
		name:          "invalid yaml",
		repo:          "badyaml/repo.json",
		wantError:     true,
		errorContains: []string{"yaml:"},
	},
	{
		name:          "inconsistent bundle ID",
		repo:          "badID/repo.json",
		wantError:     true,
		errorContains: []string{"bundle IDs in descriptor", "archive path"},
	},
	{
		name:          "io error",
		repo:          "ioerr/repo.json",
		wantError:     true,
		errorContains: []string{"io error"},
	},
	// skipped bundle
	{
		name: "skipped bundle",
		repo: "skipped/repo.json",
		expected: []model.BundleDescriptor{
			{
				ID:       "myID1",
				LeafSize: 16,
				Message:  "this is a message",
				Version:  4,
			},
			{
				ID:       "myID3",
				LeafSize: 16,
				Message:  "this is a message",
				Version:  4,
			},
		},
	},
}

func mockedStore(testcase string) storage.Store {
	// builds mocked up test scenarios
	switch testcase {
	case "happy path":
		return &mockstorage.StoreMock{
			HasFunc: func(_ context.Context, _ string) (bool, error) {
				return true, nil
			},
			KeysPrefixFunc: func(_ context.Context, _ string, prefix string, delimiter string, count int) ([]string, string, error) {
				return []string{"/key1/myID1/bundle.json", "/key2/myID2/bundle.json", "/key3/myID3/bundle.json"}, tokStr, nil
			},
			KeysFunc: func(_ context.Context) ([]string, error) {
				return nil, nil
			},
			GetFunc: func(_ context.Context, pth string) (io.ReadCloser, error) {
				parts := strings.Split(pth, "/")
				id := parts[3]
				return ioutil.NopCloser(strings.NewReader(fmt.Sprintf(`id: '%s'
leafSize: 16
message: 'this is a message'
version: 4`, id))), nil
			},
		}
	case "no repo":
		return &mockstorage.StoreMock{
			HasFunc: func(_ context.Context, _ string) (bool, error) {
				return false, nil
			},
		}
	case "no key":
		return &mockstorage.StoreMock{
			HasFunc: func(_ context.Context, _ string) (bool, error) {
				return true, nil
			},
			KeysPrefixFunc: func(_ context.Context, _ string, prefix string, delimiter string, count int) ([]string, string, error) {
				return nil, "", errors.New("storage error")
			},
		}
	case "invalid file name":
		return &mockstorage.StoreMock{
			HasFunc: func(_ context.Context, _ string) (bool, error) {
				return true, nil
			},
			KeysPrefixFunc: func(_ context.Context, _ string, prefix string, delimiter string, count int) ([]string, string, error) {
				return []string{"/key1/myID1/bundle.json", "labels/x/wrong/bundle.json"}, tokStr, nil
			},
			GetFunc: func(_ context.Context, pth string) (io.ReadCloser, error) {
				parts := strings.Split(pth, "/")
				id := parts[3]
				return ioutil.NopCloser(strings.NewReader(fmt.Sprintf(`id: '%s'
leafSize: 16
message: 'this is a message'
version: 4`, id))), nil
			},
		}
	case "no archive path":
		return &mockstorage.StoreMock{
			HasFunc: func(_ context.Context, _ string) (bool, error) {
				return true, nil
			},
			KeysPrefixFunc: func(_ context.Context, _ string, prefix string, delimiter string, count int) ([]string, string, error) {
				return []string{"/key1/myID1/bundle.json", "/key2/myID2/bundle.json", "/key3/myID3/bundle.json"}, tokStr, nil
			},
			GetFunc: func(_ context.Context, pth string) (io.ReadCloser, error) {
				return nil, errors.New("get store error")
			},
		}
	case "invalid yaml":
		return &mockstorage.StoreMock{
			HasFunc: func(_ context.Context, _ string) (bool, error) {
				return true, nil
			},
			KeysPrefixFunc: func(_ context.Context, _ string, prefix string, delimiter string, count int) ([]string, string, error) {
				return []string{"/key1/myID1/bundle.json", "/key2/myID2/bundle.json", "/key3/myID3/bundle.json"}, tokStr, nil
			},
			KeysFunc: func(_ context.Context) ([]string, error) {
				return nil, nil
			},
			GetFunc: func(_ context.Context, pth string) (io.ReadCloser, error) {
				parts := strings.Split(pth, "/")
				id := parts[3]
				return ioutil.NopCloser(strings.NewReader(fmt.Sprintf(`id: '%s'
leafSize: 16
>> dd
message: 'this is a message'
version: 4`, id))), nil
			},
		}
	case "inconsistent bundle ID":
		return &mockstorage.StoreMock{
			HasFunc: func(_ context.Context, _ string) (bool, error) {
				return true, nil
			},
			KeysPrefixFunc: func(_ context.Context, _ string, prefix string, delimiter string, count int) ([]string, string, error) {
				return []string{"/key1/myID1/bundle.json", "/key2/myID2/bundle.json", "/key3/myID3/bundle.json"}, tokStr, nil
			},
			KeysFunc: func(_ context.Context) ([]string, error) {
				return nil, nil
			},
			GetFunc: func(_ context.Context, pth string) (io.ReadCloser, error) {
				return ioutil.NopCloser(strings.NewReader(`id: 'wrong'
leafSize: 16
message: 'this is a message'
version: 4`)), nil
			},
		}
	case "io error":
		return &mockstorage.StoreMock{
			HasFunc: func(_ context.Context, _ string) (bool, error) {
				return true, nil
			},
			KeysPrefixFunc: func(_ context.Context, _ string, prefix string, delimiter string, count int) ([]string, string, error) {
				return []string{"/key1/myID1/bundle.json", "/key2/myID2/bundle.json", "/key3/myID3/bundle.json"}, tokStr, nil
			},
			GetFunc: func(_ context.Context, pth string) (io.ReadCloser, error) {
				return testReadCloserWithErr{}, nil
			},
		}
	case "skipped bundle":
		return &mockstorage.StoreMock{
			HasFunc: func(_ context.Context, _ string) (bool, error) {
				return true, nil
			},
			KeysPrefixFunc: func(_ context.Context, _ string, prefix string, delimiter string, count int) ([]string, string, error) {
				return []string{"/key1/myID1/bundle.json", "/key2/myID2/smurf.json", "/key3/myID3/bundle.json"}, tokStr, nil
			},
			KeysFunc: func(_ context.Context) ([]string, error) {
				return nil, nil
			},
			GetFunc: func(_ context.Context, pth string) (io.ReadCloser, error) {
				parts := strings.Split(pth, "/")
				id := parts[3]
				if id == "myID2" {
					return nil, errors.New("object doesn't exist")
				}
				return ioutil.NopCloser(strings.NewReader(fmt.Sprintf(`id: '%s'
leafSize: 16
message: 'this is a message'
version: 4`, id))), nil
			},
		}
	}
	return nil
}

func testListBundles(t *testing.T, concurrency int, i int) {
	for _, toPin := range bundleTestCases {
		testcase := toPin
		t.Run(fmt.Sprintf("%s-%d-%d", testcase.name, concurrency, i), func(t *testing.T) {
			t.Parallel()
			mockStore := mockedStore(testcase.name)
			res, err := ListBundles(testcase.repo, mockStore, ConcurrentBundleList(concurrency))
			if testcase.wantError {
				require.Error(t, err)
				for _, expectedMsg := range testcase.errorContains { // assert error message (opt-in)
					assert.Contains(t, err.Error(), expectedMsg)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, res)
			assert.ElementsMatch(t, testcase.expected, res)
		})
	}
}

func TestListBundles(t *testing.T) {
	for i := 0; i < 10; i++ {
		for _, concurrency := range []int{0, 1, 50, 100, 400} {
			t.Logf("simulating ListBundles with concurrency-factor=%d, iteration=%d", concurrency, i)
			testListBundles(t, concurrency, i)
		}
	}
}
