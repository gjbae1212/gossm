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
	scpCommand = &cobra.Command{
		Use:   "scp",
		Short: "Exec `scp` under AWS SSM with interactive CLI",
		Long:  "Exec `scp` under AWS SSM with interactive CLI",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			scpCommand := strings.TrimSpace(viper.GetString("scp-exec"))

			if scpCommand == "" {
				panicRed(fmt.Errorf("[err] required exec argument"))
			}

			seps := strings.Split(scpCommand, " ")
			if len(seps) < 2 {
				panicRed(fmt.Errorf("[err] invalid exec argument"))
			}

			dst := seps[len(seps)-1]
			dstSeps := strings.Split(strings.Split(dst, ":")[0], "@")
			seps = strings.Split(strings.TrimSpace(strings.Join(seps[0:(len(seps)-1)], " ")), " ")

			src := seps[len(seps)-1]
			srcSeps := strings.Split(strings.Split(src, ":")[0], "@")

			var ips []net.IP
			var err error
			switch {
			case len(srcSeps) == 2:
				ips, err = net.LookupIP(srcSeps[1])
			case len(dstSeps) == 2:
				ips, err = net.LookupIP(dstSeps[1])
			default:
				panicRed(fmt.Errorf("[err] invalid scp args"))
			}
			if err != nil {
				panicRed(fmt.Errorf("[err] invalid server domain name"))
			}

			ip := ips[0].String()
			instId, err := internal.FindInstanceIdByIp(ctx, *_credential.awsConfig, ip)
			if err != nil {
				panicRed(err)
			}
			if instId == "" {
				panicRed(fmt.Errorf("[err] not found matched server"))
			}
			targetName := instId

			internal.PrintReady("scp", _credential.awsConfig.Region, targetName)
			color.Cyan("scp " + scpCommand)

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

			// call scp
			proxy := fmt.Sprintf("ProxyCommand=%s '%s' %s %s %s '%s'",
				_credential.ssmPluginPath, string(sessJson), _credential.awsConfig.Region,
				"StartSession", _credential.awsProfile, string(paramsJson))
			sshArgs := []string{"-o", proxy}
			for _, sep := range strings.Split(scpCommand, " ") {
				if sep != "" {
					sshArgs = append(sshArgs, sep)
				}
			}

			if err := internal.CallProcess("scp", sshArgs...); err != nil {
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
	scpCommand.Flags().StringP("exec", "e", "", "[required] scp $exec, ex) \"-i ex.pem ubuntu@server:/home/ex.txt ex.txt\"")
	viper.BindPFlag("scp-exec", scpCommand.Flags().Lookup("exec"))
	rootCmd.AddCommand(scpCommand)
}
