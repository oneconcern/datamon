package core

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"sort"
	"strings"
	"sync"
	"testing"

	context2 "github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/oneconcern/datamon/pkg/storage/mockstorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"gopkg.in/yaml.v2"
)

type repoFixture struct {
	name          string
	wantError     bool
	expected      model.RepoDescriptors
	errorContains []string
}

func repoTestCases() []repoFixture {
	return []repoFixture{
		{
			name: happyPath,
			expected: model.RepoDescriptors{
				{
					Name:        "myRepo1",
					Description: "test myRepo1",
					Contributor: model.Contributor{Email: "test@example.com"},
				},
			},
		},
		{
			name:     happyWithBatches,
			expected: expectedRepoBatchFixture,
		},
	}
}

func buildRepoYaml(repo string) string {
	r := model.RepoDescriptor{
		Name:        repo,
		Description: fmt.Sprintf("test %s", repo),
		Contributor: model.Contributor{Email: "test@example.com"},
	}
	asYaml, _ := yaml.Marshal(r)
	return string(asYaml)
}

var (
	initRepoBatchFixture     sync.Once
	repoBatchFixture         []string
	expectedRepoBatchFixture model.RepoDescriptors
)

func buildRepoBatchFixture(t *testing.T) func() {
	return func() {
		repoBatchFixture = make([]string, maxTestKeys)
		expectedRepoBatchFixture = make(model.RepoDescriptors, maxTestKeys)
		for i := 0; i < maxTestKeys; i++ {
			repoBatchFixture[i] = fmt.Sprintf("repos/myRepo%0.3d/repo.json", i)
			expectedRepoBatchFixture[i] = model.RepoDescriptor{
				Name:        fmt.Sprintf("myRepo%0.3d", i),
				Description: fmt.Sprintf("test myRepo%0.3d", i),
				Contributor: model.Contributor{Email: "test@example.com"},
			}
		}
		require.Truef(t, sort.IsSorted(expectedRepoBatchFixture), "got %v", expectedRepoBatchFixture)
	}
}

func mockedRepoStore(testcase string) storage.Store {
	switch testcase {
	case happyPath:
		return &mockstorage.StoreMock{
			HasFunc: func(_ context.Context, _ string) (bool, error) {
				return true, nil
			},
			KeysPrefixFunc: func(_ context.Context, _ string, prefix string, delimiter string, count int) ([]string, string, error) {
				return []string{"repos/myRepo1/repo.json"}, "", nil
			},
			KeysFunc: func(_ context.Context) ([]string, error) {
				return nil, nil
			},
			GetFunc: func(_ context.Context, pth string) (io.ReadCloser, error) {
				parts := strings.Split(pth, "/")
				repo := parts[1]
				return ioutil.NopCloser(strings.NewReader(buildRepoYaml(repo))), nil
			},
		}
	case happyWithBatches:
		return &mockstorage.StoreMock{
			HasFunc: func(_ context.Context, _ string) (bool, error) {
				return true, nil
			},
			KeysPrefixFunc: func(_ context.Context, _ string, prefix string, delimiter string, count int) ([]string, string, error) {
				return repoBatchFixture, "", nil
			},
			KeysFunc: func(_ context.Context) ([]string, error) {
				return nil, nil
			},
			GetFunc: func(_ context.Context, pth string) (io.ReadCloser, error) {
				parts := strings.Split(pth, "/")
				repo := parts[1]
				return ioutil.NopCloser(strings.NewReader(buildRepoYaml(repo))), nil
			},
		}
	default:
		return nil
	}
}

func testListRepos(t *testing.T, concurrency int, i int) {
	initRepoBatchFixture.Do(buildRepoBatchFixture(t))
	defer goleak.VerifyNone(t)
	for _, toPin := range repoTestCases() {
		testcase := toPin

		t.Run(fmt.Sprintf("ListRepos-%s-%d-%d", testcase.name, concurrency, i), func(t *testing.T) {
			t.Parallel()
			mockStore := mockedRepoStore(testcase.name)
			stores := context2.NewStores(nil, nil, nil, nil, mockStore)
			repos, err := ListRepos(stores, ConcurrentList(concurrency), BatchSize(testBatchSize))
			assertRepos(t, testcase, repos, err)
		})
		t.Run(fmt.Sprintf("ListReposApply-%s-%d-%d", testcase.name, concurrency, i), func(t *testing.T) {
			t.Parallel()
			mockStore := mockedRepoStore(testcase.name)
			stores := context2.NewStores(nil, nil, nil, nil, mockStore)
			repos := make(model.RepoDescriptors, 0, typicalReposNum)
			err := ListReposApply(stores, func(repo model.RepoDescriptor) error {
				repos = append(repos, repo)
				return nil
			}, ConcurrentList(concurrency), BatchSize(testBatchSize))
			assertRepos(t, testcase, repos, err)
		})
	}
}

func assertRepos(t *testing.T, testcase repoFixture, repos model.RepoDescriptors, err error) {
	if testcase.wantError {
		require.Error(t, err)
		for _, expectedMsg := range testcase.errorContains { // assert error message (opt-in)
			assert.Contains(t, err.Error(), expectedMsg)
		}

		assert.Len(t, repos, len(testcase.expected)) // assert result, possibly partial
		return
	}
	require.NoError(t, err)
	assert.ElementsMatch(t, testcase.expected, repos, "expected returned repos to match expected descriptors")
	assert.Truef(t, sort.IsSorted(repos), "expected a sorted output, got: %v", repos)
}

func TestGetRepoDescriptorByRepoName(t *testing.T) {
	testcase := repoTestCases()[0]

	mockStore := mockedRepoStore(testcase.name)
	stores := context2.NewStores(nil, nil, nil, mockStore, mockStore)
	repo, err := GetRepoDescriptorByRepoName(stores, "myRepo1")
	assertRepos(t, testcase, model.RepoDescriptors{repo}, err)
}

func TestListRepos(t *testing.T) {
	for i := 0; i < 10; i++ { // check results remain stable over 10 independent iterations
		for _, concurrency := range []int{0, 1, 50, 100, 400} { // test several concurrency parameters
			t.Logf("simulating ListRepos with concurrency-factor=%d, iteration=%d", concurrency, i)
			testListRepos(t, concurrency, i)
		}
	}
}
