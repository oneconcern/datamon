// Copyright © 2018 One Concern

package storage

import (
	"context"
	"io"
	"strings"

	"github.com/oneconcern/pipelines/pkg/log"
	opentracing "github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
)

func Instrument(tr opentracing.Tracer, logs log.Factory, store Store) Store {
	return &instrumentedStore{
		tr:    tr,
		store: store,
		logs:  logs.With(zap.String("model", store.String())),
	}
}

type instrumentedStore struct {
	store Store
	tr    opentracing.Tracer
	logs  log.Factory
}

func (i *instrumentedStore) KeysPrefix(ctx context.Context, token, prefix, delimiter string, count int) ([]string, string, error) {
	span := i.spanFromContext(ctx, i.opName("KeysPrefix"))
	defer span.Finish()
	i.logs.Bg().Info("storage keys with Prefix")

	return i.store.KeysPrefix(ctx, token, prefix, delimiter, count)
}

func (i *instrumentedStore) opName(name string) string {
	return strings.Join([]string{"storage", i.String(), name}, ".")
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
	i.logs.Bg().Info("storage has", zap.String("key", key))

	return i.store.Has(ctx, key)
}

func (i *instrumentedStore) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	span := i.spanFromContext(ctx, i.opName("Get"))
	defer span.Finish()

	i.logs.Bg().Info("storage get", zap.String("key", key))
	return i.store.Get(ctx, key)
}

func (i *instrumentedStore) Put(ctx context.Context, key string, rdr io.Reader) error {
	span := i.spanFromContext(ctx, i.opName("Put"))
	defer span.Finish()

	i.logs.Bg().Info("storage put", zap.String("key", key))
	return i.store.Put(ctx, key, rdr)
}

func (i *instrumentedStore) Delete(ctx context.Context, key string) error {
	span := i.spanFromContext(ctx, i.opName("Delete"))
	defer span.Finish()

	i.logs.Bg().Info("storage delete", zap.String("key", key))
	return i.store.Delete(ctx, key)
}

func (i *instrumentedStore) Keys(ctx context.Context) ([]string, error) {
	span := i.spanFromContext(ctx, i.opName("Keys"))
	defer span.Finish()
	i.logs.Bg().Info("storage keys")

	return i.store.Keys(ctx)
}

func (i *instrumentedStore) Clear(ctx context.Context) error {
	span := i.spanFromContext(ctx, i.opName("Clear"))
	defer span.Finish()
	i.logs.Bg().Info("storage clear")

	return i.store.Clear(ctx)
}

func (i *instrumentedStore) String() string {
	return i.store.String()
}

func (i *instrumentedStore) GetAt(ctx context.Context, objectName string) (io.ReaderAt, error) {
	span := i.spanFromContext(ctx, i.opName("GetAt"))
	defer span.Finish()
	i.logs.Bg().Info("get a offset reader")
	return i.store.GetAt(ctx, objectName)
}
