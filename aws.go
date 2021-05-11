package awsclient

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
)

const (
	AWSProfileDefault = "DEFAULT"
	AWSProfileNingxia = "NINGXIA"
	AWSProfileGlobal  = "GLOBAL"
)

var (
	awsSessionInstMap = make(map[string]*session.Session, 0)
)

func GetSession() *session.Session {
	return GetSessionByProfile(AWSProfileDefault)
}

func SetSession(profile string, cfg AwsConfig) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(cfg.Region),
		Credentials: credentials.NewStaticCredentials(cfg.AccessKeyId, cfg.AccessKeySecret, ""),
	})
	if err != nil {
		log.Println(err)
	} else {
		awsSessionInstMap[profile] = sess
	}
}

func GetSessionByProfile(profile string) *session.Session {
	sessionInst, ok := awsSessionInstMap[profile]

	if !ok || sessionInst == nil {
		return nil
	}

	return sessionInst
}
