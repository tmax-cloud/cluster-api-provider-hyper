package scope

import (
	infrav1 "cluster-api-provider-hyper/apis/infrastructure/v1alpha3"
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/klog/klogr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterScopeParams defines the input parameters used to create a new Scope.
type ClusterScopeParams struct {
	Client       client.Client
	Logger       logr.Logger
	Cluster      *clusterv1.Cluster
	HyperCluster *infrav1.HyperCluster
}

// NewClusterScope creates a new Scope from the supplied parameters.
// This is meant to be called for each reconcile iteration.
func NewClusterScope(params ClusterScopeParams) (*ClusterScope, error) {
	if params.Cluster == nil {
		return nil, errors.New("failed to generate new scope from nil Cluster")
	}
	if params.HyperCluster == nil {
		return nil, errors.New("failed to generate new scope from nil HyperCluster")
	}

	if params.Logger == nil {
		params.Logger = klogr.New()
	}

	helper, err := patch.NewHelper(params.HyperCluster, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}
	return &ClusterScope{
		Logger:       params.Logger,
		client:       params.Client,
		Cluster:      params.Cluster,
		HyperCluster: params.HyperCluster,
		patchHelper:  helper,
	}, nil
}

// ClusterScope defines the basic context for an actuator to operate upon.
type ClusterScope struct {
	logr.Logger
	client      client.Client
	patchHelper *patch.Helper

	Cluster      *clusterv1.Cluster
	HyperCluster *infrav1.HyperCluster

	controllerName string
}

func (s *ClusterScope) GetClient() client.Client {
	return s.client
}

// PatchObject persists the cluster configuration and status.
func (s *ClusterScope) PatchObject() error {
	return s.patchHelper.Patch(
		context.TODO(),
		s.HyperCluster,
		patch.WithOwnedConditions{Conditions: []clusterv1.ConditionType{
			clusterv1.ReadyCondition,
		}})
}

// Close closes the current scope persisting the cluster configuration and status.
func (s *ClusterScope) Close() error {
	return s.PatchObject()
}
