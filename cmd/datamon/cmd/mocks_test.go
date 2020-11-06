package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	gcsStorage "cloud.google.com/go/storage"
	"github.com/oneconcern/datamon/internal/rand"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
)

func generateRepoName(in string) string {
	return "test-" + in + "-repo-" + rand.LetterString(10)
}

type ExitMocks struct {
	mock.Mock
	exitStatuses []int
}

func (m *ExitMocks) Fatalf(format string, v ...interface{}) {
	fmt.Printf(format+"\n", v...)
	m.exitStatuses = append(m.exitStatuses, 1)
}

func (m *ExitMocks) Fatalln(v ...interface{}) {
	fmt.Println(v...)
	m.exitStatuses = append(m.exitStatuses, 1)
}

func (m *ExitMocks) Exit(code int) {
	m.exitStatuses = append(m.exitStatuses, code)
}

func (m *ExitMocks) fatalCalls() int {
	return len(m.exitStatuses)
}

func NewExitMocks() *ExitMocks {
	exitMocks := ExitMocks{
		exitStatuses: make([]int, 0),
	}
	return &exitMocks
}

func MakeExitMock(m *ExitMocks) func(int) {
	return func(code int) {
		m.Exit(code)
	}
}

var exitMocks *ExitMocks

type AuthMock struct {
	email string
	name  string
}

func (a AuthMock) Principal(_ string) (model.Contributor, error) {
	return model.Contributor{
		Name:  a.name,
		Email: a.email,
	}, nil
}

func testContext() string {
	// the context used by all tests
	return "test-context"
}

func setupConfig(t *testing.T, flags flagsT) func() {
	r := rand.LetterString(15)
	bucketConfig := "datamon-deleteme-config" + r
	testContext := testContext()
	client, err := gcsStorage.NewClient(context.Background(), option.WithScopes(gcsStorage.ScopeFullControl))
	require.NoError(t, err, "couldn't create bucket client")
	err = client.Bucket(bucketConfig).Create(context.Background(), "onec-co", nil)
	require.NoError(t, err, "couldn't create config bucket")
	runCmd(t, []string{
		"context",
		"create",
		"--config",
		bucketConfig,
		"--context",
		testContext,
		"--blob",
		flags.context.Descriptor.Blob,
		"--wal",
		flags.context.Descriptor.WAL,
		"--vmeta",
		flags.context.Descriptor.VMetadata,
		"--meta",
		flags.context.Descriptor.Metadata,
		"--read-log",
		flags.context.Descriptor.ReadLog,
		// "--loglevel", "debug",
	}, "test and create context", false)
	err = os.Setenv("DATAMON_GLOBAL_CONFIG", bucketConfig)
	require.NoError(t, err)
	err = os.Setenv("DATAMON_CONTEXT", testContext)
	require.NoError(t, err)
	cleanup := func() {
		deleteBucket(context.Background(), t, client, bucketConfig)
	}
	return cleanup
}

func setupTests(t *testing.T) func() {
	_ = os.RemoveAll(destinationDir)
	ctx := context.Background()
	exitMocks = NewExitMocks()
	osExit = MakeExitMock(exitMocks)

	btag := rand.LetterString(15)
	name := strings.ToLower(t.Name())
	prefix := "delete-" + btag
	bucketMeta := prefix + "-meta-" + name
	bucketBlob := prefix + "-blob-" + name
	bucketVMeta := prefix + "-vmeta-" + name
	bucketWAL := prefix + "-wal-" + name
	bucketReadLog := prefix + "-read-log-" + name

	client, err := gcsStorage.NewClient(ctx, option.WithScopes(gcsStorage.ScopeFullControl))
	require.NoError(t, err, "couldn't create bucket client")

	buckets := []string{bucketBlob, bucketMeta, bucketVMeta, bucketWAL, bucketReadLog}
	bucketsToClean := make([]func(), 0)
	doBucketCleanup := func() {
		for _, fn := range bucketsToClean {
			fn()
		}
	}
	handleErr := func() {
		if err != nil {
			doBucketCleanup()
			t.Errorf("Filed to create bucket: %s", err)
		}
	}

	for _, b := range buckets {
		lb := b
		err = client.Bucket(b).Create(ctx, "onec-co", nil)
		handleErr()
		bucketsToClean = append(bucketsToClean, func() {
			deleteBucket(ctx, t, client, lb)
		})
	}

	datamonFlags.context.Descriptor.Metadata = bucketMeta
	datamonFlags.context.Descriptor.Blob = bucketBlob
	datamonFlags.context.Descriptor.VMetadata = bucketVMeta
	datamonFlags.context.Descriptor.WAL = bucketWAL
	datamonFlags.context.Descriptor.ReadLog = bucketReadLog
	c := setupConfig(t, datamonFlags)

	createAllTestUploadTrees(t)
	cleanup := func() {
		c()
		_ = os.RemoveAll(destinationDir)
		doBucketCleanup()
	}
	return cleanup
}

func runCmd(t *testing.T, cmd []string, intentMsg string, expectError bool, as ...AuthMock) {
	fatalCallsBefore := exitMocks.fatalCalls()
	bucketMeta := datamonFlags.context.Descriptor.Metadata
	bucketBlob := datamonFlags.context.Descriptor.Blob
	bucketWAL := datamonFlags.context.Descriptor.WAL
	bucketReadLog := datamonFlags.context.Descriptor.ReadLog
	bucketVMeta := datamonFlags.context.Descriptor.VMetadata
	config := datamonFlags.core.Config

	datamonFlags = flagsT{}
	datamonFlags.context.Descriptor.Metadata = bucketMeta
	datamonFlags.context.Descriptor.Blob = bucketBlob
	datamonFlags.context.Descriptor.WAL = bucketWAL
	datamonFlags.context.Descriptor.ReadLog = bucketReadLog
	datamonFlags.context.Descriptor.VMetadata = bucketVMeta
	datamonFlags.core.Config = config

	// test with metrics, depending on build flag
	datamonFlags.root.metrics.Enabled = testMetricsEnabled()

	if len(as) == 0 {
		authorizer = AuthMock{name: "tests", email: "datamon@oneconcern.com"}
	} else {
		for _, auth := range as {
			authorizer = auth
		}
	}

	datamonFlags.bundle.ID = ""
	rootCmd.SetArgs(cmd)
	require.NoError(t, rootCmd.Execute(), "error executing '"+strings.Join(cmd, " ")+"' : "+intentMsg)
	if expectError {
		require.Equal(t, fatalCallsBefore+1, exitMocks.fatalCalls(),
			"ran '"+strings.Join(cmd, " ")+"' expecting error and didn't see one in mocks : "+intentMsg)
	} else {
		require.Equal(t, fatalCallsBefore, exitMocks.fatalCalls(),
			"unexpected error in mocks on '"+strings.Join(cmd, " ")+"' : "+intentMsg)
	}
}
