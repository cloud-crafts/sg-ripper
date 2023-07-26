package builders

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"sg-ripper/pkg/core/awsClients"
	coreTypes "sg-ripper/pkg/core/types"
)

type EniBuilder struct {
	awsEc2Client    *awsClients.AwsEc2Client
	awsLambdaClient *awsClients.AwsLambdaClient
	awsElbClient    *awsClients.AwsElbClient
	awsEcsClient    *awsClients.AwsEcsClient
	cache           map[string]*coreTypes.NetworkInterfaceDetails
}

func NewEniBuilder(cfg aws.Config) *EniBuilder {
	return &EniBuilder{
		awsEc2Client:    awsClients.NewAwsEc2Client(cfg),
		awsLambdaClient: awsClients.NewAwsLambdaClient(cfg),
		awsElbClient:    awsClients.NewAwsElbClient(cfg),
		awsEcsClient:    awsClients.NewAwsEcsClient(cfg),
		cache:           make(map[string]*coreTypes.NetworkInterfaceDetails),
	}
}

func (e *EniBuilder) Build(ctx context.Context, awsEniBatch []ec2Types.NetworkInterface) ([]*coreTypes.NetworkInterfaceDetails, error) {
	enis := make([]*coreTypes.NetworkInterfaceDetails, 0)

	for _, awsEni := range awsEniBatch {
		if awsEni.NetworkInterfaceId != nil {

			// Check if Network Interface is already in the cache to avoid computing multiple times which resources
			// are using it
			if cachedEni, ok := e.cache[*awsEni.NetworkInterfaceId]; ok {
				enis = append(enis, cachedEni)
			} else {
				lambdaResultCh := make(chan awsClients.Result[*coreTypes.LambdaAttachment])
				var lambdaAttachment *coreTypes.LambdaAttachment
				e.awsLambdaClient.GetLambdaAttachment(ctx, awsEni, lambdaResultCh)

				ecsResultCh := make(chan awsClients.Result[*coreTypes.EcsAttachment])
				var ecsAttachment *coreTypes.EcsAttachment
				e.awsEcsClient.GetECSAttachment(ctx, awsEni, ecsResultCh)

				elbResultCh := make(chan awsClients.Result[*coreTypes.ElbAttachment])
				var elbAttachment *coreTypes.ElbAttachment
				e.awsElbClient.GetELBAttachment(ctx, awsEni, elbResultCh)

				vpceResultCh := make(chan awsClients.Result[*coreTypes.VpceAttachment])
				var vpceAttachment *coreTypes.VpceAttachment
				e.awsEc2Client.GetVpceAttachment(ctx, awsEni, vpceResultCh)

				for i := 0; i < 4; i++ {
					select {
					case lambdaAttachRes := <-lambdaResultCh:
						if lambdaAttachRes.Err != nil {
							return nil, lambdaAttachRes.Err
						}
						lambdaAttachment = lambdaAttachRes.Data
					case ecsAttachRes := <-ecsResultCh:
						if ecsAttachRes.Err != nil {
							return nil, ecsAttachRes.Err
						}
						ecsAttachment = ecsAttachRes.Data
					case elbAttachRes := <-elbResultCh:
						if elbAttachRes.Err != nil {
							return nil, elbAttachRes.Err
						}
						elbAttachment = elbAttachRes.Data
					case vpceAttchRes := <-vpceResultCh:
						if vpceAttchRes.Err != nil {
							return nil, vpceAttchRes.Err
						}
						vpceAttachment = vpceAttchRes.Data
					}
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

				newEni := &coreTypes.NetworkInterfaceDetails{
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
				}

				// Add the new interface to the cache
				e.cache[newEni.Id] = newEni
				enis = append(enis, newEni)
			}
		}
	}

	return enis, nil
}

// Get the IDs of the EC2 instances attached to the Network Interface
func getEC2Attachment(ifc ec2Types.NetworkInterface) *coreTypes.Ec2Attachment {
	if ifc.Attachment != nil && ifc.Attachment.InstanceId != nil {
		return &coreTypes.Ec2Attachment{InstanceId: *ifc.Attachment.InstanceId}
	}
	return nil
}
