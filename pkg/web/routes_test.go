package web

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	context2 "github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/model"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/PuerkitoBio/goquery"
)

type CoreMocks struct {
	mock.Mock
}

const (
	testrepoone = "testrepoone"
	testrepotwo = "testrepotwo"
	bundleIDone = "asdf"
	bundleIDtwo = "zxcv"
	filenameone = "filenameone"
	filenametwo = "filenametwo"
)

func (m *CoreMocks) listRepos(stores context2.Stores) []model.RepoDescriptor {
	t := time.Now()
	rv := []model.RepoDescriptor{
		{
			Name:        testrepoone,
			Description: "first test repo",
			Timestamp:   t,
			Contributor: model.Contributor{Name: "testerone", Email: "test1@oneconcern.com"},
		},
		{
			Name:        testrepotwo,
			Description: "second test repo",
			Timestamp:   t,
			Contributor: model.Contributor{Name: "testertwo", Email: "test2@oneconcern.com"},
		},
	}
	m.On("listRepos", stores).Return(rv)
	m.MethodCalled("listRepos", stores)
	return rv
}

func (m *CoreMocks) listBundles(repoName string, stores context2.Stores) []model.BundleDescriptor {
	t := time.Now()
	rv := []model.BundleDescriptor{
		{
			ID:        bundleIDone,
			Message:   "the first bundle",
			Timestamp: t,
		},
		{
			ID:        bundleIDtwo,
			Message:   "the second bundle",
			Timestamp: t,
		},
	}
	m.On("listBundles", repoName, stores).Return(rv)
	m.MethodCalled("listBundles", repoName, stores)
	return rv
}

func (m *CoreMocks) listBundleFiles(repoName string, bundleID string, stores context2.Stores) []model.BundleEntry {
	rv := []model.BundleEntry{
		{
			Hash:         "hashthefirst",
			NameWithPath: filenameone,
			Size:         256,
		},
		{
			Hash:         "hashthesecond",
			NameWithPath: filenametwo,
			Size:         128,
		},
	}
	m.On("listBundleFiles", repoName, bundleID, stores).Return(rv)
	m.MethodCalled("listBundleFiles", repoName, bundleID, stores)
	return rv
}

func newCoreMocks() *CoreMocks {
	mocks := new(CoreMocks)
	return mocks
}

var coreMocks *CoreMocks

func listReposMock(stores context2.Stores) []model.RepoDescriptor {
	return coreMocks.listRepos(stores)
}

func listBundlesMock(repoName string, stores context2.Stores) []model.BundleDescriptor {
	return coreMocks.listBundles(repoName, stores)
}

func listBundleFilesMock(repoName string, bundleID string, stores context2.Stores) []model.BundleEntry {
	return coreMocks.listBundleFiles(repoName, bundleID, stores)
}

func setupTests(t *testing.T) http.Handler {
	coreMocks = newCoreMocks()
	listRepos = listReposMock
	listBundles = listBundlesMock
	listBundleFiles = listBundleFilesMock
	srv, err := NewServer(ServerParams{})
	require.NoError(t, err, "create web server instance")
	return InitRouter(srv)
}

type repoListPage struct {
	doc *goquery.Document
}

func (page *repoListPage) repoNames(t *testing.T) map[string]bool {
	doc := page.doc
	repoNamesActual := make(map[string]bool)
	doc.Find("table tbody tr").Each(func(i int, s *goquery.Selection) {
		repoNameAnchor := s.Find("td a").First()
		require.NotNil(t, repoNameAnchor, "found repo name element")
		repoNamesActual[strings.TrimSpace(repoNameAnchor.Text())] = true
	})
	return repoNamesActual
}

type bundleListPage struct {
	doc *goquery.Document
}

func (page *bundleListPage) bundleIDs(t *testing.T) map[string]bool {
	doc := page.doc
	bundleIDsActual := make(map[string]bool)
	doc.Find("table tbody tr").Each(func(i int, s *goquery.Selection) {
		bundleIDAnchor := s.Find("td a").First()
		require.NotNil(t, bundleIDAnchor, "found bundle ID element")
		bundleIDsActual[strings.TrimSpace(bundleIDAnchor.Text())] = true
	})
	return bundleIDsActual
}

type fileListPage struct {
	doc *goquery.Document
}

func (page *fileListPage) filenames(t *testing.T) map[string]bool {
	doc := page.doc
	filenamesActual := make(map[string]bool)
	doc.Find("table tbody tr").Each(func(i int, s *goquery.Selection) {
		filenameTableDescriptor := s.Find("td").First()
		require.NotNil(t, filenameTableDescriptor, "found bundle ID element")
		filenamesActual[strings.TrimSpace(filenameTableDescriptor.Text())] = true
	})
	return filenamesActual
}

func getPageDocument(t *testing.T, routes http.Handler, relURLPath string) *goquery.Document {
	req, err := http.NewRequest("GET", relURLPath, nil)
	require.NoError(t, err, "create mock request")
	rr := httptest.NewRecorder()
	routes.ServeHTTP(rr, req)
	res := rr.Result()
	defer res.Body.Close()
	require.Equal(t, 200, res.StatusCode, "http status ok")
	doc, err := goquery.NewDocumentFromReader(res.Body)
	require.NoError(t, err, "parse body html")
	return doc
}

func TestListRepos(t *testing.T) {
	routes := setupTests(t)
	doc := getPageDocument(t, routes, "/")
	t.Logf("html doc %v", doc)
	page := &repoListPage{doc: doc}
	repoNamesExpected := map[string]bool{
		testrepoone: true,
		testrepotwo: true,
	}
	require.Equal(t, repoNamesExpected, page.repoNames(t),
		"found expected repo names")
}

func TestListBundles(t *testing.T) {
	routes := setupTests(t)
	doc := getPageDocument(t, routes, fmt.Sprintf("/repo/%s/bundles", testrepoone))
	page := bundleListPage{doc: doc}
	bundleIDsExpected := map[string]bool{
		bundleIDone: true,
		bundleIDtwo: true,
	}
	require.Equal(t, bundleIDsExpected, page.bundleIDs(t),
		"found expected bundle ids")
}

func TestListBundleFiles(t *testing.T) {
	routes := setupTests(t)
	doc := getPageDocument(t, routes, fmt.Sprintf("/repo/%s/bundles/%s", testrepoone, bundleIDone))
	page := fileListPage{doc: doc}
	filenamesExpected := map[string]bool{
		filenameone: true,
		filenametwo: true,
	}
	require.Equal(t, filenamesExpected, page.filenames(t),
		"found expected filenames")
}
