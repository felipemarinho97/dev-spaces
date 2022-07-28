# Configuring the CLI

To the CLI be able to place Spot Fleet Requests on AWS, you will need to create an Spot Fleet Request Role. This role will be used by the `CreateSpotFleetRequest` command.

```bash
$ aws iam create-role --role-name dev-spaces-spot-fleet-request-role --assume-role-policy-document '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"spotfleet.amazonaws.com"},"Action":"sts:AssumeRole"}]}'
```

Add the following managed policy to the role:

```bash
$ aws iam attach-role-policy --role-name dev-spaces-spot-fleet-request-role --policy-arn arn:aws:iam::aws:policy/service-role/AmazonEC2SpotFleetTaggingRole
```

Now, to get the `arn` of the role:

```bash
$ ROLE_ARN=$(aws iam get-role --role-name dev-spaces-spot-fleet-request-role --query 'Role.Arn' --output text)
```

## Creating the configuration file

Create a file called `config.toml` in `$HOME/.config/dev-spaces/` with the role ARN:

```bash
$ mkdir -p $HOME/.config/dev-spaces/
# download the config.toml file from the repository
$ wget -O $HOME/.config/dev-spaces/config.toml https://raw.githubusercontent.com/felipemarinho97/dev-spaces/master/examples/config.toml
# uncomment "spot_fleet_role_arn" line and replace the value with the ARN of the role you just created
$ sed -i -E "s|# spot_fleet_role_arn = \"<ROLE_ARN>\"|spot_fleet_role_arn = \"$ROLE_ARN\"|g" $HOME/.config/dev-spaces/config.toml
```


Now, all you need to have your AWS credentials set either in the environment variables or in the `~/.aws/credentials` file. The CLI will also respect the `AWS_PROFILE` and `AWS_REGION` environment variables if it is set.