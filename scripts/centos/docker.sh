yum -y install docker docker-registry
systemctl enable docker.service
systemctl start docker.service