#!/bin/bash

# abort on error
set -e

# Determine the OS (darwin, linux)
OS=$(uname -s | tr '[:upper:]' '[:lower:]')

# Determine the architecture (amd64, arm64, 386)
ARCH=$(uname -m | tr '[:upper:]' '[:lower:]')
case $ARCH in
    x86_64) ARCH=amd64 ;;
    aarch64) ARCH=arm64 ;;
    i?86) ARCH=386 ;;
esac

# Determine the latest version
REPO=https://github.com/felipemarinho97/dev-spaces
VERSION=$(curl -sI $REPO/releases/latest | grep -i "location:" | awk -F"/" '{ printf "%s", $NF }' | tr -d '\r')

# Construct the release archive url
URL=$REPO/releases/download/$VERSION/dev-spaces-$VERSION-$OS-$ARCH.tar.gz

# Download the release archive and extract it on /tmp
echo "Downloading $URL ..."
curl -sL $URL | tar -xz -C /tmp

# Make the binary executable
chmod +x /tmp/dev-spaces

# Move the binary to a PATH under home directory
HOME_BIN=$HOME/bin
mkdir -p $HOME_BIN
mv /tmp/dev-spaces $HOME_BIN

# Check if the binary is in PATH, if not, add it
if [[ ":$PATH:" != *":$HOME_BIN:"* ]]; then
    echo "Adding $HOME_BIN/dev-spaces executable to PATH ..."

    # determine the shell rcfile
    case $SHELL in
        */bash) RCFILE=~/.bashrc ;;
        */zsh) RCFILE=~/.zshrc ;;
        */fish) RCFILE=~/.config/fish/config.fish ;;
        *) RCFILE=~/.bashrc ;;
    esac

    echo "export PATH=$HOME_BIN:\$PATH" >> $RCFILE
    source $RCFILE
fi

# create a symlink to the binary named 'ds'
ln -sf $HOME_BIN/dev-spaces $HOME_BIN/ds

echo "Done!"
echo "Run 'source $RCFILE' to use 'dev-spaces' or 'ds' command on the current shell or open a new terminal"
echo "Run 'dev-spaces --help' or 'ds --help' to get started"
