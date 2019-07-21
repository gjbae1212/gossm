package cmd

import (
	"context"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/service/ssm"

	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/aws/aws-sdk-go/aws"
	. "github.com/logrusorgru/aurora"
	"github.com/spf13/viper"
)

// Set scp from interactive CLI and then its params set to viper
func setSCP() error {
	if viper.GetString("scp-exec") == "" {
		return fmt.Errorf("[err] [required] exec argument")
	}

	if viper.GetString("region") == "" {
		return fmt.Errorf("[err] don't exist region")
	}

	// parse command
	cmd := strings.TrimSpace(viper.GetString("scp-exec"))
	seps := strings.Split(cmd, " ")
	if len(seps) < 2 {
		return fmt.Errorf("[err] invalid exec argument")
	}

	dst := seps[len(seps)-1]
	dstSeps := strings.Split(strings.Split(dst, ":")[0], "@")

	seps = strings.Split(strings.TrimSpace(strings.Join(seps[0:(len(seps)-1)], " ")), " ")

	src := seps[len(seps)-1]
	srcSeps := strings.Split(strings.Split(src, ":")[0], "@")

	// lookup domain
	serverIP := ""
	var ips []net.IP
	var err error
	switch {
	case len(srcSeps) == 2:
		ips, err = net.LookupIP(srcSeps[1])
	case len(dstSeps) == 2:
		ips, err = net.LookupIP(dstSeps[1])
	default:
		return fmt.Errorf("[err] invalid scp args")
	}
	if err != nil {
		return fmt.Errorf("[err] invalid server domain name")
	}

	for _, ip := range ips {
		if ip.To4() != nil {
			serverIP = ip.String()
			break
		}
	}

	if serverIP == "" {
		return fmt.Errorf("[err] not found server domain name in DNS")
	}

	// find instanceId By ip
	instanceId, err := findInstanceIdByIp(viper.GetString("region"), serverIP)
	if err != nil {
		return err
	}
	if instanceId == "" {
		return fmt.Errorf("[err] not found your server")
	}

	viper.Set("target", instanceId)
	return nil
}

// Set ssh  from interactive CLI and then its params set to viper
func setSSH() error {
	if viper.GetString("region") == "" {
		return fmt.Errorf("[err] don't exist region")
	}

	if viper.GetString("ssh-exec") == "" {
		return setSSHWithCLI()
	} else {
		exec := generateExecCommand(viper.GetString("ssh-exec"),
			viper.GetString("ssh-identity"), "", "")
		viper.Set("ssh-exec", exec)
	}

	// parse command
	cmd := strings.TrimSpace(viper.GetString("ssh-exec"))
	seps := strings.Split(cmd, " ")
	lastArg := seps[len(seps)-1]
	lastArgSeps := strings.Split(lastArg, "@")
	server := lastArgSeps[len(lastArgSeps)-1]
	ips, err := net.LookupIP(server)
	if err != nil {
		fmt.Printf("%s\n\n", Red("[err] Invalid exec command"))
		fmt.Printf("%s\n\n", Yellow("[changing] CLI mode"))
		return setSSHWithCLI()
	}

	// lookup domain
	serverIP := ""
	for _, ip := range ips {
		if ip.To4() != nil {
			serverIP = ip.String()
			break
		}
	}
	if serverIP == "" {
		fmt.Printf("%s\n\n", Red("[err] Invalid domain name"))
		fmt.Printf("%s\n\n", Yellow("[changing] CLI mode"))
		return setSSHWithCLI()
	}

	// find instanceId By ip
	instanceId, err := findInstanceIdByIp(viper.GetString("region"), serverIP)
	if err != nil {
		return err
	}
	if instanceId == "" {
		return fmt.Errorf("[err] not found matching server in your AWS.")
	}
	viper.Set("target", instanceId)
	return nil
}

// Set region from interactive CLI and then its params set to viper
func setRegion() error {
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

// Set target from interactive CLI and then its params set to viper
func setTarget() error {
	region := viper.GetString("region")
	if region == "" {
		return fmt.Errorf("[err] don't exist region \n")
	}

	var err error
	target := viper.GetString("target")
	publicdns := ""
	if target == "" {
		target, publicdns, err = askTarget(region)
		if err != nil {
			return err
		}
		viper.Set("target", target)
		viper.Set("publicdns", publicdns)
	}

	if target == "" {
		return fmt.Errorf("[err] don't exist running instances \n")
	}

	return nil
}

// Set user from interactive CLI and then its params set to viper
func setUser() error {
	user, err := askUser()
	if err != nil {
		return err
	}
	viper.Set("user", user)
	return nil
}

func setSSHWithCLI() error {
	viper.Set("ssh-exec", "")
	if err := setTarget(); err != nil {
		return err
	}
	if err := setUser(); err != nil {
		return err
	}
	exec := generateExecCommand("",
		viper.GetString("ssh-identity"),
		viper.GetString("user"),
		viper.GetString("publicdns"))
	viper.Set("ssh-exec", exec)
	return nil
}

// interactive CLI
func askUser() (user string, err error) {
	prompt := &survey.Input{
		Message: "Type your connect user (default: root):",
	}
	survey.AskOne(prompt, &user)
	user = strings.TrimSpace(user)
	if user == "" {
		user = "root"
	}
	return
}

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

func askTarget(region string) (target, publicdns string, err error) {
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

	table := make(map[string][]string)
	for _, rv := range output.Reservations {
		for _, inst := range rv.Instances {
			name := ""
			for _, tag := range inst.Tags {
				if *tag.Key == "Name" {
					name = *tag.Value
					break
				}
			}
			table[fmt.Sprintf("%s\t(%s)", name, *inst.InstanceId)] = []string{*inst.InstanceId, *inst.PublicDnsName}
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
	target = table[selectKey][0]
	publicdns = table[selectKey][1]
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

// Print start command
func printReady(cmd string) {
	profile := viper.GetString("profile")
	region := viper.GetString("region")
	target := viper.GetString("target")
	fmt.Printf("[%s] profile: %s, region: %s, target: %s\n", Green(cmd), Yellow(profile),
		Yellow(region), Yellow(target))
}

// Create start session
func createStartSession(region string, input *ssm.StartSessionInput) (*ssm.StartSessionOutput, string, error) {
	svc := ssm.New(awsSession, aws.NewConfig().WithRegion(region))
	subctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	sess, err := svc.StartSessionWithContext(subctx, input)
	if err != nil {
		return nil, "", err
	}
	return sess, svc.Endpoint, nil
}

// Delete start session
func deleteStartSession(region, sessionId string) error {
	svc := ssm.New(awsSession, aws.NewConfig().WithRegion(region))
	fmt.Printf("%s %s \n", Yellow("Delete Session"), Yellow(sessionId))
	subctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if _, err := svc.TerminateSessionWithContext(subctx, &ssm.TerminateSessionInput{SessionId: &sessionId}); err != nil {
		return err
	}
	return nil
}

// Find IP
func findInstanceIdByIp(region, ip string) (string, error) {
	svc := ec2.New(awsSession, aws.NewConfig().WithRegion(region))
	input := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("instance-state-name"), Values: []*string{aws.String("running")}},
		},
	}

	output, err := svc.DescribeInstances(input)
	if err != nil {
		return "", err
	}
	for _, rv := range output.Reservations {
		for _, inst := range rv.Instances {
			if ip == *inst.PublicIpAddress || ip == *inst.PrivateIpAddress {
				return *inst.InstanceId, nil
			}
		}
	}
	return "", nil
}

// Generate ssh-exec
func generateExecCommand(exec, identity, user, domain string) (newExec string) {
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
