# SafeLock

SafeLock is a golang package for locking files used by distributed services.

Use SafeLock when distributed processes act on the same file on a local or remote filesystem and require
exclusive access to that file. SafeLock should not be used to protect access in non-distributed systems as better mechanisms exist.

To use this module add `import "github.com/deptofdefense/safelock"` to your package.

## Example

To use the lock in AWS S3 you could use this code:

```golang
svcS3 := s3.NewFromConfig(cfg)
bucket := "bucket"
key := "key"
kmsKeyArn := "kmsKeyArn"
l := safelock.NewS3ObjectLock(bucket, key, kmsKeyArn, &svcS3)

// Wait for the lock to become available
l.WaitForLock()

// Lock the object and defer unlocking
l.Lock()
defer l.Unlock()

// Do work on the object that was locked
```

This can also be accomplished with a local filesystem:

```golang
fs := afero.NewOsFs()
filename := "file.txt"
l := safelock.NewFileLock(filename, fs)
```

SafeLock also provides a primitive to build your own lock named `SafeLock`.

## Testing

Testing can be done using the `Makefile` targets `make test` and `make test_coverage`.

## Development

Development for this has been geared towards MacOS users. Install dependencies to get started:

```sh
brew install circleci go golangci-lint pre-commit shellcheck
```

Install the pre-commit hooks and run them before making pull requests:

```sh
pre-commit install
pre-commit run -a
```

## License

This project constitutes a work of the United States Government and is not subject to domestic copyright protection under 17 USC § 105.  However, because the project utilizes code licensed from contributors and other third parties, it therefore is licensed under the MIT License.  See LICENSE file for more information.
