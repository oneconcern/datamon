// Package wal provides a write-ahead log.
//
// The WAL keeps track of all changes to a repo, that is,
// which contributor did change what and when.
package wal

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/oneconcern/datamon/pkg/dlogger"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/wal/status"

	iradix "github.com/hashicorp/go-immutable-radix"
	"go.uber.org/zap"

	"github.com/segmentio/ksuid"

	"github.com/oneconcern/datamon/pkg/storage"
)

const (
	maxEntriesPerList = 1000
	maxConcurrency    = 1024
)

// WAL describes a write-ahead log
type WAL struct {
	mutableStore       storage.Store // Location for updating token generator object
	tokenGeneratorPath string        // Path to the token generator object in the mutable store
	walStore           storage.Store // Append only store where WAL entries are written to
	maxConcurrency     int           // Max concurrency when reading
	connectionControl  chan struct{} // How many max concurrent requests to send
	l                  *zap.Logger   // Logging
}

// Options to the write-ahead log
type Option func(w *WAL)

func MaxConcurrency(c int) Option {
	return func(w *WAL) {
		w.maxConcurrency = c
	}
}

func TokenGeneratorPath(path string) Option {
	return func(w *WAL) {
		w.tokenGeneratorPath = path
	}
}

// Logger sets a logger for this WAL
func Logger(logger *zap.Logger) Option {
	return func(w *WAL) {
		if logger != nil {
			w.l = logger
		}
	}
}

func defaultWAL() *WAL {
	logger, _ := dlogger.GetLogger(dlogger.LogLevelInfo)
	return &WAL{
		maxConcurrency:     maxConcurrency,
		l:                  logger,
		tokenGeneratorPath: model.TokenGeneratorPath,
	}
}

// New builds a new write-ahead log on some mutable store, with log entries stored at the walStore
func New(mutableStore storage.Store, walStore storage.Store, options ...Option) *WAL {
	wal := defaultWAL()
	for _, option := range options {
		option(wal)
	}
	wal.mutableStore = mutableStore
	wal.walStore = walStore
	wal.connectionControl = make(chan struct{}, maxConcurrency)
	// Check if token generator object exists
	_ = mutableStore.Put(context.Background(), wal.tokenGeneratorPath, strings.NewReader(""), storage.OverWrite)
	return wal
}

// Gets a token such that tokens are K-sortable.
func (w *WAL) getToken(ctx context.Context) (string, error) {
	err := w.updateTokenTimestamp(ctx)
	if err != nil {
		return "", status.ErrTokenGenUpdate.Wrap(err)
	}

	// Read the updateTime from the tokenGenerator object
	attr, err := w.mutableStore.GetAttr(ctx, w.tokenGeneratorPath)
	if err != nil {
		return "", status.ErrTokenAttributes.WrapWithLog(w.l, err, zap.String("token generator", w.tokenGeneratorPath))
	}

	// Use the updateTime from the token generator (gain independence from local wall clock)
	k, err := ksuid.NewRandomWithTime(attr.Updated)
	if err != nil {
		return "", status.ErrKSUID.Wrap(err)
	}

	w.l.Debug("generated token", zap.String("token", k.String()), zap.Time("updateTime", attr.Updated))

	return k.String(), nil
}

// Adds a WAL entry to WAL
func (w *WAL) Add(ctx context.Context, p string) (string, error) {
	e := model.Entry{
		Payload: p,
	}
	var err error
	e.Token, err = w.getToken(ctx)
	if err != nil {
		return "", status.ErrTokenGenerate.WrapWithLog(w.l, err, zap.String("payload", p))
	}

	err = w.walStore.Put(ctx, e.Token, strings.NewReader(e.Payload), storage.NoOverWrite) // Should be a new entry
	if err != nil {
		return "", status.ErrAddWALEntry.WrapWithLog(w.l, err, zap.String("token", e.Token))
	}
	w.l.Debug("Write wal entry", zap.String("token", e.Token))
	return e.Token, err
}

func (w *WAL) GetExpirationDuration() time.Duration {
	return 10 * time.Minute
}

func (w *WAL) updateTokenTimestamp(ctx context.Context) error {
	return w.mutableStore.Touch(ctx, w.tokenGeneratorPath)
}

// Reads the WAL starting from the tokens passed in. If startFrom is empty it will entry from beginning.
// Repeated reads can include duplicate tokens. No tokens are missed
// Returns false if the list has more entries that can be listed.
// Use the last Entry of the previous call to paginate to the next set of keys.
func (w *WAL) ListTokens(ctx context.Context, fromToken string, max int) (tokens []string, next string, err error) {
	if max <= 0 {
		return nil, "", status.ErrMaxCount.WrapWithLog(w.l, nil, zap.Int("max count", max), zap.String("token", fromToken))
	}
	k, err := ksuid.Parse(fromToken)
	if err != nil {
		return nil, "", err
	}
	if max > maxEntriesPerList {
		max = maxEntriesPerList
	}

	// Go back in time to include the keys that might have been written before the token was generated.
	b := make([]byte, 16)
	ksuidOld, err := ksuid.FromParts(k.Time().Add(-w.GetExpirationDuration()*2), b)
	if err != nil {
		return nil, "", status.ErrFirstToken.WrapWithLog(w.l, err, zap.String("fromToken", fromToken))
	}
	tokens, next, err = w.walStore.KeysPrefix(ctx, ksuidOld.String(), "", "", max)
	if err != nil {
		return nil, "", status.ErrGetTokens.WrapWithLog(w.l, err,
			zap.String("backDatedToken", ksuidOld.String()),
			zap.String("token", fromToken))
	}
	return tokens, next, err
}

