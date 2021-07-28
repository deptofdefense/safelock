# SafeLock CLI

The CLI command is presented as a working example of the SafeLock package. It is not meant for production use.

## File Example

To lock a File use:

```sh
go run github.com/deptofdefense/safelock/cmd/safelock file --filename tmp.txt --action lock
```

The output of the command is a UUID which is the ID of the lock, for example: `3218d0b5-60b4-4380-afdd-f93f49b4a907`

To unlock a File which has already been locked use:

```sh
go run github.com/deptofdefense/safelock/cmd/safelock file --filename tmp.txt --action unlock --lock-id 3218d0b5-60b4-4380-afdd-f93f49b4a907
```

## S3Object Example

To lock an S3 Object use:

```sh
go run github.com/deptofdefense/safelock/cmd/safelock s3object --s3-bucket $bucket --s3-key $key --s3-kms-key-arn $kmskeyarn --action lock
```

The output of the command is a UUID which is the ID of the lock, for example: `cdb178bf-df3b-482c-b3dc-e7484b9a20c5`

To unlock an S3 object which has already been locked use:

```sh
go run github.com/deptofdefense/safelock/cmd/safelock s3object --s3-bucket $bucket --s3-key $key --s3-kms-key-arn $kmskeyarn --action unlock --lock-id cdb178bf-df3b-482c-b3dc-e7484b9a20c5
```
