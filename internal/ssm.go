package internal

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2_types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssm_types "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/fatih/color"
)

const (
	maxOutputResults = 50
)

var (
	// default aws regions
	defaultAwsRegions = []string{
		"af-south-1",
		"ap-east-1", "ap-northeast-1", "ap-northeast-2", "ap-northeast-3", "ap-south-1", "ap-southeast-2", "ap-southeast-3",
		"ca-central-1",
		"cn-north-1", "cn-northwest-1",
		"eu-central-1", "eu-north-1", "eu-south-1", "eu-west-1", "eu-west-2", "eu-west-3",
		"me-south-1",
		"sa-east-1",
		"us-east-1", "us-east-2", "us-gov-east-1", "us-gov-west-2", "us-west-1", "us-west-2",
	}
)

type (
	Target struct {
		Name          string
		PublicDomain  string
		PrivateDomain string
	}

	User struct {
		Name string
	}

	Region struct {
		Name string
	}

	Port struct {
		Remote string
		Local  string
	}
)

// AskUser asks you which selects a user.
func AskUser() (*User, error) {
	prompt := &survey.Input{
		Message: "Type your connect ssh user (default: root):",
	}
	var user string
	survey.AskOne(prompt, &user)
	user = strings.TrimSpace(user)
	if user == "" {
		user = "root"
	}
	return &User{Name: user}, nil
}

// AskRegion asks you which selects a region.
func AskRegion(ctx context.Context, cfg aws.Config) (*Region, error) {
	var regions []string
	client := ec2.NewFromConfig(cfg)

	output, err := client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{
		AllRegions: aws.Bool(true),
	})
	if err != nil {
		regions = make([]string, len(defaultAwsRegions))
		copy(regions, defaultAwsRegions)
	} else {
		regions = make([]string, len(output.Regions))
		for _, region := range output.Regions {
			regions = append(regions, aws.ToString(region.RegionName))
		}
	}
	sort.Strings(regions)

	var region string
	prompt := &survey.Select{
		Message: "Choose a region in AWS:",
		Options: regions,
	}
	if err := survey.AskOne(prompt, &region, survey.WithIcons(func(icons *survey.IconSet) {
		icons.SelectFocus.Format = "green+hb"
	}), survey.WithPageSize(20)); err != nil {
		return nil, err
	}

	return &Region{Name: region}, nil
}

// AskTarget asks you which selects an instance.
func AskTarget(ctx context.Context, cfg aws.Config) (*Target, error) {
	table, err := FindInstances(ctx, cfg)
	if err != nil {
		return nil, err
	}

	options := make([]string, 0, len(table))
	for k, _ := range table {
		options = append(options, k)
	}
	sort.Strings(options)
	if len(options) == 0 {
		return nil, fmt.Errorf("not found ec2 instances")
	}

	prompt := &survey.Select{
		Message: "Choose a target in AWS:",
		Options: options,
	}

	selectKey := ""
	if err := survey.AskOne(prompt, &selectKey, survey.WithIcons(func(icons *survey.IconSet) {
		icons.SelectFocus.Format = "green+hb"
	}), survey.WithPageSize(20)); err != nil {
		return nil, err
	}

	return table[selectKey], nil
}

// AskMultiTarget asks you which selects multi targets.
func AskMultiTarget(ctx context.Context, cfg aws.Config) ([]*Target, error) {
	table, err := FindInstances(ctx, cfg)
	if err != nil {
		return nil, err
	}

	options := make([]string, 0, len(table))
	for k, _ := range table {
		options = append(options, k)
	}
	sort.Strings(options)
	if len(options) == 0 {
		return nil, fmt.Errorf("not found multi-target")
	}

	prompt := &survey.MultiSelect{
		Message: "Choose targets in AWS:",
		Options: options,
	}

	var selectKeys []string
	if err := survey.AskOne(prompt, &selectKeys, survey.WithPageSize(20)); err != nil {
		return nil, err
	}

	var targets []*Target
	for _, k := range selectKeys {
		targets = append(targets, table[k])
	}
	return targets, nil
}

// AskPorts asks you which select ports.
func AskPorts() (port *Port, retErr error) {
	port = &Port{}
	prompts := []*survey.Question{
		{
			Name:   "remote",
			Prompt: &survey.Input{Message: "Remote port to access:"},
		},
		{
			Name:   "local",
			Prompt: &survey.Input{Message: "Local port number to forward:"},
		},
	}
	if err := survey.Ask(prompts, port); err != nil {
		retErr = WrapError(err)
		return
	}
	if _, err := strconv.Atoi(strings.TrimSpace(port.Remote)); err != nil {
		retErr = errors.New("you must specify a valid port number")
		return
	}
	if port.Local == "" {
		port.Local = port.Remote
	}

	if len(port.Remote) > 5 || len(port.Local) > 5 {
		retErr = errors.New("you must specify a valid port number")
		return
	}

	return
}

