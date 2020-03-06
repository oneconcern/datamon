package core

import (
	context2 "github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/core/status"
	"github.com/oneconcern/datamon/pkg/errors"
	storagestatus "github.com/oneconcern/datamon/pkg/storage/status"
	"go.uber.org/zap"

	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
)

// GetSplitStore selects the metadata store for splits from a context
//
// In the current setup, split metadata are located in the vmetadata store.
func GetSplitStore(stores context2.Stores) storage.Store {
	return getVMetaStore(stores)
}

// GetSplit retrieves a split
func GetSplit(repo, diamondID, splitID string, stores context2.Stores) (model.SplitDescriptor, error) {
	s := NewSplit(repo, diamondID, stores,
		SplitDescriptor(
			model.NewSplitDescriptor(model.SplitID(splitID)),
		),
	)
	err := s.downloadDescriptor()
	if err != nil {
		return model.SplitDescriptor{}, err
	}
	return s.SplitDescriptor, nil
}

// CreateSplit persists a new split for some initialized diamond for a repo
func CreateSplit(repo, diamondID string, stores context2.Stores, opts ...SplitOption) (model.SplitDescriptor, error) {
	if err := RepoExists(repo, stores); err != nil {
		return model.SplitDescriptor{}, errors.New("cannot create split on inexistant repo").Wrap(err)
	}

	if diamondID == "" {
		return model.SplitDescriptor{}, errors.New("diamondID is required to create a split")
	}

	if err := diamondReady(repo, diamondID, stores); err != nil {
		return model.SplitDescriptor{}, errors.New("cannot create split").Wrap(err)
	}

	s := NewSplit(repo, diamondID, stores, opts...)

	err := s.downloadDescriptor()
	if err == nil {
		// split already exists:
		// this occur when the user explicitly replays the same splitID
		logger := s.l.With(
			zap.String("split_id", s.SplitDescriptor.SplitID),
			zap.Stringer("state", s.SplitDescriptor.State))
		logger.Debug("already existing split. Proceed with consistency checks")

		switch s.SplitDescriptor.State {
		case model.SplitDone:
			return model.SplitDescriptor{}, status.ErrSplitAlreadyDone.
				WrapMessage("diamond ID: %s, split ID %s, split state: %v", diamondID, s.SplitDescriptor.SplitID, s.SplitDescriptor.State)

		case model.SplitRunning:
			// the split state is persisted as running, but the user is telling us that it is not and wants to restart.
			// Trust the user. No harm done anyway if the split is somehow still hanging around: index files are written on a new location.
			logger.Warn("restarting a split in running state")

		default:
			logger.Info("proceeding with restarting split")
		}
		// the split already exists, and is in some allowed state (e.g. failed)
		return s.SplitDescriptor, nil
	}

	if !errors.Is(err, storagestatus.ErrNotExists) && !errors.Is(err, storagestatus.ErrNotFound) {
		// another error
		return model.SplitDescriptor{}, err
	}

	// with some split settings, an existing is required
	if s.mustExist {
		return model.SplitDescriptor{}, status.ErrSplitMustExist.
			WrapMessage("diamond ID: %s, split ID: %s", diamondID, s.SplitDescriptor.SplitID)
	}

	// create new split with original model
	s = NewSplit(repo, diamondID, stores, opts...)

	err = s.uploadDescriptor()
	if err != nil {
		return model.SplitDescriptor{}, status.ErrSplitUpdate.Wrap(err)
	}

	return s.SplitDescriptor, nil
}
