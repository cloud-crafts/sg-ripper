package common

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"io"
)

type TfState struct {
	Resources        []Resource        `json:"resources"`
	Outputs          map[string]Output `json:"outputs"`
	Backend          *Backend          `json:"backend"`
	Version          int               `json:"version"`
	TerraformVersion string            `json:"terraform_version"`
	Serial           int               `json:"serial"`
	Lineage          string            `json:"lineage"`
}

func ReadTfStateFromS3(client *s3.Client, bucketName, objectKey string) (*TfState, error) {
	objectOutput, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return nil, err
	}

	stateBytes, err := io.ReadAll(objectOutput.Body)
	if err != nil {
		return nil, err
	}

	var tfState TfState
	if err = json.Unmarshal(stateBytes, &tfState); err != nil {
		return nil, err
	}

	return &tfState, nil
}

type Resource struct {
	Module    string     `json:"module"`
	Mode      string     `json:"mode"`
	Type      string     `json:"type"`
	Name      string     `json:"name"`
	Each      string     `json:"each"`
	Provider  string     `json:"provider"`
	Instances []Instance `json:"instances"`
}

type Instance struct {
	IndexKey       json.RawMessage `json:"index_key"`
	SchemaVersion  int             `json:"schema_version"`
	Attributes     any             `json:"attributes"`
	AttributesFlat any             `json:"attributes_flat"`
	Private        string          `json:"private"`

	data any
}

type Output struct {
	Value any    `json:"value"`
	Type  string `json:"type"`
}

type Backend struct {
	Type   string `json:"type"`
	Config map[string]any
}
