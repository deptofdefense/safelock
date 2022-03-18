package safelock

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/gage-technologies/safelock/internal/mocks"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestS3ObjectLock(t *testing.T) {

	svcS3 := mocks.MockS3Client{
		PutObjectOutput: &s3.PutObjectOutput{},
	}

	bucket := "bucket"
	key := "key"
	kmsKeyArn := "kmsKeyArn"
	l := NewS3ObjectLock(0, bucket, key, kmsKeyArn, &svcS3)

	errLock := l.Lock()
	assert.NoError(t, errLock)

	// Verify the contents of the lock file
	data, errReadAll := ioutil.ReadAll(svcS3.PutObjectInput.Body)
	assert.NoError(t, errReadAll)
	assert.True(t, bytes.Equal(
		bytes.Join(bytes.Split(l.GetLockBody(), []byte("\n"))[:2], []byte("\n")),
		bytes.Join(bytes.Split(data, []byte("\n"))[:2], []byte("\n")),
	))

	// Indicate that the file exists
	svcS3.HeadObjectOutput = &s3.HeadObjectOutput{}
	// Set the contents of the file
	body := ioutil.NopCloser(bytes.NewReader(l.GetLockBody()))
	svcS3.GetObjectOutput = &s3.GetObjectOutput{
		Body: body,
	}
	// Ensure deletion can occur
	svcS3.DeleteObjectOutput = &s3.DeleteObjectOutput{}

	errUnlock := l.Unlock()
	assert.NoError(t, errUnlock)

	nodeCreation := time.Unix(0, int64(l.GetID()))
	assert.True(t, time.Since(nodeCreation) < time.Second)

	// Remove info about the lock
	svcS3.HeadObjectOutput = nil

	lockState, errGetLockState := l.GetLockState()
	assert.NoError(t, errGetLockState)
	assert.Equal(t, LockStateUnlocked, lockState)

	// Object Info
	assert.Equal(t, bucket, l.GetS3Bucket())
	assert.Equal(t, key, l.GetS3Key())
	assert.Equal(t, kmsKeyArn, l.GetS3KMSKeyArn())

	// Get URI Info
	assert.Equal(t, "s3://bucket/key", l.GetObjectURI())
	assert.Equal(t, "s3://bucket/key.lock", l.GetLockURI())

	// Wait
	errWaitForLock := l.WaitForLock(DefaultTimeout)
	assert.NoError(t, errWaitForLock)
}

func TestS3ObjectLockLockErrors(t *testing.T) {

	// Pretend that the lock is locked
	svcS3 := mocks.MockS3Client{
		HeadObjectOutput: &s3.HeadObjectOutput{},
	}

	bucket := "bucket"
	key := "key"
	kmsKeyArn := "kmsKeyArn"
	l := NewS3ObjectLock(0, bucket, key, kmsKeyArn, &svcS3)

	errLock := l.Lock()
	assert.Error(t, errLock)

	// Pretend that it is unlocked but can't put lock
	svcS3.HeadObjectOutput = nil
	svcS3.PutObjectOutput = nil

	errLock = l.Lock()
	assert.Error(t, errLock)
}

func TestS3ObjectLockUnlockErrors(t *testing.T) {

	// Pretend that the lock is unlocked
	svcS3 := mocks.MockS3Client{}

	bucket := "bucket"
	key := "key"
	kmsKeyArn := "kmsKeyArn"
	l := NewS3ObjectLock(0, bucket, key, kmsKeyArn, &svcS3)

	errUnlock := l.Unlock()
	assert.Error(t, errUnlock)

	// Indicate that the file exists
	svcS3.HeadObjectOutput = &s3.HeadObjectOutput{}
	// Without setting the GetObjectOutput the file contents will error
	assert.Nil(t, svcS3.GetObjectOutput)

	errUnlock = l.Unlock()
	assert.Error(t, errUnlock)

	// Try again but make the body unreadable
	body := io.NopCloser(iotest.ErrReader(errors.New("error get object reader")))
	svcS3.GetObjectOutput = &s3.GetObjectOutput{
		Body: body,
	}

	errUnlock = l.Unlock()
	assert.Error(t, errUnlock)

	// Set the contents of the file to a UUID that is not the same as the lock
	body = ioutil.NopCloser(strings.NewReader(uuid.New().String()))
	svcS3.GetObjectOutput = &s3.GetObjectOutput{
		Body: body,
	}

	errUnlock = l.Unlock()
	assert.Error(t, errUnlock)

	// Ensure that the object can't be deleted
	body = ioutil.NopCloser(bytes.NewReader(l.GetLockBody()))
	svcS3.GetObjectOutput = &s3.GetObjectOutput{
		Body: body,
	}
	// Don't set PutObjectOutput to ensure this fails
	assert.Nil(t, svcS3.PutObjectOutput)

	errUnlock = l.Unlock()
	assert.Error(t, errUnlock)
}

func TestS3ObjectLockWait(t *testing.T) {

	svcS3 := mocks.MockS3Client{
		PutObjectOutput: &s3.PutObjectOutput{},
	}

	bucket := "bucket"
	key := "key"
	kmsKeyArn := "kmsKeyArn"
	l := NewS3ObjectLock(0, bucket, key, kmsKeyArn, &svcS3)

	errLock := l.Lock()
	assert.NoError(t, errLock)

	// Indicate that the file exists
	svcS3.HeadObjectOutput = &s3.HeadObjectOutput{}
	// Set the contents of the file
	body := ioutil.NopCloser(bytes.NewReader(l.GetLockBody()))
	svcS3.GetObjectOutput = &s3.GetObjectOutput{
		Body: body,
	}
	// Ensure deletion can occur
	svcS3.DeleteObjectOutput = &s3.DeleteObjectOutput{}

	// Spin this off in a goroutine so that we can manipulate the lock
	go func() {
		errWaitForLock := l.WaitForLock(DefaultTimeout)
		assert.NoError(t, errWaitForLock)
	}()

	errUnlock := l.Unlock()
	assert.NoError(t, errUnlock)
}

func TestS3ObjectLockWaitError(t *testing.T) {

	svcS3 := mocks.MockS3Client{
		PutObjectOutput:  &s3.PutObjectOutput{},
		HeadObjectOutput: &s3.HeadObjectOutput{},
	}

	bucket := "bucket"
	key := "key"
	kmsKeyArn := "kmsKeyArn"
	l := NewS3ObjectLock(0, bucket, key, kmsKeyArn, &svcS3)

	// Timeout as fast as possible
	l.SetTimeout(1 * time.Microsecond)

	errWaitForLock := l.WaitForLock(1 * time.Microsecond)
	assert.Error(t, errWaitForLock)
	assert.Equal(t, "unable to obtain lock after 1µs: context deadline exceeded", errWaitForLock.Error())
}
