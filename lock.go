package safelock

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type LockState string

const (
	// LockStateLocked is the locked state
	LockStateLocked LockState = "locked"
	// LockStateUnlocked is the unlocked state
	LockStateUnlocked LockState = "unlocked"
	// LockStateUnknown is the unknown lock state
	LockStateUnknown LockState = "unknown"

	// DefaultTimeout is the default timeout used for locks
	DefaultTimeout time.Duration = 30 * time.Second

	// DefaultSuffix is the default lock suffix used for locks
	DefaultSuffix = ".lock"
)

// SafeLockiface is an interface for all implementations of locks
type SafeLockiface interface {
	Lock() error
	Unlock() error
	GetID() string
	SetID(string) error
	GetLockState() (LockState, error)
	GetLockURI() string
	GetLockSuffix() string
	SetLockSuffix(string)
	GetTimeout() time.Duration
	SetTimeout(time.Duration)
	WaitForLock() error
}

// SafeLock manages the internal locking and metadata for locks
type SafeLock struct {
	// This lock is internal to prevent two operations happening at the same time on this lock
	mu sync.Mutex

	id         uuid.UUID
	lockSuffix string
	timeout    time.Duration
}

// NewSafeLock creates a new instance of SafeLock
func NewSafeLock() *SafeLock {
	return &SafeLock{
		id:         uuid.New(),
		timeout:    DefaultTimeout,
		lockSuffix: DefaultSuffix,
	}
}

// Lock will lock
func (l *SafeLock) Lock() error {
	// Do operations that require the internal lock first
	l.mu.Lock()
	defer l.mu.Unlock()
	// Nothing is done here
	return nil
}

// Unlock will unlock
func (l *SafeLock) Unlock() error {
	// Do operations that require the internal lock first
	l.mu.Lock()
	defer l.mu.Unlock()
	// Nothing is done here
	return nil
}

// GetID returns the string representation of the lock's UUID
func (l *SafeLock) GetID() string {
	return l.id.String()
}

// SetID sets the lock's UUID
func (l *SafeLock) SetID(lockID string) error {
	u, errParse := uuid.Parse(lockID)
	if errParse != nil {
		return errParse
	}
	l.id = u
	return nil
}

// GetLockState returns the lock's state
func (l *SafeLock) GetLockState() (LockState, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return LockStateUnlocked, nil
}

// GetLockURI will return the URI for the lock object
func (l *SafeLock) GetLockURI() string {
	return ""
}

// GetLockSuffix returns the lock suffix being used
func (l *SafeLock) GetLockSuffix() string {
	return l.lockSuffix
}

// SetLockSuffix sets the lock suffix to use
func (l *SafeLock) SetLockSuffix(lockSuffix string) {
	l.lockSuffix = lockSuffix
}

// GetTimeout returns the timeout being used
func (l *SafeLock) GetTimeout() time.Duration {
	return l.timeout
}

// SetTimeout sets the timeout to use
func (l *SafeLock) SetTimeout(timeout time.Duration) {
	l.timeout = timeout
}

// WaitForLock waits until an object is no longer locked or cancels based on a timeout
func (l *SafeLock) WaitForLock() error {
	// Do not lock/unlock the struct here or it will block getting the lock state
	return nil
}
