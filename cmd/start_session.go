package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	startSessionCommand = &cobra.Command{
		Use:   "start-session",
		Short: "Start `start-session` under AWS SSM with interactive CLI",
		Long:  "Start `start-session` under AWS SSM with interactive CLI",
		PreRun: func(cmd *cobra.Command, args []string) {
			// set region
			if err := setEnvRegion(); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			// set instance
			if err := setInstance(); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
		},
	}
)

func init() {
	// start-session additional flag
	startSessionCommand.Flags().StringP("instance", "i", "", "[optional] it is instance-id of server in AWS that  would like to something")

	// mapping viper
	viper.BindPFlag("instance", startSessionCommand.Flags().Lookup("instance"))

	// add sub command
	rootCmd.AddCommand(startSessionCommand)
}
