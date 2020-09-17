/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type SSHinfo struct {
	Address string `json:"address"`
	SSHid   string `json:"id"`
	SSHpw   string `json:"pw"`
}

// HyperMachinePoolSpec defines the desired state of HyperMachinePool
type HyperMachinePoolSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	SSH *SSHinfo `json:"ssh"`
}

// HyperMachinePoolStatus defines the observed state of HyperMachinePool
type HyperMachinePoolStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Error  string `json:"Error,omitempty"`
	OS     string `json:"os,omitempty"`
	Kernel string `json:"kernel,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=hypermachinepools,scope=Namespaced,categories=cluster-api,shortName=hmp
// +kubebuilder:printcolumn:name="Valid",type="string",JSONPath=".metadata.labels.infrastructure\\.cluster\\.x-k8s\\.io/hypermachinepool-valid",description="is hypermachine's SSH info valid"
// +kubebuilder:printcolumn:name="Ip",type="string",JSONPath=".spec.ssh.address",description="ip:port address"
// +kubebuilder:printcolumn:name="OS",type="string",JSONPath=".status.os",description="os"

// HyperMachinePool is the Schema for the hypermachinepools API
type HyperMachinePool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HyperMachinePoolSpec   `json:"spec,omitempty"`
	Status HyperMachinePoolStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// HyperMachinePoolList contains a list of HyperMachinePool
type HyperMachinePoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HyperMachinePool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HyperMachinePool{}, &HyperMachinePoolList{})
}
