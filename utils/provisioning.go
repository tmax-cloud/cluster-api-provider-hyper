package utils

import (
	"context"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infraexp "cluster-api-provider-hyper/apis/exp.infrastructure/v1alpha3"
	"cluster-api-provider-hyper/utils/scope"
)

func CentosProvisioning(machineScope *scope.MachineScope, hmp *infraexp.HyperMachinePool, value []byte, isMaster bool) (error, string) {
	sshInfo := hmp.Spec.SSH

	cloudConfig := NewCloudConfig()
	cloudConfig.FromBytes(value)

	conn, err := SSHConnect(sshInfo.Address, sshInfo.SSHid, sshInfo.SSHpw)
	if err != nil {
		return err, "connection error"
	}

	if isMaster {
		configMap := make(map[string]string)
		configMap["$1"] = hmp.Status.NetworkInterface
		configMap["$2"] = hipNum2Pri(machineScope)
		configMap["$3"] = machineScope.HyperCluster.Spec.ControlPlaneEndpoint.Host
		if configMap["$2"] == "100" {
			configMap["$4"] = "MASTER"
		} else {
			configMap["$4"] = "BACKUP"
		}

		conn.SendCommands("rm -f /tmp/keepalived.sh")
		conn.ScpFile("scripts/centos/keepalived.sh", "/tmp/keepalived.sh", configMap)
		conn.SendCommands("chmod +x /tmp/keepalived.sh")
		conn.SendCommands("cd /tmp/ && ./keepalived.sh")
	}

	conn.SendCommands("rm -f /tmp/docker.sh")
	conn.ScpFile("scripts/centos/docker.sh", "/tmp/docker.sh", nil)
	conn.SendCommands("chmod +x /tmp/docker.sh")
	conn.SendCommands("cd /tmp/ && ./docker.sh")

	configMap := make(map[string]string)
	configMap["$1"] = (*machineScope.Machine.Spec.Version)[1:]
	conn.SendCommands("rm -f /tmp/k8s.sh")
	conn.ScpFile("scripts/centos/k8s.sh", "/tmp/k8s.sh", configMap)
	conn.SendCommands("chmod +x /tmp/k8s.sh")
	conn.SendCommands("cd /tmp/ && ./k8s.sh")

	conn.SendCommands("rm -f -r /etc/kubernetes/pki")
	conn.SendCommands("rm -f /tmp/kubeadm.yaml")
	conn.SendCommands("rm -f /tmp/kubeadm-join-config.yaml")
	conn.SendCommands("mkdir -p /etc/kubernetes/pki")
	conn.SendCommands("mkdir -p /etc/kubernetes/pki/etcd")

	for _, wf := range cloudConfig.WriteFiles {
		contents := ""
		for _, content := range wf.Contents {
			if strings.HasPrefix(content, "  name: ") {
				contents = contents + "  name: " + hmp.Status.HostName
				continue
			}
			if content == "" || len(content) == 0 {
				continue
			}
			contents = contents + content + "\\n"
		}

		if output, err := conn.SendCommands("echo -e \"" + contents + "\" >> " + wf.Path); err != nil {
			return err, string(output)
		}
		conn.SendCommands("chmod " + wf.Permissions + " " + wf.Path)
	}

	conn.SendCommands(strings.ReplaceAll(SetCentosProviderID2KubeletConfig, "bmpName", hmp.Status.HostName))
	for _, rc := range cloudConfig.RunCmd {
		if output, err := conn.SendCommands(rc); err != nil {
			return err, string(output)
		}
	}

	conn.Close()

	return nil, ""
}

