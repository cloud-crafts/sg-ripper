package clients

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	cmap "github.com/orcaman/concurrent-map/v2"
	"regexp"
	coreTypes "sg-ripper/pkg/core/types"
	"sg-ripper/pkg/core/utils"
)

const MaxResults = 1000

type AwsEc2Client struct {
	client    *ec2.Client
	vpceCache cmap.ConcurrentMap[string, *coreTypes.VpceAttachment]
}

func NewAwsEc2Client(cfg aws.Config) *AwsEc2Client {
	return &AwsEc2Client{
		client:    ec2.NewFromConfig(cfg),
		vpceCache: cmap.New[*coreTypes.VpceAttachment](),
	}
}

// DescribeSecurityGroups fetches all the Security Groups based on the list of the IDs provided. If the list  is empty,
// all the existing interfaces will be returned.
// This function expects a channel to which the response will be provided asynchronously
func (c *AwsEc2Client) DescribeSecurityGroups(ctx context.Context, securityGroupIds []string,
	resultCh chan utils.Result[[]ec2Types.SecurityGroup]) {
	go func() {
		defer close(resultCh)

		var nextToken *string = nil
		securityGroups := make([]ec2Types.SecurityGroup, 0)
		for {
			sgResponse, err := c.client.DescribeSecurityGroups(ctx,
				&ec2.DescribeSecurityGroupsInput{
					NextToken: nextToken,
					GroupIds:  securityGroupIds,
				})
			if err != nil {
				resultCh <- utils.Result[[]ec2Types.SecurityGroup]{
					Err: err,
				}
				return
			}
			nextToken = sgResponse.NextToken
			securityGroups = append(securityGroups, sgResponse.SecurityGroups...)

			if nextToken == nil {
				resultCh <- utils.Result[[]ec2Types.SecurityGroup]{
					Data: securityGroups,
				}
				return
			}
		}
	}()
}

// DescribeSecurityGroupRules returns all the Security Group Rules. (TODO: try to optimise this to grab a sublist only)
func (c *AwsEc2Client) DescribeSecurityGroupRules(ctx context.Context) ([]ec2Types.SecurityGroupRule, error) {
	var nextToken *string = nil
	securityGroupRules := make([]ec2Types.SecurityGroupRule, 0)
	for {
		sgResponse, err := c.client.DescribeSecurityGroupRules(ctx,
			&ec2.DescribeSecurityGroupRulesInput{NextToken: nextToken, MaxResults: aws.Int32(int32(MaxResults))})
		if err != nil {
			return nil, err
		}
		nextToken = sgResponse.NextToken
		securityGroupRules = append(securityGroupRules, sgResponse.SecurityGroupRules...)

		if nextToken == nil {
			break
		}
	}

	return securityGroupRules, nil
}

// DescribeNetworkInterfaces fetches all the Network Interfaces based on the list of the ENI IDs provided. If the list
// is empty, all the existing interfaces will be returned.
// This function expects a channel to which the response will be provided asynchronously
func (c *AwsEc2Client) DescribeNetworkInterfaces(ctx context.Context, eniIds []string, resultCh chan utils.Result[[]ec2Types.NetworkInterface]) {
	go func() {
		defer close(resultCh)
		var nextToken *string = nil
		for {
			ifcResponse, err := c.client.DescribeNetworkInterfaces(ctx,
				&ec2.DescribeNetworkInterfacesInput{NextToken: nextToken, NetworkInterfaceIds: eniIds})
			if err != nil {
				resultCh <- utils.Result[[]ec2Types.NetworkInterface]{
					Err: err,
				}
				return
			}

			resultCh <- utils.Result[[]ec2Types.NetworkInterface]{
				Data: ifcResponse.NetworkInterfaces,
			}
			nextToken = ifcResponse.NextToken

			if nextToken == nil {
				return
			}
		}
	}()
}

