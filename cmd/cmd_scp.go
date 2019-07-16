package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/service/ssm"
	. "github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	scpCommand = &cobra.Command{
		Use:   "scp",
		Short: "Exec `scp` under AWS SSM with interactive CLI",
		Long:  "Exec `scp` under AWS SSM with interactive CLI",
		PreRun: func(cmd *cobra.Command, args []string) {
			// set region
			if err := setRegion(); err != nil {
				fmt.Println(Red(err))
				os.Exit(1)
			}

			// set ssh
			if err := setSCP(); err != nil {
				fmt.Println(Red(err))
				os.Exit(1)
			}

			printReady("scp")
			fmt.Printf("%s\n", Green("scp "+viper.GetString("scp-exec")))
		},
		Run: func(cmd *cobra.Command, args []string) {
			exec := viper.GetString("scp-exec")
			region := viper.GetString("region")
			profile := viper.GetString("profile")
			target := viper.GetString("target")
			docName := "AWS-StartSSHSession"
			port := "22"
			input := &ssm.StartSessionInput{
				DocumentName: &docName,
				Parameters:   map[string][]*string{"portNumber": []*string{&port}},
				Target:       &target,
			}

			// create session
			sess, endpoint, err := createStartSession(region, input)
			if err != nil {
				fmt.Println(Red(err))
				os.Exit(1)
			}

			sessJson, err := json.Marshal(sess)
			if err != nil {
				fmt.Println(Red(err))
				os.Exit(1)
			}

			paramsJson, err := json.Marshal(input)
			if err != nil {
				fmt.Println(Red(err))
				os.Exit(1)
			}

			// call ssh
			proxy := fmt.Sprintf("ProxyCommand=%s '%s' %s %s %s '%s' %s",
				"session-manager-plugin", string(sessJson), region, "StartSession", profile, string(paramsJson), endpoint)
			scpArgs := []string{"-o", proxy}
			for _, sep := range strings.Split(exec, " ") {
				if sep != "" {
					scpArgs = append(scpArgs, sep)
				}
			}
			if err := callSubprocess("scp", scpArgs...); err != nil {
				fmt.Println(Red(err))
				// delete Session
				if err := deleteStartSession(region, *sess.SessionId); err != nil {
					fmt.Println(Red(err))
				}
				os.Exit(1)
			}
		},
	}
)

func init() {
	// add sub command
	scpCommand.Flags().StringP("exec", "e", "", "[required] scp $exec, ex) \"-i ex.pem ubuntu@server:/home/ex.txt ex.txt\"")

	// mapping viper
	viper.BindPFlag("scp-exec", scpCommand.Flags().Lookup("exec"))

	rootCmd.AddCommand(scpCommand)
}