func UbuntuProvisioning(machineScope *scope.MachineScope, hmp *infraexp.HyperMachinePool, value []byte, isMaster bool) (error, string) {
	sshInfo := hmp.Spec.SSH

	cloudConfig := NewCloudConfig()
	cloudConfig.FromBytes(value)

	conn, err := SSHConnect(sshInfo.Address, sshInfo.SSHid, sshInfo.SSHpw)
	if err != nil {
		return err, "connection error"
	}

	if isMaster {
		configMap := make(map[string]string)
		configMap["$1"] = hmp.Status.NetworkInterface
		configMap["$2"] = hipNum2Pri(machineScope)
		configMap["$3"] = machineScope.HyperCluster.Spec.ControlPlaneEndpoint.Host
		if configMap["$2"] == "100" {
			configMap["$4"] = "MASTER"
		} else {
			configMap["$4"] = "BACKUP"
		}

		conn.SendCommands("rm /tmp/keepalived.sh")
		conn.ScpFile("scripts/ubuntu/keepalived.sh", "/tmp/keepalived.sh", configMap)
		conn.SendCommands("chmod +x /tmp/keepalived.sh")
		conn.SendCommands("cd /tmp/ && ./keepalived.sh")
	}

	conn.SendCommands("rm /tmp/docker.sh")
	conn.ScpFile("scripts/ubuntu/docker.sh", "/tmp/docker.sh", nil)
	conn.SendCommands("chmod +x /tmp/docker.sh")
	conn.SendCommands("cd /tmp/ && ./docker.sh")

	configMap := make(map[string]string)
	configMap["$1"] = (*machineScope.Machine.Spec.Version)[1:]
	conn.SendCommands("rm /tmp/k8s.sh")
	conn.ScpFile("scripts/ubuntu/k8s.sh", "/tmp/k8s.sh", configMap)
	conn.SendCommands("chmod +x /tmp/k8s.sh")
	conn.SendCommands("cd /tmp/ && ./k8s.sh")

	conn.SendCommands("rm -r /etc/kubernetes/pki")
	conn.SendCommands("rm /tmp/kubeadm.yaml")
	conn.SendCommands("rm /tmp/kubeadm-join-config.yaml")
	conn.SendCommands("mkdir -p /etc/kubernetes/pki")
	conn.SendCommands("mkdir -p /etc/kubernetes/pki/etcd")

	for _, wf := range cloudConfig.WriteFiles {
		contents := ""
		for _, content := range wf.Contents {
			if strings.HasPrefix(content, "  name: ") {
				contents = contents + "  name: " + hmp.Status.HostName
				continue
			}
			if content == "" || len(content) == 0 {
				continue
			}
			contents = contents + content + "\\n"
		}

		if output, err := conn.SendCommands("echo -e \"" + contents + "\" >> " + wf.Path); err != nil {
			return err, string(output)
		}
		conn.SendCommands("chmod " + wf.Permissions + " " + wf.Path)
	}

	conn.SendCommands(strings.ReplaceAll(SetUbuntuProviderID2KubeletConfig, "bmpName", hmp.Status.HostName))

	for _, rc := range cloudConfig.RunCmd {
		if output, err := conn.SendCommands(rc); err != nil {
			return err, string(output)
		}
	}

	conn.Close()

	return nil, ""
}

func hipNum2Pri(machineScope *scope.MachineScope) string {
	hipNum := ""

	if val, ok := machineScope.HyperCluster.Labels[LabelHyperIPPoolName]; ok {
		hip := &infraexp.HyperIPPool{}
		machineScope.GetClient().Get(context.TODO(), types.NamespacedName{Namespace: machineScope.Cluster.Namespace, Name: val}, hip)

		oldhip := hip.DeepCopy()
		newhip := hip.DeepCopy()

		if num, err := strconv.Atoi(newhip.Labels[LabelHyperIPPool]); err == nil {
			newhip.Labels[LabelHyperIPPool] = strconv.Itoa(num + 1)
			hipNum = strconv.Itoa(100 - num)
		}

		machineScope.GetClient().Patch(context.TODO(), newhip, client.MergeFrom(oldhip))
	}

	return hipNum
}
