# How to bootstrap a Dev Space

This section describes how to bootstrap a Dev Space.

## Creating the Instance Profile

First, you need to create an AWS Instance Profile with permissions to allow the machine to attach the EBS Volume.

```bash
$ aws iam create-instance-profile --instance-profile-name MyInstanceProfile
```

Then, you need to create a role with the permissions to allow the machine to attach the EBS Volume.

```bash
$ aws iam create-role --role-name MyRole --assume-role-policy-document file://<(echo '{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}')
```

Add inline policy to the role to allow the machine to attach the EBS Volume.

```bash
$ aws iam put-role-policy --role-name MyRole --policy-name AttachVolume --policy-document file://<(echo '{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:DetachVolume",
        "ec2:AttachVolume",
        "ec2:ModifyVolume",
        "ec2:ModifyVolumeAttribute",
        "ec2:DescribeVolumeAttribute"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeInstances",
        "ec2:DescribeVolumeStatus",
        "ec2:DescribeVolumes",
        "ec2:DescribeInstanceStatus"
      ],
      "Resource": "*"
    }
  ]
}')
```

Then, you need to attach the role to the instance profile.

```bash
$ aws iam add-role-to-instance-profile --instance-profile-name MyInstanceProfile --role-name MyRole
```

Get the instance profile ARN.

```bash
$ aws iam get-instance-profile --instance-profile-name MyInstanceProfile --query 'InstanceProfile.Arn'
```

## Create a key pair to SSH into the instance

Now, create a key pair and store it in a file, if you already have a key pair, you can skip this step.

```bash
$ aws ec2 create-key-pair --key-name MyKeyPair --query 'KeyMaterial' --output text > MyKeyPair.pem
```

Remember to give the right permissions to the file.

```bash
$ chmod 400 MyKeyPair.pem
```

## Use the CLI to bootstrap a Dev Space

Download one of the following bootstrap scripts:
 - [Bootstrap Script for Arch Linux](https://raw.githubusercontent.com/felipemarinho97/dev-spaces/master/templates/v1/arch.yaml)

Edit the `key_name` and `instance_profile_arn` fields to match your key pair and instance profile you just created.

You can also edit the other fields to match your requirements, like the availability zone, the instance type, the EBS volume size, etc.

Then, you can use the CLI to bootstrap a Dev Space.

```bash
$ dev-spaces bootstrap --template arch.yaml --name MyDevSpace
```

The CLI will take care of creating the Dev Space.

Once the Dev Space is created, you can use the CLI to start it.

```bash
$ dev-spaces start -n MyDevSpace --min-cpus 2 --min-memory 4 --max-price 0.08
```