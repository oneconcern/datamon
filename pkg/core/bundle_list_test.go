package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/oneconcern/datamon/pkg/storage/mockstorage"
	"github.com/oneconcern/datamon/pkg/storage/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

type bundleFixture struct {
	name          string
	repo          string
	wantError     bool
	expected      model.BundleDescriptors
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

func bundleTestCases() []bundleFixture {
	return []bundleFixture{
		{
			name: "happy path",
			repo: "happy/repo.json",
			expected: model.BundleDescriptors{
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
		{
			name:     "happy with batches",
			repo:     "happy/repo.json",
			expected: expectedBatchFixture,
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
		// n-th batch returns an error while fetching keys
		{
			name:          batchErrorTestcase,
			repo:          "batch/repo.json",
			expected:      expectedBatchFixture[0:25], // returned 5 first batches then bailed
			wantError:     true,
			errorContains: []string{"test key fetch error"},
		},
		// n-th batch returns an error while fetching bundle
		{
			name:          batchErrorRepoTestcase,
			repo:          "batch/repo.json",
			expected:      expectedBatchFixture[0:25], // returned 5 first batches then bailed
			wantError:     true,
			errorContains: []string{"test repo fetch error"},
		},
	}
}

const (
	batchErrorRepoTestcase = "batch error repo"
	batchErrorTestcase     = "batch error"
)

var (
	initBatchKeysFixture sync.Once
	keysBatchFixture     []string
	expectedBatchFixture model.BundleDescriptors
)

func buildKeysBatchFixture(t *testing.T) func() {
	return func() {
		keysBatchFixture = make([]string, maxTestKeys)
		expectedBatchFixture = make(model.BundleDescriptors, maxTestKeys)
		for i := 0; i < maxTestKeys; i++ {
			keysBatchFixture[i] = fmt.Sprintf("/key%0.3d/myID%0.3d/bundle.json", i, i)
			expectedBatchFixture[i] = model.BundleDescriptor{
				ID:       fmt.Sprintf("myID%0.3d", i),
				LeafSize: 16,
				Message:  "this is a message",
				Version:  4,
			}
		}
		require.Truef(t, sort.IsSorted(expectedBatchFixture), "got %v", expectedBatchFixture)
	}
}

func buildYaml(id string) string {
	return fmt.Sprintf(`id: '%s'
leafSize: 16
message: 'this is a message'
version: 4`, id)
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
				return []string{"/key1/myID1/bundle.json", "/key2/myID2/bundle.json", "/key3/myID3/bundle.json"}, "", nil
			},
			KeysFunc: func(_ context.Context) ([]string, error) {
				return nil, nil
			},
			GetFunc: func(_ context.Context, pth string) (io.ReadCloser, error) {
				parts := strings.Split(pth, "/")
				id := parts[3]
				return ioutil.NopCloser(strings.NewReader(buildYaml(id))), nil
			},
		}
	case "happy with batches":
		return &mockstorage.StoreMock{
			HasFunc: func(_ context.Context, _ string) (bool, error) {
				return true, nil
			},
			KeysPrefixFunc: func(_ context.Context, next string, _ string, _ string, count int) ([]string, string, error) {
				index := 0
				window := minInt(count, len(keysBatchFixture))

				switch next {
				case "":
					break
				default:
					for i, key := range keysBatchFixture {
						if key == next {
							index = i
							break
						}
					}
				}

				var following string
				if index+window < len(keysBatchFixture) {
					following = keysBatchFixture[index+window]
				}
				last := minInt(index+window, len(keysBatchFixture))
				return keysBatchFixture[index:last], following, nil
			},
			KeysFunc: func(_ context.Context) ([]string, error) {
				return nil, nil
			},
			GetFunc: func(_ context.Context, pth string) (io.ReadCloser, error) {
				parts := strings.Split(pth, "/")
				id := parts[3]
				return ioutil.NopCloser(strings.NewReader(buildYaml(id))), nil
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
				return []string{"/key1/myID1/bundle.json", "labels/x/wrong/bundle.json"}, "", nil
			},
			GetFunc: func(_ context.Context, pth string) (io.ReadCloser, error) {
				parts := strings.Split(pth, "/")
				id := parts[3]
				return ioutil.NopCloser(strings.NewReader(buildYaml(id))), nil
			},
		}
	case "no archive path":
		return &mockstorage.StoreMock{
			HasFunc: func(_ context.Context, _ string) (bool, error) {
				return true, nil
			},
			KeysPrefixFunc: func(_ context.Context, _ string, prefix string, delimiter string, count int) ([]string, string, error) {
				return []string{"/key1/myID1/bundle.json", "/key2/myID2/bundle.json", "/key3/myID3/bundle.json"}, "", nil
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
				return []string{"/key1/myID1/bundle.json", "/key2/myID2/bundle.json", "/key3/myID3/bundle.json"}, "", nil
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
				return []string{"/key1/myID1/bundle.json", "/key2/myID2/bundle.json", "/key3/myID3/bundle.json"}, "", nil
			},
			KeysFunc: func(_ context.Context) ([]string, error) {
				return nil, nil
			},
			GetFunc: func(_ context.Context, pth string) (io.ReadCloser, error) {
				return ioutil.NopCloser(strings.NewReader(buildYaml("wrong"))), nil
			},
		}
	case "io error":
		return &mockstorage.StoreMock{
			HasFunc: func(_ context.Context, _ string) (bool, error) {
				return true, nil
			},
			KeysPrefixFunc: func(_ context.Context, _ string, prefix string, delimiter string, count int) ([]string, string, error) {
				return []string{"/key1/myID1/bundle.json", "/key2/myID2/bundle.json", "/key3/myID3/bundle.json"}, "", nil
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
				return []string{"/key1/myID1/bundle.json", "/key2/myID2/smurf.json", "/key3/myID3/bundle.json"}, "", nil
			},
			KeysFunc: func(_ context.Context) ([]string, error) {
				return nil, nil
			},
			GetFunc: func(_ context.Context, pth string) (io.ReadCloser, error) {
				parts := strings.Split(pth, "/")
				id := parts[3]
				if id == "myID2" {
					return nil, status.ErrNotExists
				}
				return ioutil.NopCloser(strings.NewReader(buildYaml(id))), nil
			},
		}
	case batchErrorTestcase:
		return &mockstorage.StoreMock{
			HasFunc: func(_ context.Context, _ string) (bool, error) {
				return true, nil
			},
			KeysPrefixFunc: func(_ context.Context, next string, _ string, _ string, count int) ([]string, string, error) {
				index := 0
				window := minInt(count, len(keysBatchFixture))

				switch next {
				case "":
					break
				default:
					for i, key := range keysBatchFixture {
						if key == next {
							index = i
							break
						}
					}
				}

				if index > 4*testBatchSize {
					return nil, "", errors.New("test key fetch error")
				}

				var following string
				if index+window < len(keysBatchFixture) {
					following = keysBatchFixture[index+window]
				}
				last := minInt(index+window, len(keysBatchFixture))
				return keysBatchFixture[index:last], following, nil
			},
			KeysFunc: func(_ context.Context) ([]string, error) {
				return nil, nil
			},
			GetFunc: func(_ context.Context, pth string) (io.ReadCloser, error) {
				parts := strings.Split(pth, "/")
				id := parts[3]
				return ioutil.NopCloser(strings.NewReader(buildYaml(id))), nil
			},
		}
	case batchErrorRepoTestcase:
		return &mockstorage.StoreMock{
			HasFunc: func(_ context.Context, _ string) (bool, error) {
				return true, nil
			},
			KeysPrefixFunc: func(_ context.Context, next string, _ string, _ string, count int) ([]string, string, error) {
				index := 0
				window := minInt(count, len(keysBatchFixture))

				switch next {
				case "":
					break
				default:
					for i, key := range keysBatchFixture {
						if key == next {
							index = i
							break
						}
					}
				}

				var following string
				if index+window < len(keysBatchFixture) {
					following = keysBatchFixture[index+window]
				}
				last := minInt(index+window, len(keysBatchFixture))
				return keysBatchFixture[index:last], following, nil
			},
			KeysFunc: func(_ context.Context) ([]string, error) {
				return nil, nil
			},
			GetFunc: func(_ context.Context, pth string) (io.ReadCloser, error) {
				parts := strings.Split(pth, "/")
				id := parts[3]
				index := 0
				for i, key := range keysBatchFixture {
					if strings.Contains(key, id) {
						index = i
						break
					}
				}
				if index > 5*testBatchSize {
					return nil, errors.New("test repo fetch error")
				}

				return ioutil.NopCloser(strings.NewReader(buildYaml(id))), nil
			},
		}
	}
	return nil
}

const (
	testBatchSize = 5
	maxTestKeys   = 100 * testBatchSize
)

func testListBundles(t *testing.T, concurrency int, i int) {
	initBatchKeysFixture.Do(buildKeysBatchFixture(t))
	defer goleak.VerifyNone(t)

	for _, toPin := range bundleTestCases() {
		testcase := toPin

		// ListBundles: blocking collection of bundles
		t.Run(fmt.Sprintf("ListBundles-%s-%d-%d", testcase.name, concurrency, i), func(t *testing.T) {
			t.Parallel()
			mockStore := mockedStore(testcase.name)
			bundles, err := ListBundles(testcase.repo, mockStore, ConcurrentBundleList(concurrency), BundleBatchSize(testBatchSize))
			assertBundles(t, testcase, bundles, err)
		})

		// ListBundlesApply emulating blocking collection of bundles
		t.Run(fmt.Sprintf("ListBundlesApply-%s-%d-%d", testcase.name, concurrency, i), func(t *testing.T) {
			t.Parallel()
			mockStore := mockedStore(testcase.name)
			bundles := make(model.BundleDescriptors, 0, typicalBundlesNum)
			err := ListBundlesApply(testcase.repo, mockStore, func(bundle model.BundleDescriptor) error {
				bundles = append(bundles, bundle)
				return nil
			}, ConcurrentBundleList(concurrency), BundleBatchSize(testBatchSize))
			assertBundles(t, testcase, bundles, err)
		})

		// ListBundlesApply with a func failing randomly
		t.Run(fmt.Sprintf("ListBundlesApplyFail-%s-%d-%d", testcase.name, concurrency, i), func(t *testing.T) {
			t.Parallel()
			mockStore := mockedStore(testcase.name)
			bundles := make(model.BundleDescriptors, 0, typicalBundlesNum)
			var fail bool
			err := ListBundlesApply(testcase.repo, mockStore, func(bundle model.BundleDescriptor) error {
				bundles = append(bundles, bundle)
				fail = rand.Intn(2) > 0
				if fail {
					return errors.New("applied test func error")
				}
				return nil
			}, ConcurrentBundleList(concurrency), BundleBatchSize(testBatchSize))

			if fail {
				require.Error(t, err)
				if !testcase.wantError {
					assert.Contains(t, err.Error(), "applied test func")
					return
				}
				switch testcase.name {
				case batchErrorTestcase, batchErrorRepoTestcase:
					assert.True(t, strings.Contains(err.Error(), testcase.errorContains[0]) || strings.Contains(err.Error(), "applied test func"))
				default:
					assertBundles(t, testcase, bundles, err)
				}
				return
			}
			assertBundles(t, testcase, bundles, err)
		})
	}
}

func assertBundles(t *testing.T, testcase bundleFixture, bundles model.BundleDescriptors, err error) {
	if testcase.wantError {
		require.Error(t, err)
		for _, expectedMsg := range testcase.errorContains { // assert error message (opt-in)
			assert.Contains(t, err.Error(), expectedMsg)
		}

		assert.Len(t, bundles, len(testcase.expected)) // assert result, possibly partial
		return
	}
	require.NoError(t, err)

	assert.ElementsMatch(t, testcase.expected, bundles)
	assert.Truef(t, sort.IsSorted(bundles), "expected a sorted output, got: %v", bundles)
}

func TestListBundles(t *testing.T) {
	for i := 0; i < 10; i++ { // check results remain stable over 10 independent iterations
		for _, concurrency := range []int{0, 1, 50, 100, 400} { // test several concurrency parameterss
			t.Logf("simulating ListBundles with concurrency-factor=%d, iteration=%d", concurrency, i)
			testListBundles(t, concurrency, i)
		}
	}
}
