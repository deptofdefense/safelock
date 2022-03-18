package safelock

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3ObjectLock will create a lock for a specific bucket/key combination
// As an example, if the URI is s3://s3Bucket/s3Key then the lock will be
// a file named s3://s3Bucket/s3Key.lock and the contents will be the lock's
// UUID.
type S3ObjectLock struct {
	*SafeLock

	s3Bucket    string
	s3Key       string
	s3KMSKeyArn string

	svcS3 LockS3Client
}

// NewS3ObjectLock creates a new instance of S3ObjectLock
func NewS3ObjectLock(node uint16, s3bucket, s3key, s3KMSKeyArn string, svcS3 LockS3Client) *S3ObjectLock {
	return &S3ObjectLock{
		SafeLock:    NewSafeLock(node),
		s3Bucket:    s3bucket,
		s3Key:       s3key,
		s3KMSKeyArn: s3KMSKeyArn,
		svcS3:       svcS3,
	}
}

// Lock will lock
func (l *S3ObjectLock) Lock() error {

	// Check first if the lock exists
	// For S3ObjectLock the error is never used and the state can only be locked/unlocked
	lockState, _ := l.GetLockState()
	if lockState == LockStateLocked {
		// conditionally handle deadlock if the lock exists and is owned by a prior session of the same node

		// check the ownership of the lock
		ownedNode, ownedSession, expired, err := l.lockStatus()
		if err != nil {
			return fmt.Errorf("failed to check lock ownership: %v", err)
		}

		// release a deadlocked file lock
		if (ownedNode && !ownedSession) || expired {
			l.mu.Lock()
			// remove file system lock
			_, err := l.svcS3.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
				Bucket: &l.s3Bucket,
				Key:    aws.String(l.GetLockPath()),
			})
			if err != nil {
				l.mu.Unlock()
				return err
			}
			l.mu.Unlock()
		} else {
			return fmt.Errorf("the object at %s is locked", l.GetObjectURI())
		}
	}

	// Lock after getting the lock state
	l.mu.Lock()
	defer l.mu.Unlock()

	// Write object to S3
	body := l.GetLockBody()
	_, errPutObject := l.svcS3.PutObject(context.TODO(), &s3.PutObjectInput{
		ACL:                  types.ObjectCannedACLPrivate,
		Bucket:               &l.s3Bucket,
		Key:                  aws.String(l.GetLockPath()),
		Body:                 bytes.NewReader(body),
		ContentType:          aws.String(http.DetectContentType(body)),
		ServerSideEncryption: types.ServerSideEncryptionAwsKms,
		SSEKMSKeyId:          &l.s3KMSKeyArn,
	})
	if errPutObject != nil {
		return errPutObject
	}
	return nil
}

// Unlock will unlock
func (l *S3ObjectLock) Unlock() error {

	// Check first if the lock exists
	// For S3ObjectLock the error is never used and the state can only be locked/unlocked
	lockState, _ := l.GetLockState()
	if lockState == LockStateUnlocked {
		return fmt.Errorf("the object at %s is not locked", l.GetObjectURI())
	}

	// Validate that the lock belongs to this code
	ownedNode, _, expired, errIsSameLock := l.lockStatus()
	if errIsSameLock != nil {
		return fmt.Errorf("unable to determine if lock is the same lock: %w", errIsSameLock)
	}

	if !ownedNode && !expired {
		return fmt.Errorf("the existing lock is not managed by this process")
	}

	// Lock after verifying the state and lock contents
	l.mu.Lock()
	defer l.mu.Unlock()

	// Remove object from S3
	_, errDeleteObject := l.svcS3.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: &l.s3Bucket,
		Key:    aws.String(l.GetLockPath()),
	})
	if errDeleteObject != nil {
		return errDeleteObject
	}
	return nil
}

// ForceUnlock will unlock despite ownership
func (l *S3ObjectLock) ForceUnlock() error {

	// Check first if the lock exists
	// For S3ObjectLock the error is never used and the state can only be locked/unlocked
	lockState, _ := l.GetLockState()
	if lockState == LockStateUnlocked {
		return fmt.Errorf("the object at %s is not locked", l.GetObjectURI())
	}

	// Lock after verifying the state and lock contents
	l.mu.Lock()
	defer l.mu.Unlock()

	// Remove object from S3
	_, errDeleteObject := l.svcS3.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: &l.s3Bucket,
		Key:    aws.String(l.GetLockPath()),
	})
	if errDeleteObject != nil {
		return errDeleteObject
	}
	return nil
}

