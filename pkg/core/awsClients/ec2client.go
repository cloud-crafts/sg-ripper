package awsClients

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"regexp"
	coreTypes "sg-ripper/pkg/core/types"
)

const MaxResults = 1000

type AwsEc2Client struct {
	client    *ec2.Client
	vpceCache map[string]*coreTypes.VpceAttachment
}

func NewAwsEc2Client(cfg aws.Config) *AwsEc2Client {
	return &AwsEc2Client{
		client:    ec2.NewFromConfig(cfg),
		vpceCache: make(map[string]*coreTypes.VpceAttachment),
	}
}

// DescribeSecurityGroups returns a list of Security Groups based on the list of Security Group IDs provided as an input
func (c *AwsEc2Client) DescribeSecurityGroups(securityGroupIds []string) ([]ec2Types.SecurityGroup, error) {
	filterName := "group-id"
	var filters []ec2Types.Filter
	if len(securityGroupIds) > 0 {
		filters = append(filters, ec2Types.Filter{Name: &filterName, Values: securityGroupIds})
	}

	var nextToken *string = nil
	securityGroups := make([]ec2Types.SecurityGroup, 0)
	for {
		sgResponse, err := c.client.DescribeSecurityGroups(context.TODO(),
			&ec2.DescribeSecurityGroupsInput{
				NextToken:  nextToken,
				Filters:    filters,
				MaxResults: aws.Int32(int32(MaxResults)),
			})
		if err != nil {
			return nil, err
		}
		nextToken = sgResponse.NextToken
		securityGroups = append(securityGroups, sgResponse.SecurityGroups...)

		if nextToken == nil {
			break
		}
	}

	return securityGroups, nil
}

// DescribeSecurityGroupRules returns all the Security Group Rules. (TODO: try to optimise this to grab a sublist only)
func (c *AwsEc2Client) DescribeSecurityGroupRules() ([]ec2Types.SecurityGroupRule, error) {
	var nextToken *string = nil
	securityGroupRules := make([]ec2Types.SecurityGroupRule, 0)
	for {
		sgResponse, err := c.client.DescribeSecurityGroupRules(context.TODO(),
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

// DescribeNetworkInterfacesUsedBySecurityGroups returns a list of Network Interfaces used by the security groups from the input slice
func (c *AwsEc2Client) DescribeNetworkInterfacesUsedBySecurityGroups(securityGroupIds []string) ([]ec2Types.NetworkInterface, error) {
	filterName := "group-id"
	var filters []ec2Types.Filter
	if len(securityGroupIds) > 0 {
		filters = append(filters, ec2Types.Filter{Name: &filterName, Values: securityGroupIds})
	}

	var nextToken *string = nil
	networkInterfaces := make([]ec2Types.NetworkInterface, 0)
	for {
		ifcResponse, err := c.client.DescribeNetworkInterfaces(context.TODO(),
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

// GetVpceAttachment returns a pointer to a VpceAttachment for the network interface. If there is no attachment found,
// the returned value is a nil.
func (c *AwsEc2Client) GetVpceAttachment(eni ec2Types.NetworkInterface) (*coreTypes.VpceAttachment, error) {
	regex := regexp.MustCompile("VPC Endpoint Interface (?P<vpceId>vpce-([a-z]|[0-9])+)")
	if eni.InterfaceType == ec2Types.NetworkInterfaceTypeVpcEndpoint && eni.Description != nil {
		match := regex.FindStringSubmatch(*eni.Description)

		if len(match) > 0 {
			vpceId := match[regex.SubexpIndex("vpceId")]

			if cachedVpce, ok := c.vpceCache[vpceId]; ok {
				return cachedVpce, nil
			}

			vpceResponse, err := c.client.DescribeVpcEndpoints(context.TODO(), &ec2.DescribeVpcEndpointsInput{
				Filters: []ec2Types.Filter{{
					Name:   aws.String("vpc-endpoint-id"),
					Values: []string{vpceId},
				}},
			})
			if err != nil {
				return nil, err
			}

			for _, vpce := range vpceResponse.VpcEndpoints {
				attachment := coreTypes.VpceAttachment{
					IsRemoved:   vpce.VpcEndpointId == nil,
					Id:          vpce.VpcEndpointId,
					ServiceName: vpce.ServiceName,
				}

				c.vpceCache[*vpce.VpcEndpointId] = &attachment

				// It is expected that we will have only one load balancer as a result
				return &attachment, nil
			}
		}
	}
	return nil, nil
}

func (c *AwsEc2Client) DescribeNetworkInterfaces(ctx context.Context, ids []string) ([]ec2Types.NetworkInterface, error) {
	filterName := "eni-id"
	var filters []ec2Types.Filter
	if len(ids) > 0 {
		filters = append(filters, ec2Types.Filter{Name: &filterName, Values: ids})
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
