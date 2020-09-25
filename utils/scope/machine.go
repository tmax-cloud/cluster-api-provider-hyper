package scope

import (
	"context"
	"fmt"

	infrav1 "cluster-api-provider-hyper/apis/infrastructure/v1alpha3"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/klogr"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MachineScopeParams defines the input parameters used to create a new MachineScope.
type MachineScopeParams struct {
	Client       client.Client
	Logger       logr.Logger
	Cluster      *clusterv1.Cluster
	Machine      *clusterv1.Machine
	HyperMachine *infrav1.HyperMachine
}

// NewMachineScope creates a new MachineScope from the supplied parameters.
// This is meant to be called for each reconcile iteration.
func NewMachineScope(params MachineScopeParams) (*MachineScope, error) {
	if params.Client == nil {
		return nil, errors.New("client is required when creating a MachineScope")
	}
	if params.Machine == nil {
		return nil, errors.New("machine is required when creating a MachineScope")
	}
	if params.Cluster == nil {
		return nil, errors.New("cluster is required when creating a MachineScope")
	}
	if params.HyperMachine == nil {
		return nil, errors.New("hyper machine is required when creating a MachineScope")
	}

	if params.Logger == nil {
		params.Logger = klogr.New()
	}

	helper, err := patch.NewHelper(params.HyperMachine, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}
	return &MachineScope{
		Logger:      params.Logger,
		client:      params.Client,
		patchHelper: helper,

		Cluster:      params.Cluster,
		Machine:      params.Machine,
		HyperMachine: params.HyperMachine,
	}, nil
}

// MachineScope defines a scope defined around a machine and its cluster.
type MachineScope struct {
	logr.Logger
	client      client.Client
	patchHelper *patch.Helper

	Cluster      *clusterv1.Cluster
	Machine      *clusterv1.Machine
	InfraCluster *infrav1.HyperCluster
	HyperMachine *infrav1.HyperMachine
}

//GetClient return client
func (m *MachineScope) GetClient() client.Client {
	return m.client
}

// Name returns the HyperMachine name.
func (m *MachineScope) Name() string {
	return m.HyperMachine.Name
}

// Namespace returns the namespace name.
func (m *MachineScope) Namespace() string {
	return m.HyperMachine.Namespace
}

// IsControlPlane returns true if the machine is a control plane.
func (m *MachineScope) IsControlPlane() bool {
	return util.IsControlPlaneMachine(m.Machine)
}

// Role returns the machine role from the labels.
func (m *MachineScope) Role() string {
	if util.IsControlPlaneMachine(m.Machine) {
		return "master"
	}
	return "worker"
}

// GetProviderID returns the HyperMachine providerID from the spec.
func (m *MachineScope) GetProviderID() string {
	if m.HyperMachine.Spec.ProviderID != nil {
		return *m.HyperMachine.Spec.ProviderID
	}
	return ""
}

// SetProviderID sets the HyperMachine providerID in spec.
func (m *MachineScope) SetProviderID(hmpName string) {
	providerID := fmt.Sprintf("hyper:///%s", hmpName)
	m.HyperMachine.Spec.ProviderID = pointer.StringPtr(providerID)
}

func (m *MachineScope) SetAddress(address string) {
	m.HyperMachine.Status.Addresses = clusterv1.MachineAddresses{
		clusterv1.MachineAddress{
			Type:    "InternalIP",
			Address: address,
		},
	}
}

// SetReady sets the HyperMachine Ready Status
func (m *MachineScope) SetReady() {
	m.HyperMachine.Status.Ready = true
}

// SetNotReady sets the HyperMachine Ready Status to false
func (m *MachineScope) SetNotReady() {
	m.HyperMachine.Status.Ready = false
}

// GetBootstrapData returns the bootstrap data from the secret in the Machine's bootstrap.dataSecretName as base64.
func (m *MachineScope) GetBootstrapData() ([]byte, error) {
	value, err := m.GetRawBootstrapData()
	if err != nil {
		return nil, err
	}
	return value, nil
}

// GetRawBootstrapData returns the bootstrap data from the secret in the Machine's bootstrap.dataSecretName.
func (m *MachineScope) GetRawBootstrapData() ([]byte, error) {
	if m.Machine.Spec.Bootstrap.DataSecretName == nil {
		return nil, errors.New("error retrieving bootstrap data: linked Machine's bootstrap.dataSecretName is nil")
	}

	secret := &corev1.Secret{}
	key := types.NamespacedName{Namespace: m.Namespace(), Name: *m.Machine.Spec.Bootstrap.DataSecretName}
	if err := m.client.Get(context.TODO(), key, secret); err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve bootstrap data secret for HyperMachine %s/%s", m.Namespace(), m.Name())
	}

	value, ok := secret.Data["value"]
	if !ok {
		return nil, errors.New("error retrieving bootstrap data: secret value key is missing")
	}

	return value, nil
}

// PatchObject persists the machine spec and status.
func (m *MachineScope) PatchObject() error {
	return m.patchHelper.Patch(
		context.TODO(),
		m.HyperMachine,
		patch.WithOwnedConditions{Conditions: []clusterv1.ConditionType{
			clusterv1.ReadyCondition,
		}})
}

// Close the MachineScope by updating the machine spec, machine status.
func (m *MachineScope) Close() error {
	return m.PatchObject()
}
