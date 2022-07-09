# Creating a DevSpace

This document describes how to bootstrap a Dev Space using the command `dev-spaces create`. This new version has a better approach on creating the environment because it's more generic and it's easier to manage. Also, it's much more faster (just take few seconds to bootstap a space) than the previous version.

## Instance Profile

There is no need to create an instance profile with EBS permissions because the `start` command will wait until the instance is ready and attach the EBS Volume by itself. You can still specify the instance profile in the create command if you want to.

## SSH Key Pair

Make sure you have a SSH key pair in your AWS account. You can see [here](BOOTSTRAP_v1.md#create-a-key-pair-to-ssh-into-the-instance) how to create one using the `aws cli`. If you want your key pair to be availiable in all regions, you can follow [this tutorial](https://aws.amazon.com/premiumsupport/knowledge-center/ec2-ssh-key-pair-regions/) from a AWS Support.

## Creating the Space

The command below will use the pre-defined template to create an space with Amazon Linux 2022 AMI.

    $ dev-spaces create -n MyAmazonLinux2022 -k MyKeyPair -i ami-034b81f0f1dd96797

This AMI have the advantage of supporting running docker inside the Dev Space.

## Command Parameters

_Parameter_|_Alias_|_Description_|_Example_|
|:--:|:--:|:--:|:--:|
|`--name`|`-n`|The name of the Dev Space|`MyDevSpace`|
|`--key-name`|`-k`|The name of the SSH key pair|`MyKeyPair`|
|`--ami`|`-a`|The AMI ID|`ami-034b81f0f1dd96797`|
|`--prefered-instance-type`|`-t`|The type of the instance|`t2.micro`|
|`--instance-profile-arn`|`-p`|The ARN of the instance profile|`arn:aws:iam::123456789012:instance-profile/MyInstanceProfile`|

## Troubleshooting

If for some reason you can't SSH into the instance, you can troubleshoot what's wrong by looking at the logs of the host machine.

SSH into the host instance (port 22):

    $ ssh -i MyKeyPair.pem -p 22 ec2-user@<IP_ADDRESS>
    ec2-user:~$ cat /var/log/user-data.log
    ec2-user:~$ cat /var/log/cloud-init-output.log
    ec2-user:~$ cat /var/log/cloud-init.log

Common problems are user-script failing to run due to wrong filesystem mount type, wrong permissions, etc.