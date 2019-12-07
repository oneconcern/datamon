package gcs

import (
	"strings"

	"github.com/oneconcern/datamon/pkg/storage/status"
	"google.golang.org/api/googleapi"
)

func apiErrors(err *googleapi.Error) error {
	switch err.Code {
	case 400:
		if strings.Contains(err.Body, "bucket is not valid") {
			return status.ErrInvalidResource.Wrap(err)
		}
		// TODO(fred): extends qualification of well known errors
		return status.ErrStorageAPI.Wrap(err)
	case 401:
		return status.ErrUnauthorized.Wrap(err)
	case 403:
		return status.ErrForbidden.Wrap(err)
	case 404:
		return status.ErrNotFound.Wrap(err)
	default:
		return status.ErrStorageAPI.Wrap(err)
	}
}

func toSentinelErrors(err error) error {
	// return sentinel errors defined by the status package
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "object doesn't exist") {
		// TODO(fred): should correspond to some google API error
		return status.ErrNotExists.Wrap(err)
	}
	if typedErr, isGoogle := err.(*googleapi.Error); isGoogle {
		return apiErrors(typedErr)
	}
	return err
}
