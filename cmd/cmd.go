package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"os"
	"sg-ripper/cmd/list"
)

// Execute - parse CLI arguments and execute command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Println("There was an error while executing cert-ripper!", err)
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
)

func init() {
	rootCmd.AddCommand(list.Cmd)
}