package instrumented

import (
	"context"

	"github.com/oneconcern/trumpet/pkg/store"
	opentracing "github.com/opentracing/opentracing-go"
)

// NewBundleStore creates a instrumented bundle store.
func NewBundleStore(repoName string, tr opentracing.Tracer, w store.BundleStore, options ...opentracing.StartSpanOption) store.BundleStore {
	return &instrumentedBundles{
		tr:       tr,
		w:        w,
		repoName: repoName,
		options:  options,
	}
}

type instrumentedBundles struct {
	tr       opentracing.Tracer
	w        store.BundleStore
	repoName string
	options  []opentracing.StartSpanOption
}

func (i *instrumentedBundles) Initialize() error { return i.w.Initialize() }
func (i *instrumentedBundles) Close() error      { return i.w.Close() }

func (i *instrumentedBundles) ListTopLevel(ctx context.Context) (result []store.Bundle, err error) {
	traced(ctx, i.tr, i.repoName+" list top level bundles", func() { result, err = i.w.ListTopLevel(ctx) })
	return
}
func (i *instrumentedBundles) ListTopLevelIDs(ctx context.Context) (result []string, err error) {
	traced(ctx, i.tr, i.repoName+" list top level bundle ids", func() { result, err = i.w.ListTopLevelIDs(ctx) })
	return
}

func (i *instrumentedBundles) ListBranches(ctx context.Context) (result []string, err error) {
	traced(ctx, i.tr, i.repoName+" list branches", func() { result, err = i.w.ListBranches(ctx) })
	return
}
func (i *instrumentedBundles) HashForBranch(ctx context.Context, branch string) (result string, err error) {
	traced(ctx, i.tr, i.repoName+" hash for branch "+branch, func() { result, err = i.w.HashForBranch(ctx, branch) })
	return
}
func (i *instrumentedBundles) CreateBranch(ctx context.Context, name string, message string) (err error) {
	traced(ctx, i.tr, i.repoName+" create branch"+name, func() { err = i.w.CreateBranch(ctx, name, message) })
	return
}
func (i *instrumentedBundles) DeleteBranch(ctx context.Context, branch string) (err error) {
	traced(ctx, i.tr, i.repoName+" delete branch "+branch, func() { err = i.w.DeleteBranch(ctx, branch) })
	return
}

func (i *instrumentedBundles) ListTags(ctx context.Context) (result []string, err error) {
	traced(ctx, i.tr, i.repoName+" list tags", func() { result, err = i.w.ListTags(ctx) })
	return
}

func (i *instrumentedBundles) HashForTag(ctx context.Context, tag string) (result string, err error) {
	traced(ctx, i.tr, i.repoName+" hash for tag "+tag, func() { result, err = i.w.HashForTag(ctx, tag) })
	return
}

func (i *instrumentedBundles) CreateTag(ctx context.Context, name string, message string) (err error) {
	traced(ctx, i.tr, i.repoName+" create tag "+name, func() { err = i.w.CreateTag(ctx, name, message) })
	return
}

func (i *instrumentedBundles) DeleteTag(ctx context.Context, tag string) (err error) {
	traced(ctx, i.tr, i.repoName+" delete tag "+tag, func() { err = i.w.DeleteTag(ctx, tag) })
	return
}

func (i *instrumentedBundles) Create(ctx context.Context, message, branch, snapshot string, parents []string, changes store.ChangeSet) (result string, isEmpty bool, err error) {
	traced(
		ctx,
		i.tr,
		i.repoName+" create bundle",
		func() { result, isEmpty, err = i.w.Create(ctx, message, branch, snapshot, parents, changes) },
	)
	return
}

func (i *instrumentedBundles) Get(ctx context.Context, id string) (result *store.Bundle, err error) {
	traced(ctx, i.tr, i.repoName+" get bundle "+id, func() { result, err = i.w.Get(ctx, id) })
	return
}
func (i *instrumentedBundles) GetObject(ctx context.Context, id string) (result store.Entry, err error) {
	traced(ctx, i.tr, i.repoName+" get object "+id, func() { result, err = i.w.GetObject(ctx, id) })
	return
}
func (i *instrumentedBundles) GetObjectForPath(ctx context.Context, path string) (result store.Entry, err error) {
	traced(ctx, i.tr, i.repoName+" get object for path "+path, func() { result, err = i.w.GetObjectForPath(ctx, path) })
	return
}
func (i *instrumentedBundles) HashForPath(ctx context.Context, path string) (result string, err error) {
	traced(ctx, i.tr, i.repoName+" hash for path "+path, func() { result, err = i.w.HashForPath(ctx, path) })
	return
}
