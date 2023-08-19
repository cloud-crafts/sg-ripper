package list

import (
	"fmt"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"sg-ripper/cmd/cmdutils"
	"sg-ripper/pkg/core"
	"sg-ripper/pkg/core/types"
)

var (
	Cmd = &cobra.Command{
		Use:   "list",
		Short: "List Security Groups with Details",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			regionFlag := cmd.Flags().Lookup("region")
			if regionFlag != nil {
				region = regionFlag.Value.String()
			}

			profileFlag := cmd.Flags().Lookup("profile")
			if profileFlag != nil {
				profile = profileFlag.Value.String()
			}

			return nil
		},
		RunE: runList,
	}

	used    bool
	unused  bool
	region  string
	profile string
	sg      *[]string
)

func runList(cmd *cobra.Command, args []string) error {
	filters := core.Filters{Status: core.All}
	if used {
		filters.Status = core.Used
	}
	if unused {
		filters.Status = core.Unused
	}

	groups, err := core.ListSecurityGroups(cmd.Context(), *sg, filters, region, profile)
	if err != nil {
		return err
	}

	for _, sg := range groups {
		err := printSecurityGroupDetails(sg)
		if err != nil {
			return err
		}
	}

	return nil
}

func printSecurityGroupDetails(sg types.SecurityGroupDetails) error {
	pterm.DefaultSection.Printf("%s (%s)", sg.Name, sg.Id)

	reasons := getReasonsAgainstRemoval(sg)
	var canBeRemoved string
	if sg.CanBeRemoved() {
		canBeRemoved = pterm.LightGreen("YES")
	} else {
		canBeRemoved = pterm.LightRed("NO")
	}

	bulletList := []pterm.BulletListItem{
		{
			Level:       0,
			TextStyle:   pterm.NewStyle(pterm.FgLightWhite),
			BulletStyle: pterm.NewStyle(pterm.FgLightWhite),
			Text:        fmt.Sprintf("Description: %s", pterm.Cyan(sg.Description)),
		},
		{
			Level:       0,
			TextStyle:   pterm.NewStyle(pterm.FgLightWhite),
			BulletStyle: pterm.NewStyle(pterm.FgLightWhite),
			Text:        fmt.Sprintf("Can be Removed: %s", canBeRemoved),
		},
	}

	if len(reasons) > 0 {
		bulletList = append(bulletList, pterm.BulletListItem{
			Level:       1,
			TextStyle:   pterm.NewStyle(pterm.FgLightWhite),
			BulletStyle: pterm.NewStyle(pterm.FgLightWhite),
			Text:        "Reasons:",
		})
		for _, reason := range reasons {
			bulletList = append(bulletList, pterm.BulletListItem{
				Level:       2,
				TextStyle:   pterm.NewStyle(pterm.FgLightYellow),
				BulletStyle: pterm.NewStyle(pterm.FgLightYellow),
				Text:        reason,
			})
		}
	}

	if len(sg.UsedBy) > 0 {
		bulletList = append(bulletList, pterm.BulletListItem{
			Level:       0,
			TextStyle:   pterm.NewStyle(pterm.FgLightWhite),
			BulletStyle: pterm.NewStyle(pterm.FgLightWhite),
			Text:        "Used by Network Interface(s):",
		})

		for _, eni := range sg.UsedBy {
			bulletList = append(bulletList, pterm.BulletListItem{
				Level:       1,
				TextStyle:   pterm.NewStyle(pterm.FgLightWhite),
				BulletStyle: pterm.NewStyle(pterm.FgLightWhite),
				Text:        fmt.Sprintf("%s (%s)", pterm.LightBlue(eni.Id), pterm.LightMagenta(eni.PrivateIPAddress)),
			})

			if eni.Description != nil {
				bulletList = append(bulletList, pterm.BulletListItem{
					Level:       2,
					TextStyle:   pterm.NewStyle(pterm.FgLightWhite),
					BulletStyle: pterm.NewStyle(pterm.FgLightWhite),
					Text:        fmt.Sprintf("Description: %s", pterm.Cyan(*eni.Description)),
				})
			}

			bulletList = append(bulletList, pterm.BulletListItem{
				Level:       2,
				TextStyle:   pterm.NewStyle(pterm.FgLightWhite),
				BulletStyle: pterm.NewStyle(pterm.FgLightWhite),
				Text:        fmt.Sprintf("Status: %s", cmdutils.GetENIStatusColor(eni.Status)),
			})

			if eni.EC2Attachment != nil {
				bulletList = append(bulletList, pterm.BulletListItem{
					Level:       2,
					TextStyle:   pterm.NewStyle(pterm.FgLightWhite),
					BulletStyle: pterm.NewStyle(pterm.FgLightWhite),
					Text:        "Attached to EC2 instance:",
				})
				bulletList = append(bulletList, pterm.BulletListItem{
					Level:       3,
					TextStyle:   pterm.NewStyle(pterm.FgCyan),
					BulletStyle: pterm.NewStyle(pterm.FgCyan),
					Text:        fmt.Sprintf("%s", eni.EC2Attachment.InstanceId),
				})
			}

			if eni.LambdaAttachment != nil {
				bulletList = append(bulletList, pterm.BulletListItem{
					Level:       2,
					TextStyle:   pterm.NewStyle(pterm.FgLightWhite),
					BulletStyle: pterm.NewStyle(pterm.FgLightWhite),
					Text:        "Associated to Lambda Function:",
				})

				if eni.LambdaAttachment.IsRemoved {
					bulletList = append(bulletList, pterm.BulletListItem{
						Level:       3,
						TextStyle:   pterm.NewStyle(pterm.FgLightYellow),
						BulletStyle: pterm.NewStyle(pterm.FgLightYellow),
						Text: fmt.Sprintf("%s - Note: This function was already removed. Please wait 15-20 minutes for the ENI to be removed by AWS.",
							eni.LambdaAttachment.Name),
					})
				} else {
					bulletList = append(bulletList, pterm.BulletListItem{
						Level:       3,
						TextStyle:   pterm.NewStyle(pterm.FgLightYellow),
						BulletStyle: pterm.NewStyle(pterm.FgLightYellow),
						Text:        fmt.Sprintf("%s (%s)", eni.LambdaAttachment.Name, *eni.LambdaAttachment.Arn),
					})
				}
			}

			if eni.ECSAttachment != nil {
				bulletList = append(bulletList, pterm.BulletListItem{
					Level:       2,
					TextStyle:   pterm.NewStyle(pterm.FgLightWhite),
					BulletStyle: pterm.NewStyle(pterm.FgLightWhite),
					Text:        "Associated to ECS Container:",
				})

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

				container := "unknown"
				if eni.ECSAttachment.ContainerName != nil {
					container = *eni.ECSAttachment.ContainerName
				}

				if eni.ECSAttachment.IsRemoved {
					bulletList = append(bulletList, pterm.BulletListItem{
						Level:       3,
						TextStyle:   pterm.NewStyle(pterm.FgLightYellow),
						BulletStyle: pterm.NewStyle(pterm.FgLightYellow),
						Text: fmt.Sprintf("%s\\%s Note: the task was already removed. Please try to remove the ENI manually!",
							cluster, service)})
				} else {
					bulletList = append(bulletList, pterm.BulletListItem{
						Level:       3,
						TextStyle:   pterm.NewStyle(pterm.FgCyan),
						BulletStyle: pterm.NewStyle(pterm.FgCyan),
						Text: fmt.Sprintf("%s\\%s\\%s (%s)",
							cluster, service, container, taskArn),
					})
				}
			}

			if eni.ELBAttachment != nil {
				bulletList = append(bulletList, pterm.BulletListItem{
					Level:       2,
					TextStyle:   pterm.NewStyle(pterm.FgLightWhite),
					BulletStyle: pterm.NewStyle(pterm.FgLightWhite),
					Text:        "Associated to Load Balancer:",
				})

				if eni.ELBAttachment.IsRemoved {
					bulletList = append(bulletList, pterm.BulletListItem{
						Level:       3,
						TextStyle:   pterm.NewStyle(pterm.FgLightYellow),
						BulletStyle: pterm.NewStyle(pterm.FgLightYellow),
						Text: fmt.Sprintf("%s Note: the load balancer was removed. Please try to remove the ENI manually!",
							eni.ELBAttachment.Name),
					})
				} else {
					bulletList = append(bulletList, pterm.BulletListItem{
						Level:       3,
						TextStyle:   pterm.NewStyle(pterm.FgLightYellow),
						BulletStyle: pterm.NewStyle(pterm.FgLightYellow),
						Text: fmt.Sprintf("%s (%s)",
							eni.ELBAttachment.Name, *eni.ELBAttachment.Arn),
					})
				}
			}

			if eni.VPCEAttachment != nil {
				bulletList = append(bulletList, pterm.BulletListItem{
					Level:       2,
					TextStyle:   pterm.NewStyle(pterm.FgLightWhite),
					BulletStyle: pterm.NewStyle(pterm.FgLightWhite),
					Text:        "Associated to VPC Endpoint:",
				})

				if eni.VPCEAttachment.IsRemoved {
					bulletList = append(bulletList, pterm.BulletListItem{
						Level:       3,
						TextStyle:   pterm.NewStyle(pterm.FgLightYellow),
						BulletStyle: pterm.NewStyle(pterm.FgLightYellow),
						Text: fmt.Sprintf("%s Note: the VPC Endpoint was removed. Please try to remove the ENI manually!",
							*eni.VPCEAttachment.Id),
					})
				} else {
					bulletList = append(bulletList, pterm.BulletListItem{
						Level:       3,
						TextStyle:   pterm.NewStyle(pterm.FgLightYellow),
						BulletStyle: pterm.NewStyle(pterm.FgLightYellow),
						Text: fmt.Sprintf("%s (%s)",
							*eni.VPCEAttachment.ServiceName, *eni.VPCEAttachment.Id),
					})
				}
			}

			if len(eni.RDSAttachments) > 0 {
				bulletList = append(bulletList, pterm.BulletListItem{
					Level:       2,
					TextStyle:   pterm.NewStyle(pterm.FgLightWhite),
					BulletStyle: pterm.NewStyle(pterm.FgLightWhite),
					Text:        "Associated to RDS instance (might be inaccurate):",
				})
				for _, attachment := range eni.RDSAttachments {
					bulletList = append(bulletList, pterm.BulletListItem{
						Level:       3,
						TextStyle:   pterm.NewStyle(pterm.FgLightYellow),
						BulletStyle: pterm.NewStyle(pterm.FgLightYellow),
						Text: fmt.Sprintf("%s",
							attachment.Identifier),
					})
				}
			}
		}
	}

	if len(sg.RuleReferences) > 0 {
		bulletList = append(bulletList, pterm.BulletListItem{
			Level:       0,
			TextStyle:   pterm.NewStyle(pterm.FgLightWhite),
			BulletStyle: pterm.NewStyle(pterm.FgLightWhite),
			Text:        "Referenced by the following Security Groups as an Inbound/Outbound rule:",
		})

		for _, ruleRef := range sg.RuleReferences {
			bulletList = append(bulletList, pterm.BulletListItem{Level: 1,
				TextStyle:   pterm.NewStyle(pterm.FgCyan),
				BulletStyle: pterm.NewStyle(pterm.FgCyan),
				Text:        fmt.Sprintf("%s", ruleRef)})
		}
	}

	return pterm.DefaultBulletList.WithItems(bulletList).Render()
}

func getReasonsAgainstRemoval(sg types.SecurityGroupDetails) []string {
	reasons := make([]string, 0)
	if !sg.CanBeRemoved() {
		if sg.Default {
			reasons = append(reasons, fmt.Sprintf("Security Group is Default in VPC %s", sg.VpcId))
		}
		if len(sg.UsedBy) > 0 {
			reasons = append(reasons, "Security Group is used by an Elastic Network Interface (ENI)")
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
