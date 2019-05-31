package core

import (
	"context"

	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
)

func ListLabels(repo string, metaStore storage.Store) ([]string, error) {
	e := RepoExists(repo, metaStore)
	if e != nil {
		return nil, e
	}
	ks, _, err := metaStore.KeysPrefix(context.Background(), "", model.GetArchivePathPrefixToLabels(repo), "", 1000000)
	if err != nil {
		return nil, err
	}
	var keys = make([]string, 0)
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
		keys = append(keys, labelName+" , "+label.Descriptor.BundleID+" , "+label.Descriptor.Timestamp.String())
	}
	return keys, nil
}
