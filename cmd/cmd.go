package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"os"
	"sg-ripper/cmd/list"
	"sg-ripper/cmd/listEni"
	"sg-ripper/cmd/remove"
)

// Execute - parse CLI arguments and execute command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Println("There was an error while executing sg-ripper!", err)
		os.Exit(1)
	}
}

var (
	appVersion = "development"
	gitCommit  = "commit"
	rootCmd    = &cobra.Command{
		Use:              "sg-ripper",
		Short:            "sg-ripper.",
		Long:             ``,
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
}

func includeValidateFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&region, "region", "",
		"[Optional] AWS Region.")
	cmd.PersistentFlags().StringVar(&profile, "profile", "",
		"[Optional] Profile.")
}
