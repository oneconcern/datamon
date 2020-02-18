package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	context2 "github.com/oneconcern/datamon/pkg/context"

	"github.com/oneconcern/datamon/internal"
	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/oneconcern/datamon/pkg/storage/gcs"

	gcsStorage "cloud.google.com/go/storage"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// GCSParams defines some parameters for a Google Cloud Storage context
type GCSParams struct {
	MetadataBucket  string
	BlobBucket      string
	VMetadataBucket string
	WAL             string
	ReadLog         string
}

var gcsParams GCSParams

// dupe: cli_test.go:deleteBucket
func deleteBucket(ctx context.Context, client *gcsStorage.Client, bucketName string) {
	mb := client.Bucket(bucketName)
	oi := mb.Objects(ctx, &gcsStorage.Query{})
	for {
		objAttrs, err := oi.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("error iterating: %v", err)
		}
		obj := mb.Object(objAttrs.Name)
		if err := obj.Delete(ctx); err != nil {
			log.Fatalf("error deleting object: %v", err)
		}
	}
	if err := mb.Delete(ctx); err != nil {
		log.Fatalf("error deleting bucket %v", err)
	}
}

func setupBuckets() (func(), error) {
	ctx := context.Background()
	btag := internal.RandStringBytesMaskImprSrc(15)
	bucketMeta := "datamonmetricsmeta-" + btag
	bucketBlob := "datamonmetricsblob-" + btag
	client, err := gcsStorage.NewClient(context.TODO(), option.WithScopes(gcsStorage.ScopeFullControl))
	if err != nil {
		return nil, err
	}
	err = client.Bucket(bucketMeta).Create(ctx, "onec-co", nil)
	if err != nil {
		return nil, err
	}
	err = client.Bucket(bucketBlob).Create(ctx, "onec-co", nil)
	if err != nil {
		return nil, err
	}
	gcsParams.MetadataBucket = bucketMeta
	gcsParams.BlobBucket = bucketBlob
	deleteBuckets := func() {
		deleteBucket(ctx, client, bucketMeta)
		deleteBucket(ctx, client, bucketBlob)
		gcsParams = GCSParams{}
	}
	return deleteBuckets, nil
}

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload a bundle",
	Long:  "Upload a bundle of randomly-generated data.",
	Run: func(cmd *cobra.Command, args []string) {
		gcsCredential := func() string {
			gcsCredential, foundCreds := os.LookupEnv("GOOGLE_APPLICATION_CREDENTIALS")
			if !foundCreds {
				log.Fatalln("didn't find GOOGLE_APPLICATION_CREDENTIALS in env")
			}
			return gcsCredential
		}()

		var metaStore storage.Store
		var blobStore storage.Store
		var vmetadata storage.Store
		var wal storage.Store
		var readLog storage.Store
		if params.upload.mockDest {
			metaStore = newMockDestStore("meta", logger)
			blobStore = newMockDestStore("blob", logger)
			vmetadata = newMockDestStore("vmetadata", logger)
			wal = newMockDestStore("wal", logger)
			readLog = newMockDestStore("readLog", logger)
		} else {
			deleteBuckets, err := setupBuckets()
			if err != nil {
				log.Fatalln(err)
			}
			defer deleteBuckets()
			if metaStore, err = gcs.New(context.TODO(), gcsParams.MetadataBucket, gcsCredential); err != nil {
				log.Fatalln(err)
			}
			if blobStore, err = gcs.New(context.TODO(), gcsParams.BlobBucket, gcsCredential); err != nil {
				log.Fatalln(err)
			}
			if vmetadata, err = gcs.New(context.TODO(), gcsParams.VMetadataBucket, gcsCredential); err != nil {
				log.Fatalln(err)
			}
			if wal, err = gcs.New(context.TODO(), gcsParams.WAL, gcsCredential); err != nil {
				log.Fatalln(err)
			}
			if readLog, err = gcs.New(context.TODO(), gcsParams.ReadLog, gcsCredential); err != nil {
				log.Fatalln(err)
			}
		}
		dmc := context2.NewStores(wal, readLog, blobStore, metaStore, vmetadata)

		var err error
		sourceStore := func() storage.Store {
			var s storage.Store
			filenames := make([]string, 0)
			numFiles := params.upload.numFiles
			for i := 0; i < numFiles; i++ {
				nextFileName := fmt.Sprintf("testfile_%v", i)
				filenames = append(filenames, nextFileName)
			}
			stripe := []byte{0xDE, 0xDB, 0xEF}
			max := int64(1024 * 1024 * params.upload.max)
			if max < 1 {
				log.Fatalln("less than one byte based on filesize param")
			}
			numChunks := params.upload.numChunks
			switch params.upload.fileType {
			case "stripe":
				s = newGenStoreRepeatingStripes(filenames, max, stripe)
			case "rand":
				s = newGenStoreRand(filenames, max)
			case "chunks":
				s = newGenStoreZeroOneChunks(filenames, max, max/int64(numChunks))
			default:
				log.Fatalln("upload file type must be among 'chunks', 'stripe', 'rand'")
			}
			return s
		}()

		contributorsTag := internal.RandStringBytesMaskImprSrc(15)
		contributors := []model.Contributor{{
			Name:  "contributors-name-" + contributorsTag,
			Email: "contributors-email-" + contributorsTag,
		}}
		bd := model.NewBundleDescriptor(
			model.Message("metrics bundle upload"),
			model.BundleContributors(contributors),
		)
		repoTag := internal.RandStringBytesMaskImprSrc(15)

		repoName := "repo-" + repoTag

		repo := model.RepoDescriptor{
			Name:        repoName,
			Description: "metrics repo",
			Timestamp:   time.Now(),
			Contributor: contributors[0],
		}
		err = core.CreateRepo(repo, dmc)
		if err != nil {
			log.Fatalln(err)
		}

		bundle := core.NewBundle(
			core.BundleDescriptor(bd),
			core.Repo(repoName),
			core.ContextStores(dmc),
			core.ConsumableStore(sourceStore),
			core.Logger(logger),
		)

		logger.Debug("beginning upload")
		var opts []core.Option
		if params.root.memProfPath != "" {
			// add some extra memory profiling on specific critical parts (i.e. bundle Upload)
			opts = append(opts, core.WithMemProf(params.root.memProfPath))
		}
		err = core.Upload(context.Background(), bundle, opts...)
		if err != nil {
			log.Fatalln(err)
		}

		logger.Debug("upload done")

		metaMockStore, ok := metaStore.(*mockDestStore)
		if ok {
			logger.Info("mock meta store info",
				zap.Int("filelist put cnt", metaMockStore.fileListUploadPutCnt),
				zap.String("store name", metaMockStore.name),
			)
		}

	},
}

func init() {

	addUploadFilesize(uploadCmd)
	addUploadNumFiles(uploadCmd)
	addUploadNumChunks(uploadCmd)
	addUploadFileType(uploadCmd)
	addUploadMockDest(uploadCmd)

	rootCmd.AddCommand(uploadCmd)
}
