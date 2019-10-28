package wal

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	gcsStorage "cloud.google.com/go/storage"
	"github.com/oneconcern/datamon/internal"
	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/oneconcern/datamon/pkg/storage/gcs"
	"github.com/oneconcern/datamon/pkg/storage/localfs"
	"github.com/segmentio/ksuid"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/api/option"
)

const (
	longPath   = "this/is/a/long/path/to/an/object/the/object/is/under/this/path/list/with/prefix/please/"
	mutable    = "mutable"
	wal        = "wal"
	payload    = "payload"
	attrError  = "getAttrErrorTest"
	putError   = "putErrorTest"
	touchError = "touchErrorTest"
)

func constStringWithIndex(i int) string {
	return longPath + fmt.Sprint(i)
}
func randSleep() {
	r := rand.Intn(200)
	time.Sleep(time.Duration(r)*time.Millisecond + 1) // min 1 ms
}

func setup(t testing.TB, numOfObjects int) (storage.Store, func()) {

	ctx := context.Background()

	bucket := "deleteme-wal-test-" + internal.RandStringBytesMaskImprSrc(15)
	log.Printf("Created bucket %s ", bucket)

	client, err := gcsStorage.NewClient(context.TODO(), option.WithScopes(gcsStorage.ScopeFullControl))
	require.NoError(t, err)
	err = client.Bucket(bucket).Create(ctx, "onec-co", nil)
	require.NoError(t, err, "Failed to create bucket:"+bucket)

	gcs, err := gcs.New(context.TODO(), bucket, "") // Use GOOGLE_APPLICATION_CREDENTIALS env variable
	require.NoError(t, err, "failed to create gcs client")
	wg := sync.WaitGroup{}
	create := func(i int, wg *sync.WaitGroup) {
		err = gcs.Put(ctx, constStringWithIndex(i), bytes.NewBufferString(constStringWithIndex(i)), storage.NoOverWrite)
		require.NoError(t, err, "Index at: "+fmt.Sprint(i))
		wg.Done()
	}
	for i := 0; i < numOfObjects; i++ {
		// Use path as payload
		wg.Add(1)
		go create(i, &wg)
	}
	wg.Wait()

	cleanup := func() {
		delete := func(key string, wg *sync.WaitGroup) {
			err = gcs.Delete(ctx, key)
			require.NoError(t, err, "failed to delete:"+key)
			wg.Done()
		}

		wg := sync.WaitGroup{}
		for i := 0; i < numOfObjects; i++ {
			wg.Add(1)
			delete(constStringWithIndex(i), &wg)
		}
		wg.Wait()

		// Delete any keys created outside of setup at the end of test.
		var keys []string
		keys, err = gcs.Keys(ctx)
		for _, k := range keys {
			wg.Add(1)
			delete(k, &wg)
		}
		wg.Wait()

		log.Printf("Delete bucket %s ", bucket)
		err = client.Bucket(bucket).Delete(ctx)
		require.NoError(t, err, "Failed to delete bucket:"+bucket)
	}

	return gcs, cleanup
}

