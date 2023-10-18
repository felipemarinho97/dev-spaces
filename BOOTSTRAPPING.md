# How to bootstrap a Dev Space

This section describes how to bootstrap a Dev Space.


## Create a key pair to SSH into the instance

Follow the steps in the [Create a key pair](KEYPAIR.md) section to create a key pair and store it in a file.

## Use the CLI to bootstrap a Dev Space

Download one of the following bootstrap scripts:
 - [Bootstrap Script for Arch Linux](https://raw.githubusercontent.com/felipemarinho97/dev-spaces/master/examples/templates/arch.yaml)

Fell free to create your own bootstrap scripts and share them here.

Edit the `key_name` field to match your key pair you just created. You can also edit the other fields to match your requirements, like the availability zone, the instance type, instance profile, the EBS volume size, etc.

Then, you can use the CLI to bootstrap a Dev Space.

```bash
$ dev-spaces bootstrap --template arch.yaml --name MyDevSpace
```

The CLI will take care of creating the Dev Space.

Once the Dev Space is created, you can use the CLI to start it.

```bash
$ dev-spaces start -n MyDevSpace --min-cpus 2 --min-memory 4 --max-price 0.08
```