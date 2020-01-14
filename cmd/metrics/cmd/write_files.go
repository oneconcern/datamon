package cmd

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/oneconcern/datamon/pkg/cafs"
	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/oneconcern/datamon/pkg/storage/localfs"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// DieIfNotAccessible exits the process if the path is not accessible.
func DieIfNotAccessible(path string) {
	_, err := os.Stat(path)
	if err != nil {
		log.Fatalln(err)
	}
}

func sanitizePath(path string) string {
	sanitizedPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		log.Fatalln(err)
	}
	return sanitizedPath
}

func createPath(path string) {
	// todo: determine proper permission bits.  previously 0700.
	err := os.MkdirAll(path, 0777)
	if err != nil {
		log.Fatalln(err)
	}
}

func DieIfNotDirectory(path string) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		log.Fatalln(err)
	}
	if !fileInfo.IsDir() {
		log.Fatalln("'" + path + "' is not a directory")
	}
}

var writeFilesCmd = &cobra.Command{
	Use:   "write-files",
	Short: "Write test data files",
	Long: `Write some files consisting of various byte patterns to disk.
This command doesn't itself invoke datamon routines to collect metrics on them
yet is generally part of the datamon metrics collection and benchmarking picture.`,
	Run: func(cmd *cobra.Command, args []string) {

		sourceStore := func() storage.Store {
			var s storage.Store
			filenames := make([]string, 0)
			numFiles := params.writeFiles.numFiles
			for i := 0; i < numFiles; i++ {
				nextFileName := fmt.Sprintf("testfile_%v", i)
				filenames = append(filenames, nextFileName)
			}
			max := int64(1024 * 1024 * params.writeFiles.fileSize)
			s = newGenStoreZeroOneChunks(filenames, max, int64(cafs.DefaultLeafSize))
			return s
		}()

		destStore := func() storage.Store {
			var s storage.Store
			destStorePath := sanitizePath(params.writeFiles.outDir)
			logger.Info("setting destination store",
				zap.String("path", destStorePath),
			)
			createPath(destStorePath)
			DieIfNotAccessible(destStorePath)
			DieIfNotDirectory(destStorePath)
			s = localfs.New(afero.NewBasePathFs(afero.NewOsFs(), destStorePath))
			return s
		}()

		ctx := context.Background()
		srcKeys, err := sourceStore.Keys(ctx)
		if err != nil {
			log.Fatalln(err)
		}
		logger.Info("preparing to write files",
			zap.Int("num files", len(srcKeys)),
		)
		cc := make(chan struct{}, params.writeFiles.parallelWrites)
		for idx, key := range srcKeys {
			logger.Info("writing file",
				zap.String("key", key),
				zap.Int("idx", idx),
			)
			rdr, err := sourceStore.Get(ctx, key)
			if err != nil {
				log.Fatalln(err)
			}
			cc <- struct{}{}
			go func(key string, rdr io.Reader, idx int) {
				defer func() { <-cc }()
				err := destStore.Put(ctx, key, rdr, storage.NoOverWrite)
				if err != nil {
					log.Fatalln(err)
				}
				logger.Info(" file written",
					zap.String("key", key),
					zap.Int("idx", idx),
				)
			}(key, rdr, idx)
		}

		logger.Info("waiting on all writes to finish")
		for i := 0; i < cap(cc); i++ {
			cc <- struct{}{}
		}
	},
}

func init() {
	addWriteFilesOutDir(writeFilesCmd)
	addWriteFilesFilesize(writeFilesCmd)
	addWriteFilesNumFiles(writeFilesCmd)
	addWriteFilesParallelWrites(writeFilesCmd)

	rootCmd.AddCommand(writeFilesCmd)
}
