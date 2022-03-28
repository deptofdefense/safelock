package safelock

import (
	"encoding/binary"
	"fmt"
	"sync"
	"time"
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
	ForceUnlock() error
	GetID() uint64
	GetNode() uint16
	GetIDBytes() []byte
	GetNodeBytes() []byte
	SetID(uint64)
	SetNode(uint16)
	SetIDBytes([]byte) error
	SetNodeBytes([]byte) error
	GetLockBody() []byte
	GetLockState() (LockState, error)
	GetLockURI() string
	GetLockSuffix() string
	SetLockSuffix(string)
	GetTimeout() time.Duration
	SetTimeout(time.Duration)
	WaitForLock(time.Duration) error
}

// SafeLock manages the internal locking and metadata for locks
type SafeLock struct {
	// This lock is internal to prevent two operations happening at the same time on this lock
	mu sync.Mutex

	node       uint16
	id         uint64
	lockSuffix string
	timeout    time.Duration
}

// NewSafeLock creates a new instance of SafeLock
func NewSafeLock(node uint16) *SafeLock {
	return &SafeLock{
		node:       node,
		id:         uint64(time.Now().UnixNano()),
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

// ForceUnlock will unlock despite a lack of ownership
func (l *SafeLock) ForceUnlock() error {
	// Do operations that require the internal lock first
	l.mu.Lock()
	defer l.mu.Unlock()
	// Nothing is done here
	return nil
}

// GetID returns the lock's id
func (l *SafeLock) GetID() uint64 {
	return l.id
}

// GetNode returns the lock's node number
func (l *SafeLock) GetNode() int {
	return int(l.node)
}

// GetIDBytes returns the lock's id in the form of a byte slice
func (l *SafeLock) GetIDBytes() []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, l.id)
	return b
}

// GetNodeBytes returns the lock's node number in the form of a byte slice
func (l *SafeLock) GetNodeBytes() []byte {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, l.node)
	return b
}

// SetID sets the lock's id
func (l *SafeLock) SetID(id uint64) {
	l.id = id
}

// SetNode sets the lock's node number
func (l *SafeLock) SetNode(node uint16) {
	l.node = node
}

// SetIDBytes sets the lock's id using a little-endian encoded uint32
func (l *SafeLock) SetIDBytes(buf []byte) error {
	if len(buf) != 8 {
		return fmt.Errorf("incorrect buffer length for serialized id: %d != 8", len(buf))
	}
	l.id = binary.LittleEndian.Uint64(buf)
	return nil
}

// SetNodeBytes sets the lock's node number using a little endian encoded uint16
func (l *SafeLock) SetNodeBytes(buf []byte) error {
	if len(buf) != 2 {
		return fmt.Errorf("incorrect buffer length for serialized id: %d != 2", len(buf))
	}
	l.node = binary.LittleEndian.Uint16(buf)
	return nil
}

// GetLockBody returns the byte slice representation of the lock for the lock file
func (l *SafeLock) GetLockBody() []byte {
	// encode timestamp in little endian
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(time.Now().UnixNano()))

	body := l.GetNodeBytes()
	body = append(body, []byte("\n")...)
	body = append(body, l.GetIDBytes()...)
	body = append(body, []byte("\n")...)
	body = append(body, buf...)
	return body
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
func (l *SafeLock) WaitForLock(time.Duration) error {
	// Do not lock/unlock the struct here or it will block getting the lock state
	return nil
}
