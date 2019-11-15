/*
 * Copyright © 2019 One Concern
 *
 */

package model

import "testing"

func TestValidateContext(t *testing.T) {
	type args struct {
		context Context
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success",
			args: args{
				context: Context{
					Name:      "context1",
					WAL:       "wal",
					ReadLog:   "read",
					Blob:      "blob",
					Metadata:  "md",
					VMetadata: "vmd",
					Version:   0,
				},
			},
			wantErr: false,
		},
		{
			name: "fail read",
			args: args{
				context: Context{
					WAL:       "wal",
					ReadLog:   "read",
					Blob:      "blob",
					Metadata:  "md",
					VMetadata: "vmd",
					Version:   0,
				},
			},
			wantErr: true,
		},
		{
			name: "fail wal",
			args: args{
				context: Context{
					Name:      "context1",
					ReadLog:   "read",
					Blob:      "blob",
					Metadata:  "md",
					VMetadata: "vmd",
					Version:   0,
				},
			},
			wantErr: true,
		},
		{
			name: "fail read",
			args: args{
				context: Context{
					Name:      "context1",
					WAL:       "wal",
					Blob:      "blob",
					Metadata:  "md",
					VMetadata: "vmd",
					Version:   0,
				},
			},
			wantErr: true,
		},
		{
			name: "fail blob",
			args: args{
				context: Context{
					Name:      "context1",
					WAL:       "wal",
					ReadLog:   "read",
					Metadata:  "md",
					VMetadata: "vmd",
					Version:   0,
				},
			},
			wantErr: true,
		},
		{
			name: "fail md",
			args: args{
				context: Context{
					Name:      "context1",
					WAL:       "wal",
					ReadLog:   "read",
					Blob:      "blob",
					VMetadata: "vmd",
					Version:   0,
				},
			},
			wantErr: true,
		},
		{
			name: "fail vmd",
			args: args{
				context: Context{
					Name:     "context1",
					WAL:      "wal",
					ReadLog:  "read",
					Blob:     "blob",
					Metadata: "md",
					Version:  0,
				},
			},
			wantErr: true,
		},
		{
			name: "fail ver",
			args: args{
				context: Context{
					Name:      "context1",
					WAL:       "wal",
					ReadLog:   "read",
					Blob:      "blob",
					Metadata:  "md",
					VMetadata: "vmd",
					Version:   2,
				},
			},
			wantErr: true,
		},
	}
	for _, tts := range tests {
		tt := tts
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := ValidateContext(tt.args.context); (err != nil) != tt.wantErr {
				t.Errorf("ValidateContext() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
