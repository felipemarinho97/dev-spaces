# VSCode Integration

To integrate your DevSpace project with VSCode, make sure you have the [Remote Development Extension Pack](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.vscode-remote-extensionpack) or the [Remote SSH Extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-ssh) installed. Once installed, you can proceed to the next step.

## SSH Configuration

By default, DevSpace will create a new SSH config entry for each of your DevSpace projects. You can find the SSH config entry for a specific project in the `~/.ssh/config.d/dev-spaces/<devspace-name>` file. You can customize this file as you want, like renaming the hostname, changing the default user, etc.

For VSCode integration, is recommended to add an `IdentityFile` entry to the SSH config entry. This will allow VSCode to automatically connect to your DevSpace project without asking for the SSH key password. Edit the `~/.ssh/config.d/dev-spaces/<devspace-name>` file and add the `IdentityFile` entry like this:

```yaml
Host <devspace-name>
    HostName 18.118.32.169
    Port 2222
    User root # if you want, you can change this to the default user of your image
    StrictHostKeyChecking no
    IdentityFile ~/.ssh/my_id_rsa # <-- Add this line with the path to your SSH key
```

This file is automatically updated by DevSpace, so any time you run `dev-spaces start`, the field `HostName` will be updated with the current public IP of your DevSpace project.

**Important:** If you created your DevSpace with **Amazon Linux**, you will need to change the `User` field to `ec2-user` instead of `root`.

### Manual Configuration

If for some reason this file was not created automatically, you can create manually the following entry in your `~/.ssh/config` file:

```yaml
Host <devspace-name>
    HostName <devspace-ip>
    Port 2222
    User <default-user>
    StrictHostKeyChecking no
    IdentityFile <path-to-ssh-key>
```

To obtain your devspace IP, you can run `dev-spaces list -o wide` and look for the `IP` column for your devspace entry. For the user, generally, the default user is `ubuntu` for Ubuntu-based images and `ec2-user` for Amazon Linux-based images.

Keep in mind that by manually creating this entry, the DevSpaces tool will not be able to update the `HostName` field automatically, so you will have to update it manually every time you restart your DevSpace project. One workaround for this is to create an dynamic DNS using a service like [No-IP](https://www.noip.com/) or [Duck DNS](https://www.duckdns.org/) and use the DNS name instead of the IP address.

## VSCode Configuration

Once you have your SSH config entry ready (with the IdentityFile path filled), you can proceed to configure VSCode to connect to your DevSpace project. To do so, inside VSCode open the command palette (Ctrl+Shift+P) and search for the `Remote-SSH: Connect to Host...` command. Select the SSH config entry for your DevSpace project and wait for VSCode to connect to your DevSpace project.

It's also possible to connect to your DevSpace from the command line using the `code` command. To do so, run the following command:

```bash
code --remote ssh-remote+<devspace-name>
```

## VSCode Extensions

Once you have connected to your DevSpace project, you can install any VSCode extension you want. The extensions will be installed inside your DevSpace project and will be available every time you connect to your DevSpace project.