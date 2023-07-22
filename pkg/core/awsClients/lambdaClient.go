package awsClients

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdaTypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/smithy-go"
	"regexp"
	coreTypes "sg-ripper/pkg/core/types"
)

type AwsLambdaClient struct {
	client *lambda.Client
	cache  map[string]*coreTypes.LambdaAttachment
}

func NewAwsLambdaClient(cfg aws.Config) *AwsLambdaClient {
	return &AwsLambdaClient{
		client: lambda.NewFromConfig(cfg),
		cache:  make(map[string]*coreTypes.LambdaAttachment),
	}
}

// GetLambdaAttachment returns a Lambda Function which is using the ENI
func (c *AwsLambdaClient) GetLambdaAttachment(eni ec2Types.NetworkInterface) (*coreTypes.LambdaAttachment, error) {
	regex := regexp.MustCompile("AWS Lambda VPC ENI-(?P<fnName>.+)-([a-z]|[0-9]){8}-(([a-z]|[0-9]){4}-){3}([a-z]|[0-9]){12}")
	if eni.InterfaceType == ec2Types.NetworkInterfaceTypeLambda && eni.Description != nil {
		match := regex.FindStringSubmatch(*eni.Description)
		if len(match) > 0 {
			fnName := match[regex.SubexpIndex("fnName")]

			if cachedElb, ok := c.cache[fnName]; ok {
				return cachedElb, nil
			}

			fnConfig, fnErr := c.getLambdaFunctionConfigByName(c.client, fnName)
			if fnErr != nil {
				return nil, fnErr
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

			c.cache[fnName] = attachment

			return attachment, nil
		}
	}
	return nil, nil
}

// Get the configuration for a Lambda function. If the function does not exist, the returned value will be nil
func (c *AwsLambdaClient) getLambdaFunctionConfigByName(client *lambda.Client, fnName string) (*lambdaTypes.FunctionConfiguration, error) {
	fnInput := lambda.GetFunctionInput{FunctionName: &fnName}

	function, err := client.GetFunction(context.TODO(), &fnInput)
	if err != nil {
		// Handle error in case the function does not exist. Do not return this error to the caller
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.(type) {
			case *lambdaTypes.ResourceNotFoundException:
				return nil, nil
			}
		}
		return nil, err
	}

	return function.Configuration, nil
}
