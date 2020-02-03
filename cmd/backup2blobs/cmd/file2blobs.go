package cmd

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/oneconcern/datamon/pkg/storage/gcs"
	"github.com/oneconcern/datamon/pkg/storage/localfs"
	"github.com/spf13/afero"

	"github.com/spf13/cobra"

	"github.com/oneconcern/datamon/pkg/model"

	"gopkg.in/yaml.v2"

	"github.com/oneconcern/datamon/pkg/cafs"
	"github.com/oneconcern/datamon/pkg/storage"
)

func UploadToBlob(sourceStore storage.Store, backupStore storage.Store, cafs cafs.Fs, fileChan chan string, wg *sync.WaitGroup, c *uint64, errC *uint64, duplicateCount *uint64) {
	logError := log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile)
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		wg.Done()
		logError.Fatalln(err)
		return
	}

	incC := func(count *uint64) {
		atomic.AddUint64(count, 1)
	}
	for {
		file, found := <-fileChan
		if !found {
			_ = logger.Sync()
			wg.Done()
			return
		}
		reader, err := sourceStore.Get(context.Background(), file)
		if err != nil {
			logger.Error("Failed to read file", zap.String("file", file), zap.Error(err))
			incC(errC)
			continue
		}
		putRes, err := cafs.Put(context.Background(), reader)
		size := putRes.Written
		key := putRes.Key
		duplicate := putRes.Found
		if err != nil {
			logger.Error("Failed to upload blob", zap.String("file", file), zap.Error(err))
			incC(errC)
			continue
		}
		if duplicate {
			incC(duplicateCount)
		}
		backingStoreEntry := model.BundleEntry{
			Size:         uint64(size),
			Hash:         key.String(),
			NameWithPath: file,
			FileMode:     os.ModePerm,
		}
		buffer, err := yaml.Marshal(backingStoreEntry)
		if err != nil {
			logger.Error("Failed to serialize", zap.String("file", file), zap.Error(err))
			incC(errC)
			reader.Close()
			continue
		}
		err = backupStore.Put(context.Background(), file, bytes.NewReader(buffer), storage.OverWrite)
		status := "success"
		if err != nil {
			if strings.Contains(err.Error(), "googleapi: Error 412:") {
				status = "File descriptor exists"
			} else {
				incC(errC)
			}
		}

		incC(c)
		logger.Info("Migrate complete", zap.String("file", file),
			zap.String("status", status),
			zap.String("key", key.String()),
			zap.Int64("size", size),
			zap.Uint64("total", atomic.LoadUint64(c)),
			zap.Bool("duplicate", duplicate),
		)
		if atomic.LoadUint64(c)%1000 == 0 {
			logger.Info("summary",
				zap.Uint64("total", atomic.LoadUint64(c)),
				zap.Uint64("errors", atomic.LoadUint64(errC)),
				zap.Uint64("duplicate", atomic.LoadUint64(duplicateCount)))
		}
		reader.Close()
	}
}

func publishFiles(fileList string, fileChan chan string, startFrom int, staggeredStart bool, wg *sync.WaitGroup) {
	file, err := os.Open(fileList)
	if err != nil {
		logger.Error("Failed to open file", zap.String("files", fileList), zap.Error(err))
		close(fileChan)
		wg.Done()
		return
	}
	defer file.Close()
	lineScanner := bufio.NewScanner(file)
	for i := startFrom; i > 0; i-- {
		lineScanner.Scan()
	}
	for lineScanner.Scan() {
		fileChan <- lineScanner.Text()
		if staggeredStart {
			time.Sleep(time.Millisecond * 500)
			staggeredStart = false
		}
	}
	close(fileChan)
	wg.Done()
}

func ProcessFiles(fileList string, sourceStore storage.Store, backupStore storage.Store, cafs cafs.Fs, maxC int, startFrom int) (err error) {
	var fileCount uint64
	var errCount uint64
	var duplicateCount uint64
	var wg sync.WaitGroup
	wg.Add(maxC)

	var fileChan = make(chan string)

	go publishFiles(fileList, fileChan, startFrom, true, &wg)

	for i := 1; i < maxC; i++ {
		go UploadToBlob(sourceStore, backupStore, cafs, fileChan, &wg, &fileCount, &errCount, &duplicateCount)
	}
	logger.Info("Waiting for routines")
	wg.Wait()
	logger.Info("Finished processing", zap.String("files", fileList), zap.Uint64("Total", fileCount), zap.Uint64("errors", errCount))
	return nil
}

var upload = &cobra.Command{
	Use:   "upload2blob",
	Short: " Upload a files in a new line separated fileList",
	Long:  `Tool to bulk import files into CAFS with a record of the files in the backing store.`,
	Run: func(cmd *cobra.Command, args []string) {
		localStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), params.pathToMount))
		backupStore, err := gcs.New(context.TODO(), params.backendStoreBucket, "")
		if err != nil {
			log.Fatalln(err)
		}
		cafsStore, err := gcs.New(context.TODO(), params.blobStoreBucket, "")
		if err != nil {
			log.Fatalln(err)
		}
		cafs, err := cafs.New(
			cafs.LeafSize(cafs.DefaultLeafSize),
			cafs.Backend(cafsStore))
		if err != nil {
			log.Fatalln(err)
		}
		err = ProcessFiles(params.fileList, localStore, backupStore, cafs, params.maxConcurrency, params.startFrom)
		if err != nil {
			log.Fatalln(err)
		}
	},
}

var rootCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Commands to help migrate data to datamon",
	Long:  "This tools helps generate a list of files then uploads it to CAFS based FS",
}

var logger *zap.Logger

func Execute() {
	logger, _ = zap.NewProduction()
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var maxConcurrency = 100
var params struct {
	fileList           string
	pathToMount        string
	backendStoreBucket string
	blobStoreBucket    string
	maxConcurrency     int
	startFrom          int
}

func init() {
	upload.Flags().StringVarP(&params.pathToMount, "parent", "p", "", "Path to the parent directory for source")
	err := upload.MarkFlagRequired("parent")
	if err != nil {
		log.Fatalln(err)
	}
	upload.Flags().StringVarP(&params.backendStoreBucket, "backend-bucket", "b", "", "Bucket name for storing list of files backedup")
	err = upload.MarkFlagRequired("backend-bucket")
	if err != nil {
		log.Fatalln(err)
	}
	upload.Flags().StringVarP(&params.blobStoreBucket, "cafs-bucket", "c", "", "Bucket name for storing cafs blobs")
	err = upload.MarkFlagRequired("cafs-bucket")
	if err != nil {
		log.Fatalln(err)
	}
	upload.Flags().StringVarP(&params.fileList, "files", "f", "", "File containing list of files to upload")
	err = upload.MarkFlagRequired("files")
	if err != nil {
		log.Fatalln(err)
	}
	upload.Flags().IntVarP(&params.maxConcurrency, "concurrency", "t", maxConcurrency, fmt.Sprintf("Max number of concurrent go routines, default:%d", maxConcurrency))
	upload.Flags().IntVarP(&params.startFrom, "start", "s", 0, "Starting line number to read from.")
	rootCmd.AddCommand(upload)
}
