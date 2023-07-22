package core

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"sg-ripper/pkg/core/awsClients"
	"sg-ripper/pkg/core/types"
)

const MaxResults = 1000

const (
	All SecurityGroupStatus = iota
	Used
	Unused
)

type SecurityGroupStatus int

type Filters struct {
	Status SecurityGroupStatus
}

// ListSecurityGroups lists the usage of Security Groups of whose IDs are provided in the securityGroupIds slice.
// If the slice is empty, all the security groups will be retrieved. Furthermore, we can apply filters to retrieved
// Security Groups, for example: we can grab only the Security Groups which are in use or just unused ones.
func ListSecurityGroups(securityGroupIds []string, filters Filters, region string, profile string) ([]types.SecurityGroup, error) {
	cfg, configErr := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region), config.WithSharedConfigProfile(profile))
	if configErr != nil {
		return nil, configErr
	}

	ec2Client := ec2.NewFromConfig(cfg)

	awsLambdaClient := awsClients.NewAwsLambdaClient(cfg)
	awsElbClient := awsClients.NewAwsElbClient(cfg)
	ecsClient := awsClients.NewAwsEcsClient(cfg)

	securityGroups, sgErr := describeSecurityGroups(ec2Client, securityGroupIds)
	if sgErr != nil {
		return nil, sgErr
	}

	securityGroupRules, sgRuleErr := describeSecurityGroupRules(ec2Client)
	if sgRuleErr != nil {
		return nil, sgRuleErr
	}

	networkInterfaces, ifcErr := describeNetworkInterfaces(ec2Client, securityGroupIds)
	if ifcErr != nil {
		return nil, ifcErr
	}

	nicCache := make(map[string]*types.NetworkInterface)

	usage := make([]types.SecurityGroup, 0)
	for _, sg := range securityGroups {
		associatedInterfaces := getAssociatedNetworkInterfaces(sg, networkInterfaces)
		associations := make([]*types.NetworkInterface, 0)
		for _, ifc := range associatedInterfaces {
			if ifc.NetworkInterfaceId != nil {

				// Check if Network Interface is already in the cache to avoid computing multiple times which resources
				// are using it
				if cachedNic, ok := nicCache[*ifc.NetworkInterfaceId]; ok {
					associations = append(associations, cachedNic)
				} else {
					lambdaAttachment, err := awsLambdaClient.GetLambdaAttachment(ifc)
					if err != nil {
						return nil, err
					}

					ecsAttachment, err := ecsClient.GetECSAttachment(ifc)
					if err != nil {
						return nil, err
					}

					elbAttachment, err := awsElbClient.GetELBAttachment(ifc)
					if err != nil {
						return nil, err
					}

					nic := types.NetworkInterface{
						Id:               *ifc.NetworkInterfaceId,
						Description:      ifc.Description,
						Type:             string(ifc.InterfaceType),
						ManagedByAWS:     *ifc.RequesterManaged,
						Status:           string(ifc.Status),
						EC2Attachment:    getEC2Attachment(ifc),
						LambdaAttachment: lambdaAttachment,
						ECSAttachment:    ecsAttachment,
						ELBAttachment:    elbAttachment,
					}

					// Add the new interface to the cache
					nicCache[nic.Id] = &nic

					associations = append(associations, &nic)
				}
			}
		}
		usage = append(usage, *types.NewSecurityGroup(*sg.GroupName, *sg.GroupId, *sg.Description, associations,
			getRuleReferences(sg, securityGroupRules), *sg.VpcId))
	}

	return applyFilters(usage, filters), nil
}

// Get a list of Network Interfaces used by the security groups from the input slice
func describeNetworkInterfaces(client *ec2.Client, securityGroupIds []string) ([]ec2Types.NetworkInterface, error) {
	filterName := "group-id"
	var filters []ec2Types.Filter
	if len(securityGroupIds) > 0 {
		filters = append(filters, ec2Types.Filter{Name: &filterName, Values: securityGroupIds})
	}

	var nextToken *string = nil
	networkInterfaces := make([]ec2Types.NetworkInterface, 0)
	for {
		ifcResponse, err := client.DescribeNetworkInterfaces(context.TODO(),
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

// Get a list of Security Groups based on the list of Security Group IDs provided as an input
func describeSecurityGroups(client *ec2.Client, securityGroupIds []string) ([]ec2Types.SecurityGroup, error) {
	filterName := "group-id"
	var filters []ec2Types.Filter
	if len(securityGroupIds) > 0 {
		filters = append(filters, ec2Types.Filter{Name: &filterName, Values: securityGroupIds})
	}

	var nextToken *string = nil
	securityGroups := make([]ec2Types.SecurityGroup, 0)
	for {
		sgResponse, err := client.DescribeSecurityGroups(context.TODO(),
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

// Get all the Security Group Rules. (TODO: try to optimise this to grab a sublist only)
func describeSecurityGroupRules(client *ec2.Client) ([]ec2Types.SecurityGroupRule, error) {
	var nextToken *string = nil
	securityGroupRules := make([]ec2Types.SecurityGroupRule, 0)
	for {
		sgResponse, err := client.DescribeSecurityGroupRules(context.TODO(),
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

// Get all the Network Interfaces which are associated to one of the Security Groups from the input list
func getAssociatedNetworkInterfaces(sg ec2Types.SecurityGroup, networkInterfaces []ec2Types.NetworkInterface) []ec2Types.NetworkInterface {
	associatedInterfaces := make([]ec2Types.NetworkInterface, 0)
	for _, ifc := range networkInterfaces {
		for _, associatedSg := range ifc.Groups {
			if *sg.GroupId == *associatedSg.GroupId {
				associatedInterfaces = append(associatedInterfaces, ifc)
			}
		}
	}
	return associatedInterfaces
}

// Get the IDs of the EC2 instances attached to the Network Interface
func getEC2Attachment(ifc ec2Types.NetworkInterface) *types.EC2Attachment {
	if ifc.Attachment != nil && ifc.Attachment.InstanceId != nil {
		return &types.EC2Attachment{InstanceId: *ifc.Attachment.InstanceId}
	}
	return nil
}

// Get the Security Group Rules which are referencing the Security Group
func getRuleReferences(sg ec2Types.SecurityGroup, securityGroupRules []ec2Types.SecurityGroupRule) []string {
	sgIds := make([]string, 0)
	for _, rule := range securityGroupRules {
		if rule.ReferencedGroupInfo == nil || rule.ReferencedGroupInfo.GroupId == nil {
			continue
		}
		if *sg.GroupId == *rule.ReferencedGroupInfo.GroupId {
			sgIds = append(sgIds, *rule.GroupId)
		}
	}
	return sgIds
}

// Apply Filters to the list of Security Group usages
func applyFilters(usages []types.SecurityGroup, filters Filters) []types.SecurityGroup {
	if filters.Status == All {
		return usages
	}

	var filterFn func(usage types.SecurityGroup) bool

	switch filters.Status {
	case Used:
		filterFn = func(usage types.SecurityGroup) bool {
			return len(usage.UsedBy) > 0
		}
	case Unused:
		filterFn = func(usage types.SecurityGroup) bool {
			return len(usage.UsedBy) <= 0
		}
	}

	result := make([]types.SecurityGroup, 0)
	for _, usage := range usages {
		if filterFn(usage) {
			result = append(result, usage)
		}
	}
	return result
}
