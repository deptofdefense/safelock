package safelock

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"time"

	"github.com/spf13/afero"
)

// FileLock will create a lock for a specific file
// As an example, if the URI is file:///filename.txt then the lock will be
// a file named file:///filename.txt.lock and the contents will be the lock's
// UUID.
type FileLock struct {
	*SafeLock

	filename string
	fs       afero.Fs
}

// NewFileLock creates a new instance of FileLock
func NewFileLock(node uint16, filename string, fs afero.Fs) *FileLock {
	return &FileLock{
		SafeLock: NewSafeLock(node),
		filename: filename,
		fs:       fs,
	}
}

// Lock will lock
func (l *FileLock) Lock() error {

	// Check first if the lock exists
	// For FileLock the error is never used and the state can only be locked/unlocked
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
			err := l.fs.Remove(l.GetLockFilename())
			if err != nil {
				l.mu.Unlock()
				return err
			}
			l.mu.Unlock()
		} else {
			return fmt.Errorf("the object at %s is locked", l.GetFilename())
		}
	}

	// Lock after getting the lock state
	l.mu.Lock()
	defer l.mu.Unlock()

	// Write object to S3
	aFile, errOpen := l.fs.OpenFile(l.GetLockFilename(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if errOpen != nil {
		return fmt.Errorf("unable to open %q: %w", l.GetLockFilename(), errOpen)
	}
	defer aFile.Close()

	_, errWrite := aFile.Write(l.GetLockBody())
	if errWrite != nil {
		return fmt.Errorf("unable to write data to %q: %w", l.GetLockFilename(), errWrite)
	}
	return nil
}

// Unlock will unlock
func (l *FileLock) Unlock() error {

	// Check first if the lock exists
	// For FileLock the error is never used and the state can only be locked/unlocked
	lockState, _ := l.GetLockState()
	if lockState == LockStateUnlocked {
		return fmt.Errorf("the object at %s is not locked", l.GetFilename())
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

	// Remove object from filesystem
	errRemove := l.fs.Remove(l.GetLockFilename())
	if errRemove != nil {
		return errRemove
	}
	return nil
}

// ForceUnlock will unlock despite ownership
func (l *FileLock) ForceUnlock() error {

	// Check first if the lock exists
	// For FileLock the error is never used and the state can only be locked/unlocked
	lockState, _ := l.GetLockState()
	if lockState == LockStateUnlocked {
		return fmt.Errorf("the object at %s is not locked", l.GetFilename())
	}

	// Lock after verifying the state and lock contents
	l.mu.Lock()
	defer l.mu.Unlock()

	// Remove object from filesystem
	errRemove := l.fs.Remove(l.GetLockFilename())
	if errRemove != nil {
		return errRemove
	}
	return nil
}

// GetFilename will return the filename for the lock
func (l *FileLock) GetFilename() string {
	return l.filename
}

// GetLockFilename will return the filename for the lock object
func (l *FileLock) GetLockFilename() string {
	lockPath := l.GetFilename() + l.GetLockSuffix()
	return lockPath
}

// GetLockState returns the lock's state
func (l *FileLock) GetLockState() (LockState, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	_, errStat := l.fs.Stat(l.GetLockFilename())
	if errStat != nil {
		if os.IsNotExist(errStat) {
			// Throw away the error here because it means the file doesn't exist
			return LockStateUnlocked, nil
		} else {
			return LockStateUnknown, errStat
		}
	}
	return LockStateLocked, nil
}

// lockStatus load the current state of the lock
// Returns
// 		nodeOwned 			- bool, whether the lock is owned by this node
//		sessionOwned 		- bool, whether the lock is owned byt his session
//		expired 			- bool, whether the lock has passed its expiration
func (l *FileLock) lockStatus() (bool, bool, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	aFile, errOpen := l.fs.Open(l.GetLockFilename())
	if errOpen != nil {
		return false, false, false, fmt.Errorf("unable to open %q: %w", l.GetLockFilename(), errOpen)
	}
	defer aFile.Close()

	body, errRead := ioutil.ReadAll(aFile)
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
func (l *FileLock) WaitForLock(timeout time.Duration) error {
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
			lockState, errGetLockState := l.GetLockState()
			if errGetLockState != nil {
				return errGetLockState
			}
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
