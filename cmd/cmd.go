package cmd

import (
	"fmt"
	"time"

	"github.com/fatih/color"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// cmdCommand is a AWS Systems Manager Run Command.
	cmdCommand = &cobra.Command{
		Use:   "cmd",
		Short: "Exec `run command` under AWS SSM with interactive CLI",
		Long:  "Exec `run command` under AWS SSM with interactive CLI",
		PreRun: func(cmd *cobra.Command, args []string) {
			initCredential()
			executor.execCommand = viper.GetString("cmd-exec")

			// set region
			if err := setRegion(credential); err != nil {
				panicRed(err)
			}

			// set multi target
			if err := setMultiTarget(credential, executor); err != nil {
				panicRed(err)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			// send command
			cmdOutput, err := sendCommand(credential.awsSession, credential.awsRegion, executor.multiTarget, executor.execCommand)
			if err != nil {
				panicRed(err)
			}

			commandId := *cmdOutput.Command.CommandId
			instanceIds := aws.StringValueSlice(cmdOutput.Command.InstanceIds)
			fmt.Printf("command: %s, commandId: %s, targets: %s\n",
				color.GreenString(executor.execCommand), color.GreenString(commandId),
				color.GreenString(fmt.Sprintf("%v", instanceIds)))
			fmt.Printf("%s\n", color.YellowString("Waiting Response ..."))

			// wait 3 seconds
			time.Sleep(time.Second * 3)

			// show result
			var inputs []*ssm.GetCommandInvocationInput
			for _, inst := range cmdOutput.Command.InstanceIds {
				inputs = append(inputs, &ssm.GetCommandInvocationInput{
					CommandId:  cmdOutput.Command.CommandId,
					InstanceId: inst,
				})
			}
			printCommandInvocation(credential.awsSession, credential.awsRegion, inputs)
		},
	}
)

func init() {
	cmdCommand.Flags().StringP("exec", "e", "", "[required] execute command")
	viper.BindPFlag("cmd-exec", cmdCommand.Flags().Lookup("exec"))
	rootCmd.AddCommand(cmdCommand)
}
