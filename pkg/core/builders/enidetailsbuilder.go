package builders

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	cmap "github.com/orcaman/concurrent-map/v2"
	"sg-ripper/pkg/core/clients"
	coreTypes "sg-ripper/pkg/core/types"
)

type EniDetailsBuilder struct {
	awsEc2Client    *clients.AwsEc2Client
	awsLambdaClient *clients.AwsLambdaClient
	awsElbClient    *clients.AwsElbClient
	awsEcsClient    *clients.AwsEcsClient
	cache           cmap.ConcurrentMap[string, *coreTypes.NetworkInterfaceDetails]
}

func NewEniBuilder(cfg aws.Config) *EniDetailsBuilder {
	return &EniDetailsBuilder{
		awsEc2Client:    clients.NewAwsEc2Client(cfg),
		awsLambdaClient: clients.NewAwsLambdaClient(cfg),
		awsElbClient:    clients.NewAwsElbClient(cfg),
		awsEcsClient:    clients.NewAwsEcsClient(cfg),
		cache:           cmap.New[*coreTypes.NetworkInterfaceDetails](),
	}
}

func (e *EniDetailsBuilder) FromAwsEniBatch(ctx context.Context, awsEniBatch []ec2Types.NetworkInterface) ([]*coreTypes.NetworkInterfaceDetails, error) {
	enis := make([]*coreTypes.NetworkInterfaceDetails, 0)

	for _, awsEni := range awsEniBatch {
		if awsEni.NetworkInterfaceId != nil {

			// Check if Network Interface is already in the cache to avoid computing multiple times which resources
			// are using it
			if cachedEni, ok := e.cache.Get(*awsEni.NetworkInterfaceId); ok {
				enis = append(enis, cachedEni)
			} else {
				resultCh := make(chan clients.Result[any])

				asyncFetchers := []func(context.Context, ec2Types.NetworkInterface, chan clients.Result[any]){
					e.getLambdaAttachmentAsync, e.getEcsAttachmentAsync, e.getElbAttachmentAsync, e.getVpcAttachmentAsync,
				}

				for _, fn := range asyncFetchers {
					go fn(ctx, awsEni, resultCh)
				}

				var lambdaAttachment *coreTypes.LambdaAttachment
				var ecsAttachment *coreTypes.EcsAttachment
				var elbAttachment *coreTypes.ElbAttachment
				var vpceAttachment *coreTypes.VpceAttachment
				for range asyncFetchers {
					result := <-resultCh
					if result.Err != nil {
						return nil, result.Err
					}
					switch result.Data.(type) {
					case *coreTypes.LambdaAttachment:
						lambdaAttachment = result.Data.(*coreTypes.LambdaAttachment)
					case *coreTypes.EcsAttachment:
						ecsAttachment = result.Data.(*coreTypes.EcsAttachment)
					case *coreTypes.ElbAttachment:
						elbAttachment = result.Data.(*coreTypes.ElbAttachment)
					case *coreTypes.VpceAttachment:
						vpceAttachment = result.Data.(*coreTypes.VpceAttachment)
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
				e.cache.Set(newEni.Id, newEni)
				enis = append(enis, newEni)
			}
		}
	}

	return enis, nil
}

func (e *EniDetailsBuilder) getLambdaAttachmentAsync(ctx context.Context, awsEni ec2Types.NetworkInterface, resultCh chan clients.Result[any]) {
	lambdaAttachment, err := e.awsLambdaClient.GetLambdaAttachment(ctx, awsEni)
	if err != nil {
		resultCh <- clients.Result[any]{Err: err}
		return
	}
	resultCh <- clients.Result[any]{
		Data: lambdaAttachment,
	}
}

func (e *EniDetailsBuilder) getEcsAttachmentAsync(ctx context.Context, awsEni ec2Types.NetworkInterface, resultCh chan clients.Result[any]) {
	ecsAttachment, err := e.awsEcsClient.GetECSAttachment(ctx, awsEni)
	if err != nil {
		resultCh <- clients.Result[any]{Err: err}
		return
	}
	resultCh <- clients.Result[any]{
		Data: ecsAttachment,
	}
}

func (e *EniDetailsBuilder) getElbAttachmentAsync(ctx context.Context, awsEni ec2Types.NetworkInterface, resultCh chan clients.Result[any]) {
	elbAttachment, err := e.awsElbClient.GetELBAttachment(ctx, awsEni)
	if err != nil {
		resultCh <- clients.Result[any]{Err: err}
		return
	}
	resultCh <- clients.Result[any]{
		Data: elbAttachment,
	}
}

func (e *EniDetailsBuilder) getVpcAttachmentAsync(ctx context.Context, awsEni ec2Types.NetworkInterface, resultCh chan clients.Result[any]) {
	vpceAttachment, err := e.awsEc2Client.GetVpceAttachment(ctx, awsEni)
	if err != nil {
		resultCh <- clients.Result[any]{Err: err}
		return
	}
	resultCh <- clients.Result[any]{
		Data: vpceAttachment,
	}
}

// Get the IDs of the EC2 instances attached to the Network Interface
func getEC2Attachment(ifc ec2Types.NetworkInterface) *coreTypes.Ec2Attachment {
	if ifc.Attachment != nil && ifc.Attachment.InstanceId != nil {
		return &coreTypes.Ec2Attachment{InstanceId: *ifc.Attachment.InstanceId}
	}
	return nil
}
