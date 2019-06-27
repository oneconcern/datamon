package cafs

import (
	"context"
	"fmt"
	"path"

	"github.com/oneconcern/datamon/internal"

	"io/ioutil"
	"os"
	"path/filepath"
)

func GenerateFile(tgt string, size int, leafSize uint32) error {
	err := os.MkdirAll(path.Dir(tgt), os.ModePerm)
	if err != nil {
		fmt.Printf("Unable to create file:%s, err:%s\n", tgt, err)
		return err
	}
	f, err := os.Create(tgt)
	if err != nil {
		fmt.Printf("Unable to create file:%s, err:%s\n", tgt, err)
		return err
	}
	if err = f.Sync(); err != nil {
		fmt.Printf("Unable to sync file:%s, err:%s\n", tgt, err)
		return err
	}

	if size <= int(leafSize) { // small single chunk file
		_, err = f.WriteString(internal.RandStringBytesMaskImprSrc(size))
		if err != nil {
			return err
		}
		return f.Close()
	}

	var parts = size / int(leafSize)
	var i int
	for i = 0; i < parts; i++ {
		_, err = f.WriteString(internal.RandStringBytesMaskImprSrc(int(leafSize)))
		if err != nil {
			return err
		}
	}
	remaining := size - (parts * int(leafSize))
	if remaining > 0 {
		_, err = f.WriteString(internal.RandStringBytesMaskImprSrc(remaining))
		if err != nil {
			return err
		}
	}
	if err = f.Sync(); err != nil {
		fmt.Printf("Unable to sync file:%s, err:%s\n", tgt, err)
		return err
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
		fmt.Printf("Failed to open file:%s, err:%s\n", src, err)
		return nil, err
	}
	defer sourceFile.Close()

	putRes, err := fs.Put(context.Background(), sourceFile)
	if err != nil {
		return nil, err
	}
	return &putRes.Key, nil
}
