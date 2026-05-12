package awssts

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

const (
	KeyAccessKeyID     = "AWS_ACCESS_KEY_ID"
	KeySecretAccessKey = "AWS_SECRET_ACCESS_KEY"
	KeySessionToken    = "AWS_SESSION_TOKEN"
	KeyDefaultRegion   = "AWS_DEFAULT_REGION"
	KeyRegion          = "AWS_REGION"
	KeyRoleARN         = "AWS_ROLE_ARN"

	defaultRegion   = "us-east-1"
	sessionDuration = 1 * time.Hour
	sessionName     = "envault"
)

// ShouldAssumeRole returns true if the vars contain AWS credentials and a
// role ARN to assume. Without a role ARN, credentials are passed through as-is.
func ShouldAssumeRole(vars map[string]string) bool {
	_, hasKey := vars[KeyAccessKeyID]
	_, hasSecret := vars[KeySecretAccessKey]
	_, hasRole := vars[KeyRoleARN]
	return hasKey && hasSecret && hasRole
}

// AssumeRole calls STS AssumeRole using the long-term credentials and role ARN
// in vars, then replaces the credentials with temporary ones.
// The AWS_ROLE_ARN key is removed from vars so it doesn't leak into the child.
func AssumeRole(vars map[string]string) error {
	region := vars[KeyRegion]
	if region == "" {
		region = vars[KeyDefaultRegion]
	}
	if region == "" {
		region = defaultRegion
	}

	creds := credentials.NewStaticCredentialsProvider(
		vars[KeyAccessKeyID],
		vars[KeySecretAccessKey],
		"",
	)

	client := sts.New(sts.Options{
		Region:      region,
		Credentials: creds,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	out, err := client.AssumeRole(ctx, &sts.AssumeRoleInput{
		RoleArn:         aws.String(vars[KeyRoleARN]),
		RoleSessionName: aws.String(sessionName),
		DurationSeconds: aws.Int32(int32(sessionDuration.Seconds())),
	})
	if err != nil {
		return fmt.Errorf("STS AssumeRole (%s): %w", vars[KeyRoleARN], err)
	}

	vars[KeyAccessKeyID] = *out.Credentials.AccessKeyId
	vars[KeySecretAccessKey] = *out.Credentials.SecretAccessKey
	vars[KeySessionToken] = *out.Credentials.SessionToken
	delete(vars, KeyRoleARN)

	return nil
}
