package cmd

import (
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"syscall"

	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/aws/aws-sdk-go/aws"
	. "github.com/logrusorgru/aurora"
	"github.com/spf13/viper"
)

// Get params from interactive CLI and then its params set to viper
func setEnvRegion() error {
	// if region don't exist, get region from prompt
	var err error
	var region = viper.GetString("region")
	if region == "" {
		region, err = askRegion()
		if err != nil {
			return err
		}
		viper.Set("region", region)
	}

	if region == "" {
		return fmt.Errorf("[err] don't exist region \n")
	}

	return nil
}

func setTarget() error {
	region := viper.GetString("region")
	if region == "" {
		return fmt.Errorf("[err] don't exist region \n")
	}

	var err error
	target := viper.GetString("target")
	if target == "" {
		target, err = askTarget(region)
		if err != nil {
			return err
		}
		viper.Set("target", target)
	}

	if target == "" {
		return fmt.Errorf("[err] don't exist running instances \n")
	}

	return nil
}

// interactive CLI
func askRegion() (region string, err error) {
	var regions []string
	svc := ec2.New(awsSession, aws.NewConfig().WithRegion("us-east-1"))
	desc, err := svc.DescribeRegions(nil)
	if err != nil {
		regions = make([]string, len(defaultRegions))
		copy(regions, defaultRegions)
	} else {
		regions = make([]string, 0, len(defaultRegions))
		for _, awsRegion := range desc.Regions {
			regions = append(regions, *awsRegion.RegionName)
		}
	}
	sort.Strings(regions)

	prompt := &survey.Select{
		Message: "Choose a region in AWS:",
		Options: regions,
	}

	if suberr := survey.AskOne(prompt, &region, survey.WithIcons(func(icons *survey.IconSet) {
		icons.SelectFocus.Format = "green+hb"
	}), survey.WithPageSize(20)); suberr != nil {
		err = suberr
		return
	}
	return
}

func askTarget(region string) (target string, err error) {
	svc := ec2.New(awsSession, aws.NewConfig().WithRegion(region))
	input := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("instance-state-name"), Values: []*string{aws.String("running")}},
		},
	}
	output, suberr := svc.DescribeInstances(input)
	if suberr != nil {
		err = suberr
		return
	}

	table := make(map[string]string)
	keyFormat := fmt.Sprint("%s\t(%s)")
	for _, rv := range output.Reservations {
		for _, inst := range rv.Instances {
			name := ""
			for _, tag := range inst.Tags {
				if *tag.Key == "Name" {
					name = *tag.Value
					break
				}
			}
			table[fmt.Sprintf(keyFormat, name, *inst.InstanceId)] = *inst.InstanceId
		}
	}

	options := make([]string, 0, len(table))
	for k, _ := range table {
		options = append(options, k)
	}
	sort.Strings(options)

	if len(options) == 0 {
		return
	}

	prompt := &survey.Select{
		Message: "Choose a target in AWS:",
		Options: options,
	}

	selectKey := ""
	if suberr := survey.AskOne(prompt, &selectKey, survey.WithIcons(func(icons *survey.IconSet) {
		icons.SelectFocus.Format = "green+hb"
	}), survey.WithPageSize(20)); suberr != nil {
		err = suberr
		return
	}
	target = table[selectKey]
	return
}

// Call command
func callSubprocess(process string, args ...string) error {
	call := exec.Command(process, args...)
	call.Stderr = os.Stderr
	call.Stdout = os.Stdout
	call.Stdin = os.Stdin

	// ignore signal(sigint)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT)
	done := make(chan bool, 1)
	go func() {
		for {
			select {
			case <-sigs:
			case <-done:
				break
			}
		}
	}()
	defer close(done)

	// run subprocess
	if err := call.Run(); err != nil {
		return err
	}
	return nil
}

func printReady(cmd string) {
	profile := viper.GetString("profile")
	region := viper.GetString("region")
	target := viper.GetString("target")
	fmt.Printf("[%s] profile: %s, region: %s, target: %s\n", Green(cmd), Yellow(profile),
		Yellow(region), Yellow(target))
}
