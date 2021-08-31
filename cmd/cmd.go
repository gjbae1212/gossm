package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/fatih/color"
	"github.com/gjbae1212/gossm/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// cmdCommand is AWS Systems Manager Run Command.
	cmdCommand = &cobra.Command{
		Use:   "cmd",
		Short: "Exec `run command` under AWS SSM with interactive CLI",
		Long:  "Exec `run command` under AWS SSM with interactive CLI",
		Run: func(cmd *cobra.Command, args []string) {
			var (
				targets []*internal.Target
				err     error
			)
			ctx := context.Background()

			exec := strings.TrimSpace(viper.GetString("cmd-exec"))
			if exec == "" {
				panicRed(fmt.Errorf("[err] not found exec command"))
			}

			// get targets
			argTarget := strings.TrimSpace(viper.GetString("cmd-target"))
			if argTarget != "" {
				table, err := internal.FindInstances(ctx, *_credential.awsConfig)
				if err != nil {
					panicRed(err)
				}
				for _, t := range table {
					if t.Name == argTarget {
						targets = append(targets, t)
						break
					}
				}
			}

			if len(targets) == 0 {
				targets, err = internal.AskMultiTarget(ctx, *_credential.awsConfig)
				if err != nil {
					panicRed(err)
				}
			}

			var targetName string
			for _, t := range targets {
				targetName += " " + t.Name + " "
			}

			internal.PrintReady(exec, _credential.awsConfig.Region, targetName)

			sendOutput, err := internal.SendCommand(ctx, *_credential.awsConfig, targets, exec)
			if err != nil {
				panicRed(err)
			}

			fmt.Printf("%s\n", color.YellowString("Waiting Response ..."))
			// wait 3 seconds
			time.Sleep(time.Second * 3)

			// show result
			var inputs []*ssm.GetCommandInvocationInput
			for _, inst := range sendOutput.Command.InstanceIds {
				inputs = append(inputs, &ssm.GetCommandInvocationInput{
					CommandId:  sendOutput.Command.CommandId,
					InstanceId: aws.String(inst),
				})
			}
			internal.PrintCommandInvocation(ctx, *_credential.awsConfig, inputs)
		},
	}
)

func init() {
	cmdCommand.Flags().StringP("exec", "e", "", "[required] execute command")
	cmdCommand.Flags().StringP("target", "t", "", "[optional] it is ec2 instanceId.")

	viper.BindPFlag("cmd-exec", cmdCommand.Flags().Lookup("exec"))
	viper.BindPFlag("cmd-target", cmdCommand.Flags().Lookup("target"))

	rootCmd.AddCommand(cmdCommand)
}
