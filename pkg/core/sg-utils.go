package core

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

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
func ListSecurityGroups(securityGroupIds []string, filters Filters, region string, profile string) ([]SecurityGroupUsage, error) {
	cfg, configErr := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region), config.WithSharedConfigProfile(profile))
	if configErr != nil {
		return nil, configErr
	}
	client := ec2.NewFromConfig(cfg)

	securityGroups, sgErr := describeSecurityGroups(client, securityGroupIds)
	if sgErr != nil {
		return nil, sgErr
	}

	securityGroupRules, sgRuleErr := describeSecurityGroupRules(client)
	if sgRuleErr != nil {
		return nil, sgRuleErr
	}

	networkInterfaces, ifcErr := describeNetworkInterfaces(client, securityGroupIds)
	if ifcErr != nil {
		return nil, ifcErr
	}

	usage := make([]SecurityGroupUsage, 0)
	for _, sg := range securityGroups {
		associatedInterfaces := getAssociatedNetworkInterfaces(sg, networkInterfaces)
		associations := make([]NetworkInterface, 0)
		for _, ifc := range associatedInterfaces {
			if ifc.NetworkInterfaceId != nil {
				nic := NetworkInterface{
					Id:            *ifc.NetworkInterfaceId,
					Description:   *ifc.Description,
					Type:          string(ifc.InterfaceType),
					ManagedByAWS:  *ifc.RequesterManaged,
					Status:        string(ifc.Status),
					EC2Attachment: getEC2Attachments(ifc),
				}
				associations = append(associations, nic)
			}
		}
		usage = append(usage, *NewSecurityGroupUsage(*sg.GroupName, *sg.GroupId, *sg.Description, associations,
			getSecurityGroupRuleReferences(sg, securityGroupRules), *sg.VpcId))
	}

	return applyFilters(usage, filters), nil
}

func describeNetworkInterfaces(client *ec2.Client, securityGroupIds []string) ([]types.NetworkInterface, error) {
	filterName := "group-id"
	var filters []types.Filter
	if len(securityGroupIds) > 0 {
		filters = append(filters, types.Filter{Name: &filterName, Values: securityGroupIds})
	}

	var nextToken *string = nil
	networkInterfaces := make([]types.NetworkInterface, 0)
	for {
		ifcResponse, err := client.DescribeNetworkInterfaces(context.TODO(), &ec2.DescribeNetworkInterfacesInput{NextToken: nextToken})
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

func describeSecurityGroups(client *ec2.Client, securityGroupIds []string) ([]types.SecurityGroup, error) {
	filterName := "group-id"
	var filters []types.Filter
	if len(securityGroupIds) > 0 {
		filters = append(filters, types.Filter{Name: &filterName, Values: securityGroupIds})
	}

	var nextToken *string = nil
	securityGroups := make([]types.SecurityGroup, 0)
	for {
		sgResponse, err := client.DescribeSecurityGroups(context.TODO(), &ec2.DescribeSecurityGroupsInput{NextToken: nextToken, Filters: filters})
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

func describeSecurityGroupRules(client *ec2.Client) ([]types.SecurityGroupRule, error) {
	var nextToken *string = nil
	securityGroupRules := make([]types.SecurityGroupRule, 0)
	for {
		sgResponse, err := client.DescribeSecurityGroupRules(context.TODO(),
			&ec2.DescribeSecurityGroupRulesInput{NextToken: nextToken})
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

func getAssociatedNetworkInterfaces(sg types.SecurityGroup, networkInterfaces []types.NetworkInterface) []types.NetworkInterface {
	associatedInterfaces := make([]types.NetworkInterface, 0)
	for _, ifc := range networkInterfaces {
		for _, associatedSg := range ifc.Groups {
			if *sg.GroupId == *associatedSg.GroupId {
				associatedInterfaces = append(associatedInterfaces, ifc)
			}
		}
	}
	return associatedInterfaces
}

func getEC2Attachments(ifc types.NetworkInterface) []EC2Attachment {
	if ifc.Attachment != nil && ifc.Attachment.InstanceId != nil {
		return []EC2Attachment{
			{InstanceId: *ifc.Attachment.InstanceId},
		}
	}
	return nil
}

func getSecurityGroupRuleReferences(sg types.SecurityGroup, securityGroupRules []types.SecurityGroupRule) []string {
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

func applyFilters(usages []SecurityGroupUsage, filters Filters) []SecurityGroupUsage {
	if filters.Status == All {
		return usages
	}

	var filterFn func(usage SecurityGroupUsage) bool

	switch filters.Status {
	case Used:
		filterFn = func(usage SecurityGroupUsage) bool {
			return len(usage.UsedBy) > 0
		}
	case Unused:
		filterFn = func(usage SecurityGroupUsage) bool {
			return len(usage.UsedBy) <= 0
		}
	}

	result := make([]SecurityGroupUsage, 0)
	for _, usage := range usages {
		if filterFn(usage) {
			result = append(result, usage)
		}
	}
	return result
}
