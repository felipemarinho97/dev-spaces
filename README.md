# DevSpaces

This is a CLI to help creating on-demand development spaces using EC2 Spot Intances.

Currently, the followind commands are availble:
* start
* stop
* status

```bash
$ dev-spaces --help
NAME:
   dev-spaces - CLI to help dev-spaces creation and management on AWS

USAGE:
   main [global options] command [command options] [arguments...]

AUTHOR:
   Felipe Marinho <felipevm97@gmail.com>

COMMANDS:
   start    
   stop     
   status   
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h  show help (default: false)
```

## Exemples
### Starting a DevSpace

You can specify the minimum desired vCPUs and Memory (GBs), as well the max price (in hours) you are willing to pay for the resources.

```bash
$ dev-spaces start --name MySpace --min-cpus 2 --min-memory 4 --max-price 0.05

spot-request-id=sfr-fac050b3-2db3-4d2f-9efa-2403eb239650
instance-id=i-001f2561a626115f5
instance-type=m1.large
```

### Listing my DevSpaces

You can list the most recent (last 48h) created DevSpaces.

```bash
$ dev-spaces status                                       
NAME      REQUEST STATE   REQUEST ID                                      CREATE TIME             STATUS    
MySpace   active          sfr-fac050b3-2db3-4d2f-9efa-2403eb239650        2022-02-13T14:37:30Z    fulfilled
teste     cancelled       sfr-6bce6369-7a7b-4d0e-a65e-1498eb5aba90        2022-02-13T13:48:13Z
```

### Terminating DevSpaces

When you are done, you can use the `stop` command to destroy the DevSpace instance(s).

Note: If you want to destroy all running DevSpaces, ommit the `--name` parameter.

```bash
$ dev-spaces stop -n MySpace
```
# FAQ

## What is a DevSpace?
A DevSpace is a elastic development environment on AWS. Because there is no need to build a machine if you can cheaply develop on the Cloud!


## How I can use it?

Right now, this CLI only help managing DevSpaces and is using a _hardcoded_ EC2 Launch Template. It's not possible to bootstrap launch templates via the CLI yet.

If you really want to use this, contact the author for more details on how to manualy bootstrap your EC2 Launch Template.

## My progress is lost when I stop my DevSpace?

No! When you `stop` a DevSpace, the CLI only destroys the instance, leaving the attached EBS Volume intact.
When you call `start` again, the EBS Volume will be attached on the new instance and you can just continue from the point you stop.

This means you are running a _stateful_ workloads on spot instances.