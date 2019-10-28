// Package status declares error constants returned by the various
// implementations of the Authable interface.
//
// NOTE: such constants are located in a separate package to avoid
// creating undue cyclical dependencies between pkg/store and one
// of its implementions.
package status

import "github.com/oneconcern/datamon/pkg/errors"

var (
	// Sentinel errors returned by implementations of interfaces defined by auth

	// ErrInvalidCredentials indicates that the credentials passed are invalid
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrUserinfo indicates that user information could not be retrieved
	ErrUserinfo = errors.New("could not retrieve userinfo")

	// ErrAuthService indicates that we coud not instantiate an authentication service
	ErrAuthService = errors.New("could not create oauth service")

	// ErrEmailScope indicates that the email scope is missing from the credentials
	ErrEmailScope = errors.New("email scope is mandatory to run datamon")
)
