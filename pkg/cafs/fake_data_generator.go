package cafs

import (
	"context"
	"path"

	"golang.org/x/sync/errgroup"

	"github.com/oneconcern/datamon/internal/rand"
	"github.com/oneconcern/datamon/pkg/errors"

	"io/ioutil"
	"os"
	"path/filepath"
)

// GenerateFile is a test utility.
//
// It builds a file on the local file system with some random content
func GenerateFile(target string, size int, leafSize uint32) (e error) {
	err := os.MkdirAll(path.Dir(target), os.ModePerm)
	if err != nil {
		return errors.New("unable to create dir").WrapMessage("%s", target).Wrap(err)
	}

	f, err := os.Create(target)
	if err != nil {
		return errors.New("unable to create file").WrapMessage("%s", target).Wrap(err)
	}

	defer func() {
		e = f.Close()
		if err != nil {
			e = err
		}
	}()

	generator := rand.Bytes

	if leafSize == 0 {
		leafSize = DefaultLeafSize
	}
	leaf := int(leafSize)

	if size <= leaf { // small single chunk file
		_, err = f.Write(generator(size))
		return
	}

	var (
		parts = size / leaf
		i     int
		wg    errgroup.Group
	)

	for i = 0; i < parts; i++ {
		wg.Go(func(idx int) func() error {
			return func() error {
				_, erw := f.WriteAt(rand.Bytes(leaf), int64(idx*leaf))
				return erw
			}
		}(i))
	}
	err = wg.Wait()
	if err != nil {
		return
	}
	_, err = f.Seek(0, 2)
	if err != nil {
		return
	}
	remaining := size - (parts * leaf)
	if remaining > 0 {
		_, err = f.Write(generator(remaining))
		if err != nil {
			return
		}
	}
	//nolint:nakedret
	return
}

// GenerateCAFSFile is a test utility.
//
// It puts some local file into a cafs store, then archive the root key to retrieve the content
func GenerateCAFSFile(src string, fs Fs, destDir string) error {
	key, err := GenerateCAFSChunks(src, fs)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(destDir, "roots", filepath.Base(src)), []byte(key.String()), 0600)
}

// GenerateCAFSChunks is a test utility.
//
// It copies a source file to a cafs store.
func GenerateCAFSChunks(src string, fs Fs) (*Key, error) {
	sourceFile, err := os.Open(src)
	if err != nil {
		return nil, errors.New("failed to open file").WrapMessage("%s", src).Wrap(err)
	}
	defer sourceFile.Close()

	putRes, err := fs.Put(context.Background(), sourceFile)
	if err != nil {
		return nil, err
	}
	return &putRes.Key, nil
}
