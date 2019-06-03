package core

import (
	"bytes"
	"context"
	"hash/crc32"
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
)

type Label struct {
	Name       string
	Descriptor model.LabelDescriptor
}

type LabelOption func(*Label)
type LabelDescriptorOption func(descriptor *model.LabelDescriptor)

func LabelContributors(c []model.Contributor) LabelDescriptorOption {
	return func(ld *model.LabelDescriptor) {
		ld.Contributors = c
	}
}

func NewLabelDescriptor(descriptorOps ...LabelDescriptorOption) *model.LabelDescriptor {
	ld := model.LabelDescriptor{
		Timestamp: time.Now(),
	}
	for _, apply := range descriptorOps {
		apply(&ld)
	}
	return &ld
}

func LabelName(name string) LabelOption {
	return func(b *Label) {
		b.Name = name
	}
}

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

func (label *Label) UploadDescriptor(ctx context.Context, bundle *Bundle) error {
	e := RepoExists(bundle.RepoID, bundle.MetaStore)
	if e != nil {
		return e
	}
	label.Descriptor.BundleID = bundle.BundleID
	buffer, err := yaml.Marshal(label.Descriptor)
	if err != nil {
		return err
	}
	lsCRC, ok := bundle.MetaStore.(storage.StoreCRC)
	if ok {
		crc := crc32.Checksum(buffer, crc32.MakeTable(crc32.Castagnoli))
		err = lsCRC.PutCRC(ctx,
			model.GetArchivePathToLabel(bundle.RepoID, label.Name),
			bytes.NewReader(buffer), storage.OverWrite, crc)

	} else {
		err = bundle.MetaStore.Put(ctx,
			model.GetArchivePathToLabel(bundle.RepoID, label.Name),
			bytes.NewReader(buffer), storage.OverWrite)
	}
	if err != nil {
		return err
	}
	return nil
}

func (label *Label) DownloadDescriptor(ctx context.Context, bundle *Bundle, checkRepoExists bool) error {
	if checkRepoExists {
		e := RepoExists(bundle.RepoID, bundle.MetaStore)
		if e != nil {
			return e
		}
	}
	rdr, err := bundle.MetaStore.Get(context.Background(), model.GetArchivePathToLabel(bundle.RepoID, label.Name))
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
