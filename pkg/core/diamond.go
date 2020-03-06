package core

import (
	"path"
	"strings"

	context2 "github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/errors"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	storagestatus "github.com/oneconcern/datamon/pkg/storage/status"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

// Diamond is a generalization of Bundle to support diamond operations
type Diamond struct {
	metaObject
	*Bundle

	DiamondDescriptor model.DiamondDescriptor

	splitIndexer  *fileIndex                  // downloads split index files
	bundleIndexer *fileIndex                  // uploads bundle index files
	deconflicter  func(string, string) string // a renaming func to move conflicting files
	_             struct{}
}

func defaultDiamond(repo string, stores context2.Stores) *Diamond {
	return &Diamond{
		metaObject:        defaultMetaObject(stores.VMetadata()),
		Bundle:            NewBundle(Repo(repo), ContextStores(stores)),
		DiamondDescriptor: *model.NewDiamondDescriptor(),
	}
}

// NewDiamond builds a new diamond instance.
//
// The default diamond gets populated with a random KSUID as diamondID.
//
// Default diamond has cnflicts handling enabled.
func NewDiamond(repo string, stores context2.Stores, opts ...DiamondOption) *Diamond {
	diamond := defaultDiamond(repo, stores)
	for _, apply := range opts {
		apply(diamond)
	}

	if diamond.deconflicter == nil {
		// points to the appropriate metadata path rendering function from model,
		// depending on the conflicts handling  mode selected.
		switch diamond.DiamondDescriptor.Mode {
		case model.EnableCheckpoints:
			diamond.deconflicter = model.GenerateCheckpointPath
		case model.ForbidConflicts:
			diamond.deconflicter = func(a, b string) string {
				diamond.l.Error("dev error: deconflicter called in inadequate context", zap.String("arg", a), zap.String("arg", b))
				panic("dev error: must not call deconflicter")
			}
		case model.EnableConflicts:
			fallthrough
		default:
			diamond.deconflicter = model.GenerateConflictPath
		}
	}

	if diamond.DiamondDescriptor.Tag != "" {
		diamond.l = diamond.l.With(zap.String("tag", diamond.DiamondDescriptor.Tag))
		diamond.Bundle.l = diamond.l
	}

	return diamond
}

func (d *Diamond) downloadDescriptor() error {
	// try retrieving descriptor in final state
	src := model.GetArchivePathToFinalDiamond(
		d.RepoID,
		d.DiamondDescriptor.DiamondID,
	)

	buffer, err := d.readMetadata(src)
	if err != nil {
		if !errors.Is(err, storagestatus.ErrNotExists) {
			return err
		}
		// try retrieving descriptor in initial state
		src = model.GetArchivePathToInitialDiamond(
			d.RepoID,
			d.DiamondDescriptor.DiamondID,
		)
		buffer, err = d.readMetadata(src)
		if err != nil {
			return err
		}
	}

	var desc model.DiamondDescriptor
	err = yaml.Unmarshal(buffer, &desc)
	if err != nil {
		return err
	}
	d.DiamondDescriptor = desc
	return nil
}

func (d *Diamond) uploadDescriptor() error {
	buffer, err := yaml.Marshal(d.DiamondDescriptor)
	if err != nil {
		return err
	}

	dest := model.GetArchivePathToDiamond(
		d.RepoID,
		d.DiamondDescriptor.DiamondID,
		d.DiamondDescriptor.State,
	)

	return d.writeMetadata(dest, storage.NoOverWrite, buffer)
}

// basenameKeyFilter applies a filter on results from some iterator (e.g. the KeysPrefix store function).
//
// This is useful to filter out items located deeper in the metadata tree, but for which the simple separator rule
// cannot be applied.
func basenameKeyFilter(filter string) func([]string, string, error) ([]string, string, error) {
	return func(keys []string, next string, err error) ([]string, string, error) {
		if err != nil {
			return keys, next, err
		}
		filtered := make([]string, 0, len(keys))
		for _, key := range keys {
			if !strings.HasPrefix(path.Base(key), filter) {
				continue
			}
			filtered = append(filtered, key)
		}
		return filtered, next, err
	}
}
