package create

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

## mount the DevSpace
FSTYPE=$(lsblk /dev/sdf1 -f -o FSTYPE | tail -1)
MOUNTPOINT=/devspace
mkdir -p $MOUNTPOINT
if [ "$FSTYPE" == "xfs" ]; then
	mount -t xfs -o nouuid "$DEVICE"1 $MOUNTPOINT
else
	mount "$DEVICE"1 $MOUNTPOINT
fi

## enable ip forwarding
echo 1 > /proc/sys/net/ipv4/ip_forward

## change SSH port to 2222
sed '/#Port 22/s/#Port 22/Port 2222/g' -i $MOUNTPOINT/etc/ssh/sshd_config
mkdir -p $MOUNTPOINT/root/.ssh/
touch $MOUNTPOINT/root/.ssh/authorized_keys
curl http://169.254.169.254/latest/meta-data/public-keys/0/openssh-key > $MOUNTPOINT/root/.ssh/authorized_keys

## boot the chroot machine
export SYSTEMD_SECCOMP=0
systemd-nspawn --boot --quiet --machine=devspace --capability=all -D $MOUNTPOINT/
`
const AMI_PATH = "/aws/service/ami-amazon-linux-latest/"

const API_PARAMETER_PREFIX = "/aws/service/ami-amazon-linux-latest/al2022-ami-minimal-kernel-default-"
