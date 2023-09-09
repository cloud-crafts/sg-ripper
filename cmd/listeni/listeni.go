package listeni

import (
	"fmt"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"sg-ripper/cmd/cmdutils"
	"sg-ripper/pkg/core"
	coreTypes "sg-ripper/pkg/core/types"
	"strings"
)

var (
	Cmd = &cobra.Command{
		Use:   "list-eni",
		Short: "List Elastic Network Interfaces with Details",
		RunE:  runList,
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

	enis, err := core.ListNetworkInterfaces(cmd.Context(), *sg, filters, region, profile)
	if err != nil {
		return err
	}
	for _, eni := range enis {
		err := printEniUsage(eni)
		if err != nil {
			return err
		}
	}

	return nil
}

func printEniUsage(eni coreTypes.NetworkInterfaceDetails) error {
	pterm.DefaultSection.Printf("%s", eni.Id)
	var bulletList []pterm.BulletListItem
	if eni.Description != nil {
		bulletList = append(bulletList, pterm.BulletListItem{
			Level:       1,
			TextStyle:   pterm.NewStyle(pterm.FgLightWhite),
			BulletStyle: pterm.NewStyle(pterm.FgLightWhite),
			Text:        fmt.Sprintf("Description: %s", pterm.Cyan(*eni.Description)),
		})
	}

	bulletList = append(bulletList, pterm.BulletListItem{
		Level:       1,
		TextStyle:   pterm.NewStyle(pterm.FgLightWhite),
		BulletStyle: pterm.NewStyle(pterm.FgLightWhite),
		Text:        fmt.Sprintf("Type: %s", pterm.Cyan(eni.Type)),
	})
	bulletList = append(bulletList, pterm.BulletListItem{
		Level:       1,
		TextStyle:   pterm.NewStyle(pterm.FgLightWhite),
		BulletStyle: pterm.NewStyle(pterm.FgLightWhite),
		Text:        fmt.Sprintf("Private IP Address: %s", pterm.Cyan(eni.PrivateIPAddress)),
	})

	if len(eni.SecondaryPrivateIPAddresses) > 0 {
		bulletList = append(bulletList, pterm.BulletListItem{
			Level:       1,
			TextStyle:   pterm.NewStyle(pterm.FgLightWhite),
			BulletStyle: pterm.NewStyle(pterm.FgLightWhite),
			Text: fmt.Sprintf("Secondary Private IP Addresses: %s",
				pterm.Cyan(strings.Join(eni.SecondaryPrivateIPAddresses[:], ", "))),
		})
	}

	bulletList = append(bulletList, pterm.BulletListItem{
		Level:       1,
		TextStyle:   pterm.NewStyle(pterm.FgLightWhite),
		BulletStyle: pterm.NewStyle(pterm.FgLightWhite),
		Text:        fmt.Sprintf("Managed By AWS: %s", cmdutils.GetENIManagedByAWSText(eni.ManagedByAWS)),
	})
	bulletList = append(bulletList, pterm.BulletListItem{
		Level:       1,
		TextStyle:   pterm.NewStyle(pterm.FgLightWhite),
		BulletStyle: pterm.NewStyle(pterm.FgLightWhite),
		Text:        fmt.Sprintf("Status: %s", cmdutils.GetENIStatusColor(eni.Status)),
	})

	if eni.EC2Attachment != nil {
		bulletList = append(bulletList, pterm.BulletListItem{
			Level:       1,
			TextStyle:   pterm.NewStyle(pterm.FgLightWhite),
			BulletStyle: pterm.NewStyle(pterm.FgLightWhite),
			Text:        "Attached to EC2 instance:"},
		)
		bulletList = append(bulletList, pterm.BulletListItem{
			Level:       2,
			TextStyle:   pterm.NewStyle(pterm.FgLightWhite),
			BulletStyle: pterm.NewStyle(pterm.FgLightWhite),
			Text:        fmt.Sprintf("%s", pterm.Cyan(eni.EC2Attachment.InstanceId)),
		})
	}

	if eni.LambdaAttachment != nil {
		bulletList = append(bulletList, pterm.BulletListItem{
			Level:       1,
			TextStyle:   pterm.NewStyle(pterm.FgLightWhite),
			BulletStyle: pterm.NewStyle(pterm.FgLightWhite),
			Text:        "Associated to Lambda Function:",
		})

		if eni.LambdaAttachment.IsRemoved {
			bulletList = append(bulletList, pterm.BulletListItem{
				Level:       2,
				TextStyle:   pterm.NewStyle(pterm.FgLightYellow),
				BulletStyle: pterm.NewStyle(pterm.FgLightYellow),
				Text: fmt.Sprintf(
					"%s - Note: This function was already removed. Please wait 15-20 minutes for the ENI to be removed by AWS.",
					eni.LambdaAttachment.Name),
			})
		} else {
			bulletList = append(bulletList, pterm.BulletListItem{
				Level:       2,
				TextStyle:   pterm.NewStyle(pterm.FgLightYellow),
				BulletStyle: pterm.NewStyle(pterm.FgLightYellow),
				Text: fmt.Sprintf("%s (%s)",
					eni.LambdaAttachment.Name, *eni.LambdaAttachment.Arn),
			})
		}
	}
	if eni.ECSAttachment != nil {
		bulletList = append(bulletList, pterm.BulletListItem{
			Level:       1,
			TextStyle:   pterm.NewStyle(pterm.FgLightWhite),
			BulletStyle: pterm.NewStyle(pterm.FgLightWhite),
			Text:        "Associated to ECS Container:",
		})

		cluster := "unknown"
		if eni.ECSAttachment.ClusterArn != nil {
			cluster = *eni.ECSAttachment.ClusterArn
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
				Level:       2,
				TextStyle:   pterm.NewStyle(pterm.FgLightYellow),
				BulletStyle: pterm.NewStyle(pterm.FgLightYellow),
				Text:        "Note: the associated task was already removed. Please try to remove the ENI manually!"})
		} else {
			bulletList = append(bulletList, pterm.BulletListItem{
				Level:       2,
				TextStyle:   pterm.NewStyle(pterm.FgLightWhite),
				BulletStyle: pterm.NewStyle(pterm.FgLightWhite),
				Text:        fmt.Sprintf("Cluster: %s", pterm.Cyan(cluster)),
			})
			bulletList = append(bulletList, pterm.BulletListItem{
				Level:       2,
				TextStyle:   pterm.NewStyle(pterm.FgLightWhite),
				BulletStyle: pterm.NewStyle(pterm.FgLightWhite),
				Text:        fmt.Sprintf("Task: %s", pterm.Cyan(taskArn)),
			})
			bulletList = append(bulletList, pterm.BulletListItem{
				Level:       2,
				TextStyle:   pterm.NewStyle(pterm.FgLightWhite),
				BulletStyle: pterm.NewStyle(pterm.FgLightWhite),
				Text:        fmt.Sprintf("Container: %s", pterm.Cyan(container)),
			})
		}
	}

	if eni.ELBAttachment != nil {
		bulletList = append(bulletList, pterm.BulletListItem{
			Level:       1,
			TextStyle:   pterm.NewStyle(pterm.FgLightWhite),
			BulletStyle: pterm.NewStyle(pterm.FgLightWhite),
			Text:        "Associated to Load Balancer:",
		})

		if eni.ELBAttachment.IsRemoved {
			bulletList = append(bulletList, pterm.BulletListItem{
				Level:       2,
				TextStyle:   pterm.NewStyle(pterm.FgLightYellow),
				BulletStyle: pterm.NewStyle(pterm.FgLightYellow),
				Text:        fmt.Sprintf("%s Note: the load balancer was removed. Please try to remove the ENI manually!", eni.ELBAttachment.Name),
			})
		} else {
			bulletList = append(bulletList, pterm.BulletListItem{
				Level:       2,
				TextStyle:   pterm.NewStyle(pterm.FgLightYellow),
				BulletStyle: pterm.NewStyle(pterm.FgLightYellow),
				Text:        fmt.Sprintf("%s (%s)", eni.ELBAttachment.Name, *eni.ELBAttachment.Arn),
			})
		}

	}
	if eni.VPCEAttachment != nil {
		bulletList = append(bulletList, pterm.BulletListItem{
			Level:       1,
			TextStyle:   pterm.NewStyle(pterm.FgLightWhite),
			BulletStyle: pterm.NewStyle(pterm.FgLightWhite),
			Text:        "Associated to VPC Endpoint:",
		})

		if eni.VPCEAttachment.IsRemoved {
			bulletList = append(bulletList, pterm.BulletListItem{
				Level:       2,
				TextStyle:   pterm.NewStyle(pterm.FgLightYellow),
				BulletStyle: pterm.NewStyle(pterm.FgLightYellow),
				Text:        fmt.Sprintf("%s Note: the VPC Endpoint was removed. Please try to remove the ENI manually!", *eni.VPCEAttachment.Id),
			})
		} else {
			bulletList = append(bulletList, pterm.BulletListItem{
				Level:       2,
				TextStyle:   pterm.NewStyle(pterm.FgLightYellow),
				BulletStyle: pterm.NewStyle(pterm.FgLightYellow),
				Text:        fmt.Sprintf("%s (%s)", *eni.VPCEAttachment.ServiceName, *eni.VPCEAttachment.Id),
			})
		}
	}

	if len(eni.RDSAttachments) > 0 {
		bulletList = append(bulletList, pterm.BulletListItem{
			Level:       1,
			TextStyle:   pterm.NewStyle(pterm.FgLightWhite),
			BulletStyle: pterm.NewStyle(pterm.FgLightWhite),
			Text:        "Associated to RDS instance (might be inaccurate):",
		})

		for _, attachment := range eni.RDSAttachments {
			bulletList = append(bulletList, pterm.BulletListItem{
				Level:       2,
				TextStyle:   pterm.NewStyle(pterm.FgLightYellow),
				BulletStyle: pterm.NewStyle(pterm.FgLightYellow),
				Text: fmt.Sprintf("%s",
					attachment.Identifier),
			})
		}
	}

	if len(eni.SecurityGroupIdentifiers) > 0 {
		bulletList = append(bulletList, pterm.BulletListItem{
			Level:       1,
			TextStyle:   pterm.NewStyle(pterm.FgLightWhite),
			BulletStyle: pterm.NewStyle(pterm.FgLightWhite),
			Text:        "Security Groups:",
		})

		for _, identifier := range eni.SecurityGroupIdentifiers {
			name := "<no-name>"
			if identifier.Name != nil {
				name = *identifier.Name
			}
			bulletList = append(bulletList, pterm.BulletListItem{
				Level:       2,
				TextStyle:   pterm.NewStyle(pterm.FgLightWhite),
				BulletStyle: pterm.NewStyle(pterm.FgLightWhite),
				Text:        fmt.Sprintf("%s (%s)", pterm.LightBlue(name), pterm.LightMagenta(identifier.Id))})
		}
	}

	return pterm.DefaultBulletList.WithItems(bulletList).Render()
}

func init() {
	includeValidateFlags(Cmd)
}

func includeValidateFlags(cmd *cobra.Command) {
	sg = cmd.Flags().StringSlice("eni", nil,
		"[Optional] Elastic Network Interface Id to be filtered. It can accept multiple values divided by comma. "+
			"Default: none (if none is specified all security groups will be retrieved)")
	cmd.Flags().BoolVarP(&used, "used", "u", false,
		"[Optional] List all network interfaces.")
	cmd.Flags().BoolVarP(&unused, "unused", "n", false,
		"[Optional] List unused network interfaces.")
}
