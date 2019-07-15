package cmd

// ssh -i pem키 -o ProxyCommand="aws ssm start-session --target 인스턴스ID --document-name AWS-StartSSHSession --parameters 'portNumber=%p'" 유저@IP또는도메인
