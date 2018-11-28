package kubeless

import (
	"archive/zip"
	"fmt"
	"github.com/bmatcuk/doublestar"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateZipFile(t *testing.T) {
	file, err := createZipFile("test-file")
	defer os.Remove(file.Name())
	require.NoError(t, err)

	if _, err := os.Stat(file.Name()); err != nil {
		if os.IsNotExist(err) {
			require.Failf(t, "expecting file "+file.Name()+" to present ", "file is %s not present", file.Name())
		}
	}

}

func TestZipFileWithoutGlob(t *testing.T) {
	td, err := ioutil.TempDir("", "datamon-test-zip-delete-")
	defer os.RemoveAll(td)
	require.NoError(t, err)

	fileContent1 := []byte("temporary file1 content")

	tempFile1, err := ioutil.TempFile(td, "temp1-*.txt")
	require.NoError(t, err)

	_, err = tempFile1.Write(fileContent1)
	require.NoError(t, err)

	err = tempFile1.Close()
	require.NoError(t, err)

	fileContent2 := []byte("temporary file2 content")
	tempFile2, err := ioutil.TempFile(td, "temp2-*.txt")
	require.NoError(t, err)

	_, err = tempFile2.Write(fileContent2)
	require.NoError(t, err)

	err = tempFile2.Close()
	require.NoError(t, err)

	contentFiles := []string{tempFile1.Name(), tempFile2.Name()}

	target := "target-zip"

	zipFileName, err := ZipFile(contentFiles, target)
	require.NoError(t, err)
	defer os.Remove(zipFileName)

	if _, err := os.Stat(zipFileName); err != nil {
		if os.IsNotExist(err) {
			require.Failf(t, "expecting file "+zipFileName+" to present ", "file is %s not present", zipFileName)
		}
	}

}

func TestZipFileWithGlob(t *testing.T) {
	td, err := ioutil.TempDir("", "datamon-test-zip-delete-")
	defer os.RemoveAll(td)
	require.NoError(t, err)

	fileContent1 := []byte("temporary file1 content")

	tempFile1, err := ioutil.TempFile(td, "temp1-*.txt")
	require.NoError(t, err)

	_, err = tempFile1.Write(fileContent1)
	require.NoError(t, err)

	err = tempFile1.Close()
	require.NoError(t, err)

	fileContent2 := []byte("temporary file2 content")
	tempFile2, err := ioutil.TempFile(td, "temp2-*.txt")
	require.NoError(t, err)

	_, err = tempFile2.Write(fileContent2)
	require.NoError(t, err)

	err = tempFile2.Close()
	require.NoError(t, err)

	contentFiles := []string{td + "/*"}

	target := "target-glob-zip"

	zipFileName, err := ZipFile(contentFiles, target)
	require.NoError(t, err)
	defer os.Remove(zipFileName)

	if _, err := os.Stat(zipFileName); err != nil {
		if os.IsNotExist(err) {
			require.Failf(t, "expecting file "+zipFileName+" to present ", "file is %s not present", zipFileName)
		}
	}

}

func TestZipFileGlobPattern(t *testing.T) {
	sourceDir, err := ioutil.TempDir("", "datamon-test-zip-delete-")
	require.NoError(t, err)

	defer os.RemoveAll(sourceDir)

	unzipFiles, globPatternMatches, err := zipfileGlobPatternData(sourceDir)
	if err != nil {
		require.NoError(t, err)
	}

	var trimmedUnzipPath []string
	for _, file := range unzipFiles {
		trimmedUnzipPath = append(trimmedUnzipPath, strings.TrimPrefix(file, sourceDir))
	}

	require.Equalf(t, len(unzipFiles), len(globPatternMatches), "glob pattern length & unzip files length is different")
	require.Equalf(t, globPatternMatches, trimmedUnzipPath, "glob pattern files and unzip files are different")
}

func TestZipFileGlobPatternContent(t *testing.T) {
	sourceDir, err := ioutil.TempDir("", "datamon-test-zip-delete-")
	require.NoError(t, err)

	defer os.RemoveAll(sourceDir)

	unzipFiles, globPatternMatches, err := zipfileGlobPatternData(sourceDir)
	if err != nil {
		require.NoError(t, err)
	}

	require.Equalf(t, len(unzipFiles), len(globPatternMatches), "glob pattern & unzip files are different")

	for index, globMatches := range globPatternMatches {
		globMatchFileContent, err := ioutil.ReadFile(globMatches)
		require.NoError(t, err)

		unzipFileContent, err := ioutil.ReadFile(unzipFiles[index])
		require.NoError(t, err)

		require.Equalf(t, globMatchFileContent, unzipFileContent, "glob file content %s and unzip file content %s", string(globMatchFileContent), string(unzipFileContent))
	}

}
func TestZipFileContent(t *testing.T) {
	td, err := ioutil.TempDir("", "datamon-test-zip-delete-")
	defer os.RemoveAll(td)
	require.NoError(t, err)

	fileContent1 := []byte("temporary file1 content")

	tempFile1, err := ioutil.TempFile(td, "temp1-*.txt")
	require.NoError(t, err)

	_, err = tempFile1.Write(fileContent1)
	require.NoError(t, err)

	err = tempFile1.Close()
	require.NoError(t, err)

	contentFiles := []string{tempFile1.Name()}

	target := "target-zip"
	log.Printf(tempFile1.Name())

	zipFileName, err := ZipFile(contentFiles, target)
	require.NoError(t, err)

	defer os.Remove(zipFileName)

	unzipFiles, err := unzip(zipFileName, td)
	require.NoError(t, err)

	zipFileContent, err := ioutil.ReadFile(unzipFiles[0])
	require.NoError(t, err)

	defer os.Remove(unzipFiles[0])
	require.Equalf(t, fileContent1, zipFileContent, "content of file and zip is not equal")
}

func unzip(src string, dest string) ([]string, error) {

	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}
		defer rc.Close()

		fpath := filepath.Join(dest, f.Name)

		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {

			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)

		} else {

			// Make File
			if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
				return filenames, err
			}

			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return filenames, err
			}

			_, err = io.Copy(outFile, rc)

			// Close the file without defer to close before next iteration of loop
			outFile.Close()

			if err != nil {
				return filenames, err
			}

		}
	}
	return filenames, nil
}

