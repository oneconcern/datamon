package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"testing"

	context2 "github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/errors"

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

const (
	happyPath              = "happy path"
	happyWithBatches       = "happy with batches"
	batchErrorRepoTestcase = "batch error repo"
	batchErrorTestcase     = "batch error"
	testBatchSize          = 5
	maxTestKeys            = 100 * testBatchSize
)

var (
	initBatchKeysFixture sync.Once
	keysBatchFixture     []string
	expectedBatchFixture model.BundleDescriptors
)

func bundleTestCases() []bundleFixture {
	return []bundleFixture{
		{
			name: happyPath,
			repo: "happy",
			expected: model.BundleDescriptors{
				fakeBD("myID1"), fakeBD("myID2"), fakeBD("myID3"),
			},
		},
		{
			name:     happyWithBatches,
			repo:     "happy",
			expected: expectedBatchFixture,
		},
		// error cases
		{
			name:          "no repo",
			repo:          "norepo",
			wantError:     true,
			errorContains: []string{"repo validation: Repo", "does not exist"},
		},
		{
			name:          "no key",
			repo:          "nokey",
			wantError:     true,
			errorContains: []string{"storage error"},
		},
		{
			name:          "invalid file name",
			repo:          "invalid",
			wantError:     true,
			errorContains: []string{"path is invalid"},
		},
		{
			name:          "no archive path",
			repo:          "noarchive",
			wantError:     true,
			errorContains: []string{"get store error"},
		},
		{
			name:          "invalid yaml",
			repo:          "badyaml",
			wantError:     true,
			errorContains: []string{"yaml:"},
		},
		{
			name:          "inconsistent bundle ID",
			repo:          "badID",
			wantError:     true,
			errorContains: []string{"bundle IDs in descriptor", "archive path"},
		},
		{
			name:          "io error",
			repo:          "ioerr",
			wantError:     true,
			errorContains: []string{"io error"},
		},
		// skipped bundle
		{
			name: "skipped bundle",
			repo: "skipped",
			expected: []model.BundleDescriptor{
				fakeBD("myID1"), fakeBD("myID3"),
			},
		},
		// n-th batch returns an error while fetching keys
		{
			name:          batchErrorTestcase,
			repo:          "batch",
			expected:      expectedBatchFixture[0:25], // returned 5 first batches then bailed
			wantError:     true,
			errorContains: []string{"test key fetch error"},
		},
		// n-th batch returns an error while fetching bundle
		{
			name:          batchErrorRepoTestcase,
			repo:          "batch",
			expected:      expectedBatchFixture[0:25], // returned 5 first batches then bailed
			wantError:     true,
			errorContains: []string{"test repo fetch error"},
		},
	}
}

func buildKeysBatchFixture(t *testing.T) func() {
	return func() {
		keysBatchFixture = make([]string, maxTestKeys)
		expectedBatchFixture = make(model.BundleDescriptors, maxTestKeys)
		for i := 0; i < maxTestKeys; i++ {
			keysBatchFixture[i] = fakeBundlePath("batch", fmt.Sprintf("myID%0.3d", i))
			expectedBatchFixture[i] = fakeBD(fmt.Sprintf("myID%0.3d", i))
		}
		require.Truef(t, sort.IsSorted(expectedBatchFixture), "got %v", expectedBatchFixture)
	}
}

