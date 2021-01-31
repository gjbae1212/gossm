package cmd

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/fatih/color"
)

// setRegion set region to cred.
func setRegion(c *Credential) error {
	// if region don't exist, get region from prompt
	var err error
	if c.awsRegion == "" {
		c.awsRegion, err = askRegion(c.awsSession)
		if err != nil {
			return err
		}
	}

	if c.awsRegion == "" {
		return fmt.Errorf("[err] don't exist aws region")
	}

	return nil
}

// setTarget set target, domain to ssm.
func setTarget(c *Credential, e *Executor) error {
	if c.awsRegion == "" {
		return fmt.Errorf("[err] don't exist region")
	}

	var err error
	if e.target == "" {
		if e.target, e.domain, err = askTarget(c.awsSession, c.awsRegion); err != nil {
			return err
		}
	} else {
		e.domain, err = findDomainByInstanceId(c.awsSession, c.awsRegion, e.target)
		if err != nil {
			return err
		}
		if e.domain == "" {
			return fmt.Errorf("[err] don't exist running instances")
		}
	}
	return nil
}

// setSSH set ssh command to ssm.
func setSSH(c *Credential, e *Executor) error {
	if c.awsRegion == "" {
		return fmt.Errorf("[err] don't exist region")
	}

	if e.execCommand == "" {
		return setSSHWithCLI(c, e)
	} else {
		e.execCommand = generateExecCommand(e.execCommand, e.sshKey, "", "")
	}

	// parse command
	cmd := strings.TrimSpace(e.execCommand)
	seps := strings.Split(cmd, " ")
	lastArg := seps[len(seps)-1]
	lastArgSeps := strings.Split(lastArg, "@")
	server := lastArgSeps[len(lastArgSeps)-1]
	ips, err := net.LookupIP(server)
	if err != nil {
		color.Red("[err] Invalid exec command")
		color.Yellow("[change] CLI mode")
		return setSSHWithCLI(c, e)
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
		color.Red("[err] Invalid domain name")
		color.Yellow("[change] CLI mode")
		return setSSHWithCLI(c, e)
	}

	// find instanceId By ip
	instanceId, err := findInstanceIdByIp(c.awsSession, c.awsRegion, serverIP)
	if err != nil {
		return err
	}
	if instanceId == "" {
		return fmt.Errorf("[err] not found matching server in your AWS.")
	}
	e.target = instanceId
	return nil
}

// setSCP set scp command to ssm.
func setSCP(c *Credential, e *Executor) error {
	if c.awsRegion == "" {
		return fmt.Errorf("[err] don't exist region")
	}

	if e.execCommand == "" {
		return fmt.Errorf("[err] [required] exec argument")
	}

	// parse command
	cmd := e.execCommand
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
	instanceId, err := findInstanceIdByIp(c.awsSession, c.awsRegion, serverIP)
	if err != nil {
		return err
	}
	if instanceId == "" {
		return fmt.Errorf("[err] not found your server")
	}
	e.target = instanceId
	return nil
}

// setMultiTarget set targets to ssm.
func setMultiTarget(c *Credential, s *Executor) error {
	if c.awsRegion == "" {
		return fmt.Errorf("[err] don't exist region \n")
	}

	if s.target == "" {
		targets, domains, err := askMultiTarget(c.awsSession, c.awsRegion)
		if err != nil {
			return err
		}
		s.multiTarget = targets
		s.multiDomain = domains
	} else {
		domain, err := findDomainByInstanceId(c.awsSession, c.awsRegion, s.target)
		if err != nil {
			return err
		}
		if domain == "" {
			return fmt.Errorf("[err] don't exist running instances \n")
		}

		s.multiTarget = []string{s.target}
		s.multiDomain = []string{domain}
	}
	return nil
}

func setSSHWithCLI(c *Credential, e *Executor) error {
	e.execCommand = ""
	if err := setTarget(c, e); err != nil {
		return err
	}

	user, err := askUser()
	if err != nil {
		return err
	}
	e.user = user

	e.execCommand = generateExecCommand("", e.sshKey, e.user, e.domain)
	return nil
}

// interactive CLI
func askUser() (user string, err error) {
	prompt := &survey.Input{
		Message: "Type your connect ssh user (default: deploy):",
	}
	survey.AskOne(prompt, &user)
	user = strings.TrimSpace(user)
	if user == "" {
		user = "deploy"
	}
	return
}

