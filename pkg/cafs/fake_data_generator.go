package cafs

import (
	"context"

	"github.com/oneconcern/datamon/internal"

	"io/ioutil"
	"os"
	"path/filepath"
)

func GenerateFile(tgt string, size int, leafSize uint32) error {
	f, err := os.Create(tgt)
	if err != nil {
		return err
	}
	defer f.Close()

	if size <= int(leafSize) { // small single chunk file
		_, err := f.WriteString(internal.RandStringBytesMaskImprSrc(size))
		if err != nil {
			return err
		}
		return f.Close()
	}

	var parts = size / int(leafSize)
	var i int
	for i = 0; i < parts; i++ {
		_, err := f.WriteString(internal.RandStringBytesMaskImprSrc(int(leafSize)))
		if err != nil {
			return err
		}
	}
	remaining := size - (parts * int(leafSize))
	if remaining > 0 {
		_, err := f.WriteString(internal.RandStringBytesMaskImprSrc(remaining))
		if err != nil {
			return err
		}
	}
	return f.Close()
}

func GenerateCAFSFile(src string, fs Fs, destDir string) error {

	key, err := GenerateCAFSChunks(src, fs)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(destDir, "roots", filepath.Base(src)), []byte(key.String()), 0600)
}

func GenerateCAFSChunks(src string, fs Fs) (*Key, error) {
	sourceFile, err := os.Open(src)
	if err != nil {
		return nil, err
	}
	defer sourceFile.Close()

	key, _, err := fs.Put(context.Background(), sourceFile)
	if err != nil {
		return nil, err
	}
	return &key, nil
}
