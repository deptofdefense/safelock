package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/gage-technologies/safelock"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	// S3 Object Lock flags
	flagS3ObjectLockBucket    = "s3-bucket"
	flagS3ObjectLockKey       = "s3-key"
	flagS3ObjectLockKMSKeyArn = "s3-kms-key-arn"
	flagS3ObjectLockAction    = "action"
	flagS3ObjectLockID        = "lock-id"
	flagS3ObjectLockNode      = "node"

	actionLock   = "lock"
	actionUnlock = "unlock"
)

var (
	validActions = []string{actionLock, actionUnlock}
)

func initS3ObjectLockFlags(flag *pflag.FlagSet) {
	flag.String(flagAWSRegion, "us-west-2", "The AWS region")
	flag.String(flagS3ObjectLockBucket, "", "The s3 bucket")
	flag.String(flagS3ObjectLockKey, "", "The s3 key")
	flag.String(flagS3ObjectLockKMSKeyArn, "", "The s3 kms key ARN")
	flag.String(flagS3ObjectLockAction, actionLock, "The action to use")
	flag.Uint64(flagS3ObjectLockID, uint64(time.Now().UnixNano()), "The id of the lock to act upon")
	flag.Uint(flagS3ObjectLockNode, math.MaxInt, "The node of the lock to act upon")
}

func checkS3ObjectLockConfig(v *viper.Viper) error {

	awsRegion := v.GetString(flagAWSRegion)
	if len(awsRegion) == 0 {
		return errors.New("An AWS region is required")
	}

	bucket := v.GetString(flagS3ObjectLockBucket)
	if len(bucket) == 0 {
		return errors.New("An s3 bucket is required")
	}

	key := v.GetString(flagS3ObjectLockKey)
	if len(key) == 0 {
		return errors.New("An s3 key is required")
	}

	kmsKeyArn := v.GetString(flagS3ObjectLockKMSKeyArn)
	if len(kmsKeyArn) == 0 {
		return errors.New("An s3 kms key ARN is required")
	}

	action := v.GetString(flagS3ObjectLockAction)
	if !stringSliceContains(validActions, action) {
		return fmt.Errorf("Action %q is not valid, must be one of %v", action, validActions)
	}
	if action == actionUnlock {
		lockID := v.GetString(flagS3ObjectLockID)
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

func s3ObjectLockCmd(cmd *cobra.Command, args []string) error {
	v, errViper := initViper(cmd)
	if errViper != nil {
		return fmt.Errorf("error initializing viper: %w", errViper)
	}

	if len(args) > 1 {
		return cmd.Usage()
	}

	if errConfig := checkS3ObjectLockConfig(v); errConfig != nil {
		return errConfig
	}

	awsRegion := v.GetString(flagAWSRegion)
	bucket := v.GetString(flagS3ObjectLockBucket)
	key := v.GetString(flagS3ObjectLockKey)
	kmsKeyArn := v.GetString(flagS3ObjectLockKMSKeyArn)
	action := v.GetString(flagS3ObjectLockAction)

	awsCfg, errCfg := config.LoadDefaultConfig(context.TODO(), config.WithRegion(awsRegion))
	if errCfg != nil {
		return errCfg
	}
	svcS3 := s3.NewFromConfig(awsCfg)

	node := v.GetUint(flagS3ObjectLockNode)

	l := safelock.NewS3ObjectLock(uint16(node), bucket, key, kmsKeyArn, svcS3)

	lockID := v.GetUint64(flagS3ObjectLockID)
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
		fmt.Println(l.GetID())
	case actionUnlock:
		errUnlock := l.Unlock()
		if errUnlock != nil {
			return errUnlock
		}
	}

	return nil
}
