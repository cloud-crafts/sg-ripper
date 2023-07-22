package awsClients

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"sg-ripper/pkg/core/types"
)

type AwsEcsClient struct {
	client *ecs.Client
	cache  map[string]*types.ECSAttachment
}

func NewAwsEcsClient(cfg aws.Config) *AwsEcsClient {
	return &AwsEcsClient{
		client: ecs.NewFromConfig(cfg),
		cache:  make(map[string]*types.ECSAttachment),
	}
}

func (c *AwsEcsClient) GetECSAttachment(eni ec2Types.NetworkInterface) (*types.ECSAttachment, error) {
	var cluster, service *string
	for _, tag := range eni.TagSet {
		if tag.Key != nil && *tag.Key == "aws:ecs:clusterName" {
			cluster = tag.Value
			continue
		}
		if tag.Key != nil && *tag.Key == "aws:ecs:serviceName" {
			service = tag.Value
		}
	}

	taskArn, container, ecsErr := c.getTaskAndContainerInfo(eni, cluster, service)

	if ecsErr != nil {
		return nil, ecsErr
	}

	if cluster != nil && service != nil {
		return &types.ECSAttachment{
			IsRemoved:     taskArn == nil,
			ClusterName:   cluster,
			ServiceName:   service,
			ContainerName: container,
			TaskArn:       taskArn,
		}, nil
	}
	return nil, nil
}

func (c *AwsEcsClient) getTaskAndContainerInfo(eni ec2Types.NetworkInterface, cluster, service *string) (*string, *string, error) {
	if cluster != nil && service != nil {
		var taskArn, containerName *string
		var nexToken *string
		for {
			tasks, err := c.client.ListTasks(context.TODO(), &ecs.ListTasksInput{
				Cluster:     cluster,
				ServiceName: service,
				MaxResults:  aws.Int32(int32(100)), // use 100 to avoid looping for DescribeTasks
				NextToken:   nexToken,
			})

			if err != nil {
				return nil, nil, err
			}

			detailedTasks, taskDescribeErr := c.client.DescribeTasks(context.TODO(), &ecs.DescribeTasksInput{
				Cluster: cluster,
				Tasks:   tasks.TaskArns,
			})

			if taskDescribeErr != nil {
				return nil, nil, taskDescribeErr
			}

		out:
			for _, task := range detailedTasks.Tasks {
				for _, container := range task.Containers {
					for _, containerEni := range container.NetworkInterfaces {
						if *eni.PrivateIpAddress == *containerEni.PrivateIpv4Address {
							containerName = container.Name
							taskArn = task.TaskArn
							break out
						}
					}
				}
			}

			if tasks.NextToken != nil {
				nexToken = tasks.NextToken
			} else {
				break
			}
		}
		return taskArn, containerName, nil
	}
	return nil, nil, nil
}
