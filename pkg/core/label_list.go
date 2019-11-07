package core

import (
	"context"
	"fmt"

	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
)

const (
	maxLabelsToList = 1000000
)

func ListLabels(repo string, metaStore storage.Store, prefix string) ([]model.LabelDescriptor, error) {
	e := RepoExists(repo, metaStore)
	if e != nil {
		return nil, e
	}
	ks, _, err := metaStore.KeysPrefix(context.Background(), "",
		model.GetArchivePathPrefixToLabels(repo, prefix), "",
		maxLabelsToList)

	if err != nil {
		return nil, err
	}
	labelDescriptors := make([]model.LabelDescriptor, 0)
	bundle := New(NewBDescriptor(),
		Repo(repo),
		MetaStore(metaStore),
	)
	for _, k := range ks {
		apc, err := model.GetArchivePathComponents(k)
		if err != nil {
			return nil, err
		}
		labelName := apc.LabelName
		label := NewLabel(nil,
			LabelName(labelName),
		)
		err = label.DownloadDescriptor(context.Background(), bundle, false)
		if err != nil {
			return nil, err
		}
		if label.Descriptor.Name == "" {
			label.Descriptor.Name = apc.LabelName
		} else if label.Descriptor.Name != apc.LabelName {
			return nil, fmt.Errorf("label names in descriptor '%v' and archive path '%v' don't match",
				label.Descriptor.Name, apc.LabelName)
		}
		labelDescriptors = append(labelDescriptors, label.Descriptor)
	}
	return labelDescriptors, nil
}
