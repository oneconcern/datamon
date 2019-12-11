// Copyright © 2018 One Concern

// Package status declares error constants returned by
// implementations of the Store interface.
//
// NOTE: such constants are located in a separate package to avoid
// creating undue cyclical dependencies between pkg/store and one
// of its implementions.
package status

import "github.com/oneconcern/datamon/pkg/errors"

var (
	// Sentinel errors returned by implementations of the interface defined by storage

	// ErrNotExists indicates that the fetched object does not exist on storage
	ErrNotExists = errors.New("object doesn't exist")

	// ErrNotFound indicates that the backend API call did not find the target resource
	ErrNotFound = errors.New("not found")

	// ErrUnauthorized indicates that you don't provided correct credentials to the API
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden indicates that the backend API forbids access to the target resource
	ErrForbidden = errors.New("forbidden")

	// ErrNotSupported indicates that the backend API does not support this call
	ErrNotSupported = errors.New("not supported")

	// ErrExists indicates that the resource already exists and cannot be overridden
	ErrExists = errors.New("exists already")

	// ErrObjectTooBig indicates that the object is too big and cannot be handled by datamon
	ErrObjectTooBig = errors.New("object too big to be read into memory")

	// ErrInvalidResource indicates that the storage resource has an invalid name
	ErrInvalidResource = errors.New("invalid storage resource name")

	// ErrStorageAPI indicates any other storage AI error
	ErrStorageAPI = errors.New("storage API error")

	// ErrNotImplemented tells that this feature has not been implemented yet
	ErrNotImplemented = errors.New("not implemented")
)
