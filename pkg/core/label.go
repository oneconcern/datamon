package core

import (
	"bytes"
	"context"
	"hash/crc32"
	"io/ioutil"

	context2 "github.com/oneconcern/datamon/pkg/context"

	"gopkg.in/yaml.v2"

	"github.com/oneconcern/datamon/pkg/core/status"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
)

// Label describes a bundle label.
//
// A label is a name given to a bundle, analogous to tags in git.
// Examples: Latest, production.
type Label struct {
	Descriptor model.LabelDescriptor
}

func defaultLabel() *Label {
	return &Label{
		Descriptor: *model.NewLabelDescriptor(),
	}
}

// NewLabel builds a new label with a descriptor
func NewLabel(opts ...LabelOption) *Label {
	label := defaultLabel()
	for _, apply := range opts {
		apply(label)
	}
	return label
}

// UploadDescriptor persists the label descriptor for a bundle
func (label *Label) UploadDescriptor(ctx context.Context, bundle *Bundle) error {
	e := RepoExists(bundle.RepoID, bundle.contextStores)
	if e != nil {
		return e
	}
	label.Descriptor.BundleID = bundle.BundleID
	buffer, err := yaml.Marshal(label.Descriptor)
	if err != nil {
		return err
	}
	lsCRC, ok := bundle.contextStores.VMetadata().(storage.StoreCRC)
	if ok {
		crc := crc32.Checksum(buffer, crc32.MakeTable(crc32.Castagnoli))
		err = lsCRC.PutCRC(ctx,
			model.GetArchivePathToLabel(bundle.RepoID, label.Descriptor.Name),
			bytes.NewReader(buffer), storage.OverWrite, crc)

	} else {
		err = bundle.contextStores.VMetadata().Put(ctx,
			model.GetArchivePathToLabel(bundle.RepoID, label.Descriptor.Name),
			bytes.NewReader(buffer), storage.OverWrite)
	}
	if err != nil {
		return err
	}
	return nil
}

// DownloadDescriptor retrieves the label descriptor for a bundle
func (label *Label) DownloadDescriptor(ctx context.Context, bundle *Bundle, checkRepoExists bool) error {
	if checkRepoExists {
		e := RepoExists(bundle.RepoID, bundle.contextStores)
		if e != nil {
			return e
		}
	}
	archivePath := model.GetArchivePathToLabel(bundle.RepoID, label.Descriptor.Name)
	has, err := getLabelStore(bundle.contextStores).Has(context.Background(), archivePath)
	if err != nil {
		return err
	}
	if !has {
		return status.ErrNotFound
	}
	rdr, err := getLabelStore(bundle.contextStores).Get(context.Background(), archivePath)
	if err != nil {
		return err
	}
	o, err := ioutil.ReadAll(rdr)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(o, &label.Descriptor)
	if err != nil {
		return err
	}
	return nil
}

// GetLabelStore tells which store holds label metadata
func GetLabelStore(stores context2.Stores) storage.Store {
	return getLabelStore(stores)
}

func getLabelStore(stores context2.Stores) storage.Store {
	return stores.VMetadata()
}
