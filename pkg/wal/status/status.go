// Package status declares error constants returned by
// the wak package.
package status

import (
	"github.com/oneconcern/datamon/pkg/errors"
)

var (
	// ErrTokenGenUpdate signals that we could update the WAL token generator
	ErrTokenGenUpdate = errors.New("failed to update the token generator")

	// ErrTokenGenerate signals that we could update a new WAL token
	ErrTokenGenerate = errors.New("failed to generate token")

	// ErrKSUID indicates that we failed to generate a new ksuid.
	// An error here is telling of an issue with the random generator.
	ErrKSUID = errors.New("failed tp generate ksuid")

	// ErrAddWALEntry indicates a failure when adding the WAL token to the entry list
	ErrAddWALEntry = errors.New("failed to add wal token entry")

	// ErrTokenAttributes indicates a failure when extracting the WAL token attributes
	ErrTokenAttributes = errors.New("failed to get token generator attributes")

	// ErrMaxCount indicates a wrong max count parameter (should be strictly positive)
	ErrMaxCount = errors.New("max count needs to be greater than 0")

	// ErrFirstToken indicates a failure when computing the first token
	ErrFirstToken = errors.New("failed to calculate first token")

	// ErrGetTokens indicates a failure when retrieving tokens
	ErrGetTokens = errors.New("failed to get tokens")
)
