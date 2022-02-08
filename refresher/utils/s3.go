package utils

import (
	"bytes"
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/infralight/pulumi/refresher/config"
)

func WriteFile(bucket, path string, content []byte, fileType string) (err error) {
	awsSession := config.LoadAwsSession()
	if awsSession == nil {
		return errors.New("failed to create AWS session.")
	}
	svc := s3.New(awsSession)

	_, err = svc.PutObject(
		&s3.PutObjectInput{
			Bucket:        aws.String(bucket),
			Key:           aws.String(path),
			Body:          bytes.NewReader(content),
			ContentLength: aws.Int64(int64(len(content))),
			ContentType:   aws.String(fileType),
		})
	return err
}
