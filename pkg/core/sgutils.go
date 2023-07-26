package core

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"sg-ripper/pkg/core/builders"
	"sg-ripper/pkg/core/clients"
	coreTypes "sg-ripper/pkg/core/types"
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
func ListSecurityGroups(ctx context.Context, securityGroupIds []string, filters Filters, region string, profile string) ([]*coreTypes.SecurityGroupDetails, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region), config.WithSharedConfigProfile(profile))
	if err != nil {
		return nil, err
	}

	ec2Client := clients.NewAwsEc2Client(cfg)

	securityGroups, err := ec2Client.DescribeSecurityGroups(ctx, securityGroupIds)
	if err != nil {
		return nil, err
	}

	securityGroupRules, err := ec2Client.DescribeSecurityGroupRules(ctx)
	if err != nil {
		return nil, err
	}

	networkInterfaces, err := ec2Client.DescribeNetworkInterfacesBySecurityGroups(ctx, securityGroupIds)
	if err != nil {
		return nil, err
	}

	eniDetailsBuilder := builders.NewEniBuilder(cfg)

	groups := make([]*coreTypes.SecurityGroupDetails, 0)
	for _, sg := range securityGroups {
		associatedInterfaces := getAssociatedNetworkInterfaces(sg, networkInterfaces)

		enis, err := eniDetailsBuilder.FromAwsEniBatch(ctx, associatedInterfaces)
		if err != nil {
			return nil, err
		}

		groups = append(groups,
			coreTypes.NewSecurityGroup(*sg.GroupName, *sg.GroupId, *sg.Description, enis,
				getRuleReferences(sg, securityGroupRules), *sg.VpcId))
	}

	return applyFilters(groups, filters), nil
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
func applyFilters(groups []*coreTypes.SecurityGroupDetails, filters Filters) []*coreTypes.SecurityGroupDetails {
	if filters.Status == All {
		return groups
	}

	var filterFn func(sg *coreTypes.SecurityGroupDetails) bool

	switch filters.Status {
	case Used:
		filterFn = func(sg *coreTypes.SecurityGroupDetails) bool {
			return sg.IsInUse()
		}
	case Unused:
		filterFn = func(sg *coreTypes.SecurityGroupDetails) bool {
			return !sg.IsInUse()
		}
	}

	result := make([]*coreTypes.SecurityGroupDetails, 0)
	for _, sg := range groups {
		if filterFn(sg) {
			result = append(result, sg)
		}
	}
	return result
}
