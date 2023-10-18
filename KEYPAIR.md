# How to create a key pair

This section describes how to create a key pair to use with Dev Spaces. A key pair is required to SSH into the Dev Space.


## Create a key pair to SSH into the instance using the CLI

Now, create a key pair and store it in a file, if you already have a key pair, you can skip this step.

```bash
$ aws ec2 create-key-pair --key-name MyKeyPair --query 'KeyMaterial' --output text > MyKeyPair.pem
```

Remember to give the right permissions to the file.

```bash
$ chmod 400 MyKeyPair.pem
```

## Create a key pair to SSH into the instance using the AWS Console

To create a key pair using Amazon EC2

1. Open the Amazon EC2 console at https://console.aws.amazon.com/ec2/.
2. In the navigation pane, under **NETWORK & SECURITY**, choose **Key Pairs**.
3. Choose **Create Key Pair**.
4. For **Name**, enter a descriptive name for the key pair. Amazon EC2 associates the public key with the name that you specify as the key name. A key name can include up to 255 ASCII characters. It can’t include leading or trailing spaces.
5. For **Key pair type**, choose either RSA or ED25519.
6. For **Private key file format**, choose the format in which to save the private key. To save the private key in a format that can be used with OpenSSH, choose pem. To save the private key in a format that can be used with PuTTY, choose ppk.
7. Choose **Create key pair**.
8. The private key file is automatically downloaded by your browser. The base file name is the name that you specified as the name of your key pair, and the file name extension is determined by the file format that you chose. Save the private key file in a safe place.
9. If you plan to use an SSH client on a macOS or Linux computer to connect to your Linux instance, use the following command to set the permissions of your private key file so that only you can read it.
    
    ```bash
    $ chmod 400 MyKeyPair.pem
    ```
    