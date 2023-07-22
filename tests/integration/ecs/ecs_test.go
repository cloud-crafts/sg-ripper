package ecs

import (
	"bytes"
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/fujiwara/tfstate-lookup/tfstate"
	"github.com/stretchr/testify/require"
	"io"
	"sg-ripper/pkg/core"
	"testing"
)

const Region = "us-east-1"
const Profile = "A4L-DEV"
const BucketName = "terraform-state-a4ldev"
const ObjectKey = "sg-ripper/ecs/terraform.tfstate"

var state *tfstate.TFState

func TestMain(m *testing.M) {
	state = fetchTfState()
	_ = m.Run()
}

func fetchTfState() *tfstate.TFState {
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

	state, parseErr := tfstate.Read(nil, bytes.NewReader(stateBytes))
	if parseErr != nil {
		panic(parseErr)
	}

	return state
}

func TestAlbSg(t *testing.T) {

	albSgId, err := state.Lookup("aws_security_group.alb_sg.id")
	if err != nil {
		println(err)
		t.Skip()
	}

	securityGroups, err := core.ListSecurityGroups([]string{albSgId.String()}, core.Filters{Status: core.All}, Region, Profile)

	require.NoError(t, err)
	require.NotEmpty(t, securityGroups)
	require.Equal(t, 1, len(securityGroups))

	albSg := securityGroups[0]
	require.NotEmpty(t, albSg.UsedBy)
}