func mockedStore(testcase string) storage.Store {
	// builds mocked up test scenarios
	switch testcase {
	case happyPath:
		return &mockstorage.StoreMock{
			HasFunc:        goodHasFunc,
			KeysPrefixFunc: goodKeysPrefixFunc("happy"),
			KeysFunc:       goodKeysFunc,
			GetFunc:        goodGetFunc,
		}
	case happyWithBatches:
		return &mockstorage.StoreMock{
			HasFunc:        goodHasFunc,
			KeysPrefixFunc: goodWindowKeysPrefixFunc(keysBatchFixture),
			KeysFunc:       goodKeysFunc,
			GetFunc:        goodGetFunc,
		}
	case "no repo":
		return &mockstorage.StoreMock{
			HasFunc: func(_ context.Context, _ string) (bool, error) {
				return false, nil
			},
		}
	case "no key":
		return &mockstorage.StoreMock{
			HasFunc: goodHasFunc,
			KeysPrefixFunc: func(_ context.Context, _ string, prefix string, delimiter string, count int) ([]string, string, error) {
				return nil, "", errors.New("storage error")
			},
		}
	case "invalid file name":
		return &mockstorage.StoreMock{
			HasFunc: goodHasFunc,
			KeysPrefixFunc: func(_ context.Context, _ string, prefix string, delimiter string, count int) ([]string, string, error) {
				return []string{fakeBundlePath("invalid", "myID1"), "labels/x/wrong/bundle.yaml"}, "", nil
			},
			GetFunc: goodGetFunc,
		}
	case "no archive path":
		return &mockstorage.StoreMock{
			HasFunc:        goodHasFunc,
			KeysPrefixFunc: goodKeysPrefixFunc("noarchive"),
			GetFunc: func(_ context.Context, pth string) (io.ReadCloser, error) {
				return nil, errors.New("get store error")
			},
		}
	case "invalid yaml":
		return &mockstorage.StoreMock{
			HasFunc:        goodHasFunc,
			KeysPrefixFunc: goodKeysPrefixFunc("badyaml"),
			KeysFunc:       goodKeysFunc,
			GetFunc: func(_ context.Context, pth string) (io.ReadCloser, error) {
				id := extractBundleID(pth)
				return ioutil.NopCloser(strings.NewReader(garbleYaml(buildBundleYaml(id)))), nil
			},
		}
	case "inconsistent bundle ID":
		return &mockstorage.StoreMock{
			HasFunc:        goodHasFunc,
			KeysPrefixFunc: goodKeysPrefixFunc("badID"),
			KeysFunc:       goodKeysFunc,
			GetFunc: func(_ context.Context, pth string) (io.ReadCloser, error) {
				return ioutil.NopCloser(strings.NewReader(buildBundleYaml("wrong"))), nil
			},
		}
	case "io error":
		return &mockstorage.StoreMock{
			HasFunc:        goodHasFunc,
			KeysPrefixFunc: goodKeysPrefixFunc("ioerr"),
			KeysFunc:       goodKeysFunc,
			GetFunc: func(_ context.Context, pth string) (io.ReadCloser, error) {
				return testReadCloserWithErr{}, nil
			},
		}
	case "skipped bundle":
		return &mockstorage.StoreMock{
			HasFunc:        goodHasFunc,
			KeysPrefixFunc: goodKeysPrefixFunc("skipped"),
			KeysFunc:       goodKeysFunc,
			GetFunc: func(_ context.Context, pth string) (io.ReadCloser, error) {
				id := extractBundleID(pth)
				if id == "myID2" {
					return nil, status.ErrNotExists
				}
				return ioutil.NopCloser(strings.NewReader(buildBundleYaml(id))), nil
			},
		}
	case batchErrorTestcase:
		// error occurs somewhere after several batches of keys are successfully retrieved
		return &mockstorage.StoreMock{
			HasFunc:        goodHasFunc,
			KeysPrefixFunc: breakAfterFourBatches(keysBatchFixture),
			KeysFunc:       goodKeysFunc,
			GetFunc:        goodGetFunc,
		}
	case batchErrorRepoTestcase:
		// error occurs somewhere after several bundles are successfully retrieved
		return &mockstorage.StoreMock{
			HasFunc:        goodHasFunc,
			KeysPrefixFunc: goodWindowKeysPrefixFunc(keysBatchFixture),
			KeysFunc:       goodKeysFunc,
			GetFunc:        breakAferFiveBundlesGetFunc,
		}
	}
	return nil
}

func mockedContextStores(scenario string) context2.Stores {
	mockStore := mockedStore(scenario)
	return context2.NewStores(nil, nil, nil, mockStore, nil)
}

func testListBundles(t *testing.T, concurrency int, i int) {
	initBatchKeysFixture.Do(buildKeysBatchFixture(t))
	defer goleak.VerifyNone(t,
		// opencensus stats collection goroutine
		goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"))

	for _, toPin := range bundleTestCases() {
		testcase := toPin

		// ListBundles: blocking collection of bundles
		t.Run(fmt.Sprintf("ListBundles-%s-%d-%d", testcase.name, concurrency, i), func(t *testing.T) {
			t.Parallel()
			bundles, err := ListBundles(testcase.repo,
				mockedContextStores(testcase.name), ConcurrentList(concurrency), BatchSize(testBatchSize))
			assertBundles(t, testcase, bundles, err)
		})

		// ListBundlesApply emulating blocking collection of bundles
		t.Run(fmt.Sprintf("ListBundlesApply-%s-%d-%d", testcase.name, concurrency, i), func(t *testing.T) {
			t.Parallel()
			bundles := make(model.BundleDescriptors, 0, typicalBundlesNum)
			err := ListBundlesApply(testcase.repo,
				mockedContextStores(testcase.name), func(bundle model.BundleDescriptor) error {
					bundles = append(bundles, bundle)
					return nil
				}, ConcurrentList(concurrency), BatchSize(testBatchSize))
			assertBundles(t, testcase, bundles, err)
		})

		// ListBundlesApply with a func failing randomly
		t.Run(fmt.Sprintf("ListBundlesApplyFail-%s-%d-%d", testcase.name, concurrency, i), func(t *testing.T) {
			t.Parallel()
			bundles := make(model.BundleDescriptors, 0, typicalBundlesNum)
			var fail bool
			err := ListBundlesApply(testcase.repo,
				mockedContextStores(testcase.name), func(bundle model.BundleDescriptor) error {
					bundles = append(bundles, bundle)
					fail = rand.Intn(2) > 0 //#nosec
					if fail {
						return errors.New("applied test func error")
					}
					return nil
				}, ConcurrentList(concurrency), BatchSize(testBatchSize))

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

	if !assert.ElementsMatch(t, testcase.expected, bundles) {
		// show details
		exp, _ := json.MarshalIndent(testcase.expected, "", " ")
		act, _ := json.MarshalIndent(bundles, "", " ")
		assert.JSONEqf(t, string(exp), string(act), "expected equal JSON bundles")
	}
	assert.Truef(t, sort.IsSorted(bundles), "expected a sorted output, got: %v", bundles)
}

func TestListBundles(t *testing.T) {
	for i := 0; i < 10; i++ { // check results remain stable over 10 independent iterations
		for _, concurrency := range []int{0, 1, 50, 100, 400} { // test several concurrency parameters
			t.Logf("simulating ListBundles with concurrency-factor=%d, iteration=%d", concurrency, i)
			testListBundles(t, concurrency, i)
		}
	}
}
