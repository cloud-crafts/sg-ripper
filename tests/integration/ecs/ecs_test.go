package ecs

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/require"
	"io"
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
	state = fetchTfState()
	_ = m.Run()
}

func fetchTfState() *common.TfState {
	cfg, configErr := config.LoadDefaultConfig(context.TODO(), config.WithRegion(Region), config.WithSharedConfigProfile(Profile))
	if configErr != nil {
		panic(configErr)
	}

	client := s3.NewFromConfig(cfg)

	// Read file from S3 bucket
	result, getErr := client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(BucketName),
		Key:    aws.String(ObjectKey),
	})

	if getErr != nil {
		panic(getErr)
	}

	stateBytes, _ := io.ReadAll(result.Body)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic(err)
		}
	}(result.Body)

	var tfState common.TfState
	parseErr := json.Unmarshal(stateBytes, &tfState)

	if parseErr != nil {
		panic(parseErr)
	}

	return &tfState
}

func TestAlbSg(t *testing.T) {

	albOutput := state.Outputs["alb_sg"]
	sgId := albOutput.Value.(string)

	securityGroups, err := core.ListSecurityGroups([]string{sgId}, core.Filters{Status: core.All}, Region, Profile)

	require.NoError(t, err)
	require.NotEmpty(t, securityGroups)
	require.Equal(t, 1, len(securityGroups))

	albSg := securityGroups[0]
	require.NotEmpty(t, albSg.UsedBy)
}
