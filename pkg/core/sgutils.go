package core

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/cloud-crafts/sg-ripper/pkg/core/builders"
	"github.com/cloud-crafts/sg-ripper/pkg/core/clients"
	coreTypes "github.com/cloud-crafts/sg-ripper/pkg/core/types"
	"github.com/cloud-crafts/sg-ripper/pkg/core/utils"
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

// ListSecurityGroups returns a slice of SecurityGroupDetails based on the input Security Group ID list and filters.
// If the slice with the IDs is empty, all the security groups will be retrieved
func ListSecurityGroups(ctx context.Context, securityGroupIds []string, filters Filters, region string, profile string) ([]coreTypes.SecurityGroupDetails, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region), config.WithSharedConfigProfile(profile))
	if err != nil {
		return nil, err
	}

	ec2Client := clients.NewAwsEc2Client(cfg)

	securityGroupRules, err := ec2Client.DescribeSecurityGroupRules(ctx)
	if err != nil {
		return nil, err
	}

	networkInterfaces, err := ec2Client.DescribeNetworkInterfacesBySecurityGroups(ctx, securityGroupIds)
	if err != nil {
		return nil, err
	}

	sgResultCh := make(chan utils.Result[[]ec2Types.SecurityGroup])
	ec2Client.DescribeSecurityGroups(ctx, securityGroupIds, sgResultCh)

	eniDetailsBuilder := builders.NewEniBuilder(cfg)

	groups := make([]coreTypes.SecurityGroupDetails, 0)
	for sgResult := range sgResultCh {
		if sgResult.Err != nil {
			return nil, sgResult.Err
		}
		for _, sg := range sgResult.Data {
			associatedInterfaces := getAssociatedNetworkInterfaces(sg, networkInterfaces)

			enis, err := eniDetailsBuilder.FromRemoteInterfaces(ctx, associatedInterfaces)
			if err != nil {
				return nil, err
			}

			groups = append(groups,
				*coreTypes.NewSecurityGroup(*sg.GroupName, *sg.GroupId, *sg.Description, enis,
					getRuleReferences(sg, securityGroupRules), *sg.VpcId))
		}
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
func applyFilters(groups []coreTypes.SecurityGroupDetails, filters Filters) []coreTypes.SecurityGroupDetails {
	if filters.Status == All {
		return groups
	}

	var filterFn func(sg coreTypes.SecurityGroupDetails) bool

	switch filters.Status {
	case Used:
		filterFn = func(sg coreTypes.SecurityGroupDetails) bool {
			return sg.IsInUse()
		}
	case Unused:
		filterFn = func(sg coreTypes.SecurityGroupDetails) bool {
			return !sg.IsInUse()
		}
	}

	filteredGroups := make([]coreTypes.SecurityGroupDetails, 0)
	for _, sg := range groups {
		if filterFn(sg) {
			filteredGroups = append(filteredGroups, sg)
		}
	}
	return filteredGroups
}

// RemoveSecurityGroupsAsync removes Security Groups based on the input list provided. This function expects a result
// channel for being able to provide removal information for the caller
func RemoveSecurityGroupsAsync(ctx context.Context, securityGroupIds []string, region string, profile string,
	resultCh chan utils.Result[string]) error {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region), config.WithSharedConfigProfile(profile))
	if err != nil {
		return err
	}

	ec2Client := clients.NewAwsEc2Client(cfg)

	ec2Client.TryRemoveAllSecurityGroups(ctx, securityGroupIds, resultCh)

	return nil
}

// RemoveENIAsync removes Elastic Network Interfaces based on the input list provided. This function expects a result
// channel for being able to provide removal information for the caller
func RemoveENIAsync(ctx context.Context, eniIds []string, region string, profile string,
	resultCh chan utils.Result[string]) error {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region), config.WithSharedConfigProfile(profile))
	if err != nil {
		return err
	}

	ec2Client := clients.NewAwsEc2Client(cfg)

	ec2Client.TryRemoveAllENIs(ctx, eniIds, resultCh)

	return nil
}
