package cmd

import "github.com/spf13/cobra"

var (
	startSessionCommand = &cobra.Command{
		Use:   "start-session",
		Short: "",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {},
	}
)

func init() {
	rootCmd.AddCommand(startSessionCommand)
}
