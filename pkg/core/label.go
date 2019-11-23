package core

import (
	"bytes"
	"context"
	"hash/crc32"
	"io/ioutil"
	"time"

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

// LabelOption is a functor to build labels
type LabelOption func(*Label)

// LabelDescriptorOption is a functor to build label descriptors
type LabelDescriptorOption func(descriptor *model.LabelDescriptor)

// LabelContributors sets a list of contributors for the label
func LabelContributors(c []model.Contributor) LabelDescriptorOption {
	return func(ld *model.LabelDescriptor) {
		ld.Contributors = c
	}
}

func getLabelStore(stores context2.Stores) storage.Store {
	return stores.VMetadata()
}

// LabelContributor sets a single contributor for the label
func LabelContributor(c model.Contributor) LabelDescriptorOption {
	return LabelContributors([]model.Contributor{c})
}

// NewLabelDescriptor builds a new label descriptor
func NewLabelDescriptor(descriptorOps ...LabelDescriptorOption) *model.LabelDescriptor {
	ld := model.LabelDescriptor{
		Timestamp: time.Now(),
	}
	for _, apply := range descriptorOps {
		apply(&ld)
	}
	return &ld
}

// LabelName sets a name for the label
func LabelName(name string) LabelOption {
	return func(l *Label) {
		l.Descriptor.Name = name
	}
}

// NewLabel builds a new label with a descriptor
func NewLabel(ld *model.LabelDescriptor, labelOps ...LabelOption) *Label {
	if ld == nil {
		ld = NewLabelDescriptor()
	}
	label := Label{
		Descriptor: *ld,
	}
	for _, apply := range labelOps {
		apply(&label)
	}
	return &label
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

// GetLabelStore extracts the versioning metadata store from some context's stores
func GetLabelStore(stores context2.Stores) storage.Store {
	return getVMetaStore(stores)
}
