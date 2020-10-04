## install script for docker
apt-get purge docker docker-engin docker.io docker-ce -y
apt-get install apt-transport-https ca-certificates curl software-properties-common -y
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add -
apt-key fingerprint 0EBFCD88
add-apt-repository \"deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable\"
apt-get update
apt-get install docker-ce=18.06.2~ce~3-0~ubuntu -y

mkdir -p /etc/docker
touch /etc/docker/daemon.json

cat <<EOF | tee /etc/docker/daemon.json
{
  \"exec-opts\": [\"native.cgroupdriver=systemd\"],
  \"log-driver\": \"json-file\",
  \"log-opts\": {
    \"max-size\": \"100m\"
  },
  \"storage-driver\": \"overlay2\"
}
EOF

mkdir -p /etc/systemd/system/docker.service.d
systemctl daemon-reload
systemctl restart docker
