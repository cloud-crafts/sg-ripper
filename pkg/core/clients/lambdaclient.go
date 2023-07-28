package clients

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdaTypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/smithy-go"
	cmap "github.com/orcaman/concurrent-map/v2"
	"regexp"
	coreTypes "sg-ripper/pkg/core/types"
)

type AwsLambdaClient struct {
	client *lambda.Client
	cache  cmap.ConcurrentMap[string, *coreTypes.LambdaAttachment]
}

func NewAwsLambdaClient(cfg aws.Config) *AwsLambdaClient {
	return &AwsLambdaClient{
		client: lambda.NewFromConfig(cfg),
		cache:  cmap.New[*coreTypes.LambdaAttachment](),
	}
}

// GetLambdaAttachment returns a pointer to an LambdaAttachment for the network interface. If there is no attachment found,
// the returned value is a nil.
func (c *AwsLambdaClient) GetLambdaAttachment(ctx context.Context, eni ec2Types.NetworkInterface) (*coreTypes.LambdaAttachment, error) {
	regex := regexp.MustCompile("AWS Lambda VPC ENI-(?P<fnName>.+)-([a-z]|[0-9]){8}-(([a-z]|[0-9]){4}-){3}([a-z]|[0-9]){12}")
	if eni.InterfaceType == ec2Types.NetworkInterfaceTypeLambda && eni.Description != nil {
		match := regex.FindStringSubmatch(*eni.Description)
		if len(match) > 0 {
			fnName := match[regex.SubexpIndex("fnName")]

			if cachedFn, ok := c.cache.Get(fnName); ok {
				return cachedFn, nil
			}

			fnConfig, err := c.getLambdaFunctionConfigByName(ctx, c.client, fnName)
			if err != nil {
				return nil, err
			}

			var attachment *coreTypes.LambdaAttachment
			if fnConfig != nil {
				attachment = &coreTypes.LambdaAttachment{
					Arn:       fnConfig.FunctionArn,
					Name:      fnName,
					IsRemoved: false,
				}
			} else {
				attachment = &coreTypes.LambdaAttachment{
					Name:      fnName,
					IsRemoved: true,
				}
			}

			c.cache.Set(fnName, attachment)
			return attachment, nil
		}
	}

	return nil, nil
}

// Get the configuration for a Lambda function. If the function does not exist, the returned value will be nil
func (c *AwsLambdaClient) getLambdaFunctionConfigByName(ctx context.Context, client *lambda.Client, fnName string) (*lambdaTypes.FunctionConfiguration, error) {
	fnInput := lambda.GetFunctionInput{FunctionName: &fnName}

	function, err := client.GetFunction(ctx, &fnInput)
	if err != nil {
		// Handle error in case the function does not exist. Do not return this error to the caller
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			var resourceNotFoundException *lambdaTypes.ResourceNotFoundException
			switch {
			case errors.As(apiErr, &resourceNotFoundException):
				return nil, nil
			}
		}
		return nil, err
	}

	return function.Configuration, nil
}
