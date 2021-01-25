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
	scpCommand = &cobra.Command{
		Use:   "scp",
		Short: "Exec `scp` under AWS SSM with interactive CLI",
		Long:  "Exec `scp` under AWS SSM with interactive CLI",
		PreRun: func(cmd *cobra.Command, args []string) {
			initCredential()
			executor.execCommand = viper.GetString("scp-exec")
			// set region
			if err := setRegion(credential); err != nil {
				panicRed(err)
			}
			// set scp
			if err := setSCP(credential, executor); err != nil {
				panicRed(err)
			}

			printReady("scp", credential, executor)
			color.Cyan("scp " + executor.execCommand)
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

			// call scp
			proxy := fmt.Sprintf("ProxyCommand=%s '%s' %s %s %s '%s' %s",
				viper.GetString("plugin"), string(sessJson), credential.awsRegion, "StartSession", credential.awsProfile, string(paramsJson), endpoint)
			scpArgs := []string{"-o", proxy}
			for _, sep := range strings.Split(executor.execCommand, " ") {
				if sep != "" {
					scpArgs = append(scpArgs, sep)
				}
			}
			if err := callSubprocess("scp", scpArgs...); err != nil {
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
	scpCommand.Flags().StringP("exec", "e", "", "[required] scp $exec, ex) \"-i ex.pem ubuntu@server:/home/ex.txt ex.txt\"")
	viper.BindPFlag("scp-exec", scpCommand.Flags().Lookup("exec"))
	rootCmd.AddCommand(scpCommand)
}
