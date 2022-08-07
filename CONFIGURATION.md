# Configuring the CLI

## Creating the configuration file

Create a file called `config.toml` in `$HOME/.config/dev-spaces/` with your configuration. Example:

```toml
default_region = "us-east-1"
```

```bash
# create the config directory
$ mkdir -p $HOME/.config/dev-spaces/
# download the config.toml example file from the repository
$ wget -O $HOME/.config/dev-spaces/config.toml https://raw.githubusercontent.com/felipemarinho97/dev-spaces/master/examples/config.toml
```

Edit the `config.toml` file and customize it to your needs.

Now, to use the CLI all you need is to have your AWS credentials set either in the environment variables or in the `~/.aws/credentials` file. The CLI will also respect the `AWS_PROFILE` and `AWS_REGION` environment variables if it is set.