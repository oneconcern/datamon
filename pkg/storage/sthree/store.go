// Package sthree implements datamon Store for AWS S3
package sthree

import (
	"context"
	"io"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/oneconcern/datamon/pkg/storage/status"
)

const PageSize = 1000

type Option func(*s3FS)

func Bucket(bucket string) Option {
	return func(fs *s3FS) {
		fs.bucket = bucket
	}
}

func AWSConfig(cfg *aws.Config) Option {
	return func(fs *s3FS) {
		fs.awsConfig = cfg
	}
}

func New(option Option, options ...Option) storage.Store {
	fs := new(s3FS)
	option(fs)
	for _, apply := range options {
		apply(fs)
	}

	fs.s3 = s3.New(session.Must(session.NewSession(fs.awsConfig)))
	fs.uploader = s3manager.NewUploaderWithClient(fs.s3)
	fs.downloader = s3manager.NewDownloaderWithClient(fs.s3)
	return fs
}

type s3FS struct {
	bucket     string
	awsConfig  *aws.Config
	s3         *s3.S3
	uploader   *s3manager.Uploader
	downloader *s3manager.Downloader
}

func (s *s3FS) Has(ctx context.Context, key string) (bool, error) {
	_, err := s.s3.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return false, filterErrNotExists(toSentinelErrors(err))
	}
	return true, nil
}

func (s *s3FS) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := s.s3.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return nil, toSentinelErrors(err)
	}
	return obj.Body, nil
}

func (s *s3FS) Put(ctx context.Context, key string, rdr io.Reader, _ bool) error {
	_, err := s.uploader.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   rdr,
	})
	return toSentinelErrors(err)
}

func (s *s3FS) Delete(ctx context.Context, key string) error {
	_, err := s.s3.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return toSentinelErrors(err)
}

func (s *s3FS) Keys(ctx context.Context) ([]string, error) {
	var keys []string
	eachPage := func(page *s3.ListObjectsOutput, more bool) bool {
		for _, obj := range page.Contents {
			key := aws.StringValue(obj.Key)
			if key != "" {
				keys = append(keys, key)
			}
		}
		return more
	}
	params := &s3.ListObjectsInput{Bucket: aws.String(s.bucket)}

	err := s.s3.ListObjectsPagesWithContext(ctx, params, eachPage)
	if err != nil {
		return nil, toSentinelErrors(err)
	}
	return keys, nil
}

func (s *s3FS) KeysPrefix(ctx context.Context, token, prefix, delimiter string, count int) ([]string, string, error) {
	var keys []string
	var isTruncated bool

	eachPage := func(page *s3.ListObjectsOutput, more bool) bool {
		isTruncated = aws.BoolValue(page.IsTruncated)

		for _, obj := range page.Contents {
			key := aws.StringValue(obj.Key)
			if key != "" {
				keys = append(keys, key)
			}
		}
		return more
	}

	params := &s3.ListObjectsInput{
		Bucket:    aws.String(s.bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String(delimiter),
		MaxKeys:   aws.Int64(PageSize),
		Marker:    aws.String(token),
	}

	err := s.s3.ListObjectsPagesWithContext(ctx, params, eachPage)
	if err != nil {
		return nil, "", toSentinelErrors(err)
	}

	log.Printf("Truncated %v ", isTruncated)
	if isTruncated {
		token = keys[len(keys)-1]
	}
	return keys, token, nil
}

func (s *s3FS) Clear(ctx context.Context) error {
	params := &s3.ListObjectsInput{Bucket: aws.String(s.bucket)}
	del := s3manager.NewBatchDeleteWithClient(s.s3)
	return toSentinelErrors(del.Delete(ctx, s3manager.NewDeleteListIterator(s.s3, params)))
}

func (s *s3FS) String() string {
	return "s3@" + s.bucket
}

func (s *s3FS) GetAt(ctx context.Context, objectName string) (io.ReaderAt, error) {
	return nil, status.ErrNotImplemented
}

func (s *s3FS) GetAttr(ctx context.Context, objectName string) (storage.Attributes, error) {
	return storage.Attributes{}, status.ErrNotImplemented
}

func (s *s3FS) Touch(ctx context.Context, objectName string) error {
	return status.ErrNotImplemented
}
