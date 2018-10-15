package instrumented

import (
	"context"

	"github.com/oneconcern/datamon/pkg/store"
	opentracing "github.com/opentracing/opentracing-go"
)

func NewRepos(tr opentracing.Tracer, w store.RepoStore) store.RepoStore {
	return &instrumentedRepos{
		tr: tr,
		w:  w,
	}
}

type instrumentedRepos struct {
	tr opentracing.Tracer
	w  store.RepoStore
}

func (i *instrumentedRepos) Initialize() error { return i.w.Initialize() }
func (i *instrumentedRepos) Close() error      { return i.w.Close() }

func (i *instrumentedRepos) List(ctx context.Context) (result []string, err error) {
	traced(ctx, i.tr, "list repos", func() { result, err = i.w.List(ctx) })
	return
}
func (i *instrumentedRepos) Get(ctx context.Context, key string) (repo *store.Repo, err error) {
	traced(ctx, i.tr, "get repo "+key, func() { repo, err = i.w.Get(ctx, key) })
	return
}
func (i *instrumentedRepos) Create(ctx context.Context, repo *store.Repo) (err error) {
	traced(ctx, i.tr, "create repo "+repo.Name, func() { err = i.w.Create(ctx, repo) })
	return
}
func (i *instrumentedRepos) Update(ctx context.Context, repo *store.Repo) (err error) {
	traced(ctx, i.tr, "update repo "+repo.Name, func() { err = i.w.Update(ctx, repo) })
	return
}
func (i *instrumentedRepos) Delete(ctx context.Context, name string) (err error) {
	traced(ctx, i.tr, "delete repo "+name, func() { err = i.w.Delete(ctx, name) })
	return
}

func traced(ctx context.Context, tr opentracing.Tracer, name string, action func()) {
	parent := opentracing.SpanFromContext(ctx)
	var opts []opentracing.StartSpanOption
	if parent != nil {
		opts = append(opts, opentracing.ChildOf(parent.Context()))
	}
	span := tr.StartSpan(name, opts...)
	defer span.Finish()
	action()
}