// FindInstances returns all of instances-map with running state.
func FindInstances(ctx context.Context, cfg aws.Config) (map[string]*Target, error) {
	var (
		client     = ec2.NewFromConfig(cfg)
		table      = make(map[string]*Target)
		outputFunc = func(table map[string]*Target, output *ec2.DescribeInstancesOutput) {
			for _, rv := range output.Reservations {
				for _, inst := range rv.Instances {
					name := ""
					for _, tag := range inst.Tags {
						if aws.ToString(tag.Key) == "Name" {
							name = aws.ToString(tag.Value)
							break
						}
					}
					table[fmt.Sprintf("%s\t(%s)", name, *inst.InstanceId)] = &Target{
						Name:          aws.ToString(inst.InstanceId),
						PublicDomain:  aws.ToString(inst.PublicDnsName),
						PrivateDomain: aws.ToString(inst.PrivateDnsName),
					}
				}
			}
		}
	)

	// get instance ids which possibly can connect to instances using ssm.
	instances, err := FindInstanceIdsWithConnectedSSM(ctx, cfg)
	if err != nil {
		return nil, err
	}

	for len(instances) > 0 {
		max := len(instances)
		// The maximum number of filter values specified on a single call is 200.
		if max >= 200 {
			max = 199
		}
		output, err := client.DescribeInstances(ctx,
			&ec2.DescribeInstancesInput{
				Filters: []ec2_types.Filter{
					{Name: aws.String("instance-state-name"), Values: []string{"running"}},
					{Name: aws.String("instance-id"), Values: instances[:max]},
				},
			})
		if err != nil {
			return nil, err
		}
		outputFunc(table, output)
		instances = instances[max:]
	}

	return table, nil
}

// FindInstanceIdsWithConnectedSSM asks you which selects instances.
func FindInstanceIdsWithConnectedSSM(ctx context.Context, cfg aws.Config) ([]string, error) {
	var (
		instances  []string
		client     = ssm.NewFromConfig(cfg)
		outputFunc = func(instances []string, output *ssm.DescribeInstanceInformationOutput) []string {
			for _, inst := range output.InstanceInformationList {
				instances = append(instances, aws.ToString(inst.InstanceId))
			}
			return instances
		}
	)

	output, err := client.DescribeInstanceInformation(ctx, &ssm.DescribeInstanceInformationInput{MaxResults: maxOutputResults})
	if err != nil {
		return nil, err
	}
	instances = outputFunc(instances, output)

	// Repeat it when if output.NextToken exists.
	if aws.ToString(output.NextToken) != "" {
		token := aws.ToString(output.NextToken)
		for {
			if token == "" {
				break
			}
			nextOutput, err := client.DescribeInstanceInformation(ctx, &ssm.DescribeInstanceInformationInput{
				NextToken:  aws.String(token),
				MaxResults: maxOutputResults})
			if err != nil {
				return nil, err
			}
			instances = outputFunc(instances, nextOutput)

			token = aws.ToString(nextOutput.NextToken)
		}
	}

	return instances, nil
}

// FindInstanceIdByIp returns instance ids by ip.
func FindInstanceIdByIp(ctx context.Context, cfg aws.Config, ip string) (string, error) {
	var (
		instanceId string
		client     = ec2.NewFromConfig(cfg)
		outputFunc = func(output *ec2.DescribeInstancesOutput) string {
			for _, rv := range output.Reservations {
				for _, inst := range rv.Instances {
					if inst.PublicIpAddress == nil && inst.PrivateIpAddress == nil {
						continue
					}
					if ip == aws.ToString(inst.PublicIpAddress) || ip == aws.ToString(inst.PrivateIpAddress) {
						return *inst.InstanceId
					}
				}
			}
			return ""
		}
	)

	output, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		MaxResults: aws.Int32(maxOutputResults),
		Filters: []ec2_types.Filter{
			{Name: aws.String("instance-state-name"), Values: []string{"running"}},
		},
	})
	if err != nil {
		return "", err
	}

	instanceId = outputFunc(output)
	if instanceId != "" {
		return instanceId, nil
	}

	// Repeat it when if instanceId isn't found and output.NextToken exists.
	if aws.ToString(output.NextToken) != "" {
		token := aws.ToString(output.NextToken)
		for {
			if token == "" {
				break
			}
			nextOutput, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
				MaxResults: aws.Int32(maxOutputResults),
				NextToken:  aws.String(token),
				Filters: []ec2_types.Filter{
					{Name: aws.String("instance-state-name"), Values: []string{"running"}},
				},
			})
			if err != nil {
				return "", err
			}

			instanceId = outputFunc(nextOutput)
			if instanceId != "" {
				return instanceId, nil
			}

			token = aws.ToString(nextOutput.NextToken)
		}
	}

	return "", nil
}

