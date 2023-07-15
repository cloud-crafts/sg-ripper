package core

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdaTypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"golang.org/x/exp/slices"
)

const MaxResults = 1000

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
func ListSecurityGroups(securityGroupIds []string, filters Filters, region string, profile string) ([]SecurityGroupUsage, error) {
	cfg, configErr := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region), config.WithSharedConfigProfile(profile))
	if configErr != nil {
		return nil, configErr
	}
	ec2Client := ec2.NewFromConfig(cfg)
	lambdaClient := lambda.NewFromConfig(cfg)

	securityGroups, sgErr := describeSecurityGroups(ec2Client, securityGroupIds)
	if sgErr != nil {
		return nil, sgErr
	}

	securityGroupRules, sgRuleErr := describeSecurityGroupRules(ec2Client)
	if sgRuleErr != nil {
		return nil, sgRuleErr
	}

	networkInterfaces, ifcErr := describeNetworkInterfaces(ec2Client, securityGroupIds)
	if ifcErr != nil {
		return nil, ifcErr
	}

	lambdaFunctions, fnErr := getLambdaFunctions(lambdaClient)
	if fnErr != nil {
		return nil, fnErr
	}

	usage := make([]SecurityGroupUsage, 0)
	for _, sg := range securityGroups {
		associatedInterfaces := getAssociatedNetworkInterfaces(sg, networkInterfaces)
		associations := make([]NetworkInterface, 0)
		for _, ifc := range associatedInterfaces {
			if ifc.NetworkInterfaceId != nil {
				nic := NetworkInterface{
					Id:               *ifc.NetworkInterfaceId,
					Description:      *ifc.Description,
					Type:             string(ifc.InterfaceType),
					ManagedByAWS:     *ifc.RequesterManaged,
					Status:           string(ifc.Status),
					EC2Attachment:    getEC2Attachments(ifc),
					LambdaAttachment: getLambdaAttachments(lambdaFunctions, sg, ifc),
				}
				associations = append(associations, nic)
			}
		}
		usage = append(usage, *NewSecurityGroupUsage(*sg.GroupName, *sg.GroupId, *sg.Description, associations,
			getSecurityGroupRuleReferences(sg, securityGroupRules), *sg.VpcId))
	}

	return applyFilters(usage, filters), nil
}

// Get a list of Network Interfaces used by the security groups from the input slice
func describeNetworkInterfaces(client *ec2.Client, securityGroupIds []string) ([]ec2Types.NetworkInterface, error) {
	filterName := "group-id"
	var filters []ec2Types.Filter
	if len(securityGroupIds) > 0 {
		filters = append(filters, ec2Types.Filter{Name: &filterName, Values: securityGroupIds})
	}

	var nextToken *string = nil
	networkInterfaces := make([]ec2Types.NetworkInterface, 0)
	for {
		ifcResponse, err := client.DescribeNetworkInterfaces(context.TODO(),
			&ec2.DescribeNetworkInterfacesInput{NextToken: nextToken, MaxResults: aws.Int32(int32(MaxResults))})
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

// Get a list of Security Groups based on the list of Security Group IDs provided as an input
func describeSecurityGroups(client *ec2.Client, securityGroupIds []string) ([]ec2Types.SecurityGroup, error) {
	filterName := "group-id"
	var filters []ec2Types.Filter
	if len(securityGroupIds) > 0 {
		filters = append(filters, ec2Types.Filter{Name: &filterName, Values: securityGroupIds})
	}

	var nextToken *string = nil
	securityGroups := make([]ec2Types.SecurityGroup, 0)
	for {
		sgResponse, err := client.DescribeSecurityGroups(context.TODO(),
			&ec2.DescribeSecurityGroupsInput{NextToken: nextToken, Filters: filters, MaxResults: aws.Int32(int32(MaxResults))})
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

// Get all the Security Group Rules. (TODO: try to optimise this to grab a sublist only)
func describeSecurityGroupRules(client *ec2.Client) ([]ec2Types.SecurityGroupRule, error) {
	var nextToken *string = nil
	securityGroupRules := make([]ec2Types.SecurityGroupRule, 0)
	for {
		sgResponse, err := client.DescribeSecurityGroupRules(context.TODO(),
			&ec2.DescribeSecurityGroupRulesInput{NextToken: nextToken, MaxResults: aws.Int32(int32(MaxResults))})
		if err != nil {
			return nil, err
		}
		nextToken = sgResponse.NextToken
		securityGroupRules = append(securityGroupRules, sgResponse.SecurityGroupRules...)

		if nextToken == nil {
			break
		}
	}

	return securityGroupRules, nil
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

// Get all the Lambda Functions which are attached to a VPC
func getLambdaFunctions(client *lambda.Client) ([]lambdaTypes.FunctionConfiguration, error) {
	var functions []lambdaTypes.FunctionConfiguration
	paginator := lambda.NewListFunctionsPaginator(client, &lambda.ListFunctionsInput{
		MaxItems: aws.Int32(int32(MaxResults)),
	})
	for paginator.HasMorePages() && len(functions) < MaxResults {
		pageOutput, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		for _, function := range pageOutput.Functions {
			if function.VpcConfig != nil && len(function.VpcConfig.SecurityGroupIds) > 0 {
				functions = append(functions, function)
			}
		}
	}

	return functions, nil
}

// Get the IDs of the EC2 instances attached to the Network Interface
func getEC2Attachments(ifc ec2Types.NetworkInterface) []EC2Attachment {
	if ifc.Attachment != nil && ifc.Attachment.InstanceId != nil {
		return []EC2Attachment{
			{InstanceId: *ifc.Attachment.InstanceId},
		}
	}
	return nil
}

// Get the Lambda Functions which are using the Security Group and the Network Interface
func getLambdaAttachments(functions []lambdaTypes.FunctionConfiguration, sg ec2Types.SecurityGroup, eni ec2Types.NetworkInterface) []LambdaAttachment {
	lambdaAttachments := make([]LambdaAttachment, 0)
	if eni.InterfaceType == ec2Types.NetworkInterfaceTypeLambda {
		for _, function := range functions {
			if slices.Contains(function.VpcConfig.SecurityGroupIds, *sg.GroupId) {
				lambdaAttachments = append(lambdaAttachments, LambdaAttachment{
					Arn:  *function.FunctionArn,
					Name: *function.FunctionName,
				})
			}
		}
	}
	return lambdaAttachments
}

// Get the Security Group Rules which are referencing the Security Group
func getSecurityGroupRuleReferences(sg ec2Types.SecurityGroup, securityGroupRules []ec2Types.SecurityGroupRule) []string {
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
