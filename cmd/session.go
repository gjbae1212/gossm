package cmd

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/fatih/color"
	"github.com/gjbae1212/gossm/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	startSessionCommand = &cobra.Command{
		Use:   "start",
		Short: "Exec `start-session` under AWS SSM with interactive CLI",
		Long:  "Exec `start-session` under AWS SSM with interactive CLI",
		Run: func(cmd *cobra.Command, args []string) {
			var (
				target *internal.Target
				err    error
			)
			ctx := context.Background()

			// get target
			argTarget := strings.TrimSpace(viper.GetString("start-session-target"))
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
			internal.PrintReady("start-session", _credential.awsConfig.Region, target.Name)

			input := &ssm.StartSessionInput{Target: aws.String(target.Name)}
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
	startSessionCommand.Flags().StringP("target", "t", "", "[optional] it is ec2 instanceId.")
	viper.BindPFlag("start-session-target", startSessionCommand.Flags().Lookup("target"))

	// add sub command
	rootCmd.AddCommand(startSessionCommand)
}
