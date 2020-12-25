package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/gjbae1212/gossm/plugin"

	"github.com/aws/aws-sdk-go/aws/session"

	. "github.com/logrusorgru/aurora"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	gossmVersion = "1.1.0"
)

var (
	// default aws session
	awsSession *session.Session

	// rootCmd represents the base command when called without any subcommands
	rootCmd = &cobra.Command{
		Use:   "gossm",
		Short: `gossm is interactive CLI tool that you select server in AWS and then could connect or send files your AWS server using start-session, ssh, scp in AWS Systems Manger Session Manager.`,
		Long:  `gossm is interactive CLI tool that you select server in AWS and then could connect or send files your AWS server using start-session, ssh, scp in AWS Systems Manger Session Manager.`,
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
		fmt.Println(Red(err))
		os.Exit(1)
	}
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

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	var err error
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(Red(err))
		os.Exit(1)
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
			fmt.Println(Red(err))
			os.Exit(1)
		}
	}

	// create session-manager-plugin
	pluginData, err := plugin.GetPlugin()
	if err != nil {
		fmt.Println(Red(err))
		os.Exit(1)
	}

	viper.Set("plugin", plugin.GetPluginFileName())
	if info, err := os.Stat(pluginFPath); os.IsNotExist(err) {
		if err := ioutil.WriteFile(pluginFPath, pluginData, 0755); err != nil {
			fmt.Println(Yellow("[warning] using default session-manager-plugin"))
		} else {
			fmt.Println(Green("[create] aws ssm plugin"))
			viper.Set("plugin", pluginFPath)
		}
	} else if err != nil {
		fmt.Println(Yellow("[warning] using default session-manager-plugin"))
	} else {
		if int(info.Size()) != len(pluginData) {
			if err := ioutil.WriteFile(pluginFPath, pluginData, 0755); err != nil {
				fmt.Println(Yellow("[warning] using default session-manager-plugin"))
			} else {
				fmt.Println(Green("[update] aws ssm plugin"))
				viper.Set("plugin", pluginFPath)
			}
		} else {
			viper.Set("plugin", pluginFPath)
		}
	}

	// set profile
	profile := viper.GetString("profile")
	if profile != "" {
		os.Setenv("AWS_PROFILE", profile)
	}
	
	// set region
	region := viper.GetString("region")
	if region != "" {
		os.Setenv("AWS_REGION", region)
	}

	awsSession = session.Must(session.NewSession())
}
