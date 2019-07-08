// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"time"
	"strconv"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/dlogger"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage/localfs"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

type bundleReadAtRes struct {
	err error
}

var bundleReadAtCmd = &cobra.Command{
	Use:   "bundle-readat",
	Short: "Test memory usage of bundle ReadAt",
	Long:  "Diagnosing excess memory usage found in read-only bundle",
	Run: func(cmd *cobra.Command, args []string) {

		repoName := "repo-bundlereadat"
		localStoresPath, err := ioutil.TempDir("", "datamon-metrics-bundlereadat")
		if err != nil {
			log.Fatalln(err)
		}

		sourceStoreFilenames := make([]string, 0)
		for i := 0; i < 4; i++ {
			nextFileName := fmt.Sprintf("testfile_%v", i)
			sourceStoreFilenames = append(sourceStoreFilenames, nextFileName)
		}
		sourceStoreMax := 1024*1024*1
		sourceStore := newGenStoreRand(sourceStoreFilenames, int64(sourceStoreMax))

		metaStoreLocal := localfs.New(afero.NewBasePathFs(afero.NewOsFs(),
			filepath.Join(localStoresPath, "meta")))
		blobStoreLocal := localfs.New(afero.NewBasePathFs(afero.NewOsFs(),
			filepath.Join(localStoresPath, "blob")))

		consumableStoreLocal := localfs.New(afero.NewBasePathFs(afero.NewOsFs(),
			filepath.Join(localStoresPath, "consumable")))
		consumableStore := consumableStoreLocal

		metaStore := metaStoreLocal
		blobStore := blobStoreLocal

		repo := model.RepoDescriptor{
			Name:        repoName,
			Description: "metrics repo",
			Timestamp:   time.Now(),
			Contributor: model.Contributor{
				Name:  "contributors-name",
				Email: "contributors-email",
			},
		}
		err = core.CreateRepo(repo, metaStore)
		if err != nil {
			log.Fatalln(err)
		}

		uploadBundle := core.New(core.NewBDescriptor(),
			core.Repo(repoName),
			core.MetaStore(metaStore),
			core.BlobStore(blobStore),
			core.ConsumableStore(sourceStore),
		)
		memProfDir := core.MemProfDir
		err = core.Upload(context.Background(), uploadBundle)
		if err != nil {
			log.Fatalln(err)
		}
		core.MemProfDir = memProfDir

		streamBundle := core.New(core.NewBDescriptor(),
			core.Repo(repoName),
			core.BundleID(uploadBundle.BundleID),
			core.MetaStore(metaStore),
			core.BlobStore(blobStore),
			core.ConsumableStore(consumableStore),
			core.Streaming(true),
		)
		fsLogger, err := dlogger.GetLogger("info")
		if err != nil {
			log.Fatalln(err)
		}
		_, err = core.NewReadOnlyFS(streamBundle, fsLogger)
		if err != nil {
			log.Fatalln(err)
		}
		if len(streamBundle.BundleEntries) != len(sourceStoreFilenames) {
			log.Fatalln(fmt.Errorf("didn't find expected number of entries in stream bundle %v/%v",
				len(streamBundle.BundleEntries), len(sourceStoreFilenames)))
		}


		// dupe: bundle_pack.go
		writeMemProf := func(profName string) {
			var f *os.File
			path := filepath.Join(params.root.memProfPath, profName + ".mem.prof")
			f, err = os.Create(path)
			if err != nil {
				log.Fatalln(err)
			}
			err = pprof.Lookup("heap").WriteTo(f, 0)
			if err != nil {
				log.Fatalln(err)
			}
			f.Close()
		}

		bundleReadAtResC := make(chan bundleReadAtRes, 0)

		go func() {
			concurrencyControlC := make(chan struct{}, len(streamBundle.BundleEntries))
			for bundleEntryIdx, bundleEntry := range streamBundle.BundleEntries {
				concurrencyControlC <- struct{}{}
				go func() {
					defer func() { <-concurrencyControlC }()
					for i :=0; i < 1024 * 3; i++ {
						destination := make([]byte, 0)
						time.Sleep(time.Millisecond * time.Duration(bundleEntryIdx))
						_, err = core.BundleReadAtImpl(streamBundle,
							"",
							bundleEntry.Hash,
							destination, 0)
						if err != nil {
							bundleReadAtResC <- bundleReadAtRes{err: err}
							return
						}
					}
//					writeMemProf("bundlereadat-i-" + strconv.Itoa(bundleEntryIdx))
					bundleReadAtResC <- bundleReadAtRes{}
				}()
			}
			for i := 0; i < cap(concurrencyControlC); i++ {
				concurrencyControlC <- struct{}{}
			}
			close(bundleReadAtResC)
		}()

		writeMemProf("bundlereadat-initial")
		for i := 0; ; i++ {
			res, more := <-bundleReadAtResC
			if !more { break }
			if res.err != nil {
				log.Fatalln(res.err)
			}
			writeMemProf("bundlereadat-" + strconv.Itoa(i))
		}
		writeMemProf("bundlereadat-final")

	},
}

func init() {
	rootCmd.AddCommand(bundleReadAtCmd)
}
