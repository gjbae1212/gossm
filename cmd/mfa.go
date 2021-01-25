package cmd

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	virtualMFADevice    = "arn:aws:iam::%s:mfa/%s"
	mfaCredentialFormat = "[default]\naws_access_key_id = %s\naws_secret_access_key = %s\naws_session_token = %s\n"
)

var (
	// temporary credential file
	mfaCredentialFile = fmt.Sprintf("%s_mfa", defaults.SharedCredentialsFilename())
)

var (
	credForMFA = fmt.Sprintf("%s_mfa", defaults.SharedCredentialsFilename())
)

var (
	mfaCommand = &cobra.Command{
		Use:   "mfa",
		Short: "It's to authenticate MFA on AWS, and save authenticated mfa token in .aws/credentials_mfa.",
		Long: `
This command is to authenticate MFA on AWS, and save authenticated mfa token in .aws/credentials_mfa.
Insert to "AWS_SHARED_CREDENTIALS_FILE"" environment variables as "$HOME/.aws/credentials_mfa". 
So you can conveniently use aws-cli or gossm.
`,
		PreRun: func(cmd *cobra.Command, args []string) {
			initCredentialForMFACommand()
		},
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				panicRed(fmt.Errorf("invalid mfa code"))
			}
			code := strings.TrimSpace(args[0])
			if code == "" {
				panicRed(fmt.Errorf("invalid mfa code"))
			}

			deadline := viper.GetInt64("mfa-deadline")
			device := viper.GetString("mfa-device")
			// if device params is empty, getting virtual mfa device.
			if device == "" {
				identity, err := sts.New(credential.awsSession).GetCallerIdentity(&sts.GetCallerIdentityInput{})
				if err != nil {
					panicRed(err)
				}
				username := strings.Split(*identity.Arn, "/")[1]
				device = fmt.Sprintf(virtualMFADevice, aws.StringValue(identity.Account), username)
			}

			output, err := sts.New(credential.awsSession).GetSessionToken(&sts.GetSessionTokenInput{
				DurationSeconds: aws.Int64(deadline),
				SerialNumber:    aws.String(device),
				TokenCode:       aws.String(code),
			})
			if err != nil {
				panicRed(err)
			}

			newCredential := fmt.Sprintf(mfaCredentialFormat, *output.Credentials.AccessKeyId,
				*output.Credentials.SecretAccessKey, *output.Credentials.SessionToken)
			if err := ioutil.WriteFile(mfaCredentialFile, []byte(newCredential), 0600); err != nil {
				panicRed(err)
			}

			color.Green("[SUCCESS] Temporary MFA credential creates %s (%s)", mfaCredentialFile, output.Credentials.Expiration.UTC())
			color.Yellow("[INFO] For Use AWS CLI using temporary MFA credential")
			fmt.Printf("%s `%s` %s\n",
				color.RedString("Must set to"),
				color.CyanString("export AWS_SHARED_CREDENTIALS_FILE=%s", mfaCredentialFile),
				color.WhiteString("in .bash_profile, .zshrc."),
			)
		},
	}
)

func init() {
	mfaCommand.Flags().Int64P("deadline", "", 21600, "[optional] deadline seconds for issued credentials. (default is 6 hours)")
	mfaCommand.Flags().StringP("device", "", "", "[optional] mfa device. (default is your virtual mfa device)")

	viper.BindPFlag("mfa-deadline", mfaCommand.Flags().Lookup("deadline"))
	viper.BindPFlag("mfa-device", mfaCommand.Flags().Lookup("device"))
	rootCmd.AddCommand(mfaCommand)
}
