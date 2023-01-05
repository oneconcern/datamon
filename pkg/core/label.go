package core

import (
	"bytes"
	"context"
	"hash/crc32"
	"io"
	"io/ioutil"
	"time"

	context2 "github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/metrics"

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
	version    string

	metrics.Enable
	m *M
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

	if label.MetricsEnabled() {
		label.m = label.EnsureMetrics("core", &M{}).(*M)
	}
	return label
}

// UploadDescriptor persists the label descriptor for a bundle
func (label *Label) UploadDescriptor(ctx context.Context, bundle *Bundle) (err error) {
	defer func(t0 time.Time) {
		if label.MetricsEnabled() {
			label.m.Usage.UsedAll(t0, "LabelUpload")(err)
		}
	}(time.Now())

	err = RepoExists(bundle.RepoID, bundle.contextStores)
	if err != nil {
		return err
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

// DownloadDescriptorVersions retrieves all versions of a label descriptor for a bundle
func (label *Label) DownloadDescriptorVersions(ctx context.Context, bundle *Bundle, checkRepoExists bool) (lds []model.LabelDescriptor, err error) {
	defer func(t0 time.Time) {
		if label.MetricsEnabled() {
			label.m.Usage.UsedAll(t0, "LabelDownloadVersions")(err)
		}
	}(time.Now())

	if checkRepoExists {
		err = RepoExists(bundle.RepoID, bundle.contextStores)
		if err != nil {
			return nil, err
		}
	}

	archivePath := model.GetArchivePathToLabel(bundle.RepoID, label.Descriptor.Name)
	has, err := getLabelStore(bundle.contextStores).Has(context.Background(), archivePath)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, status.ErrNotFound
	}

	vstore, ok := getLabelStore(bundle.contextStores).(storage.VersionedStore)
	if !ok {
		return nil, status.ErrVersionedStoreRequired
	}

	versions, err := vstore.KeyVersions(context.Background(), archivePath)
	if err != nil {
		return nil, err
	}

	for _, version := range versions {
		rdr, err := vstore.GetVersion(context.Background(), archivePath, version)
		if err != nil {
			return nil, err
		}
		o, err := ioutil.ReadAll(rdr)
		if err != nil {
			return nil, err
		}
		err = yaml.Unmarshal(o, &label.Descriptor)
		if err != nil {
			return nil, err
		}
		lds = append(lds, label.Descriptor)
	}
	return lds, nil
}

// DownloadDescriptor retrieves the label descriptor for a bundle
func (label *Label) DownloadDescriptor(ctx context.Context, bundle *Bundle, checkRepoExists bool) (err error) {
	defer func(t0 time.Time) {
		if label.MetricsEnabled() {
			label.m.Usage.UsedAll(t0, "LabelDownload")(err)
		}
	}(time.Now())

	if checkRepoExists {
		err = RepoExists(bundle.RepoID, bundle.contextStores)
		if err != nil {
			return err
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

	var rdr io.Reader
	if label.version != "" {
		vstore, ok := getLabelStore(bundle.contextStores).(storage.VersionedStore)
		if !ok {
			return status.ErrVersionedStoreRequired
		}
		rdr, err = vstore.GetVersion(context.Background(), archivePath, label.version)
	} else {
		rdr, err = getLabelStore(bundle.contextStores).Get(context.Background(), archivePath)
	}
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
