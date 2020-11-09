package core

import (
	"fmt"
	"time"

	"github.com/oneconcern/datamon/pkg/cafs"
	context2 "github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/core/status"
	"github.com/oneconcern/datamon/pkg/errors"
	"github.com/oneconcern/datamon/pkg/metrics"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	storagestatus "github.com/oneconcern/datamon/pkg/storage/status"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

// Split is a generalization of a Bundle, to support diamond operations
type Split struct {
	metaObject
	*Bundle

	SplitDescriptor model.SplitDescriptor
	DiamondID       string

	getKeys       KeyIterator
	filter        KeyFilter
	uploadIndexer *fileIndex
	// downloadIndexer *fileIndex
	mustExist bool // used to check replayed splits with splitID forced

	metrics.Enable
	m *M
}

func defaultSplit(repo, diamondID string, stores context2.Stores) *Split {
	empty := NewBundle(Repo(repo), ContextStores(stores))
	split := model.NewSplitDescriptor()
	return &Split{
		metaObject:      defaultMetaObject(GetDiamondStore(stores)), // splits live on vmetadata
		Bundle:          empty,
		SplitDescriptor: *split,
		DiamondID:       diamondID,
	}
}

// NewSplit builds a split for core operations (get, ...) and makes Bundle capabilities available to a Split
func NewSplit(repo, diamondID string, stores context2.Stores, opts ...SplitOption) *Split {
	s := defaultSplit(repo, diamondID, stores)
	for _, apply := range opts {
		apply(s)
	}

	if s.getKeys == nil {
		s.getKeys = func(_ string) ([]string, error) {
			return s.ConsumableStore.Keys(s.contexter())
		}
	}
	if s.filter != nil {
		iterator := s.getKeys
		s.getKeys = func(next string) ([]string, error) {
			keys, err := iterator(next)
			if err != nil {
				return []string{}, err
			}
			result := make([]string, 0, len(keys))
			for _, key := range keys {
				if s.filter(key) {
					result = append(result, key)
				}
			}
			return result, nil
		}
	}

	if s.SplitDescriptor.Tag != "" {
		s.l = s.l.With(zap.String("tag", s.SplitDescriptor.Tag))
		s.Bundle.l = s.l
	}

	if s.MetricsEnabled() {
		s.m = s.EnsureMetrics("core", &M{}).(*M)
	}
	return s
}

// Upload a dataset as a split, without committing as a bundle
func (s *Split) Upload(opts ...Option) error {
	return s.implUpload(opts...)
}

func (s *Split) implUpload(opts ...Option) error {
	var err error
	defer func(t0 time.Time) {
		if s.MetricsEnabled() {
			s.m.Usage.UsedAll(t0, "SplitUpload")(err)
		}
	}(time.Now())

	if s.SplitDescriptor.SplitID == "" || s.DiamondID == "" {
		return fmt.Errorf("invalid split descriptor: requires diamond and split ID to be set")
	}
	if s.SplitDescriptor.State == model.SplitDone {
		// safeguard
		return status.ErrSplitAlreadyDone
	}

	settings := defaultSettings()
	for _, apply := range opts {
		apply(&settings)
	}

	// always ensure a new generation ID every time an upload is attempted
	generationID, err := ksuid.NewRandom()
	if err != nil {
		panic(fmt.Sprintf("cannot generate random ksuid: %v", err))
	}
	s.SplitDescriptor.GenerationID = generationID.String()

	logger := s.l.With(zap.String("SplitID", s.SplitDescriptor.SplitID))

	s.uploadIndexer = newFileIndex(
		s.contextStores,
		fileIndexMeta(GetDiamondStore(s.contextStores)), // file indices live on vmetadata for splits
		fileIndexPather(newUploadSplitIterator( // iterates over any number of index files for this split
			s.RepoID,
			s.DiamondID,
			s.SplitDescriptor,
		)),
		fileIndexLogger(s.l.With(
			zap.String("diamond_id", s.DiamondID),
			zap.String("split_id", s.SplitDescriptor.SplitID),
		)),
	)

	// define the target content addressable store for file blobs
	cafsArchive, err := cafs.New(
		cafs.LeafSize(s.BundleDescriptor.LeafSize),
		cafs.Backend(s.BlobStore()),
		cafs.ConcurrentFlushes(s.concurrentFileUploads/fileUploadsPerFlush),
		cafs.LeafTruncation(s.BundleDescriptor.Version < 1),
		cafs.Logger(s.l),
		cafs.WithMetrics(s.MetricsEnabled()),
	)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			logger.Error("failed split id", zap.Error(err))
			// leave the metadata state unchanged (i.e. in "running" state)
		}
	}()

	filePackedC := make(chan filePacked)
	errorC := make(chan errorHit)
	doneOkC := make(chan struct{})

	// TODO(fred): scalability - getKeys should iterate over keys and set them through a channel like in keys.go
	files, err := s.getKeys("")
	if err != nil {
		return err
	}

	// upload files to content addressable FS store
	// NOTE(fred): we rely on the existing bundle implementation here, not on the new iterator
	go uploadBundleFiles(s.contexter(), s.Bundle, files, cafsArchive, uploadBundleChans{
		filePacked: filePackedC,
		error:      errorC,
		doneOk:     doneOkC,
	})

	if settings.profilingEnabled {
		if err = writeMemProfile(opts...); err != nil {
			return err
		}
	}

	// upload index files for this split
	logger.Debug("uploading index files")
	count, err := s.uploadIndexer.Upload(filePackedC, errorC, doneOkC)
	if err != nil {
		return err
	}

	s.BundleDescriptor.BundleEntriesFileCount = count

	// creates metadata describing the split
	logger.Debug("uploading split descriptor")
	err = s.WithState(model.SplitDone).uploadDescriptor()
	if err != nil {
		return err
	}

	logger.Info("split done")
	return nil
}

