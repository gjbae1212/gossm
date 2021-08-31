package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/fatih/color"
	"github.com/gjbae1212/gossm/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	sshCommand = &cobra.Command{
		Use:   "ssh",
		Short: "Exec `ssh` under AWS SSM with interactive CLI",
		Long:  "Exec `ssh` under AWS SSM with interactive CLI",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			exec := strings.TrimSpace(viper.GetString("ssh-exec"))
			identity := strings.TrimSpace(viper.GetString("ssh-identity"))

			if exec != "" && identity != "" {
				panicRed(fmt.Errorf("[err] don't use both exec and identity.(must use only one)"))
			}

			var sshCommand string
			var targetName string
			if exec == "" {
				target, err := internal.AskTarget(ctx, *_credential.awsConfig)
				if err != nil {
					panicRed(err)
				}
				targetName = target.Name

				sshUser, err := internal.AskUser()
				if err != nil {
					panicRed(err)
				}
				sshCommand = internal.GenerateSSHExecCommand("", identity, sshUser.Name, target.PublicDomain)
			} else {
				seps := strings.Split(exec, " ")
				lastArg := seps[len(seps)-1]
				lastArgSeps := strings.Split(lastArg, "@")
				server := lastArgSeps[len(lastArgSeps)-1]
				ips, err := net.LookupIP(server)
				if err != nil || len(ips) == 0 {
					panicRed(fmt.Errorf("[err] invalid exec command %s", exec))
				}
				ip := ips[0].String()

				instId, err := internal.FindInstanceIdByIp(ctx, *_credential.awsConfig, ip)
				if err != nil {
					panicRed(err)
				}
				if instId == "" {
					panicRed(fmt.Errorf("[err] not found matched server"))
				}
				targetName = instId
				sshCommand = internal.GenerateSSHExecCommand(exec, "", "", "")
			}

			internal.PrintReady("ssh", _credential.awsConfig.Region, targetName)
			color.Cyan("ssh " + sshCommand)

			// start session
			docName := "AWS-StartSSHSession"
			port := "22"
			input := &ssm.StartSessionInput{
				DocumentName: aws.String(docName),
				Parameters:   map[string][]string{"portNumber": []string{port}},
				Target:       aws.String(targetName),
			}

			session, err := internal.CreateStartSession(ctx, *_credential.awsConfig, input)
			if err != nil {
				panicRed(err)
			}

			sessJson, err := json.Marshal(session)
			if err != nil {
				panicRed(err)
			}

			paramsJson, err := json.Marshal(input)
			if err != nil {
				panicRed(err)
			}

			// call ssh
			proxy := fmt.Sprintf("ProxyCommand=%s '%s' %s %s %s '%s'",
				_credential.ssmPluginPath, string(sessJson), _credential.awsConfig.Region,
				"StartSession", _credential.awsProfile, string(paramsJson))
			sshArgs := []string{"-o", proxy}
			for _, sep := range strings.Split(sshCommand, " ") {
				if sep != "" {
					sshArgs = append(sshArgs, sep)
				}
			}

			if err := internal.CallProcess("ssh", sshArgs...); err != nil {
				color.Red("%v", err)
			}

			if err := internal.DeleteStartSession(ctx, *_credential.awsConfig, &ssm.TerminateSessionInput{
				SessionId: session.SessionId,
			}); err != nil {
				panicRed(err)
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
