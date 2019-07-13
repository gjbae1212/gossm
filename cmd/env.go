package cmd

import (
	"fmt"

	"github.com/spf13/viper"
)

func setEnvRegion() error {
	// if region don't exist, get region from prompt
	var region = viper.GetString("region")
	if region == "" {
		region, err := askRegion()
		if err != nil {
			return err
		}
		viper.Set("region", region)
	}
	return nil
}

func setInstance() error {
	region := viper.GetString("region")
	if region == "" {
		return fmt.Errorf("[err] unknown region")
	}

	instance := viper.GetString("instance")
	if instance == "" {
		instance, err := askInstance(region)
		if err != nil {
			return err
		}
		viper.Set("instance", instance)
	}
	return nil
}
