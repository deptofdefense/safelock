package safelock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSafeLock(t *testing.T) {

	l := NewSafeLock(0)

	// These have no affect but here for code coverage
	errLock := l.Lock()
	assert.NoError(t, errLock)
	errUnlock := l.Unlock()
	assert.NoError(t, errUnlock)

	nodeCreation := time.Unix(0, int64(l.GetID()))
	assert.True(t, time.Since(nodeCreation) < time.Second)

	lockState, errGetLockState := l.GetLockState()
	assert.NoError(t, errGetLockState)
	assert.Equal(t, LockStateUnlocked, lockState)

	// LockURI
	assert.Equal(t, "", l.GetLockURI())

	// Suffix
	assert.Equal(t, DefaultSuffix, l.GetLockSuffix())
	newSuffix := ".newlock"
	l.SetLockSuffix(newSuffix)
	assert.Equal(t, newSuffix, l.GetLockSuffix())

	// Timeout
	assert.Equal(t, DefaultTimeout, l.GetTimeout())
	newTimeout := 1 * time.Second
	l.SetTimeout(newTimeout)
	assert.Equal(t, newTimeout, l.GetTimeout())

	// Wait
	errWaitForLock := l.WaitForLock(DefaultTimeout)
	assert.NoError(t, errWaitForLock)

	// SetID
	l.SetID(37583)
	assert.Equal(t, l.GetID(), uint64(37583))
}
