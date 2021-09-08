package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/fatih/color"
	"github.com/gjbae1212/gossm/internal"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	_defaultProfile = "default"
)

var (
	// rootCmd represents the base command when called without any sub-commands
	rootCmd = &cobra.Command{
		Use:   "gossm",
		Short: `gossm is interactive CLI tool that you select server in AWS and then could connect or send files your AWS server using start-session, ssh, scp in AWS Systems Manger Session Manager.`,
		Long:  `gossm is interactive CLI tool that you select server in AWS and then could connect or send files your AWS server using start-session, ssh, scp in AWS Systems Manger Session Manager.`,
	}

	_version                 string
	_credential              *Credential
	_credentialWithMFA       = fmt.Sprintf("%s_mfa", config.DefaultSharedCredentialsFilename())
	_credentialWithTemporary = fmt.Sprintf("%s_temporary", config.DefaultSharedCredentialsFilename())
)

type Credential struct {
	awsProfile    string
	awsConfig     *aws.Config
	gossmHomePath string
	ssmPluginPath string
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(version string) {
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		panicRed(err)
	}
}

// panicRed raises error with text.
func panicRed(err error) {
	fmt.Println(color.RedString("[err] %s", err.Error()))
	os.Exit(1)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	_credential = &Credential{}
	// 1. get aws profile
	awsProfile := viper.GetString("profile")
	if awsProfile == "" {
		if os.Getenv("AWS_PROFILE") != "" {
			awsProfile = os.Getenv("AWS_PROFILE")
		} else {
			awsProfile = _defaultProfile
		}
	}
	_credential.awsProfile = awsProfile

	// 2. get region
	awsRegion := viper.GetString("region")

	// 3. update or create aws ssm plugin.
	home, err := homedir.Dir()
	if err != nil {
		panicRed(internal.WrapError(err))
	}

	_credential.gossmHomePath = filepath.Join(home, ".gossm")
	if _, err := os.Stat(_credential.gossmHomePath); os.IsNotExist(err) {
		if err := os.MkdirAll(_credential.gossmHomePath, os.ModePerm); err != nil {
			panicRed(internal.WrapError(err))
		}
	}

	plugin, err := internal.GetSsmPlugin()
	if err != nil {
		panicRed(internal.WrapError(err))
	}

	_credential.ssmPluginPath = filepath.Join(_credential.gossmHomePath, internal.GetSsmPluginName())
	if info, err := os.Stat(_credential.ssmPluginPath); os.IsNotExist(err) {
		color.Green("[create] aws ssm plugin")
		if err := ioutil.WriteFile(_credential.ssmPluginPath, plugin, 0755); err != nil {
			panicRed(internal.WrapError(err))
		}
	} else if err != nil {
		panicRed(internal.WrapError(err))
	} else {
		if int(info.Size()) != len(plugin) {
			color.Green("[update] aws ssm plugin")
			if err := ioutil.WriteFile(_credential.ssmPluginPath, plugin, 0755); err != nil {
				panicRed(internal.WrapError(err))
			}
		}
	}

	// 4. set shared credential.
	sharedCredFile := os.Getenv("AWS_SHARED_CREDENTIALS_FILE")
	if sharedCredFile == "" {
		// if gossm mfa credential is existed?
		if _, err := os.Stat(_credentialWithMFA); !os.IsNotExist(err) {
			color.Yellow("[Use] gossm default mfa credential file %s", _credentialWithMFA)
			os.Setenv("AWS_SHARED_CREDENTIALS_FILE", _credentialWithMFA)
			sharedCredFile = _credentialWithMFA
		}
	} else {
		sharedCredFile, err = filepath.Abs(sharedCredFile)
		if err != nil {
			color.Yellow("[Warning] invalid AWS_SHARED_CREDENTIALS_FILE environments path, such as %w", err)
			os.Unsetenv("AWS_SHARED_CREDENTIALS_FILE")
			sharedCredFile = ""
		} else {
			if _, err := os.Stat(sharedCredFile); os.IsNotExist(err) {
				color.Yellow("[Warning] not found AWS_SHARED_CREDENTIALS_FILE environments file, such as %s", sharedCredFile)
				os.Unsetenv("AWS_SHARED_CREDENTIALS_FILE")
				sharedCredFile = ""
			}
		}
	}

	// if shared cred file is exist.
	if sharedCredFile != "" {
		awsConfig, err := internal.NewSharedConfig(context.Background(),
			_credential.awsProfile,
			[]string{config.DefaultSharedConfigFilename()},
			[]string{sharedCredFile},
		)
		if err != nil {
			panicRed(internal.WrapError(err))
		}

		cred, err := awsConfig.Credentials.Retrieve(context.Background())
		// delete invalid shared credential.
		if err != nil || cred.Expired() || cred.AccessKeyID == "" || cred.SecretAccessKey == "" {
			color.Yellow("[Expire] gossm default mfa credential file %s", sharedCredFile)
			os.Unsetenv("AWS_SHARED_CREDENTIALS_FILE")
		} else {
			_credential.awsConfig = &awsConfig
		}
	}

	// check subcommands
	args := os.Args[1:]
	subcmd, _, err := rootCmd.Find(args)
	if err != nil {
		panicRed(internal.WrapError(err))
	}

	switch subcmd.Use {
	case "mfa": // mfa command doesn't use session token.
		if _credential.awsConfig != nil {
			cred, err := _credential.awsConfig.Credentials.Retrieve(context.Background())
			if err != nil {
				panicRed(internal.WrapError(err))
			}

			if cred.SessionToken != "" { // delete shared credentials
				os.Unsetenv("AWS_SHARED_CREDENTIALS_FILE")
				_credential.awsConfig = nil
			}
		}
	}

	if _credential.awsConfig == nil { // not use shared credential
		var temporaryCredentials aws.Credentials
		var temporaryConfig aws.Config

		if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" { // use global environments.
			temporaryConfig, err = internal.NewConfig(context.Background(),
				os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"),
				os.Getenv("AWS_SESSION_TOKEN"), awsRegion, os.Getenv("AWS_ROLE_ARN"))
			if err != nil {
				panicRed(internal.WrapError(err))
			}

			temporaryCredentials, err = temporaryConfig.Credentials.Retrieve(context.Background())
			if err != nil || temporaryCredentials.Expired() ||
				temporaryCredentials.AccessKeyID == "" || temporaryCredentials.SecretAccessKey == "" ||
				(subcmd.Use == "mfa" && temporaryCredentials.SessionToken != "") {
				panicRed(internal.WrapError(fmt.Errorf("[err] invalid global environments %s", err.Error())))
			}
		} else { // use default credential file
			// get cred by only config
			temporaryConfig, err = internal.NewSharedConfig(context.Background(), _credential.awsProfile,
				[]string{config.DefaultSharedConfigFilename()}, []string{})
			if err == nil {
				temporaryCredentials, err = temporaryConfig.Credentials.Retrieve(context.Background())
			}

			// error is raised or temporaryCredentials is invalid.
			if err != nil || temporaryCredentials.Expired() ||
				temporaryCredentials.AccessKeyID == "" || temporaryCredentials.SecretAccessKey == "" ||
				(subcmd.Use == "mfa" && temporaryCredentials.SessionToken != "") {
				// get cred by default credential file.
				temporaryConfig, err = internal.NewSharedConfig(context.Background(), _credential.awsProfile,
					[]string{config.DefaultSharedConfigFilename()}, []string{config.DefaultSharedCredentialsFilename()})
				if err != nil {
					panicRed(internal.WrapError(err))
				}

				temporaryCredentials, err = temporaryConfig.Credentials.Retrieve(context.Background())
				if err != nil {
					panicRed(internal.WrapError(err))
				}
				if temporaryCredentials.Expired() || temporaryCredentials.AccessKeyID == "" || temporaryCredentials.SecretAccessKey == "" {
					panicRed(internal.WrapError(fmt.Errorf("[err] not found credentials")))
				}

				// extract aws region if awsRegion is empty.
				if awsRegion == "" {
					awsRegion = temporaryConfig.Region
				}
			}
		}

		// [ISSUE] KMS Encrypt, must use AWS_SHARED_CREDENTIALS_FILE with SharedConfig.
		// [INFO] write temporaryCredentials to file.

		temporaryCredentialsString := fmt.Sprintf(mfaCredentialFormat, _credential.awsProfile, temporaryCredentials.AccessKeyID,
			temporaryCredentials.SecretAccessKey, temporaryCredentials.SessionToken)
		if err := ioutil.WriteFile(_credentialWithTemporary, []byte(temporaryCredentialsString), 0600); err != nil {
			panicRed(internal.WrapError(err))
		}

		os.Setenv("AWS_SHARED_CREDENTIALS_FILE", _credentialWithTemporary)
		awsConfig, err := internal.NewSharedConfig(context.Background(),
			_credential.awsProfile, []string{}, []string{_credentialWithTemporary},
		)
		if err != nil {
			panicRed(internal.WrapError(err))
		}
		_credential.awsConfig = &awsConfig
	}

	// set region
	if awsRegion != "" {
		_credential.awsConfig.Region = awsRegion
	}
	if _credential.awsConfig.Region == "" { // ask region
		askRegion, err := internal.AskRegion(context.Background(), *_credential.awsConfig)
		if err != nil {
			panicRed(internal.WrapError(err))
		}
		_credential.awsConfig.Region = askRegion.Name
	}
	color.Green("region (%s)", _credential.awsConfig.Region)
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringP("profile", "p", "", `[optional] if you are having multiple aws profiles, it is one of profiles (default is AWS_PROFILE environment variable or default)`)
	rootCmd.PersistentFlags().StringP("region", "r", "", `[optional] it is region in AWS that would like to do something`)

	// set version flag
	rootCmd.InitDefaultVersionFlag()

	// mapping viper
	viper.BindPFlag("profile", rootCmd.PersistentFlags().Lookup("profile"))
	viper.BindPFlag("region", rootCmd.PersistentFlags().Lookup("region"))
}
