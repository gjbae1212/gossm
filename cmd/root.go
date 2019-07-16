package cmd

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"

	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/spf13/cobra"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var (
	// default aws session
	awsSession *session.Session

	// rootCmd represents the base command when called without any subcommands
	rootCmd = &cobra.Command{
		Use:   "gossm",
		Short: "gossm is a convenient tool supporting a interactive CLI about the AWS Systems Manger Session Manager",
		Long: `gossm is useful when you will connect or send your AWS server using start-session, ssh, scp under the AWS Systems Manger. 

gossm supports interactive CLI and so you could select your AWS server that would like to connect quickly.
`,
	}

	// default aws regions
	defaultRegions = []string{
		"us-east-1", "us-east-2", "us-west-1", "us-west-2",
		"eu-north-1", "eu-west-3", "eu-west-2", "eu-west-1", "eu-central-1",
		"ap-south-1", "ap-northeast-2", "ap-northeast-1", "ap-southeast-1", "ap-southeast-2",
		"sa-east-1",
		"ca-central-1",
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringP("cred", "c", "", "aws credentials file (default is $HOME/.aws/.credentials)")
	rootCmd.PersistentFlags().StringP("profile", "p", "default", "[optional] if you are having multiple aws profiles in config, it is one of profiles")
	rootCmd.PersistentFlags().StringP("region", "r", "", "[optional] it is region in AWS that would like to do something")
	rootCmd.PersistentFlags().StringP("target", "t", "", "[optional] it is instanceId of server in AWS that would like to something")

	// mapping viper
	viper.BindPFlag("target", rootCmd.PersistentFlags().Lookup("target"))
	viper.BindPFlag("profile", rootCmd.PersistentFlags().Lookup("profile"))
	viper.BindPFlag("region", rootCmd.PersistentFlags().Lookup("region"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	var err error
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// mapping viper from config file, here's don't use.
	viper.AddConfigPath(home)
	viper.SetConfigType("")
	viper.AutomaticEnv() // read in environment variables that match
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

	// get credentials
	credFile, err := rootCmd.Flags().GetString("cred")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// get profile
	profile, err := rootCmd.Flags().GetString("profile")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// get session
	if credFile != "" {
		awsSession, err = session.NewSession(&aws.Config{
			Credentials: credentials.NewSharedCredentials(credFile, profile),
		})
	} else {
		awsSession, err = session.NewSessionWithOptions(session.Options{
			Profile:           profile,
			SharedConfigState: session.SharedConfigEnable,
		})
	}
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// mapping viper
	cred, err := awsSession.Config.Credentials.Get()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	viper.Set("accesskey", cred.AccessKeyID)
	viper.Set("secretkey", cred.SecretAccessKey)
	if viper.GetString("region") == "" {
		viper.Set("region", *awsSession.Config.Region)
	}
	awsSession.Config.WithRegion(viper.GetString("region"))
}