// FindDomainByInstanceId returns domain by instance id.
func FindDomainByInstanceId(ctx context.Context, cfg aws.Config, instanceId string) ([]string, error) {
	var (
		domain     []string
		client     = ec2.NewFromConfig(cfg)
		outputFunc = func(output *ec2.DescribeInstancesOutput, id string) []string {
			for _, rv := range output.Reservations {
				for _, inst := range rv.Instances {
					if aws.ToString(inst.InstanceId) == id {
						return []string{aws.ToString(inst.PublicDnsName), aws.ToString(inst.PrivateDnsName)}
					}
				}
			}
			return []string{}
		}
	)

	output, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		MaxResults: aws.Int32(maxOutputResults),
		Filters: []ec2_types.Filter{
			{Name: aws.String("instance-state-name"), Values: []string{"running"}},
		},
	})
	if err != nil {
		return []string{}, err
	}

	domain = outputFunc(output, instanceId)
	if len(domain) != 0 {
		return domain, nil
	}

	// Repeat it when if domain isn't found and output.NextToken exists.
	if aws.ToString(output.NextToken) != "" {
		token := aws.ToString(output.NextToken)
		for {
			if token == "" {
				break
			}
			nextOutput, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
				MaxResults: aws.Int32(maxOutputResults),
				NextToken:  aws.String(token),
				Filters: []ec2_types.Filter{
					{Name: aws.String("instance-state-name"), Values: []string{"running"}},
				},
			})
			if err != nil {
				return []string{}, err
			}

			domain = outputFunc(nextOutput, instanceId)
			if len(domain) != 0 {
				return domain, nil
			}

			token = aws.ToString(nextOutput.NextToken)
		}
	}

	return []string{}, nil
}

// AskUser asks you which selects a user.
func AskHost() (host string, retErr error) {
	prompt := &survey.Input{
		Message: "Type your host address you want to forward to:",
	}
	survey.AskOne(prompt, &host)
	host = strings.TrimSpace(host)
	if host == "" {
		retErr = errors.New("you must specify a host address")
		return
	}
	return
}

// CreateStartSession creates start session.
func CreateStartSession(ctx context.Context, cfg aws.Config, input *ssm.StartSessionInput) (*ssm.StartSessionOutput, error) {
	client := ssm.NewFromConfig(cfg)

	return client.StartSession(ctx, input)
}

// DeleteStartSession creates session.
func DeleteStartSession(ctx context.Context, cfg aws.Config, input *ssm.TerminateSessionInput) error {
	client := ssm.NewFromConfig(cfg)
	fmt.Printf("%s %s \n", color.YellowString("Delete Session"),
		color.YellowString(aws.ToString(input.SessionId)))

	_, err := client.TerminateSession(ctx, input)
	return err
}

// SendCommand send commands to instance targets.
func SendCommand(ctx context.Context, cfg aws.Config, targets []*Target, command string) (*ssm.SendCommandOutput, error) {
	client := ssm.NewFromConfig(cfg)

	// only support to linux (window = "AWS-RunPowerShellScript")
	docName := "AWS-RunShellScript"

	var ids []string
	for _, t := range targets {
		ids = append(ids, t.Name)
	}

	input := &ssm.SendCommandInput{
		DocumentName:   &docName,
		InstanceIds:    ids,
		TimeoutSeconds: 60,
		CloudWatchOutputConfig: &ssm_types.CloudWatchOutputConfig{
			CloudWatchOutputEnabled: true,
		},
		Parameters: map[string][]string{"commands": []string{command}},
	}

	return client.SendCommand(ctx, input)
}

// PrintCommandInvocation watches command invocations.
func PrintCommandInvocation(ctx context.Context, cfg aws.Config, inputs []*ssm.GetCommandInvocationInput) {
	client := ssm.NewFromConfig(cfg)

	wg := new(sync.WaitGroup)
	for _, input := range inputs {
		wg.Add(1)
		go func(input *ssm.GetCommandInvocationInput) {
		Exit:
			for {
				select {
				case <-time.After(1 * time.Second):
					output, err := client.GetCommandInvocation(ctx, input)
					if err != nil {
						color.Red("%v", err)
						break Exit
					}
					status := strings.ToLower(string(output.Status))
					switch status {
					case "pending", "inprogress", "delayed":
					case "success":
						fmt.Printf("[%s][%s] %s\n", color.GreenString("success"), color.YellowString(*output.InstanceId), color.GreenString(*output.StandardOutputContent))
						break Exit
					default:
						fmt.Printf("[%s][%s] %s\n", color.RedString("err"), color.YellowString(*output.InstanceId), color.RedString(*output.StandardErrorContent))
						break Exit
					}
				}
			}
			wg.Done()
		}(input)
	}

	wg.Wait()
}

// GenerateSSHExecCommand generates ssh exec command.
func GenerateSSHExecCommand(exec, identity, user, domain string) (newExec string) {
	if exec == "" {
		newExec = fmt.Sprintf("%s@%s", user, domain)
	} else {
		newExec = exec
	}

	opt := false
	for _, sep := range strings.Split(newExec, " ") {
		if sep == "-i" {
			opt = true
			break
		}
	}
	// if current ssh-exec don't exist -i option
	if !opt && identity != "" {
		// injection -i option
		newExec = fmt.Sprintf("-i %s %s", identity, newExec)
	}

	return
}

func PrintReady(cmd, region, target string) {
	fmt.Printf("[%s] region: %s, target: %s\n", color.GreenString(cmd), color.YellowString(region), color.YellowString(target))
}

// CallProcess calls process.
func CallProcess(process string, args ...string) error {
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
		return WrapError(err)
	}
	return nil
}