// DescribeNetworkInterfacesBySecurityGroups returns a list of Network Interfaces used by the security groups from the input slice
func (c *AwsEc2Client) DescribeNetworkInterfacesBySecurityGroups(ctx context.Context, securityGroupIds []string) ([]ec2Types.NetworkInterface, error) {
	filterName := "group-id"
	var filters []ec2Types.Filter
	if len(securityGroupIds) > 0 {
		filters = append(filters, ec2Types.Filter{Name: &filterName, Values: securityGroupIds})
	}

	var nextToken *string = nil
	networkInterfaces := make([]ec2Types.NetworkInterface, 0)
	for {
		ifcResponse, err := c.client.DescribeNetworkInterfaces(ctx,
			&ec2.DescribeNetworkInterfacesInput{NextToken: nextToken, MaxResults: aws.Int32(int32(MaxResults))})
		if err != nil {
			return nil, err
		}

		networkInterfaces = append(networkInterfaces, ifcResponse.NetworkInterfaces...)
		nextToken = ifcResponse.NextToken

		if nextToken == nil {
			break
		}
	}
	return networkInterfaces, nil
}

// GetVpceAttachment returns a pointer to a VPCEAttachment for the network interface. If there is no attachment found,
// the returned value is a nil.
func (c *AwsEc2Client) GetVpceAttachment(ctx context.Context, eni ec2Types.NetworkInterface) (*coreTypes.VpceAttachment, error) {
	regex := regexp.MustCompile("VPC Endpoint Interface (?P<vpceId>vpce-([a-z]|[0-9])+)")
	if eni.InterfaceType == ec2Types.NetworkInterfaceTypeVpcEndpoint && eni.Description != nil {
		match := regex.FindStringSubmatch(*eni.Description)

		if len(match) > 0 {
			vpceId := match[regex.SubexpIndex("vpceId")]

			if cachedVpce, ok := c.vpceCache.Get(vpceId); ok {
				return cachedVpce, nil
			}

			vpceResponse, err := c.client.DescribeVpcEndpoints(ctx, &ec2.DescribeVpcEndpointsInput{
				Filters: []ec2Types.Filter{{
					Name:   aws.String("vpc-endpoint-id"),
					Values: []string{vpceId},
				}},
			})
			if err != nil {
				return nil, err
			}

			for _, vpce := range vpceResponse.VpcEndpoints {
				attachment := &coreTypes.VpceAttachment{
					IsRemoved:   vpce.VpcEndpointId == nil,
					Id:          vpce.VpcEndpointId,
					ServiceName: vpce.ServiceName,
				}

				c.vpceCache.Set(*vpce.VpcEndpointId, attachment)

				// It is expected that we will have only one load balancer as a result
				return attachment, nil
			}
		}
	}
	return nil, nil
}

// TryRemoveAllSecurityGroups attempts to remove all the Security Groups from the list of IDs provided as input. If
// there is an error encountered for a removal, the function will not stop early.
func (c *AwsEc2Client) TryRemoveAllSecurityGroups(ctx context.Context, securityGroupIds []string,
	resultCh chan utils.Result[string]) {
	doneCh := make(chan struct{})

	for _, sgId := range securityGroupIds {
		sgId := sgId // capture value
		go func() {
			_, err := c.client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{GroupId: aws.String(sgId)})
			if err != nil {
				resultCh <- utils.Result[string]{
					Err: err,
				}
			} else {
				resultCh <- utils.Result[string]{
					Data: sgId,
				}
			}
			doneCh <- struct{}{}
		}()
	}

	// Wait for reach async call to finish and close the result channel
	go func() {
		for range securityGroupIds {
			<-doneCh
		}
		close(resultCh)
	}()
}

// TryRemoveAllENIs attempts to remove all the Elastic Network interfaces from the list of IDs provided as input. If
// there is an error encountered for a removal, the function will not stop early.
func (c *AwsEc2Client) TryRemoveAllENIs(ctx context.Context, eniIds []string,
	resultCh chan utils.Result[string]) {
	doneCh := make(chan struct{})

	for _, id := range eniIds {
		id := id // capture value
		go func() {
			_, err := c.client.DeleteNetworkInterface(ctx, &ec2.DeleteNetworkInterfaceInput{NetworkInterfaceId: aws.String(id)})
			if err != nil {
				resultCh <- utils.Result[string]{
					Err: err,
				}
			} else {
				resultCh <- utils.Result[string]{
					Data: id,
				}
			}
			doneCh <- struct{}{}
		}()
	}

	// Wait for reach async call to finish and close the result channel
	go func() {
		for range eniIds {
			<-doneCh
		}
		close(resultCh)
	}()
}
