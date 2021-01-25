package cmd

import (
	"encoding/json"

	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	startSessionCommand = &cobra.Command{
		Use:   "start",
		Short: "Exec `start-session` under AWS SSM with interactive CLI",
		Long:  "Exec `start-session` under AWS SSM with interactive CLI",
		PreRun: func(cmd *cobra.Command, args []string) {
			initCredential()

			// set region
			if err := setRegion(credential); err != nil {
				panicRed(err)
			}

			// set target
			if err := setTarget(credential, executor); err != nil {
				panicRed(err)
			}

			printReady("start-session", credential, executor)
		},
		Run: func(cmd *cobra.Command, args []string) {
			input := &ssm.StartSessionInput{Target: &executor.target}

			// create session
			sess, endpoint, err := createStartSession(credential, input)
			if err != nil {
				panicRed(err)
			}

			sessJson, err := json.Marshal(sess)
			if err != nil {
				panicRed(err)
			}

			paramsJson, err := json.Marshal(input)
			if err != nil {
				panicRed(err)
			}

			// call session-manager-plugin
			if err := callSubprocess(viper.GetString("plugin"), string(sessJson),
				credential.awsRegion, "StartSession", credential.awsProfile, string(paramsJson), endpoint); err != nil {
				color.Red("%v", err)
			}
			// delete Session
			if err := deleteStartSession(credential, *sess.SessionId); err != nil {
				color.Red("%v", err)
			}
		},
	}
)

func init() {
	// add sub command
	rootCmd.AddCommand(startSessionCommand)
}
