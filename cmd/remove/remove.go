package remove

import (
	"github.com/spf13/cobra"
	"sg-ripper/pkg/core"
)

var (
	Cmd = &cobra.Command{
		Use:   "remove",
		Short: "Remove unused security groups.",
		Long:  "",
		Run:   runRemove,
	}

	sg *[]string
)

func runRemove(cmd *cobra.Command, args []string) {
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

	err := core.RemoveSecurityGroups(cmd.Context(), ids, region, profile)
	if err != nil {
		cmd.PrintErrf("Error: %s", err)
		return
	}
}

func init() {
	includeValidateFlags(Cmd)
}

func includeValidateFlags(cmd *cobra.Command) {
	sg = cmd.Flags().StringSlice("sg", nil,
		"Security Group Id to be deleted. It can accept multiple values divided by comma. "+
			"Default: none")
}
