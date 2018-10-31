package kubeless

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/oneconcern/datamon/pkg/blob/sthree"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const ZIP_EXTENSION  = ".zip"
// Zip file method take list of directories or files from content attribute and
// target name of zip file and zip input files.
func ZipFile(content []string, target string) error {
	zipfile, err := os.Create(target + ZIP_EXTENSION)
	if err != nil {
		log.Printf("error creating zip file: %s, Error: %v ",target + ZIP_EXTENSION, err)
		return err
	}
	defer zipfile.Close()

	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	for _, source := range content {

		source = globPatternCheck(source)
		err = fileWalk(source, archive)
		if err != nil{
			return err
		}

	}

	return nil
}

func UploadFileToS3(fileName string) (string, error) {
	buffer, path := ReadFile(fileName + ZIP_EXTENSION)

	fileBytes := bytes.NewReader(buffer)


	awsConfig := aws.NewConfig().
		WithRegion("us-west-2").
		WithCredentialsChainVerboseErrors(true)

	bucket := aws.String("oneconcern-datamon-dev")

	bs := sthree.New(sthree.Bucket(*bucket), sthree.AWSConfig(awsConfig))
	err := bs.Put(context.Background(), path, fileBytes )

	bucketUrl := fmt.Sprintf("https://s3-%s.amazonaws.com/%s/%s", "us-west-2", *bucket, fileName + ZIP_EXTENSION)


	return bucketUrl, err

}

func ReadFile(fileDir string) ([]byte, string) {
	file, err := os.Open(fileDir)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()


	fileInfo, err := file.Stat()
	if err != nil {
		log.Printf("error in getting file stats for file %s. Error: %v ", fileDir, err)
	}

	// Get file size and read the file content into a buffer
	buffer := make([]byte, fileInfo.Size())

	file.Read(buffer)

	return buffer, file.Name()
}

//fileWalk walks the root directory, calling function on child directory and files and zip the files
func fileWalk(source string, archive *zip.Writer) error {
	fmt.Println(source)
	info, err := os.Stat(source)
	if err != nil {
		log.Printf("error in getting file Info. File name: %v, Error: %v ", source, err)
		return err
	}

	var baseDir string
	if info.IsDir() {
		baseDir = filepath.Base(source)
	}

	err = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			header, err := zip.FileInfoHeader(info)
			if err != nil {
				return err
			}

			if baseDir != "" {
				header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, source))
			}

			if info.IsDir() {
				header.Name += "/"
			} else {
				header.Method = zip.Deflate
			}

			writer, err := archive.CreateHeader(header)
			if err != nil {
				return err
			}

			if info.IsDir() {
				log.Printf("skipping source directory %s ", info.Name())
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(writer, file)
			return err
	})

	if err != nil {
		log.Printf("error in iterating source directory %s, Error: %v", source, err)

		return err
	}

	return nil

}

func globPatternCheck(source string) string  {
	splitSourceFile := strings.Split(source, "/*")

	if len(splitSourceFile) > 0 {
		source = splitSourceFile[0]
	}

	return source
}