func askRegion(sess *session.Session) (region string, err error) {
	var regions []string
	svc := ec2.New(sess, aws.NewConfig().WithRegion("us-east-1"))
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

func askTarget(sess *session.Session, region string) (target, domain string, err error) {
	table, suberr := findInstances(sess, region)
	if suberr != nil {
		err = suberr
		return
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
	domain = table[selectKey][1]
	return
}

func askMultiTarget(sess *session.Session, region string) (targets, domains []string, err error) {
	table, suberr := findInstances(sess, region)
	if suberr != nil {
		err = suberr
		return
	}

	options := make([]string, 0, len(table))
	for k, _ := range table {
		options = append(options, k)
	}
	sort.Strings(options)

	if len(options) == 0 {
		return
	}

	prompt := &survey.MultiSelect{
		Message: "Choose targets in AWS:",
		Options: options,
	}

	var selectKeys []string
	if suberr := survey.AskOne(prompt, &selectKeys, survey.WithPageSize(20)); suberr != nil {
		err = suberr
		return
	}

	for _, k := range selectKeys {
		targets = append(targets, table[k][0])
		domains = append(domains, table[k][1])
	}
	return
}

// findInstances finds instances.
func findInstances(sess *session.Session, region string) (map[string][]string, error) {
	svc := ec2.New(sess, aws.NewConfig().WithRegion(region))

	var ec2InstanceIds []string                    // used in the DescribeInstances call to filter results
	var nonEc2Instances []*ssm.InstanceInformation // used to display any non-EC2 managed instances

	managedInstances, err := findManagedInstances(sess, region) // get all ssm connected instances
	if err != nil || len(managedInstances) == 0 {
	} else {
		for _, i := range managedInstances {
			if *i.PingStatus == "Online" { // check instance is connected to ssm
				if *i.ResourceType != "EC2Instance" {
					nonEc2Instances = append(nonEc2Instances, i)
				} else {
					ec2InstanceIds = append(ec2InstanceIds, *i.InstanceId)
				}
			}
		}
	}

	// make a DescribeInstances call to get Tags for each EC2 instance.
	// allows us to display the Name Tag in the askTarget prompt.
	input := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("instance-id"), Values: aws.StringSlice(ec2InstanceIds)},
			{Name: aws.String("instance-state-name"), Values: []*string{aws.String("running")}},
		},
	}
	output, err := svc.DescribeInstances(input)
	if err != nil {
		return nil, err
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
	for _, mi := range nonEc2Instances {
		table[fmt.Sprintf("%s\t(%s)", *mi.Name, *mi.InstanceId)] = []string{*mi.InstanceId, *mi.Name}
	}
	return table, nil
}

// findManagedInstances finds instance list which is possibly connected through ssm agent.
func findManagedInstances(sess *session.Session, region string) ([]*ssm.InstanceInformation, error) {
	svc := ssm.New(sess, aws.NewConfig().WithRegion(region))
	var insts []*ssm.InstanceInformation
	err := svc.DescribeInstanceInformationPages(nil,
		func(page *ssm.DescribeInstanceInformationOutput, lastPage bool) bool {
			for _, inst := range page.InstanceInformationList {
				insts = append(insts, inst)
			}
			return true
		})
	if err != nil {
		return nil, err
	}

	return insts, nil
}

// findInstanceIdByIp finds instanceId by ip.
func findInstanceIdByIp(sess *session.Session, region, ip string) (string, error) {
	svc := ec2.New(sess, aws.NewConfig().WithRegion(region))
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
			if inst.PublicIpAddress == nil || inst.PrivateIpAddress == nil {
				continue
			}
			if ip == *inst.PublicIpAddress || ip == *inst.PrivateIpAddress {
				return *inst.InstanceId, nil
			}
		}
	}
	return "", nil
}

// findDomainByInstanceId finds domain by instanceId.
func findDomainByInstanceId(sess *session.Session, region string, instanceId string) (string, error) {
	svc := ec2.New(sess, aws.NewConfig().WithRegion(region))
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
			if *inst.InstanceId == instanceId {
				return *inst.PublicDnsName, nil
			}
		}
	}
	return "", nil
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

// Create start session
func createStartSession(c *Credential, input *ssm.StartSessionInput) (*ssm.StartSessionOutput, string, error) {
	svc := ssm.New(c.awsSession, aws.NewConfig().WithRegion(c.awsRegion))
	subctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	sess, err := svc.StartSessionWithContext(subctx, input)
	if err != nil {
		return nil, "", err
	}
	return sess, svc.Endpoint, nil
}

// Delete start session
func deleteStartSession(c *Credential, sessionId string) error {
	svc := ssm.New(c.awsSession, aws.NewConfig().WithRegion(c.awsRegion))
	fmt.Printf("%s %s \n", color.YellowString("Delete Session"), color.YellowString(sessionId))
	subctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if _, err := svc.TerminateSessionWithContext(subctx, &ssm.TerminateSessionInput{SessionId: &sessionId}); err != nil {
		return err
	}
	return nil
}

// sendCommand is to request aws ssm to run command.
func sendCommand(sess *session.Session, region string, targets []string, command string) (*ssm.SendCommandOutput, error) {
	svc := ssm.New(sess, aws.NewConfig().WithRegion(region))

	// only support to linux (window = "AWS-RunPowerShellScript")
	docName := "AWS-RunShellScript"

	// set timeout 60 seconds
	timeout := int64(60)
	input := &ssm.SendCommandInput{
		DocumentName:   &docName,
		InstanceIds:    aws.StringSlice(targets),
		TimeoutSeconds: &timeout,
		CloudWatchOutputConfig: &ssm.CloudWatchOutputConfig{
			CloudWatchOutputEnabled: aws.Bool(true),
		},
		Parameters: map[string][]*string{
			"commands": aws.StringSlice([]string{command}),
		},
	}

	return svc.SendCommand(input)
}

// printCommandInvocation prints result for sendCommand.
func printCommandInvocation(sess *session.Session, region string, inputs []*ssm.GetCommandInvocationInput) {
	svc := ssm.New(sess, aws.NewConfig().WithRegion(region))
	wg := new(sync.WaitGroup)

	for _, input := range inputs {
		wg.Add(1)
		go func(input *ssm.GetCommandInvocationInput) {
			subctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
			defer cancel()
		Exit:
			for {
				select {
				case <-time.After(1 * time.Second):
					output, err := svc.GetCommandInvocationWithContext(subctx, input)
					if err != nil {
						color.Red("%v", err)
						break Exit
					}
					status := strings.ToLower(*output.Status)
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

// Print start command
func printReady(cmd string, c *Credential, e *Executor) {
	fmt.Printf("[%s] profile: %s, region: %s, target: %s\n", color.GreenString(cmd), color.YellowString(c.awsProfile),
		color.YellowString(c.awsRegion), color.YellowString(e.target))
}
