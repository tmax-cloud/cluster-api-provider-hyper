package utils

const (
	LabelHyperMachinePoolValid = "infrastructure.cluster.x-k8s.io/hypermachinepool-valid"
	LabelHyperIPPool           = "infrastructure.cluster.x-k8s.io/hyperippool"
	LabelHyperIPPoolName       = "infrastructure.cluster.x-k8s.io/hyperippool-name"
	LabelClusterName           = "infrastructure.cluster.x-k8s.io/cluster-name"
	LabelClusterRoleMaster     = "cluster.x-k8s.io/control-plane"
	LabelClusterRoleWorker     = "cluster.x-k8s.io/deployment-name"

	SetUbuntuProviderID2KubeletConfig = "sed -i 's/--config=\\/var\\/lib\\/kubelet\\/config.yaml/--config=\\/var\\/lib\\/kubelet\\/config.yaml --provider-id hyper:\\/\\/\\/bmpName/' /etc/systemd/system/kubelet.service.d/10-kubeadm.conf"
	SetCentosProviderID2KubeletConfig = "sed -i 's/--config=\\/var\\/lib\\/kubelet\\/config.yaml/--config=\\/var\\/lib\\/kubelet\\/config.yaml --provider-id hyper:\\/\\/\\/bmpName/' /usr/lib/systemd/system/kubelet.service.d/10-kubeadm.conf"
)
