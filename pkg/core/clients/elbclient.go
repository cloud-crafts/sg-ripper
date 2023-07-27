package clients

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	cmap "github.com/orcaman/concurrent-map/v2"
	"regexp"
	coreTypes "sg-ripper/pkg/core/types"
)

type AwsElbClient struct {
	client *elasticloadbalancingv2.Client
	cache  cmap.ConcurrentMap[string, *coreTypes.ElbAttachment]
}

func NewAwsElbClient(cfg aws.Config) *AwsElbClient {
	return &AwsElbClient{
		client: elasticloadbalancingv2.NewFromConfig(cfg),
		cache:  cmap.New[*coreTypes.ElbAttachment](),
	}
}

// GetELBAttachment returns a pointer to an ElbAttachment for the network interface. If there is no attachment found,
// the returned value is a nil.
func (c *AwsElbClient) GetELBAttachment(ctx context.Context, eni ec2Types.NetworkInterface) (*coreTypes.ElbAttachment, error) {
	regex := regexp.MustCompile("ELB app/(?P<elbName>.+)/(?P<elbId>([a-z]|[0-9])+)")
	if eni.InterfaceType == ec2Types.NetworkInterfaceTypeInterface && eni.Description != nil {
		match := regex.FindStringSubmatch(*eni.Description)
		if len(match) > 0 {
			elbName := match[regex.SubexpIndex("elbName")]

			if cachedElb, ok := c.cache.Get(elbName); ok {
				return cachedElb, nil
			}

			loadBalancers, err := c.client.DescribeLoadBalancers(ctx,
				&elasticloadbalancingv2.DescribeLoadBalancersInput{Names: []string{elbName}})
			if err != nil {
				return nil, err
			}

			for _, elb := range loadBalancers.LoadBalancers {
				attachment := &coreTypes.ElbAttachment{
					IsRemoved: elb.LoadBalancerArn == nil,
					Name:      elbName,
					Arn:       elb.LoadBalancerArn,
				}

				c.cache.Set(elbName, attachment)

				// It is expected that we will have only one load balancer as a result
				return attachment, nil
			}
		}
	}
	return nil, nil
}
