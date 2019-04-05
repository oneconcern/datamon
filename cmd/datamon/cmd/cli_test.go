package cmd

import (
	"bytes"
	"context"
	"log"
	"os"
	"testing"

	"github.com/oneconcern/datamon/pkg/storage"

	"github.com/oneconcern/datamon/pkg/cafs"

	"github.com/oneconcern/datamon/pkg/storage/localfs"
	"github.com/spf13/afero"

	gcsStorage "cloud.google.com/go/storage"
	"github.com/oneconcern/datamon/internal"
	"google.golang.org/api/option"
)

const (
	destinationDir = "../../../testdata/cli"
	sourceData     = destinationDir + "/data"
	repo1          = "test-repo1"
	repo2          = "test-repo2"
)

func setupTests() func() {
	os.RemoveAll(destinationDir)
	ctx := context.Background()
	bucket := "datamontest-" + internal.RandStringBytesMaskImprSrc(15)

	client, err := gcsStorage.NewClient(context.TODO(), option.WithScopes(gcsStorage.ScopeFullControl))
	err = client.Bucket(bucket).Create(ctx, "oneconcern-1509", nil)
	if err != nil {
		log.Fatalln(err)
	}
	createTree()
	cleanup := func() {
		os.RemoveAll(destinationDir)
		log.Printf("Delete bucket %s ", bucket)
		_ = client.Bucket(bucket).Delete(ctx)
	}
	return cleanup
}

func TestCreateRepo(t *testing.T) {
	cleanup := setupTests()
	defer cleanup()
	rootCmd.SetArgs([]string{"repo",
		"create",
		"--description", "testing",
		"--repo", "ritesh-datamon-test-repo",
		"--name", "tests",
		"--email", "datamon@oneconcern.com",
	})
	rootCmd.Execute()
}

type uploadTree struct {
	path string
	size int
	data []byte
}

var testUploadTree = []uploadTree{
	{
		path: "/small/1k",
		size: 1024,
	},
	{
		path: "/leafs/leafsize",
		size: cafs.DefaultLeafSize,
	},
	{
		path: "/leafs/over-leafsize",
		size: cafs.DefaultLeafSize + 1,
	},
	{
		path: "/leafs/under-leafsize",
		size: cafs.DefaultLeafSize - 1,
	},
	{
		path: "/leafs/multiple-leafsize",
		size: cafs.DefaultLeafSize * 3,
	},
	{
		path: "/leafs/root",
		size: 1,
	},
	{
		path: "/1/2/3/4/5/6/deep",
		size: 100,
	},
	{
		path: "/1/2/3/4/5/6/7/deeper",
		size: 200,
	},
}

func createTree() {
	sourceFS := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), sourceData))
	for _, file := range testUploadTree {
		err := sourceFS.Put(context.Background(),
			file.path,
			bytes.NewReader(internal.RandBytesMaskImprSrc(file.size)),
			storage.IfNotPresent)
		if err != nil {
			log.Fatalln(err)
		}
	}
}
