curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
cat <<EOF | tee /etc/apt/sources.list.d/kubernetes.list
deb https://apt.kubernetes.io/ kubernetes-xenial main
EOF
apt-get update
swapoff -a
apt-get purge -qy kubelet kubectl kubeadm
apt-get install -qy kubelet=$1-00 kubectl=$1-00 kubeadm=$1-00
