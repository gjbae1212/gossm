package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	sshCommand = &cobra.Command{
		Use:   "ssh",
		Short: "Exec `ssh` under AWS SSM with interactive CLI",
		Long:  "Exec `ssh` under AWS SSM with interactive CLI",
		PreRun: func(cmd *cobra.Command, args []string) {
			initCredential()
			executor.execCommand = viper.GetString("ssh-exec")
			executor.sshKey = viper.GetString("ssh-identity")

			// set region
			if err := setRegion(credential); err != nil {
				panicRed(err)
			}

			// set target
			if err := setSSH(credential, executor); err != nil {
				panicRed(err)
			}

			printReady("ssh", credential, executor)
			color.Cyan("ssh " + executor.execCommand)
		},
		Run: func(cmd *cobra.Command, args []string) {
			docName := "AWS-StartSSHSession"
			port := "22"
			input := &ssm.StartSessionInput{
				DocumentName: &docName,
				Parameters:   map[string][]*string{"portNumber": []*string{&port}},
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

			// call ssh
			proxy := fmt.Sprintf("ProxyCommand=%s '%s' %s %s %s '%s' %s",
				viper.GetString("plugin"), string(sessJson), credential.awsRegion, "StartSession",
				credential.awsProfile, string(paramsJson), endpoint)
			sshArgs := []string{"-o", proxy}
			for _, sep := range strings.Split(executor.execCommand, " ") {
				if sep != "" {
					sshArgs = append(sshArgs, sep)
				}
			}
			if err := callSubprocess("ssh", sshArgs...); err != nil {
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
	sshCommand.Flags().StringP("exec", "e", "", "[optional] ssh $exec, ex) \"-i ex.pem ubuntu@server\"")

	sshCommand.Flags().StringP("identity", "i", "", "[optional] identity file path, ex) $HOME/.ssh/id_rsa")

	// mapping viper
	viper.BindPFlag("ssh-exec", sshCommand.Flags().Lookup("exec"))

	viper.BindPFlag("ssh-identity", sshCommand.Flags().Lookup("identity"))

	rootCmd.AddCommand(sshCommand)
}
