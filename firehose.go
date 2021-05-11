package awsclient

import (
	"fmt"
	"log"
	"strings"

	"github.com/avast/retry-go"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/firehose"
)

func Send2Firehose(profile, streamName string, payload []byte) error {
	sess := GetSessionByProfile(profile)
	firehoseSvc := firehose.New(sess)

	input := &firehose.PutRecordInput{
		DeliveryStreamName: aws.String(streamName),
		Record: &firehose.Record{
			Data: payload,
		},
	}

	err := retry.Do(
		func() error {
			_, err := firehoseSvc.PutRecord(input)
			return err
		},
		retry.OnRetry(func(n uint, err error) {
			log.Println(fmt.Sprintf("%s\nRetrying %d...", err, n))
		}),
		retry.RetryIf(func(err error) bool {
			return strings.Contains(err.Error(), "timeout")
		}),
		retry.Attempts(2),
	)

	return err
}
