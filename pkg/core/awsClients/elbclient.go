package awsClients

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"regexp"
	coreTypes "sg-ripper/pkg/core/types"
)

type AwsElbClient struct {
	client *elasticloadbalancingv2.Client
	cache  map[string]*coreTypes.ElbAttachment
}

func NewAwsElbClient(cfg aws.Config) *AwsElbClient {
	return &AwsElbClient{
		client: elasticloadbalancingv2.NewFromConfig(cfg),
		cache:  make(map[string]*coreTypes.ElbAttachment),
	}
}

// GetELBAttachment returns a pointer to an ElbAttachment for the network interface. If there is no attachment found,
// the returned value is a nil.
func (c *AwsElbClient) GetELBAttachment(ctx context.Context, eni ec2Types.NetworkInterface, resultCh chan Result[*coreTypes.ElbAttachment]) {
	go func() {
		defer close(resultCh)
		regex := regexp.MustCompile("ELB app/(?P<elbName>.+)/(?P<elbId>([a-z]|[0-9])+)")
		if eni.InterfaceType == ec2Types.NetworkInterfaceTypeInterface && eni.Description != nil {
			match := regex.FindStringSubmatch(*eni.Description)
			if len(match) > 0 {
				elbName := match[regex.SubexpIndex("elbName")]

				if cachedElb, ok := c.cache[elbName]; ok {
					resultCh <- Result[*coreTypes.ElbAttachment]{
						Data: cachedElb,
					}
					return
				}

				loadBalancers, err := c.client.DescribeLoadBalancers(ctx,
					&elasticloadbalancingv2.DescribeLoadBalancersInput{Names: []string{elbName}})
				if err != nil {
					resultCh <- Result[*coreTypes.ElbAttachment]{
						Err: err,
					}
					return
				}

				for _, elb := range loadBalancers.LoadBalancers {
					attachment := coreTypes.ElbAttachment{
						IsRemoved: elb.LoadBalancerArn == nil,
						Name:      elbName,
						Arn:       elb.LoadBalancerArn,
					}
					c.cache[elbName] = &attachment

					// It is expected that we will have only one load balancer as a result
					resultCh <- Result[*coreTypes.ElbAttachment]{
						Data: &attachment,
					}
					return
				}
			}
		}
	}()
}
