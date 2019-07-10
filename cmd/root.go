package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var (
	confFile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ssmsm",
	Short: "The ssmss is a convenient tool supporting a interactive CLI for the AWS Systems Manger Session Manager",
	Long: `The ssmss is useful when you will connect your aws server using start-session, ssh under the AWS Systems Manger Session Manager.

It supports interactive CLI and so you could select your aws server that would like to connect quickly.
`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {

	},
}

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

	rootCmd.PersistentFlags().StringVar(&confFile, "config", "", "conf file (default is $HOME/.ssmsm.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().StringP("profile", "p", "default", "if you is registered multiple profiles for AWS in config file, you could select one of profiles")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if confFile != "" {
		viper.SetConfigFile(confFile)
	} else {
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
	}

	// main viper config
	viper.AutomaticEnv() // read in environment variables that match

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	} else {
		awsconf := fmt.Sprintf("%s/.aws/config", home)
		awscred := fmt.Sprintf("%s/.aws/credentials", home)
		_, confErr := os.Stat(awsconf)
		_, credErr := os.Stat(awscred)
		if !os.IsNotExist(confErr) && !os.IsNotExist(credErr) {
			defaultconf := fmt.Sprintf("%s/.ssmsm.yaml", home)

			_ = defaultconf

			// TODO: config, credentials 이용해서 yaml 파일 생성
		}
	}
}
