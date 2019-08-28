package web

import (
	"context"
	"errors"
	"html/template"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/oneconcern/datamon/pkg/web/reverse"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gobuffalo/packd"
	"github.com/gobuffalo/packr/v2"
	"github.com/oneconcern/datamon/pkg/storage/gcs"
)

/* templates are divided into "drivers" and "helpers" as in examples at
 * https://golang.org/pkg/text/template/
 * this prevents conflicts with inheritance -- e.g. both home.html
 * and items.html (drivers) can use "base" and "lists" (helpers)
 * without conflict, provided every driver gets its own copy of the
 * helpers.
 *
 * compare go templates to Jinja2:  here in go, the more specific
 * templates *invoke* the less specific templates (with redefs)
 * rather than *extending* them as in Jinja.  since blocks can be
 * redefed precisely once per `Template` pointer, we need a separate
 * dependency tree for every page that appears in the app.
 */
type appTemplates map[string]*template.Template

func (tmpl appTemplates) Exec(
	s *Server, r *http.Request,
	name string, w io.Writer, data interface{}) error {
	t, has := tmpl[name]
	if !has {
		return errors.New("can't find template '" + name + "'")
	}
	return t.Lookup("driver").Execute(w,
		struct {
			Data interface{}
		}{
			Data: data,
		})
}

func loadTemplates() (appTemplates, error) {
	funcMap := template.FuncMap{
		"urlFor": reverse.Rev,
		"formatTimestamp": func(t time.Time) string {
			/* "Mon Jan _2 15:04:05 MST 2006" */
			return t.UTC().Format(time.UnixDate)
		},
	}
	helpersBox := packr.New("helperTmpls", "./tmpl/helpers")
	tmplH := template.New(helpersBox.Path)
	for _, name := range helpersBox.List() {
		if !strings.HasSuffix(name, ".html") {
			continue
		}
		tStr, err := helpersBox.FindString(name)
		if err != nil {
			return nil, err
		}
		tmplH, err = tmplH.Parse(tStr)
		if err != nil {
			return nil, err
		}
	}
	tmplH = tmplH.Funcs(funcMap)
	tmpl := make(map[string]*template.Template)
	driversBox := packr.New("driverTmpls", "./tmpl/drivers")
	for _, name := range driversBox.List() {
		if !strings.HasSuffix(name, ".html") {
			continue
		}
		tmplHClone, err := tmplH.Clone()
		if err != nil {
			return tmpl, err
		}
		tStr, err := driversBox.FindString(name)

		if err != nil {
			return nil, err
		}
		t, err := tmplHClone.New("driver").Parse(tStr)
		if err != nil {
			return tmpl, err
		}
		filename := name
		tmpl[filename] = t
	}
	return tmpl, nil
}

type ServerParams struct {
	MetadataBucket string
	Credential     string
}

type Server struct {
	tmpl   appTemplates
	params ServerParams
}

func (s *Server) metadataStore() storage.Store {
	var err error
	store, err := gcs.New(context.TODO(), s.params.MetadataBucket, s.params.Credential)
	if err != nil {
		panic(err)
	}
	return store
}

func NewServer(params ServerParams) (*Server, error) {
	tmpl, err := loadTemplates()
	if err != nil {
		return nil, err
	}
	return &Server{tmpl: tmpl, params: params}, nil
}

/* handlers */

func (s *Server) HandleHome() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repos, err := core.ListRepos(s.metadataStore())
		if err != nil {
			panic(err)
		}
		err = s.tmpl.Exec(s, r, "home.html", w, struct {
			Greeting string
			Repos    []model.RepoDescriptor
		}{
			Greeting: "Hello, world",
			Repos:    repos,
		})
		if err != nil {
			panic(err)
		}
	}
}

func (s *Server) HandleRepoListBundles() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoName := chi.URLParam(r, "repoName")
		bundles, err := core.ListBundles(repoName, s.metadataStore())
		if err != nil {
			panic(err)
		}
		err = s.tmpl.Exec(s, r, "repo__list_bundles.html", w, struct {
			Bundles  []model.BundleDescriptor
			RepoName string
		}{
			Bundles:  bundles,
			RepoName: repoName,
		})
		if err != nil {
			panic(err)
		}
	}
}

func (s *Server) HandleBundleListFiles() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoName := chi.URLParam(r, "repoName")
		bundleID := chi.URLParam(r, "bundleID")
		bundle := core.New(core.NewBDescriptor(),
			core.Repo(repoName),
			core.MetaStore(s.metadataStore()),
			core.BundleID(bundleID),
		)
		err := core.PopulateFiles(context.Background(), bundle)
		if err != nil {
			panic(err)
		}
		err = s.tmpl.Exec(s, r, "bundle__list_files.html", w, struct {
			RepoName      string
			BundleID      string
			BundleEntries []model.BundleEntry
		}{
			RepoName:      repoName,
			BundleID:      bundleID,
			BundleEntries: bundle.BundleEntries,
		})
		if err != nil {
			panic(err)
		}
	}
}

func InitRouter(srv *Server) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get(reverse.Add("home", "/"), srv.HandleHome())

	r.Get(reverse.Add("repo.list_bundles", "/repo/{repoName}/bundles", "{repoName}"),
		srv.HandleRepoListBundles())

	r.Get(reverse.Add("bundles.list_files", "/repo/{repoName}/bundles/{bundleID}", "{repoName}", "{bundleID}"),
		srv.HandleBundleListFiles())

	fileServer(r, "/assets", packr.New("static", "./public/assets"))

	return r
}

// sets up a http.FileServer handler to serve
// static files from a http.FileSystem.
func fileServer(r chi.Router, path string, box packd.HTTPBox) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit URL parameters.")
	}
	fs := http.StripPrefix(path, http.FileServer(box))
	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}
	path += "*"
	r.Get(path, http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			fs.ServeHTTP(w, r)
		}))
}
