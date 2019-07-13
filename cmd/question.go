package cmd

import (
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

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

func askInstance(region string) (instance string, err error) {
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
		Message: "Choose a instance in AWS:",
		Options: options,
	}

	selectKey := ""
	if suberr := survey.AskOne(prompt, &selectKey, survey.WithIcons(func(icons *survey.IconSet) {
		icons.SelectFocus.Format = "green+hb"
	}), survey.WithPageSize(20)); suberr != nil {
		err = suberr
		return
	}
	instance = table[selectKey]
	return
}