func TestNewWAL1(t *testing.T) {
	t.Parallel()
	type args struct {
		mutableStore storage.Store
		walStore     storage.Store
		logger       *zap.Logger
		options      []Options
	}
	l, err := zap.NewDevelopment()
	s1 := localfs.New(afero.NewOsFs())
	s2 := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), "base"))
	require.NoError(t, err)
	tests := []struct {
		name string
		args args
		want *WAL
	}{
		{
			name: "all options",
			args: args{
				mutableStore: s1,
				walStore:     s2,
				logger:       l,
				options:      []Options{MaxConcurrency(11), TokenGeneratorPath("path")},
			},
			want: &WAL{
				mutableStore:       s1,
				walStore:           s2,
				l:                  l,
				maxConcurrency:     11,
				connectionControl:  make(chan struct{}, 11),
				tokenGeneratorPath: "path",
			},
		},
		{
			name: "path",
			args: args{
				mutableStore: s1,
				walStore:     s2,
				logger:       l,
				options:      []Options{TokenGeneratorPath("path")},
			},
			want: &WAL{
				mutableStore:       s1,
				walStore:           s2,
				l:                  l,
				maxConcurrency:     maxConcurrency,
				tokenGeneratorPath: "path",
			},
		},
		{
			name: "concurrency ",
			args: args{
				mutableStore: s1,
				walStore:     s2,
				logger:       l,
				options:      []Options{MaxConcurrency(11)},
			},
			want: &WAL{
				mutableStore:       s1,
				walStore:           s2,
				l:                  l,
				maxConcurrency:     11,
				tokenGeneratorPath: tokenGeneratorPath,
			},
		},
		{
			name: "default",
			args: args{
				mutableStore: s1,
				walStore:     s2,
				logger:       l,
				options:      []Options{},
			},
			want: &WAL{
				mutableStore:       s1,
				walStore:           s2,
				l:                  l,
				maxConcurrency:     maxConcurrency,
				tokenGeneratorPath: tokenGeneratorPath,
			},
		},
	}
	for _, test := range tests {
		tt := test
		t.Run(tt.name, func(t *testing.T) {
			if got := NewWAL(tt.args.mutableStore, tt.args.walStore, tt.args.logger, tt.args.options...); !(reflect.DeepEqual(got.maxConcurrency, tt.want.maxConcurrency) ||
				reflect.DeepEqual(got.mutableStore, tt.want.mutableStore) ||
				reflect.DeepEqual(got.walStore, tt.want.walStore) ||
				reflect.DeepEqual(got.l, tt.want.l)) {
				t.Errorf("NewWAL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWAL_GetToken(t *testing.T) {
	t.Parallel()
	// Run against the real backend.
	mutableStore, cleanupMutable := setup(t, 0)
	defer cleanupMutable()
	walStore, cleanupWal := setup(t, 0)
	defer cleanupWal()
	_ = mutableStore.Put(context.Background(), tokenGeneratorPath, strings.NewReader(""), storage.OverWrite)
	l, err := zap.NewDevelopment()
	require.NoError(t, err)

	type fields struct {
		mutableStore       storage.Store
		tokenGeneratorPath string
		walStore           storage.Store
		maxConcurrency     int
		connectionControl  chan struct{}
		l                  *zap.Logger
	}
	type args struct {
		ctx context.Context
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Get a token",
			fields: fields{
				mutableStore:       mutableStore,
				tokenGeneratorPath: tokenGeneratorPath,
				walStore:           walStore,
				maxConcurrency:     maxConcurrency,
				connectionControl:  make(chan struct{}),
				l:                  l,
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: false,
		},
		{
			name: "Get a second token",
			fields: fields{
				mutableStore:       mutableStore,
				tokenGeneratorPath: tokenGeneratorPath,
				walStore:           walStore,
				maxConcurrency:     maxConcurrency,
				connectionControl:  make(chan struct{}),
				l:                  l,
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: false,
		},
		{
			name: "Get a third token",
			fields: fields{
				mutableStore:       mutableStore,
				tokenGeneratorPath: tokenGeneratorPath,
				walStore:           walStore,
				maxConcurrency:     maxConcurrency,
				connectionControl:  make(chan struct{}),
				l:                  l,
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: false,
		},
	}
	token1 := ""
	token2 := ""
	for _, test := range tests {
		tt := test
		t.Run(tt.name, func(t *testing.T) {
			token2 = token1
			w := &WAL{
				mutableStore:       tt.fields.mutableStore,
				tokenGeneratorPath: tt.fields.tokenGeneratorPath,
				walStore:           tt.fields.walStore,
				maxConcurrency:     tt.fields.maxConcurrency,
				connectionControl:  tt.fields.connectionControl,
				l:                  tt.fields.l,
			}
			token1, err = w.getToken(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("getToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if token1 == "" {
				t.Errorf("getToken() error = %v", err)
			}
			if token2 != "" {
				ks1, err := ksuid.Parse(token1)
				require.NoError(t, err, "Got error while converting token 1 to ksuid: %s", token1)
				ks2, err := ksuid.Parse(token2)
				require.NoError(t, err, "Got error while converting token2 to ksuid: %s", token2)
				time1 := ks1.Time()
				time2 := ks2.Time()
				diff := time1.Sub(time2)
				var second float64
				require.Greater(t, diff.Seconds(), second) // KSUID is at the granularity of seconds.
			}
			time.Sleep(1 * time.Second) // KSUID is at the granularity of seconds.
		})
	}
}

type mockMutableStoreTestAdd struct {
	storage.Store
	createTime  time.Time
	updateTime  time.Time
	mutex       sync.Mutex
	storeType   string
	failGetAttr bool
	failTouch   bool
	failPut     bool
}

func (m *mockMutableStoreTestAdd) String() string {
	panic("implement me")
}

func (m *mockMutableStoreTestAdd) Has(context.Context, string) (bool, error) {
	panic("implement me")
}

func (m *mockMutableStoreTestAdd) Get(context.Context, string) (io.ReadCloser, error) {
	panic("implement me")
}

func (m *mockMutableStoreTestAdd) GetAttr(context.Context, string) (storage.Attributes, error) {
	if m.failGetAttr {
		return storage.Attributes{}, fmt.Errorf(attrError)
	}
	if m.storeType != mutable {
		return storage.Attributes{}, fmt.Errorf("getattr expected only on mutable store")
	}
	return storage.Attributes{
		Created: m.createTime,
		Updated: m.updateTime,
		Owner:   "",
	}, nil
}

func (m *mockMutableStoreTestAdd) GetAt(context.Context, string) (io.ReaderAt, error) {
	panic("implement me")
}

func (m *mockMutableStoreTestAdd) Touch(_ context.Context, path string) error {
	if m.failTouch {
		return fmt.Errorf(touchError)
	}
	if m.storeType != mutable {
		return fmt.Errorf("touch expected only on mutable store")
	}
	if path != tokenGeneratorPath {
		return fmt.Errorf("expected path: %s", tokenGeneratorPath)
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()
	oldTime := m.updateTime
	m.updateTime = time.Now() // Assume stable local clock
	if m.updateTime.Before(oldTime) {
		panic("local wall clock not stable")
	}
	return nil
}

func (m *mockMutableStoreTestAdd) Put(_ context.Context, key string, reader io.Reader, overwrite storage.NewKey) error {
	if m.failPut {
		return fmt.Errorf(putError)
	}
	if m.storeType != wal {
		return fmt.Errorf("put expected only on wal store")
	}
	_, err := ksuid.Parse(key)
	if err != nil {
		return err
	}
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}
	if payload != string(b) {
		return fmt.Errorf("payload does not match: %s", string(b))
	}
	if overwrite == storage.OverWrite {
		return fmt.Errorf("no overwrites expected")
	}
	return nil
}

func (m *mockMutableStoreTestAdd) Delete(context.Context, string) error {
	panic("implement me")
}

func (m *mockMutableStoreTestAdd) Keys(context.Context) ([]string, error) {
	panic("implement me")
}

func (m *mockMutableStoreTestAdd) Clear(context.Context) error {
	panic("implement me")
}

func (m *mockMutableStoreTestAdd) KeysPrefix(ctx context.Context, pageToken string, prefix string, delimiter string, count int) ([]string, string, error) {
	panic("implement me")
}

func TestWAL_Add(t *testing.T) {
	t.Parallel()
	type fields struct {
		mutableStore       storage.Store
		tokenGeneratorPath string
		walStore           storage.Store
		maxConcurrency     int
		connectionControl  chan struct{}
		l                  *zap.Logger
	}
	type args struct {
		ctx context.Context
		p   string
	}
	l, err := zap.NewDevelopment()
	require.NoError(t, err)

	tests := []struct {
		name          string
		fields        fields
		args          args
		wantErr       bool
		validateError func(err error) bool
		errString     string
	}{
		{
			name: "add-success",
			fields: fields{
				mutableStore: &mockMutableStoreTestAdd{
					createTime: time.Now(),
					updateTime: time.Now(),
					mutex:      sync.Mutex{},
					storeType:  "mutable",
				},
				tokenGeneratorPath: tokenGeneratorPath,
				walStore: &mockMutableStoreTestAdd{
					createTime: time.Now(),
					updateTime: time.Now(),
					mutex:      sync.Mutex{},
					storeType:  "wal",
				},
				maxConcurrency:    0,
				connectionControl: nil,
				l:                 l,
			},
			args: args{
				ctx: nil,
				p:   payload,
			},
			wantErr: false,
		},
		{
			name: "add-failure-attr",
			fields: fields{
				mutableStore: &mockMutableStoreTestAdd{
					createTime:  time.Now(),
					updateTime:  time.Now(),
					mutex:       sync.Mutex{},
					storeType:   "mutable",
					failGetAttr: true,
				},
				tokenGeneratorPath: tokenGeneratorPath,
				walStore: &mockMutableStoreTestAdd{
					createTime: time.Now(),
					updateTime: time.Now(),
					mutex:      sync.Mutex{},
					storeType:  "wal",
				},
				maxConcurrency:    0,
				connectionControl: nil,
				l:                 l,
			},
			args: args{
				ctx: nil,
				p:   payload,
			},
			wantErr: true,
			validateError: func(err error) bool {
				return strings.Contains(err.Error(), attrError)
			},
		},
		{
			name: "add-failure-touch",
			fields: fields{
				mutableStore: &mockMutableStoreTestAdd{
					createTime: time.Now(),
					updateTime: time.Now(),
					mutex:      sync.Mutex{},
					storeType:  "mutable",
					failTouch:  true,
				},
				tokenGeneratorPath: tokenGeneratorPath,
				walStore: &mockMutableStoreTestAdd{
					createTime: time.Now(),
					updateTime: time.Now(),
					mutex:      sync.Mutex{},
					storeType:  "wal",
				},
				maxConcurrency:    0,
				connectionControl: nil,
				l:                 l,
			},
			args: args{
				ctx: nil,
				p:   payload,
			},
			wantErr: true,
			validateError: func(err error) bool {
				return strings.Contains(err.Error(), touchError)
			},
		},
		{
			name: "add-failure-put",
			fields: fields{
				mutableStore: &mockMutableStoreTestAdd{
					createTime: time.Now(),
					updateTime: time.Now(),
					mutex:      sync.Mutex{},
					storeType:  "mutable",
				},
				tokenGeneratorPath: tokenGeneratorPath,
				walStore: &mockMutableStoreTestAdd{
					createTime: time.Now(),
					updateTime: time.Now(),
					mutex:      sync.Mutex{},
					storeType:  "wal",
					failPut:    true,
				},
				maxConcurrency:    0,
				connectionControl: nil,
				l:                 l,
			},
			args: args{
				ctx: nil,
				p:   payload,
			},
			wantErr: true,
			validateError: func(err error) bool {
				return strings.Contains(err.Error(), putError)
			},
		},
	}
	for _, test := range tests {
		tt := test
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := &WAL{
				mutableStore:       tt.fields.mutableStore,
				tokenGeneratorPath: tt.fields.tokenGeneratorPath,
				walStore:           tt.fields.walStore,
				maxConcurrency:     tt.fields.maxConcurrency,
				connectionControl:  tt.fields.connectionControl,
				l:                  tt.fields.l,
			}
			if _, err := w.Add(tt.args.ctx, tt.args.p); (err != nil) != tt.wantErr {
				t.Errorf("Add() error = %v, wantErr %v", err, tt.wantErr)
			} else if err != nil {
				if !tt.validateError(err) {
					t.Errorf("Error validation failed")
				}
			}
		})
	}
}

type mockMutableStoreTestListEntries struct {
	storage.Store
	keys []string
}

func (m *mockMutableStoreTestListEntries) generateLexicallySortedKeys(count int) {
	m.keys = []string{}
	t := time.Now()
	a := t.Add(-1 * time.Hour)
	key, _ := ksuid.NewRandomWithTime(a)
	m.keys = append(m.keys, key.String())
	delta := time.Hour
	for i := 0; i < count-1; i++ {
		delta += time.Hour
		a = t.Add(delta)
		key, _ := ksuid.NewRandomWithTime(a)
		m.keys = append(m.keys, key.String())
	}
}

func (m *mockMutableStoreTestListEntries) String() string {
	panic("implement me")
}

func (m *mockMutableStoreTestListEntries) Has(context.Context, string) (bool, error) {
	panic("implement me")
}

type rc struct {
	s string
}

func (r *rc) Close() error {
	return nil
}

func (r *rc) Read(p []byte) (n int, err error) {
	e := Entry{
		Token:   r.s,
		Payload: r.s,
	}
	b, err := Marshal(&e)
	if err != nil {
		return 0, err
	}
	c := copy(p, b)
	return c, io.EOF
}

func (m *mockMutableStoreTestListEntries) Get(_ context.Context, key string) (io.ReadCloser, error) {
	randSleep()
	return &rc{
		s: key,
	}, nil
}

func (m *mockMutableStoreTestListEntries) GetAttr(context.Context, string) (storage.Attributes, error) {
	panic("implement me")

}

func (m *mockMutableStoreTestListEntries) GetAt(context.Context, string) (io.ReaderAt, error) {
	panic("implement me")
}

func (m *mockMutableStoreTestListEntries) Touch(_ context.Context, path string) error {
	panic("implement me")
}

func (m *mockMutableStoreTestListEntries) Put(_ context.Context, _ string, _ io.Reader, _ storage.NewKey) error {
	panic("implement me")
}

func (m *mockMutableStoreTestListEntries) Delete(context.Context, string) error {
	panic("implement me")
}

func (m *mockMutableStoreTestListEntries) Keys(context.Context) ([]string, error) {
	panic("implement me")
}

func (m *mockMutableStoreTestListEntries) Clear(context.Context) error {
	panic("implement me")
}

func (m *mockMutableStoreTestListEntries) KeysPrefix(ctx context.Context,
	pageToken string, prefix string, delimiter string, count int) ([]string, string, error) {
	var keys []string
	var next string
	var found bool
	i := 0
	randSleep()
	for {

		if i == len(m.keys)-1 {
			found = true
		} else {
			c := strings.Compare(m.keys[i+1], pageToken)
			if !found && c >= 0 {
				found = true
			}
		}
		if !found {
			i++
			continue
		}

		keys = append(keys, m.keys[i])
		if i == len(m.keys)-1 {
			next = ""
			return keys, next, nil
		}
		next = m.keys[i+1]
		i++
		if count == len(keys) {
			return keys, next, nil
		}
	}
}

func TestWAL_ListEntries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		keyCount        int
		expectError     bool
		errorValidation func(err error) bool
		setupParams     func(keys []string) (expected string, start string, max int, next string)
		maxConnections  int
	}{
		{
			name:           "List with defaults",
			keyCount:       2048,
			maxConnections: 1024,
			setupParams: func(keys []string) (expected string, start string, max int, next string) {
				return keys[0], keys[0], len(keys), keys[1000]
			},
		},
		{
			name:           "List with lower max than total keys",
			keyCount:       2048,
			maxConnections: 1024,
			setupParams: func(keys []string) (expected string, start string, max int, next string) {
				return keys[0], keys[0], 900, keys[900]
			},
		},
		{
			name:           "List with defaults, max concurrency at 1",
			keyCount:       100,
			maxConnections: 1,
			setupParams: func(keys []string) (expected string, start string, max int, next string) {
				return keys[0], keys[0], len(keys), ""
			},
		},
		{
			name:           "List ksuid between 2 keys with ksuid reduced",
			keyCount:       100,
			maxConnections: 10,
			setupParams: func(keys []string) (expected string, start string, max int, next string) {
				k, _ := ksuid.Parse(keys[2])
				s, _ := ksuid.NewRandomWithTime(k.Time().Add(-10 * time.Second))
				return keys[1], s.String(), 2, keys[3]
			},
		},
		{
			name:           "List ksuid between 2 keys with ksuid as is",
			keyCount:       100,
			maxConnections: 10,
			setupParams: func(keys []string) (expected string, start string, max int, next string) {
				return keys[1], keys[2], 2, keys[3]
			},
		},
		{
			name:           "List 0",
			keyCount:       100,
			maxConnections: 10,
			setupParams: func(keys []string) (expected string, start string, max int, next string) {
				return keys[1], keys[2], 0, keys[0]
			},
			expectError: true,
			errorValidation: func(err error) bool {
				return strings.Contains(err.Error(), "max")
			},
		},
	}
	l, err := zap.NewDevelopment()
	require.NoError(t, err)
	for _, test := range tests {
		tt := test
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := mockMutableStoreTestListEntries{}
			s.generateLexicallySortedKeys(tt.keyCount)
			first, start, max, nextExpected := tt.setupParams(s.keys)
			w := WAL{
				walStore:          &s,
				connectionControl: make(chan struct{}, tt.maxConnections),
				l:                 l,
			}
			keys, nextActual, err := w.ListEntries(context.Background(), start, max)
			if err != nil {
				if !tt.expectError {
					t.Errorf("ListEntries() error = %v", err)
					return
				}
				if !tt.errorValidation(err) {
					t.Errorf("Failed to validate error")
				}
				return
			}

			if first != keys[0].Token {
				t.Errorf("incorrect first entry. Actual:%s, expected:%s", keys[0].Token, first)
			}
			l.Debug("Next token", zap.String("actual", nextActual), zap.String("expected", nextExpected))
			if nextActual != nextExpected {
				t.Errorf("failed to get the correct next token, actual: %s, expected:%s", nextActual, nextExpected)
			}

			l.Debug("counts", zap.Int("max", max), zap.Int("length", len(keys)))

			if max >= maxEntriesPerList {
				if len(s.keys) >= maxEntriesPerList && len(keys) != maxEntriesPerList {
					t.Errorf("maxEnteriesPerList should be the number of keys retured. Actual: %d, len:%d, Expected:%d", len(keys), len(s.keys), maxEntriesPerList)
				}
				if len(s.keys) < maxEntriesPerList && len(keys) != len(s.keys) {
					t.Errorf("Number of returned keys should be the length of all keys. Actual: %d, len:%d, Expected:%d", len(keys), len(s.keys), maxEntriesPerList)
				}
			} else if max > 0 {
				if (max < len(s.keys) && len(keys) != max) || (len(keys) > len(s.keys)) {
					t.Errorf("total number of keys is incorrect. len: %d, actual:%d, max: %d", len(s.keys), len(keys), max)
				}
			}
		})
	}
}
