apt-get purge -y keeplaived
apt-get install -y keepalived
cat <<EOF | tee /etc/keepalived/keepalived.conf
vrrp_instance VI_1 {
state $4
interface $1
virtual_router_id 50
priority $2
advert_int 1
nopreempt
authentication {
        auth_type PASS
        auth_pass 7777
        }
virtual_ipaddress {
        $3
        }
}
EOF

systemctl restart keepalived
systemctl enable keepalived