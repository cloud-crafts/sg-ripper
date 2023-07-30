package lambda

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/require"
	"sg-ripper/pkg/core"
	"sg-ripper/tests/integration/common"
	"testing"
)

const Region = "us-east-1"
const Profile = "A4L-DEV"
const BucketName = "terraform-state-a4ldev"
const ObjectKey = "sg-ripper/lambda/terraform.tfstate"

var state *common.TfState

func TestMain(m *testing.M) {
	tfState, err := fetchTfState()
	if err != nil {
		panic(err)
	}

	state = tfState

	_ = m.Run()
}

func fetchTfState() (*common.TfState, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(Region), config.WithSharedConfigProfile(Profile))
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg)

	if tfState, err := common.ReadTfStateFromS3(client, BucketName, ObjectKey); err != nil {
		return nil, err
	} else {
		return tfState, nil
	}
}

func TestLambdaAttachment(t *testing.T) {

	albOut := state.Outputs["lambda_sg"]
	sgId := albOut.Value.(string)

	securityGroups, err := core.ListSecurityGroups(context.TODO(), []string{sgId}, core.Filters{Status: core.All}, Region, Profile)

	require.NoError(t, err)
	require.NotEmpty(t, securityGroups)
	require.Equal(t, 1, len(securityGroups))

	sg := securityGroups[0]

	require.Equal(t, 3, len(sg.UsedBy))
	for _, eni := range sg.UsedBy {
		require.Contains(t, *eni.Description, "AWS Lambda VPC ENI")
	}
}

func TestAnotherLambdaAttachment(t *testing.T) {

	albOut := state.Outputs["another_lambda_sg"]
	sgId := albOut.Value.(string)

	securityGroups, err := core.ListSecurityGroups(context.TODO(), []string{sgId}, core.Filters{Status: core.All}, Region, Profile)

	require.NoError(t, err)
	require.NotEmpty(t, securityGroups)
	require.Equal(t, 1, len(securityGroups))

	sg := securityGroups[0]

	require.Equal(t, 3, len(sg.UsedBy))
	for _, eni := range sg.UsedBy {
		require.Contains(t, *eni.Description, "AWS Lambda VPC ENI")
	}
}

func TestCommonLambdaAttachment(t *testing.T) {

	albOut := state.Outputs["common_lambda_sg"]
	sgId := albOut.Value.(string)

	securityGroups, err := core.ListSecurityGroups(context.TODO(), []string{sgId}, core.Filters{Status: core.All}, Region, Profile)

	require.NoError(t, err)
	require.NotEmpty(t, securityGroups)
	require.Equal(t, 1, len(securityGroups))

	sg := securityGroups[0]

	require.Equal(t, 6, len(sg.UsedBy))
	for _, eni := range sg.UsedBy {
		require.Contains(t, *eni.Description, "AWS Lambda VPC ENI")
	}
}
