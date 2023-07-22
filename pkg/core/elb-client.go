package core

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"regexp"
)

type awsElbClient struct {
	client *elasticloadbalancingv2.Client
	cache  map[string]*ELBAttachment
}

func newAwsElbClient(cfg aws.Config) *awsElbClient {
	return &awsElbClient{
		client: elasticloadbalancingv2.NewFromConfig(cfg),
		cache:  make(map[string]*ELBAttachment),
	}
}

func (c *awsElbClient) getELBAttachment(eni ec2Types.NetworkInterface) (*ELBAttachment, error) {
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
				attachment := ELBAttachment{
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
