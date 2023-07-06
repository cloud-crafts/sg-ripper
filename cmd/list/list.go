package list

import (
	"fmt"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"sg-ripper/pkg/core"
)

var (
	Cmd = &cobra.Command{
		Use:   "list",
		Short: "List Security Groups with Details",
		Long:  "",
		Run:   runList,
	}
)

func runList(cmd *cobra.Command, args []string) {
	sgUsage, err := core.ListAllSecurityGroups("us-east-1")
	if err != nil {
		cmd.PrintErrf("Error: %s", err)
	}
	for _, usage := range sgUsage {
		printSecurityGroupUsage(usage)
	}
}

func printSecurityGroupUsage(usage core.SecurityGroupUsage) {
	pterm.DefaultSection.Printf("%s(%s)", usage.SecurityGroupName, usage.SecurityGroupId)
	canRm, reason := canBeRemoved(usage)
	bulletList := []pterm.BulletListItem{
		{Level: 0, Text: fmt.Sprintf("Description: %s", usage.SecurityGroupDescription)},
		{Level: 0, Text: fmt.Sprintf("Can be Removed: %t. Reason: %s", canRm, reason)},
	}
	if usage.UsedBy != nil {
		bulletList = append(bulletList, pterm.BulletListItem{Level: 0, Text: "Used by Network Interface(s):"})
		for _, eni := range usage.UsedBy {
			bulletList = append(bulletList, pterm.BulletListItem{Level: 1, Text: fmt.Sprintf("%s", eni.Id)})
			bulletList = append(bulletList, pterm.BulletListItem{Level: 2, Text: fmt.Sprintf("Description: %s", eni.Description)})
			bulletList = append(bulletList, pterm.BulletListItem{Level: 2, Text: fmt.Sprintf("Type: %s", eni.Type)})
			bulletList = append(bulletList, pterm.BulletListItem{Level: 2, Text: fmt.Sprintf("Managed By AWS: %t", eni.ManagedByAWS)})
			bulletList = append(bulletList, pterm.BulletListItem{Level: 2, Text: fmt.Sprintf("Status: %s", eni.Status)})
			if len(eni.EC2Attachment) > 0 {
				bulletList = append(bulletList, pterm.BulletListItem{Level: 2, Text: "Attached to EC2 instance(s):"})
				for _, ec2Attachment := range eni.EC2Attachment {
					bulletList = append(bulletList, pterm.BulletListItem{Level: 3, Text: fmt.Sprintf("%s", ec2Attachment.InstanceId)})
				}
			}
		}
	}
	err := pterm.DefaultBulletList.WithItems(bulletList).Render()
	if err != nil {
		return
	}
}

func canBeRemoved(usage core.SecurityGroupUsage) (bool, string) {
	if usage.Default {
		return false, fmt.Sprintf("Security Group is Default in VPC %s", usage.VpcId)
	}
	if usage.UsedBy != nil {
		return false, "Security Group is in use"
	}
	return true, "No usage detected"
}

func init() {
	includeValidateFlags(Cmd)
}

func includeValidateFlags(cmd *cobra.Command) {
}