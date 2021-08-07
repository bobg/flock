// Package flock implements timed, advisory file locks.
package flock

import (
	"errors"
	"os"
	"time"
)

// Locker is an object that can create timed, advisory file locks.
// The zero value of Locker is ready to use.
type Locker struct {
	// Lockfile is a function that can convert a file's path into the path of the corresponding lockfile.
	// If this is unset, the default for path X is X.lock.
	Lockfile func(path string) string

	// LockDur is the amount of time that a lock is in effect.
	// A lockfile older than this does not prevent Lock from working.
	// If this is unset, the default is one minute.
	LockDur time.Duration
}

var (
	// ErrLocked is the error produced by Lock when trying to lock a path that is already locked.
	ErrLocked = errors.New("locked")

	// ErrNotLocked is the error produced by Refresh when trying to refresh to lock of a path that is not locked.
	ErrNotLocked = errors.New("not locked")
)

var defaultDur = time.Minute

// Lock tries to acquire a lock on the given path.
// If a lockfile already exists and is not older than Locker's lock duration,
// this returns with ErrLocked.
func (l Locker) Lock(path string) error {
	lockfile := l.lockfile(path)
	err := l.removeIfExpired(lockfile)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(lockfile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if errors.Is(err, os.ErrExist) {
		return ErrLocked
	}
	if err != nil {
		return err
	}
	return f.Close()
}

func (l Locker) removeIfExpired(lockfile string) error {
	info, err := os.Stat(lockfile)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.ModTime().Add(l.lockDur()).Before(time.Now()) {
		return os.Remove(lockfile)
	}
	return ErrLocked
}

// Unlock removes the lock on the given path.
// It is not an error to call this on a path that is not locked.
func (l Locker) Unlock(path string) error {
	lockfile := l.lockfile(path)
	err := os.Remove(lockfile)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// Refresh updates the timestamp on the lock for the given path.
// If the path is not locked, this returns ErrNotLocked.
func (l Locker) Refresh(path string) error {
	lockfile := l.lockfile(path)
	err := l.removeIfExpired(lockfile)
	if err != nil && !errors.Is(err, ErrLocked) {
		return err
	}
	now := time.Now()
	err = os.Chtimes(lockfile, now, now)
	if errors.Is(err, os.ErrNotExist) {
		return ErrNotLocked
	}
	return err
}

func (l Locker) lockfile(path string) string {
	if l.Lockfile != nil {
		return l.Lockfile(path)
	}
	return path + ".lock"
}

func (l Locker) lockDur() time.Duration {
	if l.LockDur != 0 {
		return l.LockDur
	}
	return defaultDur
}
