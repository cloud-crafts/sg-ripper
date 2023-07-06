package core

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type SecurityGroupUsage struct {
	SecurityGroupName        string
	SecurityGroupId          string
	SecurityGroupDescription string
	Default                  bool
	UsedBy                   []NetworkInterface
	VpcId                    string
}

func NewSecurityGroupUsage(securityGroupName string, securityGroupId string, securityGroupDescription string,
	usedBy []NetworkInterface, vpcId string) *SecurityGroupUsage {
	return &SecurityGroupUsage{
		SecurityGroupName:        securityGroupName,
		SecurityGroupId:          securityGroupId,
		SecurityGroupDescription: securityGroupDescription,
		UsedBy:                   usedBy,
		VpcId:                    vpcId,
		Default:                  securityGroupName == "default",
	}
}

type NetworkInterface struct {
	Id               string
	Description      string
	Type             string
	ManagedByAWS     bool
	Status           string
	EC2Attachment    []EC2Attachment
	LambdaAttachment []string
	ECSAttachment    []string
}

type EC2Attachment struct {
	InstanceId string
}

type LambdaAttachment struct {
	Arn  string
	Name string
}

type ECSAttachment struct {
	ServiceName string
}

func ListAllSecurityGroups(region string) ([]SecurityGroupUsage, error) {
	cfg, configErr := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if configErr != nil {
		return nil, configErr
	}

	securityGroups, sgErr := describeSecurityGroups(cfg)
	if sgErr != nil {
		return nil, sgErr
	}

	networkInterfaces, ifcErr := describeNetworkInterfaces(cfg)
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

	return usage, nil
}

func describeNetworkInterfaces(cfg aws.Config) ([]types.NetworkInterface, error) {
	svc := ec2.NewFromConfig(cfg)

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

func describeSecurityGroups(cfg aws.Config) ([]types.SecurityGroup, error) {
	svc := ec2.NewFromConfig(cfg)

	var nextToken *string = nil
	securityGroups := make([]types.SecurityGroup, 0)
	for {
		sgResponse, err := svc.DescribeSecurityGroups(context.TODO(), &ec2.DescribeSecurityGroupsInput{NextToken: nextToken})
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
