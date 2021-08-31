package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/fatih/color"
	"github.com/gjbae1212/gossm/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	fwdCommand = &cobra.Command{
		Use:   "fwd",
		Short: "Exec `fwd` under AWS SSM with interactive CLI",
		Long:  "Exec `fwd` under AWS SSM with interactive CLI",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			var (
				target     *internal.Target
				remotePort string
				localPort  string
				err        error
			)

			// get target
			argTarget := strings.TrimSpace(viper.GetString("fwd-target"))
			if argTarget != "" {
				table, err := internal.FindInstances(ctx, *_credential.awsConfig)
				if err != nil {
					panicRed(err)
				}
				for _, t := range table {
					if t.Name == argTarget {
						target = t
						break
					}
				}
			}
			if target == nil {
				target, err = internal.AskTarget(ctx, *_credential.awsConfig)
				if err != nil {
					panicRed(err)
				}
			}

			// get port
			argRemotePort := strings.TrimSpace(viper.GetString("fwd-remote-port"))
			argLocalPort := strings.TrimSpace(viper.GetString("fwd-local-port"))
			if argRemotePort == "" {
				askPort, err := internal.AskPorts()
				if err != nil {
					panicRed(err)
				}
				remotePort = askPort.Remote
				localPort = askPort.Local
			} else {
				remotePort = argRemotePort
				localPort = argLocalPort
				if localPort == "" {
					localPort = remotePort
				}
			}
			internal.PrintReady(fmt.Sprintf("start-port-forwarding %s -> %s", localPort, remotePort), _credential.awsConfig.Region, target.Name)

			docName := "AWS-StartPortForwardingSession" // https://us-east-1.console.aws.amazon.com/systems-manager/documents/AWS-StartPortForwardingSession/description?region=us-east-1
			input := &ssm.StartSessionInput{
				DocumentName: &docName,
				Parameters: map[string][]string{
					"portNumber":      []string{remotePort},
					"localPortNumber": []string{localPort},
				},
				Target: aws.String(target.Name),
			}

			// start session
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

			if err := internal.CallProcess(_credential.ssmPluginPath, string(sessJson),
				_credential.awsConfig.Region, "StartSession",
				_credential.awsProfile, string(paramsJson)); err != nil {
				color.Red("[err] %v", err.Error())
			}

			// delete session
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
	fwdCommand.Flags().StringP("remote", "z", "", "[optional] remote port to forward to, ex) 8080")
	fwdCommand.Flags().StringP("local", "l", "", "[optional] local port to use, ex) 1234")
	fwdCommand.Flags().StringP("target", "t", "", "[optional] it is ec2 instanceId.")

	// mapping viper
	viper.BindPFlag("fwd-remote-port", fwdCommand.Flags().Lookup("remote"))
	viper.BindPFlag("fwd-local-port", fwdCommand.Flags().Lookup("local"))
	viper.BindPFlag("fwd-target", fwdCommand.Flags().Lookup("target"))

	rootCmd.AddCommand(fwdCommand)
}
