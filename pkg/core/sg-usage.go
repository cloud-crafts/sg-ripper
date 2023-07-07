package core

type SecurityGroupUsage struct {
	SecurityGroupName        string
	SecurityGroupId          string
	SecurityGroupDescription string
	Default                  bool
	UsedBy                   []NetworkInterface
	VpcId                    string
}

func NewSecurityGroupUsage(securityGroupName string, securityGroupId string, securityGroupDescription string,
	usedBy []NetworkInterface, vpcId string) *SecurityGroupUsage {
	return &SecurityGroupUsage{
		SecurityGroupName:        securityGroupName,
		SecurityGroupId:          securityGroupId,
		SecurityGroupDescription: securityGroupDescription,
		UsedBy:                   usedBy,
		VpcId:                    vpcId,
		Default:                  securityGroupName == "default",
	}
}

type NetworkInterface struct {
	Id               string
	Description      string
	Type             string
	ManagedByAWS     bool
	Status           string
	EC2Attachment    []EC2Attachment
	LambdaAttachment []string
	ECSAttachment    []string
}

type EC2Attachment struct {
	InstanceId string
}

type LambdaAttachment struct {
	Arn  string
	Name string
}

type ECSAttachment struct {
	ServiceName string
}
