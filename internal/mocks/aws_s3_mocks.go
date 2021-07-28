package mocks

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// MockS3Client is a mock AWS S3 Client
type MockS3Client struct {
	S3uri *string

	// Output Data
	DeleteObjectOutput *s3.DeleteObjectOutput
	GetObjectOutput    *s3.GetObjectOutput
	HeadObjectOutput   *s3.HeadObjectOutput
	PutObjectInput     *s3.PutObjectInput
	PutObjectOutput    *s3.PutObjectOutput
}

func (s *MockS3Client) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	if s.DeleteObjectOutput == nil {
		return nil, errors.New("delete object error")
	}
	return s.DeleteObjectOutput, nil
}

func (s *MockS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if s.GetObjectOutput == nil {
		return nil, errors.New("error from get object")
	}
	return s.GetObjectOutput, nil
}

func (s *MockS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	if s.HeadObjectOutput == nil {
		return nil, errors.New("error from head object")
	}
	return s.HeadObjectOutput, nil
}

func (s *MockS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	s.PutObjectInput = params
	if s.PutObjectOutput == nil {
		return nil, errors.New("error from put object")
	}
	return s.PutObjectOutput, nil
}
