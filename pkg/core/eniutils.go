package core

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	"sg-ripper/pkg/core/awsClients"
	coreTypes "sg-ripper/pkg/core/types"
)

func ListNetworkInterfaces(ctx context.Context, eniIds []string, filters Filters, region string, profile string) ([]coreTypes.NetworkInterface, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region), config.WithSharedConfigProfile(profile))
	if err != nil {
		return nil, err
	}

	ec2Client := awsClients.NewAwsEc2Client(cfg)
	awsLambdaClient := awsClients.NewAwsLambdaClient(cfg)
	awsElbClient := awsClients.NewAwsElbClient(cfg)
	ecsClient := awsClients.NewAwsEcsClient(cfg)

	networkInterfaces, err := ec2Client.DescribeNetworkInterfaces(ctx, eniIds)
	if err != nil {
		return nil, err
	}

	enis := make([]coreTypes.NetworkInterface, 0)
	for _, awsEni := range networkInterfaces {
		lambdaAttachment, err := awsLambdaClient.GetLambdaAttachment(ctx, awsEni)
		if err != nil {
			return nil, err
		}

		ecsAttachment, err := ecsClient.GetECSAttachment(ctx, awsEni)
		if err != nil {
			return nil, err
		}

		elbAttachment, err := awsElbClient.GetELBAttachment(ctx, awsEni)
		if err != nil {
			return nil, err
		}

		vpceAttachment, err := ec2Client.GetVpceAttachment(ctx, awsEni)
		if err != nil {
			return nil, err
		}

		sgIdentifiers := make([]coreTypes.SecurityGroupIdentifier, 0)
		for _, group := range awsEni.Groups {
			if group.GroupId != nil {
				sgIdentifiers = append(sgIdentifiers, coreTypes.SecurityGroupIdentifier{
					Id:   *group.GroupId,
					Name: group.GroupName,
				})
			}
		}

		enis = append(enis, coreTypes.NetworkInterface{
			Id:                       *awsEni.NetworkInterfaceId,
			Description:              awsEni.Description,
			Type:                     string(awsEni.InterfaceType),
			ManagedByAWS:             *awsEni.RequesterManaged,
			Status:                   string(awsEni.Status),
			EC2Attachment:            getEC2Attachment(awsEni),
			LambdaAttachment:         lambdaAttachment,
			ECSAttachment:            ecsAttachment,
			ELBAttachment:            elbAttachment,
			VpceAttachment:           vpceAttachment,
			SecurityGroupIdentifiers: sgIdentifiers,
		})
	}

	return applyEniFilters(enis, filters), nil
}

// Apply Filters to the list of Network interface usages
func applyEniFilters(enis []coreTypes.NetworkInterface, filters Filters) []coreTypes.NetworkInterface {
	if filters.Status == All {
		return enis
	}

	var filterFn func(usage coreTypes.NetworkInterface) bool

	switch filters.Status {
	case Used:
		filterFn = func(eni coreTypes.NetworkInterface) bool {
			return eni.IsInUse()
		}
	case Unused:
		filterFn = func(eni coreTypes.NetworkInterface) bool {
			return !eni.IsInUse()
		}
	}

	result := make([]coreTypes.NetworkInterface, 0)
	for _, eni := range enis {
		if filterFn(eni) {
			result = append(result, eni)
		}
	}
	return result
}
