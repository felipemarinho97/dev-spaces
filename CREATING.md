# Creating a DevSpace

This document describes how to bootstrap a Dev Space using the command `dev-spaces create`. This new version has a better approach on creating the environment because it's more generic and it's easier to manage. Also, it's much more faster (just take few seconds to bootstap a space) than the previous version.

## Instance Profile

There is no need to create an instance profile with EBS permissions because the `start` command will wait until the instance is ready and attach the EBS Volume by itself. You can still specify the instance profile in the create command if you want to.

## SSH Key Pair

Make sure you have a SSH key pair in your AWS account. You can see [here](BOOTSTRAPPING.md#create-a-key-pair-to-ssh-into-the-instance) how to create one using the `aws cli`. If you want your key pair to be availiable in all regions, you can follow [this tutorial](https://aws.amazon.com/premiumsupport/knowledge-center/ec2-ssh-key-pair-regions/) from a AWS Support.

## Creating the Space

The command below will use the pre-defined template to create an space with Amazon Linux 2022 AMI.

    $ dev-spaces create -n MyAmazonLinux2022 -k MyKeyPair -i ami-034b81f0f1dd96797

If you don't know an AMI ID, you can use the filter syntax to find the latest AMI for the region that matches your criteria. Here is an example using Canonical owner id, and ubuntu 22.04 image.

    $ dev-spaces create -n Ubuntu -k MyKeyPair --ami 'name:ubuntu*22.04*,arch:x86_64,owner:099720109477'

This AMI have the advantage of supporting running docker inside the Dev Space.

Once created, you can use the command `dev-spaces start` to start the space.

    $ dev-spaces start -n MyAmazonLinux2022

Now, your instance will be available to ssh into from the port `2222`.

## Command Parameters

_Parameter_|_Alias_|_Description_|_Example_|
|:--:|:--:|:--:|:--:|
|`--name`|`-n`|The name of the Dev Space|`MyDevSpace`|
|`--key-name`|`-k`|The name of the SSH key pair|`MyKeyPair`|
|`--ami`|`-a`|The AMI ID|`ami-034b81f0f1dd96797`|
|`--prefered-instance-type`|`-t`|The type of the instance|`t2.micro`|
|`--instance-profile-arn`|`-p`|The ARN of the instance profile|`arn:aws:iam::123456789012:instance-profile/MyInstanceProfile`|
|`--custom-host-ami`|-|Custom AMI to use for the host - use this flag in combination with `--custom-startup-script`|`ami-034b81f0f1dd96797`|
|`--custom-startup-script`|-|Custom startup script file to use for the host|`./myscript.sh`|
|`--security-group-ids`|-|A list of IDs of the security groups to use|`sg-12345678`|

## Troubleshooting

If for some reason you can't SSH into the instance, you can troubleshoot what's wrong by looking at the logs of the host machine.

Also, try logging into the instance using the root user (port 2222).

SSH into the host instance (port 22):

    $ ssh -i MyKeyPair.pem -p 22 ec2-user@<IP_ADDRESS>
    ec2-user:~$ cat /var/log/user-data.log
    ec2-user:~$ cat /var/log/cloud-init-output.log
    ec2-user:~$ cat /var/log/cloud-init.log

Common problems are user-script failing to run due to wrong filesystem mount type, wrong permissions, etc.

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