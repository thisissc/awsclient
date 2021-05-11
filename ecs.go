package awsclient

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"bufio"
	"fmt"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/pkg/errors"
)

type EcsAgentMetadata struct {
	Cluster string `json:"Cluster"`
}

type EcsAgentTaskMetadata struct {
	Tasks []struct {
		Arn        string `json:"Arn"`
		Containers []struct {
			DockerId string `json:"DockerId"`
		} `json:"Containers"`
	} `json:"Tasks"`
}

// See
// https://github.com/aws/amazon-ecs-agent/issues/258
// https://github.com/aws/amazon-ecs-agent/pull/709
func ecsAgentTaskMetadata() (EcsAgentTaskMetadata, error) {
	response, err := http.Get("http://172.17.0.1:51678/v1/tasks")
	if err != nil {
		return EcsAgentTaskMetadata{}, err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return EcsAgentTaskMetadata{}, err
	}

	metadata := EcsAgentTaskMetadata{}
	err = json.Unmarshal(body, &metadata)
	if err != nil {
		return EcsAgentTaskMetadata{}, err
	}

	return metadata, nil
}

func ecsAgentMetadata() (EcsAgentMetadata, error) {
	response, err := http.Get("http://172.17.0.1:51678/v1/metadata")
	if err != nil {
		return EcsAgentMetadata{}, err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return EcsAgentMetadata{}, err
	}

	metadata := EcsAgentMetadata{}
	err = json.Unmarshal(body, &metadata)
	if err != nil {
		return EcsAgentMetadata{}, err
	}

	return metadata, nil
}

func getDockerId() (string, error) {
	file, err := os.Open("/proc/self/cgroup")
	if err != nil {
		fmt.Println("Could not open /proc/self/cgroup, using hostname instead as container id")
		return os.Hostname()
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		// Example: "2:cpu:/docker/93c562c426414f53582c9830a30bdb54d85642956e18115dd59bc9f435ae5644"
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		components := strings.Split(line, ":")
		if len(components) == 3 {
			return strings.TrimRight(path.Base(components[2]), "\n"), nil
		}
	}

	return "", fmt.Errorf("Failed to find Docker ID in /proc/self/group")
}

func GetEcsTaskPort(profile string, containerPort int) (int, error) {
	// get docker id
	dockerId, err := getDockerId()
	if err != nil {
		return 0, errors.Wrap(err, "Get DockerId error")
	}

	// get cluster name
	agentMetadata, err := ecsAgentMetadata()
	if err != nil {
		return 0, errors.Wrap(err, "Get ECS Agent Metadata error")
	}
	clusterName := agentMetadata.Cluster

	if clusterName == "" {
		return 0, fmt.Errorf("Could not find ECS cluster for docker container '%s'", dockerId)
	}

	// get task arn
	agentTaskMetadata, err := ecsAgentTaskMetadata()
	if err != nil {
		return 0, errors.Wrap(err, "Get ECS Agent Task Metadata error")
	}

	taskArn := ""
	for _, task := range agentTaskMetadata.Tasks {
		for _, container := range task.Containers {
			if strings.HasPrefix(container.DockerId, dockerId) {
				taskArn = task.Arn
				break
			}
		}
	}
	if taskArn == "" {
		return 0, fmt.Errorf("Could not find ECS task for docker container '%s' on cluster '%s'", dockerId, clusterName)
	}

	// get host port
	sess := GetSessionByProfile(profile)
	ecsSvc := ecs.New(sess)
	tasks, err := ecsSvc.DescribeTasks(&ecs.DescribeTasksInput{
		Cluster: aws.String(clusterName),
		Tasks:   []*string{aws.String(taskArn)},
	})
	if err != nil {
		return 0, errors.Wrap(err, "DescribeTasks error")
	}

	for _, task := range tasks.Tasks {
		for _, container := range task.Containers {
			for _, binding := range container.NetworkBindings {
				if binding.ContainerPort != nil && int(*binding.ContainerPort) == containerPort && binding.HostPort != nil {
					return int(*binding.HostPort), nil
				}
			}
		}
	}

	return 0, errors.New(fmt.Sprintf("Can't find binding of ContainerPort: %d", containerPort))
}

func GetEcsTaskAddr(port int) (string, error) {
	// Get ECS Dynamic Host and Port
	instanceIp, err := GetLocalIpv4()
	if err != nil {
		return "", errors.Wrap(err, "awsclient.GetLocalIpv4 error")
	}
	instancePort, err := GetEcsTaskPort(AWSProfileDefault, port)
	if err != nil {
		return "", errors.Wrap(err, "awsclient.GetEcsTaskPort error")
	}

	serviceAddr := fmt.Sprintf("%s:%d", instanceIp, instancePort)
	return serviceAddr, nil
}

func GetEcsServiceAddrList(sess *session.Session, clusterName, serviceName string) (addrList []string) {
	addrList = make([]string, 0)

	svc := ecs.New(sess)
	input := &ecs.ListTasksInput{
		Cluster:     aws.String(clusterName),
		ServiceName: aws.String(serviceName),
	}

	output, err := svc.ListTasks(input)
	if err != nil {
		log.Println(err)
		return
	}

	dtinput := &ecs.DescribeTasksInput{
		Cluster: aws.String(clusterName),
		Tasks:   output.TaskArns,
	}

	dtoutput, err := svc.DescribeTasks(dtinput)
	if err != nil {
		log.Println(err)
		return
	}

	for _, task := range dtoutput.Tasks {
		if *task.DesiredStatus != "RUNNING" {
			continue
		}

		if len(task.Containers) == 0 {
			continue
		} else if len(task.Containers[0].NetworkBindings) == 0 {
			continue
		}

		port := *task.Containers[0].NetworkBindings[0].HostPort

		input := &ecs.DescribeContainerInstancesInput{
			Cluster: aws.String(clusterName),
			ContainerInstances: []*string{
				task.ContainerInstanceArn,
			},
		}

		dcioutput, err := svc.DescribeContainerInstances(input)
		if err != nil {
			log.Println(err)
			continue
		}

		iid := dcioutput.ContainerInstances[0].Ec2InstanceId
		ipMap := GetInstancesIp(sess, []string{*iid})
		ip := ipMap[*iid]

		addr := fmt.Sprintf("%s:%d", ip, port)
		addrList = append(addrList, addr)
	}

	return addrList
}
