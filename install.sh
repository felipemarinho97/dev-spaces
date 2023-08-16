#!/bin/bash

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
URL=$REPO/releases/download/$VERSION/dev-spaces-$OS-$ARCH.tar.gz

# Download the release archive and extract it
echo "Downloading $URL ..."
curl -sL $URL | tar -xz

# Move the binary to the current directory
mv dev-spaces-$OS-$ARCH/dev-spaces .

# Remove the release archive
rm -rf dev-spaces-$OS-$ARCH

# Make the binary executable
chmod +x dev-spaces

# Move the binary to a PATH under home directory
HOME_BIN=$HOME/bin
mkdir -p $HOME_BIN
mv dev-spaces $HOME_BIN

# Check if the binary is in PATH, if not, add it
if [[ ":$PATH:" != *":$HOME_BIN:"* ]]; then
    echo "Adding $HOME_BIN to PATH ..."

    # determine the shell rcfile
    case $SHELL in
        /bin/bash) RCFILE=~/.bashrc ;;
        /bin/zsh) RCFILE=~/.zshrc ;;
        /bin/fish) RCFILE=~/.config/fish/config.fish ;;
        *) RCFILE=~/.bashrc ;;
    esac

    echo "export PATH=$HOME_BIN:\$PATH" >> $RCFILE
    source $RCFILE
fi

echo "Done!"