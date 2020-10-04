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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	HyperMachineFinalizer = "hypermachine.infrastructure.cluster.x-k8s.io"
)

// HyperMachineSpec defines the desired state of HyperMachine
type HyperMachineSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:default=""
	// +optional
	// +kubebuilder:validation:Optional
	ProviderID *string `json:"providerID,omitempty"`

	// +kubebuilder:validation:Optional
	// +optional
	HyperMachinePoolRef corev1.ObjectReference `json:"hyperMachinePoolRef"`
}

// HyperMachineStatus defines the observed state of HyperMachine
type HyperMachineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +optional
	Ready bool `json:"ready"`

	// Addresses contains the Hyper instance associated addresses.
	Addresses []clusterv1.MachineAddress `json:"addresses,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=hypermachines,scope=Namespaced,categories=cluster-api
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".metadata.labels.cluster\\.x-k8s\\.io/cluster-name",description="Cluster to which this HyperMachine belongs"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.ready",description="Machine ready status"
// +kubebuilder:printcolumn:name="Machine",type="string",JSONPath=".metadata.ownerReferences[?(@.kind==\"Machine\")].name",description="Machine object which owns with this HyperMachine"
// +kubebuilder:printcolumn:name="HyperMachinePool",type="string",JSONPath=".spec.hyperMachinePoolRef.name",description="hyperMachinePool adopted by hyperMachine"

// HyperMachine is the Schema for the hypermachines API
type HyperMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HyperMachineSpec   `json:"spec,omitempty"`
	Status HyperMachineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// HyperMachineList contains a list of HyperMachine
type HyperMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HyperMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HyperMachine{}, &HyperMachineList{})
}
