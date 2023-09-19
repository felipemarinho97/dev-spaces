package core

const DEFAULT_STARTUP_SCRIPT = `#!/bin/bash -xe
exec > >(tee /var/log/user-data.log|logger -t user-data -s 2>/dev/console) 2>&1

## add required packages
yum install -y systemd-container

## start networkd and resolved 
systemctl start systemd-resolved 
systemctl start systemd-networkd 

## wait for the volume to attach
DEVICE=/dev/sdf
while [ ! -e $DEVICE ]; do
sleep 1s
done

# get the device name of the current device on /
root_device=$(df / | tail -1 | awk '{print $1}' | cut -d "/" -f 3)

# get the biggest not mounted device
not_mounted=$(lsblk -o NAME,MOUNTPOINTS,SIZE,TYPE -x SIZE | grep part | grep -v $root_device | tail -n1 | awk '{print $1}')
DEVICE=/dev/$not_mounted

## mount the DevSpace
FSTYPE=$(lsblk $DEVICE -f -o FSTYPE | tail -1)
MOUNTPOINT=/devspace
mkdir -p $MOUNTPOINT
if [ "$FSTYPE" == "xfs" ]; then
	mount -t xfs -o nouuid "$DEVICE" $MOUNTPOINT
else
	mount "$DEVICE" $MOUNTPOINT
fi

## enable ip forwarding
echo 1 > /proc/sys/net/ipv4/ip_forward

## change SSH port to 2222
sed '/#Port 22/s/#Port 22/Port 2222/g' -i $MOUNTPOINT/etc/ssh/sshd_config
mkdir -p $MOUNTPOINT/root/.ssh/
touch $MOUNTPOINT/root/.ssh/authorized_keys
TOKEN=$(curl -X PUT "http://169.254.169.254/latest/api/token" -H "X-aws-ec2-metadata-token-ttl-seconds: 21600")
curl http://169.254.169.254/latest/meta-data/public-keys/0/openssh-key -H "X-aws-ec2-metadata-token: $TOKEN" > $MOUNTPOINT/root/.ssh/authorized_keys

## for each user, add the public key to authorized_keys
for user in $(ls $MOUNTPOINT/home); do
	mkdir -p $MOUNTPOINT/home/$user/.ssh/
	cat $MOUNTPOINT/root/.ssh/authorized_keys > $MOUNTPOINT/home/$user/.ssh/authorized_keys
done

## boot the chroot machine
export SYSTEMD_SECCOMP=0
systemd-nspawn --boot --quiet --machine=devspace --capability=all -D $MOUNTPOINT/
`
const AMI_PATH = "/aws/service/ami-amazon-linux-latest/"

const API_PARAMETER_PREFIX = "/aws/service/ami-amazon-linux-latest/al2022-ami-minimal-kernel-default-"
