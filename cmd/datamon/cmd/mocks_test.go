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

func setupTests(t *testing.T) func() {
	os.RemoveAll(destinationDir)
	ctx := context.Background()
	exitMocks = NewExitMocks()
	logFatalf = MakeFatalfMock(exitMocks)
	logFatalln = MakeFatallnMock(exitMocks)
	osExit = MakeExitMock(exitMocks)
	btag := internal.RandStringBytesMaskImprSrc(15)
	bucketMeta := "datamontestmeta-" + btag
	bucketBlob := "datamontestblob-" + btag

	client, err := gcsStorage.NewClient(ctx, option.WithScopes(gcsStorage.ScopeFullControl))
	require.NoError(t, err, "couldn't create bucket client")
	err = client.Bucket(bucketMeta).Create(ctx, "onec-co", nil)
	require.NoError(t, err, "couldn't create metadata bucket")
	err = client.Bucket(bucketBlob).Create(ctx, "onec-co", nil)
	require.NoError(t, err, "couldn't create blob bucket")
	params.repo.MetadataBucket = bucketMeta
	params.repo.BlobBucket = bucketBlob
	createAllTestUploadTrees(t)
	cleanup := func() {
		os.RemoveAll(destinationDir)
		deleteBucket(ctx, t, client, bucketMeta)
		deleteBucket(ctx, t, client, bucketBlob)
	}
	return cleanup
}

func runCmd(t *testing.T, cmd []string, intentMsg string, expectError bool, as ...AuthMock) {
	fatalCallsBefore := exitMocks.fatalCalls()
	bucketMeta := params.repo.MetadataBucket
	bucketBlob := params.repo.BlobBucket
	params = paramsT{}
	params.repo.MetadataBucket = bucketMeta
	params.repo.BlobBucket = bucketBlob

	if len(as) == 0 {
		authorizer = AuthMock{name: "tests", email: "datamon@oneconcern.com"}
	} else {
		for _, auth := range as {
			authorizer = auth
		}
	}

	params.bundle.ID = ""
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
