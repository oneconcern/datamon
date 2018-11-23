package kubeless

import (
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
)

func TestCreateZipFile(t *testing.T)  {
	file, err := createZipFile("test-file")
	defer os.Remove(file.Name())
	require.NoError(t, err)

	if _, err := os.Stat(file.Name()); err != nil {
		if os.IsNotExist(err) {
			require.Failf(t, "expecting file " + file.Name() +  " to present ", "file is %s not present" , file.Name()  )
		}
	}

	require.True(t, true)
}

func TestZipFileWithoutGlob(t *testing.T)  {
	td, err := ioutil.TempDir("", "datamon-test-zip-delete-")
	defer os.RemoveAll(td)
	require.NoError(t, err)

	fileContent1 := []byte("temporary file1 content")


	tempFile1, err := ioutil.TempFile(td, "temp1-*.txt" )
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

	zipFileName := ZipFile(contentFiles, target)


	if _, err := os.Stat(zipFileName); err != nil {
		if os.IsNotExist(err) {
			require.Failf(t, "expecting file " + zipFileName +  " to present ", "file is %s not present" , zipFileName  )
		}
	}

	require.True(t, true)

	defer os.Remove(zipFileName)

}

func TestZipFileWithGlob(t *testing.T) {
	td, err := ioutil.TempDir("", "datamon-test-zip-delete-")
	defer os.RemoveAll(td)
	require.NoError(t, err)

	fileContent1 := []byte("temporary file1 content")


	tempFile1, err := ioutil.TempFile(td, "temp1-*.txt" )
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

	zipFileName := ZipFile(contentFiles, target)


	if _, err := os.Stat(zipFileName); err != nil {
		if os.IsNotExist(err) {
			require.Failf(t, "expecting file " + zipFileName +  " to present ", "file is %s not present" , zipFileName  )
		}
	}

	require.True(t, true)

	defer os.Remove(zipFileName)
}

