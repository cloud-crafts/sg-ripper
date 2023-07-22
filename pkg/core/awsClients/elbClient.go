package awsClients

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"regexp"
	"sg-ripper/pkg/core/types"
)

type AwsElbClient struct {
	client *elasticloadbalancingv2.Client
	cache  map[string]*types.ELBAttachment
}

func NewAwsElbClient(cfg aws.Config) *AwsElbClient {
	return &AwsElbClient{
		client: elasticloadbalancingv2.NewFromConfig(cfg),
		cache:  make(map[string]*types.ELBAttachment),
	}
}

func (c *AwsElbClient) GetELBAttachment(eni ec2Types.NetworkInterface) (*types.ELBAttachment, error) {
	regex := regexp.MustCompile("ELB app/(?P<elbName>.+)/(?P<elbId>([a-z]|[0-9])+)")
	if eni.InterfaceType == ec2Types.NetworkInterfaceTypeInterface && eni.Description != nil {
		match := regex.FindStringSubmatch(*eni.Description)
		if len(match) > 0 {
			elbName := match[regex.SubexpIndex("elbName")]

			if cachedElb, ok := c.cache[elbName]; ok {
				return cachedElb, nil
			}

			loadBalancers, err := c.client.DescribeLoadBalancers(context.TODO(),
				&elasticloadbalancingv2.DescribeLoadBalancersInput{Names: []string{elbName}})
			if err != nil {
				return nil, err
			}

			for _, elb := range loadBalancers.LoadBalancers {
				attachment := types.ELBAttachment{
					IsRemoved: elb.LoadBalancerArn == nil,
					Name:      elbName,
					Arn:       elb.LoadBalancerArn,
				}
				c.cache[elbName] = &attachment

				// It is expected that we will have only one load balancer as a result
				return &attachment, nil
			}
		}
	}
	return nil, nil
}
