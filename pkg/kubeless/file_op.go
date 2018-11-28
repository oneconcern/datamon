package kubeless

import (
	"archive/zip"
	"fmt"

	"github.com/bmatcuk/doublestar"

	"io"
	"io/ioutil"
	"log"
	"os"
)

const (
	ZipExtension = "zip"
)

// Zip file method take list of directories or files from content attribute and
// target name of zip file and zip input files.
func ZipFile(content []string, target string) (string, error) {
	zipfile, err := createZipFile(target)
	if err != nil {
		return "", fmt.Errorf("creating %q: %v", zipfile.Name(), err)
	}

	archive := zip.NewWriter(zipfile)

	for _, source := range content {
		matches, err := doublestar.Glob(source)
		if err != nil {
			return "", fmt.Errorf("glob pattern %q: %v", source, err)
		}

		for _, contentDir := range matches {
			fileInfo, err := os.Stat(contentDir)
			if err != nil {
				return "", fmt.Errorf("content directory stats %q: %v", contentDir, err)
			}
			if !fileInfo.IsDir() {
				err = archiveContent(contentDir, archive)
				if err != nil {
					return "", fmt.Errorf("archiving failing %q: %v", contentDir, err)
				}
			}
		}
	}

	if err := archive.Close(); err != nil {
		return "", fmt.Errorf("archive close %v ", err)
	}

	if err := zipfile.Close(); err != nil {
		return "", fmt.Errorf("zipFile close %q : %v", zipfile.Name(), err)
	}

	log.Printf("zip file created in directory location %s ", zipfile.Name())
	return zipfile.Name(), nil
}

func createZipFile(target string) (*os.File, error) {
	zipfile, err := ioutil.TempFile("", target+"-*."+ZipExtension)
	if err != nil {
		return nil, err
	}

	return zipfile, nil
}

// Archiving files into a single zip file
func archiveContent(contentToZip string, archive *zip.Writer) error {
	zipfile, err := os.Open(contentToZip)
	if err != nil {
		return err
	}

	// Get the file information
	info, err := zipfile.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	header.Name = contentToZip

	header.Method = zip.Deflate

	writer, err := archive.CreateHeader(header)
	if err != nil {
		return err

	}
	if _, err = io.Copy(writer, zipfile); err != nil {
		return err
	}

	return zipfile.Close()
}
