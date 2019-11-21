/*
 * Copyright © 2019 One Concern
 *
 */

package model

import (
	"testing"
	"time"
)

func TestGetArchivePathToLabel(t *testing.T) {
	type args struct {
		repo      string
		labelName string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "label1",
			args: args{
				repo:      "repo1",
				labelName: "label1",
			},
			want: "labels/repo1/label1/label.yaml",
		},
	}
	for _, tts := range tests {
		tt := tts
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := GetArchivePathToLabel(tt.args.repo, tt.args.labelName); got != tt.want {
				t.Errorf("GetArchivePathToLabel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateLabel(t *testing.T) {
	type args struct {
		label LabelDescriptor
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success",
			args: args{
				label: LabelDescriptor{
					Name:      "label1",
					BundleID:  "bundleID",
					Timestamp: time.Time{},
					Contributors: []Contributor{
						{
							Name:  "name",
							Email: "email@domain.com",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "success with hyphens",
			args: args{
				label: LabelDescriptor{
					Name:      "label1-",
					BundleID:  "bundleID",
					Timestamp: time.Time{},
					Contributors: []Contributor{
						{
							Name:  "name",
							Email: "email@domain.com",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "success with Connector punctuation",
			args: args{
				label: LabelDescriptor{
					Name:      "label1_1-1",
					BundleID:  "bundleID",
					Timestamp: time.Time{},
					Contributors: []Contributor{
						{
							Name:  "name",
							Email: "email@domain.com",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "failure with /",
			args: args{
				label: LabelDescriptor{
					Name:      "label1/asd",
					BundleID:  "bundleID",
					Timestamp: time.Time{},
					Contributors: []Contributor{
						{
							Name:  "name",
							Email: "email@domain.com",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Non alphanumeric",
			args: args{
				label: LabelDescriptor{
					Name:      "label{",
					BundleID:  "bundleID",
					Timestamp: time.Time{},
					Contributors: []Contributor{
						{
							Name:  "name",
							Email: "email@domain.com",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "space failure",
			args: args{
				label: LabelDescriptor{
					Name:      "label with spaces is not supported",
					BundleID:  "bundleID",
					Timestamp: time.Time{},
					Contributors: []Contributor{
						{
							Name:  "name",
							Email: "email@domain.com",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "space failure",
			args: args{
				label: LabelDescriptor{
					Name:      "label with spaces is not supported",
					BundleID:  "bundleID",
					Timestamp: time.Time{},
					Contributors: []Contributor{
						{
							Name:  "name",
							Email: "email@domain.com",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "blank name",
			args: args{
				label: LabelDescriptor{
					Name:      "label",
					BundleID:  "bundleID",
					Timestamp: time.Time{},
					Contributors: []Contributor{
						{
							Name:  "",
							Email: "email@domain.com",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "blank email",
			args: args{
				label: LabelDescriptor{
					Name:      "label",
					BundleID:  "bundleID",
					Timestamp: time.Time{},
					Contributors: []Contributor{
						{
							Name:  "name",
							Email: "",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid email",
			args: args{
				label: LabelDescriptor{
					Name:      "label",
					BundleID:  "bundleID",
					Timestamp: time.Time{},
					Contributors: []Contributor{
						{
							Name:  "name",
							Email: "asdasfsaf",
						},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tts := range tests {
		tt := tts
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := ValidateLabel(tt.args.label); (err != nil) != tt.wantErr {
				t.Errorf("ValidateLabel() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
