package cmd

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/oneconcern/datamon/pkg/model"
	"gopkg.in/yaml.v2"

	"go.uber.org/zap"

	"github.com/oneconcern/datamon/pkg/storage"

	"github.com/oneconcern/datamon/pkg/cafs"
	"github.com/oneconcern/datamon/pkg/storage/gcs"
	"github.com/oneconcern/datamon/pkg/storage/localfs"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// Due to a race condition in uploading duplicate blobs, some objects got a bad hash, these were reuploaded but this is to verify.
// Github Issue #67

// BadHash should never occur
var BadHash = "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"

func downloadFromBlob(destination storage.Store, backupStore storage.Store, c cafs.Fs, fileChan chan string, wg *sync.WaitGroup, fileCount *uint64, errCount *uint64) {
	incC := func(count *uint64) {
		atomic.AddUint64(count, 1)
	}
	log, _ := zap.NewProduction()
	var r io.ReadCloser
	for {
		file, ok := <-fileChan
		if !ok {
			wg.Done()
			return
		}

		if !strings.HasPrefix(file, b2fParams.prefix) {
			continue
		}

		incC(fileCount)
		var err error
		if r != nil {
			r.Close()
		}
		r, err = backupStore.Get(context.Background(), file)
		if err != nil {
			log.Error("Failed to get object", zap.String("file", file), zap.Error(err))
			incC(errCount)
			continue
		}
		var entry model.BundleEntry
		b, err := ioutil.ReadAll(r)
		if err != nil {
			log.Error("Read failed", zap.Error(err))
			continue
		}
		err = yaml.Unmarshal(b, &entry)
		if err != nil {
			log.Error("Unmarshal failed", zap.String("file", file))
			incC(errCount)
			continue
		}

		key, err := cafs.KeyFromString(entry.Hash)
		if err != nil {
			log.Error("Get Keys failed", zap.Error(err))
			continue
		}

		r, err = c.Get(context.Background(), key)
		if err != nil {
			log.Error("Read from CAFS failed", zap.String("file", file), zap.String("hash", entry.Hash), zap.Error(err))
			incC(errCount)
			continue
		}
		err = destination.Put(context.Background(), file, r, storage.IfNotPresent)
		if err != nil {
			log.Error("Put from CAFS to dest failed", zap.String("file", file), zap.Error(err))
			incC(errCount)
			continue
		}
		logger.Info("Complete",
			zap.String("file", file),
			zap.Uint64("size", entry.Size),
			zap.String("key", entry.Hash),
			zap.Uint64("total", atomic.LoadUint64(fileCount)),
			zap.Uint64("errors", atomic.LoadUint64(errCount)))
		if atomic.LoadUint64(fileCount)%1000 == 0 {
			logger.Info("summary",
				zap.Uint64("total", atomic.LoadUint64(fileCount)),
				zap.Uint64("errors", atomic.LoadUint64(errCount)))
		}
	}
}

func startBlob2File(input string, destination storage.Store, backupStore storage.Store, cafs cafs.Fs, routines int, startFrom int) error {
	var fileCount uint64
	var errCount uint64
	var wg sync.WaitGroup
	wg.Add(routines)

	var fileChan = make(chan string)

	go publishFiles(input, fileChan, startFrom, true, &wg)

	for i := 1; i < routines; i++ {
		go downloadFromBlob(destination, backupStore, cafs, fileChan, &wg, &fileCount, &errCount)
	}
	logger.Info("Waiting for routines")
	wg.Wait()
	logger.Info("Finished processing", zap.String("files", input), zap.Uint64("Total", fileCount), zap.Uint64("errors", errCount))
	return nil
}

var download = &cobra.Command{
	Use:   "blob2file",
	Short: "Download files stored in blobs",
	Long:  "Download files that were migrated to a cafs blob store back to a non cafs blob store",
	Run: func(cmd *cobra.Command, args []string) {
		// Create CAFS based on the blob store
		localStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), b2fParams.destination))

		backupStore, err := gcs.New(b2fParams.backendStoreBucket)

		if err != nil {
			log.Fatalln(err)
		}

		cafsStore, err := gcs.New(b2fParams.blobStoreBucket)
		if err != nil {
			log.Fatalln(err)
		}

		cafs, err := cafs.New(
			cafs.LeafSize(cafs.DefaultLeafSize),
			cafs.Backend(cafsStore))
		if err != nil {
			log.Fatalln(err)
		}

		err = startBlob2File(b2fParams.inputFile, localStore, backupStore, cafs, b2fParams.maxConcurrency, b2fParams.startFrom)
		if err != nil {
			log.Fatalln(err)
		}
	},
}

var b2fParams struct {
	inputFile          string
	destination        string
	backendStoreBucket string
	blobStoreBucket    string
	maxConcurrency     int
	startFrom          int
	prefix             string
}

func init() {
	download.Flags().StringVarP(&b2fParams.inputFile, "files", "f", "", "File containing list of files to restore")
	err := download.MarkFlagRequired("files")
	if err != nil {
		log.Fatalln(err)
	}
	download.Flags().StringVarP(&b2fParams.destination, "destination", "d", "", "Path to the parent directory for destination")
	err = download.MarkFlagRequired("destination")
	if err != nil {
		log.Fatalln(err)
	}
	download.Flags().StringVarP(&b2fParams.backendStoreBucket, "backend-bucket", "b", "", "Bucket name for list of files backed")
	err = download.MarkFlagRequired("backend-bucket")
	if err != nil {
		log.Fatalln(err)
	}
	download.Flags().StringVarP(&b2fParams.blobStoreBucket, "cafs-bucket", "c", "", "Bucket name cafs blobs")
	err = download.MarkFlagRequired("cafs-bucket")
	if err != nil {
		log.Fatalln(err)
	}
	download.Flags().IntVarP(&b2fParams.maxConcurrency, "concurrency", "t", maxConcurrency, fmt.Sprintf("Max number of concurrent go routines, default:%d", maxConcurrency))
	download.Flags().IntVarP(&b2fParams.startFrom, "start", "s", 0, "Starting line number to read from.")
	download.Flags().StringVarP(&b2fParams.prefix, "prefix", "p", "", "prefix for files to include")
	rootCmd.AddCommand(download)
}
