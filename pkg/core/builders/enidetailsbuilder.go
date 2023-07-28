package builders

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	cmap "github.com/orcaman/concurrent-map/v2"
	"sg-ripper/pkg/core/clients"
	"sg-ripper/pkg/core/result"
	coreTypes "sg-ripper/pkg/core/types"
)

type EniDetailsBuilder struct {
	awsEc2Client    *clients.AwsEc2Client
	awsLambdaClient *clients.AwsLambdaClient
	awsElbClient    *clients.AwsElbClient
	awsEcsClient    *clients.AwsEcsClient
	awsRdsClient    *clients.AwsRdsClient
	cache           cmap.ConcurrentMap[string, *coreTypes.NetworkInterfaceDetails]
}

func NewEniBuilder(cfg aws.Config) *EniDetailsBuilder {
	return &EniDetailsBuilder{
		awsEc2Client:    clients.NewAwsEc2Client(cfg),
		awsLambdaClient: clients.NewAwsLambdaClient(cfg),
		awsElbClient:    clients.NewAwsElbClient(cfg),
		awsEcsClient:    clients.NewAwsEcsClient(cfg),
		awsRdsClient:    clients.NewAwsRdsClient(cfg),
		cache:           cmap.New[*coreTypes.NetworkInterfaceDetails](),
	}
}

// FromRemoteInterfaces returns a slice of coreTypes.NetworkInterfaceDetails
func (e *EniDetailsBuilder) FromRemoteInterfaces(ctx context.Context, awsEniBatch []ec2Types.NetworkInterface) ([]coreTypes.NetworkInterfaceDetails, error) {
	eniDetails := make([]coreTypes.NetworkInterfaceDetails, 0)

	for _, awsEni := range awsEniBatch {
		if awsEni.NetworkInterfaceId != nil {

			// Check if Network Interface is already in the cache to avoid computing multiple times which resources
			// are using it
			if cachedEni, ok := e.cache.Get(*awsEni.NetworkInterfaceId); ok {
				eniDetails = append(eniDetails, *cachedEni)
			} else {
				resultCh := make(chan result.Result[any])

				asyncFetchers := []func(context.Context, ec2Types.NetworkInterface, chan result.Result[any]){
					e.getLambdaAttachmentAsync, e.getEcsAttachmentAsync, e.getElbAttachmentAsync, e.getVpcAttachmentAsync,
					e.getRdsAttachmentAsync,
				}

				for _, fn := range asyncFetchers {
					go fn(ctx, awsEni, resultCh)
				}

				var lambdaAttachment *coreTypes.LambdaAttachment
				var ecsAttachment *coreTypes.EcsAttachment
				var elbAttachment *coreTypes.ElbAttachment
				var vpceAttachment *coreTypes.VpceAttachment
				var rdsAttachments []coreTypes.RdsAttachment
				for range asyncFetchers {
					res := <-resultCh
					if res.Err != nil {
						return nil, res.Err
					}
					switch res.Data.(type) {
					case *coreTypes.LambdaAttachment:
						lambdaAttachment = res.Data.(*coreTypes.LambdaAttachment)
					case *coreTypes.EcsAttachment:
						ecsAttachment = res.Data.(*coreTypes.EcsAttachment)
					case *coreTypes.ElbAttachment:
						elbAttachment = res.Data.(*coreTypes.ElbAttachment)
					case *coreTypes.VpceAttachment:
						vpceAttachment = res.Data.(*coreTypes.VpceAttachment)
					case []coreTypes.RdsAttachment:
						rdsAttachments = res.Data.([]coreTypes.RdsAttachment)
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

				newEni := coreTypes.NetworkInterfaceDetails{
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
					RdsAttachments:           rdsAttachments,
					SecurityGroupIdentifiers: sgIdentifiers,
				}

				// Add the new interface to the cache
				e.cache.Set(newEni.Id, &newEni)
				eniDetails = append(eniDetails, newEni)
			}
		}
	}

	return eniDetails, nil
}

func (e *EniDetailsBuilder) getLambdaAttachmentAsync(ctx context.Context, awsEni ec2Types.NetworkInterface, resultCh chan result.Result[any]) {
	lambdaAttachment, err := e.awsLambdaClient.GetLambdaAttachment(ctx, awsEni)
	if err != nil {
		resultCh <- result.Result[any]{Err: err}
		return
	}
	resultCh <- result.Result[any]{
		Data: lambdaAttachment,
	}
}

func (e *EniDetailsBuilder) getEcsAttachmentAsync(ctx context.Context, awsEni ec2Types.NetworkInterface, resultCh chan result.Result[any]) {
	ecsAttachment, err := e.awsEcsClient.GetEcsAttachment(ctx, awsEni)
	if err != nil {
		resultCh <- result.Result[any]{Err: err}
		return
	}
	resultCh <- result.Result[any]{
		Data: ecsAttachment,
	}
}

func (e *EniDetailsBuilder) getElbAttachmentAsync(ctx context.Context, awsEni ec2Types.NetworkInterface, resultCh chan result.Result[any]) {
	elbAttachment, err := e.awsElbClient.GetELBAttachment(ctx, awsEni)
	if err != nil {
		resultCh <- result.Result[any]{Err: err}
		return
	}
	resultCh <- result.Result[any]{
		Data: elbAttachment,
	}
}

func (e *EniDetailsBuilder) getVpcAttachmentAsync(ctx context.Context, awsEni ec2Types.NetworkInterface, resultCh chan result.Result[any]) {
	vpceAttachment, err := e.awsEc2Client.GetVpceAttachment(ctx, awsEni)
	if err != nil {
		resultCh <- result.Result[any]{Err: err}
		return
	}
	resultCh <- result.Result[any]{
		Data: vpceAttachment,
	}
}

func (e *EniDetailsBuilder) getRdsAttachmentAsync(ctx context.Context, awsEni ec2Types.NetworkInterface, resultCh chan result.Result[any]) {
	rdsAttachments, err := e.awsRdsClient.GetRdsAttachments(ctx, awsEni)
	if err != nil {
		resultCh <- result.Result[any]{Err: err}
		return
	}
	resultCh <- result.Result[any]{
		Data: rdsAttachments,
	}
}

// Get the IDs of the EC2 instances attached to the Network Interface
func getEC2Attachment(ifc ec2Types.NetworkInterface) *coreTypes.Ec2Attachment {
	if ifc.Attachment != nil && ifc.Attachment.InstanceId != nil {
		return &coreTypes.Ec2Attachment{InstanceId: *ifc.Attachment.InstanceId}
	}
	return nil
}
