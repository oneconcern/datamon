package core

import (
	"context"

	context2 "github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/errors"
	"github.com/oneconcern/datamon/pkg/model"

	"github.com/oneconcern/datamon/pkg/storage"
)

// GetDiamondStore selects the metadata store for diamonds from a context
//
// In the current setup, diamond metadata are located in the vmetadata store.
func GetDiamondStore(stores context2.Stores) storage.Store {
	return getVMetaStore(stores)
}

// GetDiamond retrieves a diamond
func GetDiamond(repo, diamondID string, stores context2.Stores) (model.DiamondDescriptor, error) {
	if err := RepoExists(repo, stores); err != nil {
		return model.DiamondDescriptor{}, err
	}

	d := NewDiamond(repo, stores,
		DiamondDescriptor(
			model.NewDiamondDescriptor(model.DiamondID(diamondID)),
		),
	)

	err := d.downloadDescriptor()
	if err != nil {
		return model.DiamondDescriptor{}, err
	}
	return d.DiamondDescriptor, nil
}

// DiamondExists checks if a diamond exists on a repo
func DiamondExists(repo, diamondID string, stores context2.Stores) error {
	exists, err := GetDiamondStore(stores).Has(context.Background(), model.GetArchivePathToInitialDiamond(repo, diamondID))
	if err != nil {
		return errors.New("failed to retrieve diamond from store").Wrap(err)
	}
	if !exists {
		return errors.New("diamond validation").WrapMessage("diamond %s doesn't exist for repo %s ", diamondID, repo)
	}
	return nil
}

// diamondReady returns an error if the diamond is not in one of the following states:
//
//   * DiamondInitialized
//   * DiamondRunning
//
// Diamonds in the DiamondCommitting state are not ready unless the state is older than 30s (stale lock state).
func diamondReady(repo, diamondID string, stores context2.Stores) error {
	diamond, err := GetDiamond(repo, diamondID, stores)
	if err != nil {
		return err
	}
	switch diamond.State {
	case model.DiamondInitialized:
		return nil
	default:
		if !diamond.State.IsValid() {
			return errors.New("diamond is not ready").WrapMessage("invalid metadata for state: %q", diamond.State)
		}
		return errors.New("diamond is not ready").WrapMessage("diamond state is: %q:", diamond.State)
	}
}

// CreateDiamond persists an initialized diamond with a repo descriptor
func CreateDiamond(repo string, stores context2.Stores, opts ...DiamondOption) (model.DiamondDescriptor, error) {
	if err := RepoExists(repo, stores); err != nil {
		return model.DiamondDescriptor{}, err
	}

	d := NewDiamond(repo, stores, opts...)
	if d.DiamondDescriptor.DiamondID == "" {
		return model.DiamondDescriptor{}, errors.New("a diamond must have a diamondID")
	}

	err := d.uploadDescriptor()
	if err != nil {
		return model.DiamondDescriptor{},
			errors.New("cannot update diamond descriptor").Wrap(err)
	}
	return d.DiamondDescriptor, nil
}
