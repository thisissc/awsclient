package awsclient

import (
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"

	"github.com/avast/retry-go"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pkg/errors"
)

func Upload2S3(body io.Reader, key, bucket, profile string) error {
	sess := GetSessionByProfile(profile)
	uploader := s3manager.NewUploader(sess)

	contentType := ""
	fileExt := filepath.Ext(key)
	switch fileExt {
	case "jpeg", "jpg":
		contentType = "image/jpeg"
	case "png":
		contentType = "image/png"
	case "gif":
		contentType = "image/gif"
	case "html":
		contentType = "text/html; charset=UTF-8"
	case "json":
		contentType = "application/json"
	}

	var resultErr error
	retryErr := retry.Do(
		func() error {
			uploadInput := &s3manager.UploadInput{
				Body:   body,
				Bucket: aws.String(bucket),
				Key:    aws.String(key),
			}
			if len(contentType) > 0 {
				uploadInput.ContentType = aws.String(contentType)
			}
			_, resultErr = uploader.Upload(uploadInput)
			return resultErr
		},
		retry.OnRetry(func(n uint, err error) {
			log.Println(fmt.Sprintf("%s\nRetrying %d...", err, n))
		}),
		retry.RetryIf(func(err error) bool {
			return strings.Contains(err.Error(), "timeout")
		}),
		retry.Attempts(2),
	)
	if resultErr != nil && retryErr != nil {
		return errors.Wrap(retryErr, "Upload2S3 failed")
	}
	return nil
}

func GetObjectFromS3(key, bucket, profile string) (io.ReadCloser, error) {
	sess := GetSessionByProfile(profile)
	svc := s3.New(sess)
	output, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == s3.ErrCodeNoSuchKey {
				return &EmptyReadCloser{}, nil
			}
		}

		return nil, errors.Wrap(err, "GetObjectFromS3 error")
	}

	return output.Body, nil
}

func DownloadFromS3(w io.WriterAt, key, bucket, profile string) error {
	sess := GetSessionByProfile(profile)
	downloader := s3manager.NewDownloader(sess)

	_, err := downloader.Download(w, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return errors.Wrap(err, "failed to download file")
	}
	return nil
}
