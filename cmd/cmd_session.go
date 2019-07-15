package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	. "github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	startSessionCommand = &cobra.Command{
		Use:   "start-session",
		Short: "Start `start-session` under AWS SSM with interactive CLI",
		Long:  "Start `start-session` under AWS SSM with interactive CLI",
		PreRun: func(cmd *cobra.Command, args []string) {
			// set region
			if err := setEnvRegion(); err != nil {
				fmt.Println(Red(err))
				os.Exit(1)
			}
			// set instance
			if err := setInstance(); err != nil {
				fmt.Println(Red(err))
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			svc := ssm.New(awsSession, aws.NewConfig().WithRegion(viper.GetString("region")))
			subctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
			defer cancel()

			profile := viper.GetString("profile")
			inst := viper.GetString("instance")
			params := &ssm.StartSessionInput{Target: &inst}
			sess, err := svc.StartSessionWithContext(subctx, params)
			if err != nil {
				fmt.Println(Red(err))
				os.Exit(1)
			}

			sessJson, err := json.Marshal(sess)
			if err != nil {
				fmt.Println(Red(err))
				os.Exit(1)
			}

			paramsJson, err := json.Marshal(params)
			if err != nil {
				fmt.Println(Red(err))
				os.Exit(1)
			}

			if err := callSubprocess("session-manager-plugin", string(sessJson),
				viper.GetString("region"), "StartSession", profile, string(paramsJson), svc.Endpoint); err != nil {
				fmt.Println(Red(err))
				// Delete Session
				fmt.Printf("%s %s \n", Yellow("Delete Session"), Yellow(*sess.SessionId))
				subctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
				defer cancel()
				if _, err = svc.TerminateSessionWithContext(
					subctx, &ssm.TerminateSessionInput{SessionId: sess.SessionId}); err != nil {
					fmt.Println(Red(err))
				}
				os.Exit(1)
			}
		},
	}
)

func init() {
	// start-session additional flag
	startSessionCommand.Flags().StringP("instance", "i", "", "[optional] it is instance-id of server in AWS that  would like to something")

	// mapping viper
	viper.BindPFlag("instance", startSessionCommand.Flags().Lookup("instance"))

	// add sub command
	rootCmd.AddCommand(startSessionCommand)
}
