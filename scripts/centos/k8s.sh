sudo systemctl disable firewalld
sudo systemctl stop firewalld

sudo swapoff -a
sudo sed s/\\/dev\\/mapper\\/centos-swap/\# \\/dev\\/mapper\\/centos-swap/g -i /etc/fstab

sudo setenforce 0
sudo sed -i 's/^SELINUX=enforcing$/SELINUX=permissive/' /etc/selinux/config

modprobe overlay
modprobe br_netfilter
echo '1' > /proc/sys/net/bridge/bridge-nf-call-iptables

swapoff -a

sudo cat << "EOF" | sudo tee -a /etc/yum.repos.d/kubernetes.repo
[kubernetes]
name=Kubernetes
baseurl=https://packages.cloud.google.com/yum/repos/kubernetes-el7-x86_64
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
EOF

yum remove -y kubeadm kubelet kubectl
yum install -y kubeadm-$1-0 kubelet-$1-0 kubectl-$1-0
systemctl enable kubelet --now