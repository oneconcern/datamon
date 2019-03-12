package sthree

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/oneconcern/datamon/pkg/storage"
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
		if rerr, ok := err.(awserr.RequestFailure); ok && rerr.StatusCode() == 404 {
			return false, nil
		}
		return false, fmt.Errorf("failed to get head request: %v", err)
	}
	return true, nil
}

func (s *s3FS) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := s.s3.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return nil, err
	}
	return obj.Body, nil
}

func (s *s3FS) Put(ctx context.Context, key string, rdr io.Reader, _ bool) error {
	_, err := s.uploader.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   rdr,
	})
	return err
}

func (s *s3FS) Delete(ctx context.Context, key string) error {
	_, err := s.s3.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
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
		return nil, err
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
		return nil, "", err
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
	return del.Delete(ctx, s3manager.NewDeleteListIterator(s.s3, params))
}

func (s *s3FS) String() string {
	return "s3@" + s.bucket
}

func (s *s3FS) GetAt(ctx context.Context, objectName string) (io.ReaderAt, error) {
	return nil, errors.New("unimplemented")
}
