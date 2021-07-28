package safelock

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/spf13/afero"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestFileLock(t *testing.T) {

	fs := afero.NewMemMapFs()
	filename := "file.txt"
	lockfile := filename + DefaultSuffix

	l := NewFileLock(filename, fs)

	errLock := l.Lock()
	assert.NoError(t, errLock)

	// Verify the contents of the lock file
	aFile, errOpen := fs.Open(lockfile)
	assert.NoError(t, errOpen)
	defer aFile.Close()

	data, errReadAll := ioutil.ReadAll(aFile)
	assert.NoError(t, errReadAll)
	assert.Equal(t, l.GetID(), string(data))

	errUnlock := l.Unlock()
	assert.NoError(t, errUnlock)

	_, errParse := uuid.Parse(l.GetID())
	assert.NoError(t, errParse)

	lockState, errGetLockState := l.GetLockState()
	assert.NoError(t, errGetLockState)
	assert.Equal(t, LockStateUnlocked, lockState)

	// File Info
	assert.Equal(t, filename, l.GetFilename())

	// Wait
	errWaitForLock := l.WaitForLock()
	assert.NoError(t, errWaitForLock)
}

func TestFileLockLockErrors(t *testing.T) {

	// Pretend that the lock is locked
	fs := afero.NewMemMapFs()
	filename := "file.txt"
	lockfile := filename + DefaultSuffix

	aFile, errOpen := fs.OpenFile(lockfile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	assert.NoError(t, errOpen)
	defer aFile.Close()

	_, errWrite := aFile.Write([]byte(uuid.New().String()))
	assert.NoError(t, errWrite)

	l := NewFileLock(filename, fs)

	errLock := l.Lock()
	assert.Error(t, errLock)
}

func TestFileLockUnlockErrors(t *testing.T) {

	// Pretend that the lock is unlocked
	fs := afero.NewMemMapFs()
	filename := "file.txt"
	lockfile := filename + DefaultSuffix

	l := NewFileLock(filename, fs)

	errUnlock := l.Unlock()
	assert.Error(t, errUnlock)

	// Indicate that the file exists
	aFile, errOpen := fs.OpenFile(lockfile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	assert.NoError(t, errOpen)
	defer aFile.Close()

	// Set the contents of the file to a UUID that is not the same as the lock
	_, errWrite := aFile.Write([]byte(uuid.New().String()))
	assert.NoError(t, errWrite)

	errUnlock = l.Unlock()
	assert.Error(t, errUnlock)
}

func TestFileLockWait(t *testing.T) {

	fs := afero.NewMemMapFs()
	filename := "file.txt"

	l := NewFileLock(filename, fs)

	errLock := l.Lock()
	assert.NoError(t, errLock)

	// Spin this off in a goroutine so that we can manipulate the lock
	go func() {
		errWaitForLock := l.WaitForLock()
		assert.NoError(t, errWaitForLock)
	}()

	// Wait long enough for code to loop
	time.Sleep(500 * time.Millisecond)
	errUnlock := l.Unlock()
	assert.NoError(t, errUnlock)
}

func TestFileLockWaitError(t *testing.T) {

	fs := afero.NewMemMapFs()
	filename := "file.txt"

	l := NewFileLock(filename, fs)

	// Timeout as fast as possible
	l.SetTimeout(1 * time.Nanosecond)

	errWaitForLock := l.WaitForLock()
	assert.Error(t, errWaitForLock)
	assert.Equal(t, "unable to obtain lock after 1ns: context deadline exceeded", errWaitForLock.Error())
}
