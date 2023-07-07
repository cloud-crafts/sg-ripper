package core

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
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
func ListSecurityGroups(securityGroupIds []string, filters Filters, region string) ([]SecurityGroupUsage, error) {
	cfg, configErr := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if configErr != nil {
		return nil, configErr
	}

	securityGroups, sgErr := describeSecurityGroups(cfg, securityGroupIds)
	if sgErr != nil {
		return nil, sgErr
	}

	networkInterfaces, ifcErr := describeNetworkInterfaces(cfg, securityGroupIds)
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
					Id:           *ifc.NetworkInterfaceId,
					Description:  *ifc.Description,
					Type:         string(ifc.InterfaceType),
					ManagedByAWS: *ifc.RequesterManaged,
					Status:       string(ifc.Status),
				}
				if ifc.Attachment != nil && ifc.Attachment.InstanceId != nil {
					nic.EC2Attachment = []EC2Attachment{
						{InstanceId: *ifc.Attachment.InstanceId},
					}
				}
				associations = append(associations, nic)
			}
		}
		if len(associatedInterfaces) > 0 {
			usage = append(usage, *NewSecurityGroupUsage(*sg.GroupName, *sg.GroupId, *sg.Description, associations, *sg.VpcId))
		} else {
			usage = append(usage, *NewSecurityGroupUsage(*sg.GroupName, *sg.GroupId, *sg.Description, nil, *sg.VpcId))
		}
	}

	return applyFilters(usage, filters), nil
}

func describeNetworkInterfaces(cfg aws.Config, securityGroupIds []string) ([]types.NetworkInterface, error) {
	svc := ec2.NewFromConfig(cfg)

	filterName := "group-id"
	var filters []types.Filter
	if len(securityGroupIds) > 0 {
		filters = append(filters, types.Filter{Name: &filterName, Values: securityGroupIds})
	}

	var nextToken *string = nil
	networkInterfaces := make([]types.NetworkInterface, 0)
	for {
		ifcResponse, err := svc.DescribeNetworkInterfaces(context.TODO(), &ec2.DescribeNetworkInterfacesInput{NextToken: nextToken})
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

func describeSecurityGroups(cfg aws.Config, securityGroupIds []string) ([]types.SecurityGroup, error) {
	svc := ec2.NewFromConfig(cfg)

	filterName := "group-id"
	var filters []types.Filter
	if len(securityGroupIds) > 0 {
		filters = append(filters, types.Filter{Name: &filterName, Values: securityGroupIds})
	}

	var nextToken *string = nil
	securityGroups := make([]types.SecurityGroup, 0)
	for {
		sgResponse, err := svc.DescribeSecurityGroups(context.TODO(), &ec2.DescribeSecurityGroupsInput{NextToken: nextToken, Filters: filters})
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
