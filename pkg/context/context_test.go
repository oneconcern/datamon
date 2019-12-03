/*
 * Copyright © 2019 One Concern
 *
 */

package context

import (
	"context"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/oneconcern/datamon/pkg/model"

	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/oneconcern/datamon/pkg/storage/localfs"
	"github.com/spf13/afero"
)

const (
	contextDir = "../../testdata/context"
)

func TestNewContext(t *testing.T) {
	type args struct {
		wal       storage.Store
		blob      storage.Store
		metadata  storage.Store
		vMetadata storage.Store
		readLog   storage.Store
	}
	s1 := localfs.New(afero.NewMemMapFs())
	s2 := localfs.New(afero.NewMemMapFs())
	s3 := localfs.New(afero.NewMemMapFs())
	s4 := localfs.New(afero.NewMemMapFs())
	s5 := localfs.New(afero.NewOsFs())
	tests := []struct {
		name string
		args args
		want Stores
	}{
		{
			name: "new",
			args: args{
				wal:       s1,
				blob:      s2,
				metadata:  s3,
				vMetadata: s4,
				readLog:   s5,
			},
			want: Stores{
				wal:       s1,
				blob:      s2,
				metadata:  s3,
				vMetadata: s4,
				readLog:   s5,
			},
		},
	}
	for _, tts := range tests {
		tt := tts
		t.Run(tt.name, func(t *testing.T) {
			if got := NewStores(tt.args.wal, tt.args.readLog, tt.args.blob, tt.args.metadata, tt.args.vMetadata); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewContext() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateContext(t *testing.T) {
	t.SkipNow()
	type args struct {
		ctx         context.Context
		configStore storage.Store
		context     model.Context
	}
	cleanup := func() {
		err := os.RemoveAll(contextDir)
		require.NoError(t, err)
	}
	defer cleanup()
	cleanup()
	store := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), contextDir))
	c1 := model.Context{
		Name:      "context",
		WAL:       "wal",
		ReadLog:   "readLog",
		Blob:      "blob",
		Metadata:  "metadata",
		VMetadata: "vmetadata",
		Version:   model.ContextVersion,
	}
	c2 := model.Context{
		Name:      "context2",
		WAL:       "wal",
		ReadLog:   "readLog",
		Blob:      "blob",
		Metadata:  "metadata",
		VMetadata: "vmetadata",
		Version:   model.ContextVersion,
	}
	tests :=
		[]struct {
			name    string
			args    args
			path    string
			wantErr bool
		}{
			{
				name: "success",
				path: contextDir + "/context/context.yaml",
				args: args{
					ctx:         context.Background(),
					configStore: store,
					context:     c1,
				},
				wantErr: false,
			},
			{
				name: "fail overwrite",
				path: contextDir + "/context/context.yaml",
				args: args{
					ctx:         context.Background(),
					configStore: store,
					context:     c1,
				},
				wantErr: true,
			},
			{
				name: "success 2",
				path: contextDir + "/context2/context.yaml",
				args: args{
					ctx:         context.Background(),
					configStore: store,
					context:     c2,
				},
				wantErr: false,
			},
		}
	for _, tts := range tests {
		tt := tts
		t.Run(tt.name, func(t *testing.T) {
			if err := CreateContext(tt.args.ctx, tt.args.configStore, tt.args.context); (err != nil) != tt.wantErr {
				t.Errorf("CreateContext() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				cs, err := ioutil.ReadFile(tt.path)
				require.NoError(t, err)
				c, err := model.UnmarshalContext(cs)
				require.NoError(t, err)
				require.True(t, reflect.DeepEqual(*c, tt.args.context))
			}
		})
	}
}
