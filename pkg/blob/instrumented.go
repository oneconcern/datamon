package blob

import (
	"context"
	"io"
	"strings"

	opentracing "github.com/opentracing/opentracing-go"
)

func Instrument(tr opentracing.Tracer, store Store) Store {
	return &instrumentedStore{
		tr:    tr,
		store: store,
	}
}

type instrumentedStore struct {
	store Store
	tr    opentracing.Tracer
}

func (i *instrumentedStore) opName(name string) string {
	return strings.Join([]string{"blob", i.String(), name}, ".")
}

func (i *instrumentedStore) spanFromContext(ctx context.Context, name string) opentracing.Span {
	parent := opentracing.SpanFromContext(ctx)
	var span opentracing.Span
	if parent != nil {
		span = i.tr.StartSpan(name, opentracing.ChildOf(parent.Context()))
	} else {
		span = i.tr.StartSpan(name)
	}
	return span
}

func (i *instrumentedStore) Has(ctx context.Context, key string) (bool, error) {
	span := i.spanFromContext(ctx, i.opName("Has"))
	defer span.Finish()

	return i.store.Has(ctx, key)
}

func (i *instrumentedStore) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	span := i.spanFromContext(ctx, i.opName("Get"))
	defer span.Finish()

	return i.store.Get(ctx, key)
}

func (i *instrumentedStore) Put(ctx context.Context, key string, rdr io.Reader) error {
	span := i.spanFromContext(ctx, i.opName("Put"))
	defer span.Finish()

	return i.store.Put(ctx, key, rdr)
}

func (i *instrumentedStore) Delete(ctx context.Context, key string) error {
	span := i.spanFromContext(ctx, i.opName("Delete"))
	defer span.Finish()

	return i.store.Delete(ctx, key)
}

func (i *instrumentedStore) Keys(ctx context.Context) ([]string, error) {
	span := i.spanFromContext(ctx, i.opName("Keys"))
	defer span.Finish()

	return i.store.Keys(ctx)
}

func (i *instrumentedStore) Clear(ctx context.Context) error {
	span := i.spanFromContext(ctx, i.opName("Clear"))
	defer span.Finish()

	return i.store.Clear(ctx)
}

func (i *instrumentedStore) String() string {
	return i.store.String()
}
