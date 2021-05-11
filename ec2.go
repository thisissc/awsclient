package awsclient

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/pkg/errors"
)

func GetLocalIpv4() (string, error) {
	resp, err := http.Get("http://169.254.169.254/latest/meta-data/local-ipv4")
	if err != nil {
		return "", errors.Wrap(err, "Request local-ipv4 error")
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	return string(body), nil
}

func GetInstantsOfTargetGroup(sess *session.Session, targetGroupArn string) []string {
	svc := elbv2.New(sess)
	input := &elbv2.DescribeTargetHealthInput{
		TargetGroupArn: aws.String(targetGroupArn),
	}

	result, err := svc.DescribeTargetHealth(input)
	if err != nil {
		log.Println(err)
		return make([]string, 0)
	}

	instids := make([]string, 0)
	for _, desc := range result.TargetHealthDescriptions {
		if *desc.TargetHealth.State == "healthy" {
			instids = append(instids, *desc.Target.Id)
		}
	}

	ipMap := GetInstancesIp(sess, instids)

	addrList := make([]string, 0)
	for _, desc := range result.TargetHealthDescriptions {
		if *desc.TargetHealth.State == "healthy" {
			ip := ipMap[*desc.Target.Id]
			port := *desc.Target.Port
			addrList = append(addrList, fmt.Sprintf("%s:%d", ip, port))
		}
	}

	return addrList
}

func GetInstancesIp(sess *session.Session, instids []string) map[string]string {
	inputInstids := make([]*string, len(instids))
	for i, v := range instids {
		inputInstids[i] = aws.String(v)
	}
	input := &ec2.DescribeInstancesInput{
		InstanceIds: inputInstids,
	}

	svc := ec2.New(sess)
	result, err := svc.DescribeInstances(input)
	if err != nil {
		log.Println(err)
		return nil
	}

	ipMap := make(map[string]string, 0)
	for _, item := range result.Reservations {
		instItem := item.Instances[0]
		ipMap[*instItem.InstanceId] = *instItem.PrivateIpAddress
	}
	return ipMap
}
