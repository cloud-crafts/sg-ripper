package clients

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	rdsTypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/hashicorp/go-set"
	coreTypes "sg-ripper/pkg/core/types"
)

type AwsRdsClient struct {
	client           *rds.Client
	dbInstancesCache []rdsTypes.DBInstance
}

func NewAwsRdsClient(cfg aws.Config) *AwsRdsClient {
	return &AwsRdsClient{
		client:           rds.NewFromConfig(cfg),
		dbInstancesCache: make([]rdsTypes.DBInstance, 0),
	}
}

func (c *AwsRdsClient) GetRdsAttachments(ctx context.Context, eni ec2Types.NetworkInterface) ([]coreTypes.RdsAttachment, error) {
	rdsAttachments := make([]coreTypes.RdsAttachment, 0)
	if eni.Description != nil && *eni.Description == "RDSNetworkInterface" {
		if len(c.dbInstancesCache) <= 0 {
			if err := c.populateDbInstanceCache(ctx); err != nil {
				return nil, err
			}
		}

		dbSecurityGroups := make(map[string]*set.Set[string])
		dbInstanceMap := make(map[string]rdsTypes.DBInstance)
		for _, dbInstance := range c.dbInstancesCache {
			dbInstanceMap[*dbInstance.DBInstanceIdentifier] = dbInstance
			sgIds := set.New[string](len(dbInstance.VpcSecurityGroups))
			for _, vpcSg := range dbInstance.VpcSecurityGroups {
				sgIds.Insert(*vpcSg.VpcSecurityGroupId)
			}
			dbSecurityGroups[*dbInstance.DBInstanceIdentifier] = sgIds
		}

		eniSecurityGroups := set.New[string](len(eni.Groups))
		for _, sg := range eni.Groups {
			eniSecurityGroups.Insert(*sg.GroupId)
		}

		for instanceIdentifier, sg := range dbSecurityGroups {
			if eniSecurityGroups.Equal(sg) {
				rdsAttachments = append(rdsAttachments, coreTypes.RdsAttachment{
					IsRemoved:  false,
					Identifier: instanceIdentifier,
				})
			}
		}
	}
	return rdsAttachments, nil
}

func (c *AwsRdsClient) populateDbInstanceCache(ctx context.Context) error {
	var nextToken *string
	for {
		dbInstanceResponse, err := c.client.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{Marker: nextToken})
		if err != nil {
			return err
		}

		c.dbInstancesCache = append(c.dbInstancesCache, dbInstanceResponse.DBInstances...)

		if dbInstanceResponse.Marker != nil {
			nextToken = dbInstanceResponse.Marker
		} else {
			break
		}
	}

	return nil
}
