package instrumented

import (
	"context"

	"github.com/oneconcern/trumpet/pkg/store"
	opentracing "github.com/opentracing/opentracing-go"
)

func NewObjectMeta(repo string, tr opentracing.Tracer, w store.StageMeta) store.StageMeta {
	return &instumentedStage{
		tr:   tr,
		w:    w,
		repo: repo,
	}
}

type instumentedStage struct {
	tr   opentracing.Tracer
	w    store.StageMeta
	repo string
}

func (i *instumentedStage) Initialize() error { return i.w.Initialize() }
func (i *instumentedStage) Close() error      { return i.w.Close() }

func (i *instumentedStage) Add(ctx context.Context, entry store.Entry) (err error) {
	traced(ctx, i.tr, i.repo+" add to stage "+entry.Path, func() { err = i.w.Add(ctx, entry) })
	return
}
func (i *instumentedStage) Remove(ctx context.Context, id string) (err error) {
	traced(ctx, i.tr, i.repo+" remove from stage "+id, func() { err = i.w.Remove(ctx, id) })
	return
}
func (i *instumentedStage) List(ctx context.Context) (result store.ChangeSet, err error) {
	traced(ctx, i.tr, i.repo+" list stage", func() { result, err = i.w.List(ctx) })
	return
}
func (i *instumentedStage) MarkDelete(ctx context.Context, entry *store.Entry) (err error) {
	traced(ctx, i.tr, i.repo+" mark deleted on stage "+entry.Path, func() { err = i.w.MarkDelete(ctx, entry) })
	return
}
func (i *instumentedStage) Get(ctx context.Context, id string) (entry store.Entry, err error) {
	traced(ctx, i.tr, i.repo+" get object from stage "+id, func() { entry, err = i.w.Get(ctx, id) })
	return
}
func (i *instumentedStage) HashFor(ctx context.Context, path string) (result string, err error) {
	traced(ctx, i.tr, i.repo+" stage hash for "+path, func() { result, err = i.w.HashFor(ctx, path) })
	return
}
func (i *instumentedStage) Clear(ctx context.Context) (err error) {
	traced(ctx, i.tr, i.repo+" clear stage", func() { err = i.w.Clear(ctx) })
	return
}
