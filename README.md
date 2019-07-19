# gossm

gossm is interactive CLI tool that you should select server in AWS and then could connect or send files your AWS server using start-session, ssh, scp under AWS Systems Manger Session Manager.
<p align="center">
<img src="https://storage.googleapis.com/gjbae1212-asset/gossm/start.gif" width="500", height="450" />
</p>

<p align="center"/>
<a href="https://circleci.com/gh/gjbae1212/gossm"><img src="https://circleci.com/gh/gjbae1212/gossm.svg?style=svg"></a>
<!-- <a href="https://hits.seeyoufarm.com"/><img src="https://hits.seeyoufarm.com/api/count/incr/badge.svg?url=https%3A%2F%2Fgithub.com%2Fgjbae1212%2Fgossm"/></a> -->
<a href="/LICENSE"><img src="https://img.shields.io/badge/license-MIT-GREEN.svg" alt="license" /></a>
<a href="https://goreportcard.com/report/github.com/gjbae1212/gossm"><img src="https://goreportcard.com/badge/github.com/gjbae1212/gossm" alt="Go Report Card"/></a>
</p>

## Overview
gossm is interactive CLI tool that is related AWS Systems Manger Session Manager.
It can select a ec2 server installed aws-ssm-agent and then can connect its server using start-session, ssh.
As well as files can send using scp.
     
## Prerequisite 
- [required] Your ec2 servers in aws are installed [aws ssm agent](https://docs.aws.amazon.com/systems-manager/latest/userguide/ssm-agent.html).
EC2 severs have to apply AmazonEC2RoleforSSM iam policy.     
If you would like to use ssh, scp command using gossm, aws ssm agent version 2.3.672.0 or later is installed on ec2. 
- [required] **aws access key**, **aws secret key**
- [required] **ec2:DescribeInstances**, **ssm:StartSession permission**    
- [optional] It's better to possibly get to additional permission for **ec2:DescribeRegions**, **ssm:TerminateSession**

## Install
```bash
# homebrew

# mac

# linux

# window

```

## How to use
### global command args
| args           | Description                                               | Default                |
| ---------------|-----------------------------------------------------------|------------------------|
| -c             | (optional) aws credentials file | $HOME/.aws/.credentials |
| -p             | (optional) if you are having multiple aws profiles in credentials, it is name one of profiles | default |
| -r             | (optional) region in AWS that would like to connect |  |
| -t             | (optional) instanceId of server in AWS that would like to connect | |

- If your machine don't exist $HOME/.aws/.credentials, have to pass `-c` args.  
```
# credentials file format
[default]
aws_access_key_id = AWS ACCESS KEY
aws_secret_access_key = AWS SECRET KEY
```
- `-r` or `-t` don't pass args, it can select through interactive CLI.  
### command
#### start
<p align="center">
<img src="https://storage.googleapis.com/gjbae1212-asset/gossm/start.gif" width="500", height="450" />
</p>

#### ssh
<p align="center">
<img src="https://storage.googleapis.com/gjbae1212-asset/gossm/ssh.gif" width="500", height="450" />
</p> 

### scp
<p align="center">
<img src="https://storage.googleapis.com/gjbae1212-asset/gossm/scp.gif" width="500", height="450" />
</p>

## LICENSE
This project is following The MIT.