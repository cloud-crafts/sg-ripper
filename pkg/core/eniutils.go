package core

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"sg-ripper/pkg/core/builders"
	"sg-ripper/pkg/core/clients"
	"sg-ripper/pkg/core/result"
	coreTypes "sg-ripper/pkg/core/types"
)

func ListNetworkInterfaces(ctx context.Context, eniIds []string, filters Filters, region string, profile string) ([]*coreTypes.NetworkInterfaceDetails, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region), config.WithSharedConfigProfile(profile))
	if err != nil {
		return nil, err
	}

	ec2Client := clients.NewAwsEc2Client(cfg)

	eniResultCh := make(chan result.Result[[]ec2Types.NetworkInterface])
	ec2Client.DescribeNetworkInterfaces(ctx, eniIds, eniResultCh)

	enis := make([]*coreTypes.NetworkInterfaceDetails, 0)
	eniDetailsBuilder := builders.NewEniBuilder(cfg)
	for result := range eniResultCh {
		eniDetailsBatch, err := eniDetailsBuilder.FromAwsEniBatch(ctx, result.Data)
		if err != nil {
			return nil, err
		}
		enis = append(enis, eniDetailsBatch...)
	}

	return applyEniFilters(enis, filters), nil
}

// Apply Filters to the list of Network interface usages
func applyEniFilters(enis []*coreTypes.NetworkInterfaceDetails, filters Filters) []*coreTypes.NetworkInterfaceDetails {
	if filters.Status == All {
		return enis
	}

	var filterFn func(eni *coreTypes.NetworkInterfaceDetails) bool

	switch filters.Status {
	case Used:
		filterFn = func(eni *coreTypes.NetworkInterfaceDetails) bool {
			return eni.IsInUse()
		}
	case Unused:
		filterFn = func(eni *coreTypes.NetworkInterfaceDetails) bool {
			return !eni.IsInUse()
		}
	}

	result := make([]*coreTypes.NetworkInterfaceDetails, 0)
	for _, eni := range enis {
		if filterFn(eni) {
			result = append(result, eni)
		}
	}
	return result
}