// WithState returns the split with the state updated
func (s *Split) WithState(state model.SplitState) *Split {
	if state == s.SplitDescriptor.State && state != model.SplitRunning {
		return s
	}
	s.SplitDescriptor.State = state
	switch state {
	case model.SplitRunning:
		s.SplitDescriptor.StartTime = model.GetBundleTimeStamp()
		s.SplitDescriptor.EndTime = time.Time{}
	case model.SplitDone:
		s.SplitDescriptor.EndTime = model.GetBundleTimeStamp()
	}
	return s
}

func (s *Split) uploadDescriptor() error {
	// sync with bundle fields
	s.SplitDescriptor.SplitEntriesFileCount = s.BundleDescriptor.BundleEntriesFileCount

	buffer, err := yaml.Marshal(s.SplitDescriptor)
	if err != nil {
		return err
	}

	dest := model.GetArchivePathToSplit(
		s.RepoID,
		s.DiamondID,
		s.SplitDescriptor.SplitID,
		s.SplitDescriptor.State,
	)

	return s.writeMetadata(dest, storage.NoOverWrite, buffer)
}

func (s *Split) downloadDescriptor() error {
	// try retrieving descriptor in final state
	src := model.GetArchivePathToFinalSplit(
		s.RepoID,
		s.DiamondID,
		s.SplitDescriptor.SplitID,
	)

	buffer, err := s.readMetadata(src)
	if err != nil {
		if !errors.Is(err, storagestatus.ErrNotExists) {
			return err
		}
		// try retrieving descriptor in initial state
		src = model.GetArchivePathToInitialSplit(
			s.RepoID,
			s.DiamondID,
			s.SplitDescriptor.SplitID,
		)

		buffer, err = s.readMetadata(src)
		if err != nil {
			return err
		}
	}

	var desc model.SplitDescriptor
	err = yaml.Unmarshal(buffer, &desc)
	if err != nil {
		return err
	}
	s.SplitDescriptor = desc
	return nil
}
