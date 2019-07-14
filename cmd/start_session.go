package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"

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
				fmt.Println(err)
				os.Exit(1)
			}
			// set instance
			if err := setInstance(); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			svc := ssm.New(awsSession, aws.NewConfig().WithRegion(viper.GetString("region")))
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
			defer cancel()
			inst := viper.GetString("instance")
			sess, err := svc.StartSessionWithContext(ctx, &ssm.StartSessionInput{Target: &inst})
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Println(*sess.SessionId)
			fmt.Println(*sess.StreamUrl)
			fmt.Println(*sess.TokenValue)
			a, err := json.Marshal(sess)
			fmt.Println(string(a))
			b, err := json.Marshal(&ssm.StartSessionInput{Target: &inst})
			fmt.Println(string(b))
			fmt.Println(svc.Endpoint)
			// TODO: session-manager-plugin 설치
			// TODO: session-manager-plugin 'json' region StartSession
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
