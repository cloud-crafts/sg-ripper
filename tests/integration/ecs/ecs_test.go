package ecs

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
const ObjectKey = "sg-ripper/ecs/terraform.tfstate"

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

func TestAlbAttachment(t *testing.T) {

	albOut := state.Outputs["alb_sg"]
	sgId := albOut.Value.(string)

	securityGroups, err := core.ListSecurityGroups(context.TODO(), []string{sgId}, core.Filters{Status: core.All}, Region, Profile)

	require.NoError(t, err)
	require.NotEmpty(t, securityGroups)
	require.Equal(t, 1, len(securityGroups))

	sg := securityGroups[0]

	require.Equal(t, "vpc-alb-sg", sg.Name)
	require.NotEmpty(t, sg.UsedBy)
	require.GreaterOrEqual(t, 2, len(sg.UsedBy))
}

func TestECSTaskAttachment(t *testing.T) {

	output := state.Outputs["container_sg_id"]
	sgId := output.Value.(string)

	securityGroups, err := core.ListSecurityGroups(context.TODO(), []string{sgId}, core.Filters{Status: core.All}, Region, Profile)

	require.NoError(t, err)
	require.NotEmpty(t, securityGroups)
	require.Equal(t, 1, len(securityGroups))

	sg := securityGroups[0]

	require.NotEmpty(t, sg.UsedBy)
	require.GreaterOrEqual(t, 1, len(sg.UsedBy))

	eni := sg.UsedBy[0]
	require.NotNil(t, eni)
	require.NotNil(t, eni.ECSAttachment)
	require.NotNil(t, eni.ECSAttachment.TaskArn)
	require.NotNil(t, eni.ECSAttachment.ServiceName)
	require.NotNil(t, eni.ECSAttachment.ClusterName)
}
