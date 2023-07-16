package core

type SecurityGroupUsage struct {
	SecurityGroupName           string
	SecurityGroupId             string
	SecurityGroupDescription    string
	Default                     bool
	UsedBy                      []NetworkInterface
	SecurityGroupRuleReferences []string
	VpcId                       string
}

func NewSecurityGroupUsage(securityGroupName string, securityGroupId string, securityGroupDescription string,
	usedBy []NetworkInterface, securityGroupRuleReferences []string, vpcId string) *SecurityGroupUsage {
	return &SecurityGroupUsage{
		SecurityGroupName:           securityGroupName,
		SecurityGroupId:             securityGroupId,
		SecurityGroupDescription:    securityGroupDescription,
		SecurityGroupRuleReferences: securityGroupRuleReferences,
		UsedBy:                      usedBy,
		VpcId:                       vpcId,
		Default:                     securityGroupName == "default",
	}
}

// IsInUse returns true if the Security Group is in use: it is used by at least one Network Interface, or
// it is referenced by an SG inbound/outbound rule
func (u *SecurityGroupUsage) IsInUse() bool {
	return len(u.UsedBy) > 0 || len(u.SecurityGroupRuleReferences) > 0
}

// CanBeRemoved returns true if the Security Group can be removed, meaning it is not in use, or it is not a default SG
func (u *SecurityGroupUsage) CanBeRemoved() bool {
	return !u.Default && !u.IsInUse()
}

type NetworkInterface struct {
	Id                string
	Description       *string
	Type              string
	ManagedByAWS      bool
	Status            string
	EC2Attachment     *EC2Attachment
	LambdaAttachments []LambdaAttachment
	ECSAttachment     []string
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
