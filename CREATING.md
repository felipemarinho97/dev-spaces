# Creating a DevSpace

This document describes how to bootstrap a Dev Space using the command `dev-spaces create`. For a complete list of all the options, run `dev-spaces create --help`.

## SSH Key Pair

Make sure you have a SSH key pair in your AWS account. You can see [here](KEYPAIR.md#create-a-key-pair-to-ssh-into-the-instance) how to create one using the `aws cli`. Also, you can use the [AWS Console](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-key-pairs.html#prepare-key-pair) to create a key pair. If you want your key pair to be availiable in all regions, you can follow [this tutorial](https://aws.amazon.com/premiumsupport/knowledge-center/ec2-ssh-key-pair-regions/) from a AWS Support.

It's also possible to use an existing key pair, just make sure you have the private key file in your local machine. For that, you will have to import the key pair to the region you want to use. You can see the [AWS Documentation](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-key-pairs.html#prepare-key-pair) to import a key pair.

After creating or importing a key pair, you will use the **name** of the key pair on the parameter `--key-name` or `-k` of the `dev-spaces create` command.

## Creating the Space

The simplest way to create a Dev Space is by using the `dev-spaces create` command with the `--key-name` and `--name` parameters. This will create a Dev Space with the default options. The default AMI (`--ami`) is the latest Amazon Linux available. This AMI have the advantage of being more stable, having faster startup times and is well tested in this project.

```bash
$ export AWS_REGION=us-east-1
$ dev-spaces create -n MyDevSpace -k MyKeyPair
```

If you want to use a different AMI, you can use the `--ami` (`-i`) parameter.

```bash
$ dev-spaces create -n MyAmazonLinux2023 -k MyKeyPair -i 'owner:amazon,name:*al2023*minimal*'
```

See the [Recommended AMIs](#recommended-amis) section for more information.

For all the available options, run `dev-spaces create --help`. Also, see the [Command Parameters](#command-parameters) section for more information.

> **Note**
If you are using your AWS account for the first time, it is possible that the first attempt fail on "CreateFleet" because the AWS is performing additional validation on your account. You should receive an email from AWS informing you about this. If that happens, just try again after a few minutes.

Once created, you can use the command `dev-spaces start` to start the space.

    $ dev-spaces start -n MyAmazonLinux2023 -c 1 -m 1 --wait

Now, your instance will be available to ssh into from the port `2222`.

## Command Parameters

_Parameter_|_Alias_|_Description_|_Example_|_Mandatory_
|:--:|:--:|:--:|:--:|:--:|
|`--name`|`-n`|The name of the Dev Space|`MyDevSpace`|✓|
|`--key-name`|`-k`|The name of the SSH key pair|`MyKeyPair`|✓|
|`--ami`|`-i`|Amazon Machine Image to use|`ami-034b81f0f1dd96797`| |
|`--prefered-instance-type`|`-t`|The type of the instance|`t2.micro`| |
|`--instance-profile-arn`|`-p`|The ARN of the instance profile|`arn:aws:iam::123456789012:instance-profile/MyInstanceProfile`| |
|`--custom-host-ami`|-|Custom AMI to use for the host - use this flag in combination with `--custom-startup-script`|`ami-034b81f0f1dd96797`| |
|`--custom-startup-script`|-|Custom startup script file to use for the host|`./myscript.sh`| |
|`--security-group-ids`|-|A list of IDs of the security groups to use|`sg-12345678`| |

## Troubleshooting

If for some reason you can't SSH into the instance, you can troubleshoot what's wrong by looking at the logs of the host machine.

Also, try logging into the instance using the root user (port 2222).

SSH into the host instance (port 22):

    $ ssh -i MyKeyPair.pem -p 22 ec2-user@<IP_ADDRESS>
    ec2-user:~$ cat /var/log/user-data.log
    ec2-user:~$ cat /var/log/cloud-init-output.log
    ec2-user:~$ cat /var/log/cloud-init.log

Common problems are user-script failing to run due to wrong filesystem mount type, wrong permissions, etc.

## Recommended AMIs

It's highly recommended to use the following AMIs, as they were tested and are known to work with Dev Spaces. The **Amazon Linux** ones are the most recommended.
For better startup time, use the `minimal` version of the AMI.

| Image | Description | Source | Selector |
| ----- | ----------- | ------ | --------- |
| amazon-linux-2023-minimal | Amazon Linux 2023 | [Amazon Linux](https://aws.amazon.com/linux/amazon-linux-2023/) | `owner:amazon,name:*al2023*minimal*` |
| amazon-linux-2023 | Amazon Linux 2023 | [Amazon Linux](https://aws.amazon.com/linux/amazon-linux-2023/) | `owner:amazon,name:*al2023*ami-2023*` |
| amazon-linux-2022 | Amazon Linux 2022 | [Amazon Linux](https://aws.amazon.com/amazon-linux-2/release-notes/) | `owner:amazon,name:*al2022*` |
| amazon-linux-2 | Amazon Linux 2 | [Amazon Linux](https://aws.amazon.com/amazon-linux-2/release-notes/) | `owner:amazon,name:*amzn2*` |
| ubuntu-focal-20.04-server | Ubuntu 20.04 LTS | [Ubuntu](https://cloud-images.ubuntu.com/focal/current/) | `owner:099720109477,name:ubuntu*hvm-ssd*20.04*amd64-server*` |
| ubuntu-bionic-18.04-server | Ubuntu 18.04 LTS | [Ubuntu](https://cloud-images.ubuntu.com/bionic/current/) | `owner:099720109477,name:ubuntu*hvm-ssd*18.04*amd64-server*` |

If you have any issues with any of these AMIs, please open an issue. Also, if you found an AMI that works well with DevSpaces, please open a PR to add it to this list.

## Extra: Building a Ultra-Optimized DevSpace

This section will teach you how to build a DevSpace with a very thin host image, this will make the startup time of the DevSpace much faster and will also reduce the costs of the host EBS volume.

### Step 1: Create a Host Image

Clone [this repository](https://github.com/felipemarinho97/packer-images):

    $ git clone https://github.com/felipemarinho97/packer-images

You will need to install the [packer](https://www.packer.io/) tool, follow the instructions on the [packer website](https://www.packer.io/docs/).

Make sure your AWS credentials are set up correctly.

Inside the `packer-images` directory, run the following command to build the host image:

    $ cd packer-images
    $ packer build base

Once the image is built, `packer` will output the AMI ID of the host image on your console.

### Step 2: Use the base-minimal image to create the optimized DevSpace

Now, you can use the AMI ID of the `base-minimal` image to create the optimized DevSpace. Also, you will need to provide a startup script compatible with your custom host AMI. You can use [this sample](examples/scripts/startup-script.sh) directly on the create command, or you can create your own startup script.

```bash
    $ dev-spaces create -n MyDevSpace \
        -k MyKeyPair \
        -i <your-devsapce-ami> \
        --custom-host-ami <base-minimal-ami-id> # <-- this is the AMI ID of the image created by packer
        --custom-startup-script 'https://raw.githubusercontent.com/felipemarinho97/dev-spaces/master/examples/scripts/startup-script.sh'

```

Done! This optimized DevSpace will take only few seconds to start.
