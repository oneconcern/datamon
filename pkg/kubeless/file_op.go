package kubeless

import (
	"archive/zip"
	"github.com/bmatcuk/doublestar"
	"io"
	"io/ioutil"
	"log"
	"os"
)

const (
	ZipExtension  = "zip"
)

// Zip file method take list of directories or files from content attribute and
// target name of zip file and zip input files.
func ZipFile(content []string, target string) (string) {
	zipfile, err := createZipFile(target)
	if err != nil {
		log.Fatalf("error creating zip file: %s, Error: %v ",zipfile.Name(), err)

	}

	archive := zip.NewWriter(zipfile)

	for _, source := range content {
		matches, err := doublestar.Glob(source)
		if err != nil  {
			log.Fatalf("glob pattern is throwing err %v ", err)
		}

		for _, contentDir := range matches {
			fileInfo, err := os.Stat(contentDir)
			if err != nil {
				log.Fatalf("error in getting content directory stats. content %s. error %v ", contentDir, err)
			}
			if !fileInfo.IsDir() {
				err = archiveContent(contentDir, archive)
				if err != nil {
					log.Fatalf("archiving content is failing. error %v ", err)
				}
			}
		}
	}

	if err := archive.Close(); err != nil {
		log.Fatalf("error while closing zipwriter. error %v ", err)
	}

	if err := zipfile.Close(); err != nil {
		log.Fatalf("error while closing zip file. file %s, error %v ", zipfile.Name(), err)
	}

	log.Printf("zip file created in directory location %s ", zipfile.Name())
	return zipfile.Name()
}

func createZipFile(target string)(*os.File, error)  {
	zipfile, err := ioutil.TempFile("", target +"-*."+ ZipExtension)
	if err != nil {
		log.Fatal(err)

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


