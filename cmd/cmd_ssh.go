package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/service/ssm"

	"github.com/spf13/viper"

	. "github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
)

var (
	sshCommand = &cobra.Command{
		Use:   "ssh",
		Short: "Exec `ssh` under AWS SSM with interactive CLI",
		Long:  "Exec `ssh` under AWS SSM with interactive CLI",
		PreRun: func(cmd *cobra.Command, args []string) {
			// set region
			if err := setRegion(); err != nil {
				fmt.Println(Red(err))
				os.Exit(1)
			}

			// set ssh
			if err := setSSH(); err != nil {
				fmt.Println(Red(err))
				os.Exit(1)
			}

			printReady("ssh")
			fmt.Printf("%s\n", Green("ssh "+viper.GetString("ssh-exec")))
		},
		Run: func(cmd *cobra.Command, args []string) {
			exec := viper.GetString("ssh-exec")
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
			plug := viper.Get("plugin")
			proxy := fmt.Sprintf("ProxyCommand=%s '%s' %s %s %s '%s' %s",
				plug, string(sessJson), region, "StartSession", profile, string(paramsJson), endpoint)
			sshArgs := []string{"-o", proxy}
			for _, sep := range strings.Split(exec, " ") {
				if sep != "" {
					sshArgs = append(sshArgs, sep)
				}
			}
			if err := callSubprocess("ssh", sshArgs...); err != nil {
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
	sshCommand.Flags().StringP("exec", "e", "", "[optional] ssh $exec, ex) \"-i ex.pem ubuntu@server\"")

	sshCommand.Flags().StringP("identity", "i", "", "[optional] identity file path, ex) $HOME/.ssh/id_rsa")

	// mapping viper
	viper.BindPFlag("ssh-exec", sshCommand.Flags().Lookup("exec"))

	viper.BindPFlag("ssh-identity", sshCommand.Flags().Lookup("identity"))

	rootCmd.AddCommand(sshCommand)
}
