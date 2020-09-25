package utils

const (
	LabelHyperMachinePoolValid = "infrastructure.cluster.x-k8s.io/hypermachinepool-valid"
	LabelClusterName           = "infrastructure.cluster.x-k8s.io/cluster-name"
	LabelClusterRoleMaster     = "cluster.x-k8s.io/control-plane"
	LabelClusterRoleWorker     = "cluster.x-k8s.io/deployment-name"

	SetProviderID2KubeletConfig = "sed -i 's/--config=\\/var\\/lib\\/kubelet\\/config.yaml/--config=\\/var\\/lib\\/kubelet\\/config.yaml --provider-id hyper:\\/\\/\\/bmpName/' /etc/systemd/system/kubelet.service.d/10-kubeadm.conf"
)
