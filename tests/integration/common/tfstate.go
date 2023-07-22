package common

import "encoding/json"

type TfState struct {
	Resources        []Resource        `json:"resources"`
	Outputs          map[string]Output `json:"outputs"`
	Backend          *Backend          `json:"backend"`
	Version          int               `json:"version"`
	TerraformVersion string            `json:"terraform_version"`
	Serial           int               `json:"serial"`
	Lineage          string            `json:"lineage"`
}

type Resource struct {
	Module    string    `json:"module"`
	Mode      string    `json:"mode"`
	Type      string    `json:"type"`
	Name      string    `json:"name"`
	Each      string    `json:"each"`
	Provider  string    `json:"provider"`
	Instances Instances `json:"instances"`
}

type Instances []Instance

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
