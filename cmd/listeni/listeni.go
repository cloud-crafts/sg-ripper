package listeni

import (
	"fmt"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"sg-ripper/pkg/core"
	coreTypes "sg-ripper/pkg/core/types"
)

var (
	Cmd = &cobra.Command{
		Use:   "list-eni",
		Short: "List Elastic Network Interfaces with Details",
		Long:  "",
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
		bulletList = append(bulletList, pterm.BulletListItem{Level: 1, Text: fmt.Sprintf("Description: %s", *eni.Description)})
	}
	bulletList = append(bulletList, pterm.BulletListItem{Level: 1, Text: fmt.Sprintf("Type: %s", eni.Type)})
	bulletList = append(bulletList, pterm.BulletListItem{Level: 1, Text: fmt.Sprintf("Private IP Address: %s", eni.PrivateIPAddress)})
	bulletList = append(bulletList, pterm.BulletListItem{Level: 1, Text: fmt.Sprintf("Managed By AWS: %t", eni.ManagedByAWS)})
	bulletList = append(bulletList, pterm.BulletListItem{Level: 1, Text: fmt.Sprintf("Status: %s", eni.Status)})
	if eni.EC2Attachment != nil {
		bulletList = append(bulletList, pterm.BulletListItem{Level: 1, Text: "Attached to EC2 instance:"})
		bulletList = append(bulletList, pterm.BulletListItem{Level: 2, Text: fmt.Sprintf("%s", eni.EC2Attachment.InstanceId)})
	}
	if eni.LambdaAttachment != nil {
		bulletList = append(bulletList, pterm.BulletListItem{Level: 1, Text: "Associated to Lambda Function:"})
		if eni.LambdaAttachment.IsRemoved {
			bulletList = append(bulletList, pterm.BulletListItem{Level: 2, Text: fmt.Sprintf(
				"%s - Note: This function was already removed. Please wait 15-20 minutes for the ENI to be removed by AWS.",
				eni.LambdaAttachment.Name)})
		} else {
			bulletList = append(bulletList, pterm.BulletListItem{Level: 2, Text: fmt.Sprintf("%s (%s)",
				eni.LambdaAttachment.Name, *eni.LambdaAttachment.Arn)})
		}
	}
	if eni.ECSAttachment != nil {
		bulletList = append(bulletList, pterm.BulletListItem{Level: 1, Text: "Associated to ECS Container:"})

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
			bulletList = append(bulletList, pterm.BulletListItem{Level: 2,
				Text: fmt.Sprintf("%s\\%s Note: the task was already removed. Please try to remove the ENI manually!",
					cluster, service)})
		} else {
			bulletList = append(bulletList, pterm.BulletListItem{Level: 2, Text: fmt.Sprintf("%s\\%s (%s)",
				cluster, service, taskArn)})
		}
	}
	if eni.ELBAttachment != nil {
		bulletList = append(bulletList, pterm.BulletListItem{Level: 1, Text: "Associated to Load Balancer:"})
		if eni.ELBAttachment.IsRemoved {
			bulletList = append(bulletList, pterm.BulletListItem{Level: 2,
				Text: fmt.Sprintf("%s Note: the load balancer was removed. Please try to remove the ENI manually!",
					eni.ELBAttachment.Name)})
		} else {
			bulletList = append(bulletList, pterm.BulletListItem{Level: 2, Text: fmt.Sprintf("%s (%s)",
				eni.ELBAttachment.Name, *eni.ELBAttachment.Arn)})
		}
	}
	if eni.VPCEAttachment != nil {
		bulletList = append(bulletList, pterm.BulletListItem{Level: 1, Text: "Associated to VPC Endpoint:"})
		if eni.VPCEAttachment.IsRemoved {
			bulletList = append(bulletList, pterm.BulletListItem{Level: 2,
				Text: fmt.Sprintf("%s Note: the VPC Endpoint was removed. Please try to remove the ENI manually!",
					*eni.VPCEAttachment.Id)})
		} else {
			bulletList = append(bulletList, pterm.BulletListItem{Level: 2, Text: fmt.Sprintf("%s (%s)",
				*eni.VPCEAttachment.ServiceName, *eni.VPCEAttachment.Id)})
		}
	}

	if len(eni.RDSAttachments) > 0 {
		bulletList = append(bulletList, pterm.BulletListItem{Level: 1, Text: "Associated to RDS instance (might be inaccurate):"})
		for _, attachment := range eni.RDSAttachments {
			bulletList = append(bulletList, pterm.BulletListItem{Level: 2, Text: fmt.Sprintf("%s",
				attachment.Identifier)})
		}
	}

	if len(eni.SecurityGroupIdentifiers) > 0 {
		bulletList = append(bulletList, pterm.BulletListItem{Level: 1, Text: "Security Groups:"})
		for _, identifier := range eni.SecurityGroupIdentifiers {
			name := "<no-name>"
			if identifier.Name != nil {
				name = *identifier.Name
			}
			bulletList = append(bulletList, pterm.BulletListItem{Level: 2, Text: fmt.Sprintf("%s (%s)", name, identifier.Id)})
		}
	}

	err := pterm.DefaultBulletList.WithItems(bulletList).Render()
	if err != nil {
		return err
	}

	return nil
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
