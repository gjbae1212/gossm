package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/service/ssm"
	. "github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	startSessionCommand = &cobra.Command{
		Use:   "start",
		Short: "Exec `start-session` under AWS SSM with interactive CLI",
		Long:  "Exec `start-session` under AWS SSM with interactive CLI",
		PreRun: func(cmd *cobra.Command, args []string) {
			// set region
			if err := setRegion(); err != nil {
				fmt.Println(Red(err))
				os.Exit(1)
			}

			// set target
			if err := setTarget(); err != nil {
				fmt.Println(Red(err))
				os.Exit(1)
			}
			printReady("start-session")
		},
		Run: func(cmd *cobra.Command, args []string) {
			region := viper.GetString("region")
			profile := viper.GetString("profile")
			target := viper.GetString("target")
			input := &ssm.StartSessionInput{Target: &target}

			// create session
			sess, endpoint, err := createStartSession(region, input)
			if err != nil {
				fmt.Println(Red(err))
				os.Exit(1)
			}

			sessJson, err := json.Marshal(sess)
			if err != nil {
				fmt.Println(Red(err))
				os.Exit(1)
			}

			paramsJson, err := json.Marshal(input)
			if err != nil {
				fmt.Println(Red(err))
				os.Exit(1)
			}

			// call session-manager-plugin
			if err := callSubprocess("session-manager-plugin", string(sessJson),
				region, "StartSession", profile, string(paramsJson), endpoint); err != nil {
				fmt.Println(Red(err))
				// delete Session
				if err := deleteStartSession(region, *sess.SessionId); err != nil {
					fmt.Println(Red(err))
				}
				os.Exit(1)
			}
		},
	}
)

func init() {
	// add sub command
	rootCmd.AddCommand(startSessionCommand)
}
