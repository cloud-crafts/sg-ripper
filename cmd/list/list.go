package list

import (
	"fmt"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"sg-ripper/pkg/core"
	"sg-ripper/pkg/core/types"
)

var (
	Cmd = &cobra.Command{
		Use:   "list",
		Short: "List Security Groups with Details",
		Long:  "",
		Run:   runList,
	}

	used   bool
	unused bool
	sg     *[]string
)

func runList(cmd *cobra.Command, args []string) {
	var region string
	regionFlag := cmd.Flags().Lookup("region")
	if regionFlag != nil {
		region = regionFlag.Value.String()
	}

	var profile string
	profileFlag := cmd.Flags().Lookup("profile")
	if profileFlag != nil {
		profile = profileFlag.Value.String()
	}

	var ids []string
	if sg != nil {
		ids = *sg
	}

	filters := core.Filters{Status: core.All}
	if used {
		filters.Status = core.Used
	}
	if unused {
		filters.Status = core.Unused
	}

	groups, err := core.ListSecurityGroups(cmd.Context(), ids, filters, region, profile)
	if err != nil {
		cmd.PrintErrf("Error: %s", err)
		return
	}
	for _, sg := range groups {
		printSecurityGroupDetails(sg)
	}
}

func printSecurityGroupDetails(sg *types.SecurityGroupDetails) {
	pterm.DefaultSection.Printf("%s (%s)", sg.Name, sg.Id)
	reasons := getReasonsAgainstRemoval(sg)
	bulletList := []pterm.BulletListItem{
		{Level: 0, Text: fmt.Sprintf("Description: %s", sg.Description)},
		{Level: 0, Text: fmt.Sprintf("Can be Removed: %t", sg.CanBeRemoved())},
	}
	if len(reasons) > 0 {
		bulletList = append(bulletList, pterm.BulletListItem{Level: 1, Text: "Reasons:"})
		for _, reason := range reasons {
			bulletList = append(bulletList, pterm.BulletListItem{Level: 2, Text: reason})
		}
	}
	if len(sg.UsedBy) > 0 {
		bulletList = append(bulletList, pterm.BulletListItem{Level: 0, Text: "Used by Network Interface(s):"})
		for _, eni := range sg.UsedBy {
			bulletList = append(bulletList, pterm.BulletListItem{Level: 1, Text: fmt.Sprintf("%s", eni.Id)})
			if eni.Description != nil {
				bulletList = append(bulletList, pterm.BulletListItem{Level: 2, Text: fmt.Sprintf("Description: %s", *eni.Description)})
			}
			bulletList = append(bulletList, pterm.BulletListItem{Level: 2, Text: fmt.Sprintf("Type: %s", eni.Type)})
			bulletList = append(bulletList, pterm.BulletListItem{Level: 2, Text: fmt.Sprintf("Managed By AWS: %t", eni.ManagedByAWS)})
			bulletList = append(bulletList, pterm.BulletListItem{Level: 2, Text: fmt.Sprintf("Status: %s", eni.Status)})
			if eni.EC2Attachment != nil {
				bulletList = append(bulletList, pterm.BulletListItem{Level: 2, Text: "Attached to EC2 instance:"})
				bulletList = append(bulletList, pterm.BulletListItem{Level: 3, Text: fmt.Sprintf("%s", eni.EC2Attachment.InstanceId)})
			}
			if eni.LambdaAttachment != nil {
				bulletList = append(bulletList, pterm.BulletListItem{Level: 2, Text: "Used by Lambda Function:"})
				if eni.LambdaAttachment.IsRemoved {
					bulletList = append(bulletList, pterm.BulletListItem{Level: 3, Text: fmt.Sprintf(
						"%s - Note: This function was already removed. Please wait 15-20 minutes for the ENI to be removed by AWS.",
						eni.LambdaAttachment.Name)})
				} else {
					bulletList = append(bulletList, pterm.BulletListItem{Level: 3, Text: fmt.Sprintf("%s (%s)",
						eni.LambdaAttachment.Name, *eni.LambdaAttachment.Arn)})
				}
			}
			if eni.ECSAttachment != nil {
				bulletList = append(bulletList, pterm.BulletListItem{Level: 2, Text: "Used by ECS Service:"})

				service := "unknown"
				if eni.ECSAttachment.ServiceName != nil {
					service = *eni.ECSAttachment.ServiceName
				}

				cluster := "unknown"
				if eni.ECSAttachment.ClusterName != nil {
					cluster = *eni.ECSAttachment.ClusterName
				}

				taskArn := "unknown"
				if eni.ECSAttachment.TaskArn != nil {
					taskArn = *eni.ECSAttachment.TaskArn
				}

				if eni.ECSAttachment.IsRemoved {
					bulletList = append(bulletList, pterm.BulletListItem{Level: 3,
						Text: fmt.Sprintf("%s\\%s Note: the task was already removed. Please try to remove the ENI manually!",
							cluster, service)})
				} else {
					bulletList = append(bulletList, pterm.BulletListItem{Level: 3, Text: fmt.Sprintf("%s\\%s (%s)",
						cluster, service, taskArn)})
				}
			}
			if eni.ELBAttachment != nil {
				bulletList = append(bulletList, pterm.BulletListItem{Level: 2, Text: "Used by Elastic Load Balancer:"})
				if eni.ELBAttachment.IsRemoved {
					bulletList = append(bulletList, pterm.BulletListItem{Level: 3,
						Text: fmt.Sprintf("%s Note: the load balancer was removed. Please try to remove the ENI manually!",
							eni.ELBAttachment.Name)})
				} else {
					bulletList = append(bulletList, pterm.BulletListItem{Level: 3, Text: fmt.Sprintf("%s (%s)",
						eni.ELBAttachment.Name, *eni.ELBAttachment.Arn)})
				}
			}
			if eni.VpceAttachment != nil {
				bulletList = append(bulletList, pterm.BulletListItem{Level: 2, Text: "Used by VPC Endpoint:"})
				if eni.VpceAttachment.IsRemoved {
					bulletList = append(bulletList, pterm.BulletListItem{Level: 3,
						Text: fmt.Sprintf("%s Note: the VPC Endpoint was removed. Please try to remove the ENI manually!",
							*eni.VpceAttachment.Id)})
				} else {
					bulletList = append(bulletList, pterm.BulletListItem{Level: 3, Text: fmt.Sprintf("%s (%s)",
						*eni.VpceAttachment.ServiceName, *eni.VpceAttachment.Id)})
				}
			}
			if eni.RdsAttachments != nil {
				bulletList = append(bulletList, pterm.BulletListItem{Level: 2, Text: "Used by RDS (might be inaccurate):"})
				for _, attachment := range eni.RdsAttachments {
					bulletList = append(bulletList, pterm.BulletListItem{Level: 3, Text: fmt.Sprintf("%s",
						attachment.Identifier)})
				}
			}
		}
	}
	if len(sg.RuleReferences) > 0 {
		bulletList = append(bulletList, pterm.BulletListItem{Level: 0,
			Text: "Referenced by the following Security Groups as an Inbound/Outbound rule:"})
		for _, ruleRef := range sg.RuleReferences {
			bulletList = append(bulletList, pterm.BulletListItem{Level: 1, Text: fmt.Sprintf("%s", ruleRef)})
		}
	}
	err := pterm.DefaultBulletList.WithItems(bulletList).Render()
	if err != nil {
		return
	}
}

func getReasonsAgainstRemoval(sg *types.SecurityGroupDetails) []string {
	reasons := make([]string, 0)
	if !sg.CanBeRemoved() {
		if sg.Default {
			reasons = append(reasons, fmt.Sprintf("Security Group is Default in VPC %s", sg.VpcId))
		}
		if len(sg.UsedBy) > 0 {
			reasons = append(reasons, "Security Group is used by an ENI")
		}
		if len(sg.RuleReferences) > 0 {
			reasons = append(reasons, "Security Group is references by a Security Group Rule")
		}
	}
	return reasons
}

func init() {
	includeValidateFlags(Cmd)
}

func includeValidateFlags(cmd *cobra.Command) {
	sg = cmd.Flags().StringSlice("sg", nil,
		"[Optional] Security Group Id to be filtered. It can accept multiple values divided by comma. "+
			"Default: none (if none is specified all security groups will be retrieved)")
	cmd.Flags().BoolVarP(&used, "used", "u", false,
		"[Optional] List all security groups.")
	cmd.Flags().BoolVarP(&unused, "unused", "n", false,
		"[Optional] List unused security groups security groups.")
}
