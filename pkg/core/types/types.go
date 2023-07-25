package types

type SecurityGroup struct {
	Name           string
	Id             string
	Description    string
	Default        bool
	UsedBy         []*NetworkInterface
	RuleReferences []string
	VpcId          string
}

// NewSecurityGroup creates a new SecurityGroup object and returns a pointer to it
func NewSecurityGroup(name string, id string, description string, usedBy []*NetworkInterface, ruleReferences []string,
	vpcId string) *SecurityGroup {
	return &SecurityGroup{
		Name:           name,
		Id:             id,
		Description:    description,
		RuleReferences: ruleReferences,
		UsedBy:         usedBy,
		VpcId:          vpcId,
		Default:        name == "default",
	}
}

// IsInUse returns true if the Security Group is in use: it is used by at least one Network Interface, or
// it is referenced by an SG inbound/outbound rule
func (u *SecurityGroup) IsInUse() bool {
	return len(u.UsedBy) > 0 || len(u.RuleReferences) > 0
}

// CanBeRemoved returns true if the Security Group can be removed, meaning it is not in use, or it is not a default SG
func (u *SecurityGroup) CanBeRemoved() bool {
	return !u.Default && !u.IsInUse()
}

type NetworkInterface struct {
	Id                       string
	Description              *string
	Type                     string
	ManagedByAWS             bool
	Status                   string
	EC2Attachment            *Ec2Attachment
	LambdaAttachment         *LambdaAttachment
	ECSAttachment            *EcsAttachment
	ELBAttachment            *ElbAttachment
	VpceAttachment           *VpceAttachment
	SecurityGroupIdentifiers []SecurityGroupIdentifier
}

func (eni *NetworkInterface) IsInUse() bool {
	return eni.Status == "in-use"
}

type Ec2Attachment struct {
	InstanceId string
}

type LambdaAttachment struct {
	IsRemoved bool
	Name      string
	Arn       *string
}

type EcsAttachment struct {
	IsRemoved     bool
	ServiceName   *string
	ClusterName   *string
	ContainerName *string
	TaskArn       *string
}

type ElbAttachment struct {
	IsRemoved bool
	Name      string
	Arn       *string
}

type VpceAttachment struct {
	IsRemoved   bool
	Id          *string
	ServiceName *string
}

type SecurityGroupIdentifier struct {
	Name *string
	Id   string
}
