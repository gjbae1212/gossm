package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/fatih/color"
	"github.com/gjbae1212/gossm/plugin"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	gossmVersion = "1.3.2"
)

var (
	// rootCmd represents the base command when called without any sub-commands
	rootCmd = &cobra.Command{
		Use:   "gossm",
		Short: `gossm is interactive CLI tool that you select server in AWS and then could connect or send files your AWS server using start-session, ssh, scp in AWS Systems Manger Session Manager.`,
		Long:  `gossm is interactive CLI tool that you select server in AWS and then could connect or send files your AWS server using start-session, ssh, scp in AWS Systems Manger Session Manager.`,
	}

	// default aws regions
	defaultRegions = []string{
		"af-south-1",
		"ap-east-1", "ap-northeast-1", "ap-northeast-2", "ap-northeast-3", "ap-south-1", "ap-southeast-2", "ap-southeast-3",
		"ca-central-1",
		"cn-north-1", "cn-northwest-1",
		"eu-central-1", "eu-north-1", "eu-south-1", "eu-west-1", "eu-west-2", "eu-west-3",
		"me-south-1",
		"sa-east-1",
		"us-east-1", "us-east-2", "us-gov-east-1", "us-gov-west-2", "us-west-1", "us-west-2",
	}

	// extract information for your aws account.
	credential *Credential

	executor *Executor
)

type Credential struct {
	awsKey          string
	awsSecret       string
	awsSessionToken string
	awsProfile      string
	awsRegion       string
	awsSession      *session.Session
}

type Executor struct {
	target      string
	domain      string
	user        string
	execCommand string
	sshKey      string
	localPort   string
	remotePort  string
	multiTarget []string
	multiDomain []string
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
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
	// initialize aws session manager plugin
	initPlugin()
	// initialize executor
	executor = &Executor{}
	executor.target = viper.GetString("target")
}

// initPlugin initializes aws session manager plugin.
func initPlugin() {
	var err error
	home, err := homedir.Dir()
	if err != nil {
		panicRed(err)
	}

	// mapping viper from config file, here's don't use.
	viper.AddConfigPath(home)
	viper.SetConfigType("")
	viper.AutomaticEnv() // read in environment variables that match
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("using config file:", viper.ConfigFileUsed())
	}

	// check session-manager-plugin
	configDir := filepath.Join(home, ".gossm")
	pluginFPath := filepath.Join(configDir, plugin.GetPluginFileName())
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		err := os.MkdirAll(configDir, os.ModePerm)
		if err != nil {
			panicRed(err)
		}
	}

	// create session-manager-plugin
	pluginData, err := plugin.GetPlugin()
	if err != nil {
		panicRed(err)
	}

	viper.Set("plugin", plugin.GetPluginFileName())
	if info, err := os.Stat(pluginFPath); os.IsNotExist(err) {
		if err := ioutil.WriteFile(pluginFPath, pluginData, 0755); err != nil {
			fmt.Println(color.YellowString("[warning] using default session-manager-plugin"))
		} else {
			fmt.Println(color.GreenString("[create] aws ssm plugin"))
			viper.Set("plugin", pluginFPath)
		}
	} else if err != nil {
		fmt.Println(color.YellowString("[warning] using default session-manager-plugin"))
	} else {
		if int(info.Size()) != len(pluginData) {
			if err := ioutil.WriteFile(pluginFPath, pluginData, 0755); err != nil {
				fmt.Println(color.YellowString("[warning] using default session-manager-plugin"))
			} else {
				fmt.Println(color.GreenString("[update] aws ssm plugin"))
				viper.Set("plugin", pluginFPath)
			}
		} else {
			viper.Set("plugin", pluginFPath)
		}
	}
}

// initCredential initializes credential for using inner gossm commands.
func initCredential() {
	credential = &Credential{}

	// get credentials
	credFile, err := rootCmd.Flags().GetString("cred")
	if err != nil {
		panicRed(err)
	}
	// get profile
	profile := viper.GetString("profile")

	// make session
	sess, profile, err := makeSession(credFile, profile)

	cred, err := sess.Config.Credentials.Get()
	if err != nil {
		panicRed(err)
	}

	// insert aws account information
	credential.awsSession = sess
	credential.awsProfile = profile
	credential.awsKey = cred.AccessKeyID
	credential.awsSecret = cred.SecretAccessKey
	credential.awsSessionToken = cred.SessionToken
	credential.awsRegion = viper.GetString("region")
	if credential.awsRegion == "" {
		credential.awsRegion = *sess.Config.Region
	}
}

// initCredentialForMFACommand initializes a credential which is dedicated for mfa command.
func initCredentialForMFACommand() {
	seps := strings.Split(mfaCredentialFile, "/")
	suffix := seps[len(seps)-1]

	// if AWS_SHARED_CREDENTIALS_FILE equals TemporaryCredential.
	if strings.HasSuffix(os.Getenv("AWS_SHARED_CREDENTIALS_FILE"), suffix) {
		// set AWS_SHARED_CREDENTIALS_FILE to default credential.
		os.Setenv("AWS_SHARED_CREDENTIALS_FILE", defaults.SharedCredentialsFilename())
	}
	initCredential()
}

func makeSession(credFile, profile string) (*session.Session, string, error) {
	if profile == "" {
		profile = "default"
		if os.Getenv("AWS_PROFILE") != "" {
			profile = os.Getenv("AWS_PROFILE")
		}
	}

	// if cred args is exist.
	if credFile != "" {
		sess, err := session.NewSession(&aws.Config{
			Credentials: credentials.NewSharedCredentials(credFile, profile)})
		return sess, profile, err
	}

	// default creds
	envPresent := os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != ""
	if envPresent {
		sess, err := session.NewSessionWithOptions(session.Options{
			AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
		})
		return sess, "env", err
	}
	sess, err := session.NewSessionWithOptions(session.Options{
		Profile:                 profile,
		SharedConfigState:       session.SharedConfigEnable,
		AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
	})
	return sess, profile, err
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringP("cred", "c", "", "aws credentials file (default is $HOME/.aws/.credentials)")
	rootCmd.PersistentFlags().StringP("profile", "p", "", `
[optional] if you are having multiple aws profiles, it is one of profiles (default is AWS_PROFILE environment variable or default)`)
	rootCmd.PersistentFlags().StringP("region", "r", "", `[optional] it is region in AWS that would like to do something`)
	rootCmd.PersistentFlags().StringP("target", "t", "", "[optional] it is instanceId of server in AWS that would like to something")

	// set version flag
	rootCmd.Version = gossmVersion
	rootCmd.InitDefaultVersionFlag()

	// mapping viper
	viper.BindPFlag("target", rootCmd.PersistentFlags().Lookup("target"))
	viper.BindPFlag("profile", rootCmd.PersistentFlags().Lookup("profile"))
	viper.BindPFlag("region", rootCmd.PersistentFlags().Lookup("region"))
}