func (w *WAL) getConnection() {
	w.connectionControl <- struct{}{}
}

func (w *WAL) releaseConnection() {
	<-w.connectionControl
}

type walChannels struct {
	tokens  chan []string
	entry   chan *model.Entry
	entries chan []model.Entry
	count   chan int
	oops    chan error
	done    chan struct{}
}

func newWalChannels() *walChannels {
	token := make(chan []string)
	entry := make(chan *model.Entry)
	entries := make(chan []model.Entry)
	count := make(chan int)
	done := make(chan struct{})
	return &walChannels{
		tokens:  token,
		entry:   entry,
		entries: entries,
		count:   count,
		done:    done,
	}
}

func (w *WAL) ListEntries(ctx context.Context, fromToken string, max int) ([]model.Entry, string, error) {
	if max <= 0 {
		w.l.Warn("received 0 length max",
			zap.String("fromToken", fromToken),
			zap.Int("max", max))
		return nil, "", fmt.Errorf("max count needs to be greater than 0 : %d, fromToken:%s", max, fromToken)
	}
	walChannels := newWalChannels()
	count := 0

	// ListEntries spawns a go routine for a batch of entries that spawns parallel read. The responses for the parallel reads
	// are read by one routine that collects all responses and returns them to the main thread.

	// Start routine which collects all parallel responses.
	go w.collectParallelResponses(ctx, walChannels)
	// Start routine that will issue parallel reads.
	go w.issueParallelReads(ctx, walChannels)

	// Read tokens and send to routine to entry in parallel while fetching new tokens.
	tokens, next, err := w.ListTokens(ctx, fromToken, max-count)

	if err != nil {
		return nil, "", fmt.Errorf("wal failed to ListEntries: " + err.Error())
	}
	count = len(tokens)

	if count > 0 {
		// account for case list is not complete but the response does not include any token.
		walChannels.tokens <- tokens
	} else {
		return nil, next, err
	}

	close(walChannels.tokens) // No more tokens to report.

	walChannels.count <- count // Send total count for reading responses.

	for {
		select {
		case entries := <-walChannels.entries:
			if len(entries) != count {
				panic(fmt.Sprintf("received count different than the list of tokens entries: %d count: %d", len(entries), count))
			}
			return entries, next, nil
		case err := <-walChannels.oops:
			return nil, "", err
		}
	}
}

func (w *WAL) issueParallelReads(ctx context.Context, channels *walChannels) {
	count := 0
	for tokens := range channels.tokens {
		for _, t := range tokens {
			w.getConnection() // concurrency control
			count++
			go w.read(ctx, t, channels)
		}
	}
}

func (w *WAL) read(ctx context.Context, token string, channels *walChannels) {
	defer w.releaseConnection() // concurrency control
	r, err := w.walStore.Get(ctx, token)
	w.l.Debug("Read token", zap.String("token", token))
	defer r.Close()
	if err != nil {
		channels.oops <- err
		return
	}
	b := make([]byte, 1024)
	for {
		l, e := r.Read(b)
		if e == io.EOF {
			b = b[:l]
			break
		}
	}
	entry, err := model.UnmarshalWAL(b)
	if err != nil {
		channels.oops <- fmt.Errorf("token: %s, err: %s", token, err)
		return
	}
	channels.entry <- entry
}

func (w *WAL) collectParallelResponses(ctx context.Context, channels *walChannels) {
	entriesMap := iradix.New().Txn()
	count := 0
	total := 0
	var err error

	finalize := func() {
		if err != nil {
			// Report an error occurred.
			channels.oops <- err
			return
		}
		var entries []model.Entry
		iterator := entriesMap.Root().Iterator()
		for {
			_, e, ok := iterator.Next()
			if !ok {
				// send the final entries
				channels.entries <- entries
				return
			}
			entry := e.(*model.Entry)
			entries = append(entries, *entry)
		}
	}

	for {
		// Wait for all reads to return.
		select {
		case e := <-channels.entry:
			w.l.Debug("Received entry", zap.String("token", e.Token))
			_, updated := entriesMap.Insert([]byte(e.Token), e)
			if updated {
				panic(fmt.Sprintf("Received more than one response for token: %s string: %s", e.Token, e.Payload))
			}
			count++
			if count == total {
				finalize()
				return
			}
		case total = <-channels.count:
			w.l.Info("waiting to read", zap.Int("total", total))
			if count == total {
				finalize()
				return
			}
		case err = <-channels.oops:
			// Log all errors
			w.l.Error("failed to read token", zap.Error(err))
			count++
			if count == total {
				finalize()
				return
			}
		}
	}
}
