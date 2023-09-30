package cmd

import (
	"fmt"
	"github.com/cloud-crafts/sg-ripper/cmd/list"
	"github.com/cloud-crafts/sg-ripper/cmd/listeni"
	"github.com/cloud-crafts/sg-ripper/cmd/remove"
	"github.com/cloud-crafts/sg-ripper/cmd/removeeni"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"os"
)

// Execute - parse CLI arguments and execute command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		pterm.Error.Printf("There was an error while executing sg-ripper: %s", err)
		os.Exit(1)
	}
}

var (
	appVersion = "development"
	gitCommit  = "commit"
	rootCmd    = &cobra.Command{
		Use:              "sg-ripper",
		Short:            "Security Group and ENI cleaner.",
		Version:          fmt.Sprintf("%s (%s)", appVersion, gitCommit),
		TraverseChildren: true,
	}

	region  string
	profile string
)

func init() {
	includeValidateFlags(rootCmd)
	rootCmd.AddCommand(list.Cmd)
	rootCmd.AddCommand(listeni.Cmd)
	rootCmd.AddCommand(remove.Cmd)
	rootCmd.AddCommand(removeeni.Cmd)

	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

func includeValidateFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&region, "region", "",
		"[Optional] AWS Region.")
	cmd.PersistentFlags().StringVar(&profile, "profile", "",
		"[Optional] Profile.")
}
