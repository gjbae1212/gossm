package cmd

import (
	"encoding/json"

	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	fwdCommand = &cobra.Command{
		Use:   "fwd",
		Short: "Exec `fwd` under AWS SSM with interactive CLI",
		Long:  "Exec `fwd` under AWS SSM with interactive CLI",
		PreRun: func(cmd *cobra.Command, args []string) {
			initCredential()
			executor.remotePort = viper.GetString("remote-port")
			executor.localPort = viper.GetString("local-port")

			// set region
			if err := setRegion(credential); err != nil {
				panicRed(err)
			}

			// set target
			if err := setTarget(credential, executor); err != nil {
				panicRed(err)
			}

			// set ports
			if err := setFwdPorts(credential, executor); err != nil {
				panicRed(err)
			}

			printReady("start-port-forwarding", credential, executor)
		},
		Run: func(cmd *cobra.Command, args []string) {
			docName := "AWS-StartPortForwardingSession"

			input := &ssm.StartSessionInput{
				DocumentName: &docName,
				Parameters:   map[string][]*string{"portNumber": []*string{&executor.remotePort}, "localPortNumber": []*string{&executor.localPort}},
				Target:       &executor.target,
			}

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
	fwdCommand.Flags().StringP("remote", "z", "", "[optional] remote port to forward to, ex) - 8080")
	fwdCommand.Flags().StringP("local", "l", "", "[optional] local port to use, ex) \"-l 1234\"")

	// mapping viper
	viper.BindPFlag("remote-port", fwdCommand.Flags().Lookup("remote"))
	viper.BindPFlag("local-port", fwdCommand.Flags().Lookup("local"))

	rootCmd.AddCommand(fwdCommand)
}
