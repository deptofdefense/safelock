package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	// Generic Flags for most commands
	flagAWSRegion = "aws-region"
)

func initViper(cmd *cobra.Command) (*viper.Viper, error) {
	v := viper.New()
	errBind := v.BindPFlags(cmd.Flags())
	if errBind != nil {
		return v, fmt.Errorf("error binding flag set to viper: %w", errBind)
	}
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv() // set environment variables to overwrite config
	return v, nil
}

// stringSliceContains returns true if a string is found inside of a string slice
func stringSliceContains(stringSlice []string, value string) bool {
	for _, x := range stringSlice {
		if value == x {
			return true
		}
	}
	return false
}

func main() {
	rootCommand := &cobra.Command{
		Use:                   `safelock [flags]`,
		DisableFlagsInUseLine: true,
		Short:                 "SafeLock is a golang package for locking files used by distributed services",
	}

	s3ObjectLockCommand := &cobra.Command{
		Use:                   `s3object [flags]`,
		DisableFlagsInUseLine: true,
		Short:                 "Use SafeLock on an S3 Object",
		SilenceErrors:         true,
		SilenceUsage:          true,
		RunE:                  s3ObjectLockCmd,
	}
	initS3ObjectLockFlags(s3ObjectLockCommand.Flags())

	rootCommand.AddCommand(
		s3ObjectLockCommand,
	)

	if err := rootCommand.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "safelock: "+err.Error())
		_, _ = fmt.Fprintln(os.Stderr, "Try safelock --help for more information.")
		os.Exit(1)
	}

}
