package main

import (
	"errors"
	"fmt"

	"github.com/deptofdefense/safelock"

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
)

func initFileLockFlags(flag *pflag.FlagSet) {
	flag.String(flagFileLockFilename, "", "The filename")
	flag.String(flagFileLockAction, actionLock, "The action to use")
	flag.String(flagFileLockID, "", "The id of the lock to act upon")
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

	fs := afero.NewOsFs()
	l := safelock.NewFileLock(filename, fs)

	switch action {
	case actionLock:
		errWaitForLock := l.WaitForLock()
		if errWaitForLock != nil {
			return errWaitForLock
		}
		errLock := l.Lock()
		if errLock != nil {
			return errLock
		}
		fmt.Println(l.GetID())
	case actionUnlock:
		lockID := v.GetString(flagFileLockID)
		errSetID := l.SetID(lockID)
		if errSetID != nil {
			return errSetID
		}
		errUnlock := l.Unlock()
		if errUnlock != nil {
			return errUnlock
		}
	}

	return nil
}
