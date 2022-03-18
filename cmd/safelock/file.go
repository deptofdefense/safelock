package main

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/gage-technologies/safelock"

	"github.com/google/uuid"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	// File Lock flags
	flagFileLockFilename = "filename"
	flagFileLockAction   = "action"
	flagFileLockID       = "lock-id"
	flagFileLockNode     = "node"
)

func initFileLockFlags(flag *pflag.FlagSet) {
	flag.String(flagFileLockFilename, "", "The filename")
	flag.String(flagFileLockAction, actionLock, "The action to use")
	flag.Uint64(flagFileLockID, uint64(time.Now().UnixNano()), "The id of the lock to act upon")
	flag.Uint(flagFileLockNode, math.MaxInt, "The node of the lock to act upon")
}

func checkFileLockConfig(v *viper.Viper) error {

	filename := v.GetString(flagFileLockFilename)
	if len(filename) == 0 {
		return errors.New("A filename is required")
	}

	action := v.GetString(flagFileLockAction)
	if !stringSliceContains(validActions, action) {
		return fmt.Errorf("Action %q is not valid, must be one of %v", action, validActions)
	}
	if action == actionUnlock {
		lockID := v.GetString(flagFileLockID)
		if len(lockID) == 0 {
			return errors.New("A lock ID is required when unlocking")
		}
		_, errParse := uuid.Parse(lockID)
		if errParse != nil {
			return fmt.Errorf("Lock ID %q is not a valid UUID", lockID)
		}
	}

	return nil
}

func fileLockCmd(cmd *cobra.Command, args []string) error {
	v, errViper := initViper(cmd)
	if errViper != nil {
		return fmt.Errorf("error initializing viper: %w", errViper)
	}

	if len(args) > 1 {
		return cmd.Usage()
	}

	if errConfig := checkFileLockConfig(v); errConfig != nil {
		return errConfig
	}

	filename := v.GetString(flagFileLockFilename)
	action := v.GetString(flagFileLockAction)

	node := v.GetUint(flagFileLockNode)

	fs := afero.NewOsFs()
	l := safelock.NewFileLock(uint16(node), filename, fs)

	lockID := v.GetUint64(flagFileLockID)
	l.SetID(lockID)

	switch action {
	case actionLock:
		errWaitForLock := l.WaitForLock(safelock.DefaultTimeout)
		if errWaitForLock != nil {
			return errWaitForLock
		}
		errLock := l.Lock()
		if errLock != nil {
			return errLock
		}
	case actionUnlock:
		errUnlock := l.Unlock()
		if errUnlock != nil {
			return errUnlock
		}
	}

	return nil
}
