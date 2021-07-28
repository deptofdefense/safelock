package safelock

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
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
func NewFileLock(filename string, fs afero.Fs) *FileLock {
	return &FileLock{
		SafeLock: NewSafeLock(),
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
		return fmt.Errorf("the object at %s is locked", l.GetFilename())
	}

	// Lock after getting the lock state
	l.mu.Lock()
	defer l.mu.Unlock()

	// Write object to S3
	body := []byte(l.GetID())
	aFile, errOpen := l.fs.OpenFile(l.GetLockFilename(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if errOpen != nil {
		return fmt.Errorf("unable to open %q: %w", l.GetLockFilename(), errOpen)
	}
	defer aFile.Close()

	_, errWrite := aFile.Write(body)
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
	sameLock, errIsSameLock := l.isSameLock()
	if errIsSameLock != nil {
		return fmt.Errorf("unable to determine if lock is the same lock: %w", errIsSameLock)
	}

	if !sameLock {
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

// isSameLock will determine if the current lock belongs to this lock
func (l *FileLock) isSameLock() (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	aFile, errOpen := l.fs.Open(l.GetLockFilename())
	if errOpen != nil {
		return false, fmt.Errorf("unable to open %q: %w", l.GetLockFilename(), errOpen)
	}
	defer aFile.Close()

	body, errRead := ioutil.ReadAll(aFile)
	if errRead != nil {
		return false, errRead
	}

	// If the contents of the lock and the ID are the same then it is locked
	// Check that the strings are the same using a case-insensitive test
	if strings.EqualFold(string(body), l.GetID()) {
		return true, nil
	}
	return false, nil

}

// WaitForLock waits until an object is no longer locked or cancels based on a timeout
func (l *FileLock) WaitForLock() error {
	// Do not lock/unlock the struct here or it will block getting the lock state

	// Pass a context with a timeout to tell a blocking function that it
	// should abandon its work after the timeout elapses.
	ctx, cancel := context.WithTimeout(context.Background(), l.GetTimeout())
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
