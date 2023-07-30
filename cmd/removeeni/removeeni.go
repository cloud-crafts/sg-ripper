package removeeni

import (
	"fmt"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"sg-ripper/pkg/core"
	"sg-ripper/pkg/core/utils"
)

var (
	Cmd = &cobra.Command{
		Use:   "remove-eni",
		Short: "Remove unused Elastic Network Interfaces.",
		RunE:  runRemoveENI,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			regionFlag := cmd.Flags().Lookup("region")
			if regionFlag != nil {
				region = regionFlag.Value.String()
			}

			profileFlag := cmd.Flags().Lookup("profile")
			if profileFlag != nil {
				profile = profileFlag.Value.String()
			}

			if len(*eni) <= 0 {
				return fmt.Errorf("no Security Group ID provided")
			}

			return nil
		},
	}

	eni     *[]string
	region  string
	profile string
)

func runRemoveENI(cmd *cobra.Command, args []string) error {
	resultCh := make(chan utils.Result[string])
	err := core.RemoveENIAsync(cmd.Context(), *eni, region, profile, resultCh)
	if err != nil {
		return err
	}

	for res := range resultCh {
		if res.Err != nil {
			pterm.Error.Println(res.Err)
		} else {
			pterm.Info.Println("Removed Elastic Network Interface with ID of " + pterm.LightGreen(res.Data))
		}
	}

	return nil
}

func init() {
	includeValidateFlags(Cmd)
}

func includeValidateFlags(cmd *cobra.Command) {
	eni = cmd.Flags().StringSlice("eni", nil,
		"Network Interface ID to be deleted. It can accept multiple values divided by comma. "+
			"Default: none")
}
