package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	context2 "github.com/oneconcern/datamon/pkg/context"

	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/oneconcern/datamon/pkg/storage/mockstorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"gopkg.in/yaml.v2"
)

type labelFixture struct {
	name          string
	repo          string
	prefix        string
	wantError     bool
	expected      model.LabelDescriptors
	errorContains []string
}

func labelTestCases() []labelFixture {
	return []labelFixture{
		{
			name:   happyPath,
			repo:   "myRepo",
			prefix: "myLab",
			expected: model.LabelDescriptors{
				{
					Name:      "myLabel-test",
					BundleID:  "bundle-myLabel-test",
					Timestamp: testTime(),
					Contributors: []model.Contributor{
						{Email: "test1@example.com"},
						{Email: "test2@example.com"},
					},
				},
			},
		},
		{
			name:     happyWithBatches,
			repo:     "myRepo",
			prefix:   "myLab",
			expected: expectedLabelBatchFixture,
		},
	}
}

func buildLabelYaml(id string) string {
	label := model.LabelDescriptor{
		Name:         id,
		BundleID:     fmt.Sprintf("bundle-%s", id),
		Timestamp:    testTime(),
		Contributors: []model.Contributor{{Email: "test1@example.com"}, {Email: "test2@example.com"}},
	}
	asYaml, _ := yaml.Marshal(label)
	return string(asYaml)
}

var (
	initLabelBatchFixture     sync.Once
	labelBatchFixture         []string
	expectedLabelBatchFixture model.LabelDescriptors
	baseTime                  time.Time
)

func init() {
	baseTime = time.Now().Truncate(time.Hour).UTC() // avoid loss of time resolution through yaml marshalling
}

func testTime() time.Time {
	return baseTime
}

func buildLabelBatchFixture(t *testing.T) func() {
	return func() {
		labelBatchFixture = make([]string, maxTestKeys)
		expectedLabelBatchFixture = make(model.LabelDescriptors, maxTestKeys)
		for i := 0; i < maxTestKeys; i++ {
			labelBatchFixture[i] = fmt.Sprintf("labels/myRepo/myLabel-%0.3d.yaml", i)
			expectedLabelBatchFixture[i] = model.LabelDescriptor{
				Name:      fmt.Sprintf("myLabel-%0.3d", i),
				BundleID:  fmt.Sprintf("bundle-myLabel-%0.3d", i),
				Timestamp: testTime(),
				Contributors: []model.Contributor{
					{Email: "test1@example.com"},
					{Email: "test2@example.com"},
				},
			}
		}
		require.Truef(t, sort.IsSorted(expectedBatchFixture), "got %v", expectedBatchFixture)
	}
}

func extractID(pth string) string {
	labelNameRe := regexp.MustCompile(`^(.*)\.yaml$`)
	parts := strings.Split(pth, "/")
	m := labelNameRe.FindStringSubmatch(parts[2])
	if len(m) < 2 {
		panic(fmt.Sprintf("invalid testcase: %s", pth))
	}
	return m[1]
}

func mockedLabelStore(testcase string) storage.Store {
	switch testcase {
	case happyPath:
		return &mockstorage.StoreMock{
			HasFunc: func(_ context.Context, _ string) (bool, error) {
				return true, nil
			},
			KeysPrefixFunc: func(_ context.Context, _ string, prefix string, delimiter string, count int) ([]string, string, error) {
				return []string{"labels/myRepo/myLabel-test.yaml"}, "", nil
			},
			KeysFunc: func(_ context.Context) ([]string, error) {
				return nil, nil
			},
			GetFunc: func(_ context.Context, pth string) (io.ReadCloser, error) {
				extractID(pth)
				return ioutil.NopCloser(strings.NewReader(buildLabelYaml(extractID(pth)))), nil
			},
		}
	case happyWithBatches:
		return &mockstorage.StoreMock{
			HasFunc: func(_ context.Context, _ string) (bool, error) {
				return true, nil
			},
			KeysPrefixFunc: func(_ context.Context, _ string, prefix string, delimiter string, count int) ([]string, string, error) {
				return labelBatchFixture, "", nil
			},
			KeysFunc: func(_ context.Context) ([]string, error) {
				return nil, nil
			},
			GetFunc: func(_ context.Context, pth string) (io.ReadCloser, error) {
				return ioutil.NopCloser(strings.NewReader(buildLabelYaml(extractID(pth)))), nil
			},
		}
	default:
		return nil
	}
}

func testListLabels(t *testing.T, concurrency int, i int) {
	initLabelBatchFixture.Do(buildLabelBatchFixture(t))
	defer goleak.VerifyNone(t)
	for _, toPin := range labelTestCases() {
		testcase := toPin

		t.Run(fmt.Sprintf("ListLabels-%s-%d-%d", testcase.name, concurrency, i), func(t *testing.T) {
			//t.Parallel()
			mockStore := mockedLabelStore(testcase.name)
			stores := context2.NewStores(nil, nil, nil, mockStore, mockStore)
			labels, err := ListLabels(testcase.repo, stores, testcase.prefix, ConcurrentList(concurrency), BatchSize(testBatchSize))
			assertLabels(t, testcase, labels, err)
		})
		t.Run(fmt.Sprintf("ListLabelsApply-%s-%d-%d", testcase.name, concurrency, i), func(t *testing.T) {
			//t.Parallel()
			mockStore := mockedLabelStore(testcase.name)
			labels := make(model.LabelDescriptors, 0, typicalReposNum)
			stores := context2.NewStores(nil, nil, nil, mockStore, mockStore)
			err := ListLabelsApply(testcase.repo, stores, testcase.prefix, func(label model.LabelDescriptor) error {
				labels = append(labels, label)
				return nil
			}, ConcurrentList(concurrency), BatchSize(testBatchSize))
			assertLabels(t, testcase, labels, err)
		})
	}
}

func assertLabels(t *testing.T, testcase labelFixture, labels model.LabelDescriptors, err error) {
	if testcase.wantError {
		require.Error(t, err)
		for _, expectedMsg := range testcase.errorContains { // assert error message (opt-in)
			assert.Contains(t, err.Error(), expectedMsg)
		}

		assert.Len(t, labels, len(testcase.expected)) // assert result, possibly partial
		return
	}
	require.NoError(t, err)
	if !assert.ElementsMatch(t, testcase.expected, labels, "expected returned labels to match expected descriptors") {
		// output the details upon failure
		exp, _ := json.MarshalIndent(testcase.expected, "", " ")
		act, _ := json.MarshalIndent(labels, "", " ")
		assert.JSONEqf(t, string(exp), string(act), "expected equivalent marshalled JSON")
	}
	assert.Truef(t, sort.IsSorted(labels), "expected a sorted output, got: %v", labels)
}

func TestListLabels(t *testing.T) {
	for i := 0; i < 10; i++ { // check results remain stable over 10 independent iterations
		for _, concurrency := range []int{0, 1, 50, 100, 400} { // test several concurrency parameters
			t.Logf("simulating ListLabels with concurrency-factor=%d, iteration=%d", concurrency, i)
			testListLabels(t, concurrency, i)
		}
	}
}
