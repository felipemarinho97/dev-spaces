# DevSpaces

This is a CLI to help creating on-demand development spaces using EC2 Spot Intances.

Currently, the following commands are availble:
* [start](#starting-a-devspace), [stop](#terminating-devspaces), [status, list](#listing-my-devspaces), [create](#creating-a-devspace), [bootstrap](BOOTSTRAPPING.md), [destroy](#destroying-a-devspace) and [tools](#configuration).


```bash
$ dev-spaces --help
NAME:
   dev-spaces - CLI to help dev-spaces creation and management

USAGE:
   dev-spaces [global options] command [command options] [arguments...]

AUTHOR:
   Felipe Marinho <felipevm97@gmail.com>

COMMANDS:
   help, h  Shows a list of commands or help for one command
   ADMINISTRATION:
     create     -n <name> -k <key-name> -i <ami> [-p <instance-profile-arn> -s <storage-size> -t <prefered-instance-type>]
     bootstrap  -t <template> [-n <name>]
     destroy    -n <name>
     tools
       - scale
       - copy
   DEV-SPACE:
     start   -n <name> [-c <min-cpus> -m <min-memory> --max-price <max-price> -t <timeout>]
     stop    [-n <name>]
     status  [-n <name>]
     list    [-o <output>]

GLOBAL OPTIONS:
   --region value, -r value  AWS region (default: "ap-south-1") [$AWS_REGION]
   --help, -h                show help (default: false)
```

# FAQ

## What is a DevSpace?
A DevSpace is a elastic development environment on AWS. Because there is no need to build a machine if you can cheaply develop on the Cloud!


## My progress is lost when I stop my DevSpace?

No! When you `stop` a DevSpace, the CLI only destroys the instance, leaving the attached EBS Volume intact.
When you call `start` again, the EBS Volume will be attached on the new instance and you can just continue from the point you stop.

This means you are running a _stateful_ workloads on spot instances.

## How I can use it?

```bash
go install github.com/felipemarinho97/dev-spaces@latest
```

Please, follow the steps in this document: [How to create a Dev Space](CREATING.md) and [Configuring the CLI](CONFIGURATION.md).

For the legacy way of bootstraping (for advanced users), please, follow these steps: [How to bootstrap a Dev Space from scratch](BOOTSTRAPPING.md)

If you have any issue during the bootstrap progress, contact the author for more details on how to proceed.

# Exemples
### Starting a DevSpace

You can specify the minimum desired vCPUs and Memory (GBs), as well the max price (in hours) you are willing to pay for the resources.

```bash
$ dev-spaces start --name MySpace --min-cpus 2 --min-memory 4 --max-price 0.05

spot-request-id=sfr-fac050b3-2db3-4d2f-9efa-2403eb239650
instance-id=i-001f2561a626115f5
instance-type=m1.large
```

DevSpaces will be listening by default on SSH port `2222`.

### Listing my DevSpaces

You can list the most recent (last 48h) created DevSpaces.

```bash
$ dev-spaces status                                       
NAME      REQUEST STATE   REQUEST ID                                      CREATE TIME             STATUS    
MySpace   active          sfr-fac050b3-2db3-4d2f-9efa-2403eb239650        2022-02-13T14:37:30Z    fulfilled
teste     cancelled       sfr-6bce6369-7a7b-4d0e-a65e-1498eb5aba90        2022-02-13T13:48:13Z
```

It's also possible to see all the created (regradless if they are active or not) DevSpaces using the command `list`.

```bash
$ dev-spaces list -o wide
SPACE NAME      ID                      CREATE TIME             VERSION   [...]   PUBLIC IP

MySpace         lt-0639c1eccbb51e345    2022-07-07 22:55:01     1         [...]   52.23.206.106
arch            lt-08fb20577838aa54d    2022-07-05 22:02:00     1         [...]   52.91.16.131
al2022-05       lt-0ca2cf57f06544590    2022-07-05 23:01:10     1         [...]   -
```

### Terminating DevSpaces

When you are done, you can use the `stop` command to terminate the DevSpace instance(s).

Note: If you want to stop all running DevSpaces, ommit the `--name` parameter.

```bash
$ dev-spaces stop -n MySpace
```

This will not delete your files, just terminate the DevSpace instance.

---

### Creating a DevSpace

The example below shows an example on how to create a DevSpace using the `create` command.

```bash
$ dev-spaces create --name MySpace --key MyKey --ami ami-1234567890
```

You can also optionaly specify the instance profile ARN `--instance-profile-arn`, the storage size (in GBs) `--storage-size`, and the preferred instance type `--preferred-instance-type`. See all the options [here](CREATING.md#command-parameters).

The `--preferred-instance-type` option helps to create your DevSpace in an avaliability zone with the best possible price for that instance type (this is important because once created, the DevSpace will be locked in that zone).

### Destroying a DevSpace

The command below will destroy the DevSpace instance and all it's associated resources like EBS Volumes, Launch Templates, Security Groups, etc.

```bash  
$ dev-spaces destroy -n MySpace
✓ Destroying security group sg-0b48ecc167b8a81c7 (0/-, 0 it/min) 
✓ Destroying launch template lt-01d0e11ac8523614f (0/-, 0 it/min) 
✓ Destroying volume vol-069210dc254fcdc6b (0/-, 0 it/min)
OK  
```
**This WILL destroy everythng, including all your files.**


## Tools
or Configuration `dev-spaces cfg`

### Scaling Up/Down the DevSpace

The command below will scale up or down the DevSpace instance to the desired number of vCPUs and Memory (GBs).

```bash
$ dev-spaces tools scale -i ~/.ssh/MyKey.pem -n MySpace -c 4 -m 32
```

### Copying the DevSpace to another region

You can use the command `dev-spaces tools copy` to copy the DevSpace to another region.

```bash
# lets say the current region is us-east-1
$ export AWS_REGION=us-east-1
# copy to us-west-1
$ dev-spaces tools copy -n MySpace -r us-west-1 -z us-west-1a
```

Tip: If you want to move the DevSpace to another region, you can use the `copy` command and then the `destroy` command.