func zipfileGlobPatternData(sourceDir string) ([]string, []string, error) {

	test1, err := ioutil.TempDir(sourceDir, "test1-")
	if err != nil {
		return nil, nil, fmt.Errorf("temp child directory test1 %v ", err)
	}

	test2, err := ioutil.TempDir(sourceDir, "test2-")
	if err != nil {
		return nil, nil, fmt.Errorf("temp child directory test2 %v ", err)
	}

	test3, err := ioutil.TempDir(sourceDir, "test3-")
	if err != nil {
		return nil, nil, fmt.Errorf("temp child directory test3 %v ", err)
	}

	fileContent1 := []byte("temporary file1 content")

	testFile1, err := ioutil.TempFile(test1, "test1-*.txt")
	if err != nil {
		return nil, nil, fmt.Errorf("test1 dir file %v ", err)
	}

	_, err = testFile1.Write(fileContent1)
	if err != nil {
		return nil, nil, fmt.Errorf("test1 write content %v ", err)
	}

	err = testFile1.Close()
	if err != nil {
		return nil, nil, fmt.Errorf("test1 child file close %v ", err)
	}

	fileContent2a := []byte("temporary file 2a content")

	testFile2a, err := ioutil.TempFile(test2, "test2a-*.txt")
	if err != nil {
		return nil, nil, fmt.Errorf("test2 dir file %v ", err)
	}

	_, err = testFile2a.Write(fileContent2a)
	if err != nil {
		return nil, nil, fmt.Errorf("test2a write content %v ", err)
	}

	err = testFile2a.Close()
	if err != nil {
		return nil, nil, fmt.Errorf("test2a child file close %v ", err)
	}

	fileContent2b := []byte("temporary file 2b content")
	testFile2b, err := ioutil.TempFile(test2, "test2b-*.json")
	if err != nil {
		return nil, nil, fmt.Errorf("test2 dir file %v ", err)
	}

	_, err = testFile2b.Write(fileContent2b)
	if err != nil {
		return nil, nil, fmt.Errorf("test2b write content %v ", err)
	}

	err = testFile2b.Close()
	if err != nil {
		return nil, nil, fmt.Errorf("test2b child file close %v ", err)
	}

	fileContent3 := []byte("temporary file3 content")

	testFile3, err := ioutil.TempFile(test3, "test3-*.yml")
	if err != nil {
		return nil, nil, fmt.Errorf("test3 dir file %v ", err)
	}

	_, err = testFile3.Write(fileContent3)
	if err != nil {
		return nil, nil, fmt.Errorf("test3 write content %v ", err)
	}

	err = testFile3.Close()
	if err != nil {
		return nil, nil, fmt.Errorf("test3 child file close %v ", err)
	}

	contentFiles := []string{
		test1 + "/*",
		test2 + "/*.json",
		test3 + "/*.yml",
	}

	target := "target-glob-zip"

	zipFileName, err := ZipFile(contentFiles, target)
	if err != nil {
		return nil, nil, fmt.Errorf("zip file %v ", err)
	}
	defer os.Remove(zipFileName)

	unzipFiles, err := unzip(zipFileName, sourceDir)
	if err != nil {
		return nil, nil, fmt.Errorf("unzip file %v ", err)
	}

	var globPatternMatches []string
	for _, source := range contentFiles {
		globmatches, err := doublestar.Glob(source)
		if err != nil {
			return nil, nil, fmt.Errorf("glob pattern %v ", err)
		}
		for _, matches := range globmatches {
			globPatternMatches = append(globPatternMatches, matches)
		}
	}

	return unzipFiles, globPatternMatches, err
}
