package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/spf13/viper"

	. "github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
)

var (
	// runCommand is a AWS Systems Manager Run Command.
	runCommand = &cobra.Command{
		Use:   "cmd",
		Short: "Exec `run command` under AWS SSM with interactive CLI",
		Long:  "Exec `run command` under AWS SSM with interactive CLI",
		PreRun: func(cmd *cobra.Command, args []string) {
			// check exec parameter
			if viper.GetString("run-exec") == "" {
				fmt.Println(Red("[err] [required] exec argument"))
				os.Exit(1)
			}
			// set region
			if err := setRegion(); err != nil {
				fmt.Println(Red(err))
				os.Exit(1)
			}

			// set multi target
			if err := setMultiTarget(); err != nil {
				fmt.Println(Red(err))
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			region := viper.GetString("region")
			targets := viper.GetStringSlice("targets")
			exec := viper.GetString("run-exec")

			// send command
			cmdOutput, err := sendCommand(region, targets, exec)
			if err != nil {
				fmt.Println(Red(err))
				os.Exit(1)
			}

			commandId := *cmdOutput.Command.CommandId
			instanceIds := aws.StringValueSlice(cmdOutput.Command.InstanceIds)
			fmt.Printf("command: %s, commandId: %s, targets: %v\n", Green(exec), Green(commandId), Green(instanceIds))
			fmt.Printf("%s\n", Yellow("Waiting Response ..."))

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
			printCommandInvocation(region, inputs)
		},
	}
)

func init() {
	// add sub command
	runCommand.Flags().StringP("exec", "e", "", "[required] run command")

	// mapping viper
	viper.BindPFlag("run-exec", runCommand.Flags().Lookup("exec"))

	rootCmd.AddCommand(runCommand)
}
