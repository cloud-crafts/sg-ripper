package clients

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	coreTypes "github.com/cloud-crafts/sg-ripper/pkg/core/types"
	cmap "github.com/orcaman/concurrent-map/v2"
	"regexp"
)

type AwsEcsClient struct {
	client *ecs.Client
	cache  cmap.ConcurrentMap[string, *coreTypes.EcsAttachment]
}

func NewAwsEcsClient(cfg aws.Config) *AwsEcsClient {
	return &AwsEcsClient{
		client: ecs.NewFromConfig(cfg),
		cache:  cmap.New[*coreTypes.EcsAttachment](),
	}
}

// GetEcsAttachment returns a pointer to an EcsAttachment for the network interface. If there is no attachment found,
// the returned value is a nil.
func (c *AwsEcsClient) GetEcsAttachment(ctx context.Context, eni ec2Types.NetworkInterface) (*coreTypes.EcsAttachment, error) {
	regex := regexp.MustCompile(".+:ecs:.+:attachment/.+")
	if eni.Description != nil && regex.MatchString(*eni.Description) {
		if c.cache.IsEmpty() {
			if err := c.buildCache(ctx); err != nil {
				return nil, err
			}
		}

		attachment, err := c.getAttachmentFromCache(ctx, eni)
		if err != nil {
			return nil, err
		}

		if attachment == nil {
			return &coreTypes.EcsAttachment{
				IsRemoved: true,
			}, nil
		}

		return attachment, nil
	}

	return nil, nil
}

func (c *AwsEcsClient) buildCache(ctx context.Context) error {
	var nexToken *string

	clusterArns := make([]string, 0)

	for {
		clusters, err := c.client.ListClusters(ctx, &ecs.ListClustersInput{NextToken: nexToken})
		if err != nil {
			return err
		}

		for _, arn := range clusters.ClusterArns {
			clusterArns = append(clusterArns, arn)
		}

		if nexToken == nil {
			break
		}
	}

	for _, clusterArn := range clusterArns {
		for {
			taskResponse, err := c.client.ListTasks(ctx, &ecs.ListTasksInput{
				Cluster:    &clusterArn,
				MaxResults: aws.Int32(int32(100)),
				NextToken:  nexToken,
			})

			if err != nil {
				return err
			}

			describeTaskResponse, err := c.client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
				Tasks:   taskResponse.TaskArns,
				Cluster: &clusterArn,
			})

			for _, task := range describeTaskResponse.Tasks {
				for _, container := range task.Containers {
					for _, ifc := range container.NetworkInterfaces {
						c.cache.Set(*ifc.AttachmentId, &coreTypes.EcsAttachment{
							ClusterArn:    &clusterArn,
							ContainerName: container.Name,
							TaskArn:       task.TaskArn,
						})
					}
				}
			}

			if nexToken == nil {
				break
			}
		}
	}

	return nil
}

func (c *AwsEcsClient) getAttachmentFromCache(ctx context.Context, eni ec2Types.NetworkInterface) (*coreTypes.EcsAttachment, error) {
	regex := regexp.MustCompile(".+attachment/(?P<attachmentId>.+)")
	match := regex.FindStringSubmatch(*eni.Description)
	if len(match) > 0 {
		attachmentId := match[regex.SubexpIndex("attachmentId")]
		if attachment, ok := c.cache.Get(attachmentId); ok {
			return attachment, nil
		}
	}
	return nil, nil
}