// GetS3Bucket will return the s3 bucket for the lock
func (l *S3ObjectLock) GetS3Bucket() string {
	return l.s3Bucket
}

// GetS3Key will return the s3 key for the lock
func (l *S3ObjectLock) GetS3Key() string {
	return l.s3Key
}

// GetS3KMSKeyArn will return the s3 KMS key Arn for the lock
func (l *S3ObjectLock) GetS3KMSKeyArn() string {
	return l.s3KMSKeyArn
}

// GetS3ObjectURI will return the s3 object URI for the file being locked
func (l *S3ObjectLock) GetObjectURI() string {
	uri := url.URL{
		Scheme: "s3",
		Host:   l.GetS3Bucket(),
		Path:   l.GetS3Key(),
	}
	return uri.String()
}

// GetLockURI will return the s3 object URI for the lock object
func (l *S3ObjectLock) GetLockURI() string {
	uri := url.URL{
		Scheme: "s3",
		Host:   l.GetS3Bucket(),
		Path:   l.GetLockPath(),
	}
	return uri.String()
}

// GetLockPath will return the s3 key for the lock object
func (l *S3ObjectLock) GetLockPath() string {
	lockPath := l.GetS3Key() + l.GetLockSuffix()
	return lockPath
}

// GetLockState returns the lock's state
func (l *S3ObjectLock) GetLockState() (LockState, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	_, errHeadObject := l.svcS3.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: &l.s3Bucket,
		Key:    aws.String(l.GetLockPath()),
	})
	if errHeadObject != nil {
		// Throw away the error here because it means the file doesn't exist
		// Assume that API errors also mean state is unlocked
		return LockStateUnlocked, nil
	}
	return LockStateLocked, nil
}

// lockStatus load the current state of the lock
// Returns
// 		nodeOwned 			- bool, whether the lock is owned by this node
//		sessionOwned 		- bool, whether the lock is owned byt his session
//		expired 			- bool, whether the lock has passed its expiration
func (l *S3ObjectLock) lockStatus() (bool, bool, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	getObjectOutput, errGetObject := l.svcS3.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: &l.s3Bucket,
		Key:    aws.String(l.GetLockPath()),
	})
	if errGetObject != nil {
		return false, false, false, errGetObject
	}
	body, errRead := ioutil.ReadAll(getObjectOutput.Body)
	if errRead != nil {
		return false, false, false, errRead
	}

	// split the body into the node and id
	parts := bytes.Split(body, []byte("\n"))
	if len(parts) != 3 {
		return false, false, false, fmt.Errorf("incompatible lock file format")
	}

	// set the default value for expiration to false
	expired := false

	// handle timestamp if there is a configured timeout on the lock
	if l.timeout > 0 {
		// decode the timestamp from the third position in the lock file
		ts := time.Unix(0, int64(binary.LittleEndian.Uint64(parts[2])))

		// update expired with the expiration status
		expired = time.Since(ts) > l.timeout
	}

	return bytes.Equal(parts[0], l.GetNodeBytes()), bytes.Equal(parts[1], l.GetIDBytes()), expired, nil
}

// WaitForLock waits until an object is no longer locked or cancels based on a timeout
func (l *S3ObjectLock) WaitForLock(timeout time.Duration) error {
	// Do not lock/unlock the struct here or it will block getting the lock state

	// create variable to hold the context
	var ctx context.Context
	var cancel context.CancelFunc

	// conditionally configure the context with a timeout
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}

	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("unable to obtain lock after %s: %w", l.GetTimeout(), ctx.Err())
		default:
			// For S3 there will never be an error when getting lock state
			lockState, _ := l.GetLockState()
			switch lockState {
			case LockStateUnlocked:
				return nil
			case LockStateUnknown, LockStateLocked:
				// Add jitter to the sleep of 1 second
				r := rand.Intn(100)
				time.Sleep(1*time.Second + time.Duration(r)*time.Millisecond)
			}
		}
	}
}
