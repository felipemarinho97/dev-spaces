# Creating a DevSpace

This document describes how to bootstrap a Dev Space using the command `dev-spaces create`. This new version has a better approach on creating the environment because it's more generic and it's easier to manage. Also, it's much more faster (just take few seconds to bootstap a space) than the previous version.

## Instance Profile

There is no need to create an instance profile with EBS permissions because the `start` command will wait until the instance is ready and attach the EBS Volume by itself. You can still specify the instance profile in the bootstrap template if you want to.

## Creating the Space

The command below will use the pre-defined template to create an space with Amazon Linux 2022 AMI.

    $ dev-spaces bootstrap-v2 --template-file templates/v2/al2022-mumbai.yaml --region ap-south-1

This AMI have the advantage of supporting running docker inside the Dev Space.

# DISCLAIMER

This section is a work in progress and is subject to change without notice.