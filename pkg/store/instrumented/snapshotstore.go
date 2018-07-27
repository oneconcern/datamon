package instrumented

import (
	"context"

	"github.com/oneconcern/trumpet/pkg/store"
	opentracing "github.com/opentracing/opentracing-go"
)

// NewSnapshotStore creates an instrumented snapshot store.
func NewSnapshotStore(repoName string, tr opentracing.Tracer, w store.SnapshotStore) store.SnapshotStore {
	return &instrumentedSnapshots{
		tr:   tr,
		w:    w,
		repo: repoName,
	}
}

type instrumentedSnapshots struct {
	tr   opentracing.Tracer
	w    store.SnapshotStore
	repo string
}

func (i *instrumentedSnapshots) Initialize() error { return i.w.Initialize() }
func (i *instrumentedSnapshots) Close() error      { return i.w.Close() }

func (i *instrumentedSnapshots) Create(ctx context.Context, bundle *store.Bundle) (result *store.Snapshot, err error) {
	traced(ctx, i.tr, i.repo+" create snapshot", func() { result, err = i.w.Create(ctx, bundle) })
	return
}
func (i *instrumentedSnapshots) Get(ctx context.Context, id string) (result *store.Snapshot, err error) {
	traced(ctx, i.tr, i.repo+" get snapshot "+id, func() { result, err = i.w.Get(ctx, id) })
	return
}
func (i *instrumentedSnapshots) GetForBundle(ctx context.Context, id string) (result *store.Snapshot, err error) {
	traced(ctx, i.tr, i.repo+" get for bundle "+id, func() { result, err = i.w.GetForBundle(ctx, id) })
	return
}
