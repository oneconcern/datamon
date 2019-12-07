package sthree

import (
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/oneconcern/datamon/pkg/errors"
	"github.com/oneconcern/datamon/pkg/storage/status"
)

func filterErrNotExists(err error) error {
	if errors.Is(err, status.ErrNotExists) || errors.Is(err, status.ErrNotFound) {
		return nil
	}
	return err
}

func apiErrors(err awserr.RequestFailure) error {
	// handle S3 API errors
	// https://docs.aws.amazon.com/sdk-for-go/api/aws/awserr/#RequestFailure
	switch err.StatusCode() {
	case 400:
		if err.Code() == "InvalidBucketName" {
			return status.ErrInvalidResource.Wrap(err)
		}
		return status.ErrStorageAPI.Wrap(err)
	case 401:
		return status.ErrUnauthorized.Wrap(err)
	case 403:
		return status.ErrForbidden.Wrap(err)
	case 404:
		switch err.Code() {
		case "NoSuchKey", "NoSuchBucket", "NotFound": // NotFound is a code produced by miniio and not an official AWS code
			// storable objects
			return status.ErrNotExists.Wrap(err)
		default:
			// generic S3 object
			return status.ErrNotFound.Wrap(err)
		}
	default:
		return status.ErrStorageAPI.Wrap(err)
	}
}

func toSentinelErrors(err error) error {
	// return sentinel errors defined by the status package
	// see: https://docs.aws.amazon.com/AmazonS3/latest/API/ErrorResponses.html#ErrorCodeList
	if err == nil {
		return nil
	}
	if awsErr, isAWS := err.(awserr.RequestFailure); isAWS {
		return apiErrors(awsErr)
	}
	return err
}
