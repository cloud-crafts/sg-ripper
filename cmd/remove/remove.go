package remove

import (
	"fmt"
	"github.com/cloud-crafts/sg-ripper/pkg/core"
	"github.com/cloud-crafts/sg-ripper/pkg/core/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var (
	Cmd = &cobra.Command{
		Use:   "remove",
		Short: "Remove unused Security Groups.",
		Run:   runRemove,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			regionFlag := cmd.Flags().Lookup("region")
			if regionFlag != nil {
				region = regionFlag.Value.String()
			}

			profileFlag := cmd.Flags().Lookup("profile")
			if profileFlag != nil {
				profile = profileFlag.Value.String()
			}

			if len(*sg) <= 0 {
				return fmt.Errorf("no Security Group ID provided")
			}

			return nil
		},
	}

	sg      *[]string
	region  string
	profile string
)

func runRemove(cmd *cobra.Command, args []string) {
	resultCh := make(chan utils.Result[string])
	err := core.RemoveSecurityGroupsAsync(cmd.Context(), *sg, region, profile, resultCh)
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
