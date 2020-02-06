package core

import (
	"strconv"
	"testing"

	"github.com/oneconcern/datamon/pkg/model"
	"github.com/stretchr/testify/assert"
)

const (
	patherRepo    = "myrepo"
	patherDiamond = " mydiamond"
	testID1       = "123"
	testID2       = "456"
	testID3       = "789"
	testID4       = "ABC"
)

var testIndexPather = func(i uint64) string { return "it-" + strconv.FormatUint(i, 10) }

func TestDownloadIndexIterator(t *testing.T) {
	it := newDownloadIndexIterator(5, testIndexPather)
	i := uint64(0)
	for pth := it.Next(); pth != ""; pth = it.Next() {
		assert.Equal(t, "it-"+strconv.FormatUint(i, 10), pth)
		i++
	}
	assert.Equal(t, uint64(5), i)
}

func TestUploadIndexIterator(t *testing.T) {
	it := newUploadIndexIterator(testIndexPather)
	i := uint64(0)
	for pth := it.Next(); i < 10; pth = it.Next() {
		assert.Equal(t, "it-"+strconv.FormatUint(i, 10), pth)
		i++
	}
	assert.Equal(t, uint64(10), i)
}

func TestDownloadBundleIterator(t *testing.T) {
	it := newDownloadBundleIterator(patherRepo, model.BundleDescriptor{
		ID:                     testID1,
		BundleEntriesFileCount: 10,
	})

	j := 0
	for id, pather := it.Next(); pather != nil; id, pather = it.Next() {
		i := uint64(0)
		assert.Equal(t, testID1, id)
		for pth := pather.Next(); pth != ""; pth = pather.Next() {
			// ex: bundles/myrepo/123/bundle-files-9.yaml
			assert.Equal(t, model.GetArchivePathToBundleFileList(patherRepo, testID1, i), pth)
			i++
		}
		assert.Equal(t, uint64(10), i)
		j++
	}
	assert.Equal(t, 1, j)
}

func TestUploadBundleIterator(t *testing.T) {
	it := newUploadBundleIterator(patherRepo, model.BundleDescriptor{
		ID: testID1,
	})

	j := 0
	for id, pather := it.Next(); pather != nil; id, pather = it.Next() {
		i := uint64(0)
		assert.Equal(t, testID1, id)
		for pth := pather.Next(); i < 20; pth = pather.Next() {
			// ex: bundles/myrepo/123/bundle-files-9.yaml
			assert.Equal(t, model.GetArchivePathToBundleFileList(patherRepo, testID1, i), pth)
			i++
		}
		assert.Equal(t, uint64(20), i)
		j++
	}
	assert.Equal(t, 1, j)
}

func TestDownloadAllSplitsIterator(t *testing.T) {
	it := newDownloadAllSplitsIterator(patherRepo, patherDiamond, []model.SplitDescriptor{
		{
			SplitID:               testID1,
			SplitEntriesFileCount: 5,
			GenerationID:          testID4,
		},
		{
			SplitID:               "456",
			SplitEntriesFileCount: 10,
			GenerationID:          testID4,
		},
		{
			SplitID:               "789",
			SplitEntriesFileCount: 3,
			GenerationID:          testID4,
		},
	})

	j := 0
	for id, pather := it.Next(); pather != nil; id, pather = it.Next() {
		i := uint64(0)
		for pth := pather.Next(); pth != ""; pth = pather.Next() {
			assert.Equal(t, model.GetArchivePathToSplitFileList(patherRepo, patherDiamond, id, testID4, i), pth)
			i++
		}
		switch id {
		case testID1:
			assert.Equal(t, 0, j)
			assert.Equal(t, uint64(5), i)
		case testID2:
			assert.Equal(t, 1, j)
			assert.Equal(t, uint64(10), i)
		case testID3:
			assert.Equal(t, 2, j)
			assert.Equal(t, uint64(3), i)
		default:
			t.Logf("unexpected split ID: %v", id)
			t.FailNow()
		}
		j++
	}
	assert.Equal(t, 3, j)
}

func TestDownloadSplitIterator(t *testing.T) {
	it := newDownloadSplitIterator(patherRepo, patherDiamond, model.SplitDescriptor{
		SplitID:               testID3,
		GenerationID:          testID4,
		SplitEntriesFileCount: 3,
	})

	j := 0
	for id, pather := it.Next(); pather != nil; id, pather = it.Next() {
		i := uint64(0)
		for pth := pather.Next(); pth != ""; pth = pather.Next() {
			assert.Equal(t, model.GetArchivePathToSplitFileList(patherRepo, patherDiamond, id, testID4, i), pth)
			i++
		}
		switch id {
		case testID3:
			assert.Equal(t, 0, j)
			assert.Equal(t, uint64(3), i)
		default:
			t.Logf("unexpected split ID: %v", id)
			t.FailNow()
		}
		j++
	}
	assert.Equal(t, 1, j)
}

func TestUploadSplitIterator(t *testing.T) {
	it := newUploadSplitIterator(patherRepo, patherDiamond, model.SplitDescriptor{
		SplitID:      testID3,
		GenerationID: testID4,
	})

	j := 0
	for id, pather := it.Next(); pather != nil; id, pather = it.Next() {
		i := uint64(0)
		assert.Equal(t, testID3, id)
		for pth := pather.Next(); i < 20; pth = pather.Next() {
			assert.Equal(t, model.GetArchivePathToSplitFileList(patherRepo, patherDiamond, id, testID4, i), pth)
			i++
		}
		assert.Equal(t, uint64(20), i)
		j++
	}
	assert.Equal(t, 1, j)
}
