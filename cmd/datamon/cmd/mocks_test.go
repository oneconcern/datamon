package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	gcsStorage "cloud.google.com/go/storage"
	"github.com/oneconcern/datamon/internal"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
)

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

// https://github.com/stretchr/testify/issues/610
func MakeFatalfMock(m *ExitMocks) func(string, ...interface{}) {
	return func(format string, v ...interface{}) {
		m.Fatalf(format, v...)
	}
}

func MakeFatallnMock(m *ExitMocks) func(...interface{}) {
	return func(v ...interface{}) {
		m.Fatalln(v...)
	}
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
	r := internal.RandStringBytesMaskImprSrc(15)
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
		//"--loglevel", "debug",
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

	btag := internal.RandStringBytesMaskImprSrc(15)

	bucketMeta := "datamon-deleteme-meta" + btag
	bucketBlob := "datamon-deleteme-blob" + btag
	bucketVMeta := "datamon-deleteme-vmeta" + btag
	bucketWAL := "datamon-deleteme-wal" + btag
	bucketReadLog := "datamon-deleteme-read-log" + btag

	client, err := gcsStorage.NewClient(ctx, option.WithScopes(gcsStorage.ScopeFullControl))
	require.NoError(t, err, "couldn't create bucket client")
	err = client.Bucket(bucketMeta).Create(ctx, "onec-co", nil)
	require.NoError(t, err, "couldn't create metadata bucket")
	err = client.Bucket(bucketBlob).Create(ctx, "onec-co", nil)
	require.NoError(t, err, "couldn't create blob bucket")
	err = client.Bucket(bucketWAL).Create(ctx, "onec-co", nil)
	require.NoError(t, err, "couldn't create wal bucket")
	err = client.Bucket(bucketReadLog).Create(ctx, "onec-co", nil)
	require.NoError(t, err, "couldn't create readLog bucket")
	err = client.Bucket(bucketVMeta).Create(ctx, "onec-co", nil)
	require.NoError(t, err, "couldn't create vMeta bucket")

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
		deleteBucket(ctx, t, client, bucketMeta)
		deleteBucket(ctx, t, client, bucketBlob)
		deleteBucket(ctx, t, client, bucketWAL)
		deleteBucket(ctx, t, client, bucketReadLog)
		deleteBucket(ctx, t, client, bucketVMeta)
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
