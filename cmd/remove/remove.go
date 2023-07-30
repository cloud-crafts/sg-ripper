package remove

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"sg-ripper/pkg/core"
	"sg-ripper/pkg/core/utils"
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

	resultCh := make(chan utils.Result[string])
	err := core.RemoveSecurityGroupsAsync(cmd.Context(), ids, region, profile, resultCh)
	if err != nil {
		pterm.Error.Println(err)
		return
	}

	for res := range resultCh {
		if res.Err != nil {
			pterm.Error.Println(res.Err)
		} else {
			pterm.Info.Println("Removed Security Group with ID of " + pterm.LightGreen(res.Data))
		}
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
