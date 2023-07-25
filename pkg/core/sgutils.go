package core

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"sg-ripper/pkg/core/awsClients"
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
func ListSecurityGroups(securityGroupIds []string, filters Filters, region string, profile string) ([]coreTypes.SecurityGroup, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region), config.WithSharedConfigProfile(profile))
	if err != nil {
		return nil, err
	}

	ec2Client := awsClients.NewAwsEc2Client(cfg)

	securityGroups, err := ec2Client.DescribeSecurityGroups(securityGroupIds)
	if err != nil {
		return nil, err
	}

	securityGroupRules, err := ec2Client.DescribeSecurityGroupRules()
	if err != nil {
		return nil, err
	}

	networkInterfaces, err := ec2Client.DescribeNetworkInterfacesUsedBySecurityGroups(securityGroupIds)
	if err != nil {
		return nil, err
	}

	awsLambdaClient := awsClients.NewAwsLambdaClient(cfg)
	awsElbClient := awsClients.NewAwsElbClient(cfg)
	ecsClient := awsClients.NewAwsEcsClient(cfg)

	nicCache := make(map[string]*coreTypes.NetworkInterface)

	usage := make([]coreTypes.SecurityGroup, 0)
	for _, sg := range securityGroups {
		associatedInterfaces := getAssociatedNetworkInterfaces(sg, networkInterfaces)
		associations := make([]*coreTypes.NetworkInterface, 0)
		for _, eni := range associatedInterfaces {
			if eni.NetworkInterfaceId != nil {

				// Check if Network Interface is already in the cache to avoid computing multiple times which resources
				// are using it
				if cachedNic, ok := nicCache[*eni.NetworkInterfaceId]; ok {
					associations = append(associations, cachedNic)
				} else {
					lambdaAttachment, err := awsLambdaClient.GetLambdaAttachment(eni)
					if err != nil {
						return nil, err
					}

					ecsAttachment, err := ecsClient.GetECSAttachment(eni)
					if err != nil {
						return nil, err
					}

					elbAttachment, err := awsElbClient.GetELBAttachment(eni)
					if err != nil {
						return nil, err
					}

					vpceAttachment, err := ec2Client.GetVpceAttachment(eni)
					if err != nil {
						return nil, err
					}

					nic := coreTypes.NetworkInterface{
						Id:               *eni.NetworkInterfaceId,
						Description:      eni.Description,
						Type:             string(eni.InterfaceType),
						ManagedByAWS:     *eni.RequesterManaged,
						Status:           string(eni.Status),
						EC2Attachment:    getEC2Attachment(eni),
						LambdaAttachment: lambdaAttachment,
						ECSAttachment:    ecsAttachment,
						ELBAttachment:    elbAttachment,
						VpceAttachment:   vpceAttachment,
					}

					// Add the new interface to the cache
					nicCache[nic.Id] = &nic

					associations = append(associations, &nic)
				}
			}
		}
		usage = append(usage, *coreTypes.NewSecurityGroup(*sg.GroupName, *sg.GroupId, *sg.Description, associations,
			getRuleReferences(sg, securityGroupRules), *sg.VpcId))
	}

	return applyFilters(usage, filters), nil
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
func getEC2Attachment(ifc ec2Types.NetworkInterface) *coreTypes.Ec2Attachment {
	if ifc.Attachment != nil && ifc.Attachment.InstanceId != nil {
		return &coreTypes.Ec2Attachment{InstanceId: *ifc.Attachment.InstanceId}
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
func applyFilters(groups []coreTypes.SecurityGroup, filters Filters) []coreTypes.SecurityGroup {
	if filters.Status == All {
		return groups
	}

	var filterFn func(usage coreTypes.SecurityGroup) bool

	switch filters.Status {
	case Used:
		filterFn = func(sg coreTypes.SecurityGroup) bool {
			return sg.IsInUse()
		}
	case Unused:
		filterFn = func(sg coreTypes.SecurityGroup) bool {
			return !sg.IsInUse()
		}
	}

	result := make([]coreTypes.SecurityGroup, 0)
	for _, sg := range groups {
		if filterFn(sg) {
			result = append(result, sg)
		}
	}
	return result
}
